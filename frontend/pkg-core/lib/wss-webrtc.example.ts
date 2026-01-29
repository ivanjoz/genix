import { WSSWebRTC, createWSSWebRTC, WSSWebRTCConfig } from './wss-webrtc';

/**
 * Example 1: Basic Usage with Event Listeners
 * This demonstrates the basic setup and connection flow
 */
export async function basicExample() {
  // Configuration for the WebSocket and WebRTC connection
  const config: WSSWebRTCConfig = {
    // WebSocket URL from AWS CDK deployment
    // e.g., 'wss://xxx.execute-api.us-east-1.amazonaws.com/prod'
    wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    
    // Target ID to connect to (usually "genix-bridge" as defined in DEPLOYMENT.md)
    targetId: 'genix-bridge',
    
    // Optional: Custom STUN servers for better NAT traversal
    stunServers: [
      'stun:stun.l.google.com:19302',
      'stun:global.stun.twilio.com:3478'
    ],
    
    // Optional: Disable trickle ICE (send all candidates at once)
    trickle: false,
    
    // Optional: Connection timeout in milliseconds
    timeout: 30000
  };

  // Create the bridge instance
  const bridge = new WSSWebRTC(config);

  // Register event listeners
  bridge.on('wsOpen', () => {
    console.log('✅ WebSocket connection established');
    console.log('Waiting for WebRTC handshake...');
  });

  bridge.on('connect', () => {
    console.log('🎉 P2P tunnel active!');
    console.log('You can now send data directly to the home lab server');
  });

  bridge.on('data', (data) => {
    console.log('📥 Received data from home lab server:', data);
    
    // Example: Handle different types of data
    if (typeof data === 'string') {
      try {
        const json = JSON.parse(data);
        if (json.type === 'ERP_DATA') {
          console.log('Received ERP data:', json.payload);
        }
      } catch (e) {
        // Not JSON, handle as plain string
      }
    }
  });

  bridge.on('error', (err) => {
    console.error('❌ Connection error:', err.message);
  });

  bridge.on('close', () => {
    console.log('🔌 Connection closed');
  });

  bridge.on('wsClose', () => {
    console.log('📡 WebSocket disconnected');
  });

  // Start the connection
  bridge.connect();

  // Send data after connection is established
  setTimeout(() => {
    if (bridge.isConnected()) {
      bridge.send('Requesting ERP Data...');
    } else {
      console.error('Cannot send: P2P connection not established');
    }
  }, 5000);

  // Cleanup when done
  setTimeout(() => {
    bridge.disconnect();
    console.log('Disconnected from bridge');
  }, 60000);
}

/**
 * Example 2: Using Factory Function with Promise
 * This demonstrates using the async factory function for cleaner initialization
 */
export async function factoryExample() {
  const config: WSSWebRTCConfig = {
    wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    targetId: 'genix-bridge',
    timeout: 30000
  };

  try {
    // Create and connect, resolves when P2P connection is established
    const bridge = await createWSSWebRTC(config);
    
    console.log('🎉 Connected! Bridge is ready to use');
    
    // Send data
    bridge.send(JSON.stringify({
      action: 'get_data',
      resource: 'customers'
    }));

    // Handle incoming data
    bridge.on('data', (data) => {
      console.log('Received:', data);
    });

    // Handle errors
    bridge.on('error', (err) => {
      console.error('Error:', err);
    });

    // Disconnect after some time
    setTimeout(() => bridge.disconnect(), 30000);

  } catch (error) {
    console.error('Failed to establish connection:', error);
  }
}

/**
 * Example 3: React Hook Integration
 * This demonstrates how to use WSSWebRTC in a React component
 */
export function createWSSWebRTCHook() {
  // This is a conceptual example for integration with React
  // You would use this in your React components
  
  const config: WSSWebRTCConfig = {
    wsUrl: process.env.WS_URL || 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    targetId: 'genix-bridge'
  };

  const bridge = new WSSWebRTC(config);
  
  const state = {
    connected: false,
    connecting: false,
    error: null as Error | null,
    data: null as any
  };

  bridge.on('wsOpen', () => {
    state.connecting = true;
    console.log('Connecting to P2P tunnel...');
  });

  bridge.on('connect', () => {
    state.connected = true;
    state.connecting = false;
    console.log('P2P tunnel active!');
  });

  bridge.on('error', (err) => {
    state.error = err;
    state.connecting = false;
    console.error('Connection error:', err);
  });

  bridge.on('data', (data) => {
    state.data = data;
    console.log('Received data:', data);
  });

  bridge.on('close', () => {
    state.connected = false;
    state.connecting = false;
  });

  const connect = () => bridge.connect();
  const disconnect = () => bridge.disconnect();
  const send = (data: any) => bridge.send(data);

  return { state, connect, disconnect, send };
}

/**
 * Example 4: Error Handling and Reconnection
 * This demonstrates robust error handling and reconnection logic
 */
export async function robustExample() {
  const config: WSSWebRTCConfig = {
    wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    targetId: 'genix-bridge',
    timeout: 30000
  };

  const bridge = new WSSWebRTC(config);

  let connectionAttempts = 0;
  const maxAttempts = 5;

  bridge.on('error', (err) => {
    console.error('Connection error:', err.message);
    
    connectionAttempts++;
    
    if (connectionAttempts < maxAttempts) {
      console.log(`Retrying connection (${connectionAttempts}/${maxAttempts})...`);
      
      // Wait before reconnecting
      setTimeout(() => {
        bridge.connect();
      }, 5000);
    } else {
      console.error('Max connection attempts reached. Giving up.');
    }
  });

  bridge.on('wsClose', () => {
    if (bridge.isConnected()) {
      console.warn('WebSocket closed but P2P connection still active');
    } else {
      console.warn('Both WebSocket and P2P connection closed');
    }
  });

  // Initial connection
  bridge.connect();

  // Heartbeat mechanism to detect dead connections
  const heartbeatInterval = setInterval(() => {
    if (bridge.isConnected()) {
      try {
        bridge.send(JSON.stringify({ type: 'heartbeat', timestamp: Date.now() }));
      } catch (err) {
        console.error('Heartbeat failed:', err);
        clearInterval(heartbeatInterval);
      }
    }
  }, 30000);

  // Cleanup on unmount
  return () => {
    clearInterval(heartbeatInterval);
    bridge.disconnect();
  };
}

/**
 * Example 5: Request-Response Pattern
 * This demonstrates how to implement a request-response pattern over WebRTC
 */
export async function requestResponseExample() {
  const config: WSSWebRTCConfig = {
    wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    targetId: 'genix-bridge'
  };

  const bridge = new WSSWebRTC(config);

  // Map to store pending requests
  const pendingRequests = new Map<string, {
    resolve: (value: any) => void;
    reject: (error: Error) => void;
    timeout: NodeJS.Timeout;
  }>();

  bridge.on('data', (data) => {
    try {
      const message = typeof data === 'string' ? JSON.parse(data) : data;
      
      // Check if this is a response to a pending request
      if (message.requestId && pendingRequests.has(message.requestId)) {
        const { resolve, reject, timeout } = pendingRequests.get(message.requestId)!;
        
        clearTimeout(timeout);
        pendingRequests.delete(message.requestId);
        
        if (message.error) {
          reject(new Error(message.error));
        } else {
          resolve(message.payload);
        }
      }
    } catch (e) {
      console.error('Error handling response:', e);
    }
  });

  // Function to send requests and wait for responses
  function sendRequest<T>(type: string, payload: any, timeoutMs = 10000): Promise<T> {
    return new Promise((resolve, reject) => {
      const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      
      // Set up timeout
      const timeout = setTimeout(() => {
        pendingRequests.delete(requestId);
        reject(new Error(`Request timeout after ${timeoutMs}ms`));
      }, timeoutMs);
      
      // Store the promise handlers
      pendingRequests.set(requestId, { resolve, reject, timeout });
      
      // Send the request
      bridge.send(JSON.stringify({
        requestId,
        type,
        payload
      }));
    });
  }

  // Connect and use the request-response pattern
  await createWSSWebRTC(config);
  
  // Example requests
  try {
    const customers = await sendRequest('GET_CUSTOMERS', { limit: 10 });
    console.log('Customers:', customers);
    
    const orders = await sendRequest('GET_ORDERS', { customerId: customers[0].id });
    console.log('Orders:', orders);
  } catch (err) {
    console.error('Request failed:', err);
  }
}

/**
 * Example 6: Streaming Data
 * This demonstrates how to handle streaming data over WebRTC
 */
export async function streamingExample() {
  const config: WSSWebRTCConfig = {
    wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    targetId: 'genix-bridge'
  };

  const bridge = new WSSWebRTC(config);

  bridge.on('connect', () => {
    console.log('Connected, requesting data stream...');
    
    // Request a stream of data
    bridge.send(JSON.stringify({
      type: 'start_stream',
      stream: 'realtime_data',
      interval: 1000
    }));
  });

  bridge.on('data', (data) => {
    const message = typeof data === 'string' ? JSON.parse(data) : data;
    
    if (message.type === 'stream_data') {
      console.log('Stream data:', message.data);
      
      // You could update UI here
      // updateState({ lastData: message.data });
      
    } else if (message.type === 'stream_end') {
      console.log('Stream ended:', message.reason);
    }
  });

  bridge.connect();
  
  // Stop the stream after some time
  setTimeout(() => {
    if (bridge.isConnected()) {
      bridge.send(JSON.stringify({
        type: 'stop_stream'
      }));
    }
  }, 60000);
}

/**
 * Example 7: Integration with Application State
 * This demonstrates how to integrate WSSWebRTC with your app's state management
 */
export class AppStateBridge {
  private bridge: WSSWebRTC;
  private state = {
    connected: false,
    connecting: false,
    lastUpdate: null as Date | null,
    data: new Map<string, any>()
  };

  constructor(config: WSSWebRTCConfig) {
    this.bridge = new WSSWebRTC(config);
    this.setupEventListeners();
  }

  private setupEventListeners() {
    this.bridge.on('wsOpen', () => {
      this.state.connecting = true;
      this.notifyListeners('status', { status: 'connecting' });
    });

    this.bridge.on('connect', () => {
      this.state.connected = true;
      this.state.connecting = false;
      this.notifyListeners('status', { status: 'connected' });
      
      // Request initial data
      this.requestData('all');
    });

    this.bridge.on('data', (data) => {
      const message = typeof data === 'string' ? JSON.parse(data) : data;
      
      if (message.type === 'update') {
        this.state.data.set(message.resource, message.data);
        this.state.lastUpdate = new Date();
        this.notifyListeners('data', { resource: message.resource, data: message.data });
      }
    });

    this.bridge.on('error', (err) => {
      this.state.connected = false;
      this.state.connecting = false;
      this.notifyListeners('status', { status: 'error', error: err.message });
    });

    this.bridge.on('close', () => {
      this.state.connected = false;
      this.state.connecting = false;
      this.notifyListeners('status', { status: 'disconnected' });
    });
  }

  private listeners: Map<string, Array<(data: any) => void>> = new Map();

  on(event: string, callback: (data: any) => void) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(callback);
  }

  off(event: string, callback: (data: any) => void) {
    const callbacks = this.listeners.get(event);
    if (callbacks) {
      const index = callbacks.indexOf(callback);
      if (index > -1) {
        callbacks.splice(index, 1);
      }
    }
  }

  private notifyListeners(event: string, data: any) {
    const callbacks = this.listeners.get(event);
    if (callbacks) {
      callbacks.forEach(callback => callback(data));
    }
  }

  connect() {
    this.bridge.connect();
  }

  disconnect() {
    this.bridge.disconnect();
  }

  isConnected() {
    return this.bridge.isConnected();
  }

  getState() {
    return {
      ...this.state,
      data: Object.fromEntries(this.state.data)
    };
  }

  private requestData(resource: string) {
    if (!this.bridge.isConnected()) {
      throw new Error('Bridge not connected');
    }
    
    this.bridge.send(JSON.stringify({
      type: 'request',
      resource
    }));
  }

  sendData(type: string, data: any) {
    if (!this.bridge.isConnected()) {
      throw new Error('Bridge not connected');
    }
    
    this.bridge.send(JSON.stringify({
      type,
      data
    }));
  }
}

// Example usage of AppStateBridge
export function appStateExample() {
  const config: WSSWebRTCConfig = {
    wsUrl: 'wss://your-api-id.execute-api.region.amazonaws.com/prod',
    targetId: 'genix-bridge'
  };

  const appBridge = new AppStateBridge(config);

  // Listen for status changes
  appBridge.on('status', (status) => {
    console.log('Status changed:', status);
  });

  // Listen for data updates
  appBridge.on('data', (update) => {
    console.log('Data updated:', update.resource, update.data);
  });

  // Connect to the bridge
  appBridge.connect();
}