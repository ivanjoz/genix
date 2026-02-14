#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${BACKEND_DIR}"

INPUT_PATH="${1:-libs/index_builder/productos.idx}"

GOCACHE="${GOCACHE:-/tmp/go-build}" \
  go run ./cmd/index_builder_decode_idx \
  -input "${INPUT_PATH}" \
  -sample 50

