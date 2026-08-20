# PLAN — Company credit meters on the cards (daily + remaining)

## Goal

On every company card in `SYS > Empresas`:

1. **Remove the chart's x-axis date labels** (they collide at card width and add 18px of noise).
2. **Add two meters** under the chart, each a two-cell bar (CPU green `#10b981`, AI purple `#a855f7`)
   with a proportional fill and the figure right-aligned in mono:
   - `DAILY` — credits **left today**: `DailyCPU − day_used`, `DailyInference − day_used`.
   - `CREDITS` — credits **left this month**: `MonthlyCPUCeiling − month_used`, same for inference.

The numbers must be the *same numbers the limiter enforces on*, not an independent re-derivation.

## Where the enforcement numbers live today

`server_utils/src/limiter/quota.rs:392-420` — a charge is refused when either:

| Window  | Check                                                        | Source of the counter |
| ------- | ------------------------------------------------------------ | --------------------- |
| Daily   | `company_aggregate.day_used + requested > stored.daily`       | `ShardState.subjects[(company, -1)].day_used` (in memory) |
| Monthly | `budget.month_used + requested > stored.monthly_ceiling`      | `ShardState.budgets[company].month_used` (in memory) |

Neither counter is persisted. Entitlement (`daily`, `monthly_ceiling`, `last_set`,
`budget_month_start_day`) *is* persisted, in `company_credit_budget`, by `upsert_budget`
(`storage.rs:140-150`) — a statement that names entitlement columns only.

On a cold miss the daemon rebuilds the counters from the usage rows it flushed:

- `ensure_budget` (quota.rs:546-600) sums `credit_usage_company` for the daily frames
  `month_start_day .. today` → `month_used`.
- `ensure_subject` (quota.rs:690-730) sums today's daily row → `day_used`.

`backend/config/company_credit_budget.go:246-289` (`getCompanyCreditBudget`) reimplements the first
of those sums in Go, per company, on every read. That is the whole reason the cards page cannot show
remaining credits today: doing it for N companies means N month-range scans.

## Decision: the daemon flushes its counters into `company_credit_budget`

`flush_dirty()` (quota.rs:747-795) already runs every 15s (`main.rs:95-115`, `config.flush_interval`)
and already takes each shard's mutex — the same mutex that owns `budgets` and `subjects`. Adding a
second pass that writes each **dirty** company's counters costs no extra reads and no new lock.

Consequences:

- The panel reads the daemon's own counters. `remaining = ceiling − month_used` is computed **once**,
  in Go, from the values the enforcement path used.
- `getCompanyCreditBudget` drops its 31-partition month scan and becomes a single point read.
- The cards report gains **one** read of a table with one small row per company.
- Staleness is bounded by `flush_interval` (15s), the same bound the usage charts already carry.

Rejected alternatives: a new table (nothing else would live in it; `company_credit_budget` is already
one row per company, keyed by `company_id`); Go re-summing the month for every company (N scans per
poll); the frontend deriving month-to-date from cached day rows (duplicates the formula in TS, and a
30-day window cannot cover a 31-day month).

---

## Step 1 — Schema: usage counters on `company_credit_budget`

`backend/core/types/company_credit_budget.go` — add to **both** structs (`CompanyCreditBudget` and
`CompanyCreditBudgetTable`), keys and indexes unchanged:

| Field                 | Type    | Meaning |
| --------------------- | ------- | ------- |
| `UsageDayPeriod`      | `int16` | The day the day counters belong to, indexed exactly as the daemon indexes `SubjectState.day_period` (see Step 2's note). A reader whose current period differs treats the day counters as 0. |
| `DayCPUUsed`          | `int64` | `day_used.cpu` of the company aggregate at flush time. |
| `DayInferenceUsed`    | `int64` | `day_used.inference`. |
| `UsageMonthStartDay`  | `int16` | The month the month counters belong to (`time_frame::month_start_day`). Mismatch with the current month ⇒ month counters read as 0. |
| `MonthCPUUsed`        | `int64` | `budget.month_used.cpu`. |
| `MonthInferenceUsed`  | `int64` | `budget.month_used.inference`. |
| `UsageUpdated`        | `int32` | `SUnixTime` of the flush that wrote the six fields above. `0` means "never flushed" and is what Step 3's fallback keys on. |

Written **only** by the daemon; the Go backend never writes this table (it mutates through the
daemon, `core.MutateCompanyCreditBudget`). Column adds are applied by the ORM deploy path
(`backend/genix-orm/scylla/deploy.go:863`, `ALTER TABLE … ADD`), so no hand-written migration.

Then: `cd scripts && go run . check_tables` (skill `static-project-validation`).

## Step 2 — `server_utils`: flush the counters

`src/limiter/storage.rs`
- New `BudgetUsageSnapshot { company_id, usage_day_period, day_used: Credits, usage_month_start_day,
  month_used: Credits, updated }`.
- `LimiterStore::upsert_budget_usage(snapshot)`, prepared as an INSERT naming **only** the seven new
  columns + `company_id`, idempotent. Keeping it separate from `upsert_budget` is what stops a usage
  flush from clobbering entitlement written a millisecond earlier by a `mutate_budget` (and the other
  way round).
- Extend `select_budget` with the new columns only if a reader needs them — the daemon does not; it
  recovers counters from the usage rows, which stays the authority on restart. Leave it alone.

`src/limiter/quota.rs`
- `CompanyBudgetState` gains `version` / `flushed_version` (mirroring `UsageRecord`, aggregation.rs:36-82)
  bumped whenever `month_used` changes (the `charge` path) or the month rolls over in `refresh_month`.
- `SubjectState` gains the same pair for `day_used`, bumped in `charge` and on a day rollover in
  `refresh_periods` — only the company aggregate's state is ever snapshotted, but the field is
  cheaper than a special case.
- `flush_dirty()` gains a pass, after the usage pass, per shard: for each dirty
  `budgets[company_id]`, read `subjects[(company_id, COMPANY_AGGREGATE_USER_ID)].day_used`, build one
  `BudgetUsageSnapshot`, `upsert_budget_usage`, then `mark_flushed` under the shard lock. A company
  with no traffic since the last flush writes nothing. Failures log and stay dirty, exactly as usage
  snapshots do. Shutdown already calls `flush_dirty` (`main.rs:175`), so no extra wiring.
- Log line: `company_id`, `day_cpu`, `month_cpu`, `month_start_day` at `debug`, plus the existing
  `written` count at `info`.

**Pre-existing quirk to fix here, or the DAILY meter is unexplainable.** `SubjectState.day_period`
is `unix_seconds / 86_400` — a **UTC** day — while the daily usage row it is seeded from uses
`time_frame::daily`, the **Lima (UTC-5)** business day. So the daily cap currently resets at 19:00
Lima, and a restart reseeds it from a window that does not match. Proposal: derive `day_period` from
`time_frame::daily(unix_seconds)` in `SubjectState::new`/`recovered`/`refresh_periods`, making the
enforcement window, the persisted row and the meter agree. `UsageDayPeriod` then stores that frame's
unix day. **Flagged for approval — it changes enforcement behaviour, so it is a separate commit.**

Tests (`quota.rs` `#[cfg(test)]`, existing style):
- a charge marks the budget dirty; a flush writes one usage snapshot and leaves it clean;
- a second flush with no charge writes nothing;
- a month rollover zeroes `month_used` and re-dirties the row;
- day rollover likewise (and, with the fix above, on the Lima boundary).

## Step 3 — Backend Go

`backend/config/company_credit_budget.go`
- `getCompanyCreditBudget` reads the row and uses the flushed counters:
  `MonthCPUUsed`/`MonthInferenceUsed` when `UsageMonthStartDay == currentMonthStartDay()`, else `0`.
  `CurrentCPU`/`CurrentInference` keep their meaning (`remainingCredits(ceiling, used)`).
- **Fallback**: when `UsageUpdated == 0` the daemon has never flushed for this company (rows written
  before this feature, or a daemon that has not charged anything since the deploy), so keep the
  existing month-range scan for that one company. Self-healing: the first charge after deploy makes
  the daemon rebuild and flush the true total.
- Response gains the daily side, which the modal already implies but never showed:
  `DayCPUUsed`, `DayInferenceUsed`, `DailyRemainingCPU`, `DailyRemainingInference`
  (`remainingCredits(DailyCPU, dayUsed)`), zeroed when `UsageDayPeriod` is not the current period.
- New helper `getCompanyCreditBudgetSummaries()` — one read of the whole table
  (`db.Query(&rows).AllowFilter()`, the pattern `getCompaniesUpdatedSince` already uses on a
  comparably small table) mapped to the same computed shape.

`backend/config/company_credit_usage.go` — `GetCompanyCreditUsageReport` gains a `Budgets`
collection alongside `Days` and `Routes`:

```go
type companyCreditBudgetMeter struct {
    ID                      int32 // = CompanyID, the delta-cache key
    DailyCPU                int64
    DailyInference          int64
    DailyRemainingCPU       uint64
    DailyRemainingInference uint64
    MonthlyCPUCeiling       int64
    MonthlyInferenceCeiling int64
    RemainingCPU            uint64
    RemainingInference      uint64
    IsCurrentMonth          bool
    Updated                 int32 `json:"upd"` // = currentDailyTimeFrame()
    Status                  int8  `json:"ss"`
}
```

`upd` is the current daily frame, and the rows are emitted on **every** response regardless of the
client's watermark — the same contract `Days` uses for today's row (`company_credit_usage.go:96-125`):
the figures change continuously, so a delta that withholds them would serve a stale meter. One row
per company is a few dozen bytes.

No route, no `access_list.yml`, no `api_routes.generated.go` change — the collection rides the
existing `GET.company-credit-usage-report`.

Tests: extend `backend/config/company_credit_usage_test.go` / `company_credit_budget_test.go` for
the stale-period zeroing, the `UsageUpdated == 0` fallback, and `Budgets` being present with a
warm watermark.

## Step 4 — `genix-ui` submodule: `hideXAxisLabels`

`packages/genix-ui/charts/ChartCanvas.svelte` — `dateLabels` feeds both the axis labels
(line 345) and the hover tooltip (line 410), so dropping the prop would silence the tooltip too.
Add `hideXAxisLabels?: boolean` (default `false`); `xAxisLabels` returns `[]` when it is set. The
container height already keys off `xAxisLabels.length` (line 591), so the card reclaims 18px for free.

Separate commit in the submodule + a submodule bump commit in this repo, as with the vTable change.

## Step 5 — Frontend

`frontend/routes/system/companies/company-credit-usage.model.ts`
- `ICompanyCreditBudgetMeter` mirroring the Go struct.
- `creditMeterFill(remaining, total)` — percent for the bar, reusing the `usagePercent` guard
  (`total <= 0 ⇒ 0`).

`company-credit-usage.svelte.ts` — `CompanyCreditReportService` gains
`budgetMeters: ICompanyCreditBudgetMeter[] = $state([])` from `response.Budgets`, plus the existing
`console.debug` style.

`CompanyCards.svelte` — `budgetMeterByCompanyID = $derived(new Map(...))`, passed into the card.

`CompanyCreditMeters.svelte` (new, ~40 lines) — the two labelled boxes. Structure and classes reuse
`CompanyRouteCreditCards.svelte:20-36` verbatim (rounded `border-slate-300` box, `relative` track,
absolute fill, `absolute right-4 ff-mono` figure), so the meters and the API bars read as one family:

```
DAILY                          CREDITS
┌───────────┬───────────┐      ┌───────────┬───────────┐
│▓▓▓  10000 │▓▓▓▓  2000 │      │▓▓  40000  │▓▓▓▓ 5000  │
└───────────┴───────────┘      └───────────┴───────────┘
 CPU (green)  AI (purple)
```

- Two columns per box: `grid grid-cols-2`. Fill width = remaining / limit.
- `IsCurrentMonth === false` ⇒ both boxes render `0` on an amber track with an
  `icon-[fa--exclamation-triangle]` + `T text="No budget|Sin presupuesto"` label, mirroring the
  warning `CompanyCreditBudget.svelte` already shows in the modal. Bars stay in place so card
  heights stay uniform across the grid.
- `title` on each cell spells the arithmetic out (`10000 of 12000 CPU cr. left today`), bilingual
  through `tr()`.

`CompanyCreditCard.svelte`
- Chart: add `hideXAxisLabels={true}`, drop `dateLabelEvery` and the `ui.state.deviceType` read it
  was the only consumer of (`useUI` import goes with it if nothing else uses it). Keep `dateLabels`
  and `dateLabelFormatter` — the tooltip still names the day.
- Render `<CompanyCreditMeters …/>` under the chart inside the existing
  `border-t border-slate-200` block. The card keeps `min-h-200`: the 18px the axis gave back plus
  the existing slack absorb most of the meters' ~46px; verify the grid at `lg` and `2xl`.

Tests: `company-credit-usage.model.test.ts` gains `creditMeterFill` cases (zero ceiling, remaining
above ceiling, exhausted).

## Step 6 — Docs

- `frontend/routes/system/companies/DOCUMENTATION.md` — "Review and find companies" gains the two
  meters and what they mean; "Manage a company credit budget" gains the daily-remaining figures and
  the ≤15s flush lag; refresh the `FILES` hashes (skill `document-user-routes`).
- `server_utils/README.md` + `src/limiter/mod.rs` header — the flush now also writes budget counters.
- `backend/docs/` needs nothing: no new route.

## Verification

1. `cd scripts && go run . check_tables`
2. `cd server_utils && cargo test -p genix-server-utils limiter::`
3. `cd backend && go test ./config/...`
4. `cd frontend && npx vitest run routes/system/companies`
5. Deploy tables, restart the daemon, spend a credit, and confirm within 15s that
   `company_credit_budget` carries non-zero `month_cpu_used` (skill `database-records`).
6. Skill `agent-browser`: open `SYS > Empresas`, screenshot a card — no x labels, both meters
   present, and a company with no budget shows the amber state.

## Open risks

- **15s staleness.** A card can overstate remaining credits by up to one flush interval. Acceptable
  for a monitoring panel; called out in DOCUMENTATION.md.
- **Full read of `company_credit_budget`.** One row per company with no partition filter. Fine at
  the current tenant count and consistent with how companies themselves are read; if the tenant
  count ever grows past a few thousand, this read is the thing to bucket.
- **The UTC-vs-Lima day boundary** (Step 2). Left as-is, the DAILY meter resets at 19:00 Lima and
  will look wrong to a user. Needs your call before I touch enforcement.
