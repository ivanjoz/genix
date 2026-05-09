import * as fs from 'fs';
import * as path from 'path';

const PROJECT_ROOT = process.cwd();
const SHOULD_FIX = process.argv.includes('--fix');
const DRY_RUN = process.argv.includes('--dry-run');

// TypeScript keywords that look like identifiers when greedy-matched.
const GENERIC_NAMES = new Set(['type', 'interface', 'enum', 'class', 'function', 'const', 'let', 'var']);

// Single source of truth for which top-level project directories the indexer
// and import scanner walk. Earlier versions had three near-duplicates that
// drifted; keep this list and derive the rest from it.
const SEARCH_DIRS = [
  'core',
  'domain-components',
  'routes',
  'services',
  'ui-components',
  'ecommerce',
  'libs',
];

// Framework / npm packages we never try to resolve from the local index.
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

const symbolIndex = new Map<string, ExportedSymbol[]>();
const fileExports = new Map<string, ExportedSymbol[]>();
let aliasMap: Record<string, string> = {};

/**
 * Reassemble multi-line `import` / `export` statements into single logical
 * lines so the existing single-line regex patterns can match them. We only
 * accumulate when the keyword line is itself unbalanced (open brace or
 * unterminated string); a balanced single line is emitted as-is. The
 * accumulator stops the moment braces and strings are balanced again — we do
 * NOT wait for a trailing `;`, since `export interface Foo {}` and similar
 * declarations have no terminator and would otherwise eat following code.
 */
function joinMultilineStatements(content: string, keyword: 'import' | 'export'): { line: string, lineNumber: number }[] {
  const physicalLines = content.split('\n');
  const joined: { line: string, lineNumber: number }[] = [];

  let braceDepth = 0;
  let stringChar: string | null = null;
  const updateScanState = (segment: string) => {
    for (let c = 0; c < segment.length; c += 1) {
      const ch = segment[c];
      if (stringChar) {
        if (ch === '\\') { c += 1; continue; }
        if (ch === stringChar) { stringChar = null; }
      } else {
        if (ch === '"' || ch === "'" || ch === '`') { stringChar = ch; }
        else if (ch === '{') { braceDepth += 1; }
        else if (ch === '}') { braceDepth -= 1; }
      }
    }
  };

  let i = 0;
  while (i < physicalLines.length) {
    const physical = physicalLines[i];
    const trimmed = physical.trim();
    if (!trimmed.startsWith(keyword)) {
      joined.push({ line: physical, lineNumber: i + 1 });
      i += 1;
      continue;
    }

    // Reset per-statement scan state, then scan this line.
    braceDepth = 0;
    stringChar = null;
    const startLine = i + 1;
    updateScanState(physical);

    // Balanced after one line → it's a single-line statement.
    if (braceDepth === 0 && stringChar === null) {
      joined.push({ line: physical, lineNumber: startLine });
      i += 1;
      continue;
    }

    // Unbalanced → accumulate following lines until balanced again.
    let buffer = trimmed;
    i += 1;
    while (i < physicalLines.length && (braceDepth !== 0 || stringChar !== null)) {
      updateScanState(physicalLines[i]);
      buffer += ' ' + physicalLines[i].trim();
      i += 1;
    }
    joined.push({ line: buffer, lineNumber: startLine });
  }

  return joined;
}

function readAliasMappingsFromConfig(): void {
  const configPath = path.join(PROJECT_ROOT, 'svelte.config.js');
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

function findFilesRecursively(dir: string, extensions: string[] = []): string[] {
  const files: string[] = [];

  function walk(currentDir: string) {
    if (!fs.existsSync(currentDir)) return;
    const items = fs.readdirSync(currentDir);

    for (const item of items) {
      const fullPath = path.join(currentDir, item);
      const stat = fs.statSync(fullPath);

      if (stat.isDirectory()) {
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

function convertToAliasPath(filePath: string): string | null {
  let absPath = filePath;
  if (!path.isAbsolute(filePath)) {
    absPath = path.resolve(PROJECT_ROOT, filePath);
  }

  // Sort by descending pkgDir length so the most specific alias wins.
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

      // Verify the alias path resolves back — guards against bad aliasMap entries.
      if (resolveAliasPath(result)) {
        return result;
      }
    }
  }

  return null;
}

function searchForFile(fileName: string, extensions: string[] = []): string | null {
  const cleanFileName = fileName.split('?')[0];
  const exts = extensions.length > 0 ? extensions : ['.ts', '.js', '.svelte', '.svg', '.css'];

  // Walk only the configured project dirs (single source of truth above).
  const pathsToSearch = SEARCH_DIRS.filter(dir =>
    fs.existsSync(path.join(PROJECT_ROOT, dir))
  );

  if (path.extname(cleanFileName)) {
    for (const searchDir of pathsToSearch) {
      const dirPath = path.join(PROJECT_ROOT, searchDir);
      const allFiles = findFilesRecursively(dirPath, [path.extname(cleanFileName)]);
      const files = allFiles.filter(f => !f.includes('node_modules') && !f.includes('.svelte-kit'));
      for (const file of files) {
        if (path.basename(file) === cleanFileName) {
          return file;
        }
      }
    }
  } else {
    for (const ext of exts) {
      const fullFileName = cleanFileName + ext;
      for (const searchDir of pathsToSearch) {
        const dirPath = path.join(PROJECT_ROOT, searchDir);
        const allFiles = findFilesRecursively(dirPath, [ext]);
        const files = allFiles.filter(f => !f.includes('node_modules') && !f.includes('.svelte-kit'));
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

function isAssetImport(importPath: string): boolean {
  const assetExtensions = ['.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.ico', '.json'];
  const cleanPath = importPath.split('?')[0];
  return assetExtensions.some(ext => cleanPath.endsWith(ext));
}

function isCssImport(importPath: string): boolean {
  const cleanPath = importPath.split('?')[0];
  return cleanPath.endsWith('.css');
}

function isSvelteComponentImport(importPath: string): boolean {
  const cleanPath = importPath.split('?')[0];
  return cleanPath.endsWith('.svelte') ||
         cleanPath.match(/\/components\//) !== null ||
         cleanPath.match(/\/[A-Z][a-zA-Z]+$/) !== null;
}

function getComponentNameFromPath(importPath: string): string | null {
  const cleanPath = importPath.split('?')[0];
  const parts = cleanPath.split('/');
  const lastPart = parts[parts.length - 1];
  const match = lastPart.match(/^([A-Z][a-zA-Z]*)/);
  return match ? match[1] : null;
}

function extractCssClasses(content: string): string[] {
  const classes: string[] = [];
  const classPattern = /\.([a-zA-Z][a-zA-Z0-9_-]*)\s*\{/g;
  let match;

  while ((match = classPattern.exec(content)) !== null) {
    classes.push(match[1]);
  }

  return classes;
}

function extractExports(filePath: string): ExportedSymbol[] {
  const exports: ExportedSymbol[] = [];
  const content = fs.readFileSync(filePath, 'utf-8');

  // CSS modules expose their class names as named exports.
  if (filePath.endsWith('.css')) {
    const classes = extractCssClasses(content);
    for (const className of classes) {
      exports.push({ name: className, type: 'named', filePath, lineNumber: 0 });
    }
    return exports;
  }

  // Walk reassembled (multi-line aware) statements so `export { A,\n B }` works.
  const statements = joinMultilineStatements(content, 'export');

  for (const { line, lineNumber } of statements) {
    const trimmed = line.trim();

    if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) {
      continue;
    }

    // export default Foo / class Foo / function Foo / const Foo
    const defaultMatch = trimmed.match(/^export\s+default\s+(?:class|function|const|let|var)?\s*([a-zA-Z_$][a-zA-Z0-9_$]*)/);
    if (defaultMatch) {
      const fileName = path.basename(filePath, path.extname(filePath));
      const exportName = defaultMatch[1] || fileName;
      exports.push({ name: exportName, type: 'default', filePath, lineNumber });
      continue;
    }

    // export const/let/var/function/class Foo
    const namedMatch = trimmed.match(/^export\s+(?:const|let|var|function|class)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/);
    if (namedMatch) {
      exports.push({ name: namedMatch[1], type: 'named', filePath, lineNumber });
      continue;
    }

    // export type Foo / export interface Foo
    const typeMatch = trimmed.match(/^export\s+(?:type|interface)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/);
    if (typeMatch) {
      exports.push({ name: typeMatch[1], type: 'type', filePath, lineNumber });
      continue;
    }

    // export [type] { A, B } — note this branch must not match `export ... from`.
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
          lineNumber,
        });
      }
      continue;
    }

    // export const { A, B } = something
    const destructuredConstMatch = trimmed.match(/^export\s+(const|let|var)\s+\{\s*([^}]+)\s*\}\s*=/);
    if (destructuredConstMatch) {
      const names = destructuredConstMatch[2].split(',').map(n => n.trim());
      for (const name of names) {
        if (name) {
          exports.push({ name, type: 'named', filePath, lineNumber });
        }
      }
      continue;
    }

    // export [type] * | { A } from '...'
    const exportFromMatch = trimmed.match(/^export\s+(type\s+)?(?:\*|\{([^}]+)\})\s+from\s+['"]([^'"]+)['"]/);
    if (exportFromMatch) {
      const isGlobalType = !!exportFromMatch[1];
      const names = exportFromMatch[2] ? exportFromMatch[2].split(',').map(n => n.trim()) : ['*'];
      const fromPath = exportFromMatch[3];

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
                type: (isGlobalType || isLocalType || found.type === 'type') ? 'type' : found.type,
              });
            }
          }
        }
      }
      continue;
    }
  }

  // Svelte components are addressable via their filename as a default export.
  if (filePath.endsWith('.svelte')) {
    const componentName = path.basename(filePath, '.svelte');
    exports.push({ name: componentName, type: 'default', filePath, lineNumber: 0 });

    const capitalizedName = componentName.charAt(0).toUpperCase() + componentName.slice(1);
    if (capitalizedName !== componentName) {
      exports.push({ name: capitalizedName, type: 'default', filePath, lineNumber: 0 });
    }
  }

  return exports;
}

function buildSymbolIndex() {
  let totalFiles = 0;
  let totalExports = 0;

  for (const dir of SEARCH_DIRS) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;

    const allFiles = findFilesRecursively(dirPath, ['.ts', '.js', '.svelte', '.css']);
    const files = allFiles.filter(f => !f.includes('node_modules') && !f.includes('.svelte-kit'));

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

      totalFiles += 1;
      totalExports += exports.length;
    }
  }

  console.log(`📊 Indexed ${totalFiles} files with ${totalExports} exports`);
}

function parseImports(filePath: string): ImportStatement[] {
  const content = fs.readFileSync(filePath, 'utf-8');
  const imports: ImportStatement[] = [];

  // Multi-line aware: `import {\n  Foo,\n  Bar\n} from '...'` is a single logical line.
  const statements = joinMultilineStatements(content, 'import');

  const patterns = [
    // import [type] Default, { ... } from '...'
    {
      regex: /^import\s+(type\s+)?([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,\s*\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true, isDefault: true, isSideEffect: false,
    },
    // import [type] { ... } from '...'
    {
      regex: /^import\s+(type\s+)?\{\s*([^}]+)\s*\}\s+from\s+['"]([^'"]+)['"]/,
      isNamed: true, isDefault: false, isSideEffect: false,
    },
    // import [type] Default from '...'
    {
      regex: /^import\s+(type\s+)?([a-zA-Z_$][a-zA-Z0-9_$]*)\s+from\s+['"]([^'"]+)['"]/,
      isNamed: false, isDefault: true, isSideEffect: false,
    },
    // import '...' (side effect)
    {
      regex: /^import\s+['"]([^'"]+)['"]/,
      isNamed: false, isDefault: false, isSideEffect: true,
    },
  ];

  for (const { line, lineNumber } of statements) {
    const trimmed = line.trim();
    if (!trimmed.startsWith('import')) continue;

    for (const pattern of patterns) {
      const match = trimmed.match(pattern.regex);
      if (!match) continue;

      let importPath: string;
      let importedSymbols: string[] = [];
      let isTypeOnly = false;
      let symbolsMetadata: { name: string, isType: boolean }[] = [];

      if (pattern.isSideEffect) {
        importPath = match[1];
      } else if (pattern.isDefault && pattern.isNamed) {
        // Mixed: import [type] Default, { Name } from 'path'
        isTypeOnly = !!match[1];
        importPath = match[4];
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
        // import [type] Name from 'path'
        isTypeOnly = !!match[1];
        importPath = match[3];
        const name = match[2];
        symbolsMetadata.push({ name, isType: isTypeOnly });
        importedSymbols.push(name);
      } else if (pattern.isNamed) {
        // import [type] { Name } from 'path'
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
        lineNumber,
      });
      break;
    }
  }

  return imports;
}

/**
 * For a given import statement, decide whether it has any kind of issue
 * (missing file, missing extension, unknown symbol, missing `type` keyword,
 * Svelte-component-imported-as-named) and collect rename candidates.
 */
function checkImports(filePath: string): ImportIssue[] {
  const issues: ImportIssue[] = [];
  const imports = parseImports(filePath);

  for (const imp of imports) {
    if (imp.isSideEffect) continue;

    if (EXCLUDED_IMPORTS.has(imp.importPath)) continue;
    if (imp.importPath.startsWith('@') || (!imp.importPath.startsWith('$') && !imp.importPath.startsWith('.'))) {
      continue;
    }

    // For relative imports that already include an extension and resolve
    // cleanly, skip — there's nothing for us to do.
    if (!imp.importPath.startsWith('$')) {
      const cleanPath = imp.importPath.split('?')[0];
      const hasExtension = path.extname(cleanPath) !== '';
      const resolvedPath = path.resolve(path.dirname(filePath), imp.importPath);
      if (hasExtension && pathExists(resolvedPath)) continue;
    }

    // STEP 1: try to resolve the import to a file on disk.
    let resolvedFile: string | null = null;
    if (imp.importPath.startsWith('$')) {
      resolvedFile = resolveAliasPath(imp.importPath);
    } else {
      const absPath = path.resolve(path.dirname(filePath), imp.importPath);
      resolvedFile = pathExists(absPath);
    }

    // CSS / asset default-imports that resolve correctly are fine as-is.
    if (resolvedFile && imp.isDefault && (isAssetImport(imp.importPath) || isCssImport(imp.importPath))) {
      continue;
    }
    const missingFile = resolvedFile === null;

    // STEP 2: detect missing `.svelte` extension on a Svelte-related target.
    let missingExtension = false;
    if (resolvedFile) {
      const isSvelteRelated = resolvedFile.endsWith('.svelte') ||
                              resolvedFile.endsWith('.svelte.ts') ||
                              resolvedFile.endsWith('.svelte.js');
      if (isSvelteRelated) {
        const cleanImportPath = imp.importPath.split('?')[0];
        if (!cleanImportPath.endsWith('.svelte')) {
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
      if (missingFile) {
        // Find a same-basename file anywhere in the indexed dirs as a hint.
        const cleanImportPath = imp.importPath.split('?')[0];
        const fileName = path.basename(cleanImportPath);
        const baseName = path.basename(fileName, path.extname(fileName));
        const foundFile = searchForFile(baseName);
        if (foundFile) {
          actualFileLocation = path.relative(PROJECT_ROOT, foundFile);
        }
      } else if (missingExtension) {
        actualFileLocation = path.relative(PROJECT_ROOT, resolvedFile!);
      }

      // Surface symbol matches so determineBestFix can pick a coherent path.
      for (const importName of imp.imports) {
        if (GENERIC_NAMES.has(importName)) continue;
        missingSymbols.push(importName);

        if (resolvedFile) {
          const relativePath = path.relative(PROJECT_ROOT, resolvedFile);
          const fileExps = fileExports.get(relativePath) || [];
          for (const exp of fileExps) {
            if (
              exp.name.toLowerCase() === importName.toLowerCase() ||
              (exp.name.startsWith('use') && exp.name.toLowerCase().includes(importName.toLowerCase()))
            ) {
              foundIn.push(exp);
            }
          }
        }

        const locations = symbolIndex.get(importName) || [];
        for (const loc of locations) {
          foundIn.push(loc);
        }
      }
    } else {
      // STEP 3: file exists — verify each imported symbol is actually exported.
      const relativePath = path.relative(PROJECT_ROOT, resolvedFile!);
      let fileExps = fileExports.get(relativePath) || [];

      // Resolve `directory` imports through their index file.
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

      // import { Foo } from 'Foo.svelte' should be the default — flag it.
      if ((isSvelteComponentImport(imp.importPath) || isSvelteFile) && imp.isNamed && !imp.isDefault) {
        const componentName = getComponentNameFromPath(imp.importPath) || path.basename(relativePath, '.svelte');
        if (componentName && imp.imports.length === 1 && imp.imports[0] === componentName) {
          fixAsDefaultImport = true;
        }
      }

      let allImportedAreTypes = imp.imports.length > 0;
      for (const importName of imp.imports) {
        if (GENERIC_NAMES.has(importName)) continue;

        let symbolFound = exportedNames.has(importName);
        let isType = false;

        if (!symbolFound && (isSvelteFile || isSvelteComponentImport(imp.importPath))) {
          // Case-insensitive fallback for Svelte components.
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

          // Look for a renamed/case-mismatched candidate in the same file first.
          for (const exp of fileExps) {
            if (
              exp.name.toLowerCase() === importName.toLowerCase() ||
              (exp.name.startsWith('use') && exp.name.toLowerCase().includes(importName.toLowerCase()))
            ) {
              foundIn.push(exp);
            }
          }

          const locations = symbolIndex.get(importName) || [];
          for (const loc of locations) {
            foundIn.push(loc);
          }
        } else {
          if (!isType) {
            allImportedAreTypes = false;
          } else if (!imp.isTypeOnly) {
            // Symbol is a type but lacks the inline `type` modifier.
            const metadata = imp.symbolsMetadata.find(m => m.name === importName);
            if (metadata && !metadata.isType) {
              missingTypeSymbols.push(importName);
            }
          }
        }
      }

      // If every imported symbol is a type and there's no inline `type`,
      // suggest hoisting `type` to the import keyword.
      if (!imp.isTypeOnly && allImportedAreTypes && imp.imports.length > 0) {
        const alreadyHasInlineType = imp.symbolsMetadata.some(s => s.isType);
        if (!alreadyHasInlineType) {
          shouldAddTypeKeyword = true;
        }
      }
    }

    const hasIssue =
      missingFile ||
      missingExtension ||
      missingSymbols.length > 0 ||
      missingTypeSymbols.length > 0 ||
      fixAsDefaultImport ||
      shouldAddTypeKeyword;

    if (!hasIssue) continue;

    // Suppress the issue if our proposed rewrite would be a no-op.
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
      shouldAddTypeKeyword,
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
        shouldAddTypeKeyword,
      });
    }
  }

  return issues;
}

function analyzeImports(): FixReport {
  let totalFiles = 0;
  let totalImports = 0;
  const allIssues: ImportIssue[] = [];

  for (const dir of SEARCH_DIRS) {
    const dirPath = path.join(PROJECT_ROOT, dir);
    if (!fs.existsSync(dirPath)) continue;

    const allFiles = findFilesRecursively(dirPath, ['.ts', '.js', '.svelte']);
    const files = allFiles.filter(f => !f.includes('node_modules') && !f.includes('.svelte-kit'));

    for (const file of files) {
      const imports = parseImports(file);
      const issues = checkImports(file);
      totalFiles += 1;
      totalImports += imports.length;
      allIssues.push(...issues);
    }
  }

  return {
    totalIssues: allIssues.length,
    issuesByType: {
      missingFile: allIssues.filter(i => i.missingFile).length,
      missingExtension: allIssues.filter(i => i.missingExtension).length,
      missingSymbols: allIssues.filter(i => i.missingSymbols.length > 0).length,
      missingTypeKeyword: allIssues.filter(i => i.shouldAddTypeKeyword).length,
      missingInlineType: allIssues.filter(i => i.missingTypeSymbols && i.missingTypeSymbols.length > 0).length,
    },
    issues: allIssues,
  };
}

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
  if (!fix) return false;

  console.log(`  📝 Applying fix: ${originalLine.trim()} → ${fix}`);
  lines[lineIndex] = fix;

  if (!DRY_RUN) {
    fs.writeFileSync(fullPath, lines.join('\n'));
  }
  return true;
}

function determineBestFix(issue: ImportIssue, sourceFile: string): string | null {
  const imp = issue.importStatement;

  // Render an import line and verify the proposed path actually resolves.
  const formatFix = (newPath: string, newSymbols?: string[]) => {
    let resolves = false;
    if (newPath.startsWith('$')) {
      resolves = !!resolveAliasPath(newPath);
    } else if (newPath.startsWith('.')) {
      const absSourceFile = path.resolve(PROJECT_ROOT, sourceFile);
      const absTargetPath = path.resolve(path.dirname(absSourceFile), newPath);
      resolves = !!pathExists(absTargetPath);
    }
    if (!resolves) return null;

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
      }
      return `import ${typeKeyword}${symbolsToUse[0]}, { ${symbolsToUse.slice(1).join(', ')} } from '${newPath}';`;
    }

    if (imp.isDefault && !imp.isNamed) {
      // Originally default, but we matched a named export — convert.
      if (newSymbols && newSymbols.length === 1) {
        return `import ${typeKeyword}{ ${symbolsToUse[0]} } from '${newPath}';`;
      }
      return `import ${typeKeyword}${symbolsToUse[0]} from '${newPath}';`;
    }
    if (imp.isNamed && !imp.isDefault) {
      return `import ${typeKeyword}{ ${symbolsToUse.join(', ')} } from '${newPath}';`;
    }
    if (imp.isDefault && imp.isNamed) {
      const defaultImport = symbolsToUse[0];
      const namedImports = symbolsToUse.slice(1).join(', ');
      return `import ${typeKeyword}${defaultImport}, { ${namedImports} } from '${newPath}';`;
    }
    if (imp.isSideEffect) {
      return `import '${newPath}';`;
    }
    return null;
  };

  // Branch 1: only the `type` keyword is missing.
  if (
    (issue.shouldAddTypeKeyword || (issue.missingTypeSymbols && issue.missingTypeSymbols.length > 0)) &&
    !issue.missingFile && !issue.missingExtension && !issue.fixAsDefaultImport
  ) {
    return formatFix(imp.importPath);
  }

  // Branch 2: file moved or extension missing — point at the actual location.
  if ((issue.missingExtension || issue.missingFile) && issue.actualFileLocation) {
    const aliasPath = convertToAliasPath(issue.actualFileLocation);
    if (aliasPath) return formatFix(aliasPath);
  }

  // Branch 3: Svelte component imported as named — flip to default.
  if (issue.fixAsDefaultImport) {
    let targetPath = imp.importPath;
    if (issue.actualFileLocation) {
      targetPath = convertToAliasPath(issue.actualFileLocation) || targetPath;
    }
    return formatFix(targetPath);
  }

  // Branch 4: symbol not found here — prefer a same-file rename, else point
  // at whichever file does export it.
  if (issue.missingSymbols.length > 0 && issue.foundIn.length > 0) {
    const missingSymbol = issue.missingSymbols[0];
    const uniqueFiles = new Map<string, ExportedSymbol>();
    for (const loc of issue.foundIn) {
      if (!uniqueFiles.has(loc.filePath)) {
        uniqueFiles.set(loc.filePath, loc);
      }
    }
    const locations = Array.from(uniqueFiles.values());

    // Same-file rename: matches if the import already resolves to a file
    // (alias OR relative) that re-exports the symbol under a different name.
    const importResolvedFile = (() => {
      if (imp.importPath.startsWith('$')) {
        const resolved = resolveAliasPath(imp.importPath);
        return resolved ? path.relative(PROJECT_ROOT, resolved) : null;
      }
      if (imp.importPath.startsWith('.')) {
        const absSourceFile = path.resolve(PROJECT_ROOT, sourceFile);
        const absTargetPath = path.resolve(path.dirname(absSourceFile), imp.importPath);
        const resolved = pathExists(absTargetPath);
        return resolved ? path.relative(PROJECT_ROOT, resolved) : null;
      }
      return null;
    })();

    const sameFileMatch = importResolvedFile
      ? locations.find(l => l.filePath === importResolvedFile)
      : undefined;

    if (sameFileMatch && sameFileMatch.name !== missingSymbol) {
      const newSymbols = imp.imports.map(s => s === missingSymbol ? sameFileMatch.name : s);
      const isType = issue.shouldAddTypeKeyword || imp.isTypeOnly || sameFileMatch.type === 'type';
      const typeKeyword = isType ? 'type ' : '';

      // default import that actually wants a named symbol — convert syntax.
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
    if (success) fixedCount += 1;
    else failedCount += 1;
  }

  console.log('\n' + '='.repeat(80));
  console.log(`✅ Fixed: ${fixedCount}`);
  console.log(`❌ Failed: ${failedCount}`);
  console.log('='.repeat(80));
}

function main() {
  console.log('🔍 Intelligent Import Fixer');
  console.log('='.repeat(80));

  console.log('\n⚙️  Reading alias mappings from svelte.config.js...');
  readAliasMappingsFromConfig();

  console.log('\n📚 Building symbol index...');
  buildSymbolIndex();

  console.log('\n🔍 Analyzing imports...');
  const report = analyzeImports();

  displayReport(report);

  if (SHOULD_FIX && report.totalIssues > 0) {
    applyFixes(report);
  }
}

main();
