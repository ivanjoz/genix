//! End-to-end coverage of the bridge's HTTP contract.
//!
//! These walk the sequence the real system uses: the browser opens `/sse` and gets the
//! handshake, the backend pushes an event with `/publish`, issues a blocking command with
//! `/rpc`, and the browser answers on `/in` to unblock it.

use std::{sync::Arc, time::Instant};

use axum::{
    Router,
    body::{Body, BodyDataStream},
    http::{Request, StatusCode, header},
    response::Response,
};
use base64::{Engine, engine::general_purpose};
use genix_server_utils::bridge::{
    auth::{SERVICE_AUTH_HEADER, make_service_auth_header},
    channel::ChannelRegistry,
    http::{BridgeState, routes},
};
use serde_json::{Value, json};
use tokio_stream::StreamExt;
use tower::ServiceExt;

/// Both auth schemes are keyed with the same value here so one constant covers the session
/// token (`secret_phrase` in production) and service calls (`internal_apikey`). In a real
/// deployment these are two distinct secrets.
const TEST_SECRET: &[u8] = b"K1OzWIN0yarCc9ge";

/// A colbin session token for company 7 / user 42 / "tester", produced by the Go
/// `colbin.Marshal` + `computeUserTokenHash` with TEST_SECRET.
const GO_SESSION_TOKEN_HEX: &str = "010105ca000600000001350029000000019f00d1040000011a68ed671d93b9\
                                    3189b000000000000000006a02000500000001746573746572";

/// Channel token for company 7 / user 42 / tab "N2xQaG8x", pinned by the cross-language vectors.
const TEST_CHANNEL: &str = "Byo3bFBobzE";
/// Same tab, different company: a channel this identity may not address.
const OTHER_COMPANY_CHANNEL: &str = "CCo3bFBobzE";

fn test_router() -> Router {
    routes(Arc::new(BridgeState {
        registry: ChannelRegistry::new(),
        secret_phrase: TEST_SECRET.to_vec(),
        internal_apikey: TEST_SECRET.to_vec(),
        verbose_logs: false,
        started_at: Instant::now(),
    }))
}

fn session_token() -> String {
    let compact: String = GO_SESSION_TOKEN_HEX
        .chars()
        .filter(|character| !character.is_whitespace())
        .collect();
    let bytes: Vec<u8> = (0..compact.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&compact[index..index + 2], 16).unwrap())
        .collect();
    general_purpose::STANDARD.encode(bytes)
}

/// Opens a tab's stream and consumes the handshake, leaving the body ready for events.
async fn open_stream(router: &Router, channel_token: &str) -> BodyDataStream {
    let response = router
        .clone()
        .oneshot(
            Request::get(format!("/sse?ch={channel_token}"))
                .header(header::AUTHORIZATION, format!("Bearer {}", session_token()))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();
    assert_eq!(
        response.status(),
        StatusCode::OK,
        "stream should have opened"
    );

    let mut frames = response.into_body().into_data_stream();
    // The handshake proves the channel is registered; the real client waits for it before
    // sending a turn, and so must every test before publishing.
    assert_eq!(next_frame(&mut frames).await["Type"], "bridgeReady");
    frames
}

/// Reads the next `data:` frame off an SSE body, ignoring keepalive comments.
async fn next_frame(frames: &mut BodyDataStream) -> Value {
    let mut buffered = String::new();
    loop {
        for line in std::mem::take(&mut buffered).lines() {
            if let Some(payload) = line.strip_prefix("data: ")
                && let Ok(frame) = serde_json::from_str::<Value>(payload)
            {
                return frame;
            }
        }
        let chunk = tokio::time::timeout(std::time::Duration::from_secs(3), frames.next())
            .await
            .expect("timed out waiting for an SSE frame")
            .expect("stream closed before the expected frame")
            .expect("stream errored");
        buffered.push_str(&String::from_utf8_lossy(&chunk));
    }
}

/// Issues one authenticated backend→bridge call.
async fn post_service(router: &Router, path: &str, body: Value) -> (StatusCode, Value) {
    let request = Request::post(path)
        .header(
            SERVICE_AUTH_HEADER,
            make_service_auth_header(TEST_SECRET, current_unix_seconds()),
        )
        .header(header::CONTENT_TYPE, "application/json")
        .body(Body::from(body.to_string()))
        .unwrap();
    read_json(router.clone().oneshot(request).await.unwrap()).await
}

async fn read_json(response: Response) -> (StatusCode, Value) {
    let status = response.status();
    let body = axum::body::to_bytes(response.into_body(), 64 * 1024)
        .await
        .unwrap();
    (status, serde_json::from_slice(&body).unwrap_or(Value::Null))
}

fn current_unix_seconds() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        .cast_signed()
}

#[tokio::test]
async fn publish_reaches_the_connected_tab() {
    let router = test_router();
    let mut frames = open_stream(&router, TEST_CHANNEL).await;

    let (status, body) = post_service(
        &router,
        "/publish",
        json!({
            "Channel": TEST_CHANNEL,
            "Message": { "Type": "agentStatus", "Payload": { "State": "thinking" } },
        }),
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["Delivered"], true);
    assert_eq!(next_frame(&mut frames).await["Type"], "agentStatus");
}

#[tokio::test]
async fn publish_to_a_disconnected_tab_is_dropped() {
    let router = test_router();

    let (status, body) = post_service(
        &router,
        "/publish",
        json!({ "Channel": TEST_CHANNEL, "Message": { "Type": "agentStatus" } }),
    )
    .await;

    // Dropping is the contract, so this is a 200 reporting non-delivery — not an error the
    // backend has to handle.
    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["Delivered"], false);
}

#[tokio::test]
async fn publish_from_another_tenant_cannot_reach_the_tab() {
    let router = test_router();
    let _frames = open_stream(&router, TEST_CHANNEL).await;

    // Same tab id, different company: the channel key differs, so there is nothing to
    // deliver to.
    let (_, body) = post_service(
        &router,
        "/publish",
        json!({ "Channel": OTHER_COMPANY_CHANNEL, "Message": { "Type": "agentStatus" } }),
    )
    .await;

    assert_eq!(body["Delivered"], false);
}

/// The channel token is an identifier, not a credential: editing the company id inside it must
/// not open another tenant's channel. The session token is the proof and the two must agree.
#[tokio::test]
async fn a_client_cannot_open_a_channel_of_another_identity() {
    let router = test_router();

    let response = router
        .oneshot(
            Request::get(format!("/sse?ch={OTHER_COMPANY_CHANNEL}"))
                .header(header::AUTHORIZATION, format!("Bearer {}", session_token()))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
}

#[tokio::test]
async fn an_unauthenticated_client_is_rejected() {
    let router = test_router();

    let response = router
        .oneshot(
            Request::get(format!("/sse?ch={TEST_CHANNEL}"))
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
}

#[tokio::test]
async fn an_unsigned_service_call_is_rejected() {
    let router = test_router();

    let response = router
        .oneshot(
            Request::post("/publish")
                .body(Body::from(
                    json!({ "Channel": TEST_CHANNEL, "Message": {} }).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
}

#[tokio::test]
async fn rpc_correlates_the_browser_reply() {
    let router = test_router();
    let mut frames = open_stream(&router, TEST_CHANNEL).await;

    // /rpc blocks until the browser answers, so it has to run concurrently with the reply.
    let rpc_router = router.clone();
    let rpc_call = tokio::spawn(async move {
        post_service(
            &rpc_router,
            "/rpc",
            json!({
                "Channel": TEST_CHANNEL,
                "ID": 9,
                "Message": { "ID": 9, "Type": "navigate", "Payload": { "Route": "/negocio/productos" } },
                "TimeoutMs": 3000,
            }),
        )
        .await
    });

    let command = next_frame(&mut frames).await;
    assert_eq!(command["Type"], "navigate");
    assert_eq!(command["ID"], 9);

    // The browser answers on its own short request, which is what unblocks /rpc.
    let reply = router
        .oneshot(
            Request::post(format!("/in?ch={TEST_CHANNEL}"))
                .header(header::AUTHORIZATION, format!("Bearer {}", session_token()))
                .body(Body::from(
                    json!({ "ID": 9, "Type": "result", "Payload": { "Route": "/negocio/productos" } })
                        .to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();
    let (reply_status, reply_body) = read_json(reply).await;
    assert_eq!(reply_status, StatusCode::OK);
    assert_eq!(reply_body["Delivered"], true);

    let (rpc_status, rpc_body) = rpc_call.await.unwrap();
    assert_eq!(rpc_status, StatusCode::OK);
    assert_eq!(rpc_body["Kind"], "result");
    assert_eq!(rpc_body["Payload"]["Route"], "/negocio/productos");
}

#[tokio::test]
async fn rpc_without_a_connected_tab_fails() {
    let router = test_router();

    let (status, _) = post_service(
        &router,
        "/rpc",
        json!({ "Channel": TEST_CHANNEL, "ID": 1, "Message": { "ID": 1, "Type": "navigate" } }),
    )
    .await;

    assert_eq!(status, StatusCode::CONFLICT);
}

#[tokio::test]
async fn rpc_times_out_when_the_browser_stays_silent() {
    let router = test_router();
    let _frames = open_stream(&router, TEST_CHANNEL).await;

    let (status, _) = post_service(
        &router,
        "/rpc",
        json!({
            "Channel": TEST_CHANNEL,
            "ID": 5,
            "Message": { "ID": 5, "Type": "getMenu" },
            "TimeoutMs": 200,
        }),
    )
    .await;

    assert_eq!(status, StatusCode::GATEWAY_TIMEOUT);
}

#[tokio::test]
async fn rpc_requires_an_id_to_correlate_the_reply() {
    let router = test_router();
    let _frames = open_stream(&router, TEST_CHANNEL).await;

    let (status, _) = post_service(
        &router,
        "/rpc",
        json!({ "Channel": TEST_CHANNEL, "Message": { "Type": "getMenu" } }),
    )
    .await;

    assert_eq!(status, StatusCode::BAD_REQUEST);
}

#[tokio::test]
async fn a_reconnect_replaces_the_previous_stream() {
    let router = test_router();
    let mut first_frames = open_stream(&router, TEST_CHANNEL).await;
    let mut second_frames = open_stream(&router, TEST_CHANNEL).await;

    // The evicted stream is told why it ended, so a duplicated tab can rotate its id instead
    // of the two fighting over it.
    assert_eq!(next_frame(&mut first_frames).await["Type"], "replaced");

    let (_, body) = post_service(
        &router,
        "/publish",
        json!({ "Channel": TEST_CHANNEL, "Message": { "Type": "agentStatus" } }),
    )
    .await;
    assert_eq!(body["Delivered"], true);
    assert_eq!(next_frame(&mut second_frames).await["Type"], "agentStatus");
}

#[tokio::test]
async fn publish_waits_for_a_reconnecting_tab() {
    let router = test_router();

    // The tab connects only after the publish is already waiting — the safety net behind the
    // client handshake.
    let connecting_router = router.clone();
    let late_connect = tokio::spawn(async move {
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;
        open_stream(&connecting_router, TEST_CHANNEL).await
    });

    let (_, body) = post_service(
        &router,
        "/publish",
        json!({
            "Channel": TEST_CHANNEL,
            "Message": { "Type": "agentStatus" },
            "WaitMs": 3000,
        }),
    )
    .await;

    assert_eq!(body["Delivered"], true);
    let mut frames = late_connect.await.unwrap();
    assert_eq!(next_frame(&mut frames).await["Type"], "agentStatus");
}

#[tokio::test]
async fn an_unsolicited_client_event_is_dropped() {
    let router = test_router();
    let _frames = open_stream(&router, TEST_CHANNEL).await;

    // ID 0 means "not a reply to anything"; the backend is not connected to the bridge, so
    // there is nobody to forward it to.
    let response = router
        .oneshot(
            Request::post(format!("/in?ch={TEST_CHANNEL}"))
                .header(header::AUTHORIZATION, format!("Bearer {}", session_token()))
                .body(Body::from(
                    json!({ "ID": 0, "Type": "somethingElse" }).to_string(),
                ))
                .unwrap(),
        )
        .await
        .unwrap();

    let (status, body) = read_json(response).await;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["Delivered"], false);
}

#[tokio::test]
async fn health_reports_the_connected_channel_count() {
    let router = test_router();

    let (status, body) = read_json(
        router
            .clone()
            .oneshot(Request::get("/health").body(Body::empty()).unwrap())
            .await
            .unwrap(),
    )
    .await;
    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["Ok"], true);
    assert_eq!(body["Channels"], 0);

    let _frames = open_stream(&router, TEST_CHANNEL).await;
    let (_, body) = read_json(
        router
            .oneshot(Request::get("/health").body(Body::empty()).unwrap())
            .await
            .unwrap(),
    )
    .await;
    assert_eq!(body["Channels"], 1);
}

/// The reason this whole process exists: events must reach the browser *as they happen*.
///
/// Every other test drives the router in-process, which cannot catch response buffering. This
/// one runs the real `axum::serve` over a real socket and reads the handshake **before**
/// publishing anything — if the response were buffered until completion, that first read would
/// block and this test would time out rather than fail vaguely in production.
#[tokio::test]
async fn events_are_flushed_incrementally_over_a_real_socket() {
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let server_address = listener.local_addr().unwrap();
    let router = test_router();
    tokio::spawn(axum::serve(listener, router.clone()).into_future());

    let mut stream_socket = tokio::net::TcpStream::connect(server_address)
        .await
        .unwrap();
    stream_socket
        .write_all(
            format!(
                "GET /sse?ch={TEST_CHANNEL} HTTP/1.1\r\nHost: localhost\r\n\
                 Authorization: Bearer {}\r\n\r\n",
                session_token()
            )
            .as_bytes(),
        )
        .await
        .unwrap();

    // Read until the handshake arrives. Nothing has been published yet, so this can only
    // succeed if the server flushed the SSE frame instead of holding the response open.
    let mut received = Vec::new();
    let mut read_buffer = [0_u8; 2048];
    let handshake_deadline = std::time::Duration::from_secs(3);
    while !String::from_utf8_lossy(&received).contains("bridgeReady") {
        let read_count =
            tokio::time::timeout(handshake_deadline, stream_socket.read(&mut read_buffer))
                .await
                .expect("timed out: the SSE response was buffered instead of flushed")
                .unwrap();
        assert_ne!(
            read_count, 0,
            "server closed the stream before the handshake"
        );
        received.extend_from_slice(&read_buffer[..read_count]);
    }

    let handshake_text = String::from_utf8_lossy(&received).to_string();
    assert!(
        handshake_text.starts_with("HTTP/1.1 200 OK"),
        "got {handshake_text}"
    );
    assert!(
        handshake_text.contains("text/event-stream"),
        "got {handshake_text}"
    );
    // The header nginx needs in order not to buffer the stream on its own.
    assert!(
        handshake_text.contains("x-accel-buffering: no"),
        "got {handshake_text}"
    );

    // Now push an event and confirm it arrives on the already-open connection.
    let (_, body) = post_service(
        &router,
        "/publish",
        json!({ "Channel": TEST_CHANNEL, "Message": { "Type": "agentStatus" } }),
    )
    .await;
    assert_eq!(body["Delivered"], true);

    let mut delivered = String::new();
    while !delivered.contains("agentStatus") {
        let read_count =
            tokio::time::timeout(handshake_deadline, stream_socket.read(&mut read_buffer))
                .await
                .expect("timed out waiting for the published event")
                .unwrap();
        assert_ne!(
            read_count, 0,
            "server closed the stream before delivering the event"
        );
        delivered.push_str(&String::from_utf8_lossy(&read_buffer[..read_count]));
    }
    assert!(
        delivered.contains("data: {\"Type\":\"agentStatus\"}"),
        "got {delivered}"
    );
}

/// Dropping the browser's socket must deregister the channel, or the registry leaks entries and
/// `/rpc` waiters stay parked. This is the `DeregisterOnDrop` guard's contract.
#[tokio::test]
async fn a_disconnected_socket_deregisters_the_channel() {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let server_address = listener.local_addr().unwrap();
    let router = test_router();
    tokio::spawn(axum::serve(listener, router.clone()).into_future());

    {
        use tokio::io::{AsyncReadExt, AsyncWriteExt};
        let mut stream_socket = tokio::net::TcpStream::connect(server_address)
            .await
            .unwrap();
        stream_socket
            .write_all(
                format!(
                    "GET /sse?ch={TEST_CHANNEL} HTTP/1.1\r\nHost: localhost\r\n\
                     Authorization: Bearer {}\r\n\r\n",
                    session_token()
                )
                .as_bytes(),
            )
            .await
            .unwrap();

        let mut read_buffer = [0_u8; 2048];
        let mut received = Vec::new();
        while !String::from_utf8_lossy(&received).contains("bridgeReady") {
            let read_count = tokio::time::timeout(
                std::time::Duration::from_secs(3),
                stream_socket.read(&mut read_buffer),
            )
            .await
            .expect("timed out waiting for the handshake")
            .unwrap();
            received.extend_from_slice(&read_buffer[..read_count]);
        }

        let (_, body) = read_json(
            router
                .clone()
                .oneshot(Request::get("/health").body(Body::empty()).unwrap())
                .await
                .unwrap(),
        )
        .await;
        assert_eq!(
            body["Channels"], 1,
            "the channel should be registered while connected"
        );
    } // the socket is dropped here

    // Deregistration is spawned from a Drop impl, so poll briefly instead of assuming it
    // has already run.
    for _ in 0..60 {
        tokio::time::sleep(std::time::Duration::from_millis(50)).await;
        let (_, body) = read_json(
            router
                .clone()
                .oneshot(Request::get("/health").body(Body::empty()).unwrap())
                .await
                .unwrap(),
        )
        .await;
        if body["Channels"] == 0 {
            return;
        }
    }
    panic!("the channel was never deregistered after the client disconnected");
}

#[tokio::test]
async fn a_client_preflight_allows_the_authorization_header() {
    let router = test_router();

    // EventSource cannot set headers, so the frontend reads the stream with fetch() and needs
    // Authorization allowed explicitly.
    let response = router
        .oneshot(Request::options("/sse").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::NO_CONTENT);
    let allowed_headers = response.headers()[header::ACCESS_CONTROL_ALLOW_HEADERS]
        .to_str()
        .unwrap();
    assert!(
        allowed_headers.contains("Authorization"),
        "got {allowed_headers}"
    );
}
