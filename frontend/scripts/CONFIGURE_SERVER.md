# Genix Service Deployment & Auto-Reload Strategy

This document explains the current systemd-based strategy for automatically reloading the **Genix Backend Service** when its binary is updated. This approach allows a user (e.g., `ivanjoz`) to update the application binary via SSH/SCP without needing `sudo` or root access to restart the service manually.

---

## 🚀 Strategy Overview

The setup consists of three systemd units working together:
1.  **`genix.service`**: The main application service.
2.  **`genix-restart.path`**: A "watcher" that monitors the binary file for changes.
3.  **`genix-restart.service`**: A "restarter" service triggered by the path watcher.

### How it works:
1.  **Update**: You push a new binary to `/home/ivanjoz/genix/backend/app`.
2.  **Detect**: The `genix-restart.path` unit detects that the file has been modified and closed.
3.  **Trigger**: The path unit immediately starts `genix-restart.service`.
4.  **Restart**: `genix-restart.service` (running as root) executes `systemctl restart genix.service`.

---

## 🛠 Service Configurations

### 1. Main Service: `genix.service`
Located at: `/etc/systemd/system/genix.service`
This service runs the backend application with security hardening.

```ini
[Unit]
Description=Genix Backend Service
After=network.target

[Service]
Type=simple
User=ivanjoz
Group=ivanjoz
WorkingDirectory=/home/ivanjoz/genix/backend
ExecStart=/home/ivanjoz/genix/backend/app
Restart=always
RestartSec=5

# Security Hardening
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/home/ivanjoz/genix
PrivateTmp=yes
NoNewPrivileges=yes
CapabilityBoundingSet=
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictRealtime=yes

[Install]
WantedBy=multi-user.target
```

### 2. Path Watcher: `genix-restart.path`
Located at: `/etc/systemd/system/genix-restart.path`
This monitors the binary for modifications (`PathChanged`).

```ini
[Unit]
Description=Watch for changes to genix backend binary

[Path]
PathChanged=/home/ivanjoz/genix/backend/app

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
- **Manually trigger a restart:**
  ```bash
  sudo systemctl start genix-restart.service
  ```
