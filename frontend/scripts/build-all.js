// scripts/build-all.js
import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { setupEnv } from './setup-env.js';

const BUILD_DIR = 'build';

console.log('🚀 Starting build process...');

// 0. Extract credentials and create .env files
setupEnv();

// 1. Build main app (this creates the 'build' directory via SvelteKit adapter-static)
console.log('📦 Building main app...');
execSync('bun run build:main', { stdio: 'inherit' });

// 2. Build store app
console.log('📦 Building store app...');
execSync('bun run build:store', { stdio: 'inherit' });

// 3. Copy store build to /webpage/ subdirectory in the main build
console.log('📋 Copying store build into main build...');
const copyDirectory = (src, dest) => {
  if (!fs.existsSync(src)) {
    console.warn(`⚠️  Source directory ${src} does not exist`);
    return;
  }
  
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });

  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);

    if (entry.isDirectory()) {
      copyDirectory(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
    }
  }
};

copyDirectory('webpage/build', path.join(BUILD_DIR, 'webpage'));


// Create 404.html for SPA routing
console.log('📋 Creating 404.html...');
const indexPath = path.join(BUILD_DIR, 'index.html');
const notFoundPath = path.join(BUILD_DIR, '404.html');

if (fs.existsSync(indexPath)) {
  fs.copyFileSync(indexPath, notFoundPath);
  console.log('✅ Created 404.html from index.html');
} else {
  console.warn('⚠️  index.html not found, skipping 404.html creation');
}

console.log('✅ Build completed successfully!');
console.log(`📁 Output directory: ${BUILD_DIR}`);
