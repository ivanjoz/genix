#!/usr/bin/env bun
/**
 * Intelligent Import Path Fixer
 *
 * This script fixes broken imports by searching for symbols across all packages.
 * Strategy:
 * 1. Build a comprehensive symbol index by scanning all exports in all packages
 * 2. For each file, parse all imports
 * 3. For each broken import:
 *    - Extract symbol names
 *    - Search the symbol index
 *    - If found in exactly one location: fix the import
 *    - If found in multiple locations: warn user (indicates copy instead of move)
 * 4. Update all import paths to use the correct package aliases
 *
 * Prioritizes correctness over performance.
 */

import { readFileSync, writeFileSync, existsSync, readdirSync, statSync } from 'fs';
import { resolve, relative, dirname, join, basename } from 'path';

// ============================================
// Types
// ============================================

interface SymbolExport {
  symbol: string;
  file: string;
  packageName: string;
  type: 'named' | 'default' | 'type' | 'namespace';
  isSvelteComponent: boolean;
}

interface SymbolLocation {
  symbol: string;
  locations: SymbolExport[];
}

interface ImportInfo {
  file: string;
  line: number;
  originalLine: string;
  importPath: string;
  symbols: string[];
  importType: 'named' | 'default' | 'type' | 'namespace' | 'mixed';
}

interface FileFix {
  file: string;
  line: number;
  oldImport: string;
  newImport: string;
  reason: string;
}

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const PACKAGES = ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-components', 'pkg-store', 'pkg-app'];

// Package to alias mapping
const PACKAGE_TO_ALIAS: Record<string, string> = {
  'pkg-core': '$core',
  'pkg-services': '$services',
  'pkg-ui': '$ui',
  'pkg-components': '$components',
  'pkg-store': '$store',
  'pkg-app': '$app',
};

// Known subdirectory mappings for cleaner imports
const SUBDIR_MAPPINGS: Record<string, Record<string, string>> = {
  'pkg-core': {
    'lib': '',
    'core': '',
    'assets': '',
  },
  'pkg-services': {
    'services': '',
    'shared': '',
    'functions': '',
  },
  'pkg-ui': {
    'components': '',
    'assets': '',
  },
  'pkg-components': {
    'ecommerce': '',
    'ecommerce-components': '',
    'stores': '',
  },
  'pkg-store': {
    'stores': '',
  },
  'pkg-app': {
    'routes': '',
    'lib': '',
  },
};

const DRY_RUN = process.argv.includes('--dry-run');
const VERBOSE = process.argv.includes('--verbose');

// ============================================
// Global State
// ============================================

// Symbol index: symbol name -> locations where it's exported
const symbolIndex = new Map<string, SymbolExport[]>();

// File to exports mapping
const fileExports = new Map<string, SymbolExport[]>();

// Import analysis results
const totalFilesProcessed = 0;
const importsFixed = 0;
const duplicateWarnings: string[] = [];
const unresolvedImports: Array<{ file: string; symbol: string; importPath: string }> = [];
const fileFixes: FileFix[] = [];

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

function log(message: string, level: 'info' | 'success' | 'warning' | 'error' = 'info') {
  const colors = {
    info: '\x1b[36m',
    success: '\x1b[32m',
    warning: '\x1b[33m',
    error: '\x1b[31m'
  };
  const reset = '\x1b[0m';
  console.log(`${colors[level]}${message}${reset}`);
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
// Export Parsing
// ============================================

function parseExports(filePath: string): SymbolExport[] {
  const content = readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const exports: SymbolExport[] = [];
  const packageName = getPackageName(filePath);

  if (!packageName) {
    return exports;
  }

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      continue;
    }

    // Detect Svelte component (default export in .svelte files)
    const isSvelteComponent = filePath.endsWith('.svelte');

    // Pattern 1: `export const foo = ...` or `export let foo = ...` in Svelte
    const namedMatch = trimmed.match(/export\s+(?:const|let|var|function|class|async\s+function)\s+(\w+)/);
    if (namedMatch) {
      exports.push({
        symbol: namedMatch[1],
        file: filePath,
        packageName,
        type: 'named',
        isSvelteComponent: false
      });
    }

    // Pattern 2: `export type Foo = ...`
    const typeMatch = trimmed.match(/export\s+(?:type|interface)\s+(\w+)/);
    if (typeMatch) {
      exports.push({
        symbol: typeMatch[1],
        file: filePath,
        packageName,
        type: 'type',
        isSvelteComponent: false
      });
    }

    // Pattern 3: `export default foo` or `export default class Foo {`
    const defaultMatch = trimmed.match(/export\s+default\s+(?:class|function|const|let|var)\s+(\w+)?/);
    if (defaultMatch) {
      const symbolName = defaultMatch[1] || basename(filePath, filePath.endsWith('.svelte') ? '.svelte' : '.ts');
      exports.push({
        symbol: symbolName,
        file: filePath,
        packageName,
        type: 'default',
        isSvelteComponent
      });
    }

    // Pattern 4: Named exports on one line: `export { foo, bar as baz }`
    const namedInlineMatch = trimmed.match(/export\s+\{\s*([^}]+)\s*\}/);
    if (namedInlineMatch && !trimmed.includes('from')) {
      const symbolsList = namedInlineMatch[1];
      const symbols = symbolsList.split(',').map(s => s.trim().split(/\s+as\s+/i).pop()).filter(Boolean);
      for (const symbol of symbols) {
        exports.push({
          symbol,
          file: filePath,
          packageName,
          type: 'named',
          isSvelteComponent: false
        });
      }
    }

    // Pattern 5: Type exports on one line: `export type { Foo, Bar as Baz }`
    const typeInlineMatch = trimmed.match(/export\s+type\s+\{\s*([^}]+)\s*\}/);
    if (typeInlineMatch && !trimmed.includes('from')) {
      const symbolsList = typeInlineMatch[1];
      const symbols = symbolsList.split(',').map(s => s.trim().split(/\s+as\s+/i).pop()).filter(Boolean);
      for (const symbol of symbols) {
        exports.push({
          symbol,
          file: filePath,
          packageName,
          type: 'type',
          isSvelteComponent: false
        });
      }
    }

    // Pattern 6: Svelte component exports (using `<script context="module">`)
    // We'll handle this by checking if it's a .svelte file and adding a default export
    if (isSvelteComponent && !exports.find(e => e.type === 'default')) {
      exports.push({
        symbol: basename(filePath, '.svelte'),
        file: filePath,
        packageName,
        type: 'default',
        isSvelteComponent: true
      });
    }

    // Pattern 7: `export * from './something'` (re-exports)
    const reExportMatch = trimmed.match(/export\s+\*\s+from\s+['"]([^'"]+)['"]/);
    if (reExportMatch) {
      const reExportPath = reExportMatch[1];
      // Resolve the path and get exports from that file
      const resolvedPath = resolve(dirname(filePath), reExportPath);
      // We'll handle re-exports in a second pass
    }
  }

  return exports;
}

function buildSymbolIndex() {
  log('\n🔍 Building symbol index...\n', 'info');

  const allFiles: string[] = [];
  for (const pkg of PACKAGES) {
    const packagePath = join(FRONTEND_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }

  log(`   Scanning ${allFiles.length} files for exports...`, 'info');

  for (const file of allFiles) {
    try {
      const exports = parseExports(file);
      fileExports.set(file, exports);

      for (const exp of exports) {
        if (!symbolIndex.has(exp.symbol)) {
          symbolIndex.set(exp.symbol, []);
        }
        symbolIndex.get(exp.symbol)!.push(exp);
      }
    } catch (error) {
      log(`   Warning: Could not parse exports from ${file}`, 'warning');
    }
  }

  // Handle re-exports in a second pass
  for (const file of allFiles) {
    try {
      const content = readFileSync(file, 'utf-8');
      const lines = content.split('\n');
      const packageName = getPackageName(file);

      if (!packageName) continue;

      for (const line of lines) {
        const trimmed = line.trim();
        const reExportMatch = trimmed.match(/export\s+(?:type\s+)?\*\s+from\s+['"]([^'"]+)['"]/);
        
        if (reExportMatch) {
          const reExportPath = reExportMatch[1];
          const resolvedPath = resolve(dirname(file), reExportPath);

          // Find the actual file
          let targetFile = resolvedPath;
          if (!existsSync(targetFile)) {
            if (existsSync(targetFile + '.ts')) targetFile += '.ts';
            else if (existsSync(targetFile + '.js')) targetFile += '.js';
            else if (existsSync(join(targetFile, 'index.ts'))) targetFile = join(targetFile, 'index.ts');
            else if (existsSync(join(targetFile, 'index.js'))) targetFile = join(targetFile, 'index.js');
          }

          if (fileExports.has(targetFile)) {
            const reExported = fileExports.get(targetFile)!;
            for (const exp of reExported) {
              // Add to this file's exports
              if (!symbolIndex.has(exp.symbol)) {
                symbolIndex.set(exp.symbol, []);
              }
              symbolIndex.get(exp.symbol)!.push({
                ...exp,
                file: file, // Point to the re-exporting file
                packageName
              });
            }
          }
        }
      }
    } catch (error) {
      // Skip re-export parsing errors
    }
  }

  const totalSymbols = symbolIndex.size;
  const totalExports = Array.from(symbolIndex.values()).reduce((sum, locs) => sum + locs.length, 0);
  
  log(`   Found ${totalSymbols} unique symbols across ${totalExports} exports\n`, 'success');
}

// ============================================
// Import Parsing
// ============================================

function parseImports(filePath: string): ImportInfo[] {
  const content = readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const imports: ImportInfo[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      continue;
    }

    // Match import patterns
    const importRegex = /import\s+(type\s+)?([\w\s{},*]+?)\s+from\s+['"]([^'"]+)['"]/;
    const match = trimmed.match(importRegex);

    if (match) {
      const isTypeImport = !!match[1];
      const importPath = match[3];
      const symbolsPart = match[2].trim();

      // Extract symbols
      const symbols: string[] = [];

      if (symbolsPart === '*') {
        // Namespace import: import * as foo
        const namespaceMatch = symbolsPart.match(/\*\s+as\s+(\w+)/);
        if (namespaceMatch) {
          symbols.push(namespaceMatch[1]);
        }
      } else if (symbolsPart.startsWith('{')) {
        // Named imports: import { foo, bar as baz }
        const inner = symbolsPart.slice(1, -1).trim();
        const items = inner.split(',').map(s => s.trim());
        for (const item of items) {
          if (item.includes(' as ')) {
            symbols.push(item.split(/\s+as\s+/i).pop()!);
          } else if (item.includes(':')) {
            // Type imports: import type { Foo }
            symbols.push(item.split(':')[1].trim());
          } else {
            symbols.push(item);
          }
        }
      } else {
        // Default import or mixed
        if (symbolsPart.includes(',')) {
          // Mixed: import foo, { bar }
          const [defaultPart, namedPart] = symbolsPart.split(',');
          if (defaultPart.trim()) symbols.push(defaultPart.trim());
          if (namedPart) {
            const inner = namedPart.trim().slice(1, -1);
            const items = inner.split(',').map(s => s.trim());
            for (const item of items) {
              if (item.includes(' as ')) {
                symbols.push(item.split(/\s+as\s+/i).pop()!);
              } else {
                symbols.push(item);
              }
            }
          }
        } else {
          // Default import
          symbols.push(symbolsPart);
        }
      }

      let importType: ImportInfo['importType'] = 'named';
      if (isTypeImport) importType = 'type';
      else if (symbolsPart.includes('*')) importType = 'namespace';
      else if (!symbolsPart.startsWith('{') && !symbolsPart.includes(',')) importType = 'default';

      imports.push({
        file: filePath,
        line: i,
        originalLine: line,
        importPath,
        symbols,
        importType
      });
    }
  }

  return imports;
}

// ============================================
// Import Resolution & Fixing
// ============================================

function constructImportPath(targetPackage: string, targetFile: string, sourceFile: string): string {
  const packageName = getPackageName(sourceFile);
  if (!packageName) return null;

  // Same package - use relative path
  if (targetPackage === packageName) {
    const relativePath = relative(dirname(sourceFile), targetFile);
    return relativePath.startsWith('.') ? relativePath : `./${relativePath}`;
  }

  // Different package - use alias
  const alias = PACKAGE_TO_ALIAS[targetPackage];
  if (!alias) return null;

  // Get the relative path from the package root
  const relativeFile = relative(join(FRONTEND_DIR, targetPackage), targetFile);
  
  // Special handling for $ui - it should point to pkg-ui/components
  if (alias === '$ui' && relativeFile.startsWith('components/')) {
    const remaining = relativeFile.replace(/^components\//, '');
    return remaining ? `$ui/${remaining}` : '$ui';
  }
  
  // Special handling for $components - it should point to pkg-ui/components
  // but also check pkg-components/ecommerce as a fallback
  if (alias === '$components') {
    // If the file is in pkg-ui/components, use $components directly
    if (targetFile.includes('/pkg-ui/components/')) {
      const remaining = relativeFile.replace(/^components\//, '');
      return remaining ? `$components/${remaining}` : '$components';
    }
    // If the file is in pkg-components/ecommerce, use $components
    if (targetFile.includes('/pkg-components/ecommerce/')) {
      const remaining = relativeFile.replace(/^ecommerce\//, '');
      return remaining ? `$components/${remaining}` : '$components';
    }
  }
  
  // Special handling for $store - it should point to pkg-store
  // but files might be in pkg-store/stores
  if (alias === '$store' && relativeFile.startsWith('stores/')) {
    const remaining = relativeFile.replace(/^stores\//, '');
    return remaining ? `$store/${remaining}` : '$store';
  }
  
  // Special handling for $services - it should point to pkg-services
  // but files might be in pkg-services/services
  if (alias === '$services' && relativeFile.startsWith('services/')) {
    const remaining = relativeFile.replace(/^services\//, '');
    return remaining ? `$services/${remaining}` : '$services';
  }
  
  // Check if there's a subdirectory mapping
  const subdirMappings = SUBDIR_MAPPINGS[targetPackage] || {};

  for (const [subdir, targetSubdir] of Object.entries(subdirMappings)) {
    if (relativeFile.startsWith(subdir)) {
      const remaining = relativeFile.slice(subdir.length).replace(/^\//, '');
      if (!remaining) return alias;
      return `${alias}/${remaining}`;
    }
  }

  // Default: use package alias with relative path
  return `${alias}/${relativeFile.replace(/\.(ts|js|svelte)$/, '')}`;
}

function fixImport(importInfo: ImportInfo): FileFix | null {
  const { file, line, importPath, symbols, originalLine } = importInfo;
  const packageName = getPackageName(file);

  if (!packageName) return null;

  // Try to resolve the import path
  let resolvedPath: string | null = null;

  // Handle alias imports
  if (importPath.startsWith('$')) {
    // This is already an alias import, check if it's valid
    const aliasMatch = importPath.match(/^\$(\w+)/);
    if (aliasMatch) {
      const alias = aliasMatch[0];
      // Map alias to package
      const targetPackage = Object.entries(PACKAGE_TO_ALIAS).find(([_, a]) => a === alias)?.[0];
      if (targetPackage) {
        // Check if symbols exist in this package
        for (const symbol of symbols) {
          const locations = symbolIndex.get(symbol);
          if (locations) {
            const packageLocations = locations.filter(loc => loc.packageName === targetPackage);
            if (packageLocations.length === 0) {
              // Symbol not found in this package, search elsewhere
              resolvedPath = searchAndConstructPath(symbol, file, targetPackage);
              if (resolvedPath) {
                return {
                  file,
                  line,
                  oldImport: importPath,
                  newImport: resolvedPath,
                  reason: `Symbol '${symbol}' not in ${targetPackage}, found elsewhere`
                };
              }
            }
          }
        }
      }
    }
    return null; // Import is valid
  }

  // Handle relative imports
  if (importPath.startsWith('.')) {
    const fullPath = resolve(dirname(file), importPath);
    
    // Check if file exists
    let targetFile = fullPath;
    if (!existsSync(targetFile)) {
      if (existsSync(targetFile + '.ts')) targetFile += '.ts';
      else if (existsSync(targetFile + '.js')) targetFile += '.js';
      else if (existsSync(targetFile + '.svelte')) targetFile += '.svelte';
      else if (existsSync(join(targetFile, 'index.ts'))) targetFile = join(targetFile, 'index.ts');
      else if (existsSync(join(targetFile, 'index.js'))) targetFile = join(targetFile, 'index.js');
    }

    if (existsSync(targetFile)) {
      // File exists, but check if symbols are exported
      const targetPackage = getPackageName(targetFile);
      if (targetPackage && targetPackage !== packageName) {
        // Should use alias instead of relative
        resolvedPath = constructImportPath(targetPackage, targetFile, file);
        if (resolvedPath && resolvedPath !== importPath) {
          return {
            file,
            line,
            oldImport: importPath,
            newImport: resolvedPath,
            reason: 'Convert relative import to alias'
          };
        }
      }
      return null; // Import is valid
    }

    // File doesn't exist - search for symbols
    for (const symbol of symbols) {
      const path = searchAndConstructPath(symbol, file, packageName);
      if (path) {
        return {
          file,
          line,
          oldImport: importPath,
          newImport: path,
          reason: `File not found, symbol '${symbol}' located elsewhere`
        };
      }
    }

    // Symbol not found
    for (const symbol of symbols) {
      unresolvedImports.push({
        file: relative(FRONTEND_DIR, file),
        symbol,
        importPath
      });
    }

    return null;
  }

  return null; // External dependency, keep as is
}

function searchAndConstructPath(symbol: string, sourceFile: string, expectedPackage?: string): string | null {
  const locations = symbolIndex.get(symbol);

  if (!locations || locations.length === 0) {
    return null;
  }

  // Check for duplicates across packages
  const uniquePackages = new Set(locations.map(loc => loc.packageName));
  if (uniquePackages.size > 1) {
    // Duplicate found!
    const warning = `⚠️  Duplicate symbol '${symbol}' found in multiple locations:\n` +
      locations.map(loc => `    - ${loc.packageName}: ${relative(FRONTEND_DIR, loc.file)}`).join('\n');
    
    if (!duplicateWarnings.includes(warning)) {
      duplicateWarnings.push(warning);
    }
    return null;
  }

  // Single location found
  const location = locations[0];
  return constructImportPath(location.packageName, location.file, sourceFile);
}

function applyFix(filePath: string, fix: FileFix): string {
  const content = readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  
  const line = lines[fix.line];
  const newLine = line.replace(fix.oldImport, fix.newImport);
  lines[fix.line] = newLine;
  
  return lines.join('\n');
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('🔧 Intelligent Import Path Fixer');
  console.log('='.repeat(80));
  console.log();
  log(`Frontend directory: ${FRONTEND_DIR}`, 'info');
  log(`Dry run: ${DRY_RUN}`, DRY_RUN ? 'warning' : 'info');
  log(`Verbose: ${VERBOSE}`, 'info');
  console.log();

  if (DRY_RUN) {
    log('⚠️  DRY RUN MODE - No files will be modified', 'warning');
    console.log();
  }

  // Step 1: Build symbol index
  buildSymbolIndex();

  // Step 2: Collect all files
  log('📂 Collecting files to analyze...\n', 'info');
  
  const allFiles: string[] = [];
  for (const pkg of PACKAGES) {
    const packagePath = join(FRONTEND_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }

  log(`   Found ${allFiles.length} files\n`, 'info');

  // Step 3: Fix imports
  log('🔍 Analyzing and fixing imports...\n', 'info');

  for (const file of allFiles) {
    try {
      const imports = parseImports(file);
      let fileHasFixes = false;

      for (const importInfo of imports) {
        const fix = fixImport(importInfo);
        if (fix) {
          fileFixes.push(fix);
          fileHasFixes = true;

          if (VERBOSE) {
            const relativeFile = relative(FRONTEND_DIR, file);
            log(`  ${relativeFile}:${importInfo.line + 1}`, 'info');
            log(`    ${fix.reason}`, 'info');
            log(`    ${fix.oldImport} → ${fix.newImport}`, 'success');
          }
        }
      }

      // Apply fixes for this file
      if (fileHasFixes && !DRY_RUN) {
        const content = readFileSync(file, 'utf-8');
        const lines = content.split('\n');
        
        for (const fix of fileFixes.filter(f => f.file === file)) {
          const line = lines[fix.line];
          const newLine = line.replace(fix.oldImport, fix.newImport);
          lines[fix.line] = newLine;
        }
        
        writeFileSync(file, lines.join('\n'), 'utf-8');
      }
    } catch (error) {
      log(`   Error processing ${file}: ${error}`, 'error');
    }
  }

  // Step 4: Print summary
  console.log('\n' + '='.repeat(80));
  console.log('SUMMARY');
  console.log('='.repeat(80));
  console.log();
  
  log(`Files processed: ${allFiles.length}`, 'info');
  log(`Imports fixed: ${fileFixes.length}`, fileFixes.length > 0 ? 'success' : 'info');
  log(`Unresolved imports: ${unresolvedImports.length}`, unresolvedImports.length > 0 ? 'error' : 'info');
  log(`Duplicate warnings: ${duplicateWarnings.length}`, duplicateWarnings.length > 0 ? 'warning' : 'info');
  console.log();

  // Print duplicate warnings
  if (duplicateWarnings.length > 0) {
    console.log('⚠️  DUPLICATE SYMBOL WARNINGS');
    console.log('─'.repeat(80));
    console.log();
    console.log('The following symbols exist in multiple locations.');
    console.log('This suggests files were COPIED instead of MOVED.');
    console.log('You need to manually delete or rename the duplicates.\n');
    
    for (const warning of duplicateWarnings) {
      console.log(warning);
      console.log();
    }
    console.log('─'.repeat(80));
    console.log();
  }

  // Print unresolved imports
  if (unresolvedImports.length > 0) {
    console.log('❌ UNRESOLVED IMPORTS');
    console.log('─'.repeat(80));
    console.log();
    console.log('The following imports could not be resolved:\n');
    
    for (const imp of unresolvedImports.slice(0, 20)) {
      console.log(`  ${imp.file}: '${imp.symbol}' from '${imp.importPath}'`);
    }
    if (unresolvedImports.length > 20) {
      console.log(`  ... and ${unresolvedImports.length - 20} more`);
    }
    console.log();
    console.log('─'.repeat(80));
    console.log();
  }

  // Print fixes
  if (fileFixes.length > 0 && !VERBOSE) {
    console.log('📋 IMPORTS FIXED');
    console.log('─'.repeat(80));
    console.log();
    
    for (const fix of fileFixes.slice(0, 20)) {
      const relativeFile = relative(FRONTEND_DIR, fix.file);
      console.log(`  ${relativeFile}:${fix.line + 1}`);
      console.log(`    ${fix.oldImport} → ${fix.newImport}`);
      console.log(`    Reason: ${fix.reason}`);
      console.log();
    }
    if (fileFixes.length > 20) {
      console.log(`  ... and ${fileFixes.length - 20} more fixes`);
    }
    console.log();
    console.log('─'.repeat(80));
    console.log();
  }

  // Print next steps
  console.log('📋 NEXT STEPS');
  console.log('='.repeat(80));
  console.log();
  
  if (duplicateWarnings.length > 0) {
    console.log('1. ⚠️  CRITICAL: Fix duplicate symbols listed above');
    console.log('   - Delete the unwanted duplicates OR');
    console.log('   - Rename them to avoid conflicts');
    console.log();
  }
  
  if (unresolvedImports.length > 0) {
    console.log('2. ❌ Fix unresolved imports:');
    console.log('   - Check if files are missing');
    console.log('   - Update import paths manually if needed');
    console.log();
  }
  
  if (!DRY_RUN) {
    console.log('3. Run: bun scripts/analyze-dag.ts');
    console.log('4. Run: bun scripts/check-imports.ts');
    console.log('5. Test the application');
  } else {
    console.log('3. Review the changes above');
    console.log('4. Run again without --dry-run to apply fixes');
    console.log('5. Then run: bun scripts/analyze-dag.ts');
  }
  
  console.log();
  console.log('='.repeat(80));

  // Exit with appropriate code
  const hasErrors = unresolvedImports.length > 0 || duplicateWarnings.length > 0;
  process.exit(hasErrors ? 1 : 0);
}

main();