## fn-init seeds a company that can already operate, and the bootstrap page is gone

**Context** — A company needs a Site, a Warehouse and a CashBank before it can do anything. The
sign-up wizard (`/welcome`) collects them as step 3, reusing `InitialDataForm`. But `fn-init`
writes companies 1 and 2 by hand and created none of those rows, so login had to report
`InitialDataPending` and redirect to a standalone `/initial-data` page — a second entry point
existing only for the companies the wizard never touched.

**Decision** — `seedCompanyOperatingRecords` in `exec/init.go` gives every seeded company the same
three records `PostInitialData` creates. With that, `InitialDataPending` is always false, so the
flag, `hasPendingInitialData` and the `/initial-data` route are all deleted. The wizard is the only
path, and `InitialDataForm.svelte` stays because it is what the wizard renders.

**Rationale** — the standalone page existed to compensate for an incomplete seed. Completing the
seed removes the reason for it rather than leaving two ways to reach the same three inserts, one of
which almost never ran and was therefore the one that rotted.

Deleting `hasPendingInitialData` also takes two ScyllaDB queries off every login. It was the one
part of login that touched ScyllaDB at all — users live in the cloud store — which is why it needed
a `recover()` and a "degrade to false" path that no longer has to exist.

The seeded rows use autoincremented IDs rather than literal ones, so unlike the companies and users
above they need no `reserveSeededAutoincrementIDs`: an autoincremented insert advances its own
counter.

**The accepted cost:** a company whose warehouse or cash bank is later deleted has no guided way
back. It has to be fixed from the Sites & Warehouses and Cash Banks pages, which is where those
records are managed anyway.

## Company 1 is exempt from credit budgets, not from permissions

**Context** — The credit limiter meters every request through `chargeConfiguredCredits`. It also
meters the platform operator's own company, so an exhausted budget locked company 1 out of its own
software — including the "Datos Iniciales" bootstrap, which is exactly when someone needs to get in.

**Decision** — `server_utils.CreditExemptCompanyID = 1` (the company `fn-init` seeds as
"Principal"). In `chargeConfiguredCredits`, that company's CPU and inference credits are zeroed. If
the frame then carries no required access there is nothing left to ask and the call returns nil;
if it does carry one, the frame still goes to the daemon and the access is still enforced.

**Rationale** — the seam is the single funnel every charge already passes through: API usage, the
GET top-up and agent inference all reach the daemon through it, so one guard covers all three
rather than three guards that can drift apart.

Zeroing the credits rather than skipping the frame is what keeps the exemption narrow. It mirrors
the split `ChargeAPIAccessOnly` already makes for the credit-exempt routes: no budget, same
authorization. Returning early on an empty frame is required, not an optimization —
`encodeCharge` rejects a frame carrying neither credits nor an access, so a blanket short-circuit
would have turned every exempt read into a limiter error.

**The accepted cost:** company 1 is unmetered, so its usage does not appear in the credit
accounting and cannot be capped. That is the intent — it is the operator, not a tenant — but it
does mean a runaway loop under company 1 has no budget backstop.

## Product quantities are a (whole units, sub-units) pair, not a scaled integer

**Context** — Sale orders could not express a fraction of a product. The `Sbu*` sub-unit columns
on `Product` existed but no backend code read them: the sale page pushed a sub-unit count straight
into `DetailQuantities`, so selling 3 candies deducted 3 boxes. Any fix has to serve two different
needs at once — divisible goods (rice by the gram) want fractions, packaging (a box of 6) wants an
integer factor, and 1/6 is not representable in any decimal scale.

**Decision** — `core.Quantity{Units, Sub int32}` plus a divisor, in `core/quantity.go`. The value is
`Units + Sub/Divisor`, an exact rational. `Units` is never scaled; all fractional information lives
in `Sub`/`Divisor`, so continuous goods are simply divisor 1000 and discrete ones divisor 6. `Sub` is
deliberately **not** normalized in storage: it may exceed the divisor and may be negative.

It is stored two ways, split by what the row is for. **Aggregates keep two flat `int32` columns** —
stock, the movement ledger and the sale summaries, everything that is accumulated or `SUM()`-ed.
**Documents pack** both halves into one `int32` as `Units*1000 + Sub`, so a sale-order line reads
1001 at divisor 6 as one box plus one candy. `core.Quantity` is a computation type assembled from
either form, never a column type itself.

**Rationale** — leaving `Sub` unnormalized is what makes addition closed without a carry:
`{a1,b1} + {a2,b2} = {a1+a2, b1+b2}`, order-independent. That buys two things a packed
`whole*1000+sub` integer cannot. The stock engine keeps accumulating with a plain `+` — no
decompose/normalize/recompose with signed borrow in the two hottest write paths — and Scylla can
still `SUM()` both columns independently, so `product-supply-management.go`'s pushdown survives.
Normalization is confined to validation and display, where it is isolated and unit-testable.

A packed integer breaks on the first subtraction: `stock.Quantity = prev + mov.Quantity` computes
`1000 - 4 = 996` (0 units + 996 sub-units, invalid and ~498x too large, and it passes the existing
`< 0` guard) where the answer is `2`. That argument only binds where rows are accumulated, which is
why documents pack and aggregates do not: nothing ever adds two sale-order lines together or asks
the database to sum them, so a line pays one column instead of two. `PostSaleOrder` is the single
conversion point.

The packing slot is 3 digits, and both ceilings it imposes are per line, reaching no aggregate:
`MaxQuantityDivisor` is 1000, because a normalized `Sub` has to fit 0..999 — a kilogram sold by the
gram fills it exactly, and cannot be refined further afterwards. `MaxQuantityLineUnits` is 2,147,483
whole units on one line.

**Costs accepted:** two columns per quantity instead of one, and a divisor that has to travel with
them. `int64` at facturago's `QuantityScale = 1_000_000` would have removed every headroom concern
and made the SUNAT bridge an identity cast, but with `Units` unscaled the `int32` ceiling is already
~2.1 billion whole units — the same magnitude as today — so the extra width bought nothing. Note the
ORM only *reports* a column type mismatch (`genix-orm/scylla/deploy.go:952`) and Scylla will not
convert `int` to `bigint` in place, so that choice would have meant recreating tables.

Full analysis and the rejected alternatives: `docs/QUANTITY_SCALE_PLAN.md`.

## The boundary rule is enforced by a script, not by good intentions

**Context** — Go cannot catch a module-to-module import: `accounting` importing `finance`
compiles cleanly. The six violations this refactor removed had accumulated exactly that way,
and two of them had already produced copy-pasted code rather than a compile error. Left
unchecked the rule would decay again.

**Decision** — `scripts/boundaries/check_module_imports.go`, dispatched as
`go run . check_module_imports` and registered in the `deploy.sh` TUI beside `check_tables`.
It reads `go list -json ./...` from `backend/` and classifies every import edge against the
layer table. The rule and its consequences are documented in
`backend/docs/MODULE_BOUNDARIES.md`, with pointers from `AGENTS.md`.

**Rationale** — the layer membership is written as **literal path lists**, not inferred from
directory shape. A new top-level package then fails loudly ("not classified") instead of being
silently guessed into the wrong layer, and `rg moduleBodies` shows the whole policy. Test
imports are included, since a `_test.go` in `accounting` importing `finance` is the same
violation.

Writing the test first paid for itself: it caught that the checker allowed
`finance/types -> cloud`, because the infrastructure allowance was evaluated before the
`*/types`-must-stay-a-leaf rule. That edge does not exist today, so nothing would have
surfaced it until someone added it — which is exactly the case a checker is for. The order is
now leaf-rule first, and `check_module_imports_test.go` pins all six original violations plus
the leaf property.

The checker was also verified end-to-end by injecting `accounting -> finance` and confirming a
non-zero exit and a message naming the fix, then reverting.

