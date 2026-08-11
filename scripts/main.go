package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a script to run.")
		os.Exit(1)
	}

	script := os.Args[1]

	switch script {
	case "check_tables":
		runSubpackage("./validation")

	case "sync_struct_interfaces":
		runSubpackage("./generators")

	case "generate_controllers":
		runSubpackage("./controllers")

	case "generate_menu_descriptions":
		GenerateMenuDescriptions()

	case "index_documentation":
		runBackendSubpackage("./agent/cmd/documentation-index", os.Args[2:]...)

	case "search_documentation":
		runBackendSubpackage("./agent/cmd/documentation-search", os.Args[2:]...)

	case "classify_agent_request":
		runBackendSubpackage("./agent/cmd/classifier-eval", os.Args[2:]...)

	case "follow_cloudwatch_logs":
		FollowCloudWatchLogs()

	case "deploy_vps":
		DeployVPS()

	case "deploy":
		runSubpackage("./deployer", os.Args[2:]...)

	default:
		fmt.Printf("Unknown script: %s\n", script)
		os.Exit(1)
	}
}

// Backend agent commands use backend/go.mod; running them from scripts would resolve the
// separate scripts module even though both modules intentionally use the short name app.
func runBackendSubpackage(pkg string, args ...string) {
	cmd := exec.Command("go", append([]string{"run", pkg}, args...)...)
	cmd.Dir = "../backend"
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running backend command %s: %v\n", pkg, err)
		os.Exit(1)
	}
}

// Stdin va conectado porque algunos subpaquetes son interactivos (el TUI de ./deployer).
func runSubpackage(pkg string, args ...string) {
	cmd := exec.Command("go", append([]string{"run", pkg}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s: %v\n", pkg, err)
		os.Exit(1)
	}
}
