package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
)

type Credentials struct {
	VPS_IP               string `json:"VPS_IP"`
	VPS_KEY              string `json:"VPS_KEY"`
	VPS_INSTALL_LOCATION string `json:"VPS_INSTALL_LOCATION"`
}

func DeployVPS() {
	fmt.Println("Starting VPS deployment...")

	// 1. Read credentials.json
	credPath := "../credentials.json"
	content, err := os.ReadFile(credPath)
	if err != nil {
		fmt.Printf("Error reading credentials.json: %v\n", err)
		return
	}

	var creds Credentials
	if err := json.Unmarshal(content, &creds); err != nil {
		fmt.Printf("Error parsing credentials.json: %v\n", err)
		return
	}

	// 2. Compile backend for linux amd64
	fmt.Println("Compiling backend for linux/amd64...")
	binaryName := "genix_app"
	buildDate := time.Now().Format("2006-01-02 15:04:05")
	ldflags := fmt.Sprintf("-s -w -X 'app/core.BuildDate=%s'", buildDate)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryName, ".")
	cmd.Dir = "../backend"
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error compiling backend: %v\n", err)
		return
	}
	fmt.Println("Compilation successful.")

	// 3. Compress with Zstd
	fmt.Println("Compressing binary with Zstd...")
	binaryPath := filepath.Join("../backend", binaryName)
	compressedPath := binaryPath + ".zst"

	err = func() error {
		inputFile, err := os.Open(binaryPath)
		if err != nil {
			return err
		}
		defer inputFile.Close()

		outputFile, err := os.Create(compressedPath)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		encoder, err := zstd.NewWriter(outputFile)
		if err != nil {
			return err
		}
		defer encoder.Close()

		_, err = io.Copy(encoder, inputFile)
		return err
	}()

	if err != nil {
		fmt.Printf("Error compressing binary: %v\n", err)
		return
	}
	fmt.Println("Compression successful.")

	// Expand tilde in key path if present
	keyPath := creds.VPS_KEY
	if len(keyPath) > 0 && keyPath[0] == '~' {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[1:])
	}

	// 4. Stop service before upload
	fmt.Println("Stopping genix service on VPS...")
	stopCmd := exec.Command("ssh", "-i", keyPath, fmt.Sprintf("root@%s", creds.VPS_IP), "systemctl stop genix")
	_ = stopCmd.Run() // Ignore error if it's already stopped

	// 5. Upload to VPS with rsync for progress
	targetCompressed := fmt.Sprintf("%s.zst", creds.VPS_INSTALL_LOCATION)
	fmt.Printf("Uploading compressed binary to root@%s:%s...\n", creds.VPS_IP, targetCompressed)
	
	// Using rsync -P to show progress
	rsyncCmd := exec.Command("rsync", "-ahP", "-e", fmt.Sprintf("ssh -i %s", keyPath), compressedPath, fmt.Sprintf("root@%s:%s", creds.VPS_IP, targetCompressed))
	rsyncCmd.Stdout = os.Stdout
	rsyncCmd.Stderr = os.Stderr

	if err := rsyncCmd.Run(); err != nil {
		fmt.Printf("Error uploading binary: %v\n", err)
		exec.Command("ssh", "-i", keyPath, fmt.Sprintf("root@%s", creds.VPS_IP), "systemctl start genix").Run()
		return
	}
	fmt.Println("Upload successful.")

	// 6. Decompress on VPS
	fmt.Println("Decompressing binary on VPS...")
	decompressCmd := exec.Command("ssh", "-i", keyPath, fmt.Sprintf("root@%s", creds.VPS_IP), 
		fmt.Sprintf("zstd -d --force %s -o %s && rm %s && chmod +x %s", 
			targetCompressed, creds.VPS_INSTALL_LOCATION, targetCompressed, creds.VPS_INSTALL_LOCATION))
	decompressCmd.Stdout = os.Stdout
	decompressCmd.Stderr = os.Stderr

	if err := decompressCmd.Run(); err != nil {
		fmt.Printf("Error decompressing on VPS: %v\n", err)
		return
	}

	// 7. Start service
	fmt.Println("Starting genix service on VPS...")
	startCmd := exec.Command("ssh", "-i", keyPath, fmt.Sprintf("root@%s", creds.VPS_IP), "systemctl start genix")
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr

	if err := startCmd.Run(); err != nil {
		fmt.Printf("Error starting service: %v\n", err)
		return
	}
	fmt.Println("Service started successfully. Deployment complete!")
}
