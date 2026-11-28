---
schema: 1
page_id: crm.reports-conversion
route: /crm/reports/conversion
title: Conversion (Conversión)
status: implemented
visibility: tenant
description_en: >-
  CRM conversion report per seller over a range of months. Net progress: opportunities that reached
  each stage, won (split by the stage they were won from) and lost, net of steps back, losses and
  hand-overs, with a day-by-day detail. Funnel: of the opportunities that entered the pipeline in the
  range, how far they got, the conversion from each stage, and how many days they stayed in each
  stage.
description_es: >-
  Reporte de conversión del CRM por vendedor en un rango de meses. Avance neto: oportunidades que
  llegaron a cada etapa, ganadas (según la etapa desde la que se ganaron) y perdidas, netas de
  retrocesos, pérdidas y reasignaciones, con detalle día a día. Embudo: de las oportunidades que
  entraron al pipeline en el rango, hasta dónde llegaron, la conversión desde cada etapa y cuántos
  días estuvieron en cada etapa.
---

# Conversion (Conversión)

<!-- DOC-ID: page-purpose -->
## Page purpose

Conversion (`Conversión CRM`) measures how far opportunities advance through the stages (`etapas`)
of a CRM pipeline. It has two views: **Net progress (Avance neto)**, the ground each seller gained
stage by stage, and **Funnel (Embudo)**, the sales funnel (`embudo de ventas`) of the opportunities
that entered in the range with the time they spent in each stage.

It is a read-only report. Opportunities are moved in **Opportunities (Oportunidades)**. The open
pipeline day by day is in **Flow (Flujo)**; win rate, won amount, goals and lost reasons per seller
are in **Performance (Desempeño)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Stage position (posición de la etapa):** the depth of a stage in the board, 1 for the first
  column. Columns are positions labelled with the name of the stage that sits at that depth today;
  each movement keeps the position it had when it happened, so a later reorder of the stages does
  not rewrite history.
- **Reached (alcanzó):** an opportunity at stage k has reached every stage from 1 to k.
- **Seller (vendedor):** the opportunity's **Owner (Responsable)**. Movements count for the owner,
  not for the user who made the change.
- **Show amounts (Ver montos)** switches the figures between number of opportunities and amount.

<!-- DOC-ID: rules -->
## Report filters and scope (Filtros y alcance del reporte)

- **Where:** **CRM → Reports (Reportes) → Conversion (Conversión)** at `/crm/reports/conversion`.
  The view strip on the left switches between **Net progress (Avance neto)** and **Funnel
  (Embudo)**.
- **Pipeline:** only active pipelines where the user is a member at any level, or every active
  pipeline with the **CRM Reporte completo** access. Without any, the page says "No tiene pipelines
  para reportar. Pida a un administrador que lo agregue a un pipeline o el acceso de reporte
  completo."
- **From (Desde)** and **To (Hasta):** months. From goes back to January five years ago; To reaches
  twelve months ahead. The range opens on the whole current year, and From can never pass To.
- **Filter by seller (Filtrar por vendedor):** avatars of everyone who owned an opportunity of the
  pipeline in the loaded history; select some to show only them.
- The pipeline, range and sellers chosen are kept when moving between Flow, Conversion and
  Performance.
- The figures are rebuilt from each opportunity's change history (created, moved, won, lost,
  reopened, amount or owner changed).

<!-- DOC-ID: capability.net-progress -->
## Net progress by seller (Avance neto por vendedor)

### User intention (Intención del usuario)

See how many opportunities each seller pushed to each stage, to won and to lost in the range,
without counting twice a deal that went back and forth.

### Where to find it (Dónde encontrarlo)

Conversion page → view **Net progress (Avance neto)**.

### Business rules and rationale (Reglas y razón de negocio)

- The table has one row per **Seller (Vendedor)** and a **Total** row. Its columns are every stage,
  then **Won (Ganadas)** and **Lost (Perdidas)**.
- An opportunity reaching stage k adds 1 to every stage column up to k. A won opportunity also adds
  to **Won (Ganadas)**, which is split into one subcolumn per stage it was won from, plus **Total**.
  A lost opportunity counts only in **Lost (Perdidas)**.
- Every step back, loss or hand-over subtracts what the opportunity had reached from the previous
  owner. The result is net: a deal that goes back and forth counts only its net progress, and
  negative values (red) mean the seller lost ground.
- Won and Lost cells carry a small bar to compare sellers at a glance.
- Clicking a seller opens **Day by day (Día a día)** with only the days that had a net movement for
  that seller.
- When the range spans more than one month, a bar chart shows each stage and Won month by month.

### Limitations (Limitaciones)

- A range without movements shows "No hay movimientos en los meses elegidos."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuántas oportunidades avanzó cada vendedor a la etapa de propuesta?
- ¿Por qué hay números negativos? → the seller lost ground: deals went back, were lost or were
  handed over.
- ¿Desde qué etapa se ganan las oportunidades? → the Won subcolumns.

<!-- DOC-ID: capability.funnel -->
## Funnel and time in stage (Embudo y tiempo en etapa)

### User intention (Intención del usuario)

Follow the opportunities that entered the pipeline in the range: where they leak out, what share is
won, and how long they stay in each stage (`tiempo en etapa`, `ciclo`).

### Where to find it (Dónde encontrarlo)

Conversion page → view **Funnel (Embudo)**.

### Business rules and rationale (Reglas y razón de negocio)

- The cohort is every opportunity that entered the pipeline in the range: created in it, or moved in
  from another pipeline. The seller filter applies to whoever owned it when it entered.
- Totals: **Entered (Ingresaron)**, **Won (Ganadas)** and **Lost (Perdidas)** with their share of
  entered, and **Still open or moved out (Abiertas o transferidas)**. Won and lost read each
  opportunity's state today.
- Each funnel bar counts the opportunities that reached that stage or a deeper one, with its share
  of entered and, in parentheses, the conversion from the stage above, which shows where the funnel
  leaks. A lost opportunity counts up to the stage where it was lost.
- The table gives per stage **Finished stays (Estadías terminadas)**, **Median days (Mediana
  días)**, **Average days (Promedio días)** and **Longest (Máximo)**. Only finished stays count: an
  opportunity still sitting in a stage today is left out, so a young cohort does not skew the
  durations.

### Limitations (Limitaciones)

- When no opportunity entered in the range, the view shows "Ninguna oportunidad entró al pipeline
  en los meses elegidos."
- **Show amounts (Ver montos)** does not change this view: it always counts opportunities and days.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuál es mi tasa de conversión entre etapas? ¿En qué etapa se pierden más oportunidades?
- ¿Cuántos días se quedan las oportunidades en cada etapa? ¿Dónde se estanca el pipeline?
- Embudo de ventas, funnel, conversión, cuello de botella, tiempo de ciclo.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No tiene pipelines para reportar…":** ask for membership in the pipeline in **Pipelines: Access
  (Flujos: Accesos)**, or for the **CRM Reporte completo** access.
- **A stage column shows a different name than before:** columns follow positions; they are labelled
  with the stage at that position today.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Flow (Flujo)**: open opportunities per stage day by day, and the pipeline waterfall.
- **Performance (Desempeño)**: win rate, won amount, sales cycle and goal per seller, and lost
  reasons.
- **Opportunities (Oportunidades)**: the board where the moves counted here happen.
- **Pipelines (Flujos)**: the stages and their order.

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
  - path: frontend/routes/crm/reports/conversion/+page.svelte
    role: page
    hash: sha256:b3f348ccd0ac27e4e8c7bfe4f9c157bb8e9703ee4221e2240c55dbec365ad0dd
    supports: [page-purpose, capability.net-progress, capability.funnel]
  - path: frontend/routes/crm/reports/CrmReportPage.svelte
    role: user-interface
    hash: sha256:e9db1aece648f03ccd670b76c1954aa7f5c3405c73189c7aab672e9dbc9af8dd
    supports: [concepts, rules, troubleshooting]
  - path: frontend/routes/crm/reports/ReportNetProgress.svelte
    role: user-interface
    hash: sha256:a7e46b11b4a6ebb202cb85dab54da5341a5044243d269736c69333bf9bfffa92
    supports: [capability.net-progress]
  - path: frontend/routes/crm/reports/ReportFunnel.svelte
    role: user-interface
    hash: sha256:5288afdde394dd8ede112fd4bc084bc86bf590fe1d9ecbc8c923831dd48e1113
    supports: [concepts, capability.funnel, troubleshooting]
  - path: frontend/routes/crm/reports/reports.svelte.ts
    role: frontend-service
    hash: sha256:2370c90ccd1c87242bd8cc8dbea755e6a3264c478f01204a217b976083ea6b51
    supports: [rules]
  - path: frontend/routes/crm/reports/reports.ts
    role: business-logic
    hash: sha256:d8f9bdd066fab48a269cc3e1e2734512d0db083c6989d4d3928f0045863780b8
    supports: [concepts, rules, capability.net-progress, capability.funnel]
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
