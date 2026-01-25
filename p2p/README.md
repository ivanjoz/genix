# P2P Bridge: Signaling Lambda & Home Lab Server

This project implements a serverless signaling bridge for WebRTC connections between a browser and a local environment (Home Lab).

## Project Structure

- `signaling_lambda/`: Go source for the AWS Lambda function.
- `homelab_server/`: Go source for the local server that establishes the P2P link.
- `signal/`: Shared Go types for signaling messages.
- `deploy/`: AWS CDK project for infrastructure deployment.

## Prerequisites

- **Go 1.21+**
- **Node.js & npm**
- **AWS CLI** configured with appropriate permissions.
- **Docker** (required by CDK for building the Go Lambda in a Linux environment).

## 🪜 Step 1: Deploy the Infrastructure

1. Navigate to the deployment directory:
   ```bash
   cd deploy
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Deploy to AWS:
   ```bash
   npx cdk deploy
   ```

4. **Note the Outputs:**
   - `WebSocketConnectURL`: The `wss://...` URL for the client and server to connect.
   - `SignalingStack.SignalingRelay...`: The name of the Lambda function (needed for the server to update environment variables).

## 💻 Step 2: Run the Home Lab Server

The Home Lab server needs to connect to the WebSocket and have permission to update the Lambda's environment variables.

1. Navigate back to the root directory:
   ```bash
   cd ..
   ```

2. Set the environment variables (replace with your CDK outputs):
   ```bash
   export WS_URL="wss://your-api-id.execute-api.region.amazonaws.com/prod"
   export LAMBDA_NAME="SignalingStack-SignalingRelayXXXXX"
   ```

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
