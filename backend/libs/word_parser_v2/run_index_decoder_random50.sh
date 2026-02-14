#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${BACKEND_DIR}"

GOCACHE="${GOCACHE:-/tmp/go-build}" \
  go run ./cmd/word_parser_v2_decode_idx \
  -input libs/word_parser_v2/productos.idx \
  -sample 50
