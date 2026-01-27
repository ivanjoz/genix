# Connection Issues Troubleshooting

## Quick Reference

### "The target (homelab_server) is not currently connected"

**Cause:** Homelab server not running or not connected to Lambda.

**Solution:**
```bash
# Check if running
ps aux | grep homelab_server

# Start if not running
cd /path/to/genix/p2p/homelab_server
./homelab_server
```

### WebSocket Connection Fails

**Cause:** Invalid WebSocket URL, network issues, or API Gateway not deployed.

**Solution:**
1. Verify WebSocket URL format: `wss://<api-id>.execute-api.<region>.amazonaws.com/prod`
2. Check deployment: `cd deploy && npx cdk list`
3. Re-deploy if needed: `cd .. && ./deploy.sh`
4. Test manually: `wscat -c "wss://your-api-id.execute-api.us-east-1.amazonaws.com/prod"`

### WebRTC Negotiation Times Out

**Cause:** NAT traversal issues, firewall blocking, or STUN/TURN server problems.

**Solution:**
1. Check browser console for ICE connection failures
2. Verify STUN servers are accessible (default: `stun:stun.l.google.com:19302`)
3. Try different STUN servers in configuration
4. Configure TURN server if behind restrictive NAT
5. Check firewall allows UDP traffic for WebRTC

## Diagnostic Steps

### Step 1: Check Frontend Console

Open browser console and look for:
- `[WSSWebRTC] WebSocket connected` - WebSocket is working
- `[WSSWebRTC] Generated WebRTC signal: offer` - WebRTC is working
- Error messages - indicates what's failing

### Step 2: Check Lambda Logs (CloudWatch)

Navigate to Lambda in AWS Console → View CloudWatch logs.

**Successful connection:**
```
[INFO] ===== Lambda Handler Started =====
[DEBUG] RouteKey: $connect
[INFO] Connection allowed, returning 200
```

**Homelab server connecting:**
```
[INFO] Processing sendSignal route
[DEBUG] LAPTOP_ID environment variable is set: <connection-id>
```

**Error scenario:**
```
[ERROR] Target connection <id> does not exist (homelab_server not connected)
[INFO] Sent error notification to sender
```

### Step 3: Check Homelab Server Status

```bash
# Check process
ps aux | grep homelab_server

# Check logs (systemd service)
sudo journalctl -u homelab-p2p-bridge -f

# Check log file
tail -f /path/to/homelab_server.log
```

Look for successful WebSocket connection and connection ID.

### Step 4: Test WebSocket Manually

```bash
# Install wscat
npm install -g wscat

# Connect
wscat -c "wss://your-api-id.execute-api.us-east-1.amazonaws.com/prod"

# Send test message
{"action":"sendSignal","to":"me","data":"test"}
```

Should receive the same message back.

## Common Error Messages

| Error Message | Cause | Solution |
|--------------|-------|----------|
| `The target (homelab_server) is not currently connected` | Homelab server not running or not connected | Start homelab_server and verify it connects to Lambda |
| `WebSocket connection failed` | Invalid URL or network issue | Verify WebSocket URL and network connectivity |
| `WebRTC connection timeout` | NAT traversal or firewall issue | Check STUN/TURN servers, configure firewall |
| `Internal server error` | Lambda error (older version) | Update Lambda: `go run update-lambda.go --wait` |
| `Unknown message format` | Old Lambda version | Update Lambda and re-deploy |

## Testing Procedure

### Test 1: WebSocket Only
```javascript
const ws = new WebSocket('wss://your-api-id.execute-api.us-east-1.amazonaws.com/prod');
ws.onopen = () => console.log('✅ WebSocket connected');
ws.onerror = (err) => console.error('❌ WebSocket error:', err);
```

### Test 2: With Homelab Server Running
1. Start homelab_server
2. Verify it appears in Lambda logs with LAPTOP_ID set
3. Start frontend client
4. Check for successful P2P connection

### Test 3: Without Homelab Server (Stale LAPTOP_ID)
1. Ensure homelab_server is NOT running
2. LAPTOP_ID may contain stale data (e.g., "init" from previous session)
3. Start frontend client
4. Should receive clear error message about target not connected
5. Lambda logs should show: "Verifying LAPTOP_ID connection is alive..." then "LAPTOP_ID connection is stale (homelab_server not connected)"

## Pre-Deployment Checklist

- [ ] Lambda is deployed and WebSocket URL is known
- [ ] `LAPTOP_ID` environment variable can be set by Lambda
- [ ] Homelab server is configured with correct WebSocket URL
- [ ] Network allows outbound connections from homelab server to AWS
- [ ] Network allows inbound/outbound WebRTC traffic (UDP/TCP)
- [ ] Firewall rules are configured correctly
- [ ] STUN servers are accessible (or TURN servers configured)

## Stale LAPTOP_ID Handling

The Lambda automatically detects and handles stale LAPTOP_ID connections:

1. **When frontend connects with target "laptop":**
   - Lambda resolves "laptop" to LAPTOP_ID environment variable
   - Sends a ping to verify the connection is alive

2. **If LAPTOP_ID is stale (connection doesn't exist):**
   - Lambda detects `GoneException` from the ping
   - Logs: "LAPTOP_ID connection is stale (homelab_server not connected)"
   - Marks LAPTOP_ID as unavailable for this request
   - Sends error to frontend: "The target (homelab_server) is not currently connected"

3. **If LAPTOP_ID is valid and alive:**
   - Logs: "LAPTOP_ID connection verified as alive"
   - Proceeds to relay the signal normally

**No manual cleanup required** - the Lambda handles stale connections automatically on each request.

## Update Lambda Code

After modifying `signaling_lambda/main.go`:

```bash
# Build binary
./build-lambda.sh

# Update Lambda (without full redeploy)
go run update-lambda.go --wait
```

## Related Documentation

- [WSSWebRTC README](../../frontend/pkg-core/lib/wss-webrtc.README.md) - Frontend library documentation
- [Fix: Homelab Unavailability](./fix-homelab-unavailable.md) - Specific fix for this error
- [Main README](../README.md) - Project overview and deployment guide