---
schema: 1
page_id: crm.timeline
route: /crm/timeline
title: Timeline (Línea de tiempo)
status: implemented
visibility: tenant
description_en: >-
  Gantt chart of the opportunities of one CRM pipeline, grouped by seller: the stages each opportunity
  went through day by day and how it ended (won, lost, moved to another pipeline or still open), with
  per-seller totals. Filter by date range, by start or last change, and by seller.
description_es: >-
  Gantt de las oportunidades de un pipeline del CRM por vendedor: las etapas por las que pasó cada una
  día a día y cómo terminó (ganada, perdida, pasó a otro pipeline o sigue abierta), con totales por
  vendedor. Filtros por rango de fechas, por inicio o último cambio, y por vendedor.
---

# Timeline (Línea de tiempo)

<!-- DOC-ID: page-purpose -->
## Page purpose

Timeline (`Línea de tiempo`, page title **CRM Timeline (Línea de tiempo CRM)**) is a read-only CRM
report drawn as a Gantt chart (`diagrama de Gantt`, `cronograma`). Each bar is one opportunity in one
pipeline: it starts the day the opportunity entered the pipeline and shows, with the stage colors,
each stage it entered and when, and how it ended. Bars are grouped by seller (`vendedor`,
`responsable`).

It answers how long deals take in each stage and which ones are left hanging. The page does not
change opportunities: move, close or edit them on **Opportunities (Oportunidades)**. Amounts are not
drawn here; amount-based reports are **Flow (Flujo)**, **Conversion (Conversión)**, **Performance
(Desempeño)** and **Forecast**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **bar** (`tarjeta`) is the path of one opportunity through one pipeline. An opportunity that
  moved to another pipeline has a bar in each; in the pipeline it left, its bar ends as **moved to
  another pipeline (pasó a otro pipeline)**. If it comes back, the same bar continues.
- Each bar ends with one mark: **won (ganada)** (star), **lost (perdida)** (cross), **moved to
  another pipeline (pasó a otro pipeline)**, or **still open (sigue abierta)**, whose bar runs to today.
- The bar belongs to the opportunity's current owner, even if another seller had it before.
- Only stage entries and closings are drawn; changes of amount, owner or expected close are not.

<!-- DOC-ID: capability.view-timeline -->
## View the opportunities timeline (Ver la línea de tiempo)

### User intention (Intención del usuario)

See, per seller, how each opportunity advanced through the stages in a period and how it ended.

### Where to find it (Dónde encontrarlo)

**CRM → Reports (Reportes) → Timeline (Línea de tiempo)** at `/crm/timeline`. Filters: **Pipeline**,
**Show (Mostrar)** with **Started in the range (Iniciadas en el rango)** or **Last change in the range
(Último cambio en el rango)**, **From (Desde)** and **To (Hasta)**. Below, **Filter by seller (Filtrar
por vendedor)** and the legend of the end marks. Hover a bar to see each stage it entered with its
date and how it closed.

### Business rules and rationale (Reglas y razón de negocio)

- The range defaults to the last 45 days, today included, and can be at most one year; it must end
  after it starts (**Elija un rango de hasta un año, que termine después de empezar.**).
- **Started in the range** keeps bars whose start day is inside the range; **Last change in the
  range** keeps bars whose last change is inside it.
- An open opportunity that started on or before the range's end is always shown, even with no recent
  change, so stalled deals stay visible.
- Each seller's row shows the counts of won, lost, still open and, when any, moved to another
  pipeline.

### Limitations (Limitaciones)

Read-only: clicking a bar does not open the opportunity. Amounts are not shown.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuánto tiempo pasa una oportunidad en cada etapa? ¿Qué oportunidades siguen abiertas sin avanzar?
- ¿Cuántas oportunidades ganó o perdió cada vendedor este mes? Gantt de ventas, ciclo de venta.

<!-- DOC-ID: rules -->
## Who can see which pipelines (Qué pipelines puede ver cada usuario)

The **Pipeline** selector lists the active pipelines where the user is a member at any level (View,
Edit or Full). A user holding the **CRM Reporte completo** access sees every active pipeline, even
without membership. A deleted pipeline is no longer offered. Without any, the page shows **You have
no pipelines to report on. (No tiene pipelines para reportar.)**

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No tiene pipelines para reportar.":** ask for membership in Pipelines: Access (Flujos: Accesos)
  or for the **CRM Reporte completo** access in Users & Profiles (Usuarios & Perfiles).
- **"No hay oportunidades en el rango elegido.":** widen the range, switch **Show (Mostrar)**, or
  clear the seller filter.
- **A bar appears under an unexpected seller:** the bar follows the opportunity's current owner.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Opportunities (Oportunidades)** is where the moves and closings drawn here happen.
- **Flow (Flujo)**, **Conversion (Conversión)**, **Performance (Desempeño)** and **Forecast** are the
  other CRM reports, with amounts and rates.
- **Pipelines (Flujos)** defines the stages and their colors.
- **Pipelines: Access (Flujos: Accesos)** gives the membership that opens a pipeline in this report.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.view-timeline]
  - path: frontend/routes/crm/timeline/+page.svelte
    role: page
    hash: sha256:d99abbf392d8a9ecfb03f4dfdc143494a4f86bea3236a647dcfe1635ba2e317b
    supports: [page-purpose, concepts, capability.view-timeline, rules, troubleshooting]
  - path: frontend/routes/crm/timeline/timeline.svelte.ts
    role: frontend-service
    hash: sha256:7b1faab379ca609f5011e0832a9cb64efc1e268c6d1f156c6d438abc0dee10bc
    supports: [capability.view-timeline, rules]
  - path: frontend/routes/crm/timeline/timeline.ts
    role: business-logic
    hash: sha256:53fc20c934e7c5490660f8c3783843a3f585d3bae09690406196e84c2d8161ab
    supports: [concepts, capability.view-timeline, troubleshooting]
  - path: frontend/routes/crm/reports/reports.ts
    role: business-logic
    hash: sha256:d8f9bdd066fab48a269cc3e1e2734512d0db083c6989d4d3928f0045863780b8
    supports: [rules]
  - path: backend/crm/pipeline_opportunity_status.go
    role: backend-handler
    hash: sha256:8524a3fcfc19d823cf27f05837a7e909ffbb0a28c8dd93cc8bba49d7f7a279e4
    supports: [concepts, rules]
  - path: backend/crm/types/pipeline_opportunity_status.go
    role: data-model
    hash: sha256:1320b645257c8001ce80fd4225094fa90ddd631b31cbcfd7e31f3b98a34440c0
    supports: [concepts]
  - path: backend/crm/types/pipeline_opportunity_status_rules.go
    role: business-logic
    hash: sha256:5cb3bd7f02f4143a395534352f28eb4f541b471875fa8ba7894805a62c42b80f
    supports: [concepts]
  - path: backend/crm/types/report_rules.go
    role: business-logic
    hash: sha256:39dff4dbbaefd5888caee45a3520401a1510e3b68f673acf24084b63129aa16d
    supports: [rules]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [rules, troubleshooting]
```
