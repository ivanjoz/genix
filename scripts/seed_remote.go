package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
)

// seed_remote runs genix-seed on a VPS, next to its database: each write is a local round trip
// there instead of one across the internet, and the files the seed stores land in the server's own
// file store. The run is a transient systemd unit, so it survives the SSH session; Ctrl+C while
// following only detaches, and the menu that follows offers to abort it at its next checkpoint.
//
//	go run . seed_remote [--server=host] run <module,...|all> [--flag=value ...]
//	go run . seed_remote [--server=host] logs | status | abort

const (
	remoteSeedBinaryPath = "/usr/local/bin/genix/genix_seed"
	seedUnitName         = "genix-seed"
	seedRemoteUsage      = "usage: seed_remote [--server=host] run <module,...|all> [--flag=value ...] | logs | status | abort"
)

type seedServer struct {
	target       string
	keyPath      string
	architecture string
}

func SeedRemote(arguments []string) {
	server, arguments, err := pickSeedServer(arguments)
	if err == nil && len(arguments) == 0 {
		err = fmt.Errorf("%s", seedRemoteUsage)
	}
	if err == nil {
		switch arguments[0] {
		case "run":
			err = startRemoteSeed(server, arguments[1:])
		case "logs":
			err = followUntilDetached(server)
		case "status":
			err = runRemoteCommand(server, fmt.Sprintf("systemctl status --no-pager %s; %s status", seedUnitName, remoteSeedBinaryPath))
		case "abort":
			err = abortRemoteSeed(server)
			if err == nil {
				err = followUntilDetached(server)
			}
		default:
			err = fmt.Errorf("%s", seedRemoteUsage)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// pickSeedServer takes --server=host off the arguments; without it the config must list one server.
func pickSeedServer(arguments []string) (seedServer, []string, error) {
	credentials, err := loadServerCredentials()
	if err != nil {
		return seedServer{}, nil, err
	}
	requestedHost := ""
	if len(arguments) > 0 && strings.HasPrefix(arguments[0], "--server=") {
		requestedHost, arguments = strings.TrimPrefix(arguments[0], "--server="), arguments[1:]
	}
	hosts := []string{}
	for _, server := range credentials.Servers {
		hosts = append(hosts, server.Host)
	}
	serverIndex := slices.Index(hosts, requestedHost)
	if requestedHost == "" && len(hosts) == 1 {
		serverIndex = 0
	}
	if serverIndex < 0 {
		return seedServer{}, nil, fmt.Errorf("pick a server with --server=<host>: %s", strings.Join(hosts, ", "))
	}
	server := credentials.Servers[serverIndex]
	loginUser := server.User
	if loginUser == "" {
		loginUser = "root"
	}
	return seedServer{target: loginUser + "@" + server.Host, keyPath: expandHomePath(server.Key),
		architecture: resolveTargetArchitecture(server.Arch)}, arguments, nil
}

// startRemoteSeed builds and uploads genix-seed, starts it as the genix service's own user, with
// its working directory and config, and follows it.
func startRemoteSeed(server seedServer, seedArguments []string) error {
	if isRemoteSeedActive(server) {
		return fmt.Errorf("a seed is already running on %s: use logs or abort", server.target)
	}
	compressedBinaryPath, err := buildCompressedSeedBinary(server.architecture)
	if err != nil {
		return err
	}
	fmt.Printf("Uploading genix-seed to %s:%s...\n", server.target, remoteSeedBinaryPath)
	uploadCommand := buildRsyncCommand(server.keyPath, compressedBinaryPath, server.target, remoteSeedBinaryPath+".zst")
	uploadCommand.Stdout, uploadCommand.Stderr = os.Stdout, os.Stderr
	if err := uploadCommand.Run(); err != nil {
		return fmt.Errorf("uploading: %w", err)
	}
	if err := runRemoteCommand(server, fmt.Sprintf("zstd -d -q --force %[1]s.zst -o %[1]s && rm %[1]s.zst && chmod +x %[1]s", remoteSeedBinaryPath)); err != nil {
		return fmt.Errorf("decompressing: %w", err)
	}

	serviceProperties, err := readGenixServiceProperties(server)
	if err != nil {
		return err
	}
	// systemd-run gives the run no terminal: --yes skips the confirmation, the plan still prints.
	runCommand := []string{"systemd-run", "--quiet", "--unit=" + seedUnitName, "--collect"}
	if serviceProperties["User"] != "" {
		runCommand = append(runCommand, "--uid="+serviceProperties["User"])
	}
	if serviceProperties["WorkingDirectory"] != "" {
		runCommand = append(runCommand, "--working-directory="+serviceProperties["WorkingDirectory"])
	}
	for _, environmentEntry := range strings.Fields(serviceProperties["Environment"]) {
		if strings.HasPrefix(environmentEntry, "GENIX_") {
			runCommand = append(runCommand, "--setenv="+environmentEntry)
		}
	}
	runCommand = append(runCommand, remoteSeedBinaryPath, "run")
	runCommand = append(runCommand, seedArguments...)
	runCommand = append(runCommand, "--yes")
	quotedCommand := []string{}
	for _, part := range runCommand {
		quotedCommand = append(quotedCommand, shellQuote(part))
	}
	fmt.Printf("Starting %s on %s...\n", seedUnitName, server.target)
	if err := runRemoteCommand(server, strings.Join(quotedCommand, " ")); err != nil {
		return fmt.Errorf("starting the unit: %w", err)
	}
	return followUntilDetached(server)
}

// buildCompressedSeedBinary cross-compiles genix-seed for the server's architecture.
func buildCompressedSeedBinary(architecture string) (string, error) {
	localBinaryPath, err := filepath.Abs(filepath.Join("../tmp", "genix_seed_linux_"+architecture))
	if err != nil {
		return "", err
	}
	fmt.Printf("Compiling genix-seed for linux/%s...\n", architecture)
	buildCommand := exec.Command("go", "build", "-ldflags", "-s -w", "-o", localBinaryPath, "./seeds/cmd/genix-seed")
	buildCommand.Dir = "../backend"
	buildCommand.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+architecture)
	if architecture == "arm64" {
		buildCommand.Env = append(buildCommand.Env, "GOARM64="+arm64Generation)
	}
	buildCommand.Stdout, buildCommand.Stderr = os.Stdout, os.Stderr
	if err := buildCommand.Run(); err != nil {
		return "", fmt.Errorf("compiling: %w", err)
	}
	if err := compressBinary(localBinaryPath, localBinaryPath+".zst"); err != nil {
		return "", fmt.Errorf("compressing: %w", err)
	}
	return localBinaryPath + ".zst", nil
}

// readGenixServiceProperties reads the genix unit's User, WorkingDirectory and Environment, which
// point the seed at the same config.toml, database and file store as the server.
func readGenixServiceProperties(server seedServer) (map[string]string, error) {
	showCommand := buildSSHCommand(server.keyPath, server.target, "systemctl show genix -p User -p WorkingDirectory -p Environment")
	output, err := showCommand.Output()
	if err != nil {
		return nil, fmt.Errorf("reading the genix unit on %s: %w", server.target, err)
	}
	properties := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name, value, _ := strings.Cut(line, "=")
		properties[name] = value
	}
	return properties, nil
}

// followUntilDetached streams the run's journal until the run ends. Ctrl+C reaches the remote tail
// (ssh -t), not the run; if the run is still going, a menu offers to abort it or detach.
func followUntilDetached(server seedServer) error {
	// Keep our own process alive on Ctrl+C; ssh -t forwards it to the remote side.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	// The remote tail stops by itself once the unit is gone, so a finished run returns here.
	followScript := fmt.Sprintf("journalctl -u %[1]s -f -n 100 -o cat & tail=$!; "+
		"while systemctl is-active --quiet %[1]s; do sleep 2; done; sleep 1; kill $tail", seedUnitName)
	for {
		sshArguments := []string{"-t"}
		if server.keyPath != "" {
			sshArguments = append(sshArguments, "-i", server.keyPath)
		}
		followCommand := exec.Command("ssh", append(sshArguments, server.target, followScript)...)
		followCommand.Stdin, followCommand.Stdout, followCommand.Stderr = os.Stdin, os.Stdout, os.Stderr
		_ = followCommand.Run()

		if !isRemoteSeedActive(server) {
			fmt.Println("\nThe seed is no longer running.")
			return nil
		}
		fmt.Print("\nThe seed keeps running. [a] abort at the next checkpoint, [f] follow again, [Enter] detach: ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		switch strings.TrimSpace(strings.ToLower(answer)) {
		case "a":
			if err := abortRemoteSeed(server); err != nil {
				return err
			}
		case "f":
		default:
			fmt.Println("Detached. Later: seed_remote logs | status | abort")
			return nil
		}
	}
}

// abortRemoteSeed sends the run the SIGINT Ctrl+C would: it stops at its next checkpoint, and the
// same run command resumes it.
func abortRemoteSeed(server seedServer) error {
	fmt.Println("Asking the seed to stop at its next checkpoint...")
	return runRemoteCommand(server, fmt.Sprintf("systemctl kill --signal=SIGINT %s", seedUnitName))
}

func isRemoteSeedActive(server seedServer) bool {
	return buildSSHCommand(server.keyPath, server.target, "systemctl is-active --quiet "+seedUnitName).Run() == nil
}

func runRemoteCommand(server seedServer, remoteCommand string) error {
	command := buildSSHCommand(server.keyPath, server.target, remoteCommand)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}
