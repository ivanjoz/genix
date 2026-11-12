# Online Sale Order Plan

Status: **implemented, not yet tested end to end** (2026-11-11). Replaces the model of
`docs/WEB_CHECKOUT_PLAN.md`, where every web checkout inserted a `SaleOrder` right away. Code:
`backend/sales/online_sale_order.go`, `backend/sales/web_checkout.go`,
`frontend/routes/sales/online-orders/`. Implementation notes:

- `SaleOrder` keeps `ShippingAmount` (cb 41) and gains `OnlineSaleOrderID` (cb 43); the dropped
  columns stay in the database, unused.
- The settle lock is `core.ActionOnlineSaleOrderSettle` (10). Access "Pedidos Web" is id 59, with
  sub-access 2 "Anular Pedido".
- `PostSaleOrder`'s creation was not extracted whole: the online sale must keep the checkout's
  prices and add the shipping, while the till reprices from the catalog. The shared pieces are
  `readSaleProducts` and `saleDeliveryMovements`.
- Pedidos Web preselects the Culqi account for an order charged by Culqi without a sale, and a
  warehouse of the pickup store for a pickup order.

## Why

Today a web order *is* a `SaleOrder` from the moment the shopper presses pay. Its ID is the boleta's
correlativo (`MakeSaleOrderID`), so:

- **Every declined card or abandoned deferred order spends a boleta number.** The annulled order
  voids its held document, but the correlativo stays spent: a gap in the series that SUNAT expects to
  be declared through a *comunicación de baja*, which the system does not emit.
- **Unpaid orders are sales.** They show in Gestión Ventas, reports and client week sales.
- **A held document state** (`InvoiceHeld = 7`) and its release logic exist only to park the boleta
  until payment.

## The model

A new table, `online_sale_order`, receives what the storefront sends. A `SaleOrder` is created from
it **only when something real happens: the payment, or the delivery of the goods**, whichever comes
first. That is also when SUNAT requires the comprobante to be issued for goods. To be confirmed with
the accountant.

```
storefront ──► OnlineSaleOrder (ID 1, 2, 3… per company, Status 1 pending)
                   │
                   │ paid (Culqi or back office) or delivered (back office)
                   ▼
               SaleOrder (ID = boleta correlativo, minted now) ◄── OnlineSaleOrder.SaleOrderID
                   │                                               SaleOrder.OnlineSaleOrderID
                   │ every later action (pay, deliver, annul) runs on the SaleOrder
                   ▼
               OnlineSaleOrder.Status/DebtAmount copied in the same transaction
```

- **Never paid and never delivered:** no `SaleOrder`, no correlativo, no document. Annulling it
  touches `online_sale_order` alone.
- **The shopper's number is the OnlineSaleOrder ID** ("Pedido N° 125"). The boleta number is shown
  next to it once the `SaleOrder` exists.

## 1. Table `online_sale_order`

`backend/sales/types/online_sale_order.go`, `OnlineSaleOrder` / `OnlineSaleOrderTable`.

| Column | Type | Notes |
|---|---|---|
| CompanyID | int32 | partition |
| ID | int32 | `Autoincrement(0)`: 1, 2, 3… per company |
| Date | int16 | UnixDay of the checkout |
| Lines | []SaleOrderLine | same struct as SaleOrder; priced by the backend at checkout |
| TotalAmount | int32 | cents, shipping included |
| DebtAmount | int32 | cents still owed |
| ShippingAmount | int32 | cents |
| DeliveryType | int8 | 1 home, 2 pickup |
| PickupSiteID | int32 | pickup only |
| Shipping | SaleOrderShipping | contact + address snapshot (struct already exists) |
| WebCustomerID | int32 | the storefront customer |
| ClientID | int32 | CRM client, set when flag 12 needed the DNI (≥ S/ 700) |
| PaymentMethod | int8 | WebPaymentCard / Yape / Deferred |
| CulqiChargeID | string | written as soon as Culqi approves, before the SaleOrder exists |
| SaleOrderID | int64 | 0 until the SaleOrder is created |
| AnnulReason | string | |
| Status | int8 `ss` | same values as SaleOrder: 0 annulled, 1 pending, 2 paid, 3 delivered, 4 paid + delivered |
| Created, Updated, UpdatedBy | int32 | `upd` on Updated |

Indexes:
- `TypeLocalIndex` on `WebCustomerID`: Mis Pedidos and the per-customer pending cap.
- `TypeView` on `(Status, Updated.Size(31))`: the delta cache of the Pedidos Web page.

## 2. Changes to `SaleOrder`

- **Remove** `DeliveryType`, `PickupSiteID`, `Shipping`, `WebCustomerID` and the `WebCustomerID` index.
  Pre-alpha: no compat. The DB keeps the dropped columns; the ORM leaves them alone.
- **Add** `OnlineSaleOrderID int32`: the link back, read by every SaleOrder write path.
- **Keep** `ShippingAmount`: the boleta bills the shipping line (`sale_order_to_cpe.go`), and the
  SaleOrder is what the boleta is built from.

## 3. Checkout (`PostWebCheckout`)

Validation is unchanged: lines, prices, stock, pickup site / address, flag 11 phone, flag 12 DNI
from S/ 700 (creates the CRM client), the per-customer cap of 3 pending orders (now counted on
`online_sale_order`).

1. Insert the `OnlineSaleOrder` as pending (`DebtAmount = TotalAmount`). No ID minting, no document.
2. **Deferred:** done. Email the Orden de Compra PDF, numbered with the OnlineSaleOrder ID.
3. **Online:** charge Culqi with `metadata.onlineSaleOrderID`.
   - **Declined:** annul the OnlineSaleOrder. Nothing else exists to undo.
   - **Unknown error:** stays pending, as today.
   - **Approved:**
     1. Write `CulqiChargeID` first, in its own update, so the money is traceable even if the next step fails.
     2. Then create the paid SaleOrder (§5).
     3. If that creation fails, the order shows in Pedidos Web as "pagado en Culqi, venta pendiente", and **Registrar pago** finishes it.

## 4. Back office: new page *Pedidos Web*

`frontend/routes/sales/online-orders/` (feature files per CLAUDE.md §5), a new page in `db/pages.toml`
under Ventas, read through a delta-cache `GET online-sale-orders`.

- **Lists `online_sale_order`:** number, date, customer, delivery type, address or pickup store, contact, total, status, and the SaleOrder / boleta once it exists.
- **Actions while `SaleOrderID = 0`:**
  - **Registrar pago:** asks for the cash/bank account. Creates the SaleOrder paid.
  - **Registrar entrega:** asks for the warehouse. Creates the SaleOrder with the delivery action (stock leaves the warehouse) and the debt.
  - **Pago y entrega:** both at once, the usual cash-on-delivery case.
  - **Anular:** annuls the OnlineSaleOrder.
- **Once `SaleOrderID > 0`:** the order is read-only here and links to the sale. Later actions run in Gestión Ventas.
- **Route:** `POST online-sale-order-settle {ID, Actions, CashBankID, WarehouseID}`, plus `POST online-sale-order-annul`.

## 5. Creating the SaleOrder from an OnlineSaleOrder

One function, `createSaleOrderFromOnline(onlineOrder, actions, cashBankID, warehouseID)`, used by the
Culqi success path and by the settle route.

- **One SaleOrder per online order, guaranteed.**
  - Take a lock on the OnlineSaleOrder ID: a new `core.LockAction`, `ActionOnlineSaleOrderSettle`.
  - Re-read it inside the lock: if `SaleOrderID > 0`, return that sale.
  - The lock is taken **before** minting the ID, because a second minting would spend a correlativo for nothing.
- **Mint the ID:**
  - With flag 12: `PrepareDocumentForSale` under the "Usar en Tienda Web" series. The document is created **Pending** (not held), and the emission sweep sends it.
  - Without flag 12: `MakeSaleOrderID(…, 0)`.
- **Build the SaleOrder** with the online order's `Lines`, `TotalAmount`, `ShippingAmount`, `ClientID`, the `OnlineSaleOrderID` and the requested actions (2 pay, 3 deliver).
  - **To verify at implementation:** whether `PostSaleOrder`'s creation (cash movement, stock movement, client week sales) can be extracted into a function both paths call, instead of copying it.
- **One transaction:**
  - insert the SaleOrder and its document;
  - set `OnlineSaleOrder.SaleOrderID`, `Status` and `DebtAmount`.

## 6. Keeping the two statuses tied

**The SaleOrder rules once it exists.** Every SaleOrder write path copies the result onto its
OnlineSaleOrder in the same transaction when `OnlineSaleOrderID > 0`:

- `PostSaleOrder` update actions (payment, delivery): copy `Status`, `DebtAmount`, `Updated`.
- `PostSaleOrderAnnul`: the OnlineSaleOrder is annulled too.

No write ever goes from OnlineSaleOrder to SaleOrder after the link exists. The Pedidos Web
actions refuse an order whose `SaleOrderID > 0`.

## 7. What gets deleted

- `InvoiceHeld` (state 7), `ReleaseHeldDocumentForSaleIn`, the held branch of
  `VoidPendingDocumentForSaleIn`, the `releasedHeldDocument` path in `sale_order_create.go`, and the
  state-7 label in `routes/accounting/invoicing/invoicing.ts`.
- `annulUnpaidWebOrder`'s document voiding. Annulling an unpaid web order no longer has a document.
- The web columns on `SaleOrder` (§2) and their reads in `web_customer_orders.go`, `web_order_pdf.go`.

## 8. Storefront and customer side

- **The checkout response:**
  - carries `OnlineSaleOrderID` and `OrderNumber` (= that ID);
  - carries `SaleOrderID` when it was paid online;
  - `IWebCheckoutResult` follows.
- **Mis Pedidos** reads `online_sale_order` by `WebCustomerID`, and shows the boleta number when `SaleOrderID > 0`.
- **Orden de Compra PDF:** `GET p-web-order-pdf?id=` takes the OnlineSaleOrder ID.

## 9. Data

Pre-alpha: delete the web orders already in `sale_order` (`web_customer_id > 0`) and their
documents, or reseed. No migration code.

## Open points

- The accountant confirms the SUNAT timing (comprobante at delivery or payment, whichever first).
- Partial payments of an online order: out of scope. Registrar pago settles the full debt, as the
  checkout does.
