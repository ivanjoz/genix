## The cash-movement and product-status vocabularies are owned by one module each

**Context** — `CashBankMovement.Type` was four private constants in four modules
(`logistics/purchase-order-management.go`, `finance/expenses.go`,
`accounting/asset_payment.go`, and a bare literal `Type: 8` in `sales/sale_order_create.go`),
each with a comment saying it mirrors the frontend's `cajaMovimientoTipos`. The type is the
discriminator that separates movements sharing a `DocumentID` — an asset payment and an expense
payment can carry the same document id — so a collision is a real accounting bug, yet nothing
stopped the next module from reusing a taken number. `Product.Status == 2` had the same problem
in two copies: `logistics.SupplyProductStatus` and a hand copy `supplyProductStatus` in
`accounting/asset_api.go`, in a file that also imported `logistics`.

**Decision** — `finance/types` owns `type CashMovementType int8` with **all ten** values from the
frontend list, not just the four the backend writes, so a new module cannot claim a used number.
`CashBankMovement.Type`, its `db.Col`, and `InternalCashMovement.Type` are all typed with it.
`business/types` owns `ProductStatusInactive/Active/Supply` as plain `int8` constants next to
`Product`, and both copies are deleted.

**Rationale** — the named type is worth it for the ledger because the column is written from four
modules and the values are load-bearing; typing the column (rather than only the constants) is
what makes a wrong assignment a compile error instead of an arithmetic surprise. It required
fixing the ORM first — see `genix-orm/scylla/RATIONALE.md`; a named integer type was silently
mishandled by six type switches.

`Product.Status` deliberately stays `int8`. Its three values are already enumerated in the delta
view's `FixedValues`, so the vocabulary is worth naming, but the column is read and written across
`business`, `logistics` and the delta view's `[]int64` fixed values — retyping a table that
central is a much larger change than this one, and it buys less, since `Status` is not the
cross-module discriminator that `Type` is.

Cost: `CashMovementType` is a named type inside a `types` folder, which the frontend interface
generator reads. It mapped unknown identifiers to `I<Name>` and then degraded them to `any`, so
`ICashBankMovement.type` would have silently dropped from `number` to `any` on the next
`sync_struct_interfaces` run. The generator now resolves named basic types to their underlying
TypeScript type (`scripts/generators/sync_struct_interfaces.go`, covered by
`named_basic_test.go`) — a general fix, since the plan puts more logic and types into `*/types`.

