---
schema: 1
page_id: projects.sprints
route: /projects/sprints
title: Sprints (Sprints)
status: implemented
visibility: tenant
description_en: >-
  A project's sprints: plan a sprint with backlog stories, its dates, name and goal; start it; move
  its stories across the board (Pending, In development, Blocked / In review, Finished); and close it,
  choosing where each unfinished story goes on.
description_es: >-
  Sprints de un proyecto: planificar un sprint con historias del backlog, fechas, nombre y objetivo;
  iniciarlo; mover sus historias en el tablero (Pendiente, En desarrollo, Bloqueado / En revisión,
  Finalizado); y cerrarlo, eligiendo a dónde pasa cada historia no culminada.
---

# Sprints (Sprints)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sprints organizes a project's work in time boxes (`iteraciones`, `ciclos de trabajo`): a sprint is
planned with stories from the backlog, started, worked on a board (`tablero`, `kanban`) where each
story moves through its states, and closed. It is the only place where a story's state (`estado`)
changes.

Stories themselves are created, described and ordered in **Backlog**. The sprint's burndown chart is
on **Sprint burndown (Burndown de sprints)** under Reports.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **sprint** has a name, an optional **Goal (Objetivo)**, a start and an end date, and a state:
  **Planned (Planificado)** → **Active (Activo)** → **Closed (Cerrado)**. A project has at most one
  active sprint at a time; a closed sprint is read only.
- Inside a sprint each story has its own state: **Pending (Pendiente)**, **In development (En
  desarrollo)**, **Blocked (Bloqueado)**, **In review (En revisión)** or **Finished (Finalizado)**.
  The story's state shown everywhere else is its state in the sprint it was most recently added to.
- The board has four columns: **Pendiente**, **En desarrollo**, **Bloqueado / En revisión** (two
  stacked boxes, Bloqueado on top in red) and **Finalizado**.
- **Committed hours** are the estimated hours of the sprint's stories; the progress shows the hours
  and stories finished against them.
- A story left unfinished when its sprint closes is **Not completed (No culminada)** on that sprint's
  board.

<!-- DOC-ID: capability.sprint-list -->
## See a project's sprints (Lista de sprints)

### User intention (Intención del usuario)

Check which sprints exist and how far each one got.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Projects (Proyectos) → Sprints** at `/projects/sprints`. The project is
chosen in the header search box ("P01 · Name"); the last project opened is reopened. Each sprint is a
card; clicking it opens its board.

### Business rules and rationale (Reglas y razón de negocio)

- Order: the active sprint first, then planned ones by start date, then closed ones, latest first.
- Each card shows the state badge, the dates, the goal, finished stories over total stories, and
  finished hours over committed hours with a progress bar.
- Users outside the team see "No estás en el equipo de este proyecto: solo lectura."; an archived
  project shows "Proyecto archivado: solo lectura."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué sprint está activo? ¿Cuánto avanzó el sprint? Sprint, iteración, ciclo.

<!-- DOC-ID: capability.plan-sprint -->
## Plan a sprint (Planificar un sprint)

### User intention (Intención del usuario)

Create the next sprint, or change an open sprint's dates, name, goal and stories.

### Where to find it (Dónde encontrarlo)

- **New sprint (Nuevo sprint)** on the sprint list creates one.
- The pencil **Plan … (Planificar …)** on a planned or active sprint's card edits it.

The planner shows a **Backlog** column on the left and the plan on the right: the name and goal with
the pencil **Edit the sprint's name and goal (Editar el nombre y el objetivo del sprint)**, the
**Start (Inicio)** and **End (Fin)** dates, the view **By state (Por estado)** / **By assignee (Por
responsable)**, and the total "N historias · X h". Save with **Guardar**; the × button leaves without
saving.

### Required information and prerequisites (Requisitos previos)

- Only the project's admins plan sprints, on a project not archived.
- A new sprint starts the day after the latest sprint ends (today when there is none or it already
  ended) and lasts as long as that sprint did, or 14 days for the first one.
- The modal **Sprint name and goal (Nombre y objetivo del sprint)** edits **Name (Nombre)** and
  **Goal (Objetivo)**; **Aplicar** applies them to the plan. An empty name becomes "Sprint N"
  (`Sprint N (automático)`).

### Business rules and rationale (Reglas y razón de negocio)

- The Backlog column offers only stories that are not finished and not in another open sprint, in
  backlog order, with **Filter stories (Filtrar historias)**, **Épica** and **Tipo** filters, and the
  count "historias planificables".
- On a computer a story is added by dragging it onto the plan and removed by dragging it back or
  with × **Take … out of the sprint (Quitar … del sprint)**. On a phone the + button **Add … to the
  sprint (Agregar … al sprint)** adds it.
- **Por estado** groups the plan by state; dropping a story on a group gives it that state, and
  Bloqueado asks **Why is it stopped? (¿Por qué se detiene?)** first. A story just added is
  Pendiente.
- **Por responsable** shows one group per team member plus **Sin responsable**; dropping a story on a
  person assigns it to them.
- The name can have up to 80 characters and the goal up to 500; the end date cannot be earlier than
  the start.
- A finished story, or one already in another open sprint, cannot join.
- Clicking a story in the planner opens its form; what is saved there is saved at once, apart from
  the plan.

### Result and side effects (Resultado y efectos)

Saving writes the sprint, its stories, and the states and assignees set while planning, then opens
the sprint's board. Each story added or removed gets "Agregada a …" or "Quitada de …" in its history,
and its state in the backlog follows the sprint.

### Limitations (Limitaciones)

- A closed sprint cannot be planned or changed.
- The dates are not checked against other sprints.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un sprint? ¿Cómo agrego historias al sprint? ¿Puedo agregar una historia al sprint
  activo? → yes, planning the active sprint.
- ¿Por qué no aparece una historia para planificar? → it is finished or in another open sprint.
- Sprint planning, planificación, compromiso del sprint.

<!-- DOC-ID: capability.delete-sprint -->
## Delete a planned sprint (Eliminar un sprint)

### Where to find it (Dónde encontrarlo)

In the planner of a saved, planned sprint, the trash button **Deletes this planned sprint. (Elimina
este sprint planificado.)**, then confirm "¿Eliminar …? Sus historias vuelven al backlog."

### Business rules and rationale (Reglas y razón de negocio)

Only a planned sprint can be deleted, by a project admin. An active or closed sprint cannot.

### Result and side effects (Resultado y efectos)

Its stories go back to the backlog, each with "Quitada de …" in its history.

<!-- DOC-ID: capability.start-sprint -->
## Start a sprint (Iniciar un sprint)

### Where to find it (Dónde encontrarlo)

Open the planned sprint's board → the menu button **Opens the sprint's menu (Abre el menú del
sprint)** → **Start sprint (Iniciar sprint)**.

### Business rules and rationale (Reglas y razón de negocio)

- Only a project admin starts a sprint, and only a planned one.
- Only one sprint is active per project: starting another is refused with "… sigue activo:
  ciérralo primero."

### Result and side effects (Resultado y efectos)

The sprint becomes **Activo** and its cards can move. While planned, the board says "El sprint no ha
iniciado: sus tarjetas se mueven cuando inicie."

<!-- DOC-ID: capability.board -->
## Move stories on the sprint board (Mover historias en el tablero)

### User intention (Intención del usuario)

Report progress: take a story into development, block it, send it to review, finish it.

### Where to find it (Dónde encontrarlo)

Click a sprint card to open its board. Drag a card to another column or position. On a phone, the
button **Move … (Mover …)** on the card opens **Move to (Mover a)** with the states and **Mover**.
Clicking a card opens the story's panel (Detail and Activity).

### Business rules and rationale (Reglas y razón de negocio)

- Any team member can move cards, but only on the active sprint's board. A closed sprint says
  "Sprint cerrado: solo lectura."
- Moving a card to **Bloqueado** asks **Why is it stopped? (¿Por qué se detiene?)**; the reason is
  required (up to 1000 characters) and is shown in the story's history.
- Moves show at once, with a spinner while saved, and are saved in the order they were made; two
  people moving cards at the same time both land. If a save fails, the board is read again.
- The board refreshes itself every 30 seconds while it is on screen, and when the browser window
  regains focus, so other people's moves appear.

### Result and side effects (Resultado y efectos)

A change of column writes "Pasó de … a … en …" in the story's history and updates the story's state
in the backlog and the reports. Reordering inside a column changes only the order. The day a story
enters each state feeds the Gantt; the moment it is finished feeds the calendar and the burndown.

### Limitations (Limitaciones)

The story form has no state field: the board (or the planner) is the only way to change a state.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo paso una historia a En desarrollo o a Finalizado? ¿Cómo bloqueo una tarea?
- Tablero, kanban, scrum board, columna, tarjeta, mover, bloqueado, impedimento.

<!-- DOC-ID: capability.board-progress -->
## Follow the sprint's progress and filter the board (Avance del sprint)

### Where to find it (Dónde encontrarlo)

The board's header shows the state badge, the dates, "N días restantes" (or "Pasó su fecha de fin"
in red), "X / Y historias finalizadas" and a bar with the share of hours in each state. The avatars
**Filter by assignee (Filtrar por responsable)** show only the chosen people's cards. The sprint
menu shows **Progress (Avance)** (stories and hours finished, and count and hours per column) and
**Group the board (Agrupar el tablero)**: **Board (Tablero)** or **By assignee (Por responsable)**,
one board per person. **Reports (Reportes)** in the menu opens the sprint's burndown (not for a
planned sprint).

### Business rules and rationale (Reglas y razón de negocio)

- Days left count today and the end day.
- The people filter narrows the cards shown, but the progress always counts the whole sprint.
- An active sprint is not closed automatically when its end date passes; an admin closes it.

<!-- DOC-ID: capability.close-sprint -->
## Close a sprint (Cerrar un sprint)

### User intention (Intención del usuario)

End the active sprint and decide what happens to the work not finished.

### Where to find it (Dónde encontrarlo)

Board → sprint menu → **Close sprint (Cerrar sprint)**. The dialog **Close … (Cerrar …)** lists the
unfinished stories, each with **Goes to (Pasa a)**: a planned sprint or **Back to the backlog
(Volver al backlog)**. Confirm with **Cerrar sprint**.

### Business rules and rationale (Reglas y razón de negocio)

- Only a project admin closes, and only the active sprint.
- Each unfinished story is proposed for the earliest planned sprint, or the backlog when there is
  none. When no planned sprint exists, **Create next sprint (Crear siguiente sprint)** creates one
  right after this sprint and sends the stories there.
- A carried story keeps its state in the new sprint ("allí conserva su estado").
- When every story is finished, the sprint "se cierra tal cual".

### Result and side effects (Resultado y efectos)

The sprint becomes **Cerrado** and read only. Its unfinished cards stay on its board marked **No
culminada**. Each carried story gets "No culminada en …: pasó a …" or "…: volvió al backlog" in its
history; one sent back to the backlog shows the orange badge "No culminada en …" there.

### Limitations (Limitaciones)

A closed sprint cannot be reopened, edited or deleted.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cierro el sprint? ¿Qué pasa con las historias que no se terminaron? ¿Cómo paso las tareas
  pendientes al siguiente sprint?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"… sigue activo: ciérralo primero."**: close the active sprint before starting another.
- **"La historia "…" ya está en …"**: the story is in another open sprint; take it out there first.
- **"La historia "…" ya está finalizada."**: finished stories cannot join a sprint.
- **"Indica por qué se detiene la historia."**: write the reason for Bloqueado.
- **"El sprint necesita una fecha de inicio y una de fin posterior."**: fix the dates.
- **"Solo se puede eliminar un sprint planificado."** / **"Un sprint cerrado no se puede
  modificar."**: active and closed sprints keep their history.
- **"Solo se puede cambiar el tablero del sprint activo."**: start the sprint first.
- **"Solo los administradores del proyecto pueden hacer esto."**: planning, starting, closing and
  deleting sprints are for project admins.
- No **Nuevo sprint** button: the user is not an admin of the project, or a sprint's board is open.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Backlog** creates and orders the stories a sprint takes, and shows each story's sprint.
- **Projects (Proyectos)** defines who is an admin or member of the team.
- **Sprint burndown (Burndown de sprints)** charts the hours left of an active or closed sprint.
- **Gantt** and **Calendar (Calendario)** show the sprints over their dates; **By person (Por
  persona)** shows a person's weekly load from the sprints.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.sprint-list]
  - path: frontend/routes/projects/sprints/+page.svelte
    role: page
    hash: sha256:9cd61c2408e695ffe0654f02dc32febb4154c0a510e4f5afb0b3d8a5ba318909
    supports: [page-purpose, capability.sprint-list, capability.plan-sprint, capability.start-sprint, capability.board, troubleshooting]
  - path: frontend/routes/projects/sprints/SprintList.svelte
    role: user-interface
    hash: sha256:65441301ad8b260dda0862dc547d964bb5da3f5793e325b468042673a9e45c9d
    supports: [capability.sprint-list, capability.plan-sprint]
  - path: frontend/routes/projects/sprints/SprintPlanner.svelte
    role: user-interface
    hash: sha256:50bb6dd455b120f1a21944b7d41b14ae0aa5896279a6e8d08d6ac26738417438
    supports: [capability.plan-sprint, capability.delete-sprint]
  - path: frontend/routes/projects/sprints/SprintBoard.svelte
    role: user-interface
    hash: sha256:2353b429412da6fcf7c54bef220f20d37d802ad7c638d86c38eda0d259a43bca
    supports: [concepts, capability.start-sprint, capability.board, capability.board-progress]
  - path: frontend/routes/projects/sprints/SprintCard.svelte
    role: user-interface
    hash: sha256:4d9418aa09f700a4f38a8ca063dbc0200118d37ffc83b86b0e48584be83c00e2
    supports: [capability.board, capability.close-sprint]
  - path: frontend/routes/projects/sprints/SprintMenu.svelte
    role: user-interface
    hash: sha256:eaad94a06e3df2ee7e445071858fb937b95b5130ce16f2af5eb9bc223a8c460a
    supports: [capability.start-sprint, capability.board-progress, capability.close-sprint]
  - path: frontend/routes/projects/sprints/SprintStateBar.svelte
    role: user-interface
    hash: sha256:3dae68f370b4e650acc006aa9832de982e384433fac718f0e413a077013729cf
    supports: [capability.board-progress]
  - path: frontend/routes/projects/sprints/CardMoveModal.svelte
    role: user-interface
    hash: sha256:ef57e550fd4fff13bb3231944b4593ca5cd7c21af0de1ab9a2a7c2e463e36b2b
    supports: [capability.plan-sprint, capability.board]
  - path: frontend/routes/projects/sprints/SprintCloseModal.svelte
    role: user-interface
    hash: sha256:542c4d6f580750d45ed1c184c6a355eaef3f6ecaaba4e71c3c51fc829ec67ad9
    supports: [capability.close-sprint]
  - path: frontend/routes/projects/sprints/sprints.ts
    role: business-logic
    hash: sha256:5c8fe6ad195272897b05b9c85e31838bbc857c299b97645f16227ade31971213
    supports: [concepts, capability.sprint-list, capability.plan-sprint, capability.board, capability.board-progress, capability.close-sprint]
  - path: frontend/routes/projects/sprints/sprints.svelte.ts
    role: frontend-service
    hash: sha256:26f57feb392fbc9ff24d6b1a9ed7696db65f48f78323efec092f2a39f67bf70e
    supports: [capability.plan-sprint, capability.delete-sprint, capability.start-sprint, capability.board, capability.close-sprint]
  - path: frontend/routes/projects/project.ts
    role: shared-domain
    hash: sha256:2a960298b661517ac43a5a1fc6706c0596d993288edc033c4e7ff25cea11a202
    supports: [concepts]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.sprint-list, capability.board]
  - path: frontend/routes/projects/story/StoryPanel.svelte
    role: user-interface
    hash: sha256:2b344d7e9dbaae834b0f101b6b6e918025222ac160cad5929e4e25aaff8d587d
    supports: [capability.plan-sprint, capability.board]
  - path: backend/projects/sprints.go
    role: backend-handler
    hash: sha256:0bb8d17f6c300cc0014a35ebe01e088b454d03d43a3e4b9b6afeb075ffcc091c
    supports: [capability.plan-sprint, capability.delete-sprint, capability.start-sprint, capability.board, capability.close-sprint, troubleshooting]
  - path: backend/projects/sprint_rules.go
    role: business-logic
    hash: sha256:620521e71cb9f322a4cfc756f7e435330b9c01bb7f180eb44e58a2a55ba244b2
    supports: [concepts, capability.plan-sprint, capability.delete-sprint, capability.start-sprint, capability.board, capability.close-sprint, troubleshooting]
  - path: backend/projects/story_rules.go
    role: business-logic
    hash: sha256:d6397534ef09ee0bfc9afcff78bb6064eb71ddf4d8405be51ded966beaf3c5e9
    supports: [concepts, capability.board]
  - path: backend/projects/project_access.go
    role: business-logic
    hash: sha256:7dad83cea78e4d9da0d9d386b5cea012143a095ca13a91153cd07aaf16db2659
    supports: [capability.sprint-list, troubleshooting]
  - path: backend/projects/types/sprints.go
    role: data-model
    hash: sha256:5e07f17e54d7644084cd5583b24cb303fda3ae371374fa9611725c5b43ed8511
    supports: [concepts, capability.close-sprint]
```
