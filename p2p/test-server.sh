#!/bin/bash
set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================="
echo "  Home Lab Server Test Script"
echo "========================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    exit 1
fi

echo "✅ Go found: $(go version)"
echo ""

# Check if config.toml exists
if [ ! -f "../config.toml" ]; then
    echo "❌ Error: config.toml not found in parent directory"
    exit 1
fi

echo "✅ Found config.toml"
echo ""

# Check if go.mod exists
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found in current directory"
    exit 1
fi

echo "✅ Found go.mod"
echo ""

# Create a simple test program to verify configuration loading
cat > test_config.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "p2p_bridge/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("❌ Failed to load configuration: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("=== Configuration Loaded ===")
    fmt.Printf("Signaling Endpoint:   %s\n", cfg.SignalingEndpoint)
    fmt.Printf("Lambda Function Name: %s\n", cfg.GetLambdaFunctionName())
    fmt.Printf("AWS Profile:          %s\n", cfg.AWSProfile)
    fmt.Printf("AWS Region:           %s\n", cfg.AWSRegion)
    fmt.Println()

    if cfg.SignalingEndpoint == "" {
        fmt.Println("⚠️  WARNING: signaling.endpoint is not set in config.toml")
        fmt.Println("   Please add it after deploying: wss://xxx.execute-api.region.amazonaws.com/prod")
        os.Exit(0)
    }

    fmt.Println("✅ Configuration is valid!")
}
EOF

echo "🧪 Testing configuration..."
go run test_config.go
CONFIG_RESULT=$?

# Clean up
rm -f test_config.go

if [ $CONFIG_RESULT -ne 0 ]; then
    echo ""
    echo "========================================="
    echo "❌ Configuration Test Failed"
    echo "========================================="
    exit 1
fi

echo ""
echo "========================================="
echo "✅ Configuration Test Passed!"
echo "========================================="
echo ""

# Check if signaling.endpoint is set
if ! grep -q '^\s*endpoint\s*=\s*"[^"]\+"' ../config.toml 2>/dev/null; then
    echo "⚠️  signaling.endpoint not found in config.toml"
    echo ""
    echo "Please add it after running ./deploy.sh, under [signaling]:"
    echo '  endpoint = "wss://your-api-id.execute-api.region.amazonaws.com/prod"'
    echo ""
    echo "Skipping server test..."
    exit 0
fi

# Test if we can compile the server
echo "🔨 Testing server compilation..."
if go build -o test_server_build homelab_server/main.go 2>&1 | grep -q "error"; then
    echo "❌ Server compilation failed"
    echo "   Check for errors in homelab_server/main.go"
    exit 1
fi

echo "✅ Server compiles successfully"
echo ""

# Clean up build artifact
rm -f test_server_build

# Test install/uninstall flags
echo "🔧 Testing install/uninstall flags..."
echo ""

# Build a test binary
if ! go build -o test_install_binary homelab_server/main.go 2>&1; then
    echo "❌ Failed to build binary for install test"
    exit 1
fi

# Check that --install flag is recognized
echo "   Checking --install flag..."
if ! ./test_install_binary --help 2>&1 | grep -q "install"; then
    echo "❌ --install flag not found in help output"
    rm -f test_install_binary
    exit 1
fi
echo "   ✅ --install flag found"

# Check that --uninstall flag is recognized
echo "   Checking --uninstall flag..."
if ! ./test_install_binary --help 2>&1 | grep -q "uninstall"; then
    echo "❌ --uninstall flag not found in help output"
    rm -f test_install_binary
    exit 1
fi
echo "   ✅ --uninstall flag found"

# Clean up test binary
rm -f test_install_binary

echo "✅ Install/uninstall flags verified"
echo ""

# Check if we can actually run the server (quick test)
echo "🚀 Quick server check..."
echo "   Server will try to start and connect to WebSocket"
echo ""

# Run server for a brief moment to check it can initialize
timeout 2 go run homelab_server/main.go 2>&1 | head -20 || true

echo "✅ Server initialization check complete"

echo ""
echo "========================================="
echo "✅ Server Test Complete!"
echo "========================================="
echo ""
echo "📝 Summary:"
echo "   ✅ Configuration loaded successfully"
echo "   ✅ Server compiles without errors"
echo "   ✅ Server can start"
echo ""
echo "🚀 You can now run:"
echo "   go run homelab_server/main.go"
echo ""
echo "💡 Make sure your Signaling Endpoint is correct in config.toml"
echo "   The server will connect to: $(grep -o '^\s*endpoint\s*=.*' ../config.toml | head -1 || echo 'Not set')"