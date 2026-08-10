# PLAN — Generic lock service on the server-utils TCP port

Status: **implemented**. Three deviations from the plan as written, all decided while building:

1. **`auth.rs` moved to `service/`** instead of staying under `limiter/`. The plan's own reason
   for leaving it — "it is transport authentication, not limiter logic" — is the argument for
   moving it, since both opcodes now authenticate through it.
2. **`core.Env.REQ_IP` was kept.** The plan said to delete it. It feeds `MakeReqLog`, which is
   built on six sibling per-request globals (`REQ_USER_AGENT`, `REQ_PATH`, `User`, `SessionLogs`,
   `StartTime`). Removing one of them means re-plumbing that logging subsystem, which is a larger
   and unrelated refactor. `HandlerArgs.ClientIP` is now the authoritative per-request value and
   the only one the lock uses; `Env.REQ_IP` is also set in server mode now, which it was not
   before. The pre-existing limitation of the log record is unchanged, not worsened.
3. **The client IP comes from `X-Real-IP`, not `X-Forwarded-For`.** `configure_server.py:1020`
   builds XFF with `$proxy_add_x_forwarded_for`, which *appends* — so a client that sends its own
   header lands first in the list and the plan's "first hop of X-Forwarded-For" would have been
   bypassable with one curl flag. `X-Real-IP` is written from `$remote_addr` (`:1019`) and cannot
   be forged through the proxy.

Goal: give the Go backend a way to serialize an operation across concurrent Lambdas. The daemon
stays generic — it knows only `action + identifier` — and every policy decision (which action,
which identifier, what to do when the daemon is down) lives in the Go call sites.

The first consumer is `POST p-signup-request`, which today has a read-then-write race that
parallel Lambdas defeat.

---

## 1. Why

`security/signup.go:PostSignUpRequest` reads the state of an email, decides "nothing live exists",
then sends and inserts. Scylla has no transaction there and the ORM exposes no LWT (`IF NOT
EXISTS` appears only in DDL, `genix-orm/scylla/init.go`). N Lambdas invoked in parallel all read
"nothing live", all send, all insert. `§5` of `PLAN_SIGNUP_REGISTRATION.md` names the one-live-
request check as the only brake, and that check is exactly what the race removes.

A lock fixes ordering. It does **not** by itself stop abuse: 1000 requests with 1000 different
emails produce 1000 different keys and serialize nothing. That is why the signup work in §6 also
adds a per-IP counter — the lock is what makes that counter correct.

---

## 2. Protocol — opcode routing on port 14013

The lock shares the limiter's listener and its connection. It does **not** share a frame shape:
each opcode is its own independent message, with its own length, its own codec, and its own
module. The opcode is a routing header, nothing more. Pre-alpha, so the format changes in place;
there is no compatibility shim.

```
handshake:  server -> 8-byte random nonce      (unchanged)
frame:      [opcode:1][ ...own layout... ][hmac:8]
reply:      1 byte, meaning is per-opcode
```

`0x01 CHARGE_CREDITS` — 20 bytes. Today's frame with a routing byte in front, otherwise untouched.

```
0        1          4          7   8      10      12                    20
+--------+----------+----------+---+------+-------+---------------------+
|  0x01  |company:u24|user:u24 |grp| cpu  | infer |       hmac          |
+--------+----------+----------+---+------+-------+---------------------+
```

`0x02 LOCK_ACQUIRE` — 24 bytes. Shares no field with the frame above.

```
0        1          3                     11   12      14      16       24
+--------+----------+---------------------+----+-------+-------+--------+
|  0x02  |action:u16|   identifier:i64    |maxw|wait_ms|leasems|  hmac  |
+--------+----------+---------------------+----+-------+-------+--------+
```

`0x03 LOCK_RELEASE` — 9 bytes. No payload at all: the connection already identifies the lock.

```
0        1                                 9
+--------+---------------------------------+
|  0x03  |              hmac               |
+--------+---------------------------------+
```

Read loop: read one byte, `match opcode { .. => size }`, `read_exact` the remainder, verify, then
dispatch to `limiter::` or `lock::` — which parse their own bytes into their own structs and never
see each other's. An unknown opcode closes the connection, like any other malformed frame.

What is genuinely shared is only what you asked to share: the port, the TCP connection, the
handshake nonce, and the frame sequence counter that binds the HMACs to that connection.

The HMAC covers `opcode ‖ payload` plus the connection nonce and the frame sequence, so
`limiter/auth.rs` changes only its domain constant: `genix-rate-limiter:v1` →
`genix-server-utils:v1`. The layout no longer matches the old domain, and reusing the string
would let an old client's frame authenticate under a new interpretation.

**No `permits` field.** Every lock is mutual exclusion, one holder. A semaphore was considered
and rejected: no current call site wants concurrent holders.

`action` is a Go constant enum; `identifier` is any int64 the call site chooses (an IP, a
company, a client, a packed pair). The daemon interprets neither — that is what keeps it generic.

### Reply bytes for `0x02` / `0x03`

| Byte | Meaning |
|---|---|
| `0x00` | Acquired (`0x02`) / released (`0x03`) |
| `0x01` | Busy: `max_waiters` already queued, nothing was queued |
| `0x02` | Wait timeout: `wait_ms` elapsed without reaching the front |
| `0x03` | Server capacity: `lock.max_keys` or `lock.max_total_waiters` reached |
| `0x04` | Protocol misuse: acquire while already holding, or release while not holding |

Zero means success for every opcode, matching the limiter's existing convention.

---

## 3. Ownership: the connection *is* the lock

The permit is held by the connection task and dropped when that task ends. Three failure modes
collapse into one code path:

- Client disconnects, crashes, or its Lambda is killed → socket closes → guard drops → the next
  waiter proceeds immediately.
- Client holds the lock and goes silent → while holding, the connection's read deadline is
  `lease_ms` instead of `frame_timeout` → deadline fires → connection closed → guard drops.
- Client sends `LOCK_RELEASE` → guard drops, connection returns to the pool.

So the lease needs no timer task, no sweeper, and no release token to validate: **the lease is
the read deadline.** `lease_ms` is clamped to `lock.max_lease_ms`.

Consequence: **one lock per connection at a time**. Sending `LOCK_ACQUIRE` while already holding
is a protocol error (`0x04`), not a nested lock.

---

## 4. Rust changes

### Module layout

| Path | Change |
|---|---|
| `src/service/server.rs` | Moved from `src/limiter/server.rs`. Owns the accept loop, handshake, opcode dispatch, per-connection lock guard. |
| `src/service/protocol.rs` | New. Opcode enum, frame sizes, `LOCK_ACQUIRE` codec. |
| `src/limiter/` | Keeps `quota`, `aggregation`, `credits_blob`, `storage`, `time_frame`, `auth`, and the `CHARGE_CREDITS` codec only. |
| `src/lock/registry.rs` | New. The sharded key→entry map and the acquire/release algorithm. |
| `src/main.rs` | Passes the `LockRegistry` alongside the `RateLimiter` into `service::server::run`. |

`limiter/auth.rs` stays where it is and is used by both — it is transport authentication, not
limiter logic.

### Registry

```rust
struct LockEntry { semaphore: Semaphore /* 1 permit */, queued: AtomicU32 }
struct LockRegistry { shards: Vec<Mutex<HashMap<(u16, i64), Arc<LockEntry>>>>, total_queued: AtomicU32 }
```

Sharded by `hash(action, identifier)`, reusing `config.shard_count`. Acquire:

1. Lock the shard, get-or-insert the `Arc<LockEntry>`, refuse with `0x03` if inserting would pass
   `lock.max_keys`, clone the `Arc`, unlock the shard. The shard mutex is never held across an
   await.
2. `try_acquire()` first. Success → acquired without ever touching `queued`, which is the common
   uncontended path.
3. Otherwise increment `queued` (and `total_queued`); if the prior value is `>= max_waiters`, or
   the global cap is passed, decrement and reply `0x01` / `0x03` without queueing.
4. `timeout(wait_ms, semaphore.acquire())`. Tokio's `Semaphore` is FIFO, so waiters are served in
   arrival order. Decrement `queued` on every exit path.
5. Release (explicit frame, disconnect, or lease expiry) drops the permit, then locks the shard
   and removes the entry when `Arc::strong_count == 2` (the map plus this local handle) and
   `queued == 0`. Checking with the local handle in hand is what makes the removal race-free
   against a waiter that has cloned the `Arc` but not yet incremented `queued`.

### Config

New `[lock]` section, safety caps only — never per-action policy, which belongs to Go:

```toml
[lock]
max_keys          = 100000   # live keys before the daemon replies "server capacity"
max_total_waiters = 4096     # queued waiters across all keys
max_lease_ms      = 60000    # ceiling applied to every client-supplied lease
```

Loaded in `config.rs` with the existing `optional_u64`/`optional_usize` helpers and
`LOCK_MAX_KEYS`-style environment overrides, matching how `[rate_limit]` is read.

### Tests

- `service/protocol.rs`: opcode→size table, exact `LOCK_ACQUIRE` field offsets.
- `limiter/auth.rs`: regenerated Go-matching vectors under the new domain and the new payloads.
- `lock/registry.rs`: mutual exclusion, FIFO order, `max_waiters` rejection without queueing,
  `wait_ms` timeout, guard-drop release, and entry removal when the last user leaves.

---

## 5. Go changes — `backend/core/`

### `credit_rate_limiter.go`

`makeFrame` gains the opcode prefix and the domain constant changes. The client keeps its single
mutex-guarded connection: charges stay short and strictly request/response.

### `lock_service.go` (new)

Locks cannot share that connection — a blocking acquire would head-of-line-block every credit
charge in the process. Harmless on Lambda (one request per container), fatal in VPS server mode.
So the lock client owns a **small connection pool**: a buffered channel of idle connections, dial
on empty, discard on any error.

```go
// Actions are a central enum so two features can never collide on a key space.
const ActionSignUpByIP uint16 = 1

type LockOptions struct {
    MaxWaiters uint8
    Wait       time.Duration // client-side patience; must exceed MaxWaiters × expected hold
    Lease      time.Duration // server-side read deadline while holding
}

// AcquireLock returns a release func that is idempotent and safe to defer.
func AcquireLock(ctx context.Context, action uint16, identifier int64, opts LockOptions) (func(), error)
```

Per-call failure policy needs no parameter — the caller distinguishes the errors:

| Error | Meaning | Typical handling |
|---|---|---|
| `ErrLockBusy` | Queue full or wait timeout — a real answer | Reject the client, 429 |
| `ErrLockUnavailable` | Daemon unreachable — no answer at all | Caller chooses: proceed unlocked, or 503 |

`ErrLockUnavailable` is deliberately *not* pre-decided the way `chargeConfiguredCredits` fails
open. Signup will fail closed (§6); most other call sites should fail open.

### `HandlerArgs.ClientIP`

`core.Env.REQ_IP` cannot be used. It is a process-wide global set only in `main.go:69` on the
Lambda path — `LocalHandler` never sets it, and in VPS server mode concurrent goroutines would
race on it. The client IP becomes a per-request field:

- `LambdaHandler` / `LambdaStreamingHandler`: `request.RequestContext.HTTP.SourceIP`.
- `LocalHandler`: first hop of `X-Forwarded-For`, else `request.RemoteAddr` minus the port.

`core.Env.REQ_IP` is then removed; `core/logs.go:119` reads the new field instead.

### `core.ClientIPKey(ip string) (int64, bool)`

IPv6 must be keyed on its prefix, not the address: a single residential customer is handed a
whole /64 (often a /56), so per-address limiting is free to bypass.

- IPv4 → `int64` of the 32-bit value, always below 2^32.
- IPv6 → top 64 bits of the address, shifted right one bit to stay positive. That keys the /63,
  still far narrower than the /56 a customer receives. Real IPv6 prefixes start at `2000::/3`, so
  the result lands above 2^60 and cannot collide with the IPv4 range in practice.

---

## 6. First consumer — `POST p-signup-request`

### Rule

One IP may register **at most 5 distinct emails per 20 minutes**, both values from `config.toml`.
Retrying the *same* email does not consume budget — the existing resend cooldown covers that.

Known and accepted: users behind corporate NAT or CGNAT share an IPv4 and will occasionally hit
this limit legitimately.

### Table

`security/types/signup_requests.go` gains `IP int64` (from `ClientIPKey`) plus a second local
index on it, so "this IP's requests this week" is a partition-local lookup:

```go
Indexes: []db.Index{
    {Type: db.TypeLocalIndex, Keys: db.Cols(e.Email)},
    {Type: db.TypeLocalIndex, Keys: db.Cols(e.IP)},
},
```

The index returns every row for that IP in the week; the 20-minute filter is applied in Go. Row
count per IP is bounded by the very limit being enforced (~5/20min → ~2.5k/week worst case).
After the change: `cd scripts && go run . check_tables`.

### Handler

```
1. validate the email format
2. ipKey, ok := core.ClientIPKey(req.ClientIP)     ; !ok -> 400
3. release, err := core.AcquireLock(ctx, ActionSignUpByIP, ipKey, opts)
     ErrLockBusy        -> 429
     ErrLockUnavailable -> 503        (fail closed, see below)
   defer release()
4. count distinct emails for ipKey in the last 20 min, both week partitions
   if this email is new and the count is already at the limit -> 429
5. existing logic unchanged: companyExistsWithEmail, findLatestSignUpRequestByEmail,
   resend-or-create, send, insert
```

Fail closed on `ErrLockUnavailable`: returning 503 while the daemon is down is far cheaper than
leaving an open email-spam relay, and this endpoint is on no critical path.

### Lock parameters, and why the email send stays inside

The SMTP send happens inside the critical section, because the handler deliberately delivers
before persisting (a row written before a failed send would block every retry until it expired).
That makes the hold time SMTP latency — seconds, not milliseconds. So:

```go
LockOptions{MaxWaiters: 2, Wait: 5 * time.Second, Lease: 15 * time.Second}
```

A shallow queue is correct here rather than a compromise: parallel requests from one IP for one
email are the abuse pattern, so answering the extras with a fast 429 is the desired behavior, not
a degradation. `Wait` must stay below the Lambda function timeout.

### Config

```toml
[sign_up]
max_emails_per_ip = 5
window_minutes    = 20
```

Read into `Env` through the existing `file.RateLimit`-style struct in `core/security.go`, with
`SIGNUP_MAX_EMAILS_PER_IP` / `SIGNUP_WINDOW_MINUTES` environment overrides.

---

## 7. Docs, deployment, ordering

Breaking wire change: **the Rust daemon and the Go backend must be deployed together.** An old
backend talking to a new daemon fails HMAC verification and its connection is closed — it fails
open on charges, so the symptom is silently unmetered traffic rather than an outage. Worth
watching for on the first deploy.

Updates required:

- `server_utils/README.md` — the "Rate limiter TCP contract" section becomes the opcode table;
  add the lock behavior and `[lock]` config.
- `config.example.toml` and `config.toml` — `[lock]` and `[sign_up]`.
- `AGENTS.md:55` — the server-utils line now describes three services on two ports.
- `PLAN_SIGNUP_REGISTRATION.md:§5` — the "no throttle at all" note is now false.

Suggested order, each step compiling and testable on its own:

1. Rust: opcode routing with only `CHARGE_CREDITS`, plus the domain bump. Go: matching frame
   change. Nothing behavioral — proves the reroute in isolation.
2. Rust: `lock/registry.rs` and the two lock opcodes, with unit tests.
3. Go: `lock_service.go` pool and typed errors, plus `HandlerArgs.ClientIP` and `ClientIPKey`.
4. Table column, index, `check_tables`.
5. Signup handler and config.

---

## 8. Out of scope

- **A generic quota opcode.** A `QUOTA_CHECK(action, identifier, limit, window_ms)` in-memory
  counter would remove the Scylla index and query entirely, but it loses its state on daemon
  restart and leaves no audit trail. The per-IP counter stays in Scylla; the lock is what makes
  it correct. Reconsider this if a second call site wants the same shape.
- **Multiple daemon instances.** Locks are in-memory, so this stays single-instance — the same
  constraint the rate limiter already documents.
- **Lock durability across daemon restarts.** A restart drops all locks. The TOCTOU windows being
  protected are ~1s, so the exposure is one restart-width of unguarded requests.
- **Reentrancy / lock upgrades / nested locks.** One lock per connection, no exceptions.
