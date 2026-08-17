# Company Credit Usage Report Plan

Status: implemented — this is a separate report tab inside the existing Observability page.

## 1. Goal

Add a SaaS-only **Company Credits** report inside `/system/observability` that:

- lists every platform company from highest to lowest credit usage over the last 30 UTC days;
- keeps CPU and inference credits separate because they are independent quota pools;
- shows the company name, status, 30-day totals, today's totals, active-day count, and rank;
- opens a read-only side layer when a company row is selected;
- shows that company's 30-day daily history in the layer;
- lets the operator select a day and then loads the per-API credit breakdown for only that company and day.

This is a historical usage/billing report. Its component, state, and APIs remain separate from the **Backend Services** tab, whose purpose is short-window API health and errors across the platform.

## 2. Existing Data and Constraints

### `credit_usage`

The existing table already contains the required source data:

| Key | Meaning |
|---|---|
| `company_id` | Scylla partition key |
| `user_id = -1` | aggregate for the whole company |
| `time_frame = 200_000_000 + UnixDay` | one absolute daily record |
| `used_credits` | compact per-route CPU and inference totals |

`server_utils` writes both five-minute and daily records for every company and user. The company daily row is authoritative for this report; summing five-minute records would add unnecessary reads and would reproduce data already persisted by the limiter.

The reserved platform record `(company_id = 0, user_id = -1)` is not suitable: it exists only in five-minute frames and intentionally removes company identity.

### Query consequence

Scylla cannot retrieve all companies from `credit_usage` with one efficient query because `company_id` is the partition key. The first implementation should therefore:

1. load the platform company catalog;
2. issue one partition-local range query per company for `user_id = -1` and the 30 daily frames;
3. limit concurrency to eight queries;
4. aggregate and sort the small results in Go.

Each company query returns at most 30 rows. This is bounded fan-out and uses valid partition-key queries without `ALLOW FILTERING`.

Do not add another table initially. A report index becomes justified only if measured company count or endpoint latency makes the bounded fan-out unacceptable. A future index would be partitioned by Unix day with company as a clustering key; it should not be introduced without performance evidence because it would require another write path in `server_utils`.

## 3. Report Semantics

- Window: today plus the previous 29 UTC days.
- Persisted/report dates: UnixDay `int16`.
- Current time: derive from the effective project clock (`core.Now()`/project time helpers), not a new direct persisted-time path.
- Companies: include every catalog company with `ID > 0`, including inactive companies, so recent historical usage is not hidden. Return status and place zero-usage companies at the bottom.
- Default ranking: total CPU credits descending, then total inference credits descending, then company ID ascending for deterministic ties.
- Optional frontend ranking toggle: CPU or inference. This re-sorts the already loaded response; it does not issue another request.
- Never calculate a synthetic `CPU + inference` total. The values belong to separate quota pools and combining them would create a misleading ranking.
- Daily gaps: return all 30 days and zero-fill missing daily rows.
- Route order: CPU descending, then inference descending, then route ID ascending, matching the existing credit-usage breakdown.
- Route labels: resolve from `core.APIRouteNames`; use `API UNKNOWN` with the numeric route ID when metadata is absent.

## 4. Backend APIs

### 4.1 Summary endpoint

Route: `GET.company-credit-usage-report`

No date parameters are needed in version one; the report is intentionally fixed at 30 days.

Response fields:

| Object | Fields |
|---|---|
| response | `FirstDay int16`, `LastDay int16`, `GeneratedAt int32`, `Companies []CompanySummary` |
| company summary | `CompanyID int32`, `Company string`, `Status int8`, `CPU uint64`, `Inference uint64`, `TodayCPU uint64`, `TodayInference uint64`, `ActiveDays int16`, `Days []UsageDay` |
| usage day | `Day int16`, `CPU uint64`, `Inference uint64` |

Handler algorithm:

1. Load the company catalog through a shared helper extracted from `GetEmpresas`, preserving both cloud-mirror and self-hosted behavior.
2. Remove invalid/reserved IDs and de-duplicate by company ID.
3. Preallocate one result slot per company.
4. Use `errgroup.Group.SetLimit(8)` to query each `(company_id, user_id = -1)` partition between the first and last daily frames.
5. Decode each row with the existing strict `decodeCreditUsage` function, zero-fill 30 days, and calculate summary totals.
6. Fail the whole response on a corrupt blob and log the company ID and time frame; silently showing partial billing data would be unsafe.
7. Sort deterministically and return one response.

Add structured debug logs for start, company count, first/last day, rows read, duration/completion, and failures. Do not log the credit blobs.

### 4.2 Selected-day detail endpoint

Route: `GET.company-credit-usage-detail?target-company-id=<ID>&day=<UnixDay>`

Response fields:

| Object | Fields |
|---|---|
| response | `CompanyID int32`, `Day int16`, `CPU uint64`, `Inference uint64`, `Routes []RouteUsage` |
| route usage | `RouteID int16`, `Route string`, `CPU uint64`, `Inference uint64` |

Validation:

- `company-id` must be positive and must exist in the company catalog;
- `day` must be inside the same last-30-day window and cannot be in the future;
- the SaaS-only and access-catalog checks must run before the handler;
- a missing daily row returns a valid zero-total response with an empty route list;
- malformed stored data returns a descriptive Spanish error and an internal log with company/frame context.

This endpoint performs one exact partition/range read for one daily frame and decodes the route map with the existing `makeCreditUsageRoutes` ordering and names.

## 5. Authorization and Registration

Both endpoints expose cross-company data and must be platform-only.

- Add both routes to `backend/config/main.go`.
- Add both routes to `saasOnlyRoutes` in `backend/main-handlers.go`.
- Add both backend APIs to the existing group 8 access item for `system/observability`.
- Keep the existing **System → Observability** sidebar entry; do not add another route or menu item.
- Regenerate the API route catalog instead of manually assigning route IDs.

The report must not reuse the tenant-scoped `GET.credit-usage` permission. That existing endpoint serves the authenticated user's own company and has a different security boundary.

## 6. Frontend Structure

Suggested files:

- `frontend/routes/system/observability/+page.svelte` for the two-option `OptionsStrip` shell
- `frontend/routes/system/observability/BackendServices.svelte`
- `frontend/routes/system/observability/CompanyCredits.svelte`
- `frontend/routes/system/observability/company-credit-usage.ts`
- `frontend/routes/system/observability/company-credit-usage.model.ts`
- `frontend/routes/system/observability/company-credit-usage.model.test.ts`
- `frontend/routes/system/observability/DOCUMENTATION.md`

Use a report-style functional `GET`, not `GetHandler`:

- the dataset is historical, filtered, and derived rather than master data;
- today's absolute row changes in place and has no ORM update watermark;
- page memory is sufficient for the summary;
- load once on page entry and expose a manual Refresh action;
- memoize detail responses by `(companyID, day)` only for the lifetime of the mounted page;
- explicit Refresh clears the detail memo and refetches the summary.

Do not add a 15-second observability polling loop. This report does not need real-time behavior.

## 7. Page Layout

Use the project components so the page remains accessible to the browser agent.

### Company Credits tab

- The existing `Page` keeps the `Observability|Observabilidad` title and adds `Backend Services` / `Company Credits` through `OptionsStrip`.
- Toolbar:
  - `FilterInput` for company name or ID;
  - CPU/inference ranking selector;
  - fixed `Last 30 days|Últimos 30 días` badge;
  - generated/updated timestamp;
  - `Button` Refresh action.
- Summary cards above the table: platform CPU total, inference total, companies with usage, and companies without usage.
- `Layer type="content"` around a `VTable` so the table shrinks when the side layer opens.
- Table columns:
  - rank;
  - company ID and name;
  - status;
  - 30-day CPU;
  - 30-day inference;
  - today CPU;
  - today inference;
  - active days.
- `onRowClick` selects the company and opens side layer ID `1`.
- Loading, empty, no-match, and request-error states must be explicit.
- All page text must remain at least 14px; chart axes may use the established smaller chart-axis size.

### Read-only company layer

Use `Layer type="side" id={1}` with approximately `760px` desktop width, Close only, and internal views:

1. `By day|Por día`
   - company name and 30-day totals;
   - separate CPU and inference charts so independent units are not visually stacked into one total;
   - a daily `VTable` with date, CPU, inference, and share of the company's 30-day usage;
   - row click selects that day, switches to the API view, and loads detail.
2. `APIs for day|APIs del día`
   - selected date and day totals;
   - route table with method, API path, route ID, CPU, inference, and each route's percentage of the selected day's corresponding pool;
   - safe zero-denominator formatting so the UI never renders `NaN`;
   - a clear empty state when the day has no usage.

Opening a company should use the 30 daily values already returned by the summary endpoint. Only selecting a day calls the detail endpoint.

Add concise frontend debug logs for company-layer open, detail request start/completion, cache hit, and error. Do not log complete response payloads.

## 8. Backend Implementation Files

Expected changes:

- add `backend/config/company_credit_usage.go` for response types, handlers, aggregation, validation, and logging;
- refactor the company-list read in `backend/config/empresas.go` into a small reusable helper that preserves the cloud/self-hosted choice;
- reuse `decodeCreditUsage` and `makeCreditUsageRoutes` from `backend/config/credit_usage.go`; avoid a second blob decoder;
- register handlers in `backend/config/main.go`;
- protect routes in `backend/main-handlers.go` and `backend/access_list.yml`;
- regenerate `backend/core/api_routes.generated.go` with the project generator;
- add focused tests in `backend/config/company_credit_usage_test.go`.

No schema or `server_utils` writer change is required for version one.

## 9. Tests and Verification

### Backend tests

- returns exactly 30 zero-filled days per company;
- sums CPU and inference without overflow or cross-company mixing;
- ranks companies deterministically;
- includes inactive and zero-usage companies at the bottom;
- rejects invalid company IDs and days outside the window;
- returns an empty detail for a valid company/day with no row;
- ranks API routes and resolves route names;
- reports corrupt blobs with company and frame context;
- confirms every `credit_usage` query supplies `company_id`, `user_id`, and a bounded `time_frame` range.

### Frontend tests

- CPU and inference ranking modes;
- company name/ID filtering;
- active-day and total calculations are rendered from backend data without recomputation drift;
- company click opens the layer using existing summary days;
- day click requests the correct company/day detail;
- detail memoization and Refresh invalidation;
- zero totals produce `0%`, never `NaN` or `Infinity`;
- unknown route IDs render a readable fallback.

### Final validation

- run targeted Go and Bun tests;
- run backend static checks if any table structs are touched (none are expected);
- run the production frontend build;
- use `agent_browser` as company 1/user 1 to verify tab visibility, ordering, filtering, layer navigation, empty/error states, 14px minimum text, and mobile-width behavior;
- verify a non-SaaS company cannot call either endpoint or navigate to the page.

## 10. Acceptance Criteria

- A separate Company Credits report tab exists inside Observability without coupling to Backend Services polling/state.
- Company rows cover the last 30 UTC days and are ordered from highest to lowest usage under the selected credit pool.
- All companies remain identifiable by name and ID, including zero-usage and inactive companies.
- Clicking a company opens a read-only side layer without refetching the 30-day summary.
- Clicking a day loads only that day's API detail for that company.
- CPU and inference values remain separate end to end.
- Queries never require `ALLOW FILTERING` on `credit_usage`.
- Cross-company endpoints are restricted to the SaaS owner and the existing Observability access-catalog item.
- No new database table, delta cache, or `server_utils` persistence path is introduced in version one.
