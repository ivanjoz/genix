import http from 'http';
import httpProxy from 'http-proxy';

const MAIN_PORT = 3570;
const STORE_PORT = 3571;
const PROXY_PORT = 3572;

// Parse only the pathname so query strings like /store?p=... still route to Store.
const getRequestPathname = (requestUrl = '/') => {
  try {
    return new URL(requestUrl, 'http://localhost').pathname;
  } catch {
    return requestUrl.split('?')[0] || '/';
  }
};

// Match all Store entry points: /store, /store/, and any nested path under /store.
const isStoreRequest = (requestUrl = '/') => {
  const requestPathname = getRequestPathname(requestUrl);
  return requestPathname === '/store' || requestPathname.startsWith('/store/');
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

// Handle proxy errors
mainProxy.on('error', (err, req, res) => {
  console.error('[Proxy Error] Main:', err.message);
  if (!res.headersSent) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end('Proxy error: ' + err.message);
  }
});

storeProxy.on('error', (err, req, res) => {
  console.error('[Proxy Error] Store:', err.message);
  if (!res.headersSent) {
    res.writeHead(500, { 'Content-Type': 'text/plain' });
    res.end('Proxy error: ' + err.message);
  }
});

// Log proxy requests
mainProxy.on('proxyReq', (proxyReq, req, res) => {
  console.log(`[Proxy] Main: ${req.method} ${req.url}`);
});

storeProxy.on('proxyReq', (proxyReq, req, res) => {
  console.log(`[Proxy] Store: ${req.method} ${req.url}`);
});

// Log WebSocket upgrades
mainProxy.on('proxyReqWs', (proxyReq, req, socket, options, head) => {
  console.log(`[WS] Main: ${req.url}`);
});

storeProxy.on('proxyReqWs', (proxyReq, req, socket, options, head) => {
  console.log(`[WS] Store: ${req.url}`);
});

// Handle proxy errors on WebSocket connections
mainProxy.on('proxyErrorWs', (err, req, socket) => {
  console.error('[WS Error] Main:', err.message);
  socket.end();
});

storeProxy.on('proxyErrorWs', (err, req, socket) => {
  console.error('[WS Error] Store:', err.message);
  socket.end();
});

// Create main HTTP server
const server = http.createServer((req, res) => {
  const isStore = isStoreRequest(req.url);
  const isServiceWorkerComm = req.url.startsWith('/_sw_');
  
  if (isStore) {
    console.log(`[HTTP] ${req.url} → Store`);
    storeProxy.web(req, res);
  } else if (isServiceWorkerComm) {
    // Handle Service Worker communication requests locally to avoid 404s in Main
    console.log(`[HTTP] ${req.url} → Handled by Proxy (SW communication)`);
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ok: 1 }));
  } else {
    console.log(`[HTTP] ${req.url} → Main`);
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
  console.log(`  🛒 Store:          http://localhost:${PROXY_PORT}/store`);
  console.log(`\n  🔧 Main Target:    http://localhost:${MAIN_PORT}`);
  console.log(`  🔧 Store Target:   http://localhost:${STORE_PORT}`);
  console.log('\n  Proxying HTTP requests and WebSocket connections...\n');
});
