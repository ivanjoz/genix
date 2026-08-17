---
schema: 1
page_id: business.suppliers
route: /business/suppliers
title: Suppliers (Proveedores)
status: implemented
visibility: tenant
description_en: >-
  Supplier (proveedor) management. Create and edit each supplier's name, person type, registry
  number (RUC/DNI), email, and department/province/district location; search the existing list by
  name, registry number, or email.
description_es: >-
  Gestión de proveedores. Crear y editar el nombre, tipo de persona, número de registro (RUC/DNI),
  correo y ubicación (departamento/provincia/distrito) de cada proveedor; buscar la lista por
  nombre, número de registro o correo.
---

# Suppliers (Proveedores)

<!-- DOC-ID: page-purpose -->
## Page purpose

Suppliers (`Proveedores`) is the page where Genix keeps the directory of companies or
individuals that supply products to the business (`proveedores` — as opposed to `clientes`, the
people or companies that buy from it). It creates and edits each supplier's name, person type,
registry number (`RUC`/`DNI`), email, and geographic location, and it is the only place a
supplier can be created or edited before it becomes selectable elsewhere.

This page does not manage purchase orders, payments, or stock coming from a supplier; those
workflows live in Logistics (**Purchase Orders / Órdenes de Compra** and related pages) and only
pick an existing supplier from the list created here. It also does not offer a phone or a
free-text street-address field — despite the module's older internal notes, this page has no
`Phone`/`Teléfono` input and no free-text `Address`/`Dirección` input; location is limited to a
Department/Province/District picker.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Suppliers (Proveedores)** and **Customers (Clientes)** are two views over the same
  underlying record type, distinguished only by a `Type` value (`1` = client, `2` = provider).
  Both pages reuse the identical form and table component; only the page title, the
  create/edit labels, and the fixed `Type` sent to the server differ.
- A **person type (`Tipo Persona`)** marks a supplier as **Person (Persona)** — a natural
  person/individual supplier — or **Company (Empresa)** — a legal entity.
- The **Registry Number (RUC)** field stores the supplier's national identification/tax number
  (a company's `RUC` or, informally, an individual's `DNI`). The field label always reads
  "Registry Number (RUC)" whether the record is a Person or a Company.
- The **Location (Ubicación)** shown in the list combines the province and district picked
  during creation/edit; the underlying record only stores a district-level `CityID` and a fixed
  `CountryID`, not a full street address.
- Genix deduplicates client/provider records company-wide by the combination of registry number
  and normalized name, **without separating clients from providers** — see the cross-capability
  rule below, because saving a supplier that matches an existing client (or vice versa) updates
  that same underlying record instead of creating a second one.

<!-- DOC-ID: capability.browse -->
## Find a supplier (Buscar un proveedor)

Open **Business (Negocio) → Suppliers (Proveedores)** at `/business/suppliers`. The list shows
every active supplier (records with a stored status greater than 0) with ID, Name, Person Type,
Registry/RUC number, Email, Location (province and district), and the last-updated date/time.
Use the search box (`Search by name, email or registry`) to filter; the filter matches the ID,
Name, Registry Number, or Email of the already-loaded list locally in the browser — it is not a
server-side search, so a supplier outside the currently synced set will not appear until it is
fetched. Clicking a row opens it in the same side layer used to create a new supplier, pre-filled
for editing.

<!-- DOC-ID: capability.create-edit -->
## Create or edit a supplier (Crear o editar un proveedor)

### User intention (Intención del usuario)

Register a new supplier before using it in a purchase workflow, or update an existing supplier's
name, person type, registry number, email, or location.

### Where to find it (Dónde encontrarlo)

On `/business/suppliers`, use the green **New (Nuevo)** button (top right of the list toolbar) to
open the side layer for a new supplier, or click an existing row to open the same layer
pre-filled. Save with the layer's **Save (Guardar)** action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required.
- **Person Type (Tipo Persona)** defaults to Person (Persona) for a new supplier; choose Company
  (Empresa) for a legal entity.
- **Email** is required by this page's form and must contain an `@`; the field is otherwise
  free text (no full RFC email-format check happens in the browser).
- **Department | Province | District (Departamento | Provincia | Distrito)** is required; Genix
  fixes the country to Peru (`CountryID = 604`) automatically whenever a district is picked —
  there is no visible country selector, so this page currently only supports Peruvian locations.
- **Registry Number (RUC)** is only marked required by this page's form when Person Type is
  Company, and the browser then requires it to be 7–12 digits.

### Business rules and rationale (Reglas y razón de negocio)

The server enforces a stricter rule than the visible form: **every** supplier record — Person or
Company — must have a Registry Number of 7–12 digits, plus a valid Department/Province/District
and a positive country. The form's "required" marker and its own validation only apply that
7–12 digit check when Person Type is Company, so creating an individual/Person supplier while
leaving Registry Number empty passes the visible form but is rejected by the server with
"El registro en posición 0 debe tener RegistryNumber numérico de 7 a 12 dígitos para
proveedores." — an individual supplier still needs a DNI/RUC-style number entered in that field.
Customer (`Cliente`) records saved from the sibling page are not held to that same registry-number
requirement, because the server only applies it when `Type` is Provider.

Name, Email, and Registry Number are trimmed (and Email lowercased) before saving. Saving always
sends the fixed `Type = 2` (Provider) for every record created or edited from this page, so a
supplier can never accidentally be stored as a customer from here.

### Result and side effects (Resultado y efectos)

Saving creates a new supplier record or updates the matched one, and the updated/created record
is merged back into the on-screen list immediately without a full page reload. A brand-new
supplier is stored with status `1` (active); this page exposes no field or action to change that
status afterward.

### Limitations (Limitaciones)

- There is no delete or deactivate action on this page — the side layer used here does not offer
  a **Delete (Eliminar)** button (unlike some other record pages), so once created a supplier
  cannot be removed or deactivated from `/business/suppliers`.
- There is no phone field and no free-text street address field; location is limited to
  Department/Province/District, and the country is always Peru.
- The Registry Number field always shows the English label "Registry Number (RUC)" even in the
  Spanish interface, and is used for both a Company's RUC and an individual's DNI-style number.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un proveedor nuevo?`
- `¿Por qué me pide un RUC/DNI si el proveedor es una persona natural?` Aunque el formulario sólo
  marca el Registry Number como obligatorio para Empresa, el servidor exige ese número (7 a 12
  dígitos) para todo proveedor, sea Persona o Empresa.
- `¿Cómo elimino o doy de baja un proveedor?` Actualmente esta página no ofrece esa acción.
- `¿Puedo registrar un proveedor de otro país?` No; la ubicación siempre queda fijada a Perú.
- Search terms: `proveedor`, `crear proveedor`, `editar proveedor`, `RUC`, `DNI`, `número de
  registro`, `tipo de persona`, `sede`, `ubicación`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Suppliers and customers share one save endpoint and one underlying table; Genix matches an
  incoming record to an existing one first by Registry Number and, if that is empty, by a hash
  of the normalized name plus Registry Number — **this lookup is not restricted by Type**. If a
  supplier is saved with the same Registry Number as an existing customer (or vice versa),
  Genix updates that same record and its `Type` changes instead of creating a second, separate
  record for the other role.
- Saving through this page requires the "Proveedores" access at the Full (Todo) level; that
  catalog access only offers View or Full (no intermediate Create/Edit levels), so granting
  someone the ability to save suppliers always grants full control of this page. Viewing the
  supplier list itself is open to any authenticated user of the company, since no read-only
  access is mapped for it in the catalog. The same backend save route is also reachable through
  the separate "Clientes" access, so either access independently authorizes a call to it; the
  page itself always fixes which `Type` is actually sent.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Please enter the name of the supplier." / "Debe ingresar el nombre del proveedor.":**
  the Name field was left empty.
- **"Please enter a valid email." / "Debe ingresar un email válido.":** Email was empty or did
  not contain `@`.
- **"Please enter a valid City." / "Debe ingresar un CityID válido.":** no district was
  selected in the Department/Province/District picker.
- **"El registro en posición 0 debe tener RegistryNumber numérico de 7 a 12 dígitos para
  proveedores." after saving:** occurs even for Person-type suppliers; enter a 7–12 digit
  Registry Number (RUC or DNI) before saving, not only for Company-type suppliers.
- **A supplier appears to overwrite an existing customer, or vice versa:** the two pages share
  records keyed by Registry Number; verify the Registry Number entered does not already belong
  to a record of the other type before saving.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Customers (Clientes)** at `/business/customers` uses this same form and list component with
  `Type = 1`; use it for records that buy from the business instead of supplying it.
- **Purchase Orders (Órdenes de Compra)** and other Logistics purchasing pages (Supplies &
  Materials, Purchase Order Entry, Product Supply Management) select the supplier for a purchase
  from the list created here; none of them can create a new supplier inline, so a supplier must
  be registered on this page first.
- **Sites & Warehouses (Sedes & Almacenes)** manages the branch/location catalog used elsewhere
  in Business, but the Department/Province/District picker on this page comes from the shared
  country-cities service, not from that page's branch list.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.browse, related-pages]
  - path: frontend/routes/business/suppliers/+page.svelte
    role: page
    hash: sha256:58e322b1569671bdf54af0aa42ebf84f9514e47c23bd0bc8c6f377e59e6f6556
    supports: [page-purpose, concepts, capability.create-edit, rules]
  - path: frontend/routes/business/customers/CustomersView.svelte
    role: user-interface
    hash: sha256:bf3182c34a440abdc24136d753a4c779c9bf6a732e6933ffeb063db13b30d351
    supports: [page-purpose, concepts, capability.browse, capability.create-edit, troubleshooting, related-pages]
  - path: frontend/routes/business/customers/customers.svelte.ts
    role: frontend-service
    hash: sha256:d42d73f9ef8b3ecd5e7fec9e83cbb2e5cddf8d276c4ac6c37c6babdb1d5081d3
    supports: [concepts, capability.browse, capability.create-edit, rules]
  - path: frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts
    role: shared-domain
    hash: sha256:8f3a3fdc8dc47344ada0fc5f56361026effe8099d44c68c9a6ef54a68e488302
    supports: [concepts, capability.create-edit, related-pages]
  - path: frontend/packages/genix-ui/layers/Layer.svelte
    role: user-interface
    hash: sha256:85eb394590a6d1954904e278e58a0f53d4222e0ec674111a183efb8ce41e8a49
    supports: [capability.create-edit]
  - path: frontend/routes/logistics/purchase-orders/PurchaseOrderCreate.svelte
    role: user-interface
    hash: sha256:1be98d963afdb8e485ea32615c307e730683ad2d483dd148e09580ede89315cf
    supports: [related-pages]
  - path: backend/business/client_provider.go
    role: backend-handler
    hash: sha256:c7ecb92cebcdf753b5a7f27110eb7ab5c941e65709f59db146c529063e434938
    supports: [capability.browse, capability.create-edit, rules, troubleshooting]
  - path: backend/business/types/client_provider.go
    role: data-model
    hash: sha256:83d725eaeac0c081e5b5871f1f015a171981bab763d7b2e1a700ddfdc22ce748
    supports: [concepts, capability.create-edit, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
```
