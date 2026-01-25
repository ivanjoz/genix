import * as fs from 'fs';
import * as path from 'path';

const PROJECT_ROOT = '/home/ivanjoz/projects/genix/frontend';
const SHOULD_FIX = process.argv.includes('--fix');
const DRY_RUN = process.argv.includes('--dry-run');

// TypeScript keywords to skip
const GENERIC_NAMES = new Set(['type', 'interface', 'enum', 'class', 'function', 'const', 'let', 'var']);

// Framework and npm packages to exclude
const EXCLUDED_IMPORTS = new Set([
  'svelte',
  'svelte/transition',
  'svelte/motion',
  'svelte/store',
  'svelte/internal',
  'svelte/animate',
  '$app',
  '$app/environment',
  '$app/state',
  '$app/navigation',
  '$app/paths',
  '$app/stores',
  '$app/forms',
  'svelte-i18n',
  'axios',
  'notiflix',
  '@humanspeak/svelte-virtual-list'
]);

interface ExportedSymbol {
  name: string;
  type: 'default' | 'named' | 'type';
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
}

interface ImportIssue {
  filePath: string;
  lineNumber: number;
  importStatement: ImportStatement;
  missingFile: boolean;
  missingExtension?: boolean;
  missingSymbols: string[];
  foundIn: ExportedSymbol[];
  actualFileLocation?: string;
  fixAsDefaultImport?: boolean;
}

interface FixReport {
  totalIssues: number;
  issuesByType: {
    missingFile: number;
    missingExtension: number;
    missingSymbols: number;
  };
  issues: ImportIssue[];
}

// Symbol index: symbol name → list of files that export it
const symbolIndex = new Map<string, ExportedSymbol[]>();

// File exports: file path → list of exported symbols
const fileExports = new Map<string, ExportedSymbol[]>();

// Alias mappings read from svelte.config.js
let aliasMap: Record<string, string> = {};

/**
 * Read alias mappings from svelte.config.js
 */
function readAliasMappingsFromConfig(): void {
  const configPath = path.join(PROJECT_ROOT, 'svelte.config.js');
  
  if (!fs.existsSync(configPath)) {
    console.log('⚠️  svelte.config.js not found, using default aliases');
    aliasMap = {
      '$core': 'pkg-core',
      '$store': 'pkg-store',
      '$app': 'pkg-app',
      '$routes': 'pkg-app/routes',
      '$ui': 'pkg-ui',
      '$components': 'pkg-components',
      '$shared': 'pkg-services',
      '$services': 'pkg-services'
    };
    return;
  }
  
  try {
    const configContent = fs.readFileSync(configPath, 'utf-8');
    
    // Extract the alias object using regex - handle both path.resolve() and plain strings
    const aliasMatch = configContent.match(/alias:\s*\{([^}]+)\}/s);
    if (!aliasMatch) {
      console.log('⚠️  Could not find alias mappings in svelte.config.js, using defaults');
      return;
    }
    
    const aliasContent = aliasMatch[1];
    // Match patterns like: $core: path.resolve('./pkg-core') or $core: './pkg-core'
    const aliasRegex = /(\$\w+):\s*(?:path\.resolve\(['"]([^'"]+)['"]\)|['"]([^'"]+)['"])/g;
    let match;
    
    while ((match = aliasRegex.exec(aliasContent)) !== null) {
      const alias = match[1];
      const resolvedPath = match[2] || match[3]; // match[2] for path.resolve, match[3] for plain string
      
      // Normalize the path - remove leading ./ or ./
      const normalizedPath = resolvedPath.replace(/^\.\//, '').replace(/^\.$/, '');
      
      aliasMap[alias] = normalizedPath;
      console.log(`  ${alias} -> ${normalizedPath}`);
    }
    
    console.log(`✅ Loaded ${Object.keys(aliasMap).length} alias mappings from svelte.config.js`);
  } catch (error) {
    console.log('⚠️  Error reading svelte.config.js, using default aliases:', error);
  }
}

/**
 * Recursively find all files in a directory
 */
function findFilesRecursively(dir: string, extensions: string[] = []): string[] {
  const files: string[] = [];
  
  function walk(currentDir: string) {
    const items = fs.readdirSync(currentDir);
    
    for (const item of items) {
      const fullPath = path.join(currentDir, item);
      const stat = fs.statSync(fullPath);
      
      if (stat.isDirectory()) {
        // Skip node_modules and .git
        if (item !== 'node_modules' && item !== '.git' && item !== '.svelte-kit') {
          walk(fullPath);
        }
      } else if (stat.isFile()) {
        const ext = path.extname(item);
        if (extensions.length === 0 || extensions.includes(ext)) {
          files.push(fullPath);
        }
      }
    }
  }
  
  walk(dir);
  return files;
}

/**
 * Check if a path exists and return the actual file path
 */
function pathExists(basePath: string): string | null {
  const cleanPath = basePath.split('?')[0];
  
  // Check if path itself is a file
  if (fs.existsSync(cleanPath) && fs.statSync(cleanPath).isFile()) {
    return cleanPath;
  }
  
  // Try common extensions
  const extensions = ['.ts', '.js', '.svelte', '.svg', '.css', '.json'];
  for (const ext of extensions) {
    const fullPath = cleanPath + ext;
    if (fs.existsSync(fullPath) && fs.statSync(fullPath).isFile()) {
      return fullPath;
    }
  }
  
  // Check if it's a directory
  if (fs.existsSync(cleanPath) && fs.statSync(cleanPath).isDirectory()) {
    return cleanPath;
  }
  
  return null;
}

/**
 * Resolve an alias path to an actual file path
 */
function resolveAliasPath(aliasPath: string): string | null {
  // Remove query parameters if present
  const cleanPath = aliasPath.split('?')[0];
  const parts = cleanPath.split('/');
  const alias = parts[0];
  const rest = parts.slice(1).join('/');
  
  // Use alias mappings from svelte.config.js
  const baseDir = aliasMap[alias];
  if (!baseDir) {
    console.log(`  ⚠️  Unknown alias: ${alias}`);
    return null;
  }
  
  // Build the full path from project root + alias base + relative path
  const fullPath = path.join(PROJECT_ROOT, baseDir, rest);
  const resolved = pathExists(fullPath);
  
  if (resolved) {
    console.log(`  ✓️  Resolved ${aliasPath} -> ${resolved}`);
    return resolved;
  }
  
  // Path doesn't exist - return null to indicate missing file
  console.log(`  ❌ Path not found: ${fullPath}`);
  return null;
}

/**
 * Search for a file by name across the entire project
 */
function searchForFile(fileName: string, extensions: string[] = []): string | null {
  // Remove query parameters (e.g., ?raw)
  const cleanFileName = fileName.split('?')[0];

  // Build list of extensions to try
  const exts = extensions.length > 0 ? extensions : ['.ts', '.js', '.svelte', '.svg', '.css'];

  // Automatically discover all package directories (pkg-*)
  const pathsToSearch: string[] = [];
  const items = fs.readdirSync(PROJECT_ROOT);
  for (const item of items) {
    const itemPath = path.join(PROJECT_ROOT, item);
    if (fs.statSync(itemPath).isDirectory() && item.startsWith('pkg-')) {
      pathsToSearch.push(item);
      // Also add common subdirectories
      const subItems = fs.readdirSync(itemPath);
      for (const subItem of subItems) {
        const subItemPath = path.join(itemPath, subItem);
        if (fs.statSync(subItemPath).isDirectory()) {
          pathsToSearch.push(path.join(item, subItem));
        }
      }
    }
  }

  // If already has an extension, use only that
  if (path.extname(cleanFileName)) {
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
 * Check if an import is an asset file
 */
function isAssetImport(importPath: string): boolean {
  const assetExtensions = ['.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.ico', '.json'];
  const cleanPath = importPath.split('?')[0];
  return assetExtensions.some(ext => cleanPath.endsWith(ext));
}

/**
 * Check if an import is a CSS file
 */
function isCssImport(importPath: string): boolean {
  const cleanPath = importPath.split('?')[0];
  return cleanPath.endsWith('.css');
}

/**
 * Check if an import looks like a Svelte component
 */
function isSvelteComponentImport(importPath: string): boolean {
  const cleanPath = importPath.split('?')[0];
  return cleanPath.endsWith('.svelte') || 
         cleanPath.match(/\/components\//) !== null ||
         cleanPath.match(/\/[A-Z][a-zA-Z]+$/) !== null;
}

/**
 * Get component name from import path
 */
function getComponentNameFromPath(importPath: string): string | null {
  const cleanPath = importPath.split('?')[0];
  const parts = cleanPath.split('/');
  const lastPart = parts[parts.length - 1];
  const match = lastPart.match(/^([A-Z][a-zA-Z]*)/);
  return match ? match[1] : null;
}

/**
 * Extract CSS class names from a CSS module
 */
function extractCssClasses(content: string): string[] {
  const classes: string[] = [];
  const classPattern = /\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{/g;
  let match;
  
  while ((match = classPattern.exec(content)) !== null) {
    classes.push(match[1]);
  }
  
  return classes;
}

/**
 * Extract all exports from a file
 */
function extractExports(filePath: string): ExportedSymbol[] {
  const exports: ExportedSymbol[] = [];
  const content = fs.readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  
  // For CSS modules, extract class names
  if (filePath.endsWith('.css')) {
    const classes = extractCssClasses(content);
    for (const className of classes) {
      exports.push({
        name: className,
        type: 'named',
        filePath,
        lineNumber: 0
      });
    }
    return exports;
  }
  
  // Parse TypeScript/JavaScript/Svelte files
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();
    const lineNumber = i + 1;
    
    // Skip comments
    if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) {
      continue;
    }
    
    // Default export: export default ...
    const defaultMatch = trimmed.match(/^export\s+default\s+(?:class|function|const|let|var)?\s*([a-zA-Z_$][a-zA-Z0-9_$]*)/);
    if (defaultMatch) {
      const fileName = path.basename(filePath, path.extname(filePath));
      const exportName = defaultMatch[1] || fileName;
      exports.push({
        name: exportName,
        type: 'default',
        filePath,
        lineNumber
      });
      continue;
    }
    
    // Named export: export const/let/var/function/class name
    const namedMatch = trimmed.match(/^export\s+(?:const|let|var|function|class)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/);
    if (namedMatch) {
      exports.push({
        name: namedMatch[1],
        type: 'named',
        filePath,
        lineNumber
      });
      continue;
    }
    
    // Type export: export type/interface name
    const typeMatch = trimmed.match(/^export\s+(?:type|interface)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/);
    if (typeMatch) {
      exports.push({
        name: typeMatch[1],
        type: 'type',
        filePath,
        lineNumber
      });
      continue;
    }
    
    // Destructured export: export { name1, name2 }
    const destructuredMatch = trimmed.match(/^export\s+\{\s*([^}]+)\s*\}/);
    if (destructuredMatch) {
      const names = destructuredMatch[1].split(',').map(n => n.trim().split(' as ')[0].trim());
      for (const name of names) {
        if (name && !name.startsWith('type')) {
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

    // Destructured const export: export const { A, B } = ...
    const destructuredConstMatch = trimmed.match(/^export\s+const\s+\{\s*([^}]+)\s*\}\s*=/);
    if (destructuredConstMatch) {
      const names = destructuredConstMatch[1].split(',').map(n => n.trim());
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
    
    // Export from: export * from '...' or export { name } from '...'
    const exportFromMatch = trimmed.match(/^export\s+(?:\*|\{([^}]+)\})\s+from\s+['"]([^'"]+)['"]/);
    if (exportFromMatch) {
      const names = exportFromMatch[1] ? exportFromMatch[1].split(',').map(n => n.trim().split(' as ')[0].trim()) : ['*'];
      const fromPath = exportFromMatch[2];
      
      // Resolve the from path
      let resolvedPath: string | null;
      if (fromPath.startsWith('$')) {
        resolvedPath = resolveAliasPath(fromPath);
      } else {
        resolvedPath = path.resolve(path.dirname(filePath), fromPath);
      }
      
      if (resolvedPath && fs.existsSync(resolvedPath)) {
        const fromExports = fileExports.get(path.relative(PROJECT_ROOT, resolvedPath)) || [];
        for (const name of names) {
          if (name === '*') {
            exports.push(...fromExports);
          } else {
            const found = fromExports.find(e => e.name === name);
            if (found) {
              exports.push(found);
            }
          }
        }
      }
      continue;
    }
  }
  
  // For Svelte files, add the component name as a default export
  if (filePath.endsWith('.svelte')) {
    const componentName = path.basename(filePath, '.svelte');
    exports.push({
      name: componentName,
      type: 'default',
      filePath,
      lineNumber: 0
    });
    
    // Also add capitalized version for components (standard Svelte convention)
    const capitalizedName = componentName.charAt(0).toUpperCase() + componentName.slice(1);
    if (capitalizedName !== componentName) {
      exports.push({
        name: capitalizedName,
        type: 'default',
        filePath,
        lineNumber: 0
      });
    }
  }
  
  return exports;
}

/**
 * Build the symbol index by scanning all files
 */
function buildSymbolIndex() {
  const searchDirs = [
    'pkg-core',
    'pkg-ui',
    'pkg-app',
    'pkg-services',
    'pkg-components',
    'pkg-store'
  ];
  
  let totalFiles = 0;
  let totalExports = 0;
  
  for (const dir of searchDirs) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;
    
    const files = findFilesRecursively(dirPath, ['.ts', '.js', '.svelte', '.css']);
    
    for (const file of files) {
      const relativePath = path.relative(PROJECT_ROOT, file);
      const exports = extractExports(file);
      
      fileExports.set(relativePath, exports);
      
      for (const exp of exports) {
        if (!symbolIndex.has(exp.name)) {
          symbolIndex.set(exp.name, []);
        }
        symbolIndex.get(exp.name)!.push(exp);
      }
      
      totalFiles++;
      totalExports += exports.length;
    }
  }
  
  console.log(`📊 Indexed ${totalFiles} files with ${totalExports} exports`);
}

/**
 * Parse import statements from a file
 */
function parseImports(filePath: string): ImportStatement[] {
  const content = fs.readFileSync(filePath, 'utf-8');
  const imports: ImportStatement[] = [];
  const lines = content.split('\n');
  
  // Patterns for different import styles
  const patterns = [
    // Default import: import Name from 'path'
    {
      regex: /^import\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s+from\s+['"]([^'"]+)['"]/,
      isNamed: false,
      isDefault: true,
      isSideEffect: false,
      parseTypeImports: false
    },
    // Named import: import { Name1, Name2 } from 'path'
    {
      regex: /^import\s+\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true,
      isDefault: false,
      isSideEffect: false,
      parseTypeImports: false
    },
    // Mixed import: import Default, { Name } from 'path'
    {
      regex: /^import\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,\s*\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true,
      isDefault: true,
      isSideEffect: false,
      parseTypeImports: false
    },
    // Side effect import: import 'path'
    {
      regex: /^import\s+['"]([^'"]+)['"]/,
      isNamed: false,
      isDefault: false,
      isSideEffect: true,
      parseTypeImports: false
    },
    // Type import: import type { Name } from 'path'
    {
      regex: /^import\s+type\s+\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true,
      isDefault: false,
      isSideEffect: false,
      parseTypeImports: true
    }
  ];
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();
    const lineNumber = i + 1;
    
    // Skip non-import lines
    if (!trimmed.startsWith('import')) continue;
    
    for (const pattern of patterns) {
      const match = trimmed.match(pattern.regex);
      if (match) {
        let importPath: string;
        let importedSymbols: string[] = [];
      
        if (pattern.isSideEffect) {
          importPath = match[1];
        } else if (pattern.isDefault && pattern.isNamed) {
          // Mixed import
          importPath = match[3];
          importedSymbols = [match[1], ...match[2].split(',').map(n => n.trim())];
        } else if (pattern.isDefault) {
          // Default import
          importPath = match[2];
          importedSymbols = [match[1]];
        } else {
          // Named import
          importPath = match[2];
          importedSymbols = match[1].split(',').map(n => n.trim());
        }
      
        // Filter out 'type' keyword from imports
        importedSymbols = importedSymbols.filter(n => n !== 'type');
      
        imports.push({
          raw: trimmed,
          importPath,
          imports: importedSymbols,
          isDefault: pattern.isDefault,
          isNamed: pattern.isNamed,
          isSideEffect: pattern.isSideEffect,
          lineNumber
        });
      
        break;
      }
    }
  }
  
  return imports;
}

/**
 * Check if imports are valid
 * Simple logic: check if path exists and if symbols are exported
 */
function checkImports(filePath: string): ImportIssue[] {
  const issues: ImportIssue[] = [];
  const imports = parseImports(filePath);
  
  for (const imp of imports) {
    // Skip side effect imports
    if (imp.isSideEffect) continue;
    
    // Skip asset and CSS imports
    if (imp.isDefault && (isAssetImport(imp.importPath) || isCssImport(imp.importPath))) {
      continue;
    }
    
    // Skip excluded imports
    if (EXCLUDED_IMPORTS.has(imp.importPath)) continue;
    if (imp.importPath.startsWith('@') || (!imp.importPath.startsWith('$') && !imp.importPath.startsWith('.'))) {
      continue;
    }
    
    // Skip relative imports that resolve correctly
    if (!imp.importPath.startsWith('$')) {
      const resolvedPath = path.resolve(path.dirname(filePath), imp.importPath);
      if (pathExists(resolvedPath)) continue;
    }
    
    // STEP 1: Check if the file exists
    const resolvedFile = resolveAliasPath(imp.importPath);
    const missingFile = resolvedFile === null;
    
    // Check for missing extension (e.g. $ui/components/Input instead of Input.svelte)
    let missingExtension = false;
    if (resolvedFile && resolvedFile.endsWith('.svelte')) {
      const cleanImportPath = imp.importPath.split('?')[0];
      if (!cleanImportPath.endsWith('.svelte')) {
        missingExtension = true;
        console.log(`  ⚠️  Missing .svelte extension: ${imp.importPath}`);
      }
    }

    const missingSymbols: string[] = [];
    const foundIn: ExportedSymbol[] = [];
    let actualFileLocation: string | undefined;
    let fixAsDefaultImport = false;
    
    if (missingFile || missingExtension) {
      if (missingFile) {
        // File doesn't exist - try to find it
        console.log(`  ❌ File not found: ${imp.importPath}`);
        
        // Extract filename from import path
        const cleanImportPath = imp.importPath.split('?')[0];
        const fileName = path.basename(cleanImportPath);
        const baseName = path.basename(fileName, path.extname(fileName));
        
        // Search for the file across all packages
        const foundFile = searchForFile(baseName);
        if (foundFile) {
          actualFileLocation = path.relative(PROJECT_ROOT, foundFile);
        }
      } else if (missingExtension) {
        // File found but extension missing in import
        actualFileLocation = path.relative(PROJECT_ROOT, resolvedFile!);
      }
      
      // Also search for symbols
      for (const importName of imp.imports) {
        if (GENERIC_NAMES.has(importName)) continue;
        
        missingSymbols.push(importName);
        const locations = symbolIndex.get(importName) || [];
        for (const loc of locations) {
          foundIn.push(loc);
        }
      }
    } else {
      // STEP 2: File exists, check if symbols are exported
      const relativePath = path.relative(PROJECT_ROOT, resolvedFile!);
      let fileExps = fileExports.get(relativePath) || [];
      
      // Check for index files if directory
      if (fileExps.length === 0 && fs.existsSync(resolvedFile!) && fs.statSync(resolvedFile!).isDirectory()) {
        const indexFiles = ['index.ts', 'index.js', 'index.svelte'];
        for (const indexFile of indexFiles) {
          const indexPath = path.join(resolvedFile!, indexFile);
          if (fs.existsSync(indexPath)) {
            const indexRelativePath = path.relative(PROJECT_ROOT, indexPath);
            fileExps = fileExports.get(indexRelativePath) || [];
            if (fileExps.length > 0) break;
          }
        }
      }
      
      const isSvelteFile = resolvedFile!.endsWith('.svelte') || relativePath.endsWith('.svelte');
      const exportedNames = new Set(fileExps.map(e => e.name));
      
      // Check for Svelte component imported as named import
      if ((isSvelteComponentImport(imp.importPath) || isSvelteFile) && imp.isNamed && !imp.isDefault) {
        const componentName = getComponentNameFromPath(imp.importPath) || path.basename(relativePath, '.svelte');
        if (componentName && imp.imports.length === 1 && imp.imports[0] === componentName) {
          fixAsDefaultImport = true;
          console.log(`  ⚠️  Svelte component imported as named: ${componentName}`);
        }
      }
      
      // Check if all imported symbols exist
      for (const importName of imp.imports) {
        if (GENERIC_NAMES.has(importName)) continue;
        
        // For Svelte components, do case-insensitive matching since component names
        // can differ in case from their filename (e.g., VTable vs vTable.svelte)
        let symbolFound = exportedNames.has(importName);
        
        if (!symbolFound && (isSvelteFile || isSvelteComponentImport(imp.importPath))) {
          // Try case-insensitive match for Svelte components
          for (const exportedName of exportedNames) {
            if (exportedName.toLowerCase() === importName.toLowerCase()) {
              symbolFound = true;
              console.log(`  ✓️  Svelte component '${importName}' matches exported '${exportedName}' (case-insensitive)`);
              break;
            }
          }
        }
        
        if (!symbolFound) {
          missingSymbols.push(importName);
          
          // Search for symbol in other files
          const locations = symbolIndex.get(importName) || [];
          for (const loc of locations) {
            foundIn.push(loc);
          }
          
          if (locations.length > 0) {
            console.log(`  ❌ Symbol '${importName}' not found in ${imp.importPath}, found in ${locations.length} other files`);
          } else {
            console.log(`  ❌ Symbol '${importName}' not found anywhere`);
          }
        }
      }
    }
    
    // Record issue if any problems found
    if (missingFile || missingExtension || missingSymbols.length > 0 || fixAsDefaultImport) {
      issues.push({
        filePath: path.relative(PROJECT_ROOT, filePath),
        lineNumber: imp.lineNumber,
        importStatement: imp,
        missingFile,
        missingExtension,
        missingSymbols,
        foundIn,
        actualFileLocation,
        fixAsDefaultImport
      });
    }
  }
  
  return issues;
}

/**
 * Analyze all imports in the project
 */
function analyzeImports(): FixReport {
  const searchDirs = [
    'pkg-app',
    'pkg-core',
    'pkg-ui',
    'pkg-services',
    'pkg-components',
    'pkg-store'
  ];
  
  let totalFiles = 0;
  let totalImports = 0;
  const allIssues: ImportIssue[] = [];
  
  for (const dir of searchDirs) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;
    
    const files = findFilesRecursively(dirPath, ['.ts', '.js', '.svelte']);
    
    for (const file of files) {
      const relativePath = path.relative(PROJECT_ROOT, file);
      console.log(`  Analyzing: ${relativePath}`);
      
      const imports = parseImports(file);
      const issues = checkImports(file);
      
      totalFiles++;
      totalImports += imports.length;
      allIssues.push(...issues);
    }
  }
  
  const report: FixReport = {
    totalIssues: allIssues.length,
    issuesByType: {
      missingFile: allIssues.filter(i => i.missingFile).length,
      missingExtension: allIssues.filter(i => i.missingExtension).length,
      missingSymbols: allIssues.filter(i => i.missingSymbols.length > 0).length
    },
    issues: allIssues
  };
  
  return report;
}

/**
 * Display the report
 */
function displayReport(report: FixReport) {
  console.log('\n' + '='.repeat(80));
  console.log('📋 IMPORT ANALYSIS REPORT');
  console.log('='.repeat(80));
  console.log(`Total issues found: ${report.totalIssues}`);
  console.log(`  - Missing files: ${report.issuesByType.missingFile}`);
  console.log(`  - Missing extensions: ${report.issuesByType.missingExtension}`);
  console.log(`  - Missing symbols: ${report.issuesByType.missingSymbols}`);
  console.log('='.repeat(80));
  
  if (report.totalIssues === 0) {
    console.log('✅ No issues found! All imports are correct.');
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
  
  let issueNum = 1;
  for (const [filePath, fileIssues] of issuesByFile) {
    console.log(`\n📄 ${filePath} (${fileIssues.length} issue(s))`);
    console.log('-'.repeat(80));
    
    for (const issue of fileIssues) {
      console.log(`\n  Issue #${issueNum++} (Line ${issue.lineNumber})`);
      console.log(`  Import: ${issue.importStatement.raw}`);
      
      if (issue.missingFile) {
        console.log(`  ❌ File not found: ${issue.importStatement.importPath}`);
      } else if (issue.missingExtension) {
        console.log(`  ⚠️  Missing extension in: ${issue.importStatement.importPath}`);
      }
      
      if (issue.missingSymbols.length > 0) {
        console.log(`  ❌ Missing symbols: ${issue.missingSymbols.join(', ')}`);
      }
      
      if (issue.fixAsDefaultImport) {
        console.log(`  ⚠️  Should be default import`);
      }
      
      if (issue.foundIn.length > 0) {
        const uniqueLocations = new Set(issue.foundIn.map(f => f.filePath));
        console.log(`  💡 Found in: ${Array.from(uniqueLocations).join(', ')}`);
      }
    }
  }
  
  console.log('\n' + '='.repeat(80));
}

/**
 * Fix an import issue
 */
function fixImportIssue(filePath: string, issue: ImportIssue): boolean {
  const fullPath = path.join(PROJECT_ROOT, filePath);
  const content = fs.readFileSync(fullPath, 'utf-8');
  const lines = content.split('\n');
  const lineIndex = issue.lineNumber - 1;
  
  if (lineIndex < 0 || lineIndex >= lines.length) {
    console.log(`  ❌ Invalid line number: ${issue.lineNumber}`);
    return false;
  }
  
  const originalLine = lines[lineIndex];
  const fix = determineBestFix(issue);
  
  if (!fix) {
    console.log(`  ❌ No fix available for this issue`);
    return false;
  }
  
  console.log(`  📝 Applying fix: ${originalLine} → ${fix}`);
  lines[lineIndex] = fix;
  
  if (!DRY_RUN) {
    fs.writeFileSync(fullPath, lines.join('\n'));
  }
  
  return true;
}

/**
 * Determine the best fix for an issue
 */
function determineBestFix(issue: ImportIssue): string | null {
  const imp = issue.importStatement;
  
  // Fix missing extension or missing file by suggesting correct path
  if ((issue.missingExtension || issue.missingFile) && issue.actualFileLocation) {
    const aliasPath = convertToAliasPath(issue.actualFileLocation);
    if (aliasPath) {
      if (imp.isDefault && !imp.isNamed) {
        return `import ${imp.imports[0]} from '${aliasPath}';`;
      } else if (imp.isNamed && !imp.isDefault) {
        return `import { ${imp.imports.join(', ')} } from '${aliasPath}';`;
      } else if (imp.isDefault && imp.isNamed) {
        const defaultImport = imp.imports[0];
        const namedImports = imp.imports.slice(1).join(', ');
        return `import ${defaultImport}, { ${namedImports} } from '${aliasPath}';`;
      } else if (imp.isSideEffect) {
        return `import '${aliasPath}';`;
      }
    }
  }

  // Fix Svelte component import style (named to default)
  if (issue.fixAsDefaultImport) {
    const componentName = getComponentNameFromPath(imp.importPath) || imp.imports[0];
    // Ensure we keep extension if it was there or if it should be there
    let targetPath = imp.importPath;
    if (issue.actualFileLocation) {
      targetPath = convertToAliasPath(issue.actualFileLocation) || targetPath;
    }
    return `import ${componentName} from '${targetPath}';`;
  }
  
  // Fix missing symbols by suggesting correct import path
  if (issue.missingSymbols.length > 0 && issue.foundIn.length > 0) {
    // Group foundIn by file to avoid multiple suggestions for same file
    const uniqueFiles = new Map();
    for (const loc of issue.foundIn) {
      if (!uniqueFiles.has(loc.filePath)) {
        uniqueFiles.set(loc.filePath, loc);
      }
    }
    
    const locations = Array.from(uniqueFiles.values());
    console.log(`  [DEBUG] Symbol '${issue.missingSymbols[0]}' found in: ${locations.map(l => l.filePath).join(', ')}`);
    
    // Use the first found location
    const foundFile = locations[0].filePath;
    const aliasPath = convertToAliasPath(foundFile);
    console.log(`  [DEBUG] Selected alias path: ${aliasPath} from file: ${foundFile}`);
    
    if (aliasPath && aliasPath !== imp.importPath) {
      // Reconstruct import with new path
      if (imp.isDefault && !imp.isNamed) {
        return `import ${imp.imports[0]} from '${aliasPath}';`;
      } else if (imp.isNamed && !imp.isDefault) {
        return `import { ${imp.imports.join(', ')} } from '${aliasPath}';`;
      } else if (imp.isDefault && imp.isNamed) {
        const defaultImport = imp.imports[0];
        const namedImports = imp.imports.slice(1).join(', ');
        return `import ${defaultImport}, { ${namedImports} } from '${aliasPath}';`;
      }
    }
  }
  
  return null;
}

/**
 * Convert an absolute file path to an alias path
 */
function convertToAliasPath(filePath: string): string | null {
  let absPath = filePath;
  if (!path.isAbsolute(filePath)) {
    absPath = path.join(PROJECT_ROOT, filePath);
  }
  
  const absSourceDir = PROJECT_ROOT;
  // Use alias mappings from svelte.config.js
  
  // Sort aliases by length (descending) to match longest/most specific first
  const sortedAliases = Object.entries(aliasMap).sort((a, b) => b[1].length - a[1].length);

  // Find which package directory the file is in
  for (const [alias, pkgDir] of sortedAliases) {
    const absBaseDir = path.resolve(absSourceDir, pkgDir);
    // Use a trailing separator to ensure we match a full directory name
    const baseDirWithSep = absBaseDir.endsWith(path.sep) ? absBaseDir : absBaseDir + path.sep;
    
    if (absPath.startsWith(baseDirWithSep) || absPath === absBaseDir) {
      let relativePath = path.relative(absBaseDir, absPath);
      const isSvelte = relativePath.endsWith('.svelte');
      
      // Remove extension for non-svelte files
      let processedPath = relativePath;
      if (!isSvelte) {
        processedPath = relativePath.replace(/\.(ts|js|css)$/, '');
      }
      
      // Special handling for index files
      if (path.basename(processedPath, isSvelte ? '.svelte' : undefined) === 'index') {
        processedPath = path.dirname(processedPath);
        if (processedPath === '.') processedPath = '';
      }
      
      const result = processedPath ? `${alias}/${processedPath}` : alias;
      return result;
    }
  }
  
  return null;
}

/**
 * Apply fixes to all issues
 */
function applyFixes(report: FixReport) {
  console.log('\n' + '='.repeat(80));
  console.log('🔧 APPLYING FIXES');
  console.log('='.repeat(80));
  
  if (DRY_RUN) {
    console.log('⚠️  DRY RUN MODE - No files will be modified');
  }
  
  let fixedCount = 0;
  let failedCount = 0;
  
  for (const issue of report.issues) {
    console.log(`\nFixing: ${issue.filePath}:${issue.lineNumber}`);
    const success = fixImportIssue(issue.filePath, issue);
    if (success) {
      fixedCount++;
    } else {
      failedCount++;
    }
  }
  
  console.log('\n' + '='.repeat(80));
  console.log(`✅ Fixed: ${fixedCount}`);
  console.log(`❌ Failed: ${failedCount}`);
  console.log('='.repeat(80));
}

/**
 * Main function
 */
function main() {
  console.log('🔍 Intelligent Import Fixer');
  console.log('='.repeat(80));
  
  // Read alias mappings from svelte.config.js
  console.log('\n⚙️  Reading alias mappings from svelte.config.js...');
  readAliasMappingsFromConfig();
  
  // Build symbol index
  console.log('\n📚 Building symbol index...');
  buildSymbolIndex();
  
  // Analyze imports
  console.log('\n🔍 Analyzing imports...');
  const report = analyzeImports();
  
  // Display report
  displayReport(report);
  
  // Apply fixes if requested
  if (SHOULD_FIX && report.totalIssues > 0) {
    applyFixes(report);
  }
}

main();