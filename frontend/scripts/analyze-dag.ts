#!/usr/bin/env bun
/**
 * DAG Dependency Analysis Script
 *
 * Analyzes the dependency graph between packages and identifies:
 * 1. Circular dependencies
 * 2. Hierarchy violations (wrong direction of imports)
 * 3. Missing dependencies in the expected structure
 * 4. Visualization of the actual dependency graph
 *
 * Expected hierarchy:
 *   pkg-core (base)
 *     ↓
 *   pkg-services
 *     ↓
 *   pkg-ui, pkg-components
 *     ↓
 *   pkg-store, pkg-main (leaf nodes)
 */

import { readFileSync, existsSync, readdirSync, statSync } from 'fs';
import { resolve, relative, dirname, join } from 'path';

// ============================================
// Types
// ============================================

interface PackageDependency {
  fromPackage: string;
  toPackage: string;
  files: Set<string>;
  symbols: string[];
}

interface ImportInfo {
  file: string;
  importPath: string;
  symbol: string;
  packageName: string;
}

interface Node {
  name: string;
  level: number;
  dependencies: Set<string>;
  dependents: Set<string>;
  violations: string[];
}

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const PACKAGES_DIR = FRONTEND_DIR;

const PACKAGES = ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-components', 'pkg-store', 'pkg-main'];

// Expected hierarchy levels (lower = more base)
const PACKAGE_LEVELS: Record<string, number> = {
	'pkg-core': 0,
  'pkg-components': 1,
  'pkg-services': 2,
  'pkg-ui': 3,
  'pkg-store': 4,
  'pkg-main': 5
};

// Allowed dependencies: package -> [allowed dependencies]
const ALLOWED_DEPENDENCIES: Record<string, string[]> = {
  'pkg-core': [], // No dependencies allowed
  'pkg-services': ['pkg-core'],
  'pkg-ui': ['pkg-core', 'pkg-services'],
  'pkg-components': ['pkg-core', 'pkg-services', 'pkg-ui'],
  'pkg-store': ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-components'],
  'pkg-main': ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-components']
};

// ============================================
// Global State
// ============================================

const dependencies: PackageDependency[] = [];
const packageNodes = new Map<string, Node>();
const importMap = new Map<string, Set<string>>(); // from -> [to]

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

function determineTargetPackage(importPath: string, sourcePackage: string): string | null {
  // Handle alias imports
  const aliasMap: Record<string, string> = {
    '$core': 'pkg-core',
    '$services': 'pkg-services',
    '$shared': 'pkg-services', // shared is in pkg-services
    '$components': 'pkg-ui',
    '$ecommerce': 'pkg-components',
    '$ecommerceComponents': 'pkg-ui',
    '$lib': sourcePackage, // $lib points to current package's lib
    '$http': 'pkg-core',
    '$app': null, // SvelteKit built-in, not a package
  };

  // Extract alias from path
  const aliasMatch = importPath.match(/^\$(\w+)/);
  if (aliasMatch) {
    const alias = aliasMatch[1];
    return aliasMap[`$${alias}`] || null;
  }

  // Handle relative imports
  if (importPath.startsWith('.')) {
    return sourcePackage; // Same package
  }

  // External dependency (node_modules)
  return null;
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
  const packageName = getPackageName(filePath);

  if (!packageName) {
    return [];
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith('//') || trimmed.startsWith('*') || trimmed.startsWith('/*')) {
      continue;
    }

    // Match import patterns
    const importRegex = /import\s+(?:type\s+)?[\w\s{},*]+\s+from\s+['"]([^'"]+)['"]/;
    const match = trimmed.match(importRegex);

    if (match) {
      const importPath = match[1];

      // Extract symbol name for named imports
      const symbolMatch = trimmed.match(/import\s+(?:type\s+)?\{?\s*([^},]+)/);
      const symbol = symbolMatch ? symbolMatch[1].trim() : 'default';

      imports.push({
        file: filePath,
        importPath,
        symbol,
        packageName
      });
    }
  }

  return imports;
}

// ============================================
// Dependency Analysis
// ============================================

function analyzeDependencies() {
  console.log('📊 Analyzing package dependencies...\n');

  // Collect all files
  const allFiles: string[] = [];
  for (const pkg of PACKAGES) {
    const packagePath = join(PACKAGES_DIR, pkg);
    const files = getAllFiles(packagePath);
    allFiles.push(...files);
  }

  console.log(`   Found ${allFiles.length} files\n`);

  // Parse all imports
  const allImports: ImportInfo[] = [];
  for (const file of allFiles) {
    try {
      const content = readFileSync(file, 'utf-8');
      const imports = parseImports(content, file);
      allImports.push(...imports);
    } catch (error) {
      console.warn(`   Warning: Could not parse ${file}`);
    }
  }

  console.log(`   Parsed ${allImports.length} imports\n`);

  // Build dependency map
  for (const imp of allImports) {
    const targetPackage = determineTargetPackage(imp.importPath, imp.packageName);

    if (targetPackage && targetPackage !== imp.packageName) {
      const existingDep = dependencies.find(
        d => d.fromPackage === imp.packageName && d.toPackage === targetPackage
      );

      if (existingDep) {
        existingDep.files.add(imp.file);
        existingDep.symbols.push(imp.symbol);
      } else {
        dependencies.push({
          fromPackage: imp.packageName,
          toPackage: targetPackage,
          files: new Set([imp.file]),
          symbols: [imp.symbol]
        });
      }
    }
  }

  console.log(`   Found ${dependencies.length} inter-package dependencies\n`);
}

function buildDependencyGraph() {
  console.log('🔗 Building dependency graph...\n');

  // Initialize nodes
  for (const pkg of PACKAGES) {
    packageNodes.set(pkg, {
      name: pkg,
      level: PACKAGE_LEVELS[pkg],
      dependencies: new Set(),
      dependents: new Set(),
      violations: []
    });
  }

  // Build edges
  for (const dep of dependencies) {
    const fromNode = packageNodes.get(dep.fromPackage);
    const toNode = packageNodes.get(dep.toPackage);

    if (fromNode && toNode) {
      fromNode.dependencies.add(dep.toPackage);
      toNode.dependents.add(dep.fromPackage);
    }
  }

  console.log('   Dependency graph built\n');
}

function detectViolations() {
  console.log('🚨 Detecting violations...\n');

  const violations: string[] = [];

  // Check hierarchy violations
  for (const dep of dependencies) {
    const fromLevel = PACKAGE_LEVELS[dep.fromPackage];
    const toLevel = PACKAGE_LEVELS[dep.toPackage];
    const allowed = ALLOWED_DEPENDENCIES[dep.fromPackage]?.includes(dep.toPackage);

    if (!allowed) {
      const violation = `❌ HIERARCHY: ${dep.fromPackage} → ${dep.toPackage} (not allowed)`;
      violations.push(violation);

      const node = packageNodes.get(dep.fromPackage);
      if (node) {
        node.violations.push(violation);
      }
    }
  }

  // Check for circular dependencies using DFS
  const visited = new Set<string>();
  const recStack = new Set<string>();

  function detectCycle(pkg: string, path: string[]): boolean {
    visited.add(pkg);
    recStack.add(pkg);
    path.push(pkg);

    const node = packageNodes.get(pkg);
    if (!node) return false;

    for (const dep of node.dependencies) {
      if (!visited.has(dep)) {
        if (detectCycle(dep, path)) return true;
      } else if (recStack.has(dep)) {
        const cycle = path.slice(path.indexOf(dep)).concat([dep]);
        violations.push(`🔄 CIRCULAR: ${cycle.join(' → ')}`);
        return true;
      }
    }

    recStack.delete(pkg);
    path.pop();
    return false;
  }

  for (const pkg of PACKAGES) {
    if (!visited.has(pkg)) {
      detectCycle(pkg, []);
    }
  }

  // Check for unused packages (no dependents, not a leaf node)
  for (const [name, node] of packageNodes) {
    if (node.dependents.size === 0 && node.name !== 'pkg-store' && node.name !== 'pkg-main') {
      node.violations.push(`⚠️  UNUSED: Package has no dependents`);
    }
  }

  console.log(`   Found ${violations.length} violations\n`);

  return violations;
}

function visualizeGraph() {
  console.log('📊 DEPENDENCY GRAPH');
  console.log('='.repeat(80));
  console.log();

  for (const [name, node] of packageNodes) {
    console.log(`\n${node.name} (Level ${node.level})`);
    console.log('─'.repeat(40));

    if (node.dependencies.size === 0) {
      console.log('   No dependencies (base package)');
    } else {
      console.log('   Dependencies:');
      for (const dep of node.dependencies) {
        const depNode = packageNodes.get(dep);
        const allowed = ALLOWED_DEPENDENCIES[name]?.includes(dep);
        const icon = allowed ? '✓' : '✗';
        console.log(`   ${icon} ${dep} (Level ${depNode?.level})`);
      }
    }

    if (node.dependents.size > 0) {
      console.log(`   Used by: ${Array.from(node.dependents).join(', ')}`);
    }

    if (node.violations.length > 0) {
      console.log('\n   VIOLATIONS:');
      for (const violation of node.violations) {
        console.log(`   ${violation}`);
      }
    }
  }

  console.log('\n' + '='.repeat(80));
}

function printDetailedDependencies() {
  console.log('\n📋 DETAILED DEPENDENCIES');
  console.log('='.repeat(80));
  console.log();

  for (const dep of dependencies) {
    const allowed = ALLOWED_DEPENDENCIES[dep.fromPackage]?.includes(dep.toPackage);
    const icon = allowed ? '✓' : '✗';

    console.log(`${icon} ${dep.fromPackage} → ${dep.toPackage}`);
    console.log(`   Files affected: ${dep.files.size}`);
    console.log(`   Symbols: ${[...new Set(dep.symbols)].slice(0, 10).join(', ')}${dep.symbols.length > 10 ? '...' : ''}`);

    if (!allowed) {
      console.log(`   VIOLATION: This dependency is not allowed in the hierarchy`);
      console.log(`   Suggested action: Move code or refactor to avoid this dependency`);
    }

    console.log();
  }

  console.log('='.repeat(80));
}

function printViolationsSummary(violations: string[]) {
  if (violations.length === 0) {
    console.log('\n✅ No violations detected! The dependency graph is valid.\n');
    return;
  }

  console.log('\n🚨 VIOLATIONS SUMMARY');
  console.log('='.repeat(80));
  console.log();

  const hierarchyViolations = violations.filter(v => v.includes('HIERARCHY'));
  const circularViolations = violations.filter(v => v.includes('CIRCULAR'));
  const unusedViolations = violations.filter(v => v.includes('UNUSED'));

  if (hierarchyViolations.length > 0) {
    console.log(`\nHierarchy Violations (${hierarchyViolations.length}):`);
    for (const v of hierarchyViolations) {
      console.log(`  ${v}`);
    }
  }

  if (circularViolations.length > 0) {
    console.log(`\nCircular Dependencies (${circularViolations.length}):`);
    for (const v of circularViolations) {
      console.log(`  ${v}`);
    }
  }

  if (unusedViolations.length > 0) {
    console.log(`\nUnused Packages (${unusedViolations.length}):`);
    for (const v of unusedViolations) {
      console.log(`  ${v}`);
    }
  }

  console.log('\n' + '='.repeat(80));
}

function printRecommendations() {
  console.log('\n💡 RECOMMENDATIONS');
  console.log('='.repeat(80));
  console.log();

  console.log('Expected Hierarchy:');
  console.log('  pkg-core (Level 0) - Base utilities, no dependencies');
  console.log('  pkg-services (Level 1) - Can depend on pkg-core');
  console.log('  pkg-ui (Level 2) - Can depend on pkg-core, pkg-services');
  console.log('  pkg-components (Level 2) - Can depend on pkg-core, pkg-services');
  console.log('  pkg-store (Level 3) - Can depend on pkg-core, pkg-services, pkg-ui, pkg-components');
  console.log('  pkg-main (Level 3) - Can depend on pkg-core, pkg-services, pkg-ui, pkg-components');
  console.log();

  const hasViolations = Array.from(packageNodes.values()).some(n => n.violations.length > 0);

  if (hasViolations) {
    console.log('Actions to fix violations:');
    console.log('  1. Identify the violating dependencies above');
    console.log('  2. Move shared code to the appropriate base package');
    console.log('  3. Refactor to use only allowed dependencies');
    console.log('  4. Update import paths to use the new structure');
    console.log('  5. Run this script again to verify');
  } else {
    console.log('✅ All dependencies follow the expected hierarchy!');
  }

  console.log('\n' + '='.repeat(80));
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('🔍 Starting DAG Dependency Analysis...\n');

  analyzeDependencies();
  buildDependencyGraph();
  const violations = detectViolations();

  console.log('\n' + '═'.repeat(80));
  console.log('ANALYSIS COMPLETE');
  console.log('═'.repeat(80));

  visualizeGraph();
  printDetailedDependencies();
  printViolationsSummary(violations);
  printRecommendations();

  const totalViolations = Array.from(packageNodes.values()).reduce((sum, node) => sum + node.violations.length, 0);
  process.exit(totalViolations > 0 ? 1 : 0);
}

main();
