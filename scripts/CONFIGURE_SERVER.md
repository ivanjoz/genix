# Genix Service Deployment & Auto-Reload Strategy

This document explains the systemd-based strategy for automatically reloading the **Genix Backend Service** when its binary is updated. The setup script itself must be executed as `root`, but the application binary and the main service run as a non-root user.

---

## 🚀 Strategy Overview

The setup consists of three systemd units working together:
1.  **`genix.service`**: The main application service running as `ubuntu` or the default non-root `sudo` user.
2.  **`genix-restart.path`**: A watcher that monitors the deployed binary for changes.
3.  **`genix-restart.service`**: A root-owned helper triggered by the path watcher to restart the main service.

### How it works:
1.  **Install**: Run `sudo python3 scripts/configure_server.py` or `sudo ./app.sh configure_server`.
2.  **Prepare**: The script creates `/usr/local/bin/genix/`, assigns ownership to `ubuntu` when present, or falls back to the non-root `SUDO_USER`.
3.  **Update**: You copy a new executable to `/usr/local/bin/genix/genix_app`.
4.  **Detect**: The `genix-restart.path` unit detects the binary change.
5.  **Restart**: `genix-restart.service` restarts `genix.service`, which keeps running as the non-root runtime user.

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

It also ensures `/usr/local/bin/genix/genix_app` is owned by the same non-root user and is executable.

## Usage

Run the installer as root:

```bash
sudo ./app.sh configure_server
```

Or directly:

```bash
sudo python3 scripts/configure_server.py
```

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
