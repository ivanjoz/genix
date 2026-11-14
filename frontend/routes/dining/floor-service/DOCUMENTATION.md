---
schema: 1
page_id: dining.floor-service
route: /dining/floor-service
title: Table Service (Atención de Mesas)
status: implemented
visibility: tenant
description_en: >-
  The waiters' live floor map of a restaurant layout. Each table is colored by its state (free,
  eating, ordered, waiting, late, ready to pick up); clicking a table opens its order to start it
  with a first round, add more rounds of products with notes, mark dishes ready, send them back to
  the kitchen, serve or cancel them, and see what the table owes. The bill is charged into a sale,
  paid or owed, with a factura or boleta, or the whole order is annulled. A strip lists the dishes
  waiting at each station's pickup point. Ordering reserves the products' stock in the layout's
  warehouse.
description_es: >-
  Mapa en vivo del salón para los mozos. Cada mesa se colorea según su estado (libre, comiendo,
  pedido en cocina, esperando, retrasado, listo para recoger); al hacer clic en una mesa se abre su
  pedido para iniciarlo con una primera ronda, agregar más rondas de productos con notas, marcar
  platos listos, devolverlos a cocina, servirlos o anularlos, y ver cuánto lleva consumido la mesa.
  La cuenta se cobra como una venta, pagada o a crédito, con factura o boleta, o se anula el pedido
  completo. Una franja muestra los platos que esperan en el punto de recojo de cada estación. Pedir
  reserva el stock de los productos en el almacén del salón.
---

# Table Service (Atención de Mesas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Table Service (`Atención de mesas`) is where waiters (`mozos`, `meseros`) serve a restaurant floor.
It draws a layout (`salón`) designed in **Diseño de salones** as a live map: every table shows its
state by color and how long it has waited for the kitchen, and clicking a table opens its order
(`pedido de la mesa`, `comanda`). From the order the waiter opens the table with a first round,
sends more rounds to the kitchen, follows each dish (`plato`) from the kitchen to the table, sees
the running total in soles, and finally charges the bill (`cobrar mesa`, `cerrar cuenta`) into a
sale, or annuls the whole order.

Ordering reserves the products in the layout's warehouse, so the till and the web store stop
selling what the restaurant has already promised. In this version the page does **not** split a
bill, move or merge tables, take tips, show a kitchen display per station, or take orders by QR
code from the customers' phones.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Table order (`pedido de la mesa`):** everything a table has ordered while it is occupied. A
  table has at most one open order; it is opened by the first round and it is not a sale: it uses no
  receipt number and appears in no sales report until it is charged (`cobrada`), when it becomes
  one sale. An order ends charged or annulled (`anulado`); either way the table is free again.
- **Round (`ronda`, `pedido`):** one batch of products the waiter sends to the kitchen at once
  (`Ronda 1`, `Ronda 2`...). Each round is a list of lines.
- **Line (`línea`, `plato`):** one product (and presentation) with its units, an optional note
  such as "sin cebolla", its price and the prep station it goes to. Each line moves through its own
  states (`estado del plato`):
  - **En cocina** (In the kitchen): ordered, being prepared.
  - **Listo** (Ready): prepared, waiting at the station's pickup point.
  - **Servido** (Served): delivered to the table.
  - **Anulado** (Cancelled): withdrawn before it was ready.
  - **Por confirmar** (To confirm): reserved for orders placed by QR, which no page offers yet.
- **Prep station (`estación de preparación`):** where a line is prepared (Cocina, Bar...). Each line
  takes its station when the round is sent, by the routing of the layout's stations: the station
  naming the product, else the one naming its category, else the default station.
- **Table state (`estado de la mesa`)**, the color on the map, decided by the first rule that
  matches:
  1. **Retrasado** (Late, red): a dish has been in the kitchen for at least the layout's **Retraso a
     los** minutes.
  2. **Listo para recoger** (Ready to pick up, teal): some dish is ready.
  3. **QR por confirmar** (QR to confirm, purple): some line waits for a waiter's confirmation.
  4. **Esperando** (Waiting, amber): a dish has been in the kitchen for at least the layout's
     **Alerta a los** minutes.
  5. **Pedido en cocina** (Ordered, blue): some dish is in the kitchen.
  6. **Comiendo** (Eating, green): an open order with nothing in the kitchen or ready.
  7. **Libre** (Free, white): no open order.
- **Wait (`espera`, `demora`):** how long the oldest dish still in the kitchen has waited since it
  was ordered. It shows in minutes just above the table's top-right corner (`12'`) and in the
  table's panel; it stops counting once that dish is ready.

<!-- DOC-ID: capability.floor-map -->
## Follow the floor (Ver el salón en vivo)

### User intention (Intención del usuario)

See at a glance which tables are free, which are waiting too long and which have dishes to pick up.

### Where to find it (Dónde encontrarlo)

**Comercial → Restaurante (Restaurant) → Atención de mesas (Table service)**, route
`/dining/floor-service`. The **Salón** picker chooses the layout; next to it a legend names every
color. The user needs the access **Atención de Mesas** in their profile.

### Required information and prerequisites (Requisitos previos)

A layout saved in **Diseño de salones** with its tables. When the company has a single layout it
opens by itself; otherwise the page shows "Elija un salón." until one is picked.

### Business rules and rationale (Reglas y razón de negocio)

- The map is the layout as designed; it cannot be edited here.
- The waits and colors are recalculated every 2 seconds on the device; the open orders are reloaded
  from the server every 15 seconds while the page is visible, and at once after any change made
  from this device. Changes made on another device therefore appear within about 15 seconds.
- The wait thresholds are the layout's **Alerta a los** and **Retraso a los** minutes.

### Result and side effects (Resultado y efectos)

Looking at the map changes nothing.

### Limitations (Limitaciones)

- There is no instant push between devices yet: another waiter's change takes up to 15 seconds to
  show.
- Only one layout is shown at a time.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Qué significa la mesa en rojo?` It is late: a dish has waited at least **Retraso a los**
  minutes in the kitchen.
- `¿Qué es el número arriba de la mesa?` The minutes the table has waited for its oldest dish in the
  kitchen.
- `¿Por qué no veo lo que pidió mi compañero?` Wait a few seconds; the map reloads every 15 seconds.
- Search terms: `mapa de mesas`, `plano del salón`, `estado de las mesas`, `mesas libres`, `mesas
  ocupadas`, `tiempo de espera`, `demora`, `retraso`, `mesero`, `mozo`, `camarero`, `salón`.

<!-- DOC-ID: capability.open-table-and-order -->
## Open a table and send rounds (Abrir mesa y enviar pedidos)

### User intention (Intención del usuario)

Seat customers at a table and send what they order to the kitchen, now or as more rounds later.

### Where to find it (Dónde encontrarlo)

Click a table on the map: a side panel titled `Mesa <nombre>` opens. On a free table the panel shows
the round form directly; on a table with an order, press **Nuevo pedido** (New round). The form has
**Agregar producto** (product search), **Comensales** (guests, only when opening the table), the
lines table with **Producto**, **Cant.**, **Nota**, **Subtotal** and a delete button, the round
**Total**, **Cancelar** and **Enviar a cocina** (Send to kitchen).

### Required information and prerequisites (Requisitos previos)

- The product search lists only what can be ordered now: active products not hidden from the web
  store, with units available in the layout's warehouse, each presentation shown with the units left
  in parentheses, e.g. `Chicha · Vaso (3)`. A product with nothing available is not listed.
- Picking the same product again adds one unit to its line, unless that line already has a note;
  then a new line is added, so "1 sin cebolla" and "1 normal" stay separate.
- **Cant.** takes whole units only. A line left at 0 is not sent.
- **Nota** holds up to 120 characters.
- **Comensales** is optional.

### Business rules and rationale (Reglas y razón de negocio)

- A round carries between 1 and 50 lines.
- **Prices are fixed in soles when the round is sent**, from the product's current catalog price. A
  product priced in dollars is converted as at the till: at the company's latest sell exchange rate
  (`tipo de cambio venta`) of the last 7 days, moved by the company's exchange spread when one is
  set. Without a rate in those 7 days the round is refused naming that product. Later catalog price
  or rate changes do not change lines already ordered.
- **Stock is checked when the round is sent**, with the same rule as the web store, on the order's
  warehouse: what is in stock minus what is already reserved by the till, the web store and other
  tables, plus the product's **Sobreventa permitida** margin. A round asking for more is refused, e.g.
  "No hay stock suficiente de "Lomo saltado": quedan 2."
- Each line goes to its prep station by the layout's routing and starts **En cocina**, with its
  ordered time.
- The first round opens the order with the layout's warehouse; an open order keeps that warehouse
  for all its rounds, even if the layout's warehouse is changed meanwhile.
- Two devices sending to the same table at once are handled one after the other, so a table never
  gets two open orders.

### Result and side effects (Resultado y efectos)

The order is created or extended, the panel switches to the order with the new round on top, and the
table turns **Pedido en cocina**. The ordered units are reserved (`reservado`, "por entregar") in
the warehouse, so the till and the web store see that much less available.

### Limitations (Limitaciones)

- No fractions or sub-units: dishes are sold in whole units here.
- A sent round cannot be edited; cancel its lines and send a new round.
- **Cancelar** discards the round being written; on a free table it closes the panel.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo abro una mesa?` Click it on the map, add the products and press **Enviar a cocina**.
- `¿Cómo agrego más platos a una mesa?` Open the table and press **Nuevo pedido**.
- `¿Por qué no aparece un producto en la búsqueda?` It has no units left in the layout's warehouse,
  is inactive, or is hidden from the web store.
- `¿Cómo pongo "sin cebolla"?` Write it in the **Nota** of the line.
- `¿Por qué el precio está en soles si el producto está en dólares?` Table orders are always in
  soles, converted at the company's latest sell exchange rate.
- Search terms: `abrir mesa`, `tomar pedido`, `comanda`, `enviar a cocina`, `marchar`, `nueva
  ronda`, `agregar plato`, `nota del plato`, `comensales`, `cubiertos`, `pax`, `sin stock`.

<!-- DOC-ID: capability.follow-dishes -->
## Follow, serve and cancel dishes (Seguir, servir y anular platos)

### User intention (Intención del usuario)

Mark a dish ready when the kitchen hands it over, take back a mistaken "ready", mark it served at
the table, or withdraw a dish the customer no longer wants.

### Where to find it (Dónde encontrarlo)

In the table's panel, every round lists its lines with their state and buttons:

- **En cocina**: **Listo** (Ready) and **Anular** (Cancel).
- **Listo**: **Servido** (Served) and, for 2 minutes after it was marked, **Devolver a cocina**
  (Back to kitchen).
- **Por confirmar**: **Confirmar** and **Anular**.
- **Servido** and **Anulado**: no actions.

Above the map, the **Listos para recoger** (Ready to pick up) strip lists every ready dish of the
layout, grouped by station, the longest-waiting first, each with its table, units, product, minutes
waiting and a **Servido** button.

### Required information and prerequisites (Requisitos previos)

An open order with the line in the right state.

### Business rules and rationale (Reglas y razón de negocio)

- Allowed changes: en cocina → listo, listo → servido, por confirmar → en cocina, por confirmar or
  en cocina → anulado, and listo → en cocina only within 2 minutes of the "ready" mark ("Un plato
  marcado listo solo se puede devolver a cocina en los 2 minutos siguientes."), meant to undo a
  mistaken tap.
- **A dish cannot be cancelled once ready or served.** Any other change is refused, e.g. `Un
  producto "servido" no puede pasar a "anulado".`
- A waiter may mark dishes ready from this page, so a restaurant without a kitchen screen can work
  with it.
- The panel header shows the table's state, its wait, its guests and its total: every line not
  cancelled, at the price it was ordered at.

### Result and side effects (Resultado y efectos)

- The line's state and time are saved and the table's color updates.
- **Cancelling releases the line's reservation** in the warehouse. Ready and served dishes keep
  theirs: they are settled when the table is charged.

### Limitations (Limitaciones)

- Lines are changed one at a time; there is no "serve all".

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo marco que un plato ya salió?` **Listo** on its line, then **Servido** when it reaches the
  table (or **Servido** in the "Listos para recoger" strip).
- `Marqué listo por error.` Press **Devolver a cocina** within 2 minutes.
- `¿Puedo anular un plato que ya se sirvió?` No, only before it is ready.
- `¿Cuánto lleva consumido la mesa?` The total at the top right of the table's panel.
- Search terms: `plato listo`, `servir plato`, `anular plato`, `devolver a cocina`, `pase`, `recojo`,
  `listos para recoger`, `cuenta de la mesa`, `consumo`, `total de la mesa`.

<!-- DOC-ID: capability.charge-table -->
## Charge the table (Cobrar mesa)

### User intention (Intención del usuario)

Close a table's bill when the customers pay or leave: register the sale, collect the money into a
cash register or leave it owed by the client, and issue the factura or boleta.

### Where to find it (Dónde encontrarlo)

In the table's panel, press **Cobrar mesa** (Charge table). The form shows:

- **Caja/banco del pago** (Cash/bank for payment): the first active cash register in soles comes
  selected. Clearing it leaves **SIN PAGAR** (unpaid).
- **Comprobante**: the company's active factura and boleta series. Left empty, it shows **SIN
  COMPROBANTE** (no receipt).
- **Cliente** (Client): **SIN CLIENTE**, **Seleccionar cliente** (search an existing client by name
  or document), or **Registrar cliente** (**Nº Documento** with its document type, and **Nombre del
  cliente**), as at the till.
- **Cobrar S/ …** with the bill's total, and **Cancelar**.

Pressing **Cobrar** asks for confirmation: "¿Cobrar S/ … en "<caja>"?", or, without a cash register,
"¿Cerrar la cuenta de S/ … sin pago? Quedará como deuda del cliente."

### Required information and prerequisites (Requisitos previos)

- The **Cobrar mesa** and **Anular pedido** buttons appear only to users whose profile has the
  sub-access **Cobrar Mesa** of **Atención de Mesas**.
- **Every dish must be served or cancelled.** While a line is in the kitchen, ready or to confirm,
  the form shows "Hay productos sin servir: márquelos servidos o anúlelos antes de cobrar." and the
  button is disabled. A bill with nothing served cannot be charged.
- **Registrar cliente** needs the client's name. **Seleccionar cliente** needs a client picked.
- A **factura** always needs the buyer identified (name and document); a **boleta** needs it from
  S/ 700.

### Business rules and rationale (Reglas y razón de negocio)

- The sale bills the **served** lines **at the prices they were ordered at**, in soles; cancelled
  lines are left out. Catalog changes after the order do not reprice it.
- **A payment collects the whole bill** into the chosen cash register, which must be in soles (the
  form lists only active ones in soles). There is no partial payment here; an unpaid bill is
  collected later from **Gestión Ventas**.
- The sale is born **delivered** (the diners already ate) and from the order's warehouse:
  - **Bought products** (`mercaderías`) leave the warehouse stock now and release their reservation.
  - **Made products** (`productos fabricados`, the restaurant's dishes with a recipe) keep their
    reservation; their stock is taken by the **Cierre del día** production close.
- Before the sale is created, the bought products' stock in the warehouse is checked again; if the
  warehouse no longer holds them, the charge is refused, naming the warehouse, the product and the
  units needed and held.
- The sale is created exactly as a sale from the till: same numbering, same comprobante, same
  client register.

### Result and side effects (Resultado y efectos)

- A sale (`venta`) is created and a notice shows its number ("Venta … creada."). It appears in
  **Gestión Ventas** as delivered, and paid or with a debt for the whole total.
- With a series, its factura or boleta is created with the sale and sent to SUNAT afterwards.
- A paid bill adds its total to the cash register's balance.
- A client registered here is matched to an existing client with the same identity, or saved as a
  new one in the client list. The sale counts in the client's purchases.
- The order closes as charged, leaves the map, and the table turns **Libre**; the panel closes.

### Limitations (Limitaciones)

- No split bills, partial payments, mixed payment methods or tips.
- A charged bill cannot be reopened here; corrections and annulments of the sale are done in
  **Gestión Ventas**.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo cobro una mesa?` Serve or cancel every dish, then **Cobrar mesa**, pick the cash register
  and the comprobante, and press **Cobrar**.
- `El cliente pagará después.` Clear the cash register (**SIN PAGAR**): the sale stays as a debt.
- `¿Por qué no puedo cobrar?` Some dish is not served yet, or your profile lacks **Cobrar Mesa**.
- `¿Dónde veo la venta de la mesa?` In **Gestión Ventas**, by the number the notice showed.
- Search terms: `cobrar mesa`, `cerrar cuenta`, `pedir la cuenta`, `la cuenta`, `boleta`, `factura`,
  `comprobante`, `pagar mesa`, `crédito`, `fiado`, `deuda`, `caja`.

<!-- DOC-ID: capability.annul-order -->
## Annul a table order (Anular pedido)

### User intention (Intención del usuario)

Cancel a whole table order that will not be charged: customers who left, a table opened by mistake.

### Where to find it (Dónde encontrarlo)

In the table's panel, **Anular pedido** opens a **Motivo** (Reason) box with **Anular** and
**Cancelar**. **Anular** asks for confirmation: "¿Anular el pedido de la mesa? Sus productos se
liberan y no se cobra nada."

### Required information and prerequisites (Requisitos previos)

- The sub-access **Cobrar Mesa** of **Atención de Mesas**.
- A reason, of at most 200 characters.
- Unlike charging, it works with dishes in any state.

### Business rules and rationale (Reglas y razón de negocio)

Nothing was sold, so nothing is charged and no sale, comprobante, cash movement or stock exit is
created.

### Result and side effects (Resultado y efectos)

- The reservations of every line not already cancelled are released in the warehouse, so the till
  and the web store can sell those units again.
- The order is saved as annulled with its reason, leaves the map, and the table turns **Libre**.

### Limitations (Limitaciones)

An annulled order cannot be reopened; open the table again with a new round.

### Common questions and vocabulary (Preguntas y vocabulario)

- `Se fueron sin consumir.` Use **Anular pedido** with the reason.
- `¿Anular un plato o todo el pedido?` A single dish is cancelled with **Anular** on its line; this
  annuls the whole order.
- Search terms: `anular pedido`, `anular mesa`, `cancelar mesa`, `liberar mesa`, `mesa por error`.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No hay stock suficiente de ...: quedan N."**: the warehouse has fewer units available than the
  round asks, counting what other tables, the till and the web store already reserved. Lower the
  units, or add stock or a **Sobreventa permitida** margin to the product.
- **`"<producto>" tiene precio en dólares: ...`**: the company has no sell exchange rate from the
  last 7 days; register one before ordering that product.
- **"El pedido de la mesa acaba de cerrarse. Vuelva a enviar el pedido." / "El pedido de la mesa
  no existe o ya se cerró."**: the order changed on another device; reopen the table and repeat.
- **"La mesa no existe en el salón." or "El salón no existe o fue eliminado."**: the layout was
  changed in **Diseño de salones**; reload the page.
- **A product is missing from the search:** it has no units available in the layout's warehouse, is
  inactive or is hidden from the web store.
- **The map does not show another waiter's change:** it reloads every 15 seconds while the page is
  visible.
- **No "Cobrar mesa" button:** the profile lacks the sub-access **Cobrar Mesa** of **Atención de
  Mesas**.
- **"Hay productos sin servir...":** mark every dish served, or cancel it, before charging.
- **"El pedido de la mesa cambió mientras se cobraba. Vuelva a cobrarlo.":** a dish changed on
  another device during the charge; charge again.
- **`La venta está en soles pero la caja "..." es en dólares.`:** pick a cash register in soles.
- **A factura or boleta refused for the buyer:** a factura always, and a boleta from S/ 700, needs
  the client's name and document.
- **"El motivo de la anulación no puede superar los 200 caracteres.":** shorten the reason.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Floor layout (Diseño de salones)** at `/dining/floor-layout`, in the same Restaurante menu:
  draws the tables, sets the warehouse, the wait thresholds and the prep stations this page uses. A
  table with an open order cannot be removed there, and a layout with open orders cannot be deleted.
- **Products (Productos / Mercaderías)**: the prices, presentations, **Sobreventa permitida** margin
  and web visibility that decide what can be ordered.
- **Daily close (Cierre del día)** at `/production/daily-close`: the production close of made
  products sold from a warehouse, including those of charged tables.
- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: the sales created by
  charging tables, where an unpaid one is collected later and a sale is annulled.
- **Electronic Invoicing (Facturación Electrónica)** at `/accounting/invoicing`: the facturas and
  boletas of charged tables and their status at SUNAT.

### FILES

```yaml
# Replace pending only after the completed documentation is reviewed against exact file bytes.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/dining/floor-service/+page.svelte
    role: page
    hash: sha256:acc520ffdebe74950dba39f01a0fa7c7359da801e6a265184789b828d3e23cfc
    supports: [page-purpose, concepts, capability.floor-map, capability.charge-table, troubleshooting]
  - path: frontend/routes/dining/floor-service/TableOrderPanel.svelte
    role: user-interface
    hash: sha256:de73d9153ab0e4e6bcf4cbfe59e581a5faab39e0e6a14576796137330dc75019
    supports: [capability.open-table-and-order, capability.follow-dishes, capability.charge-table, capability.annul-order]
  - path: frontend/routes/dining/floor-service/RoundComposer.svelte
    role: user-interface
    hash: sha256:fdd058a7e971d85bd2cbf0639adac627762cb62f1a64d997bf330036ed917983
    supports: [capability.open-table-and-order]
  - path: frontend/routes/dining/floor-service/ReadyToPickupStrip.svelte
    role: user-interface
    hash: sha256:6d539f64eca92060999d971c5d7b60466bfcde0d66de7bcd85790ebbe9b7cba7
    supports: [capability.follow-dishes]
  - path: frontend/routes/dining/floor-service/TableChargeForm.svelte
    role: user-interface
    hash: sha256:73b19d3e71c22139deade2956bd54de50d096039d228931728dcf4a4b8b7455d
    supports: [capability.charge-table, capability.annul-order]
  - path: frontend/routes/dining/floor-service/table-charge.ts
    role: business-logic
    hash: sha256:2de64d87fba18b6be184e7aa08b03dbdd32bcdb671611b69e1e12a532d058b18
    supports: [capability.charge-table, capability.annul-order, troubleshooting]
  - path: frontend/routes/dining/floor-service/floor-service.ts
    role: business-logic
    hash: sha256:ca9d8a7d99df317dde638d151b360d7e80db166a2343541a80af417b844c153a
    supports: [concepts, capability.open-table-and-order, capability.follow-dishes, capability.charge-table]
  - path: frontend/routes/dining/floor-service/floor-service.svelte.ts
    role: frontend-service
    hash: sha256:74fb5f70262a6bc707ddd24239ea145b14664c108dbe55f5d52605595108aebc
    supports: [capability.open-table-and-order, capability.charge-table, capability.annul-order]
  - path: frontend/routes/dining/dining.ts
    role: shared-domain
    hash: sha256:6fd904c5753945a587d1c2f696b2a6ec93d505382c362a4a645a4734186b69ed
    supports: [concepts, capability.floor-map, capability.follow-dishes]
  - path: frontend/routes/dining/dining.svelte.ts
    role: frontend-service
    hash: sha256:43495109cc9fbb91105d283b033ed185f1cf067895944fb7837a6570e8d1de58
    supports: [capability.floor-map, capability.follow-dishes]
  - path: frontend/routes/dining/LayoutCanvas.svelte
    role: user-interface
    hash: sha256:939eb16f592af21aa07d823365b16e31f5d3e4897797b6b770cc1302f29f491c
    supports: [capability.floor-map]
  - path: backend/dining/table_orders.go
    role: backend-handler
    hash: sha256:1972ba4ecd49a4b4d39fa89610fee7386c590f59e7076b662607c3c37c63a76a
    supports: [concepts, capability.open-table-and-order, capability.follow-dishes, troubleshooting]
  - path: backend/dining/table_order_charge.go
    role: backend-handler
    hash: sha256:32e32d0a5349d8e43895fec3d0ca47a09fed34037cc3534d1f69b290b9f25632
    supports: [concepts, capability.charge-table, capability.annul-order, troubleshooting]
  - path: backend/dining/dining_menu.go
    role: backend-handler
    hash: sha256:b4dec3f3cd74a9448e1ac214cfe23205b6ba83c1ce5c1b1ab25383e3f276e9b8
    supports: [capability.open-table-and-order, troubleshooting]
  - path: backend/dining/types/table_order.go
    role: data-model
    hash: sha256:facd4a33122c29642359fa29e2f24031443f22249028c00c73d8fa28ae4fa9a7
    supports: [concepts, capability.follow-dishes, capability.annul-order]
  - path: backend/dining/types/table_order_rules.go
    role: business-logic
    hash: sha256:8c2c1ccbe531d22ed47aa1137132a7a5ae6bafdb235d1feabe184c322da378d6
    supports: [concepts, capability.open-table-and-order, capability.follow-dishes, capability.charge-table, capability.annul-order]
  - path: backend/dining/types/dining_layout_rules.go
    role: business-logic
    hash: sha256:40a72fdfa154967e80553da83a8bf9874d058f304e43b0b1b9c0c6937b4a2805
    supports: [concepts]
  - path: backend/sales/saleissue/types/new_sale.go
    role: business-logic
    hash: sha256:79480f8435d4182f16c8ebca2b5717be396423cd73a571f5ef1e2a84a229e33e
    supports: [capability.charge-table]
  - path: backend/sales/saleissue/types/sale_issue_series.go
    role: business-logic
    hash: sha256:6c210d7117e055a98c571f9bd17d0cd04f55d9458365735de9b8970e7550255d
    supports: [capability.charge-table, troubleshooting]
  - path: backend/sales/saleissue/types/sale_client.go
    role: business-logic
    hash: sha256:2dbfa3112977cdced55066900cd0ae93f2a18f7e9c89db78067a6f47c5e62ac4
    supports: [capability.charge-table]
  - path: backend/sales/types/sale_actions.go
    role: business-logic
    hash: sha256:ad4c117e0bce8f4c8e88f85e6aa3677090935cf6b6127c944f72bf1601413109
    supports: [capability.charge-table]
  - path: backend/sales/types/sale_stock_check.go
    role: business-logic
    hash: sha256:952f769c018817d9ba53b4055d0b890578bc17ce8536156bc65f756ad15563f9
    supports: [capability.charge-table]
  - path: backend/sales/types/sale_currency.go
    role: business-logic
    hash: sha256:d94fcf68ff93412a00250b54e2bb4c2c59865472b1f30d86a21e2eefcc6d5f92
    supports: [capability.open-table-and-order, capability.charge-table, troubleshooting]
  - path: backend/invoicing/types/customer_identity.go
    role: business-logic
    hash: sha256:7dfd832529c2a43322972097c2c843d019f9e537c3e3a5803bf837bb06e847cc
    supports: [capability.charge-table, troubleshooting]
  - path: backend/dining/dining_layouts.go
    role: backend-handler
    hash: sha256:6dfaa4e6006aadf31fd60a77a46135e6fd1240b6e6d7f48cd7cc1667683cf260
    supports: [related-pages]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:cb61a6f79a034e4d468b46c173733b815f0af9007a01d1f3e69b92d358913d60
    supports: [capability.floor-map]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:4f2d4c16777913fddf196959c4341290e69a12a78323a5b95f8fa5d4738ca34d
    supports: [capability.floor-map, related-pages]
  - path: backend/access.toml
    role: permissions
    hash: sha256:c19cf85130065dcc0f839796e1e9b4b98a1b78df465ccb20d2b63b59a1a94e59
    supports: [capability.floor-map, capability.charge-table, capability.annul-order, troubleshooting]
```
