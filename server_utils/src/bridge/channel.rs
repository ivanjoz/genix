//! One channel is one browser tab. Everything the bridge holds lives here and only here: a
//! queue of outbound SSE frames plus the table of replies the backend is currently blocked
//! on. Nothing is persisted — a message published while the tab is disconnected is dropped
//! by design (the client resynchronizes through the normal API).

use std::{
    collections::HashMap,
    sync::Arc,
    time::{Duration, Instant},
};

use tokio::sync::{Mutex, RwLock, mpsc, oneshot, watch};
use tracing::info;

/// Bounds how long a publisher waits for a wedged client whose outbound queue is full.
/// Giving up beats blocking the backend's turn forever.
const FRAME_SEND_TIMEOUT: Duration = Duration::from_secs(10);
/// Per-tab burst buffer. A turn emits status events far faster than a slow mobile
/// connection drains them.
const OUTBOUND_QUEUE_SIZE: usize = 64;

/// The browser's answer to one backend command, correlated by the command's id.
#[derive(Clone, Debug)]
pub struct ReplyEnvelope {
    /// "result" | "error"
    pub kind: String,
    pub payload: serde_json::Value,
}

#[derive(Debug, PartialEq, Eq)]
pub enum SendFrameError {
    /// The client's stream is already gone.
    Closed,
    /// The client is connected but not draining its queue.
    Timeout,
}

/// One connected tab. Its key is the channel token itself, which decoding proves to be a
/// canonical, one-to-one name for the company/user/tab triple — so two spellings can never
/// split one tab into two registry entries.
pub struct ClientChannel {
    pub key: String,
    /// Compact JSON payloads, not framed SSE text: the HTTP layer wraps each one in its
    /// `data:` frame. Dropping every sender is what ends the stream — the receiver first
    /// drains whatever is still queued (notably the "replaced" notice), so a client always
    /// learns why its stream ended.
    frame_sender: mpsc::Sender<String>,
    closed_sender: watch::Sender<bool>,
    pending_replies: Mutex<HashMap<u64, oneshot::Sender<ReplyEnvelope>>>,
    pub connected_at: Instant,
}

impl ClientChannel {
    fn new(key: String) -> (Arc<Self>, mpsc::Receiver<String>) {
        let (frame_sender, frame_receiver) = mpsc::channel(OUTBOUND_QUEUE_SIZE);
        let (closed_sender, _) = watch::channel(false);
        let channel = Arc::new(Self {
            key,
            frame_sender,
            closed_sender,
            pending_replies: Mutex::new(HashMap::new()),
            connected_at: Instant::now(),
        });
        (channel, frame_receiver)
    }

    /// Releases every task parked on this channel. Idempotent.
    ///
    /// `send_replace`, not `send`: the latter refuses to update the value while no receiver is
    /// subscribed, which is the normal state here (waiters subscribe on demand), and would
    /// leave the channel looking open forever.
    pub fn close(&self) {
        self.closed_sender.send_replace(true);
    }

    pub fn is_closed(&self) -> bool {
        *self.closed_sender.borrow() || self.frame_sender.is_closed()
    }

    /// Resolves once the channel is closed. Used by `/rpc` to abort as soon as the tab
    /// disconnects instead of waiting out its own timeout.
    pub async fn wait_closed(&self) {
        let mut closed_receiver = self.closed_sender.subscribe();
        while !*closed_receiver.borrow_and_update() {
            if closed_receiver.changed().await.is_err() {
                return;
            }
        }
    }

    /// Queues one message for the tab's stream.
    ///
    /// `Value::to_string` is compact by construction, which is what an SSE frame requires: a
    /// frame is terminated by a blank line, so an embedded newline would split one message
    /// into two malformed ones. The Go bridge had to call `json.Compact` and could fail on
    /// invalid JSON; here the type already guarantees both.
    pub async fn send_frame(&self, message: &serde_json::Value) -> Result<(), SendFrameError> {
        match tokio::time::timeout(
            FRAME_SEND_TIMEOUT,
            self.frame_sender.send(message.to_string()),
        )
        .await
        {
            Ok(Ok(())) => Ok(()),
            Ok(Err(_)) => Err(SendFrameError::Closed),
            Err(_) => Err(SendFrameError::Timeout),
        }
    }

    /// Queues a payload without waiting. Used for the eviction notice, which must not block
    /// the incoming connection that triggered it.
    fn try_send_payload(&self, payload: &str) {
        let _ = self.frame_sender.try_send(payload.to_owned());
    }

    /// Registers a waiter for a command id and returns where its reply will arrive.
    /// `release_pending_reply` must always run afterwards so an abandoned waiter (timeout,
    /// cancelled turn) does not leak.
    pub async fn await_reply(&self, command_id: u64) -> oneshot::Receiver<ReplyEnvelope> {
        let (reply_sender, reply_receiver) = oneshot::channel();
        self.pending_replies
            .lock()
            .await
            .insert(command_id, reply_sender);
        reply_receiver
    }

    pub async fn release_pending_reply(&self, command_id: u64) {
        self.pending_replies.lock().await.remove(&command_id);
    }

    /// Hands the browser's answer to whoever is waiting for it. False when nobody is (a late
    /// reply after a timeout, or a duplicate).
    pub async fn deliver_reply(&self, command_id: u64, envelope: ReplyEnvelope) -> bool {
        let Some(reply_sender) = self.pending_replies.lock().await.remove(&command_id) else {
            return false;
        };
        reply_sender.send(envelope).is_ok()
    }

    /// Unblocks every waiter with an error. Called when the tab disconnects: its commands can
    /// never be answered now, and the backend would otherwise sit until its own timeout.
    pub async fn fail_all_pending_replies(&self, reason: &str) -> usize {
        let pending_replies: Vec<_> = self.pending_replies.lock().await.drain().collect();
        let failed_count = pending_replies.len();
        for (_, reply_sender) in pending_replies {
            let _ = reply_sender.send(ReplyEnvelope {
                kind: "error".to_owned(),
                payload: serde_json::json!({ "Message": reason }),
            });
        }
        failed_count
    }
}

#[derive(Default)]
struct RegistryState {
    channels: HashMap<String, Arc<ClientChannel>>,
    arrival_waiters: HashMap<String, Vec<oneshot::Sender<()>>>,
}

/// Maps channel keys to connected tabs and lets a publisher wait for one that is still
/// connecting.
#[derive(Default)]
pub struct ChannelRegistry {
    state: RwLock<RegistryState>,
}

impl ChannelRegistry {
    pub fn new() -> Self {
        Self::default()
    }

    /// Installs a fresh channel for `key`, evicting any previous one.
    ///
    /// Last-connection-wins: the evicted stream is told why it is ending, so a duplicated tab
    /// (which clones sessionStorage and inherits the tab id) can mint a new id instead of the
    /// two endlessly kicking each other off.
    pub async fn open_channel(&self, key: &str) -> (Arc<ClientChannel>, mpsc::Receiver<String>) {
        let (channel, frame_receiver) = ClientChannel::new(key.to_owned());

        let (previous_channel, waiters_to_wake) = {
            let mut state = self.state.write().await;
            let previous_channel = state.channels.insert(key.to_owned(), channel.clone());
            let waiters_to_wake = state.arrival_waiters.remove(key).unwrap_or_default();
            (previous_channel, waiters_to_wake)
        };

        if let Some(previous_channel) = previous_channel {
            previous_channel.try_send_payload("{\"Type\":\"replaced\"}");
            previous_channel.close();
            previous_channel
                .fail_all_pending_replies("el cliente reconectó")
                .await;
            info!(channel = key, "channel replaced by a newer connection");
        }
        for waiter in waiters_to_wake {
            let _ = waiter.send(());
        }
        (channel, frame_receiver)
    }

    /// Removes `channel` from the registry, but only if it is still the installed one — a
    /// stream that unwinds late must not evict its successor.
    pub async fn close_channel(&self, channel: &Arc<ClientChannel>) {
        {
            let mut state = self.state.write().await;
            if state
                .channels
                .get(&channel.key)
                .is_some_and(|installed| Arc::ptr_eq(installed, channel))
            {
                state.channels.remove(&channel.key);
            }
        }
        channel.close();
        let failed_count = channel
            .fail_all_pending_replies("el cliente se desconectó")
            .await;
        if failed_count > 0 {
            tracing::warn!(
                channel = channel.key,
                failed_count,
                "channel closed with pending replies"
            );
        }
    }

    pub async fn find_channel(&self, key: &str) -> Option<Arc<ClientChannel>> {
        self.state.read().await.channels.get(key).cloned()
    }

    pub async fn connected_channel_count(&self) -> usize {
        self.state.read().await.channels.len()
    }

    /// Returns the channel for `key`, waiting up to `max_wait` for it to connect.
    ///
    /// This is the server-side half of the handshake: the client normally opens its stream
    /// before sending a turn, but a reconnect mid-turn can leave a short window where the
    /// backend has something to say and nobody to say it to.
    pub async fn await_channel(&self, key: &str, max_wait: Duration) -> Option<Arc<ClientChannel>> {
        if let Some(channel) = self.find_channel(key).await {
            return Some(channel);
        }
        if max_wait.is_zero() {
            return None;
        }

        let arrival_receiver = {
            let mut state = self.state.write().await;
            // Re-check under the write lock: the channel may have opened between the lookup
            // above and here, which would leave a waiter nobody ever wakes.
            if let Some(channel) = state.channels.get(key) {
                return Some(channel.clone());
            }
            let (arrival_sender, arrival_receiver) = oneshot::channel();
            state
                .arrival_waiters
                .entry(key.to_owned())
                .or_default()
                .push(arrival_sender);
            arrival_receiver
        };

        match tokio::time::timeout(max_wait, arrival_receiver).await {
            Ok(Ok(())) => self.find_channel(key).await,
            // Timed out, or the waiter was dropped by a newer eviction: either way the
            // registry entry is stale and has to go.
            _ => {
                self.drop_arrival_waiters(key).await;
                self.find_channel(key).await
            }
        }
    }

    /// Drops the waiter list for `key`. Senders are identified by the key alone: a waiter that
    /// gave up cannot be told apart from another by value, and a spurious extra wake-up is
    /// harmless (the woken task re-reads the registry).
    async fn drop_arrival_waiters(&self, key: &str) {
        let mut state = self.state.write().await;
        if let Some(waiters) = state.arrival_waiters.get_mut(key) {
            waiters.retain(|waiter| !waiter.is_closed());
            if waiters.is_empty() {
                state.arrival_waiters.remove(key);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const TEST_KEY: &str = "Byo3bFBobzE";

    #[tokio::test]
    async fn a_published_frame_reaches_the_connected_tab() {
        let registry = ChannelRegistry::new();
        let (channel, mut frames) = registry.open_channel(TEST_KEY).await;

        channel
            .send_frame(&serde_json::json!({ "Type": "agentStatus" }))
            .await
            .unwrap();
        assert_eq!(frames.recv().await.unwrap(), "{\"Type\":\"agentStatus\"}");
    }

    #[tokio::test]
    async fn a_reconnect_replaces_the_previous_stream_and_tells_it_why() {
        let registry = ChannelRegistry::new();
        let (first_channel, mut first_frames) = registry.open_channel(TEST_KEY).await;
        let (second_channel, mut second_frames) = registry.open_channel(TEST_KEY).await;

        // The evicted stream learns why it ended, then its stream terminates.
        assert_eq!(
            first_frames.recv().await.unwrap(),
            "{\"Type\":\"replaced\"}"
        );
        assert!(first_channel.is_closed());

        // Only the newest channel is routable, and it still works.
        assert!(Arc::ptr_eq(
            &registry.find_channel(TEST_KEY).await.unwrap(),
            &second_channel
        ));
        second_channel
            .send_frame(&serde_json::json!({ "Type": "agentStatus" }))
            .await
            .unwrap();
        assert_eq!(
            second_frames.recv().await.unwrap(),
            "{\"Type\":\"agentStatus\"}"
        );
    }

    /// A stream that unwinds after its successor took over must not evict the successor.
    #[tokio::test]
    async fn a_late_close_does_not_evict_the_successor() {
        let registry = ChannelRegistry::new();
        let (first_channel, _first_frames) = registry.open_channel(TEST_KEY).await;
        let (second_channel, _second_frames) = registry.open_channel(TEST_KEY).await;

        registry.close_channel(&first_channel).await;

        assert!(Arc::ptr_eq(
            &registry.find_channel(TEST_KEY).await.unwrap(),
            &second_channel
        ));
        assert_eq!(registry.connected_channel_count().await, 1);
    }

    #[tokio::test]
    async fn a_reply_is_correlated_by_command_id() {
        let registry = ChannelRegistry::new();
        let (channel, _frames) = registry.open_channel(TEST_KEY).await;

        let reply_receiver = channel.await_reply(9).await;
        assert!(
            channel
                .deliver_reply(
                    9,
                    ReplyEnvelope {
                        kind: "result".to_owned(),
                        payload: serde_json::json!({ "Route": "/negocio/productos" }),
                    }
                )
                .await
        );

        let envelope = reply_receiver.await.unwrap();
        assert_eq!(envelope.kind, "result");
        assert_eq!(envelope.payload["Route"], "/negocio/productos");
    }

    #[tokio::test]
    async fn a_reply_nobody_waits_for_is_reported_undelivered() {
        let registry = ChannelRegistry::new();
        let (channel, _frames) = registry.open_channel(TEST_KEY).await;

        let envelope = ReplyEnvelope {
            kind: "result".to_owned(),
            payload: serde_json::Value::Null,
        };
        // Never registered, and then again after the waiter was released: both are drops.
        assert!(!channel.deliver_reply(9, envelope.clone()).await);
        drop(channel.await_reply(9).await);
        channel.release_pending_reply(9).await;
        assert!(!channel.deliver_reply(9, envelope).await);
    }

    #[tokio::test]
    async fn a_disconnect_fails_every_pending_reply() {
        let registry = ChannelRegistry::new();
        let (channel, _frames) = registry.open_channel(TEST_KEY).await;
        let first_reply = channel.await_reply(1).await;
        let second_reply = channel.await_reply(2).await;

        registry.close_channel(&channel).await;

        for reply_receiver in [first_reply, second_reply] {
            let envelope = reply_receiver.await.unwrap();
            assert_eq!(envelope.kind, "error");
            assert!(
                envelope.payload["Message"]
                    .as_str()
                    .unwrap()
                    .contains("desconect")
            );
        }
        channel.wait_closed().await; // already closed: must return immediately
    }

    #[tokio::test]
    async fn sending_to_a_closed_channel_reports_the_stream_is_gone() {
        let registry = ChannelRegistry::new();
        let (channel, frames) = registry.open_channel(TEST_KEY).await;
        drop(frames); // the browser went away

        assert_eq!(
            channel
                .send_frame(&serde_json::json!({ "Type": "agentStatus" }))
                .await,
            Err(SendFrameError::Closed)
        );
    }

    #[tokio::test]
    async fn await_channel_returns_immediately_when_already_connected() {
        let registry = ChannelRegistry::new();
        let (channel, _frames) = registry.open_channel(TEST_KEY).await;

        let awaited = registry
            .await_channel(TEST_KEY, Duration::from_millis(0))
            .await
            .unwrap();
        assert!(Arc::ptr_eq(&awaited, &channel));
    }

    #[tokio::test]
    async fn await_channel_waits_for_a_reconnecting_tab() {
        let registry = Arc::new(ChannelRegistry::new());

        let connecting_registry = registry.clone();
        let late_connect = tokio::spawn(async move {
            tokio::time::sleep(Duration::from_millis(50)).await;
            connecting_registry.open_channel(TEST_KEY).await
        });

        // The publisher is already waiting when the tab shows up.
        let awaited = registry
            .await_channel(TEST_KEY, Duration::from_secs(3))
            .await;
        assert!(awaited.is_some());
        let (channel, _frames) = late_connect.await.unwrap();
        assert!(Arc::ptr_eq(&awaited.unwrap(), &channel));
    }

    #[tokio::test]
    async fn await_channel_gives_up_after_the_wait_and_leaves_no_waiter_behind() {
        let registry = ChannelRegistry::new();

        assert!(
            registry
                .await_channel(TEST_KEY, Duration::from_millis(30))
                .await
                .is_none()
        );
        assert!(registry.state.read().await.arrival_waiters.is_empty());
        // A zero wait drops immediately, which is the no-buffering contract.
        assert!(
            registry
                .await_channel(TEST_KEY, Duration::from_millis(0))
                .await
                .is_none()
        );
    }
}
