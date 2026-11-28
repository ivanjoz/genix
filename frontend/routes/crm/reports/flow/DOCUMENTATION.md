---
schema: 1
page_id: crm.reports-flow
route: /crm/reports/flow
title: Flow (Flujo)
status: implemented
visibility: tenant
description_en: >-
  CRM pipeline flow report per seller over a range of months. Daily state: open opportunities in
  each stage at the end of each day, by count or amount. Waterfall: how the open pipeline went from
  the opening to the closing of the range through new, increased, decreased, won, lost, reopened and
  transferred opportunities, by month and by seller.
description_es: >-
  Reporte de flujo del pipeline del CRM por vendedor en un rango de meses. Estado diario:
  oportunidades abiertas en cada etapa al cierre de cada día, en cantidad o monto. Cascada: cómo
  pasó el pipeline abierto de la apertura al cierre del rango con oportunidades nuevas, aumentos,
  reducciones, ganadas, perdidas, reabiertas y transferidas, por mes y por vendedor.
---

# Flow (Flujo)

<!-- DOC-ID: page-purpose -->
## Page purpose

Flow (`Flujo CRM`) shows how the open sales pipeline (`pipeline abierto`, `cartera de
oportunidades`) of a CRM pipeline changed over time. It has two views: **Daily state (Estado
diario)**, what sat in each stage at the end of every day, and **Waterfall (Cascada)**, what made
the open pipeline grow or shrink between the start and the end of the range.

It is a read-only report. Opportunities are moved, won and lost in **Opportunities
(Oportunidades)**. Conversion between stages is in **Conversion (Conversión)**, and each seller's
results and lost reasons in **Performance (Desempeño)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Open opportunity (oportunidad abierta):** neither won (`ganada`) nor lost (`perdida`).
- **Seller (vendedor):** the opportunity's **Owner (Responsable)**. Every movement is counted for the
  owner, not for the user who made the change. A hand-over (`reasignación`) moves the opportunity
  from one seller to another.
- **Count or amount:** the **Show amounts (Ver montos)** checkbox switches every figure between the
  number of opportunities and their amount.
- **Transferred (Transferidas):** an open opportunity that changed seller, or that moved from or to
  another pipeline.

<!-- DOC-ID: rules -->
## Report filters and scope (Filtros y alcance del reporte)

- **Where:** **CRM → Reports (Reportes) → Flow (Flujo)** at `/crm/reports/flow`. The view strip on
  the left switches between **Daily state (Estado diario)** and **Waterfall (Cascada)**.
- **Pipeline:** only active pipelines where the user is a member at any level, or every active
  pipeline with the **CRM Reporte completo** access. Without any, the page says "No tiene pipelines
  para reportar. Pida a un administrador que lo agregue a un pipeline o el acceso de reporte
  completo."
- **From (Desde)** and **To (Hasta):** months. From goes back to January five years ago; To reaches
  twelve months ahead. The range opens on the whole current year, and From can never pass To.
- **Filter by seller (Filtrar por vendedor):** avatars of everyone who owned an opportunity of the
  pipeline in the loaded history; select some to show only them. With none selected, all sellers
  count.
- The pipeline, range and sellers chosen are kept when moving between Flow, Conversion and
  Performance.
- The figures are rebuilt from each opportunity's change history (created, moved, won, lost,
  reopened, amount or owner changed) and today's open opportunities, so they always reflect the
  current history.

<!-- DOC-ID: capability.daily-state -->
## See the daily state of the pipeline (Estado diario)

### User intention (Intención del usuario)

Know how many opportunities, or how much money, sat open in each stage (`etapa`) on each day, to see
whether the pipeline is filling up or emptying.

### Where to find it (Dónde encontrarlo)

Flow page → view **Daily state (Estado diario)**.

### Business rules and rationale (Reglas y razón de negocio)

- A stacked bar chart shows one bar per day, from the first day of From to the last day of To, never
  after today. Each colour is a stage of the pipeline.
- Each bar is a snapshot at the end of the day, so an opportunity that moves back and forth is
  never counted twice.
- The table lists each stage with **At the start (Al inicio)**, **At the end (Al final)** and
  **Change (Variación)**; a drop shows in red.

### Limitations (Limitaciones)

- Only open opportunities count: won and lost ones leave the chart the day they close.
- A range entirely in the future shows "Los meses elegidos están en el futuro."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuántas oportunidades tenía en cada etapa el día 15? ¿Está creciendo mi pipeline?
- ¿Cuánto dinero hay en negociación al final del mes? → tick Show amounts (Ver montos).

<!-- DOC-ID: capability.waterfall -->
## See how the open pipeline changed (Cascada del pipeline)

### User intention (Intención del usuario)

Explain why the open pipeline at the end of the range is bigger or smaller than at the start: how
much came in, how much was won or lost, and how much was handed over.

### Where to find it (Dónde encontrarlo)

Flow page → view **Waterfall (Cascada)**.

### Business rules and rationale (Reglas y razón de negocio)

- The bars go from **Opening (Apertura)** to **Closing (Cierre)** through the steps **New
  (Nuevas)**, **Increased (Aumentos)**, **Decreased (Reducciones)**, **Won (Ganadas)**, **Lost
  (Perdidas)**, **Reopened (Reabiertas)**, **Transferred in (Transferidas (entrada))** and
  **Transferred out (Transferidas (salida))**. Won is green and Lost red.
- Opening + every step = Closing, always: Genix computes each month's opening from its closing.
- **Increased (Aumentos)** and **Decreased (Reducciones)** are changes of amount of an open
  opportunity (`renegociación`). They only appear with **Show amounts (Ver montos)** ticked, because
  they do not change the count.
- A table repeats the steps for each **Month (Mes)** of the range, and another for each **Seller
  (Vendedor)** who had an open opportunity or a movement. Negative values show in red.

### Limitations (Limitaciones)

- Months after today are not shown.
- A range entirely in the future shows "Los meses elegidos están en el futuro."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué bajó mi pipeline este trimestre? ¿Cuántas oportunidades nuevas entraron?
- ¿Cuánto se reasignó a otros vendedores? ¿Qué es apertura y cierre?
- Cascada, waterfall, puente de pipeline, variación de cartera.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No tiene pipelines para reportar…":** ask for membership in the pipeline in **Pipelines: Access
  (Flujos: Accesos)**, or for the **CRM Reporte completo** access.
- **A seller has no avatar in the filter:** they did not own any opportunity of the pipeline in the
  loaded history.
- **Increased and Decreased are missing:** tick **Show amounts (Ver montos)**.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Conversion (Conversión)**: how far opportunities get through the stages, and how long they stay.
- **Performance (Desempeño)**: win rate, won amount, goal per seller, and lost reasons.
- **Opportunities (Oportunidades)**: the board where the movements counted here happen.
- **Pipelines (Flujos)**: the stages shown as colours and rows.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [rules]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, related-pages]
  - path: frontend/routes/crm/reports/flow/+page.svelte
    role: page
    hash: sha256:c12181ee6f22e37f92d932f35d4142658c55ac85d3c50c56f6a02390bd091435
    supports: [page-purpose, capability.daily-state, capability.waterfall]
  - path: frontend/routes/crm/reports/CrmReportPage.svelte
    role: user-interface
    hash: sha256:e9db1aece648f03ccd670b76c1954aa7f5c3405c73189c7aab672e9dbc9af8dd
    supports: [concepts, rules, troubleshooting]
  - path: frontend/routes/crm/reports/ReportDailyState.svelte
    role: user-interface
    hash: sha256:414e7d64cbd6879c49fe0b5672be0d9d377ec26de4abd6812816d58bb5a09b6f
    supports: [capability.daily-state]
  - path: frontend/routes/crm/reports/ReportWaterfall.svelte
    role: user-interface
    hash: sha256:f71f79002674c911084ae8420c58d3d2b54b13d55724916ffb896f70c1cc6805
    supports: [concepts, capability.waterfall, troubleshooting]
  - path: frontend/routes/crm/reports/reports.svelte.ts
    role: frontend-service
    hash: sha256:2370c90ccd1c87242bd8cc8dbea755e6a3264c478f01204a217b976083ea6b51
    supports: [rules]
  - path: frontend/routes/crm/reports/reports.ts
    role: business-logic
    hash: sha256:d8f9bdd066fab48a269cc3e1e2734512d0db083c6989d4d3928f0045863780b8
    supports: [concepts, rules, capability.daily-state, capability.waterfall]
  - path: frontend/services/crm/crm-reports.svelte.ts
    role: frontend-service
    hash: sha256:8e1f224860a2b2e1810fab6030d3cb8803d9044ca0f6cbaaec2de4ccd9161006
    supports: [rules]
  - path: backend/crm/report.go
    role: backend-handler
    hash: sha256:49358ff5d81bf3b29aea73be2dc8c9a6264b17bd79bbb2f13eb3c0921d6035e0
    supports: [rules]
  - path: backend/crm/types/report_rules.go
    role: permissions
    hash: sha256:39dff4dbbaefd5888caee45a3520401a1510e3b68f673acf24084b63129aa16d
    supports: [rules, troubleshooting]
  - path: backend/crm/types/stage_log.go
    role: data-model
    hash: sha256:f07b30e552aa3714b3cb1bf5a2bb6dd1c1f9b61d3691ba14154e145eabb030d2
    supports: [concepts, rules]
  - path: frontend/routes/crm/pipeline/OpportunityLayer.svelte
    role: user-interface
    hash: sha256:1bf5b6bddb85bea09ad026b7a71b19cb2926bb184230d200c0a4589f2c7ba982
    supports: [concepts]
```
