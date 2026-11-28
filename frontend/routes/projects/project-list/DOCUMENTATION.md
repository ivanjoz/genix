---
schema: 1
page_id: projects.project-list
route: /projects/project-list
title: Projects (Proyectos)
status: implemented
visibility: tenant
description_en: >-
  List of the company's agile projects as cards. Create a project with its name, description,
  dates, color and team (members and admins), edit or archive it, and maintain its epics. A card
  opens the project's backlog.
description_es: >-
  Lista de los proyectos ágiles de la empresa en tarjetas. Crear un proyecto con nombre,
  descripción, fechas, color y equipo (miembros y administradores), editarlo o archivarlo, y
  mantener sus épicas. Cada tarjeta abre el backlog del proyecto.
---

# Projects (Proyectos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Projects (`Proyectos`) is the entry point of the agile projects module (`proyectos ágiles`,
`gestión de proyectos`). It lists every project of the company as a card and is where a project
is created (`crear proyecto`), its team (`equipo`) is defined, it is archived (`archivar`), and its
epics (`épicas`) are maintained.

The page does not hold the work itself. The stories (`historias`, `tareas`, `bugs`) are created and
ordered in **Backlog**, planned and moved in **Sprints**, and the project's files live in
**Documents (Documentos)**. Clicking a project card opens that project's Backlog.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **project** (`proyecto`) has a code built from its number, such as **P01** or **P02**, a name, an
  optional description, optional **Start (Inicio)** and **End (Fin)** dates, a color, and a team.
- The **team** (`equipo`) is the list of users who work in the project. Each one is **Member
  (Miembro)** or **Admin (Administrador)**. Members work on the backlog and the sprint board; admins
  also edit the project and its team, delete stories and run the sprints (plan, start, close).
- Every user with access to the Projects module sees every project of the company, including the
  ones whose team they are not on; those projects are read only for them (`solo lectura`).
- An **epic** (`épica`) is a named, colored group of stories inside one project. A story has one epic
  or none (`Sin épica`).
- An **archived** project (`proyecto archivado`) is read only and hidden from the list by default.

<!-- DOC-ID: capability.create-project -->
## Create a project (Crear un proyecto)

### User intention (Intención del usuario)

Start a new agile project so its team can register stories, plan sprints and keep files.

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Projects (Proyectos) → Projects (Proyectos)** at `/projects/project-list`.
Use **New project (Nuevo proyecto)**; the side panel **New project (Nuevo proyecto)** opens on the
**Information (Información)** tab. Save with **Guardar**.

### Required information and prerequisites (Requisitos previos)

- Only users holding the Edit level of the Projects access see **New project (Nuevo proyecto)**.
- **Name (Nombre):** required, up to 120 characters.
- Optional: **Description (Descripción)** (up to 2000 characters), **Start (Inicio)**, **End (Fin)**
  and **Color** (Azul, Verde, Rojo, Ámbar, Morado, Turquesa, Rosado, Gris; Azul by default).
- **Team (Equipo):** add people with **Add a member (Agregar integrante)** and choose each one's role
  (**Miembro** or **Administrador**). The creator starts in the team as Administrador.

### Business rules and rationale (Reglas y razón de negocio)

- The end date cannot be earlier than the start date.
- The team needs at least one Administrador, so the project can always be administered.
- Only active users holding the Edit level of the Projects access can join a team; the picker offers
  only them, as its note says: "Solo los usuarios con el nivel Editar de Proyectos pueden integrar un
  equipo." A user without that level could never write in the project.
- The creator is always kept as Administrador of a new project.

### Result and side effects (Resultado y efectos)

The project gets its code (P01, P02…) and appears as a card. The panel stays open after saving, and
the **Epics (Épicas)** tab becomes available to add the first epics.

### Limitations (Limitaciones)

- Epics cannot be added before the project is saved.
- There is no delete button for a project; archive it instead.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un proyecto? ¿Dónde registro un nuevo proyecto ágil o scrum?
- ¿Por qué no aparece una persona para agregarla al equipo? → it lacks the Edit level of Projects.
- ¿Por qué no veo el botón Nuevo proyecto? → the user only has the View level of Projects.
- Proyecto, equipo, integrante, miembro, administrador, scrum, ágil.

<!-- DOC-ID: capability.edit-project -->
## Edit a project, its team or archive it (Editar o archivar un proyecto)

### User intention (Intención del usuario)

Change the project's data, add or remove people, change roles, or archive a finished project.

### Where to find it (Dónde encontrarlo)

On the project card, the pencil button **Edit project … (Editar proyecto …)**, shown only to the
project's admins. The side panel opens on **Information (Información)**; the checkbox **Archived
(read only, hidden from the list) (Archivado (solo lectura, oculto de la lista))** appears there for a
saved project. Save with **Guardar**.

### Required information and prerequisites (Requisitos previos)

The user must be an Administrador of that project's team. The same rules as on creation apply: a
name, an end not before the start, and at least one Administrador.

### Business rules and rationale (Reglas y razón de negocio)

- Only the people added in this edit are checked for the Edit level of Projects. A current member
  who has since lost that level can still be removed or changed to Miembro.
- A person is removed from the team with the trash button **Remove … from the team (Quitar a … del
  equipo)**.
- Ticking **Archivado** makes the project read only for everyone and hides its card unless **Show
  archived (Mostrar archivados)** is ticked.

### Result and side effects (Resultado y efectos)

The card shows the new name, color, dates and team. An archived project shows the **Archived
(Archivado)** badge, is marked "(archivado)" in the project pickers of the other project pages, and
those pages show "Proyecto archivado: solo lectura."

### Limitations (Limitaciones)

- Archiving cannot be undone from this page: once a project is archived, every save of it is refused
  with "El proyecto está archivado: es de solo lectura.", including unticking **Archivado**.
- Members who are not admins cannot edit the project; they only see the card.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo agrego a alguien al equipo del proyecto? ¿Cómo hago administrador a un miembro?
- ¿Cómo archivo un proyecto terminado? ¿Puedo desarchivar un proyecto?
- ¿Cómo elimino un proyecto? → there is no deletion; archive it.

<!-- DOC-ID: capability.manage-epics -->
## Maintain the project's epics (Gestionar las épicas)

### User intention (Intención del usuario)

Create, rename, recolor or remove the epics that group the project's stories in the backlog.

### Where to find it (Dónde encontrarlo)

In the project panel, tab **Epics (Épicas)**. **Épica** adds a row; each row has a color picker and
the name field **Epic name (Nombre de la épica)**; the trash button **Remove the epic … (Quitar la
épica …)** removes it. Save with **Guardar**. The same panel opens from Backlog with the pencil
**Edit the project's epics (Editar las épicas del proyecto)**.

### Required information and prerequisites (Requisitos previos)

The project must be saved and not archived, and the user must be on its team. Every kept epic needs a
name of up to 80 characters.

### Business rules and rationale (Reglas y razón de negocio)

- Any team member can create, rename and recolor epics; only the project's admins can remove an epic
  that was already saved. An epic not saved yet can be dropped by anyone editing.
- Changes are written only on **Guardar**, as the tab's note says. If nothing changed, Genix answers
  "Ninguna épica cambió".
- Up to 50 epics are saved at a time.

### Result and side effects (Resultado y efectos)

New epics appear as groups in Backlog and Gantt. Removing an epic does not delete its stories: they
stay in the project without an epic (`Sin épica`).

### Limitations (Limitaciones)

The epics tab is not shown for an archived project, nor to users outside the team.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo una épica? ¿Qué pasa con las historias si borro una épica?
- Épica, módulo, agrupador, categoría de historias.

<!-- DOC-ID: capability.browse-projects -->
## Find and open a project (Buscar y abrir un proyecto)

### User intention (Intención del usuario)

Locate a project and go to its work.

### Where to find it (Dónde encontrarlo)

The filter **Filter projects (Filtrar proyectos)** and the checkbox **Show archived (Mostrar
archivados)** at the top. Clicking a card opens the project's Backlog.

### Business rules and rationale (Reglas y razón de negocio)

- The filter matches the project code (P01), the name and the description.
- Active projects are listed first, then archived ones, each by name.
- Each card shows the code, the badges **Archivado** and the user's own role (**Administrador** or
  **Miembro**), the description, up to six team avatars plus "+N" (admins in solid color, members in
  gray), and the dates.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde están mis proyectos? ¿Por qué no veo un proyecto? → it may be archived; tick Mostrar
  archivados.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El proyecto necesita un nombre de hasta 120 caracteres."**: write a name, shorter than 120
  characters.
- **"El proyecto no puede terminar antes de empezar."**: correct **Inicio** or **Fin**.
- **"El equipo necesita al menos un administrador."**: give the Administrador role to someone
  before removing or demoting the last one.
- **"El usuario … no tiene el acceso de escritura a Proyectos."**: the person needs the Edit level of
  the Projects access, granted in Users & Profiles.
- **"El proyecto está archivado: es de solo lectura."**: an archived project cannot be changed.
- **"Solo los administradores del proyecto pueden hacer esto."**: ask an Administrador of the team.
- **"Cada épica necesita un nombre de hasta 80 caracteres."**: name every epic row or remove it.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Backlog** (`/projects/backlog`) lists and orders the project's stories by epic; a project card
  opens it.
- **Sprints** (`/projects/sprints`) plans the sprints and moves stories on the board.
- **Documents (Documentos)** (`/projects/documents`) keeps the project's files.
- **Gantt**, **Calendar (Calendario)**, **Sprint burndown (Burndown de sprints)** and **By person
  (Por persona)**, under Reports (Reportes), show the project's progress.
- **Users & Profiles (Usuarios & Perfiles)** grants the Projects access level a person needs to join
  a team.

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
    supports: [capability.create-project]
  - path: frontend/routes/projects/project-list/+page.svelte
    role: page
    hash: sha256:85c2793081991ac96329044d15f136ad83b3fec858ddfb1dc0b2199896f0a20e
    supports: [page-purpose, capability.create-project, capability.edit-project, capability.browse-projects]
  - path: frontend/routes/projects/project-list/ProjectCard.svelte
    role: user-interface
    hash: sha256:3e76a90cb732b0c3b327b36e99c733014fb76f662b64d88eebf64839bac1beca
    supports: [capability.edit-project, capability.browse-projects]
  - path: frontend/routes/projects/project-list/project-list.ts
    role: business-logic
    hash: sha256:932bb02ceef3da34d65166b9ceaf3ae0f5872f22edf8e8664313c59b7313f1fc
    supports: [capability.browse-projects]
  - path: frontend/routes/projects/ProjectPanel.svelte
    role: user-interface
    hash: sha256:6eb1fe58419f6222e63bb7e26af82fd9792067a8f6ad91a4bb4cc7b2ddc3c086
    supports: [capability.create-project, capability.edit-project, capability.manage-epics]
  - path: frontend/routes/projects/ProjectForm.svelte
    role: user-interface
    hash: sha256:c7cc685204e35c6d705af0d5d65ecd26ee6d4ffe944f5e0edfffbb9d82bbd448
    supports: [capability.create-project, capability.edit-project]
  - path: frontend/routes/projects/ProjectEpicsForm.svelte
    role: user-interface
    hash: sha256:e45fa111abec858d81a3d2e3ef0fa9ddf46796c0a5bab3b68c260f8e8eae8420
    supports: [capability.manage-epics]
  - path: frontend/routes/projects/project.ts
    role: shared-domain
    hash: sha256:2a960298b661517ac43a5a1fc6706c0596d993288edc033c4e7ff25cea11a202
    supports: [concepts, capability.create-project, capability.edit-project, capability.manage-epics, capability.browse-projects]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.edit-project, capability.manage-epics]
  - path: backend/projects/projects.go
    role: backend-handler
    hash: sha256:7b37f386fee7b9121e4372bd8b9bf07ec735621a90c0ba039664be37f0f39a1e
    supports: [concepts, capability.create-project, capability.edit-project, troubleshooting]
  - path: backend/projects/project_rules.go
    role: business-logic
    hash: sha256:a429659567cf05302acb0104f063f8406050775f82b95327ff710a0caa7b5ff7
    supports: [capability.create-project, capability.edit-project, capability.manage-epics, troubleshooting]
  - path: backend/projects/project_access.go
    role: business-logic
    hash: sha256:7dad83cea78e4d9da0d9d386b5cea012143a095ca13a91153cd07aaf16db2659
    supports: [concepts, capability.create-project, capability.edit-project, troubleshooting]
  - path: backend/projects/stories.go
    role: backend-handler
    hash: sha256:9b6f2cf5a86a59be861fd8768a2ac699ba783504a2d08f9664e0322d2177b78f
    supports: [capability.manage-epics]
  - path: backend/projects/types/projects.go
    role: data-model
    hash: sha256:dc7f0c5491eb811c5b2e0d09b33cf2c64d0ea899245446ba6eba1e8b0d8a066a
    supports: [concepts, capability.edit-project]
```
