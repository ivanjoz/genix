---
schema: 1
page_id: projects.reports.calendar
route: /projects/reports/calendar
title: Calendar (Calendario)
status: implemented
visibility: tenant
description_en: >-
  Month calendar of a project: its sprints over their dates, the project's start and end, and each
  story on the day it was finished.
description_es: >-
  Calendario mensual de un proyecto: sus sprints en sus fechas, el inicio y el fin del proyecto, y
  cada historia el día en que se finalizó.
---

# Calendar (Calendario)

<!-- DOC-ID: page-purpose -->
## Page purpose

Calendar (`Calendario`) shows one project month by month: when its sprints run and on which day each
story was finished (`historias terminadas por día`). It answers "what did we finish and when" and
"which sprint runs this month".

It does not show due dates or pending work by day: stories have no due date of their own, and only
finished stories appear. For each story's span over the sprints use **Gantt**; for a person's weekly
load use **By person (Por persona)**.

<!-- DOC-ID: capability.month-calendar -->
## See the project's month (Calendario del proyecto)

### User intention (Intención del usuario)

See the sprints of a month and the stories finished each day.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Reports (Reportes) → Calendar (Calendario)** at
`/projects/reports/calendar`. The project is chosen in the header search box ("P01 · Name"); the last
project opened is reopened. The month selector moves with **Previous month (Mes anterior)** and
**Next month (Mes siguiente)**; **Hoy** returns to the current month.

### Business rules and rationale (Reglas y razón de negocio)

- Each sprint is a bar from its start to its end date, colored by state: blue planned, green active,
  gray closed.
- Each finished story appears in green with a check on the day it was moved to **Finalizado**; when
  it finished in several sprints, the latest finish counts.
- The project's **Start (Inicio)** and **End (Fin)** dates appear as flags "Inicio del proyecto" and
  "Fin del proyecto" when set.
- Clicking a story opens it in the same panel as on Backlog; clicking a sprint opens its board on the
  Sprints page.

### Limitations (Limitaciones)

- Stories not finished, and stories never in a sprint, do not appear.
- The page has no filters by person, epic or state.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué historias se terminaron esta semana? ¿Cuándo empieza el próximo sprint? ¿Qué se entregó en
  marzo?
- Calendario, agenda, entregas, fechas del sprint.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sprints** sets the sprints' dates and moves stories to Finalizado.
- **Gantt** shows every story over its sprints and the days it was worked.
- **Sprint burndown (Burndown de sprints)** charts the hours left of a sprint.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.month-calendar]
  - path: frontend/routes/projects/reports/calendar/+page.svelte
    role: page
    hash: sha256:74adfedd4f0af49761b315552d1d932d840343b349edc5ec1bf8b129226a2a6c
    supports: [page-purpose, capability.month-calendar]
  - path: frontend/routes/projects/reports/calendar/ProjectCalendar.svelte
    role: user-interface
    hash: sha256:74792940e829140a02a4435ced14a3a64a770f2769785e671422b73d8e5d1915
    supports: [page-purpose, capability.month-calendar]
  - path: frontend/routes/projects/reports/timeline.ts
    role: business-logic
    hash: sha256:4fa5a775f464fe01a41207a1fee5a0a91b61786f0b06b2ab0c06d21b93e8be00
    supports: [page-purpose, capability.month-calendar]
  - path: frontend/domain-components/MonthInlineSelector.svelte
    role: user-interface
    hash: sha256:e0b4d0e2a097a0c35f4098475c9f30217e1c0fca85e189e2860897b23a67e875
    supports: [capability.month-calendar]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.month-calendar]
  - path: frontend/routes/projects/story/StoryPanel.svelte
    role: user-interface
    hash: sha256:2b344d7e9dbaae834b0f101b6b6e918025222ac160cad5929e4e25aaff8d587d
    supports: [capability.month-calendar]
  - path: backend/projects/types/sprints.go
    role: data-model
    hash: sha256:5e07f17e54d7644084cd5583b24cb303fda3ae371374fa9611725c5b43ed8511
    supports: [capability.month-calendar]
```
