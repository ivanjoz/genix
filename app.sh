#!/bin/bash

# This script acts as a router for other scripts.

# Check for the first argument to decide which script to run.
case "$1" in
  "check_tables")
    # For "check_tables", execute the Go script in the "scripts" folder.
    echo "Executing check_tables script..."
    (cd scripts && go run . check_tables)
    ;;
  "new_table")
    # For "new_table", execute the Go script and pass along all following arguments.
    echo "Executing new_table script..."
    (cd scripts && go run . new_table "${@:2}")
    ;;
  *)
    # If the command is not recognized, show an error and usage instructions.
    echo "Unknown command: $1"
    echo "Usage: $0 {check_tables|new_table}"
    exit 1
    ;;
esac
