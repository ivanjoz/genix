//! HTTP surface of the bridge. Two endpoints face the browser (`/sse`, `/in`) and two face
//! the backend (`/publish`, `/rpc`). The bridge never interprets the messages it moves: they
//! are opaque JSON objects framed as SSE `data:` events. The single exception is the command
//! id, which travels as its own field so a reply can be correlated without parsing the
//! payload.

use std::{
    convert::Infallible,
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

use axum::{
    Json, Router,
    extract::{Query, State},
    http::{HeaderMap, HeaderValue, StatusCode, header},
    response::{
        IntoResponse, Response,
        sse::{Event, KeepAlive, Sse},
    },
    routing::{get, post},
};
use serde::Deserialize;
use serde_json::{Value, json};
use tokio::sync::mpsc;
use tokio_stream::Stream;
use tracing::{debug, info, warn};

use crate::bridge::{
    auth::{SERVICE_AUTH_HEADER, authenticate_user, verify_service_auth},
    channel::{ChannelRegistry, ClientChannel, ReplyEnvelope, SendFrameError},
    token::decode_channel_token,
};

/// Sends an SSE comment often enough to survive the idle read timeouts of nginx and of mobile
/// carrier NATs.
const KEEPALIVE_INTERVAL: Duration = Duration::from_secs(20);
/// Applies when the backend does not state a timeout. Generous: the browser may be
/// re-rendering a whole page before it can answer.
const DEFAULT_RPC_TIMEOUT: Duration = Duration::from_secs(60);
/// Caps whatever the backend asks for, so a bad request cannot pin a task and a pending entry
/// for hours.
const MAX_RPC_TIMEOUT: Duration = Duration::from_secs(600);
/// Caps the grace period a publisher may ask the bridge to wait for a reconnecting tab.
const MAX_CHANNEL_WAIT: Duration = Duration::from_secs(30);

pub struct BridgeState {
    pub registry: ChannelRegistry,
    /// Signs/verifies the browser's session token.
    pub secret_phrase: Vec<u8>,
    /// Authenticates the backend's service calls.
    pub internal_apikey: Vec<u8>,
    pub verbose_logs: bool,
    pub started_at: Instant,
}

/// Builds the router. Client endpoints get CORS because the browser calls them cross-origin
/// from the app's own domain; the service endpoints are only ever called server-to-server.
pub fn routes(state: Arc<BridgeState>) -> Router {
    Router::new()
        .route("/sse", get(handle_client_stream).options(handle_client_preflight))
        .route("/in", post(handle_client_inbound).options(handle_client_preflight))
        .route("/publish", post(handle_service_publish))
        .route("/rpc", post(handle_service_rpc))
        .route("/health", get(handle_health))
        .with_state(state)
}

#[derive(Deserialize)]
pub struct ChannelQuery {
    #[serde(default)]
    ch: String,
}

/// Answers a browser preflight. EventSource cannot set headers, so the frontend reads the
/// stream with `fetch()` and needs `Authorization` allowed explicitly.
async fn handle_client_preflight() -> Response {
    (StatusCode::NO_CONTENT, client_cors_headers()).into_response()
}

fn client_cors_headers() -> HeaderMap {
    let mut headers = HeaderMap::new();
    headers.insert(header::ACCESS_CONTROL_ALLOW_ORIGIN, HeaderValue::from_static("*"));
    headers.insert(
        header::ACCESS_CONTROL_ALLOW_METHODS,
        HeaderValue::from_static("GET, POST, OPTIONS"),
    );
    headers.insert(
        header::ACCESS_CONTROL_ALLOW_HEADERS,
        HeaderValue::from_static("Authorization, Content-Type"),
    );
    headers.insert(header::ACCESS_CONTROL_MAX_AGE, HeaderValue::from_static("86400"));
    headers
}

fn json_error(status: StatusCode, message: impl Into<String>) -> Response {
    (status, client_cors_headers(), Json(json!({ "Error": message.into() }))).into_response()
}

fn json_body(value: Value) -> Response {
    (StatusCode::OK, client_cors_headers(), Json(value)).into_response()
}

/// Turns a browser request into the channel it may address.
///
/// The channel token names the tab, but it does NOT prove who is asking: it is a plain
/// identifier anyone could rewrite. The session token is the proof, so the two are
/// cross-checked here. Without this a client could edit the company id inside its own token
/// and attach to another tenant's stream.
async fn resolve_client_channel(
    state: &BridgeState,
    headers: &HeaderMap,
    channel_token: &str,
) -> Result<String, String> {
    let channel_token = channel_token.trim();
    if channel_token.is_empty() {
        return Err("falta el parámetro ?ch=".to_owned());
    }
    let (channel_company_id, channel_user_id, _) =
        decode_channel_token(channel_token).map_err(|error| error.to_string())?;

    let authorization = headers.get(header::AUTHORIZATION).and_then(|value| value.to_str().ok());
    let user_token =
        authenticate_user(authorization, &state.secret_phrase).map_err(|error| error.to_string())?;

    if channel_company_id != user_token.company_id || channel_user_id != user_token.id {
        return Err("el canal solicitado no pertenece al usuario autenticado".to_owned());
    }
    Ok(channel_token.to_owned())
}

/// Opens the tab's event stream. It stays open until the client disconnects or another
/// connection claims the same channel.
async fn handle_client_stream(
    State(state): State<Arc<BridgeState>>,
    Query(query): Query<ChannelQuery>,
    headers: HeaderMap,
) -> Response {
    let channel_key = match resolve_client_channel(&state, &headers, &query.ch).await {
        Ok(channel_key) => channel_key,
        Err(reason) => {
            warn!(reason, "stream rejected");
            return json_error(StatusCode::UNAUTHORIZED, reason);
        }
    };

    let (channel, frame_receiver) = state.registry.open_channel(&channel_key).await;

    // Handshake, queued only after open_channel: a client that has seen this frame is
    // guaranteed to be routable, which is what lets the frontend delay its first turn until
    // the backend can actually reach it.
    if channel.send_frame(&json!({ "Type": "bridgeReady" })).await.is_err() {
        warn!(channel = channel_key, "handshake failed");
        state.registry.close_channel(&channel).await;
        return json_error(StatusCode::INTERNAL_SERVER_ERROR, "no se pudo iniciar el stream");
    }
    info!(channel = channel_key, "stream connected");

    let event_stream = ChannelEventStream {
        frames: frame_receiver,
        deregister_on_drop: DeregisterOnDrop { state: state.clone(), channel },
    };

    let mut response = Sse::new(event_stream)
        .keep_alive(KeepAlive::new().interval(KEEPALIVE_INTERVAL).text("ping"))
        .into_response();
    response.headers_mut().extend(client_cors_headers());
    // Defeat reverse-proxy response buffering (nginx honours this), otherwise every event is
    // held back until the buffer fills.
    response.headers_mut().insert("X-Accel-Buffering", HeaderValue::from_static("no"));
    response.headers_mut().insert(header::CACHE_CONTROL, HeaderValue::from_static("no-cache"));
    response
}

/// Deregisters the channel once its stream is gone.
///
/// This lives in a `Drop` impl rather than after the send loop because the common case is the
/// browser hanging up, which drops the response body mid-iteration — code placed after the
/// loop would simply never run, leaking the registry entry and leaving `/rpc` waiters parked
/// until their own timeouts.
struct DeregisterOnDrop {
    state: Arc<BridgeState>,
    channel: Arc<ClientChannel>,
}

impl Drop for DeregisterOnDrop {
    fn drop(&mut self) {
        let state = self.state.clone();
        let channel = self.channel.clone();
        // close_channel is async, and Drop is not: hand it to the runtime.
        tokio::spawn(async move {
            state.registry.close_channel(&channel).await;
            info!(channel = channel.key, "stream disconnected");
        });
    }
}

/// The tab's outbound frames as an SSE event stream, carrying the deregistration guard so it
/// is dropped exactly when the stream is.
struct ChannelEventStream {
    frames: mpsc::Receiver<String>,
    deregister_on_drop: DeregisterOnDrop,
}

impl Stream for ChannelEventStream {
    type Item = Result<Event, Infallible>;

    fn poll_next(mut self: Pin<&mut Self>, context: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        // Every field is Unpin, so the pin can be projected away safely.
        let _ = &self.deregister_on_drop;
        self.frames
            .poll_recv(context)
            .map(|frame| frame.map(|payload| Ok(Event::default().data(payload))))
    }
}

/// The browser→backend envelope. `ID > 0` marks it as the reply to a command the backend is
/// blocked on; `ID == 0` is an unsolicited event.
#[derive(Deserialize)]
pub struct ClientInboundMessage {
    #[serde(default, rename = "ID")]
    id: u64,
    #[serde(default, rename = "Type")]
    message_type: String,
    #[serde(default, rename = "Payload")]
    payload: Value,
}

/// Routes the browser's reply to the `/rpc` call waiting for it. Unsolicited events are
/// logged and dropped: the backend that would consume them is not connected to the bridge, it
/// only calls in.
async fn handle_client_inbound(
    State(state): State<Arc<BridgeState>>,
    Query(query): Query<ChannelQuery>,
    headers: HeaderMap,
    body: String,
) -> Response {
    let channel_key = match resolve_client_channel(&state, &headers, &query.ch).await {
        Ok(channel_key) => channel_key,
        Err(reason) => return json_error(StatusCode::UNAUTHORIZED, reason),
    };

    let inbound: ClientInboundMessage = match serde_json::from_str(&body) {
        Ok(inbound) => inbound,
        Err(error) => {
            return json_error(StatusCode::BAD_REQUEST, format!("cuerpo inválido: {error}"));
        }
    };

    if inbound.id == 0 {
        info!(
            channel = channel_key,
            message_type = inbound.message_type,
            "unsolicited event dropped"
        );
        return json_body(json!({ "Delivered": false }));
    }

    let Some(channel) = state.registry.find_channel(&channel_key).await else {
        warn!(channel = channel_key, id = inbound.id, "reply for a channel that is not connected");
        return json_body(json!({ "Delivered": false }));
    };

    let was_delivered = channel
        .deliver_reply(
            inbound.id,
            ReplyEnvelope { kind: inbound.message_type.clone(), payload: inbound.payload },
        )
        .await;
    if !was_delivered {
        warn!(
            channel = channel_key,
            id = inbound.id,
            message_type = inbound.message_type,
            "reply had no waiter"
        );
    } else if state.verbose_logs {
        debug!(channel = channel_key, id = inbound.id, "reply delivered");
    }
    json_body(json!({ "Delivered": was_delivered }))
}

/// Pushes one message to a tab without expecting an answer. `WaitMs` optionally waits for a
/// reconnecting tab instead of dropping right away.
#[derive(Deserialize)]
pub struct PublishRequest {
    #[serde(default, rename = "Channel")]
    channel: String,
    #[serde(default, rename = "Message")]
    message: Value,
    #[serde(default, rename = "WaitMs")]
    wait_ms: i64,
}

/// Pushes a command and blocks until the browser answers it. `ID` must match the id inside
/// `Message` — the browser echoes it back on `/in` and that is what the bridge correlates.
#[derive(Deserialize)]
pub struct RpcRequest {
    #[serde(default, rename = "Channel")]
    channel: String,
    #[serde(default, rename = "ID")]
    id: u64,
    #[serde(default, rename = "Message")]
    message: Value,
    #[serde(default, rename = "TimeoutMs")]
    timeout_ms: i64,
    #[serde(default, rename = "WaitMs")]
    wait_ms: i64,
}

fn current_unix_seconds() -> i64 {
    SystemTime::now().duration_since(UNIX_EPOCH).map_or(0, |elapsed| elapsed.as_secs() as i64)
}

fn verify_service_call(state: &BridgeState, headers: &HeaderMap) -> Result<(), Response> {
    let header_value = headers.get(SERVICE_AUTH_HEADER).and_then(|value| value.to_str().ok());
    verify_service_auth(header_value, &state.internal_apikey, current_unix_seconds()).map_err(
        |error| {
            warn!(error = %error, "service call rejected");
            json_error(StatusCode::UNAUTHORIZED, error.to_string())
        },
    )
}

/// Checks a backend call's target. The channel token is decoded (and thereby proven canonical)
/// before being used as a registry key, even though the backend is trusted: a malformed token
/// would otherwise create a channel entry nobody can ever connect to.
fn validate_service_target(channel_token: &str, message: &Value) -> Result<String, String> {
    let channel_token = channel_token.trim();
    if channel_token.is_empty() {
        return Err("Channel es obligatorio".to_owned());
    }
    decode_channel_token(channel_token).map_err(|error| error.to_string())?;
    if message.is_null() {
        return Err("Message es obligatorio".to_owned());
    }
    Ok(channel_token.to_owned())
}

/// Turns a millisecond field into a duration, applying a default when unset and never
/// exceeding the ceiling the bridge is willing to hold.
fn clamp_duration(milliseconds: i64, when_unset: Duration, maximum: Duration) -> Duration {
    if milliseconds <= 0 {
        return when_unset;
    }
    Duration::from_millis(milliseconds as u64).min(maximum)
}

async fn handle_service_publish(
    State(state): State<Arc<BridgeState>>,
    headers: HeaderMap,
    body: String,
) -> Response {
    if let Err(rejection) = verify_service_call(&state, &headers) {
        return rejection;
    }
    let publish: PublishRequest = match serde_json::from_str(&body) {
        Ok(publish) => publish,
        Err(error) => {
            return json_error(StatusCode::BAD_REQUEST, format!("cuerpo inválido: {error}"));
        }
    };
    let channel_key = match validate_service_target(&publish.channel, &publish.message) {
        Ok(channel_key) => channel_key,
        Err(reason) => return json_error(StatusCode::BAD_REQUEST, reason),
    };

    let wait = clamp_duration(publish.wait_ms, Duration::ZERO, MAX_CHANNEL_WAIT);
    let Some(channel) = state.registry.await_channel(&channel_key, wait).await else {
        // Not an error: with no buffering, a message for a disconnected tab is dropped by
        // contract and the backend only needs to know it happened.
        warn!(channel = channel_key, "publish with no connected channel");
        return json_body(json!({ "Delivered": false }));
    };

    match channel.send_frame(&publish.message).await {
        Ok(()) => {
            if state.verbose_logs {
                debug!(channel = channel_key, "publish delivered");
            }
            json_body(json!({ "Delivered": true }))
        }
        Err(send_error) => {
            let reason = match send_error {
                SendFrameError::Closed => "el stream del cliente ya está cerrado",
                SendFrameError::Timeout => "timeout escribiendo en el stream del cliente",
            };
            warn!(channel = channel_key, reason, "publish failed");
            json_body(json!({ "Delivered": false, "Error": reason }))
        }
    }
}

async fn handle_service_rpc(
    State(state): State<Arc<BridgeState>>,
    headers: HeaderMap,
    body: String,
) -> Response {
    if let Err(rejection) = verify_service_call(&state, &headers) {
        return rejection;
    }
    let rpc: RpcRequest = match serde_json::from_str(&body) {
        Ok(rpc) => rpc,
        Err(error) => {
            return json_error(StatusCode::BAD_REQUEST, format!("cuerpo inválido: {error}"));
        }
    };
    let channel_key = match validate_service_target(&rpc.channel, &rpc.message) {
        Ok(channel_key) => channel_key,
        Err(reason) => return json_error(StatusCode::BAD_REQUEST, reason),
    };
    if rpc.id == 0 {
        return json_error(
            StatusCode::BAD_REQUEST,
            "ID es obligatorio para correlacionar la respuesta",
        );
    }

    let wait = clamp_duration(rpc.wait_ms, Duration::ZERO, MAX_CHANNEL_WAIT);
    let Some(channel) = state.registry.await_channel(&channel_key, wait).await else {
        warn!(channel = channel_key, id = rpc.id, "rpc with no connected channel");
        return json_error(StatusCode::CONFLICT, "no hay ningún cliente conectado para ese tab");
    };

    // The waiter is registered *before* the command goes out, otherwise a very fast browser
    // could answer before there is anything to answer to.
    let reply_receiver = channel.await_reply(rpc.id).await;

    if let Err(send_error) = channel.send_frame(&rpc.message).await {
        channel.release_pending_reply(rpc.id).await;
        let reason = match send_error {
            SendFrameError::Closed => "el stream del cliente ya está cerrado",
            SendFrameError::Timeout => "timeout escribiendo en el stream del cliente",
        };
        warn!(channel = channel_key, id = rpc.id, reason, "rpc could not be sent");
        return json_error(StatusCode::BAD_GATEWAY, reason);
    }

    let reply_timeout = clamp_duration(rpc.timeout_ms, DEFAULT_RPC_TIMEOUT, MAX_RPC_TIMEOUT);
    let outcome = tokio::select! {
        // A dropped sender means the channel tore down without answering, which is a
        // disconnect as far as the caller is concerned.
        reply = reply_receiver => reply.map_or(RpcOutcome::Disconnected, RpcOutcome::Answered),
        () = channel.wait_closed() => RpcOutcome::Disconnected,
        () = tokio::time::sleep(reply_timeout) => RpcOutcome::TimedOut,
    };
    channel.release_pending_reply(rpc.id).await;

    match outcome {
        RpcOutcome::Answered(envelope) => {
            if state.verbose_logs {
                debug!(channel = channel_key, id = rpc.id, kind = envelope.kind, "rpc answered");
            }
            json_body(json!({ "Kind": envelope.kind, "Payload": envelope.payload }))
        }
        RpcOutcome::Disconnected => {
            warn!(channel = channel_key, id = rpc.id, "rpc aborted: the client disconnected");
            json_error(StatusCode::CONFLICT, "el cliente se desconectó antes de responder")
        }
        RpcOutcome::TimedOut => {
            warn!(channel = channel_key, id = rpc.id, ?reply_timeout, "rpc timed out");
            json_error(
                StatusCode::GATEWAY_TIMEOUT,
                format!("el cliente no respondió en {reply_timeout:?}"),
            )
        }
    }
}

/// Why a `/rpc` call stopped waiting. `StatusCode` cannot be used in a match pattern, and
/// naming the three outcomes documents the contract better than status codes would.
enum RpcOutcome {
    Answered(ReplyEnvelope),
    Disconnected,
    TimedOut,
}

async fn handle_health(State(state): State<Arc<BridgeState>>) -> Response {
    Json(json!({
        "Ok": true,
        "Channels": state.registry.connected_channel_count().await,
        "UptimeSeconds": state.started_at.elapsed().as_secs(),
    }))
    .into_response()
}

