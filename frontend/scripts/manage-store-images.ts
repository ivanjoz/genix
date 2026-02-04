import fs from 'fs';
import path from 'path';

const IMAGES_DIR = path.resolve(process.cwd(), 'static/store-images');
const LIST_FILE = path.join(IMAGES_DIR, 'IMAGES_LIST.md');

/**
 * --get: Finds one .avif file in the directory that isn't listed in IMAGES_LIST.md
 */
function getUnlistedImage() {
  if (!fs.existsSync(IMAGES_DIR)) {
    console.error(`❌ Directory not found: ${IMAGES_DIR}`);
    process.exit(1);
  }

  const files = fs.readdirSync(IMAGES_DIR).filter(f => f.endsWith('.avif'));
  
  let listContent = '';
  if (fs.existsSync(LIST_FILE)) {
    listContent = fs.readFileSync(LIST_FILE, 'utf-8');
  } else {
    fs.writeFileSync(LIST_FILE, '# Ecommerce Images List\n\n| Name | Description | Elements | Dominant Colors | Background | Aspect Ratio | Lighting |\n|------|-------------|----------|-----------------|------------|--------------|----------|\n');
  }

  for (const file of files) {
    const escapedFile = file.replace(/[.*+?^${}()|[\\]/g, '\\$&');
    const regex = new RegExp(`\\|\\s*${escapedFile}\\s*\\|`);
    
    if (!regex.test(listContent)) {
      console.log(file);
      return;
    }
  }
  
  console.error('No unlisted images found.');
  process.exit(0);
}

/**
 * --put: Renames the file (handling collisions) and appends info to markdown
 */
function putImageInfo() {
  const args = process.argv;
  const getArg = (key: string) => {
    const idx = args.indexOf(key);
    return (idx !== -1 && args[idx + 1]) ? args[idx + 1] : '';
  };

  const oldName = getArg('--old');
  let newName = getArg('--new');
  const desc = getArg('--desc');
  const elements = getArg('--elements');
  const colors = getArg('--colors');
  const background = getArg('--bg');
  const ratio = getArg('--ratio');
  const lighting = getArg('--lighting');

  if (!oldName || !newName) {
    console.error('❌ Missing required arguments --old and --new');
    process.exit(1);
  }

  const oldParsed = path.parse(oldName);
  let newParsed = path.parse(newName);
  
  // Ensure we are working with .avif for the target collision check
  if (newParsed.ext !== '.avif') {
    newName = newParsed.name + '.avif';
    newParsed = path.parse(newName);
  }

  // 1. Handle Name Collisions for .avif
  let finalNewBaseName = newParsed.name;
  let counter = 1;
  while (fs.existsSync(path.join(IMAGES_DIR, `${finalNewBaseName}.avif`))) {
    // If the file exists and it's NOT the one we are currently renaming (in case of double run)
    if (oldName === `${finalNewBaseName}.avif`) break;
    
    finalNewBaseName = `${newParsed.name}_${counter}`;
    counter++;
  }
  
  const finalAvifName = `${finalNewBaseName}.avif`;
  const oldAvifPath = path.join(IMAGES_DIR, oldName);
  const newAvifPath = path.join(IMAGES_DIR, finalAvifName);

  // 2. Rename AVIF
  if (fs.existsSync(oldAvifPath)) {
    try {
      fs.renameSync(oldAvifPath, newAvifPath);
      console.log(`✅ Renamed AVIF: ${oldName} -> ${finalAvifName}`);
    } catch (e) {
      console.error(`❌ Error renaming AVIF: ${e}`);
      process.exit(1);
    }
  } else {
    console.error(`❌ Original AVIF file ${oldName} not found`);
    process.exit(1);
  }

  // 3. Rename Base Images (jpg, png, etc.)
  const possibleExtensions = ['.jpg', '.jpeg', '.png', '.webp', '.kra'];
  for (const ext of possibleExtensions) {
    const oldBasePath = path.join(IMAGES_DIR, oldParsed.name + ext);
    if (fs.existsSync(oldBasePath)) {
      const newBasePath = path.join(IMAGES_DIR, finalNewBaseName + ext);
      try {
        fs.renameSync(oldBasePath, newBasePath);
        console.log(`✅ Renamed Base Image: ${oldParsed.name + ext} -> ${finalNewBaseName + ext}`);
      } catch (e) {
        console.warn(`⚠️  Could not rename base image ${ext}: ${e}`);
      }
    }
  }

  // 4. Update IMAGES_LIST.md
  const clean = (s: string) => s.replace(/\|/g, '\\|').trim();
  const row = `| ${clean(finalAvifName)} | ${clean(desc)} | ${clean(elements)} | ${clean(colors)} | ${clean(background)} | ${clean(ratio)} | ${clean(lighting)} |\n`;
  
  try {
    let content = '';
    if (fs.existsSync(LIST_FILE)) {
      content = fs.readFileSync(LIST_FILE, 'utf-8');
    }
    if (content && !content.endsWith('\n')) {
      content += '\n';
    }
    fs.writeFileSync(LIST_FILE, content + row);
    console.log(`📝 Documentation updated in ${LIST_FILE}`);
  } catch (e) {
    console.error(`❌ Error updating ${LIST_FILE}: ${e}`);
    process.exit(1);
  }
}

const args = process.argv;
if (args.includes('--get')) {
  getUnlistedImage();
} else if (args.includes('--put')) {
  putImageInfo();
}
