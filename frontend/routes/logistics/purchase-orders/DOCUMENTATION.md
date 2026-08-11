---
schema: 1
page_id: logistics.purchase-orders
route: /logistics/purchase-orders
title: Purchase Orders (Órdenes de Compra)
status: implemented
visibility: tenant
---

# Purchase Orders (Órdenes de Compra)

<!-- DOC-ID: page-purpose -->
## Page purpose

Purchase Orders (`Órdenes de Compra`, commonly `OC`) records what the business intends
to buy from a supplier (`proveedor`), the destination warehouse (`almacén`), expected
dates, product quantities, prices, total, and pending debt (`deuda`). The page supports
creation, review, confirmation, limited editing, supplier payments, cancellation, and
creating a new order from an existing one.

An order is a purchasing commitment and control document. Creating or confirming it does
not by itself add product stock. Merchandise reception (`ingreso de mercadería`) is a
separate lifecycle operation.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Pending (Pendiente):** newly created and still awaiting approval. It can be edited,
  confirmed, or cancelled.
- **Confirmed (Confirmada):** approved for processing. It can receive supplier payments,
  limited header edits, cancellation, and merchandise reception through the corresponding
  receiving workflow.
- **Fulfilled (Completada/Cumplida):** merchandise reception has completed the order. It
  becomes immutable in the current purchase-order actions.
- **Canceled (Cancelada/Anulada):** stopped order. It cannot be edited, confirmed, paid,
  or cancelled again.
- **Total** is the order value; **Debt (Deuda pendiente)** starts equal to total and falls
  with each payment. **Paid (Pagado)** is displayed as `total - debt`.

<!-- DOC-ID: capability.create -->
## Create a purchase order (Crear una orden de compra)

### User intention (Intención del usuario)

Create an `OC` before buying products so the company records the supplier, intended
destination, negotiated quantities and prices, and expected delivery/payment dates.

### Where to find it (Dónde encontrarlo)

Open **Logistics (Logística) → Purchase Orders (Órdenes Compra)** at
`/logistics/purchase-orders`, choose **Orders (Órdenes)**, and use the create action. The
detail layer has **Información** for header data and **Productos** for the cart.

### Required information and prerequisites (Requisitos previos)

- An existing supplier (`proveedor`) is required.
- Select the destination warehouse (`almacén destino`) used by the order.
- Add at least one product with a non-zero quantity and a unit price.
- A product presentation (`presentación`) may be selected when the item uses one.
- Delivery date, payment date, invoice number, and notes provide operational context.

The backend accepts either product or supply-material lines, but this page currently
provides a product-entry interface only; it does not expose an insumos/materiales cart.

### Business rules and rationale (Reglas y razón de negocio)

For the current product interface, total is the sum of `quantity × unit price`. The form
derives an 18% included tax breakdown: subtotal is the integer part of `total / 1.18`, and
IGV is `total - subtotal`. The new order is assigned the current generation date, enters
**Pending (Pendiente)** status, and begins with debt equal to its total.

Product IDs, quantities, prices, and presentations are kept as aligned detail rows. The
server rejects missing products, zero product IDs, zero quantities, or inconsistent line
arrays so one quantity or price cannot accidentally belong to another product.

### Result and side effects (Resultado y efectos)

Genix creates a new pending purchase order and shows its generated order number. It does
not increase stock, confirm the order, or pay the supplier automatically.

### Limitations (Limitaciones)

- This page does not currently add supply-material (`insumo`) lines despite backend data
  support for them.
- Tax uses the fixed 18% included-tax calculation in the form; there is no tax-rate
  selector here.
- Creating an order does not reserve or receive stock.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo genero una OC para un proveedor?`
- `¿Dónde agrego productos, cantidades, presentaciones y precios?`
- `¿Crear la orden aumenta el stock?` No; reception is separate.
- Search terms: `OC`, `orden de compra`, `proveedor`, `almacén destino`, `fecha entrega`,
  `factura`, `IGV`, `deuda`.

<!-- DOC-ID: capability.review-report -->
## Find and review orders (Buscar y consultar órdenes)

Open **Report (Reporte)** to search by start/end generation date, supplier, product, and
status. Results are trimmed to the exact selected dates, sorted newest first, and limited
to 2,000 rows in the browser. Selecting an order opens its supplier, state, dates,
invoice, total, paid amount, notes, and product lines.

The default orders list focuses on pending orders. Use **Reporte** when looking for
confirmed, completed, or canceled history, or when a date/provider/product filter is
needed.

<!-- DOC-ID: capability.confirm-edit -->
## Confirm or edit an order (Confirmar o editar una OC)

### User intention (Intención del usuario)

Confirm an order when it is approved to proceed. Edit operational header information
when dates, destination, invoice reference, or notes change without renegotiating the
commercial detail.

### Where to find it (Dónde encontrarlo)

In **Report (Reporte)**, select an order and open **Acciones → Confirmar** or
**Acciones → Editar**.

### Business rules and rationale (Reglas y razón de negocio)

Only a pending order can be confirmed. Editing is allowed while the order is pending or
confirmed. Edit changes only warehouse, delivery date, payment date, invoice number, and
notes. Supplier, products, quantities, prices, totals, debt, and status are preserved to
prevent an ordinary header edit from rewriting the approved commercial or accounting
content.

Completed and canceled orders are immutable through these actions to preserve the
consistency of historical purchasing, stock, and financial records.

### Result and side effects (Resultado y efectos)

Confirmation changes **Pendiente → Confirmada**. It does not receive inventory or create
a payment. Editing updates only the permitted header fields and retains the current state.

### Limitations (Limitaciones)

To change supplier, product lines, quantities, prices, or totals, create a corrected new
order instead of using **Editar**.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Qué cambia cuando confirmo una orden?`
- `¿Puedo editar productos o proveedor después de crear la OC?` Not with the current edit
  action.
- `Why can’t I edit a completed or canceled purchase order?`

<!-- DOC-ID: capability.pay -->
## Register a supplier payment (Pagar una orden al proveedor)

### User intention (Intención del usuario)

Record a full or partial payment (`abono`) against the order's pending debt and reflect
the money leaving a selected cash or bank account (`caja o banco`).

### Where to find it (Dónde encontrarlo)

In **Report (Reporte)**, select a confirmed order and choose **Acciones → Pagar**. Select
the account and enter the payment amount.

### Required information and prerequisites (Requisitos previos)

The order must be **Confirmed (Confirmada)**, a cash/bank account must exist, and the
payment must be greater than zero and no greater than the remaining debt.

### Business rules and rationale (Reglas y razón de negocio)

Genix allows partial payments. Each valid payment reduces the order debt. Preventing a
payment above the remaining debt avoids showing a negative payable balance.

### Result and side effects (Resultado y efectos)

The selected account receives a negative `Pago Proveedor` movement linked to the order
number, its balance decreases by the payment amount, and the order's debt decreases by
the same amount. The paid amount displayed in the detail becomes `total - remaining
debt`.

### Limitations (Limitaciones)

- Paying does not change the order from Confirmed to Fulfilled; fulfillment belongs to
  merchandise reception.
- The current cash ledger does not enforce a general no-negative-balance rule, so users
  must verify available funds according to company policy.
- Cancellation does not reverse previously registered `Pago Proveedor` movements or
  restore debt. Review payments before canceling a confirmed order.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro un abono parcial al proveedor?`
- `¿Por qué no puedo pagar una OC pendiente?` Confirm it first.
- `¿Dónde se descuenta el pago?` From the selected `caja o banco`.
- Search terms: `pagar OC`, `abono`, `deuda proveedor`, `saldo pendiente`, `Pago Proveedor`.

<!-- DOC-ID: capability.cancel -->
## Cancel an order (Anular o cancelar una OC)

A pending or confirmed order can be canceled from **Report → Acciones → Anular** after a
confirmation prompt. The transition is **Pendiente/Confirmada → Cancelada**. Completed
and already canceled orders are rejected, and this page provides no restore or undo.

Cancellation changes the order status only. It does not reverse cash movements already
created by payments, restore the debt, or remove stock movements. Because confirmed
orders may already have payments, inspect **Pagado** and the linked cash history before
confirming cancellation.

<!-- DOC-ID: capability.copy -->
## Generate a new order from a copy (Generar copia de una OC)

Select an existing order in **Report** and choose **Acciones → Generar Copia**. Genix
opens the **Orders** creation view and prepares a brand-new order with the source
supplier, warehouse, and available product lines, including presentation, quantity, and
price. The original order ID, status, payments, and audit history are not reused.

Products deleted since the original order—or presentations that no longer exist—are
skipped from the copied cart. Review every copied line and complete dates, invoice, and
notes before saving. Saving creates a separate pending order with a new number and newly
calculated totals.

<!-- DOC-ID: capability.receive -->
## Merchandise reception status (Recepción e ingreso de mercadería)

The domain supports receiving a confirmed order into a warehouse. Reception adds stock
movements, changes the order to **Fulfilled (Cumplida/Completada)**, and records signed
quantity/value differences: negative for under-delivery (`faltante`) and positive for
over-delivery (`sobrante`). A mismatch is recorded rather than rejected.

However, this route currently has no visible **Recibir** action wired to that service.
Do not tell users they can complete reception from the Purchase Orders page until a
receiving interface or documented navigation path is implemented.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- State order is operational: create Pending, confirm to Confirmed, receive to Fulfilled;
  cancellation is available only before fulfillment.
- Creation does not affect stock or cash. Payment affects cash and debt. Reception affects
  stock and fulfillment status. Keeping these events separate makes each business event
  auditable.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **“Debe seleccionar un proveedor”:** choose an existing supplier before saving.
- **The order has no valid lines:** add at least one product with non-zero quantity; a
  deleted product or presentation may also have been skipped while copying.
- **Confirm is rejected:** only Pending orders can be confirmed.
- **Pay is rejected:** the order must be Confirmed; select an account and enter an amount
  between `0.01` and the remaining debt.
- **Edit is rejected:** only Pending or Confirmed orders can be edited, and only the
  allowed header fields change.
- **A canceled order still has a cash payment:** cancellation does not reverse financial
  movements; review the account ledger and follow the company's correction procedure.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash & Banks (Cajas & Bancos):** inspect the `Pago Proveedor` movement and resulting
  account balance after paying an order.
- **Suppliers (Proveedores):** create or maintain the supplier before placing an order.
- **Branches/Warehouses (Sedes y Almacenes):** create the destination warehouse used by
  the order and reception.
- Inventory/warehouse receiving is the adjacent workflow for turning a confirmed order
  into stock; the current Purchase Orders route does not expose that user action.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:14cca3289f3e701648257a2a2aff673cfda8477399b05cf3125588b6f631a59d
    supports: [page-purpose, capability.create, related-pages]
  - path: frontend/routes/logistics/purchase-orders/+page.svelte
    role: page
    hash: sha256:93158fccbd594d19d6c45b7d65e262e4ab725a967d508706743b957521e059a7
    supports: [page-purpose, capability.create, capability.review-report]
  - path: frontend/routes/logistics/purchase-orders/PurchaseOrderCreate.svelte
    role: user-interface
    hash: sha256:1be98d963afdb8e485ea32615c307e730683ad2d483dd148e09580ede89315cf
    supports: [capability.create, capability.copy, rules, troubleshooting]
  - path: frontend/routes/logistics/purchase-orders/PurchaseOrderForm.svelte
    role: user-interface
    hash: sha256:3fbbc7f60c5a0c5928078785a8f824f5235adf78f6981cb571f02a715f84a214
    supports: [capability.create, capability.confirm-edit]
  - path: frontend/routes/logistics/purchase-orders/ProductCardSearch.svelte
    role: user-interface
    hash: sha256:ef9b2efa17be68fcde03784bc0918f4a0cf2c6269490e5262ca0a883b225d420
    supports: [capability.create, capability.copy]
  - path: frontend/routes/logistics/purchase-orders/PurchaseOrderReport.svelte
    role: user-interface
    hash: sha256:138b78ae7b80a7e23641e2ac9e0218fb4885b12e55ecf952292e6e59a3125f43
    supports: [capability.review-report, capability.confirm-edit, capability.pay, capability.cancel, capability.copy, capability.receive, troubleshooting]
  - path: frontend/routes/logistics/purchase-orders/purchase_order.svelte.ts
    role: frontend-service
    hash: sha256:6a48386ff77d68cc054874abee00698871233580e5e776d37d492ab118398c7d
    supports: [concepts, capability.create, capability.review-report, capability.confirm-edit, capability.pay, capability.cancel, capability.receive, rules]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:509b38843a5841a6815a66cdd18a47ef093ce11292ffe658d09473c5fabc952d
    supports: [concepts, capability.create, capability.confirm-edit, capability.pay, capability.cancel, capability.receive, rules, troubleshooting]
  - path: backend/logistics/purchase_order.go
    role: backend-handler
    hash: sha256:a83600a74b3b355fe2ce9536a89549c7d115afbe57de8297e5154519c208aa39
    supports: [capability.review-report]
  - path: backend/logistics/product-stock-movement.go
    role: business-logic
    hash: sha256:ba0e863d7d05e10fe7604eaab46cd45b70c1f1273f175f972f08f496d0e328b1
    supports: [capability.receive, rules]
  - path: backend/logistics/types/purchase_order.go
    role: data-model
    hash: sha256:b25095c917dbab906be5199930849f8d8d51cad94f95ac6fb7eaed7b2806aa69
    supports: [concepts, capability.create, capability.confirm-edit, capability.pay, capability.cancel, capability.receive, rules]
  - path: backend/finance/cash_bank_movement.go
    role: business-logic
    hash: sha256:9a7c5fdcd87319cc596f82f6a298b661ce90e16c2b290421c3ab9007a9cf386e
    supports: [capability.pay, rules]
```
