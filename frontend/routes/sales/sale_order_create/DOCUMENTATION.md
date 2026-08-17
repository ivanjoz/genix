---
schema: 1
page_id: sales.sale_order_create
route: /sales/sale_order_create
title: Point of Sale (Punto de Venta)
status: implemented
visibility: tenant
description_en: >-
  Point of sale (POS). Search in-stock products by name or serial number, add them to a
  cart with quantity, choose the warehouse and cash account, assign an existing customer
  or register a walk-in one, and generate the sale order with payment and/or delivery
  actions. A separate Settings view edits two sales parameters.
description_es: >-
  Punto de venta (POS). Buscar productos en stock por nombre o número de serie,
  agregarlos al carrito con cantidad, elegir el almacén y la caja, asignar un cliente
  existente o registrar uno nuevo, y generar la orden de venta con las acciones de pago
  y/o entrega. Una vista de Configuración aparte edita dos parámetros de ventas.
---

# Point of Sale (Punto de Venta)

<!-- DOC-ID: page-purpose -->
## Page purpose

Point of Sale (`Punto de Venta`, POS) is where a cashier records an immediate sale (`venta`)
against one warehouse's stock: search products, build a cart, decide whether the sale
collects payment now and/or removes stock now, optionally attach a customer, and generate
the order (`Generar`). Internally the page's own title/tabs read **Ventas** / **Configuración**.

This page only creates new sale orders. It does not own the order's later lifecycle —
confirming a pending payment/delivery afterward, canceling an order, or reviewing sales
history belongs to **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`.
It also does not create products, warehouses, customers, or cash/bank accounts; those are
maintained on their own pages and only consumed here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **sale order (orden de venta)** is created at **Status 1 (Generado/Pendiente)**. Including
  the **Pagado** action moves it toward **Pagado (2)**; including **Recibido** moves it toward
  **Entregado (3)**; including both results in **Pagado + Entregado (4)**. A canceled order is
  **Status 0 (Anulado)**, set only from the related Sales Management page, not from here.
- **Pagado** (`ActionsIncluded` value 2) registers a cash/bank movement (`Cobro (Venta)`)
  against the selected **caja** for the sale total. **Recibido** (`ActionsIncluded` value 3)
  creates outbound warehouse stock movements (`Entrega a cliente final (Venta)`) that reduce
  the selected **almacén**'s stock. These are independent switches: a sale can be generated
  with neither, either, or both.
- A **sub-unit (Sub)** row is an alternate way to sell a product in a smaller unit (for
  example grams cut from a kilogram-priced product) using the product's own `SbuUnit` name
  and its separate `SbuFinalPrice`; it only appears for products configured with a sub-unit
  quantity greater than 1, alongside their normal stock row.
- **Serialized stock (Serie)** is inventory tracked by individual serial number; for those
  products the card shows clickable serial-number chips instead of quick-quantity buttons,
  and each serial is sold and validated independently.

<!-- DOC-ID: capability.search-add-products -->
## Search and add products to the cart (Buscar y agregar productos al carrito)

### User intention (Intención del usuario)

Find an in-stock product quickly by name, brand, presentation, or serial number, and add a
quantity (or a specific serial unit) to the current sale's cart.

### Where to find it (Dónde encontrarlo)

On `/sales/sale_order_create`, **Ventas** view: the **PRODUCTO...** text filter and **Serie...**
filter above the product grid; each product card exposes quick-quantity buttons (desktop:
2, 3, 4, 5, 6, 8, 10, 12; mobile: 1, 2, 5, 10) or serial-number chips. Arrow Up/Down move the
keyboard selection across the filtered grid; **Enter** adds 1 unit of the selected card.

### Required information and prerequisites (Requisitos previos)

A warehouse (**ALMACÉN**) must be selected — the page auto-selects the first warehouse loaded
if none is chosen yet — because the product grid only shows products with stock in that
warehouse. The text filter matches every typed word against the product/brand/presentation
name; the serial filter matches a partial serial number against that product's known serials.

### Business rules and rationale (Reglas y razón de negocio)

Adding to the cart is guarded client-side against the currently loaded stock: requesting more
units than the remaining stock (`stock - already in cart`) is rejected with `No hay suficiente
stock de "<producto>" para agregar <n> unidades.`; requesting one more of an already-added
serial number beyond its detail-row quantity is rejected with `La serie <serie> del producto
"<producto>" sólo posee <n> unidad(es).`. Pressing **Enter** on a serialized card is rejected
with `Seleccione una serie específica.` — bulk-adding an ambiguous serialized product by
keyboard is not supported; a specific serial chip must be clicked. Quick-quantity buttons above
the remaining stock simply do not render for that card.

### Result and side effects (Resultado y efectos)

Adds a new cart line or increases an existing line's quantity (and, for serialized adds, its
per-serial quantity map), then recalculates the cart totals.

### Limitations (Limitaciones)

There is no free-text quantity input on a card — only the fixed quick-quantity buttons, +1 via
Enter, or one unit per serial chip click. The line price is always the product's stored
`FinalPrice` (or `SbuFinalPrice` for a sub-unit row); this page offers no way to discount or
override a line's price.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo busco un producto por número de serie?` Use el campo "Serie...".
- `¿Por qué no me deja agregar más unidades?` No hay suficiente stock disponible en el almacén
  seleccionado para esa cantidad.
- `¿Puedo cambiar el precio de un producto en la venta?` No; el precio viene fijo del producto.
- Search terms: `POS`, `punto de venta`, `carrito`, `buscar producto`, `serie`, `presentación`,
  `sub unidad`.

<!-- DOC-ID: capability.manage-cart -->
## Manage the cart (Gestionar el carrito)

The cart panel lists every added line with quantity, name (and sub-unit/serial detail), and
line total; the trash icon on a line removes it completely, recalculating totals — there is no
way to only decrease a quantity, only remove-and-re-add. **Sub Total** shows `total / 1.18`
(rounded down) and **Total** shows the cart's full amount; both update live as lines change.

<!-- DOC-ID: capability.warehouse-cash-selection -->
## Choose warehouse and cash account (Elegir almacén y caja)

The **ALMACÉN** selector at the top decides which warehouse's stock is shown and is the
`WarehouseID` sent with the order — it also drives which warehouse is debited when **Recibido**
is included. The **CAJA** selector only appears when the company has at least one registered
cash/bank account (`caja o banco`); it is the account credited when **Pagado** is included, and
the page picks the first available caja by default. When no cash/bank account exists yet, the
selector and the **Pagado** action option both disappear and the page shows: `Necesitas
registrar una caja para aceptar pagos.`

<!-- DOC-ID: capability.assign-client -->
## Assign a customer (Asignar un cliente)

### User intention (Intención del usuario)

Attach the sale to an existing customer (`cliente`), register a new walk-in customer inline,
or leave the sale without a customer (anonymous sale).

### Where to find it (Dónde encontrarlo)

The small selector next to the payment/date row: **SIN CLIENTE** (default, no customer),
**Seleccionar Cliente** (search box matching by name or document/registry number), or
**Registrar Cliente** (inline **Documento / RUC** and **Nombre del cliente** fields).

### Required information and prerequisites (Requisitos previos)

Registering inline only requires **Name**; the document/registry number is optional. Existing
customers come from the same catalog as the Customers (`Clientes`) page.

### Business rules and rationale (Reglas y razón de negocio)

The backend resolves a registered walk-in customer by first matching an existing client with
the same registry number, then by a name+registry identity hash; if either matches, the sale
reuses that client's ID **without overwriting any of that client's already-stored fields**
(name, email, etc. are left untouched) — the code explicitly avoids letting sale-order input
corrupt an existing shared client record. Only when neither match is found is a brand-new
client created. Leaving **SIN CLIENTE** selected sends no client at all, so the sale is
recorded without an associated customer.

### Result and side effects (Resultado y efectos)

The resolved (or newly created) client ID is stored on the sale order as `ClientID`.

### Limitations (Limitaciones)

Registering a walk-in customer here only captures name and document number; it cannot set
email, person type, or other fields the Customers page exposes. There is no way to edit an
existing customer's data from this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Puedo vender sin asignar cliente?` Sí, dejando "SIN CLIENTE".
- `¿Qué pasa si registro un cliente con el mismo RUC/documento que uno ya existente?` Genix
  reutiliza el cliente existente en vez de crear uno duplicado.
- Search terms: `cliente`, `registrar cliente`, `venta anónima`, `RUC`, `documento`.

<!-- DOC-ID: capability.set-payment-delivery -->
## Choose Pagado / Recibido and generate the sale (Definir Pagado / Recibido y generar la venta)

### User intention (Intención del usuario)

Decide, at the moment of generating the sale, whether it should immediately register the
payment in a cash/bank account, immediately remove the sold quantities from warehouse stock,
both, or neither — then create the sale order.

### Where to find it (Dónde encontrarlo)

The **Pagado** / **Recibido** checkboxes above the client selector (both checked by default,
though **Pagado** is hidden if no caja exists) and the payment due-date field (**Date Pago**,
optional); the green **Generar** button in the cart header creates the order.

### Required information and prerequisites (Requisitos previos)

The cart must have at least one line and a warehouse must be selected before **Generar** can
succeed (`El carrito está vacío.` / `Seleccione un almacén.`). Including **Pagado** requires a
selected caja (enforced server-side too: `Se requiere LastPaymentCajaID para procesar el
pago.`). Including **Recibido** requires the warehouse and at least one detail line
(`Se requiere WarehouseID para procesar la entrega.` / `No hay productos en el detalle para
procesar la entrega.`).

### Business rules and rationale (Reglas y razón de negocio)

The backend re-validates every cart line's stock against the server's current data (not the
client's cached numbers) before saving, rejecting with a message such as `Almacén: <id> |
Producto: <id> ... Se necesita <n>. Se posee en stock: <m>.` when someone else already
consumed the stock. It also re-checks that detail arrays (products, prices, quantities) are
the same length and contain no zero product/quantity/price value.

### Result and side effects (Resultado y efectos)

Saving always creates a new **Status 1 (Generado)** sale order. Including **Pagado** additionally
books one `Cobro (Venta)` cash/bank movement for the sale total against the chosen caja and
advances Status toward **Pagado**. Including **Recibido** additionally books one outbound stock
movement (`Entrega a cliente final (Venta)`) per cart line — including one per distinct serial
number — against the chosen warehouse, and advances Status toward **Entregado**. Every save also
updates the day-level sales summary consumed by **Sales Charts** and **Sales Report**, and
schedules a background reprocess job. On success the page shows `Venta registrada con éxito`
and clears the cart, the client selection, and the amount-received/change helper fields.

### Limitations (Limitaciones)

- The cart total is always sent with **DebtAmount = 0** (per the frontend's own note:
  *"Assuming fully paid for now, adjust if UI allows debt"*), regardless of whether **Pagado**
  was included. Unchecking **Pagado** only skips the cash/bank movement — it does **not**
  register a pending debt/receivable against the order; this page currently has no way to
  generate a sale on customer credit or with a partial payment.
- Once generated, this page offers no edit, cancel, additional-payment, or delivery action for
  that order; use **Sales Management (Gestión Ventas)** for anything after creation.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Puedo generar una venta sin cobrar ni entregar el producto todavía?` Sí, desmarcando ambas
  casillas; queda "Generada" (Pendiente) sin movimiento de caja ni de stock.
- `¿Por qué no aparece la opción "Pagado"?` Falta registrar una caja/banco.
- `¿Puedo vender a crédito o con pago parcial desde esta página?` No; el monto de deuda que
  registra esta página siempre es 0.
- Search terms: `generar venta`, `pagado`, `recibido`, `entrega`, `caja`, `deuda`, `venta al
  crédito`, `cobro parcial`.

<!-- DOC-ID: capability.configure-sales-parameters -->
## Sales settings (Vista Configuración)

The **Configuración** tab reuses the shared company-parameter editor to store two sales-related
parameters: **Separar proceso de venta** (a multiselect with options **Cobro** and **Entrega de
Producto**) and **Permitir cobro parcial** (a checkbox). Both are saved to the company through
the same access as the rest of this page.

### Limitations (Limitaciones)

Saving these parameters currently has no observable effect on the **Ventas** view: the
"Separar proceso de venta" value is read into a computed flag but that flag is not used anywhere
else in the page, and "Permitir cobro parcial" is not read anywhere in the frontend. Do not tell
users that toggling these settings currently changes how products are sold, how payment/
delivery are separated, or whether partial payment becomes possible — as of this review neither
setting changes anything besides its own stored value.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- This route, and the `sale-order`, `system-parameters`, and `company-parametros` calls it
  makes, are gated by a dedicated **"Punto de Venta"** access entry in the access catalog — it
  is not open to every authenticated user the way some other pages are.
- Status only ever advances: **Pagado** can only be added from Generado(1) or Entregado(3);
  **Entregado** can only be added from Generado(1) or Pagado(2); the combination of both lands
  on Status 4. This page never sets Status 0 (Anulado) — cancellation happens elsewhere.
- Stock is authoritative on the server: the browser's own stock guard while adding to the cart
  prevents most mistakes, but the final, binding check happens again when **Generar** is
  submitted, because stock can change between loading the grid and saving.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El carrito está vacío." / "Seleccione un almacén.":** add at least one product and pick a
  warehouse before pressing **Generar**.
- **"No hay suficiente stock de ... para agregar N unidades." / "La serie ... sólo posee N
  unidad(es).":** the requested quantity exceeds the currently loaded stock; reduce the
  quantity or refresh the stock.
- **"Se necesita N. Se posee en stock: M" (server rejection at Generar):** stock changed on the
  server since the grid was loaded (e.g., another sale or reception happened); reload the page
  or reselect the warehouse and try with the updated stock.
- **"Necesitas registrar una caja para aceptar pagos.":** create a cash/bank account on **Cash
  & Banks** before a sale can include **Pagado**.
- **A generated order shows Debt = 0 even though it wasn't marked Pagado:** expected with the
  current page — it never records a nonzero debt/receivable, regardless of the Pagado/Recibido
  choice.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: review, confirm a
  pending payment or delivery, and cancel (`Anular pedido`) orders created on this page.
- **Sales Charts (Gráficos Ventas)** and **Sales Report (Reporte Ventas)**: consume the
  day-level sale summary that every **Generar** action here updates.
- **Cash & Banks (Cajas y Bancos)**: register the cash/bank account offered in the **CAJA**
  selector, and review the resulting `Cobro (Venta)` movement afterward.
- **Customers (Clientes)** at `/business/customers`: maintain full customer records beyond the
  minimal name/document captured by the inline **Registrar Cliente** flow here.
- **Sites & Warehouses (Sedes y Almacenes)** and **Product Stock**: manage warehouses and
  inspect stock levels before and after a **Recibido** sale.

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
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: page
    hash: sha256:c8650707d5e88cc6cfe386a6b6c240225b8c5ba153f63f4b00be04c92f4cbf65
    supports: [page-purpose, concepts, capability.search-add-products, capability.manage-cart, capability.warehouse-cash-selection, capability.assign-client, capability.set-payment-delivery, capability.configure-sales-parameters]
  - path: frontend/routes/sales/sale_order_create/SaleProductCard.svelte
    role: user-interface
    hash: sha256:c5413f38194021ba8c793d0bf3133044ad0b91c3494be76831654616edeb3f96
    supports: [concepts, capability.search-add-products]
  - path: frontend/routes/sales/sale_order_create/sale_order.svelte.ts
    role: frontend-service
    hash: sha256:2395527ccde4ac57d8c13334fb56f17f9b28c60406b8dabd5de7878a477b0bc6
    supports: [concepts, capability.search-add-products, capability.manage-cart, capability.set-payment-delivery, rules]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: frontend-service
    hash: sha256:5f468bbace2e4a7000cb389e48473da271bdd717bcad8f274ef4f308c3d7ea5e
    supports: [capability.search-add-products, capability.warehouse-cash-selection]
  - path: frontend/routes/business/products/products.svelte.ts
    role: shared-domain
    hash: sha256:77bb3c75bd2663b000da54b9e84385f92c2a09dcc20b44234899388f51cc49d6
    supports: [concepts, capability.search-add-products]
  - path: frontend/routes/business/customers/customers.svelte.ts
    role: shared-domain
    hash: sha256:d42d73f9ef8b3ecd5e7fec9e83cbb2e5cddf8d276c4ac6c37c6babdb1d5081d3
    supports: [capability.assign-client]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:57abecc19b2da874e32d06acbf244f6bcd786470c746551be381ca02070e6887
    supports: [capability.warehouse-cash-selection]
  - path: frontend/services/services/system-parameters.svelte.ts
    role: frontend-service
    hash: sha256:34d33bb25c2ce32df43127d9792c482c889a97d1ceababfaf42eb9c03b16a371
    supports: [capability.configure-sales-parameters]
  - path: frontend/services/system-paremeters.ts
    role: shared-domain
    hash: sha256:4834c646fb7fc71d36370aa8620f42d8e51906fe9ff5dd81494d3a8cb8de6d73
    supports: [capability.configure-sales-parameters]
  - path: frontend/domain-components/SystemParametersEditor.svelte
    role: user-interface
    hash: sha256:c4482b330dfb3eb6b2cc2ab7f6a6adbe20b5d62ba6a989fde3354ca261de6cfc
    supports: [capability.configure-sales-parameters]
  - path: backend/sales/sale_order_create.go
    role: backend-handler
    hash: sha256:03dfbe6e253c0d73c85654993442e2479e33fdb08a44d74e00fd66076643656f
    supports: [capability.assign-client, capability.set-payment-delivery, rules, troubleshooting]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:937666309631867c1693fd6935a17e43f2f68eed0d39577537248dae75fa6cbc
    supports: [concepts, capability.set-payment-delivery, rules]
  - path: backend/business/client_provider.go
    role: business-logic
    hash: sha256:c7ecb92cebcdf753b5a7f27110eb7ab5c941e65709f59db146c529063e434938
    supports: [capability.assign-client]
  - path: backend/finance/cash_bank_movement.go
    role: business-logic
    hash: sha256:dced7592fe610decf9352eb1705b09a46b75a2be73c438cff81d4be7f03a4775
    supports: [capability.set-payment-delivery]
  - path: backend/logistics/product-stock-movement.go
    role: business-logic
    hash: sha256:3f1bcf6a690c536448f47b89173b6ee636aab1150be37a0adfd11819b58a8a2e
    supports: [capability.set-payment-delivery]
  - path: backend/sales/sale_summary.go
    role: business-logic
    hash: sha256:6009c242673ce18ffeff7b157154789b459f4c3b55f97be881aec0fbbc718c5d
    supports: [capability.set-payment-delivery, related-pages]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
```
