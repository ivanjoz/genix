# Genix Server Utilities

One Rust process hosting two independent server-side services:

| Service | Transport | Port | Purpose |
|---|---|---|---|
| Credit rate limiter | Raw TCP, loopback | `rate_limit.address` (default `127.0.0.1:14013`) | Atomic CPU/inference quota checks for the Go backend. |
| SSE bridge | HTTP (TLS via Nginx) | `sse_bridge.port` (default `14012`) | Relays agent events between the backend and browser tabs. |

They share only this process: the config load, the shutdown signal, and the tokio runtime.
Neither calls into the other.

Designs: [PLAN.md](PLAN.md) (rate limiter, including all binary formats) and
[PLAN_SSE_BRIDGE.md](PLAN_SSE_BRIDGE.md) (bridge). Deployment:
[`../scripts/CONFIGURE_SERVER_UTILS.md`](../scripts/CONFIGURE_SERVER_UTILS.md).

> **One process, shared fate.** The rate limiter loads existing usage from ScyllaDB before
> admitting anything and exits when it cannot — which also stops the bridge. Deploy the backend
> tables (so `credit_usage` exists) before starting the daemon.

## Layout

One module tree per service, so each owns its own `auth` and `server` module without collisions:

```text
src/
├── main.rs      # spawns both services, one shared shutdown signal
├── config.rs    # the only thing the two services share
├── limiter/     # quota.rs (RateLimiter + policy), protocol, auth, aggregation,
│                # credits_blob, time_frame, storage, server (raw TCP)
└── bridge/      # token.rs (colbin + channel token), auth, channel, http (axum)
```

## Two secrets, split by purpose

Both are root-level keys in `config.toml` and must match the backend byte for byte:

| Key | Used for |
|---|---|
| `internal_apikey` | Service-to-service authentication: the rate limiter's TCP frame HMAC (`genix-rate-limiter:v1`) and the bridge's `X-Bridge-Auth` header (`sse-bridge:v1\|`). |
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

Add `[rate_limit]` to the project `config.toml`; the complete commented example is in
[`../config.example.toml`](../config.example.toml).

```toml
# Purpose: Configure process limits and the two global quota profiles.
[rate_limit]
address               = "127.0.0.1:14013"
flush_seconds         = 15
frame_timeout_seconds = 30
max_connections       = 1024
shards                = 0 # 0 uses the logical CPU count

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
# Purpose: Compile and verify all protocol, codec, limiter, and flush unit tests.
cd server_utils
cargo test
cargo build --release
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

## Rate limiter TCP contract

After accepting a connection, the server writes an eight-byte random nonce. Every subsequent
request is a 19-byte big-endian frame:

| Bytes | Field |
|---:|---|
| 3 | Company ID (`uint24`, positive) |
| 3 | User ID (`uint24`, positive) |
| 1 | API group (`0..5`) |
| 2 | CPU credits (`uint16`) |
| 2 | Inference credits (`uint16`) |
| 8 | Truncated authentication HMAC |

The HMAC covers the first 11 bytes plus the connection nonce and implicit frame sequence. A valid
frame receives exactly one byte: zero means accepted; a nonzero low-five-bit value identifies the
scope, time window, and exhausted credit types. Authentication, malformed-frame, initialization,
and transport failures close the connection.

## Deploying

`sudo python3 scripts/configure_server_utils.py` compiles the binary, installs the systemd units,
and writes the bridge's Nginx vhost (HTTP/3 when a certificate exists) on this host. It asks
nothing — everything comes from `config.toml`. Full details, including the generated unit and the
three non-negotiable Nginx streaming settings, are in
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
