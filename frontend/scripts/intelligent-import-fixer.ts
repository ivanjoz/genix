#!/usr/bin/env -S bun
/**
 * Intelligent Import Fixer
 * 
 * This script analyzes and fixes import statements by:
 * 1. Indexing all exported symbols across the entire codebase
 * 2. Detecting broken imports that reference non-existent files/symbols
 * 3. Finding the correct location of missing symbols
 * 4. Generating a detailed report and optionally fixing issues
 * 
 * Usage:
 *   bun run scripts/intelligent-import-fixer.ts                    # Show report only
 *   bun run scripts/intelligent-import-fixer.ts --fix              # Fix all issues
 *   bun run scripts/intelligent-import-fixer.ts --fix --dry-run     # Preview fixes
 */

import fs from 'fs';
import path from 'path';

const PROJECT_ROOT = process.cwd();
const SHOULD_FIX = process.argv.includes('--fix');
const DRY_RUN = process.argv.includes('--dry-run');

// Generic names to skip when searching by filename
const GENERIC_NAMES = new Set([
  'index', 'main', 'app', 'config', 'types', 'utils', 'helpers', 
  'store', 'state', 'api', 'routes', 'pages', 'components',
  '+page', '+layout', '+page.svelte', '+layout.svelte'
]);

// Frameworks and npm packages to exclude from analysis
const EXCLUDED_IMPORTS = new Set([
  // Svelte/SvelteKit framework
  'svelte',
  '$app/environment',
  '$app/state',
  '$app/stores',
  '$app/navigation',
  '$app/paths',
  '$app/modules',
  // Common npm packages (will be resolved by package manager)
  'notiflix',
  'roosterjs',
  'svelte-awesome-color-picker',
  '@humanspeak/svelte-virtual-list',
  '@mediapipe/tasks-genai',
  '@huggingface/transformers',
  '@xenova/transformers',
  // Add more as needed
]);

// Type definitions
interface ExportedSymbol {
  name: string;
  type: 'named' | 'default' | 'file';
  filePath: string;
  lineNumber: number;
}

interface ImportStatement {
  raw: string;
  importPath: string;
  imports: string[];
  isDefault: boolean;
  isNamed: boolean;
  isSideEffect: boolean;
  lineNumber: number;
  filePath: string;
}

interface ImportIssue {
  filePath: string;
  lineNumber: number;
  importStatement: ImportStatement;
  missingFile: boolean;
  missingSymbols: string[];
  foundIn?: ExportedSymbol[];
  suggestedPath?: string;
  actualFileLocation?: string;
}

interface FixReport {
  totalFiles: number;
  totalImports: number;
  totalIssues: number;
  issuesByType: {
    missingFile: number;
    missingSymbols: number;
  };
  issues: ImportIssue[];
}

// Global symbol index
const symbolIndex = new Map<string, ExportedSymbol[]>();
const fileExports = new Map<string, ExportedSymbol>();

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
 * Check if a path exists (with or without extensions)
 */
function pathExists(basePath: string): string | null {
  // Remove query parameters if present (e.g., ?raw, ?worker)
  const cleanPath = basePath.split('?')[0];
  
  // First, check if path itself is a file (handles paths with extensions already)
  if (fs.existsSync(cleanPath) && fs.statSync(cleanPath).isFile()) {
    return cleanPath;
  }
  
  // Check for TypeScript/JavaScript/Svelte files
  const codeExtensions = ['.ts', '.js', '.svelte'];
  for (const ext of codeExtensions) {
    const fullPath = cleanPath + ext;
    if (fs.existsSync(fullPath) && fs.statSync(fullPath).isFile()) {
      return fullPath;
    }
  }
  
  // Check for common asset files
  const assetExtensions = ['.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.ico', '.css', '.json'];
  for (const ext of assetExtensions) {
    const fullPath = cleanPath + ext;
    if (fs.existsSync(fullPath) && fs.statSync(fullPath).isFile()) {
      return fullPath;
    }
  }
  
  // Check if it's a directory (for directory imports)
  if (fs.existsSync(cleanPath) && fs.statSync(cleanPath).isDirectory()) {
    return cleanPath;
  }
  
  return null;
}

/**
 * Resolve an alias path to filesystem path
 */
function resolveAliasPath(aliasPath: string): string | null {
  const parts = aliasPath.split('/');
  const alias = parts[0];
  const rest = parts.slice(1).join('/');
  
  const aliasMap: Record<string, string> = {
    '$lib': 'pkg-app/lib',
    '$core': 'pkg-core',
    '$store': 'pkg-store/stores',
    '$routes': 'pkg-app/routes',
    '$components': 'pkg-ui/components',
    '$shared': 'pkg-services/shared',
    '$ecommerce': 'pkg-components/ecommerce',
    '$services': 'pkg-services/services'
  };
  
  const baseDir = aliasMap[alias];
  if (!baseDir) return null;
  
  let basePath = path.join(PROJECT_ROOT, baseDir);
  
  // For $core, try multiple subdirectories
  if (alias === '$core') {
    const possiblePaths = [
      path.join(basePath, 'lib', rest),
      path.join(basePath, 'core', rest),
      path.join(basePath, 'assets', rest),
      path.join(basePath, rest)
    ];
    
    for (const possiblePath of possiblePaths) {
      const resolved = pathExists(possiblePath);
      if (resolved) return resolved;
    }
    return null;
  }
  
  const resolved = pathExists(path.join(baseDir, rest));
  return resolved ? resolved : null;
}

/**
 * Search for a file by name across the entire project
 */
function searchForFile(fileName: string, extensions: string[] = []): string | null {
  // Remove query parameters (e.g., ?raw)
  const cleanFileName = fileName.split('?')[0];
  
  // Build list of extensions to try
  const exts = extensions.length > 0 ? extensions : ['.ts', '.js', '.svelte', '.svg', '.css'];
  
  // If already has an extension, use only that
  if (path.extname(cleanFileName)) {
    const pathsToSearch = [
      'pkg-app/lib',
      'pkg-app/routes',
      'pkg-app/static',
      'pkg-core/lib',
      'pkg-core/core',
      'pkg-core/assets',
      'pkg-ui/assets',
      'pkg-ui/components',
      'pkg-services/shared',
      'pkg-services/services',
      'pkg-components/ecommerce',
      'pkg-store/stores'
    ];
    
    for (const searchDir of pathsToSearch) {
      const dirPath = path.join(PROJECT_ROOT, searchDir);
      if (!fs.existsSync(dirPath)) continue;
      
      const files = findFilesRecursively(dirPath, [path.extname(cleanFileName)]);
      for (const file of files) {
        if (path.basename(file) === cleanFileName) {
          return file;
        }
      }
    }
  } else {
    // Try with different extensions
    for (const ext of exts) {
      const fullFileName = cleanFileName + ext;
      const pathsToSearch = [
        'pkg-app/lib',
        'pkg-app/routes',
        'pkg-app/static',
        'pkg-core/lib',
        'pkg-core/core',
        'pkg-core/assets',
        'pkg-ui/assets',
        'pkg-ui/components',
        'pkg-services/shared',
        'pkg-services/services',
        'pkg-components/ecommerce',
        'pkg-store/stores'
      ];
      
      for (const searchDir of pathsToSearch) {
        const dirPath = path.join(PROJECT_ROOT, searchDir);
        if (!fs.existsSync(dirPath)) continue;
        
        const files = findFilesRecursively(dirPath, [ext]);
        for (const file of files) {
          if (path.basename(file) === fullFileName) {
            return file;
          }
        }
      }
    }
  }
  
  return null;
}

/**
 * Extract all exports from a file
 */
function extractExports(filePath: string): ExportedSymbol[] {
  const content = fs.readFileSync(filePath, 'utf-8');
  const exports: ExportedSymbol[] = [];
  const lines = content.split('\n');
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();
    const lineNumber = i + 1;
    
    // Default exports: export default
    const defaultMatch = trimmed.match(/^export\s+default(?:\s+([A-Za-z_][A-Za-z0-9_]*))?/);
    if (defaultMatch) {
      const fileName = path.basename(filePath, path.extname(filePath));
      // If there's a name after export default, use it; otherwise use filename
      const exportName = defaultMatch[1] || fileName;
      exports.push({
        name: exportName,
        type: 'default',
        filePath,
        lineNumber
      });
      continue;
    }
    
    // Named exports: export const/function/class
    const namedMatch = trimmed.match(/^export\s+(?:const|function|class)\s+(\w+)/);
    if (namedMatch) {
      exports.push({
        name: namedMatch[1],
        type: 'named',
        filePath,
        lineNumber
      });
      continue;
    }
    
    // Type exports: export type/interface/enum
    const typeMatch = trimmed.match(/^export\s+(?:type|interface|enum)\s+(\w+)/);
    if (typeMatch) {
      exports.push({
        name: typeMatch[1],
        type: 'named',
        filePath,
        lineNumber
      });
      continue;
    }
  
    // Destructured exports: export const { a, b } = something
    const destructuredMatch = trimmed.match(/^export\s+const\s+\{\s*([^}]+)\s*\}\s*=/);
    if (destructuredMatch) {
      const names = destructuredMatch[1].split(',').map(n => n.trim());
      for (const name of names) {
        if (name) {
          exports.push({
            name,
            type: 'named',
            filePath,
            lineNumber
          });
        }
      }
      continue;
    }
  
    // Type exports with braces: export type { something } from '...' (must come before generic export { })
    const exportTypeFromMatch = trimmed.match(/^export\s+type\s+\{\s*([^}]+)\s*\}(?:\s+from\s+['"]([^'"]+)['"])?/);
    if (exportTypeFromMatch) {
      const names = exportTypeFromMatch[1].split(',').map(n => n.trim().replace(/^.*\s+as\s+/, ''));
      for (const name of names) {
        if (name) {
          exports.push({
            name,
            type: 'named',
            filePath,
            lineNumber
          });
        }
      }
      continue;
    }

    // Named exports with braces: export { something } (must NOT match export type { })
    const exportFromMatch = trimmed.match(/^export\s+(?!type\s+\{)\{\s*([^}]+)\s*\}(?:\s+from\s+['"]([^'"]+)['"])?/);
    if (exportFromMatch) {
      const names = exportFromMatch[1].split(',').map(n => n.trim().replace(/^.*\s+as\s+/, ''));
      for (const name of names) {
        if (name) {
          exports.push({
            name,
            type: 'named',
            filePath,
            lineNumber
          });
        }
      }
      continue;
    }
  }
  
  // Add file export for .svelte and .svelte.ts files
  const ext = path.extname(filePath);
  if (ext === '.svelte' || ext === '.ts') {
    const fileName = path.basename(filePath, ext);
    if (ext === '.svelte' || fileName.endsWith('.svelte')) {
      const baseName = ext === '.svelte' ? fileName : fileName.replace(/\.svelte$/, '');
      exports.push({
        name: baseName,
        type: 'file',
        filePath,
        lineNumber: 1
      });
    }
  }
  
  return exports;
}

/**
 * Build the global symbol index
 */
function buildSymbolIndex(): void {
  console.log('🔍 Building symbol index...\n');
  
  const searchDirs = [
    'pkg-app',
    'pkg-components',
    'pkg-store',
    'pkg-ui',
    'pkg-services',
    'pkg-core'
  ];
  
  let totalFiles = 0;
  let totalExports = 0;
  
  for (const dir of searchDirs) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;
    
    const files = findFilesRecursively(dirPath, ['.ts', '.svelte']);
    
    for (const filePath of files) {
      const relativePath = path.relative(PROJECT_ROOT, filePath);
      const exports = extractExports(filePath);
      
      for (const exp of exports) {
        // Add to symbol index
        if (!symbolIndex.has(exp.name)) {
          symbolIndex.set(exp.name, []);
        }
        symbolIndex.get(exp.name)!.push(exp);
        
        // Add to file exports map
        if (!fileExports.has(relativePath)) {
          fileExports.set(relativePath, []);
        }
        fileExports.get(relativePath)!.push(exp);
      }
      
      totalFiles++;
      totalExports += exports.length;
    }
  }
  
  console.log(`✅ Indexed ${totalExports} exports from ${totalFiles} files\n`);
}

/**
 * Parse import statements from a file
 */
function parseImports(filePath: string): ImportStatement[] {
  const content = fs.readFileSync(filePath, 'utf-8');
  const imports: ImportStatement[] = [];
  const lines = content.split('\n');
  
  // Patterns for different import types (type-aware patterns must come first!)
const patterns = [
  // Mixed imports with types: import X, { type A, B } from 'path'
  {
    regex: /import\s+(\w+),\s*\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/g,
    isNamed: true,
    isDefault: true,
    isSideEffect: false,
    parseTypeImports: true
  },
  // Named imports with types: import { type A, B } from 'path'
  {
    regex: /import\s+{\s*([^}]+)\s*}\s+from\s+['"]([^'"]+)['"]/g,
    isNamed: true,
    isDefault: false,
    isSideEffect: false,
    parseTypeImports: true
  },
  // Mixed imports: import X, { a, b } from 'path'
  {
    regex: /import\s+(\w+),\s*\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/g,
    isNamed: true,
    isDefault: true,
    isSideEffect: false
  },
  // Named imports: import { a, b } from 'path'
  {
    regex: /import\s+{\s*([^}]+)\s*}\s+from\s+['"]([^'"]+)['"]/g,
    isNamed: true,
    isDefault: false,
    isSideEffect: false
  },
  // Default import: import X from 'path'
  {
    regex: /import\s+(\w+)\s+from\s+['"]([^'"]+)['"]/g,
    isNamed: false,
    isDefault: true,
    isSideEffect: false
  },
  // Side effect import: import 'path'
  {
    regex: /import\s+['"]([^'"]+)['"]/g,
    isNamed: false,
    isDefault: false,
    isSideEffect: true
  }
];
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();
    const lineNumber = i + 1;
    
    if (!trimmed.startsWith('import')) continue;
    
    for (const pattern of patterns) {
      pattern.regex.lastIndex = 0;
      const match = pattern.regex.exec(trimmed);
      
      if (match) {
        if (pattern.isSideEffect) {
          imports.push({
            raw: trimmed,
            importPath: match[1],
            imports: [],
            isDefault: false,
            isNamed: false,
            isSideEffect: true,
            lineNumber,
            filePath
          });
        } else if (pattern.isDefault && pattern.isNamed) {
            // Mixed import
            const defaultImport = match[1];
            const namedImportsRaw = match[2]
              .replace(/^\{|\}$/g, '')  // Remove surrounding braces
              .split(',')
              .map(n => n.trim())
              .filter(n => n.length > 0);
            // For type-aware patterns, strip "type" keyword from imports
            const namedImports = pattern.parseTypeImports && match[2].includes('type')
              ? namedImportsRaw.map(n => n.replace(/^type\s+/, '').trim())
              : namedImportsRaw;
          imports.push({
            raw: trimmed,
            importPath: match[3],
            imports: [defaultImport, ...namedImports],
            isDefault: true,
            isNamed: true,
            isSideEffect: false,
            lineNumber,
            filePath
          });
        } else if (pattern.isDefault) {
          // Default only
          imports.push({
            raw: trimmed,
            importPath: match[2],
            imports: [match[1]],
            isDefault: true,
            isNamed: false,
            isSideEffect: false,
            lineNumber,
            filePath
          });
        } else if (pattern.isNamed) {
          // Named only
          const namedImportsRaw = match[1]
            .replace(/^\{|\}$/g, '')  // Remove surrounding braces
            .split(',')
            .map(n => n.trim())
            .filter(n => n.length > 0);
          // For type-aware patterns, strip "type" keyword from imports
          const namedImports = pattern.parseTypeImports && match[1].includes('type')
            ? namedImportsRaw.map(n => n.replace(/^type\s+/, '').trim())
            : namedImportsRaw;
          imports.push({
            raw: trimmed,
            importPath: match[2],
            imports: namedImports,
            isDefault: false,
            isNamed: true,
            isSideEffect: false,
            lineNumber,
            filePath
          });
        }

        break;
      }
    }
  }
  
  return imports;
}

/**
 * Check if imports resolve correctly
 */
function checkImports(filePath: string): ImportIssue[] {
  const issues: ImportIssue[] = [];
  const imports = parseImports(filePath);
  
  for (const imp of imports) {
    // Skip side effect imports
    if (imp.isSideEffect) continue;
    
    // Skip excluded framework and npm packages
    if (EXCLUDED_IMPORTS.has(imp.importPath)) continue;
    // Skip npm packages (starts with @ or doesn't start with $ or .)
    if (imp.importPath.startsWith('@') || (!imp.importPath.startsWith('$') && !imp.importPath.startsWith('.'))) {
      continue;
    }
    
    // Skip relative imports that aren't broken
    if (!imp.importPath.startsWith('$')) {
      const resolvedPath = path.resolve(path.dirname(filePath), imp.importPath);
      if (pathExists(resolvedPath)) continue;
    }
    
    // Check for asset imports that might need fixing (but don't skip them)
    if (imp.importPath.match(/\.(svg|png|jpg|jpeg|gif|webp|ico|css|json)(\?.*)?$/)) {
      // For asset files without extensions or with query params, check if they exist
      if (!imp.importPath.match(/\.(svg|png|jpg|jpeg|gif|webp|ico|css|json)$/)) {
        // Has query params but extension is at the end - continue checking
      } else {
        // Asset file with extension - continue to check if it needs path correction
      }
    }
    
    // Check if the target file exists
    const resolvedFile = resolveAliasPath(imp.importPath);
    const missingFile = resolvedFile === null;
    
    const missingSymbols: string[] = [];
    const foundIn: ExportedSymbol[] = [];
    let actualFileLocation: string | undefined;
    
    // Check if import path needs correction (e.g., $core/http → $core/lib/http)
    let importPathNeedsFix = false;
    if (!missingFile && imp.importPath.startsWith('$')) {
      const alias = imp.importPath.split('/')[0];
      if (alias === '$core') {
        // For $core imports, check if resolved path includes 'lib' subdirectory
        // but the import path doesn't
        const resolvedRelative = path.relative(PROJECT_ROOT, resolvedFile!);
        const resolvedParts = resolvedRelative.split(path.sep);
        // Debug: log what we're checking
        console.log(`  🔍 Checking path fix for: ${imp.importPath}`);
        console.log(`     - Resolved to: ${resolvedRelative}`);
        console.log(`     - Resolved parts: [${resolvedParts.join(', ')}]`);
        // If resolved path is in pkg-core/lib/ but import is $core/* (not $core/lib/*)
        if (resolvedParts.includes('pkg-core') && resolvedParts.includes('lib')) {
          const importPathParts = imp.importPath.split('/');
          console.log(`     - Import path parts: [${importPathParts.join(', ')}]`);
          if (importPathParts.length >= 2 && importPathParts[1] !== 'lib') {
            // Import is like $core/http but resolves to pkg-core/lib/http.ts
            console.log(`     - ✅ Needs fix: ${imp.importPath} → $core/lib/${importPathParts.slice(1).join('/')}`);
            importPathNeedsFix = true;
            actualFileLocation = resolvedRelative;
          }
        }
      }
    }
    
    // If file is missing but it's an asset file, try to find it
    if (missingFile) {
      const cleanImportPath = imp.importPath.split('?')[0];
      const fileName = cleanImportPath.split('/').pop() || '';
      if (fileName && (imp.importPath.includes('.') || imp.importPath.match(/\?raw|\?worker/))) {
        const foundFile = searchForFile(fileName);
        if (foundFile) {
          actualFileLocation = path.relative(PROJECT_ROOT, foundFile);
        }
      }
    }
    
    // Check if imported symbols exist in the target file
    if (!missingFile) {
      let relativePath = path.relative(PROJECT_ROOT, resolvedFile!);
      let fileExps = fileExports.get(relativePath) || [];
      
      // If the resolved path is a directory, check for index files
      if (fileExps.length === 0 && fs.existsSync(resolvedFile!) && fs.statSync(resolvedFile!).isDirectory()) {
        const indexFiles = ['index.ts', 'index.js', 'index.svelte'];
        for (const indexFile of indexFiles) {
          const indexPath = path.join(resolvedFile!, indexFile);
          if (fs.existsSync(indexPath)) {
            const indexRelativePath = path.relative(PROJECT_ROOT, indexPath);
            fileExps = fileExports.get(indexRelativePath) || [];
            if (fileExps.length > 0) {
              relativePath = indexRelativePath;
              break;
            }
          }
        }
      }
      
      const exportedNames = new Set(fileExps.map(e => e.name));
      
      for (const importName of imp.imports) {
        // Skip TypeScript keywords
        if (['type', 'interface'].includes(importName)) continue;
      
        if (!exportedNames.has(importName)) {
          missingSymbols.push(importName);
        
          // Search for the symbol in other files
          const locations = symbolIndex.get(importName) || [];
          for (const loc of locations) {
            foundIn.push(loc);
          }
        }
      }
    } else {
      // File doesn't exist, search for all symbols
      for (const importName of imp.imports) {
        if (['type', 'interface'].includes(importName)) continue;
      
        missingSymbols.push(importName);
        const locations = symbolIndex.get(importName) || [];
        for (const loc of locations) {
          foundIn.push(loc);
        }
      }
    }
    
    if (missingFile || missingSymbols.length > 0 || importPathNeedsFix) {
      issues.push({
        filePath: path.relative(PROJECT_ROOT, filePath),
        lineNumber: imp.lineNumber,
        importStatement: imp,
        missingFile,
        missingSymbols,
        foundIn,
        actualFileLocation,
        importStatement: imp
      });
    }
  }
  
  return issues;
}

/**
 * Analyze all files and generate report
 */
function analyzeImports(): FixReport {
  console.log('🔍 Analyzing imports...\n');
  
  const searchDirs = [
    'pkg-app',
    'pkg-components',
    'pkg-store',
    'pkg-ui',
    'pkg-services'
  ];
  
  let totalFiles = 0;
  let totalImports = 0;
  const allIssues: ImportIssue[] = [];
  
  for (const dir of searchDirs) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;
    
    const files = findFilesRecursively(dirPath, ['.ts', '.svelte']);
    
    for (const filePath of files) {
      const imports = parseImports(filePath);
      totalImports += imports.length;
      
      const issues = checkImports(filePath);
      allIssues.push(...issues);
      
      if (imports.length > 0) {
        totalFiles++;
      }
    }
  }
  
  const report: FixReport = {
    totalFiles,
    totalImports,
    totalIssues: allIssues.length,
    issuesByType: {
      missingFile: allIssues.filter(i => i.missingFile).length,
      missingSymbols: allIssues.filter(i => i.missingSymbols.length > 0).length
    },
    issues: allIssues
  };
  
  return report;
}

/**
 * Display the analysis report
 */
function displayReport(report: FixReport): void {
  console.log('📊 IMPORT ANALYSIS REPORT\n');
  console.log(`Total files analyzed: ${report.totalFiles}`);
  console.log(`Total imports found: ${report.totalImports}`);
  console.log(`Total issues found: ${report.totalIssues}`);
  console.log(`  - Missing files: ${report.issuesByType.missingFile}`);
  console.log(`  - Missing symbols: ${report.issuesByType.missingSymbols}\n`);
  
  if (report.issues.length === 0) {
    console.log('✅ No issues found! All imports are correct.\n');
    return;
  }
  
  // Group issues by file
  const issuesByFile = new Map<string, ImportIssue[]>();
  for (const issue of report.issues) {
    if (!issuesByFile.has(issue.filePath)) {
      issuesByFile.set(issue.filePath, []);
    }
    issuesByFile.get(issue.filePath)!.push(issue);
  }
  
  console.log('📝 ISSUES DETAILS:\n');
  
  let issueNum = 1;
  for (const [filePath, issues] of issuesByFile) {
    console.log(`📄 ${filePath}`);
  
    for (const issue of issues) {
      console.log(`  [${issueNum}] Line ${issue.lineNumber}: ${issue.importStatement.raw}`);
    
      if (issue.missingFile) {
        console.log(`     ❌ File not found: ${issue.importStatement.importPath}`);
        if (issue.actualFileLocation) {
          console.log(`     💡 Found at: ${issue.actualFileLocation}`);
        }
      }
    
      if (issue.missingSymbols.length > 0) {
        console.log(`     ❌ Missing symbols: ${issue.missingSymbols.join(', ')}`);
      }
      
      if (issue.foundIn && issue.foundIn.length > 0) {
        console.log(`     💡 Found in:`);
        const uniqueLocations = new Map<string, ExportedSymbol>();
        for (const found of issue.foundIn) {
          const key = `${found.filePath}:${found.name}`;
          if (!uniqueLocations.has(key)) {
            uniqueLocations.set(key, found);
          }
        }
        
        for (const [, found] of uniqueLocations) {
          const relPath = path.relative(PROJECT_ROOT, found.filePath);
          console.log(`        - ${found.name} → ${relPath}:${found.lineNumber}`);
        }
      }
      
      issueNum++;
      console.log('');
    }
  }
}

/**
 * Fix an import issue
 */
function fixImportIssue(issue: ImportIssue, dryRun: boolean = false): boolean {
  const filePath = path.join(PROJECT_ROOT, issue.filePath);
  let content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  
  const lineIndex = issue.lineNumber - 1;
  const originalLine = lines[lineIndex];
  
  // Only skip if we have no symbol locations AND no actual file location
  if ((!issue.foundIn || issue.foundIn.length === 0) && !issue.actualFileLocation) {
    console.log(`⚠️  Cannot fix ${issue.filePath}:${issue.lineNumber} - symbol locations not found`);
    return false;
  }
  
  // Determine the best fix
  const fix = determineBestFix(issue);
  if (!fix) {
    console.log(`⚠️  Cannot fix ${issue.filePath}:${issue.lineNumber} - no suitable fix found`);
    return false;
  }
  
  console.log(`${dryRun ? '[DRY RUN] ' : ''}🔧 Fixing ${issue.filePath}:${issue.lineNumber}`);
  console.log(`    ${issue.importStatement.raw}`);
  console.log(`    → ${fix.newImport}`);
  
  if (dryRun) {
    return true;
  }
  
  // Apply the fix
  lines[lineIndex] = fix.newImport;
  fs.writeFileSync(filePath, lines.join('\n'), 'utf-8');
  
  return true;
}

/**
 * Determine the best fix for an import issue
 */
function determineBestFix(issue: ImportIssue): { newImport: string } | null {
  const { importStatement, foundIn, actualFileLocation } = issue;
  
  console.log(`\n  🔧 determineBestFix called for: ${issue.filePath}:${issue.lineNumber}`);
  console.log(`     - Missing file: ${issue.missingFile}`);
  console.log(`     - Missing symbols: [${issue.missingSymbols.join(', ')}]`);
  console.log(`     - Found in count: ${foundIn?.length || 0}`);
  console.log(`     - Actual file location: ${actualFileLocation || 'none'}`);
  
  // If we found actual file location for assets, fix it
  if (actualFileLocation && importStatement.importPath.match(/\.(svg|js|css)(\?raw|\?worker)?$/)) {
    const newAliasPath = convertToAliasPath(actualFileLocation, path.dirname(issue.filePath));
    if (!newAliasPath) return null;
    
    // Preserve query parameters
    const queryParams = importStatement.importPath.split('?')[1] || '';
    const finalPath = queryParams ? `${newAliasPath}?${queryParams}` : newAliasPath;
    
    let newImport = '';
    if (importStatement.isSideEffect) {
      newImport = `import '${finalPath}';`;
    } else if (importStatement.isDefault) {
      newImport = `import ${importStatement.imports[0]} from '${finalPath}';`;
    } else if (importStatement.isNamed) {
      newImport = `import { ${importStatement.imports.join(', ')} } from '${finalPath}';`;
    }
    
    return { newImport };
  }
  
    // Fix import paths that need correction (e.g., $core/http → $core/lib/http)
    if (actualFileLocation && importStatement.importPath.startsWith('$') && !importStatement.importPath.match(/\.(ts|js|svg|svelte|png|jpg|jpeg|gif|webp|ico|css|json)(\?.*)?$/)) {
      console.log(`  🔧 determineBestFix: Path correction detected`);
      console.log(`     - Original import path: ${importStatement.importPath}`);
      console.log(`     - Actual file location: ${actualFileLocation}`);
    
      const newAliasPath = convertToAliasPath(actualFileLocation, path.dirname(issue.filePath));
      console.log(`     - Converted to alias path: ${newAliasPath}`);
    
      if (!newAliasPath) {
        console.log(`     - ❌ convertToAliasPath returned null`);
        return null;
      }

      let newImport = '';
      const queryParams = importStatement.importPath.split('?')[1] || '';
      const finalPath = queryParams ? `${newAliasPath}?${queryParams}` : newAliasPath;
    
      console.log(`     - Is default: ${importStatement.isDefault}, Is named: ${importStatement.isNamed}, Is side effect: ${importStatement.isSideEffect}`);
      console.log(`     - Importing symbols: [${importStatement.imports.join(', ')}]`);

      if (importStatement.isSideEffect) {
        newImport = `import '${finalPath}';`;
      } else if (importStatement.isDefault && !importStatement.isNamed) {
        newImport = `import ${importStatement.imports[0]} from '${finalPath}';`;
      } else if (importStatement.isNamed && !importStatement.isDefault) {
        newImport = `import { ${importStatement.imports.join(', ')} } from '${finalPath}';`;
      } else if (importStatement.isDefault && importStatement.isNamed) {
        const defaultImport = importStatement.imports[0];
        const namedImports = importStatement.imports.slice(1);
        newImport = `import ${defaultImport}, { ${namedImports} } from '${finalPath}';`;
      }

      console.log(`     - Generated new import: ${newImport}`);
      return { newImport };
    }
  
  if (!foundIn || foundIn.length === 0) return null;
  
  // Group found symbols by location
  const byLocation = new Map<string, Set<string>>();
  for (const found of foundIn) {
    const relPath = path.relative(PROJECT_ROOT, found.filePath);
    if (!byLocation.has(relPath)) {
      byLocation.set(relPath, new Set());
    }
    byLocation.get(relPath)!.add(found.name);
  }
  
  // Find location that has all missing symbols
  for (const [locPath, symbols] of byLocation) {
    const hasAll = issue.missingSymbols.every(sym => symbols.has(sym));
    
    if (hasAll) {
      // Calculate the alias path from the file path
      const newAliasPath = convertToAliasPath(locPath, path.dirname(issue.filePath));
      if (!newAliasPath) continue;
      
      // Build new import statement
      let newImport = '';
      
      if (importStatement.isSideEffect) {
        newImport = `import '${newAliasPath}';`;
      } else if (importStatement.isDefault && importStatement.isNamed) {
        // Mixed import
        const defaultImport = importStatement.imports[0];
        const namedImports = importStatement.imports.slice(1);
        newImport = `import ${defaultImport}, { ${namedImports.join(', ')} } from '${newAliasPath}';`;
      } else if (importStatement.isDefault) {
        newImport = `import ${importStatement.imports[0]} from '${newAliasPath}';`;
      } else if (importStatement.isNamed) {
        newImport = `import { ${importStatement.imports.join(', ')} } from '${newAliasPath}';`;
      }
      
      return { newImport };
    }
  }
  
  if (!foundIn || foundIn.length === 0) return null;
  
  // Special handling for $core imports - need to add subdirectory
  if (importStatement.importPath.startsWith('$core/') && !importStatement.importPath.startsWith('$core/lib/') && !importStatement.importPath.startsWith('$core/core/')) {
    // Determine which subdirectory symbols are in
    const subdirs = new Map<string, Set<string>>();
    for (const found of foundIn) {
      const relPath = path.relative(PROJECT_ROOT, found.filePath);
      let subdir = 'lib';
      if (relPath.startsWith('pkg-core/core/')) {
        subdir = 'core';
      } else if (relPath.startsWith('pkg-core/lib/')) {
        subdir = 'lib';
      }
      
      if (!subdirs.has(subdir)) {
        subdirs.set(subdir, new Set());
      }
      subdirs.get(subdir)!.add(found.name);
    }
    
    // Check if we can satisfy all imports from one subdirectory
    for (const [subdir, symbols] of subdirs) {
      const hasAll = issue.missingSymbols.every(sym => symbols.has(sym));
      if (hasAll) {
        const rest = importStatement.importPath.substring(6); // Remove "$core/"
        const newAliasPath = `$core/${subdir}/${rest}`;
        
        let newImport = '';
        if (importStatement.isSideEffect) {
          newImport = `import '${newAliasPath}';`;
        } else if (importStatement.isDefault && importStatement.isNamed) {
          const defaultImport = importStatement.imports[0];
          const namedImports = importStatement.imports.slice(1);
          newImport = `import ${defaultImport}, { ${namedImports.join(', ')} } from '${newAliasPath}';`;
        } else if (importStatement.isDefault) {
          newImport = `import ${importStatement.imports[0]} from '${newAliasPath}';`;
        } else if (importStatement.isNamed) {
          newImport = `import { ${importStatement.imports.join(', ')} } from '${newAliasPath}';`;
        }
        
        return { newImport };
      }
    }
  }
  
  return null;
}

/**
 * Convert a file path to an alias path
 */
function convertToAliasPath(filePath: string, sourceDir: string): string | null {
  const absPath = path.join(PROJECT_ROOT, filePath);
  const absSourceDir = path.join(PROJECT_ROOT, sourceDir);
  
  // Try to find if the file is under one of the alias directories
  const aliasMap: Record<string, string> = {
    '$lib': 'pkg-app/lib',
    '$core': 'pkg-core',
    '$store': 'pkg-store/stores',
    '$routes': 'pkg-app/routes',
    '$components': 'pkg-ui/components',
    '$shared': 'pkg-services/shared',
    '$ecommerce': 'pkg-components/ecommerce',
    '$services': 'pkg-services/services'
  };
  
  for (const [alias, baseDir] of Object.entries(aliasMap)) {
    const absBaseDir = path.join(PROJECT_ROOT, baseDir);
    
    if (absPath.startsWith(absBaseDir + path.sep)) {
      const relativePath = path.relative(absBaseDir, absPath);
      const withoutExt = relativePath.replace(/\.(ts|js|svelte)$/, '');
      return `${alias}/${withoutExt}`;
    }
  }
  
  // If not under alias, try relative path
  const relativePath = path.relative(absSourceDir, absPath);
  const withoutExt = relativePath.replace(/\.(ts|js|svelte)$/, '');
  
  // Don't use relative if it goes up too many levels
  if (withoutExt.startsWith('..')) {
    return null;
  }
  
  return withoutExt.startsWith('.') ? withoutExt : `./${withoutExt}`;
}

/**
 * Apply fixes to all issues
 */
function applyFixes(report: FixReport, dryRun: boolean = false): void {
  if (report.issues.length === 0) {
    console.log('✅ No issues to fix!\n');
    return;
  }
  
  console.log('\n🔧 APPLYING FIXES...\n');
  
  let fixedCount = 0;
  let failedCount = 0;
  
  for (const issue of report.issues) {
    const success = fixImportIssue(issue, dryRun);
    if (success) {
      fixedCount++;
    } else {
      failedCount++;
    }
  }
  
  console.log(`\n✨ ${dryRun ? '[DRY RUN] ' : ''}Fixed ${fixedCount} issues`);
  if (failedCount > 0) {
    console.log(`⚠️  Could not fix ${failedCount} issues`);
  }
}

/**
 * Main execution
 */
function main() {
  console.log('🚀 Intelligent Import Fixer\n');
  console.log(`Mode: ${DRY_RUN ? 'Dry Run' : (SHOULD_FIX ? 'Fix' : 'Report')}\n`);
  
  // Step 1: Build symbol index
  buildSymbolIndex();
  
  // Step 2: Analyze imports
  const report = analyzeImports();
  
  // Step 3: Display report
  displayReport(report);
  
  // Step 4: Apply fixes if requested
  if (SHOULD_FIX) {
    applyFixes(report, DRY_RUN);
  }
  
  console.log('\n✅ Done!');
}

// Run the script
try {
  main();
} catch (error) {
  console.error('❌ Error:', error);
  process.exit(1);
}