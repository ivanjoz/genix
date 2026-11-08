#!/usr/bin/env bash
# The load test on PostgreSQL, as run.sh runs it on ScyllaDB: a 4 GB / 2 vCPU VPS where PostgreSQL
# and the backend (built with -tags postgres) share CPUs 0-1, and fn-load-test drives them from the
# other CPUs. fareward is bypassed ([fareward].bypass): PostgreSQL owns the sequences, the VPS backend
# locks in process, and nothing charges credits or checks access. Run from backend/.
#
#   tests/loadtest/run_postgres.sh stack        PostgreSQL container on 127.0.0.1:5433 (once)
#   tests/loadtest/run_postgres.sh run [ARGS]   fresh schema, empty companies, then the ramp
#
# ARGS go to fn-load-test, as with run.sh. SKIP_BUILD=1 uses the binary already at
# tmp/loadtest/backend-pg.
#
# DB_KIND=yugabyte runs the same build against YugabyteDB's YSQL (PostgreSQL-compatible) instead: the
# container genix-load-yugabyte on 127.0.0.3:5433, its tables in the colocated database "genix"
# (`stack` starts it). DB_KIND=cockroach runs it against CockroachDB: genix-load-cockroach on
# 127.0.0.1:26257, database defaultdb. DB_KIND=mariadb runs the untagged (MariaDB) build
# (tmp/loadtest/backend-maria) against MariaDB: genix-load-mariadb on 127.0.0.1:3307, the keyspace
# its own database. DB_KIND=myrocks runs that build against MariaDB with MyRocks (RocksDB) as the
# default storage engine: genix-load-myrocks on 127.0.0.1:3308, from the image `stack` builds.
set -euo pipefail

COMMAND=${1:?usage: run_postgres.sh stack|run}
MAX_COMPANIES=100

BACKEND_DIR=$(pwd)
PROJECT_DIR=$(dirname "$BACKEND_DIR")
ROOT="$PROJECT_DIR/tmp/loadtest/run"
LIVE_LOG="$ROOT/live.log"
CONFIG="$ROOT/loadtest.config.toml"
SCHEMA=genix_load
DB_KIND=${DB_KIND:-postgres}
BINARY="$PROJECT_DIR/tmp/loadtest/backend-pg" BUILD_TAG=postgres CONFIG_SECTION=postgres
if [ "$DB_KIND" = mariadb ] || [ "$DB_KIND" = myrocks ]; then
	# MariaDB is the untagged build.
	BINARY="$PROJECT_DIR/tmp/loadtest/backend-maria" BUILD_TAG="" CONFIG_SECTION=mariadb
fi
STACK_CPUS=${STACK_CPUS:-0,1}
LOAD_CPUS=${LOAD_CPUS:-2-$(($(nproc) - 1))}

# The VPS share of 4 GB, as on ScyllaDB: the database 1.5 GB, the backend 1.5 GB.
POSTGRES_MEMORY=1536m
BACKEND_MEMORY=1536M

live_log() {
	echo "$(date +%T) $*" | tee -a "$LIVE_LOG"
}

case "$DB_KIND" in
yugabyte) CONTAINER=genix-load-yugabyte DB_HOST=127.0.0.3 DB_PORT=5433 DB_USER=yugabyte DB_PASSWORD="" DB_DATABASE=genix ;;
# CockroachDB's insecure single node only listens on localhost.
cockroach) CONTAINER=genix-load-cockroach DB_HOST=127.0.0.1 DB_PORT=26257 DB_USER=root DB_PASSWORD="" DB_DATABASE=defaultdb ;;
mariadb) CONTAINER=genix-load-mariadb DB_HOST=127.0.0.1 DB_PORT=3307 DB_USER=root DB_PASSWORD=genix DB_DATABASE="" ;;
myrocks) CONTAINER=genix-load-myrocks DB_HOST=127.0.0.1 DB_PORT=3308 DB_USER=root DB_PASSWORD=genix DB_DATABASE="" ;;
*) CONTAINER=genix-load-postgres DB_HOST=127.0.0.1 DB_PORT=5433 DB_USER=postgres DB_PASSWORD=genix DB_DATABASE=postgres ;;
esac

psql_command() {
	case "$DB_KIND" in
	yugabyte) podman exec "$CONTAINER" bin/ysqlsh -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "${2:-$DB_DATABASE}" -v ON_ERROR_STOP=1 -qAt -c "$1" ;;
	cockroach) podman exec "$CONTAINER" ./cockroach sql --insecure --host="$DB_HOST:$DB_PORT" -d "${2:-$DB_DATABASE}" --format=tsv -e "$1" ;;
	mariadb | myrocks) podman exec "$CONTAINER" mariadb -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" -p"$DB_PASSWORD" -N -B -e "$1" ;;
	*) podman exec "$CONTAINER" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "${2:-$DB_DATABASE}" -v ON_ERROR_STOP=1 -qAt -c "$1" ;;
	esac
}

# pin_database puts every process of the database container on the stack's CPUs. A connection's
# process is forked by the postmaster and inherits its affinity, so the processes started later stay
# there too.
pin_database() {
	local cgroup
	cgroup="/sys/fs/cgroup$(cut -d: -f3 /proc/"$(podman inspect "$CONTAINER" --format '{{.State.Pid}}')"/cgroup)"
	for pid in $(cat "$cgroup"/cgroup.procs "$cgroup"/*/cgroup.procs 2>/dev/null); do
		taskset -a -p -c "$STACK_CPUS" "$pid" >/dev/null 2>&1 || true
	done
}

stop_processes() {
	kill "${BACKEND_PID:-}" 2>/dev/null || true
	wait 2>/dev/null || true
}

case "$COMMAND" in
stack)
	if [ "$DB_KIND" = yugabyte ]; then
		# One node, 1.5 GB: the master 256 MB, the tablet server (DocDB and its YSQL backends) the
		# rest. Its WAL is not fsynced per write by default, like ScyllaDB's periodic commitlog.
		podman run -d --name genix-load-yugabyte --network host --memory "$POSTGRES_MEMORY" docker.io/yugabytedb/yugabyte:latest \
			bin/yugabyted start --advertise_address=127.0.0.3 --background=false \
			--master_flags="memory_limit_hard_bytes=268435456" \
			--tserver_flags="memory_limit_hard_bytes=1207959552,ysql_max_connections=120"
		until psql_command "SELECT 1" yugabyte >/dev/null 2>&1; do sleep 3; done
		# Colocation keeps the ~200 small tables in one tablet instead of a tablet each.
		psql_command "CREATE DATABASE genix WITH COLOCATION = true" yugabyte
		echo "genix-load-yugabyte is up on 127.0.0.3:5433"
		exit 0
	fi
	if [ "$DB_KIND" = myrocks ]; then
		# The official image lacks MyRocks: a local image adds MariaDB's own mariadb-plugin-rocksdb
		# package (its rocksdb.cnf loads the plugin). Tables default to ROCKSDB; InnoDB keeps only the
		# server's own tables, so its buffer pool is small. A 512 MB block cache in the 1.5 GB, and
		# rocksdb_flush_log_at_trx_commit=2 syncs the WAL once a second, matching the other stacks.
		# rocksdb_large_prefix allows InnoDB's 3072-byte index keys (VARCHAR(512) keys), and a row lock
		# waits InnoDB's 50 s instead of MyRocks' 1 s: at 1 s, ten concurrent catalog imports of 500
		# rows time each other out. Deadlock detection is on, as InnoDB's is (MyRocks' default is off,
		# so a cycle waits out the whole timeout).
		printf 'FROM docker.io/library/mariadb:latest\nRUN apt-get update && apt-get install -y --no-install-recommends mariadb-plugin-rocksdb && rm -rf /var/lib/apt/lists/*\n' |
			podman build -t localhost/genix-mariadb-rocksdb:latest -f - "$(mktemp -d)"
		podman run -d --name genix-load-myrocks --network host --memory "$POSTGRES_MEMORY" \
			-e MARIADB_ROOT_PASSWORD=genix -e MARIADB_ROOT_HOST=% localhost/genix-mariadb-rocksdb:latest \
			--port=3308 --bind-address=127.0.0.1 --max-connections=150 --default-storage-engine=ROCKSDB \
			--rocksdb-block-cache-size=512M --rocksdb-flush-log-at-trx-commit=2 --rocksdb-large-prefix=ON --rocksdb-lock-wait-timeout=50 --rocksdb-deadlock-detect=ON \
			--innodb-buffer-pool-size=64M \
			--skip-log-bin --performance-schema=OFF
		until psql_command "SELECT 1" >/dev/null 2>&1; do sleep 2; done
		echo "genix-load-myrocks is up on 127.0.0.1:3308 ($(psql_command "SELECT @@default_storage_engine"))"
		exit 0
	fi
	if [ "$DB_KIND" = mariadb ]; then
		# The server tuned for 1.5 GB: a 768 MB buffer pool. innodb_flush_log_at_trx_commit=2 writes
		# the redo log at commit and fsyncs it once a second, matching the other stacks (ScyllaDB's
		# periodic commitlog, PostgreSQL's synchronous_commit=off). No binary log. The thread pool
		# runs the statements on one thread per vCPU instead of one per connection: 15% less CPU per sale
		# than the per-connection threads on the 2 vCPUs (MARIADB_TUNING.md, step 4a). MARIADB_FLAGS adds
		# server options (the tuning steps in MARIADB_TUNING.md).
		podman run -d --name genix-load-mariadb --network host --memory "$POSTGRES_MEMORY" \
			-e MARIADB_ROOT_PASSWORD=genix -e MARIADB_ROOT_HOST=% docker.io/library/mariadb:latest \
			--port=3307 --bind-address=127.0.0.1 --max-connections=150 --innodb-buffer-pool-size=768M \
			--innodb-log-file-size=512M --innodb-flush-log-at-trx-commit=2 --skip-log-bin --performance-schema=OFF \
			--thread-handling=pool-of-threads --thread-pool-size=2 ${MARIADB_FLAGS:-}
		until psql_command "SELECT 1" >/dev/null 2>&1; do sleep 2; done
		echo "genix-load-mariadb is up on 127.0.0.1:3307"
		exit 0
	fi
	if [ "$DB_KIND" = cockroach ]; then
		# One insecure node, 1.5 GB: a quarter for its block cache, a quarter for SQL memory.
		podman run -d --name genix-load-cockroach --network host --memory "$POSTGRES_MEMORY" docker.io/cockroachdb/cockroach:latest \
			start-single-node --insecure --listen-addr=127.0.0.1:26257 --http-addr=127.0.0.1:8089 --cache=.25 --max-sql-memory=.25
		until psql_command "SELECT 1" >/dev/null 2>&1; do sleep 2; done
		# Its Raft log is fsynced on every commit by default. Turned off to match the other stacks
		# (ScyllaDB's periodic commitlog, PostgreSQL's synchronous_commit=off): a crash may lose the
		# last writes. Test-only, as the setting's name says.
		# An unsafe setting needs an interlock key the server issues per session, so one interactive
		# session reads the key from the refusal, sends it, and repeats the statement.
		unsafe_setting="SET CLUSTER SETTING kv.raft_log.synchronization.unsafe.disabled = true;"
		coproc COCKROACH_SQL { podman exec -i "$CONTAINER" ./cockroach sql --insecure --host="$DB_HOST:$DB_PORT" --set=errexit=false 2>&1; }
		echo "$unsafe_setting" >&"${COCKROACH_SQL[1]}"
		while read -r -t 20 line <&"${COCKROACH_SQL[0]}"; do
			case "$line" in "DETAIL: key: "*) interlock_key=${line#DETAIL: key: }; break ;; esac
		done
		printf "SET unsafe_setting_interlock_key = '%s';\n%s\n\\\\q\n" "$interlock_key" "$unsafe_setting" >&"${COCKROACH_SQL[1]}"
		cat <&"${COCKROACH_SQL[0]}" >/dev/null
		echo "Raft log fsync: $(psql_command "SHOW CLUSTER SETTING kv.raft_log.synchronization.unsafe.disabled" | tail -1) (t = off)"
		echo "genix-load-cockroach is up on 127.0.0.1:26257"
		exit 0
	fi
	# The server tuned for 1.5 GB. synchronous_commit=off matches ScyllaDB's periodic commitlog: a
	# commit does not wait for its WAL flush (a crash loses at most the last ~0.6 s, never consistency).
	# POSTGRES_FLAGS adds or overrides settings (-c name=value; the last one wins), see POSTGRES_TUNING.md.
	podman run -d --name genix-load-postgres --network host --memory "$POSTGRES_MEMORY" \
		-e POSTGRES_PASSWORD=genix -e POSTGRES_INITDB_ARGS="--locale=C.UTF-8" docker.io/library/postgres:18 \
		-c port=5433 -c listen_addresses=127.0.0.1 -c max_connections=120 \
		-c shared_buffers=384MB -c effective_cache_size=1GB -c work_mem=8MB -c maintenance_work_mem=128MB \
		-c synchronous_commit=off -c wal_compression=lz4 -c default_toast_compression=lz4 \
		-c checkpoint_timeout=15min -c max_wal_size=2GB -c random_page_cost=1.1 -c jit=off ${POSTGRES_FLAGS:-}
	until podman exec genix-load-postgres pg_isready -h 127.0.0.1 -p 5433 >/dev/null 2>&1; do sleep 1; done
	echo "genix-load-postgres is up on 127.0.0.1:5433"
	;;

run)
	shift
	mkdir -p "$ROOT"
	: >"$LIVE_LOG"
	echo "live log: tail -f $LIVE_LOG"
	trap stop_processes EXIT

	# The Scylla config with the daemon bypassed and the PostgreSQL server added.
	sed -e "s|__KEYSPACE__|$SCHEMA|g" -e "s|__ROOT__|$ROOT|g" \
		-e 's|^use_remote_dev_host = false$|use_remote_dev_host = false\nbypass = true|' \
		tests/loadtest/loadtest.config.toml >"$CONFIG"
	cat >>"$CONFIG" <<-EOF

	[$CONFIG_SECTION]
	host = "$DB_HOST"
	port = $DB_PORT
	user = "$DB_USER"
	password = "$DB_PASSWORD"
	database = "$DB_DATABASE"
	max_conns = ${POSTGRES_MAX_CONNS:-32}
	EOF
	if [ "${SKIP_BUILD:-0}" != 1 ]; then
		live_log "building the backend (tags: ${BUILD_TAG:-none})"
		go build -tags "$BUILD_TAG" -o "$BINARY" .
	fi

	live_log "dropping the previous schema"
	if [ "$DB_KIND" = mariadb ] || [ "$DB_KIND" = myrocks ]; then
		psql_command "DROP DATABASE IF EXISTS $SCHEMA"
	else
		psql_command "DROP SCHEMA IF EXISTS $SCHEMA CASCADE"
	fi
	rm -rf "$ROOT/files"

	live_log "creating $MAX_COMPANIES empty companies (fn-init)"
	GENIX_CONFIG_FILE="$CONFIG" "$BINARY" fn-init --companies=$MAX_COMPANIES >"$ROOT/init.log" 2>&1
	# A fresh baseline for the planner and the statistics the measurement reads. CockroachDB and
	# MariaDB (either engine) collect theirs on their own.
	if [ "$DB_KIND" = postgres ] || [ "$DB_KIND" = yugabyte ]; then
		psql_command "ANALYZE" >/dev/null
		psql_command "SELECT pg_stat_reset()" >/dev/null
	fi

	pin_database
	GENIX_CONFIG_FILE="$CONFIG" systemd-run --user --scope --quiet -p MemoryMax=$BACKEND_MEMORY \
		taskset -c "$STACK_CPUS" "$BINARY" >"$ROOT/backend.log" 2>&1 &
	BACKEND_PID=$!
	until curl -s -o /dev/null "http://127.0.0.1:34010/api/p-hello"; do sleep 0.5; done
	pin_database
	POSTMASTER_PID=$(podman inspect "$CONTAINER" --format '{{.State.Pid}}')
	live_log "server up; ramp starts (details: $ROOT/load.log)"

	# The live log's postgres column is the container's first process alone; measure_postgres.sh reads
	# the whole container's CPU from its cgroup.
	GENIX_CONFIG_FILE="$CONFIG" taskset -c "$LOAD_CPUS" "$BINARY" fn-load-test \
		--server=http://127.0.0.1:34010 --max-companies=$MAX_COMPANIES --live="$LIVE_LOG" \
		--pids=backend=$BACKEND_PID,postgres=$POSTMASTER_PID "$@" >"$ROOT/load.log" 2>&1
	live_log "ramp finished; report at the end of $ROOT/load.log"
	;;

*)
	echo "usage: run_postgres.sh stack|run [ARGS]" >&2
	exit 1
	;;
esac
