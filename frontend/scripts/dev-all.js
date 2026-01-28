// scripts/dev-all.js
import { spawn } from 'child_process';
import path from 'path';

const startApp = (app, command, cwd) => {
  return new Promise((resolve, reject) => {
    const server = spawn('bun', ['run', ...command.split(' ')], {
      cwd: path.resolve(process.cwd(), cwd),
      stdio: 'inherit',
      shell: true
    });

    server.on('error', reject);

    // Wait for server to be ready
    setTimeout(() => resolve(server), 5000);
  });
};

const main = async () => {
  console.log('🚀 Starting development environment...');

  // Start main app (without store routes)
  console.log('📋 Starting main app...');
  const mainApp = await startApp('main', 'dev:main', '.');

  // Wait a bit
  await new Promise(resolve => setTimeout(resolve, 3000));

  // Start store app
  console.log('🛒 Starting store app...');
  const storeApp = await startApp('store', 'dev', 'pkg-store');

  // Wait a bit
  await new Promise(resolve => setTimeout(resolve, 2000));

  // Start proxy server
  console.log('🔗 Starting proxy server...');
  const proxy = spawn('bun', ['scripts/proxy-server.js'], {
    cwd: process.cwd(),
    stdio: 'inherit',
    shell: true
  });

  console.log('✅ All services started successfully!');
  console.log('📋 Main (Admin): http://localhost:3572');
  console.log('🛒 Store: http://localhost:3572/store');
  console.log('\n💡 Tips:');
  console.log('   - Main app runs internally on port 3570');
  console.log('   - Store app runs internally on port 3571');
  console.log('   - Proxy server routes requests appropriately on port 3572');
  console.log('   - Ctrl+C to stop all services');

  // Handle cleanup
  const cleanup = () => {
    console.log('\n🛑 Shutting down services...');
    mainApp.kill();
    storeApp.kill();
    proxy.kill();
    process.exit(0);
  };

  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);
};

main().catch((error) => {
  console.error('❌ Failed to start development environment:', error);
  process.exit(1);
});
