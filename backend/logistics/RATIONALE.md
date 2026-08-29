# RATIONALE — logistics

Design decisions for supplies, stock and purchase orders, newest first.

## The sub-unit divisor lives on the movement, and may only be refined

**Context** — With quantities as a `(Units, Sub)` pair (see `backend/RATIONALE.md`), `Sub` is
meaningless without knowing how many sub-units make a unit. The operation that happens often is an
operator opening a product that already has stock and sales and deciding to start selling it in
pieces; changing an existing divisor is rare.

**Decision** — `WarehouseProductMovement` and `ProductStock` each carry a `SubDivisor`, with
`Product` holding the current authoritative one. A stock row converts itself on the fly when a
movement arrives at a finer divisor: `{21,4}` at 6 becomes `{21,8}` at 12, then the movement lands
with a plain add. A divisor may only be **refined to a multiple** — 6 to 12 is allowed, 12 to 6 and
6 to 5 are refused at product save.

**Rationale** — the divisor has to be on the movement, not only the product, or the ledger replay in
`RecalcProductStockByMovements` cannot interpret rows written under an older divisor and
`SUM(SubQuantity)` would add sixths to twelfths. With it there, the aggregate becomes
`GROUP BY SubDivisor` and stays pushed down.

Refinement-only is what makes the conversion exact: 6 to 12 splits every sixth into two twelfths and
loses nothing, while 12 to 6 turns 9/12 into 4.5 sixths and would silently round away real stock.

Adding a sub-unit to a product that already has history rewrites **nothing** — every existing row has
`SubQuantity = 0`, and zero sixths, zero twelfths and zero thousandths are the same zero. The
frequent operation is free; only the rare one does lazy work, and it touches one stock row rather
than the ledger.

Detail rows carry no divisor of their own: they inherit the parent stock row's. That is forced by
`ProductStock.DetailSubQuantity` being their **absolute re-sum** — per-detail divisors would make it
add sixths to twelfths. So refining a stock row converts its whole detail set atomically, and when
the triggering movement was free-bucket only (leaving the details unloaded) `ApplyMovimientos` loads
them first. The extra query fires only on an actual refinement, which is rare by design.
`RecalcProductStockByMovements` does the same mid-replay, since the ledger can span a refinement.

**Costs accepted:** coarsening a divisor is unsupported and must be refused; if it turns out to be a
real workflow it needs its own design, because it is lossy by nature. The `GROUP BY SubDivisor`
change also touches the packed view backing `product-supply-management.go`. `MaxQuantityDivisor` is
1000, set by the packed sale-line slot (see `backend/RATIONALE.md`) rather than chosen; the view's
4-digit key slot accommodates it with room to spare.

## `SubQuantity` was dead plumbing and is now load-bearing

**Context** — `ProductStock.SubQuantity`, `.DetailSubQuantity`, `.DetailComputedSubQuantity`,
`ProductStockDetail.SubQuantity`, `WarehouseProductMovement.SubQuantity` and
`InternalMovement.SubQuantity` were threaded through the whole stock engine, summed, rolled up and
persisted — but no producer ever set a non-zero value and no UI entered one. An earlier abandoned
pass at the same problem.

**Decision** — kept, not deleted, and given a producer. `stock_movement_apply.go` already accumulated
them component-wise (`stock.SubQuantity = prevSubQuantity + mov.SubQuantity`), which is exactly what
the pair representation needs.

**Rationale** — the accumulation was already correct for this design; what was missing was the
divisor, normalization at the boundaries, and validation. Three pre-existing defects had to be fixed
for it to be safe:

- `stock_movement_apply.go` validated `stock.Quantity < 0`, which misses `{0,-4}` — four candies
  oversold, whose whole-unit field is exactly zero. Now a normalized comparison.
- `ProductStock.SelfParse` derived `Status` from `Quantity == 0 && DetailQuantity == 0`, marking a row
  holding 0 boxes and 4 loose candies as "no stock" and evicting it from the delta sync.
  `SubQuantity` now enters the predicate. (`ApplyMovimientos` already handled this for *details*.)
- `MonetaryValue` ignored the sub-unit part entirely.

## Purchase orders stay whole-unit

**Context** — Receiving stock could have accepted sub-units symmetrically with sales.

**Decision** — it does not. `PurchaseOrder` gains no `DetailSubQuantity` / `DetailSubDivisor`, and the
reception path posts `InternalMovement{SubQuantity: 0}`. Sub-units are a sales-side concept.

**Rationale** — stock is bought in whole units and broken down when sold, which is the actual
workflow; supporting it on both sides would have doubled the surface for no case anyone has. The
consequence is that a shop buying boxes and selling candies accumulates a negative sub-unit balance
against whole-unit inflows — `{2,-8}` at divisor 6 is 4/6 of a box — which the pair handles correctly
and which is why the negative-stock fix above matters more than it otherwise would.

`PurchaseOrder.DifferenceQuantity` was deleted rather than taught about sub-units: it summed
quantities across product lines, so it had no unit, and nothing in the repo read it.
`DifferenceValue` is kept — money sums correctly across products.

## A supply is a Product with Status = 2, and `supply_material` is gone

**Context** — `supply_material` (table 32) was a parallel catalog to `products`, with its own name,
SKU, price, brand and provider rows. It had no stock: `warehouse_product_stock` and
`warehouse_product_movement` are keyed on `ProductID`, and `PostPurchaseOrderEntry` only built
movements from `DetailProductIDs`. The `DetailSupply*` arrays on a purchase order were stored and
then silently ignored — ordering supplies moved nothing.

**Decision** — Supplies became `Product` rows with `Status = 2`. `supply_material` is deleted, and so
are `PurchaseOrder.DetailSupplyIDs / DetailSupplyQuantity / DetailSupplyPrice`: a supply line is now
an ordinary product line. `MinimunStock` and `ProviderSupply` moved to the existing `product_supply`
table, which was already keyed by `ProductID` and already held exactly those two fields.

**Rationale** — This is what makes supplies have stock at all, and it costs no new code: they flow
through the movement engine, the lot and serial handling, and the purchase-order reception path that
products already use. The alternative — a second `supply_stock` / `supply_stock_movement` pair —
would have duplicated the whole engine for a catalog that differs from products only in not being
sold.

Every product consumer already pins `Status = 1` (`business/products.go:31`,
`product-ecommerce.go:134`, `product-supply-management.go:17`), so supplies are invisible to product
flows without a single change to those queries.

**The accepted cost:** `Status` is now both the lifecycle flag and the type discriminator. A deleted
supply is `Status = 0`, indistinguishable from a deleted product, and every future item type widens
the delta fan-out for all product queries. A separate `Type` column would have kept the two axes
independent; this was weighed and the overload chosen deliberately, to avoid adding a column to the
system's busiest table.

Existing `supply_material` rows were **not** migrated. Pre-alpha, and the table had no stock behind
it to preserve.
