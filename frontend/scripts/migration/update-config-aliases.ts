#!/usr/bin/env bun
/**
 * Config Alias Updater
 *
 * Updates the path aliases in svelte.config.js and vite.config.ts
 * to point to the new package structure.
 *
 * USAGE:
 *   bun scripts/migration/update-config-aliases.ts [--dry-run]
 */

import { readFileSync, writeFileSync } from 'fs';
import { resolve } from 'path';

// ============================================
// Configuration
// ============================================

const FRONTEND_DIR = resolve(process.cwd());
const DRY_RUN = process.argv.includes('--dry-run');

// Old aliases -> New aliases mapping
const ALIAS_MAPPINGS = {
  '$core': '$core',
  '$lib': '$lib',
  '$components': '$components',
  '$services': '$services',
  '$shared': '$services',
  '$ecommerce': '$ecommerce',
  '$ecommerceComponents': '$ecommerce',
  '$http': '$core',
};

// New paths for aliases
const ALIAS_PATHS = {
  '$core': './pkg-core',
  '$lib': './pkg-app/lib',
  '$components': './pkg-ui/components',
  '$services': './pkg-services/services',
  '$shared': './pkg-services/shared',
  '$ecommerce': './pkg-components/ecommerce',
  '$ecommerceComponents': './pkg-ui/ecommerce-components',
  '$http': './pkg-core/lib/http.ts',
};

// ============================================
// Types
// ============================================

interface ConfigUpdate {
  file: string;
  oldAlias: string;
  newAlias: string;
  oldPath: string;
  newPath: string;
}

// ============================================
// State
// ============================================

const updates: ConfigUpdate[] = [];

// ============================================
// Utilities
// ============================================

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
// Svelte Config Update
// ============================================

function updateSvelteConfig() {
  const configPath = resolve(FRONTEND_DIR, 'svelte.config.js');
  
  if (!require('fs').existsSync(configPath)) {
    log('⚠️  svelte.config.js not found', 'warning');
    return;
  }

  log('\n📝 Updating svelte.config.js...', 'info');

  let content = readFileSync(configPath, 'utf-8');
  
  // Find the alias section
  const aliasMatch = content.match(/alias:\s*\{([^}]+)\}/);
  
  if (!aliasMatch) {
    log('   No alias section found in svelte.config.js', 'warning');
    return;
  }

  const aliasSection = aliasMatch[0];
  let newAliasSection = aliasSection;

  // Update each alias
  for (const [alias, newPath] of Object.entries(ALIAS_PATHS)) {
    const oldPathRegex = new RegExp(`'\\${alias}'\\s*:\\s*path\\.resolve\\([^)]+\\)`, 'g');
    const matches = aliasSection.match(oldPathRegex);
    
    if (matches) {
      for (const match of matches) {
        const oldPathMatch = match.match(/path\.resolve\('([^']+)'\)/);
        if (oldPathMatch) {
          const oldPath = oldPathMatch[1];
          
          if (oldPath !== newPath) {
            updates.push({
              file: 'svelte.config.js',
              oldAlias: alias,
              newAlias: alias,
              oldPath,
              newPath
            });

            const newLine = `'${alias}': path.resolve('${newPath}')`;
            newAliasSection = newAliasSection.replace(match, newLine);
          }
        }
      }
    }
  }

  if (newAliasSection !== aliasSection) {
    content = content.replace(aliasSection, newAliasSection);
    
    if (!DRY_RUN) {
      writeFileSync(configPath, content, 'utf-8');
      log('   ✅ Updated svelte.config.js', 'success');
    } else {
      log('   [DRY RUN] Would update svelte.config.js', 'info');
    }
  } else {
    log('   No changes needed for svelte.config.js', 'info');
  }
}

// ============================================
// Vite Config Update
// ============================================

function updateViteConfig() {
  const configPath = resolve(FRONTEND_DIR, 'vite.config.ts');
  
  if (!require('fs').existsSync(configPath)) {
    log('⚠️  vite.config.ts not found', 'warning');
    return;
  }

  log('\n📝 Updating vite.config.ts...', 'info');

  let content = readFileSync(configPath, 'utf-8');
  
  // Find alias definitions in resolve.alias block
  const aliasBlockMatch = content.match(/alias:\s*\{([^}]+)\}/s);
  
  if (!aliasBlockMatch) {
    log('   No alias section found in vite.config.ts', 'warning');
    return;
  }

  const aliasBlock = aliasBlockMatch[0];
  let newAliasBlock = aliasBlock;

  // Update each alias
  for (const [alias, newPath] of Object.entries(ALIAS_PATHS)) {
    // Look for alias definitions in various formats
    const patterns = [
      new RegExp(`'\\${alias}'\\s*:\\s*['"]([^'"]+)['"]`, 'g'),
      new RegExp(`"\\${alias}"\\s*:\\s*['"]([^'"]+)['"]`, 'g'),
    ];

    for (const pattern of patterns) {
      const matches = aliasBlock.match(pattern);
      
      if (matches) {
        for (const match of matches) {
          const pathMatch = match.match(/:\s*['"]([^'"]+)['"]/);
          if (pathMatch) {
            const oldPath = pathMatch[1];
            
            if (oldPath !== newPath) {
              updates.push({
                file: 'vite.config.ts',
                oldAlias: alias,
                newAlias: alias,
                oldPath,
                newPath
              });

              const newLine = `'${alias}': '${newPath}'`;
              newAliasBlock = newAliasBlock.replace(match, newLine);
            }
          }
        }
      }
    }
  }

  if (newAliasBlock !== aliasBlock) {
    content = content.replace(aliasBlock, newAliasBlock);
    
    if (!DRY_RUN) {
      writeFileSync(configPath, content, 'utf-8');
      log('   ✅ Updated vite.config.ts', 'success');
    } else {
      log('   [DRY RUN] Would update vite.config.ts', 'info');
    }
  } else {
    log('   No changes needed for vite.config.ts', 'info');
  }
}

// ============================================
// TSConfig Update
// ============================================

function updateTSConfig() {
  const configPath = resolve(FRONTEND_DIR, 'tsconfig.json');
  
  if (!require('fs').existsSync(configPath)) {
    log('⚠️  tsconfig.json not found', 'warning');
    return;
  }

  log('\n📝 Updating tsconfig.json...', 'info');

  let content = readFileSync(configPath, 'utf-8');
  
  // Strip comments (tsconfig often has // comments)
  content = content.replace(/\/\/.*$/gm, '');
  content = content.replace(/\/\*[\s\S]*?\*\//g, '');
  
  const json = JSON.parse(content);

  // Ensure paths section exists
  if (!json.compilerOptions) {
    json.compilerOptions = {};
  }
  if (!json.compilerOptions.paths) {
    json.compilerOptions.paths = {};
  }

  let hasChanges = false;

  // Update each path alias
  for (const [alias, newPath] of Object.entries(ALIAS_PATHS)) {
    const aliasPattern = alias + '/*';
    const pathPattern = newPath + '/*';
    
    if (json.compilerOptions.paths[aliasPattern]) {
      const oldPath = json.compilerOptions.paths[aliasPattern][0];
      
      if (oldPath !== pathPattern) {
        updates.push({
          file: 'tsconfig.json',
          oldAlias: aliasPattern,
          newAlias: aliasPattern,
          oldPath,
          newPath: pathPattern
        });
        
        json.compilerOptions.paths[aliasPattern] = [pathPattern];
        hasChanges = true;
      }
    }
  }

  if (hasChanges) {
    const newContent = JSON.stringify(json, null, 2);
    
    if (!DRY_RUN) {
      writeFileSync(configPath, newContent, 'utf-8');
      log('   ✅ Updated tsconfig.json', 'success');
    } else {
      log('   [DRY RUN] Would update tsconfig.json', 'info');
    }
  } else {
    log('   No changes needed for tsconfig.json', 'info');
  }
}

// ============================================
// Main Execution
// ============================================

function main() {
  console.log('🔧 Config Alias Updater');
  console.log('='.repeat(80));
  console.log();
  log(`Frontend directory: ${FRONTEND_DIR}`, 'info');
  log(`Dry run: ${DRY_RUN}`, DRY_RUN ? 'warning' : 'info');
  console.log();

  if (DRY_RUN) {
    log('⚠️  DRY RUN MODE - No files will be modified', 'warning');
    console.log();
  }

  // Update configs
  updateSvelteConfig();
  updateViteConfig();
  updateTSConfig();

  // Print summary
  console.log('\n' + '='.repeat(80));
  console.log('SUMMARY');
  console.log('='.repeat(80));
  console.log();
  
  log(`Total updates: ${updates.length}`, updates.length > 0 ? 'success' : 'info');
  console.log();

  if (updates.length > 0) {
    console.log('Updates:');
    console.log('─'.repeat(80));
    
    for (const update of updates) {
      console.log(`\n📄 ${update.file}`);
      console.log(`  ${update.oldAlias}: ${update.oldPath}`);
      console.log(`  → ${update.newAlias}: ${update.newPath}`);
    }
    
    console.log('\n' + '─'.repeat(80));
  }

  console.log('\n📋 NEW ALIAS MAPPINGS');
  console.log('─'.repeat(80));
  console.log();
  
  for (const [alias, path] of Object.entries(ALIAS_PATHS)) {
    console.log(`  ${alias} → ${path}`);
  }
  
  console.log('\n' + '─'.repeat(80));
  console.log();

  if (!DRY_RUN) {
    console.log('✅ Alias updates complete!\n');
    console.log('Next steps:');
    console.log('  1. Run: bun scripts/migration/fix-imports-intelligent.ts --dry-run');
    console.log('  2. Review the changes');
    console.log('  3. Run: bun scripts/migration/fix-imports-intelligent.ts');
    console.log('  4. Run: bun scripts/analyze-dag.ts');
    console.log('  5. Test the application');
  } else {
    console.log('Next steps:');
    console.log('  1. Review the changes above');
    console.log('  2. Run again without --dry-run to apply');
  }
  
  console.log('\n' + '='.repeat(80));
}

main();
