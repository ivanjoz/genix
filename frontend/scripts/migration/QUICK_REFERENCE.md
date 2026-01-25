# Quick Reference: Turborepo Migration

## 🚀 One-Command Migration

```bash
# Dry run (DO THIS FIRST!)
./scripts/migration/run-migration.sh --dry-run

# Execute migration
./scripts/migration/run-migration.sh
```

## 📋 Individual Scripts

| Script | Purpose | Command |
|--------|---------|---------|
| `migrate-to-turbo.ts` | Move files to pkg-* | `bun scripts/migration/migrate-to-turbo.ts --dry-run` |
| `update-config-aliases.ts` | Update alias paths | `bun scripts/migration/update-config-aliases.ts --dry-run` |
| `fix-imports-intelligent.ts` | Fix broken imports | `bun scripts/migration/fix-imports-intelligent.ts --dry-run` |
| `run-migration.sh` | Run all steps | `./scripts/migration/run-migration.sh --dry-run` |

## 🔍 Verification

```bash
# Check DAG hierarchy
bun scripts/analyze-dag.ts

# Check imports are valid
bun scripts/check-imports.ts

# TypeScript check
bunx tsc --noEmit
```

## ⚠️ Common Issues

### Duplicate Symbols
**What:** Same symbol found in multiple packages  
**Why:** Files were copied instead of moved  
**Fix:** Delete unwanted duplicates or rename them

### Unresolved Imports
**What:** Import cannot be resolved  
**Why:** File missing or not exported  
**Fix:** Check file exists and is properly exported

### Hierarchy Violations
**What:** Package depends on wrong level  
**Why:** Cross-level dependency  
**Fix:** Move code to base package or refactor

## 📊 Migration Summary

```
BEFORE → AFTER
────────────────────────────────────
core/          → pkg-core/core/
types/         → pkg-core/types/
lib/           → pkg-core/lib/
services/      → pkg-services/services/
shared/        → pkg-services/shared/
components/    → pkg-ui/components/
assets/        → pkg-ui/assets/
ecommerce/     → pkg-components/ecommerce/
ecommerce-components/ → pkg-components/ecommerce-components/
routes/        → pkg-app/routes/
stores/        → pkg-store/stores/
workers/       → pkg-app/workers/
functions/     → pkg-app/functions/
static/        → pkg-app/static/
genix/         → pkg-app/genix/
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

## 💡 Tips

1. **ALWAYS dry-run first** - `--dry-run` flag
2. **Commit before migrating** - easy rollback
3. **Review warnings** - they indicate real issues
4. **Fix incrementally** - one issue at a time
5. **Test thoroughly** - each step before next

## 🆘 Quick Rollback

```bash
# Reset to pre-migration commit
git reset --hard HEAD
```

---

**Need more details?** See [README.md](./README.md)
