---
schema: 1
page_id: system.server-panel
route: /system/server-panel
title: Server Panel (Panel de Servidor)
status: implemented
visibility: saas
description_en: >-
  Machine-level health monitoring, SaaS only. Dashboard view charts the last four hours of host
  and per-service (Backend, ScyllaDB, GenixSearch, Server Utils) CPU, memory, and network; Memory
  view inspects the backend process's live heap usage by Go package.
description_es: >-
  Monitoreo de salud a nivel de máquina, exclusivo SaaS. Vista Dashboard grafica las últimas cuatro
  horas de CPU, memoria y red del host y de cada servicio (Backend, ScyllaDB, GenixSearch, Server
  Utils); vista Memory inspecta en vivo el uso de heap del proceso backend por paquete de Go.
---

# Server Panel (Panel de Servidor)

<!-- DOC-ID: page-purpose -->
## Page purpose

Server Panel (`Panel de servidor`) monitors the physical/virtual machine that hosts Genix, not any
tenant's business data. Open **Administration → System → Server Panel (Administración → System →
Server Panel)** at `/system/server-panel`. The page is visible only to the company that
administers the SaaS platform (`onlySaaS`), and both API routes it calls
(`GET.server-metrics`, `GET.system-memory-packages`) are restricted server-side to that same
company through `saasOnlyRoutes` in `main-handlers.go`.

It has two views, switched with the top tab strip: **Dashboard**, which charts historical CPU,
memory, disk, and network samples written every 5 seconds by the `server_utils` daemon into the
`server_metrics` table; and **Memory**, which asks the running backend process itself for a live
snapshot of its Go heap. This page does not own API-route credit accounting or per-route error
counts — that is **Observability (Observabilidad)** — and it does not manage tenant companies —
that is **Companies (Empresas)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **`server_metrics`** is one row every 5 seconds written directly by `genix-server-utils`, never by
  the backend itself. Every stored value is the **peak** observed during those 5 seconds (not an
  average and not a single point sample), so a one-second spike is never smoothed away. Rows are
  kept for 30 days by default (`ttl_days`, `server_utils` config) before the database expires them.
- **Host vs. per-service metrics**: the Host chart is the whole machine's CPU/memory. Backend,
  ScyllaDB, GenixSearch, and Server Utils each get their own CPU-percent-of-whole-machine and
  memory-in-MB, read from that unit's own Linux cgroup (`memory.stat`, `cpu.stat`), located by
  searching `/sys/fs/cgroup` for the unit rather than assuming a fixed path.
- **Unmeasured vs. zero (`Sin medición` vs. `0`)**: a service metric is a sentinel "not measured"
  (stored as `-1`, surfaced to the chart as a gap/`null`) whenever that unit has no cgroup on this
  box — the normal state for **Backend** when the backend actually runs on Lambda rather than on
  the monitored host — or whenever every sub-sample inside a window failed to read. This is
  intentionally different from a real `0`, which means the service was observed and was idle.
- **Dashboard vs. Memory measure different things about the backend.** The Dashboard's Backend
  chart is the external, periodic cgroup measurement of the backend's own service unit, useful even
  when the backend runs elsewhere. The Memory view instead asks the currently running backend
  process to introspect itself (Go's `runtime.MemStats` plus its own `/proc/self/status` `VmRSS`),
  so it only produces data when the backend you are actually talking to is a local/VPS process —
  it cannot describe a backend running as a Lambda.
- **Network rate (`RX`/`TX`)** is bytes/second on the configured network interface (or the first
  non-loopback interface), read from `/proc/net/dev` counters.
- **Disk (`DISK`)** is shown only as a static badge with the current percentage, not a chart: over a
  four-hour window disk usage moves by fractions of a percent, so a line for it would just be flat.

<!-- DOC-ID: capability.monitor-dashboard -->
## Monitor host and service health (Monitorear la salud del host y los servicios)

### User intention (Intención del usuario)

See whether the host or any Genix service (Backend, ScyllaDB, GenixSearch, Server Utils) is under
unusual CPU, memory, or network load over the last few hours, to diagnose slowness or a crash.

### Where to find it (Dónde encontrarlo)

`/system/server-panel`, **Dashboard** tab (selected by default). Six cards: **Host: CPU & Memory
(Host: CPU y Memoria)**, **Backend Service**, **ScyllaDB**, **GenixSearch**, **Server Utils**, and
**Network (Red)**. Each service card overlays its CPU line (blue, `%`) and memory line (red, MB) on
one chart. Use **Refresh (Actualizar)** for an immediate reload.

### Required information and prerequisites (Requisitos previos)

None from the user beyond having access to this SaaS-only page; the charts always request the
fixed **Last 4 hours (Últimas 4 horas)** window (`WINDOW_HOURS = 4`) — there is currently no control
on this page to widen or narrow that window, even though the backend endpoint itself accepts an
`hours` query parameter up to 24.

### Business rules and rationale (Reglas y razón de negocio)

- CPU series share one 0–100% axis per card so an idle service reads as idle instead of being
  auto-scaled to fill the plot; memory has no fixed ceiling, so it gets its own axis scaled to 125%
  of its own peak in the window (a quarter of headroom so a flat value does not look pinned at the
  chart's limit).
- Every plotted point already reduces 8 raw 5-second slots (40 seconds) into one point by taking the
  **maximum**, matching how the daemon itself already stores each row as a peak — averaging at this
  stage would dilute a real spike by the quiet slots beside it.
- The chart's right edge is always **now**, not the newest row the server actually returned: if
  `genix-server-utils` stops writing, the series keeps growing an empty gap at the right instead of
  looking healthy by ending at its last real sample.
- The page polls every 15 seconds while its browser tab is visible, pauses while the tab is hidden,
  and catches up immediately when the tab becomes visible again.

### Result and side effects (Resultado y efectos)

Read-only: nothing on this view creates, edits, or deletes any record. It only queries
`server_metrics` rows already written by the daemon.

### Limitations (Limitaciones)

- Nothing here can be exported, and there is no way to change the 4-hour window from the UI.
- A card with every metric unmeasured (for example every Backend* field on a Lambda deployment)
  still renders as an empty chart rather than being hidden.
- This is not a request/error monitor: route-level API failures and credit consumption live on
  **Observability**, not here.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿por qué el backend no aparece en la gráfica?` Normal si el backend corre en Lambda: ese servicio
  no tiene cgroup en esta máquina y se muestra como "Sin medición" (hueco en la gráfica), no como 0%.
- `¿cómo veo el uso de CPU del servidor?`, `memoria de ScyllaDB`, `ancho de banda de red`, `uso de
  disco`, `salud del servidor`, `host caído`, `server_utils no está corriendo`.

<!-- DOC-ID: capability.inspect-memory -->
## Inspect backend heap memory (Inspeccionar memoria heap del backend)

### User intention (Intención del usuario)

Investigate the currently running backend process's own memory: how much heap it holds, how many
objects, its OS-level resident memory (RSS), and which Go packages hold the most heap — useful when
diagnosing a memory leak or unexpected growth in the live process, as opposed to historical service
memory shown on the Dashboard.

### Where to find it (Dónde encontrarlo)

`/system/server-panel`, **Memory** tab. A status strip shows **Connected/Disconnected**, the last
update time, **Heap in-use**, **Heap objects**, and **Backend RSS**; below it a table lists up to 20
Go packages ranked by in-use heap bytes (`#`, `Package`, `In-Use (bytes)`, `In-Use (MiB)`,
`Heap %`); a **Memory Warnings** panel underneath lists any collection warnings. **Clear** empties
the table and warnings locally; **Refresh** reloads immediately.

### Required information and prerequisites (Requisitos previos)

Requires a valid session token in the browser; if none is found the view shows "No se encontró un
token válido de sesión." and stops. The backend serving the request must be running in local/VPS
server mode — a Lambda-deployed backend rejects this endpoint outright.

### Business rules and rationale (Reglas y razón de negocio)

- The view polls every 2 seconds automatically from the moment it is opened, **without pausing when
  the browser tab is hidden** — unlike the Dashboard tab, which stops polling in the background.
- Heap-in-use and heap-objects come from the backend process's own `runtime.MemStats`
  (`HeapInuse`, `HeapObjects`); the per-package breakdown comes from sampling Go's heap profile
  records and attributing each record's in-use bytes to the first non-runtime function on its
  allocation stack, then keeping only the top packages by bytes (20 by default, capped at 100 on
  the backend side though this page never asks for more than 20).
- **Backend RSS** is read from `/proc/self/status` (`VmRSS`) of the backend process answering the
  request — this is the process's own OS-reported resident memory, a different measurement path
  than the Dashboard's cgroup-based "Backend Service" memory series, though both describe closely
  related quantities for the same process.
- If the RSS read fails (for example, no `/proc/self/status` available), Backend RSS falls back to
  0 and a `backend_process:<detail>` entry appears in Memory Warnings; the heap totals and package
  table are unaffected because they come from a separate source.

### Result and side effects (Resultado y efectos)

Read-only: it triggers no memory reclamation, no restart, no configuration change. It only reads
runtime and `/proc` state at the moment of the request.

### Limitations (Limitaciones)

- Only available when the backend is running as a local/VPS server process; on a Lambda deployment
  the endpoint answers "La API de memoria por paquetes solo está disponible en modo servidor
  local/VPS." and the view shows that as its error state.
- The package list is capped at 20 rows and cannot be widened from this page.
- Most of this view's own labels (`Connected`, `Heap in-use`, `Clear`, table headers, `Memory
  Warnings`) are only in English; unlike the Dashboard tab, they are not wired through the
  bilingual `T` component, though the session/token error text is in Spanish.
- Package attribution is heuristic (first non-runtime frame on the allocation stack), so it can
  attribute shared/vendored code to whichever package happened to call into it, rather than being an
  exact ownership accounting.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿qué paquete está consumiendo más memoria del backend?`, `heap del backend`, `memoria RSS`,
  `fuga de memoria`, `memory leak`, `objetos en heap`, `por qué la vista de memoria no carga en
  Lambda`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Both tabs are gated the same way at the server: the SaaS-administering company only
  (`saasOnlyRoutes`), enforced independently of the access catalog. The Dashboard's data route
  (`GET.server-metrics`) is additionally mapped to the "Server Panel y Observabilidad" catalog
  access (shared with **Observability**, levels View/Full only); the Memory route
  (`GET.system-memory-packages`) is **not** mapped to any catalog access, so — per the general rule
  that an unmapped GET is open to any authenticated session — any signed-in user of the SaaS company
  can call it even without that specific access.
- Both views are strictly read-only monitoring: neither can restart a service, clear a cache, or
  change server configuration.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No samples in this window" / "Sin muestras en esta ventana":** `genix-server-utils` is not
  running (or not writing) on the monitored host; the Dashboard has no rows for the last 4 hours.
- **A service's CPU/memory chart is entirely empty:** that unit has no cgroup on this box — expected
  for Backend when the backend actually runs on Lambda, or for any unit that is stopped.
- **Dashboard shows a red error banner:** the `server-metrics` request failed — a session/permission
  problem, or the server could not query the table ("No se pudieron obtener las métricas del
  servidor.").
- **Memory tab says "No se encontró un token válido de sesión.":** the browser has no valid session
  token; sign in again.
- **Memory tab fails immediately:** the backend answering the request is running in serverless
  (Lambda) mode, which does not support this endpoint.
- **Backend RSS shows 0 with a warning in Memory Warnings:** the RSS read failed on that host; heap
  totals and the package table remain valid.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **System → Observability (Observabilidad)** at `/system/observability`: monitors backend API
  routes — credits, estimated/failed requests, and repeated errors — rather than machine/service
  resource usage. Shares the "Server Panel y Observabilidad" catalog access with this page's
  Dashboard tab.
- **System → Companies (Empresas)** at `/system/companies`: tenant company administration and
  30-day credit ranking; unrelated to machine health.

### FILES

```yaml
# Exact source hashes are filled after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: permissions
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/routes/system/server-panel/+page.svelte
    role: page
    hash: sha256:05fbde0f2c75a9608a357e7ffba7fa8831a5b149ccc7d7ad3f4f2a8a2ff53eff
    supports: [page-purpose, capability.monitor-dashboard, capability.inspect-memory]
  - path: frontend/routes/system/server-panel/DashboardView.svelte
    role: user-interface
    hash: sha256:0c4fa84a820ef264a3afd5fa9b5cd386a57c6083a2b30cfe93eeb4861e4c8c5f
    supports: [concepts, capability.monitor-dashboard, rules, troubleshooting]
  - path: frontend/routes/system/server-panel/MemoryView.svelte
    role: user-interface
    hash: sha256:9c23910ed8827b5a55327641af0cc07b979215f629bb3120835d33f64bd7efa3
    supports: [concepts, capability.inspect-memory, rules, troubleshooting]
  - path: frontend/routes/system/server-panel/server-metrics.model.ts
    role: business-logic
    hash: sha256:0a58529510d39075ad82a9fb6a0229e924953b9e3f86cf08250809215cd0da03
    supports: [concepts, capability.monitor-dashboard, rules]
  - path: frontend/routes/system/server-panel/server-metrics.svelte.ts
    role: frontend-service
    hash: sha256:93bd8e2c48d286c6b733e45671e93681316de139930302716768c66fb0c5665e
    supports: [capability.monitor-dashboard]
  - path: frontend/routes/system/server-panel/server-metrics.model.test.ts
    role: business-logic
    hash: sha256:88ed2f95eeac11226e82a58a603641f6971f237690b1b49f9a062831a244e079
    supports: [concepts, rules]
  - path: frontend/packages/genix-ui/charts/ChartCanvas.svelte
    role: shared-domain
    hash: sha256:53d785220f015d7130aaa7e12cb362c09e212016ae3f647562d6b85d2269a584
    supports: [capability.monitor-dashboard]
  - path: frontend/packages/genix-ui/vTable/TableStream.svelte
    role: shared-domain
    hash: sha256:5665ed12016ab48599c3acacfc2bc6e945584a729614ff39b6d2df11731e9458
    supports: [capability.inspect-memory]
  - path: backend/config/server_metrics.go
    role: backend-handler
    hash: sha256:ffc779043a8e0c58bbf19895cdab5b3f5cc16650226d60673cd3dfd66f786bd9
    supports: [capability.monitor-dashboard, rules, troubleshooting]
  - path: backend/config/system_memory_packages.go
    role: backend-handler
    hash: sha256:8748f7e70b47ca8fa4fe46897efc1111a8d2c8a8cfc654c7d95ed108dd4c4100
    supports: [capability.inspect-memory, rules, troubleshooting]
  - path: backend/system/memory_packages_report.go
    role: business-logic
    hash: sha256:3564ee1079241b2a3935b69df0a744d55daaa026f30289a4a14ec46487869f64
    supports: [concepts, capability.inspect-memory]
  - path: backend/system/metrics_collector.go
    role: business-logic
    hash: sha256:efedf0e94a3a3b8f18ece2993541f37408a5d9aa482cdd034ee8f5c0086d23f2
    supports: [concepts, capability.inspect-memory, troubleshooting]
  - path: backend/core/types/server_metrics.go
    role: data-model
    hash: sha256:ae6198f7a6de08af14ac348151de0ae0facce0f7b01f0455f40e4725b1ba6980
    supports: [concepts, rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [page-purpose, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: server_utils/PLAN_SERVER_METRICS.md
    role: reference-document
    hash: sha256:7bd6588afcc2f8d51041bee0d25835198cd34849b5d44cc12c477ceddf74a072
    supports: [concepts, rules, troubleshooting]
  - path: server_utils/src/sysmetrics/collector.rs
    role: business-logic
    hash: sha256:12fb9251f99e291297b231acc7627d6ce1e16eb906cf5c3018e549a26a30d684
    supports: [concepts, rules]
  - path: server_utils/src/config.rs
    role: business-logic
    hash: sha256:3de9e11f4f5bfed64079e6ca905b895b33f7d42b256283707b10c6786b766716
    supports: [concepts]
```
