// Using local simplepeer.min.js file to avoid npm package issues
// The library will be available as global 'SimplePeer' object
// Type definitions for SimplePeer
export interface SignalData {
  type?: 'offer' | 'answer' | 'candidate' | 'pranswer' | 'rollback';
  sdp?: string;
  candidate?: string;
  sdpMLineIndex?: number;
  sdpMid?: string;
}

/**
 * Standard application message structure for P2P communication
 */
export interface AppMessage {
  accion: string;
  id: string | number;
  body?: any;
}

/**
 * Standard application response structure
 */
export interface AppResponse {
  accion: string;
  id: string | number;
  body?: any;
  error?: string;
}

export interface PeerInstance {
  connected: boolean;
  send(data: any): void;
  signal(data: SignalData): void;
  destroy(): void;
  on(event: 'signal', callback: (data: SignalData) => void): void;
  on(event: 'connect', callback: () => void): void;
  on(event: 'close', callback: () => void): void;
  on(event: 'error', callback: (err: Error) => void): void;
  on(event: 'data', callback: (data: any) => void): void;
}

/**
 * Configuration options for the WSS WebRTC bridge
 */
export interface WSSWebRTCConfig {
  /** WebSocket URL for signaling server */
  wsUrl: string;
  /** Target ID to connect to (e.g., "laptop") */
  targetId: string;
  /** STUN servers for WebRTC NAT traversal */
  stunServers?: string[];
  /** Enable trickle ICE (true = send candidates as they arrive, false = wait) */
  trickle?: boolean;
  /** Connection timeout in milliseconds */
  timeout?: number;
}

/**
 * Events that can be emitted by the WSSWebRTC bridge
 */
export type BridgeEvents = {
  /** Emitted when P2P connection is established */
  connect: () => void;
  /** Emitted when P2P connection is closed */
  close: () => void;
  /** Emitted when an error occurs */
  error: (err: Error) => void;
  /** Emitted when a data message is received */
  data: (data: any) => void;
  /** Emitted when signaling data is available */
  signal: (data: SignalData) => void;
  /** Emitted when WebSocket connection opens */
  wsOpen: () => void;
  /** Emitted when WebSocket connection closes */
  wsClose: () => void;
  /** Emitted when WebSocket connection errors */
  wsError: (err: Event) => void;
};

/**
 * WebSocket to WebRTC bridge for P2P connections
 * Handles signaling via WebSocket and establishes direct P2P connection via WebRTC
 */
export class WSSWebRTC {
  private ws: WebSocket | null = null;
  private peer: PeerInstance | null = null;
  private config: Required<WSSWebRTCConfig>;
  private eventListeners: Partial<BridgeEvents> = {};
  private pendingRequests: Map<string | number, { resolve: (val: any) => void, reject: (err: Error) => void, timeout: any }> = new Map();
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;
  private reconnectDelay: number = 3000;
  private isManualClose: boolean = false;

  constructor(config: WSSWebRTCConfig) {
    this.config = {
      wsUrl: config.wsUrl,
      targetId: config.targetId,
      stunServers: config.stunServers || [
        'stun:stun.l.google.com:19302',
        'stun:global.stun.twilio.com:3478'
      ],
      trickle: config.trickle ?? false,
      timeout: config.timeout ?? 30000
    };
  }

  /**
   * Register event listeners
   */
  on<K extends keyof BridgeEvents>(event: K, callback: BridgeEvents[K]): void {
    this.eventListeners[event] = callback;
  }

  /**
   * Remove event listener
   */
  off<K extends keyof BridgeEvents>(event: K): void {
    delete this.eventListeners[event];
  }

  /**
   * Send a structured request and wait for a response
   */
  async request<T = any>(accion: string, body: any = {}, id?: string | number): Promise<T> {
    if (!this.peer?.connected) {
      throw new Error('P2P connection not established');
    }

    const requestId = id || Math.random().toString(36).substring(2, 11);
    const message: AppMessage = { accion, id: requestId, body };

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        if (this.pendingRequests.has(requestId)) {
          this.pendingRequests.delete(requestId);
          reject(new Error(`Request timeout: ${accion}`));
        }
      }, 10000); // 10 second timeout for app requests

      this.pendingRequests.set(requestId, { resolve, reject, timeout });
      this.send(JSON.stringify(message));
    });
  }

  /**
   * Connect to WebSocket server and start WebRTC handshake
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      console.warn('[WSSWebRTC] WebSocket already connected');
      return;
    }

    this.isManualClose = false;
    this.reconnectAttempts = 0;

    try {
      this.ws = new WebSocket(this.config.wsUrl);

      this.ws.onopen = () => this.handleWsOpen();
      this.ws.onmessage = (evt) => this.handleWsMessage(evt);
      this.ws.onclose = (evt) => this.handleWsClose(evt);
      this.ws.onerror = (err) => this.handleWsError(err);
    } catch (err) {
      this.emit('error', err as Error);
    }
  }

  /**
   * Disconnect and cleanup
   */
  disconnect(): void {
    this.isManualClose = true;
    this.cleanup();
  }

  /**
   * Send data through the P2P connection
   */
  send(data: any): void {
    if (!this.peer || !this.peer.connected) {
      const err = new Error('P2P connection not established');
      this.emit('error', err);
      return;
    }

    try {
      this.peer.send(data);
    } catch (err) {
      this.emit('error', err as Error);
    }
  }

  /**
   * Check if P2P connection is active
   */
  isConnected(): boolean {
    return this.peer?.connected ?? false;
  }

  /**
   * Handle WebSocket connection open
   */
  private handleWsOpen(): void {
    console.log('[WSSWebRTC] WebSocket connected');
    this.reconnectAttempts = 0;
    this.emit('wsOpen');

    // Start WebRTC handshake
    this.startWebRTC();
  }

  /**
   * Handle WebSocket message (signaling data or error messages)
   */
  private handleWsMessage(evt: MessageEvent): void {
    try {
      const message = JSON.parse(evt.data);

      // Check for error messages from the signaling server
      if (message.error === 'backend_unreachable') {
        console.warn('[WSSWebRTC] Backend unreachable:', message.message);

        // Mark as manual close to prevent internal reconnection attempts
        this.isManualClose = true;

        // Emit error FIRST so listeners (like WebRTCManager) can set their error state
        // before the 'close' event (triggered by destroy()) is handled.
        this.emit('error', new Error(message.message || 'The target is not currently connected'));

        this.cleanup();
        return;
      }

      const signal = message.signal || message.data;
      if (signal && (signal.type || signal.candidate)) {
        console.log('[WSSWebRTC] Received signal from signaling server');
        this.handleSignal(signal);
      } else if (message.action === 'connected' || message.action === 'ping') {
        // Ignore internal signaling messages
        console.log('[WSSWebRTC] Received control message:', message.action);
      } else {
        console.warn('[WSSWebRTC] Unknown message format:', message);
      }
    } catch (err) {
      console.error('[WSSWebRTC] Error parsing WebSocket message:', err);
      this.emit('error', err as Error);
    }
  }

  /**
   * Handle WebSocket connection close
   */
  private handleWsClose(evt: CloseEvent): void {
    console.log('[WSSWebRTC] WebSocket closed');
    console.log('[WSSWebRTC] Close code:', evt.code, 'Reason:', evt.reason);
    this.emit('wsClose');

    // If connection closes immediately without opening (first attempt), backend is likely unavailable
    // Lambda returns HTTP 503 when homelab_server is not connected
    if (!this.isManualClose && this.reconnectAttempts === 0) {
      console.error('[WSSWebRTC] Connection failed on first attempt - backend likely unavailable');
      this.isManualClose = true;
      this.emit('error', new Error('The target (homelab_server) is not currently connected. Please start the homelab_server to establish a WebRTC connection.'));
      return;
    }

    // Try to reconnect if not manually closed
    if (!this.isManualClose && this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`[WSSWebRTC] Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      setTimeout(() => this.connect(), this.reconnectDelay);
    }
  }

  /**
   * Handle WebSocket error
   */
  private handleWsError(err: Event): void {
    console.error('[WSSWebRTC] WebSocket error:', err);
    this.emit('wsError', err);
  }

  /**
   * Initialize WebRTC peer connection and send offer
   */
  private startWebRTC(): void {
    try {
      // Create peer connection as initiator
      this.peer = new (globalThis as any).SimplePeer({
        initiator: true,
        trickle: this.config.trickle,
        config: {
          iceServers: this.config.stunServers.map(server => ({ urls: server }))
        }
      }) as PeerInstance;

      // Set up peer event listeners
      this.peer.on('signal', (data: SignalData) => this.handlePeerSignal(data));
      this.peer.on('connect', () => this.handlePeerConnect());
      this.peer.on('close', () => this.handlePeerClose());
      this.peer.on('error', (err: Error) => this.handlePeerError(err));
      this.peer.on('data', (data: any) => this.handlePeerData(data));

      // Set connection timeout
      setTimeout(() => {
        if (this.peer && !this.peer.connected) {
          this.handlePeerError(new Error('WebRTC connection timeout'));
        }
      }, this.config.timeout);

    } catch (err) {
      console.error('[WSSWebRTC] Error initializing WebRTC peer:', err);
      this.emit('error', err as Error);
    }
  }

  /**
   * Handle WebRTC signal generation (send to signaling server)
   */
  private handlePeerSignal(data: SignalData): void {
    console.log('[WSSWebRTC] Generated WebRTC signal:', data.type);
    this.emit('signal', data);

    // Send signal to target via WebSocket
    const message = {
      action: 'sendSignal',
      to: this.config.targetId,
      signal: data,
      data: data // Include both for backward/forward compatibility
    };

    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.error('[WSSWebRTC] WebSocket not ready to send signal');
    }
  }

  /**
   * Handle incoming WebRTC signal from signaling server
   */
  private handleSignal(signal: SignalData): void {
    if (!this.peer) {
      console.error('[WSSWebRTC] Received signal but no peer exists');
      return;
    }

    console.log('[WSSWebRTC] Applying signal to peer:', signal.type || 'candidate');
    this.peer.signal(signal);
  }

  /**
   * Handle WebRTC peer connection established
   */
  private handlePeerConnect(): void {
    console.log('[WSSWebRTC] P2P tunnel active!');
    this.emit('connect');
  }

  /**
   * Handle WebRTC peer connection closed
   */
  private handlePeerClose(): void {
    console.log('[WSSWebRTC] P2P connection closed');
    this.cleanup();
    this.emit('close');
  }

  /**
   * Handle WebRTC peer error
   */
  private handlePeerError(err: Error): void {
    console.error('[WSSWebRTC] Peer error:', err);
    this.cleanup();
    this.emit('error', err);
  }

  /**
   * Handle data received through P2P connection
   */
  private handlePeerData(data: unknown): void {
    console.log('[WSSWebRTC] Received data:', data);

    let decodedData = data;
    if (data && typeof data === 'object') {
      const d = data as any;
      if (d.type === 'Buffer' && Array.isArray(d.data)) {
        decodedData = new TextDecoder().decode(Uint8Array.from(d.data));
      }
    }

    // Try to parse as AppResponse
    if (typeof decodedData === 'string' || decodedData instanceof Uint8Array) {
      try {
        const text = typeof decodedData === 'string' ? decodedData : new TextDecoder().decode(decodedData);
        const parsed = JSON.parse(text) as AppResponse;

        if (parsed.id && this.pendingRequests.has(parsed.id)) {
          const { resolve, reject, timeout } = this.pendingRequests.get(parsed.id)!;
          clearTimeout(timeout);
          this.pendingRequests.delete(parsed.id);

          if (parsed.error) {
            reject(new Error(parsed.error));
          } else {
            resolve(parsed.body);
          }
          return; // Handled as a request/response
        }
      } catch (e) {
        // Not a JSON message or not an AppResponse, continue to emit 'data'
      }
    }

    this.emit('data', decodedData);
  }

  /**
   * Cleanup resources
   */
  private cleanup(): void {
    this.isManualClose = true; // Prevent handleWsClose from emitting redundant errors
    if (this.peer) {
      this.peer.destroy();
      this.peer = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Emit event to registered listeners
   */
  private emit<K extends keyof BridgeEvents>(event: K, ...args: Parameters<BridgeEvents[K]>): void {
    const listener = this.eventListeners[event];
    if (listener) {
      (listener as Function)(...args);
    }
  }
}

/**
 * Factory function to create and connect WSSWebRTC bridge
 */
export async function createWSSWebRTC(config: WSSWebRTCConfig): Promise<WSSWebRTC> {
  const bridge = new WSSWebRTC(config);

  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      bridge.off('connect');
      bridge.off('error');
      reject(new Error('Connection timeout'));
    }, config.timeout ?? 30000);

    bridge.on('connect', () => {
      clearTimeout(timeout);
      bridge.off('error');
      resolve(bridge);
    });

    bridge.on('error', (err) => {
      clearTimeout(timeout);
      bridge.off('connect');
      reject(err);
    });

    bridge.connect();
  });
}
