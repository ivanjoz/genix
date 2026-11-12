#!/usr/bin/env bash
# Copies the ScyllaDB load test to the homelab in the layout run.sh expects with SKIP_BUILD=1, so the
# Scylla baseline runs on the same machine as the OceanBase one. Run from backend/.
#
#   seeds/loadtest/deploy_scylla_homelab.sh [HOST] [DIR]     default 100.64.0.4 ~/genix-scylla
#
# On the homelab:  cd genix-scylla/backend && seeds/loadtest/run.sh stack   (once)
#                  seeds/loadtest/measure_scylla.sh NAME [ARGS]
set -euo pipefail
HOST=${1:-100.64.0.4}
DIR=${2:-genix-scylla}
PROJECT_DIR=$(dirname "$(pwd)")
BINARY="$PROJECT_DIR/tmp/loadtest/backend-scylla"
FAREWARD="$PROJECT_DIR/fareward/target/release/fareward"

GOOS=linux GOARCH=amd64 go build -o "$BINARY" .
GOOS=linux GOARCH=amd64 go build -o "$BINARY-seed" ./seeds/cmd/genix-seed
# The homelab has no cargo: ship the daemon built here.
(cd "$PROJECT_DIR/fareward" && cargo build --release)
ssh "$HOST" "mkdir -p $DIR/backend/seeds/loadtest $DIR/tmp/loadtest $DIR/fareward/target/release"
rsync -az "$BINARY" "$HOST:$DIR/tmp/loadtest/backend"
rsync -az "$BINARY-seed" "$HOST:$DIR/tmp/loadtest/genix-seed"
rsync -az "$FAREWARD" "$HOST:$DIR/fareward/target/release/"
rsync -az assets "$HOST:$DIR/backend/"
rsync -az seeds/loadtest/run.sh seeds/loadtest/measure_scylla.sh seeds/loadtest/scylla_metrics_diff.sh \
	seeds/loadtest/loadtest.config.toml "$HOST:$DIR/backend/seeds/loadtest/"
echo "deployed to $HOST:$DIR"
