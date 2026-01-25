#!/usr/bin/env bun
/**
 * Fix Remaining Unresolved Imports
 * 
 * This script manually fixes the remaining 5 unresolved imports
 * that the intelligent fixer couldn't resolve.
 */

import { readFileSync, writeFileSync, readdirSync, statSync } from 'fs';
import { resolve, relative, dirname, join } from 'path';

const FRONTEND_DIR = resolve(process.cwd());

// Manual fixes for remaining unresolved imports
const MANUAL_FIXES = [
  {
    file: 'pkg-services/services/admin/login.ts',
    search: "from '../../shared/main'",
    replace: "from '$core/helpers'",
    note: "Removed shared/main, functions are in $core/helpers.ts"
  },
  {
    file: 'pkg-services/services/admin/login.ts',
    search: 'makeRamdomString',
    replace: 'generateRandomString',
    note: "Renamed to match actual function name"
  },
  {
    file: 'pkg-main/routes/develop-ui/test-table/+page.svelte',
    search: "from '../../../components/VTable'",
    replace: "from '$components/VTable'",
    note: "Fixed relative import to alias"
  },
  {
    file: 'pkg-main/routes/operaciones/productos/+page.svelte',
    search: "from '../../../components/VTable/vTable.svelte'",
    replace: "from '$components/VTable/vTable.svelte'",
    note: "Fixed relative import to alias"
  },
  {
    file: 'pkg-main/routes/+layout.svelte',
    search: "from '../workers/image-worker?worker'",
    replace: "from '$lib/workers/image-worker?worker'",
    note: "Fixed relative import to alias"
  }
];

let totalFixed = 0;

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

function applyFix(filePath: string, search: string, replace: string, note: string) {
  const fullPath = resolve(FRONTEND_DIR, filePath);
  
  if (!require('fs').existsSync(fullPath)) {
    log(`  ⚠️  File not found: ${filePath}`, 'warning');
    return false;
  }
  
  let content = readFileSync(fullPath, 'utf-8');
  const originalContent = content;
  
  // Replace the import path
  content = content.replace(search, replace);
  
  if (content !== originalContent) {
    writeFileSync(fullPath, content, 'utf-8');
    log(`  ✓ Fixed ${filePath}`, 'success');
    log(`    ${search} → ${replace}`, 'info');
    log(`    Note: ${note}`, 'info');
    return true;
  } else {
    log(`  ⚠️  Pattern not found: ${search}`, 'warning');
    return false;
  }
}

function main() {
  console.log('🔧 Fixing Remaining Unresolved Imports');
  console.log('='.repeat(80));
  console.log();
  
  for (const fix of MANUAL_FIXES) {
    console.log(`\n📝 ${fix.file}`);
    const fixed = applyFix(fix.file, fix.search, fix.replace, fix.note);
    if (fixed) totalFixed++;
  }
  
  console.log('\n' + '='.repeat(80));
  console.log('SUMMARY');
  console.log('='.repeat(80));
  console.log();
  log(`Total fixes applied: ${totalFixed}`, totalFixed > 0 ? 'success' : 'info');
  console.log();
  
  if (totalFixed === MANUAL_FIXES.length) {
    log('✅ All remaining imports have been fixed!', 'success');
    console.log();
    console.log('Next steps:');
    console.log('  1. Run: bun scripts/analyze-dag.ts');
    console.log('  2. Run: bun scripts/check-imports.ts');
    console.log('  3. Test the application');
  } else {
    log('⚠️  Some fixes could not be applied', 'warning');
    console.log();
    console.log('Please review the warnings above and fix manually.');
  }
  
  console.log('\n' + '='.repeat(80));
  
  process.exit(totalFixed === MANUAL_FIXES.length ? 0 : 1);
}

main();
