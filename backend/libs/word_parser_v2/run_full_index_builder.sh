#!/usr/bin/env bash
set -euo pipefail

# Run from backend root so package paths resolve consistently.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${BACKEND_DIR}"

# Use writable cache path for restricted environments.
GOCACHE="${GOCACHE:-/tmp/go-build}" \
  go run ./cmd/word_parser_v2_build_idx \
  -input libs/word_parser_v2/productos.txt \
  -output libs/word_parser_v2/productos.idx \
  -slots 254 \
  -top 200 \
  -fixed-slots 120
