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

	case "deploy_vps":
		DeployVPS()

	default:
		fmt.Printf("Unknown script: %s\n", script)
		os.Exit(1)
	}
}

func runSubpackage(pkg string) {
	cmd := exec.Command("go", "run", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running %s: %v\n", pkg, err)
		os.Exit(1)
	}
}
