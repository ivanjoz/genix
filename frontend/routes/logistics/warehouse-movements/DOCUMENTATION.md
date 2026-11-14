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
merchandise reception on **Purchase Orders (Órdenes Compra)**, and product delivery on sale
orders — and every one of those operations writes rows that eventually surface in this same
report.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **movement (movimiento)** is one ledger row: a signed `Quantity` change for one
  (warehouse, product, presentation) bucket, optionally scoped to a **batch/lot (lote)** and/or
  a **serial number (SKU)**. Movements are never edited or deleted; correcting stock means
  recording a new movement.
- **Movement type (Tipo Movimiento)** says what caused each movement, and every movement has
  one. The manual reasons chosen on **Stock Changes (Cambios Stock)** are **Entrada manual**,
  **Saldo inicial**, **Salida manual**, **Merma** and **Consumo interno**. The rest are written
  by their own document: **Ingreso por orden de compra**, **Compra de insumos**, **Entrega
  (venta)**, **Reingreso (anulación de venta)**, the asset movements (**Alta de activo**, **Baja
  de activo**, **Traslado de activo** and **Corrección de serie de activo**), and the two
  movements of the daily production close (**Cierre del día**): **Consumo de producción**, a
  supply leaving the supplies warehouse by a made product's recipe, and **Ingreso de producción**,
  the made product entering the sales warehouse at its production cost. Any of them can be
  isolated with the **Tipo Movimiento** filter.
- **Entrega (venta)** is written when a sale is delivered — whether the sale was delivered at the
  Point of Sale, from a web order, or from a restaurant table — one exit per line from the
  delivery warehouse, linked to the sale. The lines of a made product (`producto elaborado`) are
  the exception: their exit is not written at delivery but by the next daily production close,
  which writes the **Entrega (venta)** exits of the sales it settles, dated like its own
  movements: the day closed or yesterday, whichever is later.
- **Quantity (Cantidad)** shows whole units, plus the sub-unit part when there is one, in the
  product's sub-unit name (for example `2 + 3 g`); a movement of sub-units only shows just the
  sub-units. Exits are shown in red and entries in blue.
- A **batch/lot (lote)** groups received merchandise under one code (`Código Lote`); a
  **serial number (Nº Serie / SKU)** identifies one unit. Either can carry its own detail
  quantity per warehouse, separate from the warehouse's free (non-lot, non-serial) quantity.
- **Document (Documento)** is the linked business record ID that produced the movement — for
  example a purchase order, a sale order ID, or the production close number (`Cierre N°`) for
  Consumo / Ingreso de producción — shown as a plain number; the report does not
  turn it into a link to open that order.
- **Destination Warehouse (Almacén Destino)** is the warehouse the movement changed, with its
  resulting quantity in parentheses. **Source Warehouse (Almacén Origen)** is blank on every
  movement except an asset transfer (**Traslado de activo**), the only implemented
  warehouse-to-warehouse transfer. A transfer is written as two rows — an exit from the warehouse
  the asset leaves and an entry into the one it goes to — and on each row **Almacén Origen**
  names the *other* warehouse. So on the exit row the columns read backwards: **Almacén Destino**
  shows the warehouse the asset left and **Almacén Origen** the one it went to. The number in
  parentheses next to **Almacén Origen** is always 0; it is not that warehouse's stock. Treat this
  report as an entry/exit ledger per warehouse, not as a transfer log.

<!-- DOC-ID: capability.query-range -->
## Query movements by warehouse and date range (Consultar movimientos por almacén y rango de fechas)

### User intention (Intención del usuario)

Review what entered or left a warehouse for a product (or all products) over a period, to
audit stock changes, investigate a discrepancy, or check recent activity.

### Where to find it (Dónde encontrarlo)

Open **Operations (Operaciones) → Warehouse (Almacén) → Movements Report (Rep. Movimientos)** at
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
consultar hasta 120 días a la vez.` Almacén, Producto, and Tipo Movimiento all narrow the result:
the server answers by date range plus one of them (Tipo and Almacén together, or else Tipo, else
Almacén, else Producto), and the page then drops the loaded rows that do not match the remaining
filters.

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
- **An asset transfer shows two rows:** a transfer is recorded as an exit from the origin
  warehouse and an entry into the destination, each naming the other warehouse.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Stock Changes (Cambios Stock)** at `/logistics/products-stock`: register the manual stock
  adjustments that appear here with the reason chosen on save (Entrada manual, Saldo inicial,
  Salida manual, Merma, Consumo interno).
- **Purchase Orders (Órdenes Compra)** at `/logistics/purchase-orders`: merchandise reception
  against a confirmed order is the source of the **Ingreso por orden de compra** movements linked
  to a purchase-order document ID.
- Sale orders create the **Entrega (venta)** movements linked to a sale document ID when a sale
  is delivered — except for made products, whose exit waits for the daily production close; this
  route does not expose a page to create or edit that sale.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/logistics/warehouse-movements/+page.svelte
    role: page
    hash: sha256:057e04190b5479cc88aec7b935b3013ac637a4080f9a7b3b76ce2e03853443c6
    supports: [page-purpose, concepts, capability.query-range, capability.lookup-direct, rules]
  - path: frontend/routes/logistics/warehouse-movements/warehouse-movements.svelte.ts
    role: frontend-service
    hash: sha256:bf9ea17e300f3f6992cae1ed09f1ba0199e8c5edd602af62db7a5877c967a86e
    supports: [concepts, capability.query-range, capability.lookup-direct, troubleshooting]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:cb61a6f79a034e4d468b46c173733b815f0af9007a01d1f3e69b92d358913d60
    supports: [page-purpose, capability.query-range, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:4f2d4c16777913fddf196959c4341290e69a12a78323a5b95f8fa5d4738ca34d
    supports: [capability.query-range]
  - path: frontend/core/quantity.ts
    role: shared-domain
    hash: sha256:3dda78678bc8c4d891b7d2b679d0a0549a97feb81ce05d23ada3d4cea08194ba
    supports: [concepts]
  - path: frontend/domain-components/SideMenu.svelte
    role: user-interface
    hash: sha256:6a4e810acfdb129da0914c3615c5f88ed67d652efddd0edf51bdd7c7ee57452e
    supports: [related-pages]
  - path: frontend/routes/+layout.svelte
    role: permissions
    hash: sha256:02ee210641045619281189cb8ede513e3b6570de831a4ce37f03098f6c9ced17
    supports: [rules]
  - path: frontend/packages/genix-ui/security/create-security.ts
    role: permissions
    hash: sha256:e83bbfd60ed7823ffdc48b7846ee1745b23f83ff3f962b1059e57070e9aebb8c
    supports: [rules]
  - path: frontend/routes/security/users-profiles/users-profiles.ts
    role: shared-domain
    hash: sha256:5efed57e28e00238aab5a6995986d1ddd721a003cc59c635279418d742abb6d9
    supports: [rules]
  - path: frontend/routes/security/users-profiles/users-profiles.svelte.ts
    role: shared-domain
    hash: sha256:c4eb5b0302c1f9f54cf0b447be33d4d7972b76c88501b952a2ea6671f537d618
    supports: [rules]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: data-model
    hash: sha256:5b40e3b6c6ef8421e4a341e887d67b3bece49badd16259898d659e778ce06115
    supports: [concepts]
  - path: backend/logistics/product-stock-movement.go
    role: backend-handler
    hash: sha256:e324e10b9453a23f6c5dd1a8698cc3f0ea641e30075807f06e1c4239c31b2955
    supports: [concepts, capability.query-range, capability.lookup-direct, rules, troubleshooting]
  - path: backend/logistics/types/product-stock-movement.go
    role: data-model
    hash: sha256:9fda1c20a6e7a9bc40ca1edc0b0da501bbc660ee9d1a0ce5d56ffea252afeb4c
    supports: [concepts, rules]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:d9c2b28be2e49e0ddd10c677f4f343dc5f79fb2c354e134804be6c621f11dbf4
    supports: [concepts, related-pages]
  - path: backend/sales/sale_order_create.go
    role: business-logic
    hash: sha256:0601dd000d061f432f4a7dfa0228d673b2debc0a744448d6ddfb78a30978c78d
    supports: [concepts, related-pages]
  - path: backend/sales/saleissue/types/new_sale.go
    role: business-logic
    hash: sha256:79480f8435d4182f16c8ebca2b5717be396423cd73a571f5ef1e2a84a229e33e
    supports: [concepts, related-pages]
  - path: backend/sales/types/sale_actions.go
    role: business-logic
    hash: sha256:ad4c117e0bce8f4c8e88f85e6aa3677090935cf6b6127c944f72bf1601413109
    supports: [concepts, related-pages]
  - path: backend/sales/online_sale_order.go
    role: business-logic
    hash: sha256:6d521468f6ba3c7074be0ca54529ba9cc3e7453370483539546febb16b0dedac
    supports: [concepts]
  - path: backend/dining/table_order_charge.go
    role: business-logic
    hash: sha256:32e32d0a5349d8e43895fec3d0ca47a09fed34037cc3534d1f69b290b9f25632
    supports: [concepts]
  - path: backend/logistics/types/stock_movement_apply.go
    role: business-logic
    hash: sha256:d8cca9053dd870e394777a265228c90516a9f1f9dd9208c69d7f97a3e1c39496
    supports: [concepts, troubleshooting]
  - path: backend/accounting/asset_api.go
    role: business-logic
    hash: sha256:b4dc87ae8b587f8b71117c40def812a5c9ae612f1ce7026e053c34b3708d984f
    supports: [concepts, troubleshooting]
  - path: backend/access.toml
    role: permissions
    hash: sha256:c19cf85130065dcc0f839796e1e9b4b98a1b78df465ccb20d2b63b59a1a94e59
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:419fcf0ac2e792df25ed47a467573910ddcad540f02d929dbcce567736b88105
    supports: [rules]
  - path: frontend/core/stock-movement-type.ts
    role: shared-domain
    hash: sha256:cfcee015c4d6fadb03f9a9e5d66064e93d734cfa5216e2d36996c47140eecc9c
    supports: [concepts]
  - path: backend/logistics/types/stock_movement_type.go
    role: shared-domain
    hash: sha256:2556d42f6e9cc99c8c179a8536a80d26bf41702648d11efd34f6e9bfc3de4b8f
    supports: [concepts]
  - path: backend/production/daily_close.go
    role: business-logic
    hash: sha256:7ae0b03e927362b3c467d29657303b41f9f0dbc979ad54b92f79807ab9b025a1
    supports: [concepts]
```
