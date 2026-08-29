# RATIONALE — sales

Design decisions for sale orders and sale summaries, newest first.

## Sub-unit lines are resolved server-side, from what the operator typed

**Context** — Quantities are a `(Units, Sub)` pair plus a divisor (see `backend/RATIONALE.md`). If
the frontend converts a sub-unit entry to that pair before posting, the server receives a bare
number it has no way to verify. `PostSaleOrder` never loaded a `Product` row at all — it checked only
that the detail slices matched in length and held no zeros, so `DetailPrices` and `TotalAmount` came
straight off the request body unvalidated. A client could post a sale at any price.

**Decision** — the request carries the quantity **as entered**: `DetailQuantities` (whole units),
`DetailSubQuantity` (sub-units) and `DetailSubDivisor` (the divisor in effect at entry).
`PostSaleOrder` loads the products, validates the divisor against the product's current one, rejects
a sub-unit on a product that has none, and validates `DetailPrices` against
`FinalPrice` / `SbuFinalPrice`. `DetailSubDivisor` is persisted on the order.

**Rationale** — `CLAUDE.md` §9 says never trust the client, and the conversion is the natural place
to enforce it: the server has to load the product to resolve the divisor anyway, so validating the
price alongside costs one query and closes a hole that predates this work.

Persisting the divisor rather than re-reading the product's current one means a past order still
describes the line the way it was sold, even after the product is refined to a finer divisor.

**Cost accepted:** a `Product` query is added to a handler that previously touched only
`ProductStock`. That is a real change to the shape of the hot sale path, taken deliberately for the
validation it enables.

## A sub-unit line is charged at the sub-unit's own price

**Context** — A line's amount could be prorated from the unit price (`price × Sub / Divisor`) or
charged against the sub-unit's own `SbuFinalPrice`.

**Decision** — `core.QuantityAmount` charges whole units at `FinalPrice` and sub-units at
`SbuFinalPrice`. A sub-unit line whose `SbuFinalPrice` is unset is rejected rather than prorated.

**Rationale** — prorating is inexact in cents: four candies from a 5000-cent box of six is 3333.33.
Charging the candy's own price is an exact multiple, and `SbuFinalPrice` already exists on `Product`
and is already what the operator sets. Cost valuation on the movement ledger is the one place that
must prorate, because it has only a purchase price — that is `core.QuantityValueAtUnitPrice`, and it
prorates from the combined total so the whole and fractional parts round once together.

## The daily summary chart counts money, not mixed units

**Context** — `SaleOrdersChartsDailySummary` summed `Quantity` across every product in a day for its
delivery-progress ratio, and offered a `quantity` metric mode that did the same for paid/unpaid
totals. Adding kilograms of rice to pieces of screw produces a number with no unit. This predates the
sub-unit work; the pair representation only makes it more obviously wrong, since the sub-units being
summed would be at different divisors.

**Decision** — the delivery ratio is computed from amounts: delivered *value* per product
(`totalAmount × delivered/total`, a unit-free ratio within one product) summed across products. The
`quantity` metric mode is dropped from the daily summary. `SaleOrdersChartsByProduct` keeps it.

**Rationale** — money is the only figure that sums correctly across products, and it answers the same
question the ratio was asked to answer. Keeping the quantity mode only where every figure is already
per-product costs nothing, since that is where it was meaningful in the first place.
