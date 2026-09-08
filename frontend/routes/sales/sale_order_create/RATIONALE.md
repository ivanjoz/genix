# RATIONALE — sale_order_create

## Clearing the series selector sends `null`, not `0`
**Context** — The "SIN COMPROBANTE" selector writes through `SearchSelect`'s `save`/`saveOn`, which assigns `selectedItem[keyId]` on pick and `null` on clear. `ISaleOrder.IssueSeriesID` is typed `number`.
**Decision** — Left as-is; no normalization back to `0`.
**Rationale** — Go's `encoding/json` ignores `null` for a non-pointer `int8`, so the field stays at its zero value, which is exactly what "the till named no series" means to `MakeSaleOrderID`. Normalizing would mean an `$effect` or a submit-time coercion for no behavioural difference — but it does mean the TS type is a half-truth, and any future consumer reading `form.IssueSeriesID` in the browser must treat `null` as none.

## Backend does not validate the series — TODO
**Context** — `sale_order_create.go` only range-checks `IssueSeriesID` (0–99) via `MakeSaleOrderID`. It never checks that the id names a series that exists on the company and is active, so a crafted request can stamp a sale with a series that was never configured.
**Decision** — Left unvalidated for now, at the human's instruction: they will specify how the check should work.
**Rationale** — Recorded here so the gap is not mistaken for a finished contract. Validating means `sales` reading the company's series, which crosses a module line and costs a read per sale — the shape of that is the open question.

## The cajas `$effect` no longer drops the payment action
**Context** — That effect used to strip `SALE_ACTION_PAYMENT` from `ActionsIncluded` whenever `cajas.Cajas` was empty. `CajasService` is a `GetHandler`, so it reports ready on the cached response first and fills the list on the server response: the effect fired on the empty intermediate state and silently unticked "Pagado". Invisible until the caja selector's visibility was tied to that flag — then the form showed "Pagado" ticked (the checkbox holds its own copy of the array) while rendering the due-date input.
**Decision** — The effect only assigns `LastPaymentCajaID`. The "no caja ⇒ not paid" rule lives solely in `postSaleOrder`, which already applied it.
**Rationale** — One owner for the rule, and it runs when the data is final instead of racing the cache. Cost: between the two responses the form can carry a payment action with `LastPaymentCajaID = 0`; the submit guard is what makes that safe, so it must not be removed.

## Clearing `PaymentDueDate` when the sale is paid on creation
**Context** — The caja selector and the payment-due-date input are now mutually exclusive in the form: checking "Pagado" swaps the due-date input for the caja selector. An operator can set a due date first and *then* check "Pagado", leaving a due date in the payload that the form no longer shows.
**Decision** — `postSaleOrder` zeroes `form.PaymentDueDate` whenever `ActionsIncluded` still contains `SALE_ACTION_PAYMENT` after the caja check.
**Rationale** — Placed next to the existing "no caja ⇒ drop the payment action" normalization so both contradictions are resolved in one readable spot, instead of adding an `$effect` that fights the form state. Cost: a due date typed before checking "Pagado" is silently discarded rather than flagged as an input error.
