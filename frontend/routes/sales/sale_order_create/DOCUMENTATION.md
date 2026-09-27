---
schema: 1
page_id: sales.sale_order_create
route: /sales/sale_order_create
title: Point of Sale (Punto de Venta)
status: implemented
visibility: tenant
description_en: >-
  Point of sale (POS). Search in-stock products by name or serial number, add them to a
  cart with quantity, choose the warehouse and cash account, sell in soles or dollars
  (converting prices with the company's or BCRP's sell exchange rate), assign an existing customer
  or register a walk-in one, generate the sale order with payment and/or delivery
  actions, and reprint the thermal ticket from this device's sale history. A separate
  Settings view edits two sales parameters.
description_es: >-
  Punto de venta (POS). Buscar productos en stock por nombre o número de serie,
  agregarlos al carrito con cantidad, elegir el almacén y la caja, vender en soles o en
  dólares (convirtiendo precios con el tipo de cambio de venta de la empresa o del BCRP), asignar un
  cliente existente o registrar uno nuevo, generar la orden de venta con las acciones de
  pago y/o entrega, e imprimir el ticket desde el historial de ventas de este equipo. Una
  vista de Configuración aparte edita dos parámetros de ventas.
---

# Point of Sale (Punto de Venta)

<!-- DOC-ID: page-purpose -->
## Page purpose

Point of Sale (`Punto de Venta`, POS) is where a cashier records an immediate sale (`venta`)
against one warehouse's stock: search products, build a cart, decide whether the sale
collects payment now and/or removes stock now, choose the currency (`moneda`: soles or
dollars), optionally attach a customer, and generate the order (`Generar`). Right after
generating, the side panel switches to **Historial**, where the sale's thermal ticket
(`ticket`) can be printed. Internally the page's own title/tabs read **Ventas** /
**Configuración**.

This page only creates new sale orders. It does not own the order's later lifecycle —
confirming a pending payment/delivery afterward, canceling an order, or reviewing sales
from every till belongs to **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`;
the **Historial** here only lists this device's own recent sales for reprinting tickets.
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
- A **sub-unit** is a smaller unit the same product can be sold in — grams cut from a
  kilogram, or single candies from a box. There is **no separate sub-unit row**: a product
  configured with a sub-unit shows one card that offers both, with the whole-unit quick
  buttons in grey and the sub-unit ones in purple (labelled with the first letter of the
  sub-unit name), and both prices displayed. Sub-unit buttons stop below one whole unit —
  past that the operator adds a unit instead. Sub-units are charged at the product's
  `SbuFinalPrice`, never at a fraction of the whole-unit price.
- **Stock is shared between the two.** The card's stock figure counts both, so one box on
  hand covers six candies when the sub-unit divisor is 6, and the remaining stock is shown
  as `2 + 3 unidad` when part of a unit is left. Selling four candies deducts four candies
  from the warehouse, not four boxes.
- **Serialized stock (Serie)** is inventory tracked by individual serial number; for those
  products the card shows clickable serial-number chips instead of quick-quantity buttons,
  and each serial is sold and validated independently.
- The **sale currency (moneda de la venta)** is either **soles (S/)** or **dollars (US$)**. Every
  price, the subtotal, the total, and the payment of that sale are in its currency. A product
  keeps its own catalog currency; when it differs from the sale's, its price is converted with
  the **sell exchange rate (tipo de cambio de venta)** — the company's own, or BCRP's for a day the
  company did not load — see *Vender en soles o en
  dólares* below.

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
stock de "<producto>" para agregar <cantidad>.`; requesting one more of an already-added
serial number beyond its detail-row quantity is rejected with `La serie <serie> del producto
"<producto>" sólo posee <n> unidad(es).`. Pressing **Enter** on a serialized card is rejected
with `Seleccione una serie específica.` — bulk-adding an ambiguous serialized product by
keyboard is not supported; a specific serial chip must be clicked. Quick-quantity buttons above
the remaining stock simply do not render for that card.

### Result and side effects (Resultado y efectos)

Adds a new cart line or increases an existing line's quantity (and, for serialized adds, its
per-serial quantity map), then recalculates the cart totals. The price on each product card is
shown in the current sale's currency, so a soles product reads in dollars during a dollar sale
(and vice versa).

### Limitations (Limitaciones)

There is no free-text quantity input on a card — only the fixed quick-quantity buttons, +1 via
Enter, or one unit per serial chip click. The line price is always the product's stored
`FinalPrice` (and `SbuFinalPrice` for the sub-unit part), only restated in the sale's currency
when the product is priced in the other one; this page offers no way to discount or override a
line's price, and the server re-reads both prices from the catalog (and converts them again)
when the sale is posted, so a price sent by the browser is never trusted.

Serialized stock is sold in whole units only: a serial number identifies one physical item, so
serial chips never carry a sub-unit part.

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
(rounded down) and **Total** shows the cart's full amount; both update live as lines change,
including when the sale currency changes. Both carry the sale's currency symbol — `S/` or
`US$` — and the cart's **PRECIO** column is in that same currency.

<!-- DOC-ID: capability.warehouse-cash-selection -->
## Choose warehouse and cash account (Elegir almacén y caja)

The **ALMACÉN** selector at the top decides which warehouse's stock is shown and is the
`WarehouseID` sent with the order — it also drives which warehouse is debited when **Recibido**
is included. The **CAJA** selector sits on the same row as the action checkboxes and only appears
when **Pagado** is checked and the company has at least one registered cash/bank account
(`caja o banco`); it is the account credited when **Pagado** is included, and the page picks the
first available caja by default. Each caja option is tagged **Soles** or **Dólares**, and beside
the selector a **Moneda:** box shows the chosen caja's currency — which is also the sale's
currency while **Pagado** is checked. With **Pagado** unchecked, that same cell shows the
**Date Pago** field instead, followed by a **MONEDA** selector (**Soles** / **Dólares**) — the caja
and the due date are never visible at the same time, because a sale already paid has no due date.
When no cash/bank account exists yet, the caja selector and the **Pagado** action option both
disappear and the page shows: `Necesitas registrar una caja para aceptar pagos.`

<!-- DOC-ID: capability.assign-client -->
## Assign a customer (Asignar un cliente)

### User intention (Intención del usuario)

Attach the sale to an existing customer (`cliente`), register a new walk-in customer inline,
or leave the sale without a customer (anonymous sale).

### Where to find it (Dónde encontrarlo)

The left-hand selector on the row below the action checkboxes: **SIN CLIENTE** (default, no customer),
**Selecionar Cliente** (spelled that way in the option; opens the **Buscar cliente por nombre o
documento** search box, matching by name or document/registry number), or **Registrar Cliente**
(inline **Nº Documento** and **Nombre del cliente** fields). The **Nº Documento** field carries a
small document-type picker (`tipo de documento`: DNI, RUC, C.E. for carné de extranjería, PSP for
pasaporte) that preselects DNI for 8 digits and RUC for 11; the cashier can override it, for
example to mark a foreign document as a passport rather than a carné de extranjería.

### Required information and prerequisites (Requisitos previos)

Registering inline only requires **Name**; the document/registry number is optional — unless the
sale names an invoicing series that demands a buyer, see *Elegir el comprobante* below. Existing
customers come from the same catalog as the Customers (`Clientes`) page.

### Business rules and rationale (Reglas y razón de negocio)

The backend resolves a registered walk-in customer by first matching an existing client with
the same registry number, then by a name+registry identity hash; if either matches, the sale
reuses that client's ID **without overwriting any of that client's already-stored fields**
(name, email, etc. are left untouched) — the code explicitly avoids letting sale-order input
corrupt an existing shared client record. Only when neither match is found is a brand-new
client created. Leaving **SIN CLIENTE** selected sends no client at all, so the sale is
recorded without an associated customer.

A newly created walk-in client keeps the document type picked in **Nº Documento**; when the
picker is left without a choice, Genix derives it from the number (8 digits DNI, 11 digits RUC).
That type is part of the identity the sale's electronic document declares for the buyer. A
reused existing client keeps its own stored document type, like the rest of its fields.

### Result and side effects (Resultado y efectos)

The resolved (or newly created) client ID is stored on the sale order as `ClientID`.

### Limitations (Limitaciones)

Registering a walk-in customer here only captures name, document number and document type; it
cannot set email, person type, or other fields the Customers page exposes. The till offers only
DNI, RUC, carné de extranjería and pasaporte; other SUNAT document types must be set on the
Customers page. There is no way to edit an
existing customer's data from this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Puedo vender sin asignar cliente?` Sí, dejando "SIN CLIENTE", salvo que el comprobante elegido
  sea una factura, o una boleta desde S/ 700.
- `¿Qué pasa si registro un cliente con el mismo RUC/documento que uno ya existente?` Genix
  reutiliza el cliente existente en vez de crear uno duplicado.
- `¿Cómo registro un cliente con pasaporte?` En "Registrar Cliente", elija PSP en el selector de
  tipo de documento junto a "Nº Documento".
- Search terms: `cliente`, `registrar cliente`, `venta anónima`, `RUC`, `DNI`, `documento`,
  `tipo de documento`, `pasaporte`, `carné de extranjería`.

<!-- DOC-ID: capability.choose-invoice-series -->
## Choose the invoicing series (Elegir el comprobante)

### User intention (Intención del usuario)

Decide which SUNAT series (`serie de facturación`) the sale will be issued under, so the
electronic document later emitted for it carries the right type and code.

### Where to find it (Dónde encontrarlo)

The **SIN COMPROBANTE** selector, to the right of the client-mode selector. Options read as
type and code, for example `BOLETA · B001` or `FACTURA · F001`.

### Required information and prerequisites (Requisitos previos)

The series come from the company's **Series de Facturación** table (Configuración → Facturación).
Only **active** series of type **Boleta** and **Factura** are offered: credit and debit note
series exist to correct a document that already went out, never to open a sale.

### Business rules and rationale (Reglas y razón de negocio)

The chosen series is sent as `IssueSeriesID` and the backend folds it into the last two digits
of the sale order's ID, so the electronic document issued for that sale can derive its own ID
from the sale's. Nothing is preselected: leaving **SIN COMPROBANTE** sends `0`, which records a
sale that names no series; no electronic document is created for it, and its ticket prints as a
**NOTA DE VENTA**.

Choosing a series imposes the buyer SUNAT requires on that document type, and the sale is refused
if it is missing — at the till, not later at emission, because the series cannot be changed once
the sale exists:

- **FACTURA** — the customer must have an **11-digit RUC** and a name (razón social). The message
  is `Una factura necesita un cliente con RUC de 11 dígitos.`
- **BOLETA from S/ 700** — the customer must have a document (**DNI**, RUC, or a foreign document
  of at least 8 characters) and a name. Below S/ 700 a boleta stays anonymous. The threshold is
  always in soles: a dollar sale is compared by its soles equivalent at the sale's exchange rate,
  so a US$ 200 boleta already needs an identified buyer when that rate puts it at S/ 700 or more.
- The series must still exist and be active on the company when the sale is generated.

### Result and side effects (Resultado y efectos)

The series is fixed at creation and cannot be changed afterwards — it is part of the sale
order's ID, not an editable field.

Generating the sale also **creates its electronic document (comprobante)**, already numbered
with its correlativo, in state *Pendiente de envío*. The comprobante is issued in the sale's
currency: a dollar sale produces a document in **USD** (`dólares`), a soles sale one in **PEN**.
Nothing is sent to SUNAT from here: a
scheduled process sends it minutes later, and the result is followed on **Contabilidad →
Facturación** (`/accounting/invoicing`). Because the comprobante is created with the sale, a
sale that cannot be invoiced is refused whole — nothing is registered.

### Limitations (Limitaciones)

The page does not send the electronic document to SUNAT, nor does it show its state; that is
what the Facturación page is for. A company with no series configured sees an empty list.

Choosing a series makes the sale harder to undo: once its comprobante has been sent, the sale
can no longer be annulled and needs a nota de crédito, which Genix does not emit yet. While the
comprobante is still pending, annulling the sale voids it and it never goes out.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Puedo vender sin comprobante?` Sí, dejando "SIN COMPROBANTE".
- `¿Por qué no aparece mi serie?` Porque está inactiva, o es una serie de nota de crédito/débito.
- `¿Por qué no me deja generar la factura?` Porque la factura exige un cliente con RUC de 11
  dígitos y razón social; asígnelo en el selector de cliente.
- `¿Dónde veo si la factura llegó a SUNAT?` En **Contabilidad → Facturación**.
- `¿Puedo emitir una factura en dólares?` Sí; una venta en dólares genera su comprobante en USD.
- Search terms: `comprobante`, `serie`, `boleta`, `factura`, `SUNAT`, `RUC`, `correlativo`,
  `factura en dólares`.

<!-- DOC-ID: capability.sale-currency -->
## Sell in soles or dollars (Vender en soles o en dólares)

### User intention (Intención del usuario)

Charge a sale in dollars (`venta en dólares`) or in soles, and let the till convert the prices of
products whose catalog currency is the other one, using the company's own exchange rate.

### Where to find it (Dónde encontrarlo)

On the **Venta** panel, row of the **Recibido** / **Pagado** checkboxes:

- With **Pagado** checked, the currency follows the selected **CAJA**: a dollar caja makes a dollar
  sale, a soles caja a soles sale. The **Moneda:** box beside it shows which one.
- With **Pagado** unchecked, no caja names the currency, so the **MONEDA** selector (**Soles** /
  **Dólares**) appears next to **Date Pago** and the cashier picks it.

When the sale needs a conversion, the dollar label carries the rate actually applied in
parentheses — **Dólares (TC X.XXX)** — in the **Moneda:** box, or in the currency selector when the
sale is unpaid.

### Required information and prerequisites (Requisitos previos)

A conversion needs a **sell rate (tipo de cambio de venta)** for today or one of the previous 7
days. Genix walks back day by day from today and uses the first day that has one. For each day it
takes the company's own rate saved on **Finanzas → Tipo de Cambio** (`/finance/exchange-rate`);
when the company did not load that day, it takes the published **BCRP interbank sell rate**
(`tipo de cambio interbancario BCRP`) — the same value the Tipo de Cambio calendar shows in the
cells the company left empty. A rate the company saves always wins over BCRP for that day. If the
BCRP series cannot be downloaded, only the company's own rates are used. A **dollar sale always
needs a rate**, even when every product is already priced in dollars, because Genix stores the rate
with the sale to restate it in soles later; a soles sale needs one only when the cart holds a
product priced in dollars.

### Business rules and rationale (Reglas y razón de negocio)

- The rate is the sell rate moved by the company flag **Spread de tipo de cambio** (Configuración →
  Mi Empresa → Flags de la Empresa), always **in the store's favour**: converting soles → dollars
  divides by `(venta − spread)`, converting dollars → soles multiplies by `(venta + spread)`. A
  company with no spread converts at the plain sell rate.
- The conversion is done **per unit price and rounded to the cent**, so every line is exactly
  quantity × the unit price shown, and the lines add up to the total.
- A paid sale must be in the currency of the caja that collects it, because the payment is booked
  in that caja's currency; the server refuses a mismatch (`La venta está en dólares pero la caja
  "<caja>" es en soles.`).
- Without any sell rate — the company's or BCRP's — in the last 7 days (a rate weeks old would
  quietly misprice the sale), the till shows an amber warning `No hay tipo de cambio de venta
  (propio ni BCRP) en los últimos 7 días. Regístrelo en Finanzas > Tipo de cambio.`, converted
  prices read `0.00`, and **Generar** is refused with that same message. If the spread is equal to or larger than the sell rate, the warning reads `El
  spread de tipo de cambio no deja un tipo de cambio válido.` The server applies the same checks
  and resolves the rate itself; the till only previews it.

### Result and side effects (Resultado y efectos)

The sale stores its currency and the effective rate used. Its prices, subtotal, total, the
`Cobro (Venta)` movement in the caja, and its electronic document are all in the sale's currency,
and the printed ticket reads **TOTAL US$** or **TOTAL S/**. The day-level sales summaries behind
**Gráficos Ventas** and **Reporte Ventas** stay in soles: a dollar sale is restated there with the
rate stored on the sale, not with today's rate.

### Limitations (Limitaciones)

- The rate cannot be typed or overridden at the till; save the company's own rate for the day on
  **Tipo de Cambio** (it replaces the BCRP rate for that day) or change the spread on
  **Configuración**.
- Only soles and dollars are supported.
- The currency is fixed when the sale is created; a later payment on **Gestión Ventas** must use a
  caja of that same currency.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo hago una venta en dólares?` Marque "Pagado" y elija una caja en dólares, o desmarque
  "Pagado" y elija "Dólares" en MONEDA.
- `¿Por qué no me deja generar la venta en dólares?` No hay tipo de cambio de venta, ni propio ni
  del BCRP, en los últimos 7 días; regístrelo en Finanzas > Tipo de cambio.
- `¿Qué tipo de cambio usa el punto de venta?` El tipo de cambio de venta del día más reciente de
  los últimos 7 días: el que registró la empresa o, si no lo registró, el interbancario del BCRP;
  ajustado por el spread a favor de la tienda.
- `¿Tengo que registrar el tipo de cambio todos los días?` No; si falta el de la empresa, se usa
  el del BCRP. Regístrelo solo si quiere usar su propio tipo de cambio.
- `¿Los reportes de ventas salen en dólares?` No; los resúmenes de ventas se mantienen en soles.
- Search terms: `moneda`, `dólares`, `soles`, `US$`, `tipo de cambio`, `TC venta`, `spread`,
  `conversión`, `venta en dólares`, `BCRP`, `interbancario`.

<!-- DOC-ID: capability.set-payment-delivery -->
## Choose Pagado / Recibido and generate the sale (Definir Pagado / Recibido y generar la venta)

### User intention (Intención del usuario)

Decide, at the moment of generating the sale, whether it should immediately register the
payment in a cash/bank account, immediately remove the sold quantities from warehouse stock,
both, or neither — then create the sale order.

### Where to find it (Dónde encontrarlo)

The **Recibido** / **Pagado** checkboxes above the client selector (both checked by default,
though **Pagado** is hidden if no caja exists) and the payment due-date field (**Date Pago**,
optional, shown with the **MONEDA** selector in place of the caja selector while **Pagado** is
unchecked); the blue **Generar** button in the cart header creates the order.

### Required information and prerequisites (Requisitos previos)

The cart must have at least one line and a warehouse must be selected before **Generar** can
succeed (`El carrito está vacío.` / `Seleccione un almacén.`), and a sale that needs an exchange
rate is refused while none is available (see *Vender en soles o en dólares*). Including **Pagado**
requires a selected caja (enforced server-side too: `Se requiere LastPaymentCajaID para procesar el
pago.`) of the sale's currency. Including **Recibido** requires the warehouse and at least one detail line
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
books one `Cobro (Venta)` cash/bank movement for the sale total (in the sale's currency) against
the chosen caja and advances Status toward **Pagado**. A sale generated without **Pagado** stores
no caja at all, even though the till keeps its caja selection for the next paid sale; the caja
is chosen later, when the payment is registered on **Gestión Ventas**. Including **Recibido** additionally books one outbound stock
movement (`Entrega a cliente final (Venta)`) per cart line — including one per distinct serial
number — against the chosen warehouse, and advances Status toward **Entregado**. Every save also
updates the day-level sales summary consumed by **Sales Charts** and **Sales Report** (always in
soles), and schedules a background reprocess job. On success the page shows `Venta registrada con
éxito`, clears the cart and the client selection, records the sale in this device's history, and
switches the side panel to **Historial** so the ticket can be printed right away.

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

<!-- DOC-ID: capability.sale-history-ticket -->
## Sale history and thermal ticket (Historial y ticket de venta)

### User intention (Intención del usuario)

Print the ticket (`ticket`, `boleta impresa`) of the sale just made, or reprint one from a sale
made minutes or hours earlier at this same till.

### Where to find it (Dónde encontrarlo)

The **Historial** tab at the top of the side panel (next to **Venta**). Each card shows the
document number, the time, the document title and the customer, a **PAGADO** or **POR COBRAR**
chip, an **ENTREGADO** chip when stock was delivered, the item count, and the total. The round
print button opens the ticket preview; choose **58 mm** or **80 mm** paper and press
**Imprimir**.

### Business rules and rationale (Reglas y razón de negocio)

- The history is **local to this browser** (per company): it lists only sales generated from this
  device, newest first, and keeps the latest 200. It is not a server query, so a sale made at
  another till does not appear here.
- The title and number come from the sale's series: **FACTURA ELECTRÓNICA** or **BOLETA DE VENTA
  ELECTRÓNICA** numbered `<serie>-<correlativo de 8 dígitos>`, or **NOTA DE VENTA** numbered
  `VENTA #<id>` when the sale named no series.
- The ticket prints the company's legal name, trade name, RUC and address, then **FECHA**,
  **CAJERO** (only when the logged-in user is the one who made the sale), **ALMACEN**, **CLIENTE**
  (`VARIOS` when there is none) and **DOC**, each line with quantity × unit price (sub-units on
  their own row, serial numbers as `S/N`), then **OP. GRAVADA**, **IGV** and **TOTAL S/** or
  **TOTAL US$** according to the sale's currency, and **PAGADO - <caja>** or **PENDIENTE DE PAGO**.
  A fiscal sale adds `Representación impresa del comprobante electrónico`.
- The prices on the ticket are the ones the server charged, in the sale's currency.

### Limitations (Limitaciones)

- A card keeps the status the sale had when it was generated: paying, delivering or annulling it
  later on **Gestión Ventas** does not change the card or the reprinted ticket.
- Clearing the browser's data, or using another browser or device, loses this history; the sales
  themselves remain on the server and are reviewed on **Gestión Ventas**.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo reimprimo el ticket de una venta?` En la pestaña "Historial", botón de imprimir.
- `¿Por qué no veo una venta en el historial?` Porque se hizo desde otro equipo o navegador.
- `¿Puedo imprimir en papel de 80 mm?` Sí; elija 58 mm u 80 mm antes de imprimir.
- Search terms: `ticket`, `reimprimir`, `impresora térmica`, `historial de ventas`, `nota de venta`,
  `58 mm`, `80 mm`.

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
- Prices and the exchange rate are authoritative on the server too: the amounts the till shows
  in the chosen currency are a preview, and the server re-reads the catalog prices and the
  company's latest sell rate and spread when the sale is created.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El carrito está vacío." / "Seleccione un almacén.":** add at least one product and pick a
  warehouse before pressing **Generar**.
- **"No hay suficiente stock de ... para agregar ..." / "La serie ... sólo posee N
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
- **Amber warning "No hay tipo de cambio de venta (propio ni BCRP) en los últimos 7 días..." and
  prices at 0.00:** the sale is in dollars, or the cart holds a product priced in the other
  currency, and neither a company rate nor a BCRP rate exists for today or the previous 7 days —
  usually because the BCRP series could not be downloaded and the company has not saved its own.
  Record the rate on **Finanzas → Tipo de Cambio**, or sell in soles with soles-priced products
  only.
- **"El spread de tipo de cambio no deja un tipo de cambio válido." / "El spread de tipo de cambio
  (...) no puede ser mayor o igual al tipo de cambio (...)":** the **Spread de tipo de cambio** flag
  is equal to or larger than the sell rate; correct it on **Configuración → Mi Empresa**.
- **"La venta está en dólares pero la caja "..." es en soles." (or the reverse):** a paid sale must
  use a caja of its own currency; pick a caja of the intended currency.
- **A sale I made does not appear in Historial:** the history only lists sales generated from this
  browser; use **Gestión Ventas** to find sales from any till.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: review, confirm a
  pending payment or delivery, and cancel (`Anular pedido`) orders created on this page.
- **Sales Charts (Gráficos Ventas)** and **Sales Report (Reporte Ventas)**: consume the
  day-level sale summary that every **Generar** action here updates.
- **Cash & Banks (Cajas y Bancos)**: register the cash/bank account offered in the **CAJA**
  selector, and review the resulting `Cobro (Venta)` movement afterward.
- **Customers (Clientes)** at `/crm/customers`: maintain full customer records beyond the
  minimal name/document captured by the inline **Registrar Cliente** flow here.
- **Sites & Warehouses (Sedes y Almacenes)** and **Product Stock**: manage warehouses and
  inspect stock levels before and after a **Recibido** sale.
- **Exchange Rate (Tipo de Cambio)** at `/finance/exchange-rate`: record the company's own daily
  sell rate; a day left empty there falls back to the BCRP rate when a dollar sale, or a sale
  mixing currencies, is converted.
- **Configuration (Configuración)** at `/company/configuration`: the **Spread de tipo de cambio**
  flag under **Mi Empresa → Flags de la Empresa**, and, under **Facturación**, the invoicing series
  offered in **SIN COMPROBANTE**.
- **Invoicing (Facturación)** at `/accounting/invoicing`: follow the electronic document each
  invoiced sale creates, issued in the sale's currency.

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
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: page
    hash: sha256:19d9bce55dd5a19b8b09064cf22cf0e26dc632f925982e044595ff010a3f3135
    supports: [page-purpose, concepts, capability.search-add-products, capability.manage-cart, capability.warehouse-cash-selection, capability.assign-client, capability.choose-invoice-series, capability.sale-currency, capability.set-payment-delivery, capability.sale-history-ticket, capability.configure-sales-parameters, troubleshooting]
  - path: frontend/routes/sales/sale_order_create/SaleProductCard.svelte
    role: user-interface
    hash: sha256:9ff070990df65e898e95b31449ad3a9eefe2fdc62ebddc3262fba548cbf2680e
    supports: [concepts, capability.search-add-products, capability.sale-currency]
  - path: frontend/routes/sales/sale_order_create/sale_order.svelte.ts
    role: frontend-service
    hash: sha256:39359a3890c09db37c8ee0785d9aa511134f2d9f45033ae382a405116b6df865
    supports: [concepts, capability.search-add-products, capability.manage-cart, capability.choose-invoice-series, capability.sale-currency, capability.set-payment-delivery, rules, troubleshooting]
  - path: frontend/routes/sales/sale_order_create/sale_order.ts
    role: business-logic
    hash: sha256:d6e955d95c48a82b07e6fa7e1490cbdb375cbc715710867dcd8c69de1d54fe4b
    supports: [capability.assign-client, capability.choose-invoice-series, capability.sale-currency, rules]
  - path: frontend/routes/sales/sale_order_create/sale_history.ts
    role: business-logic
    hash: sha256:08ee9e6c4287e4c4cf5573dea0b9a3dd9e53cff1d6b274a6c11e5bd3b970f245
    supports: [capability.sale-history-ticket]
  - path: frontend/routes/sales/sale_order_create/sale_history.idb.ts
    role: frontend-service
    hash: sha256:ae1d0820f7d16815cad9fcf7676d1ce8fa331602ba308d67f6bb2eefd96ff439
    supports: [capability.sale-history-ticket, troubleshooting]
  - path: frontend/routes/sales/sale_order_create/sale_history.svelte.ts
    role: frontend-service
    hash: sha256:9f6234ec5313101b64d7e9bc9eab0aa4fa56f93261d0f0576630c0da11c59153
    supports: [capability.sale-history-ticket]
  - path: frontend/routes/sales/sale_order_create/SaleHistoryCards.svelte
    role: user-interface
    hash: sha256:b4e8a9267991cd67776fc6d8b998b8ac79e003285e49c01210959253e67f0df1
    supports: [capability.sale-history-ticket]
  - path: frontend/routes/sales/sale_order_create/SaleTicketModal.svelte
    role: user-interface
    hash: sha256:1ebeb8ec390ee7445487b762fe2470e5a6b7f394d1176edb11e9000fe308e503
    supports: [capability.sale-history-ticket]
  - path: frontend/routes/sales/sale_order_create/sale_ticket.ts
    role: business-logic
    hash: sha256:73c286e869295d9d23b2eadab519894f8ba07cb8f3cfc1c09dbf353843a45398
    supports: [capability.choose-invoice-series, capability.sale-currency, capability.sale-history-ticket]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.ts
    role: business-logic
    hash: sha256:55738f078d629f60276bbfbbfb58b46b3b1b7ba7a9626d06504b2e909294bf28
    supports: [concepts, capability.manage-cart, capability.sale-currency, troubleshooting]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.svelte.ts
    role: frontend-service
    hash: sha256:f5f401b8f1e32dd79eba283d2a7c9b64bb90eca0b81db938049b32a2db8ab333
    supports: [capability.sale-currency]
  - path: frontend/services/crm/identity-doc.ts
    role: shared-domain
    hash: sha256:9923cb088e0f23088291bf802b2459822bcb8335e7277af8a8b72a34cd9da9a8
    supports: [capability.assign-client]
  - path: frontend/routes/logistics/products-stock/stock-movement.ts
    role: frontend-service
    hash: sha256:7e32f71f5e9e74c8f10059925f34918ab35e81d24e965178f1f9debc3b5d558f
    supports: [capability.search-add-products, capability.warehouse-cash-selection]
  - path: frontend/services/production/products.svelte.ts
    role: shared-domain
    hash: sha256:64b802038c223a036d232e9f409295bcf78faf98d5cb84b0bac9d6fa85822abf
    supports: [concepts, capability.search-add-products, capability.sale-currency]
  - path: frontend/services/crm/client-provider.svelte.ts
    role: shared-domain
    hash: sha256:f51dcec7e71526b00f501213e74ad5ecb22dce74dd3bbde9110c518717398d07
    supports: [capability.assign-client]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:6b8e304283715cf7293c68424b6fd45d8d048828ed1b3d6498ab1e28cf2dcfd3
    supports: [capability.warehouse-cash-selection, capability.sale-currency]
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
  - path: frontend/routes/company/configuration/+page.svelte
    role: user-interface
    hash: sha256:28f8b985ad11d8d87bf928f7a2754dc746f3576615be099c4c5149c89af36749
    supports: [capability.choose-invoice-series, capability.sale-currency, troubleshooting, related-pages]
  - path: frontend/routes/company/configuration/CompanyFlagsPanel.svelte
    role: user-interface
    hash: sha256:5104402b9597454f663178f1538c87efb1bde665c11676b3b46c06042df5f242
    supports: [capability.sale-currency, related-pages]
  - path: backend/company_flags.toml
    role: shared-domain
    hash: sha256:251737436c1efa9be9212036e58c148c413685a1c2088ae77660c5690fefe1df
    supports: [capability.sale-currency, troubleshooting, related-pages]
  - path: backend/sales/sale_order_create.go
    role: backend-handler
    hash: sha256:8901a11e252bc4794f1ab4d2d4e626c5213765121dbc5787566dfb5958dde5f4
    supports: [capability.search-add-products, capability.assign-client, capability.sale-currency, capability.set-payment-delivery, rules, troubleshooting]
  - path: backend/sales/sale_order_currency.go
    role: business-logic
    hash: sha256:186747cd3a2892beeef7428ff285df67edc44caa494e2b3b0b2985aa46b92a6f
    supports: [capability.sale-currency, capability.set-payment-delivery, rules, troubleshooting]
  - path: backend/finance/types/exchange_rate.go
    role: business-logic
    hash: sha256:7dc6dbf27632d692cf814d4376e7284957187874e2a0e318a8d861e4dd0de377
    supports: [capability.sale-currency, troubleshooting]
  - path: backend/finance/types/bcrp_rates.go
    role: business-logic
    hash: sha256:814f197ccbd7e90bbd1b611ac3dc4291a208c7b86e621f5bea42462036b6cf8f
    supports: [concepts, capability.sale-currency, troubleshooting, related-pages]
  - path: backend/sales/sale_order_issuance.go
    role: business-logic
    hash: sha256:54d09414aae726a0e37b0f7e0ed5e053ca83be87c8b934b385adf87e8f4969b7
    supports: [capability.assign-client, capability.choose-invoice-series, rules, troubleshooting]
  - path: backend/invoicing/types/customer_identity.go
    role: business-logic
    hash: sha256:7dfd832529c2a43322972097c2c843d019f9e537c3e3a5803bf837bb06e847cc
    supports: [capability.choose-invoice-series, rules]
  - path: backend/invoicing/types/sale_order_to_cpe.go
    role: business-logic
    hash: sha256:3fff7a9720606bbf18a93512f4361605dbfb073bfc19e44c2c497aeb619a9634
    supports: [capability.assign-client, capability.choose-invoice-series, capability.sale-currency]
  - path: backend/invoicing/types/document_reserve.go
    role: business-logic
    hash: sha256:eca97837ae724317e507794654a9568bca097b05d531839d30deb50df6c45317
    supports: [capability.choose-invoice-series, capability.sale-currency]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:7db7f29652af90da6fb8c68c6e992a0111162b1aa5492ac2b4564f5dd1895427
    supports: [concepts, capability.assign-client, capability.choose-invoice-series, capability.sale-currency, capability.set-payment-delivery, rules]
  - path: backend/crm/types/client_provider_save.go
    role: business-logic
    hash: sha256:83fea49c5c006c10ecbedcdef86985286f620474d89627db2d09ee109200e190
    supports: [capability.assign-client]
  - path: backend/finance/types/cash_movement_apply.go
    role: business-logic
    hash: sha256:6b47d33cfaa81d34eec52d00172ac223307b6a0869dd2ec2b3616af9154b5e5a
    supports: [capability.set-payment-delivery]
  - path: backend/sales/sale_summary.go
    role: business-logic
    hash: sha256:9c450fd7892ee2bbb4107a9c7a41bebd54dfb01e0807d379c05fb0638bacfb0a
    supports: [capability.sale-currency, capability.set-payment-delivery, related-pages]
  - path: backend/access.toml
    role: permissions
    hash: sha256:31c7071defebf7ecec268a2ae932e58667d2f0e143dee894d0262d8eab4baa3c
    supports: [rules]
```
