---
schema: 1
page_id: crm.tasks
route: /crm/tasks
title: Tasks (Tareas)
status: implemented
visibility: tenant
description_en: >-
  CRM task inbox: the activities and emails the pipeline stages propose for each opportunity, and the
  stage-move approvals waiting for the user, grouped by due day with overdue days first. Complete,
  dismiss, send the email, or approve and reject from a side panel. A team leader reads the open tasks
  of their team members without acting on them, and each user sees their finished tasks.
description_es: >-
  Bandeja de tareas del CRM: las actividades y correos que proponen las etapas del pipeline para cada
  oportunidad y las aprobaciones de movimientos pendientes, agrupadas por día de vencimiento con los días
  vencidos primero. Completar, descartar, enviar el correo, o aprobar y rechazar desde un panel lateral.
  El líder de un equipo ve las tareas abiertas de sus integrantes sin poder actuar.
---

# Tasks (Tareas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Tasks (`Tareas`, page title **CRM Tasks (Tareas CRM)**) is the seller's CRM inbox (`bandeja de
tareas`, `pendientes`). It lists what the user has to do for their opportunities: the tasks a
pipeline stage proposes when an opportunity enters it (log a call, a visit, send an email) and the
approvals of stage moves that wait for the user's decision (`aprobaciones`). Each task is closed
from its side panel.

Tasks are only proposed, never executed on their own. The page does not create tasks by hand: they
come from the stage task templates set in **Pipelines (Flujos)** and from moves into stages with an
approver on the **Opportunities (Oportunidades)** board.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- An **activity task** asks to log an activity of a given type (for example a call) on the
  opportunity.
- A **send email task** asks to send an email to the client, starting from a given template.
- An **approval** (`aprobación`) is a request to move an opportunity forward into a stage that has an
  approver. The opportunity stays where it was until the approver decides.
- A task is **pending** (`pendiente`) until it ends as **Done (Hecha)**, **Dismissed (Descartada)**,
  **Approved (Aprobada)**, **Rejected (Rechazada)** or **Cancelled (Cancelada)**.
- A task's due day (`vencimiento`) is the day the opportunity entered the stage plus the template's
  due days. It is assigned to the opportunity's owner, or to the fixed user the template names.

<!-- DOC-ID: capability.inbox -->
## See my pending tasks (Ver mis tareas pendientes)

### User intention (Intención del usuario)

Know what to do today, what is overdue and which approvals are waiting.

### Where to find it (Dónde encontrarlo)

**CRM → Customers (Clientes) → Tasks (Tareas)** at `/crm/tasks`, tab **My tasks (Mis tareas)**.

### Business rules and rationale (Reglas y razón de negocio)

- **Approvals waiting (Aprobaciones pendientes)** come first, on their own.
- The other tasks are grouped under one divider per due day, oldest first, so overdue days come before
  today. Overdue days are shown in red; near days read **Yesterday (Ayer)**, **Today (Hoy)** and
  **Tomorrow (Mañana)**.
- Each card shows the stage badge, the task, the client (**No client (Sin cliente)** when missing),
  the opportunity and its amount in whole units. Clicking a card opens its side panel.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué tareas tengo hoy? ¿Tengo tareas vencidas? ¿Dónde veo lo que tengo que aprobar?
- Pendientes, agenda del vendedor, recordatorios, seguimientos.

<!-- DOC-ID: capability.complete-activity-task -->
## Complete or dismiss an activity task (Completar o descartar una tarea de actividad)

### User intention (Intención del usuario)

Record that the call, visit or follow-up the stage asked for was done, or drop a task that no longer
makes sense.

### Where to find it (Dónde encontrarlo)

Click the task → side panel with **Due (Vence)**, **Client (Cliente)** and **Opportunity
(Oportunidad)**. Choose the activity type (preselected from the stage's template), write what was
done in **Activity (Actividad)** and press **Complete (Completar)**, or press **Dismiss (Descartar)**.

### Business rules and rationale (Reglas y razón de negocio)

- **Complete (Completar)** needs the activity text (**Write what was done. (Escribe lo que se
  hizo.)**).
- Only the assigned user can complete or dismiss the task.
- To complete it, the user must be a member of the opportunity's pipeline, because the activity is
  written on the opportunity.

### Result and side effects (Resultado y efectos)

Complete writes the activity on the opportunity's **Activity (Actividad)** timeline and closes the
task as **Done (Hecha)** in the same step. Dismiss closes it as **Dismissed (Descartada)** without
writing anything on the opportunity.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo marco una tarea como hecha? ¿Puedo descartar una tarea que ya no aplica?

<!-- DOC-ID: capability.send-email-task -->
## Send the email a task asks for (Enviar el correo de una tarea)

### User intention (Intención del usuario)

Send the client the email the stage planned, starting from the stage's template.

### Where to find it (Dónde encontrarlo)

Click the task → **Write email (Redactar correo)** opens **Send email (Enviar correo)** with the
template already rendered for that opportunity: **To (Para)**, **CC**, **Subject (Asunto)** and the
body, all editable before sending. **Dismiss (Descartar)** drops the task instead.

### Business rules and rationale (Reglas y razón de negocio)

- Only the assigned user completes it, and only by sending the email; a hand-written activity does
  not close a send email task.
- At least one recipient and a subject; at most 20 recipients counting To and CC.
- The user must be a member of the opportunity's pipeline.

### Result and side effects (Resultado y efectos)

The email is sent under the company's name with replies to the sender, an **Email** activity is
added to the opportunity, and the task closes as **Done (Hecha)**.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo envío el correo que me pide la tarea? ¿Puedo cambiar el texto de la plantilla antes de enviar?

<!-- DOC-ID: capability.decide-approval -->
## Approve or reject a move (Aprobar o rechazar un movimiento)

### User intention (Intención del usuario)

As the stage's approver, authorize or refuse that an opportunity advances into the stage.

### Where to find it (Dónde encontrarlo)

**My tasks (Mis tareas)** → **Approvals waiting (Aprobaciones pendientes)** → click the request. The
panel shows the target stage, the client, the opportunity and **Requested by (Solicitada por)**.
Write an optional **Note (Nota)** and press **Approve (Aprobar)** or **Reject (Rechazar)**.

### Business rules and rationale (Reglas y razón de negocio)

- Only the approver decides, and only while the opportunity still waits on that request.
- The approver does not need to be a member of the pipeline; approving does not give access to its
  board.
- Approving re-checks that the opportunity is open and the target stage still exists and has its
  required data; otherwise the approval is refused.
- The note can have up to 500 characters.

### Result and side effects (Resultado y efectos)

- Approve: the opportunity moves into the stage, the move is recorded with the approver as its author,
  and the new stage proposes its tasks. With a note, the timeline gets **Aprobó el paso a la etapa
  «…»** and the note.
- Reject: the opportunity stays where it is and is released; its timeline always gets **Rechazó el
  paso a la etapa «…»**, plus the note when written.

### Limitations (Limitaciones)

An approval cannot be dismissed, only approved or rejected. The requester cancels their own request
from the opportunity's panel on the board, not here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde apruebo el paso de una oportunidad? ¿Cómo rechazo una solicitud y explico por qué?

<!-- DOC-ID: capability.team-view -->
## See my team's tasks (Ver las tareas de mi equipo)

### User intention (Intención del usuario)

As a team leader, check what each seller of the team has pending.

### Where to find it (Dónde encontrarlo)

Tab **My team (Mi equipo)**, then choose a member or **All (Todos)** in the selector. With **All
(Todos)**, each card names its seller.

### Business rules and rationale (Reglas y razón de negocio)

- The tab only appears for a user who leads at least one team; it lists the members of every team
  they lead.
- It is read-only: the side panel shows the task but no action. Only the assignee acts on a task.

### Limitations (Limitaciones)

It shows the members' open tasks only, not their finished ones.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo veo los pendientes de mis vendedores? ¿Por qué no veo la pestaña Mi equipo? → you do not lead
  a team in Teams (Equipos).

<!-- DOC-ID: capability.done-tasks -->
## See finished tasks (Ver tareas terminadas)

### User intention (Intención del usuario)

Review what the user already closed and how each task ended.

### Where to find it (Dónde encontrarlo)

Tab **Done (Hechas)**. Tasks are grouped by the day they were closed, latest first; each card shows
its result badge and the time. The side panel adds the date, who closed it and the note.

### Business rules and rationale (Reglas y razón de negocio)

The results are **Done (Hecha)** and **Approved (Aprobada)** in green, **Rejected (Rechazada)** in
red, **Dismissed (Descartada)** and **Cancelled (Cancelada)** in grey. Genix cancels a stage task
when its opportunity moves back before the task's stage, moves to another pipeline or is closed as
lost; an approval is cancelled when the requester withdraws it.

### Limitations (Limitaciones)

Finished tasks are read-only and cannot be reopened. Only the user's own finished tasks are listed.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **A task disappeared without being done:** its opportunity moved back before the task's stage,
  moved to another pipeline or was lost, and Genix cancelled it. It appears in **Done (Hechas)** as **Cancelled (Cancelada)**.
- **"No tiene acceso a este pipeline." when completing or sending:** the assignee is not a member of
  the opportunity's pipeline; ask for at least View level in Pipelines: Access (Flujos: Accesos).
- **"Esta tarea se completa enviando el correo.":** a send email task closes only with **Write email
  (Redactar correo)**.
- **"Solo el aprobador puede decidir esta solicitud." / "La solicitud de aprobación ya no está
  vigente.":** someone else must decide, or the request was already decided or cancelled.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Opportunities (Oportunidades)** is where the opportunities move, which proposes these tasks and
  sends approval requests; the requester cancels a request there.
- **Pipelines (Flujos)** defines, per stage, the task templates (tab **Tasks (Tareas)**) and the
  approver (tab **Approvals (Aprobaciones)**).
- **Teams (Equipos)** defines the leaders and members used by **My team (Mi equipo)**.
- **Dictionaries (Diccionarios)** holds the company's activity types and the email templates.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.inbox]
  - path: frontend/routes/crm/tasks/+page.svelte
    role: page
    hash: sha256:e20ff5834052e096a107d746cca05273865ef0bcc31ede730c34b309a1c696dc
    supports: [page-purpose, capability.inbox, capability.send-email-task, capability.team-view, capability.done-tasks]
  - path: frontend/routes/crm/tasks/tasks.svelte.ts
    role: frontend-service
    hash: sha256:3f2c83767152e221aa9a9f5d6f5ac018c06b78a618c67f1be60f4c574e44599e
    supports: [capability.inbox, capability.complete-activity-task, capability.team-view, capability.done-tasks]
  - path: frontend/routes/crm/tasks/tasks.ts
    role: business-logic
    hash: sha256:e9ed543dc615d9df0287ea57c1c6beae98b5763ae34fc313e68e29e2014e9fe9
    supports: [concepts, capability.inbox, capability.team-view, capability.done-tasks]
  - path: frontend/routes/crm/tasks/TaskCard.svelte
    role: user-interface
    hash: sha256:c9370eb099780d6ff752c408834306d502bd01134dbdd4e4f5c884bc86a67d58
    supports: [capability.inbox, capability.team-view, capability.done-tasks]
  - path: frontend/routes/crm/tasks/TaskLayer.svelte
    role: user-interface
    hash: sha256:e797230909951fef029753ca4fa20b3ca402d90c90a682dd17864af066f84e00
    supports: [capability.complete-activity-task, capability.send-email-task, capability.decide-approval, capability.team-view, capability.done-tasks]
  - path: frontend/domain-components/crm/OpportunityEmailModal.svelte
    role: user-interface
    hash: sha256:cd0fe4e35269b8b68c11e60a8d182abab96d96bf35d2eeaaea7db61397fa66ca
    supports: [capability.send-email-task]
  - path: frontend/services/crm/crm-task.svelte.ts
    role: frontend-service
    hash: sha256:76970e6b14d3b4ce75f5e4be41ca70f1f69891b31df65feff04df1c7a3652419
    supports: [capability.complete-activity-task, capability.decide-approval]
  - path: backend/crm/task.go
    role: backend-handler
    hash: sha256:c0f7ebf87aa25f2054dbdf2cf494d8a1ecee09eac11a3072cd62e813ec80f167
    supports: [capability.inbox, capability.complete-activity-task, capability.decide-approval, capability.team-view, capability.done-tasks, troubleshooting]
  - path: backend/crm/activity.go
    role: backend-handler
    hash: sha256:3767786ed83f5814e18724009ff1583c7172a7b7d8d33d61b258440c757e769a
    supports: [capability.complete-activity-task, troubleshooting]
  - path: backend/crm/email.go
    role: backend-handler
    hash: sha256:b055785ce762effc7d5221f78752b230fd49d425cdf22a3fdee99cd7b861a75e
    supports: [capability.send-email-task, troubleshooting]
  - path: backend/crm/types/task.go
    role: data-model
    hash: sha256:2763b5ad8481e5e46256f4b4e6fcc26da8cc9c4e02ec09a28b77f5c5e46c268b
    supports: [concepts, capability.done-tasks]
  - path: backend/crm/types/task_rules.go
    role: business-logic
    hash: sha256:92c31194e686a3f86e513a30ca18e8aaac5c5b6a9e9fce927ac614d8dc3ba93e
    supports: [concepts, capability.complete-activity-task, capability.send-email-task, capability.decide-approval, capability.team-view, capability.done-tasks, troubleshooting]
  - path: backend/crm/types/email_template_rules.go
    role: business-logic
    hash: sha256:a026474155c5983cc5b4b1ba5fca2791823ae94c4636dab6b32597f488189b2d
    supports: [capability.send-email-task]
  - path: backend/crm/types/pipeline.go
    role: data-model
    hash: sha256:272e4f0c215cf274446e0a9e22b87467413c1f54796777ff9762a4dbd4f2d639
    supports: [concepts]
  - path: backend/crm/opportunity.go
    role: backend-handler
    hash: sha256:a9b0fdd849e8f963985386bb71c6b3bcda166f3fd853bb1d7739205dbdd951cd
    supports: [capability.done-tasks]
```
