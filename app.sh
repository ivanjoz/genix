#!/bin/bash

# This script acts as a router for other scripts.

# Check for the first argument to decide which script to run.
case "$1" in
  "check_tables")
    # For "check_tables", execute the Go script in the "scripts" folder.
    echo "Executing check_tables script..."
    (cd scripts && go run . check_tables)
    ;;
  "create")
    # For "create", create a new database table structure.
    echo "Executing create command..."
    (cd scripts && go run table/create_edit_table.go create "${@:2}")
    ;;
  "edit")
    # For "edit", add a new column to an existing table.
    echo "Executing edit command..."
    (cd scripts && go run table/create_edit_table.go edit "${@:2}")
    ;;
  *)
    # If the command is not recognized, show an error and usage instructions.
    echo "Unknown command: $1"
    echo "Usage: $0 {check_tables|create|edit}"
    exit 1
    ;;
esac
