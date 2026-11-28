---
schema: 1
page_id: crm.reports-performance
route: /crm/reports/performance
title: Performance (Desempeño)
status: implemented
visibility: tenant
description_en: >-
  CRM seller performance over a range of months. Sellers: won and lost opportunities, win rate, won
  amount, average won, sales cycle days, open today, velocity per day, activities, goal and % of
  goal, with team subtotals. Lost reasons: opportunities still lost at the end of the range by
  reason (Pareto), by seller and by the stage where they were lost.
description_es: >-
  Desempeño de los vendedores del CRM en un rango de meses. Vendedores: oportunidades ganadas y
  perdidas, tasa de cierre, monto ganado, promedio ganado, días de ciclo, abiertas hoy, velocidad por
  día, actividades, meta y % de meta, con subtotales por equipo. Motivos de pérdida: oportunidades
  que siguen perdidas al final del rango por motivo (Pareto), por vendedor y por etapa donde se
  perdieron.
---

# Performance (Desempeño)

<!-- DOC-ID: page-purpose -->
## Page purpose

Performance (`Desempeño CRM`) compares how each seller (`vendedor`) of a CRM pipeline did in a range
of months, and why opportunities were lost. It has two views: **Sellers (Vendedores)**, a scorecard
(`ranking de vendedores`) with team subtotals, and **Lost reasons (Motivos de pérdida)**.

It is a read-only report. Goals are typed in **Goals (Metas)**, teams are built in **Teams
(Equipos)**, and lost reasons are maintained in **Dictionaries (Diccionarios)**. Stage-by-stage
conversion is in **Conversion (Conversión)** and the open pipeline over time in **Flow (Flujo)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Seller (vendedor):** the opportunity's **Owner (Responsable)**. Wins and losses count for the
  owner, not for the user who made the change.
- **Win rate (tasa de cierre):** won ÷ (won + lost).
- **Sales cycle (ciclo de venta):** days from the creation of an opportunity until it was won.
- **Velocity (velocidad):** the expected won amount per day: open today × win rate × average won ÷
  cycle days.
- **Lost reason (motivo de pérdida):** the **Reason (Motivo)** chosen in **Opportunities
  (Oportunidades)** when an opportunity is closed as lost.

<!-- DOC-ID: rules -->
## Report filters and scope (Filtros y alcance del reporte)

- **Where:** **CRM → Reports (Reportes) → Performance (Desempeño)** at
  `/crm/reports/performance`. The view strip on the left switches between **Sellers
  (Vendedores)** and **Lost reasons (Motivos de pérdida)**.
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

<!-- DOC-ID: capability.seller-scorecard -->
## Compare the sellers (Desempeño por vendedor)

### User intention (Intención del usuario)

Rank the sellers by what they won, see their win rate, cycle and activity, and check them against
their goal.

### Where to find it (Dónde encontrarlo)

Performance page → view **Sellers (Vendedores)**.

### Business rules and rationale (Reglas y razón de negocio)

The table has one row per seller, grouped by team: each bold team row is followed by its members,
then come the sellers without a team. Columns:

- **Won (Ganadas)** and **Lost (Perdidas):** closed in the range, net of reopenings.
- **Win rate (Tasa de cierre):** won ÷ (won + lost).
- **Won amount (Monto ganado)** and **Avg. won (Prom. ganado).**
- **Cycle days (Días de ciclo):** average days from creation to won, only for opportunities created
  since January of the first selected year; a dash when there are none.
- **Open today (Abiertas hoy):** today's open opportunities, whatever the range.
- **Velocity / day (Velocidad / día):** open today × win rate × average won ÷ cycle days.
- **Activities (Actividades):** activity entries written in the pipeline's opportunities in the
  range, counted for whoever wrote them.
- **Goal (Meta)** and **% goal (% meta):** the **Won amount (Monto ganado)** goals of the goal
  periods that start inside the range, and won amount ÷ goal; red below 100 %. A dash when there is
  no goal.

A team row adds its members' figures and computes its own win rate, average, cycle and velocity
from those sums; it never averages its members' rates. The formulas are repeated under the table.

### Limitations (Limitaciones)

- **Show amounts (Ver montos)** does not change this view: it always shows both counts and amounts.
- Only the Won amount goal is compared; goals of other metrics are not shown.
- A range without movements shows "No hay vendedores con movimientos en los meses elegidos."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué vendedor vendió más este año? ¿Cuál es la tasa de cierre de cada vendedor?
- ¿Cuántos días demora en cerrarse una venta? ¿Quién está cumpliendo su meta?
- Ranking de vendedores, efectividad, tasa de conversión, win rate, cumplimiento de cuota.

<!-- DOC-ID: capability.lost-reasons -->
## Analyse why opportunities are lost (Motivos de pérdida)

### User intention (Intención del usuario)

Find the main reasons the pipeline loses deals, which sellers lose more, and at which stage.

### Where to find it (Dónde encontrarlo)

Performance page → view **Lost reasons (Motivos de pérdida)**.

### Business rules and rationale (Reglas y razón de negocio)

- An opportunity counts once when its last state in the range is lost in this pipeline. One lost
  and then reopened inside the range does not count.
- A bar chart shows the lost opportunities per reason. The table **Reason (Motivo)** lists, largest
  first, **Deals (Oportunidades)**, **Amount (Monto)**, **% of lost (% de perdidas)** and
  **Cumulative (Acumulado)**, a Pareto reading of the few reasons that explain most losses.
  Opportunities closed without a reason show as **No reason (Sin motivo)**.
- Two more tables split the same opportunities by **Seller (Vendedor)** and by **Lost at stage
  (Etapa donde se perdió)**; the stages follow the board order.

### Limitations (Limitaciones)

- The reason names come from the **Lost reasons (Motivos de pérdida)** list of Dictionaries
  (Diccionarios); this page does not edit them.
- A range without losses shows "No hay oportunidades perdidas en los meses elegidos."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué perdemos ventas? ¿Cuál es el principal motivo de pérdida?
- ¿En qué etapa se caen más oportunidades? ¿Qué vendedor pierde más?
- Motivo de pérdida, razón de rechazo, oportunidades perdidas, Pareto.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No tiene pipelines para reportar…":** ask for membership in the pipeline in **Pipelines: Access
  (Flujos: Accesos)**, or for the **CRM Reporte completo** access.
- **Goal shows a dash:** no Won amount goal starts inside the range for that seller; set it in Goals
  (Metas).
- **Cycle days shows a dash:** none of the seller's wins in the range was created since the first
  selected year; widen From.
- **No team rows:** the sellers are not in any team (Teams (Equipos)).

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Goals (Metas)**: the Won amount goals compared in **% goal (% meta)**.
- **Teams (Equipos)**: the team subtotal rows.
- **Dictionaries (Diccionarios)**: the list of lost reasons.
- **Forecast**: the goal of the current period against what is won and expected to close.
- **Flow (Flujo)** and **Conversion (Conversión)**: the other CRM reports.

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
  - path: frontend/routes/crm/reports/performance/+page.svelte
    role: page
    hash: sha256:0cf317994b40edbef9dbfb52e6347dfe71e55b06f4e43e1f51c6d5fe38708882
    supports: [page-purpose, capability.seller-scorecard, capability.lost-reasons]
  - path: frontend/routes/crm/reports/CrmReportPage.svelte
    role: user-interface
    hash: sha256:e9db1aece648f03ccd670b76c1954aa7f5c3405c73189c7aab672e9dbc9af8dd
    supports: [rules, troubleshooting]
  - path: frontend/routes/crm/reports/ReportScorecard.svelte
    role: user-interface
    hash: sha256:c6b5929a5c7a9645eca696ebfd668ae8e7a8d36c0dbdf77d22ed5b0ecac416a9
    supports: [concepts, capability.seller-scorecard, troubleshooting]
  - path: frontend/routes/crm/reports/ReportLostReasons.svelte
    role: user-interface
    hash: sha256:dd94ddeb30c1d933c28c830de7da3fcb60d9c7eccc7d0f4913ead804bfb96d0e
    supports: [capability.lost-reasons]
  - path: frontend/routes/crm/reports/reports.svelte.ts
    role: frontend-service
    hash: sha256:2370c90ccd1c87242bd8cc8dbea755e6a3264c478f01204a217b976083ea6b51
    supports: [rules, capability.seller-scorecard]
  - path: frontend/routes/crm/reports/reports.ts
    role: business-logic
    hash: sha256:d8f9bdd066fab48a269cc3e1e2734512d0db083c6989d4d3928f0045863780b8
    supports: [concepts, rules, capability.seller-scorecard, capability.lost-reasons]
  - path: frontend/routes/crm/goals/goals.ts
    role: business-logic
    hash: sha256:4c6484024749c7aedb2c6d6dbc213b970d4e1ff586b7fc75f061aba933d4be21
    supports: [capability.seller-scorecard, troubleshooting]
  - path: frontend/services/crm/crm-reports.svelte.ts
    role: frontend-service
    hash: sha256:8e1f224860a2b2e1810fab6030d3cb8803d9044ca0f6cbaaec2de4ccd9161006
    supports: [rules, capability.seller-scorecard]
  - path: backend/crm/report.go
    role: backend-handler
    hash: sha256:49358ff5d81bf3b29aea73be2dc8c9a6264b17bd79bbb2f13eb3c0921d6035e0
    supports: [rules, capability.seller-scorecard]
  - path: backend/crm/types/report_rules.go
    role: permissions
    hash: sha256:39dff4dbbaefd5888caee45a3520401a1510e3b68f673acf24084b63129aa16d
    supports: [rules, troubleshooting]
  - path: backend/crm/types/stage_log.go
    role: data-model
    hash: sha256:f07b30e552aa3714b3cb1bf5a2bb6dd1c1f9b61d3691ba14154e145eabb030d2
    supports: [concepts, rules, capability.lost-reasons]
  - path: frontend/routes/crm/pipeline/OpportunityLayer.svelte
    role: user-interface
    hash: sha256:1bf5b6bddb85bea09ad026b7a71b19cb2926bb184230d200c0a4589f2c7ba982
    supports: [concepts]
  - path: frontend/routes/crm/pipeline/OpportunityCloseModal.svelte
    role: user-interface
    hash: sha256:4b6ac6cad2c5db8701a3c15a31a52a128a6579582608d6dedfcaab87bf8c3091
    supports: [concepts]
  - path: frontend/routes/crm/dictionaries/+page.svelte
    role: user-interface
    hash: sha256:381b2031bcf3357afad5dd11bb6df5dbd75313419024f301affb8ca158643232
    supports: [capability.lost-reasons, related-pages]
```
