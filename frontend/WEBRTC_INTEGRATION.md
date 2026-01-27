# WebRTC Integration Guide

## Overview

This document describes the integration of WebRTC P2P (peer-to-peer) tunnel functionality into the Genix frontend application. The WebRTC bridge enables secure, direct communication between the browser client and a home lab server via WebRTC, with signaling handled through an AWS WebSocket API Gateway.

## Architecture

```
┌─────────────────┐         WebSocket Signaling          ┌──────────────────┐
│   Browser       │ ◄────────────────────────────────► │   AWS Lambda     │
│   (Genix App)   │   offer/answer exchange            │   Signaling       │
│   WSSWebRTC     │                                    │   Server          │
│   Manager       │                                    └────────┬─────────┘
└────────┬────────┘                                             │
         │                                                        │
         │ WebRTC P2P Connection (Direct)                         │
         │ (encrypted, low latency)                               │
         ▼                                                        │
┌─────────────────┐                                              │
│   Home Lab      │ ◄─────────────────────────────────────────────┘
│   Server        │
│   (Go Binary)   │
└─────────────────┘
```

## Implementation Summary

### 1. Core WebRTC Library

**File**: `pkg-core/lib/wss-webrtc.ts`

- **Purpose**: Base WebSocket to WebRTC bridge implementation
- **Key Features**:
  - Handles WebSocket connection to signaling server
  - Creates WebRTC peer connection using `simple-peer`
  - Manages offer/answer signaling protocol
  - Automatic reconnection with exponential backoff
  - Event-driven architecture for connection lifecycle

**Main Class**: `WSSWebRTC`
- `connect()`: Initiate connection
- `disconnect()`: Clean up and disconnect
- `send(data)`: Send data through P2P tunnel
- `on(event, callback)`: Subscribe to events

**Factory Function**: `createWSSWebRTC(config)` - Returns Promise that resolves when P2P connection is established

### 2. WebRTC Manager

**File**: `pkg-core/lib/webrtc/manager.ts`

- **Purpose**: Global singleton manager for the entire application
- **Key Features**:
  - Singleton pattern for centralized management
  - State management with reactive updates
  - Message routing and type-based subscriptions
  - Automatic reconnection logic
  - Integration with Core application state

**Public API**:
```typescript
import { webRTCManager, useWebRTC, initWebRTC } from '$core/lib/webrtc/manager';

// Initialize at app startup
initWebRTC();

// Use in components
const webRTC = useWebRTC();
webRTC.send({ type: 'ping' });
webRTC.onMessage('data', (data) => console.log(data));
```

### 3. Build Configuration

**File**: `vite.config.ts`

- Reads `SIGNALING_ENPOINT` from `../credentials.json` during build
- Injects as `__SIGNALING_ENDPOINT__` constant via Vite's `define` option
- Available at build time, not runtime (security best practice)

**Configuration Flow**:
```
credentials.json → Vite Build → __SIGNALING_ENDPOINT__ constant
```

### 4. Environment Configuration

**File**: `pkg-core/env.ts`

- Added `SIGNALING_ENDPOINT` to `Env` object
- Type-safe access: `Env.SIGNALING_ENDPOINT`
- Falls back to empty string if not configured

### 5. Application State Integration

**File**: `pkg-core/store.svelte.ts`

Added WebRTC state to `Core`:
```typescript
webRTCConnected: false,
webRTCConnecting: false,
webRTCError: null
```

These reactive states can be used in UI components to display connection status.

### 6. Main Layout Integration

**File**: `routes/+layout.svelte`

- Initializes WebRTC manager on app startup
- Subscribes to state changes and updates Core store
- Runs only in browser environment (skipped on server)

### 7. UI Components

#### WebRTC Status Indicator

**File**: `pkg-components/WebRTCStatus.svelte`

- Displays connection status in header
- Shows connected/connecting/disconnected states
- Provides dropdown with detailed information
- Manual reconnect/disconnect buttons
- Integrates with `AppHeader`

#### WebRTC Test Panel

**File**: `routes/components/WebRTCPanel.svelte`

- Full testing and debugging interface
- Features:
  - Send test messages (ping, hello, data request)
  - Custom message input (text or JSON)
  - Auto-ping functionality
  - Message log with timestamps
  - Connection statistics
  - Advanced configuration view
  - Manual reconnect/disconnect controls

Access at: `/components` (scroll down to WebRTC Test Panel)

## Configuration

### Credentials Setup

Ensure your `credentials.json` file contains:

```json
{
  "SIGNALING_ENPOINT": "wss://waogll39kd.execute-api.us-east-1.amazonaws.com/prod"
}
```

Note: The field name is intentionally misspelled as `SIGNALING_ENPOINT` to match the existing configuration.

### Build Process

The signaling endpoint is embedded during build:

```bash
# Build the frontend
bun run build:main

# The build output will include:
# ✅ SIGNALING_ENDPOINT loaded from credentials.json: wss://...
```

## Usage Examples

### 1. Basic Usage in a Component

```svelte
<script lang="ts">
  import { useWebRTC } from '$core/lib/webrtc/manager';
  
  const webRTC = useWebRTC();
  
  function sendRequest() {
    webRTC.send({
      type: 'data_request',
      resource: 'customers'
    });
  }
</script>

<button onclick={sendRequest}>
  Request Data
</button>

{#if webRTC.isConnected}
  <div class="text-green-600">Connected to Home Lab</div>
{:else}
  <div class="text-red-600">Disconnected</div>
{/if}
```

### 2. Subscribe to Specific Message Types

```svelte
<script lang="ts">
  import { useWebRTC } from '$core/lib/webrtc/manager';
  
  const webRTC = useWebRTC();
  let lastMessage = $state(null);
  
  // Subscribe to 'erp_data' messages
  const unsubscribe = webRTC.onMessage('erp_data', (data) => {
    lastMessage = data;
    console.log('Received ERP data:', data);
  });
  
  // Cleanup on destroy
  $effect(() => {
    return unsubscribe;
  });
</script>
```

### 3. Request-Response Pattern

```typescript
import { useWebRTC } from '$core/lib/webrtc/manager';

const webRTC = useWebRTC();

async function sendRequestWithResponse<T>(
  type: string,
  payload: any,
  timeout = 10000
): Promise<T> {
  return new Promise((resolve, reject) => {
    const requestId = `req_${Date.now()}_${Math.random()}`;
    
    // Set up response listener
    const unsubscribe = webRTC.onMessage((message) => {
      if (message.requestId === requestId) {
        unsubscribe();
        if (message.error) {
          reject(new Error(message.error));
        } else {
          resolve(message.payload);
        }
      }
    });
    
    // Set timeout
    setTimeout(() => {
      unsubscribe();
      reject(new Error('Request timeout'));
    }, timeout);
    
    // Send request
    webRTC.send({
      requestId,
      type,
      payload
    });
  });
}

// Usage
try {
  const customers = await sendRequestWithResponse(
    'GET_CUSTOMERS',
    { limit: 10 }
  );
  console.log('Customers:', customers);
} catch (err) {
  console.error('Request failed:', err);
}
```

## Signaling Protocol

### Client to Signaling Server

```json
{
  "action": "sendSignal",
  "to": "laptop",
  "signal": {
    "type": "offer",
    "sdp": "v=0\r\no=- 123456789..."
  }
}
```

### Signaling Server to Client

```json
{
  "signal": {
    "type": "answer",
    "sdp": "v=0\r\no=- 987654321..."
  }
}
```

### P2P Data Messages

Once connected, messages are sent directly:

```json
{
  "type": "ping",
  "timestamp": 1234567890,
  "message": "Hello from Genix"
}
```

## Testing

### 1. Manual Testing via Test Panel

1. Navigate to `/components`
2. Scroll to WebRTC Test Panel
3. Check connection status (should be green "Connected")
4. Send test messages:
   - Click "Send Ping" to test basic connectivity
   - Click "Send Hello" to send a greeting
   - Click "Request Data" to request data from home lab
5. View messages log for responses
6. Monitor ping latency for performance

### 2. Developer Console Testing

```javascript
// Access the WebRTC manager
console.log('WebRTC Manager:', window.webRTCManager);

// Send a message
window.webRTCManager.send({ type: 'test', data: 'hello' });

// Check connection status
const status = window.webRTCManager.getConnectionStatus();
console.log('Status:', status);
```

### 3. Component Integration Testing

```svelte
<script lang="ts">
  import { Core } from '$core/store.svelte';
  
  // Access WebRTC state from Core
  const webRTCState = $derived({
    connected: Core.webRTCConnected,
    connecting: Core.webRTCConnecting,
    error: Core.webRTCError
  });
</script>

{#if webRTCState.connected}
  <div>✅ P2P tunnel active</div>
{:else if webRTCState.connecting}
  <div>⏳ Connecting...</div>
{:else}
  <div>❌ Disconnected: {webRTCState.error}</div>
{/if}
```

## Events

The WebRTC system emits the following events:

### Connection Lifecycle

| Event | Description |
|-------|-------------|
| `wsOpen` | WebSocket connection to signaling server opened |
| `wsClose` | WebSocket connection closed |
| `wsError` | WebSocket connection error |
| `connect` | P2P tunnel established and ready |
| `close` | P2P tunnel closed |

### Data Communication

| Event | Description |
|-------|-------------|
| `data` | Data received through P2P tunnel |
| `signal` | WebRTC signaling data generated |

### Error Handling

| Event | Description |
|-------|-------------|
| `error` | Connection or peer error occurred |

## Security Considerations

1. **Build-time Configuration**: Signaling endpoint is embedded at build time, not exposed in client code
2. **WSS Required**: Always use `wss://` (WebSocket Secure) for production
3. **Data Validation**: Always validate incoming data from P2P connection
4. **Authentication**: Add authentication to signaling server (future enhancement)
5. **Rate Limiting**: Implement rate limiting for connection attempts

## Troubleshooting

### Connection Issues

**Problem**: Connection stuck at "Connecting..."

**Solutions**:
1. Verify signaling endpoint is correct in `credentials.json`
2. Check home lab server is running and connected
3. Review browser console for WebRTC-specific errors
4. Ensure STUN servers are reachable
5. Check network/firewall settings

**Problem**: "Max reconnection attempts reached"

**Solutions**:
1. Check network connectivity
2. Verify signaling server is operational
3. Manually reconnect via status indicator
4. Review home lab server logs

### WebRTC Negotiation Issues

**Problem**: WebRTC connection times out

**Solutions**:
1. Add TURN servers for better NAT traversal
2. Check browser WebRTC support
3. Verify STUN server configuration
4. Try different STUN servers
5. Check for symmetric NAT (requires TURN)

### Build Issues

**Problem**: "SIGNALING_ENDPOINT not loaded"

**Solutions**:
1. Ensure `credentials.json` exists in parent directory
2. Verify field name is `SIGNALING_ENPOINT` (misspelled)
3. Check file is valid JSON
4. Verify build script has file read permissions

## Performance Optimization

1. **Bundle Size**: WebRTC code is in `shared` chunk, loaded only when needed
2. **Lazy Loading**: Test panel is only loaded on `/components` route
3. **Connection Pooling**: Singleton pattern prevents multiple connections
4. **Message Batching**: Implement for high-frequency messages
5. **Compression**: Consider enabling WebSocket compression

## Future Enhancements

1. **Authentication**: Add token-based authentication to signaling server
2. **Encryption**: Application-level encryption for sensitive data
3. **Multiplexing**: Support multiple logical channels over single P2P connection
4. **File Transfer**: Add support for large file transfers via P2P
5. **Recovery**: Implement connection state persistence and recovery
6. **Monitoring**: Add metrics and monitoring for connection health
7. **Video/Audio**: Support WebRTC media streams for video/audio communication

## Related Documentation

- [Deployment Guide](../p2p/DEPLOYMENT.md) - Setting up signaling infrastructure
- [Frontend Connection](../p2p/FRONTEND_CONNECTION.md) - Connection protocol details
- [WSSWebRTC README](pkg-core/lib/wss-webrtc.README.md) - Library documentation
- [Usage Examples](pkg-core/lib/wss-webrtc.example.ts) - Advanced usage patterns

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review browser console for errors
3. Check home lab server logs
4. Verify signaling server status in AWS Console
5. Consult WebRTC documentation: https://webrtc.org/