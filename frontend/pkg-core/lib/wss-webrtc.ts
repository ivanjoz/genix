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
  private ws: WebSocket | null = null;
  private peer: any = null; // SimplePeer instance
  private config: Required<WSSWebRTCConfig>;
  private eventListeners: Partial<BridgeEvents> = {};
  private clientId: string;

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

  connect(): void {
    this.startAppSyncSubscription();
  }

  private startAppSyncSubscription(): void {
    const url = new URL(this.config.wsUrl.replace('wss://', 'https://'));
    const host = url.host;
    
    // Use the official realtime domain for events
    // Example: d6lzr5appndydd5x35xlrz52te.appsync-api.us-east-1.amazonaws.com
    // Becomes: d6lzr5appndydd5x35xlrz52te.appsync-realtime-api.us-east-1.amazonaws.com
    let rtUrl = host.replace('appsync-api', 'appsync-realtime-api');
    rtUrl = `wss://${rtUrl}/event/realtime`;

    const amzDate = new Date().toISOString().replace(/[:\-]|\.\d{3}/g, '');
    
    // Headers must be sorted alphabetically for consistent base64 encoding
    const header = {
      host: host,
      'x-amz-date': amzDate,
      'x-api-key': this.config.apiKey
    };
    
    const headerStr = JSON.stringify(header);
    // Use standard btoa and then apply base64url replacements if needed, 
    // but the example shows a standard base64 string in the protocol
    const headerB64Sub = btoa(headerStr);

    console.log('[WSSWebRTC] Connecting to:', rtUrl);

    // Use AppSync Events subprotocols. No query params needed per working example.
    this.ws = new WebSocket(rtUrl, ['aws-appsync-event-ws', `header-${headerB64Sub}`]);

    this.ws.onopen = () => {
      console.log('[WSSWebRTC] AppSync Events Connected');
      this.ws?.send(JSON.stringify({ type: 'connection_init' }));
    };

    this.ws.onmessage = (evt) => {
      const msg = JSON.parse(evt.data);
      if (msg.type === 'connection_ack') {
        this.subscribe();
        this.startWebRTC();
      } else if (msg.type === 'event' || msg.type === 'data') {
        let events: any[] = [];
        if (msg.type === 'data' && msg.event) {
          events = [msg.event];
        } else if (msg.payload && msg.payload.events) {
          events = msg.payload.events;
        }

        for (const event of events) {
          let signal = event;
          if (typeof event === 'string') {
            try {
              signal = JSON.parse(event);
            } catch (e) {}
          }
          if (signal && signal.to === this.clientId) {
            this.handleIncomingSignal(signal);
          }
        }
      }
    };

    this.ws.onerror = (err) => {
      console.error('[WSSWebRTC] AppSync WebSocket Error:', err);
      this.emit('error', new Error('AppSync WebSocket Error'));
    };
    
    this.ws.onclose = () => {
      console.log('[WSSWebRTC] AppSync WebSocket Closed');
      this.emit('close');
    };
  }

  private subscribe(): void {
    const url = new URL(this.config.wsUrl.replace('wss://', 'https://'));
    const amzDate = new Date().toISOString().replace(/[:\-]|\.\d{3}/g, '');
    
    const header = {
      host: url.host,
      'x-amz-date': amzDate,
      'x-api-key': this.config.apiKey
    };

    const subMsg = {
      id: 'sub-' + Math.random().toString(36).substring(2, 11),
      type: 'subscribe',
      channel: `/genix-bridge/${this.clientId}`,
      authorization: header
    };

    this.ws?.send(JSON.stringify(subMsg));
  }

  private async sendSignal(action: string, data: string): Promise<void> {
    let publishUrl = this.config.wsUrl.replace('wss://', 'https://');
    if (!publishUrl.endsWith('/event')) {
      publishUrl = publishUrl.replace(/\/$/, '') + '/event';
    }

    const amzDate = new Date().toISOString().replace(/[:\-]|\.\d{3}/g, '');
    const signal = {
      from: this.clientId,
      to: this.config.targetId,
      action: action,
      data: data
    };

    try {
      const response = await fetch(publishUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': this.config.apiKey,
          'x-amz-date': amzDate
        },
        body: JSON.stringify({
          channel: `genix-bridge/server`,
          events: [JSON.stringify(signal)]
        })
      });

      if (!response.ok) {
        const errText = await response.text();
        throw new Error(`Publish failed: ${response.status} ${errText}`);
      }
    } catch (err) {
      console.error('[WSSWebRTC] Failed to send signal:', err);
      this.emit('error', err instanceof Error ? err : new Error('Signal send failed'));
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
      if (data.candidate) {
        console.log('[WSSWebRTC] Local ICE Candidate gathered:', data.candidate.candidate);
        if (data.candidate.candidate.includes(':') && !data.candidate.candidate.includes('.')) {
          console.log('[WSSWebRTC] IPv6 Candidate detected');
        }
      } else {
        console.log('[WSSWebRTC] Local Signal:', data.type);
      }
      this.sendSignal(data.type || 'candidate', JSON.stringify(data));
    });

    this.peer.on('connect', () => {
      console.log('[WSSWebRTC] P2P Connected');
      this.emit('connect');
    });

    this.peer.on('data', (data: any) => this.emit('data', data));
    this.peer.on('error', (err: any) => this.emit('error', err));
    this.peer.on('close', () => this.emit('close'));
  }

  private handleIncomingSignal(signal: any): void {
    try {
      const data = JSON.parse(signal.data);
      console.log('[WSSWebRTC] Incoming Signal:', { action: signal.action, data });
      
      if (signal.action === 'candidate') {
        // SimplePeer expects candidates to be wrapped in a 'candidate' property
        // We also sanitize the data to remove nulls that can break the RTCIceCandidate constructor
        const candidateInit: any = {
          candidate: data.candidate,
          sdpMid: data.sdpMid,
          sdpMLineIndex: data.sdpMLineIndex
        };
        
        console.log('[WSSWebRTC] Applying remote ICE candidate');
        this.peer.signal({ candidate: candidateInit });
      } else {
        console.log('[WSSWebRTC] Applying remote SDP:', signal.action);
        this.peer.signal(data);
      }
    } catch (e) {
      console.error('[WSSWebRTC] Error handling incoming signal:', e);
    }
  }

  private emit<K extends keyof BridgeEvents>(event: K, ...args: Parameters<BridgeEvents[K]>): void {
    const listener = this.eventListeners[event];
    if (listener) (listener as any)(...args);
  }

  disconnect(): void {
    this.ws?.close();
    this.peer?.destroy();
  }
}