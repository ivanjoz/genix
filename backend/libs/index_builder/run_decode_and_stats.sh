#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  run_decode_and_stats.sh [INPUT_PATH] [SAMPLE_COUNT] [SEED]

Defaults:
  INPUT_PATH=libs/index_builder/productos.idx
  SAMPLE_COUNT=10
  SEED=0

Notes:
  - SEED=0 means random seed from current time.
  - Set GOCACHE to override build cache path (default: /tmp/go-build).
EOF
}

log_info() {
  printf '[decode:%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

is_non_negative_integer() {
  [[ "$1" =~ ^[0-9]+$ ]]
}

INPUT_PATH="${1:-libs/index_builder/productos.idx}"
SAMPLE_COUNT="${2:-10}"
SEED="${3:-0}"

if [[ "${INPUT_PATH}" == "-h" || "${INPUT_PATH}" == "--help" ]]; then
  usage
  exit 0
fi

if ! is_non_negative_integer "${SAMPLE_COUNT}"; then
  printf 'error: SAMPLE_COUNT must be a non-negative integer, got: %s\n' "${SAMPLE_COUNT}" >&2
  usage >&2
  exit 1
fi
if ! is_non_negative_integer "${SEED}"; then
  printf 'error: SEED must be a non-negative integer, got: %s\n' "${SEED}" >&2
  usage >&2
  exit 1
fi

cd "${BACKEND_DIR}"
if [[ ! -f "${INPUT_PATH}" ]]; then
  printf 'error: input file not found: %s\n' "${INPUT_PATH}" >&2
  exit 1
fi

GOCACHE_PATH="${GOCACHE:-/tmp/go-build}"
log_info "backend_dir=${BACKEND_DIR}"
log_info "input=${INPUT_PATH} sample_count=${SAMPLE_COUNT} seed=${SEED} gocache=${GOCACHE_PATH}"

GOCACHE="${GOCACHE_PATH}" go run ./cmd/index_builder_decode_idx \
  -input "${INPUT_PATH}" \
  -sample "${SAMPLE_COUNT}" \
  -seed "${SEED}"
