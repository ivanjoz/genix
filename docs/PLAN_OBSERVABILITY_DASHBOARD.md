# API Observability Dashboard Plan

## Goal

Add a SaaS-administrator page at `/system/observability` that renders one compact card per API
`RouteID`. Each card shows a four-hour, five-minute-bucket chart of estimated successful requests
and actual failed requests, plus the most frequent error previews for that route.

The implementation must reuse the existing `credit_usage`, `user_logs`, and `request_errors`
data, transfer only deltas after the initial load, and resolve error text through the project's
get-by-IDs cache.

## Scope

The page is **platform-wide**, not limited to the authenticated company. That matches its location
under `System`, the existing SaaS-only policy for that menu, and the reference service-logs
dashboard. The credit source is therefore the reserved platform aggregate described below, while
`user_logs` is aggregated across every company in each requested frame.

## What the code already provides

| Concern | Existing contract | Consequence for this page |
|---|---|---|
| Credit storage | `credit_usage` is partitioned by `CompanyID`, clustered by `UserID, TimeFrame`, and contains a packed per-route credit blob. | Read company aggregate rows with `UserID=-1`; never sum user rows because that would double-count. |
| Credit intervals | Five-minute frames are `100_000_000 + unixSeconds/300`; daily frames use the `200_000_000` prefix. | The chart can use the five-minute rows directly. |
| Credit writes | The daemon replaces absolute snapshots every 15 seconds and the current five-minute row keeps changing. | A simple `TimeFrame` watermark would become stale; every delta must overlap and replace the live bucket. |
| Request logs | `user_logs` is partitioned by Unix day and has one row per recorded request. The current view key is a **15-minute** frame followed by route and company. | Query contiguous 15-minute ranges, then regroup rows into five-minute buckets in Go. Do not change the on-disk frame format merely for presentation. |
| Log retention | `user_logs` rows have a configured TTL, normally 30 days. | A four-hour dashboard is well inside the retention window. |
| Log contents | `ErrorCount` counts captured errors and `ErrorIDs` points to deduplicated descriptors. Successful requests are normally absent unless `LOG_ALL_REQUESTS` is enabled. | Failed requests are exact; total requests must remain explicitly estimated from credits. |
| Error descriptors | `request_errors` is partitioned by hashed `ID` and clustered by `CodeLine` to preserve rare hash collisions. `Text` is only a preview. | Return one cache record per ID containing an `Entries[]` array; never flatten it to one descriptor and lose collisions. |
| Route names | `core.APIRouteNames` reverses the stable generated route registry, including retired routes. | Backend responses can include readable `METHOD.route` names without a frontend registry copy. |
| Existing delta model | The cache replaces records by stable `ID` and keeps the maximum `Updated` watermark. | Use one record per natural five-minute frame; resend the mutable current frame inclusively and replace it under the same ID. |
| Authorization | System routes and menu items use both `onlySaaS` and `saasOnlyRoutes`; `access_list.yml` still maps explicit page/API access. | Apply all three layers to the new page and endpoints. |

## Accuracy rules

The page must not label the credit conversion as an exact invocation count.

- Minimum CPU cost is 2 credits for GET and 5 credits for POST.
- Larger payloads add credits, so `CPUCredits / minimumCost` is an estimate, generally an upper
  bound on charged requests.
- Public routes bypass credit charging, failed GET handlers are not charged, and unsupported
  methods such as PUT are not represented by this CPU-credit formula.
- One failed request can contain several captured errors. Preserve both `FailedRequests` (rows
  with `ErrorCount > 0`) and `ErrorOccurrences` (sum of `ErrorCount`).
- The stacked chart should use:
  - green: `max(floor(CPUCredits/minimumCost) - FailedRequests, 0)` estimated successes;
  - red: actual `FailedRequests`;
  - card metadata: CPU credits, inference credits, and error occurrences.
- Cards with errors but no metered credits must show an `Unmetered`/`Sin medición` badge instead
  of inventing a request total.

## Data flow

1. `server_utils` keeps the existing user/company credit rows and additionally maintains one
   reserved platform five-minute aggregate.
2. `GET.observability` reads the platform credit frames and the corresponding `user_logs` view
   ranges in parallel.
3. The handler decodes credit blobs, converts request IDs to timestamps, and merges both sources
   into one record per five-minute frame, with all route details inside that record.
4. The frontend delta cache replaces incoming frame records by `ID` and evicts frames that left the
   four-hour window.
5. The page collects only the visible `ErrorIDs` and resolves them through
   `GET.request-errors-by-ids` plus `getStaticRecordsByID`.

## Backend implementation

### 1. Add a platform credit aggregate safely

Files:

- `server_utils/src/limiter/quota.rs`
- `server_utils/src/limiter/aggregation.rs` if shared state helpers are useful
- relevant Rust tests

Use reserved identity `(CompanyID=0, UserID=-1)` for the platform aggregate. Keep it outside the
per-company shard maps: inserting the same reserved key into several shards would create competing
absolute snapshots and lose credits on flush.

Add one dedicated mutex-protected platform usage state to `RateLimiter`:

- lazily load the current reserved five-minute row before its first mutation;
- track the loaded frame so a frame rollover loads/creates the new row exactly once;
- increment it only after a request has passed the normal company/user limits;
- include it in dirty snapshots, successful flush marking, and clean historical pruning;
- do not use it for quota decisions;
- preserve the current absolute-write and mutation-version behavior so a concurrent mutation stays
  dirty after an older flush completes.

The first deployment will not have older reserved rows. Add a one-off backfill command under
`scripts/` that reads each company's `UserID=-1` five-minute rows for the desired window, sums their
decoded route values, and writes the reserved rows. Make it idempotent by writing absolute totals.

### 2. Expand the `user_logs` dashboard view projection

File: `backend/core/types/user_logs.go`.

Keep the existing `FrameRouteCompanyAgg` key and 15-minute packing. Expand the view payload from
`ErrorCount, CompanyID` to include at least `RouteID` and `ErrorIDs`. `RequestID` remains available
as a base primary-key column and supplies the exact request time.

An existing materialized view does not acquire new projected columns through `fn-homologate` alone.
Run `cd scripts && go run . rebuild_observability_log_view` to drop and recreate only the
`user_logs` derived view/index artifacts, then run `cd scripts && go run . check_tables`. No new
base-table column is required.

### 3. Add the delta observability handler

Suggested files:

- `backend/config/observability.go`
- `backend/config/observability_test.go`
- `backend/config/main.go`

Register `GET.observability` and restrict it in `backend/main-handlers.go` as SaaS-only.

Use a four-hour default and cap the accepted window. Use the real UTC clock because
`server_utils` also uses real UTC time; the historical test clock must not move this dashboard.

Response record `ObservabilityFrame`:

| Field | Purpose |
|---|---|
| `ID int32` | SUnix time of the five-minute frame start; the natural cache identity. |
| `TimeFrame int32` | Original `credit_usage` frame (`100_000_000 + unixSeconds/300`) for diagnostics. |
| `Details []ObservabilityRouteDetail` | Every route with credit or error activity in this frame, sorted by `RouteID`. |
| `Updated int32` as `upd` | Same frame-start SUnix value as `ID`; drives the cache watermark. |
| `Status int8` as `ss` | Always 1 for live records. |

Each `ObservabilityRouteDetail` contains `RouteID`, CPU credits, inference credits, estimated
requests, failed requests, error occurrences, and aligned `ErrorIDs`/`ErrorIDCounts` arrays. Keep
the readable route name out of every frame to avoid repeating the same string up to 48 times.

Return `ObservabilityResponse{Frames, FramesIDsToRemove, Routes}`:

- `Frames` is the delta-cached five-minute collection.
- `FramesIDsToRemove` evicts frame IDs older than the requested window and includes every resent
  overlap-frame ID so the cache deletes and immediately reinserts that absolute record.
- `Routes` is a small separately cached collection of `{ID: RouteID, Route: METHOD.route}` records
  from `core.APIRouteNames`. Return it on cold load and bump the frontend cache version when the
  generated route registry contract changes.

Naming the main collection `Frames` means the frontend sends its watermark as
`?Frames=<frameStartSUnix>`; also accept `?upd=`.

Delta algorithm:

- Cold load: query the full four-hour credit-frame range and all intersecting 15-minute log frames,
  then return at most 48 `Frames` records.
- Convert a stored five-minute frame to SUnix with
  `UnixToSunix((TimeFrame-fiveMinutePrefix)*300)`. Do not call `SUnixTime()` for an old frame;
  `SUnixTime()` means now, while this conversion must preserve the frame start.
- Refresh: convert the SUnix watermark back to its five-minute frame and query **inclusively** from
  at least one frame before it. The ordinary `Updated > watermark` rule is intentionally not used:
  the current `credit_usage` row is an absolute snapshot that changes every 15 seconds while its
  frame-derived `Updated` stays constant.
- Because that unchanged frame-derived `Updated` does not advance the cache watermark, list every
  resent overlap ID in `Frames_IDsToRemove` in the same response. The cache removes the old row and
  then applies the incoming absolute row under the same ID, making the refresh idempotent without
  inventing a poll-time `Updated` value.
- Return the previous and current frame as absolute records in the steady state. Replacing those
  two small records catches late credit flushes and request-log batches without re-downloading the
  four-hour window.
- Query credit usage from reserved `(0,-1)` and decode with the same strict canonical blob rules
  already used by `credit_usage.go`.
- Query `user_logs` per UTC-day partition with `Date.Equals(day)` and
  `FrameRouteCompanyAgg.Between(...)`; issue at most one range query per intersected day.
- Explicitly select only `Date`, `RequestID`, `FrameRouteCompanyAgg`, `RouteID`, `ErrorCount`, and
  `ErrorIDs`. Selecting the full `UserLog` record includes columns absent from the view, makes the
  ORM fall back to the base table, and Scylla rejects that range scan without `ALLOW FILTERING`.
- Decode request time with tested helpers for both stored ID layouts:
  - VPS: `SUnixTime*10_000_000 + counter`;
  - serverless: `SUnixTimeMilli*1_000_000 + salt/counter`.
  Their magnitudes are disjoint for supported dates, but the helper must reject impossible values.
- Ignore success rows from `LOG_ALL_REQUESTS`; only `ErrorCount > 0` contributes failures.
- Build a complete absolute route-detail collection for every returned frame. Re-reading an
  overlap must replace the frame record, never add its metrics to the cached copy.
- Run independent credit/log reads concurrently and log window, watermark, source-row counts,
  output frame count, and eviction count.
- Generate one expired SUnix frame ID per five-minute frame that left the window, using the same
  capped-eviction strategy as server metrics. This is substantially smaller and simpler than
  evicting one record per route and hour.

### 4. Add a collision-safe error by-IDs endpoint

Suggested file: `backend/config/request_errors.go`.

Register `GET.request-errors-by-ids`, restrict it as SaaS-only, parse IDs with
`ExtractUpdatedVersionValues`, and query `request_errors` partitions with `ID.In(...)`.

Return one record per requested hash:

- `ID`: requested error hash;
- `Entries`: every matching `{CodeLine, Text, Updated}` row, sorted by `CodeLine`.

This endpoint is intentionally the static by-ID variant. The stable identity is the code line and
the stored text is already documented as a representative preview, not the authoritative live
message. Full current messages/stacks remain in CloudWatch by `RequestID`. Give the frontend cache
a versioned namespace so a future descriptor-contract change can invalidate IndexedDB cleanly.

### 5. Route registry and access policy

- Add both handlers to `backend/config/main.go`.
- Add both APIs to `saasOnlyRoutes` in `backend/main-handlers.go`.
- Add `system/observability` and its two GET APIs to the existing platform-operations access 6 in
  `backend/access_list.yml`, so profiles that already administer Server Panel inherit the page.
- Regenerate stable route IDs with `cd scripts && go run . generate_route_ids`; never hand-edit
  `backend/core/api_routes.generated.go`.

## Frontend implementation

### 1. Route and service

Suggested files:

- `frontend/routes/system/observability/+page.svelte`
- `frontend/routes/system/observability/observability.svelte.ts`
- `frontend/routes/system/observability/observability.model.ts`
- focused model tests beside the route

`ObservabilityService` should extend `GetHandler` with:

- `route = "observability?hours=4"`;
- cache TTL below the 15-second refresh interval;
- `keyID = "ID"`;
- normal whole-record replacement; no columnar merge configuration is needed;
- a handler that sorts frames by `ID` and exposes them reactively.

The pure model layer should:

- transpose the cached five-minute frame details into one dense series per route;
- zero-fill missing credit/error slots while preserving a visibly empty state when no data exists;
- calculate estimated successes separately from actual failed requests;
- aggregate error-ID counts across the visible window;
- sort route cards by recent CPU credits, then failures, then `RouteID`;
- retain unmetered error-only routes.

After every delta merge, collect error IDs not yet resolved and call
`getStaticRecordsByID("request-errors-by-ids", ids, {cacheNamespace: "request-errors-v1"})`.
Store the returned grouped descriptors in a map; do not embed message text in the observability
delta response.

### 2. Page layout

Use the project `Page` shell with title `Observability|Observabilidad` and a responsive card grid.
The page needs:

- a `FilterInput` for method/path/error-preview search;
- a project `Button` for manual refresh;
- last-updated, four-hour-window, and approximation labels;
- visibility-aware 15-second polling with cleanup on destroy;
- one card per route with method/path header and summary totals;
- `ChartCanvas` with `useHtmlRendered={true}`, two stacked bar series, and 5-minute labels;
- green estimated-success bars and red failed-request bars;
- the top repeated error previews under the chart, showing count, `Text`, and `[CodeLine]`;
- loading, empty, stale/error, and unmetered states.

Do not build a new chart library. `ChartCanvas` already supplies HTML-rendered bars, axes, labels,
and responsive sizing.

### 3. Menu registration

Add this option under `System` in `frontend/core/modules.ts`:

- name: `Observability|Observabilidad`;
- route: `/system/observability`;
- icon: existing bar-chart/activity icon;
- `onlySaaS: true`.

The existing layout will then deny direct URL access for non-SaaS companies in addition to the
backend restriction.

## Tests and validation

### Rust

- accepted charges update user, company, and exactly one platform aggregate;
- multiple company shards cannot create competing platform snapshots;
- first mutation after restart loads and extends the current persisted platform row;
- frame rollover initializes the new frame without losing a dirty previous frame;
- mutation during flush remains dirty;
- rejected requests do not increment the platform aggregate.

### Go backend

- both request-ID layouts decode to the correct Unix/five-minute frame;
- invalid IDs are rejected rather than charted in a wrong slot;
- credit blobs merge by route and five-minute frame;
- 15-minute log queries regroup correctly into three five-minute slots;
- `FailedRequests`, `ErrorOccurrences`, and per-ID counts remain distinct;
- overlap responses are absolute/idempotent;
- GET/POST estimate formulas and unmetered methods are labeled correctly;
- UTC midnight splits database partitions but not the time series;
- frame IDs equal their frame-start SUnix value and eviction never removes a live frame;
- error hash collisions return multiple sorted entries under one ID;
- non-SaaS callers are rejected for both endpoints.

Run the table static check, targeted Go tests, `cargo test` in `server_utils`, frontend model tests,
and the normal frontend type/build validation.

### Browser verification

Use `agent_browser` after implementation to verify:

- the new menu item appears only for the SaaS company;
- direct navigation opens the Page shell;
- cold load and subsequent delta refresh both render;
- route filtering and manual refresh use registered project components;
- cards remain readable at desktop and mobile widths;
- chart bars, totals, and resolved error previews agree with the API payload.

## Rollout order

1. Add and test the platform aggregate writer.
2. Deploy the backend code, then run `rebuild_observability_log_view` so the existing view receives
   `route_id` and `error_ids`; ordinary homologation alone does not rebuild its projection.
3. Run the optional historical backfill before exposing the page.
4. Regenerate route IDs and deploy backend/server-utils together.
5. Add the frontend service, model, page, menu entry, and documentation.
6. Verify cold load, 15-second deltas, frame/hour rollover, and SaaS authorization in the real app.

## Acceptance criteria

- A cold page displays the last four hours from at most 48 five-minute frame records, one card per
  route after the frontend transposes their details.
- After the cold load, normal refreshes replace only the previous/current frame records and send
  any required frame evictions.
- The current credit bucket updates without waiting for the next five-minute boundary.
- Re-read overlap slots replace previous values and never double-count.
- Failed requests and error occurrences remain actual counts; request totals are visibly estimated.
- Error previews are absent from the main payload and resolved in batches through the by-ID cache.
- Error-hash collisions cannot overwrite one another.
- Public/unmetered error routes remain visible and are not assigned fabricated usage.
- Non-SaaS companies cannot see the menu, open the page, or call either endpoint.
