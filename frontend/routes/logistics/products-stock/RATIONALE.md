# RATIONALE — products-stock

Design decisions for the stock page, newest first.

## A missing cost stops Guardar before the summary opens

**Context** — An increase on a row with no Consolidated cost (AverageCost 0) needs a New cost. The
summary modal is where the reason is picked, and that reason only matters for opening stock.
**Decision** — Guardar checks for an uncosted increase on such a row and refuses right away,
naming the product. The full check (company flag 6 and opening stock) still runs on the modal's
confirm. The red "Requerido" warning in the New cell does not know the reason yet, so it covers
only the missing-average case.
**Rationale** — No reason picked in the summary can fix a missing cost, so opening the summary
would only lead to a refusal on confirm. The cost: there are two checks instead of one, but both
use the same `increaseNeedsUnitCost`.

## Flag 9 is retired by a comment, not reused

**Context** — Flag 9 ("block stock movements without a purchase cost") was removed. Its whole
`[logistics]` section went with it.
**Decision** — `company_flags.toml` keeps a comment that reserves id 9. No data migration was
written: a full scan of `companies` (2 rows) found no company with flag 9 set.
**Rationale** — The catalog refuses to reuse an id. The cost: a company that still had 9 stored
would fail to save its flags, because the catalog rejects unknown ids. Today none does.

## "One reason per save" is one reason per direction

**Context** — The reason is chosen once in the save summary, but a single save can raise some rows
and lower others, and no reason fits both ways (a Merma cannot add stock).
**Decision** — The summary shows up to two selects, one for the increases (Entrada manual, Saldo
inicial) and one for the decreases (Salida manual, Merma, Consumo interno). Each appears only if the
save has changes in that direction, and each defaults to the manual entry or exit.
**Rationale** — It keeps one choice per save for the common case of a save that only goes one way.
The cost: a count that finds both a merma and a consumption has to be saved in two passes.

## One cost per product row, shared by its lots and serials

**Context** — The New (Nuevo) cost column prices the stock an adjustment adds. A row's increase can come from
Stock Simple, from lots in the side panel, or from new serials, and each of those is its own item
on `POST.productos-stock`.
**Decision** — The cost is typed once on the product row and sent as `UnitCost` on every item of
that row. There is no cost column in the lot and serial panels.
**Rationale** — Under a moving average a unit costs the same whatever lot it is in, so a per-lot
cost would be the same number typed several times. The cost: two lots of the same product added
in one save at different prices cannot be told apart — save them separately.
