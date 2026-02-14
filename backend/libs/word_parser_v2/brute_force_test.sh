#!/usr/bin/env bash
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${BACKEND_DIR}"

RESULTS_FILE="libs/word_parser_v2/brute_force_results.txt"
TMP_DIR="$(mktemp -d /tmp/genix_bruteforce.XXXXXX)"
trap 'rm -rf "${TMP_DIR}"' EXIT

MAX_JOBS="${MAX_JOBS:-$(nproc)}"
BIN_PATH="${TMP_DIR}/word_parser_v2_build_idx"

echo "Building runner binary once..."
go build -o "${BIN_PATH}" ./cmd/word_parser_v2_build_idx

echo "Mode,Strategy,Slots,Fixed,Status,UniqueShapes,ContentBytes,Usage1,Usage2,Usage3" > "$RESULTS_FILE"

declare -A CASE_META
CASE_COUNTER=0

queue_case() {
  local mode="$1"
  local strategy="$2"
  local slots="$3"
  local fixed="$4"
  local case_id="$CASE_COUNTER"
  CASE_COUNTER=$((CASE_COUNTER + 1))
  CASE_META[$case_id]="$mode,$strategy,$slots,$fixed"

  while [ "$(jobs -rp | wc -l)" -ge "$MAX_JOBS" ]; do
    wait -n
  done

  (
    local two_letter_flag="false"
    if [ "$mode" = "two_letter_only" ]; then
      two_letter_flag="true"
    fi

    output=$("${BIN_PATH}" \
      -input libs/word_parser_v2/productos.txt \
      -output libs/word_parser_v2/productos.idx \
      -slots "$slots" \
      -top 200 \
      -fixed-slots "$fixed" \
      -strategy "$strategy" \
      -two-letter-only="$two_letter_flag" 2>&1)
    exit_code=$?

    unique_shapes=$(echo "$output" | grep "unique_shapes=" | sed -E 's/.*unique_shapes=([0-9]+).*/\1/')
    content_bytes=$(echo "$output" | grep "v3_index" | sed -E 's/.*content_size=([0-9]+).*/\1/')
    usage_line=$(echo "$output" | grep "syllable_length_usage")
    usage_1=$(echo "$usage_line" | sed -E 's/.*usage_1=([0-9]+).*/\1/')
    usage_2=$(echo "$usage_line" | sed -E 's/.*usage_2=([0-9]+).*/\1/')
    usage_3=$(echo "$usage_line" | sed -E 's/.*usage_3=([0-9]+).*/\1/')

    if [ -z "$unique_shapes" ]; then unique_shapes="N/A"; fi
    if [ -z "$content_bytes" ]; then content_bytes="N/A"; fi
    if [ -z "$usage_1" ]; then usage_1="N/A"; fi
    if [ -z "$usage_2" ]; then usage_2="N/A"; fi
    if [ -z "$usage_3" ]; then usage_3="N/A"; fi

    if [ $exit_code -eq 0 ]; then
      status="SUCCESS"
    else
      status="FAIL"
    fi

    echo "$mode,$strategy,$slots,$fixed,$status,$unique_shapes,$content_bytes,$usage_1,$usage_2,$usage_3" > "${TMP_DIR}/case_${case_id}.csv"
    echo "[case ${case_id}] $mode $strategy slots=$slots fixed=$fixed -> $status shapes=$unique_shapes content=$content_bytes usage=$usage_1/$usage_2/$usage_3"
  ) &
}

# 10 focused combinations only (drop lower-scored runs):
# - Keep best slot region (250/255)
# - Compare frequency vs coverage_greedy in both modes
# - Keep ratio_80_20 checkpoints at 255
queue_case "standard" "frequency" 250 60
queue_case "standard" "frequency" 255 60
queue_case "standard" "coverage_greedy" 250 60
queue_case "standard" "coverage_greedy" 255 60
queue_case "two_letter_only" "frequency" 250 60
queue_case "two_letter_only" "frequency" 255 60
queue_case "two_letter_only" "coverage_greedy" 250 60
queue_case "two_letter_only" "coverage_greedy" 255 60
queue_case "standard" "ratio_80_20" 255 60
queue_case "two_letter_only" "ratio_80_20" 255 60

wait

for ((case_id=0; case_id<CASE_COUNTER; case_id++)); do
  cat "${TMP_DIR}/case_${case_id}.csv" >> "$RESULTS_FILE"
done

echo ""
echo "=== RESULTS SUMMARY ==="
column -t -s "," "$RESULTS_FILE"
