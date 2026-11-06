#!/usr/bin/env bash
# Diffs two ScyllaDB /metrics snapshots taken during a load test, to see where the database's work
# goes: writes and reads per table, CPU per scheduling group, cache and memory.
#
#   scylla_metrics_diff.sh metrics-a.txt metrics-b.txt SECONDS_BETWEEN
# No pipefail: head closing the pipe early is expected.
set -eu
A=$1
B=$2
SECONDS_BETWEEN=$3

# delta prints "<label> <per-second rate>" for every series of a counter, summed over shards.
delta() {
	local metric=$1 label=$2
	awk -v metric="$metric" -v label="$label" -v seconds="$SECONDS_BETWEEN" '
		FNR == 1 { file++ }
		$0 ~ "^" metric "\\{" {
			match($0, label "=\"[^\"]*\""); key = substr($0, RSTART + length(label) + 2, RLENGTH - length(label) - 3)
			value[file, key] += $NF; keys[key] = 1
		}
		END { for (key in keys) { rate = (value[2, key] - value[1, key]) / seconds; if (rate > 0) printf "%-55s %10.1f\n", key, rate } }
	' "$A" "$B" | sort -k2 -nr
}

echo "== writes/s per table (base tables and their views)"
delta scylla_column_family_write_latency_count cf | head -40
echo
echo "== reads/s per table"
delta scylla_column_family_read_latency_count cf | head -25
echo
echo "== CPU ms/s per scheduling group (1000 = one full core)"
delta scylla_scheduler_runtime_ms group
echo
echo "== CQL requests/s by kind"
delta scylla_transport_cql_requests_count kind 2>/dev/null || true
echo
echo "== memory now"
grep -E '^scylla_(lsa_total_space_bytes|memory_allocated_memory|memory_free_memory|cache_bytes_used|cache_bytes_total)\{' "$B" |
	awk '{ split($1, parts, "{"); sum[parts[1]] += $NF } END { for (name in sum) printf "%-40s %8.0f MB\n", name, sum[name] / 1048576 }'
