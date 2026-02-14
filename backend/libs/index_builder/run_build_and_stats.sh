#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${BACKEND_DIR}"

INPUT_PATH="${1:-libs/index_builder/productos.txt}"
OUTPUT_PATH="${2:-libs/index_builder/productos.idx}"
MAX_WORDS="${3:-8}"
MAX_SYLLABLES_PER_WORD="${4:-7}"
SLOTS="${5:-255}"
GOCACHE_PATH="${GOCACHE:-/tmp/go-build}"

GOCACHE="${GOCACHE_PATH}" \
  go run ./cmd/index_builder_build_idx \
  -input "${INPUT_PATH}" \
  -output "${OUTPUT_PATH}" \
  -max-words "${MAX_WORDS}" \
  -max-syllables-per-word "${MAX_SYLLABLES_PER_WORD}" \
  -slots "${SLOTS}"
