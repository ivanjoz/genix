#!/bin/bash
set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================="
echo "  P2P Bridge Deployment Script"
echo "========================================="
echo ""

# Step 1: Build Lambda binary
echo "📦 Step 1: Building Lambda binary..."
if [ ! -f "build-lambda.sh" ]; then
    echo "❌ Error: build-lambda.sh not found in current directory"
    exit 1
fi

chmod +x build-lambda.sh
./build-lambda.sh

# Verify binary was created
if [ ! -f "signaling_lambda/bootstrap" ]; then
    echo "❌ Error: Lambda binary not found at signaling_lambda/bootstrap"
    exit 1
fi

echo "✅ Lambda binary built successfully"
echo ""

# Step 2: Deploy to AWS
echo "🚀 Step 2: Deploying to AWS..."
cd deploy

# Check if cdk.json exists
if [ ! -f "cdk.json" ]; then
    echo "❌ Error: cdk.json not found in deploy directory"
    exit 1
fi

# Run CDK deploy
npx cdk deploy "$@"

echo ""
echo "========================================="
echo "  Deployment Complete!"
echo "========================================="
echo ""
echo "📝 Don't forget to note the WebSocket URL from the CDK outputs!"
echo ""