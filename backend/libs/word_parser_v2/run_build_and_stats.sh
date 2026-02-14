#!/usr/bin/env bash
set -euo pipefail

# Keep execution rooted at backend so Go module paths resolve predictably.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${BACKEND_DIR}"

INPUT_PATH="${1:-libs/word_parser_v2/productos.txt}"
OUTPUT_PATH="${2:-libs/word_parser_v2/productos.idx}"
GOCACHE_PATH="${GOCACHE:-/tmp/go-build}"
BUILD_LOG_PATH="${TMPDIR:-/tmp}/wp_build_human.log"
DECODE_LOG_PATH="${TMPDIR:-/tmp}/wp_decode_human.log"

# Build index with the current default strategy and slot config.
GOCACHE="${GOCACHE_PATH}" \
  go run ./cmd/word_parser_v2_build_idx \
  -input "${INPUT_PATH}" \
  -output "${OUTPUT_PATH}" \
  -slots 255 \
  -top 200 \
  -fixed-slots 45 \
  -strategy frequency \
  > "${BUILD_LOG_PATH}" 2>&1

# Decode summary only (no random samples) to inspect size/delta metrics.
GOCACHE="${GOCACHE_PATH}" \
  go run ./cmd/word_parser_v2_decode_idx \
  -input "${OUTPUT_PATH}" \
  -sample 0 \
  > "${DECODE_LOG_PATH}" 2>&1

# Remove timestamp prefix to keep output human readable.
sanitize_line() {
  sed -E 's/^[0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} //'
}

# Extract first matching line from a log file.
extract_line() {
  local pattern="$1"
  local source_file="$2"
  grep -m1 "${pattern}" "${source_file}" | sanitize_line
}

echo "Index Summary"
echo "input: ${INPUT_PATH}"
echo "output: ${OUTPUT_PATH}"
echo

fixed_count="$(extract_line "Fixed syllables" "${BUILD_LOG_PATH}" | sed -E 's/.*Fixed syllables \(([0-9]+)\).*/\1/')"
computed_count="$(extract_line "Computed syllables" "${BUILD_LOG_PATH}" | sed -E 's/.*Computed syllables \(([0-9]+)\).*/\1/')"
top_words="$(extract_line "computed_usage_top20" "${BUILD_LOG_PATH}" | sed -E 's/.*computed_usage_top20=//')"
shape_coverage="$(extract_line "shape_coverage_top255" "${BUILD_LOG_PATH}" | sed -E 's/.*shape_coverage_top255 //')"
unique_shapes="$(extract_line "shape_diversity" "${BUILD_LOG_PATH}" | sed -E 's/.*unique_shapes=([0-9]+).*/\1/')"
extracted_total="$(extract_line "stats fixed_syllables" "${BUILD_LOG_PATH}" | sed -E 's/.*extracted_syllables_total=([0-9]+).*/\1/')"
dictionary_bytes="$(extract_line "dictionary_count=" "${DECODE_LOG_PATH}" | sed -E 's/.*dictionary_bytes=([0-9]+).*/\1/')"
shape_delta_counts="$(extract_line "shape_delta_counts" "${DECODE_LOG_PATH}" | sed -E 's/.*shape_delta_counts: //')"
shape_storage_line="$(extract_line "shape_storage_stats" "${BUILD_LOG_PATH}")"
shape_delta_bytes="$(echo "${shape_storage_line}" | sed -E 's/.*shape_delta_stream_bytes=([0-9]+).*/\1/')"
shape_size_bytes="$(echo "${shape_storage_line}" | sed -E 's/.*total_stored_shapes_bytes=([0-9]+).*/\1/')"
content_size_bytes="$(echo "${shape_storage_line}" | sed -E 's/.*actual_content_bytes=([0-9]+).*/\1/')"

echo "fixed_syllables_count: ${fixed_count}"
echo "computed_syllables_count: ${computed_count}"
echo "most_used_words_top20: ${top_words}"
echo "shape_delta_counts: ${shape_delta_counts}"
echo "shape_delta_bytes: ${shape_delta_bytes}"
echo "dictionary_bytes: ${dictionary_bytes}"
echo "extracted_syllables_total: ${extracted_total}"
echo "shape_coverage_top255: ${shape_coverage}"
echo "unique_shapes: ${unique_shapes}"
echo "shape_size_bytes: ${shape_size_bytes}"
echo "text_content_size_bytes: ${content_size_bytes}"

if command -v stat >/dev/null 2>&1; then
  INDEX_BYTES="$(stat -c "%s" "${OUTPUT_PATH}")"
  INDEX_KB="$(awk -v bytes="${INDEX_BYTES}" 'BEGIN { printf "%.2f", bytes / 1024.0 }')"
  echo "total_size_bytes: ${INDEX_BYTES}"
  echo "total_size_kb: ${INDEX_KB}"
fi
