# RATIONALE — Accounting Books

## The export blockers are a notification sent after "Consultar", not a banner

**Context** — The amber banner explaining why TXT SUNAT is disabled moved into
`sendUserNotification`. A notification is permanent, while the banner re-rendered on every change,
so it needed a moment to be sent.
**Decision** — `loadBook` sends one yellow message after the read finishes, and only if
`exportBlockers` is not empty. The title names the period (`SET 2026`), and the body lists one
blocker per line. The banner is gone.
**Rationale** — Sending from a `$effect` would repeat the message whenever the rows or the company
record re-sync. Tying it to the user's read sends it once per question they asked. The cost: after
the popup closes, the disabled TXT button no longer says why on the page itself. The reason stays
in the ⓘ list, and "Consultar" sends it again.

## A nota de crédito is negated in fields 15/17/26, and that is a guess

**Context** — the annex text is not in the repo, and the two readings of it disagree. A nota de
crédito either travels with negative importes in the ordinary amount fields, or it leaves those
at zero and puts the reduction in fields 16 and 18, which the field table describes as "Dscto BI"
and "Dscto IGV", *negativo, sólo `07`/`87`*.

**Decision** — the first reading. `salesBookRowSign` returns `-1` for a credit note and the
writer applies it to fields 15, 17, 19, 20 and 26; 16 and 18 stay at `0.00`. It is one exported
function, it is tested, and the comment on it says it is unverified.

**Rationale** — the book has to net credit notes out somewhere or the period overstates the sale,
and of the two readings this is the one the RVIE proposal is built on. Guessing silently was the
thing to avoid, not guessing: the alternative was to refuse to export any period containing a
note, which blocks a real month over a question a single look at the annex settles. Cost: if the
reading is wrong, every filed period containing a credit note is wrong in the same way — which is
why it is one function and not a sign scattered through the writer.

## The exporter refuses a period rather than filing it incomplete

**Context** — three things can make a period unfilable: a comprobante SUNAT has not answered, a
comprobante in a currency other than PEN (field 28, tipo de cambio, is mandatory then and the ERP
stores no rate), and a row whose series never joined, which would go out with fields 7 and 8
empty.

**Decision** — `salesBookExportBlockers` returns every reason at once, the page lists them, and
the TXT button is disabled while the list is not empty. A period with **no** comprobantes is not
a blocker: it exports as a "sin información" file.

**Rationale** — all three would produce a file SUNAT rejects, and a rejected upload says far less
about why than the page can. Returning a list rather than the first reason means one read of the
month tells the whole story instead of one round trip per problem. The empty period is the
opposite case and the file name encodes it — position 30 of the Tabla 6 mask is exactly the flag
for "this month has nothing", so refusing to write it would be inventing a rule. Cost: a company
that genuinely invoices in dollars cannot file at all until field 28 has a source, which is
stated in PLAN.md §5 rather than worked around.

## `downloadTextFile` encodes Latin-1, not UTF-8

**Context** — the browser's natural `Blob` of a string is UTF-8 with no BOM, and SUNAT reads
these files as ISO-8859-1. A single `Ñ` in a razón social is two bytes under one encoding and one
under the other.

**Decision** — the helper in `libs/helpers.ts` maps each code unit to a byte and substitutes `?`
above 255.

**Rationale** — the failure it prevents is the worst kind: the file uploads, most of it parses,
and one taxpayer's name comes back mangled in the Reporte de Información Recibida. `?` for the
unrepresentable is deliberate — it is visible in the file, where a silent drop would shift
nothing and be noticed by nobody. Cost: the helper is Latin-1 and not general, so the next caller
that wants a UTF-8 download has to be told.

## The Go mapper was deleted rather than kept for the exporter

**Context** — moving both joins to the browser left `accounting/types/sales_book.go` with no
caller: the handler no longer knows a document's type, because that comes off the series and the
series comes off the company record the backend stopped reading. Phase 4's TXT writer is
server-side and will need the same consolidation.

**Decision** — deleted, with its thirteen tests ported to `books.test.ts`. Nothing was kept back
for phase 4.

**Rationale** — pre-alpha keeps no speculative code, and a mapper with no caller is a mapper
nobody notices going stale. The rules are not lost — every one of them is pinned in vitest, and
the port is where the joined data now is. The cost is real and is written down in PLAN.md §4:
when the exporter is built it must implement consolidation again in Go, reading the company it
already needs for fields 1-2, and the two implementations have to be kept honest against each
other or the screen and the filed file will disagree. That is a phase-4 problem with the company
in hand, not a reason to carry dead code until then.

## `AffectedDocuments` travels as its own array, not folded into the rows

**Context** — a credit note names the comprobante it corrects (fields 29-32), and that document
is very often in an earlier month, so it is not among the period's rows.

**Decision** — the handler sends those documents in a second array and the page joins them.

**Rationale** — the alternative was to flatten the four columns onto the note's row in Go, which
is the same "map on the server" the rest of this change removed, and it would have forced the
backend to resolve the affected document's *series* to know its type. As a separate array it is
raw data: the same shape as `Documents`, joined by the same code, and the page decides what to
print. Cost: the response carries a handful of documents that are not rows of this book, and the
reader has to know why they are there.

## A structural `IBookSeries` rather than importing `IInvoiceSeries`

**Context** — the mapper needs three fields off a series: the id, the document type and the code.
`IInvoiceSeries` has them, but it lives in `routes/company/configuration/`.

**Decision** — `books.ts` declares its own `IBookSeries` with exactly those three fields. The
company record satisfies it structurally, with no import.

**Rationale** — routes are leaves; one route importing a type from another is the coupling this
same change removed from the till. A structural type costs one interface and says precisely what
the book reads, which is less than a series is. Cost: if `IInvoiceSeries` renames one of the
three, nothing fails until the map comes back empty at runtime.

## The identity catalog moved out of the client-provider service

**Context** — `books.ts` is pure by contract, and it needs `sunatIdentityDocCode`, which lived in
`services/crm/client-provider.svelte.ts` next to the service class. Importing it dragged the HTTP
runtime and the by-ids cache into a pure module — and broke its test suite outright, because
vitest cannot transform the TypeScript that chain reaches inside `node_modules`.

**Decision** — the catalog, `deriveIdentityDocType` and `sunatIdentityDocCode` are in
`services/crm/identity-doc.ts`. The service file keeps the class, the POST and the by-ids fetch.

**Rationale** — a pure lookup table has no business sitting behind a Svelte service, and the test
failure was the proof rather than the motive. It stays under `services/` and not `core/` because
the `//CATALOG:` generator only walks `routes` and `services` (`scripts/generators/sync_catalogs.go`),
so moving it to `core/` would have silently stopped it syncing from the Go catalog. Two importers
were updated; nothing was re-exported, so there is one path to each name.

## The buyer's document type stays a string while the comprobante type became a number

**Context** — `SalesBookRow.DocType` moved from SUNAT's `"01"` to the catalog id `1`, so the row
now mixes conventions: the comprobante type is an id, and `BuyerDocType` next to it is still the
character SUNAT prints, produced by `crm.SunatIdentityDocCode`.

**Decision** — left as it is. Only `DocType` and `ModifiedDocType` changed.

**Rationale** — the two are not the same kind of value. `DocType` is stored as an id on
`InvoiceSeries` and the page needs to label it, so carrying the code was a conversion applied one
step too early. `BuyerDocType` is stored as an id on `ClientProviderSnapshot` and the book only
ever prints it — nothing keys a picker on it — so converting at the mapper is converting at the
boundary, which is where it belongs. Cost: reading the struct, the inconsistency looks like an
oversight; the field comments say which is which.

## `SunatDocCode` survives with no caller

**Context** — once the row carries the id, nothing in the mapper converts to `"01"`, and
CLAUDE.md §3 says to delete deprecated code freely.

**Decision** — kept, with a test that pins the padding for all four types and the empty string
for 0.

**Rationale** — it is not deprecated, it moved: the conversion still has to happen, just at the
flat-file writer instead of the mapper, and phase 4 is the next thing that needs it. An uncalled
function does rot, so the test is what keeps it honest — `%02d` is exactly the kind of thing that
gets "simplified" into `%d`, and campo 7 unpadded is a rejected file. Its frontend twin
`sunatDocCode` in `$core/sunat-doc-type` is tested the same way.
**Context** — SUNAT allows sub-S/700 electronic boletas to be declared as a daily range, and the
range carries no buyer. Some of those boletas do have an identity: the customer gave a DNI or a
RUC at the till, and the document declared it.
**Decision** — They are consolidated anyway. Only three things cut a range: the S/ 700 threshold,
a non-Activo validity, and a gap in the correlativos.
**Rationale** — The only threshold Anexo 3 states for a boleta is S/ 700; identity is not a
criterion. Breaking on it would fragment a day's range for a reason SUNAT never asked for. The
identity is not lost either — it stays on the sale and on the document, it simply does not travel
to the book, and a boleta grants no crédito fiscal for it to matter to. Cost: the book cannot be
used to answer "who bought on this boleta"; the sale can.

## A fourth validity state for documents SUNAT has not answered
**Context** — Tabla 9 has three states (Activo, Baja, Rechazado), all of which presuppose a CDR.
This ERP holds documents that are reserved, queued or failed in transport, which are none of them.
**Decision** — `SalesBookPending`, a fourth state that is explicitly not SUNAT's. Those rows keep
their amounts and are counted on the page; the exporter is expected to refuse a period that holds
any of them.
**Rationale** — The alternatives were both wrong: declaring them Activo files operations SUNAT may
yet reject, and hiding them makes the book silently understate the month. A visible fourth state
turns "this period is not ready" into something the page can say before anyone files it. Cost: one
state the flat-file writer must translate or refuse, never emit.

## Fields this ERP cannot fill are absent from the row, not zero
**Context** — Anexo 3 has 33 informed fields. This ERP can never fill exportación (14), the two
descuentos (16, 18), ISC (21), IVAP (22-23), ICBPER (24), otros tributos (25) or the contract id
(33) — and some of those are "obligatorio" with a `0.00`.
**Decision** — `SalesBookRow` omits them. The flat-file writer emits the literal `0.00`.
**Rationale** — A column that can only ever be zero still has to be read from the database,
serialized, sent, and rendered, and it invites somebody to wonder whether it means something. The
annex's requirement is about the *file*, so it is met where the file is written. Cost: the day one
of them becomes real — ICBPER is the likely one, the moment plastic bags are sold — it has to be
added to the row rather than merely populated.

## A new access id rather than the empty "Facturación" placeholder
**Context** — `access.toml` already had id 22, `name = "Facturación"`, group 7, with no
`frontend_routes` and no `backend_apis`. It looked like a free slot for this page.
**Decision** — Left untouched. Added id 36, `Libros Contables`.
**Rationale** — Ids 22, 23 and 24 are an unwired group (Facturación, Estados Financieros, Balance)
that predates `accounting/invoicing` getting its own id 35. Whether they are dead or reserved is
not visible from the file, and an access id that is already granted to a profile means something
to whoever granted it — repurposing one silently changes what a user can reach. Cost: one more id,
and 22 stays for someone to decide about.

## The single-day period is the word "day", not an empty string
**Context** — The month selector needs an option that hands control back to the date picker, and
the natural value for "no period" is `""`.
**Decision** — The sentinel is `"day"`, read through `isWholeMonthPeriod()`.
**Rationale** — `SearchSelect` treats a falsy option id as "nothing is selected"
(`getSelectedFromProps` returns early on `!currValue`), so an empty-string option can be clicked
but never displays as chosen, leaving the field blank. Verified in the running app. Cost: one
predicate instead of a truthiness check, which is the clearer read anyway.
