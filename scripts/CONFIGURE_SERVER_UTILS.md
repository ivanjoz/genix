# Server Utils Deployment (`configure_server_utils.py`)

Installs `server_utils/` on **one** host: the systemd units that keep it running, and the Nginx
vhost that terminates TLS in front of its SSE bridge.

```bash
sudo python3 scripts/configure_server_utils.py
```

No arguments, no install modes, and **no questions at all**. Everything is read from
`config.toml`; a missing value either fails with the key name or is written into the file with a
documented default — never a prompt.

Must be run as `root`; the service itself runs as a non-root user. Endpoints and protocol:
[`../server_utils/README.md`](../server_utils/README.md). Design of the two halves:
[`../server_utils/PLAN.md`](../server_utils/PLAN.md) (rate limiter) and
[`../server_utils/PLAN_SSE_BRIDGE.md`](../server_utils/PLAN_SSE_BRIDGE.md) (bridge).

---

## 🧩 One binary, two transports

`genix-server-utils` hosts everything in one process:

| Transport | Port | Exposure | Nginx vhost |
| --------- | ---- | -------- | ----------- |
| Raw TCP — credit rate limiter and lock service, told apart by the frame's opcode | `server_utils` (default `127.0.0.1:14013`) | Loopback only. HMAC-authenticated but **not encrypted**. | No — never proxied. |
| SSE bridge | `sse_bridge.port` (default `14012`) | HTTP, must be reachable by browsers. | Yes — TLS + HTTP/3, no buffering. |

`server_utils` is a root-level key, not `rate_limit.address`: one port serves every raw-TCP
operation, so the address belongs to the process rather than to one service inside it.

They share only the process: the config load, the shutdown signal, and the tokio runtime.

> **Deploy the backend tables first.** The rate limiter loads existing usage from ScyllaDB
> before admitting anything and **exits** when it cannot — which also stops the bridge, since
> it is one process. On a fresh host, run `cd scripts && go run . check_tables` (so
> `credit_usage` exists) before enabling the unit, or the service restart-loops with
> `unconfigured table credit_usage` in `journalctl`.

Why Nginx must run on this machine: what it proxies is a permanent stream, and a second hop
buys nothing. So there is no upstream to configure — the vhost always forwards to `127.0.0.1`.

---

## 🔑 config.toml

| Key | Required | Used for |
| --- | -------- | -------- |
| `secret_phrase` | yes | Verifying the browser's session token. Must equal the backend's value. |
| `internal_apikey` | yes | Authenticating the backend's calls: the bridge's `X-Bridge-Auth` header and the rate limiter's TCP frames. Must equal the backend's value. |
| `sse_bridge.url` | yes | Its hostname becomes `server_name`, the config file name, and the `/etc/letsencrypt/live/` directory that is looked up. |
| `sse_bridge.port` | no | Defaults to **14012** (`DEFAULT_BRIDGE_PORT` in `server_utils/src/config.rs`). |
| `sse_bridge.verbose` | no | `true` logs every delivered message. |
| `server_utils` | no | Raw-TCP listen address. Defaults to **127.0.0.1:14013**; only reported in the summary. |
| `rate_limit.*` | filled in | Quota policy; see `server_utils/README.md`. The twelve credit ceilings have no default in the daemon, so the script writes `config.example.toml`'s values for any that are missing. |
| `db.*` | yes | Where usage snapshots are persisted. |

- **The two secrets are never prompted for.** They are root-level keys that initial project
  setup already writes (like `admin_password`), and both must be **byte-identical to the
  backend's**. A mismatch is not a startup error — it is every request being rejected at
  runtime, so the script checks they are present and fails loudly rather than guessing.
- **`secret_phrase` vs `internal_apikey`**: the first signs what a *user* holds (session tokens,
  password hashes); the second authenticates one *Genix process* to another. Splitting them
  means the inter-service key can be rotated without invalidating every live session.
- **`sse_bridge.url`** is never prompted for: it is not a secret, and it is the same key the
  backend and the frontend read to decide whether to use the bridge at all. Missing, malformed,
  or still pointing at the Lambda function URL (which is how the project says *no bridge*) stops
  the run with a message naming the key.
- **`sse_bridge.port`** with a bad value stops the run rather than falling back, because the
  Nginx upstream is built from it.
- **The credit ceilings are written, not demanded.** They are the one class of missing setting
  the script can fix: quotas are policy, not secrets, so a sane starting point exists — and the
  alternative is `missing required setting rate_limit.company_cpu_10s` on every restart, three
  seconds apart, forever. Values already present are left alone; a missing window is filled with
  the example default raised to the previous window's value, so completing a hand-tuned set
  cannot produce the decreasing sequence the daemon rejects (`10s <= 1h <= 24h`). They go into
  `config.toml` rather than the unit's `Environment=` so they stay visible and tunable.

---

## 🛠 What gets installed

| Unit | Role |
| ---- | ---- |
| `genix-server-utils.service` | The process, running as `ubuntu` (or the non-root `SUDO_USER`). |
| `genix-server-utils-restart.path` | Watches `/usr/local/bin/genix/genix-server-utils` for changes. |
| `genix-server-utils-restart.service` | Root helper the path unit triggers to restart the service. |

```ini
[Service]
Type=simple
User=ubuntu
WorkingDirectory=/usr/local/bin/genix
Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml
Environment=SSE_BRIDGE_PORT=14012
ExecStart=/usr/local/bin/genix/genix-server-utils
Restart=always
RestartSec=3
```

Neither secret is exported in the unit: files under `/etc/systemd/system` are world-readable,
`config.toml` is `0600`. Plus the same hardening as `genix.service` (`NoNewPrivileges`,
`ProtectSystem=strict`, `CapabilityBoundingSet=`, `ReadWritePaths=/usr/local/bin/genix`).

### Where the binary comes from

1. `server_utils/` present → compiled with `cargo build --release` into
   `server_utils/target/release/genix-server-utils`, as the repository owner so that account's
   Cargo registry and `target/` cache are reused (and nothing root-owned is left in the clone).
   `cargo` is found on `PATH`, then at `~<owner>/.cargo/bin/cargo`, then
   `/usr/local/cargo/bin/cargo`; `CARGO_BINARY=/path/to/cargo` overrides all of it (`sudo`
   strips `~/.cargo/bin` from `PATH`).
2. Otherwise the first usable ELF among `/usr/local/bin/genix/genix-server-utils`,
   `tmp/genix-server-utils_linux_<arch>`, `server_utils/target/release/genix-server-utils`.
3. Otherwise the run fails.

It is copied to `.genix-server-utils.staged` and renamed into place, so the running service
never reads a half-written file, and later deploys that overwrite it restart the service by
themselves through the path watcher.

---

## 🌐 Nginx

The vhost is written for `sse_bridge.url`'s hostname, HTTP/3 when a certificate exists and
plain HTTP otherwise. Three settings are not optional — without them the stream is delivered in
bursts or cut off:

```nginx
proxy_buffering off;      # a buffered text/event-stream is a stalled request
gzip off;                 # compressing also implies buffering
proxy_read_timeout 3600s; # an idle stream must survive between agent turns
```

It adds **no CORS headers**: the bridge sets them and answers preflights itself, and a
duplicated `Access-Control-Allow-Origin` is rejected by browsers.

`reuseport` is claimed by only the first `server` block on an address:port. When this host also
serves the backend vhost from [`CONFIGURE_SERVER.md`](CONFIGURE_SERVER.md), the script detects
that and omits the option instead of failing `nginx -t`.

---

## ✅ Verifying

```bash
systemctl status genix-server-utils
journalctl -u genix-server-utils -n 50

# The bridge, locally and through Nginx.
curl -s http://127.0.0.1:14012/health
curl -s https://<sse_bridge.url host>/health   # {"Ok":true,"Channels":0,"UptimeSeconds":…}

# The rate limiter should answer only on loopback.
ss -lntp | grep 14013
```

`RUST_LOG=genix_server_utils=debug` in the unit turns on per-request diagnostics for both
services.
