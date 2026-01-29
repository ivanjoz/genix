export interface SignalData {
  type?: 'offer' | 'answer' | 'candidate' | 'pranswer' | 'rollback';
  sdp?: string;
  candidate?: string;
  sdpMLineIndex?: number;
  sdpMid?: string;
}

export interface WSSWebRTCConfig {
  wsUrl: string; // AppSync GraphQL URL
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

  connect(): void {
    this.startAppSyncSubscription();
  }

  private startAppSyncSubscription(): void {
    const url = new URL(this.config.wsUrl.replace('wss://', 'https://'));
    const host = url.host;
    const rtUrl = this.config.wsUrl
      .replace('https://', 'wss://')
      .replace('appsync-api', 'appsync-realtime-api');

    const header = {
      host: host,
      'x-api-key': this.config.apiKey
    };
    const headerB64 = btoa(JSON.stringify(header));
    const finalUrl = `${rtUrl}?header=${headerB64}&payload=e30=`;

    this.ws = new WebSocket(finalUrl, 'graphql-ws');

    this.ws.onopen = () => {
      console.log('[WSSWebRTC] AppSync Connected');
      this.ws?.send(JSON.stringify({ type: 'connection_init' }));
    };

    this.ws.onmessage = (evt) => {
      const msg = JSON.parse(evt.data);
      if (msg.type === 'connection_ack') {
        this.subscribe();
        this.startWebRTC();
      } else if (msg.type === 'data') {
        const signal = msg.payload.data.onSignal;
        if (signal && signal.to === this.clientId) {
          this.handleIncomingSignal(signal);
        }
      }
    };

    this.ws.onerror = (err) => this.emit('error', new Error('AppSync WebSocket Error'));
  }

  private subscribe(): void {
    const header = {
      host: new URL(this.config.wsUrl.replace('wss://', 'https://')).host,
      'x-api-key': this.config.apiKey
    };
    const query = `subscription onSignal($to: String!) { onSignal(to: $to) { from to action data } }`;
    const variables = { to: this.clientId };

    this.ws?.send(JSON.stringify({
      id: 'sub-client',
      type: 'start',
      payload: {
        data: JSON.stringify({
          query,
          variables
        }),
        extensions: {
          authorization: header
        }
      }
    }));
  }

  private async sendSignal(action: string, data: string): Promise<void> {
    const query = `mutation sendSignal($from: String!, $to: String!, $action: String!, $data: String!) {
      sendSignal(from: $from, to: $to, action: $action, data: $data) { from to action data }
    }`;
    const variables = {
      from: this.clientId,
      to: this.config.targetId,
      action: action,
      data: data
    };

    await fetch(this.config.wsUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': this.config.apiKey
      },
      body: JSON.stringify({ query, variables })
    });
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
    const data = JSON.parse(signal.data);
    this.peer.signal(data);
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