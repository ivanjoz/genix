# Troubleshooting Index

Quick reference for diagnosing and resolving connection issues in the Genix P2P system.

## Quick Diagnosis

| Symptom | Likely Cause | Quick Fix |
|---------|--------------|-----------|
| "The target (homelab_server) is not currently connected" | Homelab server not running or not connected (including stale LAPTOP_ID) | Start homelab_server |
| WebSocket connection fails | Invalid URL or network issue | Verify WebSocket URL, re-deploy |
| WebRTC negotiation times out | NAT traversal or firewall issue | Check STUN servers, configure firewall |
| "Internal server error" | Old Lambda code | Update Lambda: `go run update-lambda.go --wait` |
| "Unknown message format" | Old Lambda version | Update and re-deploy Lambda |

## Common Issues

### Homelab Server Not Connected

**Check:**
```bash
ps aux | grep homelab_server
```

**Fix:**
```bash
cd /path/to/genix/p2p/homelab_server
./homelab_server
```

**Verify:** Check Lambda CloudWatch logs for LAPTOP_ID being set and connection verified as alive.

### WebSocket Connection Issues

**Test manually:**
```bash
wscat -c "wss://your-api-id.execute-api.us-east-1.amazonaws.com/prod"
{"action":"sendSignal","to":"me","data":"test"}
```

**Re-deploy if needed:**
```bash
cd /path/to/genix/p2p
./deploy.sh
```

### WebRTC Connection Timeout

**Check:** Browser console for ICE connection failures

**Fix:** Verify STUN servers are accessible or configure TURN server for restrictive NAT

## Lambda Code Updates
### Update Lambda Code

After modifying `signaling_lambda/main.go`:

```bash
# Build binary
./build-lambda.sh

# Update Lambda (no full redeploy needed)
go run update-lambda.go --wait
```

**Note:** The Lambda automatically detects and handles stale LAPTOP_ID connections. No manual cleanup needed.

## Log Locations

- **Frontend:** Browser console (F12)
- **Lambda:** AWS CloudWatch → Lambda function → View logs
- **Homelab Server:** `sudo journalctl -u homelab-p2p-bridge -f`

## Documentation

- [Connection Issues](./connection-issues.md) - Detailed troubleshooting procedures
- [Fix: Homelab Unavailable](./fix-homelab-unavailable.md) - Error handling implementation
- [Main README](../README.md) - Project overview and deployment
- [WSSWebRTC README](../../frontend/pkg-core/lib/wss-webrtc.README.md) - Frontend library documentation

## Error Response Format

When homelab_server is not connected (or LAPTOP_ID is stale):
```json
{
  "error": "backend_unreachable",
  "message": "The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.",
  "target": "<connection-id>"
}
```

The Lambda verifies LAPTOP_ID connection is alive by sending a ping before relaying signals. If the connection is stale (GoneException), it automatically sends this error to the frontend.

## Status Codes

| Status | Meaning |
|--------|---------|
| 200 | Connection accepted / Signal relayed |
| 404 | Target connection not found (homelab_server not connected) |
| 400 | Invalid route key |
| 500 | Unexpected Lambda error |
