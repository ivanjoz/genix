# Genix Service Deployment & Auto-Reload Strategy

This document explains the systemd-based strategy for automatically reloading the **Genix Backend Service** when its binary is updated, and the HAProxy edge that fronts it. The setup script itself must be executed as `root`, but the application binary and the main service run as a non-root user.

---

## 🧩 Install Modes

The normal install is **Full**: HAProxy and the backend on the same VPS, connected through a Unix socket, with the fareward SPOE agent in front of the backend when `[spoe].socket` is set. Modes 2 and 3 split them across two machines over TCP, without the agent.

| Mode | Name | What it configures | config.toml keys used |
| ---- | ---- | ------------------ | -------------------------- |
| `1` | Full | systemd units + HAProxy edge over `unix@/run/genix/api.sock`, SPOE optional | `server.domain`, `[spoe]`, `fareward.routes_file` |
| `2` | Only Systemd Service | systemd units, binary directory, runtime user | `server.port` |
| `3` | Only HAProxy edge | `/etc/haproxy/sites/<server.domain>.cfg` | `server.domain`, `server.upstream` |

After selecting Backend with the unified entrypoint, answer this second prompt for the host role:

```bash
sudo python3 scripts/configure.py 28  # latest precompiled backend; then choose 1, 2 or 3
sudo python3 scripts/configure.py 27  # compile backend source; then choose 1, 2 or 3
```

Only the credentials the selected mode needs are validated, so the edge host does not need `server.port` and the backend host does not need `server.domain`.

### Missing credentials are requested on the terminal

`config.toml` is optional. When the file is absent — the usual case on an edge-only VPS that has no full clone — or when a key the selected mode needs is missing or malformed, the script asks for that value and re-asks until it validates:

```
[*] No config.toml at /home/ubuntu/genix/config.toml. Values will be requested.
[*] server.domain is not set in config.toml.
Enter server.domain, the public domain HAProxy serves (e.g. genix-api-4.un.pe): bad domain
[!] 'bad domain' is not a valid domain name (expected e.g. genix-api-4.un.pe).
Enter server.domain, the public domain HAProxy serves (e.g. genix-api-4.un.pe): genix-api-4.un.pe
```

Validation rules:

- `server.domain` — a dotted hostname; lower-cased and stripped of a trailing dot.
- `server.upstream` — `host:port`, optionally prefixed with `http://` or `https://`. The port is mandatory.
- `server.port` — an integer in `1-65535`.

After everything resolves, the script offers to write the typed-in values back so the next run is non-interactive:

```
[*] These values were typed in and are not stored yet: server.domain=genix-api-4.un.pe, server.upstream=100.64.0.2:14010
Save them to /home/ubuntu/genix/config.toml? [Y/n]:
```

Answering yes merges the keys into the existing file, leaving all other keys intact. A newly created file is chowned to the owner of the repository directory and set to `0600`, since it holds secrets and the service reads it as a non-root user.

Two things still fail the run instead of prompting:

- A `config.toml` that exists but is not valid TOML — prompting would later overwrite a real but broken file.
- A missing value with no TTY attached (cron, CI, `ssh` without `-t`). The error names the exact key to add.

In **mode 2**, note that `genix.service` points `GENIX_CONFIG_FILE` at this path and the backend panics without it, so installing the units with no `config.toml` present prints a warning: the service is installed but will not start until the file is there.

### HAProxy → backend connection

- `server.domain` (e.g. `genix-api-4.un.pe`) names the site file `/etc/haproxy/sites/<domain>.cfg`, its backend `site_<domain>` (the shared frontend routes by `Host` to `site_<host>`), and the Let's Encrypt certificate.
- **Full mode:** the backend server is `unix@/run/genix/api.sock`. `genix.service` sets `RuntimeDirectory=genix` (systemd creates `/run/genix` and removes it on stop) and `Environment=SERVER_SOCKET=/run/genix/api.sock`. The backend then binds only that socket (mode `0666`, so HAProxy can connect) and opens no TCP port. `server.port` and `server.upstream` are not used.
- **Split host (modes 2 + 3):** `server.upstream` (e.g. `10.0.0.7:14010`) becomes the `server` line; an `https://` prefix adds `ssl verify none`. `server.port` is exported as `Environment=SERVER_PORT=` and must equal the port half of `server.upstream`.
- HAProxy reuses backend connections by itself (`http-reuse safe`), so there is no keepalive pool to configure.

The backend picks its listener as: `SERVER_SOCKET` env var (Unix socket) → `SERVER_PORT` env var → `server.port` in `config.toml` → `3589`.

### The fareward SPOE agent (mode 1 only)

When `[spoe].socket` is set, HAProxy asks the fareward daemon about every API request on its headers, before reading the body, so a refused request costs no upload, no backend CPU and no buffer (design: `fareward/plans/plan.md`). Without it, the edge forwards everything and the backend gates each request itself, as on Lambda. Split hosts never get it: the backend trusts the agent's `x-fareward-*` headers only from a local peer.

- **`[spoe].mode`** — `"shadow"` (default) logs the verdict in the access log (`journalctl -u haproxy`, the `{status|reason|error}` captures) and forwards everything; `"enforce"` answers 400/401/403/411/413/429/500 on the verdict and **503 when the agent does not answer** (fail closed). The fareward daemon reads the same key, so the two always agree; restart fareward after changing it. In enforce mode every answer with a verdict, refusals and the backend's own responses alike, carries `x-fareward-usage` (credits left, credits spent per window, reason code; `fareward/plans/plan.md` D10), and a 429 is an HTML page saying how many credits the window that refused had spent, how many requests were refused and how many seconds to wait, with the same figure in `Retry-After`.
- **`[fareward].trust_edge_headers`** — written by the script, never by hand: `true` only in enforce mode. In enforce mode the agent has already charged an admitted request, so a backend that ignored the headers would charge it twice.
- **`[fareward].routes_file`** — required; must be inside `/var/lib/genix`, the `StateDirectory=genix` the backend unit gets, the only place it may write under `ProtectSystem=strict`. The backend writes the route table there at startup and the agent refuses to start without it, so configure the backend before fareward.
- The agent's socket is `0660` and owned by the service account. HAProxy drops every supplementary group when it drops root, so it runs with `group <runtime user>` in `global` instead of adding its user to that group.
- The SPOE engine lives in the API's backend (`event on-backend-http-request`), not in the shared frontend, so the SSE bridge site never goes through it. Preflights (`OPTIONS`) are answered by the edge and skip the agent.

### What the HAProxy step sets up (modes 1 and 3)

In this order, each step idempotent:

1. **Packages** — HAProxy **3.1 or later** (`mode spop` for the agent) and `certbot`. On apt hosts with an older or missing HAProxy, the official LTS repository is added first: `haproxy.debian.net` (`<codename>-backports-3.2`) on Debian, `ppa:vbernat/haproxy-3.2` on Ubuntu. On dnf hosts the distribution package is used, and the run fails if it is older than 3.1 (Fedora 44 ships 3.0).
2. **Firewall** — TCP 80 (certificate challenges and renewals), TCP 443, and **UDP 443** for QUIC, through
   `firewall_ports.py` (firewalld, ufw or plain iptables, whichever owns the host). The cloud provider's own
   rules (OCI security list, AWS security group) are separate and must allow the same.
3. **Kernel** — `/etc/sysctl.d/90-genix-network.conf`: BBR congestion control with the `fq` qdisc, applied now
   and on boot (`/etc/modules-load.d/genix-network.conf` loads `tcp_bbr`). It changes TCP only — the HTTP/2
   fallback — since QUIC runs its own congestion control inside HAProxy.
4. **Nginx** — if it is still running from an older install, it is stopped and disabled (not uninstalled; its files stay) so HAProxy can bind :80/:443.
5. **Configuration, plain HTTP** — this domain's site file, then `/etc/haproxy/haproxy.cfg` regenerated as the shared head plus every file in `/etc/haproxy/sites/`. `haproxy -c` validates the whole file; a rejected one puts the previous files back and fails, so the other sites keep working. Then reload (or start).
6. **Certificate** — `certbot certonly --standalone --http-01-address 127.0.0.1 --http-01-port 8888`: HAProxy forwards `/.well-known/acme-challenge/` to it, so issuing and renewing never stop the edge. **ECDSA P-256**, no email. The deploy hook `/usr/local/sbin/genix-haproxy-certificate` joins chain and key into `/etc/haproxy/certs/<domain>.pem` and reloads HAProxy; `certbot.timer` renews the same way. A certificate issued earlier through `certbot --nginx` is reissued once with `--force-renewal`, so its renewal stops depending on Nginx.
7. **Configuration, TLS** — written again now that a certificate exists: `bind :443` (`h2,http/1.1`) and, when the build has QUIC, `quic4@:443` (`h3`) with `Alt-Svc`. TLS 1.3 only, 0-RTT accepted but anything other than GET/HEAD waits for the handshake (`wait-for-handshake`), plain HTTP redirected to HTTPS. An OpenSSL without the QUIC API (`+QUIC_OPENSSL_COMPAT` in `haproxy -vv`) gets `limited-quic` and no 0-RTT over QUIC; a build without QUIC gets no UDP listener and no `Alt-Svc`.

The API site also answers browser preflights itself, deletes any client-sent `x-fareward-*` header, sets `X-Real-IP` (the backend reads the caller's address from it), refuses bodies over 50 MiB (`413`) and keeps idle WebSockets and `/agent/stream` open for an hour.

---

## 🚀 Strategy Overview

The setup consists of three systemd units working together:
1.  **`genix.service`**: The main application service running as `ubuntu` or the default non-root `sudo` user.
2.  **`genix-restart.path`**: A watcher that monitors the deployed binary for changes.
3.  **`genix-restart.service`**: A root-owned helper triggered by the path watcher to restart the main service.

### How it works:
1.  **Install**: Run `sudo python3 scripts/configure.py 28`, then choose mode `2` on the backend host.
2.  **Prepare**: The script creates `/usr/local/bin/genix/`, assigns ownership to `ubuntu` when present, or falls back to the non-root `SUDO_USER`.
3.  **Build**: It compiles `backend/` into `/usr/local/bin/genix/genix_app`, or installs a prebuilt binary, or fails — see [Where the binary comes from](#-where-the-binary-comes-from).
4.  **Update**: Later, a deploy copies a new executable over `/usr/local/bin/genix/genix_app`.
5.  **Detect**: The `genix-restart.path` unit detects the binary change.
6.  **Restart**: `genix-restart.service` restarts `genix.service`, which keeps running as the non-root runtime user.

---

## 🛠 Service Configurations

### 1. Main Service: `genix.service`
Located at: `/etc/systemd/system/genix.service`
This service runs the backend binary without root privileges.

```ini
[Unit]
Description=Genix Backend Service
After=network.target

[Service]
Type=simple
User=ivanjoz
Group=ivanjoz
WorkingDirectory=/usr/local/bin/genix
Environment=GENIX_CONFIG_FILE=/home/ubuntu/genix/config.toml
Environment=GENIX_REPOSITORY_ROOT=/home/ubuntu/genix
# Full mode. A split-host (mode 2) unit has Environment=SERVER_PORT=14010 instead of these three lines.
RuntimeDirectory=genix
RuntimeDirectoryMode=0755
Environment=SERVER_SOCKET=/run/genix/api.sock
# /var/lib/genix: where the backend writes [fareward].routes_file.
StateDirectory=genix
StateDirectoryMode=0755
ExecStart=/usr/local/bin/genix/genix_app
Restart=always
RestartSec=5

# Security Hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=read-only
ProtectControlGroups=yes
ProtectKernelModules=yes
ProtectKernelTunables=yes
RestrictRealtime=yes
CapabilityBoundingSet=
ReadWritePaths=/usr/local/bin/genix

[Install]
WantedBy=multi-user.target
```

The setup script replaces `ivanjoz` with:
- `ubuntu` when that account exists.
- Otherwise the non-root user found in `SUDO_USER`.

### 2. Path Watcher: `genix-restart.path`
Located at: `/etc/systemd/system/genix-restart.path`
This monitors the binary for modifications (`PathChanged`).

```ini
[Unit]
Description=Watch for changes to genix backend binary

[Path]
PathChanged=/usr/local/bin/genix/genix_app

[Install]
WantedBy=multi-user.target
```

### 3. Restarter Helper: `genix-restart.service`
Located at: `/etc/systemd/system/genix-restart.service`
A one-shot service that performs the actual restart.

```ini
[Unit]
Description=Restart Genix Service

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart genix.service
```

## Directory Ownership

The script creates `/usr/local/bin/genix/` with:

- owner: runtime user (`ubuntu` or `SUDO_USER`)
- group: runtime user's primary group
- mode: `2775`

When `/usr/local/bin/genix/genix_app` is already deployed, it is left in place with ownership reset to the runtime user and the executable bit set.

## 🔨 Where the binary comes from

In modes `1` and `2` the selected binary source is mandatory:

1. **Option `7`: compile from source** — requires `<repo>/backend/go.mod` and `<repo>/backend/main.go`, then runs
   `go build -tags lambda.norpc -ldflags "-s -w -X 'app/core.BuildDate=<now>'" -o <repo>/tmp/genix_app_linux_<arch> .`
   in `backend/`, matching `scripts/deploy_vps.go` so both paths embed the same build metadata. Without
   `mariadb` the binary would get the Scylla driver.
2. **Option `8`: latest precompiled binary** — `scripts/configure.py` downloads
   `genix_app_linux_<arch>` and `SHA256SUMS`, verifies the asset, and places it in `<repo>/tmp/`.
   That verified artifact wins over an older installed binary. A candidate counts only if it is
   non-empty and starts with the ELF magic.
3. **Crash** — missing source under option `7`, or a missing/invalid release under option `8`, is a hard error.

### Downloading a public prebuilt backend

The unified entrypoint performs this automatically and always uses the latest public release:

```bash
sudo python3 scripts/configure.py 28
```

### Populating go.mod replace targets

`backend/go.mod` redirects two modules into the working tree:

```
replace github.com/ivanjoz/genix-orm    => ./genix-orm
replace github.com/ivanjoz/genix-orm/db => ./genix-orm/db
```

`backend/genix-orm` is a git submodule, so a plain `git clone` leaves that directory empty and the build fails with `replacement directory ./genix-orm/db does not exist`. Before compiling, the script parses `go.mod` for filesystem `replace` targets (single-line and block form, comments ignored), checks each for a `go.mod`, and initialises the matching git submodule when one is missing:

```
[*] go.mod redirects a module to /home/homelab/genix/backend/genix-orm, which has no go.mod.
[*] Initialising git submodule backend/genix-orm...
[*] Running command: sudo -u homelab -H git submodule update --init --recursive -- backend/genix-orm
```

Only the submodules the backend actually needs are initialised — `frontend/packages/genix-ui` is left alone on a backend host. `git` runs as the repository owner so it uses that account's SSH keys and does not trip git's "dubious ownership" check.

One target is not a submodule: `genix-s3` compiles from `../../genix-search/genix-s3`, a checkout **beside**
the repository. When it is missing, the script clones the public `https://github.com/ivanjoz/genix-search.git`
there, as the repository owner (`SIBLING_REPOSITORY_REMOTES`).

#### SSH falls back to HTTPS

`.gitmodules` declares `git@github.com:ivanjoz/genix-orm.git`, which is what a developer with a key uses. A server usually has no key, so that clone dies with `Permission denied (publickey)`. Since the submodule is public, the script converts the remote to its HTTPS form and retries:

```
[*] Running command: env … git submodule update --recursive --init -- backend/genix-orm
    nobody@github.com: Permission denied (publickey).
[*] Exit code: 1
[*] Cloning git@github.com:ivanjoz/genix-orm.git failed. Retrying over HTTPS: https://github.com/ivanjoz/genix-orm.git
[*] Running command: git config submodule.backend/genix-orm.url https://github.com/ivanjoz/genix-orm.git
[*] Running command: env … git submodule update --recursive -- backend/genix-orm
[*] Exit code: 0
```

Notes on that sequence:

- The rewritten URL goes into **`.git/config` only**. `.gitmodules` is a tracked file and stays untouched, so nothing is committed and developers keep pushing over SSH. The override persists on that host, so later `git submodule update` runs there keep using HTTPS.
- SSH is attempted **first**, so a host that does have a key still works — and a genuinely private submodule keeps working too.
- Both git invocations run with `GIT_TERMINAL_PROMPT=0` and `GIT_SSH_COMMAND='ssh -o BatchMode=yes'`. A failed attempt has a fallback; a script hung on a username prompt does not.
- The rewrite is mechanical (`[user@]host:path` and `ssh://[user@]host/path` → `https://host/path`), so it works for GitLab and others as well.

If the directory is still empty after both attempts, the script fails and prints the command to run manually.

### Finding the Go toolchain

`sudo` resets `PATH` to `secure_path`, which on Fedora and Debian excludes `/usr/local/go/bin`, so `go` is looked for in `PATH`, then `/usr/local/go/bin`, `/usr/lib/golang/bin`, `/usr/lib/go/bin`, `/opt/go/bin`, `/snap/bin`. Override it explicitly when the toolchain lives elsewhere:

```bash
sudo GO_BINARY=/home/homelab/sdk/go1.24/bin/go python3 scripts/configure.py 27
```

With option `7` and no toolchain found, the script fails rather than silently falling back to a stale binary.

### Compiling as the invoking user

The build runs under `sudo -u $SUDO_USER -H`, so it reuses that account's warm module cache instead of re-downloading every dependency into `/root/go`, and leaves no root-owned artifacts in the repository. `<repo>/tmp/` is chowned to the repository owner when needed. Only when there is no `SUDO_USER` (a real root login) does the compile run as root.

### Installing and starting

The compiled or found binary is copied to `.genix_app.staged` inside the install directory, chowned to the runtime user, `chmod 0750`, then `os.replace`d onto `genix_app` — an atomic rename, so the service never sees a half-copied file. `PathChanged=` watches `IN_MOVED_TO` as well as `IN_CLOSE_WRITE`, so the watcher still notices.

Because the binary lands *before* `genix-restart.path` is restarted, that first change produces no watcher event. The script therefore ends with `systemctl restart genix.service`. A failed start does **not** abort the run — the units are already correct, so it warns and points at `journalctl -u genix.service -n 50`.

### No placeholder binary

The script does **not** create an empty `genix_app`. An earlier version did, and it was counterproductive:

- `genix.service` is enabled, so on the next boot systemd would try to `ExecStart` a zero-byte file, fail with `203/EXEC`, and — with `Restart=always` — loop until the start limit put the unit in a failed state.
- It was not needed by `genix-restart.path`. A systemd path unit watches the closest existing parent directory when its target is absent, so `PathChanged=` fires on the binary's *first* creation. The watcher starts `active` with no file present.
- The deploy does not need the file to pre-exist either: `scripts/deploy_vps.go` rsyncs a `.zst` and runs `zstd -d --force -o <path> && chmod +x <path>`. The setgid `2775` install directory is what grants the write.

A zero-byte `genix_app` left behind by an older run is deleted on the next execution, and the summary warns while the real binary is still missing:

```
[*] WARNING: /usr/local/bin/genix/genix_app does not exist yet. Deploy the backend binary there; genix-restart.path is already watching for it.
```

## Usage

Run the unified installer as root and select precompiled (`28`) or source (`27`):

```bash
sudo python3 scripts/configure.py 28
```

For non-interactive diagnostics, the nested installer accepts its host mode and binary policy:

```bash
sudo python3 scripts/configure/configure_server.py 3 --binary-source precompiled
```

Non-interactive runs must use the nested command above and have every needed key already present in `config.toml`, since there is no terminal to prompt on.

---

## 📊 Useful Commands

- **Check watcher status:**
  ```bash
  systemctl status genix-restart.path
  ```
- **Check application logs:**
  ```bash
  journalctl -u genix.service -f
  ```
- **Check detected runtime user:**
  ```bash
  systemctl cat genix.service
  ```
- **Manually trigger a restart:**
  ```bash
  sudo systemctl start genix-restart.service
  ```
