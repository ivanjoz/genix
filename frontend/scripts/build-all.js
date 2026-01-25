// scripts/build-all.js
import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const BUILD_DIR = 'build';

console.log('🚀 Starting build process...');

// Clean previous builds
if (fs.existsSync(BUILD_DIR)) {
  fs.rmSync(BUILD_DIR, { recursive: true });
}
fs.mkdirSync(BUILD_DIR, { recursive: true });

// Build main app
console.log('📦 Building main app...');
execSync('bun run build:main', { stdio: 'inherit' });

// Copy main build
console.log('📋 Copying main build...');
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

// SvelteKit builds to .svelte-kit/output/client and .svelte-kit/output/server
// When using adapter-static, it also outputs to the configured 'pages' directory (usually 'build')
copyDirectory('build', BUILD_DIR);

// Build store app
console.log('📦 Building store app...');
execSync('bun run build:store', { stdio: 'inherit' });

// Copy store build to /store/ subdirectory
console.log('📋 Copying store build...');
copyDirectory('pkg-store/build', path.join(BUILD_DIR, 'store'));

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
