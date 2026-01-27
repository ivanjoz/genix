// Package install provides systemd service installation and management for the homelab server.
package install

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// ServiceName is the name of the systemd service
	ServiceName = "genix-bridge"
	// BinaryName is the name of the compiled binary
	BinaryName = "genix-bridge"
	// InstallPath is where the binary will be installed
	InstallPath = "/usr/local/bin/" + BinaryName
)

// InstallOptions contains installation configuration options
type InstallOptions struct {
	// WorkingDirectory is the directory where the service will run
	// (should be the project root where credentials.json is located)
	WorkingDirectory string
}

// RunInstall performs the complete installation of the server as a systemd service
func RunInstall() error {
	log.Println("Starting installation of genix-bridge...")

	// Check if running as root (needed for installing to /usr/local/bin and creating systemd service)
	if os.Geteuid() != 0 {
		return fmt.Errorf("installation must be run with sudo privileges")
	}

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	opts := InstallOptions{
		WorkingDirectory: wd,
	}

	// Step 1: Build the binary
	log.Println("Step 1: Building the binary...")
	if err := buildBinary(); err != nil {
		return fmt.Errorf("failed to build binary: %w", err)
	}

	// Step 2: Install the binary
	log.Println("Step 2: Installing binary to", InstallPath)
	if err := installBinary(); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	// Step 3: Create systemd service
	log.Println("Step 3: Creating systemd service...")
	if err := createSystemdService(opts); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Step 4: Enable and start the service
	log.Println("Step 4: Enabling and starting the service...")
	if err := enableAndStartService(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// RunUninstall performs the complete removal of the server and systemd service
func RunUninstall() error {
	log.Println("Starting uninstallation of homelab-p2p-bridge...")

	// Check if running as root (needed for removing systemd service and binary)
	if os.Geteuid() != 0 {
		return fmt.Errorf("uninstallation must be run with sudo privileges")
	}

	// Step 1: Stop the service if running
	log.Println("Step 1: Stopping the service...")
	if err := runCommand("systemctl", "stop", ServiceName); err != nil {
		log.Printf("Warning: Failed to stop service (may not be running): %v", err)
	}

	// Step 2: Disable the service
	log.Println("Step 2: Disabling the service...")
	if err := runCommand("systemctl", "disable", ServiceName); err != nil {
		log.Printf("Warning: Failed to disable service: %v", err)
	}

	// Step 3: Remove systemd service file
	log.Println("Step 3: Removing systemd service file...")
	servicePath := "/etc/systemd/system/" + ServiceName + ".service"
	if _, err := os.Stat(servicePath); err == nil {
		if err := os.Remove(servicePath); err != nil {
			return fmt.Errorf("failed to remove service file: %w", err)
		}
		log.Printf("Service file removed: %s", servicePath)
	} else {
		log.Printf("Service file not found (may already be uninstalled): %s", servicePath)
	}

	// Step 4: Reload systemd daemon
	log.Println("Step 4: Reloading systemd daemon...")
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Step 5: Remove the binary
	log.Println("Step 5: Removing binary...")
	if _, err := os.Stat(InstallPath); err == nil {
		if err := os.Remove(InstallPath); err != nil {
			return fmt.Errorf("failed to remove binary: %w", err)
		}
		log.Printf("Binary removed: %s", InstallPath)
	} else {
		log.Printf("Binary not found (may already be removed): %s", InstallPath)
	}

	// Step 6: Reset systemd's failed state if any
	log.Println("Step 6: Cleaning up systemd state...")
	if err := runCommand("systemctl", "reset-failed", ServiceName); err != nil {
		log.Printf("Note: Failed state reset (may not apply): %v", err)
	}

	log.Println("\nUninstallation complete. The homelab-p2p-bridge service and binary have been removed.")
	return nil
}

// buildBinary compiles the server for the current platform
func buildBinary() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	outputPath := filepath.Join(cwd, BinaryName)
	log.Printf("Building to: %s", outputPath)

	// Build using go build directly
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// installBinary moves the compiled binary to the install location
func installBinary() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	sourcePath := filepath.Join(cwd, BinaryName)

	// Check if the binary exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found at %s", sourcePath)
	}

	// Remove old binary if it exists
	if _, err := os.Stat(InstallPath); err == nil {
		log.Printf("Removing old binary at %s", InstallPath)
		if err := os.Remove(InstallPath); err != nil {
			return fmt.Errorf("failed to remove old binary: %w", err)
		}
	}

	// Move the binary to install location
	if err := os.Rename(sourcePath, InstallPath); err != nil {
		return fmt.Errorf("failed to move binary to %s: %w", InstallPath, err)
	}

	// Set executable permissions
	if err := os.Chmod(InstallPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	log.Printf("Binary installed successfully to %s", InstallPath)
	return nil
}

// createSystemdService creates the systemd service file
func createSystemdService(opts InstallOptions) error {
	servicePath := "/etc/systemd/system/" + ServiceName + ".service"

	// Check if service already exists
	if _, err := os.Stat(servicePath); err == nil {
		log.Printf("Service file already exists at %s", servicePath)
		return nil
	}

	// Create systemd service file content
	serviceContent := fmt.Sprintf(`[Unit]
Description=Home Lab P2P Bridge Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, opts.WorkingDirectory, InstallPath)

	// Write service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	log.Printf("Systemd service created at %s", servicePath)
	return nil
}

// enableAndStartService enables and starts the systemd service
func enableAndStartService() error {
	// Reload systemd daemon
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable the service
	if err := runCommand("systemctl", "enable", ServiceName); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Start the service
	if err := runCommand("systemctl", "start", ServiceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait a moment and check service status
	if err := runCommand("sleep", "2"); err != nil {
		return fmt.Errorf("sleep command failed: %w", err)
	}

	log.Println("\nService status:")
	runCommand("systemctl", "status", ServiceName)

	return nil
}

// runCommand executes a system command and streams output
func runCommand(name string, args ...string) error {
	log.Printf("Executing: %s %v", name, args)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
