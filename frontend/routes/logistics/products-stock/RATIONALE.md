# RATIONALE — products-stock

Design decisions for the stock page, newest first.

## "One reason per save" is one reason per direction

**Context** — The reason is chosen once in the save summary, but a single save can raise some rows
and lower others, and no reason fits both ways (a Merma cannot add stock).
**Decision** — The summary shows up to two selects, one for the increases (Entrada manual, Saldo
inicial) and one for the decreases (Salida manual, Merma, Consumo interno). Each appears only if the
save has changes in that direction, and each defaults to the manual entry or exit.
**Rationale** — It keeps one choice per save for the common case of a save that only goes one way.
The cost: a count that finds both a merma and a consumption has to be saved in two passes.

## One cost per product row, shared by its lots and serials

**Context** — The Costo column prices the stock an adjustment adds. A row's increase can come from
Stock Simple, from lots in the side panel, or from new serials, and each of those is its own item
on `POST.productos-stock`.
**Decision** — The cost is typed once on the product row and sent as `UnitCost` on every item of
that row. There is no cost column in the lot and serial panels.
**Rationale** — Under a moving average a unit costs the same whatever lot it is in, so a per-lot
cost would be the same number typed several times. The cost: two lots of the same product added
in one save at different prices cannot be told apart — save them separately.
