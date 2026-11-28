---
schema: 1
page_id: production.orders
route: /production/orders
title: Production Orders (Órdenes de producción)
status: implemented
visibility: tenant
description_en: >-
  Plan, release and receive manufacturing orders. A draft lists made products per size; releasing
  freezes the recipes and reserves the supplies. Each receipt enters good units into stock and
  consumes the actual supplies; hours logged per worker and a subcontracted workshop add to the
  cost. A board by stage and a plan-versus-actual summary follow each order.
description_es: >-
  Planificar, liberar y recibir órdenes de producción. Un borrador lista productos fabricados por
  talla; al liberarlo se congelan las recetas y se reservan los insumos. Cada ingreso entra las
  unidades buenas al stock y consume los insumos reales; las horas de cada trabajador y la maquila
  de un taller suman al costo. Tablero por etapa y resumen de lo planificado contra lo real.
---

# Production Orders (Órdenes de producción)

<!-- DOC-ID: page-purpose -->
## Page purpose

Production Orders (`Órdenes de producción`) is where a manufacturer plans what it will make
(`fabricar`), reserves the supplies (`insumos`) for it, and receives the finished units into stock
in one or several receipts (`ingresos`). Each order also gathers the hours workers spent on it
(`mano de obra`) and, when a workshop makes it, the material sent to that workshop (`maquila`).
The value each receipt books is the actual consumption plus labor, workshop price and overhead, so
the products enter stock at their real production cost.

The page does not define recipes (`recetas`) or production costs: those are on the **Producción**
tab of Products (Productos) and of a semi-finished supply in Supplies & Materials (Insumos &
Materiales). It does not buy supplies: Purchase requirements (Necesidades de compra) computes what
to buy. The daily production close of a restaurant or bakery is not here either: Daily close
(Cierre del día) runs it, and its closes do not appear in this list.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **production order** (`orden de producción`, `OP`) plans units of made products (`productos
  fabricados`, origin **Fabricado** on Products) or of semi-finished supplies (`semielaborados`),
  per size or presentation (`talla`, `presentación`). Each order has a number (**N°**).
- **Statuses (Estados):** **Draft (Borrador)** is editable and reserves nothing. **Released
  (Liberada)** has its recipes frozen and its supplies reserved. **In progress (En proceso)** has
  had a receipt, a shipment of material to the workshop, or hours logged. **Finished (Terminada)**
  is closed: what it still reserved is released. **Annulled (Anulada)** orders leave the list.
- **Receiving warehouse (Almacén de ingreso)** is where the finished units enter. **Supplies
  warehouse (Almacén de insumos)** is where the supplies are reserved and consumed from.
- **Supply adjustment % (Ajuste de insumos %)** asks for more (positive) or fewer (negative)
  supplies than the recipe, on top of each supply's waste (`merma`).
- **Series (Series)** is how many times a template's size run the order holds.
- **Good (Buenas)** units enter stock; **Scrapped (Falladas)** units were made but defective: they
  used supplies, never enter stock and carry no value — the good units absorb the whole cost.
- **Subcontracting (Maquila):** a workshop (`taller`), registered as a supplier, makes the order
  with supplies the company sends to the workshop's own warehouse.
- **Stage (Etapa)** is a column of the board (`Corte`, `Aparado`, `Armado`). It is information
  only: it moves no stock and changes no cost.

<!-- DOC-ID: capability.create-draft -->
## Create or edit a draft order (Crear o editar una orden en borrador)

### User intention (Intención del usuario)

Plan a production run: which products, how many units of each size, from which warehouse the
supplies come and where the products enter.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Production (Producción) → Production orders (Órdenes de producción)**
at `/production/orders`. **New (Nueva)** opens an empty draft in a side panel titled **New
production order (Nueva orden de producción)**; clicking an order in the list opens it. The draft is
edited on the **Products (Productos)** tab and saved with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Receiving warehouse (Almacén de ingreso)** and **Supplies warehouse (Almacén de insumos)**:
  both required, active warehouses.
- Optional: **Stage (Etapa)**, **Planned start (Inicio planificado)**, **Planned end (Fin
  planificado)**, **Supply adjustment % (Ajuste de insumos %)**, **Series (Series)** and **Notes
  (Notas)**.
- The size grid (`grilla de tallas`): **Product (Producto)** adds a row; pick the product and type
  the units under each size. Grey cells are sizes that product does not have. Products without
  presentations are planned in the **Units (Unidades)** column. **Total** sums the row.
- Only made products (origin **Fabricado**) and semi-finished supplies are offered. To save, each
  must have a recipe or a standard cost (`costo estándar`) defined on its **Producción** tab.

### Business rules and rationale (Reglas y razón de negocio)

- At least one product with units is required, and a product can appear only once in the grid.
- A size must be an active presentation of the product, and units must be greater than 0.
- The supply adjustment must be between -99.99% and +100%.
- The planned end cannot be before the planned start.
- The order's day (**Día**) is the day it was created.

### Result and side effects (Resultado y efectos)

The draft is saved and appears in the list as **Draft (Borrador)**. Nothing is reserved and no
stock moves until it is released.

### Limitations (Limitaciones)

Only a draft is editable. Once released, the warehouses, dates, stage field, adjustment, series,
notes and the grid are locked.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo una orden de producción? ¿Cómo planifico la fabricación de zapatos por talla?
- ¿Por qué no aparece mi producto en la orden? → it must be **Fabricado** (or a semi-finished
  supply) with a recipe or standard cost.
- Orden de fabricación, OP, lote de producción, corrida, plan de producción, tallas, curva de
  tallas.

<!-- DOC-ID: capability.create-from-template -->
## Create an order from a template (Nueva orden desde plantilla)

### User intention (Intención del usuario)

Reuse a saved list of products in their size runs instead of typing the grid again.

### Where to find it (Dónde encontrarlo)

**From template (Desde plantilla)** in the toolbar opens **New order from template (Nueva orden
desde plantilla)**: pick **Template (Plantilla)** and **Series (Series)**, then **Create (Crear)**.

### Business rules and rationale (Reglas y razón de negocio)

- Series defaults to the template's default series.
- Each size's units are the template's units × the series. The template's warehouses and notes are
  copied to the order.

### Result and side effects (Resultado y efectos)

A new draft opens with those lines; it is saved like any draft. From then on the order is
independent of the template: editing or deleting the template never changes it.

### Limitations (Limitaciones)

Creating from a template does not save the order by itself; save the draft.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo hago una orden con la plantilla "Zapatos Verano" por 3 series?
- ¿Si cambio la plantilla, cambian las órdenes ya creadas? → no.

<!-- DOC-ID: capability.save-as-template -->
## Save an order as a template (Guardar como plantilla)

### User intention (Intención del usuario)

Turn an order that worked well into a reusable template.

### Where to find it (Dónde encontrarlo)

Open a saved order → **Save as template (Guardar como plantilla)**.

### Business rules and rationale (Reglas y razón de negocio)

The template is named **Desde orden N° …**. Its one series is the order's units divided by the
order's series (1 when the order has none), rounded and never below 1 unit per size; its default
series, warehouses and notes come from the order.

### Result and side effects (Resultado y efectos)

A new template appears in Production templates (Plantillas de producción), where it can be renamed
and edited. The order does not change.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Puedo guardar esta orden para repetirla? ¿Cómo creo una plantilla desde una orden?

<!-- DOC-ID: capability.release-order -->
## Release an order (Liberar una orden)

### User intention (Intención del usuario)

Commit to producing the draft: freeze what each product needs and hold the supplies so no other
order takes them.

### Where to find it (Dónde encontrarlo)

Open a saved draft → **Release (Liberar)**, then confirm "Las recetas se congelan en la orden y se
reservan sus insumos." The draft is saved first.

### Required information and prerequisites (Requisitos previos)

Every recipe supply must be an active supply and every product must still be valid for production.
The supplies warehouse must hold enough free stock of every supply.

### Business rules and rationale (Reglas y razón de negocio)

- Per supply, the planned quantity (`Planificado`) is units × recipe quantity ÷ recipe yield × (1 +
  waste) × (1 + supply adjustment), summed over the lines and rounded up.
- Per labor type, the planned minutes are units × recipe minutes ÷ yield, and the planned cost uses
  the labor type's cost per hour at release.
- A semi-finished supply is consumed from stock; it is not exploded into its own supplies. Make it
  with its own order first.
- Free stock is the warehouse's stock minus what other orders already reserve. Lots that are not
  released (`Liberado`), such as lots in quarantine (`cuarentena`), do not count.
- When any supply is short, the release is refused and lists each one ("se necesitan … y hay …
  libres"). The page then suggests Purchase requirements (`/production/requirements?order-id=…`),
  which opens with this draft already selected.

### Result and side effects (Resultado y efectos)

The order becomes **Released (Liberada)**. The recipes are frozen on it, so later recipe edits do
not change it. Each planned supply is reserved (`reservado`) in the supplies warehouse. The
**Supplies (Insumos)**, **Labor (Mano de obra)**, **Receipts (Ingresos)** and **Summary (Resumen)**
tabs appear. No stock leaves yet.

### Limitations (Limitaciones)

There is no partial release: the whole order is released or nothing is. A released order cannot
go back to draft.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué no puedo liberar la orden? → a supply is short; check Necesidades de compra.
- ¿Qué significa liberar? ¿La receta cambia si edito el producto después? → no, it is frozen.
- Reservar insumos, explosión de materiales, congelar receta, lanzar la orden.

<!-- DOC-ID: capability.register-receipt -->
## Receive produced units (Registrar un ingreso)

### User intention (Intención del usuario)

Enter into stock the units already finished, partially or all at once, and record what was really
consumed.

### Where to find it (Dónde encontrarlo)

Open a released or in-progress order → **Receive (Ingresar)**. The **Receipt of order N° …
(Ingreso de la orden N° …)** dialog lists each product and size with **Pending (Pendiente)**,
**Good (Buenas)** and **Scrapped (Falladas)**, the box **Complete: last receipt, finishes the order
(Completar: último ingreso, termina la orden)**, and **Actual consumption (Consumo real)** per supply
in **Units (Unidades)** and **Sub-units (Sub-unidades)** next to the **Proposed (Propuesto)**
quantity. Save with **Register (Registrar)**.

### Required information and prerequisites (Requisitos previos)

At least one good unit. Units and consumptions cannot be negative.

### Business rules and rationale (Reglas y razón de negocio)

- The proposed consumption of each supply is its planned quantity × (good + scrapped) ÷ total
  planned units: scrapped units used supplies too. With **Completar** ticked, it proposes what is
  left of each supply (planned − consumed). The consumption can be edited.
- More units than planned can be received (over-production is allowed).
- A supply consumed beyond the order's own reservation must fit in the warehouse's free stock.
- Each product's value is its consumed supplies + the labor logged since the previous receipt +
  the workshop price × (good + scrapped) units + its overhead per unit (`costos indirectos`) × good
  units. A product with a standard cost enters at standard cost × good units instead; its supplies
  are still consumed.
- The labor is split among the received products by their recipes' labor; the supplies by their
  recipes' share.
- If the order changed after it was opened (for example another receipt was registered), the
  receipt is refused so a proposal made on an older order is never registered.

### Result and side effects (Resultado y efectos)

The supplies leave the supplies warehouse (or the workshop's warehouse when subcontracted) as
**Production consumption (Consumo de producción)**, and the good units enter the receiving
warehouse as **Production entry (Ingreso de producción)** at their value. A product that tracks lots
enters a lot named `OP<order>-<receipt>` with the product's shelf life and quarantine setting. The
receipt is listed in **Receipts (Ingresos)** with its **Supplies (Insumos)**, **Labor (M. de
obra)**, **Workshop (Maquila)**, **Overhead (Indirectos)** and **Value (Valor)**. The order becomes
**In progress (En proceso)**, or **Finished (Terminada)** when every line reached its planned good
units or **Completar** was ticked; finishing releases what the order still reserves.

### Limitations (Limitaciones)

A receipt cannot be undone or edited ("No se puede deshacer"). The receipt is dated the day it is
registered; there is no date field.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo ingreso al almacén lo que se fabricó? ¿Puedo recibir por partes? → yes, several receipts.
- ¿Dónde registro las unidades falladas o defectuosas? → **Falladas**.
- ¿Por qué el costo del producto sube si hay fallados? → good units absorb the whole cost.
- Ingreso de producción, parte de producción, producto terminado, mermas, defectuosos, segunda.

<!-- DOC-ID: capability.log-labor -->
## Log hours worked on an order (Registrar horas de mano de obra)

### User intention (Intención del usuario)

Record who worked on the order, on what type of work and for how many hours, so the real labor
cost reaches the products.

### Where to find it (Dónde encontrarlo)

Order panel → **Labor (Mano de obra)** tab. The top table compares **Planned h (Horas plan.)** and
**Logged h (Horas reg.)**, **Planned cost (Costo plan.)** and **Booked cost (Costo cargado)** per
labor type. Under **Hours logged (Horas registradas)**, fill the first row: **Worker (Trabajador)**,
**Labor type (Tipo de trabajo)**, **Date (Fecha)** and **Hours (Horas)**, then the check button. The
delete button removes a saved row.

### Required information and prerequisites (Requisitos previos)

- The order must be **Released** or **In progress**.
- Workers come from the company's employees; labor types from Production settings (Configuración
  de producción).
- The date is picked among the last 31 days and cannot be in the future. One row holds between 1
  minute and 24 hours.

### Business rules and rationale (Reglas y razón de negocio)

- Genix values each row at the worker's real cost per hour on that day (**Cost / hour (Costo /
  hora)** and **Amount (Monto)**). A payroll (`Planilla`) worker needs a contract in force that day,
  and their month never books more than their estimated monthly payroll cost: once reached, later
  rows book a smaller or zero amount. A worker off payroll uses their agreement's rate or, without
  one, the labor type's cost per hour.
- The worker must be active and employed on that date.
- A labor type not planned by the recipes can be logged; it is added to the comparison.

### Result and side effects (Resultado y efectos)

The order's logged labor is updated and the order becomes **In progress (En proceso)**. The next
receipt absorbs the labor logged since the previous one into the products' value.

### Limitations (Limitaciones)

- A finished order takes no more hours.
- Only receipts carry logged labor into product value; **Complete (Completar)** registers no
  receipt.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro las horas de un operario en la orden? ¿Por qué el monto sale en cero? → the
  worker's monthly payroll cost is already booked.
- Horas hombre, mano de obra directa, tareo, parte de horas, jornada.

<!-- DOC-ID: capability.subcontract -->
## Subcontract an order to a workshop (Maquila)

### User intention (Intención del usuario)

Have an external workshop (`taller de maquila`) make the order with the company's supplies, at an
agreed price per unit.

### Where to find it (Dónde encontrarlo)

Order panel → **Subcontracting (Maquila)** tab. In a draft: **Workshop (empty = made in-house)
(Taller (vacío = fabricación propia))** and **Price per unit (net of IGV) (Precio por unidad (sin
IGV))**. Once released: **Send materials (Enviar materiales)** and **Return leftovers (Devolver
sobrante)**, with a table of **Planned (Planificado)**, **Reserved (Reservado)**, **At workshop (En
taller)**, **Consumed (Consumido)** and **To move (sub-units) (A trasladar (sub-unid.))**.

### Required information and prerequisites (Requisitos previos)

Only suppliers that have their own warehouse are offered: create a warehouse with **Subcontractor
workshop (optional) (Taller de maquila (opcional))** set to that supplier in Sites & Warehouses
(Sedes & Almacenes).

### Business rules and rationale (Reglas y razón de negocio)

- **To move** is prefilled with what is still to send: planned − at workshop − consumed.
- Sending moves the stock from the supplies warehouse to the workshop's warehouse, and the
  reservation travels with it. Returning moves it back and frees it; no more than what is at the
  workshop can be returned.
- Receipts consume the supplies at the workshop's warehouse.
- The workshop is paid for every unit it made, good or scrapped: price per unit × units is added to
  each receipt's value. **Agreed for the plan (Acordado por el plan)** shows planned units × price.

### Result and side effects (Resultado y efectos)

Sending or returning material moves the order to **In progress (En proceso)**.

### Limitations (Limitaciones)

- The workshop and price can only be set while the order is a draft.
- Completing an order with material still at the workshop warns "Aún hay material en el taller":
  the material stays in the workshop's warehouse. Return the leftovers first.
- This page does not pay the workshop nor register its invoice.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo mando material al taller? ¿Cómo registro una orden de maquila? ¿Por qué no aparece mi
  taller? → it needs its own warehouse.
- Maquila, maquilador, tercerizar, servicio de confección, taller externo.

<!-- DOC-ID: capability.complete-order -->
## Complete an order (Completar una orden)

### User intention (Intención del usuario)

Close an order that will receive nothing more, even if fewer units than planned were made.

### Where to find it (Dónde encontrarlo)

Order **In progress** → **Complete (Completar)**, then confirm. Alternatively, tick **Completar** in
the last receipt.

### Result and side effects (Resultado y efectos)

The order becomes **Finished (Terminada)** and what it still reserves is released for other uses.

### Limitations (Limitaciones)

A finished order cannot be reopened, receive more units or take more hours.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cierro una orden que no se terminó completa? ¿Cómo libero los insumos que sobraron?

<!-- DOC-ID: capability.annul-order -->
## Annul an order (Anular una orden)

### User intention (Intención del usuario)

Cancel an order that will not be produced.

### Where to find it (Dónde encontrarlo)

**Annul (Anular)** in the order panel, then confirm "¿Anular la orden? Se liberan sus reservas."

### Business rules and rationale (Reglas y razón de negocio)

The page offers **Annul** on a saved draft and on a released order with no receipts. An order that
has had hours logged or material sent is already **In progress** and is no longer offered Annul;
use **Complete (Completar)** instead.

### Result and side effects (Resultado y efectos)

Its reservations are released and the order leaves the list.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cancelo una orden de producción? ¿Por qué no me deja anular? → it is already in progress.

<!-- DOC-ID: capability.board -->
## Follow orders on the board by stage (Tablero por etapa)

### User intention (Intención del usuario)

See where each order is on the shop floor and move it from one stage to the next.

### Where to find it (Dónde encontrarlo)

Page view **Board (Tablero)**; **List (Lista)** returns to the table. Stages are defined in
Production settings (Configuración de producción) → **Stages (Etapas)**.

### Business rules and rationale (Reglas y razón de negocio)

- Only **Released** and **In progress** orders appear, one column per stage in position order, plus
  **No stage (Sin etapa)** for orders without one or whose stage was deleted.
- Each card shows the products, the workshop when subcontracted and the percentage received.
- Dragging a card changes its stage; if Genix refuses, the card goes back.

### Result and side effects (Resultado y efectos)

Only the stage changes: no stock moves and no cost changes.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿En qué etapa está la orden? ¿Cómo paso una orden de corte a aparado? Kanban, tablero.

<!-- DOC-ID: capability.order-summary -->
## Compare plan and actual (Resumen de la orden)

### User intention (Intención del usuario)

Check yield, supply usage and labor of an order against its plan.

### Where to find it (Dónde encontrarlo)

Order panel → **Summary (Resumen)** tab, plus **Supplies (Insumos)** (planned, **Available
(Disponible)** at release, consumed, reserved).

### Business rules and rationale (Reglas y razón de negocio)

- **Products:** **Planned (Plan.)**, **Good (Buenas)**, **Scrapped (Falladas)**, **Yield (Rendim.)**
  = good ÷ planned, **Est. unit cost (Costo est. unit.)** from the current recipe or standard cost,
  and **Booked unit cost (Costo real unit.)** = value received ÷ good units.
- **Supplies:** **For what was made (Para lo fabricado)** is the planned share of the units made
  (good + scrapped); **Usage (Uso)** = consumed ÷ that share, so above 100% means more supplies than
  planned were used.
- **Labor:** planned versus logged minutes and cost per labor type.
- The list column **Received (Recibido)** is good units ÷ planned units and can exceed 100%.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuánto me costó realmente cada zapato? ¿Se usó más cuero del planificado? Rendimiento, costo
  real vs estimado, desviación.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Released and in-progress orders hold supplies reserved (`reservados`) in their supplies
  warehouse; another order's release sees that stock as unavailable.
- The list shows draft, released, in-progress and finished orders, newest first, and can be
  filtered by order number or product name.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Elija el almacén de ingreso y el de insumos."** Pick both warehouses.
- **"… no existe o no es fabricado por la empresa"** or **"… no tiene receta ni costo estándar."**
  Set the product's origin to **Fabricado** and its recipe or standard cost on its Producción tab.
- **"Faltan insumos para liberar la orden (revise Necesidades de compra): …"** Buy or transfer the
  missing supplies, or plan fewer units.
- **"La receta de … usa una sub-unidad anterior del insumo …: vuelva a guardarla."** Save the
  product's recipe again after the supply's sub-unit changed.
- **"La orden cambió desde que se abrió el ingreso: revíselo de nuevo."** Close the order panel,
  open the order again from the list and repeat the receipt.
- **"… no tiene costo para su ingreso de producción."** The product's consumption, labor, workshop
  and overhead add up to zero; give it a standard cost or supplies with cost.
- **"El taller de la orden no tiene almacén…"** Create the workshop's warehouse in Sedes &
  Almacenes.
- **"… no tiene un contrato vigente ese día"** or **"… no trabajaba en la empresa ese día."** Check
  the worker's contract and dates in Employees & Contracts (Empleados & Contratos).

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Products (Productos)** and **Supplies & Materials (Insumos & Materiales):** recipes, yield,
  standard cost, overhead, lots.
- **Production templates (Plantillas de producción):** reusable size runs for new orders.
- **Purchase requirements (Necesidades de compra):** what to buy and which semi-finished supplies to
  make for the drafts.
- **Production settings (Configuración de producción):** labor types and board stages.
- **Hours vs payroll (Horas vs planilla):** hours logged on orders compared with payroll.
- **Daily close (Cierre del día):** production of products already sold, without orders.
- **Sites & Warehouses (Sedes & Almacenes):** warehouses and workshop warehouses.
- **Movements Report (Rep. Movimientos):** the production consumption and entry movements.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.create-draft]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [related-pages]
  - path: frontend/routes/production/orders/+page.svelte
    role: page
    hash: sha256:9889b9760fc5a6e3a368b911b2e56a29d8f287f9d3a5a99aa9a6e25ec13977ee
    supports: [page-purpose, capability.create-draft, capability.create-from-template, capability.board, rules]
  - path: frontend/routes/production/orders/OrderLayer.svelte
    role: user-interface
    hash: sha256:4a4bde050b0a7bbd17913252b2a69fefad3bc25285eca40e6ea66fd3ed29331c
    supports: [concepts, capability.create-draft, capability.save-as-template, capability.release-order, capability.register-receipt, capability.complete-order, capability.annul-order, capability.order-summary]
  - path: frontend/routes/production/orders/ProductSizeGrid.svelte
    role: user-interface
    hash: sha256:6986b0297bcc078c6e95f9d17242fa934ed6e255eaf725219b7211056fe7cbf8
    supports: [capability.create-draft]
  - path: frontend/routes/production/orders/OrderReceipt.svelte
    role: user-interface
    hash: sha256:24ac2cd67a43428b5cb396486d907c4c26a50cd9b30b058e017551088e8067bf
    supports: [capability.register-receipt]
  - path: frontend/routes/production/orders/OrderLaborTab.svelte
    role: user-interface
    hash: sha256:79966e09066dc13ca92e040466f3e4710620e4916bd5f44332afab7dced03720
    supports: [capability.log-labor]
  - path: frontend/routes/production/orders/OrderMaterialTab.svelte
    role: user-interface
    hash: sha256:4bbe7be195a00757bc56734e8329f3c4c5b42f2549bcd3eae9e8607797b35bb8
    supports: [capability.subcontract]
  - path: frontend/routes/production/orders/OrderBoard.svelte
    role: user-interface
    hash: sha256:917e76d173bc929837dd9a7880483fe1826b1c8acd289e784d24df59ba2612b6
    supports: [capability.board]
  - path: frontend/routes/production/orders/production-order.ts
    role: business-logic
    hash: sha256:ae1e83a71061c4619e8921f42a5364212fcc9008075189a77dc20619eb8d6874
    supports: [concepts, capability.create-draft, capability.create-from-template, capability.save-as-template, capability.register-receipt, capability.subcontract, capability.order-summary, troubleshooting]
  - path: frontend/routes/production/orders/production-order.svelte.ts
    role: frontend-service
    hash: sha256:af03f65459602f36a00094d11acb4c6a8791ee2502795ec8961d0c42b28d072a
    supports: [capability.create-draft, capability.subcontract, capability.order-summary]
  - path: frontend/routes/production/products/product-production.ts
    role: business-logic
    hash: sha256:fbe2490b19f264171272ebdbb67b328e00503474ff313f35c17c343415923ef6
    supports: [capability.order-summary]
  - path: frontend/services/production/production-orders.ts
    role: data-model
    hash: sha256:d404f4b53e13a71bf34c426e61829282222c1e5e7036a3c7602ddd8dd04c8a2c
    supports: [concepts]
  - path: frontend/services/production/production-orders.svelte.ts
    role: frontend-service
    hash: sha256:df596ce61dd20414c561032eca941867ef89dd14b724eefd5d031d2a3b50c0b7
    supports: [page-purpose, capability.annul-order, rules]
  - path: frontend/core/stock-movement-type.ts
    role: shared-domain
    hash: sha256:cfcee015c4d6fadb03f9a9e5d66064e93d734cfa5216e2d36996c47140eecc9c
    supports: [capability.register-receipt]
  - path: frontend/routes/business/branches-warehouses/+page.svelte
    role: user-interface
    hash: sha256:01058b8b8d39cb72c8c49edcd852e1e66363870c646eaeb09d2d92e1830b3876
    supports: [capability.subcontract]
  - path: frontend/routes/logistics/warehouse-movements/+page.svelte
    role: user-interface
    hash: sha256:c5233b1bd06373cc61602bf42ed8be502fa92de1bd6163c2c588eedaab1f0c4a
    supports: [related-pages]
  - path: backend/production/production_orders.go
    role: backend-handler
    hash: sha256:d6e99162ca6371a0e3f00161ea2f0e47b0a554ae13d6d83e06d88bd5fef58883
    supports: [concepts, capability.create-draft, capability.release-order, capability.register-receipt, capability.subcontract, capability.complete-order, capability.annul-order, capability.board, rules, troubleshooting]
  - path: backend/production/production_receipt.go
    role: business-logic
    hash: sha256:4f0464bbfca549579bd3882492cde194b5bc22c2eaac497932d7b80087e4633a
    supports: [concepts, capability.register-receipt, troubleshooting]
  - path: backend/production/production_labor.go
    role: backend-handler
    hash: sha256:8e1de7d581d752f00c10882fd64198bfad95b25e5175486eadb78a1b7732173b
    supports: [capability.log-labor]
  - path: backend/production/types/production_order.go
    role: data-model
    hash: sha256:1fad6a84fb14b0c090f35c9e7f5d845e915d956b7895cdb9606fde3a1665ded8
    supports: [concepts, capability.order-summary]
  - path: backend/production/types/production_order_rules.go
    role: business-logic
    hash: sha256:17687af36766228e14a12cf1357c803a7a7e1968f8038bb057f0a68beeec2a4d
    supports: [capability.create-draft, capability.release-order, capability.register-receipt, troubleshooting]
  - path: backend/production/types/production_labor_entry.go
    role: business-logic
    hash: sha256:574ac7219c8b34b77f3e70616aa3b85abf2b09700d207f1f277fe82789cdefe2
    supports: [capability.log-labor]
  - path: backend/production/types/supply_consumption.go
    role: business-logic
    hash: sha256:ba6cde8e8efa09c93de47de09e360bd9ac27ebc7732a680e36e87db365bb63c0
    supports: [capability.release-order, rules]
  - path: backend/hr/types/labor_cost.go
    role: business-logic
    hash: sha256:65feaa94e6f2db9b0aa6225194be57dd5df28ff655d2da28cfbaf7bf257af1b0
    supports: [capability.log-labor, troubleshooting]
```
