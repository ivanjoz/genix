# Refactor plan — company credit usage (system/companies)

Status: **implemented**. Pre-alpha: no backwards compatibility, old routes deleted outright.

Two corrections the work forced on the plan as written:
- §3 originally proposed multi-column views. The ORM compiles those to hash views over a computed
  `zz_` column, which the Rust writer never fills. Single-real-column views were used instead.
- §5 originally derived the company list from the usage rows. That would have dropped a company
  that has never spent a credit; the catalog drives the list and usage is joined onto it.

Still to run against a live database: `./deploy.sh` to create the two materialized views.

## 1. What is wrong today

`backend/config/company_credit_usage.go` is a report generator living in the backend. Three
handlers do presentation work that belongs to the client, and all three pay for it in queries.

| Handler | Queries per request | Why |
|---|---|---|
| `company-credit-usage-report` | `1 + 2N` (N = companies) | full company catalog scan, then per company: one admin lookup **and** one 30-day range read |
| `company-credit-usage-users` | `2 + U` (U = users) | company validation, user catalog, then one 30-day range read per user |
| `company-credit-usage-detail` | `2` | company validation it does not need, then the day row |

Every request rebuilds everything from scratch — there is no watermark, so opening the panel twice
costs twice, and the report grows linearly with the number of tenants.

On top of the query cost, the backend computes what the frontend already can:

- `getCompanyCreditAdministrator` + `makeCompanyCreditAdministratorIdentity` (`:141-169`) — one
  extra query per company to produce a display label.
- `normalizeCompanyCreditCatalog` (`:310-330`) — dedup/sort of a catalog the frontend already holds
  through `EmpresasService`.
- `makeCompanyCreditUsageUser` (`:384-404`) — `FirstName + " " + LastName` fallback chains.
- `makeCompanyCreditUsageSummary`, `sortCompanyCreditUsageSummaries`, `makeCompanyCreditUsageDays` —
  zero-filling, totals, `TodayCPU`, `ActiveDays`, ranking.

## 2. Principle

The backend sends rows. The frontend joins, aggregates, zero-fills, names, ranks and filters.
Caching is the delta cache, not a hand-rolled `Map` in the component.

The enabling observation (from the user): **`credit_usage.TimeFrame` is a usable `updated`
column.** The daily row is rewritten in place all day, so a `TimeFrame >= watermark` read returns
today's row refreshed plus any newer day — exactly delta semantics, with `>=` instead of `>`
because today's frame keeps mutating under a fixed key.

Reference shape to copy: `backend/config/observability.go` — a multi-collection delta response
(`Frames`, `Frames_IDsToRemove`, version-gated `Routes` catalog) consumed by a `GetHandler` with
`keyID = 'ID'` (`frontend/routes/system/observability/observability.svelte.ts`).

## 3. Schema — two materialized views, no column changes

`backend/core/types/credit_usage.go`. Base PK stays `((company_id), user_id, time_frame)`, so
`server_utils/src/limiter/storage.rs` is **not** touched: `db.TypeView` compiles to a real Scylla
`CREATE MATERIALIZED VIEW` (`backend/genix-orm/scylla/index_view_compile.go:859`), maintained by
the database, so the Rust writer populates both views for free.

```go
Indexes: []db.Index{
    // ((company_id), time_frame, user_id) — "every user of this company over a day range"
    // is one clustering range instead of one query per user.
    {Type: db.TypeView, Keys: db.Cols(e.TimeFrame)},
    // ((time_frame), user_id, company_id) — one partition per frame holds every company's
    // aggregate row, so ranking the platform costs days, not tenants.
    {Type: db.TypeView, Keys: db.Cols(e.UserID), Partition: e.TimeFrame},
},
```

**Each view keys on exactly one real column.** This is not cosmetic. A spike against the ORM's own
select-planner harness showed that a *multi-column* `TypeView` compiles to a **hash view** over an
ORM-computed `zz_<cols>` virtual column:

```
PRIMARY KEY ((company_id), zz_pk_time_frame_user_id, user_id, time_frame)
capability: company_id|=|time_frame|=|user_id|=   (equality only)
```

That breaks this table twice over: the `zz_` column is filled by the ORM's write path, which the
Rust limiter does not run, and a hash view answers equality only — the day *range* the report needs
would still fall back to the base table. Single-real-column views avoid both, and are the same shape
`user_logs` uses for the other externally-written table.

Verified plans (ORM planner, no database needed):

| Query | Route |
|---|---|
| `CompanyID = 7 AND TimeFrame BETWEEN a,b` | `..__pk_time_frame_view WHERE company_id = ? AND time_frame >= ? AND time_frame <= ?` |
| `UserID = -1 AND TimeFrame IN (30 frames)` | `..__time_frame_user_id_view WHERE user_id = ? AND time_frame IN (?...)`, one statement |

Still to confirm at deploy time: `./deploy.sh` → static validation
(`static-project-validation` skill) and the actual `CREATE MATERIALIZED VIEW` against Scylla.

## 4. Backend — three handlers become two, plus one by-IDs endpoint

### 4.1 `GET.company-credit-usage-report` — cross-company, delta

```go
// One (company, day) aggregate row. TimeFrame doubles as the watermark: the row is rewritten in
// place all day, so a >= bound refreshes today and appends new days.
type companyCreditDay struct {
    ID        int64  // CompanyID*100000 + Day — the delta-cache record key
    CompanyID int32
    Day       int16
    CPU       uint64
    Inference uint64
    Updated   int32 `json:"upd"` // == TimeFrame
    Status    int8  `json:"ss"`
}

type creditRouteName struct {
    ID      int16
    Route   string
    Updated int32 `json:"upd"`
    Status  int8  `json:"ss"`
}

type companyCreditReport struct {
    Days   []companyCreditDay
    Routes []creditRouteName `json:",omitempty"` // version-gated catalog, observability pattern
}
```

Body:
- `watermark := core.Coalesce(req.GetQueryInt("Days"), req.GetQueryInt("upd"))`
- `lastFrame := dailyTimeFramePrefix + core.FechaUnix()`; `firstFrame := max(lastFrame-29, watermark)`
- one read over the day-partitioned view: `UserID.Equals(companyAggregateID)` +
  `TimeFrame.In(firstFrame..lastFrame)`
- `decodeCreditUsage(row.UsedCredits, nil)` for totals only; skip `CompanyID <= 0` (that is the
  observability platform aggregate, not a tenant)
- emit `Routes` only when `req.GetQueryInt("Routes") < companyCreditRoutesVersion`

Steady state: **1 query, 1 day partition**, independent of tenant count. Cold start: 30 partitions.

**Deleted:** `companyCreditUsageSummary`, `companyCreditUsageReport`, the `errgroup` fan-out,
`getCompaniesUpdatedSince(0)` call, `normalizeCompanyCreditCatalog`,
`getCompanyCreditAdministrator`, `makeCompanyCreditAdministratorIdentity`,
`makeCompanyCreditUsageSummary`, `makeCompanyCreditUsageDays`, `sortCompanyCreditUsageSummaries`.

### 4.2 `GET.company-credit-usage` — one company, replaces `-detail` **and** `-users`

```go
type companyCreditUserDay struct {
    ID        int64  // UserID*100000 + Day
    UserID    int32
    Day       int16
    CPU       uint64
    Inference uint64
    // Only on the company-aggregate row (UserID == -1): 30 days x ~40 routes is a payload the
    // client can hold; 30 days x U users x 40 routes is not.
    Routes    []creditUsageRoute `json:",omitempty"`
    Updated   int32 `json:"upd"`
    Status    int8  `json:"ss"`
}
```

One read over the per-company view:
`CompanyID.Equals(targetCompanyID).TimeFrame.Between(firstFrame, lastFrame)`, with the same
`>= watermark` narrowing. Blob decoded with `routeTotals` only for `UserID == companyAggregateID`.

No `getCompanyByID` validation: an unknown company returns no rows, which is the same answer for
one fewer query. No user catalog read and no display-name construction.

**Deleted:** `GetCompanyCreditUsageDetail`, `GetCompanyCreditUsageUsers`, `getCompanyCreditUsers`,
`makeCompanyCreditUsageDetail`, `makeCompanyCreditUsageUser`, `sortCompanyCreditUsageUsers`,
`validateCompanyCreditUsageDay`, `companyCreditUsageDetail`, `companyCreditUsageUser`,
`companyCreditUsageUsersReport`, `companyCreditQueryLimit`, `companyCreditUnknownAPI`.

### 4.3 `GET.company-users-by-ids` — new, static by-IDs, cross-company

The versioned by-IDs path cannot serve this panel: `mainHandler` strips a client-sent `cmp` on
private routes, so `ExtractUpdatedVersionValues` always resolves the partition from the token
(`backend/config/generic_records.go:11-13`). A SaaS operator reading tenant #7's users needs a
route that carries the company in the ID itself.

Key packing: `ID = CompanyID*10000 + UserID` (10k users per company; fits `int32` up to company
214,748, so the client's `concatenateInts` u32 bucket holds it).

```go
type companyUserLabel struct {
    ID        int32 // packed CompanyID*10000 + UserID
    User      string
    FirstName string
    LastName  string
}
```

`req.ExtractIDs()` → group by company → one
`query.Select(ID, User, FirstName, LastName).CompanyID.Equals(c).ID.In(userIDs...)` per distinct
company, in an `errgroup`. Precedent: `GetPublicCompanyNamesByIDs`
(`backend/config/empresas.go:89`) and `GetProductStockLotsByIDs`.

This single endpoint replaces **both** `getCompanyCreditAdministrator` (ask for
`companyID*10000 + 1`) and the user-name half of the users tab. Cache is static — a display label
in an operator panel does not need revalidation, and `getStaticRecordsByID` only asks the server
for IDs missing from memory *and* IndexedDB.

### 4.4 Registration and access

- `backend/config/main.go`: drop `GET.company-credit-usage-detail` / `-users`, add
  `GET.company-credit-usage` and `GET.company-users-by-ids`.
- `backend/access_list.yml` id 5 `backend_apis`: →
  `POST.company,GET.company-credit-usage-report,GET.company-credit-usage,GET.company-users-by-ids,GET.company-credit-budget,POST.company-credit-budget`.
- Regenerate: `./deploy.sh` → `generate_route_ids` + `generate_controllers`. Route IDs are
  assign-once and retired IDs are never reused (`backend/core/api_routes.generated.go:7-9`), so
  historical credit blobs keep meaning 107/109 = the deleted routes.

## 5. Frontend — where the deleted work lands

### `company-credit-usage.ts`
Two `GetHandler` services replacing the three bare `GET` calls:

- `CompanyCreditReportService` — `route = 'company-credit-usage-report'`, `keyID = 'ID'`,
  `useCache = { min: 0.2, ver: 1 }`. Holds `days: ICompanyCreditDay[]` and
  `routeNames: Map<number, string>` (merged like `mergeObservabilityRoutes`).
- `CompanyCreditDetailService` — `route = 'company-credit-usage?target-company-id=' + id`,
  `keyID = 'ID'`. Delta snapshots are keyed by route+query string, so each company gets its own
  independent collection and watermark for free.

### `company-credit-usage.model.ts` — gains what the backend lost
- `buildCompanyCreditSummaries(days, companies)` — group by company, 30-day zero-filled series,
  `CPU`/`Inference` totals, `TodayCPU`/`TodayInference`, `ActiveDays`; name and status joined from
  `EmpresasService`, with the `Company #N` fallback moving here.
- `buildCompanyUserSummaries(rows)` — same per user; display name from the resolved user label with
  the `User` → `User #N` fallback chain.
- `pickCompanyDayRoutes(rows, day, routeNames)` — the selected day's routes, sorted by CPU desc,
  `API.UNKNOWN` fallback here.
- Kept: `rankCompanyCreditUsage`, `usagePercent`, `splitCompanyCreditRoute`.
- The three `normalize*` functions collapse into one numeric guard applied at the service boundary.

### Components
- `CompanyCards.svelte` — drops `detailMemo`, `usersMemo`, `refreshReport`'s manual cache clearing
  and the `getCompanyCreditUsage*` imports; the delta cache owns caching. Resolves admin labels in
  bulk for the ranked list via
  `getStaticRecordsByID('company-users-by-ids', ranked.map(c => c.CompanyID*10000 + 1))` and passes
  the label down.
- `CompanyCreditCard.svelte` — takes `adminName` as a prop instead of reading `company.AdminName`.
- `CompanyUserCreditCards.svelte` — same lookup for user names, packed with the open company's ID.
- `CompanyCreditCalendar.svelte`, `CompanyRouteCreditCards.svelte` — unchanged (they already take
  plain arrays).

### Tests
- `company-credit-usage.model.test.ts` — extend for the new builders (grouping, zero-fill, today,
  active days, ranking ties).
- `backend/config/company_credit_usage_test.go` — rewrite against the two handlers: watermark
  narrowing, `>=` re-sending today, `CompanyID <= 0` exclusion, route catalog version gate, routes
  present only on the aggregate row.

### Docs
`frontend/routes/system/companies/DOCUMENTATION.md` — refresh via the `document-user-routes` skill
once the routes settle.

## 6. Net effect

| | before | after (steady state) |
|---|---|---|
| report | `1 + 2N` queries, full payload every time | 1 query, 1 day partition, delta payload |
| company drill-down | `2` + `2 + U` queries across two routes | 1 query, one route, delta payload |
| user / admin labels | `N + U` queries inside the reports | 0 (IndexedDB), server only on first sight |
| `company_credit_usage.go` | 404 lines | ~120 lines expected |

## 7. Open questions

1. **Admin label freshness.** `getStaticRecordsByID` never revalidates, so renaming a user would
   not update the operator panel until `clearCacheByIDs()`. Acceptable, or should
   `company-users-by-ids` be a versioned endpoint with a `cmp`-carrying variant instead?
2. ~~**Materialized view cost.**~~ Resolved: the limiter flushes dirty rows only, every 15 s
   (`server_utils/src/limiter/mod.rs:9`), so an active `(company, user, day)` row costs 4 writes per
   minute — 12 with both views. Even a few hundred simultaneously active rows stays under ~50
   writes/s. Not a concern.
3. **Window length.** The 30-day window is currently a backend constant. After the refactor the
   frontend decides what it renders; should the backend still cap the query at 30 days, or accept a
   `days` parameter?


## 8. Executed

| Area | Change |
|---|---|
| `backend/core/types/credit_usage.go` | two single-column materialized views; no column or key change, Rust writer untouched |
| `backend/config/company_credit_usage.go` | 404 -> 318 lines; three handlers -> two, plus the cross-tenant label endpoint |
| `backend/config/main.go`, `access_list.yml` | `-detail`/`-users` replaced by `company-credit-usage` + `company-users-by-ids` |
| `backend/core/api_routes.generated.go` | 107/109 retired, 112/113 assigned |
| `company-credit-usage.model.ts` | owns every type, constant and aggregation; no runtime imports, so it is unit-testable |
| `company-credit-usage.ts` | two `GetHandler` services only |
| `CompanyCards.svelte` | hand-rolled memo maps gone; delta cache and the by-IDs cache own caching |
| tests | 8 Go tests, 19 frontend tests |

Verification run: `go build ./...`, `go test ./config/`, `./app.sh check_tables`,
`bun test routes/system/companies/`, `npm run check` (12 errors, all pre-existing and in unrelated
modules). `app/agent/ragdocs` fails on a stale `finance/cash-banks` provenance hash that predates
this work.


## 9. Follow-up: credit_usage split into two tables

The single table with a `user_id = -1` sentinel was replaced by `credit_usage_company` (ID 50) and
`credit_usage_user` (ID 51); ID 42 is retired. The sentinel survives only inside the Rust limiter's
in-memory map, where it is a real discriminator (one charge updates the user and the company), and
`UsageKey::is_company_aggregate` routes a flush to the right table.

Why, beyond ergonomics: a Scylla MV's `WHERE` can only be `IS NOT NULL`, so the day-partitioned view
could not be restricted to the aggregate rows it existed to serve. It duplicated every user row of
every company into one partition per day.

| | one table + 2 views | two tables + 1 view each |
|---|---|---|
| stored copies per company-day | `3 x (U+1)` | `2 x (U+1)` |
| rows in one day-partition of the report view | `C x (U+1)` | `C` |
| company drill-down series | view read | native key read |

Verified plans (ORM planner, no database):

| Query | Route |
|---|---|
| `CompanyID = 7 AND TimeFrame BETWEEN a,b` on the company table | base table, no view |
| `TimeFrame IN (30 frames)` | `credit_usage_company__time_frame_company_id_view`, one statement |
| `CompanyID = 7 AND TimeFrame BETWEEN a,b` on the user table | `credit_usage_user__pk_time_frame_view` |
| `CompanyID = 7 AND UserID = 42 AND TimeFrame BETWEEN a,b` (the limiter's own read) | base table |

Touched beyond §8: `server_utils` (`aggregation.rs` owns the sentinel, `storage.rs` picks the table
from the key), `observability.go` and `observability_backfill.go` (platform aggregate is a company
row under company id 0), `credit_usage.go`, `company_credit_budget.go`, ORM docs §13 (regenerated;
it had drifted and still claimed 42 was free), and the `credit_usage` prose references across Go,
Rust and the configure scripts.

The drill-down response is now two collections, `Company` and `Users`, each with its own delta
watermark. Existing `credit_usage` rows are dropped, per decision: the daemon repopulates and the
charts rebuild over 30 days.

Verification: `cargo test` (140 passed), `go build ./...`, `go test ./config/ ./exec/ ./core/...`,
`./app.sh check_tables` (49 tables), `bun test routes/system/companies/` (19), `npm run check`.


## 10. Follow-up: single-use helpers inlined

`creditUsageFrameRange`, `makeCompanyCreditDays`, `makeCompanyCreditCompanyDays`,
`makeCompanyCreditUserDays` and `groupCompanyUserLabelIDs` each had exactly one caller and are now
inline. `creditUsageWatermark` and `makeCreditRouteNames` have two callers each and stayed.
The file is three handlers plus those two helpers.

Test coverage moved rather than vanished for most of it:
- blob decoding and every rejection path: already covered by `TestDecodeCreditUsage*` in
  `credit_usage_test.go`, which the inlined loops call directly.
- "routes only on company rows": no longer a rule that can be broken. The two-table split means
  `companyCreditUserDay` has no Routes field and `credit_usage_user` has no route data to put in
  one, so the type system enforces what the test used to assert.

Genuinely lost their unit test, and now need a live database to exercise:
- the platform aggregate (company id 0) being excluded from the company ranking;
- the packed `(company, day)`, `(user, day)` and `(company, user)` id arithmetic;
- the by-IDs grouping rejecting company id 0 and user id 0.


## 11. Follow-up: the day boundary (pre-existing bug)

The report came back empty against a database that had data. Cause: **the writer and the reader
disagreed about when a day starts.**

- Rust (`server_utils/src/limiter/time_frame.rs`) bucketed `daily()` as `unix_seconds / 86_400` —
  a UTC day.
- Go read the window from `core.FechaUnix()`, which is `(unix + hostZoneOffset) / 86_400`.

On a host set to Lima (UTC-5) the two agree for nineteen hours and disagree for the last five: from
19:00 local the daemon writes frame N+1 while the backend queries up to N, so today's row falls
outside the window and every report renders flat. Reproduced live at 23:5x local: the handler
computed `first=200020654 last=200020683` while the only row was `200020684`.

This predates the refactor — the original handler used `core.FechaUnix()` too — but the split made
it visible because there is now no other row to fall back on. `credit_usage.go` had quietly avoided
it by computing its own window in explicit UTC, which is why the per-user dashboard worked.

Resolved by pinning **both sides to the Lima business day**, which is also what the project's
UnixDay convention means everywhere else:

- `DAY_ZONE_OFFSET_SECONDS` in `time_frame.rs` shifts `daily()` and `month_start_day()`.
- `creditDayZoneOffsetSeconds` in `config/credit_usage.go` backs `currentDailyTimeFrame()`, now the
  single source for every daily window (report, drill-down, per-user dashboard, budget
  month-to-date, which had the same bug on its upper bound).
- `currentUTCMonthStartDay` became `currentMonthStartDay`, reading the calendar in the same day the
  daemon buckets by — it is compared against the daemon's own `month_start_day`.

A fixed offset, not a zone lookup: Peru has no DST, and either process can run on a host in any
timezone, including a Lambda that is always UTC. Both constants carry a comment naming the other.

Covered by `TestCurrentDailyTimeFrameFollowsTheLimaDayNotTheHostZone` (five boundary cases through
a frozen clock), `TestCurrentMonthStartDayIsReadInTheLimaDay`, and
`the_evening_stays_on_the_local_day` on the Rust side.

**Operational note:** the running credit daemon must be restarted to emit frames on the new
boundary. Until then it keeps writing the old UTC frame and the report stays empty.
