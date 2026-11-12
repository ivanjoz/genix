# Web Checkout Plan

Status: **implemented, not yet tested end to end** (2026-11-11). **The order model changed after
this plan:** a checkout now inserts an `OnlineSaleOrder`, and the `SaleOrder` (with its boleta) is
created only when the order is paid or delivered — see `docs/ONLINE_SALE_ORDER_PLAN.md`, which
overrides everything below about held boletas (state 7, removed), web columns on `SaleOrder`, and
order numbers. Other deviations from the text below:

- The shopper's draft lives in `WebStore.draft` (`frontend/webpage/services/web-store.svelte.ts`,
  persisted to localStorage), not in `Ecommerce.customer`. A guest's "Nueva dirección" is not kept in
  the browser: the checkout sends it with the order and the backend saves it to the new account.
- `PublishCompanyPub` lives in `backend/cloud/company_pub.go` (not `config/types`).
- The online payment choice is explicit (Tarjeta / Yape cards); the Culqi widget shows only the
  picked method. A 3DS `REVIEW` answer is treated as a decline.
- Shipping rates moved to new cent columns `FlatAmount` / `AmountPerKg`; run the exec
  `fn-shipping-costs-to-cents` once after deploy to copy the old soles values.
- The VPS HAProxy preflight (`scripts/configure/configure_server.py`) now allows the
  `X-Customer-Token` header; reconfigure the server for it to take effect.

## Why

The storefront's cart (`frontend/webpage/components/CartMenu.svelte`) never reaches the backend.
`CulqiCheckout.svelte` gets a card token and the UI goes straight to "¡Gracias por tu compra!".
Nobody charges that token, no sale order is created, and the shipping form (`userForm`) is thrown away.

This plan covers:

1. "Entrega a Domicilio" / "Recojo en Tienda" cards on the *Datos Envío* step.
2. A Sede flag, "Punto de recojo de compras online", and a public list of pickup stores.
3. A public checkout endpoint that charges Culqi server-side and creates the sale order, with its
   delivery data.
4. A customer account in the *Mi Usuario* layer: personal data, saved addresses and a payment-methods
   placeholder. The account and the checkout share the same data, so the shopper never types it twice.

## Decided

- **Site column:** `IsPickupPoint bool` on `business/types.Site` / `SiteTable` (`cb:"15"`). It's shown in
  the Sede modal as a checkbox labelled "Online purchase pickup point|Punto de recojo de compras online".
- **Public store list:** stored inside the existing public company file `empresas/e-{companyID}.json`.
  `CompanyPub` gets `PickupSites []CompanyPubSite` with `{ID, Name, Address, CityID}`.
- **Pickup form:** shows Nombres, Apellidos, Correo, Celular and a store picker (one card per pickup Sede).
  Departamento/Provincia/Distrito, Dirección and Referencia are hidden.
- **Home delivery form:** an address selector (see §5). The address inputs appear only when
  "Nueva dirección" is selected.
- The delivery type and the chosen store are **persisted on the sale order**.
- **Customer data lives on the backend,** not in the browser. The shopper is registered automatically
  on their first purchase (§5).
- **Medios de Pago** is a "Próximamente" placeholder in this version.
- **Shipping is charged** (old Q1). Home delivery adds a `sales.ShippingCost` to the charge. Pickup
  adds nothing.
  - **Stored as its own amount** (old Q17): `ShippingAmount int32` (cents) on `SaleOrder` (`cb:"41"`),
    included in `TotalAmount`. A boleta (flag 12) lists it as an extra "Envío" item built from this
    column. It's not a product line.
  - **Rate lookup falls back up the ubigeo tree** (old Q17): the district's rate, else its province's
    (`ParentID`), else its department's. If none of the three has one, home delivery isn't offered
    for that district ("No hacemos envíos a este distrito"), and the backend refuses it too. The
    *Costos de Delivery* page already edits all three levels. A rate of `0` counts as "not set", as
    on that page, so it falls back too (free shipping isn't expressible in this version).
  - **`FlatCost` only** (old Q18): products have no weight, so `CostPerKg` isn't applied yet.
  - The cart shows the shipping amount and the total with shipping as soon as the district is chosen.
- **Shipping costs move to `int32` cents** (old Q18): `ShippingCost.FlatCost` and `.CostPerKg` change
  from `float64` soles to `int32` cents (cents per kg for `CostPerKg`). The existing rows are
  **migrated** (`round(value * 100)`), not reseeded. The *Costos de Delivery* page keeps showing and
  accepting soles with 2 decimals and converts on load/save. The demo seed
  (`generate_demo_data.go`) is updated to cents.
- **Stock comes from every warehouse** (old Q2), except those marked **"Excluir para ventas online"**:
  a new `ExcludeFromOnlineSales bool` on `business/types.Warehouse` (`cb:"15"`), a checkbox in the
  Almacén form. The stock check sums every non-excluded active warehouse. Stock is checked but not
  moved; delivery (action 3) stays a manual step in the back office.
- **The Culqi cash bank is set on the bank account** (old Q3): a new `ReceivesCulqiPayments bool` on
  `finance/types.CashBank` (`cb:"18"`), a checkbox in the Caja/Banco form. At most one per company:
  saving a second one fails with an error that names the account that already has it. Checkout
  refuses to charge if no account has it.
- **Live vs test mode is a company toggle** (old Q4): `IsLive bool` on `CulqiConfig` (`cb:"7"`), next
  to the keys in the company form. `PublishCompanyPub` publishes `PubKeyLive` or `PubKeyDev` to match,
  and the charge uses `KeyLive` or `KeyDev` the same way.
- **Automatic boleta is a company toggle** (old Q6): company flag `12` "Issue receipt automatically
  for online sales|Emitir comprobante automáticamente en ventas online" in `[ecommerce]`.
  - **The series is marked on the series itself** (old Q19): `UseInWebStore int8` on
    `invoicing/types.InvoiceSeries` (`cb:"7"`), a "Usar en Tienda Web" check in the company's series
    list. At most one boleta series per company has it. Saving the company with flag 12 on and no such
    series fails with a descriptive error.
  - Facturas (RUC) are out of scope; boletas only.
  - **DNI above S/ 700** (old Q19): when flag 12 is on and the total (shipping included) is over
    S/ 700, *Datos Envío* shows a required "DNI" input. It's saved on `WebCustomer.DocumentNumber` and
    `SaleOrderShipping.DocumentNumber`, and the backend enforces it.
  - **Every web order is minted under the web series when flag 12 is on** (old Q21/Q22). A sale's ID
    *is* its comprobante number: `MakeSaleOrderID` packs the series into it when the sale is created,
    and it can't change afterwards. So with flag 12 on, every web order (online or deferred payment)
    takes its ID from the `UseInWebStore` series at insert (step 4 of §3). Its boleta document is
    saved in the same transaction but **held unsent**.
    - Online payment: the document is released for emission when the charge succeeds.
    - Deferred payment: **registering the payment in the ERP releases it.**
    - An order annulled unpaid (failed charge, never collected) leaves a reserved, unsent number: a
      gap in the series, which the till already accepts.
    - With flag 12 off, web orders use the series-0 counter and never get a boleta.
  - **Deferred payment (cash on delivery, pay at the store) also gets an "Orden de Compra"** (old
    Q23): a **PDF** emailed at checkout with order number, items, shipping, total, delivery type, and
    the pickup store or address. The customer shows it to collect the goods. It's also downloadable
    from *Confirmación* and *Mis Pedidos*. It's generated in Go with
    [`gpdf`](https://github.com/gpdf-dev/gpdf) (new dependency, decided). An online-paid order's
    customer gets the boleta instead.
- **Yape is a company toggle; PagoEfectivo is out** (old Q7): `YapeEnabled bool` on `CulqiConfig`
  (`cb:"8"`), published in `e-{companyID}.json`. When it's on, the *Pago* step offers Yape next to the
  card. Its exact flow (token vs Culqi order + webhook) gets verified against Culqi's docs at
  implementation time (old Q20). If it needs the webhook, the order stays pending (`Status 1`) until
  Culqi confirms it, never marked paid before.
- **Cash on delivery is a company toggle** (old Q16): company flag `13` "Include cash on delivery in
  the web store|Incluir pago contra entrega en tienda web" in `[ecommerce]`. When it's on, the *Pago*
  step offers "Pago contra entrega" next to the card (and Yape). Choosing it posts to `p-web-checkout`
  **without a Culqi token**. The order is created pending (`Status 1`, `DebtAmount = TotalAmount`),
  nothing is charged, and *Confirmación* shows the order number. The payment is registered later in
  the back office. The backend refuses a tokenless checkout when flag 13 is off.
  - **It applies to pickup too** (old Q21): labelled "Pago contra entrega" on home delivery and
    "Pago en tienda" on "Recojo en Tienda".
  - Email confirmation follows flag `14` (§5, *Email confirmation at checkout*).

## 1. Publishing the company file from two places

Today only `PostCompany` (`backend/config/empresas.go:213`) writes `e-{id}.json`. Now saving a Sede must
rewrite it too, and `business` may not import the `config` module body.

- Move the build-and-upload code to `config/types`: `PublishCompanyPub(companyID int32) error`. It reads
  the company row and the active `IsPickupPoint` sites, then calls `cloud.SaveFile`.
- `PostCompany` and `PostSite` both call it. `PostSite` calls it only when `IsPickupPoint` was or is
  true, or when Name/Address/CityID/Status changed on a pickup site.
- This requires `config/types` to import `business/types`. Types-to-types imports are allowed by
  `MODULE_BOUNDARIES.md`; a cycle has to be checked during implementation.

## 2. Sale order delivery columns

New fields on `sales/types.SaleOrder` and `SaleOrderTable`:

```go
// DeliveryType is how the goods reach the client: 0 = handed over at the till (every
// non-web sale), 1 = home delivery, 2 = pickup at a store.
DeliveryType int8 `cb:"38"`
// PickupSiteID is the Sede the client picks the order up from. Set only when DeliveryType = 2.
PickupSiteID int32 `cb:"39"`
// Shipping is the contact and address the web checkout typed. Nil on till sales.
Shipping *SaleOrderShipping `cb:"40"`
// ShippingAmount is the home-delivery charge in the sale's currency (cents), already included in
// TotalAmount. 0 on pickup and on till sales.
ShippingAmount int32 `cb:"41"`
// WebCustomerID is the storefront customer who placed the order. 0 on till sales.
WebCustomerID int32 `cb:"42"`
```

New index on `SaleOrderTable`: `(WebCustomerID, Date)`. It serves *Mis Pedidos* (§5).

```go
type SaleOrderShipping struct {
    FirstName      string `cb:"1"`
    LastName       string `cb:"2"`
    Email          string `cb:"3"`
    CityID         int32  `cb:"4"` // district; 0 on pickup
    Address        string `cb:"5"` // empty on pickup
    Reference      string `cb:"6"`
    Phone          string `cb:"7"` // required when flag 11 "Requerir número de contacto" is on
    DocumentNumber string `cb:"8"` // DNI; required with flag 12 on and a total over S/ 700
}
```

`Shipping` is a single colbin blob, like `Lines`. It isn't indexed. Nothing filters on it yet.

## 3. Public endpoint `POST.p-web-checkout?cid={companyID}`

Lives in `backend/sales/web_checkout.go`.

Request:

It requires a web customer session (§5). The customer comes from the token, never from the body.

```go
type WebCheckoutRequest struct {
    Lines        []WebCheckoutLine // {ProductID, PresentationID, Quantity}
    DeliveryType int8              // 1 = home, 2 = pickup
    PickupSiteID int32
    // PaymentMethod: 1 = card, 2 = Yape, 3 = deferred (cash on delivery / pay at the store).
    PaymentMethod int8
    // CulqiToken is required for card (and Yape, if it turns out to be token-based); empty for deferred.
    CulqiToken string
    // AddressID picks a saved address for home delivery. 0 means "Nueva dirección": the address
    // fields in Shipping are used and saved to the customer's addresses.
    AddressID int32
    // Shipping carries the contact (names, email, phone, DNI) always, and the address only when
    // AddressID is 0. With a saved address, the server copies that address into the order itself.
    Shipping types.SaleOrderShipping
}
```

Response: `{ SaleOrderID int64, TotalAmount int32, ShippingAmount int32, DocumentType int8 }`.
`DocumentType` is the boleta, or the *Orden de Compra* for deferred orders.

Flow:

1. **Validate the request.** Check the company, the lines and the names/email. For pickup, the site must
   exist, be active and have `IsPickupPoint`. For home delivery, `CityID` must be a district and the
   address must be at least 4 characters. A saved `AddressID` must belong to the session's customer.
   Phone is required with flag 11, and the DNI with flag 12 and a total over S/ 700. `PaymentMethod` 3
   needs flag 13, and 2 needs `YapeEnabled`.
2. **Price from the catalog.** Reuse `validateSaleOrderLines`. Prices are never taken from the request.
   Home delivery adds `ShippingAmount` from the district → province → department `FlatCost`, also read
   on the server.
3. **Check stock** summed over every active warehouse that isn't `ExcludeFromOnlineSales`.
4. **Insert the order as pending** (`Status 1`, `DebtAmount = TotalAmount`). With flag 12 on, the ID
   comes from the `UseInWebStore` series and its boleta document is saved in the same transaction,
   held unsent. With flag 12 off, it comes from the series-0 counter.
   A new address is saved to the customer here too.
   **Deferred payment** stops here: the order stays pending, the *Orden de Compra* PDF is emailed, and
   the order is returned. `PostSaleOrder`'s payment action (2) releases a held document when it
   settles the debt.
5. **Charge Culqi** with `POST https://api.culqi.com/v2/charges` and the secret key for the company's
   mode (`IsLive`) (`amount, currency_code: PEN, email, source_id: token, metadata: {saleOrderID}`).
6. **On success,** register the payment in the same way as `PostSaleOrder`'s action 2: a cash movement
   into the cash bank with `ReceivesCulqiPayments`. The order moves to `Status 2`.
   **On failure,** annul the order with reason `"Culqi: <user_message>"` and return the error to the
   storefront.
7. **If the order carries a boleta,** release its held document to the emission sweep
   (`ScheduleEmitPendingSweep`). A failed emission doesn't undo the paid order: it's logged and retried
   like any till document. The confirmation email is sent either way.

Inserting before charging means no refund path is needed. A failed charge leaves an annulled order, which
keeps an audit trail. A crash between steps 5 and 6 leaves a pending order whose `metadata.saleOrderID`
appears in the Culqi dashboard, so it can be reconciled by hand.

Payment and order-creation logic shared with `PostSaleOrder` gets extracted into functions that take
`companyID, userID`, not `req.User`. A web order is written with `UpdatedBy = 0` (system).

**Abuse:** the route has no back-office login, but it requires a web customer session. Online orders
are gated by the Culqi token, which can be charged only once. Deferred orders carry no token. What
gates them is flag 14 (email confirmation) plus a per-IP and per-customer limit on pending orders,
through fareward. That limit is part of this plan, not a follow-up.

## 4. Frontend

- `CartMenu.svelte`, *Datos Envío* step: two selectable cards, "Entrega a Domicilio"
  (`icon-[fa--truck]`) and "Recojo en Tienda" (`icon-[fa--building]`). The rest of the form switches
  on `deliveryType`.
- Pickup store cards come from `Env.empresa.PickupSites`. The city name comes from `CiudadesSelector`'s
  dataset.
- *Pago* step: `CulqiCheckout.onSuccess(token)` posts to `p-web-checkout`. Only a success response
  moves to *Confirmación*, which shows the order number. An error is shown and the step stays.
- The checkout logic goes in `frontend/webpage/components/web-checkout.ts`: request building and
  validation, pure and testable.
- *Datos Envío* no longer owns a local `userForm`. It reads and writes the shared customer state
  described in §5.

## 5. Customer account in *Mi Usuario*

### Layer layout

- `UsuarioMenu.svelte`'s layer gets the same width as the cart's: `w-768 max-w-[82vw]`.
- **The storefront stops reading the back-office session** (`security.getUserInfo()`). *Mi Usuario*
  shows only the web customer's own session (`WebCustomer`). An admin logged into the back office is a
  guest on the storefront.
- **Guest (no customer session).** Top tabs: **Login**, **Regístrate**, Mis Favoritos, Config.
  - *Login* shows only the login form.
  - *Regístrate* shows only the registration form.
- **Logged-in customer.** Top tabs: **Mi Cuenta**, Mis Pedidos, Mis Favoritos, Config. *Mi Cuenta*
  has a **vertical submenu** on the left, and the selected section is shown on the right:
  1. **Mis Datos**
  2. **Mis Direcciones**
  3. **Medios de Pago**: a "Próximamente" placeholder.

  "Cerrar Sesión" ends the customer session.

### Login

- Email + password.
- Plus a button to log in **with only the email**, through a **6-digit one-time code** sent by email
  and typed in place. It's a code, not a link, everywhere (login, registration, checkout): a link would
  open a new tab or the phone's mail app, away from the waiting cart.

### Regístrate / Mis Datos

- Fields: Nombres, Apellidos, Correo Electrónico, **Celular**. **No password at registration.** A
  password is optional, set later from *Config* → "Cambiar Contraseña". Until then the customer logs
  in with the code.
- **Celular** is optional unless the company checks **"Requerir número de contacto"** (company flag,
  below), and then it's required. The rule applies to *Regístrate*, *Mis Datos* and *Datos Envío* (both
  delivery types), and the backend enforces it too.
- These are **the same four fields** as on the cart's *Datos Envío* step, and both forms are bound to
  one shared object, `Ecommerce.customer` in `frontend/webpage/stores/globals.svelte.ts`. Typing in one
  fills the other.
- **Registering from *Regístrate*** always confirms the email with the code, and confirming logs the
  shopper in.
- **Automatic registration:** a guest who buys is registered by the checkout with `Shipping`'s names,
  email and phone, plus the address used when it's a home delivery.

### Email confirmation at checkout

**When:** the rule depends on the payment method, so the check runs on the *Pago* step, once the
guest has picked the method and presses pay/confirm, and before Culqi opens or the order is placed.
The backend decides one of three outcomes:

| The email… | Flag for the chosen method | What happens |
|---|---|---|
| already has a `WebCustomer` (confirmed **or not**) | any | **Code required.** Confirming is a login done in the background: the cart gets that customer's session and continues in the logged-in flow. |
| is new | **on** | **Code required.** Confirming creates the `WebCustomer` (confirmed) and logs the shopper in. |
| is new | off | **No code.** The `WebCustomer` is created **unconfirmed**, and the cart gets its session. |

**Two flags, one per kind of payment** (decided), so the company can tune each separately:
- **Online payment** (card, Yape): flag `10` "Require email confirmation on online payment|Requerir
  confirmación de correo en pago online".
- **Deferred payment** (cash on delivery, pay at the store): flag `14` "Require email confirmation
  on deferred payment|Requerir confirmación de correo en pago diferido".

- **Why they exist:** an order that isn't paid upfront could be used to spam the company with orders
  under someone else's email, so a company will usually turn on the deferred one. A paid order has
  no such problem, so the online one can stay off.
- **Why an existing email always needs the code**, even an unconfirmed one: the shopper typing it
  may not be the person who created that customer. Their data never reaches an existing account
  without the code.
- An unconfirmed customer becomes confirmed the first time they type a code (login with code, or a
  later checkout).
- A logged-in customer skips all of this.
- Before that step, there's no customer record. What a guest types, **including the addresses saved
  through "Nueva dirección"**, stays in the browser (`Ecommerce.customer`, persisted to localStorage).
  It goes to the backend when the customer is created or logged into, then it's cleared.
- **No default-address flag** (decided). The cart pre-selects the address used on the last order.

### Mis Direcciones

- Lists the customer's saved addresses. A "Nueva dirección" button opens a **modal** with the address
  form.
- Address fields: Departamento/Provincia/Distrito (`CiudadesSelector`), Dirección, Referencia.
- The address form is **one component**, `AddressForm.svelte`, used in two places:
  - inside the *Mis Direcciones* modal;
  - inline on the cart's *Datos Envío* step (below).

### Address selector in the cart's *Datos Envío* (home delivery)

- A selector lists the saved addresses plus a **"Nueva dirección"** option.
- "Nueva dirección" is **selected by default when the customer has no saved addresses**.
- Selecting "Nueva dirección" shows `AddressForm` inline, right there. Saving it stores the address, the
  same way the modal does, and selects it.
- Selecting a saved address hides the inputs. The checkout request carries that address's data in
  `Shipping`. The order keeps its own copy, so editing or deleting the address later doesn't change
  past orders.

### Backend (draft)

- **`WebCustomer`** (decided: a new table, not `crm.ClientProvider`): one record per company and
  **unique per email** (`CompanyID` + normalized lowercase `Email`). It holds `FirstName`, `LastName`,
  `Email`, `Phone`, `DocumentNumber` (DNI, see the boleta rule), `EmailConfirmed`, a password hash
  (empty until the customer sets one), plus a `ClientID` filled in later if the shopper also becomes a
  CRM client. The sale order still gets `ClientID = 0`.
- **Email confirmation:** a short-lived 6-digit code per email. One mechanism serves
  registration, passwordless login, and confirming an email that's already in use at checkout.
  Rate-limited per email and per IP.
- A customer address table: `CompanyID`, `ID`, `CustomerID`, `CityID`, `Address`, `Reference`,
  `Status`, `Updated`.
- Endpoints for the logged-in customer: read profile + addresses, save profile, save address,
  delete address, and **list my orders** (*Mis Pedidos*: newest first, read through the
  `(WebCustomerID, Date)` index, showing status, total, delivery type and the boleta or
  *Orden de Compra*).
- **Session token:** it reuses the back-office token signer with a new "web customer" kind that
  carries `CompanyID` + `WebCustomerID`. The route handler rejects that kind on every back-office
  route, and the customer routes accept only it.
- **A "before paying" endpoint** receives the guest's email, data and chosen payment method, and
  applies the table in *Email confirmation at checkout*. It either answers "code sent" or creates the unconfirmed customer
  and returns its session. The code-verify endpoint then creates the customer or logs into it.
- So `p-web-checkout` always runs with a customer session. It requires one, references the customer
  on the sale order, and saves a new address when "Nueva dirección" was used.
- **New company flags** in `backend/company_flags.toml`, in a new `[ecommerce]` section
  ("Online store|Tienda online"), with the next free ids (9 is retired):
  - `10` "Require email confirmation on online payment|Requerir confirmación de correo en pago
    online"
  - `11` "Require contact number|Requerir número de contacto"
  - `12` "Issue receipt automatically for online sales|Emitir comprobante automáticamente en ventas
    online" (old Q6)
  - `13` "Include cash on delivery in the web store|Incluir pago contra entrega en tienda web"
    (old Q16)
  - `14` "Require email confirmation on deferred payment|Requerir confirmación de correo en pago
    diferido" (old Q21)

  They're checked from the existing company flags table. The storefront needs them before calling the
  backend, so `PublishCompanyPub` (§1) also publishes them in `e-{companyID}.json`. The backend reads
  them through `core.HasCompanyFlag` and never trusts the client's copy.
## Open questions

None. Q1–Q23 are answered (see *Decided*, §3 and §5); Q5 was superseded by `WebCustomer` (§5).
