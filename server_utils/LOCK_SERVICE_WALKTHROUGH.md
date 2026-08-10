# The lock service, end to end

How a sign-up request travels from a browser through two processes and back, and what each byte
on the wire is for. Describes **what is implemented today**. The last section shows what changes
when multiplexing lands (`PLAN_MULTIPLEXING.md`).

---

## 1. The problem, concretely

Three people — or one attacker with a script — hit `POST p-signup-request` from the same IP in
the same second. AWS starts three Lambdas. There is no transaction across them, and the ORM has
no LWT, so all three read the same state and all three act on it.

```
Lambda A  ──read: "this IP used 4 of 5"──┐
Lambda B  ──read: "this IP used 4 of 5"──┼── all three read BEFORE anyone writes
Lambda C  ──read: "this IP used 4 of 5"──┘
              │           │           │
              ▼           ▼           ▼
          under limit  under limit  under limit      ← each concludes it may proceed
              │           │           │
              ▼           ▼           ▼
           send mail   send mail   send mail         ← the limit of 5 just delivered 7
              │           │           │
              ▼           ▼           ▼
           INSERT      INSERT      INSERT
```

The count is right; the *timing* is wrong. Every read happened before any write. That is the
whole bug, and it is why a counter alone cannot fix it.

---

## 2. The fix in one line

Put a gate in front of the read so only one Lambda is inside at a time. The gate lives in the
Rust daemon because it is the only thing all the Lambdas share.

```
Lambda A ──▶│ acquire ▶ GRANTED  ──▶ read(4) ─▶ send ─▶ insert ─▶ release │──▶ 200
Lambda B ──▶│ acquire ▶ ...waiting...          then ▶ read(5) ─▶ over limit│──▶ 429
Lambda C ──▶│ acquire ▶ QUEUE FULL (2 already waiting)                      │──▶ 429
             ▲
             └── one key: (action=1, identifier=<the IP>)
```

B and C are not wrong to be refused — they are the abuse pattern. What matters is that B reads
**5**, not 4, because it ran *after* A wrote.

---

## 3. Who talks to whom

```
browser                backend (Go, Lambda)              server-utils (Rust daemon)
   │                          │                                    │
   │── POST p-signup-request ▶│                                    │
   │                          │──── TCP connect ──────────────────▶│
   │                          │◀─── 8 random bytes (nonce) ────────│   handshake
   │                          │                                    │
   │                          │──── 0x02 LOCK_ACQUIRE ────────────▶│   24 bytes
   │                          │                                    │   ├ key free? take it
   │                          │◀─── reply: status 0 (granted) ─────│   5 bytes
   │                          │                                    │
   │                          │  ── Scylla: count emails for IP    │   (daemon not involved)
   │                          │  ── SMTP: send the code            │
   │                          │  ── Scylla: INSERT the request     │
   │                          │                                    │
   │                          │──── 0x03 LOCK_RELEASE ────────────▶│   9 bytes
   │                          │◀─── reply: status 0 ───────────────│
   │◀───── 200 {RequestID} ───│                                    │
```

The daemon never touches the database and knows nothing about e-mail. It answers one question:
*may I be the one running right now?*

---

## 4. The bytes

### Handshake

On connect the daemon writes **8 random bytes**. Every frame after that is signed with them, so a
frame captured on one connection cannot be replayed on another.

### Request frame

```
        ┌────────┬──────────────────────────┬──────────────┐
        │ opcode │        payload           │  HMAC (8)    │
        │  1 B   │   width fixed per opcode │              │
        └────────┴──────────────────────────┴──────────────┘
         └──────── signed together ────────┘
```

The opcode is a routing header only — the three operations share no field:

| Op | Name | Payload | Total |
|---|---|---|---|
| `0x01` | `CHARGE_CREDITS` | company u24 · user u24 · group u8 · cpu u16 · inference u16 | 20 B |
| `0x02` | `LOCK_ACQUIRE` | action u16 · identifier i64 · max_waiters u8 · wait_ms u16 · lease_ms u16 | 24 B |
| `0x03` | `LOCK_RELEASE` | — (the connection identifies the lock) | 9 B |

`HMAC = SHA256(internal_apikey, "genix-server-utils:v2" ‖ nonce ‖ sequence_be64 ‖ opcode+payload)`,
truncated to 8 bytes. The sequence counts frames on this connection and both sides increment it
in lockstep, so a frame cannot be replayed even as itself.

### Reply frame — 5 bytes

```
        ┌───────────────┬────────┬───────────────┐
        │ correlation   │ status │    detail     │
        │   u16         │  u8    │     u16       │
        └───────────────┴────────┴───────────────┘
```

`correlation` is the low 16 bits of the request's sequence, echoed back. Nothing extra is sent to
carry it — the sequence already exists for the HMAC. Today only one request is ever in flight, so
it is a desync check; under multiplexing it is what routes a reply to the right caller.

`status` — `0` always means success:

| status | `LOCK_ACQUIRE` means |
|---|---|
| `0` | granted |
| `1` | queue was full — refused without waiting |
| `2` | waited `wait_ms` and never reached the front |
| `3` | daemon at capacity (`max_keys` / `max_total_waiters`) |
| `4` | protocol misuse (already holding, or releasing nothing) |

---

## 5. The worked example

Client IP `203.0.113.45`, first frame on a fresh connection (sequence 0).

**Step 1 — the IP becomes an int64.** `backend/core/responses.go:71`

```
203.0.113.45  ─▶  0xCB00712D  ─▶  3405803821
                  (IPv6 instead keys the /63 prefix, because one customer owns a whole /64)
```

**Step 2 — the handler asks for the lock.** `backend/security/signup.go`

```go
core.AcquireLock(ctx, core.ActionSignUpByIP /* =1 */, 3405803821, 2 /* max waiters */)

// wait (5s, how long we will queue) and lease (15s, the daemon's deadline on us while we
// hold) are constants in backend/core/server_utils/locks.go, not per-call-site knobs.
```

**Step 3 — the frame on the wire.** 24 bytes:

```
 02 │ 00 01 │ 00 00 00 00 CB 00 71 2D │ 02 │ 13 88 │ 3A 98 │ <8-byte HMAC>
 ▲    ▲       ▲                         ▲    ▲       ▲
 │    │       │                         │    │       └─ lease_ms  = 15000
 │    │       │                         │    └───────── wait_ms   = 5000
 │    │       │                         └────────────── max_waiters = 2
 │    │       └──────────────────────────────────────── identifier = the IP
 │    └──────────────────────────────────────────────── action = 1 (sign-up by IP)
 └───────────────────────────────────────────────────── opcode LOCK_ACQUIRE
```

**Step 4 — the daemon decides.** `server_utils/src/lock/registry.rs:82`

```
key = (1, 3405803821)
        │
        ├─ no entry yet? create one (refuse with status 3 past max_keys)
        │
        ├─ try_acquire → SUCCESS ──────────────▶ status 0, hold the permit
        │
        └─ try_acquire → taken
              │
              ├─ waiters already == max_waiters ─▶ status 1  (never queues)
              │
              └─ queue, FIFO, up to wait_ms
                    ├─ turn arrives ─▶ status 0
                    └─ timed out ────▶ status 2
```

**Step 5 — reply.** `00 00 │ 00 │ 00 00` → correlation 0 (answering sequence 0), status 0
(granted), detail unused.

**Step 6 — the critical section**, now guaranteed to be alone on this key:

```
count distinct emails for this IP in the last 20 min   (signup.go:165)
   └─ already at max_emails_per_ip, and this email is new?  ─▶ release, 429
send the verification mail       (≤4s connect + ≤6s send)
INSERT the sign_up_request row
```

**Step 7 — release.** 9 bytes, `03 │ <HMAC>`. No payload: the connection identifies the lock.

Meanwhile B, queued at step 4, is handed the permit the instant A releases, and its step 6 count
now includes A's row.

---

## 6. When things go wrong

The design has one rule: **the permit lives in the connection task, so anything that ends that
task frees the lock.** Three failures, one mechanism.

```
(a) Lambda killed / crashed / timed out
        socket dies ─▶ read fails ─▶ task ends ─▶ guard dropped ─▶ next waiter runs
        (immediate — no waiting out a lease)

(b) Holder alive but wedged (never sends release)
        no frame for lease_ms ─▶ read deadline fires ─▶ task ends ─▶ guard dropped
        (this is the backstop; it is why Lease must exceed the SMTP timeouts)

(c) Daemon unreachable
        Go gets ErrLockUnavailable, NOT ErrLockBusy ─▶ sign-up answers 503
        (fails closed on purpose: an open mail relay is worse than a down endpoint)
```

The distinction in (c) is the whole reason the two errors are separate types:
`ErrLockBusy` is a real answer, `ErrLockUnavailable` is no answer, and each call site decides.

**What the lock does not protect against.** If the network partitions between backend and daemon
mid-critical-section, the daemon frees the key while the Go code is still working. Every
liveness-based lock has this hole. So the handler must remain safe to run twice — the lock makes
the race rare and orderly, it is not a correctness guarantee on its own.

---

## 7. Where the code is

| Concern | File |
|---|---|
| Listener, handshake, opcode dispatch | `src/service/server.rs:94` |
| Opcode table, frame widths, reply encoding | `src/service/protocol.rs` |
| Frame HMAC | `src/service/auth.rs` |
| Key registry, queueing, FIFO, pruning | `src/lock/registry.rs:82` |
| Acquire payload codec | `src/lock/protocol.rs` |
| Go client, connection pool, typed errors | `backend/core/lock_service.go:140` |
| Reply reading and correlation check | `backend/core/credit_rate_limiter.go:370` |
| IP → int64 | `backend/core/responses.go:71` |
| The sign-up call site | `backend/security/signup.go` |

---

## 8. What multiplexing changes

Today each held lock owns a connection and the credit limiter has its own, so a sign-up Lambda
holds two sockets. The plan in `PLAN_MULTIPLEXING.md` collapses that to one connection carrying
everything, with many requests in flight.

```
                    today                          after multiplexing
        ┌──────────────────────────┐      ┌──────────────────────────────┐
        │ conn 1: charges only     │      │ conn 1: charges AND locks,   │
        │ conn 2: one lock         │      │         many in flight,      │
        │ conn 3: another lock     │      │         replies out of order │
        └──────────────────────────┘      └──────────────────────────────┘
```

Three consequences worth knowing before reading that plan:

1. **`LOCK_RELEASE` grows** from 9 to 21 bytes — it must name `action + identifier + generation`,
   because the connection no longer identifies one lock.
2. **A generation counter appears.** With a shared connection, a late release from a goroutine
   that already gave up would otherwise free the lock another goroutine just took on the same
   key. The counter makes a stale release detectable.
3. **The lease stops being the read deadline.** Once charges share the socket they would refresh
   it forever, so it becomes an absolute timestamp taken at grant time, checked by the reader
   loop, and its expiry no longer kills the connection — killing it would drop every other lock
   and fail every pending request on that socket.

4. **The Go API changes.** `AcquireLock` returns `(*Lock, error)` instead of `(func(), error)`,
   so `defer releaseLock()` becomes `defer lock.Release()`. One call site today.

Failure case (a) above is unchanged: the daemon still tracks which locks belong to which
connection, so a dead socket still frees them instantly. What changes is the blast radius — one
socket error now frees *all* of that process's locks, which is why `AcquireLock` returns a handle
with a `Lost()` channel. That channel closes both when the connection dies and when the lease
elapses, and it is advisory either way: under a partition the holder may already be past the
check, so work inside a lock must stay idempotent regardless.

Each of these wire changes bumps the HMAC domain (`:v2` today, `:v3` when release widens).
Replies are not signed, so a version skew cannot be caught by the signature — bumping the domain
turns a mismatched peer into an immediate authentication failure instead of a client silently
misreading a reply that grew under it.
