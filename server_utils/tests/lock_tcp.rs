//! End-to-end lock behavior over a real socket.
//!
//! The registry's own tests cover the queueing rules in isolation. What can only be checked
//! here is the part that makes the design safe: ownership is bound to the TCP connection, so a
//! client that disconnects or goes silent loses its lock without any sweeper running.

use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use anyhow::Result;
use async_trait::async_trait;
use genix_server_utils::{
    limiter::{
        aggregation::UsageKey,
        credits_blob::Credits,
        protocol::CHARGE_PAYLOAD_SIZE,
        quota::{CreditLimits, LimitPolicy, RateLimiter, ScopeLimits},
        storage::{
            LimiterStore, StoredBudget, StoredBudgetRow, StoredBudgetUsage, StoredUsage,
            StoredUserAccess,
        },
        time_frame,
    },
    lock::registry::{LockLimits, LockRegistry},
    reqlog::writer::RequestLogSink,
    service::server,
};
use hmac::{Hmac, Mac};
use sha2::Sha256;
use tokio::{
    io::{AsyncReadExt, AsyncWriteExt},
    net::{TcpListener, TcpStream},
    sync::watch,
    time::timeout,
};

const SECRET: &[u8] = b"lock-test-secret";
const ACTION: u16 = 7;

/// The limiter is not under test here, but `server::run` needs one, and it must not touch a
/// database.
#[derive(Default)]
struct EmptyStore;

#[async_trait]
impl LimiterStore for EmptyStore {
    async fn load_exact(&self, _key: UsageKey) -> Result<Option<StoredUsage>> {
        Ok(None)
    }
    async fn load_range(&self, _c: i32, _u: i32, _s: i32, _e: i32) -> Result<Vec<StoredUsage>> {
        Ok(Vec::new())
    }
    async fn upsert(&self, _key: UsageKey, _used: Vec<u8>) -> Result<()> {
        Ok(())
    }
    async fn load_budget(&self, company_id: i32) -> Result<Option<StoredBudgetRow>> {
        let unix_seconds = SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs() as i64;
        let unlimited = Credits {
            cpu: i64::MAX as u64,
            inference: i64::MAX as u64,
        };
        let budget = StoredBudget {
            company_id,
            daily: unlimited,
            budget_month_start_day: time_frame::month_start_day(unix_seconds)?,
            monthly_ceiling: unlimited,
            last_set: unlimited,
            updated: 0,
        };
        Ok(Some(StoredBudgetRow {
            budget,
            ..Default::default()
        }))
    }
    async fn upsert_budget(&self, _budget: StoredBudget) -> Result<()> {
        Ok(())
    }
    async fn upsert_budget_usage(&self, _usage: StoredBudgetUsage) -> Result<()> {
        Ok(())
    }
    async fn load_user_access(&self, _c: i32, _u: i32) -> Result<Option<StoredUserAccess>> {
        Ok(None)
    }
}

struct TestServer {
    address: String,
    _shutdown: watch::Sender<bool>,
}

async fn start_server(frame_timeout: Duration) -> TestServer {
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap().to_string();
    let limits = CreditLimits {
        ten_seconds: 1_000,
        hour: 10_000,
    };
    let scope = ScopeLimits {
        cpu: limits,
        inference: limits,
    };
    let limiter = Arc::new(RateLimiter::new(
        2,
        LimitPolicy {
            company: scope,
            user: scope,
            company_extra_daily_cpu: 0,
        },
        Arc::new(EmptyStore),
        600,
    ));
    let locks = Arc::new(LockRegistry::new(
        2,
        LockLimits {
            max_keys: 64,
            max_total_waiters: 64,
            max_lease: Duration::from_secs(60),
        },
    ));
    let (shutdown_sender, shutdown_receiver) = watch::channel(false);
    tokio::spawn(server::run(
        listener,
        limiter,
        locks,
        // These tests drive locks and charges; request logs would be written to a database this
        // harness does not have, so the sink accepts and discards.
        RequestLogSink::disabled(),
        Arc::new(SECRET.to_vec()),
        frame_timeout,
        64,
        16,
        shutdown_receiver,
    ));
    TestServer {
        address,
        _shutdown: shutdown_sender,
    }
}

/// One client connection, which is also one lock slot.
struct Client {
    socket: TcpStream,
    nonce: [u8; 8],
    sequence: u64,
}

impl Client {
    async fn connect(server: &TestServer) -> Self {
        let mut socket = TcpStream::connect(&server.address).await.unwrap();
        let mut nonce = [0_u8; 8];
        socket.read_exact(&mut nonce).await.unwrap();
        Self {
            socket,
            nonce,
            sequence: 0,
        }
    }

    /// Writes a frame without waiting for its reply, and returns the correlation to expect.
    async fn write_frame(&mut self, opcode: u8, payload: &[u8]) -> u16 {
        let mut frame = vec![opcode];
        frame.extend_from_slice(payload);
        let mut mac = Hmac::<Sha256>::new_from_slice(SECRET).unwrap();
        mac.update(b"genix-server-utils:v6");
        mac.update(&self.nonce);
        mac.update(&self.sequence.to_be_bytes());
        mac.update(&frame);
        frame.extend_from_slice(&mac.finalize().into_bytes()[..8]);
        let correlation = self.sequence as u16;
        self.sequence += 1;
        self.socket.write_all(&frame).await.unwrap();
        correlation
    }

    /// Returns `(correlation, status, detail)` of whichever reply arrives next.
    async fn read_reply(&mut self) -> (u16, u8, u16) {
        let mut reply = [0_u8; 5];
        self.socket.read_exact(&mut reply).await.unwrap();
        (
            u16::from_be_bytes([reply[0], reply[1]]),
            reply[2],
            u16::from_be_bytes([reply[3], reply[4]]),
        )
    }

    async fn send(&mut self, opcode: u8, payload: &[u8]) -> u8 {
        let expected = self.write_frame(opcode, payload).await;
        let (correlation, status, _) = self.read_reply().await;
        assert_eq!(
            correlation, expected,
            "reply correlated to the wrong request"
        );
        status
    }

    /// A minimal, always-admissible charge: opcode 0x01 for company 1 / user 1 on route 1, with the
    /// four authorization slots left empty so nothing is asked of the (empty) access store.
    fn charge_payload() -> Vec<u8> {
        let mut payload = Vec::with_capacity(CHARGE_PAYLOAD_SIZE);
        payload.extend_from_slice(&[0, 0, 1]);
        payload.extend_from_slice(&[0, 0, 1]);
        payload.extend_from_slice(&1_u16.to_be_bytes());
        payload.extend_from_slice(&1_u16.to_be_bytes());
        payload.extend_from_slice(&0_u16.to_be_bytes());
        payload.resize(CHARGE_PAYLOAD_SIZE, 0);
        payload
    }

    fn budget_payload(operation: u8, cpu: u64, inference: u64) -> Vec<u8> {
        let mut payload = Vec::with_capacity(20);
        payload.extend_from_slice(&[0, 0, 1]);
        payload.push(operation);
        payload.extend_from_slice(&cpu.to_be_bytes());
        payload.extend_from_slice(&inference.to_be_bytes());
        payload
    }

    fn acquire_payload(identifier: i64, max_waiters: u8, wait_ms: u16, lease_ms: u16) -> Vec<u8> {
        let mut payload = Vec::with_capacity(15);
        payload.extend_from_slice(&ACTION.to_be_bytes());
        payload.extend_from_slice(&identifier.to_be_bytes());
        payload.push(max_waiters);
        payload.extend_from_slice(&wait_ms.to_be_bytes());
        payload.extend_from_slice(&lease_ms.to_be_bytes());
        payload
    }

    fn release_payload(identifier: i64, generation: u16) -> Vec<u8> {
        let mut payload = Vec::with_capacity(12);
        payload.extend_from_slice(&ACTION.to_be_bytes());
        payload.extend_from_slice(&identifier.to_be_bytes());
        payload.extend_from_slice(&generation.to_be_bytes());
        payload
    }

    /// Returns the status only; use `acquire_granting` when the generation is needed.
    async fn acquire(
        &mut self,
        identifier: i64,
        max_waiters: u8,
        wait_ms: u16,
        lease_ms: u16,
    ) -> u8 {
        self.acquire_granting(identifier, max_waiters, wait_ms, lease_ms)
            .await
            .0
    }

    /// Returns `(status, generation)`. The generation is what a later release must present.
    async fn acquire_granting(
        &mut self,
        identifier: i64,
        max_waiters: u8,
        wait_ms: u16,
        lease_ms: u16,
    ) -> (u8, u16) {
        let payload = Self::acquire_payload(identifier, max_waiters, wait_ms, lease_ms);
        let expected = self.write_frame(0x02, &payload).await;
        let (correlation, status, generation) = self.read_reply().await;
        assert_eq!(
            correlation, expected,
            "reply correlated to the wrong request"
        );
        (status, generation)
    }

    async fn release(&mut self, identifier: i64, generation: u16) -> u8 {
        let payload = Self::release_payload(identifier, generation);
        self.send(0x03, &payload).await
    }
}

#[tokio::test]
async fn budget_mutations_and_charges_share_the_authenticated_connection() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut client = Client::connect(&server).await;

    assert_eq!(
        client
            .send(0x05, &Client::budget_payload(1, 100, 100))
            .await,
        0
    );
    assert_eq!(
        client
            .send(0x05, &Client::budget_payload(2, 100, 100))
            .await,
        0
    );
    assert_eq!(client.send(0x01, &Client::charge_payload()).await, 0);
}

#[tokio::test]
async fn the_second_client_waits_for_an_explicit_release() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    let (status, generation) = holder.acquire_granting(1, 4, 2000, 15000).await;
    assert_eq!(status, 0);

    let address = server.address.clone();
    let waiter = tokio::spawn(async move {
        let mut socket = TcpStream::connect(&address).await.unwrap();
        let mut nonce = [0_u8; 8];
        socket.read_exact(&mut nonce).await.unwrap();
        let mut client = Client {
            socket,
            nonce,
            sequence: 0,
        };
        client.acquire(1, 4, 2000, 15000).await
    });

    tokio::time::sleep(Duration::from_millis(100)).await;
    assert!(
        !waiter.is_finished(),
        "the second client must still be queued"
    );
    assert_eq!(holder.release(1, generation).await, 0);
    assert_eq!(waiter.await.unwrap(), 0, "release must hand the lock over");
}

#[tokio::test]
async fn a_dropped_connection_releases_the_lock() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    assert_eq!(holder.acquire(2, 0, 0, 15000).await, 0);

    // A try-lock proves the key is really held: zero waiters, zero patience.
    let mut rival = Client::connect(&server).await;
    assert_eq!(rival.acquire(2, 0, 0, 15000).await, 1);

    // No release frame, no graceful close — exactly what a killed Lambda leaves behind.
    drop(holder);
    tokio::time::sleep(Duration::from_millis(100)).await;
    assert_eq!(
        rival.acquire(2, 0, 0, 15000).await,
        0,
        "dropping the holder's connection must free the key"
    );
}

#[tokio::test]
async fn a_silent_holder_loses_the_lock_when_its_lease_expires() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    // A 150 ms lease with a 30 s idle timeout: only the lease can end this hold, which is what
    // proves the deadline swapped when the lock was granted.
    assert_eq!(holder.acquire(3, 0, 0, 150).await, 0);

    let mut rival = Client::connect(&server).await;
    assert_eq!(rival.acquire(3, 0, 0, 15000).await, 1);

    // The holder simply says nothing; its connection is still open.
    tokio::time::sleep(Duration::from_millis(400)).await;
    assert_eq!(
        rival.acquire(3, 0, 0, 15000).await,
        0,
        "the lease must expire the hold without any release frame"
    );
}

#[tokio::test]
async fn a_queue_that_is_full_is_refused_immediately() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    assert_eq!(holder.acquire(4, 1, 5000, 15000).await, 0);

    let address = server.address.clone();
    let queued = tokio::spawn(async move {
        let mut socket = TcpStream::connect(&address).await.unwrap();
        let mut nonce = [0_u8; 8];
        socket.read_exact(&mut nonce).await.unwrap();
        let mut client = Client {
            socket,
            nonce,
            sequence: 0,
        };
        client.acquire(4, 1, 5000, 15000).await
    });
    tokio::time::sleep(Duration::from_millis(100)).await;

    let mut refused = Client::connect(&server).await;
    // One waiter is already parked, so this one is answered rather than queued — and answered
    // fast, which is the whole point of the ceiling.
    let reply = timeout(
        Duration::from_millis(500),
        refused.acquire(4, 1, 5000, 15000),
    )
    .await
    .expect("a refused acquire must not block");
    assert_eq!(reply, 1);
    queued.abort();
}

#[tokio::test]
async fn waiting_past_the_client_patience_reports_a_timeout() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    assert_eq!(holder.acquire(5, 4, 0, 15000).await, 0);

    let mut waiter = Client::connect(&server).await;
    assert_eq!(waiter.acquire(5, 4, 100, 15000).await, 2);
}

#[tokio::test]
async fn one_connection_can_hold_several_locks() {
    // What the widened release frame buys: a single shared connection is no longer a single lock,
    // so one process can serialize several keys at once.
    let server = start_server(Duration::from_secs(30)).await;
    let mut client = Client::connect(&server).await;
    let (first_status, first_generation) = client.acquire_granting(6, 4, 1000, 15000).await;
    let (second_status, second_generation) = client.acquire_granting(7, 4, 1000, 15000).await;
    assert_eq!(first_status, 0);
    assert_eq!(
        second_status, 0,
        "a second key on one connection must be allowed"
    );

    // Both are really held: a rival cannot take either.
    let mut rival = Client::connect(&server).await;
    assert_eq!(rival.acquire(6, 0, 0, 15000).await, 1);
    assert_eq!(rival.acquire(7, 0, 0, 15000).await, 1);

    // Releasing one must not disturb the other.
    assert_eq!(client.release(6, first_generation).await, 0);
    assert_eq!(rival.acquire(6, 0, 0, 15000).await, 0);
    assert_eq!(
        rival.acquire(7, 0, 0, 15000).await,
        1,
        "the other key is still held"
    );
    assert_eq!(client.release(7, second_generation).await, 0);
}

#[tokio::test]
async fn a_release_must_name_a_lock_this_connection_actually_holds() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut client = Client::connect(&server).await;
    let (status, generation) = client.acquire_granting(8, 4, 1000, 15000).await;
    assert_eq!(status, 0);

    // Right key, wrong generation: this is the shape of a release from a caller that already
    // gave up, arriving after someone else took the key.
    assert_eq!(
        client.release(8, generation.wrapping_add(1)).await,
        4,
        "a superseded generation must not end the current hold"
    );
    // A key this connection never held.
    assert_eq!(client.release(999, generation).await, 4);
    // The real one still works, proving the refusals above left it alone.
    assert_eq!(client.release(8, generation).await, 0);
    // And releasing it twice is a client bug.
    assert_eq!(client.release(8, generation).await, 4);
}

#[tokio::test]
async fn a_stale_release_cannot_end_the_hold_that_replaced_it() {
    // The race the generation exists for: two callers sharing one connection, the first giving up
    // while its release is already in flight.
    let server = start_server(Duration::from_secs(30)).await;
    let mut client = Client::connect(&server).await;
    let (_, first_generation) = client.acquire_granting(9, 4, 1000, 15000).await;
    assert_eq!(client.release(9, first_generation).await, 0);

    // Same key, taken again on the same connection: a different hold, so a different generation.
    let (_, second_generation) = client.acquire_granting(9, 4, 1000, 15000).await;
    assert_ne!(
        first_generation, second_generation,
        "each grant of a key must be distinguishable"
    );

    // The stale release from the first hold must not free the second.
    assert_eq!(client.release(9, first_generation).await, 4);
    let mut rival = Client::connect(&server).await;
    assert_eq!(
        rival.acquire(9, 0, 0, 15000).await,
        1,
        "the second hold must have survived the stale release"
    );
}

#[tokio::test]
async fn different_identifiers_do_not_block_each_other() {
    let server = start_server(Duration::from_secs(30)).await;
    let mut first = Client::connect(&server).await;
    let mut second = Client::connect(&server).await;
    assert_eq!(first.acquire(100, 0, 0, 15000).await, 0);
    assert_eq!(second.acquire(200, 0, 0, 15000).await, 0);
}

#[tokio::test]
async fn a_queued_acquire_does_not_delay_a_later_charge() {
    // The whole point of multiplexing, and false before it: a request parked in a lock queue must
    // not hold up unrelated work sent after it on the same connection.
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    let (status, generation) = holder.acquire_granting(50, 4, 3000, 15000).await;
    assert_eq!(status, 0);

    let mut client = Client::connect(&server).await;
    // This one will sit in the queue for up to 3s behind the holder above.
    let acquire_id = client
        .write_frame(0x02, &Client::acquire_payload(50, 4, 3000, 15000))
        .await;
    let charge_id = client.write_frame(0x01, &Client::charge_payload()).await;

    // The charge must come back first, while the acquire is still waiting.
    let (first, status, _) = timeout(Duration::from_millis(500), client.read_reply())
        .await
        .expect("the charge must be answered without waiting for the queued acquire");
    assert_eq!(
        first, charge_id,
        "replies did not overtake the parked acquire"
    );
    assert_eq!(status, 0, "the charge should have been admitted");

    // And the acquire still gets its own answer once the holder releases.
    assert_eq!(holder.release(50, generation).await, 0);
    let (second, status, _) = timeout(Duration::from_secs(2), client.read_reply())
        .await
        .expect("the queued acquire must still be answered");
    assert_eq!(second, acquire_id);
    assert_eq!(status, 0);
}

#[tokio::test]
async fn a_lease_expires_even_while_the_connection_stays_busy() {
    // The case the old relative read deadline silently failed: traffic on the connection used to
    // push the lease forward, so a wedged holder kept its lock forever.
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    assert_eq!(holder.acquire(60, 0, 0, 200).await, 0);

    let mut rival = Client::connect(&server).await;
    assert_eq!(
        rival.acquire(60, 0, 0, 15000).await,
        1,
        "the key must be held"
    );

    // Keep the holder's connection busy across its own lease.
    for _ in 0..6 {
        assert_eq!(holder.send(0x01, &Client::charge_payload()).await, 0);
        tokio::time::sleep(Duration::from_millis(50)).await;
    }

    assert_eq!(
        rival.acquire(60, 0, 0, 15000).await,
        0,
        "traffic on the holder's connection must not extend its lease"
    );
    // Expiry must not have killed the holder's connection, only its lock.
    assert_eq!(
        holder.send(0x01, &Client::charge_payload()).await,
        0,
        "lease expiry must leave the connection usable"
    );
}

#[tokio::test]
async fn disconnecting_with_a_queued_acquire_does_not_strand_the_lock() {
    // A client that walks away while queued must not be handed the lock afterwards: nobody would
    // ever release it, and it would sit locked until its lease ran out.
    let server = start_server(Duration::from_secs(30)).await;
    let mut holder = Client::connect(&server).await;
    let (status, generation) = holder.acquire_granting(70, 4, 5000, 15000).await;
    assert_eq!(status, 0);

    let mut leaver = Client::connect(&server).await;
    leaver
        .write_frame(0x02, &Client::acquire_payload(70, 4, 5000, 15000))
        .await;
    tokio::time::sleep(Duration::from_millis(100)).await;
    drop(leaver);
    tokio::time::sleep(Duration::from_millis(100)).await;

    assert_eq!(holder.release(70, generation).await, 0);
    tokio::time::sleep(Duration::from_millis(100)).await;

    let mut newcomer = Client::connect(&server).await;
    assert_eq!(
        newcomer.acquire(70, 0, 0, 15000).await,
        0,
        "the abandoned queued acquire must not have taken the lock"
    );
}
