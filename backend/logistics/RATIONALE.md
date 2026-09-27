# RATIONALE — logistics

Design decisions for supplies, stock and purchase orders, newest first.

## A transfer's inflow is priced at the origin's average, not at a typed price

**Context** — A stock transfer is now two ledger rows (`expandStockTransfers`), and the receiving
warehouse has its own moving average that the inflow feeds.
**Decision** — Both legs drop any `Price` the caller sent; inside the lock the inflow takes the origin
row's `AverageCost` at that moment. A transfer cannot set stock ("set to X") and must move a positive
quantity between two different warehouses.
**Rationale** — Moving goods between warehouses creates no value, so the destination must receive
them at what they cost the origin. The cost: an origin with no average (0) delivers uncosted stock,
which leaves the destination's average unchanged.

## An uncosted inflow leaves the average cost where it was

**Context** — `ProductStock.AverageCost` moves on every costed inflow. Receptions of an order in
dollars with no rate, sale-annulment returns and manual increases with no cost (flag 9 off)
all bring units in with no price.
**Decision** — They add units to the balance but do not move the average: the next costed inflow
averages against the units held, uncosted ones included, at the current average.
**Rationale** — Averaging them in at zero would drag the cost down for every later sale; leaving
them out keeps the figure a real purchase cost. The cost: those units are silently valued at the
running average, which is the gap the plan's `INV-001` diagnostic exists to show.

## AverageCost is written at movement time and is not restated later

**Context** — The cost of a reception comes from its order's comprobante, which often arrives
after the goods. An order received before its comprobante is registered has only its gross price.
**Decision** — The reception is costed with what the order holds at that moment (gross, if no
comprobante yet). Registering the comprobante later does not touch `AverageCost` or the ledger's
`MonetaryValue`. `RecalcProductStockByMovements` replays the average from `MonetaryValue`.
**Rationale** — Restating would mean rewriting the append-only ledger or re-deriving every later
outflow on each document edit. The accounting valuation (`docs/ACCOUNTING_MODEL_PLAN.md` §4.5)
resolves costs from the documents at read time, so this column is an operational figure.

## IGV is always treated as recoverable when a comprobante carries it

**Context** — `InventoryUnitCost` takes the IGV out of a factura's cost. Whether the company can
actually use the crédito fiscal depends on its tax regime (an NRUS company cannot), and there is
no regime setting yet.
**Decision** — Any document with `TaxAmount > 0` is costed net of it.
**Rationale** — Almost every company buying on factura can use the credit. The cost: an NRUS
company's average reads 18 % low on factura purchases until the plan's `TaxRegime` setting exists.

## A replace-to-the-same-quantity writes no ledger row

**Context** — `ApplyMovimientos` now accepts a zero target (writing a row off), so a "set stock to
X" item can also arrive with X equal to the current balance.
**Decision** — A zero delta is skipped after the mutation step: no movement row, no type, no cost.
**Rationale** — A ledger row of quantity 0 is noise in the Kardex. The stock row is still written
as it was, which is harmless.

## Manual-stock policy is judged inside the stock engine

**Context** — Flag 6 blocks manual increases and flag 9 requires their cost. `POST.productos-stock`
sends absolute targets, so whether an item is an increase is only known once the current balance
is read.
**Decision** — The handler reads the flags and stamps `RejectInbound` / `RequireInboundCost` on each
`InternalMovement`; `ApplyMovimientos` checks them on the delta, under the company lock.
**Rationale** — Checking in the handler would mean a second balance read outside the lock, which a
concurrent sale could invalidate. The cost: two policy booleans on the engine's input struct.

## GET.company-parametros is granted with Gestión de Stock

**Context** — The stock page reads the company flags from the company record, and access 14 only
granted `POST.productos-stock`.
**Decision** — Access 14 also grants `GET.company-parametros`, as Punto de Venta already does.
**Rationale** — It is the one route that carries the flags. The cost: a stock user can read the
company's parameters (not its secrets, which are a separate route).

## A failed express entry annuls the order it just created

**Context** — `POST.purchase-orders` with `ExpressEntry` inserts the order, then receives it. The
stock movements carry the order ID as `DocumentID`, so the order must exist first, and the ORM has
no delete to roll the insert back.
**Decision** — If `receivePurchaseOrder` fails, the new order is set to Canceled and the error names
its number.
**Rationale** — Reserving the ID up front would copy the ORM's counter internals into the handler.
The cost: a failed express entry leaves an annulled order in the report instead of nothing.

## POST.purchase-orders recomputes TotalAmount from the lines

**Context** — The create handler stored the client's `TotalAmount` as is. The comprobante sent with
a new order is now checked against it, and `DebtAmount` starts from it.
**Decision** — The handler sums `DetailProductPrice × DetailProductQuantity` and overwrites
`TotalAmount` before anything reads it.
**Rationale** — A client total would let the debt and the comprobante agree with each other and
disagree with the lines. The frontend computes the same sum, so nothing changes for it.

## The supplier's comprobante is registered with its own order action

**Context** — `InvoiceNumber` was one free-text field, edited only while the order was Pending or
Confirmed. The invoice usually arrives with the goods or after them, and a Fulfilled order was
immutable, so it could never receive one.
**Decision** — `PUT.purchase-orders?action=5` writes only the document columns, on any status but
Canceled. `POST.purchase-orders` may also carry one for a new order (same rules,
`PurchaseOrder.ApplyPurchaseDocument`); an update through it never touches the document. `InvoiceNumber` is deleted: it was empty on every stored row. `TaxAmount` is now the
invoice's IGV (RCE field 16), so the create page computes its subtotal locally instead of sending
an estimate.
**Rationale** — The document is accounting data with its own lifecycle. Reopening the whole edit
action on Fulfilled orders would also have reopened the warehouse and the dates.

## The bulk supply save validates what the single-record one never did

**Context** — `POST.product-supply` became an array route to back the Excel import. The old
single-record handler trusted `ProductID` completely: it wrote the supply row without ever checking
the product existed. That was survivable while the only caller was a side panel that had just
rendered the product, and stops being survivable the moment a spreadsheet full of typed product
names reaches the same handler.

**Decision** — the bulk path adds three checks the old one lacked: every `ProductID` must resolve to
a product in the company (one query for the whole payload), a `ProductID` may not repeat inside one
payload, and a payload above `maxProductSupplyBulkRecords` (1000) is refused. Provider validation
moved to `ValidateProviderSupplyRowsBatch`, which runs the structural checks per record but resolves
the union of provider IDs in a single query; `ValidateProviderSupplyRows` is now a one-element
wrapper over it, so `production`'s insumo save keeps its old behaviour.

**Rationale** — the per-record validator would have meant one provider query per row: 2000 rows
reading the same hundred providers 2000 times. Rejecting a duplicated `ProductID` rather than
letting the last one win keeps the outcome independent of slice order — with two rows for one
product inside a single `db.Merge`, which survives is an implementation detail, not a decision the
caller made. The 1000 cap costs the client a loop it already has (it batches at 500) and bounds what
one request can do.

## Supplies and the suppliers page left; the suppliers *route* arrived

**Context** — the Producción and Clientes (CRM) modules were carved out of `business`, and two
things that lived here had to be re-homed.

**Decision** — `supply-material-management.go` moved to `app/production` (a supply is a `Product`
row at `Status = 2`, so the catalog owns it), while `product-supply-management.go` stayed: minimum
stock and provider rows are a purchasing policy. The shared provider-row helpers became
`types.SanitizeProviderSupplyRows` / `types.ValidateProviderSupplyRows` in
`logistics/types/product_supply_providers.go`, since both modules now call them. In the frontend,
`/logistics/supplies-materials` became `/production/supplies-materials`, and `/business/suppliers`
became `/logistics/suppliers` — the supplier list sits with the purchase orders that consume it,
even though `crm/types` owns the row.

**Rationale** — the boundary follows who writes the row, not which menu shows it. Splitting on
that line keeps `PostSupplyMaterial` (which writes both `Product` and `ProductSupply`) importing
only `logistics/types`, and leaves no module body importing another module body.

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
