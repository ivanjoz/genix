package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

type ServerCredentials struct {
	Host             string `json:"host"`
	User             string `json:"user"`
	Key              string `json:"key"`
	Arch             string `json:"arch"`
	RemoteBinaryPath string `json:"bin"`
}

type Credentials struct {
	Servers []ServerCredentials `json:"SERVERS"`
}

func DeployVPS() {
	fmt.Println("Starting VPS deployment...")

	// Read the deploy targets from the shared credentials file so all server rules stay in one place.
	credentialsFilePath := "../credentials.json"
	credentialsContent, readCredentialsError := os.ReadFile(credentialsFilePath)
	if readCredentialsError != nil {
		fmt.Printf("Error reading credentials.json: %v\n", readCredentialsError)
		return
	}

	var credentials Credentials
	if parseCredentialsError := json.Unmarshal(credentialsContent, &credentials); parseCredentialsError != nil {
		fmt.Printf("Error parsing credentials.json: %v\n", parseCredentialsError)
		return
	}

	if len(credentials.Servers) == 0 {
		fmt.Println("Error: SERVERS is empty in credentials.json")
		return
	}

	// Keep generated artifacts out of source folders so deploy runs do not dirty application directories.
	localTempDirectoryPath := "../tmp"
	if createTempDirectoryError := os.MkdirAll(localTempDirectoryPath, 0o755); createTempDirectoryError != nil {
		fmt.Printf("Error creating tmp directory: %v\n", createTempDirectoryError)
		return
	}

	localCompressedBinaryPathsByArchitecture := map[string]string{}

	for _, server := range credentials.Servers {
		targetArchitecture := resolveTargetArchitecture(server.Arch)
		loginUser := server.User
		if loginUser == "" {
			loginUser = "root"
		}

		if server.RemoteBinaryPath == "" {
			fmt.Printf("Error: bin path is empty for host %s\n", server.Host)
			return
		}

		serverTarget := fmt.Sprintf("%s@%s", loginUser, server.Host)
		resolvedKeyPath := expandHomePath(server.Key)
		localCompressedBinaryPath, buildArtifactError := getOrCreateCompressedBinaryForArchitecture(localTempDirectoryPath, targetArchitecture, localCompressedBinaryPathsByArchitecture)
		if buildArtifactError != nil {
			fmt.Printf("Error preparing %s artifact for host %s: %v\n", targetArchitecture, server.Host, buildArtifactError)
			return
		}

		remoteCompressedBinaryPath := server.RemoteBinaryPath + ".zst"

		fmt.Printf("Deploying to %s\n", serverTarget)
		fmt.Printf("Debug: host=%s user=%s arch=%s key=%q bin=%s\n", server.Host, loginUser, targetArchitecture, resolvedKeyPath, server.RemoteBinaryPath)

		hasAutoReloadStrategy, strategyDetectionError := detectAutoReloadStrategy(resolvedKeyPath, serverTarget)
		if strategyDetectionError != nil {
			fmt.Printf("Error checking auto-reload strategy on %s: %v\n", serverTarget, strategyDetectionError)
			return
		}
		fmt.Printf("Debug: auto_reload_strategy=%t on %s\n", hasAutoReloadStrategy, serverTarget)

		// Only stop the service when the server does not have the path-watcher strategy configured.
		if !hasAutoReloadStrategy {
			fmt.Printf("Stopping genix service on %s...\n", serverTarget)
			stopServiceCommand := buildSSHCommand(resolvedKeyPath, serverTarget, "systemctl stop genix")
			stopServiceCommand.Stdout = os.Stdout
			stopServiceCommand.Stderr = os.Stderr

			if stopServiceError := stopServiceCommand.Run(); stopServiceError != nil {
				fmt.Printf("Error stopping service on %s: %v\n", serverTarget, stopServiceError)
				return
			}
		}

		fmt.Printf("Uploading compressed binary to %s:%s...\n", serverTarget, remoteCompressedBinaryPath)
		uploadBinaryCommand := buildRsyncCommand(resolvedKeyPath, localCompressedBinaryPath, serverTarget, remoteCompressedBinaryPath)
		uploadBinaryCommand.Stdout = os.Stdout
		uploadBinaryCommand.Stderr = os.Stderr

		if uploadError := uploadBinaryCommand.Run(); uploadError != nil {
			fmt.Printf("Error uploading binary to %s: %v\n", serverTarget, uploadError)

			if !hasAutoReloadStrategy {
				restartAfterUploadFailureCommand := buildSSHCommand(resolvedKeyPath, serverTarget, "systemctl start genix")
				restartAfterUploadFailureCommand.Stdout = os.Stdout
				restartAfterUploadFailureCommand.Stderr = os.Stderr
				_ = restartAfterUploadFailureCommand.Run()
			}

			return
		}

		fmt.Printf("Decompressing binary on %s...\n", serverTarget)
		decompressRemoteBinaryCommand := buildSSHCommand(
			resolvedKeyPath,
			serverTarget,
			fmt.Sprintf(
				"zstd -d --force %s -o %s && rm %s && chmod +x %s",
				remoteCompressedBinaryPath,
				server.RemoteBinaryPath,
				remoteCompressedBinaryPath,
				server.RemoteBinaryPath,
			),
		)
		decompressRemoteBinaryCommand.Stdout = os.Stdout
		decompressRemoteBinaryCommand.Stderr = os.Stderr

		if decompressError := decompressRemoteBinaryCommand.Run(); decompressError != nil {
			fmt.Printf("Error decompressing on %s: %v\n", serverTarget, decompressError)
			return
		}

		// The watcher strategy will restart automatically after the binary file changes.
		if hasAutoReloadStrategy {
			fmt.Printf("Auto-reload strategy detected on %s. Skipping manual service restart.\n", serverTarget)
			fmt.Printf("Deployment completed for %s.\n", serverTarget)
			continue
		}

		fmt.Printf("Starting genix service on %s...\n", serverTarget)
		startServiceCommand := buildSSHCommand(resolvedKeyPath, serverTarget, "systemctl start genix")
		startServiceCommand.Stdout = os.Stdout
		startServiceCommand.Stderr = os.Stderr

		if startServiceError := startServiceCommand.Run(); startServiceError != nil {
			fmt.Printf("Error starting service on %s: %v\n", serverTarget, startServiceError)
			return
		}

		fmt.Printf("Deployment completed for %s.\n", serverTarget)
	}

	fmt.Println("Deployment complete!")
}

func resolveTargetArchitecture(configuredArchitecture string) string {
	normalizedArchitecture := strings.TrimSpace(strings.ToLower(configuredArchitecture))
	if normalizedArchitecture == "arm64" {
		return "arm64"
	}

	return "amd64"
}

func getOrCreateCompressedBinaryForArchitecture(
	localTempDirectoryPath string,
	targetArchitecture string,
	localCompressedBinaryPathsByArchitecture map[string]string,
) (string, error) {
	if cachedCompressedBinaryPath, hasCachedArtifact := localCompressedBinaryPathsByArchitecture[targetArchitecture]; hasCachedArtifact {
		fmt.Printf("Debug: reusing cached linux/%s artifact: %s\n", targetArchitecture, cachedCompressedBinaryPath)
		return cachedCompressedBinaryPath, nil
	}

	// Build one artifact per target architecture so mixed VPS fleets can reuse the right binary safely.
	localBinaryName := fmt.Sprintf("genix_app_linux_%s", targetArchitecture)
	localBinaryPath := filepath.Join(localTempDirectoryPath, localBinaryName)
	localCompressedBinaryPath := localBinaryPath + ".zst"
	buildDate := time.Now().Format("2006-01-02 15:04:05")
	buildFlags := fmt.Sprintf("-s -w -X 'app/core.BuildDate=%s'", buildDate)

	fmt.Printf("Compiling backend for linux/%s...\n", targetArchitecture)
	buildBackendCommand := exec.Command("go", "build", "-ldflags", buildFlags, "-o", localBinaryPath, ".")
	buildBackendCommand.Dir = "../backend"
	buildBackendCommand.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+targetArchitecture)
	buildBackendCommand.Stdout = os.Stdout
	buildBackendCommand.Stderr = os.Stderr

	if buildBackendError := buildBackendCommand.Run(); buildBackendError != nil {
		return "", buildBackendError
	}
	fmt.Printf("Compilation successful for linux/%s.\n", targetArchitecture)

	fmt.Printf("Compressing linux/%s binary with Zstd...\n", targetArchitecture)
	if compressBinaryError := compressBinary(localBinaryPath, localCompressedBinaryPath); compressBinaryError != nil {
		return "", compressBinaryError
	}
	fmt.Printf("Compression successful for linux/%s.\n", targetArchitecture)

	localCompressedBinaryPathsByArchitecture[targetArchitecture] = localCompressedBinaryPath
	return localCompressedBinaryPath, nil
}

func compressBinary(localBinaryPath string, localCompressedBinaryPath string) error {
	inputBinaryFile, openInputError := os.Open(localBinaryPath)
	if openInputError != nil {
		return openInputError
	}
	defer inputBinaryFile.Close()

	outputCompressedFile, createOutputError := os.Create(localCompressedBinaryPath)
	if createOutputError != nil {
		return createOutputError
	}
	defer outputCompressedFile.Close()

	compressionWriter, createWriterError := zstd.NewWriter(outputCompressedFile)
	if createWriterError != nil {
		return createWriterError
	}
	defer compressionWriter.Close()

	_, copyError := io.Copy(compressionWriter, inputBinaryFile)
	return copyError
}

func detectAutoReloadStrategy(resolvedKeyPath string, serverTarget string) (bool, error) {
	// Check for both units because the path watcher and the helper service are required together.
	checkStrategyCommand := buildSSHCommand(
		resolvedKeyPath,
		serverTarget,
		"systemctl cat genix-restart.path >/dev/null 2>&1 && systemctl cat genix-restart.service >/dev/null 2>&1",
	)

	strategyCheckError := checkStrategyCommand.Run()
	if strategyCheckError == nil {
		return true, nil
	}

	if exitError, ok := strategyCheckError.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
		return false, nil
	}

	return false, strategyCheckError
}

func expandHomePath(originalPath string) string {
	if originalPath == "" || !strings.HasPrefix(originalPath, "~") {
		return originalPath
	}

	homeDirectory, homeDirectoryError := os.UserHomeDir()
	if homeDirectoryError != nil {
		return originalPath
	}

	return filepath.Join(homeDirectory, strings.TrimPrefix(originalPath, "~"))
}

func buildSSHCommand(resolvedKeyPath string, serverTarget string, remoteCommand string) *exec.Cmd {
	sshArguments := []string{}
	if resolvedKeyPath != "" {
		sshArguments = append(sshArguments, "-i", resolvedKeyPath)
	}

	sshArguments = append(sshArguments, serverTarget, remoteCommand)

	command := exec.Command("ssh", sshArguments...)
	return command
}

func buildRsyncCommand(resolvedKeyPath string, localCompressedBinaryPath string, serverTarget string, remoteCompressedBinaryPath string) *exec.Cmd {
	sshTransportCommand := "ssh"
	if resolvedKeyPath != "" {
		sshTransportCommand = fmt.Sprintf("ssh -i %s", resolvedKeyPath)
	}

	return exec.Command(
		"rsync",
		"-ahP",
		"-e", sshTransportCommand,
		localCompressedBinaryPath,
		fmt.Sprintf("%s:%s", serverTarget, remoteCompressedBinaryPath),
	)
}
