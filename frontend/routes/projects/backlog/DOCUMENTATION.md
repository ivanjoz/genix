---
schema: 1
page_id: projects.backlog
route: /projects/backlog
title: Backlog (Backlog)
status: implemented
visibility: tenant
description_en: >-
  A project's backlog: its stories, bugs and tasks grouped by epic in priority order. Create and edit
  stories with type, priority, estimated hours, epic, assignee, description, files and acceptance
  criteria; comment on them and read their history; filter the list and drag stories to reorder them
  or move them to another epic.
description_es: >-
  Backlog de un proyecto: sus historias, bugs y tareas agrupadas por épica en orden de prioridad.
  Crear y editar historias con tipo, prioridad, horas estimadas, épica, responsable, descripción,
  archivos y criterios de aceptación; comentarlas y ver su historial; filtrar la lista y arrastrar
  historias para ordenarlas o pasarlas a otra épica.
---

# Backlog (Backlog)

<!-- DOC-ID: page-purpose -->
## Page purpose

Backlog keeps the list of work of one project (`lista de pendientes del proyecto`): its stories
(`historias de usuario`), bugs and tasks (`tareas`), grouped by epic (`épica`) and ordered by the
team's priority. It is where a story is created, described, estimated, assigned, commented and
ordered.

The Backlog does not plan sprints or change a story's state (`estado`): stories enter a sprint and
move between Pendiente, En desarrollo, Bloqueado, En revisión and Finalizado on the **Sprints**
page. Projects, teams and archiving belong to **Projects (Proyectos)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **story** (`historia`) has a code **H** plus its number inside the project (H01, H07…), a
  **Type (Tipo)**: **Story (Historia)**, **Bug** or **Task (Tarea)**, and a **Priority (Prioridad)**:
  **Low (Baja)**, **Medium (Media)**, **High (Alta)** or **Critical (Crítica)**.
- **Estimated hours (Horas estimadas)** is the only effort figure a story holds; the totals of the
  backlog, the sprints and the reports add these estimates.
- The **assignee** (`responsable`) is one member of the project's team, or nobody (`Sin
  responsable`).
- A story's **state** (`estado`) is its state in the sprint it was most recently added to. A story
  that was never in a sprint is **Not planned (Sin planificar)**. Other states: **Pending
  (Pendiente)**, **In development (En desarrollo)**, **Blocked (Bloqueado)**, **In review (En
  revisión)** and **Finished (Finalizado)**.
- The **sprint badge** of a story names the open sprint it is in. When it came back unfinished from a
  closed sprint it reads **Not completed in … (No culminada en …)** in orange.
- **Acceptance criteria** (`criterios de aceptación`) are a checklist inside the story, each item
  marked done or not.

<!-- DOC-ID: capability.open-project-backlog -->
## Open a project's backlog (Abrir el backlog de un proyecto)

### User intention (Intención del usuario)

See the stories of a given project.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Projects (Proyectos) → Backlog** at `/projects/backlog`, or by clicking a
project card on the Projects page. The project is chosen in the header search box, whose options
read "P01 · Name" ("(archivado)" for archived projects). Without a chosen project the page says
"Elige un proyecto."

### Business rules and rationale (Reglas y razón de negocio)

- The browser remembers the last project opened in Backlog, Sprints, Documents, Gantt, Calendar or the sprint burndown, and
  reopens it when the page is opened without a project.
- The summary line shows how many stories are listed and **Estimated (Estimado)**: the sum of their
  estimated hours. Each epic group shows its count and hours and can be collapsed.
- Groups follow the epics' creation order; **No epic (Sin épica)** is always last.
- Users outside the project's team see "No estás en el equipo de este proyecto: solo lectura.", and
  an archived project shows "Proyecto archivado: solo lectura."; neither can create, edit, comment or
  reorder.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde veo las historias de un proyecto? ¿Dónde está la lista de pendientes o tareas del proyecto?
- Backlog, product backlog, lista de historias, pila del producto.

<!-- DOC-ID: capability.create-edit-story -->
## Create or edit a story (Crear o editar una historia)

### User intention (Intención del usuario)

Register a new piece of work, or complete and correct an existing one.

### Where to find it (Dónde encontrarlo)

**Story (Historia)** at the top creates a story; with an epic chosen in the filters it starts in that
epic. **Story (Historia)** in an epic group's header creates it in that epic. Clicking a story row
opens it. The side panel **New story (Nueva historia)**, or "H07 · Title" for an existing one, opens
on **Detail (Detalle)**. Save with **Guardar**.

### Required information and prerequisites (Requisitos previos)

- **Title (Título):** required, up to 200 characters.
- **Type (Tipo)** (Historia by default) and **Priority (Prioridad)** (Media by default).
- Optional: **Estimated hours (Horas estimadas)**, **Epic (Épica)**, **Assignee (Responsable)**,
  **Short description (Descripción corta)** (up to 500 characters), the long description editor,
  attached files, and **Acceptance criteria (Criterios de aceptación)**.
- The user must be on the project's team and the project must not be archived.

### Business rules and rationale (Reglas y razón de negocio)

- Estimated hours go from 0 to 2000 in steps of 0.5.
- The epic must be one of the project's epics, and the assignee must be on the project's team.
- Images cannot be pasted into the description: attach them as files.
- Each attached file can weigh up to 4 MB, with up to 30 files per story. Saving waits until uploads
  finish: "Espera a que terminen de subir los archivos."
- Acceptance criteria allow up to 100 items of up to 300 characters; empty items are dropped. The
  list shows how many are **Done (Cumplidos)**.

### Result and side effects (Resultado y efectos)

- A new story gets the next number of the project ("Historia H07 creada"), lands last in the backlog
  and starts **Sin planificar**.
- Changing the assignee writes "Asignada a …" or "Quedó sin responsable" in the story's history and
  moves the story into that person's list in the **By person (Por persona)** report.
- Files added to the story also appear on the **Documents (Documentos)** page under "Archivos en las
  historias". A file removed from the story is deleted from storage and leaves that page.

### Limitations (Limitaciones)

- The form has no state field: a story's state changes only on the Sprints page, in the sprint board
  or while planning a sprint.
- Users outside the team see the form disabled and cannot save.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo una historia de usuario? ¿Cómo registro un bug o una tarea? ¿Cómo asigno una historia?
- ¿Dónde pongo los criterios de aceptación? ¿Cómo adjunto un archivo o una imagen a una historia?
- ¿Cómo cambio el estado de una historia? → on the Sprints board.
- Historia, user story, tarea, bug, incidencia, estimación, horas estimadas, responsable, asignado.

<!-- DOC-ID: capability.story-activity -->
## Comment on a story and read its history (Comentarios e historial)

### User intention (Intención del usuario)

Discuss a story with the team, share files about it, and see what happened to it.

### Where to find it (Dónde encontrarlo)

Open a saved story → tab **Activity (Actividad)**. Write in the box "Escribe un comentario o
arrastra un archivo", attach with the paperclip (files of up to 4 MB each) or drop files on the box,
and send with the paper-plane button.

### Business rules and rationale (Reglas y razón de negocio)

- A comment needs text, files or both; text goes up to 5000 characters.
- Only the project's team comments, on a project not archived.
- The timeline lists the newest entry first and mixes comments with history lines: state changes
  ("Pasó de … a … en Sprint …", with the reason under a move to Bloqueado), assignee changes,
  "Agregada a …" and "Quitada de …" a sprint, and carry-overs when a sprint closes ("No culminada en
  …: pasó a …" or "…: volvió al backlog").

### Result and side effects (Resultado y efectos)

The story's comment counter grows (shown with a speech-bubble icon on its row). Files attached to a
comment also appear on the Documents page, marked "En un comentario".

### Limitations (Limitaciones)

Comments and history entries cannot be edited or deleted.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo comento una historia? ¿Dónde veo quién cambió el estado? ¿Por qué se bloqueó esta historia?
- Comentario, actividad, historial, bitácora.

<!-- DOC-ID: capability.delete-story -->
## Delete a story (Eliminar una historia)

### User intention (Intención del usuario)

Remove a story created by mistake or no longer needed.

### Where to find it (Dónde encontrarlo)

Open the story → the delete button **Eliminar** on the panel's title, then confirm "¿Eliminar H07
"…"? También sale de los sprints abiertos."

### Business rules and rationale (Reglas y razón de negocio)

Only the project's admins delete stories.

### Result and side effects (Resultado y efectos)

The story leaves the backlog and every open (planned or active) sprint, and leaves its assignee's
list in the By person report. Closed sprints keep it, since they are history.

### Limitations (Limitaciones)

There is no restore option on the page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué no puedo eliminar una historia? → only project admins can.

<!-- DOC-ID: capability.filter-backlog -->
## Filter the backlog (Filtrar el backlog)

### User intention (Intención del usuario)

Find stories, or see only part of the backlog.

### Where to find it (Dónde encontrarlo)

The filter button opens **Type (Tipo)**, **State (Estado)**, **Epic (Épica)**, **Assignee
(Responsable)** and **To plan only (Solo por planificar)**. Next to it, **Filter stories (Filtrar
historias)** searches text. Active filters show as removable tags in the summary line, and the filter
button changes color.

### Business rules and rationale (Reglas y razón de negocio)

- The text matches the story code, title and short description.
- State offers **Sin planificar**; Epic offers **Sin épica**; Assignee offers **Sin responsable**.
- **Solo por planificar** keeps the stories a sprint can still take: not finished and not in an open
  sprint.
- While filtering, empty epics are hidden.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué historias faltan planificar? ¿Qué tareas tiene asignadas Juan? ¿Qué bugs hay?

<!-- DOC-ID: capability.reorder-backlog -->
## Reorder stories or move them to another epic (Ordenar el backlog)

### User intention (Intención del usuario)

Set the priority order of the work, or regroup a story under another epic.

### Where to find it (Dónde encontrarlo)

Drag a story row and drop it between two others, in the same or another epic group, as the hint
says: "Arrastra las historias para ordenarlas o moverlas a otra épica."

### Business rules and rationale (Reglas y razón de negocio)

- Dragging works only for the team, with no filter active, and not on a phone.
- Dropping into another group changes the story's epic; dropping into **Sin épica** removes it.
- The move shows at once with a spinner on the row while it is saved; moves are saved in the order
  they were made. If a save fails, the backlog is read again and shows what is stored.

### Result and side effects (Resultado y efectos)

The order is shared by the team and is the order the sprint planner offers stories in.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo priorizo el backlog? ¿Por qué no puedo arrastrar las historias? → clear the filters.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"La historia necesita un título de hasta 200 caracteres."**: write a title.
- **"Las horas estimadas deben estar entre 0 y 2000, en pasos de 0.5."**: use values like 2, 2.5, 3.
- **"El responsable debe ser parte del equipo del proyecto."**: add the person to the team on the
  Projects page first.
- **"No se permiten imágenes pegadas en la descripción: adjúntalas como archivos."**: remove the
  pasted image and attach it.
- **"Un archivo puede pesar hasta 4 MB."** or **"Se pueden adjuntar hasta 30 archivos."**: reduce or
  split the files.
- **"Solo el equipo del proyecto puede hacer esto."** / **"Solo los administradores del proyecto
  pueden hacer esto."**: ask an admin to add you to the team or to do it.
- Reordering is not available: clear the filters, and use a computer instead of a phone.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Projects (Proyectos)** creates the project, its team and its epics.
- **Sprints** puts backlog stories into a sprint and moves them across the board's states.
- **Documents (Documentos)** lists every file attached to the stories and the project's own files.
- **Gantt** and **Calendar (Calendario)** show the stories over the sprints' dates; **By person (Por
  persona)** lists the stories assigned to someone across projects.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.open-project-backlog]
  - path: frontend/routes/projects/backlog/+page.svelte
    role: page
    hash: sha256:7f4f9f41df0201a2c47c55529505b85642f6a45e3558f214ee1cfb78bd4938e6
    supports: [page-purpose, capability.open-project-backlog, capability.create-edit-story]
  - path: frontend/routes/projects/backlog/BacklogView.svelte
    role: user-interface
    hash: sha256:818f08efe039a42bba57ba164f9453d13834defa7f06a4f6eef55956aeebb41f
    supports: [concepts, capability.open-project-backlog, capability.create-edit-story, capability.filter-backlog, capability.reorder-backlog]
  - path: frontend/routes/projects/backlog/BacklogFilters.svelte
    role: user-interface
    hash: sha256:80d61493c7de44dc7bb7f94d47bb69116f9b69543aa1f27d81a9b0ec4ddfb6b2
    supports: [capability.filter-backlog]
  - path: frontend/routes/projects/backlog/backlog.ts
    role: business-logic
    hash: sha256:8a14960fb8e2885b6f62250b62d8a23bfbc55e101645de63002f5059461680fe
    supports: [capability.open-project-backlog, capability.filter-backlog, capability.reorder-backlog]
  - path: frontend/routes/projects/story/StoryPanel.svelte
    role: user-interface
    hash: sha256:2b344d7e9dbaae834b0f101b6b6e918025222ac160cad5929e4e25aaff8d587d
    supports: [capability.create-edit-story, capability.story-activity, capability.delete-story, troubleshooting]
  - path: frontend/routes/projects/story/StoryForm.svelte
    role: user-interface
    hash: sha256:27007258a69a72be3d478f1a68f34732d5e3454831f940c4f1aa573158b55894
    supports: [capability.create-edit-story]
  - path: frontend/routes/projects/story/StoryTimeline.svelte
    role: user-interface
    hash: sha256:f56d0a6f662fc24606d355d1d8a98125cf2094d3e96ac6b104e0709df0268770
    supports: [capability.story-activity]
  - path: frontend/routes/projects/story/story.ts
    role: business-logic
    hash: sha256:4c8a8dbc658c2506591df9cebfbc915d877ac618ea4a09573094286dae06a399
    supports: [capability.create-edit-story, capability.story-activity]
  - path: frontend/routes/projects/story/story.svelte.ts
    role: frontend-service
    hash: sha256:7ea95bb8b6014506139d77d161088074ad6fb153a0a7dc535780e280ddec4014
    supports: [capability.create-edit-story, capability.story-activity]
  - path: frontend/routes/projects/project.ts
    role: shared-domain
    hash: sha256:2a960298b661517ac43a5a1fc6706c0596d993288edc033c4e7ff25cea11a202
    supports: [concepts, capability.open-project-backlog]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.open-project-backlog, capability.delete-story, capability.reorder-backlog]
  - path: frontend/domain-components/attachments/attachments.ts
    role: shared-domain
    hash: sha256:1ea1781343d19cad145c4862073f47d1e926df92d6302268fe49aba1169b783c
    supports: [capability.create-edit-story, troubleshooting]
  - path: backend/projects/stories.go
    role: backend-handler
    hash: sha256:9b6f2cf5a86a59be861fd8768a2ac699ba783504a2d08f9664e0322d2177b78f
    supports: [capability.create-edit-story, capability.story-activity, capability.delete-story, capability.reorder-backlog]
  - path: backend/projects/story_rules.go
    role: business-logic
    hash: sha256:d6397534ef09ee0bfc9afcff78bb6064eb71ddf4d8405be51ded966beaf3c5e9
    supports: [concepts, capability.create-edit-story, capability.story-activity, troubleshooting]
  - path: backend/projects/project_access.go
    role: business-logic
    hash: sha256:7dad83cea78e4d9da0d9d386b5cea012143a095ca13a91153cd07aaf16db2659
    supports: [capability.open-project-backlog, troubleshooting]
  - path: backend/projects/document_rules.go
    role: business-logic
    hash: sha256:8e1c88479c91031569dfa97e914bff1f9f1fb25869393efb80901be26c20ed7f
    supports: [capability.create-edit-story, capability.story-activity]
  - path: backend/projects/types/stories.go
    role: data-model
    hash: sha256:33d0fca372da7917d1abc4a46ba6410033bc4e4ca1f79d6cc487179617afbe8a
    supports: [concepts, capability.story-activity]
```
