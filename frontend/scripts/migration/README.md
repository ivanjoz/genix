# Turborepo Migration Guide

This guide explains how to migrate the frontend monorepo to a package-based structure using turborepo.

## 📋 Overview

The migration transforms the current flat structure into a hierarchical package structure that follows the expected DAG (Directed Acyclic Graph) dependency pattern:

```
pkg-core (Level 0) - Base utilities, no dependencies
    ↓
pkg-services (Level 1) - Can depend on pkg-core
    ↓
pkg-ui (Level 2) - Can depend on pkg-core, pkg-services
pkg-components (Level 2) - Can depend on pkg-core, pkg-services
    ↓
pkg-store (Level 3) - Can depend on pkg-core, pkg-services, pkg-ui, pkg-components
pkg-app (Level 3) - Can depend on pkg-core, pkg-services, pkg-ui, pkg-components
```

## 🗂️ Migration Mapping

### Source → Target Package

| Source Directory | Target Package | Target Directory |
|----------------|----------------|------------------|
| `core/` | pkg-core | core/ |
| `types/` | pkg-core | types/ |
| `lib/` | pkg-core | lib/ |
| `services/` | pkg-services | services/ |
| `shared/` | pkg-services | shared/ |
| `components/` | pkg-ui | components/ |
| `assets/` | pkg-ui | assets/ |
| `ecommerce/` | pkg-components | ecommerce/ |
| `ecommerce-components/` | pkg-components | ecommerce-components/ |
| `routes/` | pkg-app | routes/ |
| `stores/` | pkg-store | stores/ |
| `workers/` | pkg-app | workers/ |
| `functions/` | pkg-app | functions/ |
| `static/` | pkg-app | static/ |
| `genix/` | pkg-app | genix/ |

## 🚀 Quick Start

### Option 1: Run All Steps Automatically

```bash
# Dry run first (recommended)
./scripts/migration/run-migration.sh --dry-run

# If everything looks good, run for real
./scripts/migration/run-migration.sh
```

### Option 2: Run Steps Manually

#### Step 1: Migrate Files

```bash
# Dry run
bun scripts/migration/migrate-to-turbo.ts --dry-run --verbose

# Execute
bun scripts/migration/migrate-to-turbo.ts
```

#### Step 2: Update Config Aliases

```bash
# Dry run
bun scripts/migration/update-config-aliases.ts --dry-run

# Execute
bun scripts/migration/update-config-aliases.ts
```

#### Step 3: Fix Import Paths

```bash
# Dry run first to see what will be fixed
bun scripts/migration/fix-imports-intelligent.ts --dry-run --verbose

# Execute fixes
bun scripts/migration/fix-imports-intelligent.ts
```

#### Step 4: Verify

```bash
# Run DAG analysis
bun scripts/analyze-dag.ts

# Check imports
bun scripts/check-imports.ts
```

## 📝 Script Details

### 1. `migrate-to-turbo.ts`

Moves files from old structure to new pkg-* structure.

**Features:**
- Moves entire directories when possible
- Falls back to file-by-file migration if needed
- Skips build artifacts and dependencies
- Provides detailed statistics
- Dry-run mode for safe testing

**Usage:**
```bash
bun scripts/migration/migrate-to-turbo.ts [--dry-run] [--verbose]
```

### 2. `fix-imports-intelligent.ts`

Smart import fixer that searches for symbols across all packages.

**Strategy:**
1. Builds a comprehensive symbol index by scanning all exports
2. Parses all imports in all files
3. For each broken import:
   - Extracts symbol names
   - Searches the symbol index
   - If found in exactly one location: fixes the import
   - If found in multiple locations: warns user (copy instead of move)
4. Updates import paths to use correct package aliases

**Prioritizes correctness over performance** (as requested)

**Usage:**
```bash
bun scripts/migration/fix-imports-intelligent.ts [--dry-run] [--verbose]
```

**Output:**
- List of fixed imports
- Warnings for duplicate symbols (need manual resolution)
- List of unresolved imports
- Detailed statistics

### 3. `update-config-aliases.ts`

Updates path aliases in configuration files.

**Updates:**
- `svelte.config.js` - alias section
- `vite.config.ts` - resolve.alias section
- `tsconfig.json` - compilerOptions.paths section

**New Alias Mappings:**

| Alias | New Path |
|-------|----------|
| `$core` | `./pkg-core` |
| `$lib` | `./pkg-app/lib` |
| `$components` | `./pkg-ui/components` |
| `$services` | `./pkg-services/services` |
| `$shared` | `./pkg-services/shared` |
| `$ecommerce` | `./pkg-components/ecommerce` |
| `$http` | `./pkg-core/lib/http.ts` |

**Usage:**
```bash
bun scripts/migration/update-config-aliases.ts [--dry-run]
```

### 4. `run-migration.sh`

Master script that orchestrates all migration steps.

**Usage:**
```bash
./scripts/migration/run-migration.sh [--dry-run]
```

## ⚠️ Important Notes

### Before Migration

1. **Commit your changes!** The migration will move files (not copy)
2. Make sure you're on a clean git state
3. Run dry-run first to see what will happen

### During Migration

1. The migration script moves files (not copies)
2. Build artifacts are skipped automatically
3. Configuration files are updated automatically
4. Imports are fixed intelligently by searching symbols

### After Migration

#### Check for Duplicate Symbols

The intelligent import fixer will warn you if it finds symbols in multiple locations. This usually means files were **copied instead of moved**. You need to:

1. Review the duplicate warnings
2. Decide which copy to keep
3. Delete or rename the others

**Example warning:**
```
⚠️  Duplicate symbol 'ImageUploader' found in multiple locations:
    - pkg-ui: pkg-ui/components/ImageUploader.svelte
    - pkg-core: pkg-core/core/ImageUploader.svelte
```

#### Check for Unresolved Imports

Some imports might not be automatically fixable. You'll need to manually:

1. Check if files are missing
2. Update import paths manually
3. Ensure all exports are properly defined

#### Verify DAG Hierarchy

Run `bun scripts/analyze-dag.ts` to check for hierarchy violations. If found, you may need to:

1. Move shared code to a more base package
2. Refactor to avoid cross-level dependencies
3. Update imports to follow the hierarchy

## 🔍 Troubleshooting

### "Import cannot be resolved"

This means the symbol wasn't found in any package. Check:

1. Is the file exported correctly?
2. Is the symbol spelled correctly?
3. Was the file moved to the right package?

### "Duplicate symbol found"

This indicates the same symbol exists in multiple packages. You need to:

1. Determine which location is correct
2. Delete or rename the duplicates
3. Re-run the import fixer

### "Hierarchy violation detected"

The DAG analysis found a dependency that violates the expected hierarchy. You need to:

1. Identify the violating import
2. Move the code to an appropriate base package
3. Refactor to avoid the violation
4. Re-run the analysis

## 📊 Verification Steps

After migration, verify everything works:

```bash
# 1. Check DAG hierarchy
bun scripts/analyze-dag.ts

# 2. Check all imports are valid
bun scripts/check-imports.ts

# 3. Run TypeScript compiler
bunx tsc --noEmit

# 4. Build the project
bun run build

# 5. Run tests (if you have them)
bun run test
```

## 🎯 Best Practices

1. **Always dry-run first** before executing
2. **Commit before migrating** to easily rollback
3. **Review warnings carefully** - they often indicate real issues
4. **Test thoroughly** after each major step
5. **Fix issues incrementally** - don't try to fix everything at once

## 📞 Support

If you encounter issues:

1. Check the script output for detailed error messages
2. Review the DAG analysis for dependency issues
3. Look at the duplicate warnings for naming conflicts
4. Manually inspect problematic files if automation fails

## 🔄 Rollback

If something goes wrong and you need to rollback:

```bash
# Reset to your pre-migration commit
git reset --hard HEAD

# Or if you committed the migration
git reset --hard <commit-before-migration>
```

## 📚 Additional Resources

- [analyze-dag.ts](../analyze-dag.ts) - DAG dependency analyzer
- [check-imports.ts](../check-imports.ts) - Import validator
- [fix-import-paths.ts](../fix-import-paths.ts) - Original import fixer (before intelligent version)

---

**Remember:** The migration is designed to be safe and reversible. Take your time, review changes, and test thoroughly!
