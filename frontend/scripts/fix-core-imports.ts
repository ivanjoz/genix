#!/usr/bin/env -S bun
/**
 * Fix Core Imports Script
 *
 * This script removes .ts extensions from $core imports across the codebase.
 * The Vite alias resolver needs imports without extensions to properly resolve
 * to pkg-core/lib/ or pkg-core/core/ subdirectories.
 */

import fs from 'fs';
import path from 'path';

const PROJECT_ROOT = process.cwd();
const CORE_IMPORT_PATTERN = /from\s+['"](\$core\/[^'"]+\.ts)['"]/g;

interface FileFix {
  filePath: string;
  imports: string[];
  fixes: string[];
}

/**
 * Recursively find all .ts and .svelte files in a directory
 */
function findFilesRecursively(dir: string, extensions: string[]): string[] {
  const files: string[] = [];

  function walk(currentDir: string) {
    const items = fs.readdirSync(currentDir);

    for (const item of items) {
      const fullPath = path.join(currentDir, item);
      const stat = fs.statSync(fullPath);

      if (stat.isDirectory()) {
        // Skip node_modules and build directories
        if (item !== 'node_modules' && item !== '.svelte-kit' && item !== 'build') {
          walk(fullPath);
        }
      } else if (stat.isFile()) {
        const ext = path.extname(item);
        if (extensions.includes(ext)) {
          files.push(fullPath);
        }
      }
    }
  }

  walk(dir);
  return files;
}

/**
 * Find all files that contain $core imports with .ts extensions
 */
function findFilesWithCoreImports(): FileFix[] {
  const files: FileFix[] = [];

  // Search in all package directories
  const searchDirs = [
    'routes',
    'pkg-components',
    'pkg-store',
    'pkg-ui',
    'pkg-services'
  ];

  for (const dir of searchDirs) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;

    const matchedFiles = findFilesRecursively(dirPath, ['.ts', '.svelte']);

    for (const filePath of matchedFiles) {
      const content = fs.readFileSync(filePath, 'utf-8');
      const matches = [...content.matchAll(CORE_IMPORT_PATTERN)];

      if (matches.length > 0) {
        const imports = matches.map(m => m[1]);
        files.push({
          filePath: path.relative(PROJECT_ROOT, filePath),
          imports,
          fixes: imports.map(imp => imp.replace(/\.ts$/, ''))
        });
      }
    }
  }

  return files;
}

/**
 * Fix import statements in a file
 */
function fixImportsInFile(filePath: string, fixes: { original: string, fixed: string }[]): boolean {
  const fullPath = path.join(PROJECT_ROOT, filePath);
  let content = fs.readFileSync(fullPath, 'utf-8');
  let modified = false;

  for (const { original, fixed } of fixes) {
    const pattern = new RegExp(`from\\s+['"]${original.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}['"]`, 'g');
    if (pattern.test(content)) {
      content = content.replace(pattern, `from '${fixed}'`);
      modified = true;
    }
  }

  if (modified) {
    fs.writeFileSync(fullPath, content, 'utf-8');
  }

  return modified;
}

/**
 * Main execution
 */
function main() {
  console.log('🔍 Searching for $core imports with .ts extensions...\n');

  const filesWithIssues = findFilesWithCoreImports();

  if (filesWithIssues.length === 0) {
    console.log('✅ No $core imports with .ts extensions found!');
    return;
  }

  console.log(`Found ${filesWithIssues.length} file(s) with issues:\n`);

  // Display findings
  for (const file of filesWithIssues) {
    console.log(`📄 ${file.filePath}`);
    for (const imp of file.imports) {
      console.log(`   - ${imp} → ${imp.replace(/\.ts$/, '')}`);
    }
    console.log('');
  }

  // Ask for confirmation
  console.log('📊 Summary:');
  console.log(`   Total files to fix: ${filesWithIssues.length}`);
  console.log(`   Total imports to fix: ${filesWithIssues.reduce((acc, f) => acc + f.imports.length, 0)}`);

  console.log('\n🔧 Proceeding with fixes...\n');

  // Apply fixes
  let fixedFiles = 0;
  let fixedImports = 0;

  for (const file of filesWithIssues) {
    const fixes = file.imports.map(imp => ({ original: imp, fixed: imp.replace(/\.ts$/, '') }));
    const modified = fixImportsInFile(file.filePath, fixes);

    if (modified) {
      fixedFiles++;
      fixedImports += fixes.length;
      console.log(`✅ Fixed: ${file.filePath}`);
    }
  }

  console.log(`\n✨ Completed! Fixed ${fixedImports} imports in ${fixedFiles} file(s).`);

  // Also fix the app.css import in +layout.svelte
  console.log('\n🔧 Checking app.css import...');
  const layoutPath = path.join(PROJECT_ROOT, 'routes/routes/+layout.svelte');
  if (fs.existsSync(layoutPath)) {
    let content = fs.readFileSync(layoutPath, 'utf-8');
    if (content.includes("import '../app.css';")) {
      content = content.replace("import '../app.css';", "import '$lib/app.css';");
      fs.writeFileSync(layoutPath, content, 'utf-8');
      console.log('✅ Fixed app.css import in +layout.svelte');
    } else if (content.includes("import '$lib/app.css';")) {
      console.log('✅ app.css import already correct');
    }
  }
}

// Run the script
try {
  main();
} catch (error) {
  console.error('❌ Error:', error);
  process.exit(1);
}
