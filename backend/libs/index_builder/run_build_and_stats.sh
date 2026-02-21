#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  cat <<'EOF'
Usage:
  run_build_and_stats.sh [INPUT_PATH] [OUTPUT_PATH] [MAX_WORDS] [MAX_SYLLABLES_PER_WORD] [SLOTS]

Defaults:
  INPUT_PATH=libs/index_builder/productos.txt
  OUTPUT_PATH=libs/index_builder/productos.idx
  MAX_WORDS=8
  MAX_SYLLABLES_PER_WORD=7
  SLOTS=255

Notes:
  - Set GOCACHE to override build cache path (default: /tmp/go-build).
EOF
}

log_info() {
  printf '[build:%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

is_positive_integer() {
  [[ "$1" =~ ^[1-9][0-9]*$ ]]
}

INPUT_PATH="${1:-libs/index_builder/productos.txt}"
OUTPUT_PATH="${2:-libs/index_builder/productos.idx}"
MAX_WORDS="${3:-8}"
MAX_SYLLABLES_PER_WORD="${4:-7}"
SLOTS="${5:-255}"

if [[ "${INPUT_PATH}" == "-h" || "${INPUT_PATH}" == "--help" ]]; then
  usage
  exit 0
fi

if ! is_positive_integer "${MAX_WORDS}"; then
  printf 'error: MAX_WORDS must be a positive integer, got: %s\n' "${MAX_WORDS}" >&2
  usage >&2
  exit 1
fi
if ! is_positive_integer "${MAX_SYLLABLES_PER_WORD}"; then
  printf 'error: MAX_SYLLABLES_PER_WORD must be a positive integer, got: %s\n' "${MAX_SYLLABLES_PER_WORD}" >&2
  usage >&2
  exit 1
fi
if ! is_positive_integer "${SLOTS}"; then
  printf 'error: SLOTS must be a positive integer, got: %s\n' "${SLOTS}" >&2
  usage >&2
  exit 1
fi

cd "${BACKEND_DIR}"
if [[ ! -f "${INPUT_PATH}" ]]; then
  printf 'error: input file not found: %s\n' "${INPUT_PATH}" >&2
  exit 1
fi

OUTPUT_DIR="$(dirname "${OUTPUT_PATH}")"
if [[ ! -d "${OUTPUT_DIR}" ]]; then
  printf 'error: output directory does not exist: %s\n' "${OUTPUT_DIR}" >&2
  exit 1
fi

GOCACHE_PATH="${GOCACHE:-/tmp/go-build}"
log_info "backend_dir=${BACKEND_DIR}"
log_info "input=${INPUT_PATH} output=${OUTPUT_PATH} max_words=${MAX_WORDS} max_syllables=${MAX_SYLLABLES_PER_WORD} slots=${SLOTS} gocache=${GOCACHE_PATH}"

GOCACHE="${GOCACHE_PATH}" go run ./cmd/index_builder_build_idx \
  -input "${INPUT_PATH}" \
  -output "${OUTPUT_PATH}" \
  -max-words "${MAX_WORDS}" \
  -max-syllables-per-word "${MAX_SYLLABLES_PER_WORD}" \
  -slots "${SLOTS}"
