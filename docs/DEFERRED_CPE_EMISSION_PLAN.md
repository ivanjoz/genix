# Deferred CPE emission — plan

The electronic document is created when the sale is created, in `InvoicePending`, and a deferred
cron sends every pending document to SUNAT. A new `accounting/invoicing` page reports what was sent
and what was not.

Status: **implemented**, approved and built as written below. The decisions it records now live
next to the code — `backend/invoicing/RATIONALE.md`, `backend/sales/RATIONALE.md` and
`frontend/routes/accounting/invoicing/RATIONALE.md` — and this file is kept as the shape of the
change as a whole.

One thing was found while building and fixed with it: cron action 5, the per-document retry, had
never been registered with `RegisterActionHandler`, so no failed document had ever been retried.
Both actions are registered now.

## Decisions already taken by the human

| Question | Decision |
| --- | --- |
| States | Keep the current numbering: `0 = InvoiceVoided`, `1 = InvoicePending`. No migration. |
| Annulling a sale with a pending document | The document is voided (`State = 0`) and never sent. The correlativo is left as a gap in the series. |
| The document cannot be created | Fail-closed: the whole sale is rejected. |
| "Descargar el HTML" | Out of scope. The report downloads the XML and the CDR that already exist. |

The *comunicación de baja* (RA) that SUNAT expects for a voided correlativo is **not** part of this
work — `invoicing/FINDINGS.md` already lists it as unimplemented, and this plan adds one more reason
to build it.

## Today

```
PostSaleOrder ──insert sale (series in the id tail)──> done. No document, ever.

POST.invoice (no caller anywhere) ──ReserveDocument──> row in State 1 ──async lambda──> SUNAT
```

## After

```
PostSaleOrder ──validate ──> insert sale ──> reserve correlativo ──> insert document (State 1)
                                                                          │
                                                        schedule per-company sweep (cron, 5 min)
                                                                          ▼
                                            EmitPendingDocuments ──> SendDocument ──> SUNAT
                                                                          │
                                                       failure ──> existing per-document retry (action 5)
```

---

## A. Move the document builder into `invoicing/types`

`sales` is a module body and may not import the `invoicing` body
(`backend/docs/MODULE_BOUNDARIES.md`). Everything the reservation needs is already legal for a
`*/types` leaf — `db`, `core`, `crm/types`, `production/types`, `sales/types` and facturago — and no
cycle appears (`crm/types` and `production/types` import only `core` and `db`; `sales/types` does
not import `invoicing/types`).

**Moves from `invoicing/` to `invoicing/types/`:**

- `sale_order_to_cpe.go` whole: `SaleOrderToDocument`, `buildCustomer`, `buildLines`,
  `loadProductDescriptions`, `splitGrossAmount`, `identityDocTypeOf`, `sunatDocType`.
- From `emitter.go`: `ReserveDocument` and `rowFromDocument`, renamed `ReserveDocumentForSale`,
  taking the `*InvoiceSeries` as an argument instead of resolving it — the caller already has it.
- Their tests (`sale_order_to_cpe_test.go`, the reservation half of `emitter_test.go`).

**Stays in the `invoicing` body:** `RebuildDocument`, `LoadDocument`, `seriesOfDocument`,
`loadSaleOrder`, `BuildIssuer`, the whole `emit_worker.go`. They need `LoadCompanySeries`, which
reads the company through `cloud` — off limits to a leaf.

## B. Create the document inside `PostSaleOrder`

In `sales/sale_order_issuance.go`, extending what already validates the series and the buyer:

1. Build the document from the in-memory sale and the resolved series.
2. `CompleteTotals` + `ValidateDocument` against a stand-in correlativo — every content error
   surfaces here, before anything is written.
3. Reserve the correlativo from the series counter.
4. `db.Insert(sale)` — unchanged, already there.
5. `db.Insert(document)` with `State = InvoicePending`.
6. `core.ScheduleCronAction` for the per-company sweep.

Steps 1–3 run before the sale is written, so a rejected document rejects the sale with nothing
persisted. A sale that names no series (`IssueSeriesID = 0`) skips all of it.

**Residual risk, stated plainly:** steps 4 and 5 are two writes with no transaction between them.
A database failure at step 5 leaves a sale with no document. The request returns an error, so the
operator sees it, but the row stays. The alternatives are worse — an orphan document if the order is
reversed, or a scan over sales to find the gap, which the partition layout cannot serve cheaply.
Logged loudly and left; a repair sweep can be added later if it ever happens.

**Correlativo gaps** become normal: a sale rejected between steps 3 and 4, or annulled later, leaves
a number unused.

## C. Annulling a sale voids its pending document

`sales/sale_order_annul.go:86` refuses to annul any sale that has a document. With a document on
every invoiced sale, that would make annulment impossible. New rule:

- `State == InvoicePending` → void it (`State = InvoiceVoided`, `Status = 0`) and continue with the
  annulment. `FindBySaleOrder` already skips `Status == 0`, so the sale reads as un-invoiced
  afterwards.
- `State >= InvoiceQueued` → the current refusal stands: it reached SUNAT, so it needs a credit note.

Both the annulment and the sweep take the existing `core.ActionInvoiceSaleOrder` lock on the sale id
before reading the state, which is what stops the sweep from sending a document that is being voided
in the same second.

New in `invoicing/types`: `VoidPendingDocument(companyID, saleOrderID)`.

## D. The deferred sweep

- **Action id 6** (2, 3, 4 and 5 are taken), registered as `"Enviar comprobantes pendientes"`.
- **Scheduled per company, one-shot, 5-minute frame**, from step B.6. `ScheduleCronAction` already
  dedupes the same logical action inside a frame, so a hundred sales in five minutes enqueue one
  row. No recurring row idles forever and no global registry is needed.
- **The handler** reads `CompanyID.Equals(x).State.Equals(InvoicePending)`. This needs no schema
  change: the `TypeDelta` index on `State` serves a plain pinned query
  (`genix-orm/scylla/index_delta_view_test.go:671`).
- Bounded batch per run (50). If the query fills the batch, the handler re-schedules itself for the
  next frame instead of running long.
- Per document: take the sale lock, re-read the state, `SendDocument`. That function already stores
  the XML and the CDR, records what SUNAT answered and schedules the per-document retry (action 5)
  for transport failures. Nothing about it changes.

## E. The API the report reads

- **`GetInvoices` is broken for a report and gets fixed.** It calls `.Delta(updatedVersion, 1)`,
  and the delta index is keyed on `State`, so a *first* sync returns only pending documents.
  Changed to fan out over every state: `.Delta(updatedVersion, 0,1,2,3,4,5,6)`.
- **`GET.invoice-xml` is reused as-is** for both artifacts (`?tipo=cdr` for the CDR).
- **`POST.invoice` is deleted.** With the document created by the sale, it can only ever answer
  "la venta ya tiene el comprobante". `POST.invoice-retry` absorbs its useful half: it becomes
  "enviar ahora", accepted for a pending document as well as for a failed one, which is the button
  the report needs. `access.toml` id 35 and `core/api_routes.generated.go` are updated with it.
- **The sale behind a document** is read with the existing `GET.sale-order-by-ids`: a sale document
  is keyed by its sale, so the id is already in hand.

## F. The page — `frontend/routes/accounting/invoicing/`

`access.toml` id 35 already grants the route and the APIs; `frontend/core/modules.ts:147` has the
menu entry and only lacks `route: "/accounting/invoicing"` — which is why it does nothing today.

```
+page.svelte              Filters, table, detail Layer. No business logic
InvoiceDetailLayer.svelte The document, its SUNAT answer, the sale's lines, the downloads
invoicing.svelte.ts       GetHandler service over GET.invoices + the by-ids sale fetch
invoicing.ts              Pure: state labels and colours, the date/id filter predicate
invoicing.test.ts         Vitest over the filter and the state mapping
DOCUMENTATION.md          Support-facing (skill: document-user-routes)
RATIONALE.md              Design decisions
```

- **Filter by date** — over the delta-synced set in the browser. The list is one row per invoiced
  sale and syncs incrementally, so only the first load is a full read. If it ever outgrows that, the
  local index on `IssueDate` is already there to serve a ranged endpoint.
- **Filter by id** — the document id, the correlativo, or the sale id (the same number).
- **Columns** — serie · correlativo, fecha, cliente, total, estado, and what SUNAT answered
  (`SunatCode`, `SunatNotes`, `LastError`, `RetryCount`).
- **Actions** — descargar XML, descargar CDR, enviar ahora (`POST.invoice-retry`), ver la venta.

## Work order

1. A — the move. Mechanical, tests come with it. `go build`, boundary checker, tests green.
2. B + C — creation and annulment. This is the pair that changes behaviour; done together so the
   suite is never in a state where sales cannot be annulled.
3. D — the sweep.
4. E — the API fixes and the deletion.
5. F — the page, then its `DOCUMENTATION.md`.

## Not in this work

- The *comunicación de baja* for voided correlativos.
- Credit and debit notes (still unemitted, so a sent document still blocks annulment).
- The printable representation (HTML/PDF).
- The `resumen diario` for boletas.
