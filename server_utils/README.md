# Genix Server Utilities

One Rust process hosting four server-side services over two transports:

| Service | Transport | Port | Purpose |
|---|---|---|---|
| Credit rate limiter | Raw TCP, loopback | `server_utils` (default `127.0.0.1:14013`) | Atomic CPU/inference quota checks for the Go backend. |
| Lock service | Raw TCP, same port | `server_utils` | Serializes an action across concurrent Lambdas. |
| Request log | Raw TCP, same port | `server_utils` | One row per finished request, plus the code lines that failed. |
| SSE bridge | HTTP (TLS via Nginx) | `sse_bridge.port` (default `14012`) | Relays agent events between the backend and browser tabs. |

The limiter, the lock and the request log share the port, the connection, and the handshake —
nothing else. Each opcode has its own frame width, its own codec, and its own module. That shared port is why its
address is the root-level `server_utils` key rather than something under `[rate_limit]`: it
belongs to the process, not to any one service inside it.

The bridge shares nothing with either but the process: the config load, the shutdown signal, and
the tokio runtime. No service calls into another.

Start with [LOCK_SERVICE_WALKTHROUGH.md](LOCK_SERVICE_WALKTHROUGH.md) — one sign-up request end
to end, with the exact bytes. Designs: [PLAN.md](PLAN.md) (rate limiter, including all binary
formats), [PLAN_LOCK_SERVICE.md](PLAN_LOCK_SERVICE.md) and
[PLAN_MULTIPLEXING.md](PLAN_MULTIPLEXING.md) (lock service),
[PLAN_SSE_BRIDGE.md](PLAN_SSE_BRIDGE.md) (bridge). Deployment:
[`../scripts/configure/CONFIGURE_SERVER_UTILS.md`](../scripts/configure/CONFIGURE_SERVER_UTILS.md).

> **One process, shared fate.** The rate limiter loads existing usage from ScyllaDB before
> admitting anything and exits when it cannot — which also stops the bridge. Deploy the backend
> tables (including `credit_usage`, `company_credit_budget`, `user_logs`, `request_errors` and `server_metrics`) before
> starting the daemon. The request log and the metrics collector are the two halves that do *not*
> share that fate: they drop rows rather than propagate a failure, because taking the process down
> would stop everything else.

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
├── limiter/     # opcodes 0x01/0x05: charging and company-budget mutation,
│                # quota, protocol, aggregation, credits_blob, time_frame, storage
├── lock/        # opcodes 0x02/0x03: registry.rs (sharded key mutexes), protocol
├── reqlog/      # opcode 0x04: protocol (the one variable-length payload), errors
│                # (ten-minute write suppression), writer (batching, fails open)
├── sysmetrics/  # no opcode: samples the machine once a second and writes the peak
│                # of each five-second window to server_metrics. collector (/proc +
│                # cgroup v2), writer (the tick loop and the insert)
└── bridge/      # token.rs (colbin + channel token), auth, channel, http (axum)
```

## Two secrets, split by purpose

Both are root-level keys in `config.toml` and must match the backend byte for byte:

| Key | Used for |
|---|---|
| `internal_apikey` | Service-to-service authentication: the TCP frame HMAC (`genix-server-utils:v5`) and the bridge's `X-Bridge-Auth` header (`sse-bridge:v1\|`). |
| `secret_phrase` | Verifying the browser's session token only (`usrToken:v1`). Nothing else in this crate reads it. |

Each use is domain-separated, so one key serving two protocols cannot produce interchangeable
tags. Splitting the two means the inter-service key can be rotated without invalidating every
live session token.

## Rate limiter behavior

- Authenticates persistent TCP connections with an eight-byte server nonce and sequence-bound
  HMAC-SHA256 frames.
- Atomically checks company/user burst and hourly limits plus company-configured daily/monthly budgets.
- Derives each user's daily allowance as 50% of its company's CPU and inference allowances.
- Requires an explicitly activated current UTC month; a new month stays blocked until `SET_CURRENT`.
- Aggregates every accepted charge into user/company and five-minute/daily in-memory records.
- Flushes only changed absolute records to `credit_usage` every 15 seconds.
- Fails closed in the Go backend for quota exhaustion and daemon/storage unavailability.

Version one must run as a single active process. Two instances would have independent in-memory
quota state and must not write the same absolute rows.

## Configuration

Add `[server_utils]` and `[rate_limit]` to the project `config.toml`; the complete commented
example is in [`../config.example.toml`](../config.example.toml).

```toml
# The raw-TCP endpoint of the whole process, its own section: the opcode decides which service
# answers, so the address is not the rate limiter's to own.
#
# `host` is what the CLIENT dials; `public` is what the DAEMON binds — true is 0.0.0.0, false is
# 127.0.0.1. They are separate because behind NAT they cannot be one value: a cloud VM's public
# IP is never on its own interface, so binding it fails with EADDRNOTAVAIL. With public = false
# the client ignores `host` and dials loopback.
#
# public = true puts the port on the open internet. Frames are HMAC-authenticated but NOT
# encrypted, so it is only worth it when the backend runs off-box (Lambda, for instance).
[server_utils]
host   = "127.0.0.1"
port   = 14013
public = false

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

user_cpu_10s          = 1000
user_inference_10s    = 500
user_cpu_1h           = 20000
user_inference_1h     = 5000
```

The eight burst/hour ceilings are the only settings here with no built-in default: a guessed quota
is worse than none, so the process refuses to start without them. Since that refusal is a
three-second crash loop under `Restart=always`, the nested Server Utils installer writes these
defaults into `config.toml` when they are absent, rather than leaving the daemon to discover it.

The lock service adds process-wide ceilings only — per-action policy stays in the Go call sites:

```toml
# Purpose: Bound the daemon's memory; who locks what is decided by the backend.
[lock]
max_keys          = 100000
max_total_waiters = 4096
max_lease_ms      = 60000
```

The request log adds a section where every key has a default, so omitting it entirely means "on,
with these" rather than a refusal to start:

```toml
# Purpose: One row per finished request; a month of history, then the partition expires.
[request_log]
enabled             = true
ttl_days            = 30
flush_ms            = 1000
max_batch           = 128
error_cache_seconds = 600
error_cache_entries = 20000
queue_capacity      = 8192
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

All quota values must be positive and nondecreasing from ten seconds to one hour. Daily and monthly
entitlements are stored per company in `company_credit_budget`, not in this file.

`sse_bridge.url` is *not* parsed by this process — the backend reads it for service-to-service
publishing and the deployment script uses it for the Nginx `server_name`. The frontend gets the
matching public URL from the selected `[[endpoints]].bridge`; omitting that field means the
selected backend serves its own `/agent/stream`.

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
`../scripts/configure/configure_server_utils.py` installs one when the host has none.

For a host that should compile nothing, build a static binary and ship it instead. `.cargo/
config.toml` pins `rust-lld` for the musl targets, which is also what makes cross-building arm64
work — the host `cc` can only link for the host:

```bash
# Purpose: Produce a dependency-free binary; runs on any Linux of that architecture.
cargo build --release --target x86_64-unknown-linux-musl
cargo build --release --target aarch64-unknown-linux-musl
```

Every versioned [Genix GitHub Release](https://github.com/ivanjoz/genix/releases) also publishes
these static outputs as `genix-server-utils_linux_amd64` and
`genix-server-utils_linux_arm64`. Downloading `latest` is convenient for a manual install; replace
`latest/download` with `download/vX.Y.Z` to pin production automation to an immutable release.

```bash
# Map the Linux machine name to the release asset suffix.
case "$(uname -m)" in
  x86_64) release_architecture=amd64 ;;
  aarch64|arm64) release_architecture=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# Download the public binary and the manifest without requiring a GitHub token.
release_base_url=https://github.com/ivanjoz/genix/releases/latest/download
release_asset="genix-server-utils_linux_${release_architecture}"
curl --fail --location --output "$release_asset" "${release_base_url}/${release_asset}"
curl --fail --location --output SHA256SUMS "${release_base_url}/SHA256SUMS"

# Verify exact release bytes before making the daemon executable.
grep " ${release_asset}$" SHA256SUMS | sha256sum --check --strict
chmod 0755 "$release_asset"
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
| `0x04` | `LOG_REQUEST` | `[length:u16]` then date `i16` · request `i64` · route `i16` · frame `u8` · company `u24` · user `i32` · elapsed `u16` · errors `u8`, then per error: id `i32` · line `u8`+bytes · text `u16`+bytes | ≤ 1 110 |

`0x00` stays unassigned so an all-zero frame cannot route. 251 opcodes remain free; new *use
cases* for the lock cost none of them, since they are namespaced by the `u16` action instead.

`LOG_REQUEST` is the exception to both rules the other three share. It is **length-prefixed**,
because it carries strings, and it is **never answered**, because making a response wait for an
acknowledgement that a log row was stored would put this daemon's latency on the critical path of
every request in the system. The client writes the frame and returns; the sequence still advances,
since that is what the HMAC is bound to. The length header is inside the signed bytes, and anything
declaring more than the ceiling closes the connection — a length is an instruction from an
unauthenticated peer until the tag at the end says otherwise.

A malformed `0x04` payload is discarded with a warning rather than closing the connection, unlike
every other opcode. The others decide whether a request is admitted, so a frame the two sides
disagree about is a reason to stop talking; a log row is not worth taking down the charges and
locks sharing that socket.

The HMAC covers the opcode and payload plus the connection nonce and the implicit frame sequence,
so a frame can be replayed neither as itself nor as a different operation. Authentication,
malformed-frame, unknown-opcode, initialization, and transport failures close the connection.

The domain string is bumped on every wire change — `genix-server-utils:v5` today. Replies are not
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
is deliberately not a valid verdict for any opcode. Credit charges and budget mutations fail
closed; call sites for locks retain their operation-specific policy.

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

## Request log behavior

- One row per finished request in `user_logs`, batched into unlogged batches every `flush_ms` or
  at `max_batch`, whichever comes first.
- One row per distinct failing code line in `request_errors`. The code line is the identity, not
  the message: two failures at `responses.go:539` are the same error however differently they
  phrase themselves, which is what keeps that table bounded by the codebase instead of by traffic.
- A code line written under `error_cache_seconds` ago is not written again. Ten minutes of
  staleness on the preview costs nothing, because the current message is already in CloudWatch
  under the request id that referenced it.
- `user_logs` rows are written `USING TTL ttl_days`. The partition is the date, so a whole day
  expires together and Scylla drops it wholesale. `request_errors` has no TTL — a code line that
  failed once is worth keeping until it is rewritten.
- **Fails open, everywhere.** A full queue drops the record and counts it; a failed write logs a
  warning and drops the batch; statements that cannot be prepared at startup disable the writer and
  leave the process running. None of it propagates, because a log row is never worth stopping the
  limiter and the bridge for.

The dashboard reads through one index: `frame_route_company_agg`, packed frame-major so a
fifteen-minute slice of a day is a single contiguous clustering range and a poll reads forward
instead of rereading the day.

```
bits 47..40  frame      0..95, four per hour
bits 39..24  route_id   the generated number, backend/core/api_routes.generated.go
bits 23..0   company_id
```

The packing is written twice — `src/reqlog/protocol.rs` writes the column and
`backend/core/types/user_logs.go` ranges over it — and the vectors in both test files pin them
together. A drift there produces rows that look right and a chart that is quietly wrong.

## Server metrics behavior

The one part of this daemon nothing calls into: it just ticks. Design in
[PLAN_SERVER_METRICS.md](PLAN_SERVER_METRICS.md), schema in
`backend/core/types/server_metrics.go`.

- One row every `row_seconds` in `server_metrics`, partitioned by unix day and clustered by the
  slot within it (`secondsIntoDay / 5`, so 0..17279 and comfortably inside the int16 key).
- **Every value is a peak, not an average.** Sampling runs at `sample_seconds` and the row carries
  the highest of the five sub-samples, so a one-second spike survives into a five-second row. The
  price is that these rows cannot be summed: adding `net_rx_rate` across a day overstates the bytes
  actually transferred, because each value is a peak standing in for five seconds.
- Per-service memory and CPU come from the unit's cgroup — `memory.stat`'s `anon + file_mapped`
  (which reconstructs `VmRSS`: anonymous pages plus mapped file pages, with cold page cache left
  out) and `cpu.stat`'s `usage_usec`. One read covers a multi-process service correctly, and a
  missing directory is exactly the "not on this box" signal.
- The unit's cgroup directory is **searched for** under `/sys/fs/cgroup`, never assumed to be under
  `system.slice` — Scylla's packaging puts it at
  `scylla.slice/scylla-server.slice/scylla-server.service`. Resolved once and cached; a failed
  search retries every 30 s, so an absent unit costs nothing and one that starts later is picked up.
- **`-1` means not measured**, and it is the whole answer to the Lambda case: with no
  `genix.service` on the machine, the backend's two columns carry the sentinel rather than a `0`
  that would read as an idle backend. It survives to the row only when no sub-sample of the window
  produced a value.
- CPU is a percentage of the **whole machine**, so Scylla pinning eight of eight cores reads
  100.00% and not the top-style 800% that would not fit the column. Percentages are hundredths;
  memory is megabytes saturating at 32 GB; network is 5 KB/s units, which reaches 163 MB/s while
  still resolving the single-digit KB/s an idle box shows.
- Rows land on a wall-clock grid, not on a tick counter, so a restart resumes the same slots and a
  skipped tick leaves an honest hole instead of shifting every later row.
- **Fails open**, like the request log: a failed write is a warning. The insert is prepared lazily
  and retried every 60 s while it fails, so a daemon that starts before `fn-homologate` created the
  table heals itself instead of needing a restart.

## Deploying

`sudo python3 scripts/configure.py 37` compiles the binary, installs the systemd units,
and writes the bridge's Nginx vhost (HTTP/3 when a certificate exists and Nginx was built with
it) on this host. It asks nothing — everything comes from `config.toml`. It installs a C compiler
if the host has none, and after starting the service it probes `/health` rather than trusting
`systemctl restart`: this daemon exits when ScyllaDB is unreachable, and with `Restart=always`
that would otherwise look identical to a healthy start. Full details, including the generated
unit and the three non-negotiable Nginx streaming settings, are in
[`../scripts/configure/CONFIGURE_SERVER_UTILS.md`](../scripts/configure/CONFIGURE_SERVER_UTILS.md).

For a self-hosted backend, select both components (`237` or `238`) and choose Backend mode `1` or
`2`. The dispatcher installs this daemon without its public SSE Nginx vhost and does not require
`sse_bridge.url`; the backend process already serves `/agent/stream`.

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
