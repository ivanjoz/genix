---
schema: 1
page_id: crm.teams
route: /crm/teams
title: Teams (Equipos)
status: implemented
visibility: tenant
description_en: >-
  Groups CRM sellers into teams with a leader. A user belongs to one team only. Teams add subtotal
  rows to the goals grid, the forecast and the seller scorecard; the leader submits the team's
  forecast and can read the tasks of the team members.
description_es: >-
  Agrupa a los vendedores del CRM en equipos con un líder. Un usuario pertenece a un solo equipo.
  Los equipos agregan filas de subtotal en la grilla de metas, el forecast y el desempeño de
  vendedores; el líder envía el forecast del equipo y puede ver las tareas de sus integrantes.
---

# Teams (Equipos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Teams (`Equipos CRM`) groups the company's sellers (`vendedores`) into sales teams (`equipos de
venta`), each with one leader (`líder`). Teams exist to roll up the CRM numbers: they add a team
subtotal row to **Goals (Metas)**, **Forecast** and the **Sellers (Vendedores)** view of
**Performance (Desempeño)**.

The page does not decide who may work in a pipeline: pipeline membership and levels are set in
**Pipelines: Access (Flujos: Accesos)**. It does not create users either; members are picked from
the active users of the company.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **team (equipo)** has a name, a list of members (`miembros`) and one leader chosen among those
  members.
- A user belongs to **only one team**, so a team roll-up never counts a seller twice.
- The **leader (líder)** has two extra roles in the CRM: only the leader submits the team's
  forecast on the Forecast page, and the leader can read (not act on) the tasks of the team
  members in **Tasks (Tareas)**.
- A team is not tied to a pipeline. In each pipeline, the team row only counts its members who are
  also members of that pipeline.

<!-- DOC-ID: capability.manage-team -->
## Create or edit a team (Crear o editar un equipo)

### User intention (Intención del usuario)

Group sellers under a leader so goals, forecasts and results can be read per team.

### Where to find it (Dónde encontrarlo)

**CRM → Management (Gestión) → Teams (Equipos)** at `/crm/teams`. The table lists **Team
(Equipo)**, **Leader (Líder)** and **Members (Miembros)**. Use **New (Nuevo)** to open the **New
team (Nuevo equipo)** side panel, or click a row to edit that team. Save with the panel's save
button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** required.
- **Members (Miembros):** at least one. The list shows every active user with a checkbox. A user who
  already belongs to another team appears greyed as "En equipo …" with that team's name, without a
  checkbox.
- **Leader (Líder):** required, chosen only among the members already ticked. Unticking the leader
  clears the leader.

### Business rules and rationale (Reglas y razón de negocio)

- The name cannot be empty, and two active teams cannot share the same name ("Ya existe un equipo
  llamado …").
- At least one member ("El equipo necesita al menos un miembro.").
- The leader must be one of the members ("El líder debe ser miembro del equipo.").
- Every member must be an active user, and a user cannot be in another active team ("El usuario …
  ya pertenece al equipo …").

### Result and side effects (Resultado y efectos)

The team appears at once in the goals grid, the forecast and the seller scorecard as a bold team
row with its members under it. The leader can submit the team forecast and read the members' tasks.
No goals, opportunities or past forecasts are changed: team rows are sums computed when each page
is shown.

### Limitations (Limitaciones)

- A user cannot be in two teams at once; remove them from the first team before adding them to
  another.
- A team has a single leader.
- Team rows on the other pages are read-only sums: goals are typed per seller, never per team.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un equipo de ventas? ¿Cómo asigno un jefe o supervisor de ventas?
- ¿Por qué no puedo marcar a un vendedor? → it already belongs to another team.
- ¿Cómo cambio el líder del equipo?
- Equipo comercial, fuerza de ventas, supervisor, jefe de equipo, coordinador.

<!-- DOC-ID: capability.delete-team -->
## Delete a team (Eliminar un equipo)

### User intention (Intención del usuario)

Remove a team that no longer exists.

### Where to find it (Dónde encontrarlo)

Open the team from the table, then the side panel's delete button.

### Result and side effects (Resultado y efectos)

The team leaves the list and its subtotal rows disappear from Goals (Metas), Forecast and
Performance (Desempeño). Its members become free to join another team. The goals and forecasts
already saved for each seller remain.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo elimino un equipo? ¿Qué pasa con las metas si borro el equipo? → the sellers' goals stay.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El equipo necesita un nombre." / "El nombre del equipo es obligatorio.":** type a name.
- **"Elija al menos un miembro.":** tick at least one user.
- **"Elija un líder entre los miembros." / "El líder debe ser miembro del equipo.":** pick the
  leader after ticking the members.
- **"Ya existe un equipo llamado …":** choose another name.
- **"El usuario … ya pertenece al equipo …":** remove that user from the other team first.
- **The team row does not show in a pipeline's goals or forecast:** none of its members is a member
  of that pipeline; add them in Pipelines: Access (Flujos: Accesos).

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Goals (Metas)** shows each team as a subtotal row of its members' goals.
- **Forecast** shows the team row, which only the leader submits.
- **Performance (Desempeño)**, view **Sellers (Vendedores)**, adds team subtotals with their own
  win rate.
- **Tasks (Tareas)** lets the leader read the tasks of the team's members.
- **Pipelines: Access (Flujos: Accesos)** decides who works in each pipeline.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.manage-team]
  - path: frontend/routes/crm/teams/+page.svelte
    role: page
    hash: sha256:dc3aa01da2fbd596e76e4ea97fd79012d6b384810e8e143702083a52db773569
    supports: [page-purpose, capability.manage-team, capability.delete-team]
  - path: frontend/routes/crm/teams/teams.ts
    role: business-logic
    hash: sha256:e374f893af5a435ca3993cdcf65139a9d07c10652fdad575f799712ac154f424
    supports: [concepts, capability.manage-team, troubleshooting]
  - path: frontend/services/crm/crm-team.svelte.ts
    role: frontend-service
    hash: sha256:b9889828e488c83baffb14060768f7c47aae34e85b8c8ec2bf421ff127450a5d
    supports: [capability.manage-team, capability.delete-team]
  - path: frontend/routes/security/users-profiles/users-profiles.svelte.ts
    role: frontend-service
    hash: sha256:c4eb5b0302c1f9f54cf0b447be33d4d7972b76c88501b952a2ea6671f537d618
    supports: [page-purpose, capability.manage-team]
  - path: backend/crm/team.go
    role: backend-handler
    hash: sha256:195ff82f2f9435a576b91096f7075362ee05cb8507a350330a630dc17929e0cc
    supports: [capability.manage-team, capability.delete-team]
  - path: backend/crm/types/team.go
    role: data-model
    hash: sha256:8e73aaec2116779d6b0dc0e9132796f4d59d2639d964a38a56a813166333e1b9
    supports: [concepts]
  - path: backend/crm/types/team_rules.go
    role: business-logic
    hash: sha256:a91bbe401b5e0f4b934c733f09a6708f12f0883abb8091f379c00c96bbf5bce2
    supports: [concepts, capability.manage-team, capability.delete-team, troubleshooting]
  - path: frontend/routes/crm/goals/goals.ts
    role: business-logic
    hash: sha256:4c6484024749c7aedb2c6d6dbc213b970d4e1ff586b7fc75f061aba933d4be21
    supports: [concepts, capability.manage-team, troubleshooting, related-pages]
  - path: frontend/routes/crm/goals/+page.svelte
    role: user-interface
    hash: sha256:1807dca16ddbeb48f4749a7b20b0f9c504860cab9145c2598fab6188bdff6078
    supports: [capability.manage-team, related-pages]
  - path: frontend/routes/crm/forecast/+page.svelte
    role: user-interface
    hash: sha256:8b5e57100f75cb90390095b37d42498da0d6e4f6d84321229813b5fe4fa915c6
    supports: [concepts, related-pages]
  - path: frontend/routes/crm/reports/ReportScorecard.svelte
    role: user-interface
    hash: sha256:c6b5929a5c7a9645eca696ebfd668ae8e7a8d36c0dbdf77d22ed5b0ecac416a9
    supports: [page-purpose, related-pages]
  - path: backend/crm/types/forecast_rules.go
    role: business-logic
    hash: sha256:ab2072541fbf3b0dd8e85d6f6e2e0befcebbbfb29da81cab8ff6cf9dec987b05
    supports: [concepts]
  - path: backend/crm/types/task_rules.go
    role: business-logic
    hash: sha256:92c31194e686a3f86e513a30ca18e8aaac5c5b6a9e9fce927ac614d8dc3ba93e
    supports: [concepts, related-pages]
```
