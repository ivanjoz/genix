#!/usr/bin/env bash
# The load test on OceanBase, everything on the homelab: the observer (native, 127.0.0.1:2881,
# tenant `test`), the backend built with -tags oceanbase, and fn-load-test. fareward is bypassed
# ([fareward].bypass): OceanBase owns the sequences, the VPS backend locks in process, and nothing
# charges credits or checks access. See backend/tests/loadtest/plan.md ("V3").
#
# Runs from a directory holding `backend` (the oceanbase build) and tests/loadtest/ (this script and
# loadtest.config.toml); deploy_oceanbase.sh copies both there.
#
#   tests/loadtest/run_oceanbase.sh run [ARGS]   fresh database, empty companies, then the ramp
#
# ARGS go to fn-load-test, as with run.sh.
# BACKEND_CPUS / LOAD_CPUS pin the backend and the load generator; the observer keeps its own
# cpu_count and is not pinned.
set -euo pipefail

COMMAND=${1:?usage: run_oceanbase.sh run [ARGS]}
MAX_COMPANIES=100

RUN_DIR=$(pwd)
ROOT="$RUN_DIR/run"
LIVE_LOG="$ROOT/live.log"
CONFIG="$ROOT/loadtest.config.toml"
BINARY="$RUN_DIR/backend"
DATABASE=genix_load
BACKEND_CPUS=${BACKEND_CPUS:-0-3}
LOAD_CPUS=${LOAD_CPUS:-12-15}
# OB_USER / DB_PROCESS point the run at another OceanBase-engine server on 2881: seekdb has one
# tenant (OB_USER=root) and its process is `seekdb` (DB_PROCESS=seekdb, see run_seekdb.sh).
OB_USER=${OB_USER:-root@test}
DB_PROCESS=${DB_PROCESS:-observer}
# OB_ENGINE=seekdb deploys seekdb's subset (row tables, no TTL); on the observer too, to compare the
# two engines on the same tables.
OB_ENGINE=${OB_ENGINE:-}

live_log() {
	echo "$(date +%T) $*" | tee -a "$LIVE_LOG"
}

sql() {
	mariadb -h127.0.0.1 -P2881 -u"$OB_USER" --skip-ssl -N -e "$1"
}

stop_processes() {
	kill "${BACKEND_PID:-}" 2>/dev/null || true
	wait 2>/dev/null || true
}

case "$COMMAND" in
run)
	shift
	mkdir -p "$ROOT"
	: >"$LIVE_LOG"
	echo "live log: tail -f $LIVE_LOG"
	trap stop_processes EXIT

	# The Scylla config with the daemon bypassed and the OceanBase server added.
	sed -e "s|__KEYSPACE__|$DATABASE|g" -e "s|__ROOT__|$ROOT|g" \
		-e 's|^use_remote_dev_host = false$|use_remote_dev_host = false\nbypass = true|' \
		tests/loadtest/loadtest.config.toml >"$CONFIG"
	cat >>"$CONFIG" <<-EOF

	[oceanbase]
	host = "127.0.0.1"
	port = 2881
	user = "$OB_USER"
	password = ""
	partitions = 1
	max_conns = 64
	engine = "$OB_ENGINE"
	EOF

	live_log "dropping the previous database"
	sql "DROP DATABASE IF EXISTS $DATABASE"
	rm -rf "$ROOT/files"

	live_log "creating $MAX_COMPANIES empty companies (fn-init)"
	GENIX_CONFIG_FILE="$CONFIG" "$BINARY" fn-init --companies=$MAX_COMPANIES >"$ROOT/init.log" 2>&1

	GENIX_CONFIG_FILE="$CONFIG" taskset -c "$BACKEND_CPUS" "$BINARY" >"$ROOT/backend.log" 2>&1 &
	BACKEND_PID=$!
	until curl -s -o /dev/null "http://127.0.0.1:34010/api/p-hello"; do sleep 0.5; done
	OBSERVER_PID=$(pgrep -x "$DB_PROCESS" | head -1)
	live_log "server up; ramp starts (details: $ROOT/load.log)"

	GENIX_CONFIG_FILE="$CONFIG" taskset -c "$LOAD_CPUS" "$BINARY" fn-load-test \
		--server=http://127.0.0.1:34010 --max-companies=$MAX_COMPANIES --live="$LIVE_LOG" \
		--pids=backend=$BACKEND_PID,observer=$OBSERVER_PID "$@" >"$ROOT/load.log" 2>&1
	live_log "ramp finished; report at the end of $ROOT/load.log"
	;;

*)
	echo "usage: run_oceanbase.sh run [ARGS]" >&2
	exit 1
	;;
esac
