# RATIONALE — accounting/assets

Design decisions for the Activos page, newest first.

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
