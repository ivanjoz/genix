---
schema: 1
page_id: system.observability
route: /system/observability
title: Observability (Observabilidad)
status: implemented
visibility: saas
description_en: >-
  Backend API monitoring, SaaS only. Review recent activity and failures per route with CPU and
  inference credits, estimated requests, failed requests, and error counts over a four-hour window.
  Filter by method, route path, error text, or source line to find the most repeated errors.
description_es: >-
  Monitoreo de APIs del backend, exclusivo SaaS. Revisar actividad y fallas recientes por ruta con
  créditos CPU e IA, solicitudes estimadas, solicitudes fallidas y errores en una ventana de cuatro
  horas. Filtrar por método, ruta, texto de error o línea de código para hallar errores repetidos.
---

# Observability (Observabilidad)

<!-- DOC-ID: page-purpose -->
## Page purpose

Observability (`Observabilidad`) monitors recent backend API activity and failures across the SaaS
platform. Open **System → Observability (Sistema → Observabilidad)** at `/system/observability`.
This page is visible only to the company that administers the SaaS platform; it is not a tenant-level
request log or the historical company credit report.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Credits (`créditos`)** measure charged API work. GET and POST requests have different minimum
  costs, and payload processing can add more credits.
- **Estimated requests (`solicitudes estimadas`)** are an approximation calculated from CPU
  credits. They are not an exact invocation counter because a request can consume more than the
  minimum credit cost.
- **Failed requests (`solicitudes fallidas`)** count actual logged requests that captured at least
  one error. **Errors (`errores`)** count captured error occurrences, so one failed request may
  contribute several errors.
- **Unmetered (`Sin medición`)** means the route has failures but no compatible credit-based request
  estimate. Public routes, unsupported methods, and failures that were not charged can appear this
  way.

<!-- DOC-ID: capability.monitor-routes -->
## Monitor API routes (Monitorear rutas de API)

Each route card shows its method and path, route number, CPU and inference credits, estimated request
total when available, actual failures, and error occurrences. The chart uses green for estimated
successes and red for actual failed requests. Missing activity in a five-minute slot is shown as
zero, keeping every card aligned to the same four-hour time window.

Cards with more recent CPU usage appear first; failures break ties. Error-only routes remain visible
even when no request estimate can be calculated. The page refreshes every 15 seconds while its
browser tab is visible, stops polling in a background tab, and catches up after the tab becomes
visible again. Use **Refresh (Actualizar)** for an immediate update.

Common questions and vocabulary: `¿qué API está consumiendo más créditos?`, `¿qué endpoint está
fallando?`, `errores por ruta`, `API usage`, `requests por routerID`, `RouteID`, `observabilidad`.

<!-- DOC-ID: capability.find-errors -->
## Find repeated route errors (Buscar errores repetidos)

Use **Route or error (Ruta o error)** to filter cards by HTTP method, route path, error preview, or
source code line. Below a chart, the most frequent error hashes show their occurrence count,
representative text, and code-line reference.

The displayed text is a compact preview, not the authoritative full message or stack trace. A rare
hash collision may show more than one code line under the same count. Use the request logs and the
corresponding request identity in the operational logging system when a complete stack or exact
message is required.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Green values are `estimated requests - actual failed requests`, never below zero. Red values are
  actual failed-request counts, not an estimate derived from credits.
- CPU credits divided by the route method's minimum cost generally produce an upper-bound estimate:
  larger payloads can make one request consume additional credits.
- The current five-minute interval can change after it first appears because credit and error
  writers flush asynchronously. Repeated refreshes replace the interval's absolute totals rather
  than adding them again.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **No cards appear:** no credits or failed requests were recorded in the latest four hours, or the
  platform credit aggregate has not started writing yet.
- **A route says “Sin medición”:** failures exist, but its request count cannot be approximated from
  compatible CPU credits. The failures remain valid.
- **A failure count exceeds the estimated requests:** some failed/public requests are not charged,
  while the credit conversion is only an approximation. The chart preserves the actual failures.
- **An error preview is unavailable:** the referenced descriptor has not been stored or resolved;
  the route's error count is still displayed.
- **Data does not update in a background tab:** polling intentionally pauses while hidden. Return to
  the tab or select **Actualizar**.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **System → Companies (Empresas)** ranks companies by their latest 30 days of CPU or AI credits and
  provides company/day/API detail. Use it for historical tenant comparison.
- **System → Server Panel** monitors host and service CPU, memory, disk, and network behavior. Use it
  for machine health; use **Observability** for API-route credit activity and failures.
- The header credit indicator answers how much the signed-in tenant or user consumed; this page
  aggregates recent backend-route activity for the whole platform.

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
  - path: frontend/routes/system/observability/+page.svelte
    role: page
    hash: sha256:bc92b2f2c6085d29daee0007cf2c5a5669fe1e319c81115f4084087480fa40b4
    supports: [page-purpose, capability.monitor-routes]
  - path: frontend/routes/system/observability/BackendServices.svelte
    role: user-interface
    hash: sha256:f475a92907203da17c5abdc38969c2936d65e084040d3c663420c00470205349
    supports: [capability.monitor-routes, capability.find-errors, troubleshooting]
  - path: frontend/routes/system/observability/observability.model.ts
    role: business-logic
    hash: sha256:6d372e29220e3146cf9164e332d13e3578a3e7fccd04e9506ec232fc4bcc9760
    supports: [concepts, capability.monitor-routes, capability.find-errors, rules]
  - path: frontend/routes/system/observability/observability.svelte.ts
    role: frontend-service
    hash: sha256:9f63d3b6ec71291a0c401973a6ed9da0b5bf04253bf222f00c0d98adfc328c1e
    supports: [capability.monitor-routes, capability.find-errors, troubleshooting]
  - path: backend/config/observability.go
    role: backend-handler
    hash: sha256:3979ac4c811d4ce95b6ec47bc8cbf97b4c7fcbcc7a41f994147920cab1ec2dba
    supports: [concepts, capability.monitor-routes, rules, troubleshooting]
  - path: backend/config/request_errors.go
    role: backend-handler
    hash: sha256:f55ea24bade44ba6585a63d9adaa53c869304ed5decd955620541dcc490b8f93
    supports: [capability.find-errors, troubleshooting]
  - path: backend/core/server_utils/credits.go
    role: business-logic
    hash: sha256:97635d75a45c1a8e0ab1760b137877fb5f52093e50c81cf572ee03299d2452a5
    supports: [concepts, rules]
  - path: backend/core/types/request_errors.go
    role: data-model
    hash: sha256:0d128752869a43e89a8d1ce99d1cd153d02dafc6ff042da865fd15e5145a80c6
    supports: [concepts, capability.find-errors]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [page-purpose]
  - path: server_utils/src/limiter/quota.rs
    role: business-logic
    hash: sha256:5e53b1cb704d348ad6e00297796a8a98c10f7f55e0f78dba20b3acef1ad151f9
    supports: [page-purpose, capability.monitor-routes, rules, troubleshooting]
```
