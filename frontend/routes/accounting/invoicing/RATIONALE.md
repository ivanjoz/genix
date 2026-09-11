# RATIONALE — accounting/invoicing

## The panel stays open on SUNAT's answer instead of closing

**Context** — The send now waits for SUNAT and comes back with the document's real verdict. The
first version closed the side layer and re-fetched the list, which is what a save normally does.

**Decision** — The layer stays open and `selectedDocument` is replaced with the document that came
back, so the panel re-renders on the new state. The list is re-fetched in a `finally`, so it also
updates when the send failed.

**Rationale** — A rejection is the outcome worth reading, and everything that explains it — the
SUNAT code, the observations, the last error — is in that panel. Closing it would hide the answer
the operator just waited half a minute for. Re-fetching in `finally` rather than after a success is
the same reasoning: a failed send still moved the document, so the row behind it is stale either
way.

## The report filters in the browser, over the delta-synced list

**Context** — The page has to answer "by date or by id" over every document a company has issued.
The table has a local index on `IssueDate`, so a ranged endpoint was available.

**Decision** — `GET.invoices` syncs the whole list incrementally and `filterInvoices` runs in the
browser. No date parameters reach the server.

**Rationale** — The list is one row per invoiced sale and it is delta-cached, so only the first load
is a full read and every later one carries what changed. That also makes the date range, the state
and the id search one predicate instead of three endpoints, and it keeps the filter pure and
testable. Cost: a company with years of documents pays a large first sync, and the moment that hurts
is the moment to add the ranged endpoint the index is already there for.

## Voided and inactive documents stay in the list

**Context** — Most list services in the app drop `ss = 0` rows, and `InvoicesService` could have
done the same.

**Decision** — No status filter at all. A voided document appears with its own state.

**Rationale** — A voided document is exactly what somebody opens this page to look for: the sale was
annulled, the correlativo is spent, and the gap in the series has to be explainable. Hiding it would
make the report lie about the numbering. Cost: the list is longer than the set of live documents.

## "Enviar ahora" is the side layer's save button

**Context** — The layer renders Save/Delete/Close by itself, and the only command the panel offers
is sending a document out ahead of the sweep.

**Decision** — `onSave` is wired to the send and renamed to "Enviar ahora", and it is passed as
`undefined` when the document's state does not allow it, so the button disappears.

**Rationale** — It is the layer's primary action, which is the slot the component reserves for it,
and passing `undefined` is how the component is told an action does not apply — cheaper than
rendering a disabled button in the body that the layer would sit next to its own. The downloads stay
in the body because there are two of them and neither is primary.

## The sale is resolved one id at a time, and an annulled one reads as missing

**Context** — The detail panel shows the sale behind the document, with its lines and product names.
An accounting page has no reason to hold the sale list or the product catalog.

**Decision** — `getRecordByID("sale-order-by-ids", …)` for the sale and `"p-products-ids"` for each
product in it, through the by-ids cache.

**Rationale** — That cache is memory → IndexedDB → server for exactly this shape, so a panel opened
twice costs one round trip. Cost: the cache drops `ss = 0` records, and an annulled sale is `ss = 0`
— which is precisely the sale behind a voided document. The panel says the sale is annulled or
unavailable rather than rendering an empty block, but it cannot show its lines.

## The series code is resolved from the company, not from the document

**Context** — A document row carries its series only as the last two digits of its id. The code
SUNAT sees (`F001`) lives inline on the company record.

**Decision** — The page holds an `EmpresaParametrosService` and maps `SeriesID → SeriesCode`;
`invoiceNumber` falls back to `#<seriesID>-<correlativo>` when no code matches.

**Rationale** — Same source the sale form uses to offer the series, so the two pages cannot disagree
about what F001 is. The fallback exists because a retired series keeps its documents: a number must
still read as something when its series was deleted from the company.
