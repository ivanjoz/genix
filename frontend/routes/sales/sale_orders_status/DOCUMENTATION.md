---
schema: 1
page_id: sales.sale_orders_status
route: /sales/sale_orders_status
title: Sales Management (Gestión Ventas)
status: implemented
visibility: tenant
description_en: >-
  Work queue for existing sale orders (pedidos de venta): browse them grouped as Pending
  Payment, Pending Delivery, Completed or Annulled, filter by customer or product, and
  register the cash-register payment, the dispatch-warehouse delivery, or the annulment of a
  selected order.
description_es: >-
  Cola de trabajo para pedidos de venta existentes: consultarlos agrupados en Pendiente de
  Pago, Pendiente de Entrega, Finalizados o Anuladas, filtrar por cliente o producto, y
  registrar el pago (con la caja), la entrega (con el almacén de despacho) o la anulación de
  un pedido seleccionado.
---

# Sales Management (Gestión Ventas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sales Management (`Gestión Ventas`, visible on-page as `Order Management`/`Gestión de
Pedidos`) is the operational queue for sale orders (`pedidos de venta`) that already exist.
It lets a user find an order by its current workflow stage, open it to see its status,
total, debt, and product lines, and push it through the two remaining actions of its
lifecycle: registering the customer's payment (`pago`) and registering the warehouse
delivery (`entrega`). It is also where an order is annulled (`anulado`), which reverses the
payment and the stock movement it had already produced.

This page does not create new sale orders — that happens at **Point of Sale (Punto de
Venta)** (`/sales/sale_order_create`). It also does not offer a date-range historical search
across every order regardless of status; that broader lookup belongs to **Sales Report
(Reporte Ventas)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **sale order (pedido de venta)** stores a client, a destination warehouse, product
  lines (product, SKU, presentation, quantity, unit price), the order **Total**, the
  outstanding **Debt (Deuda)**, its **currency (moneda)** — soles or dollars, fixed when the sale
  was created at the Point of Sale — and a numeric **Status (Estado)**. Prices, Total and Debt are
  all in the sale's currency.
- Status values (`ss`) form a small state machine: **1 Generated (Generado)** — created,
  neither paid nor delivered; **2 Paid (Pagado)**; **3 Delivered (Entregado)**; **4
  Completed (Finalizado)** — both paid and delivered; **0 Annulled (Anulada)** — voided,
  with whatever it had collected and delivered given back. Annulment is **terminal**: an
  annulled order can never return to another status.
- The page groups orders into four tabs (`OptionsStrip`), each mapped to a specific
  backend query, not simply to one status value: **Pend. Payment (Pend. Pago)** loads
  orders in status 1 (Generated) or 3 (Delivered but unpaid); **Pend. Delivery (Pend.
  Entrega)** loads orders in status 1 (Generated) or 2 (Paid but undelivered); **Completed
  (Finalizadas)** loads only status 4; **Annulled (Anuladas)** loads only status 0.
  Switching tabs re-queries the server; it is not a local re-filter of one big list.
- In the results table, two independent letter chips summarize progress per row: **P**
  (green, checked when status is 2 or 4) means paid, **E** (blue, checked when status is 3
  or 4) means delivered. These are separate from the textual **Status** shown inside the
  order detail (`Generado`/`Pagado`/`Entregado`/`Finalizado`).
- The order detail shows **Total** and **Debt (Deuda)** with the sale's currency symbol (`S/` or
  `US$`), and the product lines' **Precio** and **Subtotal** use the same symbol. Once an order is
  marked paid through this page its Debt becomes `0`.

<!-- DOC-ID: capability.browse-filter -->
## Find and open an order (Buscar y abrir un pedido)

### User intention (Intención del usuario)

Locate the order that needs attention — one still owing money, one still to hand over to
the customer, or one already closed — and open it to review or act on it.

### Where to find it (Dónde encontrarlo)

Open **Commercial (Comercial) → Sales Management (Gestión Ventas)** at
`/sales/sale_orders_status`. Choose one of the four tabs (**Pend. Pago**, **Pend. Entrega**,
**Finalizadas**, **Anuladas**) and optionally use the **CLIENTE ::** and **PRODUCTO ::** selectors above
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
per-order summary, not a separate report. The table's **Total** and **Debt (Deuda)** columns read
in soles by default; only a dollar sale is marked, with a `US$` prefix.

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

The selector only lists **active** cash/bank accounts **of the sale's currency**: a dollar sale
offers only dollar cajas, a soles sale only soles cajas. It defaults to the first account in that
list, or to the caja an older order already stored. The order must currently be payable
(status 1 or 3); the button is disabled and dimmed otherwise.

### Business rules and rationale (Reglas y razón de negocio)

This action always registers a **full payment**: the button sends a fixed `DebtAmount: 0`
together with a caja, it does not expose a partial-amount field. A sale generated unpaid at the
Point of Sale names no caja, so **Pagar** uses the caja picked in the selector. An older unpaid
sale that was saved with a caja keeps it: for those, **Pagar** sends the stored caja and the
selector's choice is ignored. The server accepts the payment only when a cash/bank account ID is
present and that caja is in the sale's currency — the sale's amounts are fixed in its own
currency, so the money must land in a caja of that currency (`La venta está en dólares pero la
caja "<caja>" es en soles.`). It then advances the order's status (1→2 or 3→4) and stamps the
acting user and timestamp as the last payment audit.

### Result and side effects (Resultado y efectos)

The order's Debt becomes `0` and its status gains the Paid bit. The cash/bank account receives a
positive `Cobro (Venta)` movement for the full order total, in the sale's currency, visible
afterward on **Cash & Banks (Cajas & Bancos)** linked to this order's number. Because the tabs are queried by status, a paid order that was showing under
**Pend. Pago** stops matching that query and disappears from it once the page refreshes.

### Limitations (Limitaciones)

- There is no way to register a partial payment from this page; the action always settles
  the order's debt to zero in one step.
- Only one action (payment or delivery) can be in progress per order at a time; the panel
  shows a progress message and blocks a second click until the current request finishes.
- A sale cannot be paid in the other currency: a dollar sale is collected only into a dollar caja
  and a soles sale only into a soles caja. No conversion happens on this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro el pago de un pedido?`
- `¿Por qué no aparece mi caja en "Caja para Pago"?` Porque está inactiva o es de otra moneda que
  la venta.
- `¿Puedo cobrar en soles una venta en dólares?` No; se cobra en una caja de la misma moneda.
- `¿Puedo registrar un abono parcial del pedido?` No; el botón "Pagar" siempre salda la
  deuda completa del pedido.
- `¿Dónde queda registrado el ingreso de dinero?` En la caja o banco elegido, como
  movimiento "Cobro (Venta)".
- Search terms: `pagar pedido`, `caja para pago`, `cobro venta`, `deuda pedido`, `moneda`,
  `dólares`.

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
## Annul an order (Anular un pedido)

### User intention (Intención del usuario)

`Necesito anular una venta que se registró por error`, `el cliente devolvió todo y hay que
deshacer el pedido`, `cómo cancelo un pedido ya pagado y entregado`.

### Where to find it (Dónde encontrarlo)

Open the order from any tab; the detail layer shows a red trash-icon **Anular pedido** button
in its title bar. Pressing it replaces the payment/delivery panels with the annulment form.
The button is only shown to users whose profile holds the **Anular Venta** sub-access of
**Gestión Ventas**; without it the button is absent and the server refuses the operation.

### Required information and prerequisites (Requisitos previos)

- A **reason (motivo)** is mandatory, free text up to 200 characters.
- If the order was paid, a **Caja para la Devolución** must be chosen. It does **not** have to
  be the cash register or bank account that collected the money — pick wherever the refund is
  physically taken from — but it must be active and **in the sale's currency**: the selector only
  lists active cajas of that currency, because the refund gives back the collected amount as it was
  booked. The refund **amount** is not entered by the user: the server computes it from the cash
  movements the order actually produced.
- The order must not already be annulled, and its electronic document (`comprobante`), if it has
  one, must not have been sent to SUNAT yet.

### Business rules and rationale (Reglas y razón de negocio)

- Annulment reverses whatever the order actually did, read from the ledgers rather than
  assumed from its status: money collected is refunded, stock delivered re-enters the exact
  warehouse, lot and serial it left from, and the sale is removed from the day's sales
  summary. An order that was never paid refunds nothing; one that was never delivered moves
  no stock.
- A sale issued under a series carries its electronic document (`comprobante electrónico`) from
  creation. While that document is still pending, annulling the sale voids it and it never goes
  out. Once it has left the pending state (sent to SUNAT), the sale **cannot** be annulled here —
  SUNAT requires a credit note (`nota de crédito`), which this system does not issue yet. The
  error names the series number and correlativo.
- Annulment is terminal and cannot be undone. To reinstate a sale, create a new one.
- Repeating a failed annulment is safe: the server nets what it already gave back, so a retry
  returns only the part that never left.

### Result and side effects (Resultado y efectos)

The order's status becomes **0 (Anulada)** and it moves to the **Anuladas** tab on the next
query. A refund movement (`Devolución (Anulación Venta)`) appears on the chosen cash/bank
account, in the sale's currency, lowering its balance; a `Reingreso (Anulación Venta)` stock movement appears for each
delivered line; and the day's sales summary drops the order's quantities and amounts. The
detail panel then shows when it was annulled, by whom, and the reason.

### Limitations (Limitaciones)

- Partial annulment is not supported: the whole order is voided or none of it.
- The refund is a single movement to one cash/bank account even if the order was collected
  across several.
- Nothing checks that the returned stock physically exists — annulling a delivered order that
  really did leave the premises re-adds stock that is not on the shelf.
- No balance check is performed on the refund account; it can be driven negative.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo anulo o cancelo un pedido de venta?` Ábralo, presione el botón rojo de anular en el
  título, indique el motivo y (si estaba pagado) la caja de la devolución.
- `¿Puedo devolver el dinero desde otra caja?` Sí; la caja de la devolución se elige y no tiene
  que ser la que cobró, pero debe ser de la misma moneda que la venta.
- `¿Por qué no veo el botón de anular?` Su perfil no tiene el sub-acceso `Anular Venta`.
- Search terms: `anular pedido`, `cancelar venta`, `devolución`, `motivo de anulación`,
  `reingreso de stock`, `nota de crédito`.

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
- Annulment is the only transition that removes an order from the payment/delivery workflow
  entirely: an annulled order can no longer be paid or delivered, and its panels are replaced
  by the annulment record.
- Money always moves in the sale's own currency: both the payment and the annulment refund only
  offer active cajas of that currency, and the server refuses a caja of the other one.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"La orden no posee Caja ID para registrar el pago." / no caja selected:** pick an active
  cash/bank account in the payment panel before pressing Pay. If the list is empty, the company
  has no active caja in the sale's currency (for example, no dollar caja for a dollar sale);
  create one on **Cash & Banks** first.
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
- **The annul button is not visible:** the profile lacks the `Anular Venta` sub-access of
  `Gestión Ventas`. An administrator must tick it on the profile — holding the access itself
  is not enough.
- **"La venta tiene un comprobante emitido (serie ... correlativo ...); requiere una nota de
  crédito.":** the order's electronic document was already sent, so it cannot be annulled from
  here.
- **"La venta está en dólares pero la caja "..." es en soles." (or the reverse):** the caja used
  for the payment or the refund is of another currency than the sale. Use a caja of the sale's
  currency. On an older unpaid sale that already stored a caja, **Pagar** uses that stored caja
  rather than the selector, so choosing another caja there does not help.
- **The caja I want is missing from "Caja para Pago" or "Caja para la Devolución":** it is
  inactive or of the other currency; only active cajas of the sale's currency are listed.
- **"Se requiere el motivo de la anulación.":** the reason box is empty or only spaces.
- **"Se requiere la caja de la cual se devolverá el dinero." / "La caja seleccionada está
  inactiva.":** the order was paid, so a refund account is required and it must be active.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Point of Sale (Punto de Venta)** at `/sales/sale_order_create`: creates the sale orders
  that later show up here, and fixes each sale's currency; this page cannot create a new order.
- **Sales Report (Reporte Ventas)**: use it for a date-range, status, client, or product
  search across sale-order history instead of this page's four fixed tabs.
- **Sales Charts (Gráficos Ventas)**: visual/aggregated view of sales instead of the
  per-order action queue offered here.
- **Cash & Banks (Cajas & Bancos)**: review the `Cobro (Venta)` movement and account balance
  created after paying an order here, and the `Devolución (Anulación Venta)` movement created
  after annulling one.
- **Stock Changes (Cambios Stock)**: review the stock movement created after delivering an
  order here, and the `Reingreso (Anulación Venta)` movement created after annulling one.
- **Users & Profiles (Usuarios & Perfiles)**: where the `Anular Venta` sub-access of
  `Gestión Ventas` is granted to a profile.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0232bd261462eee1bcbf2bd20cc37658f85aa3a0a4aa355af975bf5b32c2be2c
    supports: [page-purpose, related-pages]
  - path: frontend/routes/sales/sale_orders_status/+page.svelte
    role: page
    hash: sha256:bd019f577ca6975347e89cd9e30b445b829f644df869eae21555bf527cf7df2f
    supports: [page-purpose, concepts, capability.browse-filter, capability.pay, capability.deliver, capability.cancel, rules, troubleshooting]
  - path: frontend/routes/sales/sale_orders_status/sale_order_status.svelte.ts
    role: frontend-service
    hash: sha256:c21cb2b41d6154e543c2d681ae63b3e20b922795348ab4cd29c721b46a3ae671
    supports: [concepts, capability.browse-filter, capability.pay, capability.deliver, capability.cancel, rules]
  - path: frontend/routes/sales/SaleOrdersTable.svelte
    role: user-interface
    hash: sha256:0945c82b822174b4a6a1e4e72a8bcece13e616441b55b0f377e8b9c0f9d242f0
    supports: [concepts, capability.browse-filter]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.ts
    role: shared-domain
    hash: sha256:55738f078d629f60276bbfbbfb58b46b3b1b7ba7a9626d06504b2e909294bf28
    supports: [concepts, capability.pay, capability.cancel, rules]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:6b8e304283715cf7293c68424b6fd45d8d048828ed1b3d6498ab1e28cf2dcfd3
    supports: [capability.pay, capability.cancel]
  - path: backend/sales/sale_orders_status.go
    role: backend-handler
    hash: sha256:b057b5ba3afbf2646d67e735e4641cfe1f0fbba9c948395ced2387a0ef081f12
    supports: [capability.browse-filter, concepts, rules, troubleshooting]
  - path: backend/sales/sale_order_create.go
    role: backend-handler
    hash: sha256:8901a11e252bc4794f1ab4d2d4e626c5213765121dbc5787566dfb5958dde5f4
    supports: [capability.pay, capability.deliver, rules, troubleshooting]
  - path: backend/sales/sale_order_currency.go
    role: business-logic
    hash: sha256:186747cd3a2892beeef7428ff285df67edc44caa494e2b3b0b2985aa46b92a6f
    supports: [concepts, capability.pay, rules, troubleshooting]
  - path: backend/sales/sale_order_annul.go
    role: backend-handler
    hash: sha256:1ef56162da8be4a98bc0ecd34c4898f039ada07e69a2f731c0739671f4853b4d
    supports: [capability.cancel, rules, troubleshooting]
  - path: backend/invoicing/types/invoice_lookup.go
    role: business-logic
    hash: sha256:6e8c81b134b11362bf751f57f57e954e21b0fdf502acf8b2f3c3e0b60b2ad87b
    supports: [capability.cancel, troubleshooting]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:7db7f29652af90da6fb8c68c6e992a0111162b1aa5492ac2b4564f5dd1895427
    supports: [concepts, capability.pay, capability.deliver, capability.cancel, rules]
  - path: backend/access.toml
    role: business-logic
    hash: sha256:31c7071defebf7ecec268a2ae932e58667d2f0e143dee894d0262d8eab4baa3c
    supports: [capability.cancel, troubleshooting]
  - path: backend/finance/types/cash_movement_apply.go
    role: business-logic
    hash: sha256:6b47d33cfaa81d34eec52d00172ac223307b6a0869dd2ec2b3616af9154b5e5a
    supports: [capability.pay, capability.cancel]
```
