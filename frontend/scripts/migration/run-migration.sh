#!/bin/bash
# ============================================
# Master Migration Script
# ============================================
# This script orchestrates the complete migration to turborepo packages
#
# USAGE:
#   ./scripts/migration/run-migration.sh [--dry-run]
#
# STEP 1: Migrate files to pkg-* structure
# STEP 2: Update config aliases
# STEP 3: Fix import paths
# STEP 4: Verify with DAG analysis
# ============================================

set -e  # Exit on error

FRONTEND_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$FRONTEND_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

DRY_RUN=""
if [ "$1" == "--dry-run" ]; then
  DRY_RUN="--dry-run"
  echo -e "${YELLOW}======================================"
  echo -e "DRY RUN MODE - No changes will be made"
  echo -e "======================================${NC}"
  echo ""
fi

log_info() {
  echo -e "${BLUE}ℹ $1${NC}"
}

log_success() {
  echo -e "${GREEN}✓ $1${NC}"
}

log_warning() {
  echo -e "${YELLOW}⚠ $1${NC}"
}

log_error() {
  echo -e "${RED}✗ $1${NC}"
}

log_step() {
  echo ""
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
}

# ============================================
# STEP 1: Migrate Files
# ============================================
log_step "STEP 1: Migrating files to pkg-* structure"

if [ -z "$DRY_RUN" ]; then
  log_warning "This will MOVE files from old structure to pkg-* folders"
  log_warning "Make sure you have committed your changes!"
  read -p "Continue? (y/N) " -n 1 -r
  echo ""
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_error "Migration cancelled"
    exit 1
  fi
fi

log_info "Running file migration..."
bun scripts/migration/migrate-to-turbo.ts $DRY_RUN

if [ $? -ne 0 ]; then
  log_error "File migration failed!"
  exit 1
fi

if [ -z "$DRY_RUN" ]; then
  log_success "Files migrated successfully"
else
  log_info "Dry run completed - files would be moved"
fi

# ============================================
# STEP 2: Update Config Aliases
# ============================================
log_step "STEP 2: Updating config aliases"

log_info "Updating svelte.config.js, vite.config.ts, and tsconfig.json..."
bun scripts/migration/update-config-aliases.ts $DRY_RUN

if [ $? -ne 0 ]; then
  log_error "Config update failed!"
  exit 1
fi

if [ -z "$DRY_RUN" ]; then
  log_success "Config aliases updated"
else
  log_info "Dry run completed - configs would be updated"
fi

# ============================================
# STEP 3: Fix Import Paths
# ============================================
log_step "STEP 3: Fixing import paths"

log_info "Analyzing and fixing imports..."
bun scripts/migration/fix-imports-intelligent.ts $DRY_RUN --verbose

if [ $? -ne 0 ]; then
  log_error "Import fixing encountered issues!"
  log_warning "Check the output above for duplicate symbols or unresolved imports"
  log_warning "You may need to manually fix these issues"
  # Don't exit here, let the user see the summary
fi

if [ -z "$DRY_RUN" ]; then
  log_success "Import paths fixed"
else
  log_info "Dry run completed - imports would be fixed"
fi

# ============================================
# STEP 4: Verify with DAG Analysis
# ============================================
log_step "STEP 4: Verifying dependency hierarchy"

log_info "Running DAG analysis..."
bun scripts/analyze-dag.ts

if [ $? -ne 0 ]; then
  log_warning "DAG analysis found violations!"
  log_warning "Check the output above for hierarchy issues"
  log_warning "You may need to refactor code to fix violations"
else
  log_success "DAG hierarchy is valid!"
fi

# ============================================
# Summary
# ============================================
log_step "MIGRATION COMPLETE"

if [ -z "$DRY_RUN" ]; then
  log_success "All migration steps completed!"
  echo ""
  echo "Next steps:"
  echo "  1. Review the warnings and errors above"
  echo "  2. Fix any duplicate symbols manually"
  echo "  3. Fix any unresolved imports manually"
  echo "  4. Fix any DAG violations by refactoring"
  echo "  5. Test the application thoroughly"
  echo "  6. Commit your changes"
else
  log_info "Dry run completed!"
  echo ""
  echo "Next steps:"
  echo "  1. Review all the changes above"
  echo "  2. If everything looks good, run again without --dry-run"
  echo "  3. Then follow the steps above"
fi

echo ""
log_info "Migration script finished"
