import { sendServiceMessage, registerServiceHandler } from '$core/lib/sw-cache';

const DEBUG_PREFIX = '[WSSWebRTC Client]';

export interface SignalData {
  type?: 'offer' | 'answer' | 'candidate' | 'pranswer' | 'rollback';
  sdp?: string;
  candidate?: string;
  sdpMLineIndex?: number;
  sdpMid?: string;
}

export interface WSSWebRTCConfig {
  wsUrl: string; // AppSync Events Endpoint (e.g., https://.../event)
  apiKey: string;
  targetId: string;
  stunServers?: string[];
  trickle?: boolean;
  timeout?: number;
}

export type BridgeEvents = {
  connect: () => void;
  close: () => void;
  error: (err: Error) => void;
  data: (data: any) => void;
  signal: (data: SignalData) => void;
};

export class WSSWebRTC {
  private peer: any = null; // SimplePeer instance
  private config: Required<WSSWebRTCConfig>;
  private eventListeners: Partial<BridgeEvents> = {};
  private clientId: string | null = null;
  private connected: boolean = false;
  private signalHandlerId: number = 0;

  constructor(config: WSSWebRTCConfig) {
    this.config = {
      wsUrl: config.wsUrl,
      apiKey: config.apiKey,
      targetId: config.targetId,
      stunServers: config.stunServers || [
        'stun:stun.l.google.com:19302',
        'stun:global.stun.twilio.com:3478'
      ],
      trickle: config.trickle ?? false,
      timeout: config.timeout ?? 30000
    };
this.clientId = 'client-' + Math.random().toString(36).substring(2, 11);
this.registerSignalHandler();
  }

  on<K extends keyof BridgeEvents>(event: K, callback: BridgeEvents[K]): void {
    this.eventListeners[event] = callback;
  }

  send(data: any): void {
    if (this.peer && this.peer.connected) {
      this.peer.send(data);
    } else {
      console.warn('[WSSWebRTC] Cannot send data: Peer not connected');
    }
  }

  async connect(): Promise<void> {
    console.log(`${DEBUG_PREFIX} Connecting to AppSync...`);

    try {
      const response = await sendServiceMessage(30, {
        wsUrl: this.config.wsUrl,
        apiKey: this.config.apiKey,
        targetId: this.config.targetId,
        stunServers: this.config.stunServers,
        trickle: this.config.trickle,
        timeout: this.config.timeout
      });

      if (response.error) {
        throw new Error(response.error);
      }

      this.clientId = response.clientId;
      this.connected = true;
      console.log(`${DEBUG_PREFIX} Connected with clientId: ${this.clientId}`);

      this.startWebRTC();
    } catch (err) {
      console.error(`${DEBUG_PREFIX} Connection failed:`, err);
      this.emit('error', err instanceof Error ? err : new Error('Connection failed'));
    }
  }

  private registerSignalHandler(): void {
    registerServiceHandler(40, (data: any) => {
      if (!this.connected && this.clientId) {
        this.connected = true;
      }
      this.handleIncomingSignal(data.signal);
    });
  }

  private async sendSignal(action: string, data: string): Promise<void> {
    try {
      await sendServiceMessage(31, {
        action: action,
        data: data
      });
    } catch (err) {
      console.error(`${DEBUG_PREFIX} Failed to send signal:`, err);
      this.emit('error', err instanceof Error ? err : new Error('Signal send failed'));
      throw err;
    }
  }

  private startWebRTC(): void {
    const SimplePeer = (globalThis as any).SimplePeer;
    if (!SimplePeer) {
      this.emit('error', new Error('SimplePeer not found'));
      return;
    }

    this.peer = new SimplePeer({
      initiator: true,
      trickle: this.config.trickle,
      config: { 
        iceServers: this.config.stunServers.map(urls => ({ urls })),
        bundlePolicy: 'max-bundle',
        rtcpMuxPolicy: 'require'
      }
    });

    this.peer.on('signal', (data: any) => {
      this.sendSignal(data.type || 'candidate', JSON.stringify(data));
    });

    this.peer.on('connect', () => {
      console.log(`${DEBUG_PREFIX} P2P Connected!`);
      this.emit('connect');
    });

    this.peer.on('data', (data: any) => {
      this.emit('data', data);
    });

    this.peer.on('error', (err: any) => {
      console.error(`${DEBUG_PREFIX} WebRTC Error:`, err);
      this.emit('error', err);
    });

    this.peer.on('close', () => {
      console.log(`${DEBUG_PREFIX} WebRTC Connection closed`);
      this.emit('close');
    });
  }

  private handleIncomingSignal(signal: any): void {
    try {
      const data = JSON.parse(signal.data);

      if (signal.action === 'candidate') {
        const candidateInit: any = {
          candidate: data.candidate,
          sdpMid: data.sdpMid,
          sdpMLineIndex: data.sdpMLineIndex
        };
        this.peer.signal({ candidate: candidateInit });
      } else {
        this.peer.signal(data);
      }
    } catch (e) {
      console.error(`${DEBUG_PREFIX} Error handling incoming signal:`, e);
    }
  }

  private emit<K extends keyof BridgeEvents>(event: K, ...args: Parameters<BridgeEvents[K]>): void {
    const listener = this.eventListeners[event];
    if (listener) {
      (listener as any)(...args);
    }
  }

  async disconnect(): Promise<void> {
    try {
      await sendServiceMessage(32, {});
    } catch (err) {
      console.error(`${DEBUG_PREFIX} Disconnect failed:`, err);
    }

    this.peer?.destroy();
    this.connected = false;
  }
}