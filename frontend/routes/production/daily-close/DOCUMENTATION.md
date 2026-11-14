---
schema: 1
page_id: production.daily-close
route: /production/daily-close
title: Daily Close (Cierre del Día)
status: implemented
visibility: tenant
description_en: >-
  Daily production close for made products. Per sales warehouse, list the days with delivered
  sales whose stock exit is still pending, preview the close of the oldest one (products produced,
  supplies consumed by recipe, blocking reasons), run it to consume the supplies, enter the
  products at their cost and take the sales' exits out of stock, and review past closes.
description_es: >-
  Cierre de producción del día para productos fabricados. Por almacén de ventas, listar los días
  con ventas entregadas cuya salida de stock sigue pendiente, revisar el resumen del cierre del día
  más antiguo (productos producidos, insumos consumidos según receta, motivos de bloqueo),
  ejecutarlo para consumir los insumos, ingresar los productos a su costo y sacar del stock las
  ventas, y consultar los cierres anteriores.
---

# Daily Close (Cierre del Día)

<!-- DOC-ID: page-purpose -->
## Page purpose

The Daily Close (`Cierre del día`, page title `Cierre del Día`) is where the company "produces on
paper" what it already sold and delivered. A made product (`producto fabricado`, Origen
`Fabricado` on the Productos page) is sold and delivered before it is registered as produced: at
delivery its line does not leave the warehouse, it is flagged **stock por regularizar** and its
units stay reserved. The close of a day settles those lines in one step:

- it consumes the supplies (`insumos`) of each product's recipe (`receta`) from the supplies
  warehouse (`almacén de insumos`);
- it enters the made products into the sales warehouse (`almacén de ventas`) at their production
  cost (`ingreso de producción`);
- it takes the sold units out of stock as the sales' deliveries, at that cost;
- it releases the reservations of those lines, and the sales leave **Por regularizar**.

This page does not sell, deliver or annul sales (use Point of Sale and Sales Management), does not
edit recipes or costs (use the **Production** tab on Productos), and does not register purchases or
manual stock changes of supplies (use Purchase Orders and Stock Changes). Merchandise
(`mercaderías`) never reaches this page: its lines leave stock at delivery.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Sale pending the close (`venta pendiente de cierre`, `stock por regularizar`):** a delivered
  sale (status Entregado or Finalizado) with at least one made-product line whose stock exit has
  not happened. Sales Management shows it with a yellow **R** and lists it in the **Por
  regularizar** tab.
- **Pending day (`día pendiente`):** a date with sales pending the close in the chosen sales
  warehouse. The day of a sale is the date the sale was registered (`fecha de la venta`), not the
  day it was delivered, so a sale registered on Monday and delivered on Wednesday belongs to
  Monday's close.
- **Sales warehouse (`Almacén de ventas`):** the warehouse the pending sales were delivered from.
  The made products enter it and the sales' exits leave it.
- **Supplies warehouse (`Almacén de insumos`):** the warehouse the recipes' supplies are consumed
  from. It starts as the sales warehouse and can be changed for each close.
- **Recipe and yield (`receta`, `Rinde (porciones)`):** the supplies and quantities that make a
  number of portions of the product, defined on the **Production (Producción)** tab of Productos.
  One recipe applies to every presentation of the product.
- **Standard cost (`Costo estándar`) and extra cost per unit (`Costo extra por unidad`):** the
  production configuration that decides what a produced unit is worth (see the cost rule below).
- **Strict mode (`Modo estricto`):** the company flag **"Bloquear el cierre del día si faltan
  insumos"** (section Producción of the company flags). With it, a supply short of stock blocks
  the close; without it, the close runs and shares the available supply.
- **Production close (`cierre de producción`):** the saved record of one run, with a number
  (`Cierre N°`), the day closed, the products and units produced with their value, the supplies
  needed, available and consumed, and the sales it settled.
- **Stock movements of a close:** **Consumo de producción** (a supply leaving the supplies
  warehouse), **Ingreso de producción** (a made product entering the sales warehouse) and the
  usual **Entrega (venta)** exit of each settled sale line.

<!-- DOC-ID: capability.review-pending-days -->
## See which days are pending (Ver los días pendientes de cierre)

### User intention (Intención del usuario)

Know which days of a warehouse still have made products sold and delivered without their stock
exit, how many sales and units each day holds, and which day can be closed now.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Production (Producción) → Daily close (Cierre del día)**, route
`/production/daily-close`. The same page is also listed in **Commercial (Comercial) → Restaurant
(Restaurante) → Daily close (Cierre del día)**, after **Table service (Atención de mesas)** and
**Floor layout (Diseño de salones)**; both entries open the same page. Pick the **Almacén de ventas** at the top. The **Días pendientes**
table lists each pending day with its **Día**, **Ventas** and **Unidades**. The page needs the
access **Cierre de Producción**, separate from the Productos access.

### Required information and prerequisites (Requisitos previos)

Only the sales warehouse is required to list the days. Until one is chosen the page shows
"Seleccione el almacén cuyas ventas va a cerrar."

### Business rules and rationale (Reglas y razón de negocio)

- Days are listed oldest first, and **only the oldest pending day can be closed**: it is shown in
  bold and previewed automatically; later days are shown muted and wait their turn.
- **Unidades** counts whole units of the pending lines. A line sold in sub-units (for example 3
  candies of a box of 6) counts as one whole unit, the same way its reservation counted it.
- Only sales delivered from the chosen warehouse count; another warehouse's sales have their own
  pending days.

### Result and side effects (Resultado y efectos)

Listing the days changes nothing.

### Limitations (Limitaciones)

- A newer day cannot be closed before an older one, even if the older one is blocked: fix what
  blocks the oldest day first.
- The list shows counts only; to see which sales are pending, use the **Por regularizar** tab of
  Sales Management.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Qué días me faltan cerrar?` / `¿Por qué no puedo elegir el día de hoy?` Solo se cierra el día
  pendiente más antiguo; cierre primero los anteriores.
- `¿Por qué aparece un día antiguo si ya lo cerré?` Una venta de ese día se entregó después del
  cierre; el día vuelve a quedar pendiente y se cierra otra vez.
- `No hay ventas pendientes de cierre.` El almacén no tiene ventas de productos fabricados por
  regularizar.
- Search terms: `cierre del día`, `cierre diario`, `cierre de producción`, `días pendientes`,
  `ventas por regularizar`, `stock por regularizar`, `producción del día`.

<!-- DOC-ID: capability.preview-close -->
## Review the close summary (Revisar el resumen del cierre)

### User intention (Intención del usuario)

Before closing, see what the close will produce, what it will consume from the supplies warehouse,
at what value, and whether anything prevents it.

### Where to find it (Dónde encontrarlo)

The right side of the page shows **Cierre del <día>** for the oldest pending day, with the tables
**Productos producidos** and **Insumos consumidos**. Changing the **Almacén de insumos** recalculates
the summary; changing the **Almacén de ventas** resets the supplies warehouse to it.

### Required information and prerequisites (Requisitos previos)

- Every made product sold that day needs a cost source: a recipe with supplies, a **Costo
  estándar**, or both (Productos → Production tab).
- The recipe's supplies must be active supplies whose default account has an inventory
  (`existencias`) account, and their quantities must use the supply's current sub-unit.
- The supplies must have stock in the supplies warehouse for the close to consume them.

### Business rules and rationale (Reglas y razón de negocio)

- **Units produced (`Unidades`):** the whole units of the day's pending lines of each product,
  across all its presentations.
- **Supply needed (`Necesario`):** for each product and supply, recipe quantity × units sold ÷
  yield, **rounded up** to the supply's sub-unit; the products' needs are added per supply.
  Example: a recipe of 20 portions with 50 g of salt, for 30 units sold, needs 75 g.
- **Available (`Disponible`):** the supply's stock in the supplies warehouse — its free stock plus
  its lots.
- **Consumed (`Se consume`):** the full need when the stock covers it. When it does not:
  - in strict mode the close is blocked with "Falta el insumo ...: se necesitan X y hay Y";
  - otherwise the available stock is **shared among the products in proportion to what each
    needs** (the rounding remainder goes to the largest need), so everything available is
    consumed and the products are produced short of that supply. Rows short of a supply are
    tinted, and a short product is marked in **Productos producidos**.
- **Which stock is consumed:** first the supply's free stock (without lot), then its lots from the
  oldest lot onward.
- **Cost of a produced product (`Costo según`, `Valor est.`):**
  - **Costo estándar**: when the product has a standard cost, its value is standard cost × units,
    whatever the supplies cost. Its supplies are still consumed.
  - **Receta**: otherwise, the value of the supplies it consumes (at the supply's average cost in
    the supplies warehouse) plus **Costo extra por unidad** × units.
  The header shows the number of sales and the estimated total value.
- **Blocking reasons:** the red box "El cierre no se puede realizar:" lists every reason, and
  **Aceptar y cerrar** stays disabled while any exists. The possible reasons are:
  - a product with neither recipe nor standard cost ("no tiene receta ni costo estándar");
  - a recipe supply that no longer exists or is inactive;
  - a supply whose account is not an inventory account ("que no es de existencias");
  - a recipe saved with an earlier sub-unit of the supply ("vuelva a guardarla");
  - a supply whose stock is kept in a sub-unit incompatible with the current one;
  - in strict mode, a short supply;
  - a product with neither a standard cost nor an extra cost per unit whose consumed supplies are
    worth nothing ("no tiene costo: sus insumos consumidos no tienen valor"), which also happens in
    non-strict mode when none of its supplies has stock. An extra cost per unit alone is enough
    value for the product to enter;
  - an earlier day still pending, or a day with no pending sales in the warehouse.
- **Movement date warning:** the close's stock movements are dated the day closed or yesterday,
  whichever is later. When they differ, the amber notice reads "Este cierre se registrará con
  fecha X, no D." The design keeps movements from landing in an older, possibly declared, period
  while letting a night or next-morning close book on the day itself (docs/PRODUCTION_PLAN.md §5.2).
- In strict mode the summary shows "Modo estricto: un insumo faltante bloquea el cierre."

### Result and side effects (Resultado y efectos)

The summary writes nothing; it is recalculated every time it loads.

### Limitations (Limitaciones)

- The values are an estimate; the close books the value the stock actually gives at the moment it
  runs.
- Only one supplies warehouse per close: a recipe cannot take one supply from one warehouse and
  another from a second one.
- A product with sub-units sold in fractions is produced in whole units (see the rules below).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cuánto insumo se va a consumir?` Mire la columna "Se consume" de "Insumos consumidos".
- `¿Por qué el botón Aceptar y cerrar está deshabilitado?` El resumen muestra en rojo por qué el
  cierre no se puede realizar.
- `¿Qué pasa si no me alcanza un insumo?` Sin modo estricto, lo disponible se reparte entre los
  productos según lo que necesita cada uno; con modo estricto, el cierre se bloquea.
- `¿De qué almacén salen los insumos?` Del "Almacén de insumos"; por defecto es el almacén de
  ventas y puede cambiarse.
- `¿Por qué dice "Costo estándar" y no "Receta"?` El producto tiene un costo estándar, que
  reemplaza el costo de los insumos.
- Search terms: `resumen del cierre`, `insumos faltantes`, `faltan insumos`, `reparto
  proporcional`, `costo de producción`, `costo estándar`, `receta`, `rendimiento`, `lote más
  antiguo`, `fecha del cierre`.

<!-- DOC-ID: capability.run-close -->
## Close the day (Cerrar el día)

### User intention (Intención del usuario)

Register the production of what was sold on the day, consume its supplies and take the sold units
out of stock, so stock, costs and the sales' status are up to date.

### Where to find it (Dónde encontrarlo)

On the summary of the oldest pending day, press the green **Aceptar y cerrar** button and confirm
"¿Cerrar el <día>? Se consumen los insumos y los productos ingresan y salen del almacén. No se
puede deshacer."

### Required information and prerequisites (Requisitos previos)

A summary with no blocking reasons. Nothing else is entered: the close uses the warehouses and day
of the summary on screen.

### Business rules and rationale (Reglas y razón de negocio)

- The close recalculates the summary when it runs. If the day's sales, what the close would
  consume or what a product is valued at changed since it was shown — a sale of the day delivered
  later, paid or annulled, supply stock that moved, a recipe change that alters the consumption, an
  edited standard cost or extra cost per unit — it is refused with "El día cambió
  desde que se revisó el resumen: revíselo de nuevo." and the page loads the new summary.
- While it runs, a payment or annulment of one of the day's pending sales waits for it to finish,
  so none of them falls half inside the close.
- A future day is refused ("No se puede cerrar un día futuro.").
- Each made product enters at its value, split among its presentations by units, right before its
  sales leave, so the sales' exits leave at the cost that entry just set.
- A product whose final value is zero is refused ("... no tiene costo para su ingreso de
  producción.").

### Result and side effects (Resultado y efectos)

In one operation, all or nothing:

1. A production close is saved and gets its number; "Cierre N° X registrado." confirms it.
2. A **Consumo de producción** exit is written in the supplies warehouse for each product, supply
   and stock bucket (free stock or lot) consumed.
3. An **Ingreso de producción** entry is written in the sales warehouse for each product and
   presentation, at the production value.
4. An **Entrega (venta)** exit is written for each settled sale line, linked to its sale.
5. The reservations those lines held are released, which frees the warehouse's sellable stock and
   the product's **Sobreventa permitida** margin again.
6. The sales lose the **stock por regularizar** mark and the yellow **R**, and leave the **Por
   regularizar** tab. Their status (Entregado / Finalizado), payment and debt do not change.

After the close the page shows the next pending day, or the same day again if sales of it were
delivered meanwhile, and the close appears under **Cierres anteriores**.

### Limitations (Limitaciones)

- **A close cannot be undone or annulled.** A mistake is corrected with manual stock movements
  on Stock Changes (docs/PRODUCTION_PLAN.md, decision P11).
- Annulling a settled sale afterwards re-enters its made products into the sales warehouse (its
  delivery exits are returned as usual); the supplies the close consumed are not returned.
- The close covers only the sales pending when it runs. A sale of the same day delivered later
  leaves the day pending again, and the day is closed again for it.
- Movements are never dated earlier than yesterday: a day closed late is booked on yesterday.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo regularizo el stock de los productos fabricados vendidos?` Cierre el día en Producción →
  Cierre del día.
- `¿Cómo saco una venta de "Por regularizar"?` Cerrando el día de esa venta en el almacén donde se
  entregó.
- `¿Puedo anular un cierre?` No; corrija con movimientos manuales de stock.
- `"El día cambió desde que se revisó el resumen"` Revise el nuevo resumen y vuelva a aceptar.
- `¿Puedo cerrar el mismo día dos veces?` Sí, cuando hay ventas de ese día entregadas después del
  primer cierre.
- Search terms: `aceptar y cerrar`, `cerrar el día`, `consumo de producción`, `ingreso de
  producción`, `descargar insumos`, `regularizar stock`, `liberar reserva`, `deshacer cierre`.

<!-- DOC-ID: capability.close-history -->
## Review past closes (Consultar cierres anteriores)

Below the summary, **Cierres anteriores** lists the closes of the chosen sales warehouse, newest
first: **N°**, **Día** closed, **Productos** (units × product), number of **Ventas** settled,
**Valor** booked for the production, and the **Alm. insumos** used. The list is read-only; the stock
movements of a close appear in Warehouse Movements as **Consumo de producción** and **Ingreso de
producción**, with the close number as their document.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- One close covers one sales warehouse, one supplies warehouse and one day.
- Oldest day first: a day is closed only when every earlier day of the warehouse is.
- **Sub-unit sales:** the close produces whole units. A line sold in sub-units produces a whole
  unit while its exit takes only the sub-units sold, so the rest of that unit stays in the sales
  warehouse's stock of the product.
- The supplies are always consumed by recipe, even when a standard cost sets the product's value.
- Strict mode is read from the company flags each time the summary loads and is stored on the
  close; it can be changed at any time in **Configuración → Flags de la Empresa → Producción**.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Hay ventas pendientes de un día anterior (...): ciérrelo primero.":** an older day of the
  warehouse is still pending; close it first.
- **"... no tiene receta ni costo estándar.":** open the product on Productos → Production and add
  its recipe or a standard cost.
- **"La receta de ... usa una sub-unidad anterior del insumo ...: vuelva a guardarla.":** the
  supply's sub-unit changed; remove that recipe line, add the supply again and save the product.
- **"El insumo ... tiene la cuenta ..., que no es de existencias.":** change the supply's default
  account on Insumos & Materiales to an inventory purchase account.
- **"Falta el insumo ...: se necesitan X y hay Y.":** strict mode is on. Bring stock into the
  supplies warehouse (purchase or manual entry), pick another **Almacén de insumos**, or turn the
  flag off to share what is available.
- **"... no tiene costo: sus insumos consumidos no tienen valor y no tiene costo estándar ni costo
  extra.":** the supplies consumed have no stock value (or no stock at all) in the supplies
  warehouse. Register their purchase or value, or set a standard cost or an extra cost per unit on
  the product.
- **"El día cambió desde que se revisó el resumen: revíselo de nuevo.":** something changed while
  the summary was open; the page reloads it, review and accept again.
- **A day closed earlier is pending again:** a sale registered that day was delivered after the
  close; close the day again.
- **The made product shows leftover stock after a close:** it was sold in sub-units; the close
  produced a whole unit and only the fraction left.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: the **Por regularizar** tab
  lists the sales pending a close. Any delivery of a made product creates them: a sale delivered at
  the **Point of Sale (Punto de Venta)** or in Sales Management, a web order delivered in **Pedidos
  Web** (`/sales/online-orders`), or a restaurant table charged in **Atención de mesas**
  (`/dining/floor-service`), which always delivers what it bills.
- **Products (Productos)** at `/production/products`: the **Production (Producción)** tab holds the
  recipe, yield, extra cost per unit and standard cost the close uses.
- **Supplies & Materials (Insumos & Materiales)** at `/production/supplies-materials`: the supplies,
  their sub-units and default account.
- **Warehouse Movements (Movimientos Almacén)** at `/logistics/warehouse-movements`: the
  **Consumo de producción**, **Ingreso de producción** and **Entrega (venta)** movements a close
  writes.
- **Stock Changes (Cambios Stock)** at `/logistics/products-stock`: supply stock, and manual
  corrections after a mistaken close.
- **Configuration (Configuración)** at `/company/configuration`: the strict-mode flag in **Flags de
  la Empresa**, section **Producción**.

### FILES

```yaml
# Replace pending only after the completed documentation is reviewed against exact file bytes.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/production/daily-close/+page.svelte
    role: page
    hash: sha256:b6029269b3fbf251500ed5df8c5dcde0d32554ca84b24c90e2790b6e5174bb5b
    supports: [page-purpose, capability.review-pending-days, capability.preview-close, capability.run-close, capability.close-history]
  - path: frontend/routes/production/daily-close/DailyCloseSummary.svelte
    role: user-interface
    hash: sha256:5a9f89f3ecc509957f8896e1918e5f7a40d9fe17603700790fc7895f816b307b
    supports: [capability.preview-close]
  - path: frontend/routes/production/daily-close/daily-close.ts
    role: business-logic
    hash: sha256:53af9a301e8515295aed0bd9ac1883a8d7679e405a54041760a7c694fa5f17de
    supports: [capability.review-pending-days, capability.preview-close, capability.close-history]
  - path: frontend/routes/production/daily-close/daily-close.svelte.ts
    role: frontend-service
    hash: sha256:aeb806bcc046b2c1e09b63f0ec29ef3eaf5dcff3d1c3d7cdd6515b7996b5d8d8
    supports: [capability.review-pending-days, capability.preview-close, capability.run-close, capability.close-history]
  - path: backend/production/daily_close.go
    role: backend-handler
    hash: sha256:7ae0b03e927362b3c467d29657303b41f9f0dbc979ad54b92f79807ab9b025a1
    supports: [concepts, capability.review-pending-days, capability.preview-close, capability.run-close, capability.close-history, rules, troubleshooting]
  - path: backend/production/types/daily_close.go
    role: business-logic
    hash: sha256:e398a9b2b9febe7eb3bc80cf921a4fcf49ab344a71e214bc9fb4966957d9b64d
    supports: [capability.preview-close, rules, troubleshooting]
  - path: backend/production/types/production_close.go
    role: data-model
    hash: sha256:bd04128b9bc99e4ad225a1dcc42bf9f909539e8a3014ad90e87249e399ca61ff
    supports: [concepts, capability.run-close, capability.close-history]
  - path: backend/production/types/product_production.go
    role: business-logic
    hash: sha256:dbfd1a1b4ef1d848c9b56f06b4b2ce3ecd10a5c07ce66fad45b626109c672901
    supports: [concepts, capability.preview-close, rules]
  - path: backend/company_flags.toml
    role: shared-domain
    hash: sha256:41281f5a4686ceecc28bc2c95a3c6deb4041e03771e7ad8533d07bb2afaf8799
    supports: [concepts, rules, troubleshooting]
  - path: frontend/routes/company/configuration/CompanyFlagsPanel.svelte
    role: user-interface
    hash: sha256:64f7141389b5db13573738c9fc49fcf6159672e69e3707e52a831ca4ab1758e0
    supports: [rules, related-pages]
  - path: backend/sales/types/pending_delivery.go
    role: business-logic
    hash: sha256:fdefbd8a318a6cafff7823707c0c42bbf28c5ca1077b9b54dccd150094e009fa
    supports: [concepts, capability.review-pending-days, capability.run-close, rules]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:3814c0a52410ab92d7a5e9ee87ee07f90c74815d15678e7391816183b64598c2
    supports: [concepts, capability.review-pending-days]
  - path: backend/sales/sale_order_create.go
    role: business-logic
    hash: sha256:0601dd000d061f432f4a7dfa0228d673b2debc0a744448d6ddfb78a30978c78d
    supports: [concepts, related-pages]
  - path: backend/sales/saleissue/types/new_sale.go
    role: business-logic
    hash: sha256:79480f8435d4182f16c8ebca2b5717be396423cd73a571f5ef1e2a84a229e33e
    supports: [page-purpose, concepts, related-pages]
  - path: backend/sales/types/sale_actions.go
    role: business-logic
    hash: sha256:ad4c117e0bce8f4c8e88f85e6aa3677090935cf6b6127c944f72bf1601413109
    supports: [page-purpose, concepts]
  - path: backend/sales/online_sale_order.go
    role: business-logic
    hash: sha256:6d521468f6ba3c7074be0ca54529ba9cc3e7453370483539546febb16b0dedac
    supports: [related-pages]
  - path: backend/dining/table_order_charge.go
    role: business-logic
    hash: sha256:32e32d0a5349d8e43895fec3d0ca47a09fed34037cc3534d1f69b290b9f25632
    supports: [related-pages]
  - path: backend/sales/sale_order_annul.go
    role: business-logic
    hash: sha256:5d184227f73c0fc0630fb2357132bf09bc4dd39f2bda73d74d9676b611aa36f7
    supports: [capability.run-close]
  - path: backend/logistics/types/stock_movement_type.go
    role: shared-domain
    hash: sha256:2556d42f6e9cc99c8c179a8536a80d26bf41702648d11efd34f6e9bfc3de4b8f
    supports: [concepts, capability.preview-close, capability.run-close, capability.close-history]
  - path: backend/logistics/types/stock_movement_apply.go
    role: business-logic
    hash: sha256:d8cca9053dd870e394777a265228c90516a9f1f9dd9208c69d7f97a3e1c39496
    supports: [capability.preview-close, capability.run-close]
  - path: frontend/core/stock-movement-type.ts
    role: shared-domain
    hash: sha256:cfcee015c4d6fadb03f9a9e5d66064e93d734cfa5216e2d36996c47140eecc9c
    supports: [concepts, capability.close-history, related-pages]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:cb61a6f79a034e4d468b46c173733b815f0af9007a01d1f3e69b92d358913d60
    supports: [page-purpose, capability.review-pending-days, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:4f2d4c16777913fddf196959c4341290e69a12a78323a5b95f8fa5d4738ca34d
    supports: [capability.review-pending-days, related-pages]
  - path: backend/access.toml
    role: permissions
    hash: sha256:c19cf85130065dcc0f839796e1e9b4b98a1b78df465ccb20d2b63b59a1a94e59
    supports: [capability.review-pending-days]
  - path: docs/PRODUCTION_PLAN.md
    role: reference-document
    hash: sha256:089b2702c775c88abb30df0e23c565080216556a2ef43c9fc95635bb2431d15e
    supports: [capability.preview-close, capability.run-close]
```
