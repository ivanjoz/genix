// postbuild.js
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const BUILD_DIR = 'build';
const STATIC_DIR = 'static';
const DOCS_DIR = path.join('..', 'docs');

/**
 * Zips only the bundled assets from the build folder, excluding static assets
 * that are already tracked in the repository.
 */
const zipBundledAssets = () => {
  console.log('--- Starting zip of bundled assets ---');
  
  const ZIP_FILE_NAME = 'frontend.zip';
  const ZIP_PATH = path.join(DOCS_DIR, ZIP_FILE_NAME);
  
  try {
    if (!fs.existsSync(BUILD_DIR)) {
      console.error(`❌ Build directory '${BUILD_DIR}' not found. Please build the project first.`);
      return;
    }

    if (!fs.existsSync(DOCS_DIR)) {
      fs.mkdirSync(DOCS_DIR, { recursive: true });
    }

    // Create serve.json inside build folder for local 'serve' command and SPA routing
    const serveConfig = {
      cleanUrls: true,
      rewrites: [
        { source: "/store", destination: "/store/index.html" },
        { source: "/store/**", destination: "/store/index.html" },
        { source: "**", destination: "/index.html" }
      ]
    };
    fs.writeFileSync(path.join(BUILD_DIR, 'serve.json'), JSON.stringify(serveConfig, null, 2));
    console.log(`✅ Created serve.json inside build folder`);

    // Get list of items in static folder to exclude them from zip
    // because they are already in the repo
    const staticItems = fs.readdirSync(STATIC_DIR);
    
    // We want to exclude these items when zipping the build folder
    // Note: build/ folder contains everything from static/ plus bundled assets
    const excludeArgs = staticItems.map(item => {
      // If it's a directory, we need to exclude its contents too
      if (fs.statSync(path.join(STATIC_DIR, item)).isDirectory()) {
        return `-x "${item}/*"`;
      }
      return `-x "${item}"`;
    }).join(' ');

    console.log(`📦 Zipping bundled assets to '${ZIP_PATH}'...`);
    
    // Remove old zip if exists
    if (fs.existsSync(ZIP_PATH)) {
      fs.unlinkSync(ZIP_PATH);
    }
    
    // Use system zip command
    // We cd into BUILD_DIR so the zip contains paths relative to it
    const command = `cd ${BUILD_DIR} && zip -r9 "../${ZIP_PATH}" . ${excludeArgs}`;
    console.log(`Executing: ${command}`);
    
    execSync(command, { stdio: 'inherit' });
    console.log(`✅ Created '${ZIP_PATH}'`);

    console.log('--- Zip process completed ---');
  } catch (error) {
    console.error('❌ An error occurred during zipping:', error);
  }
}

const args = process.argv.slice(2);
if (args.includes('--publish')) {
  zipBundledAssets();
}