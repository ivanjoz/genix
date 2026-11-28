---
schema: 1
page_id: projects.reports.gantt
route: /projects/reports/gantt
title: Gantt (Gantt)
status: implemented
visibility: tenant
description_en: >-
  Gantt chart of a project. The Weeks view shows the sprints and, per epic, each story over the
  sprints it was in, colored by its state. The By sprint view shows, per epic and person, the days
  each story was actually worked. Filters by state and by assignee.
description_es: >-
  Diagrama de Gantt de un proyecto. La vista Semanas muestra los sprints y, por épica, cada historia
  sobre los sprints en que estuvo, coloreada por su estado. La vista Por sprint muestra, por épica y
  persona, los días en que se trabajó cada historia. Filtros por estado y por responsable.
---

# Gantt (Gantt)

<!-- DOC-ID: page-purpose -->
## Page purpose

Gantt draws one project's timeline (`cronograma`, `línea de tiempo`, `diagrama de Gantt`): its
sprints over their dates and its stories over the time they were planned or worked, grouped by epic.
It is a report: it shows the plan and the progress, it does not plan.

Dates come from the sprints and from the board's state changes. Stories have no due dates of their
own, so a story that was never in a sprint has no bar. Sprints are planned on **Sprints**, and
stories are ordered on **Backlog**.

<!-- DOC-ID: capability.weeks-view -->
## See stories over their sprints (Vista Semanas)

### User intention (Intención del usuario)

See when each sprint runs and which sprints each story went through.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Reports (Reportes) → Gantt** at `/projects/reports/gantt`, view **Weeks
(Semanas)**. The project is chosen in the header search box ("P01 · Name"); the last project opened
is reopened.

### Business rules and rationale (Reglas y razón de negocio)

- The **Sprints** group comes first: one row per sprint, colored by its state (blue planned, green
  active, gray closed), with its progress as the estimated hours finished over the hours committed.
- Then one group per epic, in creation order, and **Sin épica** last. Each group's bar spans its
  stories; each story row, in backlog order, has one bar per sprint it was in, over that sprint's
  dates, labeled "Sprint · state" and colored by its state there: blue Pendiente, purple En
  desarrollo, red Bloqueado, amber En revisión, green Finalizado. A story left unfinished in a closed
  sprint reads **No culminada**.
- Stories never in a sprint are left out; when none has been in a sprint the chart says "Ninguna
  historia ha estado en un sprint todavía."
- The project's **Start (Inicio)** and **End (Fin)** dates show as markers "Inicio del proyecto" and
  "Fin del proyecto".
- Clicking a story bar opens the story; clicking a sprint bar opens its board on the Sprints page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde veo el cronograma del proyecto? ¿En qué sprints estuvo esta historia?
- Gantt, cronograma, línea de tiempo, planificación.

<!-- DOC-ID: capability.sprint-view -->
## See the days each story was worked (Vista Por sprint)

### User intention (Intención del usuario)

See who worked on what and when, inside the sprints.

### Where to find it (Dónde encontrarlo)

Same page, view **By sprint (Por sprint)**. The header shows the sprints with their dates.

### Business rules and rationale (Reglas y razón de negocio)

- Per epic (and **Sin épica**), a title row and then the people holding its stories by name,
  **Sin responsable** last. A person's stories sit side by side in as few rows as fit them.
- A finished story spans from the day it went into development to the day it was finished. Without a
  development day, the span is counted back from the finish day by its estimated hours, at 6 hours a
  day from Monday to Saturday (Sundays skipped); a story without an estimate takes one day.
- A story in development, blocked or in review spans from the day it went into development (else the
  day it entered its current state) until tomorrow.
- Pending and not planned stories are not drawn; when no story has started the chart says "Ninguna
  historia se ha iniciado todavía."
- The days used are the last day the story entered each state: a story that went back into
  development counts from that later day.
- Clicking a bar opens the story.

### Limitations (Limitaciones)

The spans come from state changes on the board, not from recorded working time: Genix projects record
an estimate per story, not hours worked.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuánto tardó una historia? ¿Qué estuvo haciendo cada persona en el sprint?

<!-- DOC-ID: capability.filters -->
## Filter the Gantt (Filtrar el Gantt)

### Where to find it (Dónde encontrarlo)

The select **All states (Todos los estados)**, which also offers **Sin planificar**, and the avatars
**Filter by assignee (Filtrar por responsable)**, one per person holding stories plus **Sin
responsable**; choosing none shows everyone. The view toggle **Semanas / Por sprint** is remembered by
the browser.

### Business rules and rationale (Reglas y razón de negocio)

The filters hide story rows only: the sprint bars and their progress still count every story.

<!-- DOC-ID: capability.open-story -->
## Open a story from the chart (Abrir una historia)

### Business rules and rationale (Reglas y razón de negocio)

A story opens in the same panel as on Backlog, with **Detail (Detalle)** and **Activity
(Actividad)**. Team members of a project not archived can edit it and comment there; the chart
follows the saved changes.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sprints** plans the sprints and moves the stories whose state changes feed this chart.
- **Backlog** sets each story's epic, order, estimate and assignee.
- **Calendar (Calendario)** shows the sprints and the finished stories by month; **By person (Por
  persona)** shows one person's stories across all projects.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.weeks-view]
  - path: frontend/routes/projects/reports/gantt/+page.svelte
    role: page
    hash: sha256:f79caed860d073984810e16f4687b52cb181371509c73678b802b544549dc355
    supports: [page-purpose, capability.weeks-view, capability.sprint-view, capability.filters, capability.open-story]
  - path: frontend/routes/projects/reports/gantt/ProjectGantt.svelte
    role: user-interface
    hash: sha256:6eaf0eadc2e127953b36cfe66822d2b47c4cdf90a698025cc4ef34b01c165d98
    supports: [capability.weeks-view, capability.sprint-view]
  - path: frontend/routes/projects/reports/timeline.ts
    role: business-logic
    hash: sha256:4fa5a775f464fe01a41207a1fee5a0a91b61786f0b06b2ab0c06d21b93e8be00
    supports: [page-purpose, capability.weeks-view, capability.sprint-view, capability.filters]
  - path: frontend/routes/projects/AssigneeFilter.svelte
    role: user-interface
    hash: sha256:2f8b041925f808ff702edb846120410312508d40f86f69bf66bd80753328d433
    supports: [capability.filters]
  - path: frontend/routes/projects/backlog/backlog.ts
    role: business-logic
    hash: sha256:8a14960fb8e2885b6f62250b62d8a23bfbc55e101645de63002f5059461680fe
    supports: [capability.weeks-view, capability.filters]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.weeks-view]
  - path: frontend/routes/projects/story/StoryPanel.svelte
    role: user-interface
    hash: sha256:2b344d7e9dbaae834b0f101b6b6e918025222ac160cad5929e4e25aaff8d587d
    supports: [capability.open-story]
  - path: backend/projects/story_rules.go
    role: business-logic
    hash: sha256:d6397534ef09ee0bfc9afcff78bb6064eb71ddf4d8405be51ded966beaf3c5e9
    supports: [capability.sprint-view]
  - path: backend/projects/types/stories.go
    role: data-model
    hash: sha256:33d0fca372da7917d1abc4a46ba6410033bc4e4ca1f79d6cc487179617afbe8a
    supports: [page-purpose, capability.sprint-view]
```
