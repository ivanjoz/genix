---
schema: 1
page_id: configuration.backups
route: /configuration/backups
title: Backups & Restore (Copias de Seguridad y Restauración)
status: implemented
visibility: tenant
description_en: >-
  Database backup management. Generate an on-demand backup of the company's operational data,
  download an existing backup file, and restore the database to the state of a selected backup.
description_es: >-
  Gestión de copias de seguridad de la base de datos. Generar un backup bajo demanda de la
  información operativa de la empresa, descargar un backup existente y restaurar la base de
  datos al estado de un backup seleccionado.
---

# Backups & Restore (Copias de Seguridad y Restauración)

<!-- DOC-ID: page-purpose -->
## Page purpose

Backups (`Backups`, also `respaldos` or `copias de seguridad`) manages point-in-time snapshots
of the company's own operational data: sales, inventory, finance, logistics, and every other
business table backed by Genix's ScyllaDB storage. From this page a user generates a new
snapshot, downloads an existing one, and restores the database back to a previously generated
snapshot.

This page does not own company configuration (name, payment gateway, email sending — that is
**My Company / Mi Empresa**), nor Security profiles and individual access assignments (**Users
/ Usuarios**, **Profiles & Access / Perfiles & Accesos**): those live in a separate storage
family and are never included in a backup or touched by a restore performed here. There is also
no automatic or scheduled backup job for a tenant; every backup listed on this page was produced
by pressing **Generate** here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **backup (`backup`, `respaldo`)** is a `.tar` archive with one zstd-compressed CSV file per
  operational table, stored per company in cloud storage. The table on this page shows each
  backup's creation date/time (`Created`), file `Name`, and `Size` in MB.
- The list is ordered so the **most recently generated backup always appears first**, followed
  by older ones — the file name itself is built from the creation time so that simply sorting
  names puts the newest backup at the top.
- **Restoring (`restaurar`)** a backup is a full replace, not a merge: for every table the backup
  contains, Genix first deletes all of that company's current rows in that table and then
  re-inserts exactly the rows recorded in the backup. Anything created or edited in those tables
  after the backup was generated is lost once that backup is restored.
- Company settings (`Mi Empresa`) and Security's profiles/individual accesses (`Perfiles &
  Accesos`, `Usuarios`) are outside the scope of both backup and restore — they keep their
  current values through a restore, because they are not part of the table set this page backs
  up.

<!-- DOC-ID: capability.browse-backups -->
## Browse available backups (Ver los backups disponibles)

Open **Administration (Administración) → Configuration (Configuración) → Backups** at
`/configuration/backups`. The left table lists up to the **latest 30 backups** stored for the
company — the listing request caps at 30 objects with no further pages, so once more than 30
backups exist, older ones stop appearing here even though they still exist in storage. Clicking
a row selects it (click again to deselect) and populates the **Restore** panel on the right;
nothing else on the page changes from a selection alone.

Viewing this list requires no dedicated permission entry in the access catalog — any
authenticated user of the company can see it, the same way viewing the Users list is open by
default. Acting on **Generate** or **Restore**, by contrast, always requires the "Backups"
access at the **Full (Todo)** level, because that access only offers **View (Visualizar)** and
**Full (Todo)** — there is no separate Create/Edit granularity for backups, so a user must have
Full to run either write action.

<!-- DOC-ID: capability.generate-backup -->
## Generate a new backup (Generar un backup)

### User intention (Intención del usuario)

Create a fresh snapshot of the current operational data — for example before a risky bulk
change, a data migration, or as routine precaution.

### Where to find it (Dónde encontrarlo)

The green **+** button at the top right of the Backups toolbar. Confirm with **Generate
Backup|Generar Backup** — "Do you want to generate the backup now? / ¿Desea generar el backup
ahora?" — Yes/No.

### Required information and prerequisites (Requisitos previos)

None: the action takes no parameters from the user. The acting user needs the "Backups" access
at Full level (see above).

### Business rules and rationale (Reglas y razón de negocio)

On confirmation, the server exports every ScyllaDB-backed business table for the requesting
user's own company to CSV, zstd-compresses each CSV, and packs them into one tar file uploaded
to the company's backup storage. Company configuration and Security's profiles/accesses are not
part of this export because they are stored outside this table set.

### Result and side effects (Resultado y efectos)

A new file appears in the company's backup storage and, after the page reloads the list, at the
top of this page's table. Generating a backup is a read-only snapshot operation: it does not
change any existing data. Once more than 30 backups exist, the oldest one drops off this list
(though it remains in storage).

### Limitations (Limitaciones)

There is no scheduling here — backups are not created automatically on any interval; this button
is the only user-reachable way to produce one. A backup cannot be named, labeled, or annotated by
the user; it is only identified by its automatic file name and creation date.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo genero un backup manualmente?`
- `¿Los backups se generan automáticamente?` No, sólo al presionar "Generar" en esta página.
- Search terms: `generar backup`, `respaldo manual`, `copia de seguridad`, `backup bajo demanda`.

<!-- DOC-ID: capability.download-backup -->
## Download a backup (Descargar un backup)

### User intention (Intención del usuario)

Retrieve the raw backup file — for example to store it outside Genix or hand it to support.

### Where to find it (Dónde encontrarlo)

Select a row in the table; the right panel shows the selected backup's name and size with a
purple download button. Without a selection, the panel shows **Select a Backup / Seleccione un
Backup** instead.

### Required information and prerequisites (Requisitos previos)

A backup must already be selected in the table.

### Business rules and rationale (Reglas y razón de negocio)

**Verified defect:** the download link is built with the company segment hardcoded to the
literal value `1`, not the signed-in user's actual company ID. For any tenant whose company ID
is not `1`, the generated link points at company `1`'s backup folder instead of the tenant's
own — the download fails (file not found) or, if a same-named file happens to exist there,
could resolve to an unrelated company's backup. This is unlike backup generation, listing, and
restore, which all correctly scope storage access to the signed-in user's real company ID on the
server.

### Result and side effects (Resultado y efectos)

Opens/starts a file download in a new tab; nothing in Genix's data changes.

### Limitations (Limitaciones)

Besides the company-ID defect above, there is no in-page preview of a backup's contents before
downloading, and no way to download more than one backup at a time.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo descargo un backup?`
- `¿Por qué no puedo descargar mi backup?` Si la empresa no es la número 1, el enlace actual
  apunta a la carpeta equivocada; es un defecto conocido, no una restricción de permisos.
- Search terms: `descargar backup`, `bajar respaldo`, `archivo de backup`.

<!-- DOC-ID: capability.restore-backup -->
## Restore a backup (Restaurar un backup)

### User intention (Intención del usuario)

Roll the company's operational data back to the exact moment a specific backup was generated —
for example after data corruption, a bad import, or an unwanted bulk change.

### Where to find it (Dónde encontrarlo)

Select the backup in the table, then use the blue **Restore|Restaurar** button in the right
panel. Confirm with **Restore Backup|Restaurar Backup** — "Restore the backup from `<date>` /
Restaurar el backup realizado el `<date>`" — Yes/No.

### Required information and prerequisites (Requisitos previos)

An existing backup selected from this company's own list; the server looks the file up by name
inside the company's own storage folder, so it can only restore a backup that this company
generated. The acting user needs the "Backups" access at Full level.

### Business rules and rationale (Reglas y razón de negocio)

For every table found inside the backup archive, Genix **deletes all of that company's current
rows in that table** and then **re-inserts every row recorded in the backup** — a full replace,
not a merge. Archive entries named `company`, `accesos`, or `perfiles` are explicitly skipped
during restore (those live in a different storage family, outside this mechanism), and any
archived table name with no matching registered table is skipped as well, so the process never
gets stuck on data it does not recognize; both are logged rather than restored. After each
table is restored, Genix recalculates that table's internal ID counter from the restored rows so
that new records created afterward continue from the correct next number instead of colliding
with restored IDs.

Once the restore call succeeds, the app also clears the browser's entire locally cached copy of
delta/synced data for the current environment, so every page re-fetches fresh data from the
server instead of continuing to show values cached before the restore. This cache clear does not
by itself reload the currently open page.

### Result and side effects (Resultado y efectos)

Every table included in the backup returns exactly to that backup's contents for the current
company; tables not included in the backup, company configuration, and Security profiles/access
assignments are left untouched.

### Limitations (Limitaciones)

Restore is all-or-nothing for the tables a backup contains — there is no option to restore only
one table. There is no undo for a completed restore other than restoring a different (for
example, more recent) backup. A restore can silently discard recent legitimate work on any
restored table, since it always overwrites current rows with the backup's rows; there is no
confirmation step beyond the single Yes/No dialog, and no dry-run or preview of what will change.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo restauro un backup?`
- `¿La restauración borra lo que hice después del backup?` Sí, en cada tabla incluida en el
  backup se borran los registros actuales antes de insertar los del backup.
- `¿Restaurar cambia mis usuarios, perfiles o los datos de mi empresa?` No, esos no forman parte
  del backup ni de la restauración.
- Search terms: `restaurar backup`, `revertir base de datos`, `deshacer cambios`, `rollback`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Generating and restoring a backup both require the "Backups" access at Full (Todo) level;
  viewing the list itself requires no explicit access mapping. This mirrors how Users and
  Profiles & Access only offer View/Full for their own management actions.
- Backup generation, listing, and restore all operate on the signed-in user's own company only,
  read from the session on the server — the one exception is the download link's known
  company-ID defect documented under **Download a backup**.
- Generate and Restore are the only two actions on this page, and both require an explicit
  Yes/No confirmation dialog before running.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Select a Backup / Seleccione un Backup"**: no row is selected yet; click one in the table
  first.
- **Download fails or opens the wrong file**: known defect — the download link currently points
  at company `1`'s storage folder regardless of the signed-in company; see **Download a
  backup**.
- **The Upload button does nothing**: the blue upload-icon button in the toolbar has no action
  wired to it; uploading an external backup file is not implemented on this page despite the
  button being visible.
- **An old backup is missing from the list**: only the latest 30 backups are returned; older
  ones still exist in storage but are not shown here.
- **Data still looks outdated right after a restore**: the app clears its local cache
  automatically, but a page already open in the browser may keep showing data it rendered before
  the restore — reload or re-navigate to it.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **My Company (Mi Empresa)** at `/configuration/parameters`: company configuration is not
  captured by backup/restore; edit it there directly.
- **Users (Usuarios)** at `/security/users` and **Profiles & Access (Perfiles & Accesos)** at
  `/security/access-profiles`: user accounts, profiles, and access assignments are never
  included in a backup or affected by a restore performed from this page.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.browse-backups, related-pages]
  - path: frontend/routes/configuration/backups/+page.svelte
    role: page
    hash: sha256:6491ab92aab89f7dd3aa61de441f6daeaf814744808be2f63cca65e796d3088b
    supports: [page-purpose, concepts, capability.browse-backups, capability.generate-backup, capability.download-backup, capability.restore-backup, rules, troubleshooting]
  - path: frontend/routes/configuration/backups/backups.svelte.ts
    role: frontend-service
    hash: sha256:19c1eb7043c3428105603c8159c8af3d1914d79a9b9d96daf65d1ac8801ba3e1
    supports: [concepts, capability.browse-backups, capability.generate-backup, capability.restore-backup]
  - path: frontend/core/env.ts
    role: frontend-service
    hash: sha256:305965c6c7acfaa89726a484504e1130874bd36744f1f9d31b072e4eaefa4bb8
    supports: [capability.download-backup, troubleshooting]
  - path: frontend/packages/genix-ui/service-worker/client.ts
    role: frontend-service
    hash: sha256:840367950e8cf6af0f5f1810df6c17a169c4f3ceb8c4c947f1d7b1f86ee45eca
    supports: [capability.restore-backup]
  - path: frontend/packages/genix-ui/service-worker/service-worker.ts
    role: frontend-service
    hash: sha256:5a360adc7a556db2f27d99e36540b15f87e6ee20fc457e68d766d7e8148b5f9b
    supports: [capability.restore-backup]
  - path: frontend/packages/genix-ui/cache/delta-cache.fetch.ts
    role: business-logic
    hash: sha256:0130b65e7b0db8043708173beb9817b11905357ef503075f86f9ded5212e57e7
    supports: [capability.restore-backup]
  - path: frontend/packages/genix-ui/buttons/Button.svelte
    role: user-interface
    hash: sha256:259cade6a388d1d274e343996bc97fdb73d0a332801529d364434b9b77ee33af
    supports: [troubleshooting]
  - path: backend/exec/backup.go
    role: backend-handler
    hash: sha256:6dbd6e4e878b1f5d91e002a84e0dfadad92ea1ad0904ca04c83792f182f670d0
    supports: [page-purpose, concepts, capability.browse-backups, capability.generate-backup, rules]
  - path: backend/exec/restore.go
    role: backend-handler
    hash: sha256:2ab65c6aafac120adef4e16259662668c2e7eeff30f77eaaefd2b214f4b2e5d2
    supports: [concepts, capability.restore-backup, rules, troubleshooting]
  - path: backend/exec/main.go
    role: backend-handler
    hash: sha256:3f29f92796fac3819bd31ea694f3563852e08af6e3ab2ea096e82e3b241576d3
    supports: [page-purpose, capability.generate-backup]
  - path: backend/genix-orm/scylla/deploy.go
    role: business-logic
    hash: sha256:bb2bd71cb7531fce4f2a68a40569e5f961dd24e891f84224fad6a7d8870deeea
    supports: [concepts, capability.restore-backup]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [capability.browse-backups, capability.generate-backup, capability.restore-backup, rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [capability.browse-backups, capability.generate-backup, capability.restore-backup, rules]
```
