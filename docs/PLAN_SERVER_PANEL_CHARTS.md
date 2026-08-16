# Plan — Server Panel: 4-hour metric charts over a delta cache

Status: **implemented**. Verified end to end against the live homelab data — see §8.

Replaces the Dashboard tab's live SSE table (`frontend/routes/system/server-panel/DashboardView.svelte`)
with six AWS-CloudWatch-style charts drawn from the `server_metrics` table that `server_utils`
already fills every five seconds (`server_utils/PLAN_SERVER_METRICS.md`).

The table is written and never read today. This plan is the read side: one API route, one frontend
service, six charts.

---

## 1. The charts

Window: the **last 4 hours**, so 2880 five-second slots.

| # | Chart | Left axis | Right axis (`useOwnAxis`) |
| - | ----- | --------- | ------------------------- |
| 1 | Host | `CpuPercent` %, 0–100 | `MemPercent` % — same axis, no second axis needed |
| 2 | Backend service | `BackendCpuPercent` %, 0–100 | `BackendMemMb` MB, auto-scaled |
| 3 | ScyllaDB | `ScyllaCpuPercent` %, 0–100 | `ScyllaMemMb` MB, auto-scaled |
| 4 | GenixSearch | `SearchCpuPercent` %, 0–100 | `SearchMemMb` MB, auto-scaled |
| 5 | Server Utils | `ServerUtilsCpuPercent` %, 0–100 | `ServerUtilsMemMb` MB, auto-scaled |
| 6 | Network | `NetRxRate` and `NetTxRate`, both KB/s on one axis | — |

`DiskPercent` has no chart of its own: it moves by fractions of a percent over four hours, so a line
of it is a flat line. It becomes a headline number above the charts, showing the newest slot's value.

Charts 2–5 pin CPU to a fixed 0–100 axis, as asked, so a service that is idle looks idle instead of
being auto-scaled up to fill the plot. Memory keeps its own axis because MB and % share no scale.

**Rendering** uses the existing `ChartCanvas` (`frontend/packages/genix-ui/charts/ChartCanvas.svelte`),
already used by `SaleOrdersChartsByProduct.svelte`. It supports exactly what is needed:
`type: "line"`, `useOwnAxis`, `yAxisMin`/`yAxisMax`, `dateLabels` + `dateLabelFormatter`, and
`values: Array<number | null>` — `null` draws a gap.

**Gaps are not zeros.** `-1` in the table means *not measured* (no cgroup for that unit — the normal
state of `Backend*` when the backend is a Lambda) and an absent slot means the daemon was not
running. Both map to `null`, never to `0`, or a Lambda deployment would show a backend flatlining at
0 % instead of an empty chart.

**Downsampling.** 2880 points into ~1000 px is 3 points per pixel. The view reduces to ~360 points by
taking the **maximum** of each group. That is the only reduction that is faithful here: every stored
value is already the peak of its five seconds, so max-of-peaks is still a peak, while an average
would silently invent a number that never happened.

---

## 2. Why the obvious shapes do not work

The delta cache keys a record by `ID` and watermarks on `upd`. `server_metrics` has neither —
`DisableDefaultColumns: true`, key `(Date, Slot)`. So the response shape has to be designed, not
derived. Three candidates:

**One record per slot** — 2880 records × 14 fields. ~576 KB of JSON key names per cold load and 2880
IndexedDB rows per sync. Rejected on payload alone.

**One record per day, columnar** (what `sale_orders_charts` does with `keyID = "Date"`) — deltas are
tiny, but the record grows all day to 17280 slots and nothing ever evicts a past day. A panel left
open accumulates ~1.3 MB per day in IndexedDB, forever.

**One record per hour, columnar** — chosen. See below.

---

## 3. Response shape

One record per **wall-clock hour**, holding that hour's slots as parallel arrays.

```go
// backend/config/server_metrics.go

// One hour of samples as parallel arrays. Columnar and not one record per slot because a five-second
// series is 2880 records per four hours, and repeating fourteen JSON key names 2880 times costs more
// than the numbers themselves.
type ServerMetricsHour struct {
    // Hours since the unix epoch (unixSeconds / 3600). Monotonic, so it doubles as the cache key and
    // as the bucket's identity; the wall-clock hour is recoverable by multiplying back.
    BucketID int32 `json:"ID"`
    // Offset of each sample inside the hour, 0..719. Positions every other array, and is the field
    // the columnar merge aligns on. Offsets rather than slot-of-day so a value stays three digits.
    SlotsInHour []int16

    CpuPercent  []int16
    MemPercent  []int16
    DiskPercent []int16
    NetRxRate   []int16
    NetTxRate   []int16

    BackendMemMb          []int16
    BackendCpuPercent     []int16
    ServerUtilsMemMb      []int16
    ServerUtilsCpuPercent []int16
    SearchMemMb           []int16
    SearchCpuPercent      []int16
    ScyllaMemMb           []int16
    ScyllaCpuPercent      []int16

    // Absolute slot (unixSeconds / 5) of the newest sample in this bucket. The delta watermark: the
    // client sends back the maximum across records and the handler returns strictly newer slots.
    Updated int32 `json:"upd"`
}

type ServerMetricsResponse struct {
    Hours []ServerMetricsHour
    // Buckets that fell out of the window. `<key>_IDsToRemove` is the cache's own eviction channel
    // (delta-cache.fetch.ts:162) and deletes the persisted rows before the delta is applied — a bare
    // list of ints, so evicting a day of buckets costs ~200 bytes.
    HoursIDsToRemove []int32 `json:"Hours_IDsToRemove,omitempty"`
}
```

The response is an **object, not a bare array**, purely so the `Hours_IDsToRemove` channel exists —
a bare array normalizes to the `_default` key and has nowhere to hang the flag. The consequence is
that the cache sends the watermark as `?Hours=<upd>` (named after the struct field) rather than
`?upd=`, per the multi-table rule in the `delta-cache-api` skill.

### Why the hour is the right bucket

- **Sealed buckets are immutable.** Once an hour is past, its record can never change, so it is
  fetched exactly once and never re-sent. Only the live bucket produces deltas.
- **Bounded record size.** 720 slots, versus a day record's unbounded 17280.
- **Eviction has a natural unit.** The window is whole buckets, so "what fell out" is a list of
  integers the handler can compute without touching the database.

---

## 4. Backend handler

New route `GET.server-metrics`, in `backend/config/server_metrics.go` next to the existing
`GetSystemMetricsStream`.

```
GET /api/server-metrics?hours=4&Hours=<watermark>
```

| Param | Meaning |
| ----- | ------- |
| `hours` | Window width, default 4, clamped to 1..24. Part of the route string, so each width is its own cache collection with its own watermark. |
| `Hours` | Delta watermark: the newest absolute slot the client holds. Absent or 0 → cold load. |

Logic:

1. `nowUnix := core.Now().Unix()`; `firstLiveBucket := nowUnix/3600 - int64(hours) + 1`.
2. `firstSlotWanted := max(firstLiveBucket*720, watermark+1)` — the watermark wins when it is inside
   the window, which is what makes the steady-state response a handful of slots.
3. Read the rows. The window spans at most two unix days, so at most two queries:
   `db.Query(&rows).Date.Equals(day).Slot.Between(fromSlot, toSlot)` — partition on `Date`,
   clustering range on `Slot`, exactly the access pattern the table was keyed for. No index needed,
   which is why the table has none.
4. Group rows into `ServerMetricsHour` by `unixSeconds/3600`, appending to the parallel arrays in
   slot order, and set each bucket's `Updated` to its newest absolute slot.
5. `HoursIDsToRemove` = every bucket in `[bucketOf(watermark) - 24, firstLiveBucket)`, capped at 400
   entries. Emitted only when the client's watermark is old enough that its live set actually
   changed, so a 15-second refresh inside the same hour carries none. The `- 24` covers the widest
   window a client may have cached; the cap bounds a client returning after a very long absence, and
   in that case a few orphan buckets survive until the next `ver` bump.
6. When `hours` is absent **and** the watermark is 0, return the full window; that is the cold path.

The handler reads **no `CompanyID`** — `server_metrics` is machine-wide, not tenant data. Access is
therefore purely the SaaS gate:

- `backend/main-handlers.go` → add `"GET.server-metrics": true` to `saasOnlyRoutes`.
- `backend/access_list.yml` → access id 6 ("Server Panel") currently has `backend_apis: ""`; set it
  to `"GET.server-metrics"`.
- `backend/config/main.go` → register `"GET.server-metrics": GetServerMetrics`.
- Regenerate `backend/core/api_routes.generated.go` and `backend/exec/controllers.generated.go`
  with `cd scripts && go run . generate_controllers`.

### Clock

The handler uses `core.Now()` per the project rule. Worth stating plainly: `server_utils` writes with
the real clock and has no `GENIX_HISTORICAL_UNIX`, so if the backend ever ran under a frozen clock
this panel would query a day the daemon never wrote. That only affects seed scripts, never the API
server — noted rather than guarded, since a guard here would be dead code.

---

## 5. Frontend service

```ts
// frontend/routes/system/server-panel/server-metrics.svelte.ts

export class ServerMetricsService extends GetHandler {
  route = 'server-metrics?hours=4'
  useCache = { min: 0.2, ver: 1 }     // 12 s TTL, under the 15 s refresh interval
  keyID = 'ID'
  columnarIDField = 'SlotsInHour'
  combineColumnarValuesOnFields = [
    'CpuPercent', 'MemPercent', 'DiskPercent', 'NetRxRate', 'NetTxRate',
    'BackendMemMb', 'BackendCpuPercent', 'ServerUtilsMemMb', 'ServerUtilsCpuPercent',
    'SearchMemMb', 'SearchCpuPercent', 'ScyllaMemMb', 'ScyllaCpuPercent',
  ]
  ...
}
```

`handler()` receives `{ Hours: [...] }`, sorts by `ID`, and flattens the buckets into one dense
series of 2880 slots indexed by `(bucketID*720 + slotInHour) - firstSlotOfWindow`, leaving every
unfilled position `null`. The view then downsamples that dense array.

The Dashboard drives a `setInterval` of 15 s calling `service.fetch()` while the document is visible,
and stops on `onDestroy` and on `visibilitychange` — a background tab polling a server panel is
traffic nobody asked for.

### What the delta actually saves

| | Records on the wire | Slots on the wire |
| - | - | - |
| Cold load (4 h) | 5 buckets | 2880 |
| Refresh after 15 s | 1 bucket | 3 |
| Refresh across an hour boundary | 2 buckets | 3, plus up to 400 ints of eviction |
| Reopened after 30 min | 1–2 buckets | ~360 |

Cold load is ~200 KB of JSON (~30 KB over the wire after compression, since it is almost entirely
digits). Every subsequent refresh is under a kilobyte. Without the delta, each 15-second refresh
would re-download the full 200 KB — which is the thing this design exists to avoid.

---

## 6. Files

| File | Change |
| ---- | ------ |
| `backend/config/server_metrics.go` | **new** — `GetServerMetrics`, the two structs, bucketing |
| `backend/config/server_metrics_test.go` | **new** — bucketing, watermark, eviction range, day rollover |
| `backend/config/main.go` | register the route |
| `backend/main-handlers.go` | `saasOnlyRoutes` |
| `backend/access_list.yml` | access id 6 `backend_apis` |
| `backend/core/api_routes.generated.go`, `backend/exec/controllers.generated.go` | regenerated |
| `frontend/routes/system/server-panel/server-metrics.model.ts` | **new** — units, flatten, downsample (no runes, so it is testable) |
| `frontend/routes/system/server-panel/server-metrics.model.test.ts` | **new** — 7 tests: units, peak reduction, gaps, window edges |
| `frontend/routes/system/server-panel/server-metrics.svelte.ts` | **new** — the `GetHandler` subclass and nothing else |
| `frontend/packages/genix-ui/charts/ChartCanvas.svelte` | `sharedAxisMaxValue` prop, and an own-axis fix (§8) |
| `frontend/routes/system/server-panel/DashboardView.svelte` | rewritten: six charts + disk headline, SSE client removed |
| `frontend/routes/system/server-panel/server-panel.md` | route description refresh |

---

## 7. Decisions taken

1. **The Dashboard section is replaced entirely.** The metrics table, the `System Messages` console
   and the `Connected / reconnects / last event` strip all go, and with them the whole SSE client in
   `DashboardView.svelte`. The tab becomes the six charts and the disk headline, nothing else. The
   **Memory tab is untouched** — `MemoryView.svelte` reads `GET.system-memory-packages`, a different
   route with nothing to do with this.
2. **Fixed 4 hours.** `?hours=` stays in the handler so a selector is a frontend-only change later,
   but the service ships one width and therefore one cache collection.
3. **Chart 1 draws both series as percentages on one 0–100 axis.** No `useOwnAxis`, no memory total
   needed — which is just as well, since `server_metrics` does not store one.

---

## 8. What the build changed, and what it measured

### Two `ChartCanvas` defects the charts exposed

**An own-axis series was still setting the shared scale.** `getLineRange` honours `useOwnAxis`, but
`getChartMetrics` summed every line into `maxChartValue` regardless — and the y-axis *labels* are
always built from that. The ScyllaDB chart therefore drew a 0–100 CPU line against an axis labelled
0–543, the memory series' own peak in MB: a scale that described neither series. `getChartMetrics`
now excludes own-axis series, which is what `useOwnAxis` already meant everywhere else. This also
fixes `SaleOrdersChartsByProduct`, where a high unit price was inflating the sales-volume bar axis.

**No way to pin the shared axis.** Added the `sharedAxisMaxValue` prop, so a CPU chart reads 0–100
whether the machine is idle or pinned. It only raises the axis, so a series exceeding the expected
ceiling is still drawn in full.

### Two view-level corrections

- **Own-axis memory had no headroom.** Scaled to exactly its own peak, a memory line always touches
  the top of the plot whatever its value, so a flat 545 MB looked like a service at its limit. The
  axis now carries 25% headroom, putting the peak at 80% of the height.
- **The last x-axis label named the wrong time.** Labels name the *start* of the span they cover,
  and the final one is drawn hard against the right edge — so the axis read "05:39 PM" on a chart
  that ran to 06:11 PM. The final label now prints the window's end.

### Measured on the live homelab data

| Request | Rows | Payload |
| ------- | ---- | ------- |
| Cold load, 4 h window | 391 | 16,888 B |
| Refresh, 15 s later | 3 | 521 B |

Three rows per fifteen seconds is exactly the sample rate, so the delta is carrying the new samples
and nothing else. Without it every refresh would re-send the full window.

The eviction channel was checked by replaying a three-hour-stale watermark against the running
backend: `Hours_IDsToRemove` came back as `[496316 … 496339]` — 24 ids, `clientNewestBucket - 24`
through `firstLiveBucket - 1`, exactly the range §4 specifies.

Only 391 of a possible 2880 rows existed: the daemon had been writing for about 33 minutes. The
chart renders that correctly as three and a half hours of gap rather than a line at zero, which is
the `-1`/absent handling doing its job.

### Consequence: `GET.system-metrics-stream` loses its only caller

Dropping the SSE client leaves `backend/config/system_metrics_sse.go` with nothing in the frontend
reading it. It is **left in place by this plan** and called out here rather than deleted quietly:
removing it is a separate decision, and it is not the reason the `sse_bridge` exists — `/agent/stream`
is, and that is unaffected either way.
