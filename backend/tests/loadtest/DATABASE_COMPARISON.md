# Sales-flow benchmark: every database compared (2026-11-06/07)

The same Go backend and the same load generator (`fn-load-test`) sell against each database through
its ORM driver. 10 companies × 4 cashiers sell from second 0, with fareward bypassed. The steady
state is measured over a 90 s window, 30 s after the 10th company starts selling. Only database and
backend CPU count; the load generator runs on separate CPUs.

The results come from **two different machines**, so only compare numbers inside the same table.

- **Local, 2 vCPU VPS budget:** ScyllaDB, PostgreSQL, YugabyteDB, CockroachDB, MariaDB (InnoDB and
  MyRocks). The database and the
  backend share CPUs 0–1, with 1.5 GB each. The model is V2 (`--sales-v2 --sales-only`).
- **Homelab, 8 cores:** OceanBase, seekdb, ScyllaDB. Ryzen 7 5700U, 16 threads. OceanBase and seekdb
  run model V3, ScyllaDB V1 or V2. See `plan.md`, sections V3 and later.

Durability is relaxed the same way everywhere, to match ScyllaDB's periodic commitlog: a commit never
waits for its fsync.

| Database | Setting |
|---|---|
| ScyllaDB | periodic commitlog (default) |
| PostgreSQL | `synchronous_commit=off` |
| YugabyteDB | WAL not fsynced per write (default) |
| CockroachDB | `kv.raft_log.synchronization.unsafe.disabled = true` |
| OceanBase, seekdb | redo through libeatmydata / `nosync_preload.so` |
| MariaDB InnoDB | `innodb_flush_log_at_trx_commit=2` |
| MariaDB MyRocks | `rocksdb_flush_log_at_trx_commit=2` |

## 1. Local, 2 vCPU: ScyllaDB vs PostgreSQL vs YugabyteDB vs CockroachDB

V2 model, sales-only, `--settle-later=30`. "Think 200" is 200 ms between a cashier's sales; at 40
cashiers that caps the test near 190 sales/s. "Saturation" is `--think-ms=0`. Each cell shows two
runs.

| | ScyllaDB 2026 | PostgreSQL 18 | YugabyteDB 2026.1 (YSQL) | CockroachDB 26.3 |
|---|---|---|---|---|
| **sales/s, think 200** | 126.2 / 126.5 | **191.2 / 191.7** (the test's cap) | 30.2 / 33.3 | 23.0 / pending |
| **sales/s, saturation** | 132.0 / 145.2 | **248.9 / 252.7** | 31.5 / 27.2 | 22.4 / pending |
| DB CPU per sale, think 200 | 7.88 / 8.01 ms | **2.71 / 2.71 ms** | 56.0 / 52.3 ms | 77.9 ms |
| DB CPU per sale, saturation | 8.07 / 7.69 ms | **3.64 / 3.65 ms** | 52.9 / 62.0 ms | 79.8 ms |
| Backend CPU per sale, saturation | 4.26 / 4.10 ms | 3.51 / 3.46 ms | 5.21 / 6.41 ms | 6.33 ms |
| DB CPU used, saturation (of 200%) | ~107–112% (one shard hot) | ~91% | ~167–169% | 179% |
| Sale p99, think 200 (whole stage) | 538 / 620 ms | **43 / 39 ms** | 3.7 / 3.7 s | 3.7 s |
| Sale p99, saturation (whole stage) | 588 / 789 ms | **256 / 246 ms** | 4.0 / 4.5 s | 3.4 s |
| Errors | 0.09% / 0 | 0 | 0 | 0 |
| Transactions per sale | — | 22.0 | 22.2 | 22.3 |

Raw files are in `backend/measure/`:

| Database | Think 200 | Saturation |
|---|---|---|
| ScyllaDB | `v2-views-{1,2}` | `scylla-sat-{1,2}` |
| PostgreSQL | `pg-v2-{1,2}` | `pg-sat-{1,2}` |
| YugabyteDB | `yb-v2-{1,2}` | `yb-sat-{1,2}` |
| CockroachDB | `crdb-v2-{1,2}` | `crdb-sat-{1,2}` |

### Reading

- **PostgreSQL wins this machine by a wide margin:** about 1.8× ScyllaDB's sales/s at less than half
  the database CPU per sale, with a p99 a third of ScyllaDB's.
  - The data fits its 384 MB buffer pool, so a B-tree update touches memory and appends to the WAL.
  - A sale is about 22 local transactions. Index maintenance is in-process, with no extra read.
  - Its per-connection processes spread evenly over both cores.
- **ScyllaDB pays per mutation and per shard:**
  - Each table a sale touches is a separate CQL mutation with its own request path.
  - Views and indexes do a read before each write; the two V2 detail views cost about 0.66 ms per
    sale, ~9%.
  - With tablets on one node, each table has one tablet, so it lives on one shard. One shard runs at
    55–69% busy while the other sits at 30–55%, and compaction runs on the hot shard. When that
    shard saturates, latency jumps.
  - Its LSM advantage (sequential writes on data larger than RAM) does not apply at this data size.
- **YugabyteDB and CockroachDB are CPU-bound at about 25–33 sales/s:**
  - Both are LSM underneath, like ScyllaDB.
  - Each write goes through a distributed-transaction layer: a provisional intent, then its
    resolution.
  - Each write is also a Raft log write plus its apply, even on one node, and passes through a full
    SQL layer on top of a key-value layer.
  - That is about 7–10× ScyllaDB's CPU per sale and 15–25× PostgreSQL's.
  - YugabyteDB splits its CPU between the storage server (`yb-tserver`, ~85%) and ~32 PostgreSQL-layer
    backends (~60%). Its master takes about 10%.
  - CockroachDB sagged from about 30 to 8–15 sales/s mid-run, probably background statistics,
    range splits and compaction. Not verified.
  - Both pay this cost to offer multi-node serializable SQL. On one node they get nothing back for
    it.

## 1b. Local, 2 vCPU: MariaDB (InnoDB, MyRocks) against PostgreSQL

Same machine, same setup as section 1. MariaDB 13.0.2 runs through `mariadbdriver`. The step-by-step
InnoDB tuning is in `MARIADB_TUNING.md`. Durability: InnoDB `innodb_flush_log_at_trx_commit=2`,
MyRocks `rocksdb_flush_log_at_trx_commit=2`.

| | PostgreSQL 18 | PostgreSQL 18, tuned | MariaDB InnoDB, first port | MariaDB InnoDB, tuned | MariaDB MyRocks |
|---|---|---|---|---|---|
| **sales/s, think 200** | 191.2 / 191.7 (cap) | 193.6 (cap) | 153 | 189.0 (cap) | 132.3 |
| **sales/s, saturation** | 248.9 / 252.7 | **350.4 / 353.1 / 338.7** | 157.0 / 166.8 | 277.8 | — |
| DB CPU per sale, think 200 | 2.71 ms | **2.31 ms** | ~7.5 ms | 3.51 ms | 10.40 ms |
| DB CPU per sale, saturation | 3.64 ms | **2.55–2.79 ms** | 7.87 / 7.49 ms | 3.74 ms | — |
| Backend CPU per sale, saturation | 3.51 ms | **2.72–2.79 ms** | 3.89 / 3.71 ms | 2.90 ms | 3.94 ms (think 200) |
| Sale p99, think 200 (stage) | 43 / 39 ms | **23 ms** | — | 43 ms | 227 ms |
| Sale p99, saturation (stage) | 256 / 246 ms | **171–179 ms** | 441 ms | 198 ms | — |

Runs: `pg-pipe-sat-{1,2,3}`, `pg-pipe-v2-1`, `maria-sat-{1,2}`, `maria-v2-1`, `maria-s5-sat-1`,
`maria-s5-v2-1`, `myrocks-v2-1`.

Both tuned columns use the same driver changes:

- write transactions pipelined with their COMMIT: one multi-statement packet on MariaDB, a pgx batch
  on PostgreSQL;
- inserts of newly minted keys skip the locking pre-read.

The concurrency is bounded differently: MariaDB with its thread pool, PostgreSQL with a backend pool
of 16 connections. See `MARIADB_TUNING.md` and `POSTGRES_TUNING.md`.

- **On equal terms, PostgreSQL leads by about 25% at saturation.** It uses about 30% less database
  CPU per sale and has a lower p99 at both loads.
- Tuned MariaDB beats stock PostgreSQL (278 against 250 sales/s). That gap was the driver, not the
  engine: PostgreSQL gains 35–40% from the same changes plus the smaller pool.
- **What got InnoDB from the first port to tuned:**
  - Driver fixes: isolation set per connection, primary-key lookups, a full-scan DELETE.
  - Batched transactions.
  - The thread pool, about +15%.
  - The pre-read skip.
- **What got PostgreSQL to tuned:**
  - A 16-connection pool instead of 32, +16%: fewer processes on 2 vCPUs.
  - The pipelined driver, a further +17–21%.
  - Server settings (WAL compression, `shared_buffers`) changed nothing.
- **MyRocks** (LSM) costs about 3× InnoDB's CPU per sale. Every read merges memtable and SST levels.
  Its row locks and deadlock detection are an extra layer over RocksDB, and none of its LSM strengths
  (data larger than RAM, compression, write amplification) matter at this data size. It was measured
  with the step 1 driver only; MyRocks was not tuned further.

## 2. Homelab, 8 cores: OceanBase vs seekdb vs ScyllaDB

From `plan.md`. OceanBase CE 5.0.1 has `cpu_count=8` and `memory_limit=8G`. ScyllaDB has `--smp 8`
and 8 GB, on CPUs 0–11 shared with the backend.

### Steady state with think time (`--settle-later=30 --think-ms=200`, 90 s window)

| | OceanBase V3 (lines columnar) | OceanBase V3 (header + lines columnar) | ScyllaDB V1 | ScyllaDB V2 |
|---|---|---|---|---|
| sales/s | 137 | 142 | **154** | 149 |
| Sale p99 (stage) | 267 ms | 307 ms | **208 ms** | 487 ms |
| Errors | 0 | 0 | 0.03% | 0.04% |
| DB CPU per sale | 63.3 ms (8.7 cores, saturated) | 57.5 ms (8.2 cores) | 27.3 ms (4.2 cores) | **23.9 ms** (3.6 cores) |
| Backend CPU per sale | 15.2 ms | 14.9 ms | 17.2 ms | 15.3 ms |
| Statements per sale | 38.1 | 38.1 | 37.7 CQL | 36.5 CQL |

### Sales-only saturation (`--think-ms=0`, 15 s warm-up, 150 s window)

| | sales/s | DB CPU per sale | Backend CPU per sale | Total CPU per sale | Sale p99 | Errors | DB memory |
|---|---|---|---|---|---|---|---|
| OceanBase V3, row tables, tuned | 214.7 | 44.4 ms | 10.2 ms | 54.6 ms | 370 ms | 0 | ~3.7 GB |
| seekdb V3, row tables | **274.0** | 28.2 ms | **9.4 ms** | 37.6 ms | **301 ms** | 0 | < 2 GB |
| ScyllaDB V2 | 218.0 | **18.1 ms** | 13.5 ms | **31.9 ms** | 2.1 s | 0.2% | 7.3 GB (preallocated) |

OceanBase "tuned" means `InterpolateParams`, with SQL audit and perf events off. Those two changes
took it from 188.6 to about 214–224 sales/s.

### Reading

- **On CPU, ScyllaDB beats OceanBase:** about 2.5× less database CPU per sale in both modes.
  - With think time, OceanBase saturates its 8 cores at 137–142 sales/s. ScyllaDB serves 149–154
    sales/s on about 4 cores.
  - About 25 ms of OceanBase's 63 ms per sale is statement execution. The rest is the observer's own
    commit, clog, freeze, compaction and RPC work.
- **At saturation, ScyllaDB's throughput is unstable.** It swings between 110 and 316 sales/s with a
  2.1 s p99, while using only about 3.9 of its 12 cores. The cause was not measured, but it fits the
  hot-shard pattern measured locally. OceanBase stays steady, slowing gently as its tables grow (240
  to 205 sales/s).
- **seekdb** is OceanBase's SQL engine as a single-tenant server. It gets the most sales/s of the
  three (+28% over OceanBase) with 36% less CPU per sale and half the memory. It has no replicas, no
  column groups and no table TTL.
- **Model note:** OceanBase runs V3 (header plus immutable lines, joined in SQL) and ScyllaDB runs V1
  or V2. Each side uses the model its engine suits, so the comparison is per engine, not per table
  layout.

## 3. Overall

| Rank | Small VPS (2 vCPU, data in RAM) | Larger node (8 cores) |
|---|---|---|
| Throughput | PostgreSQL > MariaDB InnoDB ≫ ScyllaDB ≈ MyRocks ≫ YugabyteDB ≈ CockroachDB | seekdb > ScyllaDB ≈ OceanBase (ScyllaDB unstable) |
| CPU per sale | PostgreSQL < MariaDB InnoDB < ScyllaDB < MyRocks ≪ YugabyteDB < CockroachDB | ScyllaDB < seekdb < OceanBase |
| Tail latency | PostgreSQL < MariaDB InnoDB < MyRocks < ScyllaDB ≪ YugabyteDB ≈ CockroachDB | seekdb ≈ OceanBase < ScyllaDB |

- **PostgreSQL** is the best fit for a single small VPS. It was not run on the homelab, so it is not
  compared with OceanBase or seekdb.
- **MariaDB InnoDB**, tuned, comes within about 20% of tuned PostgreSQL (section 1b). Both rankings
  compare the tuned drivers; the other engines ran the original driver.
- **ScyllaDB** is the most CPU-efficient of the scale-out engines. Its single-node problem is
  imbalance across shards, not raw cost.
- **YugabyteDB and CockroachDB** buy multi-node consistent SQL with 7–25× the CPU per sale. On one
  node that cost buys nothing.
- **OceanBase and seekdb** have the steadiest tails at 8 cores, at 1.5–2.5× ScyllaDB's CPU per sale.

## Not covered

- Replication, where every engine except seekdb would add replica writes.
- Data larger than RAM.
- PostgreSQL, YugabyteDB and CockroachDB on the homelab, and OceanBase and seekdb on the 2 vCPU budget.
- Tuning beyond defaults for YugabyteDB and CockroachDB: non-colocated tables, upsert mode, more
  memory.
