# RATIONALE — sale_order_create

## The cart columns declare no `mobile` config on purpose

**Context** — The cart list is now a `VTable` (CANT. / PRODUCTO / PRECIO) whose remove button is
the new `onRowHover` action. `VTable` switches to its mobile card list when the viewport is under
580px **and at least one column declares `mobile`** — and the card list does not render
`onRowHover`, so in that mode the cart would have no way to remove a line.

**Decision** — None of the three columns declares `mobile`, so the panel keeps the table layout at
every width.

**Rationale** — The alternative was a second, always-visible remove affordance defined as a mobile
card cell, i.e. two definitions of the same action. Three narrow columns inside the detail panel
still read fine on a phone, and touch browsers fire `:hover` on tap. Cost: no card layout for this
list, so if the columns ever grow past three this decision has to be revisited.

## The form checks the buyer against the series, and takes both from the page

**Context** — The backend now refuses a sale whose series demands a buyer it does not have
(`backend/sales/sale_order_issuance.go`). Mirroring that in the form needs two things the
`SaleOrderState` does not hold: the company's series, to know whether the pick is a factura, and the
selected client row, to read its `RegistryNumber`.

**Decision** — `postSaleOrder(allSeries, selectedClient)` takes both as arguments, and
`+page.svelte` supplies them from the services it already instantiates. The rule itself is a pure
copy of the Go one in `sale_order.ts`.

**Rationale** — The alternative was giving `SaleOrderState` its own `EmpresaParametrosService` and
`ClientProviderService`, duplicating two services the page already has for one comparison. Passing
them in keeps `+page.svelte` doing what it is for — wiring — and leaves the state class free of
service lifecycle. Cost: two more arguments at the only call site, and a second place the identity
rule is written down, which is why `sale_order.ts` names the Go file it mirrors.

## Backend does not validate the series — resolved

**Context** — This entry recorded that `sale_order_create.go` only range-checked `IssueSeriesID`
(0–99), so a sale could be stamped with a series the company never configured, or with a factura
series and no client to bill.

**Decision** — Validated on the server as of `backend/sales/sale_order_issuance.go`: the series must
exist and be active, must not be a note series, and the buyer must satisfy the document type. The
reasoning behind the checks is in `backend/sales/RATIONALE.md`.

**Rationale** — Kept as a heading rather than deleted so the gap reads as closed and not as
forgotten.

## Clearing the series selector sends `null`, not `0`
**Context** — The "SIN COMPROBANTE" selector writes through `SearchSelect`'s `save`/`saveOn`, which assigns `selectedItem[keyId]` on pick and `null` on clear. `ISaleOrder.IssueSeriesID` is typed `number`.
**Decision** — Left as-is; no normalization back to `0`.
**Rationale** — Go's `encoding/json` ignores `null` for a non-pointer `int8`, so the field stays at its zero value, which is exactly what "the till named no series" means to `MakeSaleOrderID`. Normalizing would mean an `$effect` or a submit-time coercion for no behavioural difference — but it does mean the TS type is a half-truth, and any future consumer reading `form.IssueSeriesID` in the browser must treat `null` as none.

## The cajas `$effect` no longer drops the payment action
**Context** — That effect used to strip `SALE_ACTION_PAYMENT` from `ActionsIncluded` whenever `cajas.Cajas` was empty. `CajasService` is a `GetHandler`, so it reports ready on the cached response first and fills the list on the server response: the effect fired on the empty intermediate state and silently unticked "Pagado". Invisible until the caja selector's visibility was tied to that flag — then the form showed "Pagado" ticked (the checkbox holds its own copy of the array) while rendering the due-date input.
**Decision** — The effect only assigns `LastPaymentCajaID`. The "no caja ⇒ not paid" rule lives solely in `postSaleOrder`, which already applied it.
**Rationale** — One owner for the rule, and it runs when the data is final instead of racing the cache. Cost: between the two responses the form can carry a payment action with `LastPaymentCajaID = 0`; the submit guard is what makes that safe, so it must not be removed.

## Clearing `PaymentDueDate` when the sale is paid on creation
**Context** — The caja selector and the payment-due-date input are now mutually exclusive in the form: checking "Pagado" swaps the due-date input for the caja selector. An operator can set a due date first and *then* check "Pagado", leaving a due date in the payload that the form no longer shows.
**Decision** — `postSaleOrder` zeroes `form.PaymentDueDate` whenever `ActionsIncluded` still contains `SALE_ACTION_PAYMENT` after the caja check.
**Rationale** — Placed next to the existing "no caja ⇒ drop the payment action" normalization so both contradictions are resolved in one readable spot, instead of adding an `$effect` that fights the form state. Cost: a due date typed before checking "Pagado" is silently discarded rather than flagged as an input error.
