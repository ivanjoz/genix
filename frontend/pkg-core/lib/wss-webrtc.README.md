# WSSWebRTC Bridge

A WebSocket to WebRTC bridge for establishing secure peer-to-peer (P2P) connections between a browser client and a home lab server. This library handles the signaling protocol via WebSocket and creates a direct P2P connection using WebRTC.

## Features

- 🔒 **Secure P2P Connection**: Direct encrypted connection between client and server
- 🔄 **Automatic Reconnection**: Built-in reconnection logic with configurable attempts
- 📡 **WebSocket Signaling**: Uses WebSocket API for WebRTC signaling exchange
- ⚡ **Type-Safe**: Written in TypeScript with full type definitions
- 🎯 **Event-Driven**: Simple event system for connection lifecycle management
- 🛠️ **Configurable**: Customizable STUN servers, timeouts, and connection options

## Installation

This library requires `simple-peer` as a peer dependency:

```bash
bun add simple-peer @types/simple-peer
```

## Quick Start

```typescript
import { WSSWebRTC, createWSSWebRTC } from '$lib/pkg-core/lib/wss-webrtc';

// Method 1: Using the class directly
const bridge = new WSSWebRTC({
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge'
});

bridge.on('connect', () => {
  console.log('P2P tunnel active!');
  bridge.send('Hello from browser!');
});

bridge.on('data', (data) => {
  console.log('Received:', data);
});

bridge.connect();

// Method 2: Using the async factory function
const bridge = await createWSSWebRTC({
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge'
});
// Connection is now established and ready to use
```

## Configuration Options

```typescript
interface WSSWebRTCConfig {
  // WebSocket URL for signaling server (required)
  wsUrl: string;
  
  // Target ID to connect to, e.g., "genix-bridge" (required)
  targetId: string;
  
  // STUN servers for NAT traversal (optional)
  stunServers?: string[];
  
  // Enable/disable trickle ICE (default: false)
  trickle?: boolean;
  
  // Connection timeout in milliseconds (default: 30000)
  timeout?: number;
}
```

### Default Configuration

```typescript
{
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge',
  stunServers: [
    'stun:stun.l.google.com:19302',
    'stun:global.stun.twilio.com:3478'
  ],
  trickle: false,
  timeout: 30000
}
```

## API Reference

### `WSSWebRTC`

Main class for managing WebSocket to WebRTC connections.

#### Constructor

```typescript
new WSSWebRTC(config: WSSWebRTCConfig)
```

Creates a new bridge instance with the specified configuration.

#### Methods

##### `connect()`
```typescript
bridge.connect(): void
```
Connects to the WebSocket server and initiates the WebRTC handshake.

##### `disconnect()`
```typescript
bridge.disconnect(): void
```
Disconnects and cleans up all resources. Stops automatic reconnection.

##### `send(data: any)`
```typescript
bridge.send(data: any): void
```
Sends data through the P2P connection. Throws an error if the connection is not established.

##### `isConnected()`
```typescript
bridge.isConnected(): boolean
```
Returns `true` if the P2P connection is active and ready to send data.

##### `on<K>(event: K, callback: BridgeEvents[K])`
```typescript
bridge.on('connect', () => console.log('Connected!'))
bridge.on('data', (data) => console.log('Received:', data))
```
Registers an event listener for the specified event.

##### `off<K>(event: K)`
```typescript
bridge.off('connect')
```
Removes the event listener for the specified event.

### `createWSSWebRTC`

Factory function that creates and connects a bridge, resolving when the P2P connection is established.

```typescript
async function createWSSWebRTC(config: WSSWebRTCConfig): Promise<WSSWebRTC>
```

**Example:**
```typescript
const bridge = await createWSSWebRTC({
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge',
  timeout: 30000
});
// Connection is ready
bridge.send('Hello!');
```

## Events

The bridge emits the following events:

### Connection Lifecycle

| Event | Description | Callback Signature |
|-------|-------------|-------------------|
| `wsOpen` | WebSocket connection opened | `() => void` |
| `wsClose` | WebSocket connection closed | `() => void` |
| `wsError` | WebSocket connection error | `(err: Event) => void` |
| `connect` | P2P connection established | `() => void` |
| `close` | P2P connection closed | `() => void` |

### Data Communication

| Event | Description | Callback Signature |
|-------|-------------|-------------------|
| `data` | Data received from peer | `(data: any) => void` |
| `signal` | WebRTC signal generated | `(data: SignalData) => void` |

### Error Handling

| Event | Description | Callback Signature |
|-------|-------------|-------------------|
| `error` | Connection or peer error | `(err: Error) => void` |

## Usage Examples

### Basic Example with All Events

```typescript
const bridge = new WSSWebRTC({
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge'
});

// WebSocket events
bridge.on('wsOpen', () => console.log('📡 WebSocket connected'));
bridge.on('wsClose', () => console.log('📡 WebSocket disconnected'));
bridge.on('wsError', (err) => console.error('📡 WebSocket error:', err));

// P2P connection events
bridge.on('connect', () => {
  console.log('🎉 P2P tunnel active!');
  bridge.send('Requesting data...');
});
bridge.on('close', () => console.log('🔌 P2P connection closed'));

// Data events
bridge.on('data', (data) => {
  console.log('📥 Received:', data);
});

// Error handling
bridge.on('error', (err) => {
  console.error('❌ Error:', err.message);
});

// Start connection
bridge.connect();
```

### Request-Response Pattern

```typescript
const bridge = new WSSWebRTC({
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge'
});

const pendingRequests = new Map();

bridge.on('data', (data) => {
  const { requestId, payload } = JSON.parse(data);
  if (pendingRequests.has(requestId)) {
    const { resolve } = pendingRequests.get(requestId);
    pendingRequests.delete(requestId);
    resolve(payload);
  }
});

async function sendRequest(type, payload, timeoutMs = 10000) {
  return new Promise((resolve, reject) => {
    const requestId = `req_${Date.now()}_${Math.random()}`;
    
    pendingRequests.set(requestId, {
      resolve,
      timeout: setTimeout(() => {
        pendingRequests.delete(requestId);
        reject(new Error('Request timeout'));
      }, timeoutMs)
    });
    
    bridge.send(JSON.stringify({ requestId, type, payload }));
  });
}

// Usage
bridge.on('connect', async () => {
  const customers = await sendRequest('GET_CUSTOMERS', { limit: 10 });
  console.log('Customers:', customers);
});

bridge.connect();
```

### React Integration

```typescript
import { useEffect, useState, useRef } from 'svelte';
import { WSSWebRTC } from '$lib/pkg-core/lib/wss-webrtc';

function useWSSWebRTC(wsUrl, targetId) {
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState(null);
  const [data, setData] = useState(null);
  const bridgeRef = useRef(null);

  useEffect(() => {
    const bridge = new WSSWebRTC({ wsUrl, targetId });
    bridgeRef.current = bridge;

    bridge.on('wsOpen', () => setConnecting(true));
    bridge.on('connect', () => {
      setConnected(true);
      setConnecting(false);
    });
    bridge.on('data', setData);
    bridge.on('error', setError);
    bridge.on('close', () => {
      setConnected(false);
      setConnecting(false);
    });

    bridge.connect();

    return () => bridge.disconnect();
  }, [wsUrl, targetId]);

  const send = (data) => {
    bridgeRef.current?.send(data);
  };

  return { connected, connecting, error, data, send };
}
```

## Error Handling

The bridge handles errors at multiple levels:

### WebSocket Errors
```typescript
bridge.on('wsError', (err) => {
  console.error('WebSocket connection failed:', err);
  // Bridge will attempt automatic reconnection
});
```

### Target Not Connected Error
```typescript
bridge.on('error', (err) => {
  if (err.message.includes('not currently connected')) {
    console.error('Homelab server is not running!');
    // Notify user that they need to start the homelab_server
    // The server must be connected to the signaling server before WebRTC can be established
  }
});
```

### WebRTC Peer Errors
```typescript
bridge.on('error', (err) => {
  console.error('WebRTC peer error:', err);
  // Connection will be cleaned up
  // Implement custom reconnection logic if needed
});
```

### Connection Timeout
```typescript
const bridge = await createWSSWebRTC({
  wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
  targetId: 'genix-bridge',
  timeout: 30000
}).catch(err => {
  console.error('Connection timeout:', err);
  // Handle timeout scenario
});
```

## Signaling Protocol

The bridge communicates with the signaling server using a JSON-based protocol:

### Client to Signaling Server
```json
{
  "action": "sendSignal",
  "to": "genix-bridge",
  "signal": {
    "type": "offer",
    "sdp": "..."
  }
}
```

### Signaling Server to Client
```json
{
  "signal": {
    "type": "answer",
    "sdp": "..."
  }
}
```

## Reconnection Behavior

The bridge automatically attempts to reconnect to the WebSocket server if the connection is lost:

- **Max Attempts**: 5 (configurable via `maxReconnectAttempts`)
- **Delay**: 3000ms between attempts (configurable via `reconnectDelay`)
- **Manual Close**: Calling `disconnect()` stops automatic reconnection

To disable automatic reconnection or customize the behavior, you can extend the class:

```typescript
class CustomBridge extends WSSWebRTC {
  constructor(config) {
    super(config);
    this.maxReconnectAttempts = 3;
    this.reconnectDelay = 5000;
  }
}
```

## Security Considerations

1. **Always Use WSS**: Ensure your WebSocket URL uses `wss://` (WebSocket Secure)
2. **Validate Data**: Always validate incoming data before processing
3. **Error Boundaries**: Implement proper error handling in production
4. **Connection Limits**: Implement rate limiting for connection attempts
5. **Authentication**: Add authentication to your signaling server (not handled by this library)

## Troubleshooting

### Connection Fails
- Verify the WebSocket URL is correct and accessible
- Check that the home lab server is running and connected to the signaling server
- If you see "The target (homelab_server) is not currently connected", ensure the homelab_server binary is running
- Ensure network connectivity and firewall rules allow WebRTC traffic

### WebRTC Negotiation Times Out
- Check STUN server configuration
- Verify NAT traversal settings
- Try different STUN servers or add TURN servers
- Check browser console for WebRTC-specific errors

### Intermittent Disconnections
- Network instability can cause WebRTC connections to drop
- Implement proper reconnection logic in your application
- Check WebSocket server logs for connection patterns

## Architecture

```
┌─────────────┐         WebSocket          ┌──────────────┐
│   Browser   │ ◄──────────────────────► │   Lambda     │
│  (Client)   │   Signaling Messages     │  (Relay)     │
│   WSSWebRTC │   (offer/answer)          │   Signaling   │
└─────────────┘                        └──────┬───────┘
                                              │
                                              │ WebRTC P2P
                                              ▼
                                     ┌─────────────────┐
                                     │   Home Lab     │
                                     │   Server       │
                                     │  (Go Binary)   │
                                     └─────────────────┘
```

## Dependencies

- **simple-peer**: WebRTC wrapper library
- **@types/simple-peer**: TypeScript definitions for simple-peer

## License

Part of the Genix project. See project LICENSE for details.

## See Also

- [Deployment Guide](../../p2p/DEPLOYMENT.md) - Setting up the signaling infrastructure
- [Frontend Connection Guide](../../p2p/FRONTEND_CONNECTION.md) - Connection protocol details
- [Usage Examples](./wss-webrtc.example.ts) - More advanced usage patterns