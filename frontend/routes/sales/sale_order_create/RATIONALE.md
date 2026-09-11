# RATIONALE — sale_order_create

## A client registered at the till is fetched by id, not stored on the row

**Context** — The history stores `clientID` and resolves the name through the page's
`ClientProviderService`. That service is a delta-cached list loaded on mount, so a client the
operator typed into the form is created by `POST sale-order` *after* the list was fetched: the id
comes back on the sale, the record is nowhere in the browser, and the ticket printed "CLIENTE:
VARIOS" for a buyer it had been given explicitly — on a factura, which cannot legally be anonymous.

**Decision** — After a successful post, the page calls `clientesService.syncIDs([sale.ClientID])`
before registering the history row. `clientsByID` in `TicketContext` is built from
`clientesService.records`, not from `recordsMap`.

**Rationale** — `syncIDs` is the project's by-id cache path (`routeByID = client-provider-ids`): it
fetches only what is missing and is a no-op when the client was picked from the list, so the common
case costs nothing and the rule that rows hold references survives intact. The `records` detail is
load bearing and not a style choice — `syncIDs` merges by mutating `recordsMap` in place, and a
plain `Map` is not reactive in Svelte 5, so a `$derived` reading the map would never recompute.
Cost: one extra request on a sale that registers a new client, and the post now awaits it before
the view switches.

## The detail layer is 4px shorter than the viewport

**Context** — `LayerStatic` here was `h-[calc(100vh-var(--header-height))]`, while the `Page` shell
it sits in is deliberately 4px shorter: `min-height: calc(100vh - var(--header-height) - 4px)` in
`domain-components/Page.svelte`. The layer therefore overflowed its own shell by exactly 4px. That
was invisible until this feature added the first modal to the route — `Modal` locks `html`'s
scrolling, `body` is a scroll container because `app.css` sets `overflow-x: hidden` on it, and the
4px surfaced as a full-height scrollbar beside the open dialog.

**Decision** — The layer now uses `calc(100vh - var(--header-height) - 4px)`, the shell's own
expression.

**Rationale** — Fixed in this route rather than in `Page`, whose `-4px` is intentional and shared by
every other route; changing it there to make one layer fit would move layout everywhere. Copying
the expression keeps the two in sync by construction instead of by a bare number. Cost: the
constant is now written in two places, so a change to the shell's padding has to be repeated here.

## What a history row stores, given that it must store references

**Context** — The rule for the local history is that nothing derivable is persisted: the row keeps
ids and the sale id, and `TicketContext` resolves them against the services the page already holds.
But three per-line values are not references and are not recoverable either — the sub-unit divisor
and the two prices the line was sold at.

**Decision** — `SaleHistoryLine` stores `quantity`, `subDivisor`, `unitPrice` and `subUnitPrice`,
and nothing else beyond `productID` / `presentationID` / `serialNumbers`. The line's amount is not
stored: it is `quantityAmount` of the three.

**Rationale** — The catalog is free to reprice or reconfigure a product, and a ticket already handed
to a customer cannot change when it does. `subDivisor` is the sharpest case: `quantity.sub` is
meaningless without it, so a product whose `SbuQuantity` changes would make every older row
misread. Cost: a line whose product was deleted prints `(producto no encontrado)` — the price
survives, the name does not, which is the trade the rule asks for.

## The correlativo is padded to 8 digits here and not in the invoicing panel

**Context** — `invoiceNumber` in `routes/accounting/invoicing/invoicing.ts` renders
`B001-130`. This ticket renders `B001-00000130` from the same two numbers.

**Decision** — `saleDocumentNumber` pads to 8 digits. The invoicing panel was left alone.

**Rationale** — They are not the same artifact. The panel is a list column an operator scans; the
ticket is the printed representation of the comprobante, and SUNAT numbers those with a zero-padded
8-digit correlativo. Cost: the same sale reads two ways across two screens, so anyone comparing
them has to know which one is the document.

## The printed copy is a second `<pre>`, outside the dialog

**Context** — Printing is done with `@media print` over the open modal. `Modal`'s dialog box carries
`transform: translateY(...)` for its open transition, and a transform makes an element the
containing block for any `position: fixed` descendant — so a print element placed inside the dialog
anchors to the dialog, not to the page.

**Decision** — `SaleTicketModal` renders the ticket text twice: the on-screen preview inside the
`Modal`, and a `_printCopy` in its own `Portal` at body level, `display: none` except in print.

**Rationale** — The alternative was reaching into `Modal` to drop the transform while printing,
which would change a shared component for one caller. Both elements render the same
`renderSaleTicket` string, so there is one layout and only the element is duplicated. Cost: two
nodes holding the same text, and a reader has to notice why.

## The preview is 14px on screen and 2.5mm only on paper

**Context** — The ticket is sized for the physical roll: a 1.5 mm character cell, which at Courier's
0.6em advance means a 2.5 mm type size. On a 96 dpi screen that is about 9 px — below the project's
14 px floor and genuinely hard to read.

**Decision** — The preview renders at `text-[14px]` with its box set to `{columns}ch`; only the
print rule uses millimetres.

**Rationale** — Line breaks are decided in the string by character count, not by the font, so the
preview wraps identically at any type size — the fidelity that matters is preserved while the
fidelity that does not (absolute size) is dropped. `box-sizing: content-box` on that box is load
bearing: with the inherited `border-box` the horizontal padding eats into the `ch` width and the
widest rows silently lose their last characters.

## Three `sale_history` files instead of one

**Context** — The approved plan had a single `sale_history.idb.ts`. In writing it the module ended
up doing three jobs: the Dexie schema, the rules that shape a row from a posted sale, and the
reactive list the view renders.

**Decision** — Split along the folder convention: `sale_history.idb.ts` (persistence and the row
shape), `sale_history.ts` (pure — `buildSaleHistoryRow`, the status predicates),
`sale_history.svelte.ts` (`SaleHistoryState`).

**Rationale** — `sale_history.ts` is the half with rules in it and it is now unit-testable without
Dexie or a browser. Cost: three files for a feature whose whole surface is a list of sales.

## `sale_ticket.ts` formats money itself

**Context** — `$libs/helpers` exports `formatN`, but it also imports notiflix and calls
`Loading.init()` at module scope.

**Decision** — `formatTicketAmount` is eight lines inside `sale_ticket.ts`.

**Rationale** — Importing `formatN` would make the renderer — and its test — need a DOM for a
thousands separator. Cost: a second money formatter in the codebase, which is why it says so where
it is defined.

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
