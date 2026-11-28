---
schema: 1
page_id: crm.settings
route: /crm/settings
title: Pipelines (Flujos)
status: implemented
visibility: tenant
description_en: >-
  Define the CRM sales pipelines: their stages (board columns) with color, default probability, stale
  days, forecast category and required amount or expected close; the custom fields each pipeline
  shows and the stage from which they are required; the tasks each stage proposes; the approver of
  each stage; and the goals period. Also maintains the company-wide catalog of custom fields.
description_es: >-
  Define los flujos de venta (pipelines) del CRM: sus etapas (columnas del tablero) con color,
  probabilidad, días para marcar estancada, categoría de forecast y monto o cierre obligatorio; los
  campos personalizados que muestra cada pipeline y desde qué etapa son obligatorios; las tareas que
  propone cada etapa; el aprobador de cada etapa; y el periodo de metas. También el catálogo de campos
  personalizados.
---

# Pipelines (Flujos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Pipelines (`Flujos`) is where an administrator designs the CRM sales process (`proceso de venta`,
`embudo`). A pipeline (`pipeline` or `flujo`) is one kanban board of the **Opportunities
(Oportunidades)** page; its stages (`etapas`) are the board's columns. For each pipeline this page
sets the stages and their rules, the custom fields its opportunities show, the tasks each stage
proposes, who must approve entering a stage, and the period of its goals. The **Custom fields
(Campos personalizados)** tab keeps the company-wide list of extra opportunity fields.

The page does not decide who sees or works in a pipeline: members and levels are set in **Pipelines:
Access (Flujos: Accesos)**. It does not hold the lost reasons, activity types or email templates:
those are in **Dictionaries (Diccionarios)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **stage** (`etapa`) has a name, a color, a default **Probability % (Probabilidad %)** given to
  opportunities entering it, **Stale days (Días estancada)** after which a card is marked **Stale
  (Estancada)** (0 = never), and a **Forecast** category: **Pipeline**, **Best case (Mejor caso)** or
  **Commit (Compromiso)**, which tells the Forecast page how to count its open opportunities.
- **Requires (Exige)**: **Amount (Monto)** and/or **Close (Cierre)** must be filled to enter the
  stage or any later stage.
- A **custom field** (`campo personalizado`) is an extra opportunity field. The catalog is shared by
  all pipelines; each pipeline picks which ones it shows and may make each one required from a stage
  on.
- A **stage task** (`tarea de la etapa`) is proposed to someone every time an opportunity enters that
  stage: log an activity or send an email. It never runs by itself.
- A stage **approver** (`aprobador`) must authorize every forward move into that stage.

<!-- DOC-ID: capability.manage-pipeline -->
## Create or edit a pipeline and its stages (Crear o editar un pipeline y sus etapas)

### User intention (Intención del usuario)

Set up a sales process: name it, list its steps in order, and give each step its probability and
rules.

### Where to find it (Dónde encontrarlo)

**CRM → Management (Gestión) → Pipelines (Flujos)** at `/crm/settings`, tab **Pipelines (Flujos)**.
The table lists each pipeline with its number of stages, custom fields and members. **New (Nuevo)**
opens **New pipeline (Nuevo pipeline)**; clicking a row opens it. Tab **Information (Información)**:
**Name (Nombre)**, **Color**, **Description (Descripción)**, **Goals (Metas)** and the table **Stages
(board columns) (Etapas (columnas del tablero))**. **Stage (Etapa)** adds a stage at the end; the
arrows reorder and the trash icon removes. Save with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required; a new pipeline starts with one stage **New (Nuevo)** at 10%.
- Every stage needs a name; at least one stage must remain.
- **Goals (Metas)**: **Monthly (Mensuales)**, **Weekly (Semanales)**, **Daily | Month (Diaria | Mes)**
  or **Daily | Week (Diaria | Semana)**; the unit in which Goals and Forecast work for this pipeline.

### Business rules and rationale (Reglas y razón de negocio)

- Stage probability must be between 0 and 100; stale days cannot be negative.
- A stage that still holds open opportunities cannot be removed: **La etapa … todavía tiene
  oportunidades abiertas; muévalas antes de eliminarla.** Removed stages are kept hidden so the
  history keeps their names.
- The order of the stages is the order of the board's columns, and it decides which stages count as
  "earlier" or "later" for required data, approvals and stage tasks.

### Result and side effects (Resultado y efectos)

The board on **Opportunities (Oportunidades)** shows the new columns and rules. The user who creates a
pipeline becomes its first member with **Full (Completo)** level, so it is visible to them.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un embudo de ventas? ¿Cómo agrego o cambio el orden de las etapas?
- ¿Cómo hago que una etapa exija el monto? ¿Qué es "Días estancada"? ¿Qué significa Mejor caso o
  Compromiso?

<!-- DOC-ID: capability.pipeline-fields -->
## Choose the custom fields of a pipeline (Campos visibles del pipeline)

### User intention (Intención del usuario)

Show extra fields (for example product line or source of the lead) on this pipeline's opportunities
and force them to be filled from a given stage.

### Where to find it (Dónde encontrarlo)

Pipeline panel → **Information (Información)** → **Custom fields shown (Campos personalizados
visibles)**: tick each field; for a ticked field choose **Required from (Obligatorio desde)**:
**Optional (Opcional)** or a stage.

### Business rules and rationale (Reglas y razón de negocio)

- A field required from a stage is required in that stage and every later one.
- Only saved stages can be chosen: **Save the pipeline to require a field from a stage you just
  added. (Guarde el pipeline para exigir un campo desde una etapa recién agregada.)**
- Required from the first stage means required when the opportunity is created.
- The list shows only the fields of the catalog; create them first in **Custom fields (Campos
  personalizados)**.

<!-- DOC-ID: capability.stage-tasks -->
## Configure the tasks each stage proposes (Tareas de las etapas)

### User intention (Intención del usuario)

Make sure the seller does the follow-up each step needs, for example call the client the next day or
send the proposal email.

### Where to find it (Dónde encontrarlo)

Pipeline panel → tab **Tasks (Tareas)**. **Task (Tarea)** adds a row in the first stage. Columns:
**Stage (Etapa)**, **Task (Tarea)** (its title), **Kind (Tipo)** (**Log activity (Registrar
actividad)** or **Send email (Enviar correo)**), **Activity / Template (Actividad / Plantilla)**,
**Due (days) (Vence (días))** and **Assign to (Asignar a)** (**Card owner (Responsable de la
tarjeta)** or a fixed user).

### Business rules and rationale (Reglas y razón de negocio)

- Title required, up to 120 characters; due days from 0 to 365, counted from the day the card enters
  the stage.
- A send email task needs an active email template; an activity task needs an existing activity type.
- A fixed assignee must be an active user.
- Only an unsaved task can change stage; a saved one stays in its stage.

### Result and side effects (Resultado y efectos)

Each time an opportunity enters the stage, one pending task per template appears in **Tasks
(Tareas)** for the assignee, unless the same task is still pending for that opportunity.

<!-- DOC-ID: capability.stage-approvals -->
## Set stage approvers (Aprobadores de etapa)

### User intention (Intención del usuario)

Require a supervisor's authorization before a deal advances into a sensitive stage.

### Where to find it (Dónde encontrarlo)

Pipeline panel → tab **Approvals (Aprobaciones)**: for each stage choose an **Approver (Aprobador)**
or **No approval (Sin aprobación)**.

### Business rules and rationale (Reglas y razón de negocio)

- The approver must be an active user; they do not need to be a member of the pipeline.
- Only forward moves into the stage wait for approval; moving back does not. A forward move cannot
  skip a stage that has an approver.

### Result and side effects (Resultado y efectos)

A forward move into the stage becomes an approval request in the approver's **Tasks (Tareas)** and
the card shows **Pending approval (Por aprobar)** until it is decided.

<!-- DOC-ID: capability.delete-pipeline -->
## Delete a pipeline (Eliminar un pipeline)

### Where to find it (Dónde encontrarlo)

Open the pipeline → the panel's delete button, then confirm.

### Business rules and rationale (Reglas y razón de negocio)

A pipeline with open opportunities cannot be deleted: **El pipeline todavía tiene oportunidades
abiertas; muévalas antes de eliminarlo.** A deleted pipeline leaves the board selector and the CRM
reports.

<!-- DOC-ID: capability.custom-fields -->
## Maintain the custom fields catalog (Catálogo de campos personalizados)

### User intention (Intención del usuario)

Create the extra fields the pipelines can show on opportunities.

### Where to find it (Dónde encontrarlo)

Tab **Custom fields (Campos personalizados)**. The table shows **Field (Campo)**, **Type (Tipo)** and
the number of **Options (Opciones)**. **New (Nuevo)** opens **New field (Nuevo campo)** with **Name
(Nombre)** and **Type (Tipo)**; select types add an **Options (Opciones)** table with **Option
(Opción)** to add rows.

### Required information and prerequisites (Requisitos previos)

Types: **Text (Texto)**, **Long text (Texto largo)**, **Integer (Entero)**, **Decimal**, **Money
(Monto)**, **Date (Fecha)**, **Select (Selección)**, **Multi-select (Selección múltiple)**,
**Checkbox (Casilla)** and **User (Usuario)**. Select and multi-select need at least one option.

### Business rules and rationale (Reglas y razón de negocio)

- The type cannot change once an opportunity holds a value for the field.
- Removing an option hides it for new values; opportunities keep the value they stored.
- Deleting a field (panel's delete button, then confirm) removes it from the catalog, from the
  **Custom fields shown (Campos personalizados visibles)** list and from the opportunity forms.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo agrego un campo propio a las oportunidades? ¿Puedo poner una lista desplegable?
- ¿Por qué no puedo cambiar el tipo del campo?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El pipeline necesita al menos una etapa activa.":** keep at least one stage.
- **"El campo personalizado … es obligatorio desde una etapa que no existe o fue eliminada.":**
  choose another stage in **Required from (Obligatorio desde)**.
- **"La plantilla de correo de la tarea … no existe." / "El tipo de actividad de la tarea … no
  existe.":** the template or type was deleted in Dictionaries; choose another one.
- **"El aprobador de la etapa … no existe o está inactivo.":** choose an active user or **No approval
  (Sin aprobación)**.
- **"No se puede cambiar el tipo de un campo que ya tiene valores.":** create a new field instead.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Opportunities (Oportunidades)** is the board these pipelines define.
- **Pipelines: Access (Flujos: Accesos)** sets who sees each pipeline and at what level.
- **Dictionaries (Diccionarios)** holds the activity types and email templates the stage tasks use.
- **Tasks (Tareas)** receives the stage tasks and approval requests.
- **Goals (Metas)** and **Forecast** use each pipeline's goals period and the stages' forecast
  category.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.manage-pipeline]
  - path: frontend/routes/crm/settings/+page.svelte
    role: page
    hash: sha256:71d9677c61d393ae467d1c41f2953117fc9e1c8e0513bac8fe75d5d297633394
    supports: [page-purpose, capability.custom-fields]
  - path: frontend/routes/crm/settings/PipelinesTab.svelte
    role: user-interface
    hash: sha256:4b65e9f5c6b5f39407c2caeb8f32ecb34a0bb6309855dad9a099be5064ab43a4
    supports: [page-purpose, concepts, capability.manage-pipeline, capability.pipeline-fields, capability.stage-tasks, capability.stage-approvals, capability.delete-pipeline, troubleshooting]
  - path: frontend/routes/crm/settings/CustomFieldsTab.svelte
    role: user-interface
    hash: sha256:020d7d76fc04683526477597fba0f4337925c99374b38c05e245bbdf68585206
    supports: [capability.custom-fields]
  - path: frontend/routes/crm/settings/settings.ts
    role: business-logic
    hash: sha256:6ece7897a7b8781b528ccba40bcbefa4ea00fcf27d5067d6db1549a81742a041
    supports: [capability.manage-pipeline, capability.pipeline-fields, capability.stage-tasks, capability.custom-fields]
  - path: frontend/services/crm/crm-pipeline.svelte.ts
    role: frontend-service
    hash: sha256:2d85131a009325fddeb1f86b4f571ecdcbf18471951ba1219e514ce45b33523d
    supports: [concepts, capability.pipeline-fields]
  - path: backend/crm/pipeline.go
    role: backend-handler
    hash: sha256:d4f1b83c8db222b41fd9955a2ed6ca09e6960846b76ec74401aa3fe67c95feb7
    supports: [capability.manage-pipeline, capability.stage-tasks, capability.stage-approvals, capability.delete-pipeline]
  - path: backend/crm/custom_field.go
    role: backend-handler
    hash: sha256:a89c146b627e51f5d7df9b9e3dd1e09f2df51ec75720b1b8aed6ed1b35c5efda
    supports: [capability.custom-fields, troubleshooting]
  - path: backend/crm/types/pipeline.go
    role: data-model
    hash: sha256:272e4f0c215cf274446e0a9e22b87467413c1f54796777ff9762a4dbd4f2d639
    supports: [concepts, capability.manage-pipeline, capability.pipeline-fields, capability.stage-tasks]
  - path: backend/crm/types/pipeline_rules.go
    role: business-logic
    hash: sha256:f574d970c74d1cf8966d1bcde3c8b9640af596d51830624e06a605650c757c48
    supports: [capability.manage-pipeline, capability.pipeline-fields, troubleshooting]
  - path: backend/crm/types/custom_field.go
    role: data-model
    hash: sha256:78adbc523ad881543f590d527df519d42787d3308fd107c13f5ba787abba26e7
    supports: [capability.custom-fields]
  - path: backend/crm/types/custom_field_rules.go
    role: business-logic
    hash: sha256:39ab7b72c7ba884ef3be7f35d2f62a2875a818af910c1b891850c681402ec30a
    supports: [concepts, capability.pipeline-fields, capability.custom-fields, troubleshooting]
  - path: backend/crm/types/task_rules.go
    role: business-logic
    hash: sha256:92c31194e686a3f86e513a30ca18e8aaac5c5b6a9e9fce927ac614d8dc3ba93e
    supports: [capability.stage-tasks, capability.stage-approvals, troubleshooting]
  - path: backend/crm/types/report_rules.go
    role: business-logic
    hash: sha256:39dff4dbbaefd5888caee45a3520401a1510e3b68f673acf24084b63129aa16d
    supports: [capability.delete-pipeline]
  - path: frontend/routes/crm/pipeline/pipeline.ts
    role: business-logic
    hash: sha256:fbfccc898f175d1520d9d2d9b074e98b01092e364eb41dddcb1c8a17443340ad
    supports: [capability.custom-fields, capability.delete-pipeline]
```
