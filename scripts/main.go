package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a script to run.")
		os.Exit(1)
	}

	script := os.Args[1]

	switch script {
	case "check_tables":
		CheckTables()

	default:
		fmt.Printf("Unknown script: %s\n", script)
		os.Exit(1)
	}
}
