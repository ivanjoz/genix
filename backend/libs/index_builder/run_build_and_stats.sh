#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  run_build_and_stats.sh [EMPRESA_ID]

Defaults:
  EMPRESA_ID=1

Notes:
  - Builds the combined productos index (text + taxonomy) from database tables.
  - Output path is fixed by handler: libs/index_builder/productos.idx
  - Set GOCACHE to override build cache path (default: /tmp/go-build).
EOF
}

log_info() {
  printf '[build:%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

is_positive_integer() {
  [[ "$1" =~ ^[1-9][0-9]*$ ]]
}

EMPRESA_ID="${1:-1}"

if [[ "${EMPRESA_ID}" == "-h" || "${EMPRESA_ID}" == "--help" ]]; then
  usage
  exit 0
fi

if ! is_positive_integer "${EMPRESA_ID}"; then
  printf 'error: EMPRESA_ID must be a positive integer, got: %s\n' "${EMPRESA_ID}" >&2
  usage >&2
  exit 1
fi

cd "${BACKEND_DIR}"

GOCACHE_PATH="${GOCACHE:-/tmp/go-build}"
log_info "backend_dir=${BACKEND_DIR}"
log_info "empresa_id=${EMPRESA_ID} gocache=${GOCACHE_PATH}"

GOCACHE="${GOCACHE_PATH}" go run ./cmd/index_builder_build_from_db \
  -empresa-id "${EMPRESA_ID}"
