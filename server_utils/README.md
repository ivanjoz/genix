# Genix Server Utilities

This Rust process currently hosts the raw-TCP credit rate limiter. The crate lives directly in
`server_utils/` so future server-side utilities can share the same binary and support code.

The full design and binary formats are in [PLAN.md](PLAN.md).

## Behavior

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

The process also reads root `secret_phrase` and `[db].host`, `port`, `name`, `user`, and `password`.
Set `GENIX_CONFIG_FILE` to select a non-default TOML file. Every setting can be overridden by its
uppercase environment equivalent, such as `RATE_LIMIT_USER_CPU_10S` or `DB_HOST`.

All quota values must be positive and nondecreasing from 10 seconds to one hour to 24 hours. The
24-hour values cannot exceed `uint32`, which is the largest persisted blob width.

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

## TCP contract

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

## systemd example

```ini
# Purpose: Run one hardened, automatically restarted limiter beside the Genix backend.
[Unit]
Description=Genix Server Utilities
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/usr/local/bin/genix
Environment=GENIX_CONFIG_FILE=/etc/genix/config.toml
ExecStart=/usr/local/bin/genix/genix-server-utils
Restart=always
RestartSec=3
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=read-only

[Install]
WantedBy=multi-user.target
```

The raw TCP listener should remain on loopback or a private network. HMAC authenticates messages,
but the protocol does not encrypt traffic.

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
