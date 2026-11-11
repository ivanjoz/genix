# Cache audit — decisions for the owner

Audit of 2026-11-10: every page, on load or reload, must read from a delta cache, a cache-groups
cache or the records-by-ID cache. Cases that needed an owner decision were NOT implemented; they are
listed here per module with options and a recommendation. Everything else was converted (or was
already cached) — see the git diff of the same date.

## HR

All hr pages already load through delta caches (`hr-employees`, `hr-dependents`,
`hr-employee-concepts`, `hr-payslips` per period, `hr-org`, `hr-payroll-config`, `hr-contracts`,
`hr-employee-records`, `hr-legal-parameters`). No code changes.

1. **Payroll page sends `POST.hr-payroll-ensure` on every open** (`routes/hr/payroll/RunsTab.svelte`,
   `onMount(updatePayroll)`). It reads registration period, every run and all payroll inputs, then
   creates/recalculates months, and answers the month window (`FirstPeriod`/`LastPeriod`).
   - A: keep — deliberate "opening the page brings runs up to date"; it skips up-to-date months
     without locking.
   - B: writes of records/contracts/employees mark affected runs stale; the page derives the window
     from cached data and recalculates only stale runs or on demand.
   - Recommendation: A for now (a write the business flow relies on); B is a payroll redesign.
2. **`GET.hr-payslips-employee` and the `TypeDelta(EmployeeID, Status)` index on `hr_payslips` have
   no consumer.** Every payslip write still maintains the index.
   - Keep for a planned per-worker payslip history page, or delete route + handler + index +
     access.toml entry (reseed).
   - Recommendation: delete unless that page is planned soon.
3. **`TypeCacheGroup(Period)` on `hr_payslips` has no consumer** ("multi-period cost report" never
   built); every payslip write writes its group marks.
   - Recommendation: keep only if the cost report is on the roadmap; otherwise drop (reseed).

## Logistics, sales, production

Done: `GET.product-supply` → `Delta()` (new `ver`/`mdt` columns, TypeDelta on Status);
`GET.warehouse-movements-grouped` deleted — purchase-management now folds the `products-stock` delta
and the per-day `warehouse-movements` cache groups in the browser
(`routes/logistics/purchase-management/supply-management.ts`); the unused movement TypeView was
dropped; warehouse-movements page sent `almacen-id`/`producto-id` while the handler reads
`warehouse-id`/`product-id` (filters were ignored) — fixed.

1. **`ProductStockDetail` is replaced whole on every sync** (`GET.warehouse-product-stock`): its key
   is composite (`ProductStockID, LotID, SerialNumber`), so no TypeDelta.
   - A: keep. B: autoincrement ID + unique index on the three columns; the stock engine
     (`stock_movement_apply.go`, `RecalcProductStockByMovements`) and `accounting/asset_edit.go`
     must resolve that ID on every upsert.
   - Recommendation: A for now (concurrency-sensitive engine).
2. **Purchase-management downloads raw movement rows** (one per sale line) for 30 days instead of
   pre-summed rows. Days are cached; today's group re-downloads after each sale.
   - Keep, or add a day×product summed table/DataFrame exposed as a cache group.
   - Recommendation: keep unless busy tenants show a heavy first load.
   - Visible changes: the chart covers exactly 30 days; days before a product's first movement show
     stock reconstructed back from today (was 0); products with stock but no movement now get a row;
     current stock reads `ProductID` directly (the old `/10000` decode looked wrong).
3. **`GET.sale-summary` uses the DataFrame day-file cache** (`GETFrameFiles` + `ExecChanged`), a
   fourth mechanism that does keep days in the browser. Recommendation: accept as compliant.
4. **`backend/exec/test_selects.go` `TestSelects2`** runs a GroupBy on the movement table whose
   TypeView was dropped; it may fail at runtime. Recommendation: delete that demo.

## Projects

Done: `GET.projects-story-detail` (replace on every open) → by-IDs `GET.projects-story-details-ids`
(`projects_story_detail` key renamed `StoryID` → `ID`, `CacheByIDs`); `GET.projects-team-candidates`
deleted — candidates are derived in the browser from the cached `GET.users` list by
`teamCandidates()` (`routes/projects/project.ts`, same rule as backend `canWorkInProjects`). Other
project routes were already deltas.

1. **Stale story detail window**: the by-IDs cache answers from memory for 5 s after a fetch; open →
   save → reopen within 5 s may show the old detail once. Fix would be a genix-ui helper that writes
   a saved record into the by-IDs cache. Recommendation: leave it.

## Accounting, finance

Done: `GET.sales-book` / `GET.purchases-book` → delta responses from the same period logs as
`accounting-operations` (`?date=` removed; the browser cuts the day), corrected documents of earlier
periods via new by-IDs `GET.invoice-document-ids`; `GET.asset-depreciation` → cache group per asset;
`GET.cash-banks-movements` (date range) → cache groups per (cash bank, day), max 120 days;
`GET.cash-banks-reconciliations` → delta per cash bank (shows every count, not the last 200);
`GET.cash-bank-movement-by-id` → cache group per document (unused `reference-id` mode deleted; the
expense payment list now keeps only expense payments — an asset payment sharing a document ID used
to show up); `GET.expense-schedule-periods` → cache group per schedule (still creates missing
periods first).

1. **`GET.accounting-cash-balances` (financial statements cash flow) is still uncached.** It needs
   each cash bank's balance at the last movement on or before a date, which may be any earlier
   month.
   - A (recommended): new `cash_bank_period_balances` table (cash bank × month closing balance),
     written in `ApplyCashBankMovementIn` and `reconcileCashBankIn`, synced as a delta; the browser
     takes the latest row up to each period. Adds writes to an accounting-critical path.
   - B: compute in the browser from the period's movements + current balance (wrong with back-dated
     movements or idle months).
   - C: leave uncached (one small read per cash bank).
2. **Cash-banks Movements tab shows "latest 200 movements"** — a fixed count can't be a cache group
   and a per-cash-bank delta would grow unbounded; it is still an uncached pass-through read.
   Recommendation: show the last N days (e.g. 30) through the cache-group route (a date window
   instead of a count).

## Business, config, security, invoicing, webpage

Done: `GET.perfiles` → `Delta()` from the primary DB (like users); `GET.gallery-images` → delta
(key `Image` string → `ImageID` int32); `GET.company-secrets` → delta (encrypted columns excluded);
`GET.ecommerce-page-content` → delta per page (key → `ID = PageID*1000 + SectionID`, max 999
sections); `GET.website-config` → cached whole-list replace (composite key, ≤9 rows);
`GET.parametros`/`POST.parametros` deleted (no consumer; read every tenant's rows and trusted the
client's CompanyID). Live monitoring, usage counters, searches, downloads and login/public routes
excluded.

1. **`GET.empresas` and `GET.company-parametros`**: companies live only in the cloud mirror when it
   is enabled, so they can't move to a primary-DB delta. Both are cached whole-list replaces;
   `GET.empresas` reads an `?upd=` the client never sends.
   - A (recommended): keep; delete the unused `upd` param.
   - B: make the primary DB authoritative for companies and sync by delta (changes mirror semantics).
2. ~~`company_secrets` has `CacheByIDs: true` with no by-IDs route.~~ Done: flag dropped.
3. ~~Mirror composite-range code.~~ Done: `buildCompositeRange` and the test fixture are gone. A
   mirror query now constrains exactly one indexed column (`buildIndexKeyRange`): the single-key
   range stays because `GET.empresas` reads companies by `updated >=` on the mirror.
4. `docs/COMPANY_CONFIG_BLOB_PLAN.md:157` still mentions the deleted `PostParametros`.

## Cross-cutting

1. ~~`GET.users` ships `PasswordHash` to every session.~~ Done: `GET.users`, `GET.users-ids` and
   `POST.users`/`user-self` blank the hash before answering (`security/usuarios.go`
   `clearPasswordHashes`); login no longer logs both hashes. The access blobs still travel (used
   client-side by the projects candidates).
