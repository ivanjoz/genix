---
schema: 1
page_id: crm.pipeline-access
route: /crm/pipeline-access
title: 'Pipelines: Access (Flujos: Accesos)'
status: implemented
visibility: tenant
description_en: >-
  Grid of users by CRM pipelines where an administrator gives each user a level in each pipeline:
  None, View, Edit or Full. A user only sees the pipelines they are assigned to; Edit creates and moves
  opportunities, Full also edits any opportunity and reassigns its owner.
description_es: >-
  Cuadrícula de usuarios por pipelines del CRM donde se asigna a cada usuario su nivel en cada flujo:
  Ninguno, Ver, Editar o Completo. Un usuario solo ve los pipelines asignados; Editar crea y mueve
  oportunidades, Completo también edita cualquier oportunidad y reasigna su responsable.
---

# Pipelines: Access (Flujos: Accesos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Pipelines: Access (`Flujos: Accesos`) decides which CRM pipelines each user can see and what they can
do in each one (`permisos por pipeline`, `miembros del flujo`). It is a grid: one row per active user,
one column per pipeline, and in each cell the user's level in that pipeline.

The page does not create pipelines or change their stages: that is **Pipelines (Flujos)**. It does
not grant access to the CRM pages themselves; page access is given by the user's profile in **Users &
Profiles (Usuarios & Perfiles)**.

<!-- DOC-ID: concepts -->
## Member levels (Niveles de acceso al pipeline)

As the page explains: **A user only sees the pipelines they are assigned to. Edit can create and move
opportunities; Full can also edit any opportunity and reassign its owner. (Un usuario solo ve los
pipelines que tiene asignados. Editar puede crear y mover oportunidades; Completo también edita
cualquier oportunidad y reasigna su responsable.)**

- **None (Ninguno):** not a member; the pipeline is not offered on the board and its opportunities
  cannot be opened.
- **View (Ver):** sees the board and the opportunities, writes activities and sends emails, but does
  not create, move, close or edit.
- **Edit (Editar):** also creates, moves and closes opportunities, and edits the ones they created or
  own. Moving to another pipeline needs Edit in both pipelines.
- **Full (Completo):** also edits any opportunity of the pipeline and changes its owner.

Any level makes the pipeline available in the CRM reports for that user.

<!-- DOC-ID: capability.assign-levels -->
## Assign users to pipelines (Asignar usuarios a los pipelines)

### User intention (Intención del usuario)

Give a new seller access to their sales pipeline, change what someone can do, or remove someone.

### Where to find it (Dónde encontrarlo)

**CRM → Management (Gestión) → Pipelines: Access (Flujos: Accesos)** at `/crm/pipeline-access`.
Pick the level in the cell of the user (row **User (Usuario)**) and the pipeline (column), then
press **Save (Guardar)**.

### Business rules and rationale (Reglas y razón de negocio)

- Changes are kept on screen until **Save (Guardar)**; the button stays disabled while nothing changed.
  Saving sends only the pipelines whose members changed.
- Only active users are listed and accepted.
- The creator of a pipeline is added automatically as **Full (Completo)** when the pipeline is
  created in Pipelines (Flujos).

### Result and side effects (Resultado y efectos)

The user gets or loses the pipeline in the **Opportunities (Oportunidades)** selector,
with the new level's actions, and in the CRM reports.

### Limitations (Limitaciones)

- A stage approver does not need to be a member to approve; being an approver gives no access to the
  board.
- Seeing every pipeline in the CRM reports without being a member is not set here; it is the **CRM
  Reporte completo** access in Users & Profiles (Usuarios & Perfiles).

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo le doy acceso a un vendedor a un pipeline? ¿Por qué un usuario no ve el tablero de
  oportunidades?
- ¿Qué diferencia hay entre Ver, Editar y Completo? ¿Quién puede reasignar oportunidades?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **A user says "Aún no tiene pipelines asignados":** give them at least **View (Ver)** in a pipeline
  here.
- **A user cannot create or drag cards:** their level is **View (Ver)**; raise it to **Edit (Editar)**.
- **A user cannot change the owner of an opportunity:** only **Full (Completo)** can.
- **A user does not appear in the grid:** only active users are listed.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Pipelines (Flujos)** creates the pipelines shown as columns here.
- **Opportunities (Oportunidades)** is where the levels take effect.
- **Users & Profiles (Usuarios & Perfiles)** gives access to the CRM pages and the **CRM Reporte
  completo** access.
- **Timeline (Línea de tiempo)** and the other CRM reports cover the pipelines where the user is a
  member.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.assign-levels]
  - path: frontend/routes/crm/pipeline-access/+page.svelte
    role: page
    hash: sha256:078483991c77ec1c6c7517ce3b358f1ebac35ae06ac6d1a3b42412b4513d6df4
    supports: [page-purpose, concepts, capability.assign-levels, troubleshooting]
  - path: frontend/routes/crm/pipeline-access/pipeline-access.ts
    role: business-logic
    hash: sha256:c2ec38a62a48667ce337f09a285420c9b052f442610f174bcda0e4aa7984eaf5
    supports: [concepts, capability.assign-levels]
  - path: frontend/services/crm/crm-pipeline.svelte.ts
    role: frontend-service
    hash: sha256:2d85131a009325fddeb1f86b4f571ecdcbf18471951ba1219e514ce45b33523d
    supports: [capability.assign-levels]
  - path: backend/crm/pipeline.go
    role: backend-handler
    hash: sha256:d4f1b83c8db222b41fd9955a2ed6ca09e6960846b76ec74401aa3fe67c95feb7
    supports: [capability.assign-levels]
  - path: backend/crm/types/pipeline_rules.go
    role: business-logic
    hash: sha256:f574d970c74d1cf8966d1bcde3c8b9640af596d51830624e06a605650c757c48
    supports: [concepts, capability.assign-levels]
  - path: backend/crm/types/pipeline.go
    role: data-model
    hash: sha256:272e4f0c215cf274446e0a9e22b87467413c1f54796777ff9762a4dbd4f2d639
    supports: [concepts]
  - path: backend/crm/opportunity.go
    role: backend-handler
    hash: sha256:a9b0fdd849e8f963985386bb71c6b3bcda166f3fd853bb1d7739205dbdd951cd
    supports: [concepts]
  - path: backend/crm/activity.go
    role: backend-handler
    hash: sha256:3767786ed83f5814e18724009ff1583c7172a7b7d8d33d61b258440c757e769a
    supports: [concepts]
  - path: backend/crm/types/report_rules.go
    role: business-logic
    hash: sha256:39dff4dbbaefd5888caee45a3520401a1510e3b68f673acf24084b63129aa16d
    supports: [concepts, capability.assign-levels]
  - path: backend/crm/types/task_rules.go
    role: business-logic
    hash: sha256:92c31194e686a3f86e513a30ca18e8aaac5c5b6a9e9fce927ac614d8dc3ba93e
    supports: [capability.assign-levels]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [capability.assign-levels, related-pages]
  - path: frontend/routes/crm/pipeline/+page.svelte
    role: user-interface
    hash: sha256:c4735df7a4d87d722145f6c05e8a6097f4f9b40fddd4f0d6006ef8f396047f3e
    supports: [troubleshooting]
```
