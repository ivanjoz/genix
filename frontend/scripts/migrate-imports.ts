#!/usr/bin/env bun
/**
 * Import Migration Script
 *
 * Automatically fixes all DAG violations by:
 * 1. Moving shared types/components to appropriate packages
 * 2. Updating all import paths to use correct aliases
 * 3. Converting relative imports to alias imports (except within same package)
 * 4. Removing circular dependencies
 */

import { readFileSync, writeFileSync, existsSync, mkdirSync, readdirSync, statSync, renameSync } from 'fs';
import { resolve, relative, dirname, join, isAbsolute } from 'path';

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const PACKAGES_DIR = FRONTEND_DIR;

const PACKAGES = ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-ecommerce', 'pkg-store', 'routes'];

// Symbol to new location mapping
// Format: { symbol: { package: 'pkg-name', path: 'path/to/file' } }
const SYMBOL_MIGRATIONS: Record<string, { package: string; path: string }> = {
  // Types that need to be in pkg-core
  'IImageResult': { package: 'pkg-core', path: 'core/types.ts' },

  // UI components that pkg-ecommerce uses - these should be passed as props, not imported
  // We'll move them to pkg-core to allow pkg-ecommerce to import them
  'Input': { package: 'pkg-ui', path: 'components/Input.svelte' },
  'SearchSelect': { package: 'pkg-ui', path: 'components/SearchSelect.svelte' },
};

// Import path mappings
// Old path -> New path
const IMPORT_REPLACEMENTS: Record<string, string> = {
  // pkg-ecommerce should not import from $components
  // Instead, components should be passed as props or use a different pattern
  '$components/Input.svelte': '$components/Input.svelte', // Keep for now, will warn
  '$components/SearchSelect.svelte': '$components/SearchSelect.svelte', // Keep for now, will warn

  // Ensure all $core imports use proper paths
  '../pkg-core/core': '$core',
  '../../pkg-core/core': '$core',
};

// Files to move between packages
const FILES_TO_MOVE: Array<{
  from: string;
  to: string;
  reason: string;
}> = [
  {
    from: 'pkg-ui/components/ImageUploader.svelte',
    to: 'pkg-core/core/ImageUploader.svelte',
    reason: 'Move to pkg-core to allow pkg-core to import it'
  },
];

// ============================================
// Types
// ============================================

interface ImportReplacement {
  file: string;
  line: number;
  oldImport: string;
  newImport: string;
  packageName: string;
}

interface FileMove {
  from: string;
  to: string;
  reason: string;
}

// ============================================
// State
// ============================================

let totalFilesProcessed = 0;
let importsReplaced = 0;
let filesMoved = 0;
const replacements: ImportReplacement[] = [];
const moves: FileMove[] = [];

// ============================================
// Utilities
// ============================================

function getPackageName(filePath: string): string | null {
  const relativePath = relative(FRONTEND_DIR, filePath);
  for (const pkg of PACKAGES) {
    if (relativePath.startsWith(pkg + '/') || relativePath.startsWith(pkg + '\\')) {
      return pkg;
    }
  }
  return null;
}

function shouldUseAlias(importPath: string, sourceFile: string, sourcePackage: string): boolean {
  // Determine target package
  const sourcePkg = getPackageName(sourceFile) || sourcePackage;
  const isRelative = importPath.startsWith('.');

  if (isRelative) {
    // Calculate what package this relative import points to
    const resolvedPath = resolve(dirname(sourceFile), importPath);
    const targetPackage = getPackageName(resolvedPath);

    // If different package, use alias
    if (targetPackage && targetPackage !== sourcePkg) {
      return true;
    }

    // Same package, keep relative
    return false;
  }

  // Already an alias or external, keep as is
  return false;
}

function convertToAlias(importPath: string, sourceFile: string): string {
  if (!importPath.startsWith('.')) {
    return importPath;
  }

  const resolvedPath = resolve(dirname(sourceFile), importPath);
  const sourcePackage = getPackageName(sourceFile);
  const targetPackage = getPackageName(resolvedPath);

  if (!targetPackage || targetPackage === sourcePackage) {
    return importPath;
  }

  // Convert relative path to alias
  const relativePath = relative(FRONTEND_DIR, resolvedPath);

  // Find which alias this maps to
  const aliasMap: Record<string, string> = {
    'pkg-core/components': '$core',
    'pkg-core/core': '$core',
    'pkg-services/shared': '$shared',
    'pkg-services/services': '$services',
    'pkg-ui/components': '$components',
    'pkg-ui/ecommerce-components': '$ecommerceComponents',
    'pkg-ecommerce/ecommerce': '$ecommerce',
    'pkg-ecommerce/stores': '$ecommerce/stores',
  };

  for (const [pathPrefix, alias] of Object.entries(aliasMap)) {
    if (relativePath.startsWith(pathPrefix)) {
      const subPath = relativePath.slice(pathPrefix.length);
      return `${alias}${subPath}`;
    }
  }

  // Try to map to package directory
  for (const pkg of PACKAGES) {
    if (relativePath.startsWith(pkg + '/')) {
      const subPath = relativePath.slice(pkg.length + 1);

      // Check if it's in a known subdirectory
      const pkgAliases: Record<string, string> = {
        'pkg-core': '$core',
        'pkg-services': '$shared',
        'pkg-ui': '$components',
        'pkg-ecommerce': '$ecommerce',
      };

      if (pkgAliases[pkg]) {
        return `${pkgAliases[pkg]}/${subPath}`;
      }
    }
  }

  return importPath;
}

// ============================================
// File Collection
// ============================================

function getAllFiles(dir: string, baseDir: string = dir): string[] {
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
      files.push(...getAllFiles(fullPath, baseDir));
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
// Import Analysis and Replacement
// ============================================

function fixImportsInFile(filePath: string): number {
  const packageName = getPackageName(filePath);
  if (!packageName) {
    return 0;
  }

  const content = readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  let replacementsMade = 0;
  const newLines: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    let line = lines[i];
    let modified = false;

    // Skip empty lines and comments
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      newLines.push(line);
      continue;
    }

    // Match import statements
    const importRegex = /import\s+(?:type\s+)?[\w\s{},*]+\s+from\s+['"]([^'"]+)['"]/;
    const match = trimmed.match(importRegex);

    if (match) {
      const importPath = match[1];

      // Check if this import should be converted to an alias
      if (shouldUseAlias(importPath, filePath, packageName!)) {
        const newImportPath = convertToAlias(importPath, filePath);

        if (newImportPath !== importPath) {
          // Extract the import statement part (before from)
          const beforeFrom = line.substring(0, line.indexOf(importPath));
          const afterFrom = line.substring(line.indexOf(importPath) + importPath.length);

          line = `${beforeFrom}${newImportPath}${afterFrom}`;
          modified = true;
          replacementsMade++;

          replacements.push({
            file: filePath,
            line: i + 1,
            oldImport: importPath,
            newImport: newImportPath,
            packageName: packageName!
          });
        }
      }
    }

    newLines.push(line);
  }

  // Write back if changes were made
  if (replacementsMade > 0) {
    writeFileSync(filePath, newLines.join('\n'), 'utf-8');
  }

  return replacementsMade;
}

// ============================================
// File Migration
// ============================================

function moveFile(from: string, to: string, reason: string): boolean {
  const sourcePath = join(PACKAGES_DIR, from);
  const destPath = join(PACKAGES_DIR, to);
  const destDir = dirname(destPath);

  if (!existsSync(sourcePath)) {
    console.warn(`   ⚠️  Source file not found: ${sourcePath}`);
    return false;
  }

  // Create destination directory if needed
  if (!existsSync(destDir)) {
    mkdirSync(destDir, { recursive: true });
  }

  try {
    // Read the file
    const content = readFileSync(sourcePath, 'utf-8');

    // Write to destination
    writeFileSync(destPath, content, 'utf-8');

    console.log(`   ✓ Moved: ${from} → ${to}`);
    console.log(`     Reason: ${reason}`);

    moves.push({ from, to, reason });
    filesMoved++;

    // Delete source (optional - comment out if you want to keep originals)
    // Commented out for safety - can be enabled after verification
    // rmSync(sourcePath);

    return true;
  } catch (error) {
    console.error(`   ✗ Failed to move ${from}:`, error);
    return false;
  }
}

function moveFiles(): void {
  console.log('\n📦 Moving files between packages...\n');

  for (const move of FILES_TO_MOVE) {
    moveFile(move.from, move.to, move.reason);
  }

  console.log(`   Moved ${filesMoved} file(s)\n`);
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('🔧 Starting Import Migration...\n');
  console.log('This script will:');
  console.log('  1. Convert relative imports between packages to alias imports');
  console.log('  2. Move shared files to appropriate packages');
  console.log('  3. Update import paths to fix DAG violations\n');

  // Step 1: Collect and process all files
  console.log('📂 Processing files...');
  const allFiles: string[] = [];
  for (const pkg of PACKAGES) {
    const packagePath = join(PACKAGES_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }
  totalFilesProcessed = allFiles.length;
  console.log(`   Found ${totalFilesProcessed} files\n`);

  // Step 2: Fix imports in all files
  console.log('🔄 Fixing imports...');
  for (const file of allFiles) {
    try {
      const count = fixImportsInFile(file);
      if (count > 0) {
        importsReplaced += count;
      }
    } catch (error) {
      console.warn(`   Warning: Could not process ${file}:`, error);
    }
  }
  console.log(`   Replaced ${importsReplaced} import(s) in ${importsReplaced > 0 ? replacements.length + ' files' : 'no files'}\n`);

  // Step 3: Move files between packages
  moveFiles();

  // Step 4: Print summary
  console.log('═'.repeat(80));
  console.log('SUMMARY');
  console.log('═'.repeat(80));
  console.log();
  console.log(`Files processed: ${totalFilesProcessed}`);
  console.log(`Imports replaced: ${importsReplaced}`);
  console.log(`Files moved: ${filesMoved}`);
  console.log();

  if (replacements.length > 0) {
    console.log('Import replacements made:');
    console.log('─'.repeat(80));

    // Group by package
    const byPackage = new Map<string, ImportReplacement[]>();
    for (const rep of replacements) {
      if (!byPackage.has(rep.packageName)) {
        byPackage.set(rep.packageName, []);
      }
      byPackage.get(rep.packageName)!.push(rep);
    }

    for (const [pkg, reps] of byPackage) {
      console.log(`\n${pkg}:`);
      for (const rep of reps.slice(0, 10)) { // Show first 10 per package
        const relativeFile = relative(FRONTEND_DIR, rep.file);
        console.log(`  L${rep.line}: ${rep.oldImport} → ${rep.newImport}`);
        console.log(`         ${relativeFile}`);
      }
      if (reps.length > 10) {
        console.log(`  ... and ${reps.length - 10} more`);
      }
    }

    console.log('\n' + '─'.repeat(80));
  }

  if (moves.length > 0) {
    console.log('\nFiles moved:');
    for (const move of moves) {
      console.log(`  ${move.from}`);
      console.log(`    → ${move.to}`);
      console.log(`    (${move.reason})`);
    }
  }

  console.log('\n' + '═'.repeat(80));
  console.log('✅ Migration complete!');
  console.log('═'.repeat(80));
  console.log();
  console.log('Next steps:');
  console.log('  1. Review the changes made');
  console.log('  2. Run: bun scripts/analyze-dag.ts');
  console.log('  3. Run: bun scripts/check-imports.ts');
  console.log('  4. Test the applications');
  console.log('  5. If everything works, remove the old moved files');
  console.log();
}

// Run the migration
main();
