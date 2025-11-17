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
 * Recursively finds all HTML files in a given directory.
 * @param {string} dir - The directory to search in.
 * @returns {string[]} An array of HTML file paths.
 */
function findHtmlFiles(dir) {
  const dirents = fs.readdirSync(dir, { withFileTypes: true });
  const files = dirents.map((dirent) => {
    const res = path.resolve(dir, dirent.name);
    return dirent.isDirectory() ? findHtmlFiles(res) : res;
  });
  return Array.prototype.concat(...files).filter((file) => file.endsWith('.html'));
}

/**
 * The main function to process HTML files and inline CSS.
 */
function inlineCss() {
  console.log('--- Starting CSS inlining script ---');

  try {
    const htmlFiles = findHtmlFiles(BUILD_DIR);

    if (htmlFiles.length === 0) {
      console.warn(`No HTML files found in the '${BUILD_DIR}' directory. Exiting.`);
      return;
    }

    console.log(`Found ${htmlFiles.length} HTML file(s) to process.`);

    // A regex to find the SvelteKit-generated CSS link tag.
    // It specifically looks for a link tag pointing to a .css file inside the immutable assets folder.
    const cssLinkRegex = /<link href="(\.?\/?_app\/immutable\/assets\/[^"]+\.css)" rel="stylesheet">/;

    for (const htmlPath of htmlFiles) {
      try {
        // 1. Read the HTML file content
        let htmlContent = fs.readFileSync(htmlPath, 'utf-8');
        const match = htmlContent.match(cssLinkRegex);

        if (match && match[1]) {
          const cssRelativePath = match[1];
          const linkTag = match[0];

          // 2. Construct the full path to the CSS file
          // The path in the href is relative to the HTML file's location.
          const cssFullPath = path.join(path.dirname(htmlPath), cssRelativePath);

          // 3. Read the CSS file content
          let cssContent = fs.readFileSync(cssFullPath, 'utf-8');

          const TAILWIND_BLOCK_REGEX = /\/\*! tailwindcss v4\.1\.12[\s\S]*?--tw-drop-shadow-size:initial\}\}\}/g;
          cssContent = cssContent.replace(TAILWIND_BLOCK_REGEX, '');

          // const SPACING_CALC_REGEX = /calc\(var\(--spacing\)\s*\*\s*([0-9]+(?:\.[0-9]+)?)\)/g;
          //cssContent = cssContent.replace(SPACING_CALC_REGEX, '$1');

          // 4. Create the inline style tag
          const styleTag = `<style>${cssContent}</style>`;

          // 5. Replace the original link tag with the new style tag
          htmlContent = htmlContent.replace(linkTag, styleTag);

          // 6. Write the modified content back to the HTML file
          fs.writeFileSync(htmlPath, htmlContent, 'utf-8');

          console.log(`✅ Inlined CSS for: ${path.basename(htmlPath)} | ${cssRelativePath}`);
        } else {
          console.log(`⏩ Skipped: No SvelteKit CSS link found in ${path.basename(htmlPath)}.`);
        }
      } catch (fileError) {
        console.error(`❌ Error processing file ${htmlPath}:`, fileError);
      }
    }
  } catch (error) {
    console.error('❌ An error occurred during the inlining process:', error);
  }

  console.log('--- CSS inlining script finished ---');
}

/**
 * Recursively copies a directory from source to destination.
 * @param {string} src - The source directory path.
 * @param {string} dest - The destination directory path.
 */
function copyDirectory(src, dest) {
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
function publishToDocs() {
  console.log('--- Starting publish to docs folder ---');
  
  const DOCS_DIR = 'docs';
  
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

inlineCss();
  
// Check if 'publish' parameter is passed
const args = process.argv.slice(2);
console.log("args....", args)
if (args.includes('--publish')) {
  publishToDocs();
}
