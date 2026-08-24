# RATIONALE — accounting/assets

Design decisions for the Activos page, newest first.

## The payment form is a Modal, and it leads with the amount

**Context** — The payment fields sat inline in the asset detail panel, under a "Compra" heading,
with the Dispose button on its own line below. The panel is mostly a read surface — balances,
depreciation ledger — so a live form in the middle of it competed with the numbers it changes.

**Decision** — Payment moved into `Modal id=11`, opened by a button that now shares a row with
Dispose. The "Compra" subtitle is gone. Inside the dialog the amount input comes first, not the
cash-register select.

**Rationale** — The field order is load-bearing, not cosmetic: `Modal.focusDialogContent()` focuses
the dialog's first `input`, and `SearchSelect` opens its dropdown on focus, so leading with the
register covered the date field with the option list. Leading with the amount also lands focus on
the field users actually edit, since it opens pre-filled with the outstanding balance. The dialog
carries `css="min-h-0!"` because `Modal` hard-codes `min-h-460`, which left ~300px of dead space
under a two-row form.

## The depreciation period label is composed here, not read off the row

**Context** — The ledger rows arrive from `expenses` with no `Name`: the backend stopped storing
`"Depreciación 7/60"` because a persisted label cannot be translated.

**Decision** — `depreciationSchedule(entries, totalMonths)` in `assets.ts` sorts the rows by
`Date` and numbers them, returning a `Depreciation n/N|Depreciación n/N` label per row.

**Rationale** — Sorting on `Date` rather than trusting arrival order means the number refers
to the period, not to when the row happened to be inserted — a re-run that interleaved entries
would still number them correctly. It costs one array copy per render of the panel.

## Serials decide granularity, so the form asks for them instead of asking the question

**Context** — Buying five identical laptops is either five assets or one. Both are legitimate: it
depends on whether the business tracks them individually.

**Decision** — The acquisition form has a Quantity field and a Serial Numbers textarea, one serial
per line. Entering serials produces one asset per serial; leaving it empty produces one grouped
asset with the quantity on it.

**Rationale** — The user already knows whether their laptops have asset tags. Asking "track these
individually?" as a separate checkbox is asking them to answer the same question twice, and lets the
two answers disagree. The presence of serials *is* the answer, and it matches the stock layer, where
a serial is exactly what promotes a unit to its own `ProductStockDetail` row.

## The page runs depreciation on load

**Context** — Depreciation is generated lazily by `POST.asset-depreciation-run`. Something has to
call it or the register shows stale book values.

**Decision** — An `$effect` calls it on mount, refetches the assets if anything was posted, and
otherwise says nothing. Failures log to the console rather than raising a toast.

**Rationale** — The run is idempotent — the backend filters each period against the asset's own
watermark — so calling it on every visit is safe, and it removes the need for a cron. A failure here
must not block the page: the register is still readable with values that are merely a month behind,
and a red toast on every load would be worse than a slightly stale number.

## Pure math lives in `assets.ts`, not in the service

**Context** — Book value, percentage depreciated and remaining months are needed by the table, the
detail panel and (eventually) the balance sheet.

**Decision** — They are plain functions in `assets.ts` with no Svelte and no fetch; `assets.svelte.ts`
holds only the reactive service and the API calls.

**Rationale** — The project convention, and it pays off here specifically: remaining months is
derived from accumulated depreciation rather than from the calendar, so it agrees with the ledger
even before a run has happened today. That rule is worth being able to test without a component.
