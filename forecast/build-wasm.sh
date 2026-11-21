#!/bin/bash
# Builds the forecast engine for the browser into frontend/libs/genix-forecast/, where the sales
# forecast page's worker imports it. The output is committed, so the frontend builds without Rust:
# run this after every change to the crate.
set -euo pipefail
cd "$(dirname "$0")"
OUT_DIR=../frontend/libs/genix-forecast
wasm-pack build --release --target web --out-dir "$OUT_DIR" --out-name genix_forecast
# wasm-pack's package.json and its ignore-everything .gitignore are for publishing to npm.
rm -f "$OUT_DIR/package.json" "$OUT_DIR/.gitignore" "$OUT_DIR/README.md"
# The generated glue is untyped JS; its .d.ts types the imports, so svelte-check skips the body.
sed -i '1i // @ts-nocheck' "$OUT_DIR/genix_forecast.js"
