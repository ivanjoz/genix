import { sendServiceMessage, registerServiceHandler } from '$core/lib/sw-cache';

const DEBUG_PREFIX = '[WSSWebRTC Client]';

const getTS = () => {
  const d = new Date();
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}.${d.getMilliseconds().toString().padStart(3, '0')}`;
};

export interface SignalData {
  type?: 'offer' | 'answer' | 'candidate' | 'pranswer' | 'rollback';
  sdp?: string;
  candidate?: string;
  sdpMLineIndex?: number;
  sdpMid?: string;
  sessionToken?: string;
}

interface CachedConnectionData {
  candidates: RTCIceCandidateInit[];
  timestamp: number;
  sessionToken: string;
}

class ConnectionCache {
  private static readonly CACHE_KEY = 'webrtc_conn_cache';
  private static readonly MAX_AGE = 10 * 60 * 1000; // 10 minutes

  static save(data: CachedConnectionData): void {
    try {
      localStorage.setItem(this.CACHE_KEY, JSON.stringify(data));
    } catch (e) {
      console.error(`${DEBUG_PREFIX} Failed to save cache`, e);
    }
  }

  static load(): CachedConnectionData | null {
    try {
      const raw = localStorage.getItem(this.CACHE_KEY);
      if (!raw) return null;
      const data: CachedConnectionData = JSON.parse(raw);
      if (Date.now() - data.timestamp > this.MAX_AGE) {
        localStorage.removeItem(this.CACHE_KEY);
        return null;
      }
      return data;
    } catch (e) {
      return null;
    }
  }

  static clear(): void {
    localStorage.removeItem(this.CACHE_KEY);
  }
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
  private sessionToken: string = '';
  private connectionMode: 'fast' | 'normal' = 'fast';
  private fallbackTimeout: any = null;

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
    console.log(`[${getTS()}] ${DEBUG_PREFIX} Connecting to AppSync...`);

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
      console.log(`[${getTS()}] ${DEBUG_PREFIX} Connected with clientId: ${this.clientId}`);

      const cache = ConnectionCache.load();
      if (cache) {
        console.log(`[${getTS()}] ${DEBUG_PREFIX} Attempting FAST RELOAD with token: ${cache.sessionToken}`);
        this.connectionMode = 'fast';
        this.sessionToken = cache.sessionToken;
        this.startWebRTC(cache.candidates);

        // Fallback timer: 2 seconds to connect or we switch to normal
        this.fallbackTimeout = setTimeout(() => {
          if (!this.peer?.connected) {
            console.warn(`[${getTS()}] ${DEBUG_PREFIX} Fast Reload timed out, falling back to normal connection`);
            this.switchToNormalMode();
          }
        }, 2500);
      } else {
        this.connectionMode = 'normal';
        this.sessionToken = 'sess-' + Math.random().toString(36).substring(2, 11);
        this.startWebRTC();
      }
    } catch (err) {
      console.error(`${DEBUG_PREFIX} Connection failed:`, err);
      this.emit('error', err instanceof Error ? err : new Error('Connection failed'));
    }
  }

  private switchToNormalMode(): void {
    if (this.fallbackTimeout) clearTimeout(this.fallbackTimeout);
    this.connectionMode = 'normal';
    ConnectionCache.clear();

    if (this.peer) {
      this.peer.destroy();
    }
    this.startWebRTC();
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
      // Inject session token into signaling
      const signalPayload = JSON.parse(data);
      signalPayload.sessionToken = this.sessionToken;

      await sendServiceMessage(31, {
        action: action,
        data: JSON.stringify(signalPayload)
      });
    } catch (err) {
      console.error(`${DEBUG_PREFIX} Failed to send signal:`, err);
      this.emit('error', err instanceof Error ? err : new Error('Signal send failed'));
      throw err;
    }
  }

  private startWebRTC(preInjectCandidates: RTCIceCandidateInit[] = []): void {
    const SimplePeer = (globalThis as any).SimplePeer;
    if (!SimplePeer) {
      this.emit('error', new Error('SimplePeer not found'));
      return;
    }

    this.peer = new SimplePeer({
      initiator: true,
      trickle: true, // Always allow trickle for robustness
      config: {
        iceServers: this.config.stunServers.map(urls => ({ urls })),
        bundlePolicy: 'max-bundle',
        rtcpMuxPolicy: 'require'
      }
    });

    // Aggressive candidate injection for Fast Reload
    if (preInjectCandidates.length > 0) {
      console.log(`[${getTS()}] ${DEBUG_PREFIX} Pre-injecting ${preInjectCandidates.length} cached candidates`);
      // Sort candidates: IPv6 first
      const sorted = [...preInjectCandidates].sort((a, b) => {
        const aIsV6 = a.candidate?.includes(':') ? 1 : 0;
        const bIsV6 = b.candidate?.includes(':') ? 1 : 0;
        return bIsV6 - aIsV6;
      });

      sorted.forEach(c => {
        try {
          this.peer.signal({ candidate: c });
        } catch (e) {
          // Ignore injection errors
        }
      });
    }

    this.peer.on('signal', (data: any) => {
      if (this.peer?.connected) return; // Stop signaling once connected
      this.sendSignal(data.type || 'candidate', JSON.stringify(data));
    });

    this.peer.on('connect', () => {
      if (this.fallbackTimeout) clearTimeout(this.fallbackTimeout);
      console.log(`[${getTS()}] ${DEBUG_PREFIX} P2P Connected (${this.connectionMode} mode)!`);
      this.emit('connect');

      // Save success to cache
      this.saveConnectionToCache();
    });

    this.peer.on('data', (data: any) => {
      this.emit('data', data);
    });

    this.peer.on('error', (err: any) => {
      // If we are already connected, ignore errors from late/redundant signals
      if (this.peer?.connected) {
        console.warn(`[${getTS()}] ${DEBUG_PREFIX} WebRTC non-critical error (already connected):`, err.message);
        return;
      }

      console.error(`[${getTS()}] ${DEBUG_PREFIX} WebRTC Error:`, err);
      if (this.connectionMode === 'fast') {
        console.warn(`[${getTS()}] ${DEBUG_PREFIX} Fast mode error, falling back...`);
        this.switchToNormalMode();
      } else {
        this.emit('error', err);
      }
    });

    this.peer.on('close', () => {
      console.log(`${DEBUG_PREFIX} WebRTC Connection closed`);
      if (this.fallbackTimeout) clearTimeout(this.fallbackTimeout);
      this.emit('close');
    });
  }

  private saveConnectionToCache(): void {
    if (!this.peer?._pc) return; // SimplePeer internal PeerConnection

    const pc: RTCPeerConnection = this.peer._pc;
    const remoteSdp = pc.remoteDescription?.sdp;

    if (remoteSdp) {
      const candidates = this.extractCandidatesFromSDP(remoteSdp);
      if (candidates.length > 0) {
        ConnectionCache.save({
          candidates,
          sessionToken: this.sessionToken,
          timestamp: Date.now()
        });
        console.log(`${DEBUG_PREFIX} Cached ${candidates.length} candidates for next reload`);
      }
    }
  }

    private extractCandidatesFromSDP(sdp: string): RTCIceCandidateInit[] {
      const candidates: RTCIceCandidateInit[] = [];
      const seen = new Set<string>();
      const lines = sdp.split('\n');
      
      for (const line of lines) {
        if (line.startsWith('a=candidate:')) {
          const candidate = line.substring(2).trim();
          // Simple de-duplication
          if (seen.has(candidate)) continue;
          seen.add(candidate);
  
          candidates.push({
            candidate,
            sdpMid: '0',
            sdpMLineIndex: 0
          });
        }
      }
      // Limit to top 12 candidates to avoid bloat
      return candidates.slice(0, 12);
    }
      private handleIncomingSignal(signal: any): void {
        if (!this.peer || this.peer.destroyed) return;
    
        try {
          // Ignore ALL signals if we are already connected to avoid "Called in wrong state" errors
          if (this.peer.connected) {
            console.log(`[${getTS()}] ${DEBUG_PREFIX} Ignored late ${signal.action} signal (already connected)`);
            return;
          }
    
          const data = JSON.parse(signal.data);
          
          // Update session token if server sends a new one
          if (signal.sessionToken) {
            this.sessionToken = signal.sessionToken;
          }
    
          if (signal.action === 'candidate') {
            const candidateInit: any = {
              candidate: data.candidate,
              sdpMid: data.sdpMid,
              sdpMLineIndex: data.sdpMLineIndex
            };
            this.peer.signal({ candidate: candidateInit });
          } else {
            // Only signal if not connected
            this.peer.signal(data);
          }
        } catch (e) {
          if (!this.peer.destroyed) {
            console.error(`[${getTS()}] ${DEBUG_PREFIX} Error handling incoming signal:`, e);
          }
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
