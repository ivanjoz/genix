# Plan — Per-route credit usage, and user_logs as a failure table

Two changes that together split one job in two. Today `user_logs` writes a row for every finished
request and `credit_usage` carries a six-bucket size/method dimension nobody reads. After this,
`user_logs` holds only requests that went wrong, and `credit_usage` breaks its credits down by API
route — so "what did this user spend, on which endpoint" is answered by a compact daily row instead
of by scanning a per-request table.

The two phases touch disjoint files and can land in either order. Phase 3 depends on Phase 2.

## Decisions taken

| Question | Decision |
| --- | --- |
| Scope | Both changes, one plan. |
| 5-minute rows | Per-route breakdown in **both** frames, same as the daily rows. Symmetric; costs a fatter blob on the hot flush path. |
| What lands in `user_logs` | Requests with captured errors, plus credit-limit rejections. No slow-request threshold. |
| Old data | Wiped, not migrated. Pre-alpha. |

## Phase 1 — user_logs only keeps failures

### 1.1 The config key

`config.toml` already has a `[request_log]` section owned by the Rust daemon (`config.rs:195-245`:
`ttl_days`, `flush_ms`, `max_batch`, `enabled`, …). Add one key to that same section, read by Go:

```toml
[request_log]
log_all_requests = false   # true writes a row for every finished request, not only failures
```

The filter belongs on the **Go** side, not in the daemon. Go is what decides to send opcode `0x04`
at all; filtering there means an error-free request costs no frame, no HMAC, and no queue slot in
the writer. Filtering in Rust would still pay all three and then throw the record away.

- `backend/core/security.go:242` — add to `fileConfig`:
  ```go
  RequestLog struct {
      LogAllRequests bool `toml:"log_all_requests"`
  } `toml:"request_log"`
  ```
- `backend/core/security.go:57` — add `LOG_ALL_REQUESTS bool` to `EnvStruct`, near the existing
  `LOGS_FULL` / `LOGS_DEBUG` / `LOGS_ONLY_SAVE` flags, and populate it where `env.ln =
  file.SecretPhrase` and its neighbours are assigned.

The Rust `request_log.enabled` switch stays what it is — a daemon-wide off switch that accepts and
discards. The new key is the backend's "what is worth sending" rule. Two different questions, so
two keys.

### 1.2 The filter

`backend/core/request_errors.go:113` — `EmitRequestLog`. Insert the gate **after** the
`TakeRequestErrors()` drain and after `entries` is built, never before:

```go
if len(entries) == 0 && !Env.LOG_ALL_REQUESTS {
    return
}
```

Draining first is not optional. `TakeRequestErrors` is what clears the per-request accumulator
(`request_errors.go:98`), and returning before it in server mode would leak this request's errors
into whichever request drains next.

### 1.3 Credit-limit rejections need no code

A rejection already produces a captured error: `enforceAPICreditLimit`
(`backend/main-handlers.go:272`) calls `MakeCreditRateLimitResponse`, which builds the 429 through
`req.MakeErrCode` (`backend/core/server_utils_api.go:102`), and `MakeErrCode` records the handler's
line into the accumulator — that is exactly why `logNoCapture` exists (`backend/core/logs.go:142`).
So a rejected request arrives at `EmitRequestLog` with `len(entries) > 0` and passes the gate.

This is worth a test rather than a change, because it is a behaviour we are now depending on:
a rejection that stopped capturing would silently vanish from the table.

The 503 branch (daemon unreachable) captures the same way, which is right — an unavailable limiter
is a failure worth a row.

### 1.4 What this costs

Nothing reads `user_logs` yet — the table and its `FrameRouteCompanyAgg` view exist
(`backend/core/types/user_logs.go`), written only by the daemon, with no Go or frontend reader. So
there is no dashboard to break. What is given up is the ability to ask "how many requests did this
company make" from this table; that question moves to `credit_usage` in Phase 2, which counts
credits rather than requests. If a raw request *count* is wanted later, the honest answer is to add
a counter, not to re-enable per-request rows.

### 1.5 Tests

- `EmitRequestLog` with zero errors and the flag off sends nothing; with the flag on it sends.
- `EmitRequestLog` with zero errors and the flag off still drains the accumulator (assert a second
  call sees nothing left).
- A handler returning a 429 through `MakeCreditRateLimitResponse` leaves at least one captured
  error behind.

## Phase 2 — `api_group` becomes `route_id`

### 2.1 Why this is safe

`api_group` never takes part in a limit decision. `quota.rs` charges and checks against
`sum(&groups)` (`credits_blob.rs:93`); the map is only a breakdown. Its one Go reader,
`decodeCreditUsage` (`backend/config/credit_usage.go:107`), parses the group out of each header and
discards it, summing everything into per-day CPU/inference totals. And the group does not affect
price either — `APICPUCredits` derives credits from method and bytes independently
(`backend/core/server_utils/credits.go:63`). So the dimension is inert today, and replacing it
changes no enforcement and no billing.

### 2.2 Wire format — 11 bytes becomes 12

`[opcode][company:u24][user:u24][route:i16][cpu:u16][inference:u16][hmac:8]`

- `server_utils/src/limiter/protocol.rs` — `CHARGE_PAYLOAD_SIZE: 11 → 12`; `Request.api_group: u8`
  → `route_id: u16`; read big-endian from `payload[6..8]`; cpu/inference shift to `[8..10]` and
  `[10..12]`. Delete `API_GROUP_COUNT`.
- `backend/core/server_utils/credits.go` — `creditChargePayloadSize: 11 → 12`, the offsets in
  `Charge`, and the layout comment at line 13.
- `backend/core/server_utils/locks_test.go:39` — the frame-size table used by the fake daemon.

**Validation must not encode the route table.** The current check rejects `api_group >=
API_GROUP_COUNT`; the equivalent mistake here would be rejecting IDs above `MaxAPIRouteID`, which
would mean every route added in Go gets refused by a daemon built before it — and since charging
fails open (`credits.go:158`), those routes would quietly stop being counted. The daemon must not
know the route table. Replace the check with a plain range check against the encoding ceiling
(2.3), and accept `0` as the unknown-route bucket: `APIRouteID` returns zero for a path that
matched no generated entry (`api_routes.generated.go`), and those credits are still real.

### 2.3 Blob format — a 2-byte header

Today: one header byte, `(api_group << 2) | width_code`, so `MAX_API_GROUP = 63`
(`credits_blob.rs:7,53`). Route IDs are already at 103.

The same design, one byte wider — header becomes a big-endian `u16`:

```
header:u16 = (route_id << 2) | width_code      route_id 0..16383, width_code 0..3
[cpu:width][inference:width]                   width = width_code + 1, 1..4 bytes
```

Entries stay strictly ascending by route, all-zero entries stay omitted, and the smallest-width
rule stays — every canonical-form check in `decode` carries over unchanged, which is the point of
generalising rather than redesigning. An entry costs 2 + 2×width bytes instead of 1 + 2×width.

- `server_utils/src/limiter/credits_blob.rs` — `GroupedCredits: BTreeMap<u8, Credits>` →
  `BTreeMap<u16, Credits>`; `MAX_API_GROUP: u8 = 63` → `MAX_ROUTE_ID: u16 = 16_383`; `encode` /
  `decode` header handling; rename the error variants.
- `server_utils/src/limiter/aggregation.rs:40` — `increment(&mut self, route_id: u16, …)`.
- `server_utils/src/limiter/quota.rs:453` — pass `request.route_id`.
- `backend/config/credit_usage.go:107` — `decodeCreditUsage` reads the 2-byte header. All four
  canonical-form errors it raises must survive the change; it is the only thing standing between a
  corrupt blob and a wrong number on a chart.

`scripts/routes/route_ids_generator.go` — fail generation if the next assigned ID would exceed
16383. At the current rate that is unreachable, which is exactly why it should be a build error and
not a runtime surprise.

### 2.4 Blob size

Four rows are written per charge — user and company aggregate, 5-minute and daily
(`quota.rs:443`). All four now carry the route breakdown, and every dirty flush rewrites the whole
blob absolutely (`quota.rs:399`).

- Today: ≤6 entries, ~54 bytes worst case.
- After: a user's daily row holds one entry per route touched — 20-40 typical, 103 possible — so
  roughly 200 bytes to ~1 KB. The company aggregate daily row is the fattest, since it unions every
  user's routes.

That is the price of the decision to keep both frames symmetric, and it is a real cost on the
5-minute rows, which churn hardest and whose per-route detail nothing will read. Recorded here so
that if flush latency becomes a problem, collapsing the 5-minute frame to a single bucket is the
first thing to try, and it is a change local to `increment_usage`.

### 2.5 Dead code to delete

`APIGroup()` (`backend/core/server_utils/credits.go:41`) and the `apiGroupSmallBytes` /
`apiGroupMediumBytes` constants become unreachable — the group is no longer computed anywhere.
Delete them, along with the `api_group::` field in the accepted-charge log line
(`backend/main-handlers.go:290`), which should print `route::` instead.

`WithCreditRateLimitIdentity(ctx, companyID, userID, apiGroup)` (`credits.go:122`) carries the
group so that a later inference charge lands in the same bucket as its request. It now carries the
route ID; the call sites in `backend/agent/` pass `args.RouteID`.

### 2.6 Data

`credit_usage` has no TTL (`backend/core/types/credit_usage.go`), the format is not
self-versioning, and old rows would decode as garbage under the new header. Truncate the table as
part of the deploy. Pre-alpha, and agreed.

### 2.7 Tests

- Rust: the 12-byte charge round-trips at exact offsets; route `0` is accepted; route 16384 is
  rejected; a blob with routes `{1, 103, 16383}` round-trips; every existing canonical-form
  rejection still fires.
- Go: `decodeCreditUsage` agrees with the Rust encoder on a shared vector — the two decoders are
  independent implementations of one format, and the existing pairing is what keeps them honest.
- Go: the 12-byte payload the client writes matches the offsets the daemon reads.

## Phase 3 — surface the breakdown

`GET.credit-usage` (`backend/config/credit_usage.go:43`) returns 15 days of CPU/inference totals
per scope. The per-route data now exists in the rows it already reads.

Add to `creditUsageScope` a `Routes` breakdown aggregated over the whole queried range — route ID,
CPU, inference — sorted by CPU descending. Range-aggregated rather than per-day-per-route: 15 days
× ~40 routes of nested objects is a payload nothing will plot, while "which endpoints cost this
user the most" is the actual question.

`decodeCreditUsage` currently discards the key; have it return the per-route map alongside the
totals, so there is one parse and not two.

Route IDs travel as numbers. The frontend resolves them to names — `APIRouteNames`
(`api_routes.generated.go`) is the authority, including for retired routes, and that mapping should
be emitted for the client rather than duplicated by hand.

Frontend rendering of the breakdown is **out of scope** here; this phase ends with the API
returning the data.

## Order of work

1. Phase 1 — config key, `EmitRequestLog` gate, tests. Independent, ships alone.
2. Phase 2 — protocol, blob, both decoders, generator bound, dead code, tests.
3. Truncate `credit_usage`, deploy backend and daemon together (the wire format changes on both
   sides at once; a version skew here is a rejected frame, and charges fail open, so a skewed
   deploy silently stops charging).
4. Phase 3 — the API breakdown.

## Out of scope

- Any frontend view of either table.
- Request *counts* per route. Phase 2 gives credits, not counts; nothing after this change can
  answer "how many requests" without a new counter.
- A slow-request threshold for `user_logs`.
- Rolling `credit_usage` up beyond the daily frame.
