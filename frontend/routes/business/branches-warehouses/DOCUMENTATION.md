---
schema: 1
page_id: business.branches-warehouses
route: /business/branches-warehouses
title: Branches & Warehouses (Sedes & Almacenes)
status: implemented
visibility: tenant
description_en: >-
  Branch and warehouse management. Sedes tab: create and edit business locations with name,
  phone, address, and geographic location (department/province/district). Almacenes tab: create
  and edit warehouses linked to a branch, and design each warehouse's storage grid (rows and
  levels of storage blocks).
description_es: >-
  Gestión de sedes y almacenes. Pestaña Sedes: crear y editar sucursales con nombre, teléfono,
  dirección y ubicación geográfica (departamento/provincia/distrito). Pestaña Almacenes: crear y
  editar almacenes asociados a una sede, y diseñar la grilla de almacenamiento (filas y niveles de
  bloques) de cada almacén.
---

# Branches & Warehouses (Sedes & Almacenes)

<!-- DOC-ID: page-purpose -->
## Page purpose

Branches & Warehouses (`Sedes & Almacenes`) is the page where a company registers its physical
locations. A branch (`sede`) is a business site — an office, store, or facility — identified by
name, address, phone, and geographic location. A warehouse (`almacén`) is a storage facility that
always belongs to one branch and, optionally, has a designed storage grid (`layout`) of rows and
levels used to organize physical storage blocks (`bloques`).

Other Genix pages (Cash & Banks accounts, Sales Orders, Purchase Orders, Products Stock, and
Warehouse Movements) reference the branches and warehouses created here; this page only manages
the sites and warehouses themselves, not the stock, cash, or sales activity that happens at them.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **branch (sede)** is a company location with a name, description, address, and a geographic
  district (`distrito`). Genix derives the location's province and department from the selected
  district's ubigeo hierarchy and shows them together in the branch list's City column.
- A **warehouse (almacén)** always belongs to exactly one branch (`SiteID`). It has its own name
  and description, independent of the branch's.
- A **layout** is a named storage-grid section inside a warehouse, defined by a **Rows (Filas)**
  count and a **Levels (Niveles)** count. A warehouse can have several layout sections (for
  example, different racks or storage zones), each with its own grid size.
- A **block (`bloque`)** is one labeled cell inside a layout's grid — the specific row/level
  position where a text label (e.g. a location code) can be recorded. Only cells the user
  actually types a label into become blocks; empty, untouched cells are not saved as blocks.
- Country/political-division data is limited to Peru in the current interface: the branch form's
  location selector only loads districts for country ID 604 (Peru), so this page currently has no
  way to select a location outside Peru's department/province/district hierarchy.

<!-- DOC-ID: capability.browse -->
## Find a branch or warehouse (Buscar una sede o un almacén)

Open **Business (Negocio) → Sites & Warehouses (Sedes & Almacenes)** at
`/business/branches-warehouses`. The page has two tabs: **Branches (Sedes)** and
**Warehouses (Almacenes)**.

- **Sedes** lists every branch with ID, Name, Address, City (shown as `Province > District`), and
  last-updated date/time.
- **Almacenes** lists every warehouse with ID, the owning branch's name (or `Sede-{ID}` if the
  branch can't be resolved locally), Name, a Layout summary, a raw Status number, and last-updated
  date/time.

The filter box above each list narrows it locally (against the already-loaded data, not a
server-side search): the Sedes filter matches name, address, or city; the Almacenes filter matches
warehouse name or its branch's name. Clicking a row opens it for editing in the corresponding
modal.

<!-- DOC-ID: capability.manage-branches -->
## Create or edit a branch (Crear o editar una sede)

### User intention (Intención del usuario)

Register a new business location, or update an existing one's description, address, or geographic
location.

### Where to find it (Dónde encontrarlo)

On the **Sedes** tab, use the green create button to open the branch modal for a new branch, or
click an existing row to open the same modal pre-filled for editing. Save with the modal's
**Save (Guardar)** action.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** and **Address (Dirección)**: both required, and the form blocks saving unless
  each has at least 4 characters ("Name and address must be at least 4 characters.|El nombre y la
  dirección deben tener al menos 4 caracteres.").
- **Description (Descripción)** and **Phone (Teléfono)**: optional, free-text fields shown in the
  form.
- **Department | Province | District (Departamento | Provincia | Distrito)**: one searchable
  selector whose options are already-composed district names (`Department ► Province ► District`);
  choosing one sets the branch's district directly — this is a single combo box, not three
  cascading dropdowns.
- **Name** and **Phone** are only editable while creating a new branch: once a branch has been
  saved (has an ID), both fields are disabled in the edit form and cannot be changed afterward from
  this page.

### Business rules and rationale (Reglas y razón de negocio)

The server independently re-checks the name length (its message currently reads "El nombre debe
poseer al menos 3 caracteres.", though the actual check requires at least 4) and requires a valid
district (`Debe seleccionar una ciudad válida para la sede.`) before saving; it does not
independently re-check the address length that the frontend enforces.

### Result and side effects (Resultado y efectos)

Saving creates or updates the branch record. The branch becomes selectable everywhere else in
Genix that assigns a branch (warehouses on this page, cash/bank accounts, sales orders, and
similar workflows).

### Limitations (Limitaciones)

- The **Phone (Teléfono)** field is accepted by the form but is not part of the data the server
  actually stores for a branch — the branch record Genix persists has no phone attribute at all.
  A value typed into Phone stays visible in the browser for the rest of that session (because the
  page merges the submitted form back into its local list without re-reading the server's stored
  record), but it is never saved and disappears the next time branch data is reloaded from the
  server.
- There is no field on this page to change a branch's status; a new branch is simply active, and
  nothing here deactivates one.
- Once created, a branch's Name cannot be renamed from this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo una sede o sucursal nueva?`
- `¿Por qué no puedo cambiar el nombre o el teléfono de una sede ya creada?`
- `¿Se guarda el teléfono de la sede?` Actualmente no: el campo se ve en el formulario pero el
  valor no se persiste en el servidor.
- Search terms: `sede`, `sucursal`, `local`, `dirección`, `teléfono`, `departamento`, `provincia`,
  `distrito`, `ubigeo`.

<!-- DOC-ID: capability.manage-warehouses -->
## Create or edit a warehouse (Crear o editar un almacén)

### User intention (Intención del usuario)

Register a warehouse/storage facility under a branch, or update its name, branch assignment, or
description.

### Where to find it (Dónde encontrarlo)

On the **Almacenes** tab, use the green create button to open the warehouse modal, or click an
existing row to open it pre-filled for editing. Save with the modal's **Save (Guardar)** action.

### Required information and prerequisites (Requisitos previos)

- **Branch (Sede)**: required; a searchable selector over the branches already registered on the
  Sedes tab. The frontend blocks saving with `Please select a branch.|Debe seleccionar una sede.`
  if none is chosen.
- **Name (Nombre)**: required, at least 4 characters
  (`Name must be at least 4 characters.|El nombre debe tener al menos 4 caracteres.`).
- **Description (Descripción)**: optional.

Unlike branches, a warehouse's Branch and Name remain editable after creation — this modal does
not disable either field once the warehouse has an ID.

### Business rules and rationale (Reglas y razón de negocio)

The server re-checks name length and that a branch was selected, returning
`Faltan propiedades del almacén` when either is missing.

### Result and side effects (Resultado y efectos)

Saving creates or updates the warehouse record and makes it selectable wherever Genix assigns a
warehouse (Purchase Orders, Products Stock, Warehouse Movements, and similar logistics workflows).
A basic warehouse (with no layout sections defined) saves without issue.

### Limitations (Limitaciones)

There is no field on this page to change a warehouse's status; the Status column shows the raw
stored number instead of a label, and nothing on this page changes it.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo un almacén y lo asigno a una sede?`
- `¿Puedo cambiar el almacén de sede después de creado?` Sí, el selector de Sede sigue habilitado
  al editar.
- Search terms: `almacén`, `bodega`, `depósito`, `sede`, `crear almacén`.

<!-- DOC-ID: capability.warehouse-layout -->
## Design the warehouse storage grid (Diseñar el layout / grilla del almacén)

### User intention (Intención del usuario)

Define one or more storage-grid sections for a warehouse — for example separate racks or zones —
each as a grid of rows and levels, and label individual storage positions inside that grid.

### Where to find it (Dónde encontrarlo)

On the **Almacenes** tab, each row's Layout cell shows a pencil button; clicking it opens a side
panel titled "Layout {warehouse name}". Use the panel's green **+** button to add a new layout
section. Each section shows a **Name**, a **Rows (Filas)** count, and a **Levels (Niveles)** count,
followed by a grid of text cells (one per row/level combination) where a short label can be typed
per storage position. A trash icon removes a whole section. Saving uses the same side panel's
**Save** action (which calls the warehouse save operation).

### Required information and prerequisites (Requisitos previos)

Editing a layout requires an already-created warehouse (the pencil action is only available from
the Almacenes list, not from the create-warehouse modal). Each section needs a Name, and the grid
dimensions come from its Rows and Levels counts, which default to 2 and 3 respectively for a new
section.

### Business rules and rationale (Reglas y razón de negocio)

The server rejects the save with `Hay un layout mal creado.` when a layout section's ID is missing
or its Name is empty.

### Result and side effects (Resultado y efectos)

Intended behavior: saving stores the layout sections (name, row/level counts) and the labeled
blocks under the warehouse record, and the Almacenes list's Layout column then summarizes them
(number of sections, and their average level/row counts).

### Limitations (Limitaciones)

<!-- DOCUMENTATION_GAP: This is a confirmed, currently-blocking defect, not a stylistic
limitation; recorded here as a limitation because there is no other DOC-ID it fits under and the
gap format is reserved for missing rationale, not for verified bugs. -->
Saving currently fails whenever a warehouse has at least one layout section. The side panel lets a
user type a Name, Rows, and Levels for a section and appears to accept it, but every save request
built from that data reaches the server with the section's Name always empty (a mismatch between
the property names the browser sends and the ones the server expects), so the server's own
validation rejects it with `Hay un layout mal creado.` every time, regardless of what the user
typed. In the current implementation, a warehouse's layout/grid cannot actually be saved through
this page; only the warehouse's own Name, Branch, and Description reliably persist. A warehouse
with no layout sections defined saves without problem.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo agrego filas y niveles de almacenamiento a un almacén?`
- `¿Por qué me sale "Hay un layout mal creado" al guardar el layout?` Es una falla conocida
  actual: el layout no se guarda correctamente aunque los datos se vean completos en el
  formulario.
- Search terms: `layout`, `grilla`, `filas`, `niveles`, `bloques`, `ubicación de almacén`,
  `racks`, `zonas de almacén`.

<!-- DOC-ID: capability.remove-row -->
## Remove a branch or warehouse from the list (Quitar una sede o almacén de la lista)

Both the branch modal and the warehouse modal expose a **Delete (Eliminar)** action once an
existing record is open. Pressing it re-submits the currently loaded form data unchanged (the same
save request as an edit) and then removes that row only from the browser's local list; there is no
confirmation prompt. This does **not** send any deletion or deactivation signal to the server —
the record's stored status is not changed, so the branch or warehouse remains active. Because
saving also marks the underlying cached data as needing a refresh, the removed row can reappear
once the list reloads from the server (for example after reopening the page), since the server
still returns it as an active record.

### Limitations (Limitaciones)

Treat **Eliminar** here as removing the row from view only, not as deactivating or deleting the
branch/warehouse. There is currently no verified way on this page to deactivate or permanently
delete a branch or warehouse.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo elimino una sede o un almacén?` El botón "Eliminar" sólo la quita de la lista visible;
  no la desactiva ni la borra en el servidor.
- `¿Por qué la sede/almacén que "eliminé" reaparece después de recargar?` Porque su registro
  sigue activo en el servidor.
- Search terms: `eliminar sede`, `eliminar almacén`, `borrar almacén`, `desactivar sede`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Creating or editing either a branch or a warehouse requires the "Sedes & Almacenes" access at
  the Full (Todo) level; that access only offers View or Full (no intermediate Create/Edit
  levels), so granting someone the ability to save on this page always means granting full
  control of it. Viewing this page's lists is currently open to any authenticated user of the
  company, since no read-only access is mapped for the GET route in the catalog.
- Every save on this page (branch or warehouse) goes through the same "submit the whole form,
  merge it into the local list" pattern, which is why **Eliminar** behaves identically for both
  entities (local-only removal, described above).

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El nombre y la dirección deben tener al menos 4 caracteres" / "El nombre debe tener al menos
  4 caracteres":** lengthen the Name (and, for a branch, the Address) to at least 4 characters.
- **"Debe seleccionar una ciudad válida para la sede":** pick a district from the location
  selector; an empty selection is rejected by the server even if the frontend's own check passed.
- **"Faltan propiedades del almacén":** select a Branch and enter a Name of at least 4 characters
  for the warehouse.
- **"Hay un layout mal creado" every time a layout is saved:** current, confirmed limitation — see
  the warehouse-layout section above. There is no supported workaround from this page today.
- **A branch's phone number doesn't stick after reloading:** expected given the current
  limitation — the Phone field is not part of what the server stores for a branch.
- **A "deleted" branch/warehouse reappears:** the current Delete action only hides the row
  locally; it does not deactivate the record on the server.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash & Banks (Cajas & Bancos)** at `/finance/cash-banks`: cash/bank accounts are configured
  against a branch created here.
- **Purchase Orders**, **Products Stock**, and **Warehouse Movements** (Logistics/Órdenes de
  Compra, Stock de Productos, Movimientos de Almacén): assign a warehouse created on this page's
  Almacenes tab as the location for stock and inventory movements.
- Sales order creation assigns a branch created on this page to a sale.
- This page is only for configuring branches, warehouses, and warehouse layouts; use the pages
  above for the stock, cash, or sales activity that happens at them.

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
  - path: frontend/routes/business/branches-warehouses/+page.svelte
    role: page
    hash: sha256:98d524134fe109cfa4c915e12eb8238a032d0b09622b86e0a2994b7f1b457421
    supports: [page-purpose, concepts, capability.browse, capability.manage-branches, capability.manage-warehouses, capability.warehouse-layout, capability.remove-row, rules, troubleshooting]
  - path: frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts
    role: frontend-service
    hash: sha256:8f3a3fdc8dc47344ada0fc5f56361026effe8099d44c68c9a6ef54a68e488302
    supports: [concepts, capability.manage-branches, capability.manage-warehouses, capability.warehouse-layout, rules]
  - path: frontend/routes/business/branches-warehouses/WarehouseLayoutEditor.svelte
    role: user-interface
    hash: sha256:806487a047fb3577101fcaf6c8e1042f8a89e8786cc31b78ab3670d6d0ace1f1
    supports: [capability.warehouse-layout]
  - path: backend/business/locations-warehouses.go
    role: backend-handler
    hash: sha256:591db493e761ac7d062bca3889e7ff4b3f6e85ced92017621658957d8742af39
    supports: [capability.manage-branches, capability.manage-warehouses, capability.warehouse-layout, rules, troubleshooting]
  - path: backend/business/main.go
    role: backend-handler
    hash: sha256:2672bd44b6ca86e692c601c6c7389de2bc72eef25ccfa584c7b75233e5786a83
    supports: [capability.manage-branches, capability.manage-warehouses]
  - path: backend/business/types/productos.go
    role: data-model
    hash: sha256:a26703a968048adf7aa708d753e7d689bfeea5e0d9ad5c98447f078cc1a63ef5
    supports: [concepts, capability.manage-branches, capability.manage-warehouses, capability.warehouse-layout, rules]
  - path: backend/business/types/generales.go
    role: data-model
    hash: sha256:9d96bb2ae3bc6a5d8626bdc4a6a6348e63ddb97072434bde3403ec70b3178789
    supports: [concepts, capability.manage-branches]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
```
