#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  cat <<'USAGE'
Usage:
  run_build_and_stats_local.sh [EMPRESA_ID]

Defaults:
  EMPRESA_ID=1

Notes:
  - This script fetches products/brands/categories from DB.
  - It runs no-persist build path (no S3 upload, no app cache persistence).
  - It writes the generated index to libs/index_builder/productos.idx.
  - It uses a temporary Go build cache directory and removes it on exit.
USAGE
}

log_info() {
  printf '[build-local:%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
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
OUTPUT_IDX_PATH="libs/index_builder/productos.idx"

TEMP_GOCACHE_PATH="$(mktemp -d /tmp/go-build-local.XXXXXX)"
cleanup() {
  rm -rf "${TEMP_GOCACHE_PATH}"
}
trap cleanup EXIT

log_info "backend_dir=${BACKEND_DIR}"
log_info "empresa_id=${EMPRESA_ID}"
log_info "output_idx=${OUTPUT_IDX_PATH}"
log_info "temporary_gocache=${TEMP_GOCACHE_PATH}"

GOCACHE="${TEMP_GOCACHE_PATH}" go run ./cmd/index_builder_build_from_db_no_persist \
  -empresa-id "${EMPRESA_ID}" \
  -output "${OUTPUT_IDX_PATH}"

if [[ ! -f "${OUTPUT_IDX_PATH}" ]]; then
  printf 'error: expected output index file was not generated: %s\n' "${OUTPUT_IDX_PATH}" >&2
  exit 1
fi
log_info "output_idx_bytes=$(wc -c < "${OUTPUT_IDX_PATH}")"

log_info "done (db no-persist build)"
