# SSE Bridge Deployment (`configure_sse_bridge.py`)

Installs `sse_bridge/` on **one** host: the systemd units that keep it running, and the Nginx
vhost that terminates TLS in front of it.

```bash
sudo ./app.sh configure_sse_bridge
```

No arguments, no install modes, and exactly one possible question —
`sse_bridge.apikey`, and only when `config.toml` does not already have it. Everything else
is read from `config.toml` or defaulted; a missing value fails with the key name instead of
opening a prompt.

Why it is simpler than [`configure_server.py`](CONFIGURE_SERVER.md): that script covers a backend
host and an Nginx edge host that are usually different machines. Here Nginx must run on the very
machine the bridge runs on — what it proxies is a permanent stream, and a second hop buys nothing
— so there is no upstream to configure. The vhost always forwards to `127.0.0.1`.

Must be run as `root`; the bridge itself runs as a non-root user. Background:
[`../PLAN_SSE_BRIDGE.md`](../PLAN_SSE_BRIDGE.md), endpoints:
[`../sse_bridge/README.md`](../sse_bridge/README.md).

---

## 🔑 config.toml

The bridge host holds a **minimal** `config.toml` — not the backend's, which also carries
database, AWS and SMTP secrets this process has no business seeing:

```toml
secret_phrase = ""

[sse_bridge]
url    = "https://genix-sse.un.pe/"
apikey = "<the backend's secret_phrase>"
```

| Key | Required | Used for |
| --- | -------- | -------- |
| `sse_bridge.url` | yes | Its hostname becomes `server_name`, the config file name, and the `/etc/letsencrypt/live/` directory that is looked up. |
| `sse_bridge.apikey` | yes | The shared secret. Asked for once and stored if absent. |
| `sse_bridge.port` | no | Defaults to **14012** (`defaultListenPort` in `sse_bridge/config.go`). |

- **`sse_bridge.url`** is never prompted for: it is not a secret, and it is the same key the
  backend and the frontend read to decide whether to use the bridge at all. Missing, malformed,
  or still pointing at the Lambda function URL (which is how the project says *no bridge*) stops
  the run with a message naming the key.
- **`sse_bridge.apikey`** must be **byte-identical to the backend's `secret_phrase`**: it is the
  HMAC key of the session tokens the backend issues and of the `X-Bridge-Auth` header it signs.
  A mismatch is not a startup error — it is every request being rejected at runtime.
  On a developer machine, where the backend's full `config.toml` is present, the
  `secret_phrase` key is used directly and nothing is asked.
- **`sse_bridge.port`** with a bad value stops the run rather than falling back, because the
  Nginx upstream is built from it.

When the key has to be typed, it is read with `getpass` (not echoed, never printed back in the
summary) and written into `config.toml` as `sse_bridge.apikey`, `chmod 0600`. That is not
offered as a choice: the service cannot start without it, so declining would only install a unit
that fails on boot. With no TTY attached (cron, CI, `ssh` without `-t`) the run fails instead.

---

## 🛠 What gets installed

| Unit | Role |
| ---- | ---- |
| `genix-sse-bridge.service` | The bridge, running as `ubuntu` (or the non-root `SUDO_USER`). |
| `genix-sse-bridge-restart.path` | Watches `/usr/local/bin/genix/sse_bridge` for changes. |
| `genix-sse-bridge-restart.service` | Root helper the path unit triggers to restart the bridge. |

```ini
[Service]
Type=simple
User=ubuntu
WorkingDirectory=/usr/local/bin/genix
Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml
Environment=SSE_BRIDGE_PORT=14012
ExecStart=/usr/local/bin/genix/sse_bridge
Restart=always
RestartSec=3
```

The key is **not** exported in the unit: files under `/etc/systemd/system` are world-readable,
`config.toml` is `0600`. Plus the same hardening as `genix.service` (`NoNewPrivileges`,
`ProtectSystem=strict`, `CapabilityBoundingSet=`, `ReadWritePaths=/usr/local/bin/genix`).

### Where the binary comes from

1. `sse_bridge/` present → compiled with `go build -ldflags "-s -w"` into
   `tmp/sse_bridge_linux_<arch>`, as the repository owner so the Go module cache is reused.
   Unlike the backend there are no git submodules or local `replace` targets to populate.
2. Otherwise the first usable ELF among `/usr/local/bin/genix/sse_bridge`,
   `tmp/sse_bridge_linux_<arch>`, `sse_bridge/sse_bridge`.
3. Otherwise the run fails.

It is copied to `.sse_bridge.staged` and renamed into place, so the running service never reads a
half-written file, and later deploys that overwrite it restart the service by themselves through
the path watcher.

---

## 🔐 Nginx vhost

Written to `/etc/nginx/conf.d/<domain>.conf`, validated with `nginx -t` before Nginx is
restarted, and left alone entirely when the rendered file is byte-identical to what is there.

**With a certificate** (`/etc/letsencrypt/live/<domain>/fullchain.pem` + `privkey.pem`, or
`ssl_certificate*` lines preserved from an existing Certbot-managed file) it emits HTTP/3:

```nginx
listen 443 quic reuseport;
listen 443 ssl;
listen [::]:443 quic reuseport;
listen [::]:443 ssl;
http2 on;

ssl_protocols TLSv1.3;
add_header Alt-Svc 'h3=":443"; ma=86400' always;
```

**Without one**, a plain `listen 80` vhost, so the host is reachable and `certbot --nginx` has a
server block to attach to. Re-run the script afterwards to get the HTTP/3 version.

Requires Nginx ≥ 1.25 for `listen … quic` and `http2 on`.

### Details that are deliberate

- **`reuseport` is dropped automatically** when another file under `/etc/nginx` already claims
  it. Only one server block per `address:port` may set it; a second one fails `nginx -t` with
  "duplicate listen options", which is exactly what happens when the bridge shares its host with
  a backend vhost written by `configure_server.py`.
- **`proxy_buffering off` / `proxy_cache off` / `gzip off`** — without these, Nginx accumulates
  the agent's events and hands them over only when the response ends, which for a permanent
  stream is never.
- **`proxy_read_timeout 3600s`** — an idle stream between agent turns must not be killed. The
  bridge sends `: ping` every 20s, so a dead peer is still detected quickly.
- **`proxy_http_version 1.1` + `proxy_set_header Connection ""`** — the bridge speaks plain
  HTTP; there is no WebSocket upgrade to tunnel, unlike the backend vhost.
- **No CORS headers** — `withClientCORS` in `sse_bridge/handlers.go` already sets them and
  answers preflights. A duplicated `Access-Control-Allow-Origin` makes browsers reject the
  response.
- **`ssl_early_data` stays off** — `POST /in` is not replay-safe, and 0-RTT gains nothing for a
  connection that stays open for the whole session.

---

## ✅ After installing

```bash
systemctl status genix-sse-bridge.service
journalctl -u genix-sse-bridge.service -f
curl -s http://127.0.0.1:14012/health          # {"Ok":true,"Channels":0,...}
curl -s https://genix-sse.un.pe/health         # through Nginx
```

Then set the same `sse_bridge.url` in the `config.toml` used to **build the backend and the
frontend** — that is what switches both sides onto the bridge. Leaving it equal to `aws.lambda_url`
keeps the old native-SSE behaviour no matter what is installed here.

Tests: `python3 -m unittest discover -s scripts/tests -p "test_configure_*.py"`.
