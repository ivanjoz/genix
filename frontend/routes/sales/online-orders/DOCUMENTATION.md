---
schema: 1
page_id: sales.online-orders
route: /sales/online-orders
title: Online Orders (Pedidos Web)
status: implemented
visibility: tenant
description_en: >-
  Back office of the web store orders. Tabs Pending, In progress, Completed and Annulled. Open an
  order to see the customer, delivery and products; register its payment, its payment and delivery,
  or annul it; move it through In preparation, On the way or Ready for pickup, and Delivered.
  Registering payment or delivery creates the sale.
description_es: >-
  Gestión de los pedidos de la tienda web. Pestañas Pendientes, En proceso, Completados y Anulados.
  Abrir un pedido para ver cliente, entrega y productos; registrar el pago, el pago y la entrega, o
  anularlo; avanzar su estado En Preparación, En Camino o Para Recoger, y Entregado. Registrar el pago
  o la entrega crea la venta.
---

# Online Orders (Pedidos Web)

<!-- DOC-ID: page-purpose -->
## Page purpose

Online Orders (`Pedidos Web`) is where the store follows the orders customers place in the company's
web store (`tienda web`, `tienda online`, `e-commerce`): who bought, what, the delivery or pickup, and
how they chose to pay. From here the operator registers the payment and/or the delivery of an order,
which turns it into a sale (`venta`), tracks the shipping progress, or annuls an order that will not
go ahead.

An online order is not a sale yet. The sale, and its boleta when the company issues it
automatically, is created the moment the order is paid or delivered, whichever comes first. From
then on later payments and annulments are done on the sale in **Sales Management (Gestión Ventas)**,
not here. Sales typed at the till are created in **Point of Sale (Punto de Venta)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Tabs:** **Pending (Pendientes)** are orders with no payment and no delivery; **In progress (En
  proceso)** are paid but not delivered, or delivered but not paid; **Completed (Completados)** are paid
  and delivered; **Annulled (Anulados)**.
- **Status (Estado)** of an order: **Pendiente**, **Pagado, por entregar**, **Entregado, por cobrar**,
  **Completado** or **Anulado**.
- **Delivery type:** **Home delivery (Entrega a domicilio)**, with address, reference, province and
  district, or **Store pickup (Recojo en tienda)** at a site (`sede`).
- **Payment method** chosen at checkout: **Card (Tarjeta)**, **Yape**, or **On delivery / in store
  (Contra entrega / pago en tienda)**. The detail's **Payment (Pago)** shows **Pendiente** while money
  is owed, and **Pagado (método)** once collected.
- **Shipping progress** steps: **In preparation (En Preparación)**, **On the way (En Camino)** or, for
  pickup, **Ready for pickup (Para Recoger)**, and **Delivered (Entregado)**. The first two are labels
  only; **Entregado** is the real delivery that takes the goods out of a warehouse.
- An order's units are reserved in a warehouse from checkout until they are delivered or the order is
  annulled.

<!-- DOC-ID: capability.review-orders -->
## Review web orders (Consultar pedidos web)

### User intention (Intención del usuario)

See the new orders to attend, their customer and destination, and what they contain.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Sales (Ventas) → Online Orders (Pedidos Web)** at `/sales/online-orders`.
Choose a tab; the table shows **N°**, **Fecha**, **Cliente**, **Destino**, **Total**, **Pago**,
**Entrega** and **Top Productos** (the three products with the largest amount). Click a row to open
**Pedido N° …**.

### Business rules and rationale (Reglas y razón de negocio)

- The detail shows **Fecha**, **Estado**, **Pago**, **Envío** (shipping charged), **Total** and **Cargo
  Culqi** (the card charge ID), the customer's name, email, phone and DNI, the delivery address or
  pickup store, and each product with **Cant.**, **Precio** and **Subtotal**.
- Newest orders come first. The order number (`N°`) is the one the customer sees.
- A row tinted amber, with **Culqi** and a warning icon in **Pago**, is an order whose card was charged
  but whose sale was not registered (see **Register the payment**).

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde veo los pedidos de mi tienda web? ¿Cómo veo la dirección de entrega del cliente?
- ¿Qué pedidos faltan entregar o cobrar? → tab **En proceso**.

<!-- DOC-ID: capability.settle -->
## Register the payment or the delivery (Registrar pago o entrega)

### User intention (Intención del usuario)

Record that a pending order was paid, or paid and handed over, so it becomes a sale with its money in
an account and, when delivered, its stock out of the warehouse.

### Where to find it (Dónde encontrarlo)

Open a pending order. Choose **Cash/bank for payment (Caja/banco para pago)** and, for a delivery,
**Warehouse for delivery (Almacén para entrega)**. Then press **Register payment (Registrar pago)** or
**Payment and delivery (Pago y entrega)**.

### Required information and prerequisites (Requisitos previos)

- The order must be **Pendiente** and have no sale yet.
- Payment accounts offered: active cash or bank accounts (`cajas y bancos`) in soles. The first one is
  preselected.
- Delivery warehouses offered: active warehouses not marked **Exclude from online sales (Excluir
  para ventas online)**. For a pickup order, a warehouse of the pickup store is preselected. The
  chosen warehouse stays selected for the next orders opened.
- With **Issue receipt automatically for online sales (Emitir comprobante automáticamente en ventas
  online)** active in the company configuration, a boleta series marked **Use in Web Store (Usar en
  Tienda Web)** must exist.

### Business rules and rationale (Reglas y razón de negocio)

- The sale bills exactly what the customer agreed at checkout: the order's prices and shipping,
  not today's catalog prices.
- A payment collects the whole order total; there is no partial payment here.
- The account must be active and in soles, and cannot be the detracciones account.
- A delivery checks that the warehouse has stock (or allowed overselling) for every product.
- With the automatic receipt option the sale is the next boleta of the web store series and the boleta
  is issued; without it the sale has no comprobante.

### Result and side effects (Resultado y efectos)

- Genix creates the sale and shows "Venta … creada." The order shows **Venta …** with the note that
  later payments and annulments are done in **Gestión Ventas**, and moves to its new tab.
- **Registrar pago:** the money enters the chosen account; the order becomes **Pagado, por entregar**
  and its units stay reserved until the delivery.
- **Pago y entrega:** the money enters the account and the products leave the chosen warehouse; the
  order becomes **Completado**.

### Limitations (Limitaciones)

- Once the sale exists, this page no longer registers payments or annuls; use **Gestión Ventas**.
- Only orders in soles are collected here, so dollar accounts are not offered.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo confirmo el pago de un pedido web? ¿Cómo registro un pago por Yape de la tienda?
- ¿Cuándo se emite la boleta del pedido web? ¿Por qué el pedido no generó comprobante?
- Términos: pedido web, pedido online, cobrar pedido, venta web, boleta, Culqi, Yape.

<!-- DOC-ID: capability.culqi-reconciliation -->
## Card charged but sale missing (Pagado en Culqi, venta pendiente)

### User intention (Intención del usuario)

Finish an order whose card payment Culqi approved but whose sale was not registered.

### Where to find it (Dónde encontrarlo)

The order shows the amber banner **Pagado en Culqi, venta pendiente**: "Se cobró la tarjeta pero la
venta no se registró. Use Registrar pago con la cuenta de Culqi." The account field is labeled
**Account (choose Culqi) (Cuenta (elija la de Culqi))**.

### Business rules and rationale (Reglas y razón de negocio)

The account marked **Receives Culqi payments (web store) (Recibe los pagos de Culqi (tienda web))** in
Cash & Banks is preselected, because the money is already there. Press **Registrar pago**.

### Common questions and vocabulary (Preguntas y vocabulario)

- El cliente pagó con tarjeta pero el pedido sigue pendiente, ¿qué hago?

<!-- DOC-ID: capability.fulfillment-steps -->
## Track preparation and delivery (Estado de preparación y entrega)

### User intention (Intención del usuario)

Show how far the shipment has progressed and register the moment the goods are handed over.

### Where to find it (Dónde encontrarlo)

In the order detail, the **Status (Estado)** arrows: **En Preparación**, **En Camino** (or **Para
Recoger**) and **Entregado**. Click a later step.

### Business rules and rationale (Reglas y razón de negocio)

- Steps only move forward; a step can be skipped but not undone. Annulled orders have no steps.
- **En Preparación** and **En Camino / Para Recoger** change only the label: no stock moves and no sale
  is created. They cannot be set once the order is delivered.
- **Entregado** needs **Almacén para entrega** and a confirmation **Registrar Entrega** ("Los productos
  salen del almacén …"). If the order has no sale yet, the sale is created delivered and owing the
  total (**Entregado, por cobrar**); collect it later in **Gestión Ventas**. If the order already has a
  sale (paid), that sale is delivered and the order becomes **Completado**.

### Result and side effects (Resultado y efectos)

On **Entregado** the products leave the selected warehouse and their reservation is released.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo marco un pedido como enviado o en camino? ¿Cómo aviso que está listo para recoger?
- ¿Cuándo se descuenta el stock de un pedido web? → on **Entregado**.

<!-- DOC-ID: capability.annul -->
## Annul an order (Anular un pedido web)

### User intention (Intención del usuario)

Cancel an order that will not be paid or delivered: a declined card, a customer who withdrew, or an
order the store cannot fulfill.

### Where to find it (Dónde encontrarlo)

Open a pending order → **Anular**, write the **Reason (Motivo)**, press **Anular** and confirm **Anular
Pedido**.

### Required information and prerequisites (Requisitos previos)

The order must be **Pendiente** with no sale. A reason of at most 200 characters is required. The user
needs the sub-access **Anular Pedido** of Pedidos Web.

### Result and side effects (Resultado y efectos)

The order moves to **Anulados** with its reason shown as **Motivo de anulación**, and its reserved units
are released. No sale, comprobante, money or stock movement existed, so nothing else is undone.

### Limitations (Limitaciones)

An order that already has a sale is annulled by annulling the sale in **Gestión Ventas**. An annulled
order cannot be reopened.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cancelo un pedido web? ¿Por qué no aparece el botón Anular? → the order already has a sale or
  is not pending.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Seleccione la caja/banco que recibe el pago." / "Seleccione el almacén del que salen los
  productos.":** choose the account or warehouse first.
- **"El pedido N° … ya tiene la venta …: continúe en Gestión Ventas.":** the order is already a sale.
- **"El pedido está en soles pero la caja … no.":** choose an account in soles.
- **"La tienda no tiene una serie de boleta marcada "Usar en Tienda Web".":** mark a boleta series
  **Usar en Tienda Web** in the company configuration.
- **"… no tiene stock ni sobreventa permitida suficiente en el almacén …":** deliver from another
  warehouse or enter the missing stock.
- **"No tiene permiso para anular pedidos web.":** the user lacks the sub-access **Anular Pedido**.
- **"Se requiere un motivo para anular.":** write the reason.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Management (Gestión Ventas):** payments, deliveries and annulments of the sale once the
  order became one.
- **Cash & Banks (Cajas & Bancos):** the accounts that receive the payments, and the account marked to
  receive Culqi payments.
- **Sites & Warehouses (Sedes & Almacenes):** the **Excluir para ventas online** option of each
  warehouse.
- **Configuration (Configuración):** the automatic receipt option and the **Usar en Tienda Web**
  series.
- **Shipping Costs (Costos de Envio):** the delivery rate per district that the web store charges as
  **Envío** at checkout.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.review-orders, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.review-orders]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [capability.annul, troubleshooting]
  - path: frontend/routes/sales/online-orders/+page.svelte
    role: page
    hash: sha256:f1d02386b63f26cced56dafe30621cbc5f8af1a484d0dc45c2f95b133bf71604
    supports: [page-purpose, capability.review-orders, capability.settle, capability.culqi-reconciliation, capability.fulfillment-steps, capability.annul, troubleshooting]
  - path: frontend/routes/sales/online-orders/online-orders.ts
    role: business-logic
    hash: sha256:474df458fdaff60edb8cc042e43eba4a11d5e9df635a3fde54b4330825126d67
    supports: [concepts, capability.review-orders, capability.settle, capability.culqi-reconciliation, capability.fulfillment-steps, capability.annul, troubleshooting]
  - path: frontend/routes/sales/online-orders/online-orders.svelte.ts
    role: frontend-service
    hash: sha256:f05a856a8f5387072560d8c27190545d922290c45cf47ffada5315f663b5742f
    supports: [capability.settle, capability.fulfillment-steps, capability.annul]
  - path: frontend/routes/sales/online-orders/OnlineOrderStatusSteps.svelte
    role: user-interface
    hash: sha256:d321c46bb31931880bdebcb5f05059ffcab7ab9b72a2fcf1c36962225e789372
    supports: [capability.fulfillment-steps]
  - path: backend/sales/online_sale_order.go
    role: backend-handler
    hash: sha256:6d521468f6ba3c7074be0ca54529ba9cc3e7453370483539546febb16b0dedac
    supports: [page-purpose, concepts, capability.settle, capability.fulfillment-steps, capability.annul, troubleshooting]
  - path: backend/sales/types/online_sale_order.go
    role: data-model
    hash: sha256:6d9001c416b08cf9514fbfbdbb9a408cb08a671cf68bc8dd8ca0393a236c08df
    supports: [page-purpose, concepts, capability.settle, capability.culqi-reconciliation]
  - path: backend/sales/saleissue/types/new_sale.go
    role: business-logic
    hash: sha256:cf2f1b0a88260d5ab94c6602749f23f8bbfff19e99fb99d8dfb653d3132edfda
    supports: [capability.settle, capability.fulfillment-steps]
  - path: backend/sales/types/sale_stock_check.go
    role: business-logic
    hash: sha256:952f769c018817d9ba53b4055d0b890578bc17ce8536156bc65f756ad15563f9
    supports: [capability.settle, troubleshooting]
  - path: backend/core/company_flags.toml
    role: shared-domain
    hash: sha256:41281f5a4686ceecc28bc2c95a3c6deb4041e03771e7ad8533d07bb2afaf8799
    supports: [capability.settle, related-pages]
  - path: frontend/routes/company/configuration/InvoiceSeriesTable.svelte
    role: user-interface
    hash: sha256:e31ade2c20c6feab4bdd59ae6a89acc3abb81f89dc58932ebeee85b85ce0ee08
    supports: [capability.settle, troubleshooting, related-pages]
  - path: frontend/routes/finance/cash-banks/CajaForm.svelte
    role: user-interface
    hash: sha256:c28980b23db17eb85759c27c21beda0f3a8836566a35d5d62b05d385e666a940
    supports: [capability.culqi-reconciliation, related-pages]
  - path: frontend/routes/business/branches-warehouses/+page.svelte
    role: user-interface
    hash: sha256:01058b8b8d39cb72c8c49edcd852e1e66363870c646eaeb09d2d92e1830b3876
    supports: [capability.settle, related-pages]
  - path: backend/sales/web_checkout.go
    role: business-logic
    hash: sha256:720da509dfab33c993f1f8282920f4df2ca1964e8d967f92ea150f5e87229600
    supports: [concepts, related-pages]
```
