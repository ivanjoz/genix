# Install Package

The `install` package provides systemd service installation and management functionality for the homelab P2P bridge server.

## Overview

This package handles all aspects of installing the server as a systemd service, including:
- Building the server binary
- Installing the binary to `/usr/local/bin/`
- Creating and configuring a systemd service
- Enabling and starting the service
- Uninstalling the service and cleaning up

## Constants

### Service Management
```go
const (
    ServiceName = "homelab-p2p-bridge"  // Name of the systemd service
    BinaryName  = "homelab-p2p-bridge"  // Name of the compiled binary
    InstallPath = "/usr/local/bin/homelab-p2p-bridge"  // Installation directory
)
```

## Types

### InstallOptions

```go
type InstallOptions struct {
    WorkingDirectory string  // Directory where service runs (should contain credentials.json)
}
```

## Functions

### RunInstall

Performs complete installation of the server as a systemd service.

```go
func RunInstall() error
```

**Steps performed:**
1. Checks for root/sudo privileges
2. Builds the server binary
3. Installs binary to `/usr/local/bin/homelab-p2p-bridge`
4. Creates systemd service file
5. Enables and starts the service

**Returns:** `nil` on success, error on failure

**Example:**
```go
if err := install.RunInstall(); err != nil {
    log.Fatalf("Installation failed: %v", err)
}
```

### RunUninstall

Performs complete removal of the server and systemd service.

```go
func RunUninstall() error
```

**Steps performed:**
1. Stops the service if running
2. Disables the service
3. Removes systemd service file
4. Reloads systemd daemon
5. Removes the binary
6. Cleans up systemd state

**Returns:** `nil` on success, error on failure

**Example:**
```go
if err := install.RunUninstall(); err != nil {
    log.Fatalf("Uninstallation failed: %v", err)
}
```

## Usage

### Command Line Installation

The package is typically used through command-line flags in `main.go`:

```bash
# Install as systemd service
cd homelab_server
sudo go run main.go --install

# Uninstall service
sudo go run main.go --uninstall
```

### Programmatic Installation

You can also use the package programmatically:

```go
package main

import (
    "log"
    "p2p_bridge/homelab_server/install"
)

func main() {
    // Install the server
    if err := install.RunInstall(); err != nil {
        log.Fatalf("Failed to install: %v", err)
    }
    
    log.Printf("Service installed: %s", install.ServiceName)
    log.Printf("Binary location: %s", install.InstallPath)
}
```

## Requirements

- Root/sudo privileges required for installation and uninstallation
- systemd must be available on the system
- Go must be installed for building the binary
- `credentials.json` must exist in the working directory

## Service Configuration

The installed systemd service is configured with:

- **Type:** `simple`
- **User:** `root`
- **Working Directory:** The directory where `--install` was run
- **Auto-restart:** Always (5-second delay)
- **Logging:** Output sent to systemd journal
- **Boot start:** Enabled (auto-starts on system boot)

## Service Management

After installation, manage the service using standard systemd commands:

```bash
# Check service status
sudo systemctl status homelab-p2p-bridge

# View real-time logs
sudo journalctl -u homelab-p2p-bridge -f

# Stop the service
sudo systemctl stop homelab-p2p-bridge

# Start the service
sudo systemctl start homelab-p2p-bridge

# Restart the service
sudo systemctl restart homelab-p2p-bridge

# Reload service after config changes
sudo systemctl daemon-reload
```

## Error Handling

The package provides descriptive error messages for common issues:

- **Missing sudo privileges:** "installation must be run with sudo privileges"
- **Build failures:** "failed to build binary: {specific error}"
- **Permission errors:** "failed to set executable permissions: {specific error}"
- **Service conflicts:** Warns if service or binary already exists

## Security Considerations

- The service runs as `root` by default
- Ensure `credentials.json` has appropriate file permissions (0600)
- The binary is installed in `/usr/local/bin/` which is in the system PATH
- All service actions are logged to the systemd journal

## Troubleshooting

### Installation Fails
- Verify you have sudo privileges
- Check that `go` is in your PATH
- Ensure `go.mod` exists in the working directory
- Check for existing service or binary conflicts

### Service Won't Start
- Check logs: `sudo journalctl -u homelab-p2p-bridge -n 50`
- Verify `credentials.json` exists in the working directory
- Ensure network is available (service runs `After=network.target`)
- Check WebSocket URL in configuration

### Uninstall Fails
- Stop the service manually first: `sudo systemctl stop homelab-p2p-bridge`
- Remove files manually if needed
- Reset systemd: `sudo systemctl daemon-reload`

## Example Output

### Installation
```
2024/01/26 10:00:00 Starting installation of homelab-p2p-bridge...
2024/01/26 10:00:00 Step 1: Building the binary...
2024/01/26 10:00:01 Building to: /home/user/projects/genix/p2p/homelab_server/homelab-p2p-bridge
2024/01/26 10:00:02 Step 2: Installing binary to /usr/local/bin/homelab-p2p-bridge
2024/01/26 10:00:02 Removing old binary at /usr/local/bin/homelab-p2p-bridge
2024/01/26 10:00:02 Binary installed successfully to /usr/local/bin/homelab-p2p-bridge
2024/01/26 10:00:02 Step 3: Creating systemd service...
2024/01/26 10:00:02 Systemd service created at /etc/systemd/system/homelab-p2p-bridge.service
2024/01/26 10:00:02 Step 4: Enabling and starting the service...
2024/01/26 10:00:02 Executing: systemctl daemon-reload
2024/01/26 10:00:02 Executing: systemctl enable homelab-p2p-bridge
2024/01/26 10:00:03 Executing: systemctl start homelab-p2p-bridge

Service status:
● homelab-p2p-bridge.service - Home Lab P2P Bridge Server
     Loaded: loaded (/etc/systemd/system/homelab-p2p-bridge.service; enabled; preset: disabled)
     Active: active (running) since Fri 2024-01-26 10:00:03 UTC; 50ms ago
   Main PID: 12345 (homelab-p2p-br)
      Tasks: 5 (limit: 4915)
        CPU: 12ms
2024/01/26 10:00:05 Installation completed successfully!
2024/01/26 10:00:05 Service status can be checked with: systemctl status homelab-p2p-bridge
```

### Uninstallation
```
2024/01/26 10:00:00 Starting uninstallation of homelab-p2p-bridge...
2024/01/26 10:00:00 Step 1: Stopping the service...
2024/01/26 10:00:00 Executing: systemctl stop homelab-p2p-bridge
2024/01/26 10:00:00 Step 2: Disabling the service...
2024/01/26 10:00:01 Executing: systemctl disable homelab-p2p-bridge
2024/01/26 10:00:01 Step 3: Removing systemd service file...
2024/01/26 10:00:01 Service file removed: /etc/systemd/system/homelab-p2p-bridge.service
2024/01/26 10:00:01 Step 4: Reloading systemd daemon...
2024/01/26 10:00:01 Executing: systemctl daemon-reload
2024/01/26 10:00:01 Step 5: Removing binary...
2024/01/26 10:00:01 Binary removed: /usr/local/bin/homelab-p2p-bridge
2024/01/26 10:00:01 Step 6: Cleaning up systemd state...
2024/01/26 10:00:01 Executing: systemctl reset-failed homelab-p2p-bridge

Uninstallation complete. The homelab-p2p-bridge service and binary have been removed.
2024/01/26 10:00:01 Uninstallation completed successfully!
```

## See Also

- [Main Server Documentation](../../README.md)
- [Configuration Package](../../config/README.md)
- [Deployment Guide](../../DEPLOYMENT.md)