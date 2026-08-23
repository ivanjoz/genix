# RATIONALE — invoicing

Design decisions behind the SUNAT integration, newest first. The library's own decisions live in
`backend/facturago/RATIONALE.md`; this file covers the ERP side of the boundary.

Full design in `PLAN.md`.

## Three bugs the live emission found, and what they have in common

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
