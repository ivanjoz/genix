package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pelletier/go-toml/v2"
)

// Same path that scripts/configure_server.py hardcodes in SERVICE_BINARY_PATH and writes into
// ExecStart= and PathChanged=, so "bin" only needs to be set for servers configured by hand.
const defaultRemoteBinaryPath = "/usr/local/bin/genix/genix_app"

type ServerCredentials struct {
	Host             string `toml:"host"`
	User             string `toml:"user"`
	Key              string `toml:"key"`
	Arch             string `toml:"arch"`
	RemoteBinaryPath string `toml:"bin"`
}

type Credentials struct {
	Servers []ServerCredentials `toml:"servers"`
}

func DeployVPS() {
	fmt.Println("Starting VPS deployment...")

	// Use the environment selected by deploy.sh, with the original path as a standalone fallback.
	configFilePath := strings.TrimSpace(os.Getenv("GENIX_CONFIG_FILE"))
	if configFilePath == "" {
		configFilePath = "../config.toml"
	}
	fmt.Printf("Reading VPS deployment targets from: %s\n", configFilePath)
	configContent, readConfigError := os.ReadFile(configFilePath)
	if readConfigError != nil {
		fmt.Printf("Error reading config file %s: %v\n", configFilePath, readConfigError)
		return
	}

	var credentials Credentials
	if parseConfigError := toml.Unmarshal(configContent, &credentials); parseConfigError != nil {
		fmt.Printf("Error parsing config file %s: %v\n", configFilePath, parseConfigError)
		return
	}

	if len(credentials.Servers) == 0 {
		fmt.Printf("Error: servers is empty in %s\n", configFilePath)
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

		remoteBinaryPath := strings.TrimSpace(server.RemoteBinaryPath)
		if remoteBinaryPath == "" {
			remoteBinaryPath = defaultRemoteBinaryPath
		}

		serverTarget := fmt.Sprintf("%s@%s", loginUser, server.Host)
		resolvedKeyPath := expandHomePath(server.Key)
		localCompressedBinaryPath, buildArtifactError := getOrCreateCompressedBinaryForArchitecture(localTempDirectoryPath, targetArchitecture, localCompressedBinaryPathsByArchitecture)
		if buildArtifactError != nil {
			fmt.Printf("Error preparing %s artifact for host %s: %v\n", targetArchitecture, server.Host, buildArtifactError)
			return
		}

		remoteCompressedBinaryPath := remoteBinaryPath + ".zst"

		fmt.Printf("Deploying to %s\n", serverTarget)
		fmt.Printf("Debug: host=%s user=%s arch=%s key=%q bin=%s\n", server.Host, loginUser, targetArchitecture, resolvedKeyPath, remoteBinaryPath)

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
				remoteBinaryPath,
				remoteCompressedBinaryPath,
				remoteBinaryPath,
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
