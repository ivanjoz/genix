#!/usr/bin/env bash
# Makes the homelab observer's redo-log writes stop waiting for the disk, for the load test only:
# libeatmydata, preloaded into the observer's user unit (oceanbase.service), strips O_SYNC/O_DSYNC
# from open() and turns fsync/fdatasync into no-ops. A commit then costs what it costs on a server SSD
# with power-loss protection, instead of this consumer NVMe's ~6–8 ms flush (btrfs + zstd), and the
# comparison with ScyllaDB's periodic commitlog is fair.
#
# NOT DURABLE while enabled: a power cut or a host crash can lose committed transactions or leave the
# store unrecoverable. Disable it once the benchmarks are done.
#
# Needs the libeatmydata package (sudo dnf install libeatmydata); the unit is the user's, so the rest
# runs without sudo.
#
#   seeds/loadtest/ob_fast_redo.sh enable     add the drop-in, restart the observer
#   seeds/loadtest/ob_fast_redo.sh disable    remove the drop-in, restart the observer
#   seeds/loadtest/ob_fast_redo.sh status     is the observer preloaded?
set -euo pipefail
DROP_IN_DIR="$HOME/.config/systemd/user/oceanbase.service.d"
DROP_IN="$DROP_IN_DIR/eatmydata.conf"

# wait_for_observer waits until the tenant answers SQL again after a restart.
wait_for_observer() {
	until mariadb -h127.0.0.1 -P2881 -uroot@test --skip-ssl -N -e "SELECT 1" >/dev/null 2>&1; do sleep 2; done
}

case "${1:-}" in
enable)
	LIBRARY=$(rpm -ql libeatmydata | grep -m1 '/libeatmydata\.so')
	mkdir -p "$DROP_IN_DIR"
	printf '[Service]\nEnvironment=LD_PRELOAD=%s\n' "$LIBRARY" >"$DROP_IN"
	systemctl --user daemon-reload
	systemctl --user restart oceanbase.service
	wait_for_observer
	echo "enabled: $DROP_IN"
	;;
disable)
	rm -f "$DROP_IN"
	systemctl --user daemon-reload
	systemctl --user restart oceanbase.service
	wait_for_observer
	echo "disabled"
	;;
status)
	if tr '\0' '\n' </proc/"$(pgrep -x observer | head -1)"/environ | grep -q '^LD_PRELOAD=.*eatmydata'; then
		echo "observer preloads libeatmydata (redo not durable)"
	else
		echo "observer syncs normally"
	fi
	;;
*)
	echo "usage: ob_fast_redo.sh enable|disable|status" >&2
	exit 1
	;;
esac
