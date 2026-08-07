// scripts/dev-all.js
import { spawn } from 'child_process';
import net from 'net';
import path from 'path';
import { setupEnv } from './setup-env.js';

const MAIN_PORT = 3570;
const STORE_PORT = 3571;
// Keep readiness checks and displayed URLs aligned with the proxy process.
const PROXY_PORT = Number.parseInt(process.env.GENIX_PROXY_PORT || '3572', 10);
const READY_TIMEOUT_MS = 120000;

const spawnService = (label, command, cwd) => {
  const child = spawn('bun', ['run', ...command.split(' ')], {
    cwd: path.resolve(process.cwd(), cwd),
    stdio: 'inherit',
    shell: true
  });
  child.on('error', (error) => {
    console.error(`❌ ${label} failed to spawn:`, error.message);
  });
  return child;
};

// Se prueban las dos familias: Vite escucha solo en [::1] y el proxy (node http) en
// 0.0.0.0, así que fijar una sola daría ECONNREFUSED para siempre en uno de los dos.
const LOOPBACK_HOSTS = ['::1', '127.0.0.1'];

const tryConnect = (port, host) => new Promise((resolve) => {
  const socket = net.connect({ port, host });
  const done = (ok) => { socket.destroy(); resolve(ok) };
  socket.once('connect', () => done(true));
  socket.once('error', () => done(false));
});

// TCP connect is the readiness signal: Vite and the proxy bind their port only once
// they can actually serve, so a successful connect means ready. Polling this instead
// of sleeping on a fixed timer is what keeps startup at the real cost of the servers.
const waitForPort = async (port, timeoutMs = READY_TIMEOUT_MS) => {
  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    const results = await Promise.all(LOOPBACK_HOSTS.map(host => tryConnect(port, host)));
    if (results.some(Boolean)) { return true }
    await new Promise(resolve => setTimeout(resolve, 50));
  }

  return false;
};

const main = async () => {
  const startedAt = Date.now();
  console.log('🚀 Starting development environment...');

  // Setup environment variables
  setupEnv();

  // All three start at once. The proxy handles requests to targets that are not up
  // yet (see the 'error' handlers in proxy-server.js), so there is nothing to order.
  console.log('📋 Starting main app, 🛒 store app and 🔗 proxy...');
  const mainApp = spawnService('main app', 'dev:main', '.');
  const storeApp = spawnService('store app', 'dev', 'webpage');
  const proxy = spawn('bun', ['scripts/proxy-server.js'], {
    cwd: process.cwd(),
    stdio: 'inherit',
    shell: true
  });
  proxy.on('error', (error) => console.error('❌ proxy failed to spawn:', error.message));

  // Registered before the await so Ctrl+C during startup still tears everything down.
  const cleanup = () => {
    console.log('\n🛑 Shutting down services...');
    mainApp.kill();
    storeApp.kill();
    proxy.kill();
    process.exit(0);
  };
  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);

  const [mainReady, storeReady, proxyReady] = await Promise.all([
    waitForPort(MAIN_PORT),
    waitForPort(STORE_PORT),
    waitForPort(PROXY_PORT)
  ]);

  const elapsed = ((Date.now() - startedAt) / 1000).toFixed(2);
  const status = (ready, port) => ready ? `ready (:${port})` : `NOT ready (:${port}) ⚠️`;

  if (mainReady && storeReady && proxyReady) {
    console.log(`✅ All services started successfully in ${elapsed}s!`);
  } else {
    console.log(`⚠️ Some services did not come up within ${READY_TIMEOUT_MS / 1000}s (${elapsed}s elapsed):`);
  }
  console.log(`   📋 Main app  ${status(mainReady, MAIN_PORT)}`);
  console.log(`   🛒 Store app ${status(storeReady, STORE_PORT)}`);
  console.log(`   🔗 Proxy     ${status(proxyReady, PROXY_PORT)}`);
  console.log(`📋 Main (Admin): http://localhost:${PROXY_PORT}`);
  console.log(`🛒 Store: http://localhost:${PROXY_PORT}/webpage-app`);
  console.log('\n💡 Tips:');
  console.log('   - Main app runs internally on port 3570');
  console.log('   - Store app runs internally on port 3571');
  console.log(`   - Proxy server routes requests appropriately on port ${PROXY_PORT}`);
  console.log('   - Ctrl+C to stop all services');
};

main().catch((error) => {
  console.error('❌ Failed to start development environment:', error);
  process.exit(1);
});
