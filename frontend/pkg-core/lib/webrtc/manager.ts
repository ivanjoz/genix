import { WSSWebRTC, type WSSWebRTCConfig } from '../wss-webrtc';
import { Env } from '../../env';
import { browser } from '$app/environment';

/**
 * Global WebRTC manager for the Genix application
 * Handles the P2P connection to the home lab server
 */

interface WebRTCManagerState {
	connected: boolean;
	connecting: boolean;
	error: string | null;
	lastErrorTime: number;
	reconnectAttempts: number;
	bridge: WSSWebRTC | null;
}

/**
 * Singleton WebRTC Manager
 * Manages the P2P connection lifecycle for the entire application
 */
class WebRTCManager {
	private state: WebRTCManagerState = {
		connected: false,
		connecting: false,
		error: null,
		lastErrorTime: 0,
		reconnectAttempts: 0,
		bridge: null
	};

	private listeners: Set<() => void> = new Set();
	private messageListeners: Map<string, Array<(data: any) => void>> = new Map();
	private config: WSSWebRTCConfig | null = null;
	private isInitialized = false;
	private reconnectTimeout: NodeJS.Timeout | null = null;

	/**
	 * Get the current state (reactive getter)
	 */
	get stateValue(): WebRTCManagerState {
		return { ...this.state };
	}

	/**
	 * Check if connected
	 */
	get isConnected(): boolean {
		return this.state.connected;
	}

	/**
	 * Check if currently connecting
	 */
	get isConnecting(): boolean {
		return this.state.connecting;
	}

	/**
	 * Get the bridge instance
	 */
	get bridge(): WSSWebRTC | null {
		return this.state.bridge;
	}

	/**
	 * Initialize the WebRTC manager
	 * Should be called once when the app starts
	 */
	initialize(signalingEndpoint?: string): void {
		if (this.isInitialized) {
			console.warn('[WebRTCManager] Already initialized');
			return;
		}

		if (!browser) {
			console.log('[WebRTCManager] Running on server, skipping initialization');
			return;
		}

		const wsUrl = signalingEndpoint || Env.SIGNALING_ENDPOINT;
		
		if (!wsUrl) {
			console.warn('[WebRTCManager] No signaling endpoint configured, WebRTC disabled');
			this.state.error = 'No signaling endpoint configured';
			this.notifyListeners();
			return;
		}

		this.config = {
			wsUrl,
			targetId: 'laptop',
			stunServers: [
				'stun:stun.l.google.com:19302',
				'stun:global.stun.twilio.com:3478'
			],
			trickle: false,
			timeout: 30000
		};

		console.log('[WebRTCManager] Initializing with:', {
			wsUrl: this.config.wsUrl,
			targetId: this.config.targetId
		});

		this.isInitialized = true;
		this.connect();
	}

	/**
	 * Connect to the P2P bridge
	 */
	private connect(): void {
		if (!this.config) {
			console.error('[WebRTCManager] Cannot connect: no config');
			return;
		}

		if (this.state.bridge) {
			console.warn('[WebRTCManager] Bridge already exists, disconnecting first');
			this.state.bridge.disconnect();
		}

		this.state.connecting = true;
		this.state.error = null;
		this.notifyListeners();

		console.log('[WebRTCManager] Creating new bridge instance...');
		this.state.bridge = new WSSWebRTC(this.config);

		// Set up event listeners
		this.setupBridgeListeners(this.state.bridge);

		// Initiate connection
		this.state.bridge.connect();
	}

	/**
	 * Set up event listeners for the bridge
	 */
	private setupBridgeListeners(bridge: WSSWebRTC): void {
		bridge.on('wsOpen', () => {
			console.log('[WebRTCManager] ✅ WebSocket connected');
			this.state.connecting = true;
			this.state.reconnectAttempts = 0;
			this.notifyListeners();
		});

		bridge.on('connect', () => {
			console.log('[WebRTCManager] 🎉 P2P tunnel active!');
			this.state.connected = true;
			this.state.connecting = false;
			this.state.error = null;
			this.state.reconnectAttempts = 0;
			this.notifyListeners();

			// Send initial greeting
			this.send({ type: 'app_start', timestamp: Date.now() });
		});

		bridge.on('data', (data) => {
			this.handleIncomingData(data);
		});

		bridge.on('close', () => {
			console.log('[WebRTCManager] 🔌 P2P connection closed');
			this.state.connected = false;
			this.state.connecting = false;
			this.notifyListeners();

			// Attempt reconnection if not manually disconnected
			if (!this.state.error) {
				this.scheduleReconnect();
			}
		});

		bridge.on('wsClose', () => {
			console.log('[WebRTCManager] 📡 WebSocket disconnected');
			this.state.connecting = false;
			this.notifyListeners();

			if (!this.state.connected && !this.state.error) {
				this.scheduleReconnect();
			}
		});

		bridge.on('wsError', (err) => {
			console.error('[WebRTCManager] 📡 WebSocket error:', err);
			this.state.connecting = false;
			this.notifyListeners();
		});

		bridge.on('error', (err) => {
			console.error('[WebRTCManager] ❌ Bridge error:', err.message);
			this.state.error = err.message;
			this.state.lastErrorTime = Date.now();
			this.state.connected = false;
			this.state.connecting = false;
			this.notifyListeners();
		});
	}

	/**
	 * Handle incoming data from the bridge
	 */
	private handleIncomingData(data: unknown): void {
		try {
			const message = typeof data === 'string' ? JSON.parse(data) : data;
			console.log('[WebRTCManager] 📥 Received message:', message);

			// Emit to message listeners
			if (message.type && this.messageListeners.has(message.type)) {
				const listeners = this.messageListeners.get(message.type)!;
				listeners.forEach(callback => callback(message));
			}

			// Also emit to any '*' listeners
			if (this.messageListeners.has('*')) {
				const listeners = this.messageListeners.get('*')!;
				listeners.forEach(callback => callback(message));
			}
		} catch (err) {
			console.error('[WebRTCManager] Error handling incoming data:', err);
		}
	}

	/**
	 * Schedule a reconnection attempt
	 */
	private scheduleReconnect(): void {
		if (this.reconnectTimeout) {
			clearTimeout(this.reconnectTimeout);
		}

		if (this.state.reconnectAttempts >= 5) {
			console.error('[WebRTCManager] Max reconnection attempts reached');
			this.state.error = 'Max reconnection attempts reached';
			this.notifyListeners();
			return;
		}

		const delay = Math.min(3000 * Math.pow(2, this.state.reconnectAttempts), 30000);
		this.state.reconnectAttempts++;

		console.log(
			`[WebRTCManager] Scheduling reconnection attempt ${this.state.reconnectAttempts}/5 in ${delay}ms`
		);

		this.reconnectTimeout = setTimeout(() => {
			this.connect();
		}, delay);
	}

	/**
	 * Send data through the P2P connection
	 */
	send(data: any): void {
		if (!this.state.bridge || !this.state.connected) {
			console.warn('[WebRTCManager] Cannot send: not connected');
			return;
		}

		try {
			const message = typeof data === 'string' ? data : JSON.stringify(data);
			console.log('[WebRTCManager] 📤 Sending:', message);
			this.state.bridge.send(message);
		} catch (err) {
			console.error('[WebRTCManager] Error sending data:', err);
		}
	}

	/**
	 * Subscribe to state changes
	 */
	subscribe(callback: () => void): () => void {
		this.listeners.add(callback);
		return () => {
			this.listeners.delete(callback);
		};
	}

	/**
	 * Subscribe to messages of a specific type
	 */
	onMessage(type: string, callback: (data: any) => void): () => void {
		if (!this.messageListeners.has(type)) {
			this.messageListeners.set(type, []);
		}
		this.messageListeners.get(type)!.push(callback);
		return () => {
			const listeners = this.messageListeners.get(type);
			if (listeners) {
				const index = listeners.indexOf(callback);
				if (index > -1) {
					listeners.splice(index, 1);
				}
			}
		};
	}

	/**
	 * Disconnect and cleanup
	 */
	disconnect(): void {
		console.log('[WebRTCManager] Disconnecting...');

		if (this.reconnectTimeout) {
			clearTimeout(this.reconnectTimeout);
			this.reconnectTimeout = null;
		}

		if (this.state.bridge) {
			this.state.bridge.disconnect();
			this.state.bridge = null;
		}

		this.state.connected = false;
		this.state.connecting = false;
		this.notifyListeners();
	}

	/**
	 * Reconnect manually
	 */
	reconnect(): void {
		console.log('[WebRTCManager] Manual reconnect requested');
		this.state.reconnectAttempts = 0;
		this.state.error = null;
		this.disconnect();
		setTimeout(() => this.connect(), 1000);
	}

	/**
	 * Notify all state listeners
	 */
	private notifyListeners(): void {
		this.listeners.forEach((callback) => callback());
	}

	/**
	 * Get connection status for UI display
	 */
	getConnectionStatus(): {
		connected: boolean;
		connecting: boolean;
		error: string | null;
		reconnectAttempts: number;
	} {
		return {
			connected: this.state.connected,
			connecting: this.state.connecting,
			error: this.state.error,
			reconnectAttempts: this.state.reconnectAttempts
		};
	}
}

// Singleton instance
export const webRTCManager = new WebRTCManager();

// Convenience function to initialize from the app
export function initWebRTC(signalingEndpoint?: string): void {
	webRTCManager.initialize(signalingEndpoint);
}

// Convenience functions to use in components
export function useWebRTC() {
	return {
		state: webRTCManager.stateValue,
		isConnected: webRTCManager.isConnected,
		isConnecting: webRTCManager.isConnecting,
		send: (data: any) => webRTCManager.send(data),
		onMessage: (type: string, callback: (data: any) => void) => 
			webRTCManager.onMessage(type, callback),
		subscribe: (callback: () => void) => webRTCManager.subscribe(callback),
		reconnect: () => webRTCManager.reconnect(),
		disconnect: () => webRTCManager.disconnect(),
		getConnectionStatus: () => webRTCManager.getConnectionStatus()
	};
}