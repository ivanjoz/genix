#!/bin/bash

# --- File Names ---
CONFIG_FILE="credentials.json"

echo "--- ScyllaDB Connectivity Tool for Fedora ---"

# 1. Check for jq (needed to read JSON)
if ! command -v jq &> /dev/null; then
    echo "jq not found. Installing..."
    sudo dnf install -y jq
fi

# 2. Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: $CONFIG_FILE not found in the current folder."
    exit 1
fi

# 3. Extract credentials using jq
SCYLLA_HOST=$(jq -r '.DB_HOST' "$CONFIG_FILE")
SCYLLA_PORT=$(jq -r '.DB_PORT' "$CONFIG_FILE")
SCYLLA_USER=$(jq -r '.DB_USER' "$CONFIG_FILE")
SCYLLA_PASS=$(jq -r '.DB_PASSWORD' "$CONFIG_FILE")

# 4. Check if cqlsh is installed
if ! command -v cqlsh &> /dev/null; then
    echo "cqlsh not found. Installing via pip..."
    pip install --user cqlsh
    export PATH=$PATH:$HOME/.local/bin
fi

echo "Connecting to $SCYLLA_HOST on port $SCYLLA_PORT as $SCYLLA_USER..."

# 5. Launch cqlsh
cqlsh "$SCYLLA_HOST" "$SCYLLA_PORT" \
    -u "$SCYLLA_USER" \
    -p "$SCYLLA_PASS"
