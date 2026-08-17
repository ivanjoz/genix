# Companies Credit Cards Migration Plan

Status: proposed

## 1. Goal

Move the platform-wide company credit report out of `/system/observability` and integrate it into
the existing `/system/companies` page.

The Companies page will stop using a table as its primary view. It will render one card per company,
ordered from highest to lowest 30-day credit usage. Each card will show:

- company name and numeric ID;
- the company's canonical administrator display name and login;
- 30 days of CPU credits in green;
- 30 days of inference credits in purple;
- an edit button that appears on hover on pointer devices and remains accessible on keyboard,
  focus, and touch devices.

Clicking the card body will preserve the implemented read-only drill-down: company totals and daily
history, followed by per-API detail for a selected day. Clicking the edit button will continue to
open the existing company edit modal and must not open the usage layer.

## 2. Scope and Non-Goals

### In scope

- Replace the Companies `VTable` with responsive company cards.
- Move the existing company-credit frontend service, model, tests, and drill-down UI from the
  Observability route into the Companies route.
- Add the administrator identity needed by each card to the existing report response.
- Move authorization for the two credit report APIs from the Observability access item to the
  Companies access item.
- Reduce Observability back to the Backend Services dashboard only.
- Keep the report platform-wide and SaaS-only.

### Not in scope

- No `credit_usage` schema or `server_utils` writer changes.
- No new database table, cache table, or delta protocol.
- No changes to the five-minute observability aggregation.
- No tenant-visible credit report.
- No attempt to infer administrators from profiles or access levels.

## 3. Administrator Semantics

The current user model does not have an `is_admin` field. The enforced backend invariant is that
user ID `1` in each company is the company administrator and its login is fixed to `admin`.

For this report, "administrator" therefore means `(company_id, user_id = 1)`.

The report should expose only a sanitized card projection:

| Field | Meaning |
|---|---|
| `AdminName` | `FirstName + LastName`, trimmed; fallback to the login when both are empty |
| `AdminUser` | login handle, normally `admin` |

Do not return password hashes, access lists, profile IDs, document numbers, or other user fields.
If a legacy company has no user ID `1`, render `Administrator unavailable|Administrador no disponible`
without failing the entire credit report.

## 4. Data Flow

The page combines two existing sources:

1. `EmpresasService` remains the editable company master-data source and continues using its delta
   cache.
2. `GET.company-credit-usage-report` remains a fresh, on-demand 30-day historical report.

The credit report remains a functional `GET`, not a `GetHandler`, because today's daily aggregate
changes in place and the response is derived historical data without a useful frontend delta
watermark.

The page loads both sources on entry and joins them by company ID. Credit data supplies ranking,
administrator projection, totals, and 30 daily points. Company master data supplies the complete
editable record used by the existing modal.

After a company is created, updated, or deleted, refresh the credit report after the company save
finishes so the card collection, name, status, and administrator projection cannot remain stale.
The explicit Refresh button also clears the per-company/day detail memo.

## 5. Backend Changes

### 5.1 Extend the existing summary response

Keep the route name `GET.company-credit-usage-report` and add these fields to each company summary:

- `AdminName string`
- `AdminUser string`

No new endpoint is necessary.

For each catalog company, resolve the administrator with an exact primary-key lookup:

- Scylla: `company_id = <company>` and `id = 1`;
- cloud mirror: the equivalent composite-ID lookup.

Run the administrator lookup inside the existing company-scoped bounded worker. Keep the overall
company concurrency limit at eight so the report does not create unbounded credit and user queries.
An absent administrator is a valid empty projection; a database failure is a report failure and must
log the company ID without logging user records.

Add concise debug logs for administrator lookup failures and report completion counts, including the
number of companies missing an administrator.

### 5.2 Preserve credit queries

Do not change the current efficient credit reads:

- one exact company partition at a time;
- `user_id = -1` for the company aggregate;
- daily frames for the fixed 30-day window;
- no `ALLOW FILTERING` on `credit_usage`;
- one exact company/day query for API drill-down.

The existing report and detail route names remain stable, so regenerating API route IDs should be a
verification step only and must produce no ID changes.

### 5.3 Authorization migration

In `backend/access_list.yml`:

- add `GET.company-credit-usage-report` and `GET.company-credit-usage-detail` to access item `5`
  (`Empresas` / `system/companies`);
- remove those APIs from access item `6`;
- rename item `6` back to Server Panel and Observability wording because it will no longer own the
  company-credit report.

Keep both endpoints in `saasOnlyRoutes`. Moving the UI and catalog permission must not weaken the
platform-owner boundary.

## 6. Frontend Structure

Use separate files so company CRUD, card presentation, report state, and drill-down remain focused:

- `frontend/routes/system/companies/+page.svelte`
  - page shell, filter/create toolbar, company service, save/delete logic, and edit modal;
- `frontend/routes/system/companies/CompanyCards.svelte`
  - report fetch, ranking mode, master/report join, responsive card grid, refresh, and usage layer;
- `frontend/routes/system/companies/CompanyCreditCard.svelte`
  - one accessible `Card`, administrator text, totals/legend, grouped 30-day chart, and edit button;
- `frontend/routes/system/companies/company-credit-usage.ts`
  - moved report/detail interfaces and functional `GET` calls;
- `frontend/routes/system/companies/company-credit-usage.model.ts`
  - moved and extended normalization/ranking helpers;
- `frontend/routes/system/companies/company-credit-usage.model.test.ts`
  - moved and extended model tests.

Move files rather than duplicate them. Remove the old Company Credits component/imports from
`frontend/routes/system/observability` after the Companies page owns the report.

## 7. Companies Page Layout

### Toolbar

Keep the existing company filter and green create button. Add the compact report controls previously
used by Company Credits:

- CPU / inference ranking selector;
- `Last 30 days|Últimos 30 días` badge;
- updated timestamp;
- Refresh button.

Default sorting is CPU credits descending. The selector changes the primary sort between CPU and
inference without making another request. The non-selected pool is the secondary tie-breaker, then
company ID ascending. Never sort by `CPU + inference`, because they are independent quota pools.

The filter matches company ID, company name, legal name, RUC, administrator display name, and
administrator login while preserving the company's global rank.

Do not carry the four platform summary tiles from the Observability report into this page; the
company cards are the primary information surface.

### Responsive card grid

Wrap the grid in `Layer type="content"` so it shrinks when the usage layer opens.

- desktop: three cards per row where space permits;
- medium widths: two cards per row;
- mobile: one card per row;
- stable card height so charts align across a row;
- explicit loading, request-error, empty-company, and no-filter-match states;
- all text at least 14px except the established chart-axis labels.

Use the project `Card` component with `onClick` so each card is keyboard-accessible and visible to
the browser agent. Use the project `Button` for editing; it already stops click propagation.

### Card content

Each card contains:

1. Header
   - company name and `#ID`;
   - administrator display name and `@login`;
   - edit pencil in the upper-right corner.
2. Usage summary
   - CPU 30-day total with a green marker;
   - inference 30-day total with a purple marker.
3. Thirty-day chart
   - 30 UnixDay labels from the summary response;
   - CPU grouped bar in green (`#10b981` family);
   - inference grouped bar in purple (`#a855f7` family);
   - shared numeric axis, but side-by-side bars so independent pools are compared rather than
     visually summed;
   - readable zero-usage baseline and no `NaN`/`Infinity` values.

`ChartCanvas` currently stacks multiple bar series. Add a small optional grouped-bar mode to the
shared component, defaulting to its current stacked behavior so existing charts do not change. The
new cards opt into grouped mode. Include the render mode in the shared chart cache key.

On pointer devices, the pencil may transition from hidden to visible on card hover. It must also be
visible on `:focus-within`, and always visible at touch/mobile breakpoints because hover does not
exist there.

## 8. Interactions and Layers

### Edit

- Pencil click copies the matching `ICompany` master record into `empresaForm`.
- Open the existing edit modal unchanged.
- The button click must not trigger the card's usage action.
- Create, validation, save, and delete behavior remain unchanged.

### Usage drill-down

- Card body click opens the existing read-only side layer.
- Reuse the 30 daily rows already present in the summary response; do not refetch them.
- Keep the layer's `By day|Por día` and `APIs for day|APIs del día` views.
- Selecting a day calls only
  `GET.company-credit-usage-detail?target-company-id=<ID>&day=<UnixDay>`.
- Keep page-lifetime memoization by `(companyID, day)` and clear it on Refresh.
- Preserve actual method/API labels, safe percentages, unknown-route fallback, and explicit loading,
  error, and empty states.

The card chart is a compact overview. The layer may retain separate larger CPU and inference charts
because they provide a clearer detailed comparison at full layer width.

## 9. Observability Cleanup

After the move, `/system/observability` returns to one purpose: backend service health and errors.

- Remove the `OptionsStrip` and Company Credits option from its page shell.
- Render `BackendServices.svelte` directly inside `Page`.
- Remove the old `CompanyCredits.svelte` after its reusable drill-down logic is relocated.
- Remove company-credit source references and capability text from the Observability documentation.
- Keep all current Backend Services behavior, polling, route metadata, errors, and font sizing intact.

Do not remove the Observability sidebar entry.

## 10. Documentation Changes

- Create or upgrade `frontend/routes/system/companies/DOCUMENTATION.md` with the company card list,
  filter/ranking behavior, create/edit/delete flow, administrator fallback, chart semantics, and
  usage drill-down.
- Update `frontend/routes/system/observability/DOCUMENTATION.md` so it documents only Backend
  Services.
- Keep `frontend/routes/system/companies/empresas.md` only if another indexer still consumes the
  legacy description; otherwise fold its content into the route documentation and remove it.
- Refresh documentation provenance hashes and run the documentation validator for both routes.

## 11. Tests

### Backend

- resolves user ID `1` for each company with an exact composite-key query;
- builds `AdminName` from first and last name and falls back to `AdminUser`;
- returns an empty administrator projection for a missing legacy admin;
- never exposes sensitive user fields;
- preserves 30 zero-filled days, CPU/inference totals, deterministic ranking, and API detail;
- preserves SaaS-only enforcement after the access-catalog migration;
- confirms credit queries still include company, aggregate user, and bounded daily frame keys.

### Frontend model/component

- default CPU ranking and optional inference ranking;
- deterministic ties and preserved global rank under filtering;
- filter matches company and administrator fields;
- missing administrator renders the localized fallback;
- CPU/inference arrays always contain 30 finite values;
- zero usage never produces `NaN` or `Infinity`;
- edit click does not open usage detail;
- card click opens the correct company layer;
- day selection requests the correct company/day and memoizes the response;
- Refresh invalidates detail memoization.

### Shared chart

- existing stacked mode remains the default;
- grouped mode gives each series a non-overlapping bar slot for the same day;
- grouped geometry remains inside the plot bounds for one or multiple series;
- grouped/stacked cache keys cannot collide.

## 12. Validation

1. Run focused backend company-credit tests and the full Go suite.
2. Run company credit model and shared chart tests.
3. Verify the generated API registry is unchanged/current.
4. Run `git diff --check` and the production frontend build.
5. Validate both route documentation files.
6. Use `agent_browser` as platform company `1`, user `1` to verify:
   - cards are ordered by CPU by default and reorder by inference;
   - filter searches company and administrator content;
   - green/purple grouped bars render all 30 days;
   - card click opens usage detail;
   - pencil click opens only the edit modal;
   - create/edit/delete refresh the card collection;
   - hover, keyboard focus, and mobile/touch edit affordances are usable;
   - company/day/API detail retains actual route names;
   - Observability no longer shows Company Credits.
7. Verify a non-SaaS company cannot navigate to Companies or call either cross-company credit API.

## 13. Acceptance Criteria

- `/system/companies` uses company cards instead of the current company table.
- Every card identifies the company and its canonical administrator without exposing sensitive user
  data.
- Every card shows 30 daily CPU values in green and inference values in purple.
- Cards are ranked by the selected independent credit pool from greatest to least usage.
- The edit pencil is hover-revealed on desktop, keyboard/touch accessible, and opens only the
  existing edit modal.
- Card selection preserves the company/day/API read-only drill-down.
- `/system/observability` contains only Backend Services.
- Credit report APIs are authorized through the Companies access item and remain SaaS-only.
- No new database table, delta cache, or `server_utils` writer path is introduced.
