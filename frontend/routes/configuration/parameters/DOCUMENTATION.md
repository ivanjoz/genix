---
schema: 1
page_id: configuration.parameters
route: /configuration/parameters
title: My Company (Mi Empresa)
status: implemented
visibility: tenant
description_en: >-
  Tenant company settings. Edit the company's name, tax ID, legal name, email, phone, address,
  city, and representative; configure SMTP credentials for outgoing notification email; and enter
  Culqi payment-gateway keys (test and live) for ecommerce.
description_es: >-
  Configuración de la empresa del tenant. Editar nombre, RUC, razón social, correo, teléfono,
  dirección, ciudad y representante; configurar credenciales SMTP para el correo de
  notificaciones; e ingresar llaves de la pasarela de pago Culqi (pruebas y live) para ecommerce.
---

# My Company (Mi Empresa)

<!-- DOC-ID: page-purpose -->
## Page purpose

My Company (`Mi Empresa`) is the single-record settings page for the company (`empresa`) that
owns the current tenant session. The page heading reads "Company Parameters" (`Parámetros de la
Empresa"); it edits one company record: identity data (name, tax ID/RUC, legal name/razón social,
email, phone, address, city, representative), the SMTP credentials used for notification email,
and the Culqi payment-gateway keys used for ecommerce checkout.

This page is not the SaaS-wide **Companies (Empresas)** admin screen at `/system/companies`,
which lists every tenant registered on the platform and is restricted to the company that
administers the SaaS itself. My Company only ever reads and writes the record belonging to the
signed-in user's own company (`req.User.CompanyID`); it has no picker and cannot open another
tenant's record.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- The **company (empresa)** record is the same record used across the tenant: its RUC and
  `LegalName`/razón social also feed the sales receipt/voucher (`comprobante`) header on the
  Point of Sale (`Punto de Venta`) screen, which is why POS can read this data even without the
  Mi Empresa access (see permissions below).
- **Ecommerce (Culqi)** keys configure the Culqi payment gateway used by the online store: a test
  mode pair (`Llave Pública/Privada (Pruebas)`) and a live mode pair (`Llave Pública/Privada
  (Live)`), plus an RSA key/RSA key ID pair used for Culqi's 3-D Secure/antifraud flow.
- **SMTP parameters** are the outgoing mail server credentials (`Host`, `Port`, `Username`,
  `Password`, `Email`) Genix would use to send notification email on the company's behalf.

<!-- DOC-ID: capability.edit-company-data -->
## Edit company data (Editar los datos de la empresa)

### User intention (Intención del usuario)

Keep the tenant's own legal/operational identity (name, tax ID, legal name, contact data) current,
since this is the same data used on sales receipts and in company-wide notifications.

### Where to find it (Dónde encontrarlo)

Open **Administration (Administración) → Configuration (Configuración) → My Company (Mi
Empresa)** at `/configuration/parameters`. Edit the fields in the left form section directly (no
separate edit mode) and use the top-right **Save (Guardar)** button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)**, **RUC**, **Legal Name (Razón Social)**, and **Email** are marked required on
  the form; the page blocks Save locally with "Missing required data to save. (Faltan datos a
  guardar.)" when any of the four is empty.
- **Phone (Teléfono)**, **Representative (Representante)**, **Legal Address (Dirección Legal)**,
  and **City (Ciudad)** are optional operational fields.
- The server independently rejects a save whose Name, RUC, Legal Name, or Email is empty with
  "Falta alguno de los siguiente parámetros: Nombre, Razon-Social, RUC, Email." — but, unlike the
  SaaS **Companies** page, it does not enforce any minimum character length on these values here.

### Business rules and rationale (Reglas y razón de negocio)

Saving always targets the signed-in user's own company ID; the ID present in the record on screen
is not sent as a separate selector. The server also guarantees the company has an internal form
API key, generating a random one only the first time it is missing, so a save never revokes an
existing key even though this page has no field to view or regenerate it.

### Result and side effects (Resultado y efectos)

A successful save updates the single company record and republishes a small public JSON file used
by the online store (name and Culqi identifiers); see the Limitations note below on which values
actually reach that file. It does not create any other record, and the same updated company feeds
the POS receipt header the next time a sale is created.

### Limitations (Limitaciones)

**Phone, Representative, Legal Address, and City do not currently save or reload.** The form binds
these four fields under Spanish property names, but the stored company record expects their
English equivalents; the mismatch means the value you type into any of these four fields is
silently dropped on save, and the fields always reload empty even after a successful save,
regardless of what was entered. Only **Name**, **RUC**, **Legal Name**, and **Email** reliably
persist from the general data section.

There is no visible field for the company's `FormApiKey`, and this page cannot regenerate it.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo edito el RUC o la razón social de mi empresa?`
- `¿Por qué el teléfono, la dirección o el representante no se guardan?` Esos cuatro campos
  actualmente no persisten al guardar, aunque el formulario los muestre editables.
- `¿Dónde configuro los datos de mi empresa (no de otra empresa del sistema)?` Aquí, en Mi
  Empresa; la administración de todas las empresas del SaaS está en otra pantalla reservada al
  administrador de la plataforma.
- Search terms: `empresa`, `RUC`, `razón social`, `mi empresa`, `datos de la empresa`, `teléfono`,
  `dirección`, `representante`.

<!-- DOC-ID: capability.configure-smtp -->
## Configure SMTP for notifications (Configurar SMTP para notificaciones)

### User intention (Intención del usuario)

Point Genix at the company's own outgoing mail server so notification email can be sent using the
company's identity instead of a shared default.

### Where to find it (Dónde encontrarlo)

The **SMTP parameters for notifications** card on the right side of the page: **Host**, **Port**,
**Username (Usuario)**, **Password**, and **Email**. Saved together with the rest of the page
through the same **Save (Guardar)** action.

### Required information and prerequisites (Requisitos previos)

All five SMTP fields are optional on this page; nothing blocks Save if they are left empty.

### Business rules and rationale (Reglas y razón de negocio)

**Host, Username, Password, and Email save and reload correctly. Port does not**: the stored
record's field for the SMTP port does not match the name this form saves under, so a Port value
typed here is silently dropped on save and the field always reloads empty.

### Limitations (Limitaciones)

Because Port never actually persists, an SMTP configuration saved from this page is only usable
as long as the intended mail provider's port can be assumed or configured elsewhere; this page
cannot currently be used to record a working port value.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo configuro el correo (SMTP) para las notificaciones?`
- `¿Por qué el puerto SMTP no se guarda?` Es un problema conocido de esta página: el puerto no
  persiste aunque el resto de los datos SMTP sí.
- Search terms: `SMTP`, `correo saliente`, `notificaciones por correo`, `host`, `puerto`, `usuario
  SMTP`.

<!-- DOC-ID: capability.configure-culqui -->
## Configure Culqi ecommerce keys (Configurar llaves de Culqi para ecommerce)

### User intention (Intención del usuario)

Connect the company's own Culqi payment-gateway credentials so the online store can process card
payments under the company's account rather than a shared/test account.

### Where to find it (Dónde encontrarlo)

The **Ecommerce** section of the form: **Public Key (Test)**, **Private Key (Test)**, **Public Key
(Live)**, **Private Key (Live)**, **Culqi RSA Key ID**, and **Culqi RSA Key**. Saved together with
the rest of the page through **Save (Guardar)**.

### Required information and prerequisites (Requisitos previos)

None of the six Culqi fields are required to save the page.

### Business rules and rationale (Reglas y razón de negocio)

**Only Culqi RSA Key and Culqi RSA Key ID save and reload correctly.** The other four fields
—Public Key (Test), Private Key (Test), Public Key (Live), and Private Key (Live)— use property
names on this form that do not match the stored record's field names, so any value entered in
those four fields is silently dropped on save and they always reload empty.

### Result and side effects (Resultado y efectos)

A save republishes the company's public ecommerce file (used by the online store) with the
company name and the RSA Key/RSA Key ID values. Because the Public Key (Test) field never actually
persists (see above), the public key portion of that published file is currently always blank, and
the Live-mode keys never reach any stored configuration through this page.

### Limitations (Limitaciones)

Treat the four Public/Private Key (Test/Live) fields as **not functional** on this page today: do
not rely on them to configure a working Culqi checkout. Only the RSA Key and RSA Key ID can
currently be set here.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo configuro Culqi para cobrar en la tienda online?`
- `¿Por qué mis llaves públicas/privadas de Culqi no se guardan?` Es una limitación actual de esta
  página: sólo el RSA Key y el RSA Key ID persisten.
- Search terms: `Culqi`, `pasarela de pago`, `llave pública`, `llave privada`, `modo pruebas`,
  `modo live`, `RSA key`, `ecommerce`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Saving this page (any of its three field groups) requires the "Mi Empresa" access; that access
  only offers **View (Visualizar)** or **Full (Todo)** levels, so granting someone the ability to
  save here always means granting full control of the page, not a partial edit level.
- Reading the underlying company record is also allowed through the **Punto de Venta** (Point of
  Sale) access, without needing Mi Empresa, because the POS screen reads this same record's RUC
  and legal name for the sale receipt header. Punto de Venta access alone does **not** grant
  permission to Save changes on this page.
- All three field groups (general data, SMTP, Culqi keys) share the same field-name mismatch
  pattern between the form and the stored record: fields whose form property name differs from the
  stored record's field name do not persist. This currently affects Phone, Representative, Legal
  Address, City, SMTP Port, and four of the six Culqi keys.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Missing required data to save. (Faltan datos a guardar.)":** Name, RUC, Legal Name, or Email
  is empty; all four are required before Save is allowed.
- **A field always shows empty after saving it:** Phone, Representative, Legal Address, City, SMTP
  Port, and the four Public/Private Culqi keys (Test/Live) do not currently persist; re-typing
  them will not fix this, since the value is dropped on save, not on reload.
- **"El user no posee alguno de los accesos: Mi Empresa":** the acting user's profile(s) do not
  include the Mi Empresa access; ask an administrator to grant it at the Full (Todo) level to
  allow saving.
- **The receipt on Punto de Venta shows the wrong RUC/razón social:** confirm the values saved
  here (Name, RUC, Legal Name persist correctly), since the sale screen reads this same record.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **System → Companies (Sistema → Empresas)** at `/system/companies` administers every tenant
  company registered on the SaaS platform and is restricted to the platform-administering
  company; it is a different page from this one and is not reachable from here.
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
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/routes/configuration/parameters/+page.svelte
    role: page
    hash: sha256:42bb0cf6813b363122f6ff566bf34f34bb8656219b23362a2bf54c9ca334e5a3
    supports: [page-purpose, concepts, capability.edit-company-data, capability.configure-smtp, capability.configure-culqui, troubleshooting]
  - path: frontend/routes/configuration/parameters/empresas.svelte.ts
    role: frontend-service
    hash: sha256:a226c50d7dff27ae5cf1cc5769c45cdcb491ea5dcfadd353831f30e06b34a122
    supports: [concepts, capability.edit-company-data, capability.configure-smtp, capability.configure-culqui, rules]
  - path: backend/config/empresas.go
    role: backend-handler
    hash: sha256:d4c5e8f6e6b247497423e07247d326252913709c3b0cb0a362f02ce3e7488752
    supports: [page-purpose, capability.edit-company-data, capability.configure-smtp, capability.configure-culqui, rules, troubleshooting, related-pages]
  - path: backend/config/types/empresas.go
    role: data-model
    hash: sha256:c751f3a161094d85cb735e64eadeb9f4f2f243b8391449675c1bb1dfeaf2de25
    supports: [concepts, capability.edit-company-data, capability.configure-smtp, capability.configure-culqui, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules, troubleshooting]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules, troubleshooting]
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: user-interface
    hash: sha256:c8650707d5e88cc6cfe386a6b6c240225b8c5ba153f63f4b00be04c92f4cbf65
    supports: [concepts, rules, related-pages]
```
