#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DECODE_SCRIPT="${SCRIPT_DIR}/run_decode_and_stats.sh"
INPUT_PATH="${1:-libs/index_builder/productos.idx}"
SEED="${2:-0}"

if [[ "${INPUT_PATH}" == "-h" || "${INPUT_PATH}" == "--help" ]]; then
  cat <<'EOF'
Usage:
  run_index_decoder_random50.sh [INPUT_PATH] [SEED]

Defaults:
  INPUT_PATH=libs/index_builder/productos.idx
  SEED=0
EOF
  exit 0
fi

if [[ ! -x "${DECODE_SCRIPT}" ]]; then
  printf 'error: decode script not found or not executable: %s\n' "${DECODE_SCRIPT}" >&2
  exit 1
fi

cd "${BACKEND_DIR}"
"${DECODE_SCRIPT}" "${INPUT_PATH}" 50 "${SEED}"
