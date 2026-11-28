---
schema: 1
page_id: crm.pipeline
route: /crm/pipeline
title: Opportunities (Oportunidades)
status: implemented
visibility: tenant
description_en: >-
  Kanban board of the open sales opportunities of one pipeline, one column per stage. Create an
  opportunity for a client or a new prospect, edit its amount, probability, expected close and custom
  fields, move it between stages or pipelines (with stage approvals and required data), close it as won
  or lost, log activities with files and send emails from templates.
description_es: >-
  Tablero kanban de las oportunidades de venta abiertas de un pipeline, una columna por etapa. Crear
  una oportunidad para un cliente o un nuevo prospecto, editar monto, probabilidad, cierre estimado y
  campos personalizados, moverla entre etapas o pipelines (con aprobaciones y datos obligatorios),
  cerrarla como ganada o perdida, registrar actividades con archivos y enviar correos con plantillas.
---

# Opportunities (Oportunidades)

<!-- DOC-ID: page-purpose -->
## Page purpose

Opportunities (`Oportunidades`) is the CRM sales board (`tablero kanban`, `embudo de ventas`). It
shows the open sales opportunities (`oportunidades de venta`, also called `negocios`, `tratos` or
`leads`) of one pipeline (`pipeline` or `flujo de venta`), one column per stage (`etapa`). Here the
seller creates an opportunity, moves it from stage to stage, records what was done with the client
(`actividades`: calls, visits, notes, files), sends emails from templates, and finally closes it as
won (`ganada`) or lost (`perdida`).

The page does not define pipelines or stages: that is **Pipelines (Flujos)**. It does not decide who
sees each pipeline: that is **Pipelines: Access (Flujos: Accesos)**. Winning an opportunity does not
create a sales order, quote or invoice; it only closes the opportunity and turns the prospect into a
customer. The tasks the stages propose and the approvals are worked from **Tasks (Tareas)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- An **opportunity** (`oportunidad`) is a possible sale to one client (`cliente`) with a title, an
  amount (`monto`), a probability % (`probabilidad`), an expected close date (`cierre estimado`), an
  owner (`responsable`, the seller in charge) and the pipeline's custom fields. It sits in exactly one
  pipeline and one stage.
- A **prospect** (`prospecto`) is a lead that has not bought yet. It is a customer record marked as a
  prospect; winning one of its opportunities removes that mark.
- **Won** and **Lost** are outcomes (`resultados`), not stages. A closed opportunity leaves the board.
- **Probability** starts at the stage's default probability and the user may change it. Every move to
  another stage sets it again to the new stage's default.
- A **stale** card (`Estancada`) has stayed in its stage longer than the stage's stale days. Stages
  with 0 stale days never mark cards.
- **Member levels** of a pipeline: **View (Ver)** sees the board and can write activities and send
  emails; **Edit (Editar)** can also create, move and close opportunities, and edit the ones they
  created or own; **Full (Completo)** can also edit any opportunity and change its owner.

<!-- DOC-ID: capability.browse-board -->
## Browse the board (Ver el tablero)

### User intention (Intención del usuario)

See where each open deal stands, how much money each stage holds and which cards are stuck.

### Where to find it (Dónde encontrarlo)

**CRM → Customers (Clientes) → Opportunities (Oportunidades)** at `/crm/pipeline`. Choose the
pipeline in the **Pipeline** selector; the board reopens the last pipeline chosen on that browser.
**Filter by owner (Filtrar por responsable)** shows one avatar per seller holding cards; picking
avatars shows only their opportunities, and picking none shows everyone's.

### Business rules and rationale (Reglas y razón de negocio)

- The selector lists only the pipelines where the user is a member, at any level.
- Each card shows the title, the client, the amount, the probability %, the owner's avatar, and
  **From (Desde)** with the pipeline and stage it came from after a move. It shows **Pending approval
  (Por aprobar)** while a move waits for approval, otherwise **Stale (Estancada)** when it is stale.
- Each column footer shows the total amount of its cards and, after `≈`, the weighted amount: each
  amount multiplied by its probability %, the column's forecast.
- Only open opportunities are on the board; won and lost ones are not listed here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde veo mis oportunidades? ¿Cómo veo solo las oportunidades de un vendedor?
- ¿Qué significa "Estancada"? ¿Qué es el monto con ≈ al pie de la columna?
- Embudo de ventas, funnel, pipeline de ventas, tablero de negocios, kanban.

<!-- DOC-ID: capability.create-opportunity -->
## Create an opportunity (Crear una oportunidad)

### User intention (Intención del usuario)

Register a new deal for an existing customer, or for a new lead that is not a customer yet.

### Where to find it (Dónde encontrarlo)

Button **New (Nueva)** on the board toolbar; the side panel **New opportunity (Nueva oportunidad)**
opens. Save with the panel's save button. The opportunity is created in the first stage of the
selected pipeline.

### Required information and prerequisites (Requisitos previos)

- Edit or Full level in the pipeline; the **New (Nueva)** button is hidden otherwise.
- **Title (Título)**: required.
- **Client (Cliente)**: an existing customer, or a name typed in **…or new prospect (…o nuevo
  prospecto)**. With a prospect name and no client chosen, Genix first creates a customer record with
  that name marked as a prospect, then the opportunity.
- **Amount (Monto)**, **Probability % (Probabilidad %)**, **Expected close (Cierre estimado)**:
  optional unless the first stage requires the amount or the expected close.
- **Owner (Responsable)**: the user creating it by default; only a Full member can choose another.
- The pipeline's custom fields; those required from the first stage are marked required.

### Business rules and rationale (Reglas y razón de negocio)

- The amount cannot be negative and the probability must be between 0 and 100.
- The client must be an active customer.
- Every custom field and standard field the first stage requires must be filled.

### Result and side effects (Resultado y efectos)

The card appears in the first column. Its history (**Activity (Actividad)** tab) starts with
**Created in (Creada en)** the stage, and the tasks configured for the first stage are proposed to
the owner or to the user each task names, in **Tasks (Tareas)**.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro un nuevo negocio o lead? ¿Puedo crear una oportunidad para alguien que todavía no es
  cliente? → write the name in "…o nuevo prospecto".
- ¿Por qué no veo el botón Nueva? → your level in that pipeline is View.

<!-- DOC-ID: capability.edit-opportunity -->
## Edit an opportunity (Editar una oportunidad)

### User intention (Intención del usuario)

Update the amount, probability, expected close, client, owner or custom fields of a deal.

### Where to find it (Dónde encontrarlo)

Click a card; the side panel opens on the **Information (Información)** tab with **Title (Título)**,
**Client (Cliente)**, **Amount (Monto)**, **Probability % (Probabilidad %)**, **Expected close
(Cierre estimado)**, **Owner (Responsable)** and the pipeline's custom fields. The panel header shows
the current stage.

### Business rules and rationale (Reglas y razón de negocio)

- A Full member edits any opportunity. An Edit member edits only the ones they created or own; the
  others open read-only. A View member cannot edit.
- Only a Full member can change the **Owner (Responsable)**.
- An edit cannot leave empty a field the current stage (or an earlier stage) requires.
- A closed opportunity cannot be edited.

### Result and side effects (Resultado y efectos)

Changes of amount, owner or expected close are written to the opportunity's history as **Changed …
(Cambió monto, responsable, fecha de cierre)**, and the opportunity's open tasks show the new title,
client and amount.

### Limitations (Limitaciones)

The stage is never changed by editing; use the move actions. There is no option to delete an
opportunity on this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cambio el monto o la fecha de cierre? ¿Cómo reasigno una oportunidad a otro vendedor?
- ¿Por qué no puedo editar la oportunidad de mi compañero?

<!-- DOC-ID: capability.move-opportunity -->
## Move an opportunity to another stage or pipeline (Mover una oportunidad)

### User intention (Intención del usuario)

Advance a deal to the next step of the sale, send it back, or pass it to another pipeline.

### Where to find it (Dónde encontrarlo)

Drag the card to another column, or open the card and use **Move to... (Mover a...)**. The dialog
**Move opportunity (Mover oportunidad)** asks **Pipeline** and **Stage (Etapa)**; stages with an
approver show **(needs approval) (requiere aprobación)**.

### Required information and prerequisites (Requisitos previos)

- Edit or Full level in the pipeline; dragging is disabled for View members.
- To another pipeline: Edit or Full level in both the source and the target pipeline. The dialog
  lists only pipelines where the user has Edit or Full.
- The data the target stage requires. When a dragged card lacks it, the move dialog opens with that
  stage chosen and asks under **This stage requires: (Esta etapa exige:)** for the missing amount,
  expected close or custom fields. Genix saves them on the opportunity, then moves it.

### Business rules and rationale (Reglas y razón de negocio)

- A stage requires its own required fields and those of every earlier stage, so skipping a stage
  does not skip its rule.
- Only a Full member or the owner/creator with Edit can fill the missing data; otherwise the move is
  refused with **La etapa de destino exige datos que solo el responsable o un usuario Completo puede
  llenar**.
- A closed opportunity, or one waiting for an approval, cannot be moved.
- Moving forward into a stage that has an approver does not move the card: it is sent for approval
  (see the approval section).

### Result and side effects (Resultado y efectos)

- The card changes column, its probability takes the new stage's default, and its stale counter
  restarts. The history records **Moved from X to Y (Pasó de X a Y)**.
- Pending stage tasks of stages after the new one, or of another pipeline, are cancelled; the new
  stage proposes its own tasks, without repeating one still pending.
- Moved to another pipeline, the card leaves this board, keeps its custom field values, and in the
  new pipeline shows **From (Desde)** with where it came from.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo paso una oportunidad a la siguiente etapa? ¿Puedo arrastrar la tarjeta?
- ¿Por qué me pide el monto al mover la tarjeta? → the stage requires it.
- ¿Cómo paso una oportunidad a otro pipeline?

<!-- DOC-ID: capability.stage-approval -->
## Moves that need approval (Movimientos con aprobación)

### User intention (Intención del usuario)

Advance a deal into a stage that a supervisor must authorize (for example, a discount or a final
offer), and follow or withdraw that request.

### Where to find it (Dónde encontrarlo)

Any move into a stage with an approver. The card then shows **Pending approval (Por aprobar)**; the
panel shows the request with **Waiting for (Esperando a)** and the approver's name, and the person
who asked sees **Cancel request (Cancelar solicitud)**.

### Business rules and rationale (Reglas y razón de negocio)

- Only forward moves need approval; moving back to an earlier stage never does. Entering a stage with
  an approver from another pipeline also needs it.
- A forward move cannot jump over a stage that has an approver: **Debe pasar primero por la etapa …,
  que requiere aprobación**.
- While the request waits, the card cannot be moved, closed as won or lost; the move and close buttons
  give way to the request.
- Only the user who asked can cancel the request.
- The approver must be an active user.

### Result and side effects (Resultado y efectos)

The message **Sent for approval (Enviada a aprobación)** appears and the approver receives an
approval task in **Tasks (Tareas)**. If approved, the card moves; if rejected or cancelled, it stays
in its stage and is released.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué mi tarjeta no se movió y dice "Por aprobar"? ¿Quién tiene que aprobar?
- ¿Cómo retiro una solicitud de aprobación?

<!-- DOC-ID: capability.close-opportunity -->
## Close an opportunity as won or lost (Cerrar como ganada o perdida)

### User intention (Intención del usuario)

Record that the client bought (`venta cerrada`, `ganada`) or that the deal was lost, and why.

### Where to find it (Dónde encontrarlo)

Open the card → **Won (Ganada)** opens **Close as won (Cerrar como ganada)**; **Lost (Perdida)**
opens **Close as lost (Cerrar como perdida)** with **Reason (Motivo)** and **Note (Nota)**.

### Required information and prerequisites (Requisitos previos)

- Edit or Full level in the pipeline.
- To close as lost, a **Reason (Motivo)** from the lost reasons list; the note is optional.

### Business rules and rationale (Reglas y razón de negocio)

The lost reason must be active. An opportunity waiting for an approval cannot be closed.

### Result and side effects (Resultado y efectos)

- The card leaves the board and its history records **Closed as won (Cerrada como ganada)** or
  **Closed as lost (Cerrada como perdida)**.
- Won: the client stops being a prospect (`El cliente deja de ser prospecto`). Its pending tasks are
  left as they are.
- Lost: its pending stage tasks are cancelled.

### Limitations (Limitaciones)

Closing does not create a sale, quote or invoice. Won and lost opportunities are not listed on this
board.

<!-- DOCUMENTATION_GAP: The panel has a Reopen (Reabrir) button for closed opportunities and the
backend supports reopening, but the board only loads open opportunities, so no visible path opens a
closed opportunity in the panel. Confirm where users reopen an opportunity. Excluded from RAG. -->

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo marco una oportunidad como ganada? ¿Cómo registro que perdí la venta?
- ¿Dónde agrego más motivos de pérdida? → Dictionaries (Diccionarios).

<!-- DOC-ID: capability.log-activity -->
## Log activities and see the history (Registrar actividades e historial)

### User intention (Intención del usuario)

Leave a record of each call, meeting, visit, WhatsApp or note with the client, attach files, and see
everything that happened to the opportunity.

### Where to find it (Dónde encontrarlo)

Open a saved card → **Activity (Actividad)** tab. Choose the type (**Note (Nota)** is preselected),
write in **Write the activity or drop a file (Escribe la actividad o arrastra un archivo)**, attach
files with the clip button or by dropping them on the text box, and save with the send button.

### Business rules and rationale (Reglas y razón de negocio)

- Any member of the pipeline can write activities, View level included.
- An activity needs text or at least one file. Text up to 4000 characters; up to 30 files, each up
  to 4 MB.
- Types: the fixed ones (Note, Call, Meeting, Email, WhatsApp, Visit, Proposal sent, Demo, Follow-up)
  plus the company's own types from Dictionaries.

### Result and side effects (Resultado y efectos)

The list shows, newest first, the activities with their type, author and date, mixed with the stage
history: created, moved, changed, closed. The CRM reports also read these activities, by pipeline.

### Limitations (Limitaciones)

Activities cannot be edited or deleted from the panel. An activity whose company type was later
deleted keeps showing under the generic label **Activity (Actividad)**.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro una llamada o una visita al cliente? ¿Puedo adjuntar una cotización en PDF?
- ¿Dónde veo el historial de la oportunidad? Bitácora, seguimiento, notas.

<!-- DOC-ID: capability.send-email -->
## Send an email to the client (Enviar correo al cliente)

### User intention (Intención del usuario)

Email the client from the opportunity using a company template, and keep the email in its history.

### Where to find it (Dónde encontrarlo)

Card panel → **Activity (Actividad)** tab → **Send email (Enviar correo)**. The **Send email (Enviar
correo)** dialog has **Template (Plantilla)**, **To (Para)**, **CC**, **Subject (Asunto)** and the
body editor.

### Required information and prerequisites (Requisitos previos)

- Choosing a template fills the subject and body with the client's, opportunity's, owner's and
  company's data, and **To (Para)** with the client's email when it has one and nothing was typed.
- At least one recipient and a subject. Several addresses are separated by commas, semicolons or
  spaces.

### Business rules and rationale (Reglas y razón de negocio)

- Any member of the pipeline can send, View level included.
- At most 20 recipients counting To and CC; every address must be valid.
- A template that uses quote variables cannot be used from here: they only work when sending a quote.
- The email is refused if it still contains a `{{variable}}` typed by hand.

### Result and side effects (Resultado y efectos)

The email goes out under the company's name, and replies go to the sender's own email. An **Email**
activity with the subject and recipients is added to the opportunity's history.

### Limitations (Limitaciones)

If the email left but could not be recorded, Genix says so (**El correo se envió, pero no se pudo
registrar en la oportunidad**); do not resend it.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Puedo mandar un correo al cliente desde la oportunidad? ¿Dónde creo las plantillas de correo? →
  Dictionaries (Diccionarios).

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Aún no tiene pipelines asignados. Pida acceso a un administrador."**: the user is not a member
  of any pipeline; an administrator assigns it in Pipelines: Access (Flujos: Accesos).
- **No New button, cards cannot be dragged:** the user's level in that pipeline is View.
- **"Faltan datos obligatorios para la etapa …":** the stage requires the amount, the expected close
  or custom fields; fill them in the panel or in the move dialog.
- **"La oportunidad espera una aprobación; cancele la solicitud o espere la decisión.":** wait for the
  approver or cancel the request from the card panel.
- **"El aprobador de la etapa está inactivo…":** ask an administrator to change the stage's approver
  in Pipelines (Flujos).
- **"Necesita nivel Editar en el pipeline de origen y en el de destino.":** ask for Edit level in the
  target pipeline.
- **"Solo el nivel Completo puede cambiar el responsable.":** only a Full member reassigns owners.
- **"La oportunidad está cerrada; reábrala para editarla.":** closed opportunities cannot be edited.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Pipelines (Flujos)** defines the pipelines, stages, probabilities, stale days, required fields,
  stage tasks and approvers this board uses, plus the custom fields.
- **Pipelines: Access (Flujos: Accesos)** decides who sees each pipeline and at what level.
- **Dictionaries (Diccionarios)** holds the lost reasons, the company's activity types and the email
  templates.
- **Tasks (Tareas)** is where the owner works the tasks the stages propose and the approver decides
  approvals.
- **Customers (Clientes)** lists the customers, including prospects created from here.
- **Timeline (Línea de tiempo)**, **Flow (Flujo)**, **Conversion (Conversión)**, **Performance
  (Desempeño)** and **Forecast** report on these opportunities.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.browse-board]
  - path: frontend/routes/crm/pipeline/+page.svelte
    role: page
    hash: sha256:c4735df7a4d87d722145f6c05e8a6097f4f9b40fddd4f0d6006ef8f396047f3e
    supports: [page-purpose, capability.browse-board, capability.create-opportunity, capability.move-opportunity, troubleshooting]
  - path: frontend/routes/crm/pipeline/pipeline.svelte.ts
    role: frontend-service
    hash: sha256:5f3ff0b6918742544726feb2068e5ffd4e37db057be3eaedfa2f6a7f8b6d16df
    supports: [capability.browse-board, capability.move-opportunity, capability.stage-approval]
  - path: frontend/routes/crm/pipeline/pipeline.ts
    role: business-logic
    hash: sha256:fbfccc898f175d1520d9d2d9b074e98b01092e364eb41dddcb1c8a17443340ad
    supports: [concepts, capability.browse-board, capability.create-opportunity, capability.edit-opportunity, capability.move-opportunity, capability.log-activity]
  - path: frontend/routes/crm/pipeline/OpportunityCard.svelte
    role: user-interface
    hash: sha256:1e7a19106fc0703b3e39eaf2946c1bbb7201514bd6f1c840ba72a251a88655bc
    supports: [capability.browse-board, capability.stage-approval]
  - path: frontend/routes/crm/pipeline/OpportunityLayer.svelte
    role: user-interface
    hash: sha256:1bf5b6bddb85bea09ad026b7a71b19cb2926bb184230d200c0a4589f2c7ba982
    supports: [capability.create-opportunity, capability.edit-opportunity, capability.stage-approval, capability.close-opportunity, capability.send-email]
  - path: frontend/routes/crm/pipeline/OpportunityMoveModal.svelte
    role: user-interface
    hash: sha256:2e684ae85eec8041f3b4db2e2b336985c5135077c8b124f052bf18b357c94fb7
    supports: [capability.move-opportunity, capability.stage-approval]
  - path: frontend/routes/crm/pipeline/OpportunityCloseModal.svelte
    role: user-interface
    hash: sha256:4b6ac6cad2c5db8701a3c15a31a52a128a6579582608d6dedfcaab87bf8c3091
    supports: [capability.close-opportunity]
  - path: frontend/routes/crm/pipeline/OpportunityActivity.svelte
    role: user-interface
    hash: sha256:df71865b3fec64faae6c057febcfcf4980aad4b79978317f93d0c2cfe3cf2a8f
    supports: [capability.log-activity]
  - path: frontend/domain-components/crm/OpportunityEmailModal.svelte
    role: user-interface
    hash: sha256:cd0fe4e35269b8b68c11e60a8d182abab96d96bf35d2eeaaea7db61397fa66ca
    supports: [capability.send-email]
  - path: frontend/services/crm/crm-opportunity.svelte.ts
    role: frontend-service
    hash: sha256:b221d63fa939990d78cff7f9ecdeccd87ebe927d6afe5c5143907bc3712485ba
    supports: [capability.move-opportunity, capability.close-opportunity]
  - path: frontend/services/crm/crm-task.svelte.ts
    role: frontend-service
    hash: sha256:76970e6b14d3b4ce75f5e4be41ca70f1f69891b31df65feff04df1c7a3652419
    supports: [capability.stage-approval]
  - path: frontend/services/crm/crm-activity-types.ts
    role: shared-domain
    hash: sha256:2a09bf27436973b1edb986bb368bf439fa539008249d6e4dc0587b21b01d34be
    supports: [capability.log-activity]
  - path: frontend/services/crm/crm-email-template.ts
    role: frontend-service
    hash: sha256:da6e8a2f1605b6de2dbbc4eae22c55fdef976540a548de87f8cb9b291864a94d
    supports: [capability.send-email]
  - path: backend/crm/opportunity.go
    role: backend-handler
    hash: sha256:a9b0fdd849e8f963985386bb71c6b3bcda166f3fd853bb1d7739205dbdd951cd
    supports: [capability.browse-board, capability.create-opportunity, capability.edit-opportunity, capability.move-opportunity, capability.close-opportunity]
  - path: backend/crm/task.go
    role: backend-handler
    hash: sha256:c0f7ebf87aa25f2054dbdf2cf494d8a1ecee09eac11a3072cd62e813ec80f167
    supports: [capability.move-opportunity, capability.stage-approval, troubleshooting]
  - path: backend/crm/activity.go
    role: backend-handler
    hash: sha256:3767786ed83f5814e18724009ff1583c7172a7b7d8d33d61b258440c757e769a
    supports: [capability.log-activity]
  - path: backend/crm/email.go
    role: backend-handler
    hash: sha256:b055785ce762effc7d5221f78752b230fd49d425cdf22a3fdee99cd7b861a75e
    supports: [capability.send-email]
  - path: backend/crm/report.go
    role: backend-handler
    hash: sha256:49358ff5d81bf3b29aea73be2dc8c9a6264b17bd79bbb2f13eb3c0921d6035e0
    supports: [capability.log-activity]
  - path: backend/crm/types/opportunity_rules.go
    role: business-logic
    hash: sha256:1b7c0bc3cf75de02ebdd3e2c9991827dd58c369cf74525baa301bd991c46a48a
    supports: [concepts, capability.create-opportunity, capability.edit-opportunity, capability.move-opportunity, capability.close-opportunity, troubleshooting]
  - path: backend/crm/types/pipeline_rules.go
    role: business-logic
    hash: sha256:f574d970c74d1cf8966d1bcde3c8b9640af596d51830624e06a605650c757c48
    supports: [concepts, capability.edit-opportunity, capability.move-opportunity]
  - path: backend/crm/types/custom_field_rules.go
    role: business-logic
    hash: sha256:39ab7b72c7ba884ef3be7f35d2f62a2875a818af910c1b891850c681402ec30a
    supports: [capability.create-opportunity, capability.edit-opportunity, capability.move-opportunity, troubleshooting]
  - path: backend/crm/types/task_rules.go
    role: business-logic
    hash: sha256:92c31194e686a3f86e513a30ca18e8aaac5c5b6a9e9fce927ac614d8dc3ba93e
    supports: [capability.create-opportunity, capability.move-opportunity, capability.stage-approval]
  - path: backend/crm/types/activity.go
    role: data-model
    hash: sha256:696dd8c915253d80418d7ad030fdedf9f2d687996deef9729327a9d54f12d403
    supports: [capability.log-activity]
  - path: backend/crm/types/activity_rules.go
    role: business-logic
    hash: sha256:4d8f1266376a76a30509c137f6d38bbab70322eee9c7ad07f1737a2b446c3c77
    supports: [capability.log-activity]
  - path: backend/crm/types/email_template_rules.go
    role: business-logic
    hash: sha256:a026474155c5983cc5b4b1ba5fca2791823ae94c4636dab6b32597f488189b2d
    supports: [capability.send-email]
  - path: backend/crm/types/opportunity.go
    role: data-model
    hash: sha256:9af1bd10a9466c6f87cf263eb4173480a6d19e328695dfae402883e819363298
    supports: [concepts, capability.close-opportunity]
  - path: backend/crm/types/pipeline.go
    role: data-model
    hash: sha256:272e4f0c215cf274446e0a9e22b87467413c1f54796777ff9762a4dbd4f2d639
    supports: [concepts, capability.move-opportunity]
  - path: backend/crm/types/client_provider.go
    role: data-model
    hash: sha256:d8f051ef033fae9ba37de0133d6391283f6703310848af61e3f957794b541e87
    supports: [concepts, capability.close-opportunity]
```
