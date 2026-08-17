---
schema: 1
page_id: sales.sale_orders_status
route: /sales/sale_orders_status
title: Sales Management (Gestión Ventas)
status: implemented
visibility: tenant
description_en: >-
  Work queue for existing sale orders (pedidos de venta): browse them grouped as Pending
  Payment, Pending Delivery, or Completed, filter by customer or product, and register the
  cash-register payment or the dispatch-warehouse delivery for a selected order.
description_es: >-
  Cola de trabajo para pedidos de venta existentes: consultarlos agrupados en Pendiente de
  Pago, Pendiente de Entrega o Finalizados, filtrar por cliente o producto, y registrar el
  pago (con la caja) o la entrega (con el almacén de despacho) de un pedido seleccionado.
---

# Sales Management (Gestión Ventas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sales Management (`Gestión Ventas`, visible on-page as `Order Management`/`Gestión de
Pedidos`) is the operational queue for sale orders (`pedidos de venta`) that already exist.
It lets a user find an order by its current workflow stage, open it to see its status,
total, debt, and product lines, and push it through the two remaining actions of its
lifecycle: registering the customer's payment (`pago`) and registering the warehouse
delivery (`entrega`).

This page does not create new sale orders — that happens at **Point of Sale (Punto de
Venta)** (`/sales/sale_order_create`). It also does not offer a date-range historical search
across every order regardless of status; that broader lookup belongs to **Sales Report
(Reporte Ventas)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **sale order (pedido de venta)** stores a client, a destination warehouse, product
  lines (product, SKU, presentation, quantity, unit price), the order **Total**, the
  outstanding **Debt (Deuda)**, and a numeric **Status (Estado)**.
- Status values (`ss`) form a small state machine: **1 Generated (Generado)** — created,
  neither paid nor delivered; **2 Paid (Pagado)**; **3 Delivered (Entregado)**; **4
  Completed (Finalizado)** — both paid and delivered. Status **0 (Anulado/annulled)**
  exists in the data model but this page's data source filters those rows out, so an
  annulled order never appears in any of its three tabs.
- The page groups orders into three tabs (`OptionsStrip`), each mapped to a specific
  backend query, not simply to one status value: **Pend. Payment (Pend. Pago)** loads
  orders in status 1 (Generated) or 3 (Delivered but unpaid); **Pend. Delivery (Pend.
  Entrega)** loads orders in status 1 (Generated) or 2 (Paid but undelivered); **Completed
  (Finalizadas)** loads only status 4. Switching tabs re-queries the server; it is not a
  local re-filter of one big list.
- In the results table, two independent letter chips summarize progress per row: **P**
  (green, checked when status is 2 or 4) means paid, **E** (blue, checked when status is 3
  or 4) means delivered. These are separate from the textual **Status** shown inside the
  order detail (`Generado`/`Pagado`/`Entregado`/`Finalizado`).
- **Paid (Pagado)** amount is derived on screen as `Total - Debt`; once an order is marked
  paid through this page its Debt becomes `0`, so Paid equals Total.

<!-- DOC-ID: capability.browse-filter -->
## Find and open an order (Buscar y abrir un pedido)

### User intention (Intención del usuario)

Locate the order that needs attention — one still owing money, one still to hand over to
the customer, or one already closed — and open it to review or act on it.

### Where to find it (Dónde encontrarlo)

Open **Commercial (Comercial) → Sales Management (Gestión Ventas)** at
`/sales/sale_orders_status`. Choose one of the three tabs (**Pend. Pago**, **Pend. Entrega**,
**Finalizadas**) and optionally use the **CLIENTE ::** and **PRODUCTO ::** selectors above
the table. Click a row to open the order in the right-side detail layer.

### Required information and prerequisites (Requisitos previos)

None to browse: the page loads the selected tab automatically on open (`onMount`) and every
time a different tab is chosen. The **CLIENTE**/**PRODUCTO** selectors only list clients and
products that already appear among the orders currently loaded for the active tab — they are
not a global customer/product picker.

### Business rules and rationale (Reglas y razón de negocio)

The client/product filter narrows the already-loaded rows in the browser; it does not issue
a new server search, so it only ever restricts what the current tab already fetched. Each
tab's server query returns up to 5,000 matching orders sorted newest first; a business with
more open orders in one bucket than that will not see the extra rows on this page.

The table's **Top Products (Top Productos)** column aggregates the order's line items by
product, ranks them by line amount (`cantidad × precio`), and shows up to three product
cards with quantity and amount (more are summarized as "(+N more)" on mobile); this is a
per-order summary, not a separate report.

### Result and side effects (Resultado y efectos)

Opening a row only opens the read/action side panel; browsing does not change any order.

### Limitations (Limitaciones)

- No date-range filter exists on this page; every tab always shows its full matching set
  (up to the 5,000-row cap) regardless of when the order was created.
- The client/product filter is local to the tab's loaded data, so switching tabs clears
  the effect of a filter typed while looking at a different bucket.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde veo los pedidos que aún deben pagarme?` En la pestaña "Pend. Pago".
- `¿Dónde veo los pedidos que aún debo entregar?` En la pestaña "Pend. Entrega".
- `¿Por qué un pedido ya pagado sigue en "Pend. Entrega"?` Porque esa pestaña agrupa
  Generado y Pagado; sólo desaparece de ahí al registrar la entrega.
- Search terms: `pedidos de venta`, `gestión de ventas`, `pendiente de pago`, `pendiente de
  entrega`, `finalizadas`, `top productos`.

<!-- DOC-ID: capability.pay -->
## Register a customer payment (Registrar el pago de un pedido)

### User intention (Intención del usuario)

Record that the customer has paid the order and reflect the money entering the selected
cash register (`caja`) or bank account.

### Where to find it (Dónde encontrarlo)

Open the order's detail layer and use the payment panel's **Cash Register for Payment
(Caja para Pago)** selector and the **Pay (Pagar)** button. The panel only appears while the
order's status is 1 (Generated) or 3 (Delivered); once paid, it is replaced by the payment
audit line (date, caja, user).

### Required information and prerequisites (Requisitos previos)

An active cash/bank account (`ss` greater than 0) must exist to appear in the selector; it
defaults to the order's own `LastPaymentCajaID` if already set, otherwise to the first active
account. The order must currently be payable (status 1 or 3); the button is disabled and
dimmed otherwise.

### Business rules and rationale (Reglas y razón de negocio)

This action always registers a **full payment**: the button sends a fixed `DebtAmount: 0`
together with the chosen caja, it does not expose a partial-amount field. The server accepts
the payment only when a cash/bank account ID is present, advances the order's status (1→2 or
3→4), and stamps the acting user and timestamp as the last payment audit.

### Result and side effects (Resultado y efectos)

The order's Debt becomes `0` (so Paid = Total in the detail view) and its status gains the
Paid bit. The selected cash/bank account receives a positive `Cobro (Venta)` movement for
the full order total, visible afterward on **Cash & Banks (Cajas & Bancos)** linked to this
order's number. Because the tabs are queried by status, a paid order that was showing under
**Pend. Pago** stops matching that query and disappears from it once the page refreshes.

### Limitations (Limitaciones)

- There is no way to register a partial payment from this page; the action always settles
  the order's debt to zero in one step.
- Only one action (payment or delivery) can be in progress per order at a time; the panel
  shows a progress message and blocks a second click until the current request finishes.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro el pago de un pedido?`
- `¿Puedo registrar un abono parcial del pedido?` No; el botón "Pagar" siempre salda la
  deuda completa del pedido.
- `¿Dónde queda registrado el ingreso de dinero?` En la caja o banco elegido, como
  movimiento "Cobro (Venta)".
- Search terms: `pagar pedido`, `caja para pago`, `cobro venta`, `deuda pedido`.

<!-- DOC-ID: capability.deliver -->
## Register a warehouse delivery (Registrar la entrega de un pedido)

### User intention (Intención del usuario)

Record that the ordered products left the warehouse and reached the customer, discounting
the corresponding stock.

### Where to find it (Dónde encontrarlo)

Open the order's detail layer and use the delivery panel's **Warehouse for Delivery (Almacén
para Entrega)** selector and the **Deliver (Entregar)** button. The panel only appears while
the order's status is 1 (Generated) or 2 (Paid); once delivered, it is replaced by the
delivery audit line (date, almacén, user).

### Required information and prerequisites (Requisitos previos)

An active warehouse (`ss` greater than 0) must exist to appear in the selector; it defaults
to the order's own `WarehouseID` if already set. The order must have at least one product
line and currently be deliverable (status 1 or 2).

### Business rules and rationale (Reglas y razón de negocio)

The server validates that the chosen warehouse holds enough available stock for every
product line of the order — checked against the plain product stock for lines without a lot
or serial number, or against the matching lot/serial stock detail otherwise — before
advancing the status (1→3 or 2→4). Choosing a warehouse without enough stock is rejected
line by line rather than silently partially fulfilled.

### Result and side effects (Resultado y efectos)

A negative internal stock movement is created for each product line at the chosen
warehouse (visible later on the stock-movement history), the order's status gains the
Delivered bit, and the delivery time/user are stamped. A delivered order showing under
**Pend. Entrega** no longer matches that tab's query and disappears from it once the page
refreshes.

### Limitations (Limitaciones)

- There is no partial delivery here: the action ships every line of the order in one step
  against the single chosen warehouse.
- If the chosen warehouse lacks stock for any line, the whole action is rejected; the user
  must pick a warehouse (or resolve the stock shortage) that can cover every line.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro la entrega de un pedido?`
- `¿Por qué me rechaza la entrega si cambio de almacén?` Porque ese almacén no tiene stock
  suficiente para algún producto del pedido.
- `¿La entrega descuenta stock?` Sí, del almacén elegido, por cada producto del detalle.
- Search terms: `entregar pedido`, `almacén para entrega`, `descuento de stock`, `stock
  insuficiente`.

<!-- DOC-ID: capability.cancel -->
## Cancel button (Botón de anular)

The order detail layer shows a trash-icon **Cancel order (Anular pedido)** button in its
title bar. Clicking it currently only shows a "Cancellation coming soon (Anulación
disponible próximamente)" notice; it does not call the server, does not change the order's
stored status, and does not reverse any payment or stock movement. Treat this control as not
implemented yet rather than as a way to void an order.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo anulo o cancelo un pedido de venta?` Actualmente no es posible desde esta página;
  el botón de anular sólo muestra un aviso y no realiza ningún cambio.
- Search terms: `anular pedido`, `cancelar venta`, `anulación disponible próximamente`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Payment and delivery are independent transitions that can happen in either order; only
  status 4 (Completed) requires both to have already happened.
- Both actions read the order's current record from the server before writing, so pressing
  Pay or Deliver always uses the latest stored total, debt, and product lines rather than a
  possibly stale value the browser had cached.
- Whichever tab an order currently satisfies is purely a function of its stored status
  against that tab's query; there is no separate "hide/show" flag a user sets — the order
  moves tabs automatically as its status changes.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"La orden no posee Caja ID para registrar el pago." / no caja selected:** pick an active
  cash/bank account in the payment panel before pressing Pay.
- **"La orden no posee Almacén ID para registrar la entrega." / no warehouse selected:** pick
  an active warehouse in the delivery panel before pressing Deliver.
- **Delivery rejected with an "Almacén / Producto ... Se necesita X. Se posee en stock: Y"
  message:** the chosen warehouse does not have enough stock for one of the order's product
  lines; choose a warehouse that covers every line or resolve the stock shortage first.
- **"Ya se está procesando una acción para esta orden.":** wait for the in-progress action
  (shown as a loading message) to finish before clicking Pay or Deliver again.
- **An order does not appear in the expected tab:** confirm its current status — Pend. Pago
  and Pend. Entrega both include status 1, so a brand-new order appears in both until it is
  paid and delivered.
- **The Cancel button does nothing:** it is not wired to the server yet; it only displays a
  notice.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Point of Sale (Punto de Venta)** at `/sales/sale_order_create`: creates the sale orders
  that later show up here; this page cannot create a new order.
- **Sales Report (Reporte Ventas)**: use it for a date-range, status, client, or product
  search across sale-order history instead of this page's fixed three tabs.
- **Sales Charts (Gráficos Ventas)**: visual/aggregated view of sales instead of the
  per-order action queue offered here.
- **Cash & Banks (Cajas & Bancos)**: review the `Cobro (Venta)` movement and account balance
  created after paying an order here.
- **Stock Changes (Cambios Stock)**: review the stock movement created after delivering an
  order here.

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
  - path: frontend/routes/sales/sale_orders_status/+page.svelte
    role: page
    hash: sha256:0823d47f681dc029a6e900ee7fe7e1407baea786f37cd21907ec09263510eb40
    supports: [page-purpose, concepts, capability.browse-filter, capability.pay, capability.deliver, capability.cancel, rules, troubleshooting]
  - path: frontend/routes/sales/sale_orders_status/sale_order_status.svelte.ts
    role: frontend-service
    hash: sha256:6bb6df93714c0918e463ed2a369b7808b4e6ba2f059a12d0821b4bbf184f7a1b
    supports: [concepts, capability.browse-filter, capability.pay, capability.deliver, rules]
  - path: frontend/routes/sales/SaleOrdersTable.svelte
    role: user-interface
    hash: sha256:187483386db7e604710e8893046b160bc0d523d81540137b0e3c482733466b1f
    supports: [concepts, capability.browse-filter]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:57abecc19b2da874e32d06acbf244f6bcd786470c746551be381ca02070e6887
    supports: [capability.pay]
  - path: backend/sales/sale_orders_status.go
    role: backend-handler
    hash: sha256:536150f96a4e074412e7e173d943c3d97372b10d731d3cc7861fdd75b016e9fb
    supports: [capability.browse-filter, rules, troubleshooting]
  - path: backend/sales/sale_order_create.go
    role: backend-handler
    hash: sha256:03dfbe6e253c0d73c85654993442e2479e33fdb08a44d74e00fd66076643656f
    supports: [capability.pay, capability.deliver, rules, troubleshooting]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:937666309631867c1693fd6935a17e43f2f68eed0d39577537248dae75fa6cbc
    supports: [concepts, capability.pay, capability.deliver, rules]
  - path: backend/finance/cash_bank_movement.go
    role: business-logic
    hash: sha256:dced7592fe610decf9352eb1705b09a46b75a2be73c438cff81d4be7f03a4775
    supports: [capability.pay]
```
