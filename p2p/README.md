# P2P Bridge: Signaling Lambda & Home Lab Server

This project implements a serverless signaling bridge for WebRTC connections between a browser and a local environment (Home Lab).

## Project Structure

- `signaling_lambda/`: Go source for the AWS Lambda function.
- `homelab_server/`: Go source for the local server that establishes the P2P link.
- `signal/`: Shared Go types for signaling messages.
- `config/`: Configuration package for reading credentials.json.
- `deploy/`: AWS CDK project for infrastructure deployment.

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
  "aws_region": "us-east-1",
  "aws_account": ""
}
```

**Configuration Fields:**
- `aws_profile`: AWS profile name (as configured in ~/.aws/credentials). Case-insensitive. If not set, uses default AWS profile.
- `app_name`: **Required.** Base application name used to generate other names.
- `signaling_app_name`: **Optional.** Specific name for the signaling app. If empty, defaults to `app_name + "-signaling"`.
- `signaling_stack_name`: **Optional.** CDK stack name for CloudFormation. If empty, defaults to `app_name + "-signaling"`.
- `aws_region`: AWS region (leave empty to use default from AWS profile).
- `aws_account`: AWS account ID (leave empty to use default from AWS profile). Not required if `aws_profile` is set.

**Derived Values:**
- `lambda_function_name`: Automatically set to `app_name + "-signaling"`

**Environment Variable Overrides:**
You can override configuration values using environment variables:
- `AWS_PROFILE` → overrides `aws_profile`
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
2. Deploys to AWS using CDK (`cdk deploy`)

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

**Note the Outputs:**
- `WebSocketConnectURL`: The `wss://...` URL for the client and server to connect.

## 💻 Step 2: Run the Home Lab Server

**Important:** Whenever you modify `signaling_lambda/main.go`, you must rebuild the binary before deploying. The `./deploy.sh` script handles this automatically.

The Home Lab server needs to connect to the WebSocket and have permission to update the Lambda's environment variables.

1. Navigate to the project root directory (where `credentials.json` is located).

2. Set the WebSocket URL environment variable (from your CDK output):
   ```bash
   export WS_URL="wss://your-api-id.execute-api.region.amazonaws.com/prod"
   ```

   **Note:** The lambda function name and AWS profile are now automatically read from `credentials.json`.

3. Run the server:
   ```bash
   go run homelab_server/main.go
   ```

The server will:
- Connect to the WebSocket.
- Receive its `connectionID`.
- Update the Lambda's `LAPTOP_ID` environment variable.
- Wait for incoming WebRTC "offer" signals.

## 🌐 Step 3: Frontend Connection

Your frontend (browser) should:
1. Connect to the same `WS_URL`.
2. Send an "offer" signal to the `laptop` target:
   ```json
   {
     "action": "sendSignal",
     "to": "laptop",
     "data": { "type": "offer", "sdp": "..." }
   }
   ```
3. The Lambda will forward this to the current `LAPTOP_ID`.
4. The Home Lab server will respond with an "answer" via the Lambda back to the browser.
