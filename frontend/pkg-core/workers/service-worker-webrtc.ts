/// <reference lib="WebWorker" />
"use-strict"

import { sendClientMessage } from "./service-worker-cache"

export interface SignalData {
  type?: 'offer' | 'answer' | 'candidate' | 'pranswer' | 'rollback';
  sdp?: string;
  candidate?: string;
  sdpMLineIndex?: number;
  sdpMid?: string;
}

export interface AppSyncWebRTCConfig {
  wsUrl: string; // AppSync Events Endpoint (e.g., https://.../event)
  apiKey: string;
  targetId: string;
  stunServers?: string[];
  trickle?: boolean;
  timeout?: number;
  __client__: number;
}

// Store active WebSocket connections by targetId for reuse across clients/reloads
const connectionsByTarget: Map<string, AppSyncConnection> = new Map()
// Store which targetId each client is associated with
const clientTargetMap: Map<number, string> = new Map()
// Store connection configs for reconnection
const connectionConfigs: Map<number, Required<AppSyncWebRTCConfig>> = new Map()

class AppSyncConnection {
  private ws: WebSocket | null = null;
  private config: Required<AppSyncWebRTCConfig>;
  private clientId: string;
  private clientIDs: Set<number> = new Set();
  private subscribed = false;
  private connectPromise: Promise<{ clientId: string; connected: boolean }> | null = null;

  constructor(config: AppSyncWebRTCConfig) {
    this.config = {
      wsUrl: config.wsUrl,
      apiKey: config.apiKey,
      targetId: config.targetId,
      stunServers: config.stunServers || [
        'stun:stun.l.google.com:19302',
        'stun:global.stun.twilio.com:3478'
      ],
      trickle: config.trickle ?? false,
      timeout: config.timeout ?? 30000,
      __client__: config.__client__
    };
    this.clientId = 'client-' + Math.random().toString(36).substring(2, 11);
    this.clientIDs.add(config.__client__);
  }

  addClient(clientID: number) {
    this.clientIDs.add(clientID);
  }

  removeClient(clientID: number) {
    this.clientIDs.delete(clientID);
  }

  hasClients(): boolean {
    return this.clientIDs.size > 0;
  }

  async connect(): Promise<{ clientId: string; connected: boolean }> {
    if (this.isConnected()) {
      return { clientId: this.clientId, connected: true };
    }

    if (this.connectPromise) {
      return this.connectPromise;
    }

    this.connectPromise = new Promise((resolve, reject) => {
      try {
        this.startAppSyncSubscription(resolve, reject);
      } catch (err) {
        this.connectPromise = null;
        reject(err);
      }
    });

    return this.connectPromise;
  }

  private startAppSyncSubscription(resolve: (v: any) => void, reject: (e: any) => void): void {
    const url = new URL(this.config.wsUrl.replace('wss://', 'https://'));
    const host = url.host;

    let rtUrl = host.replace('appsync-api', 'appsync-realtime-api');
    rtUrl = `wss://${rtUrl}/event/realtime`;

    const amzDate = new Date().toISOString().replace(/[:\-]|\.\d{3}/g, '');
    const header = {
      host: host,
      'x-amz-date': amzDate,
      'x-api-key': this.config.apiKey
    };

    const headerB64Sub = btoa(JSON.stringify(header));

    console.log('[AppSync] Connecting to', rtUrl);
    this.ws = new WebSocket(rtUrl, ['aws-appsync-event-ws', `header-${headerB64Sub}`]);

    const timeout = setTimeout(() => {
      if (this.ws?.readyState !== WebSocket.OPEN) {
        this.ws?.close();
        this.connectPromise = null;
        reject(new Error('Connection timeout'));
      }
    }, this.config.timeout);

    this.ws.onopen = () => {
      this.ws?.send(JSON.stringify({ type: 'connection_init' }));
    };

    this.ws.onmessage = (evt) => {
      const msg = JSON.parse(evt.data);

      if (msg.type === 'connection_ack') {
        clearTimeout(timeout);
        this.subscribe();
        resolve({ clientId: this.clientId, connected: true });
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
            try { signal = JSON.parse(event); } catch (e) {}
          }
          if (signal && signal.to === this.clientId) {
            this.handleIncomingSignal(signal);
          }
        }
      }
    };

    this.ws.onerror = (err) => {
      clearTimeout(timeout);
      this.connectPromise = null;
      this.sendErrorToClient(new Error('WebSocket Error'));
      reject(err);
    };

    this.ws.onclose = () => {
      this.connectPromise = null;
      this.sendCloseToClient();
    };
  }

  private subscribe(): void {
    if (this.subscribed) return;

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
    this.subscribed = true;
  }

  async sendSignal(action: string, data: string): Promise<void> {
    let publishUrl = this.config.wsUrl.replace('wss://', 'https://');
    if (!publishUrl.endsWith('/event')) {
      publishUrl = publishUrl.replace(/\/$/, '') + '/event';
    }

    const amzDate = new Date().toISOString().replace(/[:\-]|\.\d{3}/g, '');
    
    let sessionToken = '';
    try {
      const parsedData = JSON.parse(data);
      if (parsedData.sessionToken) {
        sessionToken = parsedData.sessionToken;
        delete parsedData.sessionToken; // Remove from nested data if preferred, or keep it.
        data = JSON.stringify(parsedData);
      }
    } catch (e) {}

    const signal: any = {
      from: this.clientId,
      to: this.config.targetId,
      action: action,
      data: data
    };
    
    if (sessionToken) {
      signal.sessionToken = sessionToken;
    }

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
        throw new Error(`Publish failed: ${response.status}`);
      }
    } catch (err) {
      this.sendErrorToClient(err instanceof Error ? err : new Error('Signal send failed'));
      throw err;
    }
  }

  private handleIncomingSignal(signal: any): void {
    for (const clientID of this.clientIDs) {
      sendClientMessage(clientID, {
        __response__: 40,
        signal: {
          action: signal.action,
          data: signal.data,
          sessionToken: signal.sessionToken
        }
      });
    }
  }

  private sendErrorToClient(err: Error): void {
    for (const clientID of this.clientIDs) {
      sendClientMessage(clientID, {
        __response__: 30,
        error: err.message
      });
    }
  }

  private sendCloseToClient(): void {
    for (const clientID of this.clientIDs) {
      sendClientMessage(clientID, {
        __response__: 30,
        closed: true
      });
    }
  }

  disconnect(): void {
    this.ws?.close();
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  getClientId(): string {
    return this.clientId;
  }

  getTargetId(): string {
    return this.config.targetId;
  }
}

// Handler 30: Connect to AppSync WebSocket
export const connectAppSync = async (config: AppSyncWebRTCConfig): Promise<any> => {
  const targetId = config.targetId;
  const clientID = config.__client__;

  // Store config for possible re-connection needs
  const requiredConfig: Required<AppSyncWebRTCConfig> = {
    wsUrl: config.wsUrl,
    apiKey: config.apiKey,
    targetId: targetId,
    stunServers: config.stunServers || ['stun:stun.l.google.com:19302', 'stun:global.stun.twilio.com:3478'],
    trickle: config.trickle ?? false,
    timeout: config.timeout ?? 30000,
    __client__: clientID
  };
  connectionConfigs.set(clientID, requiredConfig);
  clientTargetMap.set(clientID, targetId);

  let connection = connectionsByTarget.get(targetId);

  if (connection && connection.isConnected()) {
    console.log('[AppSync] Reusing connection for target:', targetId);
    connection.addClient(clientID);
    return { clientId: connection.getClientId(), connected: true };
  }

  if (connection) {
    console.log('[AppSync] Connection exists but not connected, reconnecting...');
    connection.disconnect();
  }

  connection = new AppSyncConnection(requiredConfig);
  connectionsByTarget.set(targetId, connection);

  return await connection.connect();
};

// Handler 31: Send signal via AppSync
export const sendSignalToAppSync = async (args: {
  __client__: number;
  action: string;
  data: string
}): Promise<any> => {
  const targetId = clientTargetMap.get(args.__client__);
  if (!targetId) {
    throw new Error('No target associated with this client. Call connect first.');
  }

  let connection = connectionsByTarget.get(targetId);

  if (!connection || !connection.isConnected()) {
    const config = connectionConfigs.get(args.__client__);
    if (!config) throw new Error('No connection config found');

    connection = new AppSyncConnection(config);
    connectionsByTarget.set(targetId, connection);
    await connection.connect();
  }

  try {
    await connection.sendSignal(args.action, args.data);
  } catch (err) {
    // Retry once on failure
    const config = connectionConfigs.get(args.__client__);
    if (config) {
      connection.disconnect();
      connection = new AppSyncConnection(config);
      connectionsByTarget.set(targetId, connection);
      await connection.connect();
      await connection.sendSignal(args.action, args.data);
    } else {
      throw err;
    }
  }

  return { success: true };
};

// Handler 32: Disconnect from AppSync
export const disconnectAppSync = async (args: { __client__: number }): Promise<any> => {
  const targetId = clientTargetMap.get(args.__client__);
  if (targetId) {
    const connection = connectionsByTarget.get(targetId);
    if (connection) {
      connection.removeClient(args.__client__);
      if (!connection.hasClients()) {
        connection.disconnect();
        connectionsByTarget.delete(targetId);
      }
    }
    clientTargetMap.delete(args.__client__);
  }
  connectionConfigs.delete(args.__client__);
  return { success: true };
};

// Get active connection info
export const getConnectionInfo = (clientID: number): { clientId?: string; targetId?: string } | null => {
  const targetId = clientTargetMap.get(clientID);
  if (!targetId) return null;

  const connection = connectionsByTarget.get(targetId);
  if (!connection) return null;

  return {
    clientId: connection.getClientId(),
    targetId: connection.getTargetId()
  };
};
