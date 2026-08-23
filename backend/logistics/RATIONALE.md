# RATIONALE — logistics

Design decisions for supplies, stock and purchase orders, newest first.

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
