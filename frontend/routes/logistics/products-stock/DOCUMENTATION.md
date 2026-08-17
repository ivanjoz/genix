---
schema: 1
page_id: logistics.products-stock
route: /logistics/products-stock
title: Product Stock (Inventario de Productos)
status: implemented
visibility: tenant
description_en: >-
  Warehouse stock management. Movement view: see current stock per product and warehouse
  (simple, batch, and serial-tracked) and set the current quantity as a manual adjustment. PO
  Entry view: receive goods from a Confirmed purchase order into a warehouse, completing it.
description_es: >-
  Gestión de stock en almacenes. Vista Movimiento: ver el stock actual por producto y almacén
  (simple, por lote y por serie) y fijar la cantidad actual como ajuste manual. Vista Ingreso OC:
  recibir mercadería de una orden de compra Confirmada hacia un almacén, completándola.
---

# Product Stock (Inventario de Productos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Product Stock (menu label **Stock Changes / Cambios Stock**; page title **Inventario de
Productos**; access-catalog name **Gestión de Stock**) is where the quantity Genix believes is
physically present in a warehouse (`almacén`) is reviewed and corrected. It has two views
selected from the page's top tabs: **Movement (Movimiento)**, a per-warehouse grid of every
product's stock that lets a user type a new quantity as a manual count/adjustment, and **PO
Entry (Ingreso OC)**, which receives merchandise against an already-Confirmed purchase order and
completes it.

This page does not create or confirm purchase orders — that happens on **Purchase Orders**
(`/logistics/purchase-orders`) — and it does not manage warehouses, products, or presentations
themselves. It only reads and writes the stock quantities that already exist for those records.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Stock buckets:** for one (product, presentation) pair in one warehouse, Genix tracks up to
  three independent quantities that sum into the row's **Total Stock (Stock Total)**: **Simple
  Stock (Stock Simple)** — a free bucket with no lot or serial; **Batch Stock (Stock Loteado)** —
  grouped by a lot code (`Lote`) but without individual serials; and **Serials (Seriales)** —
  individually numbered units, each optionally tied to a lot too.
- **Presentation (`Presentación`)** — a specific packaged variant of a product (for example a box
  size); stock is always tracked per (product, presentation) pair, where presentation `0` means
  "no presentation".
- **Lot (`Lote`):** the user always types a lot code/name. Genix resolves it to an internal lot
  record by hashing (date, supplier, code); the same code typed the same day for the same
  supplier reuses the same lot instead of creating a duplicate. A brand-new code creates a new
  lot automatically — there is no separate "create lot" step.
- **Purchase order status relevant here:** only a **Confirmed (Confirmada)** order can be
  received through this page's PO Entry view. Receiving marks it **Fulfilled
  (Cumplida/Completada)**; **Pending**, already **Fulfilled**, and **Canceled** orders never
  appear in the PO Entry picker.

<!-- DOC-ID: capability.movement-view -->
## Review and adjust warehouse stock (Ver y ajustar el stock por almacén)

### User intention (Intención del usuario)

Check how much of each product Genix currently has in a specific warehouse, and correct it —
after a physical count, a loss, or any adjustment that isn't a sale, purchase, or transfer.

### Where to find it (Dónde encontrarlo)

`/logistics/products-stock`, **Movement (Movimiento)** tab (the default view). Pick a warehouse
in the **ALMACÉN ::** selector; the grid only loads once a warehouse is selected. Use the filter
box to narrow by product name/SKU or by a lot/serial value already shown in the grid, and the
**All Products (Todos los Productos)** checkbox to also list every catalog product/presentation
that has no stock row yet in this warehouse (shown at zero), so an initial count can be entered
for it. Save with the **Save (Guardar)** button, which only appears once a warehouse is chosen.

### Required information and prerequisites (Requisitos previos)

- A warehouse must be selected; nothing else on the page works until then.
- Editing **Simple Stock (Stock Simple)** is done directly on the grid row (numeric cell).
  Editing batch or serial quantities requires opening their side layer first (see
  `capability.batch-serial-detail`).

### Business rules and rationale (Reglas y razón de negocio)

Typing a new number in **Simple Stock** sets that bucket to the typed value; it is not a
delta/movement entry — Genix always stores the field as an absolute new count and computes the
resulting ledger movement (in or out) internally as the difference from the previous value. The
grid shows the change as `previous → new` in red until the record is saved. Saving sends only
rows that were actually changed (`_hasUpdated`) or newly filled in through **All Products**.

### Result and side effects (Resultado y efectos)

Saving posts the changed rows to the stock-adjustment endpoint, which recomputes each affected
product's stock and detail rows and appends one entry per change to the warehouse movement
ledger (used by the Movements Report). A resulting negative stock (Simple or the batch/serial
total) is rejected for the whole request rather than partially applied.

### Limitations (Limitaciones)

- There is no field on this page to enter `SubQuantity` (a secondary quantity unit the data
  model supports); only the main `Quantity` is editable here.
- There is no delete action for a stock row; setting a quantity to `0` is the only way to clear
  it, and the row itself remains as a record at zero.
- The grid has no server-side search — the filter box matches only rows already loaded for the
  selected warehouse (plus any product added through **All Products**).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo hago un ajuste de inventario después de un conteo físico?`
- `¿Por qué no veo un producto en la lista?` Actívalo con **Todos los Productos**, o revisa que
  tenga stock en ese almacén.
- `¿El ajuste registra entrada o salida?` Genix calcula automáticamente si el nuevo valor implica
  entrada o salida según la diferencia con el valor anterior.
- Search terms: `ajuste de stock`, `conteo físico`, `inventario`, `stock simple`, `cambio de
  stock`, `stock negativo`.

<!-- DOC-ID: capability.batch-serial-detail -->
## Manage batch and serial detail (Gestionar detalle por lote y por serie)

### User intention (Intención del usuario)

Break down a product's stock into individually tracked lots (`lotes`) or serial numbers
(`números de serie`) instead of one free-form quantity — for traceability of perishable goods,
warranty items, or regulated products.

### Where to find it (Dónde encontrarlo)

On the **Movement** grid, click the **Serials (Seriales)** or **Batch Stock (Stock Loteado)**
column of a row to open its side layer (a table of that product's existing lot/serial rows plus
one always-present blank row to add a new one). Changes are kept only in the browser until the
page's main **Save (Guardar)** button is used; closing the layer keeps the edits pending, it does
not discard them.

### Required information and prerequisites (Requisitos previos)

- A new serial row requires a **Serial** value; a new batch row requires a **Batch (Lote)**
  code. Typing either auto-defaults that row's quantity to `1` if none was set yet, and a fresh
  blank row is appended automatically so the user can keep entering more without an explicit
  "add row" action.
- Existing (already-saved) rows cannot have their Serial or Batch code edited — only their
  Quantity — the code fields are locked for any row that isn't new.

### Business rules and rationale (Reglas y razón de negocio)

Within one product's layer, Genix rejects a duplicate Serial+Lot combination (for the Serials
panel) or a duplicate Lot code (for the Batch panel) with a warning toast, so the same unit or
lot line cannot be entered twice by mistake. The uniqueness check only looks at rows for that
same product/warehouse/presentation group, not across the whole warehouse.

### Result and side effects (Resultado y efectos)

On save, each pending detail row with content becomes one adjustment item, keyed by
(warehouse, product, presentation, lot, serial); the backend resolves/creates the lot the same
way as in `capability.movement-view` and appends its own line to the warehouse movement ledger.
Zeroing an existing row's quantity retires it (the backend marks it inactive) rather than
deleting the record.

### Limitations (Limitaciones)

- A duplicate Batch+Serial combination anywhere across all edited rows on the whole page (not
  only inside one layer) blocks the entire save with a single warning, so it must be corrected
  before anything is saved.
- No screen shows historical batch/serial movements here; that is the Movements Report's job.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo agrego un lote o número de serie nuevo a un producto?`
- `¿Por qué no puedo cambiar el código de un lote ya guardado?` Solo las filas nuevas permiten
  editar el código; las existentes solo permiten cambiar la cantidad.
- Search terms: `lote`, `número de serie`, `serial`, `trazabilidad`, `stock loteado`.

<!-- DOC-ID: capability.po-entry -->
## Receive a purchase order into stock (Ingreso de mercadería por Órden de Compra)

### User intention (Intención del usuario)

Record what physically arrived from a supplier against a Confirmed purchase order, put it into
a warehouse's stock, and mark that order as fulfilled — the merchandise-reception step that
Purchase Orders itself does not expose.

### Where to find it (Dónde encontrarlo)

`/logistics/products-stock`, **PO Entry (Ingreso OC)** tab. Pick an order in **ÓRDEN DE COMPRA
::** (only Confirmed orders are listed), optionally type a **Lote** to tag the next clicks, then
click product cards on the left to add lines to the entry table on the right. Adjust each line's
quantity, expiry date, and lot inline; click the serial icon on a line to open its serial-number
editor (side layer). Save with the panel's **Save (Guardar)** button.

### Required information and prerequisites (Requisitos previos)

- An order must be selected, and it must be **Confirmed (Confirmada)** — Pending, already
  Fulfilled, and Canceled orders are not offered.
- A destination **Warehouse (Almacén)** is required; it defaults to the order's own warehouse
  but can be changed before saving.
- At least one entry line with a quantity greater than zero (or with at least one serial that
  has a quantity) is required.

### Business rules and rationale (Reglas y razón de negocio)

Clicking a product card adds one unit to a matching existing line (same product, presentation,
and current **Lote** text) or creates a new line; lines are grouped and shown under a lot header
(`LOTE <code>` or `SIN LOTE`). A card for a product/presentation whose ordered quantity has
already been fully entered drops out of the list; cards for lines not present on the order at
all (`ordered <= 0`, e.g. ad-hoc extras) always stay visible. Entering a serial on a line
defaults its quantity to `1`; if the sum of a line's serial quantities exceeds its own quantity
field, the line quantity is raised to match — the serial total is authoritative once serials are
used. On save, a line with serials is exploded into one item per serial (each carrying that
serial's own quantity); a line without serials is sent as a single quantity item. The backend
compares total received vs. ordered per (product, presentation) and records the signed
difference — negative for a shortage (`faltante`), positive for an overage (`sobrante`) — on the
order; a mismatch is recorded, not rejected.

### Result and side effects (Resultado y efectos)

Saving applies the same stock-adjustment mechanism as `capability.movement-view` (creating or
updating Simple/Batch/Serial rows and the movement ledger), tagging every movement with the
purchase order's ID and using the order's provider and per-line price so lot resolution and
valuation stay consistent with the order. The order's status changes to **Fulfilled**, and its
recorded receiving difference (quantity and value) is updated. The local entry table then clears
and the just-saved order disappears from the Confirmed picker (it is no longer Confirmed).

### Limitations (Limitaciones)

- Only orders in status Confirmed can be received here; use **Purchase Orders** to confirm one
  first.
- There is no undo for a submitted entry from this page; correcting a wrong reception means
  using **Movement** to adjust the resulting stock, or the company's own correction procedure.
- If a product referenced by an order line is no longer in the loaded product catalog, its card
  still appears but shows a placeholder name (`Producto-<ID>`) instead of being hidden.
- This entry interface only handles product lines; it does not expose supply-material
  (`insumo`) lines even where the purchase-order data model supports them.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro la recepción de una orden de compra?`
- `¿Por qué no aparece mi OC en la lista?` Solo se listan órdenes en estado Confirmada.
- `¿Qué pasa si llega más o menos cantidad de la pedida?` Genix registra la diferencia
  (sobrante/faltante) en la orden; no bloquea el ingreso.
- Search terms: `ingreso de mercadería`, `recepción de OC`, `recibir orden de compra`, `sobrante`,
  `faltante`, `cumplir OC`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Both views write through the same stock-adjustment mechanism (product/warehouse/presentation
  rows, their lot/serial detail, and the movement ledger); the PO Entry view additionally links
  its movements to the order and updates the order's status and receiving difference.
- The two views require different, page-specific access levels: saving on **Movement** needs the
  "Gestión de Stock" access at its Full level, while saving on **PO Entry** needs the "Órdenes
  Compra" access at its Full level — a user could have one without the other. Viewing stock (both
  tabs, before saving) has no dedicated access mapped, so it is available to any signed-in user
  of the company.
- Both save paths reject the request outright if it would leave any bucket's resulting quantity
  negative, rather than applying part of it.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Debe seleccionar un almacén" (red banner, Movement tab):** pick a warehouse before the grid
  or Save button appear.
- **"No hay registros a actualizar." on Save (Movement):** nothing on the grid was changed since
  it loaded; edit a quantity or add a batch/serial row first.
- **"Hay detalles duplicados (mismo Lote + Serial). Verifica antes de guardar.":** two edited rows
  share the same Batch+Serial combination; fix one before saving.
- **"El serial '...' ya fue ingresado." (PO Entry serial editor):** the same serial text is
  already used by another row in that line's serial list.
- **Stock save rejected as negative:** the new Simple/Batch/Serial quantity would take that
  bucket below zero; re-check the count being entered.
- **"Seleccione una Órden de Compra." / "Seleccione el Almacén destino." / "No hay productos para
  ingresar." (PO Entry Save):** select an order, a destination warehouse, and add at least one
  product line with quantity before saving.
- **"La orden no está en estado Confirmada y no puede recibirse.":** the order changed status
  (e.g. someone else already received or canceled it) between opening the picker and saving;
  refresh and confirm its current state on Purchase Orders.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Purchase Orders (Órdenes de Compra)** at `/logistics/purchase-orders`: create, confirm, pay,
  or cancel an order. This page's PO Entry view only receives an order that is already
  Confirmed there.
- **Movements Report (Rep. Movimientos)** at `/logistics/warehouse-movements`: query the
  historical ledger of every stock movement this page creates, filterable by date, product,
  warehouse, lot, or serial.
- **Sites & Warehouses (Sedes & Almacenes)** at `/business/branches-warehouses`: create the
  warehouse this page operates on.

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
  - path: frontend/routes/logistics/products-stock/+page.svelte
    role: page
    hash: sha256:b93b1f2f277d5a9b4009612ad00f421a076e35616f0979476c9e0d9c7ed1f545
    supports: [page-purpose, capability.movement-view, capability.po-entry]
  - path: frontend/routes/logistics/products-stock/ProductStockMovement.svelte
    role: user-interface
    hash: sha256:748351cbe3111f27d61e83c78e6ddbbe79652e2d2adaf42d71775bec003b06c5
    supports: [concepts, capability.movement-view, capability.batch-serial-detail, rules, troubleshooting]
  - path: frontend/routes/logistics/products-stock/PurchaseOrderEntry.svelte
    role: user-interface
    hash: sha256:b8d974088bd0f8681ce392db8c28510d090b26746dfcbf72424ad0781e749e10
    supports: [concepts, capability.po-entry, rules, troubleshooting]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: frontend-service
    hash: sha256:5f468bbace2e4a7000cb389e48473da271bdd717bcad8f274ef4f308c3d7ea5e
    supports: [concepts, capability.movement-view, capability.batch-serial-detail, rules]
  - path: frontend/routes/logistics/purchase-orders/purchase_order.svelte.ts
    role: frontend-service
    hash: sha256:6a48386ff77d68cc054874abee00698871233580e5e776d37d492ab118398c7d
    supports: [concepts, capability.po-entry, rules]
  - path: frontend/routes/business/products/products.svelte.ts
    role: shared-domain
    hash: sha256:77bb3c75bd2663b000da54b9e84385f92c2a09dcc20b44234899388f51cc49d6
    supports: [concepts, capability.movement-view, capability.po-entry]
  - path: frontend/routes/business/branches-warehouses/branches-warehouses.svelte.ts
    role: shared-domain
    hash: sha256:8f3a3fdc8dc47344ada0fc5f56361026effe8099d44c68c9a6ef54a68e488302
    supports: [concepts, capability.movement-view, capability.po-entry, related-pages]
  - path: frontend/routes/business/customers/customers.svelte.ts
    role: shared-domain
    hash: sha256:d42d73f9ef8b3ecd5e7fec9e83cbb2e5cddf8d276c4ac6c37c6babdb1d5081d3
    supports: [capability.po-entry]
  - path: backend/logistics/product-stock-movement.go
    role: business-logic
    hash: sha256:3f1bcf6a690c536448f47b89173b6ee636aab1150be37a0adfd11819b58a8a2e
    supports: [concepts, capability.movement-view, capability.batch-serial-detail, capability.po-entry, rules, troubleshooting]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:d0c3d59fe6e6f5f47a5a82ef4e0f5fc8fa62f10466dba242af3f5ba39bff65a8
    supports: [capability.po-entry, rules, troubleshooting]
  - path: backend/logistics/types/product-stock.go
    role: data-model
    hash: sha256:b57cae079f6d72ca3440ec42a586a0ddec6f06788fef8aa8e5674f0ca12d56c2
    supports: [concepts, capability.movement-view, capability.batch-serial-detail]
  - path: backend/logistics/types/product-stock-movement.go
    role: data-model
    hash: sha256:11638ab2ed6b9a8698cd4f2da79d6eb4ddb9ce3e2ddce5341615c6b046245d5d
    supports: [concepts, capability.movement-view, capability.po-entry]
  - path: backend/logistics/types/purchase_order.go
    role: data-model
    hash: sha256:b25095c917dbab906be5199930849f8d8d51cad94f95ac6fb7eaed7b2806aa69
    supports: [concepts, capability.po-entry]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
```
