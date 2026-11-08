# Fareward Deployment (`configure_fareward.py`)

Installs `fareward/` on **one** host. The systemd service always provides the raw-TCP rate
limiter and lock service. Its public SSE HAProxy site is installed only for a Lambda deployment.

The public entrypoint selects whether to compile (`37`) or use the latest verified release (`38`):

```bash
sudo python3 scripts/configure.py 38
```

The nested installer asks **no questions**. Everything is read from `config.toml`; a missing value
either fails with the key name or is written into the file with a documented default. The unified
dispatcher supplies its binary policy and, for a self-hosted backend, `--service-only`.

With the unified selection `238`, choosing Backend mode `1` or `2` means self-hosted/VPS mode.
The dispatcher passes `--service-only`: `sse_bridge.url` is not required and HAProxy is not touched,
because that backend serves `/agent/stream` itself. The HAProxy side of the SPOE agent belongs to
the API site and is written by `configure_server.py` (see
[`CONFIGURE_SERVER.md`](CONFIGURE_SERVER.md#the-fareward-spoe-agent-mode-1-only)); this script only
installs the daemon that serves `[spoe].socket`.

Selecting Fareward **on its own** (`37`, `38`) is the case the dispatcher has to work out, and
it does so by looking for `/etc/systemd/system/genix.service`.

Finding it settles the question outright and **nothing is asked**: a local backend serves
`/agent/stream` itself, so no deployment wants a bridge in front of it. Finding nothing settles
nothing — the backend could be on Lambda, which is the whole reason the bridge exists, or on another
VPS serving its own stream — so that is the one case that prompts, defaulting to the bridge. That
default keeps a missing `sse_bridge.url` an install-time error with the key name, rather than
quietly producing a Lambda companion that cannot do the one job it was installed for.

This used to demand `sse_bridge.url` on *every* fareward-only run, which failed the install on
hosts that already served `/agent/stream` themselves.

Must be run as `root`; the service itself runs as a non-root user. Endpoints and protocol:
[`../../fareward/README.md`](../../fareward/README.md). Design of the two halves:
[`../../fareward/PLAN.md`](../../fareward/PLAN.md) (rate limiter); the bridge half is
covered by the README above.

---

## 🧩 One binary, two transports

`fareward` hosts everything in one process:

| Transport | Port | Exposure | HAProxy site |
| --------- | ---- | -------- | ----------- |
| Raw TCP — credit rate limiter and lock service, told apart by the frame's opcode | `fareward.port` (default `14013`) | Loopback, or the whole internet with `fareward.public = true`. HMAC-authenticated but **not encrypted**. | No — never proxied. |
| SSE bridge | `sse_bridge.port` (default `14012`) | HTTP, must be reachable by browsers. | Yes — TLS + HTTP/3, no buffering. |
| SPOE agent (optional) | Unix socket `spoe.socket` | HAProxy on this host only. | Part of the API site (`configure_server.py`). |

`[fareward]` is its own section, not `rate_limit.address`: one port serves every raw-TCP
operation, so it belongs to the process rather than to one service inside it.

The bind address is **derived from `public`** (`0.0.0.0` or `127.0.0.1`) and never taken from
`host`, which is the client's half of the section. That split is not cosmetic: a cloud VM's public
IP lives in the provider's NAT and never on the VM's interface, so a daemon told to bind it dies
at startup with `Cannot assign requested address (os error 99)`. Set `public = true` only when the
backend runs off-box — on Lambda, say — and remember it puts an unencrypted (though signed)
protocol on the open internet, so the port also needs opening in the host firewall **and** in the
cloud provider's security list.

They share only the process: the config load, the shutdown signal, and the tokio runtime.

> **No database.** The rate limiter loads existing usage from its own store (`[store].path`,
> default `/var/lib/fareward/store`, created by the unit's `StateDirectory=fareward`) before
> admitting anything and **exits** when it cannot open it — which also stops the bridge, since it
> is one process. It reads and writes no MariaDB table: the backend pushes the grants and writes
> its own request log. Back that directory up: it is the whole state of the credit system.

Why HAProxy must run on this machine: what it proxies is a permanent stream, and a second hop
buys nothing. So there is no upstream to configure — the site always forwards to `127.0.0.1`.

---

## 🔑 config.toml

| Key | Required | Used for |
| --- | -------- | -------- |
| `secret_phrase` | yes | Verifying the browser's session token. Must equal the backend's value. |
| `internal_apikey` | yes | Authenticating the backend's calls: the bridge's `X-Bridge-Auth` header and the rate limiter's TCP frames. Must equal the backend's value. |
| `sse_bridge.url` | Lambda only | Its hostname names the site file `/etc/haproxy/sites/<host>.cfg`, its backend `site_<host>`, and the Let's Encrypt certificate this script issues. Not read in VPS service-only mode. |
| `sse_bridge.port` | no | Defaults to **14012** (`DEFAULT_BRIDGE_PORT` in `fareward/src/config.rs`). |
| `sse_bridge.verbose` | no | `true` logs every delivered message. |
| `fareward.port` | no | Raw-TCP listen port. Defaults to **14013** (`DEFAULT_LISTEN_PORT` in `fareward/src/config.rs`). |
| `fareward.public` | no | `true` binds `0.0.0.0`, absent or `false` binds `127.0.0.1`. |
| `fareward.host` | no | Read by the **backend**, not by the daemon: the address it dials. Ignored when `public = false`, where loopback is the only thing that can work. |
| `fareward.use_remote_dev_host` | no | Read by the **backend**, and only when it was launched with the `dev` argument: a dev machine dials `127.0.0.1` even with a public `host`, because the daemon that matches its checkout is the one running beside it. `true` dials `host` instead. |
| `fareward.unix_socket` | no | Read by **both**. The daemon also listens on this Unix domain socket (mode `0660`; the TCP port stays up for Lambda), and a backend/agent not running on Lambda dials it instead of TCP. Use `/run/fareward/fareward.sock`: the unit's `RuntimeDirectory=fareward` creates that directory. Set it only when the backend shares the host with the daemon. |
| `rate_limit.*` | filled in | Quota policy; see `fareward/README.md`. The twelve credit ceilings have no default in the daemon, so the script writes `config.example.toml`'s values for any that are missing. |
| `store.path` | no | The store directory. Defaults to `/var/lib/fareward/store`, inside the unit's `StateDirectory`. |
| `server_metrics.database_unit` | no | The database's systemd unit, sampled into the metrics' database columns. Defaults to `mariadb.service`. |

- **The two secrets are never prompted for.** They are root-level keys that initial project
  setup already writes (like `admin_password`), and both must be **byte-identical to the
  backend's**. A mismatch is not a startup error — it is every request being rejected at
  runtime, so the script checks they are present and fails loudly rather than guessing.
- **`secret_phrase` vs `internal_apikey`**: the first signs what a *user* holds (session tokens,
  password hashes); the second authenticates one *Genix process* to another. Splitting them
  means the inter-service key can be rotated without invalidating every live session.
- **`sse_bridge.url`** is never prompted for in Lambda mode: it is not a secret, and the backend uses it to
  publish through this bridge. The matching login option exposes this URL separately as
  `[[endpoints]].bridge`. Missing, malformed, or still pointing at the Lambda function URL
  (which is how the backend says *no bridge*) stops the run with a message naming the key.
- **`sse_bridge.port`** with a bad value stops the run rather than falling back, because the
  HAProxy backend is built from it.
- **The credit ceilings are written, not demanded.** They are the one class of missing setting
  the script can fix: quotas are policy, not secrets, so a sane starting point exists — and the
  alternative is `missing required setting rate_limit.company_cpu_short` on every restart, three
  seconds apart, forever. Values already present are left alone. A missing window is filled with the example default raised to the previous window's value, so
  completing a hand-tuned set cannot produce the decreasing sequence the daemon rejects
  (`short <= long`). There is no daily ceiling to write: the daily and monthly entitlements are
  per company, in the daemon's budget store. The window lengths (`short_window_seconds`, `long_window_minutes`)
  have a default in the daemon and are never written. The ceilings go into `config.toml` rather
  than the unit's `Environment=` so they stay visible and tunable.

---

## 🛠 What gets installed

| Unit | Role |
| ---- | ---- |
| `fareward.service` | The process, running as `ubuntu` (or the non-root `SUDO_USER`). |
| `fareward-restart.path` | Watches `/usr/local/bin/genix/fareward` for changes. |
| `fareward-restart.service` | Root helper the path unit triggers to restart the service. |

```ini
[Service]
Type=simple
User=ubuntu
WorkingDirectory=/usr/local/bin/genix
Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml
Environment=SSE_BRIDGE_PORT=14012
ExecStart=/usr/local/bin/genix/fareward
Restart=always
RestartSec=3
```

Neither secret is exported in the unit: files under `/etc/systemd/system` are world-readable,
`config.toml` is `0600`. Plus the same hardening as `genix.service` (`NoNewPrivileges`,
`ProtectSystem=strict`, `CapabilityBoundingSet=`, `ReadWritePaths=/usr/local/bin/genix`).

### Where the binary comes from

1. **Option `7`** requires `fareward/` and compiles it with `cargo build --release` into
   `fareward/target/release/fareward`, as the repository owner so that account's
   Cargo registry and `target/` cache are reused (and nothing root-owned is left in the clone).
   `cargo` is found on `PATH`, then at `~<owner>/.cargo/bin/cargo`, then
   `/usr/local/cargo/bin/cargo`; `CARGO_BINARY=/path/to/cargo` overrides all of it (`sudo`
   strips `~/.cargo/bin` from `PATH`).
2. **Option `8`** downloads `fareward_linux_<arch>` and `SHA256SUMS` from the latest
   public release, verifies the checksum, and deploys that `tmp/` asset ahead of any older
   installed binary. Cargo is never invoked in this mode, even when source is present.
3. Missing source under option `7`, or a missing/invalid release under option `8`, stops the run.

The unified entrypoint performs the latest download and verification automatically:

```bash
sudo python3 scripts/configure.py 38
```

It is copied to `.fareward.staged` and renamed into place, so the running service
never reads a half-written file, and later deploys that overwrite it restart the service by
themselves through the path watcher.

---

## 🌐 HAProxy

The bridge is one site on the shared HAProxy edge that `configure_server.py` also uses (same
install, certificate and validation steps: see
[`CONFIGURE_SERVER.md`](CONFIGURE_SERVER.md#what-the-haproxy-step-sets-up-modes-1-and-3)). This
script writes only `/etc/haproxy/sites/<sse_bridge.url host>.cfg`, then regenerates
`haproxy.cfg` around every site on the host and issues the hostname's certificate.

```haproxy
backend site_genix-sse.un.pe
    timeout server 3600s      # an idle stream must survive between agent turns
    server bridge 127.0.0.1:14012
```

HAProxy forwards response data as it arrives, so a `text/event-stream` is never held back and
there is no buffering to turn off. The site adds **no CORS headers**: the bridge sets them and
answers preflights itself, and a duplicated `Access-Control-Allow-Origin` is rejected by browsers.
Early data on `POST /in` waits for the TLS handshake (the shared frontend's rule): it is not
replay-safe.

---

## ✅ Verifying

```bash
systemctl status fareward
journalctl -u fareward -n 50

# The bridge, locally and through HAProxy.
curl -s http://127.0.0.1:14012/health
curl -s https://<sse_bridge.url host>/health   # {"Ok":true,"Channels":0,"UptimeSeconds":…}

# Where the raw-TCP port ended up: 127.0.0.1 with public = false, 0.0.0.0 with public = true.
ss -lntp | grep 14013
```

`RUST_LOG=fareward=debug` in the unit turns on per-request diagnostics for both
services.

When the daemon does not come up, the script reads the journal and names the cause
(`KNOWN_SERVICE_FAILURE_CAUSES`): a bind address that is not on this host, a port already taken,
a store directory it cannot create or open, or a missing config key. If none match it
says so rather than guessing — the daemon prints its reason and exits, so the last `Error:` line
in the log above is the answer.
