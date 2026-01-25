// scripts/proxy-server.js
import http from 'http';
import { createProxyMiddleware } from 'http-proxy-middleware';

const MAIN_PORT = 3570;
const STORE_PORT = 3571;
const PROXY_PORT = 3570; // Use same port as before

// Create proxy middleware for store
const storeProxy = createProxyMiddleware({
  target: `http://localhost:${STORE_PORT}`,
  changeOrigin: true,
  pathRewrite: {
    '^/store': '' // Remove /store prefix when proxying to store app
  },
  ws: true // Enable WebSocket proxying
});

// Create proxy middleware for main
const mainProxy = createProxyMiddleware({
  target: `http://localhost:${MAIN_PORT}`,
  changeOrigin: true,
  ws: true
});

// Create main server
const server = http.createServer((req, res) => {
  console.log(`${req.method} ${req.url} → ${req.url.startsWith('/store') ? 'store' : 'main'}`);
  
  if (req.url.startsWith('/store')) {
    storeProxy(req, res);
  } else {
    mainProxy(req, res);
  }
});

// Handle WebSocket upgrades
server.on('upgrade', (req, socket, head) => {
  if (req.url.startsWith('/store')) {
    storeProxy.upgrade(req, socket, head);
  } else {
    mainProxy.upgrade(req, socket, head);
  }
});

server.listen(3572, () => { // Use 3572 for proxy to avoid conflict during dev
  console.log(`🚀 Development proxy server running on http://localhost:3572`);
  console.log(`📋 Main (Admin): http://localhost:3572`);
  console.log(`🛒 Store: http://localhost:3572/store`);
});
