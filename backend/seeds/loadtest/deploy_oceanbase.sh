#!/usr/bin/env bash
# Builds the backend with -tags oceanbase and copies it, with the load-test scripts and config, to
# the homelab, where run_oceanbase.sh and measure_oceanbase.sh run. Run from backend/.
#
#   seeds/loadtest/deploy_oceanbase.sh [HOST] [DIR]     default 100.64.0.4 ~/genix-loadtest
set -euo pipefail
HOST=${1:-100.64.0.4}
DIR=${2:-genix-loadtest}
BINARY="$(dirname "$(pwd)")/tmp/loadtest/backend-oceanbase"

GOOS=linux GOARCH=amd64 go build -tags oceanbase -o "$BINARY" .
GOOS=linux GOARCH=amd64 go build -tags oceanbase -o "$BINARY-seed" ./seeds/cmd/genix-seed
ssh "$HOST" "mkdir -p $DIR/seeds/loadtest/nosync"
rsync -az "$BINARY" "$HOST:$DIR/backend"
rsync -az "$BINARY-seed" "$HOST:$DIR/genix-seed"
# fn-init reads its seed files (ubigeo, …) from assets/ in the working directory.
rsync -az assets "$HOST:$DIR/"
rsync -az seeds/loadtest/run_oceanbase.sh seeds/loadtest/measure_oceanbase.sh seeds/loadtest/bench_sales_oceanbase.sh seeds/loadtest/ob_metrics_diff.py \
	seeds/loadtest/ob_fast_redo.sh seeds/loadtest/run_seekdb.sh seeds/loadtest/loadtest.config.toml "$HOST:$DIR/seeds/loadtest/"
rsync -az seeds/loadtest/nosync/nosync_preload.c "$HOST:$DIR/seeds/loadtest/nosync/"
echo "deployed to $HOST:$DIR"
