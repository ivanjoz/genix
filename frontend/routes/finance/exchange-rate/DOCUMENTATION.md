---
schema: 1
page_id: finance.exchange-rate
route: /finance/exchange-rate
title: Exchange Rate (Tipo de Cambio)
status: implemented
visibility: tenant
description_en: >-
  Continuous calendar of the USD to PEN exchange rate. Weekdays are the columns, each split into
  Día, Compra and Venta, and the rows are the weeks of each month under its own month title. Days
  the company has not loaded show the BCRP interbank rate in gray as a reference; what the company
  types shows in blue. Rates are typed in with three decimals and one Save writes every month that
  was edited.
description_es: >-
  Calendario continuo del tipo de cambio dólar a soles. Los días de la semana son las columnas,
  cada una dividida en Día, Compra y Venta, y las filas son las semanas de cada mes bajo su título
  de mes. Los días que la empresa no ha cargado muestran en gris el tipo de cambio interbancario
  del BCRP como referencia; lo que la empresa escribe se muestra en azul. Los tipos de cambio se
  escriben con tres decimales y un solo Guardar escribe todos los meses editados.
---

# Exchange Rate (Tipo de Cambio)

<!-- DOC-ID: page-purpose -->
## Page purpose

Exchange Rate (`Tipo de Cambio`) is where the company keeps the daily USD to PEN rate
(`tipo de cambio dólar a soles`) it works with, day by day. The page is a maintainer
(`mantenedor`) drawn as a calendar: it stores the rates and nothing else. It does not download
them from SUNAT, and it does not convert any document by itself — it is the table other work,
such as the Point of Sale, reads the rate from.

As a reference while loading, every day the company has not filled in shows, in gray, the
interbank rate (`tipo de cambio interbancario`) published by the BCRP. That gray number is not
stored and is not the company's rate: it is there to be read or copied over. It is, however, the
fallback the **Point of Sale (Punto de Venta)** uses for a day the company left empty — see
Limitations below.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- The buy rate (`tipo de cambio compra`) and the sell rate (`tipo de cambio venta`) are the two
  rates published for the same day. Each weekday column holds both, side by side.
- A rate is stored with three decimals (`3.752`). Genix keeps it internally as an integer
  multiplied by 1000, which is why the cell always shows three decimal places.
- A rate in **blue** is the company's own: somebody typed it here and it is stored. A rate in
  **gray** is the BCRP default (`por defecto`): published data shown as a reference, not saved
  anywhere. The colour is the only thing that separates them, and the legend at the top of the page
  repeats it.
- An empty cell means neither exists (`sin tipo de cambio`) — typically a Saturday, a Sunday, or a
  holiday (`feriado`), when the market does not trade and the BCRP publishes nothing, or a recent
  day the BCRP has not published yet. It is not a rate of zero.
- Rates are stored one row per month (`un registro por mes`), identified by the month itself in
  YYMM form: January 2026 is `2601`. This is why saving touches whole months, not single days.

<!-- DOC-ID: capability.load-rates -->
## Load or correct the daily rates (Cargar o corregir los tipos de cambio)

### User intention (Intención del usuario)

Record the daily exchange rate so accounting, purchases, and sales in dollars have an official
company rate to work from — either loading today's rate, or filling a past month.

### Where to find it (Dónde encontrarlo)

Open **Finances (Finanzas) → Exchange Rate (Tipo de Cambio)** at `/finance/exchange-rate`.

- The calendar opens on the current month and runs backwards, month after month, covering the
  last five years. There is no year selector: scrolling down is how you reach an older month.
- Each month starts with its own title row (`SEPTIEMBRE 2026`), followed by one row per week of
  that month.
- The columns are the weekdays from **Lunes** to **Domingo**, and each one holds three cells: the
  day number — marked with a dark badge, under no label because it needs none — and its **Compra**
  and **Venta** rates.
- Click a Compra or Venta cell to type in it, then press Tab or click elsewhere to confirm it. A
  gray cell opens with the BCRP value already in it, so it can be read or edited: leaving it as it
  is — even confirming the same number — changes nothing and the cell stays gray. Typing a
  different number is what makes the day the company's own, and the cell turns blue.
- **Save (Guardar)** sits at the top right and shows, in parentheses, how many months will be
  written.

### Required information and prerequisites (Requisitos previos)

- Nothing needs to exist first: every day of every month in range is already editable.
- A cell accepts a number with up to three decimals, such as `3.752`. A value with more decimals is
  rounded to three.

### Business rules and rationale (Reglas y razón de negocio)

- The first and last week of a month have empty slots where the week runs into the neighbouring
  month; those cells carry no day number and cannot be typed into. Each day is edited once, in its
  own month.
- Emptying a cell, or typing something that is not a rate (letters, a negative number, or a value
  above 1000.000), leaves the day with no rate instead of storing the value. The server rejects the
  same cases, so a rate above 1000.000 never reaches the database.
- **Save (Guardar)** stays disabled while there is nothing to save, and it sends only the months
  that were edited — not everything on screen. The gray BCRP rates are never saved, no matter how
  many of them are on screen.
- The BCRP publishes the interbank rate with a couple of business days' delay, so the last days of
  the current month usually have no gray reference yet. The most recent ones may also be an
  estimate the source fills in while the BCRP catches up (within ±0.5%); the page does not mark
  them apart.

### Result and side effects (Resultado y efectos)

Each edited month is written as one record holding its buy and sell arrays. The page keeps itself
in sync afterwards: rates saved from another session or another tab appear on the next refresh,
and a refresh landing while you are typing does not overwrite the months you have open.

### Limitations (Limitaciones)

- Only the USD to PEN rate is kept. There is no currency selector and no second currency pair.
- Rates are not imported from SUNAT and cannot be uploaded from Excel; they are typed in.
- The gray BCRP reference is **not valid for tax purposes**: for IGV, accounting books and exchange
  difference, Peruvian rules point at the SUNAT/SBS rate, which is not what is shown. It is the
  market rate, shown to help while loading.
- The **Point of Sale (Punto de Venta)** reads the sell rate day by day the way this calendar shows
  it: the company's own (blue) rate when there is one, otherwise the gray BCRP rate. It uses the
  most recent day of the last 7 that has either, then applies the company's exchange-rate spread.
  Saving a blue rate for a day therefore replaces the BCRP rate the till would use for that day.
  If the BCRP source cannot be reached, the till only has the company's saved rates.
- The reference comes from an external source over the internet. Without a connection the gray
  values simply do not appear; nothing else on the page changes.
- The calendar reaches five years back from the current month. Older months are not shown.
- Documents already issued are not recalculated when a rate is corrected — the page stores the
  rate, it does not revalue anything.
- There is no per-day audit: a corrected rate replaces the previous one, and the record only keeps
  who last wrote the month and when.

### Common questions and vocabulary (Preguntas y vocabulario)

- "¿Dónde cargo el tipo de cambio del día?" → Finanzas → Tipo de Cambio; el mes actual está arriba,
  busque la semana y la columna del día.
- "¿Cómo pongo el tipo de cambio de venta?" (`TC venta`, `tipo cambio venta`) → en la columna
  Venta del mismo día.
- "¿Por qué hay celdas vacías al inicio del mes?" → esa semana empieza en el mes anterior; esos
  días se cargan en su propio mes.
- "Cargué el mes y no se guardó" → falta presionar **Guardar**; el botón indica cuántos meses
  están pendientes.
- "¿Por qué algunos números están en gris y otros en azul?" → el gris es el interbancario del BCRP
  que se muestra por defecto y no está guardado; el azul es el tipo de cambio que cargó la empresa.
- "¿El gris ya está guardado?" → no; solo se guarda lo que se escribe (azul). Aun así, el Punto de
  Venta usa el gris del BCRP para convertir precios en un día que la empresa no cargó.
- "¿Tengo que cargar el tipo de cambio para vender en dólares?" → no es obligatorio: si falta el de
  la empresa, el Punto de Venta usa el del BCRP de los últimos 7 días. Cárguelo si quiere usar su
  propio tipo de cambio.
- "¿Ese tipo de cambio es el de SUNAT?" → no, es el interbancario del BCRP y no sirve para efectos
  tributarios; el de SUNAT hay que cargarlo a mano.
- Terms users mix: `tipo de cambio`, `TC`, `tasa de cambio`, `dólar`, `compra / venta`,
  `tipo de cambio SUNAT`.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash & Banks (Cajas & Bancos)** keeps each account in its own currency and does not convert
  between them; the rate kept here is what a conversion would be based on.
- **Expenses (Gastos)** and **Purchase Orders (Órdenes de Compra)** record their own currency per
  document (PEN or USD) and do not read this page automatically.
- **Invoicing (Facturación Electrónica)** issues documents in the currency of the sale; the rate
  loaded here is the company reference, not a value the electronic document takes by itself.
- **Point of Sale (Punto de Venta)** at `/sales/sale_order_create`: converts prices for a dollar
  sale, or a sale mixing currencies, with the latest sell rate of the last 7 days — the company's
  rate from this page, or the BCRP rate for a day left empty — adjusted by the **Spread de tipo de
  cambio** company flag. With neither available it refuses the sale and points back here.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0232bd261462eee1bcbf2bd20cc37658f85aa3a0a4aa355af975bf5b32c2be2c
    supports: [page-purpose, capability.load-rates, related-pages]
  - path: frontend/routes/finance/exchange-rate/+page.svelte
    role: page
    hash: sha256:01930730b1e278f9f636c90109f8f5447314b1517049ad2cb04d85b0187dac29
    supports: [page-purpose, concepts, capability.load-rates]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.ts
    role: business-logic
    hash: sha256:55738f078d629f60276bbfbbfb58b46b3b1b7ba7a9626d06504b2e909294bf28
    supports: [concepts, capability.load-rates, related-pages]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.svelte.ts
    role: frontend-service
    hash: sha256:f5f401b8f1e32dd79eba283d2a7c9b64bb90eca0b81db938049b32a2db8ab333
    supports: [capability.load-rates]
  - path: backend/finance/exchange_rate.go
    role: backend-handler
    hash: sha256:381616a2ae84db75ae191832ea74979ee531a3325bb98a1096c9efe6077130b3
    supports: [capability.load-rates]
  - path: backend/finance/types/exchange_rate.go
    role: data-model
    hash: sha256:7dc6dbf27632d692cf814d4376e7284957187874e2a0e318a8d861e4dd0de377
    supports: [concepts, capability.load-rates, related-pages]
  - path: backend/finance/types/bcrp_rates.go
    role: business-logic
    hash: sha256:814f197ccbd7e90bbd1b611ac3dc4291a208c7b86e621f5bea42462036b6cf8f
    supports: [page-purpose, capability.load-rates, related-pages]
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: shared-domain
    hash: sha256:19d9bce55dd5a19b8b09064cf22cf0e26dc632f925982e044595ff010a3f3135
    supports: [page-purpose, capability.load-rates, related-pages]
  - path: backend/sales/sale_order_currency.go
    role: shared-domain
    hash: sha256:186747cd3a2892beeef7428ff285df67edc44caa494e2b3b0b2985aa46b92a6f
    supports: [related-pages]
```
