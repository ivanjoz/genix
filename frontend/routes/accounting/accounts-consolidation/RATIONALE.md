## Diario de gestión in the Movimientos layout: shared cells, shared search text
**Context** — The Diario moved to one row per operation with Debe/Haber sub-columns, and its
search box moved up to the period row.
**Decision** — The Debe/Haber cell renderers live in `accounts-consolidation.cells.ts`, used by
both tables, and `entrySides` replaces `buildJournalRows`. One search text, in the page's top row,
filters whichever tab is open, so it carries over when the user switches between Movimientos and
the Diario. The Diario's date is `OCT-13`, like Movimientos.
**Rationale** — The two tables draw an entry the same way from one place. Cost: a search typed in
one tab still applies in the other.

## Movimientos groups: account matching, fixed account width, ticking only single rows
**Context** — Movimientos now shows one row per `groupOperations` group, with Debe/Haber cells
that stack one fixed-height line per account. Some details were left open.
**Decision** — Two lines with the same account on the same side count as one in the group key and
are summed, even if one is a nature line and the other is derived. An account line is `w-240
truncate`, because VTable ignores column `width` (the table uses auto layout) and a fixed width is
the only way to get the ellipsis. A group of several operations shows `(N)` in Nº Doc even when
none has a comprobante, so it no longer shows the "sin comprobante" chip; the "Sin comprobante"
filter still works. The side layer lists a group's operations (CUO, Nº Doc, Monto), with a "Volver a la
lista" link back from an opened entry.
**Rationale** — Keeps the table one row per day and entry shape without touching genix-ui. Cost:
the account column does not grow with the screen, and bulk classification of a large group means
opening it.

## PurchaseBookDocument carries the comprobante fields flat, not an embedded PurchaseDocument
**Context** — colbin does not flatten an embedded struct: the fields went nested under
`PurchaseDocument`, so `DocIssueDate`, `DocType`… were undefined in the browser and every purchase
order dropped out of the page (the 5032 vs 5060 seen before this work). JSON flattens, which hid it.
**Decision** — The ten fields are declared on `PurchaseBookDocument` (cb 18–27) and filled by
`withPurchaseDocument`. The request bodies that embed it are JSON-decoded and stay as they are.
**Rationale** — Fixes this page and the Registro de Compras with no client-side unwrapping.

## Depreciation travels in the Expenses key; inferred payments get the CUO `CJD-<type>-<document>`
**Context** — `loadPeriodDepreciation` was deleted, and an inferred payment has no movement ID.
**Decision** — `PurchaseBookDocument` gained `AssetID` and `Date`, so a type-4 expense is a
depreciation entry. An inferred payment's SourceID is its document's ID; cash classification
matches a movement by CUO (`CJ-<ID>`), so an inferred one can never pick up a real movement's
targets.
**Rationale** — One key per table (one delta log each), and no fake IDs.

## Annulled sales, canceled orders and removed expenses are not listed on a first sync
**Context** — Status 0 is logged as a delete, so a delta sync evicts them anyway.
**Decision** — First syncs list Status 1–4 (sales), 1/2/4 (orders), 1–3 (expenses); every
comprobante State (a voided one is skipped by the builder, its State is not its Status).
**Rationale** — The builder never posted them; listing them would only make first and later
syncs disagree.

## Receptions keep the "purchase not found" problem; payments lost theirs
**Context** — Payments now carry their counter-account, receptions their purchase account.
**Decision** — A reception still names its purchase (label and official flag) through the by-IDs
cache and reports when it is missing; a payment needs no lookup. A reception whose account has
no stock pair is dropped whether or not its purchase is found.
**Rationale** — Only the reception still reads its document.

## Sell rates are computed in the browser with `latestSellRate`, BCRP included
**Context** — `SellRatesByDay` falls back to the BCRP series, which the plan's port did not mention.
**Decision** — The page loads the exchange-rates service and `BcrpDefaultRates` for the period's
month and the one before (7-day look-back), and builds `SellRates` per day with the existing
`latestSellRate` — the backend's rule.
**Rationale** — Reuses the till's rule; costs one BCRP year file per browser, cached.

## Consolidado & Cuentas grants GET.assets and the two new by-IDs routes
**Context** — A route listed in `access.toml` needs one of its accesses, and `GET.assets` was only
under Activos.
**Decision** — Access 38 lists `GET.assets`, `GET.purchase-order-ids`, `GET.expense-ids`.
**Rationale** — The page reads assets itself now; without it a user without Activos gets a 403.

## Plan de cuentas shows the period's trial balance, only accounts that moved
**Context** — The tab could list the whole 556-account chart or only what the period touched, and
the period's movement alone is not an opening-to-closing balance.
**Decision** — `buildTrialBalance` lists only accounts with movement in the period, rolls each
line's Debe/Haber up to every parent account (2 digits and up), and orders by code as text, with the
unclassified lines (account 0) last. Saldo = Debe − Haber of the period only.
**Rationale** — A full chart of zeros hides what matters; the tree with parent totals reads like a
balance de comprobación. Opening balances need the earlier periods and belong to phase 7.

## The Diario de gestión shows every entry
**Context** — The management journal and the official Diario share one set of entries.
**Decision** — The tab lists all operations, with or without comprobante (a chip marks the
undocumented ones). Unclassified accounts show as a red warning, not hidden.
**Rationale** — Phase 6 filters these same rows by `InOfficialBook`, so the two can't diverge.

## What the page lets the user classify, and how a bulk apply is bounded
**Context** — Only nature lines are classifiable, but several operations take their accounts from
another record, and a bulk apply needs one account list for everything selected.
**Decision** — `classificationTargetsOf` maps each operation to the record fields behind it:
- an expense → its `AccountCode` (supply list for an inventory expense)
- an order → its supply lines, by index
- an asset, its depreciation or its disposal → the asset's 33x
- a classifiable cash movement → its counter-account

Bulk apply works only when every target shares one PCGE kind. Cash movements are left out of it,
because each type has its own options. A bulk apply on an order sets all its supply lines.
**Rationale** — Classifying the asset fixes its acquisition, every depreciation and its disposal at
once. One kind per bulk keeps the picker to the accounts the server will accept.

## PUT.accounting-classification validates everything before writing anything
**Decision** — The handler loads every named record, applies and validates each classification
in memory, and writes each table once only when all of them hold. An unspecified cash movement
may be retyped only into withdrawal, loss or collection, with the sign `ValidateManualCashMovement`
requires. Never into a transfer or a physical count, which have their own writers.
**Rationale** — A bulk apply that fails halfway would leave a period half classified with no clue
which half. Cash movements have no `Updated` column, so they are written without one; the cash
page reads them by range, not by delta, so it still sees the change.

## Row selection: VTable has it, Movimientos draws its own checkbox column
**Decision** — `VTable` (genix-ui) takes `selectedIDs` (a `SvelteSet`) and `getRowID`, and
prepends a checkbox column built on the existing `onCellClick` cell; it is kept as a generic
feature. Movimientos does not use it: only a row of one operation can be ticked, which VTable's
column cannot express per row, so the page declares its own click column. No page uses VTable's
selection today.
**Rationale** — An ordinary click cell renders on desktop, in the plain table and on mobile cards,
and the agent can already click it.

## The cost of a sale follows the sale in or out of the official Diario
**Context** — `SALE_COST_PLAN.md` §4 called the cost entry internal and always official.
**Decision** — `CV-<sale>-<day>` is `InOfficialBook` only when its sale has a comprobante.
**Rationale** — The same reason the collections follow the sale: the official books would
otherwise deduct a cost (6912) against revenue they never recorded, and 2011 would drop for goods
whose sale is not in them. The cost: the official 2011 balance is higher than the physical stock by
what undocumented sales took out.

## The endpoint sends the product facts; the builder needs no products cache
**Context** — The revenue split and the cost entry need each product's `Origin`, and a supply's
`AccountCode`. Supplies are not in the browser's `ProductsService`.
**Decision** — `GET.accounting-operations` returns `Products` (ID, status, Origin, AccountCode)
for every product its sales, order lines and stock movements name. `IOperationsContext` keeps
only the series.
**Rationale** — One read on the server instead of two caches joined in the page, and supplies
come along for free.

## The operations endpoint reuses the purchases-book loaders and sends the older documents too
**Context** — `GET.accounting-operations` needs the same orders and expenses the Registro de
Compras reads, plus their lines and accounts. A payment or reception often points to a document
from an earlier period.
**Decision** — `PurchaseBookDocument` gained `AccountCode`, `ExpenseType`, `ProductID` and
`Lines`, and both endpoints share the loaders. The response also carries the sales, orders and
expenses that the period's comprobantes, notes, payments and receptions name, even when they
are older. The builder posts only records dated in the period; the older ones are lookups.
**Rationale** — One set of queries, no second struct to keep in sync. Cost: the purchases book
payload carries a few fields it does not read.

## A purchase without a comprobante is owed on 4699 and carries no IGV
**Context** — The plan said "Cr 4212 / 4699" for expenses without saying which.
**Decision** — A comprobante → Cr 4212 Facturas emitidas, and Dr 40111 only when its type grants
crédito fiscal (factura, ticket, recibo de servicios). No comprobante → the whole total is
cost and is owed on 4699. Its payment uses the same rule. Assets always use 4654.
**Rationale** — 4212 means "an invoice was issued", which is false without one, and no IGV can
be credited without a comprobante. Keeping one rule for the purchase and its payment makes the
payable account return to zero.

## Dollar amounts use the document's rate, else the official sell rate of the day
**Decision** — A purchase comprobante uses its own `ExchangeRate` (RCE field 27). Everything
else uses the official sell rate of its day: sales, comprobantes, cash in a USD bank, and
undocumented purchases. That rate is the company's own, else BCRP's, and a day without one takes
the last rate within 7 days (`finance.SellRatesByDay`). With no rate, the operation is incomplete.
**Rationale** — It follows `ACCOUNTING_MODEL_PLAN` D11: the books use the official rate, and the
till's commercial rate (with spread) is a pricing rule. Each amount is converted on its own, and the
receivable or payable is the sum of the converted lines, so the entry always balances.

## Revenue splits between 7012 and 7022 by line value
**Decision** — The comprobante's base (taxable + exempt + unaffected) is split by the value of
the sale's lines with `Origin = 2` against the rest. A supply sold, or a product with no Origin,
counts as merchandise. A sale without a comprobante posts its `TaxAmount` to 40111 as well.
**Rationale** — The comprobante holds the totals and the sale holds the lines, so this is the
only split both agree on. The undocumented sale keeps its IGV so that its revenue matches the
sale's own net amount; the official Diario does not hold it anyway.

## Asset disposals post to 6551, not 65514
**Context** — The plan named 65514, which is not in the official PCGE 2019 chart.
**Decision** — Dr 3952x the accumulated depreciation · Dr 6551 Costo neto de enajenación de
activos inmovilizados (the book value left) · Cr 33x the acquisition value.
**Rationale** — 6551 is the postable leaf for the net cost of a disposed fixed asset.

## Smaller entry rules the plan left open
- A purchase-order line with `AccountCode = 0` is a product line and posts to 6011, locked. The
  server stamps every supply line, so a 0 never means "unclassified" on an order.
- A reception posts only when the purchase account has a stock pair (60x). A 656 or 33x purchase
  never enters stock accounts, and an unclassified inventory expense shows no reception until it
  is classified.
- A transfer is booked once, from its outgoing row (Dr destination · Cr origin).
- A donated asset's Dr 33x · Cr 501 is an internal entry of the official Diario: it is not a
  purchase, so the "no comprobante" exclusion does not apply.
- An asset's acquisition is dated at its comprobante's issue date, else its acquisition date.

## The payments of an undocumented sale or purchase also stay out of the official Diario
**Context** — The official Diario excludes sales and purchases without comprobante. Their
collections and payments are separate cash movements, and the rule did not say whether they
follow the document out.
**Decision** — They follow it: a collection or payment whose sale, purchase order, expense or
asset has no comprobante is not `InOfficialBook`.
**Rationale** — Otherwise the official book would hold Dr 101 / Cr 1212 for a sale it never
recorded, leaving 1212 Clientes (or 4212 Proveedores) with a balance no document explains.
Excluding the pair keeps the receivable and payable accounts consistent; the cost is that the
official cash balance is lower than the real one by the undocumented sales.
