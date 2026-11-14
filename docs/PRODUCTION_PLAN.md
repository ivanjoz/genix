# Production Plan: Mercaderías / Productos, recipes, daily close

Status: **implemented, steps 1–5 of §8** (2026-11-13). The accounting entries (§7) still await the
accountant's confirmation. It comes **before** `docs/DINING_PLAN.md`, which
depends on it (DINING D3, D4, D14).

A made product (`Product.Origin = ProductOriginMade`) is sold before it exists, up to its "Sobreventa
permitida" margin (`WebExtraStock`). Its warehouse exit stays pending. At the end of the day a manual
**Cierre del día** produces what was sold:
- it takes the supplies out of stock by the product's recipe;
- it puts the product into stock at its cost;
- it records the pending sale exits;
- it restores the margin.

## Settled decisions

| # | Decision |
|---|---|
| P1 | The catalog splits in two pages: **Mercaderías** (`Origin = 1`, bought to resell) and **Productos** (`Origin = 2`, made by the company). One table, two lists. |
| P2 | A made product has a **"Producción" tab**: the supplies, their quantities, and the portions the recipe yields. |
| P3 | **Cost of a produced unit.** A **standard cost**, when set, overrides everything. Otherwise it is **calculated from the supplies** the close actually consumes, at their moving average, **plus an optional extra cost per unit**. With neither, the product cannot be closed. |
| P4 | The recipe is written **per yield in portions**. |
| P5 | A **company flag** makes the close **strict** with supply stock (a shortage blocks it). When it is off, the available supply is shared among the products in proportion to what they need, the user accepts a summary, and the products are produced short of a supply. |
| P6 | The close is **manual**, per warehouse and per day. |
| P7 | The **accounting entries** of the close are in scope. |
| P8 | **Any sale** of a made product leaves its exit pending: dining, till and web delivery alike. The statuses stay 0–4: the sale is **Entregado (3/4)** for the customer, and a new indexed column **`PendingStockMovement = 1`** marks that its stock exit is still pending. No status 5/6. |
| P9 | Comercial › Catálogo has two entries, **Mercaderías** and **Productos**, the same as Operaciones › Producción. |
| P10 | The supplies come from **the sales' warehouse by default**; the close page lets the user pick another. |
| P11 | **A close cannot be annulled** in v1. A mistake is corrected with manual stock movements. |
| P12 | A recipe holds **supplies only**: no merchandise and no sub-recipes. |
| P13 | The configuration is **flat `Production*` columns on `Product`** (`cb:"48"`–`cb:"51"`); `cb:"47"`, the first nested `Production` struct, is retired. |
| P14 | A close may date its movements **at most one day in the past** (§5.2). |
| P15 | **Supplies get sub-units** (`SbuQuantity`/`SbuUnit`), so a recipe can say "50 g" (§2.4). |
| P16 | The public `p-products-ids` answers **only the fields the storefront needs**; the admin app reads a private `products-ids` (§2.3). |
| P17 | **One recipe per product**, applied to every presentation. |

---

## 1. Mercaderías / Productos split

### 1.1 Data first

- `PostProducts` (`production/products.go:153`) coerces `Origin = 0` to 1 and rejects anything other
  than 1 or 2, so new rows are already clean. **Old rows** can still hold 0.
- `exec/account_code_backfill.go` already sets 0 → 1 (`:91-94`). It is run with `apply` on every
  company before the split, and its output is checked for zero rows left.
- Supplies keep `Origin = 0` (Status 2). The split lists only Status 1.

### 1.2 Backend

- **No new GET.** `GET.products` (`production/products.go:17`) already sends every active product,
  of both origins, to the 13 pages that use `ProductsService`. The two pages filter client-side by
  `Origin`. A server-side filter would split one shared cache in two for no gain.
- `PostProducts` stays as is. It is the page that sets `Origin` on a new row.

### 1.3 Frontend

| File | Change |
|---|---|
| `routes/production/products/ProductCatalog.svelte` (new) | The current `+page.svelte` body (list, form, categories/brands, Excel) moved into a component with an `origin: 1 \| 2` prop. |
| `routes/production/products/+page.svelte` | `<ProductCatalog origin={2} />`, **Productos** |
| `routes/production/merchandise/+page.svelte` (new) | `<ProductCatalog origin={1} />`, **Mercaderías** |
| `services/production/products.svelte.ts` | add `PRODUCT_ORIGIN_MADE = 2` |

Inside `ProductCatalog.svelte`:
- The list filters `productos.records` by `e.Origin === origin` together with the text filter
  (today `+page.svelte:606-608`).
- "New" creates `{ ss: 1, Origin: origin }` (today `:591`, hard-coded to bought).
- The **Origin selector stays in the form** (`:779-789`). Changing it moves the product to the other
  list. The save refuses made → bought while the product has a pending exit (§3.5).
- The **Producción tab** (§2) is in the tab list only when `form.Origin === 2`.
- The Excel import (`products.excel.ts`) sets `Origin = origin` on new rows. Existing rows keep their
  own.
- Page titles and `DOCUMENTATION.md` follow the page. `products/DOCUMENTATION.md` is split into two
  documents (skill `document-user-routes`).

### 1.4 Menu and access

- `db/pages.toml`: a new entry `production/merchandise` ("Merchandise|Mercaderías"). The
  `production/products` entry is renamed "Products|Productos". Its description says made products.
- `db/modules.toml`:
  - Comercial › Catálogo: `routes = "production/merchandise, production/products, crm/customers"`
  - Operaciones › Producción: `routes = "production/merchandise, production/products, production/supplies-materials, production/daily-close"`
- `backend/access.toml`, access **8 "Productos"**:
  - `frontend_routes = "production/products,production/merchandise"`. Matching is by segment prefix,
    so `products` does not cover `merchandise`.
  - `backend_apis` gains `GET.supply-material`. Today that GET is mapped only under access 19
    "Gastos", so a user holding only "Productos" is denied the supply list the recipe needs.
- `frontend/routes/+page.svelte:82` hard-links `/production/products`. It is checked and kept or
  pointed to the right page.

---

## 2. The production configuration

### 2.1 Types (`backend/production/types/product.go`)

`Product` gains four flat columns, each with its `db.Col` in `ProductTable`. Costs are PEN cents,
net of IGV; all four are zero on a product that is not made.

- `cb:"47"` is **retired**. It held the first version, one nested `Production ProductProduction`
  struct, and rows saved while testing still carry that blob. It is never reused.
- `cb:"52"`–`cb:"55"` are reserved by `docs/MANUFACTURING_PLAN.md` (`ProductionLabor`, `TracksLots`,
  `ShelfLifeDays`, `QualityHoldOnEntry`).

```go
// ProductionYieldPortions is how many units of the product the recipe makes. The supply
// quantities are for that many units, so a pinch of salt is expressible without a sub-unit fraction.
ProductionYieldPortions int16              `cb:"48"`
ProductionSupplies      []ProductionSupply `cb:"49"`
// ProductionExtraCostPerUnit is added to the consumed supplies' value, per unit (labor, overhead).
ProductionExtraCostPerUnit int32 `cb:"50"`
// ProductionStandardCost, when > 0, is the unit cost of every production entry, whatever the
// supplies cost. The supplies are still consumed.
ProductionStandardCost int32 `cb:"51"`

// ProductionSupply is one supply of the recipe, for ProductionYieldPortions units of the product.
type ProductionSupply struct {
	SupplyID int32 `cb:"1"` // a Product with Status = 2
	// Quantity is packed like SaleOrderLine.Quantity (Units*QuantityLineScale + Sub), in the
	// supply's sub-unit divisor, recorded so a later refinement of the supply never restates it.
	Quantity   int32 `cb:"2"`
	SubDivisor int16 `cb:"3"`
}
```

`(*Product).HasProductionConfig()` is true when any of the four columns is set.

### 2.2 Validation in `PostProducts`

`ValidateProductProduction(product, suppliesByID)` checks the four columns:

- They must all be zero unless `Origin = 2`.
- When there are supplies: `ProductionYieldPortions > 0`, no supply twice, every quantity > 0, and
  `SubDivisor` equal to the supply's current `SbuQuantity` (or 1).
- Every `SupplyID` is an active supply (Status 2) of the company **whose account has a stock
  account**. A supply on 656 or 33x (expense or asset) has no stock account, so its consumption
  could not be posted (§6).
- `ProductionExtraCostPerUnit` and `ProductionStandardCost` ≥ 0.

### 2.3 Not leaking the recipe

- `GET.p-products-ids` is **public**, and `QueryCachedIDs` selects every non-virtual column
  (`genix-orm/scylla/cache_by_ids.go:211`). Today it already exposes `AccountCode`, `Origin`,
  `WebExtraStock` and the audit fields. With the new `Production*` columns, the recipe and the
  standard cost would join them, readable by anyone who knows a company id.
- New **private** `GET.products-ids` with today's handler: every column. The admin consumers move to
  it:
  - `services/production/products.svelte.ts:118` (`routeByID`);
  - `accounting/invoicing/InvoiceDetailLayer.svelte:58`;
  - `accounting/accounts-consolidation/accounts-consolidation.svelte.ts:55`;
  - `demo/records-by-id.svelte.ts:9`;
  - `system/developer/TestTextIndexes.svelte:95`.
- The **public** `GET.p-products-ids` keeps one consumer, the storefront's `ProductCard.svelte:130`.
  It answers each record **projected onto a whitelist**: the read is unchanged, and the handler copies
  only the allowed fields into a fresh `Product` before answering. The whitelist is:
  - `ID`, `Name`, `Description`, `CategoryIDs`, `BrandID`;
  - `Price`, `Discount`, `FinalPrice`, `CurrencyID`;
  - `SbuQuantity`, `SbuUnit`, `SbuPrice`, `SbuDiscount`, `SbuFinalPrice`;
  - `Presentations`, `ImageMain`, `ImageIDs`, `ImageDescriptions`;
  - `Status`, `Updated`.

  It is a named function `publicProductFields(product)`, with a test that fails if a new `Product`
  field is not classified as public or private. A future column then cannot leak by default. The
  four `Production*` columns are classified private.

### 2.4 Supplies need sub-units

- Today the supply form, `SupplyMaterialPayload` and `GET.supply-material` have **no
  `SbuQuantity`/`SbuUnit`**.
- `PostSupplyMaterial`'s `db.Merge` (`supply_material.go:154-172`) writes those columns as zero on
  every save.
- Quantities are not stored as decimals. They are the pair (whole units, sub-units of a divisor);
  0.750 kg is {0, 750} at divisor 1000 (`core/quantity.go`).
  - A document line, a recipe line included, packs the pair as `Units × 1000 + Sub` in one int32:
    1 kg + 50 g is 1050 at divisor 1000.
  - The divisor is at most 1000 (`MaxQuantityDivisor` = `QuantityLineScale`): grams of a kilo fit,
    milligrams do not.
- A recipe that says "50 g of salt" needs the supply to be "kg, 1000 g". So:
  - `SupplyMaterialPayload` gains `SbuQuantity`, `SbuUnit`.
  - `GET.supply-material` selects them.
  - The form gains the two fields, with the same divisor rules as products
    (`validateProductSubUnitDivisors`).
  - `ISupplyMaterial` gains them.
- The `Merge` keeps excluding the columns a supply does not own. **The four `Production*` columns
  are added to that exclusion list**, so a supply save never writes them.

### 2.5 The "Producción" tab

In the product form `Layer` (tabs at `+page.svelte:620-647`), a new tab `[5, "Production|Producción"]`
shown only when `Origin === 2`, in `routes/production/products/ProductProductionTab.svelte`. It has:

- **Yield** (portions), **extra cost per unit** and **standard cost**: numeric inputs, cents ↔
  decimals as the price fields do.
- **Supplies**: an editable VTable modeled on `routes/crm/settings/PipelinesTab.svelte` (`onCellEdit`,
  `cellSelectMode: 'pick'`, delete button in a `cellRenderer`). Columns: supply (pick from
  `SupplyMaterialService`), quantity (units + sub-units of that supply, packed with
  `core/quantity.ts`), unit, estimated cost.
- **Estimated cost per unit**, read-only: Σ (quantity × supply average) ÷ yield + extra cost, or the
  standard cost marked "Costo estándar".
  - The supply average comes from the existing `GET.products-stock` (`ProductStockSimpleService`,
    every warehouse): Σ `StockValue` ÷ Σ units over the supply's rows.
  - `IProductStock` gains `StockValue` and `SubDivisor`. **No new endpoint.**
  - A supply with no stock value shows "sin costo" and the estimate is flagged incomplete.
- The business rules (estimate, packing, validation messages) live in
  `routes/production/products/product-production.ts`, with `product-production.test.ts`.

---

## 3. Sales: pending stock exits

The statuses stay 0–4. A delivered sale is delivered for the customer, whatever its stock state. The
pending warehouse exit is a separate fact, held in two places:

- **On the sale**, for querying: which sales still owe an exit.
- **On the line**, for the close: which lines. The product's origin is not enough, because it can
  change between the sale and the close.

### 3.1 Types

- `SaleOrder.PendingStockMovement int8 cb:"44"`: 1 = delivered (3/4), but some line's stock exit is
  pending until a close; 0 otherwise.
  - An `int8`, not a bool, so it can pin a delta index: `FixedValues {Col: PendingStockMovement,
    Min: 0, Max: 1}`.
  - A delta index on **`(PendingStockMovement)`**, read with the pinned value 1. It feeds the "Por
    regularizar" tab and the close.
  - The pending set holds only the sales since the last close, so the close reads it and keeps W and
    D in memory. No composite index is needed.
- `SaleOrderLine.StockExitPending bool cb:"9"`: this line's exit is pending. Only lines of made
  products.
- Existing rows default to 0. No sale is pending today, so there is no data fix.
- `AddStatus`, `IsDeliveryPending` and the `Status` `FixedValues` (`Max: 4`) **do not change**.

### 3.2 The delivery paths

`saleDeliveryMovements` (`sale_order_create.go:360`) becomes
`saleDeliveryMovements(sale, madeProductIDs)`:
- It emits exits for bought lines only.
- It sets `StockExitPending = true` on made lines.
- It sets `sale.PendingStockMovement = 1` when any line stayed pending.

The delivery itself still takes the sale to 3/4 through `AddStatus(3)`. It has two callers, and both
read the products already (`readSaleProducts` gains `Origin` in its select):

| Path | Reservation today | Change |
|---|---|---|
| Till sale created with delivery (`PostSaleOrder`, `:167`) | none (it delivers at once) | **made lines reserve** `PendingDeliveryStore` (+1) in the same transaction, on `sale.WarehouseID` |
| Delivery of a stored sale (`PostSaleOrder` update, `:166-172`) | releases all lines from `storedSale.WarehouseID` | releases **bought lines** only; made lines keep theirs. If the delivery warehouse differs from the stored one, the made lines' reservation **moves** (−1 old, +1 new), because the close works by `sale.WarehouseID` |
| Online settle with delivery (`online_sale_order.go:284`) | releases all from `PendingWarehouseID` (Online counter) | same rule, on the Online counter |
| Pay-only update | — | unchanged: 3 → 4 already works, and `PendingStockMovement` is kept |

### 3.3 Stock check (`validateSaleStock`, `:557`)

- Made lines are checked by the **sellable rule**:
  `net − pending ≥ −WebExtraStock`, shared by the product's presentations (`logistics.WebOrderFits`),
  over the sale's warehouse. That is the margin, not physical stock.
- Made lines may not carry a `LotID` or `Sku`: a made product only ever enters the free bucket.
- Bought lines keep today's check.

### 3.4 What changes, and what does not

Every reader of `Status` stays as it is, because a served sale is simply 3/4:
- the sale summary DataFrames (`eachSummaryLine`): the customer has the goods, so nothing is pending
  delivery;
- the Gestión Ventas and Pedidos Web tabs;
- the shopper's labels;
- revenue in accounting (`operationSaleStatuses` = 1–4);
- payment (3 → 4).

What changes:

| Where | Change |
|---|---|
| `sale_order_annul.go:151` | Today it releases reservations only when `IsDeliveryPending()` (1/2). It also releases the reservations of the `StockExitPending` lines when `PendingStockMovement = 1`. The ledger return (`netStockReturns`) already covers only what left. The annulled sale keeps no pending flag (set to 0). |
| `exec/pending_delivery_recompute.go:111-120` | also reserves the `StockExitPending` lines of sales with `PendingStockMovement = 1` |
| `sale_orders_status.go` + `sale_orders_status/+page.svelte` | a new tab **"Por regularizar"**: a delta read of `PendingStockMovement = 1` (`pending-stock=1`). Rows show their pending lines. |
| `routes/sales/SaleOrdersTable.svelte:176` | a badge "R" (stock por regularizar) next to P/E when `PendingStockMovement = 1` |
| `accounts-consolidation` | a period with sales still pending is marked **incomplete** ("ventas con stock por regularizar"): their cost of sales is not booked until the close |
| Tests | `pending_delivery_test.go` (release of pending lines on annulment), a delivery test with mixed bought/made lines |

### 3.5 Other rules

- A product cannot change **made → bought** while any sale has a pending line of it. It cannot
  change **bought → made** while it has free stock. Otherwise the close would produce on top of real
  stock.
- `seeds/sales` assumes a delivery takes every line (`:128-136`). Seeds keep generating bought
  products only. Made products in seeds come with a seed for recipes and closes, outside this plan.

---

## 4. Company flag

`backend/company_flags.toml`, new section. Id 15 is the next free one; 9 is retired.

```toml
[production]
label = "Production|Producción"
15 = { name = "Block the daily close when supplies are short|Bloquear el cierre del día si faltan insumos" }
```

The constant goes in `production/types` (`CompanyFlagStrictSupplyClose = int16(15)`).

---

## 5. Cierre del día

### 5.1 What one close covers

- One **warehouse W** (where the sales left and the products enter), one **day D**, and a **supplies
  warehouse S** (default W, P10).
- It settles every `StockExitPending` line of the sales with `PendingStockMovement = 1`,
  `WarehouseID = W` and `Date = D` **that exist when it runs**. A sale delivered later on day D is left pending, and **the same day
  can be closed again**. Closes are incremental, so selling never has to stop while closing.
- **Oldest first:** a day cannot be closed while W has pending lines on an earlier day.
- D ≤ today.

### 5.2 Movement dates

The stock engine stamps every movement with today (`stock_movement_apply.go:295`,
`core.FechaUnix()`), and no caller can set a date. A close of day D run after midnight would book the
production, the consumption and the cost of sales on D+1. On the last day of a month, they land in
the next month while the revenue stays in the first.

Decision (P14): `InternalMovement` gains an optional `Date int16`.
- It is honoured only for types 15, 16 and the 8 of a close; any other type with a `Date` is refused.
- It may be **today or yesterday**, never earlier and never later. `Period` follows it.
- A close of day D dates its movements **`max(D, yesterday)`**. Closing at night or the next morning
  books on D.
- A day left open longer is still closable, dated yesterday. Its pending lines never get stuck. The
  preview warns "este cierre se registrará con fecha X, no D" whenever the two differ.

### 5.3 The calculation (pure: `production/types/daily_close.go`, with `daily_close_test.go`)

```
sold[p]        = Σ whole units on the pending lines of made product p (a sub-unit line counts its
                 whole unit, as its reservation did). The recipe applies to every presentation (P17).
needed[p][s]   = ceil(recipe(p).quantity[s] × sold[p] ÷ recipe(p).yield), in sub-units of s
needed[s]      = Σ_p needed[p][s]
available[s]   = free bucket + lots of s in S (presentation 0)
consumed[p][s] = needed[p][s]                                   if available[s] ≥ needed[s]
               = floor(available[s] × needed[p][s] ÷ needed[s]) otherwise (not strict only); the
                 remainder goes to the largest share, so Σ_p consumed[p][s] = available[s]
```

Blocking reasons (any mode):
- a product with no cost source (no supplies and no standard cost);
- a product without a standard cost whose consumed supplies are worth 0 (the engine refuses an
  uncosted inflow into a row with no average, `stock_movement_apply.go:517`);
- a supply whose account has no stock account.

Strict mode also blocks on any `available[s] < needed[s]`.

**Lots:** an exit without a lot draws only from the free bucket, and buckets are checked separately
(`:587`). So the consumption of one supply takes the free bucket first, then its lots in ascending
`LotID` order (oldest first). That gives one exit per (product, supply, bucket).

### 5.4 Endpoints

| Route | Access | What |
|---|---|---|
| `GET production-close-pending?warehouse-id` | Cierre de Producción | the days with pending lines in W and their unit counts |
| `GET production-close-preview?warehouse-id&supplies-warehouse-id&date` | Cierre de Producción | §5.3 without writing: per product (units, cost source, estimated value), per supply (needed / available / consumed), blocking reasons, and a `PreviewHash` of the pending line set + consumption |
| `POST production-close` | Cierre de Producción | `{WarehouseID, SuppliesWarehouseID, Date, PreviewHash}` |
| `GET production-closes?warehouse-id` | Cierre de Producción | delta of past closes (history) |

### 5.5 The write: one transaction (`db.RunTransaction`)

1. **Lock the pending sales** of (W, D) (`db.LockByKeys`) and re-read them inside the transaction. A
   concurrent payment (3→4) or annulment of one of them then waits. The stock engine locks only the
   stock rows it touches; there is no company lock.
2. Recompute §5.3. If its hash differs from `PreviewHash`, refuse with "the day changed, review the
   summary again".
3. Insert the `ProductionClose` (`ID` autoincrement). Its ID is the `DocumentID` of the movements.
4. `ApplyMovimientosIn` with the **consumption exits** (type 15), from S, one per (product, supply,
   bucket).
5. Read those ledger rows back by `DocumentID` + type 15, in the same transaction. Their stamped
   `MonetaryValue` is each product's real supply cost. The engine returns only an error, so this is
   how the value reaches step 6.
6. `value[p]` = `ProductionStandardCost × sold`, or Σ its consumption values +
   `ProductionExtraCostPerUnit × sold`.
7. `ApplyMovimientosIn` with the **production entries** (type 16, into W, `InflowValue = value[p]`)
   **followed by** the **sale exits** (type 8, one per pending line, `DocumentID` = the sale). The
   engine applies movements in the given order, and an outflow from an empty row is valued at 0. So
   the entry must come first. The exits then leave at the cost the entry set.
8. `ApplyPendingDeliveryIn` releases the made lines' reservations (Store or Online counter, by the
   sale).
9. Each sale: clear `StockExitPending` on its lines and set `PendingStockMovement = 0`. Its Status
   does not change, so the online order mirror, the client-week rows and the frames are untouched.
10. Update the `ProductionClose` with the products' values.

If any step fails, the transaction rolls back everything.

### 5.6 Table `production_close`

`backend/production/types/production_close.go`. Table ID: the next free one from `check_tables`
(`scripts/validation/check_tables.go:417`). The highest in use is 95; `TABLE_ID_ASSIGNMENT.md` is out
of date.

| Column | Type | Notes |
|---|---|---|
| CompanyID | int32 | partition |
| ID | int32 | autoincrement |
| WarehouseID | int32 | W |
| SuppliesWarehouseID | int32 | S |
| Date | int16 | D (UnixDay) |
| IsStrict | bool | the flag when it ran |
| Products | []ProductionCloseProduct | blob: ProductID, Units, CostSource (1 standard, 2 supplies), Value, ShortSupplies bool |
| Supplies | []ProductionCloseSupply | blob: SupplyID, Needed, Available, Consumed (sub-units + divisor) |
| SaleOrderIDs | []int64 | the sales it settled |
| Status | int8 | 1 |
| Created, CreatedBy, Updated, Version, Metadata | | standard |

Indexes: a delta index on `(WarehouseID)` for the history.

### 5.7 Page `routes/production/daily-close/`

- `+page.svelte` (shell), `DailyCloseSummary.svelte` (the preview tables),
  `daily-close.svelte.ts` (state + calls), `daily-close.ts` (pure: formatting of shortages, totals),
  `daily-close.test.ts`, `DOCUMENTATION.md`.
- Flow:
  - Pick W (and S).
  - The pending days are listed oldest first. The oldest one is selectable.
  - The preview shows the products, the supplies and the blocking reasons. Rows short of a supply are
    highlighted.
  - "Aceptar y cerrar" is disabled while anything blocks.
  - The history of closes sits below.
- `db/pages.toml` gets `production/daily-close` ("Daily close|Cierre del día"). It is listed in
  Operaciones › Producción and in Comercial › Restaurante (DINING §10).
- `backend/access.toml` gets a new access id **60 "Cierre de Producción"** (group 9):
  - `frontend_routes = "production/daily-close"`;
  - `backend_apis = "POST.production-close,GET.production-close-pending,GET.production-close-preview,GET.production-closes"`.

---

## 6. Stock movement types 15/16

`backend/logistics/types/stock_movement_type.go`:

```go
StockMovementTypeProductionConsumption StockMovementType = 15 // a supply used up by a production close
StockMovementTypeProductionEntry       StockMovementType = 16 // a made product produced by a close

StockMovementTypeProductionConsumption: {"Consumo de producción", "production_close", -1},
StockMovementTypeProductionEntry:       {"Ingreso de producción", "production_close", 1},
```

Every place that must learn them:
- `frontend/core/stock-movement-type.ts` (hand-kept mirror: constants + labels). The warehouse
  movements page reads its labels and filter from it.
- `routes/logistics/warehouse-movements/DOCUMENTATION.md`, `products-stock/DOCUMENTATION.md`.
- `stock_movement_type_test.go`.
- `RecalcProductStockByMovements` needs nothing. It replays inflows before outflows within one
  `Created`, so 16 replays before 8, and it keeps recorded values.

---

## 7. Accounting

### 7.1 Backend

- `accounting/accounting_operations_api.go:133` `operationStockTypes` gains 15 and 16. Update the
  `OperationStockMovement` doc comment.
- The delta copy carries `Date`, `DocumentID`, `ProductID`, `MonetaryValue` and
  `PurchaseAccountCode`. That is enough: the supply's account is read from the product (`AccountCode`).

### 7.2 `frontend/routes/accounting/accounts-consolidation/accounts-consolidation.ts`

- **Today every type other than 8/9 goes down the reception path** (`:406-413`). With no
  `PurchaseAccountCode`, that path silently skips the movement. The dispatch becomes explicit:
  - 3 and 4 → reception;
  - 8 and 9 → cost of sales;
  - 15 and 16 → production;
  - anything else → a visible "unhandled movement type" warning instead of a silent skip.
- `operationProductIDs` (`:148`) also loads the products of 15/16. `referencedRecordIDs` (`:136`)
  ignores them, since they have no document to fetch.
- One operation per close and day: key `PR-<closeID>-<date>`, kind `OPERATION_PRODUCTION`, description
  "Production close|Cierre de producción", in the official Diario.

| Movement | Entry |
|---|---|
| 15, supply consumption | Dr the variation account of the supply's account · Cr its stock account (`findPCGEAccount(AccountCode)`: 6032 → Dr **6132** · Cr **252**) |
| 16, production entry | Dr **2111** Productos terminados · Cr **7111** Variación de la producción almacenada |

- New constants `ACCOUNT_PRODUCTION_VARIATION = 7111`; 2111 already exists.
- A supply whose account has no stock account cannot reach a close (§2.2). It is still flagged if it
  appears.
- Tests in `accounts-consolidation.test.ts`: a close with two supplies and a standard-cost product.
  The cost of sales of the settled sales (type 8: Dr 6922 · Cr 2111) needs no change.

**To be confirmed with the accountant**, like the other accounting plans.

---

## 8. Order of work

1. Origin data fix + Mercaderías / Productos split + menus and access (§1).
2. Supply sub-units (§2.4), the `Product.Production*` columns + validation + the private by-IDs route (§2.1-2.3),
   the Producción tab (§2.5).
3. Sales: `PendingStockMovement` + its index, the line flag, delivery paths, stock check, annulment,
   recompute, the "Por regularizar" tab (§3).
4. Flag, movement types, `production_close`, the movement date (§5.2), preview + close + page (§4-6).
5. Accounting (§7).
6. Dining (`docs/DINING_PLAN.md`).

Each step leaves the system working: steps 1–2 change nothing for sales, and step 3 only matters
once a made product is sold.

---

## 9. Found during the review, outside this plan

- `docs/SALE_COST_PLAN.md:40` and `backend/logistics/RATIONALE.md:149` speak of a "company lock" in
  the stock engine. It locks stock rows only.
- `docs/TABLE_ID_ASSIGNMENT.md` says "nothing implemented", but `check_tables` already lists free IDs.
  Its range table is stale too.
- Company flags 1, 2, 3, 4, 5 and 8 (`[comercial]`) are read by no code.
- A pay-only `PostSaleOrder` update persists a request `WarehouseID` (`sale_order_create.go:81-83`),
  which can move a sale away from its reservation. It is reachable only by a direct API call.

---

## 10. Pending decisions

None. P14–P17 settled the review's points.
