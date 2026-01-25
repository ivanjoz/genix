# Migration Plan & Analysis

## ✅ What Has Been Created

I've created a complete migration solution with the following components:

### 📁 Scripts Created (in `scripts/migration/`)

1. **`migrate-to-turbo.ts`** (12K)
   - Moves files from old structure to pkg-* packages
   - Moves entire directories when possible
   - Falls back to file-by-file migration
   - Comprehensive statistics and logging
   - Dry-run and verbose modes

2. **`fix-imports-intelligent.ts`** (26K) 
   - **Key feature:** Searches symbols across ALL packages
   - Builds comprehensive symbol index
   - Detects duplicate symbols (copy vs move)
   - Fixes broken imports intelligently
   - Reports unresolved imports
   - Prioritizes correctness over performance

3. **`update-config-aliases.ts`** (9.6K)
   - Updates svelte.config.js aliases
   - Updates vite.config.ts aliases  
   - Updates tsconfig.json paths
   - Dry-run mode

4. **`run-migration.sh`** (4.6K)
   - Master orchestration script
   - Runs all steps in order
   - Interactive confirmation
   - Comprehensive logging

5. **Documentation**
   - `README.md` (7.9K) - Complete migration guide
   - `QUICK_REFERENCE.md` (2.8K) - Quick reference card

## 🎯 Migration Strategy

### Phase 1: File Migration
- Move directories to appropriate packages
- Preserve file structure within packages
- Skip build artifacts and dependencies

### Phase 2: Configuration Update
- Update path aliases in all config files
- Point aliases to new package locations
- Maintain backward compatibility where possible

### Phase 3: Import Fixing (Intelligent)
- Build symbol index across all packages
- For each import:
  - Check if path is valid
  - If not, search for symbol across packages
  - If found in ONE location → fix it
  - If found in MULTIPLE locations → warn user
  - If NOT found → report as unresolved

### Phase 4: Verification
- Run DAG analysis to check hierarchy
- Validate all imports
- Identify any remaining issues

## 📊 Package Mapping

```
SOURCE              → TARGET PACKAGE/PATH
────────────────────────────────────────────
core/               → pkg-core/core/
types/              → pkg-core/types/
lib/                → pkg-core/lib/
services/           → pkg-services/services/
shared/             → pkg-services/shared/
components/         → pkg-ui/components/
assets/             → pkg-ui/assets/
ecommerce/          → pkg-components/ecommerce/
ecommerce-components/ → pkg-components/ecommerce-components/
routes/             → pkg-app/routes/
stores/             → pkg-store/stores/
workers/            → pkg-app/workers/
functions/          → pkg-app/functions/
static/             → pkg-app/static/
genix/              → pkg-app/genix/
```

## 🔑 New Alias Mappings

```
$core          → ./pkg-core
$lib           → ./pkg-app/lib  
$components    → ./pkg-ui/components
$services      → ./pkg-services/services
$shared        → ./pkg-services/shared
$ecommerce     → ./pkg-components/ecommerce
$http          → ./pkg-core/lib/http.ts
```

## 🚀 How to Execute

### Option 1: Automated (Recommended)

```bash
# Step 1: Dry run to see what will happen
./scripts/migration/run-migration.sh --dry-run

# Step 2: Review the output carefully
# - Check duplicate warnings
# - Check unresolved imports
# - Verify mapping looks correct

# Step 3: Execute for real
./scripts/migration/run-migration.sh

# Step 4: Fix any issues reported
# - Delete duplicate files manually
# - Fix unresolved imports manually
# - Refactor to fix DAG violations

# Step 5: Verify
bun scripts/analyze-dag.ts
bun scripts/check-imports.ts
```

### Option 2: Manual Step-by-Step

```bash
# Step 1: Move files (dry run first)
bun scripts/migration/migrate-to-turbo.ts --dry-run --verbose
bun scripts/migration/migrate-to-turbo.ts

# Step 2: Update configs
bun scripts/migration/update-config-aliases.ts --dry-run
bun scripts/migration/update-config-aliases.ts

# Step 3: Fix imports (dry run first)
bun scripts/migration/fix-imports-intelligent.ts --dry-run --verbose
bun scripts/migration/fix-imports-intelligent.ts

# Step 4: Verify
bun scripts/analyze-dag.ts
```

## ⚠️ Critical Points to Understand

### 1. The Import Fixer is Intelligent

The `fix-imports-intelligent.ts` script doesn't just replace paths - it:

- **Scans ALL files** to build a symbol index
- **Searches for each symbol** across all packages
- **Detects duplicates** (same symbol in multiple locations)
- **Only fixes** when it finds exactly one match
- **Reports conflicts** for manual resolution

### 2. Duplicate Symbols = Copy Instead of Move

If you see warnings like:
```
⚠️  Duplicate symbol 'ImageUploader' found in multiple locations:
    - pkg-ui: pkg-ui/components/ImageUploader.svelte
    - pkg-core: pkg-core/core/ImageUploader.svelte
```

This means the same file exists in both places. You need to:
1. Decide which location is correct
2. Delete the other copy
3. Re-run the import fixer

### 3. Unresolved Imports Need Manual Attention

Some imports might not be auto-fixable because:
- Symbol doesn't exist (file missing?)
- Symbol is not exported
- Typo in import name

You'll need to fix these manually.

### 4. DAG Hierarchy Must Be Followed

The expected hierarchy:
```
pkg-core (Level 0)
  ↓
pkg-services (Level 1)  
  ↓
pkg-ui, pkg-components (Level 2)
  ↓
pkg-store, pkg-app (Level 3)
```

If violations are found, you need to refactor code to follow this hierarchy.

## 🔍 What Happens After Migration

### Files Will Be MOVED (Not Copied)

The migration moves files from old locations to new package locations. Old directories will be removed (except if they're not empty).

### Config Files Will Be Updated

All alias paths will be updated to point to new package locations.

### Imports Will Be Fixed

Broken imports will be fixed by:
- Searching for symbols across all packages
- Updating paths to use correct aliases
- Converting relative imports between packages to alias imports

### You May Need to:

1. **Delete duplicate files** (if warned)
2. **Fix unresolved imports manually** (if reported)
3. **Refactor to fix DAG violations** (if found)
4. **Test the application thoroughly**

## 📈 Success Criteria

Migration is successful when:

✅ All files moved to correct packages  
✅ All config aliases updated  
✅ All imports resolve correctly  
✅ No duplicate symbols  
✅ DAG hierarchy is valid  
✅ Application builds and runs  

## 🔄 Rollback Strategy

If anything goes wrong:

```bash
# Reset to pre-migration state
git reset --hard HEAD

# Or if you committed
git reset --hard <commit-before-migration>
```

**IMPORTANT:** Commit your changes before migrating!

## 📚 Additional Resources

- `scripts/migration/README.md` - Detailed migration guide
- `scripts/migration/QUICK_REFERENCE.md` - Quick reference card
- `scripts/analyze-dag.ts` - DAG dependency analyzer
- `scripts/check-imports.ts` - Import validator

## 💡 Tips for Success

1. **Start with dry-run** - See what will happen before executing
2. **Commit before migrating** - Easy rollback if needed
3. **Review warnings carefully** - They indicate real issues
4. **Fix issues incrementally** - Don't try to fix everything at once
5. **Test each step** - Verify before proceeding to next step

---

## 🎯 Next Steps

1. **Review this plan** - Make sure you understand the strategy
2. **Read the README** - `scripts/migration/README.md` for detailed instructions
3. **Commit your changes** - Ensure you can rollback if needed
4. **Run dry-run** - `./scripts/migration/run-migration.sh --dry-run`
5. **Review output** - Check for duplicates, unresolved imports, violations
6. **Execute migration** - Remove `--dry-run` flag
7. **Fix issues** - Address any warnings or errors
8. **Verify** - Run analysis and test application

---

**Ready to migrate?** Run: `./scripts/migration/run-migration.sh --dry-run`
