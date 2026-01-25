
import * as fs from 'fs';
import * as path from 'path';

const PROJECT_ROOT = process.cwd();

// This is a minimal version of the fixer's indexing logic to debug VTable
function findFilesRecursively(dir: string, exts: string[]): string[] {
  let results: string[] = [];
  const list = fs.readdirSync(dir);
  list.forEach(file => {
    file = path.join(dir, file);
    const stat = fs.statSync(file);
    if (stat && stat.isDirectory()) {
      results = results.concat(findFilesRecursively(file, exts));
    } else {
      if (exts.includes(path.extname(file))) {
        results.push(file);
      }
    }
  });
  return results;
}

function extractExports(filePath: string) {
  const exports = [];
  const content = fs.readFileSync(filePath, 'utf-8');
  
  if (filePath.endsWith('.svelte')) {
    const componentName = path.basename(filePath, '.svelte');
    exports.push({ name: componentName, filePath });
  }
  
  const ext = path.extname(filePath);
  const fileName = path.basename(filePath);
  if (fileName === `index${ext}`) {
    const dirPath = path.dirname(filePath);
    const dirFiles = fs.readdirSync(dirPath);
    for (const dirFile of dirFiles) {
      if (dirFile.startsWith('.') || dirFile === fileName) continue;
      const dirFilePath = path.join(dirPath, dirFile);
      if (fs.statSync(dirFilePath).isFile()) {
        const baseName = path.basename(dirFile, path.extname(dirFile));
        exports.push({ name: baseName, filePath: dirFilePath });
      }
    }
  }
  return exports;
}

const symbolIndex = new Map();
const searchDirs = ['pkg-core', 'pkg-ui', 'pkg-main', 'pkg-services', 'pkg-components', 'pkg-store'];

for (const dir of searchDirs) {
  const dirPath = path.join(PROJECT_ROOT, dir);
  if (!fs.existsSync(dirPath)) continue;
  const files = findFilesRecursively(dirPath, ['.ts', '.js', '.svelte']);
  for (const file of files) {
    const exports = extractExports(file);
    for (const exp of exports) {
      if (!symbolIndex.has(exp.name)) symbolIndex.set(exp.name, []);
      symbolIndex.get(exp.name).push(exp);
    }
  }
}

console.log('--- DEBUG VTable ---');
console.log('VTable:', JSON.stringify(symbolIndex.get('VTable'), null, 2));
console.log('vTable:', JSON.stringify(symbolIndex.get('vTable'), null, 2));
