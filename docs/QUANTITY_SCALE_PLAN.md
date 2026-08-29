# Fractional quantities and sub-units — analysis and refactoring plan

**Status:** implemented, phases 1–7 and 9. Phase 8 (wipe + reseed) is an operational step
for the human — it needs a live database. Remaining known gap is listed in §6.
**Scope:** `business` (product catalog), `logistics` (stock, ledger, purchase orders),
`sales` (orders, summaries), `invoicing` (SUNAT bridge), and every frontend route that
enters or renders a quantity.

**Design decided** (see §4 for how it was reached):

| Decision | Value |
| --- | --- |
| Aggregates (stock, ledger, summaries) | `(Quantity, SubQuantity)` split — two `int32` columns |
| Documents (sale-order lines) | packed `Units*1000 + Sub` in one `int32` |
| `Quantity` | whole base units, **unscaled** |
| `SubQuantity` | count of sub-units, **unnormalized** (may exceed the divisor, may be negative) |
| Divisor | stored per movement, per stock row and per document line; `Product` holds the current one |
| `MaxQuantityDivisor` | 1000 — a normalized `Sub` must fit the packed slot's 0..999 |
| Divisor changes | refinement only — the new divisor must be a **multiple** of the old |
| Adding a sub-unit to an existing product | **no-op**, zero rows rewritten |
| Migration | wipe all ERP data and reseed (alpha) |

---

## 1. What the code does today

### 1.1 Quantities are whole units, everywhere

Every quantity is an `int32` counting whole base units. There is no scale factor anywhere:
`sales.SaleOrder.DetailQuantities`, `sales.SaleOrderProductStats.Quantity`,
`logistics.PurchaseOrder.DetailProductQuantity`, `logistics.ProductStock.Quantity` /
`.DetailQuantity` / `.DetailComputedQuantity`, `logistics.ProductStockDetail.Quantity`,
`logistics.WarehouseProductMovement.Quantity` / `.WarehouseQuantity`,
`logistics.InternalMovement.Quantity`, `logistics.ProductSupply.MinimunStock` /
`.SalesPerDayEstimated`, `business.WarehouseStockMin.Quantity`,
`invoicing.InvoiceDocument.DetailQuantity`.

### 1.2 Sub-units are inert, and where they are wired they corrupt stock

`Product` carries `SbuQuantity`, `SbuUnit`, `SbuPrice`, `SbuDiscount`, `SbuFinalPrice`
(`business/types/productos.go:56-60`), editable on the products form and shown in the list.
**No backend code reads any of them.**

The only consumer is the sale-order page, and it is broken end to end:

- `sale_order_create/+page.svelte:210-223` synthesizes a `<key>_s` cart row for any product
  with `SbuQuantity > 1` and sets its **available stock to the literal `SbuQuantity`** — a
  constant, not `stock × SbuQuantity`. 40 boxes with a 6-per-box sub-unit offers 6 sub-units
  for sale, forever.
- `sale_order.svelte.ts:213` prices it at `SbuFinalPrice` and pushes the sub-unit count
  straight into `DetailQuantities` with no conversion.
- `sales/sale_order_create.go:189` hands that number to `ApplyMovimientos` as
  `Quantity: -cantidad`. **Selling 3 candies deducts 3 boxes.**

### 1.3 The `SubQuantity` plumbing already exists and already has the right shape

`ProductStock.SubQuantity` / `.DetailSubQuantity` / `.DetailComputedSubQuantity`,
`ProductStockDetail.SubQuantity`, `WarehouseProductMovement.SubQuantity` and
`InternalMovement.SubQuantity` are threaded through the whole stock engine. No producer
ever sets a non-zero value and no UI enters one
(`products-stock/DOCUMENTATION.md:96` says so), so it is dead today — but
`stock_movement_apply.go` **already accumulates it exactly the way this design needs**:

```go
stock.SubQuantity = prevSubQuantity + mov.SubQuantity        // component-wise, no carry
s.subQuantity += d.SubQuantity                                // detail re-sum pass
accumulate(wh, movement.Quantity, movement.SubQuantity, mov)  // ledger replay
```

The engine is largely built. What is missing is the divisor, normalization at the
boundaries, validation, and a producer.

### 1.4 The sale-order handler trusts the client

`PostSaleOrder` never loads a `Product` row — `validateSaleStock` queries `ProductStock`
only. It checks that the detail slices are the same length and that no value is zero
(`sale_order_create.go:68-75`); `DetailPrices` and `TotalAmount` come straight off the
request body and are never checked against `Product.FinalPrice`. **A client can post a sale
at any price today.** This violates `CLAUDE.md` §9 and is fixed as a side effect of §5.

### 1.5 facturago already has fixed-point quantities

`facturago/model/money.go:23-31` — `Quantity` is an `int64` scaled by
`QuantityScale = 1_000_000`, and `model/totals.go:87` already does
`MulDivRound(Quantity, unitValue, QuantityScale)`. The ERP throws it away at the boundary:
`invoicing/sale_order_to_cpe.go:184` does `model.Units(int(quantity))` and
`invoicing/emitter.go:112` does `int32(line.Quantity / model.QuantityScale)` — a lossy
round trip that assumes whole units. This is where a wrong number becomes a legal document.

### 1.6 Money is already fixed-point, and the UI primitive exists

Money is `int32` cents (`formatMo = n => formatN(n/100, 2)`), and
`genix-ui/form/Input.svelte` has `baseDecimals`, a generic fixed-point entry prop.

---

## 2. The model

### 2.1 Representation

A quantity is a pair of `int32` columns plus a divisor:

```
Quantity     — whole base units (boxes, kilograms, pieces). Unscaled.
SubQuantity  — count of sub-units. Not normalized: may exceed the divisor, may be negative.
Divisor      — how many sub-units make one base unit.
```

The real value is `Quantity + SubQuantity / Divisor`, an exact rational. `0.750 kg` at
divisor 1000 is `(0, 750)`. `1 box + 4 candies` at divisor 6 is `(1, 4)`. A product with no
sub-unit is `(N, 0)` at any divisor.

**`Quantity` is never scaled, for any product.** All fractional information lives in
`SubQuantity`/`Divisor`, so there is no per-product "is this discrete or continuous"
classification to make at creation time and get wrong later. Continuous goods are simply
divisor 1000.

### 2.2 Aggregates split, documents pack

Two representations, one conversion point. Anything that is **accumulated or summed** keeps
the halves in separate columns; anything **written once and read back whole** packs them.
`PostSaleOrder` converts between them with `core.UnpackQuantityLine`.

| | representation | why |
| --- | --- | --- |
| `WarehouseProductMovement`, `ProductStock`, `ProductStockDetail`, `SaleOrderProductStats` | split | accumulated with a plain `+`, and `SUM()`-ed by Scylla |
| `SaleOrder.DetailQuantities` | packed | a document — never accumulated, so it pays one column instead of two |

The rest of this section is about the split, since that is where the arithmetic happens.

Component-wise addition is **closed without carrying**:

```
(a₁,b₁) + (a₂,b₂) = (a₁+a₂, b₁+b₂)
```

No borrow, no normalization, order-independent. Worked example — `+1 box`, `+1 box`,
`−4 candies`, `−4 candies` at divisor 6:

```
SUM(Quantity) = 2 ,  SUM(SubQuantity) = −8  →  2 − 8/6 = 4/6
```

By hand: 2 boxes = 12 candies, sold 8, 4 remain = 4/6 box. ✓

Two consequences that decide the design:

- The hot path (`ApplyMovimientos`, `RecalcProductStockByMovements`) stays a plain integer
  `+`. No decompose/normalize/recompose per movement, and no signed-borrow logic in the
  code that computes what is in the warehouse.
- **`SUM()` still pushes down to Scylla** on both columns independently
  (`product-supply-management.go:171`). A packed `whole×1000+sub` integer would make
  `SUM` wrong (`1004+1004 = 2008`, unnormalized) and force that query back into Go.

Neither argument applies to a document line, which is why sale orders pack: nothing ever
adds two `DetailQuantities` together or asks the database to sum them.

Normalization is confined to validation and display boundaries, where it is isolated and
unit-testable.

### 2.3 Adding a sub-unit is a no-op

The operation that happens **often** is an operator opening a product that already has
stock and sales and deciding to start selling it in pieces. Under this model that rewrites
nothing, because every historical row already has `SubQuantity = 0`, and zero sixths, zero
twelfths and zero thousandths are the same zero:

| | rows rewritten |
| --- | --- |
| `ProductStock` / `ProductStockDetail` | 0 |
| `WarehouseProductMovement` | 0 |
| `SaleOrder.DetailQuantities` | 0 |
| `ProductSaleSummary` | 0 |

Set the divisor on the product; done.

### 2.4 Changing a divisor is lazy, and refinement-only

The **infrequent** operation is changing an existing divisor. A stock row stores the
divisor it was last written under and converts on the fly when a movement arrives with a
different one:

```
stock (21, 4) divisor 6  →  4/6 = 8/12  →  (21, 8) divisor 12  →  + (0,2)  →  (21, 10) divisor 12
```

Exact, because 6 divides 12. This is exact **only when the new divisor is a multiple of the
old**:

| change | example | exact? |
| --- | --- | --- |
| 6 → 12 | `4/6 = 8/12` | yes |
| 1000 → 2000 | higher precision later | yes |
| 12 → 6 | `9/12 × 6 = 4.5` | **no** |
| 6 → 5 | `4/6 × 5 = 3.33` | **no** |

**Rule: a divisor may be refined by an integer factor, never coarsened, never changed to a
non-multiple.** Enforced at product save with a descriptive error naming the valid
divisors. Coarsening requires a deliberate, separate operation and is out of scope here.

### 2.5 Pricing a sub-unit line

The line amount for a sub-unit sale is `SbuFinalPrice × subCount`, **not**
`price × sub / divisor`. Selling 4 candies from a 5000-cent box is 3333.33 cents the second
way; the first is exact and uses a column that already exists on `Product`.

---

## 3. Wire format and server-side resolution

A document line packs both halves into one `int32` and carries its divisor alongside:

```
DetailQuantities:  []int32   // Units*1000 + Sub — 1001 at divisor 6 is one box plus one candy
DetailSubDivisor:  []int16   // divisor in effect at entry
```

**Documents pack, aggregates split.** A sale order is written once and read back whole, so it
never needs the component-wise accumulation or the Scylla-side `SUM()` that the movement
ledger does — it pays one column instead of two. `WarehouseProductMovement`, `ProductStock`
and the summaries keep `Quantity` and `SubQuantity` in separate columns, because those are
accumulated and summed. `PostSaleOrder` is the single conversion point:
`core.UnpackQuantityLine` on the way in, feeding `InternalMovement{Quantity, SubQuantity,
SubDivisor}`.

The packing slot is 3 digits, which sets two ceilings — both per-line, neither reaching any
aggregate:

- **`MaxQuantityDivisor` = 1000.** A normalized `Sub` must fit 0..999. A kilogram sold by the
  gram is exactly 1000 and fills it. The cost is that such a product can never be refined
  further: there is no room for half-grams.
- **`MaxQuantityLineUnits` = 2,147,483** whole units on one line.

`PostSaleOrder` loads the `Product` rows, validates `DetailSubDivisor` against the
product's current divisor, rejects `SubQuantity` on a product with no sub-unit, and
validates `DetailPrices` against `FinalPrice`/`SbuFinalPrice` — closing the price hole in
§1.4. This adds a `Product` query to a handler that currently only queries `ProductStock`.

`DetailSubDivisor` is persisted on the order so the invoice and any re-render describe the
line the way it was sold, without re-deriving it from the product's current divisor.

---

## 4. Alternatives considered and rejected

**Conditional scale — `1` means milli-units, or one sub-unit, depending on whether the
product has a sub-unit.** The stored value stops being self-describing: every consumer
(stock, ledger, summary, invoice, charts, storefront) would need the product row to do
arithmetic, and `RecalcProductStockByMovements` scans a company's whole ledger. Worse, the
meaning is mutable — saving a sub-unit name on a product retroactively rescales all of its
history by 1000× with no migration and no way to repair it, because the residue is not
recoverable.

**`floor(1000/N)` — one sub-unit is `floor(1000/N)` milli-units.** Inexact whenever `N` does
not divide 1000, and `floor` always biases one way, so the error is monotonic *phantom
stock*:

| N | floor(1000/N) | N × that | stranded/unit | drift |
| --- | --- | --- | --- | --- |
| 2, 4, 5, 8, 10, 20, 25, 40, 50, 100, 125, 250, 500 | — | 1000 | 0 | exact |
| 3 | 333 | 999 | 1 | 0.1% |
| 6 | 166 | 996 | 4 | 0.4% |
| 7 | 142 | 994 | 6 | 0.6% |
| 12 | 83 | 996 | 4 | 0.4% |
| 24 | 41 | 984 | 16 | 1.6% |

At N=6, selling 250 boxes one candy at a time invents one box that does not exist, and it
never self-corrects. The proposed "6 × 166 must equal 1" reconciliation cannot be
implemented: a balance of `39004` is 39 boxes plus 4 unexplainable milli-units, and `830`
is either 5 candies or 0.83 kg — the balance has no memory of which. Recovering the intent
means replaying the ledger with a per-movement marker recording "this was N sub-units" —
at which point the sub-unit count has been stored, which is what §2.1 does directly.

**Packed mixed-radix everywhere — `whole × 1000 + sub` in one column, including stock and
the ledger.** Rejected *for aggregates* because the field stops being a number you can add:
`stock.Quantity = prevQuantity + mov.Quantity` computes `1000 − 4 = 996` (0 units + 996
sub-units, invalid, ~498× too large, and it passes the existing `< 0` guard) where the
answer is `2`. Every `+`, `−` and comparison would become decompose/normalize/recompose
with signed borrow, in the two hottest write paths in the system, and Scylla-side `SUM()`
would die.

None of that applies to a **document** line, which is neither accumulated nor summed — so
sale orders do pack (§2.2). The rejection is scoped to the tables that aggregate.

**Base unit = smallest sellable unit, pack factor as a display lens, with a `× N` ledger
refinement when a sub-unit is added.** Exact and single-column, but it puts the expensive
migration on the *frequent* operation (§2.3) and makes the *rare* one cheap — backwards.
A product that gains a sub-unit would need its entire ledger rewritten.

**`int64 × 1_000_000` matching facturago.** Removes every headroom concern and makes the
SUNAT bridge an identity cast, but §2.1 keeps `Quantity` unscaled, so `int32` headroom is
already ~2.1 billion whole units — the same magnitude as today. The extra width buys
nothing here. Note the ORM's deploy only *reports* a column type mismatch
(`genix-orm/scylla/deploy.go:952`; the only `ALTER` it emits is `ADD`, line 969) and Scylla
will not convert `int` → `bigint` in place, so a type change means recreating tables.

**Two fields kept normalized on every write (carry when `sub ≥ divisor`).** Loses the
closed-addition property of §2.2 for no benefit — normalization is only needed at the
boundaries.

---

## 5. Execution plan

Ordered so the system compiles and the §6 checks pass at every step.

### Phase 0 — decisions taken

1. **`ProductSupply` thresholds stay whole units.** `MinimunStock`, `SalesPerDayEstimated`
   and `ProviderSupply[].Capacity` get no `SubQuantity` companion. They are reorder
   heuristics, not balances, so whole-unit precision is enough. Comparisons against stock
   truncate the sub-unit part.
2. **`finance.Expense.Quantity` and `accounting.Asset.Quantity` stay untouched.** They
   count whole indivisible things.
3. **Purchase orders get no sub-unit entry.** Receiving stays whole-unit, so
   `PurchaseOrder` gains no `DetailSubQuantity` / `DetailSubDivisor`, and the reception
   path posts `InternalMovement{SubQuantity: 0}`. Sub-units are a sales-side concept in
   this pass.
4. **The two cross-product sums** (§6) are resolved as follows:
   - **`PurchaseOrder.DifferenceQuantity` is deleted.** It and `DifferenceValue` are both
     write-only — assigned at fulfillment (`purchase-order-management.go:161-162`),
     persisted, and read by nothing in the repo. `DifferenceQuantity` sums across product
     lines, so it has no unit; nothing reads it, so nothing needs one.
     `DifferenceValue` is kept: money sums correctly across products and a purchase-order
     report may want it.
   - **The daily-summary delivery ratio becomes money-based.** Compute the delivered
     *amount* per product (`totalAmount × delivered/total` — the ratio is unit-free within
     one product) and sum the amounts. Same line count, and money is summable.
   - **The `quantity` metric mode is dropped from the daily summary.** Its top-products
     ranking is per-product and would survive, but its paid/unpaid totals sum across
     products. `SaleOrdersChartsByProduct` keeps the quantity mode, where every figure is
     already per-product.

### Phase 1 — the contract

- `core` gains the quantity type and its helpers, with unit tests:
  `core.Quantity{Units, Sub int32}`, `Normalize(divisor)`, `Compare(a, b, divisor)`,
  `IsNegative(divisor)`, `ToRational`, `RefineDivisor(from, to)` (rejects non-multiples).
  One place, imported everywhere, grepable.
- Write the `RATIONALE.md` entries for `backend/logistics` and `backend/sales` **before**
  the mechanical edits — that file is the review surface.

### Phase 2 — the divisor on the data model

- `Product` gains the divisor as the authoritative current value; redefine `SbuQuantity`
  unambiguously as **sub-units per base unit** (today the fixture reads `SbuQuantity = 6,
  SbuUnit = "caja"` while the sale page treats it as available stock).
- `WarehouseProductMovement` gains `SubDivisor int16` — without it the ledger replay cannot
  interpret rows written under an older divisor, and `SUM(SubQuantity)` would add sixths to
  twelfths.
- `ProductStock` gains `SubDivisor int16` (the "last divisor" of §2.4).
- `product-supply-management.go:171` adds the divisor to the group key
  (`GroupBy(Date, ProductID, Type, SubDivisor, Quantity.Sum(), SubQuantity.Sum())`) and
  combines per-divisor groups in Go. This changes the packed view backing that query.
- Enforce refinement-only at product save.

`SubQuantity` is **retained**, not deleted — §1.3.

### Phase 3 — fix the three engine defects

1. **Negative-stock validation silently allows overselling.**
   `stock_movement_apply.go:293` (`stock.Quantity < 0 || stock.DetailQuantity < 0`) and
   `:305` (`detail.Quantity < 0`) miss `(Quantity=0, SubQuantity=−4)`, which is −4/6 units
   oversold and passes because `0` is not `< 0`. Must become a normalized comparison.
2. **`ProductStock.SelfParse` drops stock that exists.**
   `product-stock.go:40` — `Status = If(Quantity == 0 && DetailQuantity == 0, 0, 1)` marks
   a row holding 0 boxes and 4 loose candies as "no stock", evicting it from the delta
   sync. `SubQuantity` must enter the predicate. (`ApplyMovimientos` already handles this
   correctly for *details* but not for the stock row.)
3. **`MonetaryValue`** (`stock_movement_apply.go:254`) becomes
   `Quantity × Price + SubQuantity × Price / Divisor`, guarded against `int32` overflow.

### Phase 4 — carry the pair through the domain

- `SaleOrder.DetailQuantities` becomes the packed form and `SaleOrder` gains
  `DetailSubDivisor []int16`. No `DetailSubQuantity` column: the sub-unit rides in the
  packed value. `PurchaseOrder` gains nothing (Phase 0 decision 3).
- `SaleOrderProductStats` gains `SubQuantity` and `SubQuantityPendingDelivery`.
  `libs.SerializeInt30Struct` packs any integer field, so this is additive — but note it
  saturates at `2^30−1` **without erroring** (`normalizeInt30Unsigned`), so the new fields
  inherit that ceiling.
- `sale_summary.go:37` — `lineAmount` accounts for the sub-unit part per §2.5.
- `purchase-order-management.go:130-131` — delete `diffQuantity` and the
  `DifferenceQuantity` column (Phase 0 decision 4); `diffValue` stays as-is, since
  receiving is whole-unit.

### Phase 5 — the sale-order handler

- `PostSaleOrder` loads `Product`, resolves and validates the divisor, and validates
  `DetailPrices` against `FinalPrice`/`SbuFinalPrice` (§3, closing §1.4).
- `validateSaleStock` compares normalized quantities at a common divisor.
- `InternalMovement` carries `SubQuantity` and `SubDivisor` from both the sale and purchase
  paths.

### Phase 6 — frontend

- Delete the synthesized `_s` cart row (`+page.svelte:210-223`) and its flat
  `cant: SbuQuantity` stock ceiling. Replace with a per-line unit selector on the real row.
- Send `DetailQuantities` / `DetailSubQuantity` / `DetailSubDivisor` as entered; the server
  resolves.
- Adopt `Input baseDecimals` for quantity entry and add a `formatQty(units, sub, divisor)`
  helper next to `formatMo`.
- `products-stock` and `warehouse-movements` render the pair. `purchase-orders` stays
  whole-unit (Phase 0 decision 3).
- `SaleOrdersChartsDailySummary.svelte` — switch the delivery ratio to money and drop the
  `quantity` metric mode (Phase 0 decision 4). `SaleOrdersChartsByProduct` keeps it.

### Phase 7 — the SUNAT boundary

`sale_order_to_cpe.go:184` converts the pair to `model.Quantity` at
`QuantityScale = 1_000_000` instead of `model.Units(int(quantity))`; `emitter.go:112` stops
truncating; `InvoiceDocument.DetailQuantity` carries the sub-unit part. `model.Line.UnitCode`
is per-line, so a sub-unit sale invoices in its own unit. Extend `emitter_test.go` with a
fractional case and validate the XML through facturago.

### Phase 8 — reseed and verify

`generate_erp_history` and `generate_sale_orders` emit sub-unit lines; wipe and reseed.
Then: `cd backend && go build ./... && go vet ./... && go test ./...`,
`cd scripts && go run . check_tables`, `go run . check_module_imports`,
`cd frontend && bun run check`, and the `agent-browser` skill against the sale-order,
products-stock and purchase-order pages.

### Phase 9 — docs

`frontend/routes/business/products/DOCUMENTATION.md` (the sub-unit description is wrong in
a new way), `sale_order_create/DOCUMENTATION.md`, `products-stock/DOCUMENTATION.md` (drop
the "no field for SubQuantity" note), and the `(Quantity, SubQuantity, Divisor)` convention
into `CLAUDE.md` §9 next to UnixDay/SUnixTime.

---

## 6. Open risks

- **Silent saturation.** `SerializeInt30Struct` truncates at `2^30−1` without erroring and
  Scylla's `SUM` on an `int` column wraps. Neither can report a problem at the moment it
  happens. `Quantity` stays unscaled so both keep today's magnitudes, but the new
  `SubQuantity` accumulators are unnormalized and can grow large before anyone normalizes
  them — they need an explicit guard.
- **Two cross-product quantity sums were already unit-nonsense** before this work — both
  add kilograms to screws today. Resolved in Phase 0 decision 4:
  `SaleOrdersChartsDailySummary.svelte:129` (`totalQuantity += totalQuantityForProduct`
  across products in a day, rendered at :302) moves to money, and
  `purchase-order-management.go:130` (`diffQuantity`) is deleted along with its column,
  since nothing reads it.
- **Coarsening a divisor is unsupported.** An operator who goes 12 → 6 must be refused. If
  that turns out to be a real workflow it needs its own design — it is lossy by nature.
- **The `Product` query added to `PostSaleOrder`** changes the shape of the hot sale path.
  It is required to stop trusting client prices, but it is a new dependency in that handler.
- **`Product.Stock []WarehouseStockMin`** rides in a serialized blob column, so it needs no
  `ALTER`, but the storefront reads it — the public webpage bundle must be checked for
  quantity rendering, and must not gain a `$core/security` import in the process.
