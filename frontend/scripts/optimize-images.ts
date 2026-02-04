import fs from 'fs';
import path from 'path';
import { spawnSync } from 'child_process';
import sharp from 'sharp';

const IMAGES_DIR = path.resolve(process.cwd(), 'static/store-images');
const MAX_PIXELS = 1400000; // 1.4 Megapixels
const MAX_FILE_SIZE = 200 * 1024; // 200 KB
const INITIAL_QUALITY = 80;
const MIN_QUALITY = 65; // Quality floor
const FORCE = process.argv.includes('--force');

function checkBinary(cmd: string): boolean {
  try {
    const result = spawnSync('which', [cmd], { encoding: 'utf8' });
    return result.status === 0;
  } catch (e) {
    return false;
  }
}

async function optimizeImages() {
  const isLinux = process.platform === 'linux';
  const hasMagick = checkBinary('magick') || checkBinary('convert');
  const magickCmd = checkBinary('magick') ? 'magick' : 'convert';

  console.log(`🚀 Starting high-effort optimization in ${IMAGES_DIR}...`);
  console.log(`⚖️  Target: < 200KB (or Quality >= 65) | Density: 1.4MP | Format: AVIF`);
  if (FORCE) console.log(`🔥 Force mode enabled: Re-processing all files.`);

  if (!fs.existsSync(IMAGES_DIR)) {
    console.error(`❌ Directory not found: ${IMAGES_DIR}`);
    return;
  }

  const files = fs.readdirSync(IMAGES_DIR);
  let processed = 0;
  let skipped = 0;

  for (const file of files) {
    const ext = path.extname(file).toLowerCase();
    if (['.avif', '.txt', '.kra', '.map'].includes(ext)) continue;

    const filePath = path.join(IMAGES_DIR, file);
    const avifPath = path.join(IMAGES_DIR, `${path.parse(file).name}.avif`);

    if (!FORCE && fs.existsSync(avifPath)) {
      const size = fs.statSync(avifPath).size;
      if (size <= MAX_FILE_SIZE) {
        console.log(`| Skipping ${file} (Already optimized: ${(size / 1024).toFixed(1)} KB)`);
        skipped++;
        continue;
      }
    }

    try {
      let currentQuality = INITIAL_QUALITY;
      let finished = false;

      while (!finished) {
        if (isLinux && hasMagick) {
          spawnSync(magickCmd, [
            filePath,
            '-resize', `${MAX_PIXELS}@>`,
            '-define', 'heic:speed=0',
            '-quality', String(currentQuality),
            avifPath
          ]);
        } else {
          const image = sharp(filePath);
          const metadata = await image.metadata();
          let processor = image;

          if (metadata.width && metadata.height && (metadata.width * metadata.height > MAX_PIXELS)) {
            const ratio = Math.sqrt(MAX_PIXELS / (metadata.width * metadata.height));
            processor = image.resize(Math.floor(metadata.width * ratio), Math.floor(metadata.height * ratio), { fit: 'inside' });
          }

          await processor.avif({ quality: currentQuality, effort: 9 }).toFile(avifPath);
        }

        const newSize = fs.statSync(avifPath).size;
        
        // Conditions to stop: 
        // 1. We are under 200KB
        // 2. We reached the quality floor (MIN_QUALITY)
        if (newSize <= MAX_FILE_SIZE || currentQuality <= MIN_QUALITY) {
          console.log(`✨ [${isLinux && hasMagick ? 'Magick' : 'Sharp'}] Processed ${file} -> ${(newSize / 1024).toFixed(1)} KB (Q: ${currentQuality}${newSize > MAX_FILE_SIZE ? ' - Limit exceeded to preserve quality' : ''})`);
          finished = true;
        } else {
          console.log(`  ⏳ File too large (${(newSize / 1024).toFixed(1)} KB), reducing quality... (Current: ${currentQuality})`);
          currentQuality -= 5;
          if (currentQuality < MIN_QUALITY) currentQuality = MIN_QUALITY;
        }
      }

      processed++;
    } catch (error) {
      console.error(`❌ Error processing ${file}:`, error);
    }
  }

  console.log(`
🏁 Done! Processed: ${processed}, Skipped: ${skipped}`);
}

optimizeImages().catch(console.error);