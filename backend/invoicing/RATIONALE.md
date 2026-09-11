# RATIONALE — invoicing

Design decisions behind the SUNAT integration, newest first. The library's own decisions live in
`backend/facturago/RATIONALE.md`; this file covers the ERP side of the boundary.

Full design in `PLAN.md`. Open problems in `FINDINGS.md`.

## The issuer declares the company's own address, because the establishment code is 0000

**Context** — SUNAT accepted the first document with four observations, all about the same block:
4093 (ubigeo `000001` not in the catalog), 4096, 4097 and 4098 (province, department and district
empty). The address came from the sale's site, and two things were wrong with it: the seeded site
points at `CityLocation` id 1 — which is the *department* AMAZONAS, not a district — and
`BuildIssuer` never filled the three names facturago writes into `cbc:CityName`,
`cbc:CountrySubentity` and `cbc:District`.

**Decision** — `fiscalAddress` builds the block from the company: its `CityID` (the district), the
district, province and department names resolved into the blob, and its legal address line.
`BuildIssuer` no longer takes a site.

**Rationale** — The address travels under `cbc:AddressTypeCode` = `0000`, which is SUNAT's code for
the *main establishment*, meaning the domicilio fiscal. Declaring a branch's street under 0000 is a
misdeclaration however valid the ubigeo is — which is what the module was doing. A branch is
declared under the annex code SUNAT assigned to it, and nothing stores those yet, so today every
document is issued from the fiscal address and the site plays no part in it.

`CompanyConfig.Sites` stays in the blob. Nothing reads it now, and it was put there one turn earlier
at the human's instruction; it is also exactly what the annex-code work will need. Left rather than
removed and re-added.

**The ubigeo is the catalog id.** `city_locations` is keyed by the INEI code itself — 150101 sits
under 1501 under 15 — so the three rows that name an address are one `ID.In` away, and the padding
to six digits is only putting back the leading zeros the integer column drops.

## The sites travel in the config blob, and SiteID 0 resolves to the only active one

**Context** — `BuildIssuer` refused any series with `SiteID = 0` — "no se pudo determinar la sede
que emite el comprobante" — while every seeded series carries exactly that, the series editor labels
it "Dirección legal", and the code comment claimed a fallback that was never written. No company
could emit until somebody assigned a site to the series. The site was also a cluster read on every
emission.

**Decision** — At the human's instruction the sites now travel inside the company config blob
(`CompanyConfig.Sites`, read by `cloud.readSitesForConfig`), and `resolveIssuingSite` answers from
there with no read at all. A series that names a site uses it, retired or not. A series that names
none falls back to the company's **only active site**; with several, it refuses and asks for the
series to name one. The blob version went to 2 so every cached copy is rebuilt.

**Rationale** — The address is declared to SUNAT as the place the document was issued from, so the
one case that can be answered without guessing is the company that has one place — which is every
company until somebody opens a branch. Guessing between two branches would be a misdeclaration on a
legal document, and there is nothing in the data that says which one. The company record was the
other candidate and cannot serve: its `City` is a free-text input, not the six-digit INEI code the
ubigeo needs, while a site's `CityID` already is one.

Retired sites are kept in the blob, unlike retired parameters, because a series outlives the branch
it was opened for and the documents issued under it still have to build. The version bump is not
cosmetic: a version-1 blob decodes fine with an empty `Sites`, and an issuer reading it would refuse
a document that is perfectly issuable.

## "Enviar ahora" waits for SUNAT inside the request, at the human's instruction

**Context** — The endpoint queued an asynchronous send and returned immediately. The human asked
for the opposite: the button must send at that moment and the API must wait for the answer.

**Decision** — `PostInvoiceRetry` runs `SendDocumentNow` inline and returns the document with
SUNAT's verdict already written on it. `SendDocumentAsync` had no callers left and was deleted;
`EmitHandler` stays as the cron retry and the `fn-emit-cpe` command line.

**Rationale** — The operator presses that button precisely to learn the answer, and the previous
shape made them press it, read "puesto en cola", and then go looking for what happened. The cost is
real and was raised before building: SUNAT regularly takes tens of seconds, the client gives up at
45, and an API Gateway deployment cuts a request at 30 — so behind a gateway the caller can time out
while the send completes. That is safe rather than merely tolerable, because the row is written by
`SendDocument` and not by the handler: the state is correct whether or not the answer got back, and
the page shows it on the next load. On the standalone server nothing cuts the request at all.

**Both paths share one function.** `SendDocumentNow` takes the sale's lock, re-reads the state and
sends, and the sweep calls it too — so the rule about what may be sent lives in
`types.CanSendInvoice` and cannot differ between the automatic path and the manual one. It returns
the reloaded row even when the send failed, because a rejection is a state the caller has to show.

## The document is created with the sale, so the builder moved into the types leaf

**Context** — A document is now written when the sale is created, and `sales` is the module that
creates it. A module body may not import another module body, so `ReserveDocument` and
`SaleOrderToDocument` were unreachable from where they had to be called.

**Decision** — Both moved to `invoicing/types` (`sale_order_to_cpe.go`, `document_reserve.go`), and
the reservation split in two: `PrepareDocumentForSale` builds, validates and numbers without
writing, `SaveNewDocument` writes. The series is passed in rather than resolved, which is what keeps
the leaf free of `cloud`. `RebuildDocument`, `LoadDocument` and the whole send path stayed in the
body — they need `LoadCompanySeries`, and that reads the company through `cloud`.

**Rationale** — Everything the builder touches (`crm/types`, `production/types`, `sales/types`,
`db`, `core`, facturago) is legal for a leaf, and no cycle appears. The split into prepare/save is
not cosmetic: it is what lets the caller put every failure before the sale is written and the single
unavoidable write after it. Cost: `invoicing/types` is no longer a small package of table
definitions — it now holds the document builder, which is the biggest piece of logic in the module.

## The sweep is one-shot per company, enqueued by whoever wrote a document

**Context** — Something has to pick up documents sitting in `InvoicePending`. The cron offers a
recurring row per company, a single global tick over a registry of dirty companies (what
`business/product-ecommerce-cron.go` does), or a one-shot enqueued on demand.

**Decision** — One-shot, action id 6, scheduled from `ScheduleEmitPendingSweep` every time a
document is written. `ScheduleCronAction` already dedupes the same logical action inside a frame, so
a hundred sales in five minutes enqueue one row. A run that fills its batch of 50 enqueues the next
one itself.

**Rationale** — No recurring row idles forever for a company that stopped selling, and no registry
of dirty companies to keep in sync. The read it performs is a range read, not a scan: the delta
index on `State` serves a query that pins the partition and the state
(`genix-orm/scylla/index_delta_view_test.go:671`), so no schema change was needed. Cost: the enqueue
is best-effort — it swallows its errors, because the document is already persisted and a scheduler
that cannot write its row must not fail the sale. A company whose only sale of the day hit that
failure has a document nobody sweeps until the next sale; the report shows it as pending and
"enviar ahora" sends it.

## The retry action had never been registered

**Context** — `recordFailure` has always scheduled cron action 5 for a retryable transport failure.
Nothing ever called `RegisterActionHandler` for it, so the executor logged "missing handler" and
skipped the row: no failed document has ever been retried.

**Decision** — Both actions are registered in an `init()` in `emit_worker.go`, next to what they
run: 6 to the sweep and 5 to `EmitHandler`, which was already the right function and was only
reachable as an async lambda entry point.

**Rationale** — Found while adding the sweep, and fixed with it rather than filed: the sweep and the
retry are the same subsystem, and shipping a sweep on top of a retry that never ran would have hidden
the failure mode one layer deeper.

## POST.invoice is deleted; the retry endpoint absorbed "send now"

**Context** — `POST.invoice` reserved a document for a sale. With the document created by the sale
itself, the only thing it could ever answer is "la venta ya tiene el comprobante".

**Decision** — Deleted, and its route id marked retired by the generator. `POST.invoice-retry` now
accepts a document in `InvoicePending` as well as one that failed, and refuses a voided one.

**Rationale** — Pre-alpha: a dead endpoint that can only error is not worth keeping, and the report
needs exactly one button — "send it now instead of waiting for the sweep" — which is the same
operation as a retry with a different reason. Cost: the name no longer describes everything it does.

## GetInvoices fanned out over one state and hid the rest

**Context** — The delta index of `invoice_document` is keyed on `State`, and `GetInvoices` called
`.Delta(updatedVersion, 1)`. On a first sync `Delta` keeps only the values named, so the endpoint
returned only pending documents; a report built on it would have shown a document until it was sent
and then lost it.

**Decision** — `.Delta(updatedVersion, types.AllInvoiceStates...)`, with the list of every state
declared next to the constants.

**Rationale** — The alternative was re-keying the delta index on `Status`, which is a schema change
for a table that has a perfectly good index — the fan-out is what that index is for. The list has to
be complete, which is why it lives beside the constants it enumerates rather than at the call site.

## BuildIssuer reads the config blob; the credentials test still reads the database

**Context** — `BuildIssuer` now takes everything but the fiscal address from
`cloud.LoadCompanyConfig`: identity, SOL credentials and the signing material, which the blob
carries as an unwrapped PKCS#8 key plus leaf certificate rather than the `.pfx` they came from.
That leaves `buildIssuerForTest` — the "test my credentials against SUNAT" button — reading
`loadCompany` and `LoadActiveSecrets` directly, which now looks like a leftover.

**Decision** — It is deliberate and stays. The test button runs right after an upload, against the
row that was just written.

**Rationale** — The write-through that refreshes the blob cannot fail the save, so it may not have
run when the user presses Test. Pointing that button at the blob would test a stale certificate and
report a result about the wrong file — the one failure mode that button exists to rule out. It is
also the only caller for which one extra database read costs nothing.

**Also decided here:** `CertRUC` and `CertValidTo` in the blob are read off the certificate at
compaction, not copied from the `company_secrets` row. The row's copy is what the upload recorded;
the blob's is what will actually sign, and the RUC check before an emission has to be about the
latter.

## A rotation inserts before it retires, and inherits the SOL password it never asked for

**Context** — Uploading a certificate over an existing one now writes a second row and retires the
first, which is two writes with no transaction between them. The panel that sends the upload does
not re-ask for the SOL password either — it is stored, and no endpoint returns it — so the new row
would be born unable to authenticate.

**Decision** — `PostCompanySecrets` reads the row being edited first, copies its `SolPasswordEnc`
into the new one when the body carries no password, inserts, and only then writes `Status = 0` on
the old row. A brand-new record (no predecessor to inherit from) is refused without a SOL password,
the same way it is refused without a certificate.

**Rationale** — the order is what decides how a half-finished rotation fails. Insert-then-retire can
leave two active rows, and `LoadActiveSecrets` takes the active one of the highest id, which is the
certificate that was just uploaded — the right one. Retire-then-insert would leave a company with
no active credentials and no way to invoice until someone noticed. The cost is that a failed retire
needs a second save to clean up, which the panel makes obvious: it lists the retired ones, and the
old row is still sitting among the active.

## The series save through their own endpoint, and the default is settled server-side

**Context** — The series live inline on the company row, so the obvious thing is for the company form
to write them with everything else. That makes one Save button claim two panels, and a stale tab able
to wipe series it never loaded.

**Decision** — `POST.invoice-series` takes **one** series and writes it. `POST.company-parametros`
reads the stored series and carries them forward untouched, so it only writes what its panel shows.
The endpoint owns three things the caller cannot get right on its own: the id of a new series, its
active status, and which series of a document type is the default (`ApplyDefaultSeries`).

**Rationale** — A set-shaped body is what lets a stale client delete series it never saw, so the body
is one series and the reply is the whole set. Being the default is a property of the *set*, though:
sending one series marked default would leave the previous holder still marked, which
`ValidateSeries` then refuses — the save would fail with a confusing message about two defaults. It
is settled where the whole set is visible, and the client just says which series should hold it.

Three bugs this design had, all found by calling the endpoint rather than by reading it:

- nothing set `IsDefault`, so a series added through the API had none and `ResolveSeries` fell through
  to "first active of the type" — right by accident;
- moving the default produced two of them and failed validation, because the client cleared the old
  one locally and then sent only the new one;
- `if body.Status == 0 { body.Status = 1 }` flipped a retire straight back to active, so a series
  could never be retired. Defaulting to active now happens only when the id is being allocated.

The lock is `ActionSaveInvoiceSeries`, keyed on the company: the set is read, changed and written
back, so two saves have to be ordered or the later one drops what the earlier one added.

## Editing one series touches only that series, and only that column

**Context** — The series are one inline array on the company row. The point of giving them their own
endpoint was that saving one must not disturb anything else, and two things were still short of that.

**Decision** — `PostInvoiceSeries` reads the stored set, replaces the one entry whose id matches, and
writes back — under a per-company lock, so the read and the write cannot interleave with another
save. An id the caller names that does not exist is refused rather than appended. And
`SaveCompanySeries` writes with `db.Update(rows, table.InvoiceSeries, table.Updated)` rather than
`db.Insert`.

**Rationale** — A whole-row write is the failure this endpoint exists to prevent, just moved one level
down: `db.Insert` rewrites the RUC, the address and the Culqi keys from a copy read moments earlier,
so a company form saved in between would be silently undone. The two endpoints only stay independent
if the writes are column-scoped as well as the payloads.

Appending on an unknown id was the other gap. Ids come from `NextSeriesID`; honouring a client-chosen
one would let a stale tab burn numbers out of the ninety-nine that exist, or resurrect an id it
remembers from a set it no longer has. Naming a series that is not there is a mistake, not an insert.

The mirror write stays whole-row because `cloud` has no column-scoped update. It is a read replica for
reports rather than the record anything is decided from, and the copy it writes was read inside the
lock a moment earlier.

## A series is never deleted, only deactivated

**Context** — A series started with a trash button in its table row. Deleting one is not something
the data model can support.

**Decision** — There is no delete anywhere: no button in the row, none in the dialog. An "Activa"
checkbox is the whole of it. The table lists every series with the inactive ones sorted to the end and
marked in red.

**Rationale** — `NextSeriesID` is `max + 1` over what is stored, so removing a series would hand its
id to the next one — which would then inherit its correlativo counter (`cpe_{company}_{id}`) and start
numbering mid-sequence, under a code that means something else. Documents already issued resolve their
type and code by that id, so the id has to outlive any interest in the series. A delete button that
can only ever mean "deactivate" is worse than no delete button, because it promises something the
system will not do.

That is also why the table stopped filtering to active series: hiding an inactive one makes it
unreachable, and the checkbox that brings it back lives in its dialog. Sorting them last keeps them
out of the way without hiding them.

`ApplyDefaultSeries` had to become total rather than incremental to make the checkbox safe: an
inactive series is never the default, and any document type left with active series but no active
default adopts one. Without that, deactivating a default left its type with none and a till that names
no series stopped being able to sell it. Verified against the running backend — deactivating a default
stripped it, and reactivating the remaining series of that type promoted it.

## The series is fixed by the sale, not chosen when invoicing

**Context** — With `ID = SaleOrderID`, the document that bills a sale is keyed by that sale. That only
holds if the document is issued under the series the sale already carries in its last two digits, so
the series stopped being something `POST.invoice` could pick.

**Decision** — `seriesOfSale` replaces `resolveSeries` on that path: the series comes from
`order.SeriesID()`. `PostInvoiceBody` no longer takes a `SeriesID`, and its `DocType` is only an
assertion — naming a type the sale's series does not match is refused rather than honoured. A sale
whose tail is `00` cannot be invoiced at all.

**Rationale** — Any other series would key the document where the sale cannot find it, so this is not
a policy choice but the invariant restated. `00` means the company had no electronic invoicing
configured when the sale was made; the sale simply has no document, which is why `FindBySaleOrder` is
a plain primary-key read and correctly finds nothing for it.

What this costs is switching a boleta to a factura after the sale exists. The sale would need a
different id, and the id is its identity in the cash ledger, the warehouse ledger and the summaries —
so the answer is a new sale, not a re-keyed one. Worth knowing before somebody asks for the button.

I had first built both lookups as a scan of the hundred-id block the sale and its documents share,
to cover a `00` sale being invoiced later. That case does not exist: no electronic invoicing means no
document. The scan is gone and the lookups are key reads again.

## The seeded series are six, and they carry no site

**Context** — Every new company should be able to invoice immediately. The four series named were
F001, B001, FC01 and FD01.

**Decision** — Six: `1 F001`, `2 B001`, `3 FC01`, `4 FD01`, `5 BC01`, `6 BD01`. Seeded at all three
company-creation paths — public sign-up, `POST.company` and `exec/init` — with `SiteID` 0. Only the
two selling series are marked default.

**Rationale** — A note has to carry the prefix of the document it corrects, so `FC01` can only correct
facturas. Boletas are the majority of sales, so stopping at four would leave the common case unable to
issue a credit note. The three seeding sites are explicit rather than defaulted in `SelfParse`, because
`POST.company-parametros` writes the record whole: a payload that omitted the field would be re-seeded
over the company's real series.

`SiteID` 0 is not "missing" — it means the company's own fiscal address, SUNAT's main establishment,
coded `0000`. `BuildIssuer` already falls back to it. Requiring a site would have made the seeded
series unsaveable until somebody created one, so the validation that demanded it is gone and the UI
renders 0 as "Dirección legal".

Notes get no default series. `ResolveSeries` now refuses to pick one for a credit or debit note at all:
the right series depends on what is being corrected, and guessing would issue an F-series note against
a boleta whenever the caller stayed quiet.


## The artifacts are named by document id, not by sale id

**Context** — The instruction was to drop `XmlPath`/`CdrPath` and derive the location from the sale
id in a predefined folder.

**Decision** — `cpe/{companyID}/{documentID}.xml` and `cpe/{companyID}/{documentID}-cdr.zip`.

**Rationale** — A sale can carry more than one document: that is the whole point of putting the
series in the tail of the id, so a boleta and the credit note correcting it can coexist. Naming the
objects after the sale would have the note overwrite the boleta's signed XML — the one artifact there
is a legal obligation to keep for five years. The document id contains the sale's counter and random
digits, so the folder still reads as "this sale's documents"; it just distinguishes them.

The cost of dropping the columns is that nothing records whether the file exists. `GetInvoiceXML`
refuses outright only while the document is still `Pending`; past that it asks storage and lets a
"not found" answer be the answer.

## The lines are rebuilt from the sale on every send

**Context** — Dropping the `Detail*` arrays means the row can no longer reconstruct its own document,
which the previous design treated as the point of storing them: "a retry that happens tomorrow must
produce the same XML even if the product was renamed today."

**Decision** — `RebuildDocument` loads the sale and re-runs `SaleOrderToDocument` on every attempt.
The correlativo and the issue date come off the row, not the clock.

**Rationale** — The immutability guarantee that matters is the signed XML in object storage, which
exists from the first successful `Emit` and is what SUNAT accepted. Before that point there is nothing
to be inconsistent with. What could still drift is a product renamed between reserve and send, which
changes a line description on the retry — a cosmetic difference in a document SUNAT has not accepted,
and the sale itself is frozen while it has a live document.

Pinning the number and the date on the row is the part that is not optional: rebuilding those from
the clock would declare the day the retry succeeded rather than the day the document was issued,
which is a rejection.

## Series filtering moved to the client

**Context** — The series is in the tail of the id rather than in a column, so there is nothing to
index. `PLAN.md` specifies a list filtered by date, series and state.

**Decision** — Indexes on `IssueDate` and `SaleOrderID`, delta on `State`, none on the series.
Filtering a list by series happens on the client.

**Rationale** — `GET.invoices` is a delta endpoint, so the frontend already holds every document it
is allowed to see; filtering a set it has in memory needs no index. Adding a `SeriesID` column purely
to index it would reintroduce exactly the duplication this change removed. If a server-side series
filter is ever needed for volume reasons, the column is the answer and it can be added then.

## Three small removals that followed from the reduction

**Context** — Cutting the columns exposed code that only existed to serve them.

**Decision** — `SunatDescription` is gone, derived from `SunatCode` via `sunat.ErrorDescription`.
The `identityDocTypeCode`/`igvTypeCode` converter pairs are gone with the client and line columns
that stored their output. `recordFailure`'s two identical branches are collapsed into one.

**Rationale** — All three were duplication the columns had been hiding. The dead branch is the one
worth naming: it was reported in `FINDINGS.md` as a possible missing distinction between a retryable
and a final failure, and the answer is that there is none — the state is the same either way, and
what differs is whether a retry is scheduled. Collapsing it says so.

## The annul message no longer names the SUNAT series code

**Context** — `sale_order_annul.go` refused an annulment with "La venta tiene el comprobante F001-123
emitido". `SeriesCode` is no longer on the document; it lives on the company.

**Decision** — The message names the series number and the correlativo instead.

**Rationale** — Resolving the code needs the company record, and `sales` reaching it would cross a
module body — the same boundary that put `FindBySaleOrder` in `invoicing/types` in the first place.
Passing the code down would mean `sales` loading the company to format an error. The series number
and correlativo identify the document well enough for somebody to go find it, and the alternative was
storing a copy of the code on every row to keep one error message shorter.


## Series ids are never reused, and deleting one retires it

**Context** — Series moved inline onto the company, so the set is now rewritten whole on every save
and a row can simply vanish from the slice. The id is what partitions the correlativo counter and
what a document points at.

**Decision** — `NextSeriesID` is `max + 1` over the whole set, never the first free slot. The table's
delete button sets `ss = 0` and clears `IsDefault` rather than removing the element. Code uniqueness
is only enforced among active series.

**Rationale** — Reusing id 3 after deleting it would silently attach every document ever issued under
the old series to the new one, and their correlativos share a counter — so the new series would start
numbering from wherever the old one stopped. Retiring instead of removing keeps the row that the
documents' `SeriesID` resolves against, which is what makes a five-year-old document still printable.
The cost is that the ninety-nine ids are consumed by every series a company ever had, not just its
live ones; at a handful of series per company that ceiling is not reachable in practice.

Letting a retired code reappear is the other half: a company that re-registers B001 with SUNAT has to
be able to enter it, and blocking that would have made a retired series permanently poison its code.

## The company reader is duplicated rather than shared

**Context** — `invoicing` has to read a company to get its series. Companies live in Scylla only when
the cloud mirror is off, so every reader needs the same two-branch lookup, and `config` already had
one. The obvious move is to promote it to `config/types`, where a module body may import it.

**Decision** — `invoicing/company_series.go` carries its own copy. `config/empresas.go` keeps its
private one.

**Rationale** — The shared version was written and reverted: `config/types` would have to import
`app/cloud`, and `cloud`'s own test reads `config.Company` to prove every mirrored schema compiles.
That closes an import cycle in the test binary. The alternatives were worse — dropping Company from
that test defeats its stated purpose of covering *every* mirrored table, and it is the only thing
standing between a bad index declaration and a broken deploy. Eight duplicated lines is the cheaper
half of that trade, and the comment at both sites says why so neither gets "cleaned up" later.

## `loadCompany` in issuer.go ignored the cloud mirror

**Context** — Unifying the readers surfaced that `issuer.go`'s `loadCompany` queried Scylla directly.

**Decision** — It now delegates to the mirror-aware reader and keeps only the RUC check that is its
own concern.

**Rationale** — On a mirrored deployment companies are not in Scylla, so `BuildIssuer` would have
failed with "la empresa no existe" on every emission. Not a regression from this change — it was
always that way and nothing had run against a mirrored deployment yet. Recorded because it is a
behaviour fix that was not asked for, and because the RUC check moving out of the shared reader is
deliberate: reading a company's series is legitimate before anyone has typed a RUC.

## `WarehouseID` is gone from the series

**Context** — `InvoiceSeries` carried `SiteID` and `WarehouseID`. Only one of them is read.

**Decision** — `WarehouseID` dropped. `SiteID` stays.

**Rationale** — `SiteID` feeds the establishment code and fiscal address in the XML, which SUNAT
requires. `WarehouseID` was declared, written by nothing, and read by nothing — a column that only
ever held zero and suggested a relationship the emission path does not have.

## Adding the first series of a type makes it the default

**Context** — A till that names no series falls back to the default for the document type, and a set
with no default for a type cannot issue that type at all.

**Decision** — `makeSeries` sets `IsDefault = 1` when no active series of that document type exists
yet. Clicking the column moves the default rather than toggling it off.

**Rationale** — The failure it prevents is silent and late: series configured, first sale refused with
"la empresa no tiene una serie configurada". There is no meaningful state where a type has exactly one
series and it is not the default, so making the user click twice to reach the only useful arrangement
is friction with no upside. Not offering an "off" click follows from the same thing — clearing the
last default breaks issuing for that type.

## Open findings went in their own file, and one of them contradicts this one

**Context** — A read of the emission path turned up three verified defects, the worst being that the
correlativo allocation can hand the same number to two concurrent inserts. They were asked to be
written down; nothing was fixed.

**Decision** — They live in `FINDINGS.md`, not here, and the section below titled *The correlativo is
the ORM's autoincrement, packed into the key* is left standing rather than edited.

**Rationale** — This file is for decisions that hold; a problem with no chosen fix is not one, and
folding open questions into a record of settled ones makes the record unreadable for review. Leaving
the correlativo section untouched is the part worth flagging: its closing claim — that a duplicate
becomes a primary-key collision — is **false**, because Scylla upserts on a duplicate key and the ORM
issues no LWT. Rewriting it now would mean guessing which of the three fixes in `FINDINGS.md` §1 gets
chosen, and the honest state is that none has been. The cost is that this file contains a paragraph
known to be wrong until that decision is made, which is why the pointer is in the header rather than
only in the section.

**Context** — Phase 5 was written, unit-tested and building cleanly before anything was sent. The
first real emission against SUNAT's beta service failed three times in a row, each for a different
reason, none of which any test had caught.

**Decision** — All three fixed, and recorded here because the pattern is worth remembering.

**Rationale** — What they had in common is that each depended on something only the real system
knows.

*The stored correlativo was always zero.* `SelfParse` derived it from the packed key, but the ORM
assigns that key during the insert, **after** `SelfParse` has run — so the column was written as
zero and then disagreed with the number the document was actually issued under. The column is gone;
`Correlativo()` reads it off the key, leaving one source of truth. A stored copy of a generated key
cannot be right.

*`Autoincrement(8)` is eight random digits, not eight digits of counter.* The first document came
back numbered `171652747`. The ORM appends that many random digits to the counter, which is right
for an opaque id and wrong for a document number — a correlativo has to be the clean sequence
1, 2, 3. `Autoincrement(0)` is the correct spelling, and it is not a default: it is a requirement.

*Validation ran before the number existed.* `ReserveDocument` validated the document, and validation
rejects a correlativo of zero — which is what it always is before the insert. It now validates
against a stand-in, and `Emit` validates again at send time with the real number.

None of these are logic errors in the sense a unit test finds. They are all mistakes about what a
dependency does — the ORM's ordering, the ORM's naming, the ORM's assignment point — and the only
thing that surfaced them was running the real path against the real service. Worth keeping in mind
for phase 6: the same class of mistake is still available in the ticket flow.

## An unidentified buyer is declared, not rejected

**Context** — The first emission also refused to build: the sale had no client, and the mapper
required one. Every walk-in sale in the database has `client_id = 0`, which is most of them.

**Decision** — A boleta with no customer declares an unidentified buyer: document type `0`, number
`00000000`, name `CLIENTES VARIOS`. A factura still requires a real RUC and refuses without one.

**Rationale** — Somebody paying cash at a till and leaving is the ordinary case, not an error, and
SUNAT still requires the customer block to be present — so it has to be filled with something, and
this is the convention every point of sale in the country uses. A factura is the opposite: it names
a business by definition, so refusing is the correct answer and the message says why.

## The till prices with IGV included, so the split is done here and passed explicitly

**Context** — `SaleOrder.DetailPrices` is what the customer paid: the shelf price, IGV included.
SUNAT wants the operation declared as a net value plus the tax on it. Something has to divide one
into the other, and the obvious place is facturago, which knows the rate.

**Decision** — The split happens in `sale_order_to_cpe.go`, in integers, and both halves are handed
to facturago explicitly as `Value` and `IGV` rather than letting it derive them. `net` is the gross
divided through the rate and rounded; `tax` is whatever remains, so the two always sum back exactly.

**Rationale** — A price of S/ 100.00 has no exact net value: 84.7457… either way. Letting the
library compute the tax from a rounded net gives 84.75 + 15.26 = 100.01, and the invoice then totals
one cent more than the till charged — the kind of discrepancy that is discovered by a customer, not
by a test. Deriving the tax as the remainder keeps the document tied to the money that actually
changed hands. The cost is that the declared tax can differ from an exact 18 % of the declared net
by a cent, which is inside what SUNAT tolerates and is the same tradeoff every Peruvian POS makes.

## Reserving a number and sending it are separate steps

**Context** — Issuing needs a correlativo, and transmitting takes seconds and fails in ways that say
nothing about the document. Doing both in the request that asked for it means the till freezes
whenever SUNAT is slow, which is most afternoons.

**Decision** — `ReserveDocument` numbers and persists; `SendDocument` signs and transmits. The
handler does the first and fires the second as an asynchronous invoke, returning immediately.

**Rationale** — The customer's receipt does not depend on SUNAT's answer: the document exists and is
numbered the moment the row is written. Persisting in between is also what makes a retry safe — it
reuses the number it already has instead of burning a new one per attempt, so a series does not
develop a gap for every timeout. The row carries the lines in full for the same reason: a retry
tomorrow must rebuild the same document even if the product was renamed today.

## Boletas are sent individually — assumption, not a decision

**Context** — SUNAT accepts a boleta two ways: transmitted on its own like a factura, or reported in
the next daily summary. The regulation describes the summary; most providers send individually and
get an immediate CDR. This was raised three times and has not been answered.

**Decision** — Boletas go through `SendBill` like everything else, and the daily summary is kept for
voiding. `resolveSeries` also defaults to a boleta when the caller names no document type.

**Rationale** — It is what the approved plan says and it gives the better till experience, so it was
the safer thing to build first rather than block on. It is recorded here as an assumption because it
is a compliance question, not a technical one, and the answer belongs to an accountant. Switching is
cheap and contained: the routing is the one branch in `SendDocument`, and the summary path already
exists and is verified against beta.

## Schema IDs are confirmed by the validator, not by reading

**Context** — A table declares a numeric `TableSchema.ID` that is packed into cache keys, so two
tables sharing one silently share cached slots. Picking a free number looks like a job for grep.

**Decision** — The four tables took 26, 42, 53 and 54, chosen by running `check_tables` and moving
whatever it rejected.

**Rationale** — Three greps produced three different answers and all were wrong: 44 and 45 belong to
`user_logs` and `request_errors`, and 47 to `server_metrics`.

Every failure had the same cause — a comment. A pattern anchored on `ID:` following the opening
brace misses the three tables that explain themselves first; widening it to three lines still misses
`server_metrics`, whose `ID:` sits four lines down; and a window that scans to the first nested brace
fails outright on the thirteen schemas whose comments push it past the window. The tables that
defeat a regex are the well-documented ones, which is the opposite of the tables one can afford to
miss.

The real mistake was measuring text layout as a proxy for what the ORM claims at runtime.
`check_tables` compiles every schema and calls `ClaimTableID`, so it sees what the program sees.
There was never a regex that would have been right, only one that had not been wrong yet — so when
the next table is added, run the validator rather than reading.

The tooling changes that would stop this recurring are written up in `docs/TABLE_ID_ASSIGNMENT.md`.

## The abandoned encoding/xml model took two demo functions with it

**Context** — `invoice.go` and `invoice_types.go` modelled the UBL with `encoding/xml` struct tags,
which cannot produce canonical output and therefore cannot be signed. `exec/demo.go` had two
scratch functions, registered as `fn015` and `fn016`, whose only purpose was to dump that model.

**Decision** — All four are gone: the two files, the two functions, and their registry entries.

**Rationale** — `encoding/xml` controls neither attribute order, nor namespace prefix declarations,
nor self-closing tags, so its output is not canonical and a signature over it could never validate;
UBL element order is also conditional on the document type, which struct tags cannot express.
Neither is fixable by editing those files. The demo functions called `NewInvoice` and
`MakeSignature`, so once those were gone they could not compile, let alone run — there was no
version of this change that kept them. Their replacement is `facturago`, whose output is pinned by
golden files and accepted by SUNAT rather than printed to a log.

## SUNAT logic lives in a separate repository

**Context** — Electronic invoicing is a large body of knowledge (catalogs, UBL, XMLDSig, SOAP,
error codes) that has nothing to do with this ERP, plus a small amount of orchestration that has
everything to do with it. Keeping both in `backend/invoicing/` would have been about thirty files.

**Decision** — The SUNAT half is `github.com/ivanjoz/facturago`, a standalone zero-dependency
library, wired in the same way as `genix-orm`: submodule at `backend/facturago` plus a `replace`
directive. `backend/invoicing/` keeps eleven files: tables, the `SaleOrder` mapper, the issuer
decryption, the emitter, the worker and the handlers.

**Rationale** — The hard, exacting part — canonical XML and a signature that must match byte for
byte — becomes testable with nothing running, and it can be finished and verified against SUNAT
BETA before this repo touches it. It also keeps the ERP out of a public repository. The cost is a
third submodule to keep pushed, and the discipline that facturago must never learn a Genix type.

## The correlativo is the ORM's autoincrement, packed into the key

**Context** — SUNAT numbering must never repeat within a series, and the backend runs as concurrent
Lambdas. The obvious approaches are a counter row read-then-written under the lock service, or a
correlativo assigned by the caller.

**Decision** — `InvoiceDocument.ID` packs `DocTypeSeries` (6 digits) with an 8-digit autoincrement,
with `AutoincrementPart: DocTypeSeries`. The ORM allocates from a Scylla `counter`, which is atomic,
and the number is recovered as `ID % 1e8`.

**Rationale** — The SUNAT identity of the document becomes its primary key, so a duplicate is a
primary-key collision rather than a business rule someone has to remember to check — and no lock is
held across a network call to SUNAT. The cost is that a failed emission burns a number; that is
acceptable because SUNAT tolerates gaps, and the row is persisted before signing so a retry reuses
its own number instead of taking a new one.

## Emission is an async invoke, with the cron-action as the retry net

**Context** — `sendBill` can take 40 seconds and the API sits behind a 30-second gateway, so the
emission cannot happen inside the request. The existing tool for deferred work is the cron-action
scheduler, whose frames are aligned to five minutes.

**Decision** — `POST.invoice` persists the document and fires `cloud.ExecLambda` with
`InvokeAsEvent`, returning immediately. A retryable failure schedules a cron-action at 5, 15 and 30
minutes.

**Rationale** — Five minutes is an eternity at a point of sale, and the async invoke starts the
emission now while behaving identically in both runtimes. The cron-action is still worth having for
what it is genuinely good at: it survives a panic, caps at ten attempts and leaves an auditable row.
A rejection (SUNAT 2000–3999) is never rescheduled — that XML will be invalid forever.

## Secrets are decrypted per call and never cached

**Context** — The certificate and the SOL password live encrypted in `CompanySecrets`. A warm Lambda
serves many companies, so anything cached in process memory is shared between tenants.

**Decision** — `IssuerFromSecrets` decrypts with `core.Decrypt` and builds a `facturago.Issuer` for
that one emission. Nothing is memoized. The `.pfx` is parsed by facturago on each call.

**Rationale** — A cached issuer is a document signed with another taxpayer's certificate, which is
the worst failure this module could have. Parsing a PKCS#12 costs single-digit milliseconds against
a SUNAT call measured in seconds, so there is nothing to optimize. Note that `core.Decrypt` ignores
an explicitly passed key (`core/helpers.go:1477`) — this module must never rely on that parameter.
