#!/usr/bin/env -S bun
/**
 * Fix Missing Imports Script
 *
 * This script fixes remaining broken import paths after the migration:
 * 1. $core/store.svelte → $core/core/store.svelte
 * 2. $core/MobileMenu.svelte → $core/core/MobileMenu.svelte
 * 3. $lib/globals.svelte.ts → $store/globals.svelte.ts (needs $store alias)
 * 4. $lib/routes/* → correct paths
 */

import fs from 'fs';
import path from 'path';

const PROJECT_ROOT = process.cwd();

interface FileFix {
  filePath: string;
  originalImport: string;
  fixedImport: string;
}

/**
 * Recursively find all .ts and .svelte files
 */
function findFilesRecursively(dir: string, extensions: string[]): string[] {
  const files: string[] = [];

  function walk(currentDir: string) {
    try {
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
    } catch (e) {
      // Ignore permission errors
    }
  }

  walk(dir);
  return files;
}

/**
 * Check if a file exists at various possible locations
 */
function findFileLocation(importPath: string): string | null {
  const fileBases = [
    path.join(PROJECT_ROOT, importPath),
    path.join(PROJECT_ROOT, importPath + '.ts'),
    path.join(PROJECT_ROOT, importPath + '.js'),
    path.join(PROJECT_ROOT, importPath + '.svelte')
  ];

  for (const base of fileBases) {
    if (fs.existsSync(base)) {
      return base;
    }
  }

  return null;
}

/**
 * Analyze and collect all fixes needed
 */
function collectFixes(): FileFix[] {
  const fixes: FileFix[] = [];

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

    const files = findFilesRecursively(dirPath, ['.ts', '.svelte']);

    for (const filePath of files) {
      const content = fs.readFileSync(filePath, 'utf-8');

      // Pattern 1: $core/store.svelte → $core/core/store.svelte
      const storeMatch = content.match(/from\s+['"](\$core\/store\.svelte)['"]/);
      if (storeMatch) {
        fixes.push({
          filePath: path.relative(PROJECT_ROOT, filePath),
          originalImport: storeMatch[1],
          fixedImport: '$core/core/store.svelte'
        });
      }

      // Pattern 2: $core/MobileMenu.svelte → $core/core/MobileMenu.svelte
      const mobileMenuMatch = content.match(/from\s+['"](\$core\/MobileMenu\.svelte)['"]/);
      if (mobileMenuMatch) {
        fixes.push({
          filePath: path.relative(PROJECT_ROOT, filePath),
          originalImport: mobileMenuMatch[1],
          fixedImport: '$core/core/MobileMenu.svelte'
        });
      }

      // Pattern 3: $lib/globals.svelte.ts → $store/globals.svelte.ts
      const libGlobalsMatch = content.match(/from\s+['"](\$lib\/globals\.svelte\.ts)['"]/);
      if (libGlobalsMatch) {
        fixes.push({
          filePath: path.relative(PROJECT_ROOT, filePath),
          originalImport: libGlobalsMatch[1],
          fixedImport: '$store/globals.svelte.ts'
        });
      }

      // Pattern 4: $lib/routes/admin/usuarios/usuarios.svelte → correct path
      const usuariosMatch = content.match(/from\s+['"](\$lib\/routes\/admin\/usuarios\/usuarios\.svelte)['"]/);
      if (usuariosMatch) {
        fixes.push({
          filePath: path.relative(PROJECT_ROOT, filePath),
          originalImport: usuariosMatch[1],
          fixedImport: '$routes/admin/usuarios/usuarios.svelte'
        });
      }

      // Pattern 5: $lib/globals.svelte (without .ts) → $store/globals.svelte.ts
      const libGlobalsNoExtMatch = content.match(/from\s+['"](\$lib\/globals\.svelte)['"]/);
      if (libGlobalsNoExtMatch) {
        fixes.push({
          filePath: path.relative(PROJECT_ROOT, filePath),
          originalImport: libGlobalsNoExtMatch[1],
          fixedImport: '$store/globals.svelte'
        });
      }
    }
  }

  return fixes;
}

/**
 * Apply fixes to files
 */
function applyFixes(fixes: FileFix[]): number {
  let fixedCount = 0;

  for (const fix of fixes) {
    const fullPath = path.join(PROJECT_ROOT, fix.filePath);
    let content = fs.readFileSync(fullPath, 'utf-8');

    const pattern = new RegExp(
      `from\\s+['"]${fix.originalImport.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}['"]`,
      'g'
    );

    if (pattern.test(content)) {
      content = content.replace(pattern, `from '${fix.fixedImport}'`);
      fs.writeFileSync(fullPath, content, 'utf-8');
      fixedCount++;
      console.log(`✅ Fixed: ${fix.filePath}`);
      console.log(`   ${fix.originalImport} → ${fix.fixedImport}\n`);
    }
  }

  return fixedCount;
}

/**
 * Update svelte.config.js to add $store alias
 */
function updateSvelteConfig(): boolean {
  const configPath = path.join(PROJECT_ROOT, 'svelte.config.js');

  if (!fs.existsSync(configPath)) {
    console.log('⚠️  svelte.config.js not found');
    return false;
  }

  let content = fs.readFileSync(configPath, 'utf-8');

  // Check if $store alias already exists
  if (content.includes("$store:")) {
    console.log('✅ $store alias already exists in svelte.config.js');
    return false;
  }

  // Find the alias section and add $store
  const aliasPattern = /alias:\s*\{([^}]+)\}/;
  const match = content.match(aliasPattern);

  if (match) {
    const newAlias = `alias: {
			$store: path.resolve('./pkg-store/stores'),
			$routes: path.resolve('./routes'),${match[1].replace('alias:', '').trim()}`;
    content = content.replace(aliasPattern, newAlias);
    fs.writeFileSync(configPath, content, 'utf-8');
    console.log('✅ Added $store and $routes aliases to svelte.config.js');
    return true;
  }

  console.log('⚠️  Could not find alias section in svelte.config.js');
  return false;
}

/**
 * Main execution
 */
function main() {
  console.log('🔍 Searching for missing imports...\n');

  const fixes = collectFixes();

  if (fixes.length === 0) {
    console.log('✅ No missing imports found!');
  } else {
    console.log(`Found ${fixes.length} fix(es) needed:\n`);

    for (const fix of fixes) {
      console.log(`📄 ${fix.filePath}`);
      console.log(`   ${fix.originalImport} → ${fix.fixedImport}`);
    }

    console.log('\n🔧 Applying fixes...\n');
    const fixedCount = applyFixes(fixes);
    console.log(`\n✨ Fixed ${fixedCount} file(s)`);
  }

  console.log('\n🔧 Checking svelte.config.js aliases...\n');
  updateSvelteConfig();

  console.log('\n✅ Done! Try running `bun run dev` to test.');
}

// Run the script
try {
  main();
} catch (error) {
  console.error('❌ Error:', error);
  process.exit(1);
}
