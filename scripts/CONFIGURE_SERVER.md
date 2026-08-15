# Genix Service Deployment & Auto-Reload Strategy

This document explains the systemd-based strategy for automatically reloading the **Genix Backend Service** when its binary is updated, and the Nginx reverse proxy that fronts it. The setup script itself must be executed as `root`, but the application binary and the main service run as a non-root user.

---

## 🧩 Install Modes

The backend host and the Nginx edge host are usually two different machines, so the script asks which side to configure:

| Mode | Name | What it configures | config.toml keys used |
| ---- | ---- | ------------------ | -------------------------- |
| `1` | Full | systemd units + Nginx reverse proxy | `server.port`, `server.nginx_domain`, `server.nginx_process` |
| `2` | Only Systemd Service | systemd units, binary directory, runtime user | `server.port` |
| `3` | Only Nginx Proxy | `/etc/nginx/conf.d/<server.nginx_domain>.conf` | `server.nginx_domain`, `server.nginx_process` |

Pass the mode as an argument (`1`/`full`, `2`/`systemd`, `3`/`nginx`) or answer the interactive prompt.

```bash
sudo ./app.sh configure_server 3    # Nginx VPS
sudo ./app.sh configure_server 2    # backend host
```

Only the credentials the selected mode needs are validated, so the Nginx host does not need `server.port` and the backend host does not need `server.nginx_domain`.

### Missing credentials are requested on the terminal

`config.toml` is optional. When the file is absent — the usual case on an Nginx-only VPS that has no full clone — or when a key the selected mode needs is missing or malformed, the script asks for that value and re-asks until it validates:

```
[*] No config.toml at /home/ubuntu/genix/config.toml. Values will be requested.
[*] server.nginx_domain is not set in config.toml.
Enter server.nginx_domain, the public domain Nginx serves (e.g. genix-api-4.un.pe): bad domain
[!] 'bad domain' is not a valid domain name (expected e.g. genix-api-4.un.pe).
Enter server.nginx_domain, the public domain Nginx serves (e.g. genix-api-4.un.pe): genix-api-4.un.pe
```

Validation rules:

- `server.nginx_domain` — a dotted hostname; lower-cased and stripped of a trailing dot.
- `server.nginx_process` — `host:port`, optionally prefixed with `http://` or `https://`. The port is mandatory.
- `server.port` — an integer in `1-65535`.

After everything resolves, the script offers to write the typed-in values back so the next run is non-interactive:

```
[*] These values were typed in and are not stored yet: server.nginx_domain=genix-api-4.un.pe, server.nginx_process=100.64.0.2:14010
Save them to /home/ubuntu/genix/config.toml? [Y/n]:
```

Answering yes merges the keys into the existing file, leaving all other keys intact. A newly created file is chowned to the owner of the repository directory and set to `0600`, since it holds secrets and the service reads it as a non-root user.

Two things still fail the run instead of prompting:

- A `config.toml` that exists but is not valid TOML — prompting would later overwrite a real but broken file.
- A missing value with no TTY attached (cron, CI, `ssh` without `-t`). The error names the exact key to add.

In **mode 2**, note that `genix.service` points `GENIX_CONFIG_FILE` at this path and the backend panics without it, so installing the units with no `config.toml` present prints a warning: the service is installed but will not start until the file is there.

### Split-host wiring

- `server.nginx_domain` (e.g. `genix-api-4.un.pe`) becomes `server_name`, the config file name, and the Let's Encrypt certificate directory that is looked up under `/etc/letsencrypt/live/`.
- `server.nginx_process` (e.g. `100.64.0.2:14010`) becomes the `proxy_pass` upstream. A bare `host:port` is prefixed with `http://`; a value that already carries a scheme is used verbatim.
- `server.port` (e.g. `14010`) is exported as `Environment=SERVER_PORT=` in `genix.service` and is the port the backend binds to. It must equal the port half of `server.nginx_process`.

The backend resolves its listen port as: `SERVER_PORT` env var → `server.port` in `config.toml` → `3589`.

---

## 🚀 Strategy Overview

The setup consists of three systemd units working together:
1.  **`genix.service`**: The main application service running as `ubuntu` or the default non-root `sudo` user.
2.  **`genix-restart.path`**: A watcher that monitors the deployed binary for changes.
3.  **`genix-restart.service`**: A root-owned helper triggered by the path watcher to restart the main service.

### How it works:
1.  **Install**: Run `sudo python3 scripts/configure_server.py 2` or `sudo ./app.sh configure_server 2`.
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
Environment=SERVER_PORT=14010
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

In modes `1` and `2` the script guarantees a runnable binary at `/usr/local/bin/genix/genix_app` before it writes the units, in this order:

1. **Compile from source** — if `<repo>/backend/go.mod` and `<repo>/backend/main.go` both exist, it runs
   `go build -ldflags "-s -w -X 'app/core.BuildDate=<now>'" -o <repo>/tmp/genix_app_linux_<arch> .`
   in `backend/`, matching `scripts/deploy_vps.go` so both paths embed the same build metadata.
2. **Reuse a prebuilt binary** — with no source present, it searches in order:
   `/usr/local/bin/genix/genix_app` → `<repo>/tmp/genix_app_linux_<arch>` → `<repo>/backend/genix_app` → `<repo>/genix_app`.
   A candidate counts only if it is non-empty and starts with the ELF magic, so an old empty placeholder or a stray shell script is skipped.
3. **Crash** — neither source nor binary is a hard error naming both places it looked.

### Downloading a public prebuilt backend

When deploying from a minimal tree without `backend/main.go`, place the verified release asset at
`tmp/genix_app_linux_<arch>` and the normal search order above installs it. The script intentionally
does not fetch the network by itself: the operator chooses an immutable version or the moving
`latest` release explicitly.

```bash
# Resolve the filename that configure_server.py already searches for on this host.
case "$(uname -m)" in
  x86_64) release_architecture=amd64 ;;
  aarch64|arm64) release_architecture=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# For production, replace latest/download with download/vX.Y.Z before downloading.
release_base_url=https://github.com/ivanjoz/genix/releases/latest/download
release_asset="genix_app_linux_${release_architecture}"
mkdir -p tmp
curl --fail --location --output "tmp/${release_asset}" "${release_base_url}/${release_asset}"
curl --fail --location --output tmp/SHA256SUMS "${release_base_url}/SHA256SUMS"

# Verify from tmp/ so the manifest's bare filename resolves to the downloaded binary.
(
  cd tmp
  grep " ${release_asset}$" SHA256SUMS | sha256sum --check --strict
)
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
sudo GO_BINARY=/home/homelab/sdk/go1.24/bin/go python3 scripts/configure_server.py 2
```

With source present and no toolchain found, the script fails rather than silently falling back to a stale binary.

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

Run the installer as root, optionally passing the install mode:

```bash
sudo ./app.sh configure_server        # prompts for the mode
sudo ./app.sh configure_server 1      # full
sudo ./app.sh configure_server 2      # only systemd service
sudo ./app.sh configure_server 3      # only Nginx proxy
```

Or directly:

```bash
sudo python3 scripts/configure_server.py 3
```

Non-interactive runs (cron, CI, `ssh` without a TTY) must pass the mode and have every needed key already present in `config.toml`, since there is no terminal to prompt on.

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
