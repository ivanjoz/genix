#!/usr/bin/env bash
# bench_sales_oceanbase.sh NAME [fn-load-test args...]: the short benchmark for tuning OceanBase.
# Only sales and their restocking (--sales-only), no think time, so the cashiers saturate the stack: 10
# companies × CASHIERS (default 4) concurrent sellers, a 15 s warm-up and a 45 s
# window, about two minutes in all. Prints sales/s, CPU per sale and statements per sale
# (measure_oceanbase.sh). Run from the directory run_oceanbase.sh runs from.
set -u
CASHIERS=${CASHIERS:-4}
WARMUP_SECONDS=15 WINDOW_SECONDS=45 \
	RUN_ARGS="--companies-per-stage=10 --max-companies=10 --stage-seconds=75 --think-ms=0 --cashiers=$CASHIERS --sales-only --max-p99-ms=60000 --max-error-pct=100" \
	exec tests/loadtest/measure_oceanbase.sh "$@"
