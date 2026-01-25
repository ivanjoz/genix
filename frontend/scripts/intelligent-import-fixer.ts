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
  isTypeOnly: boolean;
  symbolsMetadata: { name: string, isType: boolean }[];
  lineNumber: number;
}

interface ImportIssue {
  filePath: string;
  lineNumber: number;
  importStatement: ImportStatement;
  missingFile: boolean;
  missingExtension?: boolean;
  missingSymbols: string[];
  missingTypeSymbols: string[];
  foundIn: ExportedSymbol[];
  actualFileLocation?: string;
  fixAsDefaultImport?: boolean;
  shouldAddTypeKeyword?: boolean;
}

interface FixReport {
  totalIssues: number;
  issuesByType: {
    missingFile: number;
    missingExtension: number;
    missingSymbols: number;
    missingTypeKeyword: number;
    missingInlineType: number;
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



  // Reset alias map - NO hardcoded fallbacks

  aliasMap = {};



  if (!fs.existsSync(configPath)) {

    console.log('⚠️  svelte.config.js not found');

    return;

  }



  try {

    const configContent = fs.readFileSync(configPath, 'utf-8');

    const aliasMatch = configContent.match(/alias:\s*\{([^}]+)\}/s);



    if (aliasMatch) {

      const aliasContent = aliasMatch[1];

      const aliasRegex = /(\$\w+):\s*(?:path\.resolve\(['"]([^'"]+)['"]\)|['"]([^'"]+)['"])/g;

      let match;



      while ((match = aliasRegex.exec(aliasContent)) !== null) {

        const alias = match[1];

        const resolvedPath = match[2] || match[3];



        // Normalize: remove leading ./ and handle absolute paths relative to PROJECT_ROOT

        let normalizedPath = resolvedPath.replace(/^\.\//, '').replace(/^\.$/, '');

        if (path.isAbsolute(normalizedPath)) {

          normalizedPath = path.relative(PROJECT_ROOT, normalizedPath);

        }



        aliasMap[alias] = normalizedPath;

      }

    }



    console.log(`✅ Loaded ${Object.keys(aliasMap).length} alias mappings from config`);

  } catch (error) {

    console.log('⚠️  Error reading svelte.config.js:', error);

  }

}



/**
 * Recursively find all files in a directory
 */
function findFilesRecursively(dir: string, extensions: string[] = []): string[] {
  const files: string[] = [];

  function walk(currentDir: string) {
    if (!fs.existsSync(currentDir)) return;
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

  if (fs.existsSync(cleanPath) && fs.statSync(cleanPath).isFile()) {
    return cleanPath;
  }

  const extensions = ['.ts', '.js', '.svelte', '.svelte.ts', '.svelte.js', '.svg', '.css', '.json'];
  for (const ext of extensions) {
    const fullPath = cleanPath + ext;
    if (fs.existsSync(fullPath) && fs.statSync(fullPath).isFile()) {
      return fullPath;
    }
  }

  if (fs.existsSync(cleanPath) && fs.statSync(cleanPath).isDirectory()) {
    return cleanPath;
  }

  return null;
}

/**
 * Resolve an alias path to an actual file path
 */
function resolveAliasPath(aliasPath: string): string | null {
  const cleanPath = aliasPath.split('?')[0];
  const parts = cleanPath.split('/');
  const alias = parts[0];
  const rest = parts.slice(1).join('/');

  const baseDir = aliasMap[alias];
  if (!baseDir) return null;

  const fullPath = path.join(PROJECT_ROOT, baseDir, rest);
  const resolved = pathExists(fullPath);

  if (resolved) {
    if (fs.statSync(resolved).isDirectory()) {
      const indexFiles = ['index.ts', 'index.js', 'index.svelte'];
      for (const indexFile of indexFiles) {
        const indexPath = path.join(resolved, indexFile);
        if (fs.existsSync(indexPath) && fs.statSync(indexPath).isFile()) {
          return indexPath;
        }
      }
      return null;
    }
    return resolved;
  }

  return null;
}

/**
 * Convert an absolute file path to an alias path
 */
function convertToAliasPath(filePath: string): string | null {
  let absPath = filePath;
  if (!path.isAbsolute(filePath)) {
    absPath = path.resolve(PROJECT_ROOT, filePath);
  }

  const sortedAliases = Object.entries(aliasMap).sort((a, b) => b[1].length - a[1].length);

  for (const [alias, pkgDir] of sortedAliases) {
    const absBaseDir = path.resolve(PROJECT_ROOT, pkgDir);
    const baseDirWithSep = absBaseDir.endsWith(path.sep) ? absBaseDir : absBaseDir + path.sep;

    if (absPath.startsWith(baseDirWithSep) || absPath === absBaseDir) {
      let relativePath = path.relative(absBaseDir, absPath);

      const svelte5Exts = ['.svelte.ts', '.svelte.js'];
      let isSvelte5 = false;
      for (const ext of svelte5Exts) {
        if (relativePath.endsWith(ext)) {
          isSvelte5 = true;
          break;
        }
      }

      let processedPath = relativePath;
      if (isSvelte5) {
        processedPath = relativePath.replace(/\.svelte\.(ts|js)$/, '.svelte');
      } else if (!relativePath.endsWith('.svelte')) {
        processedPath = relativePath.replace(/\.(ts|js|css)$/, '');
      }

      const baseName = path.basename(processedPath);
      if (baseName.split('.')[0] === 'index' && !processedPath.endsWith('.svelte')) {
        processedPath = path.dirname(processedPath);
        if (processedPath === '.') processedPath = '';
      }

      const result = processedPath ? `${alias}/${processedPath}` : alias;

      // FINAL SAFETY: verify that the generated alias path actually resolves back to a file
      if (resolveAliasPath(result)) {
        return result;
      }
    }
  }

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
 * Check if an import looks like a Svelte component or Svelte-related file
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
    const destructuredMatch = trimmed.match(/^export\s+(type\s+)?\{\s*([^}]+)\s*\}/);
    if (destructuredMatch && !trimmed.includes('from')) {
      const isGlobalType = !!destructuredMatch[1];
      const names = destructuredMatch[2].split(',').map(n => n.trim());
      for (const n of names) {
        if (!n) continue;
        const isLocalType = n.startsWith('type ');
        const cleanName = isLocalType ? n.substring(5).trim() : n;
        const name = cleanName.split(/\s+as\s+/i)[0].trim();
        exports.push({
          name,
          type: (isGlobalType || isLocalType) ? 'type' : 'named',
          filePath,
          lineNumber
        });
      }
      continue;
    }

    // Destructured const export: export const { A, B } = ...
    const destructuredConstMatch = trimmed.match(/^export\s+(const|let|var)\s+\{\s*([^}]+)\s*\}\s*=/);
    if (destructuredConstMatch) {
      const names = destructuredConstMatch[2].split(',').map(n => n.trim());
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
    const exportFromMatch = trimmed.match(/^export\s+(type\s+)?(?:\*|\{([^}]+)\})\s+from\s+['"]([^'"]+)['"]/);
    if (exportFromMatch) {
      const isGlobalType = !!exportFromMatch[1];
      const names = exportFromMatch[2] ? exportFromMatch[2].split(',').map(n => n.trim()) : ['*'];
      const fromPath = exportFromMatch[3];

      // Resolve the from path
      let resolvedPath: string | null;
      if (fromPath.startsWith('$')) {
        resolvedPath = resolveAliasPath(fromPath);
      } else {
        resolvedPath = path.resolve(path.dirname(filePath), fromPath);
      }

      if (resolvedPath && fs.existsSync(resolvedPath)) {
        const fromExports = fileExports.get(path.relative(PROJECT_ROOT, resolvedPath)) || [];
        for (const n of names) {
          if (n === '*') {
            exports.push(...fromExports);
          } else {
            const isLocalType = n.startsWith('type ');
            const cleanName = isLocalType ? n.substring(5).trim() : n;
            const name = cleanName.split(/\s+as\s+/i)[0].trim();
            const found = fromExports.find(e => e.name === name);
            if (found) {
              exports.push({
                ...found,
                type: (isGlobalType || isLocalType || found.type === 'type') ? 'type' : found.type
              });
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
    'routes',
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
      regex: /^import\s+(type\s+)?([a-zA-Z_$][a-zA-Z0-9_$]*)\s+from\s+['"]([^'"]+)['"]/,
      isNamed: false,
      isDefault: true,
      isSideEffect: false
    },
    // Named import: import { Name1, Name2 } from 'path'
    {
      regex: /^import\s+(type\s+)?\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true,
      isDefault: false,
      isSideEffect: false
    },
    // Mixed import: import Default, { Name } from 'path'
    {
      regex: /^import\s+(type\s+)?([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,\s*\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true,
      isDefault: true,
      isSideEffect: false
    },
    // Side effect import: import 'path'
    {
      regex: /^import\s+['"]([^'"]+)['"]/,
      isNamed: false,
      isDefault: false,
      isSideEffect: true
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
        let isTypeOnly = false;
        let symbolsMetadata: { name: string, isType: boolean }[] = [];

        if (pattern.isSideEffect) {
          importPath = match[1];
        } else if (pattern.isDefault && pattern.isNamed) {
          // Mixed import: import type Default, { Name } from 'path'
          isTypeOnly = !!match[1];
          importPath = match[4] || match[3];
          const defaultName = match[2];
          const namedPart = match[3];

          symbolsMetadata.push({ name: defaultName, isType: isTypeOnly });
          importedSymbols.push(defaultName);

          const names = namedPart.split(',').map(n => n.trim());
          for (const n of names) {
            if (!n) continue;
            const isLocalType = n.startsWith('type ');
            const name = isLocalType ? n.substring(5).trim() : n;
            const cleanName = name.split(/\s+as\s+/i)[0].trim();
            symbolsMetadata.push({ name: cleanName, isType: isTypeOnly || isLocalType });
            importedSymbols.push(cleanName);
          }
        } else if (pattern.isDefault) {
          // Default import: import type Name from 'path'
          isTypeOnly = !!match[1];
          importPath = match[3];
          const name = match[2];
          symbolsMetadata.push({ name, isType: isTypeOnly });
          importedSymbols.push(name);
        } else if (pattern.isNamed) {
          // Named import: import type { Name } from 'path'
          isTypeOnly = !!match[1];
          importPath = match[3];
          const names = match[2].split(',').map(n => n.trim());
          for (const n of names) {
            if (!n) continue;
            const isLocalType = n.startsWith('type ');
            const name = isLocalType ? n.substring(5).trim() : n;
            const cleanName = name.split(/\s+as\s+/i)[0].trim();
            symbolsMetadata.push({ name: cleanName, isType: isTypeOnly || isLocalType });
            importedSymbols.push(cleanName);
          }
        } else {
          continue;
        }

        imports.push({
          raw: trimmed,
          importPath,
          imports: importedSymbols,
          isDefault: pattern.isDefault,
          isNamed: pattern.isNamed,
          isSideEffect: pattern.isSideEffect,
          isTypeOnly,
          symbolsMetadata,
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

    // Skip excluded imports
    if (EXCLUDED_IMPORTS.has(imp.importPath)) continue;
    if (imp.importPath.startsWith('@') || (!imp.importPath.startsWith('$') && !imp.importPath.startsWith('.'))) {
      continue;
    }

    // Skip relative imports that resolve correctly with their explicit extension
    if (!imp.importPath.startsWith('$')) {
      const cleanPath = imp.importPath.split('?')[0];
      const hasExtension = path.extname(cleanPath) !== '';
      const resolvedPath = path.resolve(path.dirname(filePath), imp.importPath);

      // Only skip if it HAS an extension and exists.
      // If it LACKS an extension, we want to check if it SHOULD have one (like .svelte)
      if (hasExtension && pathExists(resolvedPath)) continue;
    }

    // STEP 1: Check if the file exists
    let resolvedFile: string | null = null;
    if (imp.importPath.startsWith('$')) {
      resolvedFile = resolveAliasPath(imp.importPath);
    } else {
      const absPath = path.resolve(path.dirname(filePath), imp.importPath);
      resolvedFile = pathExists(absPath);
    }

    // Skip asset and CSS imports ONLY if they exist
    if (resolvedFile && imp.isDefault && (isAssetImport(imp.importPath) || isCssImport(imp.importPath))) {
      continue;
    }
    const missingFile = resolvedFile === null;

    // Check for missing extension (e.g. $ui/components/Input instead of Input.svelte)
    let missingExtension = false;
    if (resolvedFile) {
      const isSvelteRelated = resolvedFile.endsWith('.svelte') ||
                              resolvedFile.endsWith('.svelte.ts') ||
                              resolvedFile.endsWith('.svelte.js');

      if (isSvelteRelated) {
        const cleanImportPath = imp.importPath.split('?')[0];
        // Enforce .svelte extension even for .svelte.ts/.js files
        const hasCorrectExt = cleanImportPath.endsWith('.svelte');

        if (!hasCorrectExt) {
          missingExtension = true;
        }
      }
    }

        const missingSymbols: string[] = [];

        const missingTypeSymbols: string[] = [];

        const foundIn: ExportedSymbol[] = [];

        let actualFileLocation: string | undefined;

        let fixAsDefaultImport = false;

        let shouldAddTypeKeyword = false;



        if (missingFile || missingExtension) {

          // ... (existing logic for missing file/extension)

          if (missingFile) {

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



            // SEARCH OPTION 1: Local search in the broken file's path (if we can guess it)

            if (resolvedFile) {

              const relativePath = path.relative(PROJECT_ROOT, resolvedFile);

              const fileExps = fileExports.get(relativePath) || [];

              for (const exp of fileExps) {

                if (exp.name.toLowerCase() === importName.toLowerCase() ||

                    exp.name.startsWith('use') && exp.name.toLowerCase().includes(importName.toLowerCase())) {

                  foundIn.push(exp);

                }

              }

            }



            // SEARCH OPTION 2: Search for symbol in other files (global)

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

          const symbolMap = new Map(fileExps.map(e => [e.name, e]));

          const importedSymbolsSet = new Set(imp.imports);



          // Check for Svelte component imported as named import

          if ((isSvelteComponentImport(imp.importPath) || isSvelteFile) && imp.isNamed && !imp.isDefault) {

            const componentName = getComponentNameFromPath(imp.importPath) || path.basename(relativePath, '.svelte');

            if (componentName && imp.imports.length === 1 && imp.imports[0] === componentName) {

              fixAsDefaultImport = true;

            }

          }



          // Check if all imported symbols exist and if they are types

          let allImportedAreTypes = imp.imports.length > 0;

          for (const importName of imp.imports) {

            if (GENERIC_NAMES.has(importName)) continue;



            let symbolFound = exportedNames.has(importName);

            let isType = false;



                        if (!symbolFound && (isSvelteFile || isSvelteComponentImport(imp.importPath))) {



                          // Try case-insensitive match for Svelte components



                          for (const [exportedName, exp] of symbolMap.entries()) {



                            if (exportedName.toLowerCase() === importName.toLowerCase()) {



                              symbolFound = true;



                              isType = exp.type === 'type';



                              if (exp.type === 'default' && imp.isNamed && !imp.isDefault && imp.imports.length === 1) {



                                fixAsDefaultImport = true;



                              }



                              break;



                            }



                          }



                        } else if (symbolFound) {



                          const exp = symbolMap.get(importName)!;



                          isType = exp.type === 'type';



                          if (exp.type === 'default' && imp.isNamed && !imp.isDefault && imp.imports.length === 1) {



                            fixAsDefaultImport = true;



                          }



                        }



            if (!symbolFound) {

              missingSymbols.push(importName);

              allImportedAreTypes = false;



              // SEARCH SAME FILE for alternatives

              for (const exp of fileExps) {

                if (exp.name.toLowerCase() === importName.toLowerCase() ||

                    exp.name.startsWith('use') && exp.name.toLowerCase().includes(importName.toLowerCase())) {

                  foundIn.push(exp);

                }

              }



              // Search for symbol in other files

              const locations = symbolIndex.get(importName) || [];

              for (const loc of locations) {

                foundIn.push(loc);

              }

            } else {

              if (!isType) {

                allImportedAreTypes = false;

              } else if (!imp.isTypeOnly) {

                // It's a type, but NOT imported with 'type' keyword (global or inline)

                const metadata = imp.symbolsMetadata.find(m => m.name === importName);

                if (metadata && !metadata.isType) {

                  missingTypeSymbols.push(importName);

                }

              }

            }

          }



                // Check if we should add the 'type' keyword



                if (!imp.isTypeOnly && allImportedAreTypes && imp.imports.length > 0) {



                  // Only suggest adding global 'type' if NO symbols have an inline 'type' keyword



                  const alreadyHasInlineType = imp.symbolsMetadata.some(s => s.isType);



                  if (!alreadyHasInlineType) {



                    shouldAddTypeKeyword = true;



                  }



                }



              }







              // Record issue if any problems found



              if (missingFile || missingExtension || missingSymbols.length > 0 || missingTypeSymbols.length > 0 || fixAsDefaultImport || shouldAddTypeKeyword) {



                // Final check: does determineBestFix actually produce a DIFFERENT line?



                const proposedFix = determineBestFix({



                  filePath: path.relative(PROJECT_ROOT, filePath),



                  lineNumber: imp.lineNumber,



                  importStatement: imp,



                  missingFile,



                  missingExtension,



                  missingSymbols,



                  missingTypeSymbols,



                  foundIn,



                  actualFileLocation,



                  fixAsDefaultImport,



                  shouldAddTypeKeyword



                }, path.relative(PROJECT_ROOT, filePath));







                if (proposedFix && proposedFix.trim() !== imp.raw.trim()) {



                  issues.push({



                    filePath: path.relative(PROJECT_ROOT, filePath),



                    lineNumber: imp.lineNumber,



                    importStatement: imp,



                    missingFile,



                    missingExtension,



                    missingSymbols,



                    missingTypeSymbols,



                    foundIn,



                    actualFileLocation,



                    fixAsDefaultImport,



                    shouldAddTypeKeyword



                  });



                }



              }



            }







            return issues;



          }

/**
 * Analyze all imports in the project
 */
function analyzeImports(): FixReport {
  const searchDirs = [
    'routes',
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
      missingSymbols: allIssues.filter(i => i.missingSymbols.length > 0).length,
      missingTypeKeyword: allIssues.filter(i => i.shouldAddTypeKeyword).length,
      missingInlineType: allIssues.filter(i => i.missingTypeSymbols && i.missingTypeSymbols.length > 0).length
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

      const fix = determineBestFix(issue, filePath);
      if (fix) {
        console.log(`  📝 Proposed fix: ${fix}`);
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
  const fix = determineBestFix(issue, filePath);

  if (!fix) {
    return false;
  }

  console.log(`  📝 Applying fix: ${originalLine.trim()} → ${fix}`);
  lines[lineIndex] = fix;

  if (!DRY_RUN) {
    fs.writeFileSync(fullPath, lines.join('\n'));
  }

  return true;
}

/**
 * Determine the best fix for an issue
 */
function determineBestFix(issue: ImportIssue, sourceFile: string): string | null {
  const imp = issue.importStatement;

  // Helper to format the final import line with correct type keyword
  const formatFix = (newPath: string, newSymbols?: string[]) => {
    // Safety lock: check if the newPath actually resolves from the sourceFile's package
    let resolves = false;
    if (newPath.startsWith('$')) {
      resolves = !!resolveAliasPath(newPath);
    } else if (newPath.startsWith('.')) {
      const absSourceFile = path.resolve(PROJECT_ROOT, sourceFile);
      const absTargetPath = path.resolve(path.dirname(absSourceFile), newPath);
      resolves = !!pathExists(absTargetPath);
    }

    if (!resolves) {
      return null;
    }

                const isGlobalType = issue.shouldAddTypeKeyword || imp.isTypeOnly;

                const typeKeyword = isGlobalType ? 'type ' : '';

                const symbolsToUse = (newSymbols || imp.imports).map(name => {

                  const isInlineType = issue.missingTypeSymbols?.includes(name) ||

                                       imp.symbolsMetadata.find(m => m.name === name)?.isType;

                  return (isInlineType && !isGlobalType) ? `type ${name}` : name;

                });

        

                if (issue.fixAsDefaultImport) {

                  if (symbolsToUse.length === 1) {

                    return `import ${typeKeyword}${symbolsToUse[0]} from '${newPath}';`;

                  } else {

                    return `import ${typeKeyword}${symbolsToUse[0]}, { ${symbolsToUse.slice(1).join(', ')} } from '${newPath}';`;

                  }

                }

        

                if (imp.isDefault && !imp.isNamed) {

                  // If it was default, but we found a named export match

                  if (newSymbols && newSymbols.length === 1) {

                    return `import ${typeKeyword}{ ${symbolsToUse[0]} } from '${newPath}';`;

                  }

                  return `import ${typeKeyword}${symbolsToUse[0]} from '${newPath}';`;

                } else if (imp.isNamed && !imp.isDefault) {

                  return `import ${typeKeyword}{ ${symbolsToUse.join(', ')} } from '${newPath}';`;

                } else if (imp.isDefault && imp.isNamed) {

                  const defaultImport = symbolsToUse[0];

                  const namedImports = symbolsToUse.slice(1).join(', ');

                  return `import ${typeKeyword}${defaultImport}, { ${namedImports} } from '${newPath}';`;

                }

     else if (imp.isSideEffect) {
      return `import '${newPath}';`;
    }
    return null;
  };

  // 1. If only type keyword is missing (global or inline)
  if ((issue.shouldAddTypeKeyword || (issue.missingTypeSymbols && issue.missingTypeSymbols.length > 0)) && !issue.missingFile && !issue.missingExtension && !issue.fixAsDefaultImport) {
    return formatFix(imp.importPath);
  }

  // 2. Fix missing extension or missing file by suggesting correct path
  if ((issue.missingExtension || issue.missingFile) && issue.actualFileLocation) {
    const aliasPath = convertToAliasPath(issue.actualFileLocation);
    if (aliasPath) {
      return formatFix(aliasPath);
    }
  }

  // 3. Fix Svelte component import style (named to default)
  if (issue.fixAsDefaultImport) {
    const componentName = getComponentNameFromPath(imp.importPath) || imp.imports[0];
    let targetPath = imp.importPath;
    if (issue.actualFileLocation) {
      targetPath = convertToAliasPath(issue.actualFileLocation) || targetPath;
    }
    return formatFix(targetPath);
  }

  // 4. Fix missing symbols by suggesting correct import path or symbol name
  if (issue.missingSymbols.length > 0 && issue.foundIn.length > 0) {
    const missingSymbol = issue.missingSymbols[0];
    const uniqueFiles = new Map();
    for (const loc of issue.foundIn) {
      if (!uniqueFiles.has(loc.filePath)) {
        uniqueFiles.set(loc.filePath, loc);
      }
    }

    const locations = Array.from(uniqueFiles.values());

    // Check if any location is in the SAME file but with a different name
    const sameFileMatch = locations.find(l => {
      const resolved = resolveAliasPath(imp.importPath);
      return resolved && path.relative(PROJECT_ROOT, resolved) === l.filePath;
    });

    if (sameFileMatch && sameFileMatch.name !== missingSymbol) {
      const newSymbols = imp.imports.map(s => s === missingSymbol ? sameFileMatch.name : s);
      const isType = issue.shouldAddTypeKeyword || imp.isTypeOnly || sameFileMatch.type === 'type';
      const typeKeyword = isType ? 'type ' : '';

      // If it was default import (imp.isDefault && !imp.isNamed) and we found a named symbol,
      // convert to named import syntax { Symbol }
      if (imp.isDefault && !imp.isNamed && sameFileMatch.type !== 'default') {
        return `import ${typeKeyword}{ ${sameFileMatch.name} } from '${imp.importPath}';`;
      }
      return formatFix(imp.importPath, newSymbols);
    }

    const foundFile = locations[0].filePath;
    const aliasPath = convertToAliasPath(foundFile);

    if (aliasPath) {
      const newSymbols = imp.imports.map(s => s === missingSymbol ? locations[0].name : s);
      return formatFix(aliasPath, newSymbols);
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
