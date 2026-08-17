//! Raw TCP listener, connection handshake, opcode dispatch, and graceful stop coordination.
//!
//! Each connection is a **single state owner**: the reader task is the only thing that touches
//! the locks this connection holds. Handlers run as spawned tasks so a queued acquire cannot
//! delay anything behind it, but they never mutate that state — they report results back over a
//! channel. That is what removes the three-way race between release, lease expiry, and
//! disconnect: there is only ever one writer.

use std::{
    collections::HashMap, io::ErrorKind, net::SocketAddr, sync::Arc, time::Duration, time::Instant,
};

use anyhow::{Context, Result, anyhow};
use tokio::{
    io::{AsyncReadExt, AsyncWriteExt},
    net::{TcpListener, TcpStream},
    sync::{Semaphore, mpsc, watch},
    task::JoinSet,
    time::timeout,
};
use tracing::{debug, info, warn};

use crate::{
    limiter::{
        budget::{MUTATE_BUDGET_PAYLOAD_SIZE, parse_budget_mutation},
        protocol::{CHARGE_PAYLOAD_SIZE, parse_charge},
        quota::RateLimiter,
    },
    lock::{
        protocol::{
            ACQUIRE_PAYLOAD_SIZE, LockReply, RELEASE_PAYLOAD_SIZE, parse_acquire, parse_release,
        },
        registry::{LockGuard, LockOutcome, LockRegistry},
    },
    reqlog::{protocol::parse_request_log, writer::RequestLogSink},
    service::{
        auth,
        protocol::{
            AUTH_TAG_SIZE, LENGTH_PREFIX_SIZE, MAX_FRAME_SIZE, OPCODE_SIZE, Opcode, PayloadWidth,
            REPLY_SIZE, UNAVAILABLE_STATUS, encode_reply,
        },
    },
};

type LockKey = (u16, i64);

/// One lock this connection holds. The guard is what actually owns it; dropping this entry —
/// on release, on expiry, or when the connection ends — is the only way it is freed.
struct HeldLock {
    /// Held purely for its `Drop`: while this value exists the lock is ours, and removing the
    /// entry is what frees it. Never read, hence the underscore.
    _guard: LockGuard,
    /// Identifies this grant specifically, so a release from a caller that already gave up
    /// cannot end the hold that replaced it on the same key.
    generation: u16,
    expires_at: Instant,
}

/// What a spawned acquire reports back. `None` means it was refused or timed out, which the
/// reader still needs so it can drop its pending count.
struct AcquireOutcome {
    key: LockKey,
    granted: Option<HeldLock>,
}

pub async fn run(
    listener: TcpListener,
    limiter: Arc<RateLimiter>,
    locks: Arc<LockRegistry>,
    request_logs: RequestLogSink,
    secret: Arc<Vec<u8>>,
    frame_timeout: Duration,
    max_connections: usize,
    max_inflight: usize,
    mut shutdown: watch::Receiver<bool>,
) -> Result<()> {
    let connection_slots = Arc::new(Semaphore::new(max_connections));
    let mut connections = JoinSet::new();
    info!(address = %listener.local_addr()?, max_connections, max_inflight, "server utils listening");

    loop {
        tokio::select! {
            changed = shutdown.changed() => {
                if changed.is_err() || *shutdown.borrow() {
                    break;
                }
            }
            accepted = listener.accept() => {
                // Transient accept failures (EMFILE when the fd limit is reached, a peer that
                // vanished between the SYN and the accept) must not take the whole listener
                // down: propagating here would silently stop serving every future connection.
                let (socket, peer) = match accepted {
                    Ok(accepted) => accepted,
                    Err(accept_error) => {
                        warn!(error = %accept_error, "accept failed, continuing to listen");
                        tokio::time::sleep(Duration::from_millis(20)).await;
                        continue;
                    }
                };
                let Ok(slot) = connection_slots.clone().try_acquire_owned() else {
                    warn!(%peer, "rejecting connection because the connection limit is full");
                    drop(socket);
                    continue;
                };
                socket.set_nodelay(true).context("failed to set TCP_NODELAY")?;
                let limiter = limiter.clone();
                let locks = locks.clone();
                let request_logs = request_logs.clone();
                let secret = secret.clone();
                let connection_shutdown = shutdown.clone();
                connections.spawn(async move {
                    let _slot = slot;
                    if let Err(connection_error) = handle_connection(
                        socket,
                        peer,
                        limiter,
                        locks,
                        request_logs,
                        &secret,
                        frame_timeout,
                        max_inflight,
                        connection_shutdown,
                    ).await {
                        debug!(%peer, error = %connection_error, "connection closed with error");
                    }
                });
            }
            completed = connections.join_next(), if !connections.is_empty() => {
                if let Some(Err(join_error)) = completed {
                    warn!(error = %join_error, "connection task failed");
                }
            }
        }
    }

    // Connection tasks observe the same shutdown channel and drop their sockets promptly.
    while let Some(result) = connections.join_next().await {
        if let Err(join_error) = result {
            warn!(error = %join_error, "connection task failed during shutdown");
        }
    }
    Ok(())
}

async fn handle_connection(
    socket: TcpStream,
    peer: SocketAddr,
    limiter: Arc<RateLimiter>,
    locks: Arc<LockRegistry>,
    request_logs: RequestLogSink,
    secret: &[u8],
    frame_timeout: Duration,
    max_inflight: usize,
    mut shutdown: watch::Receiver<bool>,
) -> Result<()> {
    let (mut reader, mut writer) = socket.into_split();
    let nonce = auth::new_nonce()?;
    writer
        .write_all(&nonce)
        .await
        .context("failed to write connection nonce")?;
    debug!(%peer, "connection nonce sent");

    // Bounded on purpose. One request per socket used to provide backpressure for free;
    // multiplexing removes it, so replies and in-flight work each need an explicit ceiling or one
    // authenticated client could spawn tasks and buffer replies without limit.
    let (reply_sender, mut reply_receiver) = mpsc::channel::<[u8; REPLY_SIZE]>(max_inflight);
    let writer_task = tokio::spawn(async move {
        while let Some(reply) = reply_receiver.recv().await {
            if writer.write_all(&reply).await.is_err() {
                break;
            }
        }
    });

    let (acquire_sender, mut acquire_receiver) = mpsc::channel::<AcquireOutcome>(max_inflight);
    let inflight = Arc::new(Semaphore::new(max_inflight));
    // Dropping this aborts every spawned handler, including an acquire still parked in the
    // registry's queue. Without that, a disconnected client's queued acquire would still be
    // granted a lock that nobody holds and nobody will release.
    let mut handlers = JoinSet::new();

    // Owned solely by this loop. Sharing it with the spawned handlers would defeat the crash
    // path: a task parked for `wait_ms` would keep it alive, so dropping it on disconnect would
    // not release anything until that task finished.
    let mut held: HashMap<LockKey, HeldLock> = HashMap::new();
    let mut sequence = 0_u64;
    let mut frame = [0_u8; MAX_FRAME_SIZE];

    let outcome = loop {
        drop_expired(&mut held, peer);

        // Derived from stamped deadlines, never from the lease itself, so arriving traffic cannot
        // push a hold forward. A connection holding a lock is bounded by its earliest expiry and
        // not by the idle timeout: a caller legitimately holding a 30s lease is quiet, not dead.
        let idle_timeout = match held.values().map(|lock| lock.expires_at).min() {
            Some(expiry) => expiry.saturating_duration_since(Instant::now()),
            None => frame_timeout,
        };

        let read_result = tokio::select! {
            changed = shutdown.changed() => {
                if changed.is_err() || *shutdown.borrow() {
                    break Ok(());
                }
                continue;
            }
            reported = acquire_receiver.recv() => {
                // The single writer of `held`.
                if let Some(outcome) = reported
                    && let Some(lock) = outcome.granted
                    && held.insert(outcome.key, lock).is_some()
                {
                    // The registry hands out one permit per key, so a second grant cannot arrive
                    // while the first is still held. If it ever does, the overwrite would drop a
                    // guard the client still believes it owns.
                    warn!(%peer, action = outcome.key.0, identifier = outcome.key.1,
                          "a grant replaced a hold this connection already had");
                }
                continue;
            }
            result = timeout(idle_timeout, reader.read_exact(&mut frame[..OPCODE_SIZE])) => result,
        };

        match read_result {
            // A lease elapsed. Never fatal: killing the socket here would drop every other lock
            // on it and fail every request still in flight.
            Err(_) if !held.is_empty() => continue,
            Err(_) => break Err(anyhow!("idle timeout waiting for an opcode")),
            Ok(Err(error)) if error.kind() == ErrorKind::UnexpectedEof => break Ok(()),
            Ok(Err(error)) => break Err(error).context("opcode read failed"),
            Ok(Ok(_)) => {}
        }

        let Some(opcode) = Opcode::from_byte(frame[0]) else {
            break Err(anyhow!("unknown opcode {}", frame[0]));
        };

        // Where the payload starts and how long it is. Fixed-width opcodes answer both from the
        // opcode alone; the request log states its own length, which is read first and checked
        // against the ceiling before a single byte of it is buffered — a length header is an
        // instruction from an unauthenticated peer until the tag at the end says otherwise.
        let (payload_offset, payload_size) = match opcode.payload_width() {
            PayloadWidth::Fixed(size) => (OPCODE_SIZE, size),
            PayloadWidth::LengthPrefixed { maximum } => {
                if let Err(error) = timeout(
                    frame_timeout,
                    reader.read_exact(&mut frame[OPCODE_SIZE..OPCODE_SIZE + LENGTH_PREFIX_SIZE]),
                )
                .await
                .map_err(|_| anyhow!("frame length read timed out"))
                .and_then(|read| read.context("frame length read failed"))
                {
                    break Err(error);
                }
                let declared =
                    u16::from_be_bytes([frame[OPCODE_SIZE], frame[OPCODE_SIZE + 1]]) as usize;
                if declared > maximum {
                    break Err(anyhow!(
                        "frame declares a {declared}-byte payload, over the {maximum}-byte ceiling"
                    ));
                }
                (OPCODE_SIZE + LENGTH_PREFIX_SIZE, declared)
            }
        };
        let frame_size = payload_offset + payload_size + AUTH_TAG_SIZE;

        // The rest of the frame is already in flight, so an EOF here is a truncated frame and
        // not a clean disconnect.
        if let Err(error) = timeout(
            frame_timeout,
            reader.read_exact(&mut frame[payload_offset..frame_size]),
        )
        .await
        .map_err(|_| anyhow!("frame body read timed out"))
        .and_then(|read| read.context("frame body read failed"))
        {
            break Err(error);
        }

        let tag_offset = frame_size - AUTH_TAG_SIZE;
        let mut received_tag = [0_u8; AUTH_TAG_SIZE];
        received_tag.copy_from_slice(&frame[tag_offset..frame_size]);
        // The opcode is inside the signed bytes, so a frame cannot be replayed as another
        // operation, and the sequence keeps it from being replayed as itself.
        if !auth::verify_hash(
            secret,
            &nonce,
            sequence,
            &frame[..tag_offset],
            &received_tag,
        )? {
            warn!(%peer, sequence, opcode = frame[0], "frame authentication failed");
            break Ok(());
        }
        let frame_sequence = sequence;
        sequence = match sequence.checked_add(1) {
            Some(next) => next,
            None => break Err(anyhow!("frame sequence overflow")),
        };

        // Refusing beyond the ceiling keeps one connection from spawning without bound. Taken
        // before spawning so the permit's lifetime is exactly the handler's.
        let permit = inflight.clone().try_acquire_owned().ok();

        match opcode {
            Opcode::ChargeCredits => {
                let payload: &[u8; CHARGE_PAYLOAD_SIZE] = frame[OPCODE_SIZE..tag_offset]
                    .try_into()
                    .expect("the opcode fixes the payload width");
                let request = match parse_charge(payload) {
                    Ok(request) => request,
                    Err(error) => break Err(error).context("authenticated frame is invalid"),
                };
                let Some(permit) = permit else {
                    warn!(%peer, "in-flight ceiling reached, refusing a charge");
                    send_reply(&reply_sender, frame_sequence, UNAVAILABLE_STATUS, 0).await;
                    continue;
                };
                let limiter = limiter.clone();
                let reply_sender = reply_sender.clone();
                handlers.spawn(async move {
                    let _permit = permit;
                    // A cold subject loads its usage from Scylla, so this cannot be inlined in the
                    // reader without head-of-line blocking every other request behind it.
                    let status = match limiter.admit(request).await {
                        Ok(decision) => decision.map_or(0, |violation| violation.response_byte()),
                        Err(admit_error) => {
                            warn!(error = %admit_error, "charge admission failed");
                            UNAVAILABLE_STATUS
                        }
                    };
                    send_reply(&reply_sender, frame_sequence, status, 0).await;
                });
            }
            Opcode::MutateCompanyBudget => {
                let payload: &[u8; MUTATE_BUDGET_PAYLOAD_SIZE] = frame[OPCODE_SIZE..tag_offset]
                    .try_into()
                    .expect("the opcode fixes the payload width");
                let mutation = match parse_budget_mutation(payload) {
                    Ok(mutation) => mutation,
                    Err(error) => break Err(error).context("authenticated frame is invalid"),
                };
                let Some(permit) = permit else {
                    warn!(%peer, "in-flight ceiling reached, refusing a budget mutation");
                    send_reply(&reply_sender, frame_sequence, UNAVAILABLE_STATUS, 0).await;
                    continue;
                };
                let limiter = limiter.clone();
                let reply_sender = reply_sender.clone();
                handlers.spawn(async move {
                    let _permit = permit;
                    let status = match limiter.mutate_budget(mutation).await {
                        Ok(outcome) => outcome as u8,
                        Err(mutation_error) => {
                            warn!(error = %mutation_error, "budget mutation failed");
                            UNAVAILABLE_STATUS
                        }
                    };
                    send_reply(&reply_sender, frame_sequence, status, 0).await;
                });
            }
            Opcode::LockAcquire => {
                let payload: &[u8; ACQUIRE_PAYLOAD_SIZE] = frame[OPCODE_SIZE..tag_offset]
                    .try_into()
                    .expect("the opcode fixes the payload width");
                let request = match parse_acquire(payload) {
                    Ok(request) => request,
                    Err(error) => break Err(error).context("authenticated frame is invalid"),
                };
                let Some(permit) = permit else {
                    warn!(%peer, "in-flight ceiling reached, refusing an acquire");
                    send_reply(&reply_sender, frame_sequence, LockReply::Capacity as u8, 0).await;
                    continue;
                };
                let key = (request.action, request.identifier);
                let lease = locks.clamp_lease(request.lease);
                let locks = locks.clone();
                let reply_sender = reply_sender.clone();
                let acquire_sender = acquire_sender.clone();
                handlers.spawn(async move {
                    let _permit = permit;
                    let (status, detail, granted) = match locks.acquire(request).await {
                        LockOutcome::Acquired(guard) => {
                            let generation = guard.generation();
                            (
                                LockReply::Ok as u8,
                                generation,
                                Some(HeldLock {
                                    _guard: guard,
                                    generation,
                                    expires_at: Instant::now() + lease,
                                }),
                            )
                        }
                        LockOutcome::Busy => (LockReply::Busy as u8, 0, None),
                        LockOutcome::WaitTimeout => (LockReply::WaitTimeout as u8, 0, None),
                        LockOutcome::Capacity => (LockReply::Capacity as u8, 0, None),
                    };
                    // Hand ownership to the reader before answering. If it is gone the guard
                    // drops here instead, which releases the lock rather than stranding it.
                    if acquire_sender
                        .send(AcquireOutcome { key, granted })
                        .await
                        .is_err()
                    {
                        return;
                    }
                    send_reply(&reply_sender, frame_sequence, status, detail).await;
                });
            }
            Opcode::LockRelease => {
                let payload: &[u8; RELEASE_PAYLOAD_SIZE] = frame[OPCODE_SIZE..tag_offset]
                    .try_into()
                    .expect("the opcode fixes the payload width");
                let request = parse_release(payload);
                let key = (request.action, request.identifier);
                // Both the key and the generation must match. A caller that gave up while its
                // release was already in flight would otherwise end whichever hold replaced it.
                let status = match held.get(&key) {
                    Some(lock) if lock.generation == request.generation => {
                        held.remove(&key);
                        debug!(%peer, sequence = frame_sequence, action = key.0, identifier = key.1,
                               "released lock");
                        LockReply::Ok as u8
                    }
                    Some(lock) => {
                        warn!(%peer, action = key.0, identifier = key.1,
                              held = lock.generation, presented = request.generation,
                              "refusing a release from a superseded hold");
                        LockReply::Misuse as u8
                    }
                    None => {
                        warn!(%peer, action = key.0, identifier = key.1,
                              "release names a lock this connection does not hold");
                        LockReply::Misuse as u8
                    }
                };
                send_reply(&reply_sender, frame_sequence, status, 0).await;
            }
            Opcode::LogRequest => {
                // The only opcode that answers nothing, so it also takes no in-flight permit: the
                // work is a channel send that cannot block, and parking a slot for it would let a
                // burst of log frames starve the charges and locks sharing this connection.
                drop(permit);
                match parse_request_log(&frame[payload_offset..tag_offset]) {
                    // Submit never blocks: a full queue drops the record and counts it.
                    Ok(record) => request_logs.submit(record),
                    // A malformed payload is a client bug worth seeing, but not worth closing a
                    // connection over — that would take this backend's charges and locks with it,
                    // for the sake of a log row. The other opcodes break the loop here because
                    // their frames decide whether a request is admitted at all.
                    Err(parse_error) => {
                        warn!(%peer, sequence = frame_sequence, error = %parse_error,
                              "discarding a malformed request log frame");
                    }
                }
            }
        }
    };

    // Order matters. Aborting first stops a queued acquire from being granted a lock nobody would
    // ever release; dropping `held` then frees everything this connection still holds, which is
    // what makes a dead socket an immediate release.
    handlers.abort_all();
    drop(handlers);
    held.clear();
    drop(reply_sender);
    let _ = writer_task.await;
    outcome
}

/// Drops every hold whose stamped deadline has passed. Absolute, so no amount of arriving
/// traffic can extend one.
fn drop_expired(held: &mut HashMap<LockKey, HeldLock>, peer: SocketAddr) {
    if held.is_empty() {
        return;
    }
    let now = Instant::now();
    held.retain(|key, lock| {
        let alive = lock.expires_at > now;
        if !alive {
            warn!(%peer, action = key.0, identifier = key.1, "lock lease expired without a release");
        }
        alive
    });
}

async fn send_reply(
    sender: &mpsc::Sender<[u8; REPLY_SIZE]>,
    sequence: u64,
    status: u8,
    detail: u16,
) {
    // A closed channel means the connection is already going away, so the reply has nowhere to go.
    let _ = sender.send(encode_reply(sequence, status, detail)).await;
}
