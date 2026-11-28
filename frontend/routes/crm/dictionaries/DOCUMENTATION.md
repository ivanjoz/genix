---
schema: 1
page_id: crm.dictionaries
route: /crm/dictionaries
title: Dictionaries (Diccionarios)
status: implemented
visibility: tenant
description_en: >-
  Maintain the CRM value lists: the lost reasons chosen when an opportunity is closed as lost, the
  company's own activity types (next to the fixed ones such as Call or Meeting), and the email
  templates with client, opportunity, owner and company variables that sellers send from opportunities
  and stage tasks.
description_es: >-
  Mantiene las listas de valores del CRM: los motivos de pérdida que se eligen al cerrar una
  oportunidad como perdida, los tipos de actividad propios de la empresa (junto a los fijos como
  Llamada o Reunión) y las plantillas de correo con variables de cliente, oportunidad, responsable y
  empresa que se envían desde las oportunidades y las tareas.
---

# Dictionaries (Diccionarios)

<!-- DOC-ID: page-purpose -->
## Page purpose

Dictionaries (`Diccionarios`) keeps the small lists of values the CRM picks from (`catálogos`,
`listas`, `maestros`). The left column chooses which one to edit: **Lost reasons (Motivos de
pérdida)**, **Activity types (Tipos de actividad)** or **Email templates (Plantillas de correo)**.

These lists are used on **Opportunities (Oportunidades)**, **Tasks (Tareas)** and **Pipelines
(Flujos)**. The page does not hold the pipelines, stages or custom fields: those are in **Pipelines
(Flujos)**.

<!-- DOC-ID: capability.lost-reasons -->
## Maintain lost reasons (Motivos de pérdida)

### User intention (Intención del usuario)

Define why deals are lost (price, competitor, no budget…) so every lost opportunity records a
comparable reason.

### Where to find it (Dónde encontrarlo)

**CRM → Management (Gestión) → Dictionaries (Diccionarios)** at `/crm/dictionaries` → **Lost reasons
(Motivos de pérdida)**. **Add (Agregar)** puts an empty row at the top; type the name in the **Reason
(Motivo)** cell. Edit a name in its cell; the trash button deletes after confirmation. Each change
saves at once, without a save button.

### Business rules and rationale (Reglas y razón de negocio)

- The name cannot be empty; an empty edit is ignored and the cell keeps its previous value.
- Deleting is soft: opportunities already closed with that reason keep it, but it is no longer offered
  when closing.

### Result and side effects (Resultado y efectos)

The active reasons are the **Reason (Motivo)** options of **Close as lost (Cerrar como perdida)** on
the Opportunities board, and the **Performance (Desempeño)** report groups lost opportunities by
reason.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde agrego motivos de pérdida? ¿Cómo cambio la lista de razones por las que se pierde una venta?

<!-- DOC-ID: capability.activity-types -->
## Maintain activity types (Tipos de actividad)

### User intention (Intención del usuario)

Add the kinds of interaction the company tracks beyond the standard ones, for example "Delivery" or
"Tasting", each with an icon.

### Where to find it (Dónde encontrarlo)

**Activity types (Tipos de actividad)**. **Always available: (Siempre disponibles:)** lists the fixed
types: **Note (Nota)**, **Call (Llamada)**, **Meeting (Reunión)**, **Email**, **WhatsApp**, **Visit
(Visita)**, **Proposal sent (Propuesta enviada)**, **Demo (Demostración)** and **Follow-up
(Seguimiento)**. Below, the company's own types: **Add (Agregar)** adds a row, the **Type (Tipo)**
cell holds the name, the **Icon (Ícono)** cell opens **Choose an icon (Elige un ícono)**, and the
trash button deletes after confirmation. Changes save at once.

### Business rules and rationale (Reglas y razón de negocio)

- Name required, up to 60 characters. The icon comes from a fixed set of 16.
- The fixed types cannot be edited or deleted.
- Deleting a company type is soft: activities already logged with it remain, shown under the generic
  label **Activity (Actividad)**, and it is no longer offered.

### Result and side effects (Resultado y efectos)

Company types appear after the fixed ones in the activity selector of an opportunity, in the Tasks
panel and in the stage task templates of Pipelines (Flujos).

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Puedo crear un tipo de actividad propio? ¿Puedo editar "Llamada"? → no, the fixed types are always
  available as they are.

<!-- DOC-ID: capability.email-templates -->
## Maintain email templates (Plantillas de correo)

### User intention (Intención del usuario)

Prepare reusable emails (welcome, proposal follow-up, reminder) that fill in the client's and
opportunity's data automatically.

### Where to find it (Dónde encontrarlo)

**Email templates (Plantillas de correo)**. The table shows **Template (Plantilla)** and **Subject
(Asunto)**. **New (Nuevo)** opens **New template (Nueva plantilla)**; clicking a row opens it. Fields:
**Name (Nombre)**, **Subject (Asunto)** and the body editor. Variables are inserted from the pickers of
the subject and the body, grouped as **Client (Cliente)**, **Opportunity (Oportunidad)**, **Owner
(Responsable)**, **Company (Empresa)** and **Quote (Cotización)**. Save or delete with the panel's
buttons.

### Required information and prerequisites (Requisitos previos)

- Name up to 120 characters, subject up to 200 characters, and a body (up to 200 KB), all required.
- Available variables: client name and email, opportunity title and amount, owner name and email,
  company name, and quote number, total and validity.

### Business rules and rationale (Reglas y razón de negocio)

- An unknown variable is refused on save (**La variable {{…}} no existe.**).
- **Quote variables only work when sending a quote. (Las variables de cotización solo funcionan al
  enviar una cotización.)** A template using them is refused when sent from an opportunity or a task.
- Variable values are filled when the email is prepared for a specific opportunity; the amount is
  written as `1,234.56`.

### Result and side effects (Resultado y efectos)

Active templates are offered in **Send email (Enviar correo)** on an opportunity, in **Write email
(Redactar correo)** of a send email task, and as the template of send email stage tasks in Pipelines
(Flujos).

### Limitations (Limitaciones)

Deleting a template is soft; a stage task that still names it makes that pipeline's next save fail
until another template is chosen.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo una plantilla de correo para mis clientes? ¿Cómo pongo el nombre del cliente
  automáticamente en el correo?

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Opportunities (Oportunidades)** uses the lost reasons, activity types and email templates.
- **Tasks (Tareas)** uses the activity types and email templates to complete stage tasks.
- **Pipelines (Flujos)** uses activity types and email templates in the stage task templates.
- **Performance (Desempeño)** shows the lost opportunities by lost reason.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.lost-reasons]
  - path: frontend/routes/crm/dictionaries/+page.svelte
    role: page
    hash: sha256:381b2031bcf3357afad5dd11bb6df5dbd75313419024f301affb8ca158643232
    supports: [page-purpose]
  - path: frontend/routes/crm/dictionaries/LostReasonsTable.svelte
    role: user-interface
    hash: sha256:c88679c7cbaad0a4c7ecbfa58d8f63ebcb0b3a0e186e20518edb1474263c9da3
    supports: [capability.lost-reasons]
  - path: frontend/routes/crm/dictionaries/ActivityTypesTable.svelte
    role: user-interface
    hash: sha256:bb422255edcd735d6e863639a108e378e3ead800386fc5b290b0a6ff72d6a07e
    supports: [capability.activity-types]
  - path: frontend/routes/crm/dictionaries/EmailTemplatesTable.svelte
    role: user-interface
    hash: sha256:d68713431f0364bf65ae5d72ca99070b3baf32536d012cbc47be68e5badaf17a
    supports: [capability.email-templates]
  - path: frontend/services/crm/crm-activity-types.ts
    role: shared-domain
    hash: sha256:2a09bf27436973b1edb986bb368bf439fa539008249d6e4dc0587b21b01d34be
    supports: [capability.activity-types]
  - path: frontend/services/crm/crm-email-template.ts
    role: frontend-service
    hash: sha256:da6e8a2f1605b6de2dbbc4eae22c55fdef976540a548de87f8cb9b291864a94d
    supports: [capability.email-templates]
  - path: frontend/routes/crm/pipeline/pipeline.ts
    role: business-logic
    hash: sha256:fbfccc898f175d1520d9d2d9b074e98b01092e364eb41dddcb1c8a17443340ad
    supports: [capability.activity-types]
  - path: frontend/routes/crm/pipeline/OpportunityCloseModal.svelte
    role: user-interface
    hash: sha256:4b6ac6cad2c5db8701a3c15a31a52a128a6579582608d6dedfcaab87bf8c3091
    supports: [capability.lost-reasons]
  - path: backend/crm/lost_reason.go
    role: backend-handler
    hash: sha256:ca6ce94da3fe8b30fc2b6e46defe015c830b2ea9ec1f684d94c7ac6c875971c2
    supports: [capability.lost-reasons]
  - path: backend/crm/types/lost_reason.go
    role: data-model
    hash: sha256:1c807e633f7646f3120029a6cf9f80ad61e79358516789e8ce40a25749a8e07e
    supports: [capability.lost-reasons]
  - path: backend/crm/activity.go
    role: backend-handler
    hash: sha256:3767786ed83f5814e18724009ff1583c7172a7b7d8d33d61b258440c757e769a
    supports: [capability.activity-types]
  - path: backend/crm/types/activity.go
    role: data-model
    hash: sha256:696dd8c915253d80418d7ad030fdedf9f2d687996deef9729327a9d54f12d403
    supports: [capability.activity-types]
  - path: backend/crm/types/activity_rules.go
    role: business-logic
    hash: sha256:4d8f1266376a76a30509c137f6d38bbab70322eee9c7ad07f1737a2b446c3c77
    supports: [capability.activity-types]
  - path: backend/crm/email.go
    role: backend-handler
    hash: sha256:b055785ce762effc7d5221f78752b230fd49d425cdf22a3fdee99cd7b861a75e
    supports: [capability.email-templates]
  - path: backend/crm/types/email_template_rules.go
    role: business-logic
    hash: sha256:a026474155c5983cc5b4b1ba5fca2791823ae94c4636dab6b32597f488189b2d
    supports: [capability.email-templates]
  - path: backend/crm/types/task_rules.go
    role: business-logic
    hash: sha256:92c31194e686a3f86e513a30ca18e8aaac5c5b6a9e9fce927ac614d8dc3ba93e
    supports: [capability.email-templates]
  - path: backend/crm/opportunity.go
    role: backend-handler
    hash: sha256:a9b0fdd849e8f963985386bb71c6b3bcda166f3fd853bb1d7739205dbdd951cd
    supports: [capability.lost-reasons]
  - path: frontend/routes/crm/reports/ReportLostReasons.svelte
    role: user-interface
    hash: sha256:dd94ddeb30c1d933c28c830de7da3fcb60d9c7eccc7d0f4913ead804bfb96d0e
    supports: [capability.lost-reasons, related-pages]
  - path: frontend/routes/crm/reports/performance/+page.svelte
    role: page
    hash: sha256:0cf317994b40edbef9dbfb52e6347dfe71e55b06f4e43e1f51c6d5fe38708882
    supports: [capability.lost-reasons, related-pages]
```
