# PLAN — accounting module (Assets & Depreciation)

Status: **implemented.** Decisions and their trade-offs are in `RATIONALE.md`.

## 1. The model, in one paragraph

A **supply/material is a Product with `Status = 2`**. The separate `supply_material` table is
deleted. An **asset is a supply that has a depreciation schema** (`Product.DepreciationMonths > 0`).
The asset's *existence* is its stock: a `ProductStock` row when tracked as a group, a
`ProductStockDetail` row when it has a serial number — so assets live in warehouses and move through
the same movement engine as everything else. Its *accounting state* (what it is worth, what has been
depreciated, when it was acquired, when it was disposed of) lives in one thin overlay table,
`accounting.Asset`, because none of it fits in the stock tables and it has to outlive them.

**An asset is not an expense.** Acquiring one writes nothing to `expenses`: its cost, payment state,
supplier and due date live on the `Asset` row, and it is settled through `POST.asset-payment`
against a cash register. The only row an asset ever contributes to `expenses` is its monthly
depreciation.

`Expense.Type` therefore has three values:

| Type | Meaning | Cash? |
| --- | --- | --- |
| 1 | Simple expense — straight to P&L | yes |
| 2 | Inventory purchase | yes |
| 4 | Depreciation entry | **no** — non-cash, `Status = 3` |

`Asset.PurchaseAmount` is what is owed; `Asset.AcquisitionValue` is what the thing is worth. For a
computer the owner donated to the business, `PurchaseAmount = 0` and `AcquisitionValue = 10000`: it
never cost cash, it is never payable, it is still on the balance sheet, and it still depreciates.

## 2. Why the overlay table instead of columns on stock

Three properties of the existing schema forced it, all verified in source:

- **`ProductStockDetail` has no monetary or temporal columns** (`logistics/types/product-stock.go:100-115`)
  — only quantities and `ExpirationDate`. `Created` is a write timestamp, which is the wrong date for
  a backdated or donated asset.
- **The stock key is not a stable identity.** `ProductStockID` packs `WarehouseID+ProductID+PresentationID`
  (`product-stock.go:75-79`), so a warehouse transfer changes the detail row's key. An asset's
  accounting identity must survive being moved.
- **Disposal destroys the stock row.** `ProductStock.SelfParse` sets `Status = 0` when quantity
  reaches zero (`product-stock.go:39-41`), and the delta protocol then evicts it from clients. The
  balance sheet needs disposed assets to remain readable.

`ProductStock` is also the highest-traffic table in the system; four accounting columns used by a
minority of rows do not belong on it.

## 3. Schema changes

All table edits go through `scripts/CREATE_EDIT_TABLE.md`, then `cd scripts && go run . check_tables`.

### 3.1 New table — `accounting.Asset`, `TableSchema.ID = 52`

52 verified free against every `GetSchema()` in the repo; `db.ClaimTableID` panics on collision as a
second check.

| Column | Type | Note |
| --- | --- | --- |
| `CompanyID` | int32 | partition |
| `ID` | int32 | `Autoincrement(0)` — stable across transfers and disposal |
| `ProductID` | int32 | the supply row (`Product.Status = 2`) this instantiates |
| `SerialNumber` | string | empty = grouped asset |
| `Quantity` | int32 | 1 when serial-tracked; N for a grouped acquisition lot |
| `WarehouseID` | int32 | current location — mutable, mirrors the stock row |
| `Name` `Description` | string | the asset's own labels — it borrows nothing from an expense |
| `SupplierID` | int32 | who it was bought from |
| `PurchaseAmount` | int32 | cash owed, in cents. 0 = donated, never payable |
| `PaidAmount` | int32 | server-maintained sum of payments applied |
| `PaymentStatus` | int8 | 0 none (donated) · 1 pending · 2 paid |
| `DueDate` | int16 | UnixDay the purchase is due |
| `AcquisitionDate` | int16 | UnixDay — the real date, not `Created` |
| `AcquisitionValue` | int32 | book value at acquisition, in cents |
| `DepreciationMonths` | int16 | copied from the Product at acquisition, so later catalog edits don't rewrite history |
| `AccumulatedDepreciation` | int32 | server-maintained running sum |
| `LastDepreciationDate` | int16 | UnixDay of the last generated period — dedupes generation |
| `DisposalDate` | int16 | 0 = still held |
| `Status` | int8 | 0 removed · 1 active · 2 fully depreciated · 3 disposed |

Indexes: `TypeDelta` on `Status` (`FixedValues` 0..3), `TypeLocalIndex` on `ProductID`, and
`TypeLocalIndex` on `SerialNumber` for serial lookup.

### 3.2 `Product` (`business/types/productos.go`)

- `+ DepreciationMonths int16` — the depreciation template, on the catalog row, as agreed.
- `FixedValues` on `Status` widen from `{0, 1}` to `{0, 1, 2}`.

Every existing product consumer already pins `Status = 1` — `business/products.go:31`,
`product-ecommerce.go:134`, `product-supply-management.go:17` — so supplies are invisible to product
flows with no further change. **Known cost, accepted:** a deleted supply becomes `Status = 0` and is
then indistinguishable from a deleted product.

### 3.3 `Expense` (`finance/types/expenses.go`)

- `+ Type int8` — 1/2/4 per the table above. Existing rows read as 0; backfill to 1.
- `+ AssetID int32` — set on Type 4 rows, naming the asset being depreciated.
- `+ ProductID int32`, `+ WarehouseID int32`, `+ Quantity int32` — set on Type 2 rows.

There is no `Value` column and no asset type: an asset writes no expense row at all.

Type-4 rows must never enter the payment lifecycle: `GetExpenses` excludes them from the
Pend. Pago / Pagados tabs, and `PostExpensePayment` rejects them.

### 3.4 Deleted

- `logistics/types/supply_material.go` — the whole table.
- `logistics/supply-material-management.go` — both handlers.
- `PurchaseOrder.DetailSupplyIDs / DetailSupplyQuantity / DetailSupplyPrice`. Supply lines become
  ordinary product lines, which is what finally makes them move stock — today they are stored and
  silently ignored (`logistics/purchase-order-management.go:139-155`).

Pre-alpha: deleted outright, no compatibility shim. Any existing `supply_material` rows are
abandoned, not migrated — confirm that is acceptable before execution.

## 4. Depreciation

Straight-line, computed by a pure function in `accounting/depreciation.go`:
`monthly = AcquisitionValue / DepreciationMonths`, with the final period taking the rounding
remainder so the sum equals `AcquisitionValue` exactly and the book value lands on zero, never on -3
cents.

Generation is **lazy**, mirroring `GetExpenseSchedulePeriods` (`finance/expenses.go:344`): opening
the Activos page materializes any missing Type-4 `Expense` rows up to today, deduped by
`Asset.LastDepreciationDate`. No cron, nothing to schedule, and the period walker in
`schedulePeriodDates` (`finance/expenses.go:292`) is reused as a pure function.

Assumed unless corrected: depreciation starts the month *after* `AcquisitionDate`, and a disposed
asset stops depreciating on `DisposalDate`.

## 5. Handlers — `backend/accounting/`

```
types/asset.go        Asset + AssetTable + GetSchema
asset_api.go          GET.assets (delta) · POST.asset · PUT.asset-disposal · PUT.asset-transfer
asset_payment.go      POST.asset-payment — settles the purchase against a cash register
asset_test.go         Pending amount, donated-but-valuable, book value vs. payment
inventory_expense.go  POST.expense-inventory — Type-2 expense + inbound stock movement
depreciation_run.go   POST.asset-depreciation-run · GET.asset-depreciation
depreciation.go       Pure schedule math + the lazy materializer. Unit-testable, no DB.
depreciation_test.go  Rounding, final-period remainder, disposal cut-off, donated (value>0, price 0)
main.go               Handler registration
RATIONALE.md          Per §1 of CLAUDE.md
```

`POST.asset` is the acquisition path and writes **no expense**: it validates the supply, inserts the
`Asset` rows, then applies the inbound stock movement through `ApplyMovimientos` — in that order, so
each movement can carry its own asset as `DocumentID`. `POST.asset-payment` settles the purchase
against a cash register. Registering an asset is **only** done from the Activos page, as specified —
the Expenses page does not create assets.

`access_list.yml` entry 25 ("Activos", group 7) gets `frontend_routes: "accounting/assets"` and its
`backend_apis` filled; today both are empty.

## 6. Frontend

```
routes/accounting/assets/
  +page.svelte          List + the acquisition Layer
  AssetRegister.svelte  Acquire: supply picker or create-on-the-fly, warehouse, serial, value, price
  AssetDepreciation.svelte  Per-asset schedule and book value
  assets.svelte.ts      IAsset, AssetsService (delta), post/dispose/transfer calls
  assets.ts             Pure display math — book value, % depreciated, remaining months
  RATIONALE.md
```

`routes/logistics/supplies-materials/` is repointed at products with `Status = 2` and gains the
depreciation-months field. Its `SupplyMaterialService` is deleted along with the backend route.

The Expenses page gets the `Simple | Inventory` tab split you asked for. **Inventory only** — the
asset case is not reachable from here, per your instruction that assets are registered on the
Activos page. An inventory expense selects a supply (or creates one inline), a warehouse and a
quantity, writes a Type-2 `Expense`, and applies the stock movement.

## 7. Phases

Each phase builds and passes `go build ./... && go vet ./...`, `go test ./accounting/...` and
`cd frontend && bun run check` before the next begins.

1. **Supplies become products.** `Product.DepreciationMonths` + `Status = 2`; delete
   `supply_material`; repoint the supplies page; convert PO supply lines to product lines. Nothing
   accounting-related yet. *This is the largest and riskiest phase — it touches the PO flow.*
2. **Expense.Type.** Add the four columns, backfill existing rows to Type 1, exclude Type 4 from the
   payment tabs and from `PostExpensePayment`.
3. **The Asset table and the pure depreciation math**, with tests, before any UI.
4. **Activos page** — list, acquire, dispose, transfer.
5. **Expenses Simple | Inventory tabs.**
6. Verify against the running app with the `agent-browser` skill.

## 8. What this deliberately does not do

- **No general ledger.** There is still no chart of accounts and no double entry. Type 2/3 rows keep
  a purchase off the P&L, which is what stops an asset being counted twice — once as an expense and
  again as depreciation — but Estados Financieros and Balance remain unbuilt.
- **No revaluation, impairment, or non-straight-line methods.**
- **POs and Expenses stay unlinked**, as you decided: on a PO it needs no Expense, and on an Expense
  it needs no PO.
