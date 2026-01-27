/// <reference types="vite/client" />

/**
 * Build-time constants injected by Vite
 * These are defined in vite.config.ts using the 'define' option
 */

// Signaling endpoint for WebRTC P2P connection
// This is read from credentials.json during the build process
declare const __SIGNALING_ENDPOINT__: string;