#!/usr/bin/env bash
# Load test of the backend as a 4 GB / 2 vCPU VPS: ScyllaDB, fareward and the backend share CPUs
# 0-1 and about 4 GB, and genix-seed loadtest drives them from the other CPUs. Run from backend/.
#
#   seeds/loadtest/run.sh stack        ScyllaDB container on 127.0.0.2 (once)
#   seeds/loadtest/run.sh run [ARGS]   fresh keyspace, empty companies, then the ramp
#
# Follow it from its first second:  tail -f tmp/loadtest/run/live.log
#
# There is no seeded history. fn-init creates the companies empty, and every stage a few of them
# onboard through the server (staff, catalog, providers, first stock) and start selling, while the
# ones before keep working.
#
# ARGS go to genix-seed loadtest: --companies-per-stage=5 --stage-seconds=45 --max-companies=100
#                          --catalog=2000 --cashiers=4 --think-ms=2000 --max-p99-ms=1000
# TABLETS=false run ...    the keyspace on vnodes instead of tablets
# TABLET_BALANCING=false run ...   no tablet migrations during the ramp
# FAREWARD_BYPASS=1 run ...        [fareward].bypass: no credit charge, access check or request log
#                                  (fareward still serves the sequences)
# SKIP_BUILD=1 run ...             use the binary already at tmp/loadtest/backend (a deployed layout,
#                                  see deploy_scylla_homelab.sh)
# STACK_CPUS, LOAD_CPUS, SCYLLA_SMP, SCYLLA_MEMORY, BACKEND_MEMORY override the VPS profile below.
set -euo pipefail

COMMAND=${1:?usage: run.sh stack|run}
MAX_COMPANIES=100

BACKEND_DIR=$(pwd)
PROJECT_DIR=$(dirname "$BACKEND_DIR")
ROOT="$PROJECT_DIR/tmp/loadtest/run"
LIVE_LOG="$ROOT/live.log"
CONFIG="$ROOT/loadtest.config.toml"
BINARY="$PROJECT_DIR/tmp/loadtest/backend"
SEED_BINARY="$PROJECT_DIR/tmp/loadtest/genix-seed"
FAREWARD="$PROJECT_DIR/fareward/target/release/fareward"
KEYSPACE=genix_load
STACK_CPUS=${STACK_CPUS:-0,1}
LOAD_CPUS=${LOAD_CPUS:-2-$(($(nproc) - 1))}

# The VPS share of 4 GB: ScyllaDB 1.5 GB, backend 1.5 GB, fareward 256 MB.
SCYLLA_MEMORY=${SCYLLA_MEMORY:-1500M}
BACKEND_MEMORY=${BACKEND_MEMORY:-1536M}
FAREWARD_MEMORY=256M

# live_log prints a setup step and appends it to the live log, beside the ramp's status lines.
live_log() {
	echo "$(date +%T) $*" | tee -a "$LIVE_LOG"
}

cql() {
	podman exec genix-load-scylla cqlsh 127.0.0.2 --request-timeout=600 -e "$1"
}

start_fareward() {
	# exec: $! is then fareward itself (systemd-run --scope and taskset exec in place).
	(cd "$PROJECT_DIR/fareward" && GENIX_CONFIG_FILE="$CONFIG" exec systemd-run --user --scope --quiet \
		-p MemoryMax=$FAREWARD_MEMORY taskset -c $STACK_CPUS "$FAREWARD" >"$ROOT/fareward.log" 2>&1) &
	FAREWARD_PID=$!
	sleep 2
}

start_backend() {
	GENIX_CONFIG_FILE="$CONFIG" systemd-run --user --scope --quiet -p MemoryMax=$BACKEND_MEMORY \
		taskset -c $STACK_CPUS "$BINARY" >"$ROOT/backend.log" 2>&1 &
	BACKEND_PID=$!
	until curl -s -o /dev/null "http://127.0.0.1:34010/api/p-hello"; do sleep 0.5; done
}

stop_processes() {
	for pid in ${BACKEND_PID:-} ${FAREWARD_PID:-}; do kill "$pid" 2>/dev/null || true; done
	wait 2>/dev/null || true
	curl -sf -X POST "http://127.0.0.2:10000/storage_service/tablets/balancing?enabled=true" || true
}

case "$COMMAND" in
stack)
	# Rootless podman has no cpuset controller here. --cpuset picks the two shards' CPUs, but with
	# --overprovisioned seastar does not pin its threads, so `run` pins them with taskset.
	# No container memory cap: seastar reserves memory beyond --memory and refuses to start under
	# one; --memory alone bounds it.
	# SCYLLA_SMP=1 compares one shard on the two shared cores.
	podman run -d --name genix-load-scylla --network host \
		docker.io/scylladb/scylla:latest \
		--smp ${SCYLLA_SMP:-2} --cpuset $STACK_CPUS --memory $SCYLLA_MEMORY --overprovisioned 1 --developer-mode 1 \
		--listen-address 127.0.0.2 --rpc-address 127.0.0.2 --broadcast-rpc-address 127.0.0.2 \
		--api-address 127.0.0.2 --prometheus-address 127.0.0.2 --seeds 127.0.0.2
	until cql 'SELECT now() FROM system.local' >/dev/null 2>&1; do sleep 2; done
	echo "genix-load-scylla is up on 127.0.0.2:9042"
	;;

run)
	shift
	mkdir -p "$ROOT"
	: >"$LIVE_LOG"
	echo "live log: tail -f $LIVE_LOG"
	trap stop_processes EXIT

	live_log "building the backend"
	sed -e "s|__KEYSPACE__|$KEYSPACE|g" -e "s|__ROOT__|$ROOT|g" seeds/loadtest/loadtest.config.toml >"$CONFIG"
	if [ "${FAREWARD_BYPASS:-0}" = 1 ]; then
		sed -i 's|^use_remote_dev_host = false$|use_remote_dev_host = false\nbypass = true|' "$CONFIG"
	fi
	if [ "${SKIP_BUILD:-0}" != 1 ]; then go build -o "$BINARY" . && go build -o "$SEED_BINARY" ./seeds/cmd/genix-seed; fi

	# Every run starts from empty companies. ScyllaDB snapshots each table it drops: a large keyspace
	# can time the DROP out (it goes on in the background) or answer it before it is done, so retry
	# until the keyspace is gone and then wait until it stops truncating.
	live_log "dropping the previous keyspace"
	until [ -z "$(cql "SELECT keyspace_name FROM system_schema.keyspaces WHERE keyspace_name='$KEYSPACE'" | grep " $KEYSPACE\$" || true)" ]; do
		cql "DROP KEYSPACE IF EXISTS $KEYSPACE" || sleep 5
	done
	until [ -z "$(podman logs --since 15s genix-load-scylla 2>&1 | grep -E 'Truncat|pre-drop' || true)" ]; do sleep 3; done
	rm -rf "$ROOT/files"

	# TABLETS=false compares vnodes: fn-init creates the keyspace only IF NOT EXISTS, so creating it
	# here first, with tablets off, decides how every table is distributed across the shards.
	if [ "${TABLETS:-true}" = false ]; then
		live_log "creating $KEYSPACE with vnodes (tablets off)"
		cql "CREATE KEYSPACE $KEYSPACE WITH replication = {'class': 'NetworkTopologyStrategy', 'replication_factor': 1} AND tablets = {'enabled': false}"
	fi
	live_log "creating $MAX_COMPANIES empty companies (fn-init)"
	GENIX_CONFIG_FILE="$CONFIG" "$BINARY" fn-init --companies=$MAX_COMPANIES >"$ROOT/init.log" 2>&1
	# A tablets keyspace creates every table with one tablet per shard and the balancer then merges
	# them, one topology transition at a time; writes during that churn time out. Wait until there
	# has been no transition for 15 seconds in a row.
	live_log "waiting for the tablet balancer to settle"
	quiet_checks=0
	while [ $quiet_checks -lt 3 ]; do
		if cql 'SELECT transition_state FROM system.topology' | grep -q ' null$'; then
			quiet_checks=$((quiet_checks + 1))
		else quiet_checks=0; fi
		sleep 5
	done

	# TABLET_BALANCING=false stops the balancer for the ramp: on one node it only moves tablets
	# between shards, and each move's barrier_and_drain stalls writes long enough to time them out.
	if [ "${TABLET_BALANCING:-true}" = false ]; then
		live_log "tablet balancing off for the ramp"
		curl -sf -X POST "http://127.0.0.2:10000/storage_service/tablets/balancing?enabled=false"
	fi

	# fn-init creates the keyspace fareward connects to, so the daemon starts after it.
	start_fareward
	start_backend
	# The container's own PID is its init; the database is the scylla process under it.
	SCYLLA_PID=$(pgrep -f '^/usr/bin/scylla .*127\.0\.0\.2' | head -1)
	# --overprovisioned turns off seastar's thread pinning, so --cpuset alone leaves every scylla
	# thread free to run on any host CPU: the "2 vCPU" stack then had up to ~2 more cores than a VPS.
	# Pin all its threads (reactors, syscall and timer threads) to the stack's CPUs from the host.
	taskset -a -p -c $STACK_CPUS "$SCYLLA_PID" >/dev/null
	live_log "server up; ramp starts (details: $ROOT/load.log)"

	GENIX_CONFIG_FILE="$CONFIG" taskset -c "$LOAD_CPUS" "$SEED_BINARY" loadtest \
		--server=http://127.0.0.1:34010 --max-companies=$MAX_COMPANIES --live="$LIVE_LOG" \
		--pids=backend=$BACKEND_PID,fareward=$FAREWARD_PID,scylla=$SCYLLA_PID "$@" >"$ROOT/load.log" 2>&1
	live_log "ramp finished; report at the end of $ROOT/load.log"
	;;

*)
	echo "usage: run.sh stack|run" >&2
	exit 1
	;;
esac
