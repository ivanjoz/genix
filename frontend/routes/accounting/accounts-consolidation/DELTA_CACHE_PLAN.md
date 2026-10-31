# Delta cache for `GET accounting-operations` — implementation plan

Status: **plan, not implemented.** Goal: opening or reloading Consolidado & Cuentas only downloads
what changed in the period since the last visit, and stops downloading the cash movements that are
the mere payment of a document (5027 of the 5032 operations of Oct 2026).

## 1. Today

- `accounts-consolidation.svelte.ts:18` calls a plain `GET` with no cache; nothing survives a reload.
- `GetAccountingOperations` (`backend/accounting/accounting_operations_api.go`) reads the whole period
  on every call (~8 parallel reads), then the older documents the period's payments and receptions
  point to, then the products all of those name.
- Every payment and collection is fetched and shown as its own "Movimiento de caja · Derivada" row,
  and resolves its counter-account by looking up the document it pays.

## 2. Design

### 2.1 A period delta needs a pinned period column

A delta log is read per **pinned value** (an equality), so each period-scoped table gets a stored
`Period` column and an index `{Type: db.TypeDelta, Keys: db.Cols(e.Period, …, <sync filter>)}`.
A `Computed` column cannot be a delta key (the ORM rejects it). The ORM already logs a record whose
pinned value changed as leaving its old folder, and a `Status = 0` write as a delete (annulled sales
are evicted — fine, the builder skips them, `accounts-consolidation.ts:251`).

- **`Period int16` = YYMM** (Oct 2026 → `2610`), `FixedValues {Min: 0, Max: 9912}`.
- Computed from each document's **accounting date**, one method per record so the Period, the
  same-day rule and the builder agree:

| Table | Accounting date | Delta index Keys |
|---|---|---|
| `sale_order` | `Date` | `(Period, Status)` |
| `invoice_document` | `IssueDate` | `(Period, State)` |
| `purchase_order` | `DocIssueDate` | `(Period, Status)` |
| `expenses` | depreciation (type 4): `Date`; any other: `DocIssueDate`, else `Date` | `(Period, Status)` |
| `warehouse_product_movement` | `Date` | `(Period, Type)` |
| `cash_bank_movements` | `Date` | `(Period, SameDayMovement, Type)` |

- `warehouse_product_movement` and `cash_bank_movements` have no `Version`: add it (TypeDelta
  requires it). They have no `Status` and are never hard-deleted, so `Type` is the sync filter.
- `SameDayMovement` is a **pinned** key, not the sync filter: a delta read returns every value of the
  sync filter, so only pinning keeps same-day rows out of the log the page reads.
- The ORM has no pre-write hook, so every writer of these tables sets `Period` (through the record's
  accounting-date method). The backfill sets it on existing rows.

### 2.2 Same-day payments are inferred, not fetched

`SameDayMovement int8` on `cash_bank_movements`: **1** when the movement alone settles the whole
document on the document's accounting date, at the document's cash bank and currency; **0** otherwise.

| Cash movement type | Document | Can be 1 |
|---|---|---|
| 8 Sale collection | `sale_order` | Yes |
| 6 Supplier payment | `purchase_order` | Yes |
| 9 Expense payment | `expenses` | Yes |
| 10 Asset payment | `accounting_asset` | Yes |
| 11 Sale refund | — (does not pay a document) | Never |
| 1, 2, 3, 4, 5, 7 | none | Never |

The page never downloads a `SameDayMovement = 1` row. It needs a signal **on the document** to know
it must infer one, and the cash bank to post it to (101 vs 1041, and its currency). Hence:

- **`SameDayCashBankID int32` on `sale_order`, `purchase_order`, `expenses`, `accounting_asset`**: the
  cash bank of the movement that settled it same-day; 0 when no movement did. It is the mirror of
  `SameDayMovement`, and both are written together. **Settled.** A bare flag is not enough: the
  inferred entry posts to 101 or 1041 by the cash bank's `Type` and shows its `Name`, and
  `purchase_order`, `expenses` and `accounting_asset` record no cash bank. One column is both the
  flag (≠ 0) and the cash bank, so the two can never disagree. Purely additive: no existing column
  changes (`sale_order.LastPaymentCajaID` stays as is).
- The inferred entry: Dr cash (101/1041 of `SameDayCashBankID`) · Cr the payable/receivable, for the
  document's total, at the same rate as the document. Same lines the real movement would post.

**One pure stamping rule**, `finance/types`: given a document's accounting date, total, currency and
its payment movements, it returns which movement (if any) is the same-day settlement. It is pure so it
can be unit-tested and so no module body imports another: each module's write path calls it, then
writes its own document's `SameDayCashBankID` and the movements' `SameDayMovement`.

Writes that must re-run it (any of them can break the rule):

| Module | File | Write |
|---|---|---|
| sales | `sale_order_create.go` (`PostSaleOrder`) | create + collect, later collection |
| sales | `sale_order_annul.go` (`PostSaleOrderAnnul`) | **every** annulment clears it: the refund is always a real movement, so its collection must be one too (they cancel in 101 and 1212) |
| logistics | `purchase-order-management.go` | supplier payment (~497), `PutPurchaseOrder` edits |
| finance | `expenses.go` | `PostExpensePayment`, `PostExpenses` / `savePaidExpenseDocument` edits |
| accounting | `asset_payment.go`, `asset_edit.go` | asset payment, asset edits |

### 2.3 Self-contained movements (no lookup back to the document)

- **Payments stamp their counter-account.** Types 6, 8, 9, 10, 11 get `CounterAccountCode` written at
  creation (1212, 4212 / 469x, 4654) instead of 0. For supplier and expense payments, officialness is
  then `CounterAccountCode == 4212` (a documented purchase) — no extra column. Collections and refunds
  stay official by the sale ID's series. `ResolveCashMovementAccount` must accept these types. An edit
  that changes a purchase's / expense's `DocType` re-stamps its payments (same write paths as above).
- **Receptions stamp their purchase account.** New `PurchaseAccountCode int32` on
  `warehouse_product_movement`, written for types 3 (order line's `AccountCode`, else 6011) and 4 (the
  expense's `AccountCode`) by `receivePurchaseOrder` and the inventory-expense reception
  (`accounting/inventory_expense.go`). The builder derives the stock/variation pair from it.
- The "No se encontró la venta cobrada / la compra pagada" problems go away: the movement was written
  by its document's own process.

### 2.4 What is still read outside the period (by-IDs cache)

`QueryCachedIDs` (memory → IndexedDB → server; tables opt in with `CacheByIDs: true`). After the
redesign only these remain:

| Table | Why | Fields the builder reads | Route |
|---|---|---|---|
| `sale_order` | A credit note of the period corrects an older sale (`AffectedDocID`) | `Lines[].ProductID`, `Lines[].Amount` | `sale-order-by-ids` exists — move it to `QueryCachedIDs` |
| `purchase_order` | Reception of an older order | `DocType`, `DocSeries`, `DocNumber`, `Name` | new `purchase-order-ids` |
| `expenses` | Supplies reception of an older expense | `DocType`, `DocSeries`, `DocNumber`, `Name` | new `expense-ids` |
| `products` | Every product the sale lines and stock movements name | `ss`, `Origin`, `AccountCode` | `p-products-ids` exists |

A comprobante never needs an out-of-period sale (sale and comprobante share the month: the sale ID
carries the series). Deliveries need no sale (only `DocumentID`). Payments need no document (§2.3).

### 2.5 Company-wide sets

- Assets: the existing delta route (`accounting/asset_api.go`). Depreciation now arrives in the
  period's `Expenses` (type 4, pinned by `Date`) — `loadPeriodDepreciation` is deleted.
- Cash banks: the existing delta route (`finance/cash_banks.go`).
- Sell rates: the existing exchange-rates delta route (`finance/exchange_rate.go`); the per-day
  lookup `SellRatesByDay` is ported to a pure function in `accounts-consolidation.ts`.
- `FirstDay` / `LastDay`: computed in the browser from the period.

### 2.6 The route

`GET accounting-operations?period=YYYYMM` stays one multi-key route; each key has its own cursor,
read with `req.GetDeltaCursor("<Key>")`:

```go
type AccountingOperationsResponse struct {
	SalesDocuments []SalesBookDocument      // invoice_document, (Period, State)
	Sales          []OperationSale          // sale_order, (Period, Status)
	PurchaseOrders []PurchaseBookDocument   // purchase_order, (Period, Status)
	Expenses       []PurchaseBookDocument   // expenses, (Period, Status) — depreciation included
	StockMovements []OperationStockMovement // types 3, 4, 8, 9
	CashMovements  []OperationCashMovement  // SameDayMovement = 0 only
}
```

- `Purchases` splits into `PurchaseOrders` and `Expenses`: one key per table, so one log per key.
- The projections keep the ID and carry `Version` (`ver`) and `Metadata` (`mdt`) through, so the
  delta-cache client can merge them.
- Deleted: `AffectedDocuments` (never read by the page), `Assets`, `Depreciation`, `CashBanks`,
  `SellRates`, `Products`, `Period`/`FirstDay`/`LastDay`, and `loadReferencedSales`,
  `loadReferencedPurchases`, `loadOperationProducts`, `loadOperationAssets`, `loadPeriodDepreciation`.

### 2.7 Frontend

- `accounts-consolidation.svelte.ts`: the period GET with `useCache`; then the by-IDs fetches of §2.4
  for the IDs the period names; the assets / cash banks / exchange-rates services.
- `accounts-consolidation.ts` (pure):
  - `referencedRecordIDs(sources)` — what to ask the by-IDs cache for.
  - `inferSameDayPayments(sources)` — one cash operation per document with `SameDayCashBankID`.
  - The builder reads payments' stamped `CounterAccountCode` and receptions' `PurchaseAccountCode`.
- **Movimientos** tab: derived operations (payments, inferred or not) hidden by default; the Diario
  keeps them. The summary counters still count everything.

## 3. Phases

| # | Phase | Output |
|---|---|---|
| 0 | **Spike** — confirm the delta-cache client merges multi-key projections carrying `ver`/`mdt`, and per-key cursors on a parameterized route | Go/no-go on §2.6 shape |
| 0b | **Baseline** — record Oct 2026 (and one more period with credit notes / older payments) from the current endpoint: counts per kind, Debe = Haber, per-account totals | Numbers to compare against in phase 6 |
| 1 | **Schema** — `Period` + TypeDelta indexes on the 6 tables; `Version` on both movement tables; `SameDayMovement`; `SameDayCashBankID` on 4 documents; `PurchaseAccountCode`; `CacheByIDs` on `purchase_order`, `expenses`, `sale_order`; accounting-date methods; every writer sets `Period`. Run `static-project-validation` | Tables deploy |
| 2 | **Stamping** — pure same-day rule + Go tests; wire every write path of §2.2; payment `CounterAccountCode` and reception `PurchaseAccountCode` at creation and on `DocType` edits | Correct flags on new writes |
| 3 | **Backfill** — `backend/exec/accounting_delta_backfill.go`: Period on the 6 tables, then the same stamping functions over every document with payments, counter-accounts on old payments, reception accounts. Idempotent | Old data in the new shape |
| 4 | **Backend route** — rewrite `GetAccountingOperations` per §2.6; `purchase-order-ids`, `expense-ids`; `sale-order-by-ids` on `QueryCachedIDs`; delete what §2.6 lists | New contract |
| 5 | **Frontend** — service, pure builder changes, by-IDs, Movimientos filter; update `accounts-consolidation.test.ts` | Page on the delta cache |
| 6 | **Verify** — same numbers as phase 0b; reload shows only the delta (network tab); edit a sale / pay a purchase / annul a sale and see only those rows travel; `RATIONALE.md` entries | Done |

### Phase 0 results (2026-10-30)

**Spike — go.** Read in `packages/genix-ui/cache/delta-cache.fetch.ts` and `delta-cache.cursor.ts`:
- Records merge by key per response key (`keyID` / `keysIDs`, default `ID`); a record whose `ver`
  equals the held one is skipped. Metadata records are any record with a string `mdt`, so a
  projection only has to carry `ID`, `ver` and `mdt` through — whole records are not required.
- A logged delete, or a record that left the pinned folder, is resolved log ID → record ID (the ID
  itself for a one-field key), so projections keyed by the table's ID work.
- Only array keys are cached: a scalar key (`Period`, `FirstDay`) would be lost on a cache hit, and
  `SellRates` (an array of numbers) would break the record key lookup. §2.6 already drops all three.
- `PurchaseBookDocument` is keyed by `SourceID`: the two purchase keys set
  `keysIDs: { PurchaseOrders: "SourceID", Expenses: "SourceID" }` (or the projection gains `ID`).
- `useCache.min` is a TTL with **no** network call inside it: use `min: 0` so every open syncs the delta.

**Baseline — current endpoint, company 1, Oct 2026** (run through the page's own builder with an
empty series map; script and raw response in `/tmp/genix-baseline/`). Only Oct 2026 holds data
locally; Sep and Nov are empty.

| | |
|---|---|
| Response | 1774 sales, 1 comprobante, 28 purchases, 10665 stock movements, 1652 cash movements (1622 type 8, 28 type 6, 2 type 7), 394 products — 202 KB JSON |
| Operations | 5060: 1774 sales, 28 purchase orders, 28 receptions, 1652 cash, 1578 cost of sales; 5 official |
| Debe = Haber | 175,145,779 = 175,145,779 (céntimos) |
| Trial balance | 101: 34,227,506 / 27,885,192 · 1212: 35,356,095 / 32,205,345 · 2011: 27,885,192 / 21,906,602 · 40111: 0 / 5,393,641 · 4699: 27,885,192 / 27,885,192 · 501: 0 / 2,022,161 · 6011: 27,885,192 / 0 · 6111: 0 / 27,885,192 · 6912: 21,906,602 / 0 · 7012: 0 / 29,962,454 |

Same-day rule against this data: **all 1622 sale collections and all 28 supplier payments qualify**
(same date as the document, whole total, single payment). After the redesign October's
`CashMovements` key carries 2 rows instead of 1652.

Verification caveats:
- The data changes while the app is used (the screenshot showed 5032 operations; this run 5060).
  Phase 6 re-captures this baseline right before the old endpoint is replaced, with no writes between.
- No credit note, no older-period payment or reception exists locally: those paths are covered by
  unit tests (`accounts-consolidation.test.ts`, the Go stamping tests), or by sample data generated
  with `backend/tests/sample_records`.

## 4. Risks

- **A write path that skips the stamping** leaves a wrong flag: a payment counted twice (inferred and
  fetched) or never. Mitigation: one rule, called from a short list of paths (§2.2); phase 6 checks
  Debe = Haber and 101 against the cash banks' balances.
- **Projections through the delta cache** (phase 0 spike). If the client needs whole records, the
  route returns the records with a `Select` of the needed columns instead.
- **The backfill rewrites many rows** and appends them all to the new logs: run it before any client
  holds a cursor (fresh `useCache.ver`).

## 5. Settled in review

1. `PurchaseAccountCode` is the name of the reception's stamped account.
2. The "not found" problems on payments are dropped (§2.3).
3. Derived operations are hidden by default in Movimientos (no daily summary line); the Diario keeps them.
