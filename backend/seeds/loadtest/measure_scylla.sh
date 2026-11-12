#!/usr/bin/env bash
# measure_scylla.sh NAME [genix-seed loadtest args...]: the ScyllaDB counterpart of measure_oceanbase.sh, run
# from backend/ of a deploy_scylla_homelab.sh layout. Same steady state (10 companies) and the same
# 90 s window starting 30 s after the 10th company starts selling. Each end of the window snapshots
# the CPU of scylla, fareward and the backend and ScyllaDB's /metrics; the sales come from the live
# log's sales/s lines inside the window, so the per-sale numbers are computed the same way as on
# OceanBase without scanning the sale table.
#
# The stack (scylla + fareward + backend) shares STACK_CPUS; run.sh's other overrides apply.
# WARMUP_SECONDS (30), WINDOW_SECONDS (90) and RUN_ARGS (the 10-company steady state) override the
# window and the load, as in measure_oceanbase.sh.
set -u
NAME=$1; shift
WARMUP_SECONDS=${WARMUP_SECONDS:-30}
WINDOW_SECONDS=${WINDOW_SECONDS:-90}
RUN_ARGS=${RUN_ARGS:---companies-per-stage=10 --max-companies=10 --stage-seconds=180 --think-ms=200}
OUT="$(pwd)/measure/$NAME"
RUN="$(dirname "$(pwd)")/tmp/loadtest/run"
mkdir -p "$OUT"

# snapshot DIR: CPU ticks (1/100 s) per process and the database's metrics.
snapshot() {
	local dir=$1
	mkdir -p "$dir"
	date +%s >"$dir/time"
	echo "scylla $(awk '{print $14 + $15}' /proc/"$(pgrep -f '^/usr/bin/scylla .*127\.0\.0\.2' | head -1)"/stat)" >"$dir/cpu"
	echo "fareward $(awk '{print $14 + $15}' /proc/"$(pgrep -f 'target/release/fareward$' | head -1)"/stat)" >>"$dir/cpu"
	echo "backend $(awk '{print $14 + $15}' /proc/"$(pgrep -f 'tmp/loadtest/backend$' | head -1)"/stat)" >>"$dir/cpu"
	curl -s http://127.0.0.2:9180/metrics >"$dir/metrics.txt"
}

# shellcheck disable=SC2086 # RUN_ARGS is a list of arguments.
TABLET_BALANCING=false SKIP_BUILD=1 FAREWARD_BYPASS=1 seeds/loadtest/run.sh run $RUN_ARGS "$@" \
	>"$OUT/run.out" 2>&1 &
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

SECONDS_BETWEEN=$(( $(cat "$OUT/b/time") - $(cat "$OUT/a/time") ))
# Mean sales/s over the status lines (one every 5 s) stamped inside the window.
SALES_PER_SECOND=$(awk -v from="$(date -d @"$(cat "$OUT/a/time")" +%T)" -v to="$(date -d @"$(cat "$OUT/b/time")" +%T)" '
	$1 >= from && $1 <= to && /sales\/s/ { for (i = 1; i < NF; i++) if ($i == "sales/s") { sum += $(i + 1); n++ } }
	END { printf "%.2f", n ? sum / n : 0 }' "$OUT/live.log")
{
	echo "window ${SECONDS_BETWEEN}s, ${SALES_PER_SECOND} sales/s"
	join "$OUT/a/cpu" "$OUT/b/cpu" | awk -v seconds="$SECONDS_BETWEEN" -v rate="$SALES_PER_SECOND" '{
		cpu = ($3 - $2) / 100
		per_sale = rate > 0 ? cpu * 1000 / (rate * seconds) : 0
		printf "%-9s %6.1f%% CPU  %7.2f ms CPU per sale\n", $1, cpu / seconds * 100, per_sale }'
	echo
	seeds/loadtest/scylla_metrics_diff.sh "$OUT/a/metrics.txt" "$OUT/b/metrics.txt" "$SECONDS_BETWEEN"
} >"$OUT/metrics-diff.txt"
echo "$NAME done"
head -60 "$OUT/metrics-diff.txt"
