import http from 'http';
import httpProxy from 'http-proxy';

const MAIN_PORT = 3570;
const STORE_PORT = 3571;
// Launcher variants can select a separate public entry point.
const PROXY_PORT = Number.parseInt(process.env.GENIX_PROXY_PORT || '3572', 10);

// Parse only the pathname so query strings like /webpage-app?p=... still route to Store.
const getRequestPathname = (requestUrl = '/') => {
  try {
    return new URL(requestUrl, 'http://localhost').pathname;
  } catch {
    return requestUrl.split('?')[0] || '/';
  }
};

// Match all Store entry points: /webpage-app, /webpage-app/, and any nested path under it.
// NOTE: this MUST differ from the admin app's shared-source URL prefix (/webpage/*, which
// Vite emits for the frontend/webpage/ folder), or the proxy misroutes the admin app's own
// modules to the Store server and the page loads two Svelte instances (effect_orphan).
const isStoreRequest = (requestUrl = '/') => {
  const requestPathname = getRequestPathname(requestUrl);
  return requestPathname === '/webpage-app' || requestPathname.startsWith('/webpage-app/');
};

// Create proxy instances for each target
const mainProxy = httpProxy.createProxyServer({
  target: `http://localhost:${MAIN_PORT}`,
  ws: true,
  changeOrigin: true
});

const storeProxy = httpProxy.createProxyServer({
  target: `http://localhost:${STORE_PORT}`,
  ws: true,
  changeOrigin: true
});

// http-proxy 1.18 emits a single 'error' event for both HTTP and WebSocket failures.
// On an upgrade the third argument is the raw socket, not a ServerResponse, so anything
// that calls res.writeHead() there throws and takes the proxy process down.
const handleProxyError = (label) => (err, req, resOrSocket) => {
  const isWebSocket = typeof resOrSocket?.writeHead !== 'function';
  // Bun leaves .message empty on connection refusals, so fall back to the errno code.
  const reason = err.message || err.code || 'unknown error';
  console.error(`[Proxy Error] ${label}${isWebSocket ? ' (ws)' : ''}: ${reason} — ${req.url}`);

  if (isWebSocket) {
    resOrSocket?.destroy?.();
    return;
  }
  if (!resOrSocket.headersSent) {
    resOrSocket.writeHead(500, { 'Content-Type': 'text/plain' });
  }
  resOrSocket.end('Proxy error: ' + reason);
};

mainProxy.on('error', handleProxyError('Main'));
storeProxy.on('error', handleProxyError('Store'));

// Log WebSocket upgrades
mainProxy.on('proxyReqWs', (proxyReq, req, socket, options, head) => {
  console.log(`[WS] Main: ${req.url}`);
});

storeProxy.on('proxyReqWs', (proxyReq, req, socket, options, head) => {
  console.log(`[WS] Store: ${req.url}`);
});

// Create main HTTP server
const server = http.createServer((req, res) => {
  const isStore = isStoreRequest(req.url);
  const isServiceWorkerComm = req.url.startsWith('/_sw_');
  
  if (isStore) {
    // console.log(`[HTTP] ${req.url} → Store`);
    storeProxy.web(req, res);
  } else if (isServiceWorkerComm) {
    // Handle Service Worker communication requests locally to avoid 404s in Main
    // console.log(`[HTTP] ${req.url} → Handled by Proxy (SW communication)`);
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ok: 1 }));
  } else {
     // console.log(`[HTTP] ${req.url} → Main`);
    mainProxy.web(req, res);
  }
});

// Handle WebSocket upgrade requests
server.on('upgrade', (req, socket, head) => {
  const isStore = isStoreRequest(req.url);
  
  if (isStore) {
    console.log(`[WS Upgrade] ${req.url} → Store`);
    storeProxy.ws(req, socket, head);
  } else {
    console.log(`[WS Upgrade] ${req.url} → Main`);
    mainProxy.ws(req, socket, head);
  }
});

// Start the proxy server
server.listen(PROXY_PORT, () => {
  console.log('╔════════════════════════════════════════════════════════════╗');
  console.log('║           🚀 Development Proxy Server Running               ║');
  console.log('╚════════════════════════════════════════════════════════════╝');
  console.log(`\n  📦 Main (Admin):    http://localhost:${PROXY_PORT}/`);
  console.log(`  🛒 Store:          http://localhost:${PROXY_PORT}/webpage-app`);
  console.log(`\n  🔧 Main Target:    http://localhost:${MAIN_PORT}`);
  console.log(`  🔧 Store Target:   http://localhost:${STORE_PORT}`);
  console.log('\n  Proxying HTTP requests and WebSocket connections...\n');
});
