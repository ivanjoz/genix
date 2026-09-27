# RATIONALE — products-stock

Design decisions for the stock page, newest first.

## One cost per product row, shared by its lots and serials

**Context** — The Costo column prices the stock an adjustment adds. A row's increase can come from
Stock Simple, from lots in the side panel, or from new serials, and each of those is its own item
on `POST.productos-stock`.
**Decision** — The cost is typed once on the product row and sent as `UnitCost` on every item of
that row. There is no cost column in the lot and serial panels.
**Rationale** — Under a moving average a unit costs the same whatever lot it is in, so a per-lot
cost would be the same number typed several times. The cost: two lots of the same product added
in one save at different prices cannot be told apart — save them separately.

## The Costo column is hidden when manual entries are blocked

**Context** — With flag 6 on, this page can only lower stock, and a decrease leaves at the current
average cost.
**Decision** — The Costo column is not shown; Costo actual still is.
**Rationale** — An input whose value can never be used only invites the question of why it was
ignored.
