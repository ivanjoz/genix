# ScyllaDB tablets vs vnodes — findings for a future review

Status: **open decision.** The keyspaces stay on tablets for now. Measured on 2026-11-05 on the local
benchmark stack (see `backend/tests/bench/README.md`).

## Setup measured

- One ScyllaDB node (`scylladb/scylla:latest`, 2026.3), `--smp 2`, developer mode, container
  `genix-bench-scylla`.
- Keyspace `genix_bench`, created like production: `CreateKeyspaceIfNotExists`
  (`genix-orm/scylla/init.go`) and `scripts/configure/configure_db.py` don't set a `tablets` option,
  so the keyspace gets **tablets by default**.
- About 244 tables plus materialized views per keyspace.
- Benchmark: `BenchmarkRouter/first/...` at 16 concurrent requests, colbin + zstd responses.
- Datasets:
  - single company: `10k` (company 102) and `50k` (company 103);
  - `multi8x10k`: eight companies of 10k products each, requests spread round-robin across them.
    These are `bench.MultiCompanyDatasets`, IDs `201–204, 210–212, 214`.

## 1. With the defaults, every table has one tablet

ScyllaDB computes the tablet count of each table, then enforces `tablets_per_shard_goal` (default
100) **as the last step**. That last step scales every table down until the goal is met, and it
overrides per-table options such as `min_tablet_count` and `min_per_shard_tablet_count`. With ~244
tables on 2 shards, every table ends up with **1 tablet**, so each table is served by **one shard**.

Effect seen while benchmarking: one shard at 100% CPU with 55–115 reads queued, the other shard idle,
no matter how many companies were in the test.

## 2. Measured throughput (multi-company first sync, req/s)

| Endpoint | Tablets (default) | Tablets, goal raised to 2000 | Vnodes |
|---|---|---|---|
| products | 19.8 | ~22 | 34.9 |
| products-stock | 48.5 | — | 74.9 |
| sale-orders | ~102 | ~101 | 137 |

- Vnodes = a copy of the keyspace created with `tablets = {'enabled': false}` (`genix_bench_vnodes`).
- Raising the goal: `UPDATE system.config SET value='2000' WHERE name='tablets_per_shard_goal'`.
  This is a live setting that resets when the node restarts. ScyllaDB's docs recommend setting it
  cluster-wide with `ALTER CLUSTER WITH tablets_per_shard_goal = ...`.
- Keyspace option `'initial'` is deprecated and ignored.

**Bias in this comparison:** the eight multi-company IDs were picked so they split evenly between the
two shards *under the vnodes token-to-shard formula*
(`shard = ((token + 2^63) << 12) * smp >> 64`; see `/tmp/footprint/shard_of.py` in that session).
That favours vnodes.

## 3. Why raising the goal did not help

### The delta views are co-located with their base table

An external write-up (`plan/scylladb_diagnostic.md`) assumed the delta views have their own
partition key, so they would have their own tablet map that never splits. **That is wrong for
Genix:** the views keep the base table's partition key. For example:

```
products__dlt_status_view   PRIMARY KEY (company_id, zz_dlt_status, id)
products                    partition key: company_id
```

A view with the same partition key as its base table is a *co-located table*: it has no tablet map
of its own (`system.tablets.tablet_count` is `null` for it) and splits together with the base table.
Setting `tablets = {'min_per_shard_tablet_count': ...}` on a view therefore changes nothing.

### The split tablets were placed unevenly

With the goal at 2000, `products` split into 8 tablets. The load balancer evens out **disk usage
across all tables combined**, not the load of one table, and placed them like this:

| Shard | `products` tablets | Bench companies whose token falls there |
|---|---|---|
| 0 | 6 | 214, 201, 210, 211, 204, 212, 203 |
| 1 | 2 | 202 |

Seven of the eight benchmark companies were on shard 0, so the load stayed on one core.

Commands used:

```cql
SELECT table_name, tablet_count FROM system.tablets WHERE keyspace_name='genix_bench' ALLOW FILTERING;
SELECT last_token, replicas FROM system.tablets
  WHERE keyspace_name='genix_bench' AND table_name='products' ALLOW FILTERING;
SELECT DISTINCT company_id, token(company_id) FROM genix_bench.products;
```

## 4. The basic limit: one company is one partition

Every delta read (first sync and incremental) reads **one partition: one company**. One partition is
always served by one shard, with tablets or with vnodes. So:

- On one node, tablets can at best **match** vnodes for this access pattern; they can't beat it.
- A single company's first sync is bound to one core in both modes.
- With many real companies, both modes spread the load across shards about equally on average.
  Tablets can still be unlucky per table, as in section 3.
- Tablets pay off on a **multi-node cluster**: fast node add/remove, rebalancing, and splitting big
  tables by size. That isn't what limits a single 4–8 GB VPS.

## 5. Other things tested

- **Shard-aware routing** (`DisableInitialHostLookup = false` in the gocql connection): with the
  default `true`, ~49% of operations cross shards. Turning it off was 13–19% *slower* with 1 tablet
  and within noise with vnodes, so it stays `true`.
- **VPS snapshot cache** (memory 256 MB + disk 1 GB, `[file_store].vps_snapshot_*`): makes repeated
  first syncs 4–8× faster on tablets. For example, 10k products 21.8 → 177.9 req/s and multi-company
  products 24.5 → 173.1 req/s. That is well past the vnodes gain, so first syncs are no longer the
  reason to switch.

## 6. Options for the review

1. **Keep tablets** (current choice). The snapshot cache covers first syncs. Revisit if a profile
   shows one shard saturated in production.
2. **Experiment: balance `products` tablets by hand** (4 per shard, through the REST tablet-move
   API) and re-run the multi-company benchmark. This shows whether a balanced layout reaches the
   vnodes numbers. It is not a fix: the balancer may move them back, and no setting keeps one table
   balanced across shards. Also worth trying: `force_capacity_based_balancing: true` (live setting,
   documented for object-storage keyspaces; it treats every tablet as the same size).
3. **Switch to vnodes.** A keyspace can't be altered between tablets and vnodes: drop it, recreate
   it with `tablets = {'enabled': false}` in `CreateKeyspaceIfNotExists` and
   `configure_db.py`, then reseed. Pre-alpha, so a reseed is acceptable. Trade-off: gives up the
   tablets benefits on a future multi-node cluster.

Before the next measurement, re-pick the multi-company IDs so they are balanced under the layout
being tested, or use enough companies (e.g. 32+) that placement averages out.

## Leftovers in the bench container

- Extra keyspaces `genix_bench_vnodes` and `genix_bench_tab8` (with `tablet_probe_*` tables).
- `tablets = {'min_per_shard_tablet_count': '4'}` set on `genix_bench.products` and
  `products__dlt_status_view`; it has no effect.
- `tablets_per_shard_goal = 2000` set live; it resets when the container restarts.

## Sources

- Data distribution with tablets: https://docs.scylladb.com/manual/master/architecture/tablets.md
- Per-table tablet options (DDL): https://docs.scylladb.com/manual/master/cql/ddl.md
- Glossary, co-located table: https://docs.scylladb.com/manual/master/reference/glossary.md
- Admin guide, `force_capacity_based_balancing`:
  https://docs.scylladb.com/manual/master/operating-scylla/admin.md
- Migrating a keyspace from vnodes to tablets:
  https://docs.scylladb.com/manual/branch-2026.4/operating-scylla/procedures/config-change/migrate-vnodes-to-tablets.md
