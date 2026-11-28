---
schema: 1
page_id: projects.reports.by-person
route: /projects/reports/by-person
title: By person (Por persona)
status: implemented
visibility: tenant
description_en: >-
  The stories assigned to one person across every project in the last 180 days: as a list, as a
  Gantt of the sprints each story was in, and as a weekly calendar of estimated hours against a
  40-hour week that warns of overload.
description_es: >-
  Las historias asignadas a una persona en todos los proyectos en los últimos 180 días: en lista, en
  un Gantt de los sprints en que estuvo cada historia y en un calendario semanal de horas estimadas
  frente a una semana de 40 horas que avisa la sobrecarga.
---

# By person (Por persona)

<!-- DOC-ID: page-purpose -->
## Page purpose

By person (`Por persona`) shows the work assigned to one person (`carga de trabajo`, `asignaciones`,
`tareas de una persona`) across every project of the company, in three tabs: **List (Lista)**,
**Gantt** and **Calendar (Calendario)**. It answers "what does this person have" and "is their week
overloaded".

It is read only. Assigning a story to someone is done in the story's form (on **Backlog** or any page
that opens a story) or by dropping it on a person in the sprint planner (**Sprints**, view **Por
responsable**).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A story counts for the person who is its **assignee** (`responsable`) now. When the story is
  reassigned or deleted, it leaves this person's report.
- The report covers stories assigned to the person in the last 180 days, dated by when they were
  assigned to that person.
- Workload is measured in the stories' **estimated hours** (`horas estimadas`) against a capacity of
  40 hours per week.

<!-- DOC-ID: capability.choose-person -->
## Choose the person (Elegir la persona)

### Where to find it (Dónde encontrarlo)

**Projects (Proyectos) → Reports (Reportes) → By person (Por persona)** at
`/projects/reports/by-person`. The select **Person (Persona)** at the top; the note reads "Historias
asignadas en los últimos 180 días, en todos los proyectos."

### Business rules and rationale (Reglas y razón de negocio)

- The page opens on the current user. The chosen person stays in the address, so a reload or a shared
  link keeps it.
- The select offers the users who can join a project team (Edit level of the Projects access), plus
  the current user.
- There is no project picker: every project's stories are included, each row naming its project as
  "P01 · Name".

<!-- DOC-ID: capability.list -->
## List a person's stories (Lista)

### Where to find it (Dónde encontrarlo)

Tab **List (Lista)**, with columns **Project (Proyecto)**, **Story (Historia)** (the code, H07),
**Title (Título)**, **State (Estado)**, **Priority (Prioridad)** and **Assigned (Asignada)**, the date
and time it was assigned.

### Business rules and rationale (Reglas y razón de negocio)

- Newest assignment first.
- Deleted stories are left out. With nothing to show the table says "No se asignó ninguna historia a
  esta persona en los últimos 180 días."

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué tareas tiene asignadas María? ¿Cuáles son mis historias pendientes en todos los proyectos?

<!-- DOC-ID: capability.person-gantt -->
## See a person's stories over the sprints (Gantt)

### Where to find it (Dónde encontrarlo)

Tab **Gantt**, with the zoom **Weeks (Semanas)** / **Months (Meses)**.

### Business rules and rationale (Reglas y razón de negocio)

- One group per project, by name, whose bar spans its stories; under it each story with one bar per
  sprint it was in, colored by its state there, as on the project Gantt.
- Stories never in a sprint have no dates: they are left out and counted ("N historias sin planificar
  no se muestran").
- Clicking a bar opens that sprint's board on the Sprints page.

<!-- DOC-ID: capability.weekly-load -->
## See the weekly workload (Calendario semanal)

### User intention (Intención del usuario)

Check whether a person's weeks fit in 40 hours, and what they finished.

### Where to find it (Dónde encontrarlo)

Tab **Calendar (Calendario)**: four week columns, by default the previous week, this one (tagged
**This week (Esta semana)**) and the next two. **Previous week (Semana anterior)** and **Next week
(Semana siguiente)** move one week; **Today (Hoy)** returns.

### Business rules and rationale (Reglas y razón de negocio)

- Each week shows the assigned hours over 40 h, the **Finished (Finalizadas)** hours, and a meter;
  every meter shares one scale so the weeks compare at a glance. Above 40 h the hours turn red with a
  warning icon **Overload (Sobrecarga)**.
- Each story sits in exactly one week, decided by its latest sprint:
  - A finished story goes to the week it was moved to Finalizado.
  - An unfinished story goes to a week of its sprint from the current week on (the sprint's last week
    when the sprint is over). Weeks fill up to 40 h; a story that does not fit moves to the sprint's
    next week, and the sprint's last week takes whatever is left. An empty week takes a story even
    if it alone exceeds 40 h.
  - Among unfinished stories, the sprint that ends first fills the earliest weeks, and inside a
    sprint the stories already started go first, then the board order.
- Inside a week, stories are grouped by project and then by sprint, with an icon for the sprint's
  state (planned, active, closed), and show their code, title, state and estimated hours.
- Stories never in a sprint do not appear in the calendar.

### Limitations (Limitaciones)

The weekly placement is a projection from estimates and sprints, not a record of hours worked: Genix
projects record an estimate per story, not time spent.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Está sobrecargado Pedro esta semana? ¿Cuántas horas tiene asignadas? ¿Qué terminó la semana
  pasada?
- Carga de trabajo, capacidad, sobrecarga, asignación semanal, workload.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Backlog** assigns stories and sets their estimated hours.
- **Sprints** places stories in sprints and moves them to Finalizado; its planner can assign them by
  person.
- **Gantt** shows the same bars for one whole project.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.choose-person]
  - path: frontend/routes/projects/reports/by-person/+page.svelte
    role: page
    hash: sha256:78c08b0a40f303523a2d8fcb397b4fdc22eadd6a13802229d2a672ee953f957e
    supports: [page-purpose, capability.choose-person, capability.list]
  - path: frontend/routes/projects/reports/by-person/PersonGantt.svelte
    role: user-interface
    hash: sha256:4081b626f57715e55653bf1a7382d3b2cdf840c69cee44d5f4ccbc398e36915e
    supports: [capability.person-gantt]
  - path: frontend/routes/projects/reports/by-person/PersonWeeks.svelte
    role: user-interface
    hash: sha256:4fff8cc1d86cadf1fb02693917f6dc5221080983ee475c2a9a2bb001076d4757
    supports: [capability.weekly-load]
  - path: frontend/routes/projects/reports/by-person/by-person.ts
    role: business-logic
    hash: sha256:705c1d8f4f5152d680ecaec52c40ec49936038e5fd934c419aac03b8d585f5c7
    supports: [concepts, capability.list, capability.person-gantt, capability.weekly-load]
  - path: frontend/routes/projects/reports/by-person/by-person.svelte.ts
    role: frontend-service
    hash: sha256:fe63799d70eb6472a3c76c4fbb0a69bfc90200eb1dbe1c3ae98566ea99bc916e
    supports: [concepts, capability.list]
  - path: frontend/routes/projects/reports/timeline.ts
    role: business-logic
    hash: sha256:4fa5a775f464fe01a41207a1fee5a0a91b61786f0b06b2ab0c06d21b93e8be00
    supports: [capability.person-gantt]
  - path: frontend/routes/projects/project.ts
    role: shared-domain
    hash: sha256:2a960298b661517ac43a5a1fc6706c0596d993288edc033c4e7ff25cea11a202
    supports: [capability.choose-person]
  - path: frontend/routes/projects/project.svelte.ts
    role: frontend-service
    hash: sha256:dd3dd8889051439eb8661fa7f5b2fac97c964863c9ef1e01b2fd3f3c30a992df
    supports: [capability.choose-person]
  - path: backend/projects/stories.go
    role: backend-handler
    hash: sha256:9b6f2cf5a86a59be861fd8768a2ac699ba783504a2d08f9664e0322d2177b78f
    supports: [concepts, capability.list]
  - path: backend/projects/story_rules.go
    role: business-logic
    hash: sha256:d6397534ef09ee0bfc9afcff78bb6064eb71ddf4d8405be51ded966beaf3c5e9
    supports: [concepts]
  - path: backend/projects/types/stories.go
    role: data-model
    hash: sha256:33d0fca372da7917d1abc4a46ba6410033bc4e4ca1f79d6cc487179617afbe8a
    supports: [concepts, capability.weekly-load]
```
