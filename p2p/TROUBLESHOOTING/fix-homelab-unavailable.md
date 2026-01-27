# Fix: Homelab Server Unavailability Error Handling

## Problem

When the homelab_server is not running or not connected to the signaling Lambda, the frontend receives a generic "Internal server error" with no context, causing confusion and poor user experience.

## Root Cause

1. Frontend sends WebRTC offer signal with target `"laptop"`
2. Lambda resolves `"laptop"` to `LAPTOP_ID` environment variable
3. If homelab_server isn't connected, `LAPTOP_ID` is empty or connection doesn't exist
4. Lambda's `PostToConnection` call fails with `GoneException`
5. Lambda catches error and returns generic 500 "Internal server error"
6. Frontend doesn't recognize this error format

## Solution

### Lambda Changes (`signaling_lambda/main.go`)

Detect `GoneException` when target connection doesn't exist and send structured error response:

```go
// Check if the error is due to connection not found (GoneException)
errStr := err.Error()
if strings.Contains(errStr, "GoneException") || strings.Contains(errStr, "Connection does not exist") {
    logError(fmt.Sprintf("Target connection %s does not exist (homelab_server not connected)", targetID), err)
    
    // Send an error message back to the sender
    errorMsg := map[string]interface{}{
        "error":   "backend_unreachable",
        "message": "The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.",
        "target":  targetID,
    }
    errorPayload, _ := json.Marshal(errorMsg)
    
    _, sendErr := apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
        ConnectionId: aws.String(connectionID),
        Data:         errorPayload,
    })
    if sendErr != nil {
        logError("Failed to send error message back to sender", sendErr)
    } else {
        logInfo("Sent error notification to sender")
    }
    
    return events.APIGatewayProxyResponse{StatusCode: 404}, fmt.Errorf("target connection not found")
}
```

Returns:
- HTTP 404 (Not Found) instead of 500
- Structured error message sent back via WebSocket
- Clear, actionable error message

### Frontend Changes (`frontend/pkg-core/lib/wss-webrtc.ts`)

Parse and handle error messages from Lambda:

```typescript
// Check for error messages from the signaling server
if (message.error === 'backend_unreachable') {
  console.warn('[WSSWebRTC] Target not connected:', message.message);
  this.emit('error', new Error(message.message || 'The target is not currently connected'));
  return;
}
```

Recognizes structured error and emits proper error event with descriptive message.

## Error Response Format

When homelab_server is not connected, Lambda sends:

```json
{
  "error": "backend_unreachable",
  "message": "The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.",
  "target": "<connection-id>"
}
```

## Deployment

### 1. Rebuild Lambda Binary

```bash
cd /path/to/genix/p2p
./build-lambda.sh
```

### 2. Update Lambda Code

```bash
go run update-lambda.go --wait
```

This uses `update-lambda.go` to upload the new binary without redeploying the entire stack.

### 3. Verify

Check CloudWatch logs for:
```
[ERROR] Target connection <id> does not exist (homelab_server not connected)
[INFO] Sent error notification to sender
```

## Application Usage

Handle the error in your application:

```typescript
bridge.on('error', (err) => {
  if (err.message.includes('not currently connected')) {
    // Show user-friendly notification
    alert('Homelab server is not running. Please start it and try again.');
  }
});
```

## Files Modified

1. `genix/p2p/signaling_lambda/main.go` - Connection existence check and error response
2. `genix/frontend/pkg-core/lib/wss-webrtc.ts` - Error message parsing and handling
3. `genix/frontend/pkg-core/lib/wss-webrtc.README.md` - Updated documentation

## Benefits

- Users receive clear, actionable error messages
- Proper HTTP status codes (404 instead of 500)
- Better debugging with structured error responses
- Maintains WebSocket connection for subsequent attempts
