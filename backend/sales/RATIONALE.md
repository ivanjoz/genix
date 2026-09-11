## The sale id carries the correlativo, so there is one counter and not two

**Context** — The human's instruction for the id layout was `[correlativo][rand:2][series:2]`. What
was built was `[counter][rand:2][series:2]` with a *separate* per-series counter for the correlativo:
two autoincrements for one number, and a sale id that said nothing about the comprobante it would
produce.

**Decision** — `MakeSaleOrderID` draws from `IssueSeriesCounterName(companyID, seriesID)` — the same
`cpe_{company}_{series}` sequence invoicing was using — and `SaleOrder.Correlativo()` reads it back
off the id. `CorrelativoCounterName` is gone from `invoicing/types`, and the global
`x{company}_sale_order_0` counter with it. The sale 1301102 *is* F001-130.

Four things around it were mine to settle:

- **The counter name lives in `sales/types`.** `invoicing/types` already imports it — since the
  document builder moved there — so the other direction would be a cycle. A note that needs its own
  series counter reaches it from there.
- **Series 0 gets a counter of its own.** A sale that issues no comprobante still needs an id;
  drawing from `cpe_{company}_0` keeps one rule for every sale and never consumes a number anybody
  declares to SUNAT.
- **A note cannot follow the rule.** It is issued under its own series but its id has to stay
  adjacent to the sale it corrects, so its correlativo keeps coming from the `Correlativo` column.
  The derivation is for facturas and boletas.
- **The document is validated before the id is minted.** `PrepareDocumentForSale` now assigns
  `order.ID` itself, after `ValidateDocument` passes. With the counters merged, minting first would
  mean a document facturago rejects leaves a permanent gap in the series — the exact thing the old
  ordering existed to prevent.

**Rationale** — One number, one sequence, and no lookup to go from a sale to its comprobante. The
cost is that every sale under a series consumes a correlativo at creation, which was already true
since the document is created with the sale. The migration cost was real and paid once: the ids
already written came from the global counter, and per-series counters restarting at 1 would have
re-minted into that range, so the company's test rows were deleted rather than left to collide.
The `cpe_1_2` counter was deliberately **not** reset — SUNAT beta already holds F001-1, so the next
factura has to be F001-2 or it is refused as a duplicate.

## A sale writes its own electronic document, and the one write that can leave the two out of step

**Context** — The document is created with the sale now. The sale row and the document row are two
ORM writes and there is no transaction between them, so something had to give: either a sale can
exist without its document, or a document can exist without its sale.

**Decision** — Ordered so only the first is possible, and only on a database failure. Building the
document, validating it against facturago and reserving the correlativo all happen **before**
`db.Insert(sale)`; the document is written **after** it. A failure before the sale rejects the whole
request with nothing persisted. A failure on the document write returns an error and logs
`VENTA SIN COMPROBANTE::` with the sale id.

**Rationale** — A document must be written after its sale because every send rebuilds it from the
sale: a document whose sale cannot be read is one nobody can send, and the sweep would keep failing
on it. Reversing the order would trade a rare recoverable state for a permanent orphan. The residual
window is one ORM insert wide and is the only thing between this and a transaction the database does
not offer. Cost: a sale can, in that window, exist with no comprobante and nothing sweeps for it —
finding those means scanning sales, which the partition layout cannot serve cheaply, so the log line
is the trail.

## Annulling a sale voids its pending document, under the invoicing lock

**Context** — The annulment refused any sale that had a document. Now every sale issued under a
series has one from birth, so that rule would have made annulment impossible.

**Decision** — `VoidPendingDocumentForSale` in `invoicing/types`: a document still in
`InvoicePending` is set to `InvoiceVoided` with `Status = 0` and the annulment proceeds; one that
reached SUNAT is returned and the annulment is refused as before. It runs under
`core.ActionInvoiceSaleOrder` on the sale id — the same lock the sweep takes before sending.

**Rationale** — The lock is the whole correctness argument: without it the sweep could read a
pending document, and this could void it, and the document would go to SUNAT anyway. The annulment
takes it nested inside its own `ActionAnnulSaleOrder` lock and nothing takes them in the other
order, so no cycle exists. `FindBySaleOrder` already skips `Status = 0`, so the voided document
stops blocking anything without a second rule. Cost: the correlativo stays spent and the series
keeps a gap, which SUNAT expects to be declared through a comunicación de baja that this system does
not emit yet.

## The customer-identity rule is shared with the emission path, which changes what a boleta may hide

**Context** — `PostSaleOrder` had to start refusing a sale stamped with a factura series and no
client. The same rule already existed at emission (`buildCustomer`), and two copies of it drifting
apart is the worst outcome: a sale accepted at the till and refused hours later has an id that
cannot be re-keyed.

**Decision** — The rule lives once, in `invoicing/types/customer_identity.go`, as
`RequiresCustomerIdentity` + `ValidateCustomerIdentity`, and both `sales` (creation) and `invoicing`
(emission) call it. Three choices inside it are mine:
- **The boleta threshold now binds at emission too.** `buildCustomer` used to fall back to
  `CLIENTES VARIOS` for *any* anonymous boleta; from S/ 700 it now errors instead.
- **A boleta buyer needs a document of 8 characters or more**, not exactly a DNI's 8 digits, so a
  carné de extranjería or a passport still identifies them.
- **A factura RUC must be 11 *digits***, stricter than `identityDocTypeOf`, which infers a RUC from
  length alone and would let 11 letters through to SUNAT.

**Rationale** — Sharing costs `sales` an import of `invoicing/types`, which the module rules allow
and which is already how the sale id learns its series width. The emission change is the point:
SUNAT rejects an unidentified boleta from S/ 700, so the old fallback only moved the refusal to the
slowest, least fixable place. Cost: a company that was quietly issuing large anonymous boletas will
now see them fail at the till instead — which is the correct failure, but it is a failure that did
not happen yesterday.

## A sale reads the series from the cached company config, and pays for the client with one read

**Context** — Validating the series meant `sales` learning the company's series set, and the open
question recorded in the frontend RATIONALE was what that read should cost per sale.

**Decision** — `cloud.LoadCompanyConfig` (memory cache with a 20 s TTL in front of the sealed blob),
so the common path is no cluster read at all. The buyer costs one read of the client row, and only
when the series actually demands an identity — a small boleta reads nothing. A `ClientID` that
matches no row is also an error now, and a note series (credit/debit) is refused on a sale.

**Rationale** — The blob is the same source the issuer already builds from, so the creation check
and the emission check cannot disagree about which series exist. Cost: a series activated or retired
in the last twenty seconds is judged on stale data — one sale refused, or one sale accepted for a
series that is about to be gone, and the emission check catches the second case.

## The sale id carries the invoicing series, and is minted here rather than by the ORM

**Context** — A sale id was `Autoincrement(2)`: a counter with two random digits. The electronic
document now derives its own id from the sale's, replacing the last two digits with the series it is
issued under, which needs those two digits to be reserved for that purpose.

**Decision** — `sale_order` declares a plain `Keys: db.Cols(e.ID)` and `MakeSaleOrderID` builds
`[counter][rand:2][series:2]` in `sales/types/sale_order_id.go`. The counter is reserved with
`db.GetAutoincrementID` under the ORM's own historical name, `x{companyID}_sale_order_0`.

**Rationale** — The ORM cannot express this layout: its autoincrement appends random digits and then
takes the rest of the key, and the table-level `Autoincrement()` builds a synthetic column with no
settable decimal width, so a non-last packing slot overflows (`insert-update.go:373`, shift of
10^19). Extending the ORM was the alternative; minting here costs nothing now that the reservation
itself is serialized by fareward, and it puts the layout somewhere a reader can find it.

Keeping the ORM's counter name is not cosmetic. The sequence is live, so a fresh counter starting at
1 would mint ids inside the range of rows already written — `1*10000 + …` lands on top of an existing
sale at counter 100. Nothing else writes that row now that the table declares no autoincrement, so
owning it from here is safe. There is a test pinning the name for exactly this reason.

**Consequence worth knowing:** `ResetCounter` skips tables with no autoincrement column
(`deploy.go:458`), so it no longer covers `sale_order` — and it never covered the new correlativo
counters. After a restore, neither can be realigned with the rows that exist. It was already wrong
for this table (it derived the target from the packed key), so nothing regressed, but the gap is now
total rather than partial. Recorded in `invoicing/FINDINGS.md`.

## IssueSeriesID is a request field, not a column

**Context** — The series has to arrive with the request that creates the sale, but it also lives in
the id afterwards.

**Decision** — `SaleOrder.IssueSeriesID` is on the record and not on the table, like
`ActionsIncluded` and `ClientInfo`. Reads go through `SeriesID()`, which derives from the id.

**Rationale** — A stored copy would be a second source of truth for something the key already
encodes. The trap is that a request-only field reads back as zero after a query, so the two are named
differently on purpose: code that reads `IssueSeriesID` off a row it loaded is visibly asking for the
wrong thing. Zero is legitimate on the way in — a till that does not say which series it will invoice
under leaves the tail at 00, and the document takes its own series when it is issued.

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
