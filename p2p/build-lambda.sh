#!/bin/bash
set -e

echo "Building Lambda binary..."

# Navigate to signaling_lambda directory
cd "$(dirname "$0")/signaling_lambda"

# Build for Linux with CGO disabled
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go

echo "Lambda binary built successfully: signaling_lambda/bootstrap"

# Navigate back to original directory
cd ..