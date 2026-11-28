---
schema: 1
page_id: crm.goals
route: /crm/goals
title: Goals (Metas)
status: implemented
visibility: tenant
description_en: >-
  Grid to set CRM sales goals per seller for one pipeline, metric and year: won amount, won count,
  new opportunities or activities, by month, week or day as the pipeline defines. Team rows add up
  their members. Fill a row, copy last year's goals, and save only the edited cells.
description_es: >-
  Grilla para fijar las metas de venta del CRM por vendedor para un pipeline, métrica y año: monto
  ganado, cantidad ganada, oportunidades nuevas o actividades, por mes, semana o día según el
  pipeline. Las filas de equipo suman a sus integrantes. Repetir una fila, copiar las metas del año
  anterior y guardar solo las celdas editadas.
---

# Goals (Metas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Goals (`Metas CRM`) is where a manager sets the sales targets (`metas de venta`, `cuotas`) of each
seller (`vendedor`) in a CRM pipeline. Goals are typed in a grid of sellers × periods for one
pipeline, one metric and one year, and saved together.

The page only stores targets. It does not measure results: the won amount against the goal is
shown in **Forecast** and in **Performance (Desempeño)**. Sellers are listed only when they are
members of the pipeline (**Pipelines: Access (Flujos: Accesos)**), and they are grouped by the
teams of **Teams (Equipos)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **goal (meta)** is the target of one seller, in one pipeline, for one period and one metric.
- **Metric (Métrica):** **Won amount (Monto ganado)** in money, or a count: **Won count
  (Ganadas)**, **New opportunities (Oportunidades nuevas)** or **Activities (Actividades)**.
- **Goal period (unidad de la meta):** each pipeline decides it in **Pipelines (Flujos)**, field
  **Goals (Metas)**: monthly (`Mensuales`, 12 columns), weekly (`Semanales`, the weeks whose Monday
  falls in the year, shown as number · Monday date) or daily (`Diaria | Mes` shows the days of one
  month at a time; `Diaria | Semana` the days of one week at a time).
- A **team row (fila de equipo)** is the sum of its members' goals in that pipeline. It is read
  only.

<!-- DOC-ID: capability.set-goals -->
## Set the goals of the sellers (Fijar las metas de los vendedores)

### User intention (Intención del usuario)

Give each seller a target for the year, period by period, so progress can be tracked against it.

### Where to find it (Dónde encontrarlo)

**CRM → Management (Gestión) → Goals (Metas)** at `/crm/goals`. Choose **Pipeline**, **Metric
(Métrica)** and **Year (Año)**; for a daily pipeline also choose the **Month (Mes)** or **Week
(Semana)** to show. Type the values in the cells and press **Save (Guardar)**.

### Required information and prerequisites (Requisitos previos)

- The pipeline must have members; only members appear as rows, grouped under their team, then the
  members without a team.
- The year list offers next year, the current year and the two previous years. A daily pipeline
  opens on today's month or week.

### Business rules and rationale (Reglas y razón de negocio)

- A cell cannot be negative; a negative entry becomes 0.
- **Won amount (Monto ganado)** is typed in currency units, and its team rows and totals show two
  decimals; the other metrics are counts, saved as whole numbers.
- The **Total** column adds the periods shown, for every seller and team row.
- **Save (Guardar)** sends only the cells whose value changed and stays disabled while nothing
  changed. Setting a cell to 0 clears that goal.
- Changing the pipeline or the metric discards the unsaved edits.
- Genix rejects a goal for an inactive pipeline or user, or for a period that does not match the
  pipeline's goal unit ("El periodo … no es válido para este pipeline.").

### Result and side effects (Resultado y efectos)

The goals are stored per seller. The **Won amount (Monto ganado)** goals appear in the **Goal
(Meta)** column of **Forecast** and of the **Sellers (Vendedores)** view of **Performance
(Desempeño)**, where **% goal (% meta)** compares them with the amount won.

### Limitations (Limitaciones)

- Goals are set per seller; a team goal is always the sum of its members and cannot be typed.
- Only **Won amount (Monto ganado)** goals are compared with results in other pages. Goals of
  **Won count (Ganadas)**, **New opportunities (Oportunidades nuevas)** and **Activities
  (Actividades)** are stored but no page compares them with results yet.
- Changing the pipeline's goal unit does not convert goals already typed in the previous unit.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo pongo la meta mensual de cada vendedor? ¿Dónde asigno la cuota de ventas?
- ¿Puedo poner metas semanales o diarias? → set the pipeline's goal unit in Pipelines (Flujos).
- ¿Cómo borro una meta? → set the cell to 0 and save.
- ¿Por qué no aparece un vendedor? → it is not a member of the pipeline.
- Meta de ventas, cuota, objetivo comercial, presupuesto de ventas, target.

<!-- DOC-ID: capability.fill-and-copy -->
## Fill a row or copy last year (Repetir una fila o copiar el año anterior)

### User intention (Intención del usuario)

Avoid typing the same target in every column, or start from last year's targets.

### Where to find it (Dónde encontrarlo)

- The pencil at the end of a seller row repeats the row's first value across all the columns shown.
- **Copy previous year (Copiar año anterior)** fills the grid with last year's goals for every
  seller of the pipeline, for the selected metric: month by month, or week n to week n.

### Business rules and rationale (Reglas y razón de negocio)

- Copy only brings periods that had a goal above 0 last year, and replaces the unsaved edits on the
  grid.
- Both actions only change the grid. Nothing is stored until **Save (Guardar)**.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo copio las metas del año pasado? ¿Cómo pongo la misma meta todos los meses?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **A seller is missing from the grid:** add them as a member of the pipeline in Pipelines: Access
  (Flujos: Accesos).
- **The team row cannot be edited:** it is the sum of its members; type the goals in the seller
  rows.
- **Save (Guardar) is disabled:** no cell differs from what is stored.
- **"El usuario … no existe o está inactivo.":** the user was deactivated; their goals cannot be
  changed.
- **The goal does not appear in Forecast or Performance:** only the Won amount (Monto ganado) metric
  is read there, for the same pipeline.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Pipelines (Flujos)** sets the goal unit (monthly, weekly or daily) of each pipeline.
- **Pipelines: Access (Flujos: Accesos)** decides which sellers appear in the grid.
- **Teams (Equipos)** groups sellers into the team subtotal rows.
- **Forecast** compares the period's goal with what is won and expected to close.
- **Performance (Desempeño)** compares the goals with the amount won over a range of months.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.set-goals]
  - path: frontend/routes/crm/goals/+page.svelte
    role: page
    hash: sha256:1807dca16ddbeb48f4749a7b20b0f9c504860cab9145c2598fab6188bdff6078
    supports: [page-purpose, concepts, capability.set-goals, capability.fill-and-copy, troubleshooting]
  - path: frontend/routes/crm/goals/goals.ts
    role: business-logic
    hash: sha256:4c6484024749c7aedb2c6d6dbc213b970d4e1ff586b7fc75f061aba933d4be21
    supports: [concepts, capability.set-goals, capability.fill-and-copy, troubleshooting]
  - path: frontend/services/crm/crm-goal.svelte.ts
    role: frontend-service
    hash: sha256:8ab1a5d4485032bea9cf378d33baa288a349f4748c6f2bbe9a178afc4db5d13c
    supports: [capability.set-goals]
  - path: frontend/services/crm/crm-goal-metric.ts
    role: shared-domain
    hash: sha256:562d7dc2cc92411524b345d06df3a78a05dc20a513de6fcec2b83107a401ab85
    supports: [concepts]
  - path: backend/crm/goal.go
    role: backend-handler
    hash: sha256:5fc032c56948e1c8df5eff099e73f050e8336fc0e8db4a51fccbada066571ceb
    supports: [capability.set-goals, troubleshooting]
  - path: backend/crm/types/goal.go
    role: data-model
    hash: sha256:30430b7c4280341902876a84396d6df594630d4ffa4758a709764824caebe392
    supports: [concepts, capability.set-goals]
  - path: backend/crm/types/goal_rules.go
    role: business-logic
    hash: sha256:05536fa4d8ce2df78c265c291910b6133747219d3c69941d210e2f3ed9826d8c
    supports: [concepts, capability.set-goals, troubleshooting]
  - path: backend/crm/types/pipeline.go
    role: data-model
    hash: sha256:272e4f0c215cf274446e0a9e22b87467413c1f54796777ff9762a4dbd4f2d639
    supports: [concepts]
  - path: backend/crm/types/pipeline_rules.go
    role: business-logic
    hash: sha256:f574d970c74d1cf8966d1bcde3c8b9640af596d51830624e06a605650c757c48
    supports: [capability.set-goals]
  - path: frontend/routes/crm/settings/PipelinesTab.svelte
    role: user-interface
    hash: sha256:4b65e9f5c6b5f39407c2caeb8f32ecb34a0bb6309855dad9a099be5064ab43a4
    supports: [concepts, related-pages]
  - path: frontend/routes/crm/forecast/+page.svelte
    role: user-interface
    hash: sha256:8b5e57100f75cb90390095b37d42498da0d6e4f6d84321229813b5fe4fa915c6
    supports: [capability.set-goals, troubleshooting, related-pages]
  - path: frontend/routes/crm/reports/ReportScorecard.svelte
    role: user-interface
    hash: sha256:c6b5929a5c7a9645eca696ebfd668ae8e7a8d36c0dbdf77d22ed5b0ecac416a9
    supports: [capability.set-goals, troubleshooting, related-pages]
```
