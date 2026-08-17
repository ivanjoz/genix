---
schema: 1
page_id: system.cron-actions
route: /system/cron-actions
title: Cron Actions (Acciones Cron)
status: implemented
visibility: saas
description_en: >-
  Scheduled task (cron job) execution history, SaaS only. Review each queued or run action's time
  slot, action name, target company, parameters, invocation count, status, and reported messages,
  and reload the list on demand.
description_es: >-
  Historial de ejecución de tareas programadas (cron jobs), exclusivo SaaS. Revisar la franja
  horaria, nombre de acción, empresa, parámetros, número de invocaciones, estado y mensajes de
  cada acción encolada o ejecutada, y recargar la lista bajo demanda.
---

# Cron Actions (Acciones Cron)

<!-- DOC-ID: page-purpose -->
## Page purpose

Cron Actions (`Acciones Cron`) is the read-only viewer for the platform's internal scheduled-task
queue (`cron jobs`). Open **System (Sistema) → Cron Actions (Acciones Cron)** at
`/system/cron-actions`. It lists every scheduled/queued row (`fila cron`) the backend has recorded,
its execution status, how many times it was attempted, and the free-text messages the handler
reported — the same information an operator would otherwise have to find in the server logs.

This page is visible only to the company that administers the SaaS platform; it is not a
tenant-level activity log. It does not create, cancel, retry, or edit any cron row — it only
displays what the scheduler already did, and lets the operator reload the list. It also does not
show HTTP API traffic/errors (that is **Observability**) or host/service health (that is
**Server Panel**).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **cron action (`acción cron`)** is one internal background job the backend schedules for
  itself — for example rebuilding a company's product catalog snapshot
  (`Reconstruir .db de productos`, every 30 minutes), reprocessing the sales summary
  (`Reprocesar Resumen de Ventas`), or removing a company's previous Cloudflare hostname
  (`Borrar dominio anterior en Cloudflare`) after a domain change. These are platform-internal
  maintenance tasks, not something a tenant user schedules from the product.
- A cron action can be **one-shot** (runs once and is done) or **recurring**: a recurring row
  carries its own cadence in minutes, so the scheduler re-enqueues the next occurrence itself
  regardless of whether the previous run succeeded, panicked, or was abandoned.
- **Time Slot (Franja Horaria)** is the 5-minute window (`UnixMinutesFrame`) the row is scheduled
  to run in; the table's Updated column is the last time the row's status changed.
- **Status** is shown as the raw number with its meaning next to it: **Pending (0) / Pendiente
  (0)** still queued or waiting for a retry, **Done (1) / Ejecutada (1)** the handler finished
  without an error, **Abandoned (2) / Abandonada (2)** the row exhausted its retry attempts and
  the scheduler stopped trying.
- **A-ID (Action ID)** identifies which registered handler runs the row; **Action|Acción** is that
  handler's display name. When a row references an ID with no handler registered in the running
  backend, the page shows **"No registrada"** instead of a name.
- **Invocations (Invocaciones)** is how many times the scheduler has attempted this row; under it
  the table also prints the row's own numeric **ID** (composed from the company, the action, and a
  hash of its parameters) so an operator can grep server logs for that same `cron_id`.
- **Parameters (Parámetros)** are up to six generic slots (`p1`…`p6`, four numeric and two text)
  the scheduling code fills for that specific action; the column only prints the slots that were
  actually sent, as `[index]=value`.
- **Messages (Mensajes)** are the free-text lines the handler reported through its own
  `AddMessage` calls during its last attempt, plus a `panic:`/`error:` line when the run failed;
  only the latest attempt's messages are kept, not an accumulated history across retries.

<!-- DOC-ID: capability.monitor-schedule -->
## Review the cron action history (Revisar el historial de acciones cron)

### User intention (Intención del usuario)

Check whether a background job ran, is still pending, or was abandoned, and read the error text a
handler reported, without searching the raw server logs.

### Where to find it (Dónde encontrarlo)

**System (Sistema) → Cron Actions (Acciones Cron)** at `/system/cron-actions`. The page has a
single table listing every row the initial query returns: Time Slot, A-ID, Action, Company,
Parameters, Invocations (with the row ID underneath), Status, Updated, and Messages.

### Required information and prerequisites (Requisitos previos)

None from the user besides being signed in to the SaaS-administering company; the table has no
create/edit form. On first load the page fetches rows updated in roughly the last 7 days; use
**Reload (Recargar)** for the latest state (see below).

### Business rules and rationale (Reglas y razón de negocio)

Use **Route or error**-style local filtering: the filter box (`Filtrar`) matches locally against
the already-loaded rows — action ID, action name, company ID, the formatted parameters, the row
ID, invocation count, the raw status number, and the messages text — not a server-side search.
Rows are sorted by Time Slot, most recent first.

### Result and side effects (Resultado y efectos)

Purely read-only: nothing on this page writes, cancels, or reschedules a cron row. Loading or
reloading the page only reads data; it never changes the scheduler's own state.

### Limitations (Limitaciones)

- There is no action to retry, cancel, or delete a row from here; recovery only happens through
  the scheduler's own automatic retries (see Cross-capability business rules below).
- Company is shown only as its numeric ID (`CompanyID`); the page does not resolve or display the
  company's name.
- The Messages column shows a compact preview of the last attempt only, not a full history of
  every previous attempt's messages.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo veo si una tarea programada (cron) se ejecutó?`
- `¿Por qué dice "No registrada" en la columna Acción?` El ID de acción de esa fila no corresponde
  a ningún manejador registrado en el backend que está corriendo ahora mismo.
- `¿Qué significa el número debajo de las invocaciones?` Es el ID interno de la fila cron, útil
  para buscarla en los logs del servidor.
- Search terms: `cron`, `tareas programadas`, `trabajos en segundo plano`, `acciones cron`,
  `franja horaria`, `estado pendiente`, `abandonada`, `mensajes de error`.

<!-- DOC-ID: capability.reload -->
## Reload the list (Recargar la lista)

### User intention (Intención del usuario)

Pull the latest scheduler state instead of relying on the last snapshot the browser cached.

### Where to find it (Dónde encontrarlo)

**Reload (Recargar)** button (with a refresh icon) at the top-right of the toolbar, next to the
filter box.

### Business rules and rationale (Reglas y razón de negocio)

The list is cached locally for a short time (about 15 seconds) before the page would refresh it on
its own on the next load; clicking **Reload** forces an immediate refresh from the server instead
of waiting on that cache window. There is no automatic background polling on this page — unlike
**Observability**, the table does not refresh itself every few seconds; the user must reload
manually to see newer rows.

### Limitations (Limitaciones)

Reloading replaces the in-memory list; it does not merge or preserve any local sort/filter beyond
what the filter box already applies to the new data.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué no se actualiza solo?` Esta página no hace polling automático; hay que pulsar
  "Recargar".
- Search terms: `recargar`, `actualizar lista`, `refrescar cron`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Rows are scheduled onto 5-minute frames (`UnixMinutesFrame`). A backend ticker (once a minute)
  re-scans a rolling **60-minute lookback window** and retries every row still at Status 0
  (Pending) inside it; a row is marked **Abandoned (2)** once it reaches 10 attempts
  (`cronMaxInvocations`). A row whose scheduled frame falls **outside** that 60-minute window
  without ever completing can remain shown as Pending indefinitely on this page (for example when
  its Action ID has no registered handler) — the list's own ~7-day fetch window is only how far
  back this page reads history, not how long the scheduler keeps retrying a row.
- A **recurring** action's cadence lives on the row itself, so the next occurrence is enqueued
  after every attempt regardless of outcome (success, handler error, or panic); a **one-shot**
  action is not re-enqueued.
- Duplicate scheduling of the same logical action (same company, action, and parameters) within
  the same 5-minute frame is skipped while an existing row for it is still Pending, so re-triggering
  the same job repeatedly does not create parallel duplicate rows in that frame.
- The scheduler takes a best-effort lease before running a row, so two backend processes will not
  normally execute the same action at the same time; the lease alone releases after about a minute
  if a process dies mid-run, without requiring a cleanup step.
- Failures logged while a cron row runs (including a recovered panic) are **not** sent to
  **Observability**'s per-route error catalog: that catalog only drains its error accumulator at
  the end of an HTTP request, and the cron ticker runs on its own background timer outside any
  request. This page's Messages column (and the server's own log stream) is the place to read a
  failed cron action's error text, not Observability.
- Viewing this page's data is gated only by being signed in to the company that administers the
  SaaS platform (company ID 1); there is no separate view/edit access level enforced by the
  backend for the underlying query beyond that company check, so the "Acciones Cron" catalog entry
  only controls whether the menu item and route are shown to a given user.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **A row stays at "Pending (0)" and never changes:** either it is still inside the 60-minute
  retry window and simply has not been picked up yet, or its scheduled frame already fell outside
  that window (commonly because its Action ID has no handler registered in the currently running
  backend, shown as "No registrada"), in which case the scheduler will not retry it again.
- **A row shows "Abandoned (2)":** it reached the maximum of 10 attempts; check its Messages
  column for the last recorded error or panic text.
- **The Messages column is empty on a Done row:** the handler completed without calling
  `AddMessage`; an empty Messages column on a successful run is expected, not an error.
- **A cron failure does not appear in Observability:** expected — cron errors are not routed into
  the per-route error catalog that page reads; rely on this page's Messages or the server log
  stream instead.
- **The list looks stale:** this page does not auto-refresh; press **Reload**.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **System → Observability (Observabilidad)** at `/system/observability`: monitors HTTP API
  traffic and failures per route. It does not cover cron job execution — use this page instead for
  scheduled-task history and handler messages.
- **System → Server Panel** monitors host/service CPU, memory, disk, and network; use it for
  machine health, not for whether a specific scheduled job ran.
- **System → Companies (Empresas)**: another SaaS-only System page, for tenant/credit comparison
  rather than scheduler state.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: permissions
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages, rules]
  - path: frontend/routes/system/cron-actions/+page.svelte
    role: page
    hash: sha256:fb71efcbb0b6a629695c370855aba4d51b7a1a0936822f27bef4c6f0877309bb
    supports: [page-purpose, concepts, capability.monitor-schedule, capability.reload]
  - path: frontend/routes/system/cron-actions/cron-actions.svelte.ts
    role: frontend-service
    hash: sha256:704af17dfdabb6e6aa7d4a45d890e265d2d2aaf68851ae1aaf5dfea10b04343b
    supports: [concepts, capability.monitor-schedule, capability.reload, rules]
  - path: backend/config/cron-actions-scheduled.go
    role: backend-handler
    hash: sha256:5ba12d0701bfbf72d3fe5db8c1e2b1c09919ee1e3383dbc30b16c39f875b8c5a
    supports: [capability.monitor-schedule, rules]
  - path: backend/core/cron-action.go
    role: data-model
    hash: sha256:487e1d9ef89ba728dd576aeef9e622fb7b78f395677cb14724fb1a860d7b6554
    supports: [concepts, rules]
  - path: backend/core/cron-action-scheduler.go
    role: business-logic
    hash: sha256:5ee6d703cd395c52dec267afab350957d30b5ff7f4d91768d7110da8ac15021a
    supports: [concepts, rules, troubleshooting]
  - path: backend/core/types.go
    role: data-model
    hash: sha256:79a3db836d91673e4dc1bd8753daf7597c3b4151c243c18f296ef27552d8a61a
    supports: [concepts]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [page-purpose, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/core/request_errors.go
    role: business-logic
    hash: sha256:ca6216e2ecf05a12a055f1cb8412907efd7b500d292b0b0c234ea499fa556a30
    supports: [rules, troubleshooting]
  - path: backend/business/product-ecommerce-cron.go
    role: business-logic
    hash: sha256:334033ba9b575b9498e06e20573d61e2b0516f1437a887aa0b571817c279838d
    supports: [concepts]
  - path: backend/webpage/domain_cleanup_cron.go
    role: business-logic
    hash: sha256:e5febead164e6f945a0d9fd07e7dd76d4543b0480d569e74817759b5e65ed4ed
    supports: [concepts]
  - path: backend/sales/main.go
    role: business-logic
    hash: sha256:b1f47217c39ad2e920ed7916a7a2e104af3ebf3610743c69d58733ef097e8d90
    supports: [concepts]
```
