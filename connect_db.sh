#!/bin/bash

CONFIG_FILE="config.toml"

echo "--- ScyllaDB Connectivity Tool for Fedora ---"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: $CONFIG_FILE not found in the current folder."
    exit 1
fi

# tomllib es stdlib desde Python 3.11 y ya es requisito de scripts/.
read_config_value() {
    python3 -c "import tomllib,sys; print(tomllib.load(open('$CONFIG_FILE','rb'))['db']['$1'])"
}

SCYLLA_HOST=$(read_config_value host)
SCYLLA_PORT=$(read_config_value port)
SCYLLA_USER=$(read_config_value user)
SCYLLA_PASS=$(read_config_value password)

# Check if cqlsh is installed
if ! command -v cqlsh &> /dev/null; then
    echo "cqlsh not found. Installing via pip..."
    pip install --user cqlsh
    export PATH=$PATH:$HOME/.local/bin
fi

echo "Connecting to $SCYLLA_HOST on port $SCYLLA_PORT as $SCYLLA_USER..."

cqlsh "$SCYLLA_HOST" "$SCYLLA_PORT" \
    -u "$SCYLLA_USER" \
    -p "$SCYLLA_PASS"
