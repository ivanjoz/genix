# Genix Server Utilities

One Rust process hosting three server-side services over two transports:

| Service | Transport | Port | Purpose |
|---|---|---|---|
| Credit rate limiter | Raw TCP, loopback | `server_utils` (default `127.0.0.1:14013`) | Atomic CPU/inference quota checks for the Go backend. |
| Lock service | Raw TCP, same port | `server_utils` | Serializes an action across concurrent Lambdas. |
| SSE bridge | HTTP (TLS via Nginx) | `sse_bridge.port` (default `14012`) | Relays agent events between the backend and browser tabs. |

The limiter and the lock share the port, the connection, and the handshake — nothing else. Each
opcode has its own frame width, its own codec, and its own module. That shared port is why its
address is the root-level `server_utils` key rather than something under `[rate_limit]`: it
belongs to the process, not to any one service inside it.

The bridge shares nothing with either but the process: the config load, the shutdown signal, and
the tokio runtime. No service calls into another.

Start with [LOCK_SERVICE_WALKTHROUGH.md](LOCK_SERVICE_WALKTHROUGH.md) — one sign-up request end
to end, with the exact bytes. Designs: [PLAN.md](PLAN.md) (rate limiter, including all binary
formats), [PLAN_LOCK_SERVICE.md](PLAN_LOCK_SERVICE.md) and
[PLAN_MULTIPLEXING.md](PLAN_MULTIPLEXING.md) (lock service),
[PLAN_SSE_BRIDGE.md](PLAN_SSE_BRIDGE.md) (bridge). Deployment:
[`../scripts/CONFIGURE_SERVER_UTILS.md`](../scripts/CONFIGURE_SERVER_UTILS.md).

> **One process, shared fate.** The rate limiter loads existing usage from ScyllaDB before
> admitting anything and exits when it cannot — which also stops the bridge. Deploy the backend
> tables (so `credit_usage` exists) before starting the daemon.

## Layout

`service/` owns everything the raw-TCP operations share — the listener, the handshake, the frame
HMAC and the opcode table. Each operation's own codec and logic live in its own tree, so adding
one touches the opcode table and nothing else:

```text
src/
├── main.rs      # spawns both transports, one shared shutdown signal
├── config.rs    # the only thing they share
├── service/     # the raw-TCP port: server (listener, handshake, opcode dispatch),
│                # protocol (opcode table), auth (frame HMAC)
├── limiter/     # opcode 0x01: quota.rs (RateLimiter + policy), protocol, aggregation,
│                # credits_blob, time_frame, storage
├── lock/        # opcodes 0x02/0x03: registry.rs (sharded key mutexes), protocol
└── bridge/      # token.rs (colbin + channel token), auth, channel, http (axum)
```

## Two secrets, split by purpose

Both are root-level keys in `config.toml` and must match the backend byte for byte:

| Key | Used for |
|---|---|
| `internal_apikey` | Service-to-service authentication: the TCP frame HMAC (`genix-server-utils:v3`) and the bridge's `X-Bridge-Auth` header (`sse-bridge:v1\|`). |
| `secret_phrase` | Verifying the browser's session token only (`usrToken:v1`). Nothing else in this crate reads it. |

Each use is domain-separated, so one key serving two protocols cannot produce interchangeable
tags. Splitting the two means the inter-service key can be rotated without invalidating every
live session token.

## Rate limiter behavior

- Authenticates persistent TCP connections with an eight-byte server nonce and sequence-bound
  HMAC-SHA256 frames.
- Atomically checks company and user limits for CPU and inference credits.
- Uses a token bucket for ten-second limits and fixed UTC hour/day counters.
- Aggregates every accepted charge into user/company and five-minute/daily in-memory records.
- Flushes only changed absolute records to `credit_usage` every 15 seconds.
- Fails closed when existing usage cannot be loaded from ScyllaDB.

Version one must run as a single active process. Two instances would have independent in-memory
quota state and must not write the same absolute rows.

## Configuration

Add `server_utils` and `[rate_limit]` to the project `config.toml`; the complete commented
example is in [`../config.example.toml`](../config.example.toml).

```toml
# The raw-TCP endpoint of the whole process, root level: the opcode decides which service
# answers, so the address is not the rate limiter's to own.
server_utils = "127.0.0.1:14013"

# Purpose: Configure process limits and the two global quota profiles.
[rate_limit]
flush_seconds         = 15
frame_timeout_seconds = 30
max_connections       = 1024
shards                = 0 # 0 uses the logical CPU count
# Requests one connection may have in flight at once. Multiplexing removed the backpressure that
# one-request-per-socket used to give for free, so it has to be stated.
max_inflight_per_connection = 64

company_cpu_10s       = 2000
company_inference_10s = 1000
company_cpu_1h        = 40000
company_inference_1h  = 10000
company_cpu_24h       = 200000
company_inference_24h = 20000

user_cpu_10s          = 1000
user_inference_10s    = 500
user_cpu_1h           = 20000
user_inference_1h     = 5000
user_cpu_24h          = 100000
user_inference_24h    = 10000
```

The twelve credit ceilings are the only settings here with no built-in default: a guessed quota
is worse than none, so the process refuses to start without them. Since that refusal is a
three-second crash loop under `Restart=always`, `scripts/configure_server_utils.py` writes these
defaults into `config.toml` when they are absent, rather than leaving the daemon to discover it.

The lock service adds process-wide ceilings only — per-action policy stays in the Go call sites:

```toml
# Purpose: Bound the daemon's memory; who locks what is decided by the backend.
[lock]
max_keys          = 100000
max_total_waiters = 4096
max_lease_ms      = 60000
```

The SSE bridge adds one small section:

```toml
# Purpose: Expose the bridge's HTTP port; the public URL is only read by the backend/frontend.
[sse_bridge]
url     = "https://genix-sse.example.com/"
port    = 14012
verbose = false
```

The process also reads root `secret_phrase`, root `internal_apikey`, and `[db].host`, `port`,
`name`, `user`, and `password`. Set `GENIX_CONFIG_FILE` to select a non-default TOML file. Every
setting can be overridden by its uppercase environment equivalent, such as
`RATE_LIMIT_USER_CPU_10S`, `SSE_BRIDGE_PORT`, or `DB_HOST`.

All quota values must be positive and nondecreasing from 10 seconds to one hour to 24 hours. The
24-hour values cannot exceed `uint32`, which is the largest persisted blob width.

`sse_bridge.url` is *not* parsed by this process — the backend and the frontend read it to decide
whether to use a bridge at all (empty, or equal to `aws.lambda_url`, means the backend serves its
own `/agent/stream`). The deployment script uses it for the Nginx `server_name`.

## Build and test

```bash
# Purpose: Compile and verify all protocol, codec, limiter, lock, and flush tests.
cd server_utils
cargo test
cargo build --release
```

`cargo test` also runs `tests/lock_tcp.rs`, which drives a real socket: that is where the claims
this design rests on are checked — that a queued acquire does not delay a charge sent after it,
that a lease expires while the connection stays busy, and that a dropped connection frees
everything it held.

Building needs a C compiler even though no crate here contains C: rustc shells out to `cc` to
link, and a `build.rs` is itself an executable that has to be linked before cargo can run it.
`../scripts/configure_server_utils.py` installs one when the host has none.

For a host that should compile nothing, build a static binary and ship it instead. `.cargo/
config.toml` pins `rust-lld` for the musl targets, which is also what makes cross-building arm64
work — the host `cc` can only link for the host:

```bash
# Purpose: Produce a dependency-free binary; runs on any Linux of that architecture.
cargo build --release --target x86_64-unknown-linux-musl
cargo build --release --target aarch64-unknown-linux-musl
```

Before starting the daemon, deploy the backend tables so the generated Genix controller creates
`credit_usage`:

```bash
# Purpose: Regenerate/validate controllers and deploy tables through the normal Genix workflow.
cd scripts
go run . generate_controllers
go run . check_tables
```

Run locally from `server_utils/` (it finds `../config.toml`):

```bash
# Purpose: Enable detailed request and flush diagnostics during local development.
RUST_LOG=genix_server_utils=debug cargo run
```

## SSE bridge HTTP contract

```
navegador                     bridge                        backend (Lambda)
   |--- GET /sse?ch= ---------->| registra el canal
   |<-- data:{bridgeReady} -----| handshake
   |                            |<--- POST /publish ---------| evento (no bloquea)
   |<-- data:{agentStatus} -----|
   |                            |<--- POST /rpc -------------| comando (BLOQUEA)
   |<-- data:{ID:7,navigate} ---|
   |--- POST /in {ID:7,...} --->|
   |                            |---- 200 {Kind,Payload} --->| request() retorna
```

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/sse?ch=<token>` | session token | Opens the stream. First frame `{"Type":"bridgeReady"}`, keepalive comment every 20s. |
| `POST` | `/in?ch=<token>` | session token | Browser reply `{ID,Type,Payload}`. Wakes the `/rpc` waiting on that `ID`. |
| `POST` | `/publish` | service HMAC | `{Channel,Message,WaitMs}` → `{Delivered}`. Does not block. |
| `POST` | `/rpc` | service HMAC | `{Channel,ID,Message,TimeoutMs,WaitMs}` → `{Kind,Payload}`. Blocks until the reply. |
| `GET` | `/health` | — | `{Ok,Channels,UptimeSeconds}`. |

Messages are opaque JSON and **nothing is buffered**: a message for a disconnected tab is
dropped (`Delivered:false`). The bridge holds no business logic and never touches ScyllaDB.

### Channel token

A channel is one browser tab, named by a single string:

```
bytes = uvarint(companyID) ‖ uvarint(userID) ‖ 6 random bytes (tab)
token = base64url(bytes), unpadded
```

For ordinary ids that is **11 characters** (`7/42` → `Byo3bFBobzE`). The decoder rejects
non-canonical encodings, which makes the token bijective with the triple — that is what lets it
be the registry key directly: two distinct strings can never name the same channel.

**It is an identifier, not a credential.** The browser still proves who it is with its session
token, and the bridge checks that the identity *inside* the channel token matches the
authenticated one. Without that cross-check, editing the company id would attach a client to
another tenant's stream.

The format is mirrored in `src/bridge/token.rs`, `backend/agent/channel.go`, and
`frontend/core/agent/channel.ts`; the vectors in `token.rs` pin all three byte for byte.

## TCP contract

After accepting a connection, the server writes an eight-byte random nonce. Every subsequent
request is `[opcode:1][payload][hmac:8]`, big-endian. The opcode routes the payload; it is not a
shared frame shape, and the three operations have no field in common.

| Op | Name | Payload | Frame |
|---|---|---|---|
| `0x01` | `CHARGE_CREDITS` | company `u24` · user `u24` · API group `u8` · CPU `u16` · inference `u16` | 20 |
| `0x02` | `LOCK_ACQUIRE` | action `u16` · identifier `i64` · max_waiters `u8` · wait_ms `u16` · lease_ms `u16` | 24 |
| `0x03` | `LOCK_RELEASE` | action `u16` · identifier `i64` · generation `u16` | 21 |

`0x00` stays unassigned so an all-zero frame cannot route. 252 opcodes remain free; new *use
cases* for the lock cost none of them, since they are namespaced by the `u16` action instead.

The HMAC covers the opcode and payload plus the connection nonce and the implicit frame sequence,
so a frame can be replayed neither as itself nor as a different operation. Authentication,
malformed-frame, unknown-opcode, initialization, and transport failures close the connection.

The domain string is bumped on every wire change — `genix-server-utils:v3` today. Replies are not
themselves authenticated, so a version skew cannot be caught by the signature: without the bump
an old client would keep authenticating fine and then misread a reply that grew under it.

### Replies are multiplexed

Requests travel in order; replies do not. An acquire can sit in a lock queue for seconds while
charges sent after it are answered immediately. Every reply is therefore five bytes:

```
[correlation:u16][status:u8][detail:u16]
```

`correlation` is the low 16 bits of the request's frame sequence, echoed back. Nothing extra
travels on the wire to carry it — the sequence already exists for the HMAC — and it is what lets
one connection serve many callers at once. `detail` carries the lock generation on a granted
acquire and is zero everywhere else.

Zero is success for every opcode. `CHARGE_CREDITS` rejections use the low five bits to identify
the scope, time window, and exhausted credit types. Lock replies are `1` queue full, `2` wait
timed out, `3` daemon at capacity, `4` protocol misuse (releasing a lock this connection does not
hold, or presenting a superseded generation). `0xFF` means the daemon could not answer at all; it
is deliberately not a valid verdict for any opcode, so a client applies its own policy — charges
fail open, sign-up locks fail closed.

The client must assign a sequence and write its frame atomically. Two callers taking 5 and 6 but
writing 6, 5 would desynchronize the HMAC and every later frame would fail.

## Lock behavior

One holder per `(action, identifier)` — every lock is mutual exclusion. The daemon interprets
neither field: the Go call sites decide what is being serialized (a client IP, a company, a
packed pair), which is what makes one service cover every case in the project.

**Ownership is bound to the connection.** The permit lives in the connection task, so a
disconnect, a crash and a killed Lambda all free the lock at once — no sweeper, and no waiting
out a lease. One connection may hold several keys, and losing it frees all of them.

**The lease is an absolute deadline**, stamped when the lock is granted and checked by the
reader. It is the backstop for a holder that stays connected but wedged. It is not the socket's
read timeout: with charges and locks sharing one connection, arriving traffic would push that
forward forever and a wedged holder would keep its key. Expiry drops that one lock and leaves the
connection running, because killing it would take every other lock and every pending request with
it. While a connection holds anything, the idle timeout does not apply — a caller legitimately
holding a 30s lease is quiet, not dead.

**Each grant carries a generation**, returned in the reply's `detail` and required by the
release. Without it, a release from a caller that already gave up would end whichever hold
replaced it on that key — a real risk now that several callers share one connection. The counter
is registry-wide rather than per-key: an idle key's entry is pruned, so a per-key counter would
restart at zero and the stale release would match exactly.

`max_waiters` is checked before queueing: past the ceiling a caller is refused immediately rather
than parked, because with an unbounded queue the wait itself becomes the denial of service.
`rate_limit.max_inflight_per_connection` bounds the other direction — multiplexing removed the
backpressure that one-request-per-socket used to provide for free.

Locks are in-memory: a restart drops all of them, and two daemon instances would hand the same
key to two holders. Single active process, same as the limiter.

A lock orders callers; it does not make them safe. A partition can free a key while its holder is
still working, which is true of every liveness-based lock, so work inside one must remain safe to
run twice.

## Deploying

`sudo python3 scripts/configure_server_utils.py` compiles the binary, installs the systemd units,
and writes the bridge's Nginx vhost (HTTP/3 when a certificate exists and Nginx was built with
it) on this host. It asks nothing — everything comes from `config.toml`. It installs a C compiler
if the host has none, and after starting the service it probes `/health` rather than trusting
`systemctl restart`: this daemon exits when ScyllaDB is unreachable, and with `Restart=always`
that would otherwise look identical to a healthy start. Full details, including the generated
unit and the three non-negotiable Nginx streaming settings, are in
[`../scripts/CONFIGURE_SERVER_UTILS.md`](../scripts/CONFIGURE_SERVER_UTILS.md).

The raw TCP listener should remain on loopback or a private network. HMAC authenticates messages,
but the protocol does not encrypt traffic. The bridge's HTTP port speaks plain HTTP; Nginx
terminates TLS in front of it.

## Go charging rules

The backend uses uncompressed bytes and binary KiB (`1 KiB = 1024 bytes`):

- GET groups `0/1/2` use response sizes `<32 KiB`, `32..256 KiB`, and `>256 KiB`.
- POST groups `3/4/5` use request-body sizes with the same boundaries.
- GET CPU usage is two base credits for the first 8 KiB, then one credit per started 16 KiB.
- POST CPU usage is five base credits for the first 8 KiB, then one credit per started 8 KiB.
- Successful inference usage is one credit per started 8 KiB of provider input and two credits per
  started 8 KiB of provider output.

Authenticated private POST requests are admitted before their handler runs. Successful GET
responses are admitted after serialization because their response size is not known earlier.
