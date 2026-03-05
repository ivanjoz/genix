# Backend/VPS Metrics SSE API Plan

## Objective
Create a backend API that streams server metrics every second to the frontend using Server-Sent Events (SSE), including:
- Memory usage
- CPU usage
- Disk usage
- Current connections
- Bandwidth usage per second

## Scope and Assumptions
- Primary runtime target: local/self-host HTTP server mode (`IS_LOCAL=true`), not AWS Lambda API Gateway responses.
- Metrics endpoint is operational/diagnostic and must be protected by auth + explicit permission.
- VPS is Linux-based (required for accurate bandwidth and system metrics via `/proc` and `syscall`).
- Stream cadence is 1 second by default, configurable with a safe min/max.

## Key Constraints Discovered in Current Codebase
- Current response path (`core.SendLocalResponse`) always compresses (`gzip`/`zstd`), which is incompatible with SSE streaming flush behavior.
- Local server currently uses `WriteTimeout: 60s`, which can cut long-lived SSE connections.
- Existing handler pattern is request/response oriented, so SSE needs either:
- a dedicated direct writer path in the main handler flow, or
- a dedicated handler response mode that bypasses compression and writes chunked events.

## High-Level Architecture
1. Add a dedicated SSE route (example: `GET.system-metrics-stream`).
2. Authenticate and authorize request (for example admin/superuser or specific `AccesoID`).
3. Open SSE stream with required headers.
4. Every second, collect a metrics snapshot from a system collector module.
5. Emit one SSE event with JSON payload.
6. Flush immediately.
7. Stop cleanly on client disconnect or context cancellation.

## Metrics Contract (Proposed)
Each event payload should include:
- `timestamp_unix`
- `cpu.percent_used`
- `memory.total_bytes`
- `memory.used_bytes`
- `memory.available_bytes`
- `memory.percent_used`
- `disk.mount_path`
- `disk.total_bytes`
- `disk.used_bytes`
- `disk.free_bytes`
- `disk.percent_used`
- `connections.http_active`
- `bandwidth.rx_bytes_per_sec`
- `bandwidth.tx_bytes_per_sec`
- `bandwidth.interface_name`

Event names:
- `event: metrics` for normal updates
- `event: warning` for degraded data (partial metrics)
- `event: error` for terminal stream errors
- optional keepalive comment line every 15 seconds

## Proposed Backend File Changes
- `backend/handlers/main.go`
- Register route: `GET.system-metrics-stream`.

- `backend/handlers/system_metrics_sse.go` (new)
- SSE handler with auth, validation, stream loop, flush, and disconnect handling.

- `backend/system/metrics_collector.go` (new)
- Snapshot collector logic (CPU/memory/disk/connections/bandwidth).
- Cache prior network counters to compute per-second deltas.

- `backend/system/metrics_types.go` (new)
- Strongly typed payload structs for stream events.

- `backend/core/responses.go`
- Add explicit bypass path for SSE to avoid compression and write event chunks directly.
- Preserve existing behavior for all non-SSE routes.

- `backend/main.go`
- Adjust server timeouts for SSE compatibility:
- keep `ReadHeaderTimeout` and `ReadTimeout`,
- avoid global `WriteTimeout` kill for stream endpoints (route-aware handling or large safe timeout).

## Detailed Implementation Phases

## Phase 1: Stream Transport Support
- Add SSE-aware response flow that:
- sets `Content-Type: text/event-stream`,
- sets `Cache-Control: no-cache`,
- sets `Connection: keep-alive`,
- disables compression for this route,
- uses `http.Flusher` and fails early if unsupported.
- Add strong debug logs for stream start, per-tick send, disconnect reason, and errors.

## Phase 2: Metrics Collector
- CPU:
- compute usage percent from deltas between successive `/proc/stat` reads.
- Memory:
- parse `/proc/meminfo` (`MemTotal`, `MemAvailable`) and derive used/percent.
- Disk:
- use `syscall.Statfs` on configured mount (default `/`).
- Connections:
- track active HTTP connections with server `ConnState` counters (preferred),
- optional fallback: parse `/proc/net/tcp` + `/proc/net/tcp6`.
- Bandwidth:
- parse `/proc/net/dev` for selected interface,
- calculate per-second RX/TX deltas from prior sample.
- Collector returns partial data + warnings instead of failing whole stream when one metric source errors.

## Phase 3: Handler + Route
- Implement `GetSystemMetricsStream(req *core.HandlerArgs) core.HandlerResponse`.
- Validate:
- authenticated user exists,
- explicit permission check,
- optional query params (`interval_ms`, `iface`, `mount`) with safe bounds.
- Start ticker loop (default 1000ms).
- Build JSON event payload and emit as SSE.
- Exit on `req.ReqContext.Context().Done()`.

## Phase 4: Runtime Safety
- Connection limits:
- cap concurrent stream clients (example: 5-20) to protect VPS.
- Backpressure:
- if write/flush takes too long, drop connection with logged reason.
- Resource cleanup:
- stop ticker and decrement counters on any exit path.

## Phase 5: Frontend Consumption Contract
- Frontend uses `EventSource` to consume `GET /system-metrics-stream`.
- Reconnect behavior:
- rely on default EventSource retries,
- optional custom `retry:` directive in SSE stream.
- Client should display:
- current values,
- warning state when partial metrics,
- “stream disconnected” state with retry indicator.

## Validation and Test Plan
- Unit tests for collector parsers:
- `/proc/stat` CPU delta calculations,
- `/proc/meminfo` parsing,
- `/proc/net/dev` delta calculations.
- Integration test (local):
- open SSE connection,
- assert first event <= 2s,
- assert event cadence ~1s,
- assert disconnect cleanup on client close.
- Manual checks:
- open 2-3 simultaneous clients,
- verify CPU/memory overhead remains acceptable,
- verify no gzip/zstd headers on SSE response.

## Observability and Logging
- Add structured debug logs with fields:
- `route`, `user_id`, `empresa_id`, `stream_id`,
- `tick_ms`, `collect_duration_ms`, `write_duration_ms`,
- `active_streams`, `warnings_count`, `error`.
- Add startup log indicating SSE metrics endpoint enabled.

## Security Checklist
- Require authenticated user.
- Require explicit authorization (new access code recommended).
- Validate and clamp query parameters to safe values.
- Do not expose sensitive host identifiers or raw process lists.
- Throttle reconnect storms (basic per-IP short window limit recommended).

## Deployment Notes
- For AWS Lambda/API Gateway path, SSE is not a good fit with current architecture.
- If SSE is required in cloud deployment, run the API behind a persistent server process (EC2/VPS/container) instead of Lambda response mode for this route.

## Open Decisions You Should Confirm
- Route name: `system-metrics-stream` or another naming convention.
- Authorization rule: admin role only vs specific `AccesoID`.
- Default network interface for bandwidth (`eth0`, `ens3`, auto-detect).
- Default disk mount (`/` or configurable from env).
- Max concurrent SSE clients allowed.

## Delivery Checklist
- [ ] SSE route registered
- [ ] Stream transport path implemented without compression
- [ ] Metrics collector implemented and tested
- [ ] Auth + permission validation added
- [ ] Per-second payload emitted and flushed
- [ ] Timeout and connection limits configured
- [ ] Unit/integration tests passing
- [ ] Frontend EventSource integration documented
