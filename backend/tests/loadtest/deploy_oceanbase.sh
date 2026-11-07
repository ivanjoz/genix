#!/usr/bin/env bash
# Builds the backend with -tags oceanbase and copies it, with the load-test scripts and config, to
# the homelab, where run_oceanbase.sh and measure_oceanbase.sh run. Run from backend/.
#
#   tests/loadtest/deploy_oceanbase.sh [HOST] [DIR]     default 100.64.0.4 ~/genix-loadtest
set -euo pipefail
HOST=${1:-100.64.0.4}
DIR=${2:-genix-loadtest}
BINARY="$(dirname "$(pwd)")/tmp/loadtest/backend-oceanbase"

GOOS=linux GOARCH=amd64 go build -tags oceanbase -o "$BINARY" .
ssh "$HOST" "mkdir -p $DIR/tests/loadtest"
rsync -az "$BINARY" "$HOST:$DIR/backend"
# fn-init reads its seed files (ubigeo, …) from assets/ in the working directory.
rsync -az assets "$HOST:$DIR/"
rsync -az tests/loadtest/run_oceanbase.sh tests/loadtest/measure_oceanbase.sh tests/loadtest/bench_sales_oceanbase.sh tests/loadtest/ob_metrics_diff.py \
	tests/loadtest/ob_fast_redo.sh tests/loadtest/run_seekdb.sh tests/loadtest/nosync_preload.c tests/loadtest/loadtest.config.toml "$HOST:$DIR/tests/loadtest/"
echo "deployed to $HOST:$DIR"
