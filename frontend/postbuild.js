// post-build.js
// This script runs after the SvelteKit build to inline CSS into the final HTML files.
import fs from 'fs';
import path from 'path';

// --- CONFIGURATION ---
// The directory where SvelteKit outputs the final static site.
// This is typically 'build' for adapter-static.
const BUILD_DIR = 'build';
// ---------------------

/**
 * Recursively copies a directory from source to destination.
 * @param {string} src - The source directory path.
 * @param {string} dest - The destination directory path.
 */
const copyDirectory = (src, dest) => {
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
}

/**
 * Publishes the build folder to the docs folder for GitHub Pages.
 */
const publishToDocs = () => {
  console.log('--- Starting publish to docs folder ---');
  
  const DOCS_DIR = path.join('..', 'docs');
  
  try {
    // Check if build directory exists
    try {
      fs.accessSync(BUILD_DIR);
    } catch {
      console.error(`❌ Build directory '${BUILD_DIR}' not found. Please build the project first.`);
      return;
    }

    // Remove only directories from docs folder
    try {
      if (fs.existsSync(DOCS_DIR)) {
        const entries = fs.readdirSync(DOCS_DIR, { withFileTypes: true });
        for (const entry of entries) {
          if (entry.isDirectory()) {
            const dirPath = path.join(DOCS_DIR, entry.name);
            fs.rmSync(dirPath, { recursive: true, force: true });
            console.log(`🗑️  Removed directory '${entry.name}' from '${DOCS_DIR}'`);
          }
        }
      } else {
        // Create docs directory if it doesn't exist
        fs.mkdirSync(DOCS_DIR, { recursive: true });
        console.log(`📁 Created '${DOCS_DIR}' directory`);
      }
    } catch (error) {
      console.error(`❌ Error cleaning docs directory:`, error);
    }

    // Copy build folder to docs folder
    copyDirectory(BUILD_DIR, DOCS_DIR);
    console.log(`✅ Copied contents from '${BUILD_DIR}' to '${DOCS_DIR}'`);

    // Copy index.html as 404.html for GitHub Pages SPA routing
    const indexPath = path.join(DOCS_DIR, 'index.html');
    const notFoundPath = path.join(DOCS_DIR, '404.html');
    
    try {
      fs.copyFileSync(indexPath, notFoundPath);
      console.log(`✅ Created 404.html from index.html`);
    } catch (error) {
      console.error(`❌ Error creating 404.html:`, error);
    }

    console.log('--- Publish to docs folder completed ---');
  } catch (error) {
    console.error('❌ An error occurred during publishing:', error);
  }
}

// Check if 'publish' parameter is passed
const args = process.argv.slice(2);
console.log("args....", args)
if (args.includes('--publish')) {
  publishToDocs();
}
