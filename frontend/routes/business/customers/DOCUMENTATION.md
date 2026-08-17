---
schema: 1
page_id: business.customers
route: /business/customers
title: Customers (Clientes)
status: implemented
visibility: tenant
description_en: >-
  Customer (cliente) management. Create and edit customers with name, person type
  (individual/company), registry number (RUC/DNI), email, and a Peru department/province/district
  location. Search the list by name, email, or registry number.
description_es: >-
  Gestión de clientes. Crear y editar clientes con nombre, tipo de persona (natural/empresa),
  número de registro (RUC/DNI), correo y una ubicación de departamento/provincia/distrito del
  Perú. Buscar la lista por nombre, correo o número de registro.
---

# Customers (Clientes)

<!-- DOC-ID: page-purpose -->
## Page purpose

Customers (`Clientes`) is the administrative page for the buyer records (`clientes`) a company
sells to. It creates and edits each customer's name, person type (natural person or company),
registry number (`RUC`/`DNI`), email, and a Peru department/province/district location, and lists
every active customer with a local search box.

This page does not manage sales, invoices, or debt for a customer; it only owns the customer's
identity data. Choosing a customer for a transaction happens on the **Sales Order (Orden de
Venta)** creation screen and related sales pages, which read this same customer list. This page
also shares its entire form and table implementation with **Suppliers (Proveedores)**
(`/business/suppliers`) — both routes render the identical `CustomersView`/`ClientesProveedoresView`
component and save through the same backend record type, distinguished only by a fixed `Type`
value (1 = cliente, 2 = proveedor) that each route passes in.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **customer (cliente)** and a **supplier (proveedor)** are stored as the same underlying
  `client_provider` record; only the `Type` field (1 = cliente, 2 = proveedor) tells them apart.
  This page always saves and lists `Type = 1` records; it never mixes in supplier rows.
- **Person type (Tipo Persona)** is `Person|Persona` (natural person) or `Company|Empresa`
  (legal entity). It decides whether a registry number is required on this page and, together
  with `Type`, how the backend enforces the registry-number format.
- **Registry Number (RUC)** is the customer's tax/registration identifier. Genix uses it (plus
  the normalized name) to detect an already-existing customer when saving, so it doubles as a
  deduplication key, not only a display field.
- **Location (Ubicación)** on this page means picking a Peru district (`distrito`) from a
  department ► province ► district catalog; it is not a free-text street address. Selecting a
  district also fixes the record's country to Peru.

<!-- DOC-ID: capability.browse -->
## Find a customer (Buscar un cliente)

Open **Business (Negocio) → Customers (Clientes)** at `/business/customers`. The table lists
every active customer (`ss > 0`) with ID, Name, Person Type, Registry/RUC, Email, Location
(province and district), and the last-updated date/time. Use the search box (placeholder "Search
by name, email or registry|Buscar por nombre, email o registro") to filter locally against the
already-loaded list; it matches Name, Registry Number, Email, or the numeric ID, and does not
query the server. Clicking a row opens it in the edit layer described below.

<!-- DOC-ID: capability.create-edit -->
## Create or edit a customer (Crear o editar un cliente)

### User intention (Intención del usuario)

Register a new customer before using them in a sale, or update an existing customer's name,
person type, registry number, email, or location.

### Where to find it (Dónde encontrarlo)

On `/business/customers`, use the green **New|Nuevo** button to open the side layer for a new
customer, or click an existing row to open the same layer pre-filled for editing. Save with the
layer's Save action; the layer title switches between "New Client|Nuevo Cliente" and "Edit
Client|Editar Cliente" depending on whether a record is selected.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** is required.
- **Email** is required by this page's form and must contain "@"; note this is stricter than the
  backend, which treats `Email` as optional and only rejects it when a non-empty value fails
  address-format validation.
- **Person Type (Tipo Persona)** defaults to `Person|Persona`.
- **Registry Number (RUC)** is required only when Person Type is `Company|Empresa`, and the
  frontend then requires 7–12 digits.
- **Department | Province | District** (the location selector) is required by this page's form
  for every customer; selecting a district also sets the country to Peru (`CountryID = 604`).
  The district catalog itself only ever lists Peru locations, so this page cannot register a
  customer outside Peru.

### Business rules and rationale (Reglas y razón de negocio)

Name, Email, and Registry Number are trimmed (Email is also lowercased) before saving. Saving a
customer whose normalized name plus registry number matches an existing customer updates that
existing record instead of creating a duplicate — the server resolves the record's ID by
`RegistryNumber` first, then by a name+registry hash, before merging, with no on-screen warning
that an existing customer was matched.

The backend's own 7–12 digit registry-number format check only fires for `Type = Proveedor`
records; for customers (`Type = Cliente`) that check is skipped entirely; the requirement seen on
this page for company customers is enforced by the frontend only. Likewise, the backend's
required-`CountryID`/required-`CityID` checks only apply to `Type = Proveedor`; this page still
asks every customer for a district because its own form marks the location as required.

Saving here requires the acting user to hold the **Full (Todo)** level of either the "Clientes"
or the "Proveedores" catalog access — both accesses map to the same `POST.client-provider`
endpoint, and the endpoint itself does not check which access was used, only that one of the two
was granted at Full. Viewing this page's list has no mapped read access, so any authenticated
user of the company can browse it.

### Result and side effects (Resultado y efectos)

Saving creates or updates the customer record, refreshing this page's cached list (and the
Suppliers page's cache, since both share the same delta-cache refresh). A new customer's
creation date/user are stamped by the server and its status is set active; editing preserves
those original creation fields.

### Limitations (Limitaciones)

- There is no phone field and no free-text street address on this page — despite older internal
  notes describing a phone/address field, the only location data stored is the Peru department
  /province/district selection plus the fixed country.
- There is no delete, deactivate, or archive action anywhere on this page; once created, a
  customer stays in the active list indefinitely from this screen.
- The location selector's label renders as just "Department" (English) or "Province" (Spanish)
  instead of the full "Department | Province | District" phrase, because the label text itself
  contains extra "|" separators that the bilingual text splitter consumes; the selector still
  works correctly and lists department ► province ► district, only the visible caption is
  shortened.
- Customers cannot be registered outside Peru from this page; the district catalog is fixed to
  Peru regardless of the `CountryID` value stored on the record.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un cliente nuevo?`
- `¿Por qué me pide el RUC/documento sólo quiero registrar una persona?` Sólo se exige el
  Registry Number cuando el Tipo Persona es Empresa.
- `¿Puedo registrar un cliente de otro país?` No desde esta página: el selector de ubicación
  sólo ofrece distritos del Perú.
- `¿Cómo busco un cliente por su RUC o nombre?` Usa el buscador sobre la tabla; filtra por
  nombre, email, RUC o ID ya cargados en pantalla.
- `¿Por qué se actualizó un cliente en vez de crearse uno nuevo?` Porque el nombre y el
  RegistryNumber ya coincidían con un cliente existente.
- Search terms: `cliente`, `crear cliente`, `editar cliente`, `RUC`, `DNI`, `tipo persona`,
  `natural`, `empresa`, `distrito`, `ubicación`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Customers and suppliers are the same entity family and the same save endpoint
  (`POST.client-provider`); only the fixed `Type` each route passes keeps the two lists apart in
  the UI. A profile with Full access to either the "Clientes" or "Proveedores" catalog entry can
  save through this endpoint regardless of which record type is being saved.
- Several backend validations (registry-number format, required country, required city) are
  conditioned on `Type = Proveedor` only; on this Customers page those same checks are enforced
  exclusively by the frontend form, not re-verified by the server for `Type = Cliente` records.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Please enter the name of the client.|Debe ingresar el nombre del cliente."**: the Name field
  is empty.
- **"Please enter a valid email.|Debe ingresar un email válido."**: Email is empty or missing
  "@"; this page always requires an email even though the server itself treats it as optional.
- **"For companies, Registry Number must be 7–12 digits.|Para empresa, el RegistryNumber debe
  tener entre 7 y 12 dígitos."**: occurs when Person Type is Company and Registry Number is
  missing or not 7–12 digits.
- **"Please enter a valid City.|Debe ingresar un CityID válido."** / **"...valid Country...
  CountryID válido."**: no district was selected in the location selector.
- **A saved customer looks like it "merged" into an older row instead of creating a new one**:
  its normalized name and registry number matched an existing customer; Genix updated that
  record instead of creating a duplicate.
- **The location field shows only "Department" or "Province" instead of the full three-level
  label**: this is the known label-splitting display issue described above; the underlying
  department/province/district selector still functions.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Suppliers (Proveedores)** at `/business/suppliers`: the same form and table, fixed to
  `Type = Proveedor` instead of `Type = Cliente`; use it to manage vendors instead of buyers.
- **Sales Order creation (Orden de Venta)** under Sales/Commercial reads this same customer list
  (`ClientProviderService` with `Type = Cliente`) to let a user pick or search a customer by name
  or document when starting a sale; create the customer here first if it is missing there.
- **Sites & Warehouses (Sedes & Almacenes)** at `/business/branches-warehouses` is where the
  shared Peru department/province/district catalog used by this page's location selector is also
  consumed for branch addresses, but this page does not manage branches itself.

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
  - path: frontend/routes/business/customers/+page.svelte
    role: page
    hash: sha256:419beda428b9235c11d648b2f28850f80de9144fe7f331f49bf91d84deb2c0cc
    supports: [page-purpose, concepts]
  - path: frontend/routes/business/customers/CustomersView.svelte
    role: user-interface
    hash: sha256:bf3182c34a440abdc24136d753a4c779c9bf6a732e6933ffeb063db13b30d351
    supports: [page-purpose, concepts, capability.browse, capability.create-edit, rules, troubleshooting]
  - path: frontend/routes/business/customers/customers.svelte.ts
    role: frontend-service
    hash: sha256:d42d73f9ef8b3ecd5e7fec9e83cbb2e5cddf8d276c4ac6c37c6babdb1d5081d3
    supports: [concepts, capability.browse, capability.create-edit, rules]
  - path: frontend/routes/business/suppliers/+page.svelte
    role: page
    hash: sha256:58e322b1569671bdf54af0aa42ebf84f9514e47c23bd0bc8c6f377e59e6f6556
    supports: [page-purpose, related-pages]
  - path: frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts
    role: shared-domain
    hash: sha256:8f3a3fdc8dc47344ada0fc5f56361026effe8099d44c68c9a6ef54a68e488302
    supports: [concepts, capability.create-edit, related-pages]
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: user-interface
    hash: sha256:c8650707d5e88cc6cfe386a6b6c240225b8c5ba153f63f4b00be04c92f4cbf65
    supports: [related-pages]
  - path: backend/business/client_provider.go
    role: backend-handler
    hash: sha256:c7ecb92cebcdf753b5a7f27110eb7ab5c941e65709f59db146c529063e434938
    supports: [capability.create-edit, rules, troubleshooting]
  - path: backend/business/types/client_provider.go
    role: data-model
    hash: sha256:83d725eaeac0c081e5b5871f1f015a171981bab763d7b2e1a700ddfdc22ce748
    supports: [concepts, capability.create-edit, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [capability.create-edit, rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [capability.create-edit, rules]
```
