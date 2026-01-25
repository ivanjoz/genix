#!/usr/bin/env bun
/**
 * Turborepo Migration Script
 *
 * This script migrates the frontend monorepo structure to a package-based structure
 * following the expected DAG hierarchy:
 *   pkg-core (base)
 *     ↓
 *   pkg-services
 *     ↓
 *   pkg-ui, pkg-components
 *     ↓
 *   pkg-store, pkg-app (leaf nodes)
 *
 * USAGE:
 *   bun scripts/migration/migrate-to-turbo.ts [--dry-run] [--verbose]
 *
 * OPTIONS:
 *   --dry-run    Show what would be moved without actually moving
 *   --verbose    Show detailed file-by-file information
 */

import { readFileSync, existsSync, readdirSync, statSync, renameSync, mkdirSync, writeFileSync } from 'fs';
import { resolve, relative, dirname, join, basename } from 'path';

// ============================================
// Types
// ============================================

interface MigrationRule {
  sourceDir: string;
  targetPackage: string;
  targetDir: string;
  description: string;
}

interface MigrationStats {
  totalFiles: number;
  totalDirectories: number;
  skipped: number;
  errors: string[];
  movedFiles: { from: string; to: string }[];
  movedDirs: { from: string; to: string }[];
}

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const DRY_RUN = process.argv.includes('--dry-run');
const VERBOSE = process.argv.includes('--verbose');

// Migration rules - maps source directories to target packages
const MIGRATION_RULES: MigrationRule[] = [
  // Core utilities and types
  {
    sourceDir: 'core',
    targetPackage: 'pkg-core',
    targetDir: 'core',
    description: 'Core utilities and store'
  },
  {
    sourceDir: 'types',
    targetPackage: 'pkg-core',
    targetDir: 'types',
    description: 'Type definitions'
  },
  {
    sourceDir: 'lib',
    targetPackage: 'pkg-core',
    targetDir: 'lib',
    description: 'Core library utilities (helpers, http, security, etc.)'
  },
  
  // Services layer
  {
    sourceDir: 'services',
    targetPackage: 'pkg-services',
    targetDir: 'services',
    description: 'API services'
  },
  {
    sourceDir: 'shared',
    targetPackage: 'pkg-services',
    targetDir: 'shared',
    description: 'Shared utilities'
  },
  
  // UI components
  {
    sourceDir: 'components',
    targetPackage: 'pkg-ui',
    targetDir: 'components',
    description: 'Reusable UI components'
  },
  {
    sourceDir: 'assets',
    targetPackage: 'pkg-ui',
    targetDir: 'assets',
    description: 'Shared assets (SVGs, icons)'
  },
  
  // Domain-specific components
  {
    sourceDir: 'ecommerce',
    targetPackage: 'pkg-components',
    targetDir: 'ecommerce',
    description: 'Ecommerce-specific components'
  },
  {
    sourceDir: 'ecommerce-components',
    targetPackage: 'pkg-components',
    targetDir: 'ecommerce-components',
    description: 'Additional ecommerce components'
  },
  
  // Application routes and logic
  {
    sourceDir: 'routes',
    targetPackage: 'pkg-app',
    targetDir: 'routes',
    description: 'SvelteKit application routes'
  },
  {
    sourceDir: 'stores',
    targetPackage: 'pkg-store',
    targetDir: 'stores',
    description: 'Svelte stores'
  },
  
  // Infrastructure
  {
    sourceDir: 'workers',
    targetPackage: 'pkg-app',
    targetDir: 'workers',
    description: 'Web workers'
  },
  {
    sourceDir: 'functions',
    targetPackage: 'pkg-app',
    targetDir: 'functions',
    description: 'Utility functions'
  },
  {
    sourceDir: 'static',
    targetPackage: 'pkg-app',
    targetDir: 'static',
    description: 'Static files'
  },
  {
    sourceDir: 'genix',
    targetPackage: 'pkg-app',
    targetDir: 'genix',
    description: 'Genix-specific code'
  },
];

// Folders to skip (not migrated)
const SKIP_FOLDERS = [
  'node_modules',
  '.git',
  '.svelte-kit',
  'build',
  'dist',
  '.turbo',
  'pkg-core',
  'pkg-services',
  'pkg-ui',
  'pkg-components',
  'pkg-store',
  'pkg-app',
  'scripts',
  'tmp'
];

// Files to skip (not migrated)
const SKIP_FILES = [
  'package.json',
  'tsconfig.json',
  'svelte.config.js',
  'vite.config.ts',
  '.gitignore',
  '.npmrc',
  'README.md',
  'bun.lock',
  'plugins.js',
  'postbuild.js',
  'env.ts',
  'app.css',
  'app.d.ts',
  'app.html',
  'globals.d.ts',
  'STORE.md'
];

// ============================================
// Global State
// ============================================

const stats: MigrationStats = {
  totalFiles: 0,
  totalDirectories: 0,
  skipped: 0,
  errors: [],
  movedFiles: [],
  movedDirs: []
};

// ============================================
// Utilities
// ============================================

function log(message: string, level: 'info' | 'success' | 'warning' | 'error' = 'info') {
  const colors = {
    info: '\x1b[36m',    // Cyan
    success: '\x1b[32m', // Green
    warning: '\x1b[33m', // Yellow
    error: '\x1b[31m'    // Red
  };
  const reset = '\x1b[0m';
  console.log(`${colors[level]}${message}${reset}`);
}

function ensureDirectoryExists(dirPath: string) {
  if (!existsSync(dirPath)) {
    mkdirSync(dirPath, { recursive: true });
  }
}

function getAllFiles(dir: string): string[] {
  if (!existsSync(dir)) {
    return [];
  }

  const files: string[] = [];
  const entries = readdirSync(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = join(dir, entry.name);

    if (entry.isDirectory()) {
      files.push(...getAllFiles(fullPath));
    } else if (entry.isFile()) {
      files.push(fullPath);
    }
  }

  return files;
}

function shouldSkip(path: string, isDirectory: boolean): boolean {
  const relativePath = relative(FRONTEND_DIR, path);
  const name = basename(path);

  if (isDirectory) {
    return SKIP_FOLDERS.includes(name) || name.startsWith('pkg-');
  } else {
    return SKIP_FILES.includes(name);
  }
}

// ============================================
// Migration Functions
// ============================================

function migrateFile(sourcePath: string, targetPath: string): boolean {
  try {
    if (VERBOSE) {
      log(`  📄 ${basename(sourcePath)} → ${basename(targetPath)}`, 'info');
    }

    if (!DRY_RUN) {
      // Ensure target directory exists
      ensureDirectoryExists(dirname(targetPath));
      
      // Move the file
      renameSync(sourcePath, targetPath);
    }

    stats.movedFiles.push({ from: sourcePath, to: targetPath });
    stats.totalFiles++;
    return true;
  } catch (error) {
    const errorMsg = `Failed to move ${sourcePath}: ${error}`;
    stats.errors.push(errorMsg);
    log(`  ❌ ${errorMsg}`, 'error');
    return false;
  }
}

function migrateDirectory(sourceDir: string, targetDir: string): boolean {
  try {
    if (VERBOSE) {
      log(`  📁 ${basename(sourceDir)} → ${basename(targetDir)}`, 'info');
    }

    if (!DRY_RUN) {
      // Move entire directory
      renameSync(sourceDir, targetDir);
    }

    stats.movedDirs.push({ from: sourceDir, to: targetDir });
    stats.totalDirectories++;
    return true;
  } catch (error) {
    const errorMsg = `Failed to move ${sourceDir}: ${error}`;
    stats.errors.push(errorMsg);
    log(`  ❌ ${errorMsg}`, 'error');
    return false;
  }
}

function executeMigrationRule(rule: MigrationRule) {
  const sourcePath = resolve(FRONTEND_DIR, rule.sourceDir);
  
  if (!existsSync(sourcePath)) {
    log(`⚠️  Source directory does not exist: ${rule.sourceDir}`, 'warning');
    stats.skipped++;
    return;
  }

  const targetPath = resolve(FRONTEND_DIR, rule.targetPackage, rule.targetDir);
  
  log(`\n🔄 ${rule.description}`, 'info');
  log(`   ${rule.sourceDir} → ${rule.targetPackage}/${rule.targetDir}`, 'info');

  // Try to move the entire directory first
  const moved = migrateDirectory(sourcePath, targetPath);
  
  if (!moved) {
    // If moving directory failed, try file-by-file
    log(`   Moving file-by-file...`, 'warning');
    const files = getAllFiles(sourcePath);
    
    for (const file of files) {
      const relativePath = relative(sourcePath, file);
      const targetFilePath = join(targetPath, relativePath);
      migrateFile(file, targetFilePath);
    }
    
    // Remove empty source directory
    if (!DRY_RUN && existsSync(sourcePath)) {
      try {
        // Try to remove the empty directory
        const fs = require('fs');
        fs.rmSync(sourcePath, { recursive: true, force: true });
      } catch (error) {
        // Directory might not be empty, that's ok
      }
    }
  }
}

function createPackageStructure() {
  log('\n🏗️  Creating package structure...', 'info');
  
  // Create all package directories
  const packages = ['pkg-core', 'pkg-services', 'pkg-ui', 'pkg-components', 'pkg-store', 'pkg-app'];
  
  for (const pkg of packages) {
    const pkgPath = resolve(FRONTEND_DIR, pkg);
    ensureDirectoryExists(pkgPath);
    
    // Create lib directory for each package
    ensureDirectoryExists(join(pkgPath, 'lib'));
    
    if (VERBOSE) {
      log(`  ✓ Created ${pkg}/`, 'success');
    }
  }
}

function printSummary() {
  console.log('\n' + '='.repeat(80));
  console.log('MIGRATION SUMMARY');
  console.log('='.repeat(80));
  console.log();
  
  log(`Mode: ${DRY_RUN ? 'DRY RUN (no changes made)' : 'LIVE (files moved)'}`, 'info');
  console.log();
  
  log(`Directories moved: ${stats.totalDirectories}`, 'success');
  log(`Files moved: ${stats.totalFiles}`, 'success');
  log(`Skipped: ${stats.skipped}`, 'warning');
  log(`Errors: ${stats.errors.length}`, stats.errors.length > 0 ? 'error' : 'success');
  
  if (stats.errors.length > 0) {
    console.log('\nErrors:');
    for (const error of stats.errors) {
      log(`  ❌ ${error}`, 'error');
    }
  }
  
  if (stats.movedDirs.length > 0 && VERBOSE) {
    console.log('\nMoved directories:');
    for (const move of stats.movedDirs) {
      const from = relative(FRONTEND_DIR, move.from);
      const to = relative(FRONTEND_DIR, move.to);
      log(`  ${from} → ${to}`, 'info');
    }
  }
  
  if (stats.movedFiles.length > 0 && VERBOSE) {
    console.log('\nMoved files (showing first 20):');
    for (const move of stats.movedFiles.slice(0, 20)) {
      const from = relative(FRONTEND_DIR, move.from);
      const to = relative(FRONTEND_DIR, move.to);
      log(`  ${from} → ${to}`, 'info');
    }
    if (stats.movedFiles.length > 20) {
      log(`  ... and ${stats.movedFiles.length - 20} more files`, 'info');
    }
  }
  
  console.log('\n' + '='.repeat(80));
  console.log();
}

function printNextSteps() {
  console.log('📋 NEXT STEPS');
  console.log('='.repeat(80));
  console.log();
  console.log('1. Review the migration summary above');
  console.log('2. If this was a dry run, run again without --dry-run to execute');
  console.log('3. Update import paths in all files using the fix-imports script');
  console.log('4. Update svelte.config.js and vite.config.ts aliases');
  console.log('5. Update tsconfig.json paths if needed');
  console.log('6. Run the analyze-dag script to verify dependency hierarchy');
  console.log('7. Test the application thoroughly');
  console.log();
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('🚀 Turborepo Migration Script');
  console.log('='.repeat(80));
  console.log();
  log(`Frontend directory: ${FRONTEND_DIR}`, 'info');
  log(`Dry run: ${DRY_RUN}`, DRY_RUN ? 'warning' : 'info');
  log(`Verbose: ${VERBOSE}`, 'info');
  console.log();
  
  if (DRY_RUN) {
    log('⚠️  DRY RUN MODE - No files will be moved', 'warning');
    console.log();
  }
  
  // Create package structure
  createPackageStructure();
  
  // Execute migration rules
  for (const rule of MIGRATION_RULES) {
    executeMigrationRule(rule);
  }
  
  // Print summary
  printSummary();
  
  // Print next steps
  printNextSteps();
  
  // Exit with appropriate code
  process.exit(stats.errors.length > 0 ? 1 : 0);
}

main();
