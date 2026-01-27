# Deployment Guide - P2P Bridge

This guide walks you through deploying the P2P Bridge WebRTC signaling infrastructure to AWS.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Quick Deployment](#quick-deployment)
- [Manual Deployment](#manual-deployment)
- [Running the Home Lab Server](#running-the-home-lab-server)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before deploying, ensure you have:

- **Go 1.21+** - For building Lambda and server
- **Node.js & npm** - For AWS CDK CLI
- **AWS CLI** - Configured with appropriate permissions
- **AWS Credentials** - Set up in `~/.aws/credentials` with a profile that has:
  - `lambda:UpdateFunctionConfiguration` (for server)
  - CloudFormation permissions (for deployment)
  - IAM permissions to create roles

---

## Configuration

The P2P Bridge uses a centralized configuration file located at:

```
/home/ivanjoz/projects/genix/credentials.json
```

### Configuration Fields

```json
{
  "AWS_PROFILE": "ivanjoz",
  "APP_NAME": "genix",
  "SIGNALING_APP_NAME": "",
  "SIGNALING_STACK_NAME": "",
  "AWS_REGION": "us-east-1",
  "AWS_ACCOUNT": ""
}
```

| Field | Required | Description | Default |
|-------|----------|-------------|---------|
| `AWS_PROFILE` | No | AWS profile from `~/.aws/credentials` | Uses default profile |
| `APP_NAME` | **Yes** | Base application name | - |
| `SIGNALING_APP_NAME` | No | Signaling app name | `APP_NAME + "-signaling"` |
| `SIGNALING_STACK_NAME` | No | CDK stack name | `APP_NAME + "-signaling"` |
| `AWS_REGION` | No | AWS region | From AWS profile |
| `AWS_ACCOUNT` | No | AWS account ID | From AWS profile |

### Environment Variable Overrides

You can override configuration with environment variables:

```bash
export AWS_PROFILE="my-profile"
export AWS_REGION="us-west-2"
export AWS_ACCOUNT="123456789012"
```

---

## Quick Deployment

The fastest way to deploy is using the automated script:

```bash
cd /home/ivanjoz/projects/genix/p2p
./deploy.sh
```

This script will:

1. 📦 Build the Lambda binary for Linux
2. 🚀 Deploy to AWS using CDK
3. ✅ Display the WebSocket URL

### Deploy Script Features

- **Automatic Lambda compilation**: Builds `signaling_lambda/bootstrap` for Linux
- **Error checking**: Verifies binary creation before deployment
- **Pass-through arguments**: Accepts all CDK options:
  ```bash
  ./deploy.sh --require-approval never
  ./deploy.sh --profile production
  ```

---

## Manual Deployment

If you need more control, deploy in steps:

### Step 1: Build the Lambda Binary

```bash
cd /home/ivanjoz/projects/genix/p2p
./build-lambda.sh
```

This compiles the Lambda function as a static Linux binary at:
```
signaling_lambda/bootstrap
```

**Build Options:**
- Target OS: Linux (`GOOS=linux`)
- Target Arch: AMD64 (`GOARCH=amd64`)
- CGO: Disabled (`CGO_ENABLED=0`)

### Step 2: Deploy to AWS

```bash
cd deploy
npx cdk deploy
```

### Common CDK Commands

```bash
# View what will be deployed
npx cdk diff

# View deployed stacks
npx cdk list

# Synthesize CloudFormation templates (no deployment)
npx cdk synth

# Destroy deployed infrastructure
npx cdk destroy
```

---

## Deployment Outputs

After successful deployment, CDK will display outputs including:

```
Outputs:
genix-signaling.WebSocketConnectURL = wss://xxx.execute-api.us-east-1.amazonaws.com/prod
```

**Save this URL** - you'll need it for:
- The home lab server
- Your frontend application

---

## Running the Home Lab Server

The home lab server connects to the WebSocket API and receives WebRTC offers.

### Setup

1. **Set the WebSocket URL** (from CDK outputs):
   ```bash
   export WS_URL="wss://your-api-id.execute-api.region.amazonaws.com/prod"
   ```

2. **Run the server**:
   ```bash
   cd /home/ivanjoz/projects/genix/p2p
   go run homelab_server/main.go
   ```

### What the Server Does

1. ✅ Connects to WebSocket API
2. ✅ Receives connection ID
3. ✅ Updates Lambda's `LAPTOP_ID` environment variable
4. ✅ Listens for WebRTC "offer" signals
5. ✅ Responds with "answer" to establish P2P connection

### Server Configuration

The server automatically loads configuration from `credentials.json`:
- AWS profile for authentication
- Lambda function name for updates

---

## Architecture Overview

```
┌─────────────┐         WebSocket          ┌──────────────┐
│   Browser   │ ◄──────────────────────► │   Lambda     │
│  (Client)   │   Signaling Messages     │  (Relay)     │
└─────────────┘                        └──────┬───────┘
                                              │
                                              ▼
                                     ┌─────────────────┐
                                     │   Home Lab     │
                                     │   Server       │
                                     │  (Go Binary)   │
                                     └─────────────────┘
```

### Connection Flow

1. **Home Lab Server**: Connects to WebSocket, registers connection ID
2. **Browser**: Connects to WebSocket, sends "offer" to "laptop"
3. **Lambda**: Forwards "offer" to registered connection ID
4. **Home Lab Server**: Sends "answer" back via Lambda
5. **P2P Connection**: Browser and server communicate directly (WebRTC)

---

## Troubleshooting

### "bootstrap binary not found"

**Problem**: Lambda binary wasn't compiled
**Solution**:
```bash
cd /home/ivanjoz/projects/genix/p2p
./build-lambda.sh
```

### "credentials.json not found"

**Problem**: Config file not in expected location
**Solution**: Ensure `credentials.json` exists at `/home/ivanjoz/projects/genix/credentials.json`

### "APP_NAME is required"

**Problem**: Missing required field in config
**Solution**: Add to `credentials.json`:
```json
{
  "APP_NAME": "your-app-name"
}
```

### AWS Profile Not Working

**Problem**: CDK can't use specified profile
**Solution**: Verify profile exists:
```bash
aws configure list-profiles
aws --profile your-profile s3 ls
```

### Lambda Times Out

**Problem**: Lambda connection timeout (WebSocket disconnects)
**Solution**: The issue is likely in your WebRTC negotiation. Check:
- STUN server configuration
- NAT traversal settings
- Firewall rules

### "spawnSync docker ENOENT"

**Problem**: CDK trying to use Docker for bundling
**Solution**: This shouldn't happen with the current setup. Ensure:
- Lambda binary is pre-compiled
- Using the updated `deploy.sh` script
- Docker is not required for deployment

### Server Can't Update Lambda

**Problem**: Permission denied updating Lambda environment
**Solution**: Verify AWS profile has these permissions:
```
lambda:UpdateFunctionConfiguration
iam:PassRole
```

---

## Development Workflow

### Making Changes

1. **Modify Lambda code** (`signaling_lambda/main.go`):
   ```bash
   # Make changes
   vim signaling_lambda/main.go
   
   # Rebuild and deploy
   ./deploy.sh
   ```

2. **Modify Server code** (`homelab_server/main.go`):
   ```bash
   # Make changes
   vim homelab_server/main.go
   
   # Just rebuild, no need to deploy
   go run homelab_server/main.go
   ```

3. **Modify Infrastructure** (`deploy/deploy.go`):
   ```bash
   # Make changes
   vim deploy/deploy.go
   
   # Deploy (Lambda rebuild not needed for infra changes)
   cd deploy && npx cdk deploy
   ```

---

## Cost Considerations

**Free Tier Usage:**
- WebSocket API: $1.00 per million messages (first million free)
- Lambda: 1M free requests/month (then $0.20 per million)
- Data Transfer: 100GB/month free

**Estimated Monthly Cost** (light usage):
- $0-5 depending on usage

---

## Security Best Practices

1. **Never commit** `credentials.json` to version control
2. **Use specific IAM roles** with least privilege
3. **Enable WebSocket authentication** in production
4. **Use WSS** (WebSocket Secure) always
5. **Rotate credentials** regularly
6. **Monitor CloudWatch logs** for suspicious activity

---

## Additional Resources

- [AWS CDK Documentation](https://docs.aws.amazon.com/cdk/)
- [WebSocket API Guide](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-websocket-api.html)
- [Pion WebRTC Library](https://github.com/pion/webrtc)
- [Project README](./README.md)