#!/usr/bin/env bun
/**
 * Import Path Fixer Script
 * 
 * Updates all import paths to point to correct locations after refactoring
 * Handles:
 * 1. Symbol relocations (e.g., ImageUploader moved from $components to $core)
 * 2. Relative imports between packages -> convert to alias imports
 * 3. Same-package imports -> keep as relative
 */

import { readFileSync, writeFileSync, existsSync, readdirSync } from 'fs';
import { resolve, relative, dirname, join } from 'path';

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const PACKAGES_DIR = FRONTEND_DIR;

const PACKAGES = ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-ecommerce', 'pkg-store', 'pkg-main'];

// Symbol relocations: when a file moved, update all imports
// Format: oldPath -> newPath
const IMPORT_RELOCATIONS: Record<string, string> = {
  '$components/ImageUploader.svelte': '$core/ImageUploader.svelte',
  '$components/SearchSelect.svelte': '$components/SearchSelect.svelte', // Keep in pkg-ui
  '$components/Input.svelte': '$components/Input.svelte', // Keep in pkg-ui
};

// Package to alias mapping
const PACKAGE_TO_ALIAS: Record<string, string> = {
  'pkg-core': '$core',
  'pkg-services': '$shared',
  'pkg-ui': '$components',
  'pkg-ecommerce': '$ecommerce',
  'pkg-store': '$lib', // pkg-store uses $lib for its own lib
  'pkg-main': '$lib',
};

// Subdirectory mappings within packages
const SUBDIR_ALIASES: Record<string, Record<string, string>> = {
  'pkg-core': {
    'core': '',
    'types': 'core/types.ts',
  },
  'pkg-ui': {
    'components': '',
    'ecommerce-components': '$ecommerceComponents',
  },
  'pkg-ecommerce': {
    'ecommerce': '',
    'stores': '$ecommerce/stores',
  },
  'pkg-services': {
    'services': '$services',
    'shared': '$shared',
  },
};

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
  packageName: string;
}> = [];

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

function resolveRelativePath(importPath: string, sourceFile: string): string {
  if (!importPath.startsWith('.')) {
    return importPath;
  }

  const resolvedPath = resolve(dirname(sourceFile), importPath);
  const resolvedRelative = relative(FRONTEND_DIR, resolvedPath);
  
  return resolvedRelative;
}

function convertRelativeToAlias(resolvedPath: string, sourcePackage: string): string {
  const parts = resolvedPath.split('/');
  if (parts.length < 2) {
    return resolvedPath; // Not an inter-package import
  }

  const targetPackage = parts[0];
  const subPath = parts.slice(1).join('/');

  // Same package - keep relative
  if (targetPackage === sourcePackage) {
    return null as any; // Signal to keep relative
  }

  // Different package - convert to alias
  const alias = PACKAGE_TO_ALIAS[targetPackage];
  if (!alias) {
    return resolvedPath; // Unknown package, keep as is
  }

  // Check if it's a known subdirectory
  const subdirs = SUBDIR_ALIASES[targetPackage] || {};
  for (const [subdir, aliasPath] of Object.entries(subdirs)) {
    if (subPath.startsWith(subdir)) {
      const remaining = subPath.slice(subdir.length);
      const separator = remaining && aliasPath ? '/' : '';
      return aliasPath ? `${aliasPath}${separator}${remaining}` : null;
    }
  }

  // Default: use package alias
  return `${alias}/${subPath}`;
}

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
// Import Fixing
// ============================================

function fixImportsInFile(filePath: string): number {
  const packageName = getPackageName(filePath);
  if (!packageName) {
    return 0;
  }

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
      let newImportPath: string | null = importPath;
      
      // Priority 1: Check for relocations
      if (IMPORT_RELOCATIONS[importPath]) {
        newImportPath = IMPORT_RELOCATIONS[importPath];
      }
      // Priority 2: Convert relative imports to aliases
      else if (importPath.startsWith('.')) {
        const resolvedPath = resolveRelativePath(importPath, filePath);
        const aliasPath = convertRelativeToAlias(resolvedPath, packageName!);
        
        if (aliasPath !== null) {
          newImportPath = aliasPath;
        } else if (resolvedPath.startsWith(packageName!)) {
          // Same package - keep relative but normalize
          const subPath = resolvedPath.slice(packageName!.length + 1);
          newImportPath = subPath.startsWith('./') ? subPath : `./${subPath}`;
        }
      }

      // Apply replacement if needed
      if (newImportPath && newImportPath !== importPath) {
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
          packageName: packageName!
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
// Main Execution
// ============================================

function main() {
  console.log('🔧 Starting Import Path Fixer...\n');
  
  console.log('This script will:');
  console.log('  1. Update imports for relocated files (e.g., ImageUploader)');
  console.log('  2. Convert relative inter-package imports to aliases');
  console.log('  3. Keep same-package imports as relative\n');

  // Collect all files
  console.log('📂 Collecting files...');
  const allFiles: string[] = [];
  for (const pkg of PACKAGES) {
    const packagePath = join(PACKAGES_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }
  totalFilesProcessed = allFiles.length;
  console.log(`   Found ${totalFilesProcessed} files\n`);

  // Fix imports
  console.log('🔄 Fixing imports...\n');
  for (const file of allFiles) {
    try {
      const count = fixImportsInFile(file);
      if (count > 0) {
        importsUpdated += count;
      }
    } catch (error) {
      console.warn(`   Warning: Could not process ${file}:`, error);
    }
  }

  console.log(`   Updated ${importsUpdated} import(s) in ${changes.length > 0 ? 'multiple files' : 'no files'}\n`);

  // Print summary
  console.log('═'.repeat(80));
  console.log('SUMMARY');
  console.log('═'.repeat(80));
  console.log();
  console.log(`Files processed: ${totalFilesProcessed}`);
  console.log(`Imports updated: ${importsUpdated}`);
  console.log();

  if (changes.length > 0) {
    console.log('Changes made:');
    console.log('─'.repeat(80));
    
    // Group by type of change
    const relocations = changes.filter(c => c.oldImport in IMPORT_RELOCATIONS);
    const conversions = changes.filter(c => c.oldImport.startsWith('.') && !(c.oldImport in IMPORT_RELOCATIONS));
    const otherChanges = changes.filter(c => !(c.oldImport in IMPORT_RELOCATIONS) && !c.oldImport.startsWith('.'));
    
    if (relocations.length > 0) {
      console.log(`\n📦 File Relocations (${relocations.length}):`);
      for (const change of relocations.slice(0, 10)) {
        const relativeFile = relative(FRONTEND_DIR, change.file);
        console.log(`  ${relativeFile}:${change.line}`);
        console.log(`    ${change.oldImport} → ${change.newImport}`);
      }
      if (relocations.length > 10) {
        console.log(`  ... and ${relocations.length - 10} more`);
      }
    }
    
    if (conversions.length > 0) {
      console.log(`\n🔀 Relative → Alias (${conversions.length}):`);
      for (const change of conversions.slice(0, 10)) {
        const relativeFile = relative(FRONTEND_DIR, change.file);
        console.log(`  ${relativeFile}:${change.line}`);
        console.log(`    ${change.oldImport} → ${change.newImport}`);
      }
      if (conversions.length > 10) {
        console.log(`  ... and ${conversions.length - 10} more`);
      }
    }
    
    if (otherChanges.length > 0) {
      console.log(`\n🔧 Other Changes (${otherChanges.length}):`);
      for (const change of otherChanges.slice(0, 5)) {
        const relativeFile = relative(FRONTEND_DIR, change.file);
        console.log(`  ${relativeFile}:${change.line}`);
        console.log(`    ${change.oldImport} → ${change.newImport}`);
      }
      if (otherChanges.length > 5) {
        console.log(`  ... and ${otherChanges.length - 5} more`);
      }
    }
    
    console.log('\n' + '─'.repeat(80));
  }

  console.log('\n✅ Import path fixing complete!\n');
  console.log('Next steps:');
  console.log('  1. Review the changes above');
  console.log('  2. Run: bun scripts/analyze-dag.ts');
  console.log('  3. Run: bun scripts/check-imports.ts');
  console.log('  4. Test the applications');
  console.log('═'.repeat(80));
  
  process.exit(importsUpdated > 0 ? 0 : 1);
}

main();