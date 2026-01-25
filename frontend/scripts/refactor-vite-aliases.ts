#!/usr/bin/env bun
/**
 * Vite Alias Refactor Script
 *
 * Refactors Vite config aliases to a simpler, more intuitive format and updates all imports accordingly.
 *
 * Changes:
 * - Simplifies aliases to point directly to package roots
 * - Consolidates duplicate aliases ($shared and $services both point to pkg-services)
 * - Removes complex subdirectory paths from alias definitions
 * - Updates all import statements to use the new aliases
 */

import { readFileSync, writeFileSync, existsSync, readdirSync } from 'fs';
import { resolve, relative, dirname, join } from 'path';

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const VITE_CONFIG_PATH = join(FRONTEND_DIR, 'vite.config.ts');

// Import path transformations
// Maps old import patterns to new import patterns
const IMPORT_TRANSFORMATIONS: Array<{
  pattern: RegExp;
  replacement: string;
  description: string;
}> = [
  // Remove unnecessary subdirectory paths
  {
    pattern: /\$store\/stores\//g,
    replacement: '$store/',
    description: '$store/stores/* → $store/*'
  },
  {
    pattern: /\$services\/services\//g,
    replacement: '$services/',
    description: '$services/services/* → $services/*'
  },
  {
    pattern: /\$shared\//g,
    replacement: '$services/',
    description: '$shared/* → $services/*'
  },
  {
    pattern: /\$ecommerce\//g,
    replacement: '$components/',
    description: '$ecommerce/* → $components/*'
  },
  // Convert $lib to $app where appropriate
  {
    pattern: /\$lib\//g,
    replacement: '$app/',
    description: '$lib/* → $app/*'
  },
  // Keep $core, $routes, $ui, $components as is
];

// ============================================
// State
// ============================================

let totalFilesProcessed = 0;
let importsUpdated = 0;
const changes: Array<{
  file: string;
  line: number;
  oldImport: string;
  newImport: string;
  transformation: string;
}> = [];

// ============================================
// File Collection
// ============================================

function getAllFiles(dir: string): string[] {
  if (!existsSync(dir)) {
    return [];
  }

  const files: string[] = [];
  const entries = readdirSync(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = join(dir, entry.name);

    if (['node_modules', '.git', '.svelte-kit', 'build', 'dist', '.turbo'].includes(entry.name)) {
      continue;
    }
    if (entry.name.startsWith('.') && entry.name !== '.npmrc') {
      continue;
    }

    if (entry.isDirectory()) {
      files.push(...getAllFiles(fullPath));
    } else if (entry.isFile()) {
      const ext = entry.name;
      if (['.ts', '.tsx', '.svelte', '.js', '.jsx'].some(e => ext.endsWith(e))) {
        files.push(fullPath);
      }
    }
  }

  return files;
}

// ============================================
// Vite Config Update
// ============================================

function updateViteConfig(): boolean {
  if (!existsSync(VITE_CONFIG_PATH)) {
    console.error(`❌ vite.config.ts not found at ${VITE_CONFIG_PATH}`);
    return false;
  }

  console.log('📝 Updating vite.config.ts...\n');

  let content = readFileSync(VITE_CONFIG_PATH, 'utf-8');
  const originalContent = content;

  // Update the alias map in the esbuild plugin
  // Find the alias configuration object and replace it
  const aliasRegex = /const baseDir = \{[\s\S]*?\}\[alias\];/;

  const newAliasConfig = `const baseDir = {
    '$core': 'pkg-core',
    '$store': 'pkg-store',
    '$routes': 'routes',
    '$ui': 'pkg-ui',
    '$components': 'pkg-ui/components',
    '$shared': 'pkg-services',
    '$services': 'pkg-services'
  }[alias];`;

  content = content.replace(aliasRegex, newAliasConfig);

  // Update the comment about multiple subdirectories
  content = content.replace(
    /\/\/ For \$core, try multiple subdirectories/,
    '// For $core and $components, try multiple subdirectories'
  );

  // Add special handling for $components to also check pkg-components/ecommerce
  const possiblePathsRegex = /(if \(alias === '\$core'\) \{[\s\S]*?possiblePaths\.push\(path\.join\(baseDir, rest\)\);[\s\S]*?\})/;
  const newPossiblePaths = `if (alias === '$core') {
            possiblePaths.push(path.join(baseDir, 'lib', rest));
            possiblePaths.push(path.join(baseDir, 'core', rest));
            possiblePaths.push(path.join(baseDir, 'assets', rest));
            possiblePaths.push(path.join(baseDir, rest));
          } else if (alias === '$components') {
            possiblePaths.push(path.join(baseDir, rest));
            possiblePaths.push(path.join('pkg-components/ecommerce', rest));
          } else {
            possiblePaths.push(path.join(baseDir, rest));
          }`;

  content = content.replace(possiblePathsRegex, newPossiblePaths);

  if (content !== originalContent) {
    writeFileSync(VITE_CONFIG_PATH, content, 'utf-8');
    console.log('✅ vite.config.ts updated successfully\n');
    return true;
  } else {
    console.log('⚠️  No changes needed in vite.config.ts\n');
    return false;
  }
}

// ============================================
// Import Path Update
// ============================================

function updateImportPaths(filePath: string): number {
  const content = readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  let updatesMade = 0;
  const newLines: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    let line = lines[i];
    const trimmed = line.trim();
    let modified = false;

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      newLines.push(line);
      continue;
    }

    // Match import statements
    const importRegex = /import\s+(?:type\s+)?[\w\s{},*]+\s+from\s+['"]([^'"]+)['"]/;
    const match = trimmed.match(importRegex);

    if (match) {
      const importPath = match[1];
      let newImportPath = importPath;
      let transformation = '';

      // Apply transformations
      for (const transform of IMPORT_TRANSFORMATIONS) {
        if (transform.pattern.test(newImportPath)) {
          newImportPath = newImportPath.replace(transform.pattern, transform.replacement);
          transformation = transform.description;
          break;
        }
      }

      // Apply replacement if needed
      if (newImportPath !== importPath) {
        const importStart = line.indexOf(importPath);
        const beforeImport = line.substring(0, importStart);
        const afterImport = line.substring(importStart + importPath.length);

        line = `${beforeImport}${newImportPath}${afterImport}`;
        modified = true;
        updatesMade++;

        changes.push({
          file: filePath,
          line: i + 1,
          oldImport: importPath,
          newImport: newImportPath,
          transformation: transformation!
        });
      }
    }

    newLines.push(line);
  }

  // Write back if changes were made
  if (updatesMade > 0) {
    writeFileSync(filePath, newLines.join('\n'), 'utf-8');
  }

  return updatesMade;
}

// ============================================
// Intelligent Import Fixer Integration
// ============================================

function runIntelligentImportFixer(): void {
  console.log('🔧 Running intelligent-import-fixer.ts to fix remaining issues...\n');

  try {
    const { spawnSync } = require('child_process');
    const result = spawnSync('bun', ['scripts/intelligent-import-fixer.ts'], {
      cwd: FRONTEND_DIR,
      stdio: 'inherit'
    });

    if (result.status === 0) {
      console.log('\n✅ Intelligent import fixer completed successfully');
    } else {
      console.log('\n⚠️  Intelligent import fixer had some issues (see above)');
    }
  } catch (error) {
    console.error('❌ Error running intelligent-import-fixer:', error);
  }
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('╔═══════════════════════════════════════════════════════════════╗');
  console.log('║       Vite Alias Refactor - Simplify Import Paths            ║');
  console.log('╚═══════════════════════════════════════════════════════════════╝');
  console.log();

  // Step 1: Update vite.config.ts
  console.log('Step 1/3: Update vite.config.ts');
  console.log('─'.repeat(80));
  const configUpdated = updateViteConfig();

  // Step 2: Update all import paths
  console.log('\nStep 2/3: Update import paths in codebase');
  console.log('─'.repeat(80));

  console.log('📂 Collecting files...');
  const packageDirs = ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-components', 'pkg-store', 'routes'];
  const allFiles: string[] = [];
  for (const pkg of packageDirs) {
    const packagePath = join(FRONTEND_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }
  totalFilesProcessed = allFiles.length;
  console.log(`   Found ${totalFilesProcessed} files\n`);

  console.log('🔄 Updating import paths...\n');
  for (const file of allFiles) {
    try {
      const count = updateImportPaths(file);
      if (count > 0) {
        importsUpdated += count;
      }
    } catch (error) {
      console.warn(`   Warning: Could not process ${file}:`, error);
    }
  }

  console.log(`   Updated ${importsUpdated} import(s) across the codebase\n`);

  // Step 3: Run intelligent import fixer
  console.log('\nStep 3/3: Fix remaining import issues');
  console.log('─'.repeat(80));
  runIntelligentImportFixer();

  // Print summary
  console.log('\n╔═══════════════════════════════════════════════════════════════╗');
  console.log('║                          SUMMARY                               ║');
  console.log('╚═══════════════════════════════════════════════════════════════╝');
  console.log();
  console.log(`✅ vite.config.ts: ${configUpdated ? 'Updated' : 'No changes needed'}`);
  console.log(`📁 Files processed: ${totalFilesProcessed}`);
  console.log(`🔄 Imports updated: ${importsUpdated}`);
  console.log();

  if (changes.length > 0) {
    console.log('Sample changes:');
    console.log('─'.repeat(80));

    // Group by transformation type
    const byTransformation: Record<string, typeof changes> = {};
    for (const change of changes) {
      if (!byTransformation[change.transformation]) {
        byTransformation[change.transformation] = [];
      }
      byTransformation[change.transformation].push(change);
    }

    for (const [transformation, changesList] of Object.entries(byTransformation)) {
      console.log(`\n${transformation}:`);
      for (const change of changesList.slice(0, 3)) {
        const relativeFile = change.file.replace(FRONTEND_DIR + '/', '');
        console.log(`  ${relativeFile}:${change.line}`);
        console.log(`    ${change.oldImport} → ${change.newImport}`);
      }
      if (changesList.length > 3) {
        console.log(`  ... and ${changesList.length - 3} more`);
      }
    }

    console.log('\n' + '─'.repeat(80));
  }

  console.log('\n🎉 Alias refactoring complete!\n');
  console.log('New alias structure:');
  console.log('  $core → pkg-core');
  console.log('  $store → pkg-store');
  console.log('  $routes → routes');
  console.log('  $ui → pkg-ui');
  console.log('  $components → pkg-ui/components');
  console.log('  $shared → pkg-services');
  console.log('  $services → pkg-services');
  console.log();
  console.log('Next steps:');
  console.log('  1. Review the changes above');
  console.log('  2. Test the dev server: bun run dev');
  console.log('  3. If issues remain, run: bun scripts/intelligent-import-fixer.ts');
  console.log();

  process.exit(0);
}

main();
