---
schema: 1
page_id: company.configuration
route: /company/configuration
title: Configuration (Configuración)
status: implemented
visibility: tenant
description_en: >-
  Tenant company settings and database backups on one page, split into three tabs. My Company edits
  the company's name, tax ID, legal name, email, phone, address, city and representative. Store
  holds the Culqi payment-gateway keys for the online store, separated into Culqi Test and Culqi
  Live sections. Backups generates an on-demand snapshot of the company's operational data,
  downloads an existing backup file, and restores the database to a selected backup.
description_es: >-
  Configuración de la empresa del tenant y copias de seguridad en una sola página, dividida en tres
  pestañas. Mi Empresa edita nombre, RUC, razón social, correo, teléfono, dirección, ciudad y
  representante. Tienda contiene las llaves de la pasarela de pago Culqi para la tienda online,
  separadas en las secciones Culqi Pruebas y Culqi Live. Backups genera un respaldo bajo demanda de
  la información operativa, descarga un backup existente y restaura la base de datos al estado de un
  backup seleccionado.
---

# Configuration (Configuración)

<!-- DOC-ID: page-purpose -->
## Page purpose

Configuration (`Configuración`) is the single settings page for the company (`empresa`) that owns
the current tenant session. It is divided into three tabs shown at the top of the page:

- **My Company (Mi Empresa)** — edits the identity data of one company record (name, tax ID/RUC,
  legal name/razón social, email, phone, address, city, representative). Its form heading reads
  "Company Parameters" (`Parámetros de la Empresa`).
- **Store (Tienda)** — the online store's settings on the same company record. Today it holds one
  card, **Culqi Configuration (Configuración Culqi)**, split into a **Culqi Test (Culqi Pruebas)**
  section, a **Culqi Live** section, and an **RSA Encryption (Encriptación RSA)** section.
- **Backups** (also `respaldos` or `copias de seguridad`) — manages point-in-time snapshots of the
  company's own operational data: sales, inventory, finance, logistics and every other business
  table backed by Genix's ScyllaDB storage. From here a user generates a new snapshot, downloads an
  existing one, and restores the database back to a previously generated snapshot.

My Company and Store edit the same underlying company record and share one Save button each, but
Backups is independent of both: a backup never contains the company configuration edited on those
two tabs, and a restore never overwrites it. Security's users, profiles and access
assignments (**Users & Profiles / Usuarios & Perfiles**) are likewise outside backup and restore
scope, and are edited on their own page.

This page is not the SaaS-wide **Companies (Empresas)** admin screen at `/system/companies`, which
lists every tenant registered on the platform and is restricted to the company that administers the
SaaS itself. My Company only ever reads and writes the record belonging to the signed-in user's own
company; it has no picker and cannot open another tenant's record. There is also no automatic or
scheduled backup job for a tenant: every backup listed here was produced by pressing **Generate**
on the Backups tab.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- The **company (empresa)** record is the same record used across the tenant: its RUC and
  `LegalName`/razón social also feed the sales receipt/voucher (`comprobante`) header on the
  Point of Sale (`Punto de Venta`) screen, which is why POS can read this data even without the
  Configuración access (see permissions below).
- **Culqi Configuration (Configuración Culqi)**, on the **Store (Tienda)** tab, keys configure the
  Culqi payment gateway used by the online store: a test mode pair (`Llave Pública`/`Llave Privada`
  under **Culqi Pruebas**) and a live mode pair (the same two labels under **Culqi Live**), plus one
  RSA key/RSA key ID pair used for Culqi's 3-D Secure/antifraud flow. There is a single RSA pair for
  both environments, not one per environment, which is why it sits in its own section rather than
  inside either one.
- **Outgoing notification email** is *not* configured here. Genix sends mail (sign-up
  verification, contact form) through a single platform-wide mail server configured by the
  operator; a company cannot point Genix at its own mail server from any screen.
- A **backup (`backup`, `respaldo`)** is a `.tar` archive with one zstd-compressed CSV file per
  operational table, stored per company in cloud storage. The table on the Backups tab shows each
  backup's creation date/time (`Created`), file `Name`, and `Size` in MB.
- The backup list is ordered so the **most recently generated backup always appears first**,
  followed by older ones — the file name itself is built from the creation time so that simply
  sorting names puts the newest backup at the top.
- **Restoring (`restaurar`)** a backup is a full replace, not a merge: for every table the backup
  contains, Genix first deletes all of that company's current rows in that table and then
  re-inserts exactly the rows recorded in the backup. Anything created or edited in those tables
  after the backup was generated is lost once that backup is restored.
- The company record edited on the My Company tab and Security's profiles/individual accesses
  (`Perfiles & Accesos`, `Usuarios`) are outside the scope of both backup and restore — they keep
  their current values through a restore, because they are not part of the table set the Backups
  tab backs up.

<!-- DOC-ID: capability.edit-company-data -->
## Edit company data (Editar los datos de la empresa)

### User intention (Intención del usuario)

Keep the tenant's own legal/operational identity (name, tax ID, legal name, contact data) current,
since this is the same data used on sales receipts and in company-wide notifications.

### Where to find it (Dónde encontrarlo)

Open **Administration (Administración) → Configuration (Configuración)** at
`/company/configuration` and stay on the first tab, **My Company (Mi Empresa)**. Edit the fields in
the form directly (no separate edit mode) and use the top-right **Save (Guardar)** button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)**, **RUC**, and **Legal Name (Razón Social)** are marked required on the form;
  the page blocks Save locally and names the empty ones, e.g. "Missing required data: RUC, Legal
  Name (Faltan datos a guardar: RUC, Razón Social)".
- **Email (Correo Electrónico)**, **Phone (Teléfono)**, **Representative (Representante)**, **Legal
  Address (Dirección Legal)**, and **City (Ciudad)** are optional. Email is optional because
  companies created by seeding or import have none; only public sign-up sets one.
- The server independently rejects a save whose Name, RUC, or Legal Name is empty with
  "Falta alguno de los siguiente parámetros: Nombre, Razon-Social, RUC." — but, unlike the
  SaaS **Companies** page, it does not enforce any minimum character length on these values here.

### Business rules and rationale (Reglas y razón de negocio)

Saving always targets the signed-in user's own company ID; the ID present in the record on screen
is not sent as a separate selector. The server also guarantees the company has an internal form
API key, generating a random one only the first time it is missing, so a save never revokes an
existing key even though this page has no field to view or regenerate it.

### Result and side effects (Resultado y efectos)

A successful save updates the single company record and republishes a small public JSON file used
by the online store, carrying the company name plus the Culqi Public Key (Test), RSA Key and RSA
Key ID. It does not create any other record, and the same updated company feeds the POS receipt
header the next time a sale is created.

### Limitations (Limitaciones)

A save writes the whole company record, so a field left blank on screen is stored blank — the form
is the complete state of the company, not a patch over it.

There is no visible field for the company's internal form API key, and this page cannot regenerate
it.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo edito el RUC o la razón social de mi empresa?`
- `¿Dónde configuro los datos de mi empresa (no de otra empresa del sistema)?` Aquí, en la pestaña
  Mi Empresa; la administración de todas las empresas del SaaS está en otra pantalla reservada al
  administrador de la plataforma.
- Search terms: `empresa`, `RUC`, `razón social`, `mi empresa`, `datos de la empresa`, `teléfono`,
  `dirección`, `representante`.

<!-- DOC-ID: capability.configure-culqui -->
## Configure Culqi ecommerce keys (Configurar llaves de Culqi para ecommerce)

### User intention (Intención del usuario)

Connect the company's own Culqi payment-gateway credentials so the online store can process card
payments under the company's account rather than a shared/test account.

### Where to find it (Dónde encontrarlo)

The **Store (Tienda)** tab, in the **Culqi Configuration (Configuración Culqi)** card. It is
divided into three sections: **Culqi Test (Culqi Pruebas)** with **Public Key (Llave Pública)** and
**Private Key (Llave Privada)**; **Culqi Live** with its own **Public Key** and **Private Key**; and
**RSA Encryption (Encriptación RSA)** with **RSA Key ID (ID de Llave RSA)** and **RSA Key (Llave
RSA)**. All six are saved by the **Save (Guardar)** button at the top right of the tab.

### Required information and prerequisites (Requisitos previos)

None of the six Culqi fields are required to save the page. The credentials themselves come from
the company's own Culqi dashboard.

### Business rules and rationale (Reglas y razón de negocio)

The six fields are stored together as one Culqi configuration on the company record. Saving from
the Store tab writes the **whole** company record — the same request the My Company tab sends — so
it also validates Name, RUC and Legal Name and refuses with "Faltan datos a guardar: …" if any of
those three is empty, even though none of them appear on this tab. Test and live credentials are
kept in two separate boxes, so a company can be configured against Culqi's test environment before
switching the store to live keys. Both **Private Key** fields are masked as password fields with a
reveal toggle; the two public keys and the RSA pair are shown in clear text.

### Result and side effects (Resultado y efectos)

A save republishes the company's public ecommerce file used by the online store, carrying the
company name plus the **Public Key (Test)**, **Culqi RSA Key** and **Culqi RSA Key ID**. The two
private keys and the live public key stay on the company record and are never written to that
public file.

### Limitations (Limitaciones)

A save writes the whole company record, so clearing a Culqi field on screen clears it in storage
too. The page performs no validation against Culqi: a wrong or expired key is accepted here and
only fails later, at checkout in the online store. Only the Public Key (Test) reaches the public
ecommerce file, so a store running on live keys still reads its public key from that test slot.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo configuro Culqi para cobrar en la tienda online?`
- `¿Cuál es la diferencia entre las llaves de pruebas y las llaves live?` Las de pruebas sirven
  para probar el checkout sin cobrar de verdad; las live cobran a la tarjeta del cliente.
- `¿Dónde configuro la tienda online?` En la pestaña **Tienda** de Configuración.
- Search terms: `Culqi`, `tienda`, `pasarela de pago`, `llave pública`, `llave privada`,
  `modo pruebas`, `modo live`, `RSA key`, `ecommerce`.

<!-- DOC-ID: capability.browse-backups -->
## Browse available backups (Ver los backups disponibles)

Open **Administration (Administración) → Configuration (Configuración)** at
`/company/configuration` and switch to the **Backups** tab. The left table lists up to the
**latest 30 backups** stored for the company — the listing request caps at 30 objects with no
further pages, so once more than 30 backups exist, older ones stop appearing here even though they
still exist in storage. Clicking a row selects it (click again to deselect) and populates the
**Restore** panel on the right; nothing else on the page changes from a selection alone.

Viewing this list requires no dedicated permission entry in the access catalog — any authenticated
user of the company who can reach the page can see it. Acting on **Generate** or **Restore**, by
contrast, always requires the "Backups" access at the **Full (Todo)** level, because that access
only offers **View (Visualizar)** and **Full (Todo)** — there is no separate Create/Edit
granularity for backups, so a user must have Full to run either write action.

<!-- DOC-ID: capability.generate-backup -->
## Generate a new backup (Generar un backup)

### User intention (Intención del usuario)

Create a fresh snapshot of the current operational data — for example before a risky bulk
change, a data migration, or as routine precaution.

### Where to find it (Dónde encontrarlo)

The green **+** button at the top right of the Backups toolbar, on the **Backups** tab. Confirm
with **Generate Backup|Generar Backup** — "Do you want to generate the backup now? / ¿Desea
generar el backup ahora?" — Yes/No.

### Required information and prerequisites (Requisitos previos)

None: the action takes no parameters from the user. The acting user needs the "Backups" access
at Full level (see above).

### Business rules and rationale (Reglas y razón de negocio)

On confirmation, the server exports every ScyllaDB-backed business table for the requesting
user's own company to CSV, zstd-compresses each CSV, and packs them into one tar file uploaded
to the company's backup storage. The company configuration edited on the other tab, and Security's
profiles/accesses, are not part of this export because they are stored outside this table set.

### Result and side effects (Resultado y efectos)

A new file appears in the company's backup storage and, after the tab reloads the list, at the
top of its table. Generating a backup is a read-only snapshot operation: it does not change any
existing data. Once more than 30 backups exist, the oldest one drops off this list (though it
remains in storage).

### Limitations (Limitaciones)

There is no scheduling here — backups are not created automatically on any interval; this button
is the only user-reachable way to produce one. A backup cannot be named, labeled, or annotated by
the user; it is only identified by its automatic file name and creation date.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo genero un backup manualmente?`
- `¿Los backups se generan automáticamente?` No, sólo al presionar "Generar" en esta pestaña.
- Search terms: `generar backup`, `respaldo manual`, `copia de seguridad`, `backup bajo demanda`.

<!-- DOC-ID: capability.download-backup -->
## Download a backup (Descargar un backup)

### User intention (Intención del usuario)

Retrieve the raw backup file — for example to store it outside Genix or hand it to support.

### Where to find it (Dónde encontrarlo)

On the **Backups** tab, select a row in the table; the right panel shows the selected backup's name
and size with a purple download button. Without a selection, the panel shows **Select a Backup /
Seleccione un Backup** instead.

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

On the **Backups** tab, select the backup in the table, then use the blue **Restore|Restaurar**
button in the right panel. Confirm with **Restore Backup|Restaurar Backup** — "Restore the backup
from `<date>` / Restaurar el backup realizado el `<date>`" — Yes/No.

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
company; tables not included in the backup, the company configuration on the My Company tab, and
Security profiles/access assignments are left untouched.

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

- The three tabs carry two different accesses: **My Company (Mi Empresa)** and **Store (Tienda)**
  both need the "Configuración" access — they edit the same record through the same endpoint —
  while **Backups** needs the "Backups" access. Reaching the page requires only one of the two, and
  the tab strip then shows only the tabs the user is allowed to read: a user holding just
  "Configuración" sees My Company and Store, and a user holding just "Backups" lands directly on
  Backups and never sees the other two.
- Both accesses only offer **View (Visualizar)** or **Full (Todo)** levels, so granting someone the
  ability to save company data — or to generate/restore a backup — always means granting full
  control of that tab, not a partial edit level.
- Reading the underlying company record is also allowed through the **Punto de Venta** (Point of
  Sale) access, without needing the Configuración access, because the POS screen reads this same
  record's RUC and legal name for the sale receipt header. Punto de Venta access alone does **not**
  grant permission to Save changes here.
- My Company and Store hold two Save buttons but write the same whole record in one request each:
  there is no way to save the company parameters without also writing the Culqi configuration, or
  the other way round. Both therefore enforce the same required fields (Name, RUC, Legal Name).
- Backup generation, listing, and restore all operate on the signed-in user's own company only,
  read from the session on the server — the one exception is the download link's known company-ID
  defect documented under **Download a backup**.
- Generate and Restore are the only two write actions on the Backups tab, and both require an
  explicit Yes/No confirmation dialog before running.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Missing required data: … (Faltan datos a guardar: …)":** the message lists the empty required
  fields by name — Name, RUC and/or Legal Name. Fill the ones it names and Save again. Email is not
  required. This can also appear when saving from the **Store (Tienda)** tab, because that Save
  writes the same whole company record; the fields it names are on the **My Company** tab.
- **A field shows empty after saving it:** a save writes the whole company record, so a field left
  blank on screen is stored blank. Check that the value was still in the field when Save was
  pressed.
- **"El user no posee alguno de los accesos: Configuración":** the acting user's profile(s) do not
  include the Configuración access; ask an administrator to grant it at the Full (Todo) level to
  allow saving.
- **A tab is missing:** the missing tab's access is not granted to the acting user's profile(s) —
  the Backups tab needs the "Backups" access, and My Company and Store both need the "Configuración"
  access, so those two always appear or disappear together.
- **The receipt on Punto de Venta shows the wrong RUC/razón social:** confirm the values saved
  here (Name, RUC, Legal Name persist correctly), since the sale screen reads this same record.
- **"Select a Backup / Seleccione un Backup"**: no row is selected yet; click one in the table
  first.
- **Download fails or opens the wrong file**: known defect — the download link currently points
  at company `1`'s storage folder regardless of the signed-in company; see **Download a
  backup**.
- **The Upload button does nothing**: the blue upload-icon button in the Backups toolbar has no
  action wired to it; uploading an external backup file is not implemented despite the button being
  visible.
- **An old backup is missing from the list**: only the latest 30 backups are returned; older
  ones still exist in storage but are not shown here.
- **Data still looks outdated right after a restore**: the app clears its local cache
  automatically, but a page already open in the browser may keep showing data it rendered before
  the restore — reload or re-navigate to it.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **System → Companies (Sistema → Empresas)** at `/system/companies` administers every tenant
  company registered on the SaaS platform and is restricted to the platform-administering
  company; it is a different page from this one and is not reachable from here.
- **Users & Profiles (Usuarios & Perfiles)** at `/security/users-profiles`: user accounts,
  profiles, and access assignments — including the two accesses that gate this page's tabs — are
  managed there, and are never included in a backup or affected by a restore performed here.
- **Where do I configure the email Genix sends?** Nowhere in the app: outgoing mail is a
  platform-level setting owned by whoever operates the Genix installation, not a per-company one.
- **Point of Sale (Punto de Venta)** at `/sales/sale_order_create` reads this same company record
  (RUC, razón social) to print the sale receipt header, and separately embeds its own **System
  Parameters** editor for POS-specific settings unrelated to this page.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:f60ca2c4d17f36984b5994230c5df7fbe056e5646d2f1817707ce6827e4cab25
    supports: [page-purpose, capability.edit-company-data, capability.browse-backups, related-pages]
  - path: frontend/routes/company/configuration/+page.svelte
    role: page
    hash: sha256:245a3c35c96126eb7869b9014d5ee2c1fe349df94cd5f8ca148cf98a9e96f537
    supports: [page-purpose, capability.edit-company-data, capability.browse-backups, rules, troubleshooting]
  - path: frontend/routes/company/configuration/CompanyTab.svelte
    role: user-interface
    hash: sha256:3185971c0b916d534a918776cb6cc3f58c256821de7b76acd2ce6722319ecd6f
    supports: [page-purpose, concepts, capability.edit-company-data, troubleshooting]
  - path: frontend/routes/company/configuration/StoreTab.svelte
    role: user-interface
    hash: sha256:8384d4c7ed96fd654d93fd8f39a73bd9ad00d2aae251d184e7708ca2b2f6eba2
    supports: [page-purpose, concepts, capability.configure-culqui, rules, troubleshooting]
  - path: frontend/routes/company/configuration/BackupsTab.svelte
    role: user-interface
    hash: sha256:157677f6f1c549726a33c4f8c88d33b2ede62585f8d73e07045e9f7ad8a2ff20
    supports: [page-purpose, concepts, capability.browse-backups, capability.generate-backup, capability.download-backup, capability.restore-backup, rules, troubleshooting]
  - path: frontend/routes/company/configuration/empresas.svelte.ts
    role: frontend-service
    hash: sha256:bcd8e9e49f13a27253a101d56e4f550168e4c85e67dd8a65c58fce46eef01596
    supports: [concepts, capability.edit-company-data, capability.configure-culqui, rules]
  - path: frontend/routes/company/configuration/backups.svelte.ts
    role: frontend-service
    hash: sha256:5864d664ed51594e43a42b35f77c917361d83d0fc1aa6c553b33ac20df714027
    supports: [concepts, capability.browse-backups, capability.generate-backup, capability.restore-backup, rules]
  - path: frontend/core/env.ts
    role: frontend-service
    hash: sha256:617de03880bc13ffe209c46f03750cbd8162c18f0d05a4e2b43acf979fbe8e7c
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
    hash: sha256:b02edda57a5a52cfd466d43ab60fca25bca39188b30e83e7b0b61fdc9d4feb96
    supports: [capability.restore-backup]
  - path: frontend/packages/genix-ui/buttons/Button.svelte
    role: user-interface
    hash: sha256:259cade6a388d1d274e343996bc97fdb73d0a332801529d364434b9b77ee33af
    supports: [troubleshooting]
  - path: frontend/packages/genix-ui/navigation/OptionsStrip.svelte
    role: user-interface
    hash: sha256:f69bb747e70bf25e27eb18d1298482bc89a5e78ebde01fba3ed362bb1cb0c6a0
    supports: [page-purpose, rules]
  - path: backend/config/empresas.go
    role: backend-handler
    hash: sha256:d4c5e8f6e6b247497423e07247d326252913709c3b0cb0a362f02ce3e7488752
    supports: [page-purpose, capability.edit-company-data, capability.configure-culqui, rules, troubleshooting, related-pages]
  - path: backend/config/types/empresas.go
    role: data-model
    hash: sha256:bd34d84da599878932680347e22f858cf915afcb0c1e1fdd890a5fa42534ff1f
    supports: [concepts, capability.edit-company-data, capability.configure-culqui, rules]
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
    hash: sha256:00805be381d64678157e84008762dd97e9978b2e0a0f098f82e42a3dcd6ce712
    supports: [page-purpose, capability.generate-backup]
  - path: backend/genix-orm/scylla/deploy.go
    role: business-logic
    hash: sha256:bddb0793b217cc905b2a852b2fe42a26f41f0b1d7026fcf401d84d2b3bbdecef
    supports: [concepts, capability.restore-backup]
  - path: backend/access.toml
    role: permissions
    hash: sha256:ec1a17b2bd06f28bd9749d1dc8097171986737abe9dba06822452a529ab7ada7
    supports: [rules, capability.browse-backups, capability.generate-backup, capability.restore-backup, troubleshooting]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:0e4a825ccd2fe08e6a586cfcd24406b715f5e548c9e71d93091421a2d39e8956
    supports: [rules, capability.browse-backups, capability.generate-backup, capability.restore-backup, troubleshooting]
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: user-interface
    hash: sha256:7f91bd3b80638f70432d1bc9d8db4f5c3812ac709af4c61fba2a4b44cef663d4
    supports: [concepts, rules, related-pages]
```
