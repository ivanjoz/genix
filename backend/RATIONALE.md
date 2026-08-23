## The stock movement engine lives in `logistics/types`, and the `sales` copy of its key packer is gone

**Context** — `ApplyMovimientos` is the only writer of `ProductStockV2` and the movement ledger,
and `sales` (delivery) and `accounting` (asset acquisition, inventory expense) both have to call
it. Both reached it by importing `app/logistics` (V2, V4). The cost of that missing seam was
already visible: `sales/sale_order_create.go` carried `packProductStockIDForSale`, a hand copy of
`logistics.packProductStockID` — the ORM's `KeyIntPacking` formula for `ProductStockV2`,
duplicated in two places where it must agree with the schema. Its comment justified the copy with
a circular dependency that did not exist: `logistics` imports only `business/types`, never
`business`.

**Decision** — `logistics/product-stock-movement.go:262`–end plus all of
`logistics/stock-reprocess.go` moved into `logistics/types/stock_movement_apply.go`. The
duplicate in `sales` is deleted and calls `logistics.PackProductStockID`.

**Rationale** — the recalc moved with the engine (not left behind) because the two share one
per-company write lock: splitting them would have forced `getApplyMovimientosCompanyLock` to be
exported for no reason other than the file boundary. `RecalcProductStockByMovements` is called
from `exec`, a composition root, so exporting it costs nothing.

One correction to the plan's Q3 reasoning: moving the recalc keeps the **lock** private, but
`packProductStockID` had to be exported as `PackProductStockID` regardless, because `sales` needs
the packed key for stock validation — which is precisely why the duplicate existed. Trading one
export for one deleted copy of a schema-coupled formula is the right side of that trade; the plan
was wrong to imply both could stay private.

**Cost** — `logistics/types` is now the largest package in the module by a wide margin, most of it
the engine rather than type declarations. The filename and a file-level comment carry the
explanation; the folder name does not.

