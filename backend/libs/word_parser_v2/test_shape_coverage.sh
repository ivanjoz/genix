#!/usr/bin/env bash
set -euo pipefail

# Run from backend root so package paths resolve consistently.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${BACKEND_DIR}"

LOG_PATH="${SCRIPT_DIR}/shape_coverage_test.log"
OUTPUT_PATH="libs/word_parser_v2/productos.idx"

# Build the index and capture logs to parse coverage metrics.
GOCACHE="${GOCACHE:-/tmp/go-build}" \
  go run ./cmd/word_parser_v2_build_idx \
  -input libs/word_parser_v2/productos.txt \
  -output "${OUTPUT_PATH}" \
  -slots 200 \
  -top 200 \
  -fixed-slots 120 \
  -strategy atomic_first 2>&1 | tee "${LOG_PATH}"

echo
echo "Shape compact coverage summary (top 255 most common shapes):"
grep "shape_coverage_top255" "${LOG_PATH}" || {
  echo "shape_coverage_top255 line not found in ${LOG_PATH}"
  exit 1
}

echo "Header shape counters summary:"
grep "v3_index" "${LOG_PATH}" || {
  echo "v3_index line not found in ${LOG_PATH}"
  exit 1
}

echo
echo "Output index: ${BACKEND_DIR}/${OUTPUT_PATH}"
echo "Captured log: ${LOG_PATH}"
