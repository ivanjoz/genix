# Plan: SaleOrderV2 + SaleOrderV2Detail write-amplification benchmark

## Goal

Measure what a header + detail sale model costs, against the current `sale_order`:

- `SaleOrderV2` keeps the header.
- `SaleOrderV2Detail` holds one row per line, and each row carries the sale's `Status`.
- Every status change of a sale rewrites all of its detail rows.
- Two materialized views on the detail table answer the sales summaries with a native
  `GROUP BY` + `SUM`, in place of the two DataFrames (`sale-summary`, `sale-client-product`).

The verdict is writes per sale, ScyllaDB CPU per sale and sales/s at the same load as the
baseline. Nothing in the current flow changes: V2 is a second flow, in its own files, chosen by
a load-test flag.

### Baseline (current model, pinned 2 vCPU, movement-schema build)

| Metric | Value |
|---|---|
| Steady state, 10 companies | 136.9 sales/s |
| Ramp, 10 / 20 / 30 companies | 112.6 / 110.5 / 84.6 sales/s |
| ScyllaDB partition writes per sale | 55 (`sale_order` 17.97 of them) |
| `sale_order` product indexes (`zz_igs_*`, two) | 5.79 each, about 5.4 lines per sale |
| ScyllaDB CPU per sale | 7.9–11.4 ms |

In today's load test only annulments change a status:
- 1% of sales are annulled;
- the 10% unpaid and 5% undelivered sales are never paid or delivered later.

V2's extra cost lives in the status changes, so step 4 adds them to both runs.

## 1. Tables (`sales/types/sale_order_v2.go`)

### `SaleOrderV2`, table `sale_order_v2`, schema ID 84

`SaleOrder` without its two product indexes, its computed `DetailProductIDs` column and its
DataFrames. The other indexes stay as they are:
- the `Date`, `Date+Status` and `Date+ClientID` `TypeCacheGroupS3` groups;
- the two deltas;
- `CacheByIDs`.

It keeps the `Lines` blob, so the sales report and every handler still read a whole sale from
one row. The detail rows are a copy of `Lines` for the views. See question Q1.

### `SaleOrderV2Detail`, table `sale_order_v2_detail`, schema ID 86 (85 is `employee_records`)

| Column | Type | Note |
|---|---|---|
| `CompanyID` | int32 | partition |
| `SaleID` | int64 | key 1 |
| `LineNumber` | int8 | key 2: the line's position in `Lines` |
| `Date` | int16 | the sale's date |
| `ClientID` | int32 | the sale's client |
| `ProductID` | int32 | |
| `SubDivisor` | int16 | |
| `Status` | int8 | copied from the sale, rewritten on every change |
| `SoldUnits`, `SoldSub` | int32 | unpacked from the line's `Quantity` |
| `AmountPEN` | int32 | line amount in soles (`SaleOrder.AmountInPEN`) |
| `Updated` | int32 | |

There's no debt or pending column. In `eachSummaryLine` (`sales/types/sales.go:186`) both depend
only on the status:
- the amount is debt when the status is Pending or Delivered;
- the quantity is pending when the status is Pending or Paid;
- an Annulled sale adds nothing.

With `Status` in the views' keys, the read derives both from the per-status sums.

### The two views (`TypeView`, packed keys, `Cols` = the summed columns)

A view needs a packed key, with every key column sized, to be read as a `Date` range. Without
sizes the ORM compiles a hash view, which only answers equalities
(`fixed_value_view_test.go`, `TestFixedValueViewWithTwoUndeclaredKeysStaysAHashView`). The pattern
to copy is the movement view, `logistics/types/product-stock-movement.go:171`, which is read with
`GroupBy(...)` in `logistics/product-supply-management.go:164`.

1. **By product**, replacing `sale-summary`: `Date 16 + ProductID 27 + Status 3 + SubDivisor 14`
   = 60 bits. `Cols`: `SoldUnits`, `SoldSub`, `AmountPEN`.
2. **By client and product**, replacing `sale-client-product`: `Date 16 + ClientID + ProductID +
   Status 3` doesn't fit 64 bits at full width, and with `SubDivisor` it gets worse. See question Q2.

## 2. The V2 handlers (`sales/sale_order_v2_create.go`, `sales/sale_order_v2_annul.go`, `sales/sale_order_v2_query.go`)

`PostSaleOrder` and `PostSaleOrderAnnul` touch the `sale_order` table in five places:
- read the stored sale: `sale_order_create.go:57` and `sale_order_annul.go:63`;
- insert: `sale_order_create.go:239`;
- update: `sale_order_create.go:342` and `sale_order_annul.go:186`.

Everything else is shared with V2 and is part of the cost being compared, so it must stay
identical: stock movements, cash movements, the invoice document, fareward locks and sequences.
See question Q3 for how to share it.

The V2 writes:
- **Insert:** `db.Insert` of the header, then `db.Insert` of its detail rows (one partition, one
  batch). The detail goes after the header, as invoicing does today.
- **Status change** (payment, delivery, annulment): `db.Update` of the header's columns as today,
  then `db.Update(&details, Status, Updated)` on every row of the sale. The rows are rebuilt from
  the header's `Lines`, so no read is needed.
- **New routes:** `POST.sale-order-v2`, `POST.sale-order-annul-v2`, and
  `GET.sale-summary-v2` / `GET.sale-client-product-v2`, which run the `GroupBy` over a `Date` range
  and fold the per-status sums into the same response shape as today's summaries.

## 3. Deploy

Both tables go through `fn-init` / homologate like any table: added to the sales module's table
list, with no hand-written CQL. Check that `run.sh`'s fresh keyspace creates them and the two views.

## 4. Load-test changes (`tests/sample_records/load_flow.go`)

- **`--sales-v2`:** the cashiers post to `POST.sale-order-v2` and annul through
  `POST.sale-order-annul-v2`. The manager's `sale-summary` report reads `GET.sale-summary-v2`.
  Everything else stays the same.
- **`--settle-later`, used in both runs:**
  - each unpaid sale gets its payment `N` seconds later;
  - each undelivered sale gets its delivery `N` seconds later;
  - both go through the existing `POST.sale-order` update with `ActionsIncluded` 2 or 3.

  Each settlement is a status change: a header update in both models, plus the detail rewrite in V2.
- **`--status-change-percent=P`:** raises the unpaid and undelivered percentages, to plot writes
  per sale against status changes per sale. See question Q4.

## 5. Measurement

Both runs use the same build, `run.sh run --companies-per-stage=10 --think-ms=200`, the stack
pinned, and `TABLET_BALANCING=false`:

1. Baseline with `--settle-later`.
2. V2 with `--settle-later`.
3. Optionally, V2 at two or three `--status-change-percent` values.

For each run, from `scylla_metrics_diff.sh` over the same steady-state window:

- **Partition writes per sale, by table,** including `sale_order_v2_detail` and its two views.
- **View rows per sale,** from `scylla_database_total_view_updates_pushed_local`.
  - Partition writes undercount the views. A view partitioned by `company_id` gets all ~5.4 rows of
    a sale in one partition mutation, unlike today's `zz_igs_*` indexes, whose partition is the
    product: one mutation per product.
  - So partition writes alone would make V2 look cheaper than it is.
- **Reads per sale,** the views' `GROUP BY` included.
- **CPU per sale** (backend and ScyllaDB), **sales/s**, **p99**, and write timeouts.

### Expected numbers, to check against

These are estimates, not measured.

- **Insert:** about 55 partition writes, the same as today.
  - Removed: 11.6 (the two `zz_igs_*` indexes) and 1.07 (`sale_order__dtf_log`).
  - Added: about 1 detail base mutation and 2 view mutations, which carry about 2 × 5.4 view rows.
- **Status change:** about 3 partition writes, but about 21.6 view rows (5.4 lines × 2 views × 2).
  - The partition writes are the detail mutation plus one per view.
  - Each line's view row is deleted under the old status and written under the new one.
- **DataFrame files** stop being written and compacted: less backend CPU.
- **Reads:** each summary report reads every line in its range instead of the precomputed sums,
  more ScyllaDB CPU per report, growing with history.

## Results (2026-11-06, steady state)

10 companies, `--think-ms=200 --settle-later=30`, stack pinned, `TABLET_BALANCING=false`, one 180 s
stage. ScyllaDB metrics over a 90 s window starting 30 s after the 10th company onboarded.
Raw files: `/tmp/v2bench/{baseline,v2}`.

| Metric (window) | Baseline | V2 |
|---|---|---|
| Sales/s (window / whole stage) | 95.5 / 92.7 | 113.8 / 105.7 |
| Sale p50 / p99 (stage) | 151 ms / 502 ms | 100 ms / 508 ms |
| Settlements per sale | 0.12 | 0.13 |
| Partition writes per sale, all tables | 55.0 | 45.5 |
| Partition writes per sale, sale tables | 21.1 | 11.8 |
| View rows per sale (`view_updates_pushed_local`) | 39.4 | 30.0 |
| Reads per sale | 70.0 | 40.0 |
| ScyllaDB CPU per sale | 13.9 ms | 11.5 ms |
| Summary report p50 / p99 | 116 / 618 ms | 31 / 139 ms |
| Errors | 0 | 0 |

- **Writes are a clean comparison.** The detail table and its two views take one partition write
  each per sale write or status change (132/s each), in place of the two `zz_igs_*` indexes
  (11.7 per sale) and `sale_order__dtf_log`.
- **Reads and CPU favour V2.** `sale_order` alone is read 3,193 times/s in the baseline (33 per
  sale), and almost not at all in V2. The cause is not established. Part of it is the reads that
  still go to `sale_order` (the cashier's `sync:sale-orders`, the `sale-order-query` and
  `sales-book` reports), which find an empty table in a V2 run.
- **The summary report** reads only a few minutes of history here; its cost grows with history.

### What the two detail views cost (2026-11-06, A/B, sales-only)

The same V2 run with and without the two `sale_order_v2_detail` views; the second binary has the
detail table with no `Indexes`. `measure_scylla.sh NAME --sales-v2 --sales-only --settle-later=30`:
stack pinned to 2 vCPU (CPUs 0,1), 10 companies, `--think-ms=200`, fareward bypassed, 90 s window.
No reports, because `GET.sale-summary-v2` needs the views. Runs alternated views, no views, views,
no views. Raw files: `backend/measure/v2-{views,noviews}-{1,2}`.

| Run | sales/s | ScyllaDB CPU per sale | backend CPU per sale | compaction CPU (ms/s) |
|---|---|---|---|---|
| views 1 | 126.2 | 7.88 ms | 4.36 ms | 188 |
| views 2 | 126.5 | 8.01 ms | 4.43 ms | 174 |
| no views 1 | 131.5 | 7.18 ms | 4.28 ms | 166 |
| no views 2 | 123.8 | 7.39 ms | 4.43 ms | 137 |

- **The two views cost about 0.66 ms of ScyllaDB CPU per sale:** 7.95 against 7.29 ms on average,
  about 9% of ScyllaDB's work per sale. That's at 0.13 settlements per sale, each of which moves the
  sale's rows in both views again.
- **sales/s doesn't separate them:** the runs vary by about ±4 sales/s. The CPU per sale is the
  figure to trust.
- **Backend CPU is the same:** the views are maintained by ScyllaDB.
- **Compaction is about 30 ms/s higher with the views,** a large part of the difference: two more
  tables are flushed and compacted.
- **On the 2 vCPU stack:** backend + ScyllaDB + fareward come to about 12.4 ms per sale with the
  views and 11.7 ms without. That puts the CPU ceiling at about 160 and 170 sales/s (about −6%).

### V1 against V2, writes only (2026-11-06, sales-only)

V1 (`sale_order`) in the same mode as the views A/B: `measure_scylla.sh NAME --sales-only
--settle-later=30`. Raw files: `backend/measure/v1-salesonly-{1,2}`.

| Run | sales/s | ScyllaDB CPU per sale | backend CPU per sale | compaction (ms/s) | view updates per sale |
|---|---|---|---|---|---|
| V1 1 | 132.9 | 8.08 ms | 4.51 ms | 217 | 39.4 |
| V1 2 | 135.2 | 6.81 ms | 4.09 ms | 138 | 39.5 |
| V2 with views (mean of 2) | 126.4 | 7.95 ms | 4.40 ms | 181 | 30.0 |
| V2 without views (mean of 2) | 127.6 | 7.29 ms | 4.36 ms | 151 | 27.5 |

- **On writes alone, V1 and V2 cost the same.** V1 averages 7.45 ms of ScyllaDB CPU per sale; V2
  with views, 7.95 ms. The difference is inside the spread between runs: V1's two runs differ by
  1.3 ms, mostly in compaction (217 against 138 ms/s).
- **V1 makes more view updates** (39.5 against 30.0 per sale): its two product indexes write about
  11.7 separate partitions per sale (one per product). V2's views write one partition per sale but
  carry its ~5.4 rows, and they move them on every status change. The two costs come out about equal.
- **V2's lead in the first comparison (11.5 against 13.9 ms) was reads,** not table design. The
  reports that still read `sale_order` (`sale-order-query`, `sales-book`, the accounting reports)
  found it empty in the V2 run: `sale_order` was read 3,193 times/s in V1 and almost not at all in V2.

## Decisions (2026-11-06)

- **Q1:** keep the `Lines` blob.
- **Q2:** (b). View 2 is keyed `Date + ClientID.Size(31)` (Date takes the remaining bits: the ORM
  refuses `Size()` on a packed view's first column). Its `Cols` carry `ProductID`, `SubDivisor`,
  `Status` and the sums, and `GET.sale-client-product-v2` reads its rows over the `Date` range and
  sums them per client and product in memory. A status change rewrites the row in place.
- **Q3:** (b). `saleOrderStore` (`sales/sale_order_store.go`): `readSale`, `insertSale`,
  `updateSaleSettlement`, `updateSaleAnnulment`. `saleOrderTableStore` is `sale_order`,
  `saleOrderV2Store` (`sales/sale_order_v2.go`) the header + detail.
- **Q4:** `--status-change-percent=P` added: P% unpaid and, drawn apart, P% undelivered.
- **Q5:** open.

## Open questions (as first written)

- **Q1.** Does `SaleOrderV2` keep the `Lines` blob, so the lines are stored twice? Or do they live
  only in the detail, and every sale read also reads its detail?
  - Recommendation: keep the blob.
  - Without it, the sale report, annulment, by-IDs and the delta caches all need a second read.
  - That measures something other than the views.
- **Q2.** How does view 2 (client + product) fit in 64 bits? Options:
  - (a) `ClientID 22 + ProductID 22 + Status 3 + Date 16` = 63 bits: 4M clients and 4M products
    per company, with no `SubDivisor` in the key.
  - (b) A second view keyed `Date + ClientID` only, with the product summed in memory.
  - (c) Let it be a hash view on `ClientID + ProductID`, read per client and product, with no
    `Date` range.
  - Recommendation: (a). The SubDivisor problem appears only when sub-units of different divisors
    mix within one client and product.
- **Q3.** How does V2 share the create logic?
  - (a) A copy of `PostSaleOrder` / `PostSaleOrderAnnul` in the V2 files, with the five table
    touches swapped. That's about 700 duplicated lines, and they drift.
  - (b) Extract the five touches into a small interface, `saleOrderStore`, implemented once per
    model, with the handlers taking it as a parameter. That touches the current files, but the
    flow stays identical.
  - Recommendation: (b), because it guarantees the only difference measured is the table model.
- **Q4.** Add `--status-change-percent`, or are the current unpaid and undelivered percentages
  enough, once settled later?
- **Q5.** Where should the plan and the V2 code live once the test is done: delete the V2 files, or
  keep them behind the flag?

## V3: the OceanBase sale model (2026-11-06)

V2 copies the sale's `Status` and `ClientID` onto every detail row, so each payment, delivery or
annulment rewrites all ~5.4 of them plus their view rows. V3 splits what changes from what doesn't,
and joins them at read time, which OceanBase can do and ScyllaDB cannot.

### Tables (`sales/types/sale_order_v3.go`)

- **`SaleOrderV3`, `sale_order_v3`, ID 87: the header.** Everything an update changes lives only
  here, so a settlement or annulment writes one row. It keeps the `Lines` blob (a handler reads a
  whole sale from one row), `CacheByIDs`, and only the two delta indexes (`Status`,
  `Period+Status`) its syncs need. No date or cache-group index: every date-ranged read starts from
  the lines.
- **`SaleOrderV3Line`, `sale_order_v3_line`, ID 88: one row per line, written once at insert and
  never updated.** Key `(company_id, date, sale_id, line_number)`: the sale's `Date` never changes,
  so it leads the key and a date range is a key range. It holds only immutable data (product, divisor,
  units, sub-units, amount in PEN); status and client come from the header. `ColumnStore: true`: a
  hybrid row + column table (`WITH COLUMN GROUP(all columns, each column)`). No secondary index: a
  product filter scans the range's `product_id` column.

### Reads (`sales/sale_order_v3_query.go`), SQL reports through `db.QuerySQL`

- `GET.sale-summary-v3`, `GET.sale-client-product-v3`: lines of the range joined to their header on
  `(company_id, sale_id)` (the header's primary key), `GROUP BY` day, [client,] product, divisor
  and status, folded by the V2 folds into the same response shapes.
- `GET.sale-order-query-v3`: the sale IDs of the range from the lines (product a line's, status or
  client the header's), then the headers by primary key. Returns `[]SaleOrderV3`, without the
  browser's group cache.
- `GET.sale-orders-v3`: the status-tab delta sync, as `GET.sale-orders`.

Writes: `POST.sale-order-v3` / `POST.sale-order-annul-v3` through the shared flow
(`saleOrderV3Store`): insert = header + its lines, every update = the header only.

### ORM support added for it

- `TableSchema.ColumnStore` (obdriver deploys and homologates the hybrid store; other drivers
  ignore it).
- `db.QuerySQL[Row]` / `db.SQLTable[Record]` (backend facade, `obdriver.QuerySQL`): read-only SQL
  scanned into a row struct by column label. OceanBase only; other builds fail loudly.

### Running it (homelab, fareward bypassed)

- `[fareward].bypass = true`: no credit charge, no access check (every session holds every access),
  no request log; refused on Lambda.
- `tests/loadtest/deploy_oceanbase.sh` builds `-tags oceanbase` and copies it with the scripts to
  `100.64.0.4:~/genix-loadtest`; there `tests/loadtest/run_oceanbase.sh run --sales-v3 …` is the
  ramp and `tests/loadtest/measure_oceanbase.sh NAME --sales-v3 …` the 10-company steady state.
- Metrics (`ob_metrics_diff.py`): observer and backend CPU per sale, and statements, server CPU and
  elapsed per sale by kind and table from the plan cache (`GV$OB_PLAN_CACHE_PLAN_STAT`).
- `--sales-v3` moves every sale read of the run to V3 (summary, sale query, the cashiers'
  `sale-orders` sync). The accounting reports (`sales-book`, `accounting-operations`) still read
  `sale_order`, as under V2.
- The Scylla baseline runs on the same machine: `tests/loadtest/deploy_scylla_homelab.sh` copies the
  build, fareward and the scripts to `~/genix-scylla`; there, from `backend/`,
  `STACK_CPUS=0-11 SCYLLA_SMP=8 SCYLLA_MEMORY=8G tests/loadtest/run.sh stack` once, and
  `STACK_CPUS=0-11 SCYLLA_SMP=8 BACKEND_MEMORY=4G LOAD_CPUS=12-15 tests/loadtest/measure_scylla.sh NAME …`
  (fareward bypassed, still serving the sequences). Stop the observer first: both do not fit in RAM.
- OceanBase redo: the homelab NVMe takes 6–8 ms per synced write (15 ms commits).
  `ob_fast_redo.sh enable` preloads libeatmydata for the benchmark (0.5 ms commits, NOT durable);
  `disable` afterwards.

### V3 results (2026-11-06, homelab, 10 companies, `--settle-later=30 --think-ms=200`, 90 s window)

Resources: OceanBase CE 5.0.1, `cpu_count=8`, `memory_limit=8G`, backend pinned to CPUs 0–3. Scylla:
`--smp 8`, 8 GB, on CPUs 0–11 shared with the backend and fareward. The load generator ran on CPUs 12–15
in both runs.

| | OceanBase, V3 (lines columnar) | OceanBase, V3 (header and lines columnar) | ScyllaDB, V1 | ScyllaDB, V2 |
|---|---|---|---|---|
| sales/s | 137 | 142 | 154 | 149 |
| sale p99 (stage) | 267 ms | 307 ms | 208 ms | 487 ms |
| errors | 0 | 0 | 0.03% | 0.04% |
| DB process CPU per sale | **63.3 ms** (8.7 cores, saturated) | **57.5 ms** (8.2 cores) | **27.3 ms** (4.2 cores) | **23.9 ms** (3.6 cores) |
| backend CPU per sale | 15.2 ms | 14.9 ms | 17.2 ms | 15.3 ms |
| statements per sale | 38.1 (plan cache) | 38.1 | 37.7 CQL requests | 36.5 CQL requests |

- OceanBase is CPU-bound at its `cpu_count`. The plan cache accounts for 25 ms of the 63 ms. The rest
  is the observer's own work outside statement execution: transaction commit, clog, minor freezes and
  compaction, RPC.
- Most server CPU per sale goes to:
  - the stock upsert (`warehouse_product_stock`, 2.6 ms);
  - the index-updated marks (`warehouse_product_movement__index_updated`, 2.1 ms);
  - the by-IDs slot upserts (`_cache_by_ids_slots`, 2.5 per sale, 1.7 ms);
  - the selects on tables the run doesn't attribute (1.9 ms).
- `_sequences` still costs a SELECT + UPDATE + INSERT round per sale (1.14 each). That is most likely
  the invoice correlativo (`GetAutoincrementID`), which must stay gapless and so cannot use key blocks.

### Tuning OceanBase for throughput (2026-11-06, sales-only benchmark)

`tests/loadtest/bench_sales_oceanbase.sh NAME` (`--sales-only --sales-v3 --think-ms=0`):
- 10 companies × 4 cashiers selling as fast as they can; the back office stays because it restocks;
  no page syncs or manager reports;
- a 15 s warm-up and a 45 s window, about two minutes in all;
- redo through libeatmydata.

Each step adds to the previous one:

| step | sales/s | observer CPU per sale |
|---|---|---|
| t0 baseline | 188.6 | 50.3 ms |
| t1 `InterpolateParams` in obdriver: one round trip per statement instead of prepare + execute + close | **211.5** | **44.5 ms** |
| t2 `enable_sql_audit = false`, `enable_perf_event = false` | **224.2** | **42.3 ms** |
| t3 tenant unit `mini_unit` 4 → 6 CPUs (the whole `cpu_count = 8` with sys's 2) | 214.2 | 44.9 ms |
| t4 `syslog_level = ERROR`, `syslog_io_bandwidth_limit = 1MB` (reverted afterwards) | 215.5 | 43.6 ms |
| t5 8 cashiers per company (80 sellers) | 205.1 | 41.7 ms |

- Only t1 and t2 counted. Runs vary by about ±15 sales/s, so t2–t5 are all the same.
- The host is a Ryzen 7 5700U: 8 cores, 16 threads.
  - During a run, `vmstat` shows about 45% user, 24% system, about 30% idle, and 150k context switches/s.
  - Twice the sellers gave the same throughput at twice the latency.
  - So the ceiling is not the tenant's CPU quota. It is the per-sale commit path and the rows every
    sale of a company shares: the till's `cash_banks` row and the stock rows.
- `enable_perf_event = false` empties `GV$OB_PLAN_CACHE_PLAN_STAT`'s counters, so the per-statement
  table of `ob_metrics_diff.py` reads 0. `enable_sql_audit = false` fails `TestSpikeReadShapes`, which
  reads plans from the audit. Turn both back on for diagnosis.

### seekdb vs OceanBase (2026-11-06, sales-only benchmark, same tables)

seekdb 1.4.0.0 (`docker.io/oceanbase/seekdb`) is the OceanBase SQL engine as one single-tenant server.

Setup:
- `tests/loadtest/run_seekdb.sh start` replaces the observer on port 2881: `CPU_COUNT=8`,
  `MEMORY_LIMIT=8G`, redo through `nosync_preload.so`;
- then `OB_USER=root DB_PROCESS=seekdb OB_ENGINE=seekdb tests/loadtest/bench_sales_oceanbase.sh NAME`.

seekdb lacks two things:
- column groups (no hybrid store);
- table TTL, which the DataFrame log uses.

`[oceanbase].engine = "seekdb"` (`obdriver.EngineSeekDB`) deploys that subset: row tables only, and a
log that keeps its rows. The observer ran with the same setting, so both engines had the same tables.

| | window | sales/s | DB CPU per sale | backend CPU per sale |
|---|---|---|---|---|
| OceanBase, column groups (t2) | 45 s | 224.2 | 42.3 ms | 9.9 ms |
| OceanBase, row tables (t7) | 45 s | 212.0 | 41.1 ms | 10.1 ms |
| seekdb, row tables (s1) | 45 s | 314.3 | 24.0 ms | 8.6 ms |
| OceanBase, row tables (t8) | 150 s | **214.7** | **44.4 ms** | 10.2 ms |
| seekdb, row tables (s2) | 150 s | **274.0** | **28.2 ms** | 9.4 ms |

- Sustained over 150 s, seekdb gives about 28% more throughput and uses about 36% less database CPU
  per sale. Column groups made no difference on OceanBase.
- seekdb's database has no sys or meta tenant beside it, which probably accounts for most of the saving.
- Both slow down as the tables grow:
  - OceanBase: from about 240 to about 205 sales/s;
  - seekdb: from about 330 to about 260 sales/s.
- seekdb's memtable freezes show as dips: about 160 sales/s for 10–20 s, when its memory drops from
  about 1.7 GB to 0.8 GB. Its memory stays under 2 GB of the 8 GB limit, against about 3.7 GB for the
  observer.
- seekdb is single-node: no replicas and no distributed mode of its own.

### seekdb vs ScyllaDB v2 (2026-11-06, the same sales-only benchmark)

ScyllaDB v2 ran the same load: `--sales-only --sales-v2`, 10 companies × 4 cashiers, no think time,
15 s warm-up and a 150 s window. Setup:
- `STACK_CPUS=0-11 BACKEND_MEMORY=4G LOAD_CPUS=12-15 WARMUP_SECONDS=15 WINDOW_SECONDS=150 RUN_ARGS=... tests/loadtest/measure_scylla.sh sv2-long`;
- the container from `SCYLLA_SMP=8 SCYLLA_MEMORY=8G run.sh stack`;
- the observer stopped during the run.

| | sales/s | DB CPU per sale | backend CPU per sale | total CPU per sale | sale p99 | errors | DB memory |
|---|---|---|---|---|---|---|---|
| OceanBase v3, row tables (t8) | 214.7 | 44.4 ms | 10.2 ms | 54.6 ms | 370 ms | 0 | ~3.7 GB |
| seekdb v3, row tables (s2) | **274.0** | 28.2 ms | 9.4 ms | 37.6 ms | 301 ms | 0 | < 2 GB |
| ScyllaDB v2 (sv2-long) | 218.0 | **18.1 ms** | 13.5 ms | **31.9 ms** (+0.4 fareward) | 2.1 s | 89 (0.2%) | 7.3 GB (preallocated) |

- About 4% of sale attempts were refused for lack of stock in every run, so refusals don't skew the
  comparison.
- ScyllaDB uses the least CPU per sale, but its throughput swings between about 110 and 316 sales/s,
  with p99 latency at 2.1 s and some errors. seekdb holds about 260–330 sales/s apart from its freeze dips.
- The backend costs more on ScyllaDB (13.5 against 9.4 ms per sale) and sends about 32 CQL requests
  per sale (EXECUTE 18, BATCH 14). The cause has not been measured.
- ScyllaDB used about 3.9 of its 12 cores, so CPU did not cap its throughput. The cause of its
  swings has not been measured.
- Statements per sale were not measured on OceanBase or seekdb, because perf events are off.
