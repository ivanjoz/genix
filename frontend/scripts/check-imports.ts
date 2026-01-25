#!/usr/bin/env bun
/**
 * Import Path Validator and Fixer
 * 
 * This script validates import paths across all packages and can automatically fix them.
 * Usage:
 *   bun run scripts/check-imports.ts              # Check imports
 *   bun run scripts/check-imports.ts --fix        # Check and fix imports
 *   bun run scripts/check-imports.ts --fix --verbose # Fix with detailed output
 */

import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from 'fs';
import { resolve, relative, dirname, join, isAbsolute } from 'path';

// ============================================
// Types
// ============================================

interface ExportInfo {
  file: string;
  symbol: string;
  type: 'default' | 'named' | 'type';
  isSvelteComponent: boolean;
}

interface ImportInfo {
  file: string;
  line: number;
  column: number;
  symbol: string;
  originalSymbol: string;
  fromPath: string;
  type: 'default' | 'named' | 'type' | 'namespace';
  isSvelteComponent: boolean;
  fullLine: string;
  importIndex: number;
}

interface FileImportInfo {
  file: string;
  imports: ImportInfo[];
}

interface Config {
  aliases: Record<string, string>;
}

interface ResolutionResult {
  valid: boolean;
  resolvedFile?: string;
  resolvedPath?: string;
  suggestions?: string[];
  message?: string;
}

// ============================================
// Global State
// ============================================

const FIX_MODE = process.argv.includes('--fix');
const VERBOSE_MODE = process.argv.includes('--verbose');
const DRY_RUN = process.argv.includes('--dry-run');

const FRONTEND_DIR = resolve(process.cwd());
const PACKAGES_DIR = FRONTEND_DIR;
const PACKAGES = ['pkg-main', 'pkg-store', 'pkg-ui', 'pkg-core', 'pkg-ecommerce', 'pkg-services'];

// Maps for tracking all exports
const exportsByFile = new Map<string, ExportInfo[]>();
const exportsByDirectory = new Map<string, ExportInfo[]>();
const symbolToExports = new Map<string, ExportInfo[]>();

// Configuration from svelte.config.js
const configMap = new Map<string, Config>();

// Statistics
let totalFiles = 0;
let totalImports = 0;
let invalidImports = 0;
let fixedImports = 0;
let unfixableImports = 0;

// ============================================
// Configuration Loading
// ============================================

function loadSvelteConfig(packagePath: string): Config {
  const configPath = join(packagePath, 'svelte.config.js');
  
  if (!existsSync(configPath)) {
    return { aliases: {} };
  }

  try {
    const content = readFileSync(configPath, 'utf-8');
    const aliases: Record<string, string> = {};
    
    // Extract alias configuration using regex
    const aliasMatch = content.match(/alias\s*:\s*\{([^}]+)\}/s);
    if (aliasMatch) {
      const aliasBlock = aliasMatch[1];
      const aliasRegex = /['"]?(\$\w+)['"]?\s*:\s*path\.resolve\(['"]([^'"]+)['"]\)/g;
      let match;
      
      while ((match = aliasRegex.exec(aliasBlock)) !== null) {
        aliases[match[1]] = match[2];
      }
    }
    
    return { aliases };
  } catch (error) {
    console.warn(`Warning: Failed to parse svelte.config.js for ${packagePath}`);
    return { aliases: {} };
  }
}

function loadConfigForPackage(packageName: string): Config {
  if (configMap.has(packageName)) {
    return configMap.get(packageName)!;
  }
  
  const packagePath = join(PACKAGES_DIR, packageName);
  const config = loadSvelteConfig(packagePath);
  configMap.set(packageName, config);
  return config;
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
    
    // Skip node_modules, .git, hidden files, build outputs
    if (['node_modules', '.git', '.svelte-kit', 'build', 'dist', '.turbo'].includes(entry.name)) {
      continue;
    }
    if (entry.name.startsWith('.') && entry.name !== '.npmrc') {
      continue;
    }

    if (entry.isDirectory()) {
      files.push(...getAllFiles(fullPath, baseDir));
    } else if (entry.isFile()) {
      // Only process relevant files
      const ext = entry.name;
      if (['.ts', '.tsx', '.svelte', '.js', '.jsx'].some(e => ext.endsWith(e))) {
        files.push(fullPath);
      }
    }
  }

  return files;
}

// ============================================
// Import Parsing
// ============================================

function parseImports(content: string, filePath: string): ImportInfo[] {
  const imports: ImportInfo[] = [];
  const lines = content.split('\n');

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      continue;
    }

    // Match various import statement patterns
    const patterns = [
      // import * as name from 'path'
      {
        regex: /import\s+\*\s+as\s+(\w+)\s+from\s+['"]([^'"]+)['"]/,
        type: 'namespace'
      },
      // import { a, b as c, type d } from 'path'
      {
        regex: /import\s+\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
        type: 'named'
      },
      // import type { a, b } from 'path'
      {
        regex: /import\s+type\s+\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
        type: 'type'
      },
      // import default from 'path'
      {
        regex: /import\s+(\w+)\s+from\s+['"]([^'"]+)['"]/,
        type: 'default'
      }
    ];

    for (const pattern of patterns) {
      const match = trimmed.match(pattern.regex);
      if (match) {
        const symbols = match[1];
        const fromPath = match[2];
        const type = pattern.type;

        // Find the column where import starts
        const importIndex = line.indexOf('import');
        const column = importIndex;

        if (type === 'namespace') {
          imports.push({
            file: filePath,
            line: i + 1,
            column,
            symbol: symbols,
            originalSymbol: symbols,
            fromPath,
            type,
            isSvelteComponent: fromPath.endsWith('.svelte'),
            fullLine: line,
            importIndex
          });
        } else {
          // Parse named imports (can include 'as' and 'type' modifiers)
          const symbolList = symbols.split(',').map(s => s.trim());
          for (const symbol of symbolList) {
            if (!symbol) continue;

            // Handle 'type a' or 'a as b' patterns
            const parts = symbol.split(/\s+/);
            let cleanSymbol = parts[0];
            
            if (parts[0] === 'type') {
              cleanSymbol = parts[1];
            }

            imports.push({
              file: filePath,
              line: i + 1,
              column,
              symbol: cleanSymbol,
              originalSymbol: symbol,
              fromPath,
              type,
              isSvelteComponent: fromPath.endsWith('.svelte'),
              fullLine: line,
              importIndex
            });
          }
        }
      }
    }
  }

  return imports;
}

// ============================================
// Export Parsing
// ============================================

function parseExports(content: string, filePath: string): ExportInfo[] {
  const exports: ExportInfo[] = [];
  const lines = content.split('\n');

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    // Skip comments and empty lines
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      continue;
    }

    // Svelte component: <script lang="ts"> export let name; or <script> export const name = ...
    if (filePath.endsWith('.svelte')) {
      const componentMatch = filePath.match(/([^/]+)\.svelte$/);
      if (componentMatch) {
        const componentName = componentMatch[1];
        exports.push({
          file: filePath,
          symbol: componentName,
          type: 'default',
          isSvelteComponent: true
        });
      }

      // Named exports in Svelte
      const svelteExportRegex = /export\s+(let|const|function|class)\s+(\w+)/;
      const svelteMatch = trimmed.match(svelteExportRegex);
      if (svelteMatch) {
        exports.push({
          file: filePath,
          symbol: svelteMatch[2],
          type: 'named',
          isSvelteComponent: false
        });
      }

      // Type exports in Svelte
      const svelteTypeRegex = /export\s+type\s+(\w+)/;
      const svelteTypeMatch = trimmed.match(svelteTypeRegex);
      if (svelteTypeMatch) {
        exports.push({
          file: filePath,
          symbol: svelteTypeMatch[1],
          type: 'type',
          isSvelteComponent: false
        });
      }

      continue;
    }

    // Default export: export default name;
    const defaultMatch = trimmed.match(/export\s+default\s+(?:class\s+)?(\w+)/);
    if (defaultMatch) {
      exports.push({
        file: filePath,
        symbol: defaultMatch[1],
        type: 'default',
        isSvelteComponent: false
      });
      continue;
    }

    // Named export: export const/let/function/class name
    const namedExportRegex = /export\s+(?:const|let|function|class|interface|type)\s+(\w+)/;
    const namedMatch = trimmed.match(namedExportRegex);
    if (namedMatch) {
      exports.push({
        file: filePath,
        symbol: namedMatch[1],
        type: 'named',
        isSvelteComponent: false
      });
      continue;
    }

    // Destructured export: export const { a, b } = pkg
    const destructuredExportRegex = /export\s+const\s+\{\s*([^}]+)\s*\}\s*=/;
    const destructuredMatch = trimmed.match(destructuredExportRegex);
    if (destructuredMatch) {
      const symbols = destructuredMatch[1].split(',').map(s => s.trim());
      for (const symbol of symbols) {
        if (symbol) {
          exports.push({
            file: filePath,
            symbol: symbol,
            type: 'named',
            isSvelteComponent: false
          });
        }
      }
      continue;
    }

    // Export from: export { a, b as c } from 'path'
    const reExportRegex = /export\s*\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/;
    const reExportMatch = trimmed.match(reExportRegex);
    if (reExportMatch) {
      const symbols = reExportMatch[1].split(',').map(s => s.trim());
      for (const symbol of symbols) {
        if (symbol) {
          // Handle 'as' pattern
          const parts = symbol.split(/\s+/);
          exports.push({
            file: filePath,
            symbol: parts[0],
            type: 'named',
            isSvelteComponent: false
          });
        }
      }
      continue;
    }

    // Type export: export type { name }
    const typeExportRegex = /export\s+type\s*\{\s*([^}]+)\s*\}/;
    const typeExportMatch = trimmed.match(typeExportRegex);
    if (typeExportMatch) {
      const symbols = typeExportMatch[1].split(',').map(s => s.trim());
      for (const symbol of symbols) {
        if (symbol) {
          exports.push({
            file: filePath,
            symbol: symbol,
            type: 'type',
            isSvelteComponent: false
          });
        }
      }
      continue;
    }
  }

  return exports;
}

// ============================================
// Path Resolution
// ============================================

function resolveImportPath(importPath: string, sourceFile: string, packageName: string): ResolutionResult {
  const config = loadConfigForPackage(packageName);
  const packageDir = join(PACKAGES_DIR, packageName);
  const sourceDir = dirname(sourceFile);

  // Handle SvelteKit built-in aliases ($app/*)
  if (importPath.startsWith('$app/')) {
    return {
      valid: true,
      message: 'SvelteKit built-in - not validated'
    };
  }

  // Handle alias imports ($components, $core, etc.)
  if (importPath.startsWith('$')) {
    const aliasName = Object.keys(config.aliases).find(alias => importPath.startsWith(alias));
    
    if (aliasName) {
      const aliasPath = config.aliases[aliasName];
      const relativePath = importPath.slice(aliasName.length);
      const resolvedPath = join(packageDir, aliasPath, relativePath);
      
      if (existsSync(resolvedPath)) {
        return {
          valid: true,
          resolvedPath: resolvedPath,
          resolvedFile: resolvedPath
        };
      } else if (existsSync(resolvedPath + '.ts')) {
        return {
          valid: true,
          resolvedPath: resolvedPath + '.ts',
          resolvedFile: resolvedPath + '.ts'
        };
      } else if (existsSync(resolvedPath + '.svelte')) {
        return {
          valid: true,
          resolvedPath: resolvedPath + '.svelte',
          resolvedFile: resolvedPath + '.svelte'
        };
      } else if (existsSync(resolvedPath + '.js')) {
        return {
          valid: true,
          resolvedPath: resolvedPath + '.js',
          resolvedFile: resolvedPath + '.js'
        };
      }
      
      return {
        valid: false,
        message: `Alias path does not exist: ${resolvedPath}`
      };
    }
    
    return {
      valid: false,
      message: `Unknown alias: ${importPath}`
    };
  }

  // Handle relative imports (./, ../)
  if (importPath.startsWith('.')) {
    const resolvedPath = resolve(sourceDir, importPath);
    
    if (existsSync(resolvedPath)) {
      return {
        valid: true,
        resolvedPath: resolvedPath,
        resolvedFile: resolvedPath
      };
    } else if (existsSync(resolvedPath + '.ts')) {
      return {
        valid: true,
        resolvedPath: resolvedPath + '.ts',
        resolvedFile: resolvedPath + '.ts'
      };
    } else if (existsSync(resolvedPath + '.svelte')) {
      return {
        valid: true,
        resolvedPath: resolvedPath + '.svelte',
        resolvedFile: resolvedPath + '.svelte'
      };
    } else if (existsSync(resolvedPath + '.js')) {
      return {
        valid: true,
        resolvedPath: resolvedPath + '.js',
        resolvedFile: resolvedPath + '.js'
      };
    }
    
    return {
      valid: false,
      message: `Relative path does not exist: ${resolvedPath}`
    };
  }

  // Handle node_modules imports
  return {
    valid: true,
    message: 'External dependency - not validated'
  };
}

// ============================================
// Symbol Validation
// ============================================

function findSymbol(symbol: string, resolvedPath: string, importType: string): boolean {
  // If it's a directory (package), check exports from index or all files
  if (existsSync(resolvedPath) && statSync(resolvedPath).isDirectory()) {
    const indexPath = join(resolvedPath, 'index.ts');
    if (existsSync(indexPath)) {
      const exports = exportsByFile.get(indexPath) || [];
      return exports.some(e => e.symbol === symbol);
    }
    
    // Check all files in the directory
    const allExports: ExportInfo[] = [];
    for (const [file, fileExports] of exportsByFile) {
      if (file.startsWith(resolvedPath)) {
        allExports.push(...fileExports);
      }
    }
    return allExports.some(e => e.symbol === symbol);
  }

  // If it's a file, check its exports
  if (existsSync(resolvedPath)) {
    const exports = exportsByFile.get(resolvedPath) || [];
    
    // For default imports, check if the file has a default export or is a Svelte component
    if (importType === 'default') {
      return exports.some(e => e.type === 'default') || resolvedPath.endsWith('.svelte');
    }
    
    // For named imports, check if the symbol is exported
    return exports.some(e => e.symbol === symbol);
  }

  return false;
}

function searchForSymbol(symbol: string, packageName: string): string[] {
  const suggestions: string[] = [];
  
  for (const [file, exports] of exportsByFile) {
    const matchingExport = exports.find(e => e.symbol === symbol);
    if (matchingExport) {
      // Convert absolute path to relative from package root
      const packagePath = join(PACKAGES_DIR, packageName);
      let relativePath = relative(packagePath, file);
      
      // Remove extension for cleaner paths
      if (relativePath.endsWith('.ts') || relativePath.endsWith('.js') || relativePath.endsWith('.svelte')) {
        relativePath = relativePath.slice(0, -3);
      }
      
      suggestions.push(`$/${relativePath}`);
      suggestions.push(relativePath);
    }
  }
  
  return suggestions;
}

// ============================================
// Validation and Fixing
// ============================================

function validateImport(importInfo: ImportInfo): ResolutionResult {
  const { fromPath, file, symbol, type, isSvelteComponent } = importInfo;
  
  // Determine which package this file belongs to
  const relativeFile = relative(FRONTEND_DIR, file);
  const packageName = PACKAGES.find(pkg => relativeFile.startsWith(pkg));
  
  if (!packageName) {
    return {
      valid: true,
      message: 'File not in a known package'
    };
  }

  // Resolve the import path
  const resolution = resolveImportPath(fromPath, file, packageName);
  
  if (!resolution.valid) {
    return resolution;
  }

  // Skip symbol validation for SvelteKit built-ins and external dependencies
  if (resolution.message === 'SvelteKit built-in - not validated' || 
      resolution.message === 'External dependency - not validated') {
    return { valid: true };
  }

  // For Svelte components, just check if the file exists
  if (isSvelteComponent) {
    if (resolution.resolvedFile && existsSync(resolution.resolvedFile)) {
      return { valid: true };
    }
    return { valid: false, message: 'Svelte component file not found' };
  }

  // For other imports, validate the symbol
  if (resolution.resolvedFile) {
    const symbolExists = findSymbol(symbol, resolution.resolvedFile, type);
    
    if (!symbolExists) {
      // Search for alternative locations
      const suggestions = searchForSymbol(symbol, packageName);
      return {
        valid: false,
        message: `Symbol '${symbol}' not found in ${resolution.resolvedFile}`,
        suggestions
      };
    }
  }

  return { valid: true };
}

function fixImport(importInfo: ImportInfo, resolution: ResolutionResult): boolean {
  const { file, line, fromPath, symbol, suggestions, fullLine } = importInfo;
  
  if (!suggestions || suggestions.length === 0) {
    return false;
  }

  // Don't try to fix SvelteKit built-ins
  if (resolution.message === 'SvelteKit built-in - not validated') {
    return false;
  }

  if (VERBOSE_MODE) {
    console.log(`  Fixing: ${symbol} -> ${suggestions[0]}`);
  }

  // Read the file
  let content: string;
  try {
    content = readFileSync(file, 'utf-8');
  } catch (error) {
    console.error(`  Error reading file ${file}:`, error);
    return false;
  }

  const lines = content.split('\n');
  const lineIndex = line - 1;

  if (lineIndex < 0 || lineIndex >= lines.length) {
    return false;
  }

  // Replace the import path
  const newLine = fullLine.replace(fromPath, suggestions[0]);
  lines[lineIndex] = newLine;

  // Write back to file (unless dry run)
  if (!DRY_RUN) {
    try {
      writeFileSync(file, lines.join('\n'), 'utf-8');
      return true;
    } catch (error) {
      console.error(`  Error writing file ${file}:`, error);
      return false;
    }
  }

  return true;
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('🔍 Starting import path validation...\n');
  console.log(`Fix mode: ${FIX_MODE ? 'ON' : 'OFF'}`);
  console.log(`Verbose mode: ${VERBOSE_MODE ? 'ON' : 'OFF'}`);
  console.log(`Dry run: ${DRY_RUN ? 'ON' : 'OFF'}\n`);

  // Step 1: Collect all files
  console.log('📂 Collecting files...');
  const allFiles: string[] = [];
  for (const pkg of PACKAGES) {
    const packagePath = join(PACKAGES_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }
  totalFiles = allFiles.length;
  console.log(`   Found ${totalFiles} files\n`);

  // Step 2: Parse exports from all files
  console.log('📦 Parsing exports...');
  for (const file of allFiles) {
    try {
      const content = readFileSync(file, 'utf-8');
      const exports = parseExports(content, file);
      
      if (exports.length > 0) {
        exportsByFile.set(file, exports);
        
        // Also index by directory
        const dir = dirname(file);
        if (!exportsByDirectory.has(dir)) {
          exportsByDirectory.set(dir, []);
        }
        exportsByDirectory.get(dir)!.push(...exports);
        
        // Index by symbol
        for (const exp of exports) {
          if (!symbolToExports.has(exp.symbol)) {
            symbolToExports.set(exp.symbol, []);
          }
          symbolToExports.get(exp.symbol)!.push(exp);
        }
      }
    } catch (error) {
      console.warn(`   Warning: Could not parse exports from ${file}`);
    }
  }
  
  const totalExports = Array.from(exportsByFile.values()).reduce((sum, exps) => sum + exps.length, 0);
  console.log(`   Parsed ${totalExports} exports from ${exportsByFile.size} files\n`);

  // Step 3: Parse imports from all files
  console.log('📥 Parsing imports...');
  const allImports: ImportInfo[] = [];
  for (const file of allFiles) {
    try {
      const content = readFileSync(file, 'utf-8');
      const imports = parseImports(content, file);
      
      if (imports.length > 0) {
        allImports.push(...imports);
        totalImports += imports.length;
      }
    } catch (error) {
      console.warn(`   Warning: Could not parse imports from ${file}`);
    }
  }
  console.log(`   Parsed ${totalImports} imports\n`);

  // Step 4: Validate imports
  console.log('✅ Validating imports...\n');
  const invalidImportInfos: { importInfo: ImportInfo; resolution: ResolutionResult }[] = [];

  for (const importInfo of allImports) {
    const resolution = validateImport(importInfo);
    
    if (!resolution.valid) {
      invalidImports++;
      invalidImportInfos.push({ importInfo, resolution });

      const { file, line, symbol, fromPath } = importInfo;
      const relativeFile = relative(FRONTEND_DIR, file);
      
      console.log(`❌ Invalid import in ${relativeFile}:${line}`);
      console.log(`   Symbol: ${symbol}`);
      console.log(`   From: ${fromPath}`);
      console.log(`   Reason: ${resolution.message}`);
      
      if (resolution.suggestions && resolution.suggestions.length > 0) {
        console.log(`   Suggestions: ${resolution.suggestions.join(', ')}`);
      }
      console.log();
    }
  }

  // Step 5: Fix imports if in fix mode
  if (FIX_MODE && invalidImportInfos.length > 0) {
    console.log('🔧 Fixing imports...\n');
    
    for (const { importInfo, resolution } of invalidImportInfos) {
      if (resolution.suggestions && resolution.suggestions.length > 0) {
        const success = fixImport(importInfo, resolution);
        if (success) {
          fixedImports++;
          if (VERBOSE_MODE) {
            console.log(`   ✅ Fixed: ${importInfo.symbol} in ${relative(FRONTEND_DIR, importInfo.file)}`);
          }
        } else {
          unfixableImports++;
          console.log(`   ❌ Could not fix: ${importInfo.symbol} in ${relative(FRONTEND_DIR, importInfo.file)}`);
        }
      } else {
        unfixableImports++;
      }
    }
    
    console.log();
  }

  // Step 6: Print summary
  console.log('='.repeat(60));
  console.log('SUMMARY');
  console.log('='.repeat(60));
  console.log(`Total files scanned: ${totalFiles}`);
  console.log(`Total imports checked: ${totalImports}`);
  console.log(`Invalid imports found: ${invalidImports}`);
  if (FIX_MODE) {
    console.log(`Imports fixed: ${fixedImports}`);
    console.log(`Unfixable imports: ${unfixableImports}`);
  }
  console.log('='.repeat(60));

  if (invalidImports > 0) {
    process.exit(1);
  }
}

// Run the script
main();