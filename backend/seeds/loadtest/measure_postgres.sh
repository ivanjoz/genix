#!/usr/bin/env bash
# measure_postgres.sh NAME [genix-seed loadtest args...]: the PostgreSQL counterpart of measure_scylla.sh, run
# from backend/. Same steady state (10 companies) and the same 90 s window starting 30 s after the
# 10th company starts selling. Each end of the window snapshots the CPU of the whole PostgreSQL
# container (its cgroup: every connection is its own process) and of the backend, and
# pg_stat_database; the sales come from the live log's sales/s lines inside the window.
set -u
NAME=$1; shift
OUT="$(pwd)/measure/$NAME"
RUN="$(dirname "$(pwd)")/tmp/loadtest/run"
mkdir -p "$OUT"

# DB_KIND=yugabyte measures YugabyteDB's YSQL instead, DB_KIND=cockroach CockroachDB, DB_KIND=mariadb
# MariaDB on InnoDB and DB_KIND=myrocks on MyRocks (run_postgres.sh).
DB_KIND=${DB_KIND:-postgres}
BACKEND_BINARY=backend-pg
case "$DB_KIND" in
yugabyte)
	CONTAINER=genix-load-yugabyte
	PSQL=(podman exec "$CONTAINER" bin/ysqlsh -h 127.0.0.3 -p 5433 -U yugabyte -d genix)
	STAT_DATABASE=genix
	;;
cockroach) CONTAINER=genix-load-cockroach ;;
mariadb) CONTAINER=genix-load-mariadb BACKEND_BINARY=backend-maria MARIADB_PORT=3307 ;;
myrocks) CONTAINER=genix-load-myrocks BACKEND_BINARY=backend-maria MARIADB_PORT=3308 ;;
*)
	CONTAINER=genix-load-postgres
	PSQL=(podman exec "$CONTAINER" psql -h 127.0.0.1 -p 5433 -U postgres)
	STAT_DATABASE=postgres
	;;
esac

# database_counters prints "name value" lines: pg_stat_database's counters, or on CockroachDB (which
# has no pg_stat_database) its node's transaction and statement counters, or MariaDB's global status.
database_counters() {
	if [ "$DB_KIND" = mariadb ] || [ "$DB_KIND" = myrocks ]; then
		podman exec "$CONTAINER" mariadb -h 127.0.0.1 -P "$MARIADB_PORT" -uroot -pgenix -N -B -e \
			"SELECT LOWER(variable_name), variable_value FROM information_schema.global_status WHERE variable_name IN
			 ('COM_COMMIT', 'COM_ROLLBACK', 'QUESTIONS', 'COM_SELECT', 'COM_INSERT', 'COM_UPDATE', 'COM_DELETE',
			 'INNODB_ROWS_READ', 'INNODB_ROWS_INSERTED', 'INNODB_ROWS_UPDATED', 'INNODB_ROWS_DELETED',
			 'INNODB_ROW_LOCK_WAITS', 'INNODB_DEADLOCKS', 'INNODB_DATA_FSYNCS', 'HANDLER_READ_KEY', 'HANDLER_READ_NEXT', 'HANDLER_READ_RND_NEXT', 'HANDLER_WRITE', 'HANDLER_UPDATE', 'HANDLER_DELETE', 'COM_STMT_EXECUTE', 'THREADS_CREATED', 'ROCKSDB_ROWS_READ', 'ROCKSDB_ROWS_INSERTED',
			 'ROCKSDB_ROWS_UPDATED', 'ROCKSDB_ROWS_DELETED', 'ROCKSDB_ROW_LOCK_WAIT_TIMEOUTS', 'ROCKSDB_ROW_LOCK_DEADLOCKS',
			 'ROCKSDB_WAL_SYNCED', 'ROCKSDB_COMPACT_WRITE_BYTES', 'ROCKSDB_BYTES_WRITTEN')" | awk '{ print $1, $2 }'
		return
	fi
	if [ "$DB_KIND" = cockroach ]; then
		podman exec "$CONTAINER" ./cockroach sql --insecure --host=127.0.0.1:26257 --format=tsv -e \
			"SET allow_unsafe_internals = true; SELECT name, value::INT FROM crdb_internal.node_metrics
			 WHERE name IN ('sql.txn.commit.count', 'sql.txn.rollback.count', 'sql.txn.abort.count', 'sql.query.count',
			 'sql.select.count', 'sql.insert.count', 'sql.update.count', 'sql.delete.count', 'txn.commits') ORDER BY name" |
			awk -F'\t' 'NF == 2 && $1 != "name" { print $1, $2 }'
		return
	fi
	"${PSQL[@]}" -qAt -F' ' -c \
		"SELECT xact_commit, xact_rollback, tup_returned, tup_fetched, tup_inserted, tup_updated, tup_deleted, blks_read, blks_hit, deadlocks
		 FROM pg_stat_database WHERE datname = '$STAT_DATABASE'" | awk '{
		split("commits rollbacks rows_returned rows_fetched rows_inserted rows_updated rows_deleted blocks_read blocks_hit deadlocks", names, " ")
		for (i = 1; i <= 10; i++) print names[i], $i }'
}

postgres_cgroup() {
	echo "/sys/fs/cgroup$(cut -d: -f3 /proc/"$(podman inspect "$CONTAINER" --format '{{.State.Pid}}')"/cgroup)"
}

# snapshot DIR: CPU ticks (1/100 s) per process and PostgreSQL's database counters.
snapshot() {
	local dir=$1
	mkdir -p "$dir"
	date +%s >"$dir/time"
	echo "$DB_KIND $(awk '/^usage_usec/ {printf "%d", $2 / 10000}' "$(postgres_cgroup)/cpu.stat")" >"$dir/cpu"
	echo "backend $(awk '{print $14 + $15}' /proc/"$(pgrep -f "tmp/loadtest/$BACKEND_BINARY\$" | head -1)"/stat)" >>"$dir/cpu"
	database_counters | sort >"$dir/pg_stat"
}

DB_KIND=$DB_KIND SKIP_BUILD=1 seeds/loadtest/run_postgres.sh run --companies-per-stage=10 \
	--max-companies=10 --stage-seconds=180 --think-ms=200 "$@" >"$OUT/run.out" 2>&1 &
RUN_PID=$!
sleep 5
until [ "$(grep -c 'starts selling' "$RUN/live.log" 2>/dev/null)" -ge 10 ]; do
	if ! kill -0 $RUN_PID 2>/dev/null || grep -q FAILED "$RUN/live.log"; then echo "$NAME: run failed"; break; fi
	sleep 2
done
sleep 30
snapshot "$OUT/a"
sleep 90
snapshot "$OUT/b"
wait $RUN_PID
cp "$RUN/live.log" "$RUN/load.log" "$OUT/"

SECONDS_BETWEEN=$(( $(cat "$OUT/b/time") - $(cat "$OUT/a/time") ))
# Mean sales/s over the status lines (one every 5 s) stamped inside the window.
SALES_PER_SECOND=$(awk -v from="$(date -d @"$(cat "$OUT/a/time")" +%T)" -v to="$(date -d @"$(cat "$OUT/b/time")" +%T)" '
	$1 >= from && $1 <= to && /sales\/s/ { for (i = 1; i < NF; i++) if ($i == "sales/s") { sum += $(i + 1); n++ } }
	END { printf "%.2f", n ? sum / n : 0 }' "$OUT/live.log")
SALES_IN_WINDOW=$(awk -v rate="$SALES_PER_SECOND" -v seconds="$SECONDS_BETWEEN" 'BEGIN { print rate * seconds }')
{
	echo "window ${SECONDS_BETWEEN}s, ${SALES_PER_SECOND} sales/s"
	join "$OUT/a/cpu" "$OUT/b/cpu" | awk -v seconds="$SECONDS_BETWEEN" -v rate="$SALES_PER_SECOND" '{
		cpu = ($3 - $2) / 100
		per_sale = rate > 0 ? cpu * 1000 / (rate * seconds) : 0
		printf "%-9s %6.1f%% CPU  %7.2f ms CPU per sale\n", $1, cpu / seconds * 100, per_sale }'
	echo
	echo "== database counters per sale"
	join "$OUT/a/pg_stat" "$OUT/b/pg_stat" | awk -v sales="$SALES_IN_WINDOW" '{ printf "%-24s %10.2f\n", $1, ($3 - $2) / sales }'
} >"$OUT/metrics-diff.txt"
echo "$NAME done"
cat "$OUT/metrics-diff.txt"
