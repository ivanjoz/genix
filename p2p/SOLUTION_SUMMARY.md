# Homelab Server Unavailability Fix - Summary

## Problem

When the frontend client attempts to establish a WebRTC connection with the homelab_server, but the homelab_server is not running or not connected to the signaling Lambda, the system returns a generic "Internal server error" with no context. This results in:
- Frontend logs: "Unknown message format: {message: 'Internal server error', ...}"
- Users have no actionable information about what's wrong
- Poor debugging experience

## Root Cause

1. Frontend sends WebRTC offer signal with target `"laptop"`
2. Lambda resolves `"laptop"` to `LAPTOP_ID` environment variable
3. If homelab_server isn't connected, `LAPTOP_ID` is empty or connection doesn't exist
4. Lambda verifies LAPTOP_ID connection is alive by sending a ping
5. If LAPTOP_ID is stale (connection doesn't exist), Lambda detects `GoneException`
6. Lambda sends structured error message back via WebSocket

## Solution

### Lambda Changes (`signaling_lambda/main.go`)

Detect `GoneException` when target connection doesn't exist and send structured error response:

```go
// Verify the LAPTOP_ID connection is still alive before proceeding
if targetID != "" {
    logInfo("Verifying LAPTOP_ID connection is alive...")
    // Try to send a minimal ping to verify connection exists
    pingMsg := map[string]interface{}{
        "action": "ping",
        "from":   connectionID,
    }
    pingPayload, _ := json.Marshal(pingMsg)
    _, pingErr := apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
        ConnectionId: aws.String(targetID),
        Data:         pingPayload,
    })
    if pingErr != nil {
        errStr := pingErr.Error()
        if strings.Contains(errStr, "GoneException") || strings.Contains(errStr, "Connection does not exist") {
            logError("LAPTOP_ID connection is stale (homelab_server not connected)", pingErr)
            // Set targetID to empty so it will trigger the error handling below
            targetID = ""
            logDebug("Marking LAPTOP_ID as unavailable")
        }
    }
}

// Then send the error if target is not available
if targetID == "" {
    errorMsg := map[string]interface{}{
        "error":   "backend_unreachable",
        "message": "The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.",
        "target":  targetID,
    }
    errorPayload, _ := json.Marshal(errorMsg)
    
    apigw.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
        ConnectionId: aws.String(connectionID),
        Data:         errorPayload,
    })
    
    return events.APIGatewayProxyResponse{StatusCode: 404}, fmt.Errorf("target connection not found")
}
```

**Result:**
- Verifies LAPTOP_ID connection is alive before attempting to send signals
- Detects stale connections even if LAPTOP_ID environment variable contains old data
- Returns HTTP 404 (Not Found) instead of 500
- Sends structured error message back via WebSocket
- Provides clear, actionable error message

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

**Result:**
- Recognizes structured error messages
- Emits proper error event with descriptive message
- Prevents "Unknown message format" warnings

## Error Response Format

```json
{
  "error": "backend_unreachable",
  "message": "The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.",
  "target": "<connection-id>"
}
```

## Application Usage

Handle the error in your application:

```typescript
bridge.on('error', (err) => {
  if (err.message.includes('not currently connected')) {
    alert('Homelab server is not running. Please start it and try again.');
  }
});
```

## Deployment

After modifying code:

```bash
# Build Lambda binary
./build-lambda.sh

# Update Lambda code (no full redeploy needed)
go run update-lambda.go --wait
```

## Files Modified

1. `genix/p2p/signaling_lambda/main.go` - Connection existence check and error response
2. `genix/frontend/pkg-core/lib/wss-webrtc.ts` - Error message parsing and handling
3. `genix/frontend/pkg-core/lib/wss-webrtc.README.md` - Updated documentation
4. `genix/p2p/README.md` - Added troubleshooting section and update instructions

## Documentation

- `genix/p2p/TROUBLESHOOTING/README.md` - Quick reference for common issues
- `genix/p2p/TROUBLESHOOTING/connection-issues.md` - Detailed troubleshooting procedures
- `genix/p2p/TROUBLESHOOTING/fix-homelab-unavailable.md` - Complete fix documentation

## Testing

1. **Without homelab_server:** Should receive clear error message about target not connected
   - Even if LAPTOP_ID environment variable contains stale data
   - Lambda detects connection is stale and sends error

2. **With homelab_server running:** Should establish successful P2P connection

3. **Stale LAPTOP_ID scenario:**
   - LAPTOP_ID contains old connection ID (e.g., "init")
   - Homelab_server is not running
   - Lambda verifies connection is alive via ping
   - Detects GoneException and sends error to frontend

4. **Verify Lambda logs:** Check CloudWatch for proper error messages when target not found
   - Look for: "Verifying LAPTOP_ID connection is alive..."
   - Look for: "LAPTOP_ID connection is stale (homelab_server not connected)"
