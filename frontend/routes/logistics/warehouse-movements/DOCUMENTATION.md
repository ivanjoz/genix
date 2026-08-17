---
schema: 1
page_id: logistics.warehouse-movements
route: /logistics/warehouse-movements
title: Warehouse Movements (Almacén Movimientos)
status: implemented
visibility: tenant
description_en: >-
  Warehouse movement report. Query the history of stock entries and exits recorded for a
  warehouse, filtering by warehouse, date range, product, movement type, batch/lot code, serial
  number, or document number; search the loaded results by free text.
description_es: >-
  Reporte de movimientos de almacén. Consultar el historial de entradas y salidas de stock
  registradas para un almacén, filtrando por almacén, rango de fechas, producto, tipo de
  movimiento, código de lote, número de serie o número de documento; buscar en los resultados
  cargados por texto libre.
---

# Warehouse Movements (Almacén Movimientos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Warehouse Movements — labeled **Movements Report (Rep. Movimientos)** in the side menu — is a
read-only report over the append-only stock ledger (`warehouse_product_movement`). It shows
every recorded stock entry (`entrada`) and exit (`salida`) for a product in a warehouse: date,
product, batch/lot, serial number (SKU), movement type, quantity, resulting warehouse quantity,
linked document, and the user who created it.

This page only queries and displays the ledger; it does not create, edit, or delete movements.
Stock is changed elsewhere — manual stock adjustments on **Stock Changes (Cambios Stock)**,
merchandise reception on **Purchase Orders (Órdenes de Compra)**, and product delivery on sale
orders — and every one of those operations writes rows that eventually surface in this same
report.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **movement (movimiento)** is one ledger row: a signed `Quantity` change for one
  (warehouse, product, presentation) bucket, optionally scoped to a **batch/lot (lote)** and/or
  a **serial number (SKU)**. Movements are never edited or deleted; correcting stock means
  recording a new movement.
- **Movement type (Tipo Movimiento)** on this page's filter and column only recognizes two
  labels: **Entrada Manual** and **Salida Manual**, which correspond to manual stock adjustments
  made on **Stock Changes (Cambios Stock)**. Merchandise received through Purchase Orders is
  also stored as an unlabeled positive movement (it is not literally a manual entry but is not
  distinguished from one here), and product deliveries created by a sale order carry a different
  internal type that this page does not name — those rows still appear in the list and count
  in totals, but their **Movimiento** cell renders as `-` and they cannot be isolated with the
  **Tipo Movimiento** filter.
- A **batch/lot (lote)** groups received merchandise under one code (`Código Lote`); a
  **serial number (Nº Serie / SKU)** identifies one unit. Either can carry its own detail
  quantity per warehouse, separate from the warehouse's free (non-lot, non-serial) quantity.
- **Document (Documento)** is the linked business record ID that produced the movement — for
  example a purchase order or a sale order ID — shown as a plain number; the report does not
  turn it into a link to open that order.
- **Source Warehouse (Almacén Origen)** and **Destination Warehouse (Almacén Destino)** are
  both rendered on every row, intended for warehouse-to-warehouse transfers. In the currently
  implemented movement sources (manual adjustment, purchase reception, sale delivery), nothing
  populates a source/origin warehouse, so **Almacén Origen** is always blank in practice — Genix
  does not currently have an implemented action that records a two-sided warehouse transfer.
  Treat this report as an entry/exit ledger per warehouse, not as a transfer log, despite the
  column existing for that purpose.

<!-- DOC-ID: capability.query-range -->
## Query movements by warehouse and date range (Consultar movimientos por almacén y rango de fechas)

### User intention (Intención del usuario)

Review what entered or left a warehouse for a product (or all products) over a period, to
audit stock changes, investigate a discrepancy, or check recent activity.

### Where to find it (Dónde encontrarlo)

Open **Logistics (Logística) → Movements Report (Rep. Movimientos)** at
`/logistics/warehouse-movements`. Click the search icon (top-left button) to open the filter
panel, set the fields, and press **Buscar**. The panel closes automatically after the button is
pressed, whether or not the query actually ran.

### Required information and prerequisites (Requisitos previos)

- **Warehouse (Almacén)** is required for this range query; the page auto-selects the first
  warehouse from the company's warehouse list as soon as it loads, but it can be cleared or
  changed in the filter panel. Clicking **Buscar** with no warehouse selected (and none of the
  direct-lookup fields filled — see below) silently does nothing: no request is sent and no
  error is shown.
- **Start Date (Fecha Inicio)** and **End Date (Fecha Fin)** default to the last 7 days and are
  both required for a range query.
- **Product (Producto)** and **Movement Type (Tipo Movimiento)** are optional narrowing filters.
- The strip below the search button always echoes the currently applied Start/End date,
  Almacén, Producto, and Tipo values (showing `Todos` for an unset Producto or Tipo).

### Business rules and rationale (Reglas y razón de negocio)

The server rejects a range query when the start or end date is missing, when the end date is
earlier than the start date, or when the range spans more than **120 days** — `Sólo se pueden
consultar hasta 120 días a la vez.` A narrower combination of Almacén, Producto, and Tipo routes
the query through a more specific server-side index; when both Tipo and Almacén are set they are
combined together, otherwise only one of Tipo, Almacén, or Producto is actually used to narrow
the result beyond the date range.

### Result and side effects (Resultado y efectos)

No records are created, edited, or removed; the table simply repopulates with the matching
ledger rows, sorted newest first. Batch/lot codes referenced by the results are resolved and
cached locally so the **Batch (Lote)** column can show the lot's name instead of its numeric ID.

### Limitations (Limitaciones)

- There is no export or print action on this page; results only display in the on-screen table.
- The table has no server-side pagination; very broad filters over a long date range can return
  a large result set that is all loaded into the browser at once.
- Nothing here can revert, cancel, or annotate a movement.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo veo los movimientos de un almacén en un rango de fechas?`
- `¿Por qué no pasa nada cuando presiono Buscar?` Falta seleccionar un almacén (o llenar lote,
  serie o documento) para que la consulta se ejecute.
- `¿Por qué el rango de fechas me da error?` Sólo se permite consultar hasta 120 días seguidos.
- Search terms: `movimientos de almacén`, `entradas y salidas`, `kardex`, `historial de stock`,
  `rango de fechas`, `tipo de movimiento`.

<!-- DOC-ID: capability.lookup-direct -->
## Look up a specific batch, serial, or document (Buscar por lote, serie o documento)

### User intention (Intención del usuario)

Find every movement tied to one specific batch/lot code, one serialized unit, or one linked
business document, without needing to know its warehouse or date.

### Where to find it (Dónde encontrarlo)

In the same filter panel, fill **Batch Code (Código Lote)**, **Document # (N° Documento)**, or
**Serial # (N° Serie)** instead of relying on the date range, then press **Buscar**.

### Required information and prerequisites (Requisitos previos)

Filling any one of Código Lote, N° Documento, or N° Serie switches the query to this direct
lookup: Almacén and the date range are no longer required and, if set, are ignored by the
server. Producto and Tipo Movimiento are also ignored in this mode; only the entered
lote/serie/documento value narrows the result.

### Business rules and rationale (Reglas y razón de negocio)

If more than one of these three fields is filled at once, only one applies: **Serial # (N°
Serie)** takes priority over **Document # (N° Documento)**, which takes priority over **Batch
Code (Código Lote)**. A batch code with no matching lot returns the error `No se encontró un
lote con ese código.`; a batch code can match more than one internal lot record sharing the same
name, and results include movements for all of them.

### Result and side effects (Resultado y efectos)

Returns every matching ledger row regardless of date, replacing the table's current contents.

### Limitations (Limitaciones)

Combining a direct-lookup field with Almacén/Producto/Tipo does not narrow the result further;
those extra filters are simply ignored while a lote/serie/documento value is present.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo busco todos los movimientos de un lote sin saber la fecha?`
- `¿Puedo buscar por número de serie o SKU?` Sí, con el campo "N° Serie".
- `¿Por qué mi filtro de almacén no afectó la búsqueda por documento?` Porque la búsqueda directa
  por lote/serie/documento ignora almacén, fecha, producto y tipo.
- Search terms: `buscar por lote`, `buscar por serie`, `SKU`, `número de documento`, `código de
  lote`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- This page has exactly one behavior: querying and displaying the movement ledger. Nothing on
  it edits stock, cancels a movement, or writes to any other record.
- The free-text search box (top right) filters only the rows already loaded in the browser,
  matching against the product name and the creating user's username — it does not search
  batch/lot, serial number, warehouse names, or movement type, and it does not trigger a new
  server query.
- The **User (Usuario)** column shows who was signed in when the movement was recorded — the
  person who confirmed the purchase reception, completed the sale, or ran the manual
  adjustment — not a generic "system" label.
- Reaching this page requires the **Rep. Movimientos** access; that access only offers the
  **Visualizar (View)** and **Todo (Full)** levels in the catalog (no separate Crear/Editar), and
  either one is enough to open the route — a redirect with `No posee el acceso "Rep.
  Movimientos" para acceder a /logistics/warehouse-movements` sends the user back to `/`
  otherwise. That check only runs in the frontend route guard: the server's
  `GET.warehouse-movements` endpoint itself has no access mapped to it in the catalog, so any
  authenticated user of the company who calls the API directly (bypassing the page) can read
  movement data regardless of whether they were granted this access.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **Nothing happens after pressing Buscar:** the filter panel closes regardless of whether a
  query actually ran. Check that a warehouse is selected, or that a batch code/serial
  number/document number is filled in for a direct lookup.
- **"Debe especificar el rango de dates." / "La date final no puede ser menor a la date
  inicial." / "Sólo se pueden consultar hasta 120 días a la vez.":** these are the server's
  literal validation messages for a range query with a missing date, an inverted range, or a
  span over 120 days; adjust Fecha Inicio/Fecha Fin accordingly.
- **"No se encontró un lote con ese código.":** the batch/lot code typed in Código Lote does not
  match any recorded lot for the company.
- **A movement's "Movimiento" cell shows `-`:** the row comes from an operation (for example a
  sale delivery) whose internal movement type has no label configured on this page; it is still
  a real, correctly recorded entry or exit.
- **Almacén Origen is always empty:** expected today — no implemented Genix action currently
  records the source side of a warehouse-to-warehouse transfer.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Stock Changes (Cambios Stock)** at `/logistics/products-stock`: register the manual stock
  adjustments that appear here as **Entrada Manual** / **Salida Manual**.
- **Purchase Orders (Órdenes de Compra)** at `/logistics/purchase-orders`: merchandise reception
  against a confirmed order is the source of the unlabeled positive movements linked to a
  purchase-order document ID.
- Sale orders create the unlabeled outbound movements linked to a sale document ID when a sale
  includes product delivery; this route does not expose a page to create or edit that sale.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/logistics/warehouse-movements/+page.svelte
    role: page
    hash: sha256:24283d7bada7d287190c030d2037da32c7f8b56a040a3727ec1061828dd4354f
    supports: [page-purpose, concepts, capability.query-range, capability.lookup-direct, rules]
  - path: frontend/routes/logistics/warehouse-movements/warehouse-movements.svelte.ts
    role: frontend-service
    hash: sha256:65bce92904055963a9bdda22f445f35d7d16f465e449b4d86441963d6fe5cce6
    supports: [concepts, capability.query-range, capability.lookup-direct, troubleshooting]
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.query-range]
  - path: frontend/domain-components/SideMenu.svelte
    role: user-interface
    hash: sha256:bf9d01b2e3e83aa5186d7e5105af8a6e4674a4e1cea084160b2a4ba34d7620a5
    supports: [related-pages]
  - path: frontend/routes/+layout.svelte
    role: permissions
    hash: sha256:d39952cdbb641e90e952ae87109727da0300c178dac868654ed70d80ff2c60af
    supports: [rules]
  - path: frontend/packages/genix-ui/security/create-security.ts
    role: permissions
    hash: sha256:390de882f2a453a905322a7a5ef6a7265fdce7d5cc6b07b4d6ddbc0f0979fcbc
    supports: [rules]
  - path: frontend/routes/security/access-profiles/+page.svelte
    role: shared-domain
    hash: sha256:f113cbaee8d9ad9f180f07993ce42054dde14523c91e38075b9178b1723f7a55
    supports: [rules]
  - path: frontend/routes/security/access-profiles/access-profiles.svelte.ts
    role: shared-domain
    hash: sha256:34269127c34cd8b1964aa51143315a914bd52b1f98ab98469c9dff70e195c72f
    supports: [rules]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: data-model
    hash: sha256:5f468bbace2e4a7000cb389e48473da271bdd717bcad8f274ef4f308c3d7ea5e
    supports: [concepts]
  - path: backend/logistics/product-stock-movement.go
    role: backend-handler
    hash: sha256:3f1bcf6a690c536448f47b89173b6ee636aab1150be37a0adfd11819b58a8a2e
    supports: [concepts, capability.query-range, capability.lookup-direct, rules, troubleshooting]
  - path: backend/logistics/types/product-stock-movement.go
    role: data-model
    hash: sha256:11638ab2ed6b9a8698cd4f2da79d6eb4ddb9ce3e2ddce5341615c6b046245d5d
    supports: [concepts, rules]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:d0c3d59fe6e6f5f47a5a82ef4e0f5fc8fa62f10466dba242af3f5ba39bff65a8
    supports: [concepts, related-pages]
  - path: backend/sales/sale_order_create.go
    role: business-logic
    hash: sha256:03dfbe6e253c0d73c85654993442e2479e33fdb08a44d74e00fd66076643656f
    supports: [concepts, related-pages]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
```
