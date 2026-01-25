// TODO: This script is not yet integrated into the build process.
// To enable thumbhash generation:
// 1. Add "prebuild:store": "node scripts/thumbhash-prebuild.js" to package.json scripts
// 2. Update build process to run prebuild:store before building the store
// 3. See STORE.md for more implementation details

import fs from "fs";
import { console } from "inspector";
import path from "path";
import sharp from "sharp";
import { rgbaToThumbHash, thumbHashToDataURL } from "thumbhash";

const IMAGES_DIR = "./static/images";
const THUMBHASH_FILE = "thumbhash.txt";

// Supported image formats
const IMAGE_EXTENSIONS = [".jpg", ".jpeg", ".png", ".webp", ".gif"];

async function getImageRGBA(imagePath) {
  try {
    const image = sharp(imagePath);
    const { width: originalWidth, height: originalHeight } =
      await image.metadata();

    // Calculate dimensions to fit within 100x100 while maintaining aspect ratio
    const maxSize = 100;
    let newWidth = originalWidth;
    let newHeight = originalHeight;

    if (originalWidth > maxSize || originalHeight > maxSize) {
      const aspectRatio = originalWidth / originalHeight;

      if (originalWidth > originalHeight) {
        newWidth = maxSize;
        newHeight = Math.round(maxSize / aspectRatio);
      } else {
        newHeight = maxSize;
        newWidth = Math.round(maxSize * aspectRatio);
      }
    }

    console.log(
      `  Resizing from ${originalWidth}x${originalHeight} to ${newWidth}x${newHeight}`,
    );

    // Resize and get raw RGBA buffer
    const { data, info } = await image
      .resize(newWidth, newHeight, {
        fit: "inside",
        withoutEnlargement: false,
      })
      .ensureAlpha()
      .raw()
      .toBuffer({ resolveWithObject: true });

    return {
      rgba: new Uint8Array(data),
      width: info.width,
      height: info.height,
    };
  } catch (error) {
    console.error(`Error processing image ${imagePath}:`, error);
    return null;
  }
}

async function generateThumbhash(imagePath) {
  const imageData = await getImageRGBA(imagePath);
  if (!imageData) return null;

  const { rgba, width, height } = imageData;
  const thumbhash = rgbaToThumbHash(width, height, rgba);

  // Convert to base64
  const base64 = Buffer.from(thumbhash).toString("base64");
  return base64;
}

function getFilenameSuffix(base64Hash) {
  return base64Hash.replaceAll("/", "_").replaceAll("=", "-").slice(0, 12);
}

function isImageFile(filename) {
  const ext = path.extname(filename).toLowerCase();
  return IMAGE_EXTENSIONS.includes(ext);
}

async function processImages() {
  const imagesPath = path.resolve(IMAGES_DIR);
  const thumbhashFilePath = path.join(imagesPath, THUMBHASH_FILE);

  // Check if images directory exists
  if (!fs.existsSync(imagesPath)) {
    console.error(`Images directory not found: ${imagesPath}`);
    process.exit(1);
  }

  // Get all image files
  const files = fs.readdirSync(imagesPath);
  const imageFiles = files.filter(isImageFile);

  if (imageFiles.length === 0) {
    console.log("No image files found in the images directory.");
    return;
  }

  console.log(`Found ${imageFiles.length} image files to process...`);

  const thumbhashes = [];
  const processedFiles = [];

  for (const filename of imageFiles) {
    const originalPath = path.join(imagesPath, filename);

    console.log(`Processing: ${filename}`);

    try {
      // Generate thumbhash
      const thumbhash = await generateThumbhash(originalPath);

      if (!thumbhash) {
        console.error(`Failed to generate thumbhash for: ${filename}`);
        continue;
      }

      verifyThumbhash(thumbhash);

      // Get last 12 characters for new filename
      const suffix = getFilenameSuffix(thumbhash);
      const extension = path.extname(filename);
      const newFilename = `${suffix}${extension}`;
      const newPath = path.join(imagesPath, newFilename);

      // Check if new filename already exists
      if (fs.existsSync(newPath) && newPath !== originalPath) {
        console.warn(`File already exists: ${newFilename}, skipping rename...`);
      } else if (newPath !== originalPath) {
        // Rename the file
        fs.renameSync(originalPath, newPath);
        console.log(`Renamed: ${filename} -> ${newFilename}`);
      }

      // Store thumbhash data
      thumbhashes.push(`${thumbhash}`);
      processedFiles.push({
        original: filename,
        renamed: newFilename,
        thumbhash: thumbhash,
      });
    } catch (error) {
      console.error(`Error processing ${filename}:`, error);
    }
  }

  // Write thumbhashes to file
  if (thumbhashes.length > 0) {
    const content = thumbhashes.join("\n");
    fs.writeFileSync(thumbhashFilePath, content, "utf8");
    console.log(`\nThumbhashes written to: ${thumbhashFilePath}`);

    // Summary
    console.log(`\nSummary:`);
    console.log(`- Processed: ${processedFiles.length} images`);
    console.log(`- Thumbhash file: ${THUMBHASH_FILE}`);
    console.log(`- Location: ${imagesPath}`);
  } else {
    console.log("No images were processed successfully.");
  }
}

function verifyThumbhash(base64Hash) {
  try {
    const buffer = Buffer.from(base64Hash, "base64");
    const dataURL = thumbHashToDataURL(buffer);
    // console.log("Generando preview de:", base64Hash);
    // console.log(dataURL);
    return dataURL !== null;
  } catch (error) {
    return false;
  }
}

// Run the script
// processImages().catch(console.error);

export { processImages, generateThumbhash, getFilenameSuffix };
