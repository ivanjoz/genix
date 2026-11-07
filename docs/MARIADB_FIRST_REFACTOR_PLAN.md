# MariaDB-first refactor plan

**Status:** plan, not started. Written 2026-11-07 from a quick scan, for an agent to execute.

**Direction:** MariaDB (InnoDB) is now the primary database and PostgreSQL the compatible second;
ScyllaDB is no longer supported.
Much of the backend was shaped around ScyllaDB's limits: no transactions, no joins, no `IN` over a
secondary index, no row locks, and eventually consistent views. This plan finds that code and
rewrites it the way a relational database wants it.

Read first: `CLAUDE.md` (root), `backend/docs/MODULE_BOUNDARIES.md`,
`backend/docs/ORM_DATABASE_QUERY.md` (§5.2b Transactions), `backend/genix-orm/db/cache/CACHE_GROUPS.USER.md`
("On PostgreSQL and MariaDB").

---

## 0. Decisions (answered by the user, 2026-11-07)

1. **Scylla is dropped.** Only MariaDB and PostgreSQL are supported. The Scylla driver,
   `backend/db/*_scylla.go` and every Scylla-only workaround can be deleted outright (§7). A rewrite
   no longer has to keep working on Scylla; `db.RunTransaction` may assume a real transaction.
   Conformance goldens are recorded on Scylla today (`GENIX_CONFORMANCE_UPDATE=1`): move recording
   to MariaDB before deleting the Scylla driver.
2. **Schema changes go through reseed, never migration.** Replacing hash columns with composite
   unique keys (§3), dropping packed columns or changing index types is allowed: wipe and reseed.
3. **There is no production; this is pre-alpha dev.** §1 is a cleanup toward the multi-instance
   design, not an urgent correctness fix. Nothing needs backwards compatibility.

Still ask the user before: new dependencies, API contract changes (route shapes, response keys), and
deleting the other drivers (`fdb`, `ydb`, `oceanbase`): only Scylla was decided.

---

## 1. In-process locks → database row locks (highest value, correctness)

Scylla has no row locks, so writers serialize through Go mutexes. Those only hold inside one process.

| Where | What | Rewrite |
|---|---|---|
| `logistics/types/stock_movement_apply.go:23-36, 190-206` | `applyMovimientosLockByCompany`, `LockCompanyStock`: a per-company `sync.Mutex` around stock reads and writes | `db.LockByKeys` on the `ProductStock` rows inside `db.RunTransaction` (the pattern `finance/types/cash_movement_apply.go` already uses). Lock rows in sorted key order to avoid deadlocks. Note: `RecalcProductStockByMovements` also takes this lock. |
| `sales/sale_order_create.go` (`validateSaleStock` + `LockCompanyStock`) | The stock check runs under the mutex, then the write | Move the check inside the sale's transaction, against the rows `LockByKeys` returns. Stock reads then come from a locked read instead of a plain one. |
| `main.go:143` `serializedAPIRequests` | A global request mutex | Find out why it exists. If it guards DB consistency, replace it with a transaction. |
| `production/products.go:138` | A mutex around an existing-products map | Check whether it is only goroutine bookkeeping (fine) or guards a read-then-write (move to a transaction). |

Done when no business write depends on a Go mutex for consistency. Load-test two backend processes
against one MariaDB to prove it.

## 2. Read-modify-write without a transaction → `RunTransaction`

Only 4 non-test files use `db.RunTransaction`, while 43 files call `db.Update`. Audit every handler
that reads a record, computes from it, and writes it back. Balances, counters, statuses and debts are
lost-update candidates.

- Pattern to apply: `RunTransaction { LockByKeys → compute → UpdateIn/InsertIn/UpsertIn/InsertUpdateIncludeIn }`.
  The ORM reuses rows that `LockByKeys` read, so the write adds no second read.
- Priority list to start with:
  - `sales/sale_order_annul.go`: refund plus stock return plus the sale update. Make it one
    transaction, like `PostSaleOrder` now is.
  - `logistics/purchase-order-management.go`: supplier payment plus reception.
  - `finance/expenses.go`, `accounting/asset_payment.go`, `hr/payroll_cash.go`: they read with
    `GetCaja`, then write a cash movement. Use `finance.ApplyCashBankMovementIn` inside the
    document's transaction.
  - `crm/types/client_provider_save.go` (`db.Merge`).
  - Every `db.Merge` call site (8 files) is a read-then-write and should become a transaction.
- Rule: a body must derive its writes from what it reads inside the body, because a deadlock reruns it.
- Rule: business code may wrap ORM errors with `core.Err`. The ORM still retries deadlocks, because
  `Transaction.noteDriverError` hands the raw error to the runner.

## 3. Hash and concatenated keys → composite keys and `IN`

Scylla cannot serve an `IN` over a secondary index, and a global index needs a single hash column.
On MariaDB, a composite unique index on the real columns is cheaper and greppable.

| Where | Today | Rewrite |
|---|---|---|
| `logistics/types/stock_movement_apply.go:695-788` `resolveLotIDsForMovements` | `Hash = MakeKeyConcat(date, supplier, name)` plus a `TypeGlobalIndex`, looked up one query per hash, in parallel | A unique index `(company_id, date, supplier_id, name)`. One `IN`/OR read, or an upsert. |
| `logistics/types/product-stock.go:222, 269` | `TypeGlobalIndex` on `Hash` | Same: a composite index on the real columns. |
| `crm/types/client_provider_snapshot.go:113-120` `ResolveIdentitySnapshots` | One indexed read per identity ("Scylla cannot serve an IN over a secondary index") | One `IN` read. |
| `config/types/empresas.go:87` | `TypeGlobalIndex` on `Email` | On MariaDB this is just a unique secondary index. Check whether it needs to be global. |
| `config/observability.go:147`, `config/observability_test.go:26` | Columns chosen to dodge `ALLOW FILTERING` | Re-evaluate: on MariaDB a filter on an indexed prefix is fine. |

Grep: `MakeKeyConcat` (19 uses), `TypeGlobalIndex`, `"ALLOW FILTERING"`, `"cannot route"`.

## 4. Parallel fan-out reads → one query

Scylla prefers many small parallel single-partition reads. MariaDB prefers one query with `IN` or a
join: fewer round trips and one plan.

- 23 files use `errgroup`. Review each fan-out of **reads of the same table** (`SetLimit` loops, one
  query per ID/hash) and merge them into one `IN`.
- Keep fan-outs across **different tables** (for example `accounting/accounting_operations_api.go`):
  those are legitimate parallelism.
- Writes in a goroutine fan-out (as `PostSaleOrder` had) must move into one transaction. A
  transaction is one connection, so its writes run sequentially.
- Top files by goroutine count: `hr/payroll_data.go` (8), `business/product-ecommerce.go` (5),
  `sales/sale_order_annul.go` (4), `config/company_credit_usage.go` (4), `config/credit_usage.go`,
  `business/initial-data.go`, `accounting/purchases_book_api.go`.

## 5. Index types shaped by Scylla

Current counts in the backend schemas: TypeDelta 51, TypeLocalIndex 25, TypeView 17, TypeCacheGroup 14,
TypeCacheGroupFromKey 5, TypeCacheGroupS3 3, TypeGlobalIndex 3, TypeCacheGroupNoIndex 1.

- **`TypeCacheGroupS3`** (`sales/types/sales.go:309-317`): on SQL drivers, the group snapshot table
  `<table>__group_data` replaced the S3 file store. With Scylla dropped (decision 0.1) the type
  collapses into `TypeCacheGroup`.
- **`TypeView` (17)**: on Scylla these are materialized views, and they carry the rule "the view's
  key columns must be updated together" (`ResolveUpdateColumns` panics otherwise; see the
  `PostSaleOrder` update column list). On MariaDB a view is a plain composite index, and that rule is
  only needed for the ORM's packed columns. For each view:
  - Is it still read?
  - Can it be a `TypeLocalIndex`?
  - Can the "update together" constraint be relaxed on SQL drivers?
  This is an ORM change: keep it in `genix-orm`, with conformance tests.
- **`TypeDelta` with `FixedValues` packing** (`Status`, `Period`): that packing exists so one
  clustering column serves the delta range. On MariaDB, a composite index `(company_id, period,
  status, updated)` does the same without the packed `zz_dlt_*` column. Measure before changing:
  the packed column is also what the delta cursor reads.
- **`.Size()` bit budgets** on cache-group keys (`sales/types/sales.go:322`): needed only for the
  packed `index_value`. Keep them.

## 6. Denormalization and recompute jobs built around missing joins and aggregates

- **`ScheduleCronAction` ActionID 2** (`sales/sale_order_create.go:358-365`): it is scheduled on
  every sale, and the scan found **no handler** for ActionID 2. Verify that. If nothing consumes it,
  delete the scheduling. If it recomputes a summary, the sale DataFrames/aggregate tables
  (`sales/types/sale_frames_sql.go`) may already cover it on SQL.
- **`RecalcProductStockByMovements`** (`stock_movement_apply.go:795+`): it reads the whole ledger
  into memory. On MariaDB the quantity rollup can be a `GROUP BY` over `warehouse_product_movement`,
  but the value replay (average cost) is order-dependent and must stay in Go. Only the quantity part
  is a candidate.
- **Snapshot tables** (`crm/types/client_provider_snapshot.go`, `invoicing/types/invoice_document.go`
  `ClientSnapshotID`): these exist for legal immutability (the identity a comprobante printed), not
  only for Scylla. **Keep them.** Only their lookup path (§3) changes.
- **Counters by read-then-write**: grep `CurrentAmount`, `DebtAmount`, `Quantity +`. Any counter
  updated outside a transaction is either an `UPDATE … SET x = x + ?` candidate or must sit inside a
  §2 transaction.

## 7. Infrastructure and naming

- `backend/db/`:
  - `driver_mariadb.go`, `facade_mariadb.go` are the primary path; make MariaDB the default
    build/driver in `config.example.toml`, the scripts and `CLAUDE.md`.
  - `autoincrement_scylla.go`, `column_rename_scylla*.go`, `facade_scylla.go`, `driver_scylla.go`
    and the genix-orm `scylla/` driver are deleted (decision 0.1), with the build tags and scripts
    that select them.
  - `driver_fdb.go`, `driver_ydb.go`, `driver_oceanbase.go`: ask whether they stay.
    `obdriver` is another session's work; do not touch it.
- **Comments and docs that justify code by Scylla's limits** (`rg -il scylla` → db 17, core 15,
  security 4, config 3, production 2, invoicing 2, others 1): after each rewrite, delete the Scylla
  rationale or reword it in MariaDB terms.
  Skills to update: `create-database-tables`, `cache-groups-index-internals`,
  `delta-cache-index-internals`, `by-ids-cache-internals`.
- **ID allocation** (`db.GetAutoincrementID`, `_sequences`, fareward): IDs are reserved outside the
  transaction on purpose, so a retry burns none. Keep that. The sale ID layout
  (`sales/types/sale_order_id.go`) needs it.

## 8. Method

1. §0 is answered; re-read it.
2. Work section by section in this order: §1 → §2 → §3 → §4 → §6 → §5 → §7. Correctness first,
   then round trips, then the ORM and schema.
3. Per change:
   - Backend: `go build ./... && go test ./...`.
   - Any `genix-orm` change: conformance on MariaDB and PostgreSQL.
     - Tests: `cd backend/genix-orm/conformance && GENIX_CONFORMANCE=1 GENIX_MARIADB_PORT=3309 go test -tags mariadb -count=1 .`
       (`GENIX_POSTGRES_PORT=5434 -tags postgres`).
     - Goldens are recorded on Scylla today (`GENIX_ORM_SCYLLA_TEST=127.0.0.2:9042 -tags scylla`
       with `GENIX_CONFORMANCE_UPDATE=1`). First task of §7: move recording to MariaDB
       (`harness_test.go`, `recording:`), then delete the Scylla driver.
   - genix-orm is a submodule: commit **and push inside it**.
4. Measure the hot paths with the full-mix load test: `backend/tests/loadtest/measure_postgres.sh`,
   `DB_KIND=mariadb|postgres`, `--settle-later=30 --stage-seconds=150`.
   - It runs with `SKIP_BUILD=1`, so rebuild `tmp/loadtest/backend-maria` / `backend-pg` with
     `go build -tags mariadb|postgres` first.
   - Compare against the baseline below.
5. Ask before every schema, API-contract or dependency change (CLAUDE.md §1).

### Baseline: full mix, 2026-11-07, after one transaction per sale

| | MariaDB | PostgreSQL |
|---|---|---|
| Sales/s | 164.7 | 175.2 |
| DB CPU per sale | 4.47 ms | 3.02 ms |
| Backend CPU per sale | 4.58 ms | 4.64 ms |
| Commits per sale | 1.73 | n/a (PostgreSQL counts read-only transactions) |
| Sale p50 / p99 | 13 / 230 ms | 5 / 178 ms |
| accounting-operations p50 | 85 ms | 70 ms |

That MariaDB run had 3 deadlock errors reach the client. After the retry fix (§2, last rule), a
re-run gave 165.2 sales/s at 4.48 ms DB CPU per sale. Its 2 deadlocks were both retried, and no
request failed.
