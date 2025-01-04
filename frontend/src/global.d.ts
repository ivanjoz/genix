/// <reference types="@solidjs/start/env" />

declare module "*.module.css";
// declare module "*.svg";
declare module "*.svg?raw";
// declare module "*.module.scss";

interface Window {
  appId: string
  CompressionStream: any
  DexieDB: any,
  DexieEcommerceDB: any,
  EvaluacionesDB: any,
  _broadcast: BroadcastChannel
  _pendingRequests: any[],
  _fechaHelper: any,
  _isProd: boolean
  _isLocal: boolean
  _buildInfo: any,
  _modules: any[],
  _cache: { [key: string]: any }
  _params: { [key: string]: any }
  _cacheTiles: { [key: string]: string }
  _throttleTimer?: NodeJS.Timeout
  _fecha0: Date
  _zoneOffset: number
  _dexieVersion: number
  _counterID?: number
  _route?: string
  API_ROUTES: { MAIN: string }
  S3_URL: string
}