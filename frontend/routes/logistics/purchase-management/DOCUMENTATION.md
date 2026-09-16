---
schema: 1
page_id: logistics.purchase-management
route: /logistics/purchase-management
title: Purchase Management (Gestión de Compras)
status: implemented
visibility: tenant
description_en: >-
  Product supply planning (`abastecimiento`). Compare each product's current stock against a
  configured minimum stock and view a 30-day inflow/outflow/final-stock chart; configure a
  product's minimum stock, estimated daily sales, and supplier options (price, capacity, delivery
  time), one product at a time or in bulk through an Excel export/import. Does not create purchase
  orders.
description_es: >-
  Planificación de abastecimiento por producto. Comparar el stock actual de cada producto contra
  un stock mínimo configurado y ver un gráfico de 30 días de entradas/salidas/stock final;
  configurar el stock mínimo, las ventas diarias estimadas y las opciones de proveedor (precio,
  capacidad, tiempo de entrega) de un producto, uno por uno o masivamente con exportación e
  importación de Excel. No crea órdenes de compra.
---

# Purchase Management (Gestión de Compras)

<!-- DOC-ID: page-purpose -->
## Page purpose

Purchase Management (`Gestión de Compras`, this page's own name for what the code and API call
`abastecimiento`/supply) is a per-product supply-planning table. For every product in the
catalog it shows the current stock (`Stock Actual`) next to a manually configured minimum stock
(`Stock mínimo`), a 30-day chart of daily inflows/outflows and reconstructed ending stock, the
estimated daily sales (`Ventas/Día`), and the suppliers registered for that product with their
price, capacity, and delivery time.

This page does **not** generate a purchase request or purchase order to a supplier — despite its
menu name suggesting "compras" (purchasing), saving here only stores planning parameters
(`MinimunStock`, `SalesPerDayEstimated`, `ProviderSupply`) against the product. Creating an
actual order that a supplier fulfills happens on the separate **Purchase Orders (Órdenes de
Compra)** page.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Supply configuration (`configuración de abastecimiento`)** is the per-product record this
  page edits: a minimum stock threshold, an estimated daily sales figure, and a list of supplier
  options. A product with no configuration ever saved simply shows zeros for these fields; the
  product still appears in the table because rows are generated from the product catalog, not
  from existing supply configurations.
- **Current stock (`Stock Actual`)** is read from the product's live stock snapshot (the same
  warehouse stock ledger used by **Stock Changes/Cambios Stock**), summed across all of that
  product's stock buckets company-wide. **Minimum stock (`Stock mínimo`)** is purely a
  configured comparison value on this page; Genix does not send alerts or automatically trigger
  any reorder action when current stock falls below it.
- **Stock Movements chart (`Movimientos Stock`)**: a fixed 30-day window of bars for outflows
  (`Salidas`, red) and inflows (`Entradas`, blue) per day, plus a black line for the
  reconstructed daily ending stock (`Stock Final`). The ending-stock line is not stored per day;
  it is calculated backward from the current stock snapshot by reversing each later day's net
  movement, so it depends on the movement history Genix has for that product.
- **Provider/supplier option (`Proveedor`)** on this page is one row of Provider + Capacity +
  Delivery time (`Entrega`, in days) + Price, used for planning/comparison only. It is unrelated
  to placing an order; it does not reserve capacity or commit to a price with the supplier.
- This page's suppliers list only offers business partners of type **Provider (`Proveedor`)**
  from the shared client/provider catalog — the same catalog used by Purchase Orders and Cash &
  Banks — never plain customers.

<!-- DOC-ID: capability.browse-supply -->
## Review supply levels (Revisar niveles de abastecimiento)

Open **Logistics (Logística) → Purchase Management (Gestión de Compras)** at
`/logistics/purchase-management`. The table lists every product in the catalog with: the
product name; a two-bar comparison of current stock (green) versus minimum stock (blue, or red
when the minimum exceeds the current stock); the 30-day inflow/outflow/final-stock chart
described above; the estimated sales per day; and up to the first two configured suppliers with
their price and delivery time (additional suppliers beyond the first two are not shown in this
column). Use the search box (`Buscar producto o proveedor`) to filter the already-loaded rows
locally by product name, minimum stock, estimated sales, or supplier name. Selecting a row opens
that product's supply configuration in the side layer described below.

<!-- DOC-ID: capability.configure-supply -->
## Configure a product's supply (Configurar el abastecimiento de un producto)

### User intention (Intención del usuario)

Record how much stock should be kept on hand for a product, how fast it typically sells, and
which suppliers can provide it (with their price, capacity, and delivery time) for planning and
comparison purposes.

### Where to find it (Dónde encontrarlo)

Click a product row to open the side layer titled "Abastecimiento {nombre del producto}". It
shows the read-only product name, **Minimum Stock (Stock mínimo)**, **Estimated Sales/Day
(Ventas/Día estimadas)**, and a **Proveedores** list. Use the green **+** button to add a
supplier row, and the trash icon on a supplier card to remove it. Save with the layer's
**Save (Guardar)** action.

### Required information and prerequisites (Requisitos previos)

- The product must already exist in the product catalog; this page cannot create a product.
- Minimum stock and estimated sales/day are optional and default to 0.
- A supplier row only needs to be filled in if the user wants a supplier registered; once any
  field of a row (provider, capacity, delivery time, or price) is set, that row requires a
  **valid, active Provider (`Proveedor`)**, and capacity, delivery time, and price must each be
  zero or positive. The same provider cannot appear twice for one product.

### Business rules and rationale (Reglas y razón de negocio)

Saving from this side layer sends that one product's configuration (the same endpoint accepts many
at once, which is what the Excel import uses). The server keeps the product's key fixed and merges
the submitted values into the existing configuration for that (company, product) pair, or creates a
new one. It also rejects a configuration pointing at a product that does not exist. Empty supplier
rows (no provider,
capacity, delivery time, or price set) are dropped both by the browser and again by the server
before validation, so an accidentally added blank row never blocks saving.

### Result and side effects (Resultado y efectos)

Saving creates or updates exactly one supply-configuration record for the product, stamps it as
active, and records who updated it and when. The page updates its local table immediately and
also triggers a background refresh so the cached list stays in sync with the server. Saving does
**not** create a purchase order, reserve supplier capacity, or change the product's stock in any
warehouse.

### Limitations (Limitaciones)

- There is no delete/remove action for a saved supply configuration in this layer; to clear it,
  edit the values back to 0 and remove every supplier row instead.
- The suppliers list column shows at most the first two configured suppliers per product; check
  the side layer for the full list.
- The 30-day chart and current-stock comparison are read-only visualizations; nothing on this
  page edits stock quantities or warehouse movements.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo configuro el stock mínimo de un producto?`
- `¿Aquí genero una orden de compra al proveedor?` No; esta página sólo guarda parámetros de
  planificación. Usa **Órdenes de Compra** para generar la orden real.
- `¿Por qué no veo todos los proveedores en la tabla?` La columna de proveedores sólo muestra los
  dos primeros; abre la fila para ver el resto.
- Search terms: `abastecimiento`, `stock mínimo`, `ventas por día`, `proveedores`, `capacidad`,
  `tiempo de entrega`, `gestión de compras`.

<!-- DOC-ID: capability.import-export-supply -->
## Update supply in bulk with Excel (Actualizar el abastecimiento masivamente con Excel)

### User intention (Intención del usuario)

`Quiero exportar el abastecimiento a Excel`, `necesito cambiar el stock mínimo de muchos productos
a la vez`, `quiero cargar los proveedores de todos mis productos desde un archivo`.

### Where to find it (Dónde encontrarlo)

The **Exportar** and **Importar** buttons sit in the page toolbar, to the right of the search box
and next to the record counter.

### Required information and prerequisites (Requisitos previos)

The file is the round trip of the export, so the normal way to import is: export, edit, upload.
The sheet has one row per product and these columns — `Producto`, `Stock Actual`, `Stock Mínimo`,
`Ventas / Día Estimadas`, and then one group per supplier (`Proveedor 1`, `Proveedor 2`,
`Proveedor 3`, …) each holding `Proveedor`, `Capacidad`, `Entrega` and `Precio`.

Products and suppliers are matched **by name**, ignoring case and accents, so those names must
already exist in the platform. The sheet normally carries three supplier groups; if some product
has more suppliers configured, the export adds as many groups as that product needs.

### Business rules and rationale (Reglas y razón de negocio)

- `Stock Actual` is exported for reference only. It is reconstructed from warehouse movements, and
  whatever is typed into that column is ignored on import — this page never changes stock.
- `Precio` is written and read in currency units (e.g. `19.90`), the same as the side panel shows.
- Only rows whose values **differ from what the platform already has** are listed in the preview
  and sent when saving. Re-uploading an unedited export saves nothing.
- The sheet is authoritative for the row: the supplier list it carries replaces the saved one, so
  clearing a supplier's cells removes that supplier from the product.
- A row with an unknown or ambiguous product/supplier name is reported as an error and is excluded;
  the remaining rows can still be saved once the errors are resolved.

### Result and side effects (Resultado y efectos)

Uploading only previews. Nothing is written until **Importar** is pressed in the modal, at which
point the changed rows are sent in batches of up to 500 and the table refreshes. Each saved row has
exactly the same effect as saving that product in the side panel.

### Limitations (Limitaciones)

- The export writes the rows currently listed, so an active search filter narrows the file too.
  Clear the filter to export the whole catalog.
- The import cannot create products or suppliers; a name that does not exist is an error.
- Changing the header texts, or exporting in one language and importing with the app in the other,
  leaves columns unmatched and they are silently ignored.
- The preview highlights changed cells but is read-only; corrections are made in the spreadsheet
  and the file is uploaded again.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué no aparece ninguna fila después de subir el archivo?` Porque ninguna fila cambió
  respecto a lo guardado, o todas tienen errores listados bajo el selector de archivo.
- `¿El Excel puede modificar el stock?` No; la columna `Stock Actual` es informativa.
- `¿Cómo quito un proveedor de un producto?` Borra sus celdas en el Excel y vuelve a subirlo.
- Search terms: `exportar`, `importar`, `excel`, `carga masiva`, `plantilla`, `abastecimiento`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Viewing this page's data (the supply table and the movements chart) is currently open to any
  authenticated user of the company, because no read/GET access is mapped for it in the access
  catalog. Saving a supply configuration requires the **"Gestión de Compras"** access, which — like
  Users/Perfiles & Accesos — only offers **View (Visualizar)** and **Full (Todo)** levels, so
  granting someone the ability to save here always grants full control of it rather than a
  partial edit level.
- The visible 30-day chart window is fixed regardless of how much movement history exists for a
  product; the ending-stock reconstruction anchors on the current stock snapshot and works
  backward through whatever movement history is available.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Debe seleccionar un producto válido" (frontend):** the layer was opened or saved without a
  resolvable product; reopen it from a table row.
- **"Debe enviar un ProductID válido." (server):** mirrors the client check above.
- **"Cada fila de proveedor debe tener un proveedor válido.":** a supplier row was left with
  capacity, delivery time, or price filled in but no provider selected.
- **"Uno o más proveedores no existen." / "...están inactivos." / "...no son proveedores.":** a
  selected supplier was deleted, deactivated, or is a customer-type record rather than a
  provider-type one; pick a valid, active Provider.
- **"No se puede repetir el mismo proveedor en un product.":** remove the duplicate supplier row.
- **"El stock mínimo no puede ser negativo." / "Las ventas por día estimadas no pueden ser
  negativas." / capacity, delivery time, or price rejected as negative:** correct the numeric
  field; none of these values may be negative.
- **"producto no encontrado …" / "proveedor no encontrado …" (Excel import):** the name in the
  sheet does not match any product/supplier; fix the spelling or create the record first.
- **"hay más de un producto llamado …" (Excel import):** two catalog records share that name once
  case and accents are ignored, so the row is rejected rather than guessed; rename one of them.
- **"hay una columna de proveedor con datos pero sin nombre" (Excel import):** a supplier group has
  capacity/delivery/price filled in but its `Proveedor` cell is empty; either name the supplier or
  clear the whole group.
- **"El producto N no existe." (server):** the imported row points at a product that is not in the
  company; re-export the file to get current product names.
- **"El producto N está repetido en el envío." (server):** the sheet lists the same product on two
  rows; keep only one.
- **"No se pueden guardar más de 1000 registros de abastecimiento por solicitud." (server):** only
  reachable by calling the API directly, since the page batches at 500.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Purchase Orders (Órdenes de Compra)** at `/logistics/purchase-orders`: create the actual
  order sent to a supplier; this page only plans minimum stock and supplier options, it does not
  place orders.
- **Stock Changes (Cambios Stock)** at `/logistics/products-stock`: the warehouse stock and
  manual movement page whose current-stock snapshot this page reads for the current-vs-minimum
  comparison.
- **Movements Report (Rep. Movimientos)** at `/logistics/warehouse-movements`: the detailed,
  filterable ledger behind the summarized 30-day inflow/outflow chart shown here.
- **Supplies (Suministros)** at `/production/supplies-materials`: manages the separate
  supply-material (`insumo`) catalog, which has its own independent provider/price
  configuration, distinct from the finished-product supply configuration on this page.

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
  - path: frontend/routes/logistics/purchase-management/+page.svelte
    role: page
    hash: sha256:5f0558abbdcd3d3dda7313cdb1442a55fa18f4aa940579f873f273a524046fbb
    supports: [page-purpose]
  - path: frontend/routes/logistics/purchase-management/ProductSupplyManagement.svelte
    role: user-interface
    hash: sha256:b961309172184dad04f68bf156ee606aa865263778021d519a3fbb2b13a44be8
    supports: [page-purpose, concepts, capability.browse-supply, capability.configure-supply, capability.import-export-supply, rules, troubleshooting]
  - path: frontend/routes/logistics/purchase-management/supply-management.svelte.ts
    role: frontend-service
    hash: sha256:69121c4ea2b5a2920cbdab44beaede0351be269b8c24f5a79a4d7e652d9a71bc
    supports: [concepts, capability.browse-supply, capability.configure-supply, capability.import-export-supply, rules]
  - path: frontend/services/production/products.svelte.ts
    role: shared-domain
    hash: sha256:77bb3c75bd2663b000da54b9e84385f92c2a09dcc20b44234899388f51cc49d6
    supports: [concepts, capability.browse-supply, capability.configure-supply]
  - path: frontend/services/crm/client-provider.svelte.ts
    role: shared-domain
    hash: sha256:d42d73f9ef8b3ecd5e7fec9e83cbb2e5cddf8d276c4ac6c37c6babdb1d5081d3
    supports: [concepts, capability.configure-supply]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: data-model
    hash: sha256:5f468bbace2e4a7000cb389e48473da271bdd717bcad8f274ef4f308c3d7ea5e
    supports: [concepts, capability.browse-supply]
  - path: frontend/routes/logistics/purchase-management/supply-management.excel.ts
    role: frontend-service
    hash: sha256:cfea7121f31d890152cd2156d6da91bc2b94545e781cbe6ca544269f17719c9a
    supports: [capability.import-export-supply, troubleshooting]
  - path: backend/logistics/types/product_supply_providers.go
    role: data-model
    hash: sha256:1bdba99849003fb73d341c4fbde6281040e2f1b09a8bb7ebc193abbeafa1ebe1
    supports: [capability.configure-supply, capability.import-export-supply, troubleshooting]
  - path: backend/logistics/product-supply-management.go
    role: backend-handler
    hash: sha256:a269dc49f8473a2e1816e4c31eb9378d4cb070f0c396e9dda35d9e0584955a38
    supports: [capability.browse-supply, capability.configure-supply, capability.import-export-supply, rules, troubleshooting]
  - path: backend/logistics/types/product_supply.go
    role: data-model
    hash: sha256:d6e1c813b7974f7e16027bc804f243914122eb93462acbd4cf2120573df39a8e
    supports: [concepts, capability.configure-supply, rules]
  - path: backend/logistics/types/product-stock-movement.go
    role: data-model
    hash: sha256:11638ab2ed6b9a8698cd4f2da79d6eb4ddb9ce3e2ddce5341615c6b046245d5d
    supports: [concepts, capability.browse-supply]
  - path: backend/logistics/types/product-stock.go
    role: data-model
    hash: sha256:b57cae079f6d72ca3440ec42a586a0ddec6f06788fef8aa8e5674f0ca12d56c2
    supports: [concepts, capability.browse-supply]
  - path: backend/access.toml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:0e4a825ccd2fe08e6a586cfcd24406b715f5e548c9e71d93091421a2d39e8956
    supports: [rules]
```
