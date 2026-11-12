#!/usr/bin/env bash
# seekdb (github.com/oceanbase/seekdb: the OceanBase SQL engine as one single-tenant server) in place
# of the homelab's observer, for the same benchmarks. It listens on the observer's port 2881, so the
# observer is stopped first. Run from the directory run_oceanbase.sh runs from.
#
#   seeds/loadtest/run_seekdb.sh start    stop the observer, start seekdb (8 CPUs, 8 GB, fast redo)
#   seeds/loadtest/run_seekdb.sh stop     remove seekdb (its data volume stays), start the observer
#
# Then the OceanBase scripts with seekdb's user and process:
#   OB_USER=root DB_PROCESS=seekdb seeds/loadtest/bench_sales_oceanbase.sh NAME
#
# The redo goes through nosync/nosync_preload.so (built from nosync/nosync_preload.c, NOT durable), as
# the observer's does through libeatmydata during the benchmarks: Fedora's libeatmydata needs a newer
# glibc than the image's Rocky 9. The C source sits in its own folder because cgo refuses a .c file
# next to the loadtest Go package.
set -euo pipefail
SHIM="$(pwd)/seeds/loadtest/nosync/nosync_preload.so"

case "${1:-}" in
start)
	systemctl --user stop oceanbase.service
	podman rm -f genix-seekdb >/dev/null 2>&1 || true
	# :z relabels the shim for SELinux; the same sizes as the observer (cpu_count 8, memory_limit 8G).
	podman run -d --name genix-seekdb --network host -v genix-seekdb-data:/var/lib/seekdb \
		-v "$SHIM:/opt/nosync.so:ro,z" -e LD_PRELOAD=/opt/nosync.so \
		-e CPU_COUNT=8 -e MEMORY_LIMIT=8G -e LOG_DISK_SIZE=12G docker.io/oceanbase/seekdb:latest >/dev/null
	until mariadb -h127.0.0.1 -P2881 -uroot --skip-ssl -N -e "SELECT 1" >/dev/null 2>&1; do sleep 2; done
	# The observer's tuned state (plan.md, "Tuning OceanBase"): no perf events. seekdb has no SQL
	# audit to turn off.
	mariadb -h127.0.0.1 -P2881 -uroot --skip-ssl -e "ALTER SYSTEM SET enable_perf_event = false"
	echo "seekdb is up on 127.0.0.1:2881 (user root)"
	;;
stop)
	podman rm -f genix-seekdb >/dev/null
	systemctl --user start oceanbase.service
	until mariadb -h127.0.0.1 -P2881 -uroot@test --skip-ssl -N -e "SELECT 1" >/dev/null 2>&1; do sleep 2; done
	echo "observer is back"
	;;
*)
	echo "usage: run_seekdb.sh start|stop" >&2
	exit 1
	;;
esac
