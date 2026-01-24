import fs from 'fs';
import path from 'path';

// Configuration
const SOURCE_DIR = path.resolve(process.cwd(), '../store/static/images');
const DEST_DIR = path.resolve(process.cwd(), 'static/images');

// Files to copy
const FILES_TO_COPY = [
  'casaca_icon_sm3.webp',
  'categoria_1.webp',
  'categoria_11.webp',
  'input.jpeg',
  'nucNFwLCytpb.jpg',
  'placeholder.webp',
  'scripts.txt',
  'thumbhash.txt',
  'uwgGHwD2aYl3.webp'
];

/**
 * Copy a single file from source to destination
 */
function copyFile(filename) {
  const sourcePath = path.join(SOURCE_DIR, filename);
  const destPath = path.join(DEST_DIR, filename);

  // Check if source file exists
  if (!fs.existsSync(sourcePath)) {
    console.warn(`⚠️  Source file not found: ${filename}`);
    return false;
  }

  // Check if destination already exists
  if (fs.existsSync(destPath)) {
    console.log(`⏭️  Skipping (already exists): ${filename}`);
    return false;
  }

  try {
    // Copy the file
    fs.copyFileSync(sourcePath, destPath);
    console.log(`✅ Copied: ${filename}`);
    return true;
  } catch (error) {
    console.error(`❌ Error copying ${filename}:`, error.message);
    return false;
  }
}

/**
 * Main execution
 */
function main() {
  console.log('📦 Bulk Copy Script - Store to Frontend');
  console.log('=========================================\n');
  console.log(`Source: ${SOURCE_DIR}`);
  console.log(`Destination: ${DEST_DIR}\n`);

  // Check if directories exist
  if (!fs.existsSync(SOURCE_DIR)) {
    console.error(`❌ Source directory not found: ${SOURCE_DIR}`);
    process.exit(1);
  }

  if (!fs.existsSync(DEST_DIR)) {
    console.error(`❌ Destination directory not found: ${DEST_DIR}`);
    process.exit(1);
  }

  // Copy files
  let copiedCount = 0;
  let skippedCount = 0;
  let errorCount = 0;

  for (const filename of FILES_TO_COPY) {
    const result = copyFile(filename);
    if (result === null) {
      errorCount++;
    } else if (result === true) {
      copiedCount++;
    } else {
      skippedCount++;
    }
  }

  // Summary
  console.log('\n=========================================');
  console.log('📊 Summary:');
  console.log(`   ✅ Copied: ${copiedCount} files`);
  console.log(`   ⏭️  Skipped: ${skippedCount} files`);
  console.log(`   ❌ Errors: ${errorCount} files`);
  console.log('=========================================');

  if (errorCount > 0) {
    process.exit(1);
  }
}

// Run the script
main();