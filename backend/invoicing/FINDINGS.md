# FINDINGS — invoicing

Open correctness findings against the code as it stands. Nothing here has been fixed; no file was
changed while these were written. Each one was verified by reading the code, not inferred from
`PLAN.md` — where the two disagree, this file follows the code.

This is not `RATIONALE.md`: that file records decisions that stand, this one records problems that
are still open and need a decision.

Findings 1 and 3 are in `genix-orm`, which is a separate repository — fixing them means committing
and pushing inside the submodule.

---

## 1. The correlativo allocation is not collision-proof — RESOLVED

**Fixed by the fareward sequence service** (`docs/SEQUENCE_SERVICE_PLAN.md`, wired unconditionally in
`backend/db/autoincrement.go`). The daemon is now the sole writer of the `sequences` table and
allocates in hi-lo blocks under a per-name lock, so two concurrent callers can no longer be handed
the same value. `db.GetAutoincrementID` goes through it too, which is what makes an explicit
correlativo reservation safe.

Finding 1b — no LWT on the insert — is unchanged and still true: a duplicate key is a silent Scylla
upsert, not an error. It no longer has a way to happen through the counter, which was the only
mechanism that produced one in practice.

The original finding is kept below because the reasoning is what justified the design, and because
1b is still live.

---

**Original finding. Severity: high — this is a compliance problem, not a retryable error.**

`types/invoice_document.go:51-63` and `PLAN.md:450-461` both rest on the same claim:

> Numbering is atomic without any lock being held across a call to SUNAT, and a duplicate becomes a
> primary-key collision rather than a rule somebody has to remember to check.

Neither half holds as implemented.

### 1a. The counter is read-then-increment

`genix-orm/scylla/main.go:177-205`:

```go
Query(&result).Name.Equals(name).Exec()                                            // :180 read
currentValue, counterIncrement := nextCounterRange(storedCounterValue, increment)  // :190 derive
QueryExec("UPDATE ... SET current_value = current_value + ? WHERE name = ?", ...)  // :199 write
return currentValue, nil                                                          // :204
```

The `UPDATE` is atomic, but the value **returned** is derived from the read at `:180`. Two concurrent
callers on the same counter both read `5` and both return `6`. The stored counter still lands on `7`
— both increments apply — so the damage is invisible in the sequence itself.

CQL has no atomic fetch-and-add read-back for counters, so this cannot be fixed inside `GetCounter`.

### 1b. A duplicate key does not collide

Every statement the ORM generates is a plain `INSERT INTO ... VALUES ...`
(`genix-orm/scylla/insert-update.go:430`, `:518`, `:976`). There is no `IF NOT EXISTS` anywhere in
the ORM. In Scylla that is an upsert: the second write silently overwrites the first — no error, no
constraint violation, nothing to catch in `ReserveDocument`.

### What actually happens

Two sales invoiced in the same `(CompanyID, DocTypeSeries)` inside the window between `:180` and
`:199` get the same correlativo. The second `db.Insert` at `emitter.go:59` overwrites the first row.
Both documents are then queued for SUNAT under the same número: SUNAT accepts one and rejects the
other as a duplicate, and the ERP has lost a row it is legally required to keep for five years.

The fareward lock in `invoice_api.go:44` does not cover this — it is keyed on `SaleOrderID`, so two
*different* sales sharing series B001 are never serialized against each other. That is deliberate
(`invoice_api.go:42-43`: "unrelated sales never queue behind each other") and correct for the
duplicate-submission problem it was written for; it just does not protect the counter.

### Why this table specifically

`sale_order` runs the same code path (`sales/types/sales.go:126`) but uses `Autoincrement(2)`. Those
two random digits mean a lost race still collides only one time in a hundred, which has been masking
the underlying race everywhere else in the ERP.

`invoice_document` uses `Autoincrement(0)` (`types/invoice_document.go:232`) — correctly, because
SUNAT requires a clean 1, 2, 3 sequence, and `RATIONALE.md` records the emission that came back
numbered `171652747` when it did not. So the fix for one problem removed the accidental mitigation
for another, in the one table where the consequence is worst.

### Options

| | Approach | Cost |
| --- | --- | --- |
| a | Widen the lock: acquire on `(DocTypeSeries)` around `ReserveDocument` only. | Cheapest, contained to this module. **Does not violate the stated design goal** — that goal is no lock held *across the SUNAT call*, and transmission is already async (`SendDocumentAsync`, `invoice_api.go:66`), so the lock covers one local insert. Cost: two tills selling on the same series serialize for the duration of an insert. |
| b | `IF NOT EXISTS` (LWT) on the insert, ORM-wide or opt-in per table. | Correct and makes the doc comment true. Touches `genix-orm` globally; Paxos is expensive and would slow every insert unless opt-in. |
| c | Read back after insert and reallocate on mismatch. | No schema or ORM change, but it is a repair loop over a problem the other two prevent, and the read-back is not free either. |

Whichever is chosen, the doc comment at `types/invoice_document.go:56-59` and `PLAN.md:460-461` must
be corrected — they currently assert a guarantee the database does not provide.

---

## 2. The retry cron action is never registered, so the retry net is inert

**Severity: medium — degraded automation, no data loss.**

`emit_worker.go:143-150` schedules cron action `5` whenever a failure is retryable. Nothing
registers a handler for id 5. Every registration in the backend:

```
sales/main.go:23                      -> 2
business/product-ecommerce-cron.go:18 -> 3  (productsDbRebuildActionID)
webpage/domain_cleanup_cron.go:17     -> previousDomainCleanupActionID
```

`invoicing/main.go` has no `init()` at all; the other modules register theirs in one. The executor
skips unknown ids (`core/cron-action-scheduler.go:296-300`):

```go
actionHandler, exists := actionHandlerMap[pendingAction.ActionID]
if !exists {
    Log("RunPendingCronActions missing handler:", "cron_id", ..., "action_id", ...)
    continue
}
```

So a retryable transport failure sets `State = InvoiceException`, increments `RetryCount`, writes a
cron row — and that row is logged once and dropped forever. Today the only paths that actually
re-send are the initial async invoke and the manual `POST.invoice-retry`.

`PLAN.md:596` reserved id 5 ("IDs to reserve: cron-action `5`"); it was reserved and never wired.

**Fix:** an `init()` in `invoicing` calling
`core.RegisterActionHandler(emitRetryActionID, "Reintentar emisión de comprobante", EmitHandler)`.
`EmitHandler` already has the right signature and is already the body of both paths
(`emit_worker.go:46-59`). The document keeps its correlativo across a retry either way, so nothing
is lost beyond the automation.

---

## 3. `ResetCounter` is removed, not fixed — counters are never realigned

**Severity: medium. Not destructive, but a restore is now knowingly incomplete.**

The implementation was wrong in three ways and only ever right for a table whose key is a bare
`Autoincrement(0)`:

- the counter name hardcoded the autoincrement part as `0`, so a table declaring `AutoincrementPart`
  had its real per-part counters missed and a phantom one written instead;
- the target was `max(key)`, but a key packs the counter with random digits and any `KeyIntPacking`
  columns, so the figure was orders of magnitude too large;
- it skipped tables with no autoincrement column — which, since sale and document ids became
  caller-built, is both of the tables anyone would want to realign.

Rather than keep writing a wrong value confidently, `resetCounterForTable`
(`genix-orm/scylla/deploy.go`) is a no-op and `exec/restore.go`'s `ResetCounters` prints one warning
naming what was skipped. Both carry a TODO describing what a correct version needs.

**What this means in practice.** Counters keep climbing from wherever they are, so an id is never
reused — only skipped. That is the safe direction to fail in. The case it does not survive is a
restore into a *fresh* keyspace: every counter starts at zero while the restored rows carry high ids,
and the next insert collides with one of them.

A correct implementation also has to reach further than the old one did. The ids this project mints
itself — sale orders and invoice correlativos, both allocated with `db.GetAutoincrementID` under their
own counter names — are invisible to any table-driven walk and must be reset by name.
`applyCounterReset` is the primitive: it already routes through the fareward allocator, so a live
reservation is dropped in the same critical section as the move, which is what opcode `0x08` was
built for.

## 4. A Modal's Save button is invisible to the product's own agent

**Severity: low, but it is not specific to this feature.**

`Modal` registers itself with a `close` method and nothing else — its Save button never calls
`agentRegister`. Reading `/company/configuration` through the agentic API shows
`26 Modal "Nueva serie" close` and no handle for saving, so the agent can open a dialog, fill it, and
then have no way to commit it.

Found while verifying the invoicing-series dialog, which is why that endpoint had to be exercised
over HTTP instead. It affects every modal in the app, not this one, so the fix belongs in
`packages/genix-ui/layers/Modal.svelte` rather than here.

## 5. Minor

- ~~**Dead branch** in `recordFailure`.~~ **Resolved.** There was no missing distinction: the state is
  `InvoiceException` either way, and what differs is whether a retry is scheduled. The branch is
  collapsed and says so.

---

## `PLAN.md` drift

`PLAN.md` was not updated as phases 4 and 5 landed. The code is right in each case below; the plan is
stale. Listed so nobody implements phase 6 from a stale spec.

| `PLAN.md` | Says | Code | Verdict |
| --- | --- | --- | --- |
| `:450-458` | The correlativo is an ORM autoincrement packed into the key | The key is plain; the id is `DocumentIDForSale(sale, series)` and the correlativo is a column reserved with `db.GetAutoincrementID` | **Plan superseded.** The whole packed-key section no longer describes the code. |
| `:460-461` | "a double submission cannot duplicate numbering" | The counter is serialized by fareward, and one sale plus one series maps to one key | Now true, by a different mechanism than the plan describes. |
| `:481-483` | A retry re-sends the signed XML already in S3, to preserve the digest | `SendDocument` rebuilds from the **sale** and re-signs on every attempt | Code deviates, now more so. The digest changes between attempts and `DigestValue` is overwritten, so the QR matches whatever was last sent. Reasoned through in `RATIONALE.md`; the plan's objection still has not been answered on its own terms. |
| `:504` | `ReferenceDate.DecimalSize(5)` + `Autoincrement(3)` | `IssueDate.DecimalSize(5)` + `Autoincrement(0)` (`types/invoice_summary.go:98-99`) | Code right and deliberate — the comment at `types/invoice_summary.go:33-34` explains the identifier is built from the send date, and `:96-97` explains the clean sequence. |
| `:553` | Retry caps at ten attempts | `maxRetries = int8(6)` (`emit_worker.go:32`) | Code right; plan stale. |

---

## Not findings — known gaps, already planned

Recorded only so they are not re-reported as bugs. All are phase 6/7 in `PLAN.md:671-672`:

- `Ticket` columns exist (`types/invoice_document.go:121`, `types/invoice_summary.go`) and
  `saveDocument` writes the column (`emit_worker.go:183`), but nothing ever sets it and
  `sunat.Client.Status(ticket)` is never called. Boletas go out individually via `SendBill`, so no
  ticket is produced yet.
- `InvoiceSummary` (table 54) is defined and registered but has no read or write path anywhere. RC
  and RA are not implemented; `facturago.EmitSummary` / `EmitVoided` / `SendSummary` are unused.
- Credit and debit notes are modelled (`AffectedDocID`, `NoteReasonCode`, `NoteReason`) but no
  handler emits one, which is why `sales/sale_order_annul.go:86-95` refuses to annul an invoiced
  sale.
- No frontend exists for issuing documents. `access.toml` grants `accounting/invoicing`, which does
  not exist yet; the live driver is `exec/invoicing_beta.go`. Series are the exception — they moved
  inline onto the company and are maintained from `company/configuration`.
- `sunat.Configure` is never called, so the client runs on its 45 s default timeout. `PLAN.md:695`
  flags that the worker Lambda needs a timeout >= 60 s; still unconfirmed against the deployment.
