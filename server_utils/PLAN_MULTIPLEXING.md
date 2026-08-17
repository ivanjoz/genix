# PLAN — One connection, multiplexed

Status: **proposal, pending approval**. No code written yet.

Goal: one TCP connection per backend process carrying every opcode — charges and locks — with
many requests in flight at once. Today each lock occupies a whole connection and the limiter has
its own, so a process doing sign-up holds two sockets and a blocked acquire owns one of them for
the length of its wait.

---

## 1. The trade being made, stated up front

Multiplexing breaks how the lease is enforced today, but **it does not force a sweeper.**

What breaks is that the current deadline is *relative and restarted on every read*:
`timeout(lease, read)` begins again each loop iteration. That is safe only while nothing else
uses the connection. Once charges flow down the same socket they push the deadline forward
continuously, and a wedged holder keeps its lock for as long as unrelated traffic keeps arriving.

Note what the problem is not: it is not ambiguity about which lock is which. A perfect lock id
would not help, because the deadline would still restart. The fix is to make the deadline
**absolute** — an `Instant` stamped at grant time and compared against, so no arriving frame can
extend it. The generation introduced in §2 is orthogonal; it solves stale releases.

Enforcement then stays **in the reader**, where the connection's timing already lives — no
background timer, no per-lock task. Each iteration derives its read timeout from the earliest
stamped deadline instead of from the lease, and drops whatever has expired when it wakes. The one
wrinkle is that acquires are spawned, so a granting task can insert a deadline earlier than the
sleep the reader already committed to; a `Notify` it signals after inserting makes the reader
recompute (§3).

**Expiry must stop killing the connection.** Today a lease timeout returns an error and drops the
socket. Under multiplexing that is unacceptable: one wedged lock would tear down every other
in-flight request and every other lock the process holds. Expiry removes its own entry and the
connection keeps running.

What is *not* lost: instant crash release. The daemon still tracks which locks belong to which
connection, so a dead socket drops all of them immediately. The absolute deadline is only the
backstop for a connection that stays open while its owner is wedged.

What *is* traded: a single socket error now releases every lock that process holds, not one.
Blast radius goes from one request to all concurrent holders, so the client must actively tell
holders they lost their lock instead of letting them run on. And expiry becomes a routine event
rather than an anomaly, so the handler must stay safe to run twice.

---

## 2. Wire changes

### Replies grow from 1 byte to a fixed 5

```
[correlation:u16][status:u8][detail:u16]
```

- **correlation** — the low 16 bits of the frame sequence the request was sent with. No new
  request field is needed: the sequence already exists, is already per-connection and monotonic,
  and both sides already track it for the HMAC. Wrapping needs 65 536 requests in flight at once.
- **status** — unchanged codes. Zero is still success; charge rejections keep their bit layout,
  lock replies keep `1` busy, `2` wait timeout, `3` capacity, `4` misuse.
- **detail** — the lock generation on a granted acquire, zero everywhere else.

Fixed width keeps the client's reader loop a single `read_exact` of five bytes with no lookup in
the middle.

### Requests: only release changes

| Op | Payload | Frame |
|---|---|---|
| `0x01` `CHARGE_CREDITS` | unchanged | 20 |
| `0x02` `LOCK_ACQUIRE` | unchanged | 24 |
| `0x03` `LOCK_RELEASE` | `action:u16 · identifier:i64 · generation:u16` | 21 (was 9) |

Release has to name its lock now that the connection no longer identifies one.

### The generation, and why it is needed now

A per-key counter the daemon increments on every grant and returns in `detail`. Release must
present the matching value or it is refused.

It was unnecessary while each lock owned a connection. With one shared connection it is not:
goroutine A and goroutine B in the same process now share a socket, so a late release from A —
one that timed out and gave up while its frame was already in flight — would otherwise free the
lock B just acquired on the same key. Same key, same connection, different holder. The counter
makes that exact rather than probabilistic, at the same two bytes a random token would cost.

### The HMAC construction is unchanged, but the domain is bumped

The inputs are the same — domain, nonce, sequence, opcode+payload. What changes is the domain
*string*, and it must change on **every wire change, replies included**.

Replies are not authenticated, so a version skew cannot be caught by the signature itself: an old
client would keep authenticating fine, read 1 byte of a 5-byte reply, and misread every frame
after that. Bumping the domain turns that into an immediate authentication failure and a closed
connection instead. `genix-server-utils:v2` already ships with the widened reply; **widening
`LOCK_RELEASE` in step 4 bumps it again**.

Requests still travel in order, so the sequence stays in lockstep; only replies are reordered.
The one hard client requirement: **assigning the sequence and writing the frame must be atomic**.
If two goroutines took sequences 5 and 6 but wrote them as 6, 5, every subsequent frame would
fail authentication. One write mutex, held for a socket write and never for a round trip.

---

## 3. Daemon changes (Rust)

`service/server.rs` `handle_connection` stops being a read/process/reply loop:

The structure is a **single per-connection state owner**. The reader loop is the only thing that
touches `held`; spawned tasks never mutate it, they only report back. That is what removes the
three-way race between release, expiry, and disconnect — there is nothing to race on.

```
split the socket; spawn a writer task draining a BOUNDED mpsc<[u8;5]>
held:    HashMap<(u16,i64), HeldLock>   // plain map, NO Mutex — the reader owns it outright
granted: mpsc::Receiver<Granted>        // acquire tasks report results here, they never insert
tasks:   JoinSet                        // every spawned handler; dropping it aborts them all
inflight: Semaphore(max_inflight_per_connection)

reader loop:
  // Derived from stamped deadlines, never from the lease, so traffic cannot extend it. A
  // connection holding locks is bounded by its earliest expiry, not by the idle timeout: a
  // caller that legitimately holds a lock for 30s is silent, not dead.
  idle_timeout = if held.is_empty() { frame_timeout }
                 else { earliest(held.expires_at) - now }

  select:
    read opcode   => drop_expired(&mut held); dispatch
    timeout       => if held.is_empty() { close, as today }
                     else { drop_expired(&mut held); continue }   // never fatal
    granted.recv  => insert (guard, generation, expires_at) into held   // sole writer
    shutdown      => break

  dispatch (each first takes an inflight permit, released when the task ends):
    CHARGE_CREDITS => tasks.spawn { admit; reply }
    LOCK_ACQUIRE   => tasks.spawn { registry.acquire().await;
                                    granted.send(Granted{key, guard, gen, expires_at});
                                    reply }
    LOCK_RELEASE   => look up in held, check generation, remove (drops guard), reply

on exit: drop(tasks)  -> aborts every parked acquire, dropping any permit it holds
         drop(held)   -> releases every lock this connection holds, immediately
```

- **Charges are spawned too, not inlined.** A cold subject makes `admit` hit Scylla through
  `load_exact`, and inlining that would head-of-line block the reader — the exact thing
  multiplexing exists to remove.
- **`held` has no `Mutex` and no `Arc`.** Sharing it with spawned tasks would defeat the crash
  path: a task parked in `registry.acquire()` for `wait_ms` would keep a handle alive, so
  "drop `held` on disconnect" would not actually release anything until that task finished.
  Reporting results over a channel keeps the reader the only owner.
- **Disconnect aborts pending acquires.** `JoinSet` aborts on drop; a task cancelled while
  awaiting the semaphore never takes the permit, and one cancelled just after taking it drops it.
  Without this, a disconnected client's queued acquire would still be granted a lock nobody holds.
- **In-flight work is bounded per connection**, and the reply channel is bounded. Multiplexing
  removes the natural backpressure that one-request-per-socket provided: without a cap, a single
  authenticated connection could spawn unbounded tasks and buffer unbounded replies.
- **Expiry is enforced by the reader**, not by a task per lock — the loop already owns the
  timing, and single ownership makes the scan trivial. A `DelayQueue` (tokio-util) would be the
  right shape if a connection ever held many locks; for the handful it will hold, a min over the
  map avoids a dependency for no gain. Revisit if that assumption changes.
- **Expiry is never fatal to the connection.** It removes its own entry and the loop continues.
  Killing the socket would drop every other lock and fail every pending request on it.
- **The idle timeout no longer applies while locks are held.** Otherwise a caller quietly holding
  a 30s lease would lose it — and everything else on the connection — at `frame_timeout`.
- **Removal is generation-matched**, so dropping an expired entry can never remove a newer
  acquisition of the same key.

`lock/registry.rs` gains the per-key generation counter. `service/protocol.rs` gains the reply
width; `MAX_FRAME_SIZE` is unchanged, since acquire's 15-byte payload is still the widest.

---

## 4. Client changes (Go)

`credit_rate_limiter.go` and `lock_service.go` collapse into one `ServerUtilsClient`.

**The public API does change.** `ChargeAPIUsage` and `ChargeInferenceUsage` keep their signatures,
but `AcquireLock` goes from `(func(), error)` to `(*Lock, error)`, because a holder now has to be
told when its lock is gone. There is one call site today — `PostSignUpRequest` — so the edit is
`defer releaseLock()` becoming `defer lock.Release()`, but the plan should not pretend the
surface is unchanged.

```go
type muxConnection struct {
    conn      net.Conn
    nonce     [8]byte
    writeMu   sync.Mutex              // guards sequence AND the write, together
    sequence  uint64
    pendingMu sync.Mutex
    pending   map[uint16]chan muxReply
    closed    chan struct{}
}
```

- `send` registers `pending[seq]` **before** writing, or a fast reply could arrive with nobody
  waiting for it.
- One reader goroutine per connection reads 5 bytes, correlates, delivers, deletes.
- On any read error: close the socket, close `closed`, and fail every pending request. One dead
  socket must not leave goroutines blocked forever.
- Reconnect is lazy — the next `send` dials. A new connection means the daemon dropped the old
  one's locks, so holders must not keep believing they hold anything.
- **A giving-up caller must not simply forget its request.** If a context is cancelled after the
  acquire frame was written, the daemon may still grant that lock afterwards. The pending entry
  is therefore marked *abandoned* rather than deleted: if a grant arrives for it, the reader
  immediately sends the matching release. Otherwise the key stays locked until its lease expires,
  with nobody holding it.

`AcquireLock` returns a handle rather than a bare closure, because holders now need to be told:

```go
type Lock struct{ /* ... */ }
func (lock *Lock) Release()
func (lock *Lock) Lost() <-chan struct{}   // closed when this lock is no longer ours
```

`Lost()` must close on **two** events, not one:

1. the connection died, so the daemon dropped every lock on it;
2. the lease elapsed. The daemon expires the hold on its own clock and the client would otherwise
   keep believing it holds the key. There is no server-push frame in this protocol, so the client
   arms a local timer at grant time. It fires marginally later than the daemon's — the grant reply
   takes a round trip to arrive — so it is advisory, not authoritative.

That caveat is the general one: `Lost()` cannot make anything safe under a partition, because the
holder may already be past the check by the time it fires. Work inside a lock must stay
idempotent regardless; the channel narrows the window and makes the failure visible instead of
silent. Sign-up will not select on it — its critical section is a straight line.

---

## 5. Tests

The two that actually prove the design:

- **Rust:** an acquire parked in the queue must not delay a charge sent afterwards on the same
  connection. That is the whole point of the change and it is false today.
- **Rust:** a lease must expire while the connection stays open *and busy with other frames* —
  the case the relative read deadline silently fails — and the connection must still be usable
  afterwards, with any second lock on it untouched.

- **Rust:** a client that disconnects *while one of its acquires is still queued* must release
  everything immediately — not after that queued acquire finishes. This is the case an
  `Arc<Mutex<held>>` shared with spawned tasks would silently break.

Plus: connection drop releases every lock held on it; release with a stale generation is refused;
two keys can be held at once on one connection; a lock granted while the reader is already
sleeping still expires on time; a quiet connection holding a lock is not closed at
`frame_timeout`; the per-connection in-flight cap refuses rather than spawns without bound. On
the Go side: out-of-order replies correlate to the right callers; connection death fails all
pending and closes `Lost()`; a lease elapsing closes `Lost()` without any frame arriving; an
abandoned acquire that is granted late is released automatically; concurrent senders produce
strictly increasing sequences.

---

## 6. Ordering and risk

Breaking wire change again — daemon and backend deploy together, and an old backend against a new
daemon fails authentication silently (charges fail closed, so the symptom is HTTP 503 responses).

1. ~~Reply widening to 5 bytes with correlation, both sides, still single-request-in-flight.~~
   **Done**, with the domain bumped to `genix-server-utils:v2`.
2. Rust: the per-connection state owner — split socket, bounded writer channel, `JoinSet` for
   spawned handlers, reader-owned `held`, generation, in-flight cap, and reader-enforced absolute
   deadlines with non-fatal expiry.
3. Go: the multiplexing client, merging the two existing ones, including abandoned-acquire
   tracking and `Lost()`.
4. Release frame widening plus generation checking — bumps the domain to `:v3`.
5. Delete the lock connection pool and `LockServiceClient`; update the one `AcquireLock` call site.

## 7. Out of scope

- **More than one connection per process.** If one socket's write mutex ever becomes a
  bottleneck — it should not; a 20-byte write is microseconds — the manager can hold N
  connections and hash onto them. Not now.
- **Lease renewal.** The timer makes it possible, and it is the honest fix for a critical section
  that outgrows its lease, but nothing needs it yet.
- **Fencing tokens carried into the database.** The generation guards the daemon's own state, not
  Scylla writes. A partitioned holder can still write after losing its lock; that is inherent to
  every liveness-based lock and is why the handler must remain safe to run twice.
