#!/usr/bin/env bash
# measure_oceanbase.sh NAME [genix-seed loadtest args...]: one steady-state run at 10 companies on the
# homelab's OceanBase, as /tmp/v2bench/measure.sh does on ScyllaDB: a 90 s window starting 30 s after
# the 10th company starts selling. Each end of the window snapshots the observer's and the backend's
# CPU, the sales stored, and the plan cache (executions and CPU per statement); ob_metrics_diff.py
# turns the two into per-sale numbers. Run from the directory run_oceanbase.sh runs from.
#
# WARMUP_SECONDS (30), WINDOW_SECONDS (90) and RUN_ARGS (the 10-company steady state) override the
# shape; bench_sales_oceanbase.sh uses them for the short sales-only benchmark.
set -u
WARMUP_SECONDS=${WARMUP_SECONDS:-30}
WINDOW_SECONDS=${WINDOW_SECONDS:-90}
RUN_ARGS=${RUN_ARGS:---companies-per-stage=10 --max-companies=10 --stage-seconds=180 --think-ms=200}
NAME=$1; shift
OUT="$(pwd)/measure/$NAME"
RUN="$(pwd)/run"
DATABASE=genix_load
mkdir -p "$OUT"

OB_USER=${OB_USER:-root@test}
DB_PROCESS=${DB_PROCESS:-observer}

sql() {
	mariadb -h127.0.0.1 -P2881 -u"$OB_USER" --skip-ssl -N -B -e "$1"
}

# snapshot DIR: CPU ticks (1/100 s) of the observer and the backend, sales stored, plan cache rows.
snapshot() {
	local dir=$1
	mkdir -p "$dir"
	date +%s >"$dir/time"
	# Labelled "observer" whatever the engine, so the diff reads the same.
	echo "observer $(awk '{print $14 + $15}' /proc/"$(pgrep -x "$DB_PROCESS" | head -1)"/stat)" >"$dir/cpu"
	echo "backend $(awk '{print $14 + $15}' /proc/"$(pgrep -f "^$(pwd)/backend$" | head -1)"/stat)" >>"$dir/cpu"
	sql "SELECT COUNT(*) FROM $DATABASE.sale_order" >"$dir/sales"
	sql "SELECT sql_id, plan_id, executions, cpu_time, elapsed_time,
		REPLACE(REPLACE(SUBSTR(query_sql, 1, 300), '\n', ' '), '\t', ' ')
		FROM oceanbase.V\$OB_PLAN_CACHE_PLAN_STAT" >"$dir/plans.tsv"
}

# shellcheck disable=SC2086 # RUN_ARGS is a list of arguments.
seeds/loadtest/run_oceanbase.sh run $RUN_ARGS "$@" >"$OUT/run.out" 2>&1 &
RUN_PID=$!
sleep 5
until [ "$(grep -c 'starts selling' "$RUN/live.log" 2>/dev/null)" -ge 10 ]; do
	if ! kill -0 $RUN_PID 2>/dev/null || grep -q FAILED "$RUN/live.log"; then echo "$NAME: run failed"; break; fi
	sleep 2
done
sleep "$WARMUP_SECONDS"
snapshot "$OUT/a"
sleep "$WINDOW_SECONDS"
snapshot "$OUT/b"
wait $RUN_PID
cp "$RUN/live.log" "$RUN/load.log" "$OUT/"
python3 seeds/loadtest/ob_metrics_diff.py "$OUT/a" "$OUT/b" "$DATABASE" >"$OUT/metrics-diff.txt"
echo "$NAME done"
cat "$OUT/metrics-diff.txt" | head -40
