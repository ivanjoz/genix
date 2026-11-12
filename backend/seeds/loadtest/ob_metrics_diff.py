"""Diffs two OceanBase snapshots of measure_oceanbase.sh into per-sale numbers.

    python3 ob_metrics_diff.py SNAPSHOT_A SNAPSHOT_B DATABASE

Per sale stored in the window: observer and backend CPU, and per (statement kind, table) the
executions and the server CPU of the plan cache. A plan evicted between the snapshots counts from 0,
so its executions are a lower bound.
"""

import re
import sys
from collections import defaultdict


def read_snapshot(directory):
    with open(f"{directory}/time") as time_file:
        moment = int(time_file.read().split()[0])
    cpu_ticks = {}
    with open(f"{directory}/cpu") as cpu_file:
        for line in cpu_file:
            name, ticks = line.split()
            cpu_ticks[name] = int(ticks)
    with open(f"{directory}/sales") as sales_file:
        sales = int(sales_file.read().split()[0])
    plans = {}
    # A statement sent with inlined arguments keeps its binary literals in the sample SQL.
    with open(f"{directory}/plans.tsv", errors="replace") as plans_file:
        for line in plans_file:
            fields = line.rstrip("\n").split("\t")
            if len(fields) < 6:
                continue
            sql_id, plan_id, executions, cpu_time, elapsed_time, query = fields[:6]
            plans[(sql_id, plan_id)] = (int(executions), int(cpu_time), int(elapsed_time), query)
    return moment, cpu_ticks, sales, plans


def statement_group(query, database):
    """(kind, table): the statement's first word and the first table of the database it names."""
    kind = (query.split() or ["?"])[0].upper()
    table_match = re.search(rf"`?{database}`?\.`?(\w+)`?", query)
    return kind, table_match.group(1) if table_match else "(other)"


def main():
    snapshot_a, snapshot_b, database = sys.argv[1], sys.argv[2], sys.argv[3]
    time_a, cpu_a, sales_a, plans_a = read_snapshot(snapshot_a)
    time_b, cpu_b, sales_b, plans_b = read_snapshot(snapshot_b)
    seconds, sales = time_b - time_a, sales_b - sales_a
    if sales <= 0:
        print(f"no sales stored in the window ({seconds} s)")
        return
    print(f"window {seconds} s, {sales} sales stored, {sales / seconds:.1f} sales/s")
    print()
    print("== CPU per sale (ms) and cores used")
    for name in sorted(cpu_b):
        cpu_ms = (cpu_b[name] - cpu_a.get(name, 0)) * 10
        print(f"{name:<10} {cpu_ms / sales:8.2f} ms/sale   {cpu_ms / 1000 / seconds:6.2f} cores")

    totals = defaultdict(lambda: [0, 0, 0])
    for plan_key, (executions_b, cpu_time_b, elapsed_b, query) in plans_b.items():
        executions_a, cpu_time_a, elapsed_a, _ = plans_a.get(plan_key, (0, 0, 0, ""))
        executions = executions_b - executions_a
        if executions <= 0:
            continue
        group_total = totals[statement_group(query, database)]
        group_total[0] += executions
        group_total[1] += cpu_time_b - cpu_time_a
        group_total[2] += elapsed_b - elapsed_a

    print()
    print("== statements per sale, by kind and table (plan cache; server CPU and elapsed in µs per sale)")
    print(f"{'kind':<8} {'table':<40} {'stmts/sale':>10} {'cpu µs':>9} {'elapsed µs':>11}")
    all_executions = all_cpu = 0
    for (kind, table), (executions, cpu_time, elapsed) in sorted(totals.items(), key=lambda item: -item[1][0]):
        if table == "(other)" and kind not in ("SELECT", "INSERT", "UPDATE", "DELETE"):
            continue
        all_executions += executions
        all_cpu += cpu_time
        print(f"{kind:<8} {table:<40} {executions / sales:10.2f} {cpu_time / sales:9.0f} {elapsed / sales:11.0f}")
    print(f"{'total':<49} {all_executions / sales:10.2f} {all_cpu / sales:9.0f}")


if __name__ == "__main__":
    main()
