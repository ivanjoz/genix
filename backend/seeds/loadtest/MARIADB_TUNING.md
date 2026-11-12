# MariaDB (InnoDB) tuning toward PostgreSQL (2026-11-07)

Goal: the V2 sales-only benchmark on the 2 vCPU budget (`run_postgres.sh` / `measure_postgres.sh`,
`DB_KIND=mariadb`) at PostgreSQL's level. MariaDB 13.0.2 is the newest stable (non-LTS) release; 13.1
is only a release candidate.

| Target (PostgreSQL 18) | think 200 | saturation |
|---|---|---|
| sales/s | 191 (the test's cap) | 249–253 |
| DB CPU per sale | 2.71 ms | 3.64 ms |
| backend CPU per sale | 3.0 ms | 3.5 ms |
| sale p99 (stage) | 39–43 ms | 246–256 ms |

On 2 shared vCPUs, saturation throughput ≈ 2000 ms / (DB + backend CPU per sale). Matching
PostgreSQL means about 4.5 ms of DB CPU per sale or less.

## Steps

Each step keeps the previous ones. Runs are 10 companies, `--sales-v2 --sales-only --settle-later=30`,
90 s window (raw files `backend/measure/<run>`).

| step | change | run | sales/s | DB CPU per sale | backend CPU per sale | p99 (stage) |
|---|---|---|---|---|---|---|
| 0 | first port (isolation per transaction, IN lists rewritten as joins) | maria-sat-1 / -2 | 157.0 / 166.8 | 7.87 / 7.49 ms | 3.89 / 3.71 ms | 441 ms / — |
| 1 | isolation set per connection; key lists kept as key lookups, `FORCE INDEX (PRIMARY)` | maria-fixed-sat-1 / -2 | 208.5 / 221.0 | 5.70 / 5.45 ms | 3.17 / 3.01 ms | 259 / 240 ms |
| 1 | same, think 200 | maria-fixed-v2-1 / -2 | 160.3 / 170.5 | 7.22 / 6.03 ms | 3.75 / 3.25 ms | 139 / 105 ms |
| 2 | a one-record element-index DELETE written as equalities (below) | maria-fix2-sat-1 | 216.1 | 4.95 ms | 3.56 ms | 288 ms |
| 2 | same, think 200 | maria-fix2-v2-1 | 181.0 | 4.78 ms | 3.52 ms | 87 ms |
| 3 | write transactions batched: autocommit-off pool, queued writes + COMMIT in one multi-statement packet; IDs reserved by one upsert | maria-s3-sat-1 | 236.2 | 4.54 ms | 3.31 ms | 242 ms |
| 3 | same, think 200 | maria-s3-v2-1 | 183.5 | 4.33 ms | 3.21 ms | 101 ms |
| 4a | `thread_handling=pool-of-threads`, `thread_pool_size=2` (one group per vCPU) | maria-s4a-sat-1 | 268.0 | 3.88 ms | 2.93 ms | 204 ms |
| 4b | + `innodb_adaptive_hash_index=ON`: no gain, not kept | maria-s4b-sat-1 | 269.3 | 3.87 ms | 2.96 ms | 205 ms |
| 4c | + `innodb_flush_log_at_trx_commit=0`: +2%, within noise; not kept (2 survives a server crash) | maria-s4c-sat-1 | 274.6 | 3.77 ms | 2.94 ms | 202 ms |
| 5 | 4a + an insert whose every key was just minted skips the locking pre-read (nothing can be stored under a new key): 19.1 → 16.8 SELECT per sale | maria-s5-sat-1 | 277.8 | 3.74 ms | 2.90 ms | 198 ms |
| 5 | same, think 200 | maria-s5-v2-1 | 189.0 | 3.51 ms | 2.85 ms | 43 ms |
| 6 | + `optimizer_use_condition_selectivity=1`, `use_stat_tables=NEVER`: no gain, not kept | maria-s6-sat-1 | 278.5 | 3.71 ms | 2.84 ms | 202 ms |
| 6 | same, think 200 | maria-s6-v2-1 | 189.3 | 3.52 ms | 2.92 ms | 40 ms |

## Result

Step 5 is the tuned configuration: the driver of steps 1–3 and 5, the thread pool (now the
`run_postgres.sh stack` default), everything else at MariaDB's defaults.

| | PostgreSQL 18 | MariaDB 13.0.2 InnoDB, tuned | MariaDB, first port |
|---|---|---|---|
| sales/s, saturation | 248.9 / 252.7 | **277.8** | 157.0 / 166.8 |
| DB CPU per sale, saturation | **3.64 ms** | 3.74 ms | 7.87 / 7.49 ms |
| backend CPU per sale, saturation | 3.51 ms | **2.90 ms** | 3.89 / 3.71 ms |
| sale p99, saturation | 256 / 246 ms | **198 ms** | 441 ms |
| sales/s, think 200 | 191.2 / 191.7 (the cap) | 189.0 (the cap) | 153 |
| DB CPU per sale, think 200 | **2.71 ms** | 3.51 ms | ~7.5 ms |
| sale p99, think 200 | 43 / 39 ms | 43 ms | — |

- **Saturation: MariaDB now beats PostgreSQL by about 10% on sales/s, with a lower p99.** Its database
  CPU per sale is about equal. Its backend CPU per sale is lower, because the batched transactions
  mean fewer round trips for the client to process.
- **Think 200: same throughput (both at the cap) and the same p99.** MariaDB still spends about 30%
  more database CPU per sale. With the thread pool half idle, the cost of waking a pool thread per
  statement is spread over fewer sales.
- Single runs; the run-to-run spread of earlier pairs is about ±3–5%.

### Fairness: what PostgreSQL did not get

- **Batched transactions (step 3) are MariaDB-only.**
  - pgdriver sends one statement per round trip plus BEGIN and COMMIT.
  - pgx could do the same with a `pgx.Batch` (pipelining), probably with a similar gain.
- **The minted-key skip (step 5) is in mariadbdriver only.**
  - pgdriver still pre-reads the rows for inserts of new keys.
  - It is about 2.3 fewer statements per sale and would help pgdriver the same way.
- So PostgreSQL has headroom from the same ORM changes, and is the better per-statement engine at
  low load. The comparison shows MariaDB can reach PostgreSQL's level on this workload; it does not
  show that MariaDB is faster.
- **Confirmed later the same day (`POSTGRES_TUNING.md`):** pgdriver got the same pipelining and the
  minted-key skip, and both databases ran with a backend pool of 16. PostgreSQL then reaches
  338–353 sales/s at 2.55–2.79 ms per sale, against MariaDB's 276–278 at 3.74–3.79 ms. A 16-connection
  pool does not help MariaDB (`maria-pool16-sat-1`: 276.2 sales/s).

### Not pursued

- **Prepared statements.**
  - MariaDB does not cache plans for prepared statements; it only skips the parse.
  - The binary protocol cannot ride in a multi-statement packet, so they conflict with step 3's
    batching.
  - go-sql-driver without interpolation costs prepare + execute + close round trips unless every
    statement is cached per connection.
- **The application's repeated reads.** Per sale, a sale reads its cash bank twice, its products
  twice and `cron_actions` twice (general log, step 3). This is identical on every database, so it
  does not change the comparison.

### Step 3 detail: the batched write transaction

- `runTransaction` takes a session from a second pool, opened with `autocommit=0`. The transaction
  opens with its first statement, so START TRANSACTION is gone.
- Statements without a result are queued. They go to the server in one multi-statement packet
  (`multiStatements` + `interpolateParams`):
  - before the next read in the transaction, which must see them;
  - or together with the COMMIT.
- A sale's write becomes 1–2 round trips: its locking read, then its writes and the commit.
- A failed statement stops the packet; the caller sends ROLLBACK, and a deadlock retries the whole
  transaction.
- Statements per sale went from 57.7 to 44.4 (MariaDB counts each statement of a packet), and the
  round trips fell much further.

### Step 1 finding: a one-tuple IN on a DELETE is a full scan

`userstat` + a slow log of every statement examining ≥ 100 rows, over a 3-minute run: the
purchase-order element side table was read 131 M times (98% of all rows examined). MariaDB plans a
single-table `DELETE … WHERE (a, b) IN ((?, ?))` with one tuple as a full scan; with two or more tuples
it uses the index. Most writes are of one record, so nearly every purchase-order update scanned the
whole element table (~90 k rows). Fix (step 2): one record's key is written as `a = ? AND b = ?`.
