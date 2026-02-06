// Type declarations for simple-peer library

export interface SignalData {
  type?: 'offer' | 'answer' | 'candidate' | 'pranswer' | 'rollback';
  sdp?: string;
  candidate?: string;
  sdpMLineIndex?: number;
  sdpMid?: string;
}

export interface PeerOptions {
  initiator?: boolean;
  channelConfig?: RTCDataChannelInit;
  channelName?: string;
  config?: RTCConfiguration;
  offerOptions?: RTCOfferOptions;
  answerOptions?: RTCAnswerOptions;
  sdpTransform?: (sdp: string) => string;
  stream?: MediaStream;
  streams?: MediaStream[];
  trickle?: boolean;
  allowHalfTrickle?: boolean;
  iceCompleteTimeout?: number;
}

export interface PeerInstance extends NodeJS.EventEmitter {
  connected: boolean;
  destroyed: boolean;
  
  signal(data: SignalData | string): void;
  send(data: any): void;
  addStream(stream: MediaStream): void;
  removeStream(stream: MediaStream): void;
  addTrack(track: MediaStreamTrack, stream: MediaStream): void;
  removeTrack(track: MediaStreamTrack, stream: MediaStream): void;
  destroy(err?: Error): void;
  
  on(event: 'signal', callback: (data: SignalData) => void): this;
  on(event: 'connect', callback: () => void): this;
  on(event: 'close', callback: () => void): this;
  on(event: 'error', callback: (err: Error) => void): this;
  on(event: 'data', callback: (data: any) => void): this;
  on(event: 'stream', callback: (stream: MediaStream) => void): this;
  on(event: 'track', callback: (track: MediaStreamTrack, stream: MediaStream) => void): this;
  
  once(event: 'signal', callback: (data: SignalData) => void): this;
  once(event: 'connect', callback: () => void): this;
  once(event: 'close', callback: () => void): this;
  once(event: 'error', callback: (err: Error) => void): this;
  once(event: 'data', callback: (data: any) => void): this;
  once(event: 'stream', callback: (stream: MediaStream) => void): this;
  once(event: 'track', callback: (track: MediaStreamTrack, stream: MediaStream) => void): this;
  
  off(event: 'signal', callback: (data: SignalData) => void): this;
  off(event: 'connect', callback: () => void): this;
  off(event: 'close', callback: () => void): this;
  off(event: 'error', callback: (err: Error) => void): this;
  off(event: 'data', callback: (data: any) => void): this;
  off(event: 'stream', callback: (stream: MediaStream) => void): this;
  off(event: 'track', callback: (track: MediaStreamTrack, stream: MediaStream) => void): this;
}

declare const Peer: {
  new (opts?: PeerOptions): PeerInstance;
};

export default Peer;