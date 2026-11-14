---
schema: 1
page_id: sales.sale_orders_status
route: /sales/sale_orders_status
title: Sales Management (Gestión Ventas)
status: implemented
visibility: tenant
description_en: >-
  Work queue for existing sale orders (pedidos de venta): browse them grouped as Pending
  Payment, Pending Delivery, Completed, Annulled or To Regularize (delivered made products whose
  stock exit is pending), filter by customer or product, and register the cash-register payment,
  the dispatch-warehouse delivery, or the annulment of a selected order.
description_es: >-
  Cola de trabajo para pedidos de venta existentes: consultarlos agrupados en Pendiente de
  Pago, Pendiente de Entrega, Finalizados, Anuladas o Por regularizar (productos fabricados
  entregados con salida de stock pendiente), filtrar por cliente o producto, y registrar el pago
  (con la caja), la entrega (con el almacén de despacho) o la anulación de un pedido seleccionado.
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
payment and the stock movement it had already produced, and where the delivered sales of made
products whose stock exit is still pending are listed (**Por regularizar**).

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
- The page groups orders into five tabs (`OptionsStrip`), each mapped to a specific
  backend query, not simply to one status value: **Pend. Payment (Pend. Pago)** loads
  orders in status 1 (Generated) or 3 (Delivered but unpaid); **Pend. Delivery (Pend.
  Entrega)** loads orders in status 1 (Generated) or 2 (Paid but undelivered); **Completed
  (Finalizadas)** loads only status 4; **Annulled (Anuladas)** loads only status 0; **To
  regularize (Por regularizar)** loads delivered orders (status 3 or 4) with stock to
  regularize. Switching tabs re-queries the server; it is not a local re-filter of one big list.
- **Stock to regularize (`stock por regularizar`):** a made product (`producto fabricado`,
  Origen `Fabricado` on the Productos page) is delivered without its warehouse exit. Its line is
  marked as pending, and its units stay reserved on the delivery warehouse instead of leaving
  stock. A sale with such a line is "por regularizar" until the daily production close (**Cierre
  del día**, Operaciones → Producción) of the sale's day settles it in the delivery warehouse.
  Merchandise lines are not affected: they leave stock at delivery as usual.
- In the results table, two independent letter chips summarize progress per row: **P**
  (green, checked when status is 2 or 4) means paid, **E** (blue, checked when status is 3
  or 4) means delivered. A yellow **R** next to them marks a sale with stock to regularize. These
  are separate from the textual **Status** shown inside the order detail
  (`Generado`/`Pagado`/`Entregado`/`Finalizado`).
- The order detail shows **Total** and **Debt (Deuda)** with the sale's currency symbol (`S/` or
  `US$`), and the product lines' **Precio** and **Subtotal** use the same symbol. Once an order is
  marked paid through this page its Debt becomes `0`.

<!-- DOC-ID: capability.browse-filter -->
## Find and open an order (Buscar y abrir un pedido)

### User intention (Intención del usuario)

Locate the order that needs attention — one still owing money, one still to hand over to
the customer, or one already closed — and open it to review or act on it.

### Where to find it (Dónde encontrarlo)

Open **Commercial (Comercial) → Sales (Ventas) → Sales Management (Gestión Ventas)** at
`/sales/sale_orders_status`. Choose one of the five tabs (**Pend. Pago**, **Pend. Entrega**,
**Finalizadas**, **Anuladas**, **Por regularizar**) and optionally use the **CLIENTE ::** and **PRODUCTO ::** selectors above
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
- `¿Qué significa la "R" amarilla?` La venta entregó productos fabricados cuyo stock aún no salió
  del almacén; aparece en la pestaña "Por regularizar".
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

The server validates that the chosen warehouse holds enough stock for every merchandise line
of the order — checked against the plain product stock for lines without a lot or serial
number, or against the matching lot/serial stock detail otherwise — before advancing the
status (1→3 or 2→4). Choosing a warehouse without enough stock is rejected line by line rather
than silently partially fulfilled.

Lines of made products (`productos fabricados`) follow a different rule: they are not checked
against physical stock but against what that warehouse can still sell — its stock less the units
already reserved, plus the product's **Sobreventa permitida** — when the sale's reservation is not
already on that warehouse. They may not name a lot or serial number.

### Result and side effects (Resultado y efectos)

A negative internal stock movement is created for each merchandise line at the chosen
warehouse (visible later on the stock-movement history), the order's status gains the
Delivered bit, and the delivery time/user are stamped. A delivered order showing under
**Pend. Entrega** no longer matches that tab's query and disappears from it once the page
refreshes.

Lines of made products get **no** stock movement at delivery: they are marked "stock por
regularizar", their units stay reserved — moved to the chosen warehouse if the sale was reserved
on another one — and the sale gets the yellow **R** and appears under **Por regularizar**.

### Limitations (Limitaciones)

- There is no partial delivery here: the action ships every line of the order in one step
  against the single chosen warehouse.
- If the chosen warehouse lacks stock for any line, the whole action is rejected; the user
  must pick a warehouse (or resolve the stock shortage) that can cover every line.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro la entrega de un pedido?`
- `¿Por qué me rechaza la entrega si cambio de almacén?` Porque ese almacén no tiene stock
  suficiente para algún producto del pedido.
- `¿La entrega descuenta stock?` Sí, del almacén elegido, por cada mercadería del detalle. Los
  productos fabricados no descuentan stock al entregar: quedan reservados y "por regularizar".
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
  no stock. An undelivered order releases the units it reserved; a delivered order with stock to
  regularize releases the reservation of its made products' lines (they never left the
  warehouse, so nothing re-enters) and leaves the **Por regularizar** tab.
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

<!-- DOC-ID: capability.stock-to-regularize -->
## Review sales with stock to regularize (Ventas por regularizar)

### User intention (Intención del usuario)

See which delivered sales of made products still have their stock exit pending, and which lines
of each sale are affected.

### Where to find it (Dónde encontrarlo)

Choose the **To regularize (Por regularizar)** tab. In any tab, a yellow **R** next to the P/E
chips marks such a sale. In the order's detail, each affected product line shows
**· stock por regularizar** after its name.

### Required information and prerequisites (Requisitos previos)

A sale appears here only when it is delivered (status 3 Entregado or 4 Finalizado) and at least
one of its lines is a made product delivered without its warehouse exit.

### Business rules and rationale (Reglas y razón de negocio)

The pending lines keep their units reserved on the delivery warehouse, so those units still count
against what that warehouse can sell (Point of Sale and web store) and against the product's
**Sobreventa permitida**. While a product has sales here, its origin cannot be changed from
`Fabricado` to `Comprado` on the Productos page.

### Result and side effects (Resultado y efectos)

The tab is read-only: it lists the sales and opens the same detail layer, where payment and
annulment work as in the other tabs.

### Limitations (Limitaciones)

There is no action on this page to regularize a sale. Sales are regularized on **Operaciones →
Producción → Cierre del día** (`/production/daily-close`): closing the sale's day for its delivery
warehouse consumes the recipes' supplies, takes the made products' units out of stock, releases
their reservation and removes the sale from this tab and its **R**, without changing its status,
payment or debt. Annulling the sale also removes it.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué mi venta de productos fabricados no descontó stock?` Su salida de almacén queda
  pendiente: la venta aparece en "Por regularizar" hasta el cierre del día.
- `¿Cómo saco una venta de "Por regularizar"?` Cerrando su día en Producción → Cierre del día, en
  el almacén donde se entregó; o anulándola.
- Search terms: `por regularizar`, `stock por regularizar`, `regularizar stock`, `R amarilla`,
  `producto fabricado`, `cierre de producción`, `salida de almacén pendiente`.

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
- **Delivery rejected with "... no tiene stock ni sobreventa permitida suficiente en el almacén
  ...":** a made product's line does not fit what the chosen warehouse can still sell (stock less
  reserved units, plus its Sobreventa permitida). Choose another warehouse or raise the product's
  Sobreventa permitida on the Productos page.
- **"... es un producto fabricado: su venta no puede indicar lote ni serie.":** a made product's
  line names a lot or serial number, which made products do not use.
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
- **Products (Productos)** at `/production/products`: where a product's origin (`Fabricado`) and
  its **Sobreventa permitida** are set, which decide how its lines are delivered here.
- **Daily close (Cierre del día)** at `/production/daily-close`: settles the **Por regularizar**
  sales of a warehouse, day by day.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:4f2d4c16777913fddf196959c4341290e69a12a78323a5b95f8fa5d4738ca34d
    supports: [capability.browse-filter, related-pages]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:cb61a6f79a034e4d468b46c173733b815f0af9007a01d1f3e69b92d358913d60
    supports: [page-purpose, capability.browse-filter, related-pages]
  - path: frontend/routes/sales/sale_orders_status/+page.svelte
    role: page
    hash: sha256:f7b2e9378257c86825b23ec8571a1b23d87f9acf2f44634c5f009a4edadb949e
    supports: [page-purpose, concepts, capability.browse-filter, capability.pay, capability.deliver, capability.cancel, capability.stock-to-regularize, rules, troubleshooting]
  - path: frontend/routes/sales/sale_orders_status/sale_order_status.svelte.ts
    role: frontend-service
    hash: sha256:5aac8d90666e8a9fd9908d3ad55154fe294826902577d02cba6489b3d6d35067
    supports: [concepts, capability.browse-filter, capability.pay, capability.deliver, capability.cancel, capability.stock-to-regularize, rules]
  - path: frontend/routes/sales/SaleOrdersTable.svelte
    role: user-interface
    hash: sha256:89ef978523309136bf2c2723b87cf573d5af1368deb0409d554af1237635524d
    supports: [concepts, capability.browse-filter, capability.stock-to-regularize]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.ts
    role: shared-domain
    hash: sha256:5a5b23729c283c270bb18dfea2b17758572f9405c2fa9a01cbdf96d98d6c0cba
    supports: [concepts, capability.pay, capability.cancel, rules]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:6cc31cb3b2acc93150884c4ceb3290384c8e786857dcc27f21b20e3c6e3d6d34
    supports: [capability.pay, capability.cancel]
  - path: backend/sales/sale_orders_status.go
    role: backend-handler
    hash: sha256:c34311168921683a414fe2a0bbcf5dd450dfa8af8f5e2963186fb4fdd48c0262
    supports: [capability.browse-filter, concepts, capability.stock-to-regularize, rules, troubleshooting]
  - path: backend/sales/sale_order_create.go
    role: backend-handler
    hash: sha256:0601dd000d061f432f4a7dfa0228d673b2debc0a744448d6ddfb78a30978c78d
    supports: [concepts, capability.pay, capability.deliver, capability.stock-to-regularize, rules, troubleshooting]
  - path: backend/sales/types/sale_currency.go
    role: business-logic
    hash: sha256:d94fcf68ff93412a00250b54e2bb4c2c59865472b1f30d86a21e2eefcc6d5f92
    supports: [concepts, capability.pay, capability.cancel, rules, troubleshooting]
  - path: backend/sales/types/sale_actions.go
    role: business-logic
    hash: sha256:ad4c117e0bce8f4c8e88f85e6aa3677090935cf6b6127c944f72bf1601413109
    supports: [capability.pay, capability.deliver]
  - path: backend/sales/types/sale_stock_check.go
    role: business-logic
    hash: sha256:952f769c018817d9ba53b4055d0b890578bc17ce8536156bc65f756ad15563f9
    supports: [capability.deliver, troubleshooting]
  - path: backend/sales/types/sale_products.go
    role: business-logic
    hash: sha256:f8ab3f0e08952b9069dc162484ce6ed84cdd82ffe037833500133ce853124930
    supports: [capability.deliver, capability.stock-to-regularize]
  - path: backend/sales/saleissue/types/new_sale.go
    role: business-logic
    hash: sha256:79480f8435d4182f16c8ebca2b5717be396423cd73a571f5ef1e2a84a229e33e
    supports: [capability.pay]
  - path: backend/sales/sale_order_annul.go
    role: backend-handler
    hash: sha256:5d184227f73c0fc0630fb2357132bf09bc4dd39f2bda73d74d9676b611aa36f7
    supports: [capability.cancel, capability.stock-to-regularize, rules, troubleshooting]
  - path: backend/invoicing/types/invoice_lookup.go
    role: business-logic
    hash: sha256:726b3ad0a54ae459de3a43fa3ad5745b9d14d3ed69d9e4949c666c9d06757752
    supports: [capability.cancel, troubleshooting]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:3814c0a52410ab92d7a5e9ee87ee07f90c74815d15678e7391816183b64598c2
    supports: [concepts, capability.pay, capability.deliver, capability.cancel, capability.stock-to-regularize, rules]
  - path: backend/sales/types/pending_delivery.go
    role: business-logic
    hash: sha256:fdefbd8a318a6cafff7823707c0c42bbf28c5ca1077b9b54dccd150094e009fa
    supports: [concepts, capability.deliver, capability.stock-to-regularize]
  - path: backend/logistics/types/web_sellable.go
    role: business-logic
    hash: sha256:86e47492cdd858a77ec4502047f4c9929d951e875b9cc93bf0fc770f7d5524ab
    supports: [capability.deliver, capability.stock-to-regularize, troubleshooting]
  - path: backend/production/product_origin.go
    role: business-logic
    hash: sha256:a53fd15103a1d58292f4b56bef9ccd8d81986bb32959cdd04d3c8a92c11ebc31
    supports: [capability.stock-to-regularize]
  - path: backend/access.toml
    role: business-logic
    hash: sha256:c19cf85130065dcc0f839796e1e9b4b98a1b78df465ccb20d2b63b59a1a94e59
    supports: [capability.cancel, troubleshooting]
  - path: backend/finance/types/cash_movement_apply.go
    role: business-logic
    hash: sha256:81837d0a968b7cdd3f1bfff5f048395dfc72cea1dd65df3388f542c2b118780c
    supports: [capability.pay, capability.cancel]
  - path: backend/production/daily_close.go
    role: business-logic
    hash: sha256:7ae0b03e927362b3c467d29657303b41f9f0dbc979ad54b92f79807ab9b025a1
    supports: [concepts, capability.stock-to-regularize, related-pages]
```
