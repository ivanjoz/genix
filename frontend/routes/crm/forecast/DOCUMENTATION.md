---
schema: 1
page_id: crm.forecast
route: /crm/forecast
title: Forecast (Forecast)
status: implemented
visibility: tenant
description_en: >-
  Sales forecast of a CRM pipeline per seller and team for one goal period (month, week or day):
  goal, amount already won, open opportunities expected to close in the period by forecast category
  (commit, best case, pipeline), and the forecasts each seller or team leader submitted, with a
  chart of how the submitted forecast evolved.
description_es: >-
  Forecast de ventas de un pipeline del CRM por vendedor y equipo para un periodo de metas (mes,
  semana o día): meta, monto ya ganado, oportunidades abiertas que se esperan cerrar en el periodo
  por categoría de forecast (compromiso, mejor caso, pipeline) y los forecasts que envió cada
  vendedor o líder de equipo, con un gráfico de cómo evolucionó el forecast enviado.
---

# Forecast (Forecast)

<!-- DOC-ID: page-purpose -->
## Page purpose

Forecast (`Forecast CRM`) answers "how much will we sell this period?" (`¿cuánto vamos a vender
este mes?`) for one CRM pipeline. For each seller (`vendedor`) and each team (`equipo`) it puts
side by side the goal (`meta`), what is already won (`ganado`), the open opportunities
(`oportunidades abiertas`) expected to close in the period grouped by forecast category, and the
forecast the seller or the team leader declared (`forecast enviado`).

The page only reads opportunities; it does not move, win or lose them (that is done in
**Opportunities (Oportunidades)**). It does not set goals (**Goals (Metas)**), build teams
(**Teams (Equipos)**) or decide which stage counts as commit or best case (**Pipelines (Flujos)**).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Goal period (periodo de metas):** the unit the pipeline uses for goals and forecasts, chosen in
  **Pipelines (Flujos)** with the **Goals (Metas)** field: monthly (`Mensuales`), weekly
  (`Semanales`) or daily (`Diaria | Mes`, `Diaria | Semana`). The Forecast page follows it: the
  period selector is labelled **Month (Mes)**, **Week (Semana)** or **Day (Día)** accordingly. A
  week is numbered within the year and shown with its Monday.
- **Forecast category (categoría de forecast):** every stage of the pipeline is set as
  **Pipeline**, **Best case (Mejor caso)** or **Commit (Compromiso)**. An open opportunity counts in
  the category of the stage it sits in. Won and lost are outcomes, not stages, so they are not
  categories.
- **Expected close date (fecha esperada de cierre):** an open opportunity counts in a period only
  when its expected close date falls in that period. Opportunities without an expected close date
  are not counted.
- **Forecast submission (forecast enviado):** what a seller or a team leader declares for the
  period: a Commit amount, a Best case amount and a note. Every submission is kept as a new record,
  so the page can show how the declared forecast evolved during the period.

<!-- DOC-ID: capability.view-forecast -->
## Review the forecast of a period (Revisar el forecast del periodo)

### User intention (Intención del usuario)

See, per seller and team, how close the period is to its goal and how much of the open pipeline
is likely to close in it.

### Where to find it (Dónde encontrarlo)

**CRM → Reports (Reportes) → Forecast** at `/crm/forecast`. Choose **Pipeline**, **Year (Año)**
and the period (**Month (Mes)**, **Week (Semana)** or **Day (Día)**). The year list offers next year,
the current year and the two previous years; the period opens on the one holding today, or on the first period of
the year when another year is chosen.

### Required information and prerequisites (Requisitos previos)

- The pipeline list only offers active pipelines the user may report on: those where the user is a
  member at any level, or every active pipeline when the user holds the **CRM Reporte completo**
  access. Without any, the page shows "No tiene pipelines para reportar."
- Rows list the pipeline's members, grouped by team: each team row (in bold) is followed by its
  members who work in the pipeline, then the members with no team. Teams without members in the
  pipeline are not shown.

### Business rules and rationale (Reglas y razón de negocio)

The table columns are:

- **Seller (Vendedor):** the seller or the team name.
- **Goal (Meta):** the seller's **Won amount (Monto ganado)** goal for that period, from Goals
  (Metas). A team row adds up its members' goals.
- **Won (Ganado):** the amount won inside the period's days, net of opportunities reopened in the
  same period, attributed to the opportunity's owner.
- **% goal (% meta):** Won ÷ Goal, rounded; 0 when there is no goal.
- **Commit (Compromiso)**, **Best case (Mejor caso)** and **Pipeline:** the amount of today's open
  opportunities whose expected close date falls in the period, split by the forecast category of
  their stage.
- **Won + commit (Ganado + compromiso):** shown in red when a goal exists and this sum is still
  below it.
- **Submitted before start (Enviado antes del inicio):** the Commit of the last submission made
  before the period started. There is no forecast cut-off date, so the page shows this one next to
  the latest.
- **Last submitted (Último enviado):** the Commit of the newest submission. A dash (`—`) means
  nothing was submitted.

Clicking a row opens **Submitted forecast (Forecast enviado)** for that seller or team: a line chart
with the Commit and Best case of every submission in the period, and the goal as a flat reference
line. Without submissions it says "Aún no se envió un forecast para este periodo." Clicking the
row again hides it.

### Limitations (Limitaciones)

- Commit, Best case and Pipeline always read the opportunities open today; they are not a
  historical picture of a past period.
- The page does not edit opportunities. To change an expected close date, amount or stage, use
  **Opportunities (Oportunidades)**; the forecast follows those changes.
- Only the **Won amount (Monto ganado)** goal is used; goals of other metrics do not appear here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuánto vamos a vender este mes? ¿Cuánto le falta a cada vendedor para su meta?
- ¿Qué es commit y mejor caso? ¿Por qué una oportunidad no aparece en el forecast? → its expected
  close date is in another period, or it has none.
- ¿Por qué "Ganado + compromiso" sale en rojo? → won plus commit does not reach the goal.
- Pronóstico de ventas, proyección de ventas, previsión, forecast semanal, cumplimiento de meta.

<!-- DOC-ID: capability.submit-forecast -->
## Submit a forecast (Enviar forecast)

### User intention (Intención del usuario)

Declare how much the seller, or the team, commits to close in the period, so managers can compare
what was promised with what happened.

### Where to find it (Dónde encontrarlo)

On the Forecast table, the pencil button at the end of the user's own row, or of the team row when
the user is that team's leader. It opens the **Submit forecast (Enviar forecast)** dialog with
**Commit (Compromiso)**, **Best case (Mejor caso)** and **Note (Nota)**; save it to send.

### Required information and prerequisites (Requisitos previos)

- Commit and Best case start filled with the amounts the page computed for that row; the user may
  change them.
- Each seller submits only their own row. Only the team's leader submits the team row
  ("Solo el líder del equipo puede enviar el forecast del equipo.").
- The user must be a member of the pipeline ("No tiene acceso a este pipeline.").

### Business rules and rationale (Reglas y razón de negocio)

- Amounts cannot be negative ("Los montos del forecast no pueden ser negativos.").
- If both Commit and Best case are sent as 0, Genix stores the computed amounts instead.
- When saving, Genix recomputes the forecast from the opportunities at that moment and stores it
  next to what the user declared, so "what was said" can be contrasted with "what the data said
  then".

### Result and side effects (Resultado y efectos)

A new submission is added for that pipeline and period. **Last submitted (Último enviado)** and
the **Submitted forecast (Forecast enviado)** chart update. Opportunities, goals and stages do not
change.

### Limitations (Limitaciones)

- A submission cannot be edited or deleted: submitting again adds a newer one, and the previous
  ones remain in the evolution chart.
- The **Note (Nota)** is saved with the submission but is not shown on this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo envío mi forecast del mes? ¿Cómo corrijo un forecast enviado? → submit a new one.
- ¿Quién envía el forecast del equipo? → the team leader set in Teams (Equipos).
- ¿Por qué no veo el lápiz en una fila? → it is another seller's row, or a team the user does not
  lead.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No tiene pipelines para reportar.":** the user is not a member of any active pipeline; ask for
  membership in **Pipelines: Access (Flujos: Accesos)** or for the **CRM Reporte completo** access.
- **A seller is missing from the table:** only members of the selected pipeline are listed; add
  them in Pipelines: Access (Flujos: Accesos).
- **No team row appears:** the team has no members in this pipeline, or the seller is not in any
  team (Teams (Equipos)).
- **Goal shows 0:** no Won amount goal was set for that seller and period in Goals (Metas).
- **"Solo el líder del equipo puede enviar el forecast del equipo.":** only the leader chosen in
  Teams (Equipos) submits the team row.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Goals (Metas)** sets the Won amount goal shown in **Goal (Meta)**.
- **Teams (Equipos)** groups sellers into the team rows and names the leader who submits for the
  team.
- **Pipelines (Flujos)** sets each stage's forecast category and the pipeline's goal period.
- **Opportunities (Oportunidades)** is where amounts, stages and expected close dates change.
- **Performance (Desempeño)** compares won amount against the same goals over a range of months.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.view-forecast]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, related-pages]
  - path: frontend/routes/crm/forecast/+page.svelte
    role: page
    hash: sha256:8b5e57100f75cb90390095b37d42498da0d6e4f6d84321229813b5fe4fa915c6
    supports: [page-purpose, concepts, capability.view-forecast, capability.submit-forecast, troubleshooting]
  - path: frontend/routes/crm/forecast/forecast.svelte.ts
    role: frontend-service
    hash: sha256:dc540c6b7fa58c4e33e1a8284a3cb5060e54d2e07334acbf393de820dc652c0e
    supports: [capability.view-forecast, capability.submit-forecast]
  - path: frontend/routes/crm/forecast/forecast.ts
    role: business-logic
    hash: sha256:abe4fe0f4e417ff36fcf0744ff841aafc60d89a9b0b34182bab431132e34cc04
    supports: [concepts, capability.view-forecast]
  - path: frontend/routes/crm/goals/goals.ts
    role: business-logic
    hash: sha256:4c6484024749c7aedb2c6d6dbc213b970d4e1ff586b7fc75f061aba933d4be21
    supports: [concepts, capability.view-forecast, troubleshooting]
  - path: frontend/routes/crm/reports/reports.svelte.ts
    role: frontend-service
    hash: sha256:2370c90ccd1c87242bd8cc8dbea755e6a3264c478f01204a217b976083ea6b51
    supports: [capability.view-forecast]
  - path: frontend/routes/crm/reports/reports.ts
    role: business-logic
    hash: sha256:d8f9bdd066fab48a269cc3e1e2734512d0db083c6989d4d3928f0045863780b8
    supports: [capability.view-forecast, troubleshooting]
  - path: frontend/services/crm/crm-forecast.svelte.ts
    role: frontend-service
    hash: sha256:5b3d99d0494e3709f2a566881eb9bfcd897dcade1c9d342fef10d5748625f0c0
    supports: [capability.submit-forecast]
  - path: frontend/routes/crm/settings/PipelinesTab.svelte
    role: user-interface
    hash: sha256:4b65e9f5c6b5f39407c2caeb8f32ecb34a0bb6309855dad9a099be5064ab43a4
    supports: [concepts, related-pages]
  - path: backend/crm/forecast.go
    role: backend-handler
    hash: sha256:9d9f7d7d95d31ad786a87521b90f60c9cf1e406e9eba3d44ce7256ebcfdf53f4
    supports: [capability.submit-forecast]
  - path: backend/crm/types/forecast.go
    role: data-model
    hash: sha256:c510536d241d4a073151fffecc0085b194f8795a29849523f31c8d254327dd15
    supports: [concepts, capability.submit-forecast]
  - path: backend/crm/types/forecast_rules.go
    role: business-logic
    hash: sha256:ab2072541fbf3b0dd8e85d6f6e2e0befcebbbfb29da81cab8ff6cf9dec987b05
    supports: [concepts, capability.submit-forecast, troubleshooting]
  - path: backend/crm/types/pipeline.go
    role: data-model
    hash: sha256:272e4f0c215cf274446e0a9e22b87467413c1f54796777ff9762a4dbd4f2d639
    supports: [concepts]
  - path: backend/crm/types/report_rules.go
    role: permissions
    hash: sha256:39dff4dbbaefd5888caee45a3520401a1510e3b68f673acf24084b63129aa16d
    supports: [capability.view-forecast, troubleshooting]
```
