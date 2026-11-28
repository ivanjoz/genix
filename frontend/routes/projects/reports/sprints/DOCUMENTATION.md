---
schema: 1
page_id: projects.reports.sprints
route: /projects/reports/sprints
title: Sprint burndown (Burndown de sprints)
status: implemented
visibility: tenant
description_en: >-
  Burndown chart of a project's active or closed sprints: the estimated hours of the stories not
  finished yet at the end of each day, against the ideal line from the hours committed on the first
  day.
description_es: >-
  Gráfico burndown de los sprints activos o cerrados de un proyecto: las horas estimadas de las
  historias aún no finalizadas al final de cada día, frente a la línea ideal desde las horas
  comprometidas el primer día.
---

# Sprint burndown (Burndown de sprints)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sprint burndown (`Burndown de sprints`, page title **Sprints**) charts how the work of a sprint burns
down (`gráfico de quemado`, `trabajo restante`): the estimated hours still unfinished at the end of
each day, next to the ideal pace. It helps tell whether the sprint will finish its commitment.

It is read only. Stories are moved on the **Sprints** board, which is what makes the line go down.

<!-- DOC-ID: capability.burndown -->
## Read a sprint's burndown (Ver el burndown de un sprint)

### User intention (Intención del usuario)

Follow the active sprint's pace, or review how a closed sprint went.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Reports (Reportes) → Sprint burndown (Burndown de sprints)** at
`/projects/reports/sprints`, or **Reports (Reportes)** in a sprint board's menu, which opens that
sprint. The project is chosen in the header search box ("P01 · Name"). The left column lists the
project's sprints; the tab **Burndown** shows the chart.

### Business rules and rationale (Reglas y razón de negocio)

- Only started sprints are listed (active and closed); a planned sprint has nothing to report. Without
  a chosen sprint, the active one is shown, else the latest closed one.
- **Hours left (Horas restantes)**: the estimated hours of the sprint's stories not finished yet at
  the end of each day, drawn up to today for an active sprint and over every day for a closed one.
- **Ideal**: a straight line from the hours committed on the first day down to 0 on the last day.
  The commitment counts the stories that were in the sprint on its first day.
- A story added after the start raises the line on that day; the ideal line keeps the first day's
  commitment, so the added scope shows. A finished story that is reopened counts again.
- Stories without estimated hours weigh nothing. When no story has hours the chart says "Las
  historias de este sprint no tienen horas estimadas: no hay nada que mostrar."

### Limitations (Limitaciones)

- The chart uses estimated hours, not hours actually worked: stories hold only an estimate.
- Burndown is the only report on this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Vamos a terminar el sprint a tiempo? ¿Cuántas horas faltan en el sprint? ¿Por qué subió la línea
  del burndown? → a story was added or reopened.
- Burndown, gráfico de avance, trabajo restante, velocidad del sprint.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sprints** plans the sprint's stories and moves them to Finalizado.
- **Backlog** holds each story's estimated hours.
- **Gantt** and **Calendar (Calendario)** show the sprints and stories over time.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.burndown]
  - path: frontend/routes/projects/reports/sprints/+page.svelte
    role: page
    hash: sha256:b6ac7310e27a2f8787c12f7b0f35ba23131fed7b0ae9f1c89958eb2ba011d3e4
    supports: [page-purpose, capability.burndown]
  - path: frontend/routes/projects/reports/sprints/SprintBurndown.svelte
    role: user-interface
    hash: sha256:6efc1a68a94ede9b91657036c13713307daaa9ae23360d2a8f09cf40b2b112eb
    supports: [capability.burndown]
  - path: frontend/routes/projects/sprints/sprints.ts
    role: business-logic
    hash: sha256:5c8fe6ad195272897b05b9c85e31838bbc857c299b97645f16227ade31971213
    supports: [capability.burndown]
  - path: frontend/routes/projects/sprints/SprintMenu.svelte
    role: user-interface
    hash: sha256:eaad94a06e3df2ee7e445071858fb937b95b5130ce16f2af5eb9bc223a8c460a
    supports: [capability.burndown]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.burndown]
  - path: backend/projects/sprint_rules.go
    role: business-logic
    hash: sha256:620521e71cb9f322a4cfc756f7e435330b9c01bb7f180eb44e58a2a55ba244b2
    supports: [capability.burndown]
  - path: backend/projects/types/sprints.go
    role: data-model
    hash: sha256:5e07f17e54d7644084cd5583b24cb303fda3ae371374fa9611725c5b43ed8511
    supports: [capability.burndown]
  - path: backend/projects/types/stories.go
    role: data-model
    hash: sha256:33d0fca372da7917d1abc4a46ba6410033bc4e4ca1f79d6cc487179617afbe8a
    supports: [capability.burndown]
```
