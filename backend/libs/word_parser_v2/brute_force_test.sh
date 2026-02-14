#!/usr/bin/env bash
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
cd "${BACKEND_DIR}"

RESULTS_FILE="libs/word_parser_v2/brute_force_results.txt"
rm -f "$RESULTS_FILE"
echo "Strategy,Slots,Fixed,Status,UniqueShapes" > "$RESULTS_FILE"

strategies=("frequency" "atomic_first" "atomic_digraph")
slots=(200 225 254)
fixed_counts=(30 60 90 120)

for strategy in "${strategies[@]}"; do
  for total_slots in "${slots[@]}"; do
    for fixed in "${fixed_counts[@]}"; do
      echo "Testing Strategy=$strategy Slots=$total_slots Fixed=$fixed..."
      
      # Run the build command and capture both stdout and stderr
      output=$(go run ./cmd/word_parser_v2_build_idx \
        -input libs/word_parser_v2/productos.txt \
        -output libs/word_parser_v2/productos.idx \
        -slots "$total_slots" \
        -top 200 \
        -fixed-slots "$fixed" \
        -strategy "$strategy" 2>&1)
      
      exit_code=$?
      
      # Extract unique shapes using sed for compatibility
      unique_shapes=$(echo "$output" | grep "unique_shapes=" | sed -E 's/.*unique_shapes=([0-9]+).*/\1/')
      if [ -z "$unique_shapes" ]; then unique_shapes="N/A"; fi
      
      if [ $exit_code -eq 0 ]; then
        status="SUCCESS"
      else
        status="FAIL"
      fi
      
      echo "$strategy,$total_slots,$fixed,$status,$unique_shapes" >> "$RESULTS_FILE"
      echo "  -> $status (Shapes: $unique_shapes)"
    done
  done
done

echo ""
echo "=== RESULTS SUMMARY ==="
column -t -s "," "$RESULTS_FILE"
