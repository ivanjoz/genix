# P2P Bridge: Signaling Lambda & Home Lab Server

This project implements a serverless signaling bridge for WebRTC connections between a browser and a local environment (Home Lab).

## Project Structure

- `signaling_lambda/`: Go source for the AWS Lambda function.
- `homelab_server/`: Go source for the local server that establishes the P2P link.
- `signal/`: Shared Go types for signaling messages.
- `config/`: Configuration package for reading credentials.json.
- `deploy/`: AWS CDK project for infrastructure deployment.
- `homelab_server/install/`: Systemd service installation package.

## Prerequisites

- **Go 1.21+**
- **Node.js & npm**
- **AWS CLI** configured with appropriate permissions.

## Configuration

All configuration is centralized in `credentials.json` in the project root. Create this file with your settings:

```json
{
  "aws_profile": "default",
  "app_name": "p2p-bridge",
  "signaling_app_name": "",
  "stack_name": "",
  "signaling_endpoint": "",
  "lambda_function_name": "",
  "aws_region": "us-east-1",
  "aws_account": ""
}
```

**Configuration Fields:**
- `aws_profile`: AWS profile name (as configured in ~/.aws/credentials). Case-insensitive. If not set, uses default AWS profile.
- `app_name`: **Required.** Base application name used to generate other names.
- `signaling_app_name`: **Optional.** Specific name for the signaling app. If empty, defaults to `app_name + "-signaling"`.
- `signaling_stack_name`: **Optional.** CDK stack name for CloudFormation. If empty, defaults to `app_name + "-signaling"`.
- `signaling_endpoint`: **Optional.** Signaling endpoint URL for home lab server (e.g., `wss://xxx.execute-api.region.amazonaws.com/prod`). Set after deployment.
- `lambda_function_name`: **Optional.** Actual Lambda function name from CDK output. Defaults to `app_name + "-signaling"` if not set.
- `aws_region`: AWS region (leave empty to use default from AWS profile).
- `aws_account`: AWS account ID (leave empty to use default from AWS profile). Not required if `aws_profile` is set.

**Derived Values:**
- `lambda_function_name`: Set from `lambda_function_name` in config, or defaults to `app_name + "-signaling"`

**Environment Variable Overrides:**
You can override configuration values using environment variables:
- `AWS_PROFILE` → overrides `aws_profile`
- `SIGNALING_ENDPOINT` → overrides `signaling_endpoint`
- `LAMBDA_FUNCTION_NAME` → overrides `lambda_function_name`
- `AWS_REGION` → overrides `aws_region`
- `AWS_ACCOUNT` → overrides `aws_account`

## 🪜 Step 1: Deploy the Infrastructure

**Quick Deployment:**

Use the convenience script that builds the Lambda and deploys in one command:

```bash
./deploy.sh
```

This script automatically:
1. Builds the Lambda binary for Linux (`./build-lambda.sh`)
2. Deploys infrastructure to AWS using CDK (`cdk deploy`)

**For Lambda code updates only:**

```bash
./build-lambda.sh
go run update-lambda.go --wait
```

**Manual Deployment:**

If you prefer to run steps manually:

1. Build the Lambda binary:
   ```bash
   ./build-lambda.sh
   ```

2. Deploy to AWS:
   ```bash
   cd deploy
   npx cdk deploy
   ```

3. **Note the Outputs:**
   - `WebSocketConnectURL`: The `wss://...` URL for the client and server to connect.
   - Copy the WebSocket URL and Lambda function name to your `credentials.json`:
     ```bash
     # Example values from CDK output:
     "signaling_endpoint": "wss://waogll39kd.execute-api.us-east-1.amazonaws.com/prod",
     "lambda_function_name": "genix-signaling-genixsignaling744441AF-MmLGyR5UQrPq"
     ```

## 💻 Step 2: Run the Home Lab Server

**Important:** Whenever you modify `signaling_lambda/main.go`, you must rebuild the binary before deploying. The `./deploy.sh` script handles this automatically.

The Home Lab server needs to connect to the WebSocket and have permission to update the Lambda's environment variables.

1. Navigate to the project root directory (where `credentials.json` is located).

2. The home lab server now automatically reads configuration from `credentials.json`. Make sure it includes:
   ```json
   "signaling_endpoint": "wss://your-api-id.execute-api.region.amazonaws.com/prod",
   "lambda_function_name": "genix-signaling-genixsignaling744441AF-MmLGyR5UQrPq"
   ```

   **Note:** After running `cdk deploy`, copy these values from the CDK output and add them to your `credentials.json`.

3. Run the server:
   ```bash
   go run homelab_server/main.go
   ```

### Option A: Run Manually

The server will:
- Load configuration from `credentials.json` (WebSocket URL, Lambda name, AWS profile)
- Connect to the WebSocket.
- Receive its `connectionID`.
- Update the Lambda's `LAPTOP_ID` environment variable using the configured Lambda function name.
- Wait for incoming WebRTC "offer" signals.

### Option B: Install as a Systemd Service

For production use, you can install the server as a systemd service that runs automatically on boot:

```bash
cd homelab_server
sudo go run main.go --install
```

The `--install` command will:
- Build and compile the server binary
- Install the binary to `/usr/local/bin/homelab-p2p-bridge`
- Create a systemd service file at `/etc/systemd/system/homelab-p2p-bridge.service`
- Enable and start the service

**Service Features:**
- Automatically start on system boot
- Restart automatically if it crashes (5 second delay)
- Log all output to systemd journal
- Keep the computer connected to the AWS Lambda WebSocket

**Manage the Service:**

```bash
# Check service status
sudo systemctl status homelab-p2p-bridge

# View logs
sudo journalctl -u homelab-p2p-bridge -f

# Stop the service
sudo systemctl stop homelab-p2p-bridge

# Restart the service
sudo systemctl restart homelab-p2p-bridge
```

**Uninstall the Service:**

To remove the service and binary:
```bash
cd homelab_server
sudo go run main.go --uninstall
```

The `--uninstall` command will:
- Stop the service if running
- Disable the service
- Remove the systemd service file
- Reload systemd daemon
- Remove the installed binary
- Clean up systemd state

## 🔧 Troubleshooting

**Service fails to start:**
- Check that `credentials.json` exists in the parent directory
- Verify `SIGNALING_ENDPOINT` is set correctly
- Ensure AWS profile has permissions to update Lambda environment variables
- Check systemd logs: `sudo journalctl -u homelab-p2p-bridge -n 50`

**Service not connecting to WebSocket:**
- Verify WebSocket URL is correct and accessible
- Check AWS Lambda function exists and is deployed
- Verify IAM permissions for the AWS profile
- After updating Lambda code, run `go run update-lambda.go --wait` to apply changes

**Permission denied when installing:**
- The `--install` command must be run with `sudo` privileges
- Ensure you have write access to `/usr/local/bin` and `/etc/systemd/system/`

**Build fails:**
- Ensure Go 1.21+ is installed
- Check that `go.mod` exists in the project root
- Run `go mod tidy` if dependencies are out of sync

## 📝 Testing

Run the test script to verify configuration and server setup:

```bash
./test-server.sh
```

This script will:
- Verify Go installation
- Check `credentials.json` exists
- Validate configuration loading
- Test server compilation
- Verify install/uninstall flags

## 🚀 Development Workflow

**When modifying Lambda code:**
```bash
# Edit code
vim signaling_lambda/main.go
# Build binary
./build-lambda.sh
# Update Lambda code (without full redeploy)
go run update-lambda.go --wait
```

**When modifying server code:**
```bash
# Edit code
vim homelab_server/main.go
# Test (no deployment needed)
./test-server.sh
# Run
go run homelab_server/main.go
```

**When modifying infrastructure:**
```bash
# Edit code
vim deploy/deploy.go
# Deploy infrastructure (Lambda code not affected)
cd deploy && npx cdk deploy
```

## 📚 Additional Documentation

- [DEPLOYMENT.md](./DEPLOYMENT.md) - Detailed deployment guide
- [IMPLEMENTATION.md](./IMPLEMENTATION.md) - Implementation details and architecture
- [homelab_server/install/README.md](./homelab_server/install/README.md) - Systemd service installation documentation
- [TROUBLESHOOTING/connection-issues.md](./TROUBLESHOOTING/connection-issues.md) - Comprehensive connection issues troubleshooting guide

## 💡 Common Issues & Fixes

### WebSocket "Internal server error" (Messages not reaching Lambda)

If the WebSocket connection established successfully (`$connect` works) but subsequent messages fail with a generic `{"message": "Internal server error"}`, it's likely a permission or routing issue at the API Gateway level.

**Cause:**
- The Lambda function lacks permission to be invoked by API Gateway for routes other than `$connect`.
- The `$request.body.action` route selection expression might not match any route, and there is no `$default` route.

**Solution:**
1. **CDK Update (Recommended):** The deployment script has been updated to explicitly grant `lambda:InvokeFunction` for all routes and to include a `$default` route. Re-deploying with `cdk deploy` will apply these fixes.
2. **Manual Fix (Immediate):** Grant permission to API Gateway to invoke the Lambda for all routes:
   ```bash
   aws lambda add-permission \
     --function-name <your-lambda-function-name> \
     --statement-id "AllowAllRoutes" \
     --action "lambda:InvokeFunction" \
     --principal "apigateway.amazonaws.com" \
     --source-arn "arn:aws:execute-api:<region>:<account-id>:<api-id>/*"
   ```

## 🆘 Connection Issues

If you're experiencing connection problems such as:
- "The target (homelab_server) is not currently connected" error
- WebSocket connection failures
- WebRTC negotiation timeouts
- "Unknown message format" warnings

Please refer to the [Connection Issues Troubleshooting Guide](./TROUBLESHOOTING/connection-issues.md) for detailed diagnostic steps and solutions.
