---
schema: 1
page_id: sales.sales-forecast
route: /sales/sales-forecast
title: Sales Forecast (Pronóstico de Ventas)
status: implemented
visibility: tenant
description_en: >-
  Daily demand forecast per site and product, computed from the sales history, the end-of-day stock
  and the commercial calendar. Shows a 4-week projection in units or soles, current stock, units sold
  in the last 28 days and a confidence score, with a chart and a list of data warnings. Model
  parameters are tuned with sliders.
description_es: >-
  Pronóstico diario de la demanda por sede y producto, calculado con el historial de ventas, el stock
  de cierre de cada día y el calendario comercial. Muestra la proyección de 4 semanas en unidades o en
  soles, el stock actual, lo vendido en los últimos 28 días y un índice de confianza, con un gráfico y
  una lista de advertencias de datos. Los parámetros del modelo se ajustan con deslizadores.
---

# Sales Forecast (Pronóstico de Ventas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sales Forecast (`Pronóstico de Ventas`) estimates how many units of each product will sell per day
in each site (`sede`) over the coming weeks (`proyección de demanda`, `previsión de ventas`). It learns
from the recorded sales, the end-of-day stock of each warehouse and the **Commercial Calendar
(Calendario Comercial)**, and shows a 4-week projection per product next to its current stock.

The forecast is calculated in the browser each time the page opens or a parameter changes. It saves
nothing in Genix: it does not create purchase orders, production orders or a sales plan, and it does
not change stock or prices. The manual weekly plan per product is a different page, **Sales Planning
(Proyección Ventas)**, and this forecast does not read it.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **series** is one product in one site. Units sold without a warehouse belong to no site and are
  left out (warning **Ventas sin sede**). Only products that sold in the history appear.
- The history ends **yesterday**; today's sales are still coming in. Its last **28 days** are held
  out of the learning and used to score the forecast, which is what **Confidence**, WAPE and MASE
  measure.
- The history starts no earlier than the company's first recorded sale, even if more days are asked.
- A **stockout** (`quiebre de stock`, `sin stock`) day is one with nothing on hand and no sale while the
  product's stock is tracked: its zero sales hide demand, so the forecast does not read it as demand.
- **Model (Modelo):** **Holt-Winters** for products that sell regularly, **Intermittent
  (Intermitente)** for products that sell only now and then. **Scenario (Esc.)** A, B or C depends on
  how long the product has history: from B (180 days) the yearly season and the calendar effects
  apply; A forecasts without them.
- **Confidence (Confianza)** is a 0–1 score, 1 the best: the average of the share of sales the
  forecast got right and its advantage over repeating last week. Below 0.50 it shows in red.

<!-- DOC-ID: capability.read-forecast -->
## Read the forecast (Consultar el pronóstico)

### User intention (Intención del usuario)

Know how much of each product will sell in the next four weeks, which products are running out, and
how far the forecast can be trusted, for example before buying or producing.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Reports (Reportes) → Sales Forecast (Pronóstico de Ventas)** at
`/sales/sales-forecast`. Use **Site (Sede)** (empty means **All sites (Todas las sedes)**) and the
**Product…** search box. The status text next to them shows progress (**Leyendo ventas y stock…**,
**Ajustando … series…**) and finally how many products were calculated.

### Required information and prerequisites (Requisitos previos)

Sales recorded for at least 56 days (28 to learn plus 28 held out). With fewer, the page shows
"Ventas registradas por menos de 56 días: no alcanza para pronosticar." and with no sales at all
"No hay ventas en la historia.".

### Business rules and rationale (Reglas y razón de negocio)

- Table columns: **Product (Producto)**; **Stock**, the stock today, red when the product is out of
  stock now; **Sold 28d (Vendido 28d)**, units sold in the last 28 days; **4-Week Projection
  (Proyección 4 Semanas)**, one column per week headed by its first day, with the week's total over
  small daily bars; and **Confidence (Confianza)**.
- With **All sites**, each product shows as one row adding up its sites. Choosing a site shows that
  site's rows only.
- The chart shows the last 56 days of sales and stock, then the forecast for the whole horizon. It
  adds up every row the filters leave; clicking a row shows that product only, and **Show all (Ver
  todos)** goes back to the total.
- **Calculate in money (Calcular en dinero)** turns the projection into soles at each product's
  current list price; a dollar price is converted at the latest selling exchange rate. Stock and
  Sold stay in units. A product without a price that can be converted shows `-` ("Sin precio en
  soles").
- **Show statistical indicators (Mostrar Indicadores Estadísticos)** adds **Model**, **Esc.**,
  **WAPE** and **MASE**. MASE above 1, in red, means the forecast missed more than simply repeating
  last week's same weekday. **Show bars in cells (Mostrar barras en celdas)** hides or shows the
  daily bars.

### Result and side effects (Resultado y efectos)

Nothing is stored; leaving the page discards the result.

### Limitations (Limitaciones)

- The table always shows the first 4 weeks, even when the horizon is longer; the chart shows the full
  horizon.
- The money projection uses today's list price, not the prices actually charged.
- No export or purchase suggestion is generated from this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuánto voy a vender la próxima semana? ¿Qué productos se van a quedar sin stock?
- ¿Cuánto venderé en soles este mes? → **Calcular en dinero**.
- ¿Qué tan confiable es el pronóstico? ¿Qué significa WAPE o MASE?
- Términos: pronóstico, forecast, proyección de demanda, previsión, quiebre de stock, estacionalidad.

<!-- DOC-ID: capability.tune-parameters -->
## Tune the forecast parameters (Ajustar los parámetros)

### User intention (Intención del usuario)

Adapt the forecast to the business: more or less history, a longer horizon, or how stockouts and
trends are treated.

### Where to find it (Dónde encontrarlo)

The left column of the page. Each slider recalculates the forecast when released. **Restore defaults
(Restaurar valores)** puts every parameter back to its default.

### Business rules and rationale (Reglas y razón de negocio)

- **History days (Días de historia):** 90 to 730, default 365. Days of sales the models learn from. A
  year or more learns the yearly season and the calendar; changing it reads the sales again.
- **Forecast horizon (Horizonte de pronóstico):** 7 to 90 days, default 28.
- **Stockout threshold (Umbral de quiebre):** 0 to 20 units, default 0. A day that closes at or below
  this stock had its demand limited: its sales count as a minimum, not as the demand.
- **Empty shelf fade (Desvanecer sin stock):** 7 to 90 days, default 28. Days without stock the
  forecast holds; after them it halves every as many days, as for a product no longer sold.
- **Trend reach (Alcance de tendencia):** 0.80 to 0.98, default 0.90. How far a rising or falling trend
  is carried into the future.
- **Weekly shape from category (Forma semanal de la categoría):** 0 to 26 weeks, default 8. Weeks of
  history a product needs before its own weekday pattern weighs as much as its category's; 0 uses only
  the product's.
- **Intermittency threshold (Umbral de intermitencia):** 1 to 3, default 1.32. A product selling on
  average less often than once every this many days is forecast as intermittent.

### Result and side effects (Resultado y efectos)

The parameters are kept in this browser for the company, so they return the next time the page is
opened here. They are not shared with other users or devices.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo hago que el pronóstico cubra 60 días? → **Horizonte de pronóstico**.
- ¿Por qué un producto agotado sigue con pronóstico? → **Desvanecer sin stock**.

<!-- DOC-ID: capability.warnings -->
## Review the forecast warnings (Advertencias del pronóstico)

### User intention (Intención del usuario)

Find what in the data makes the forecast less reliable, and fix it.

### Where to find it (Dónde encontrarlo)

The **Warnings (Advertencias)** button shows how many there are; it is red when a high-severity
warning exists. It opens **Forecast warnings (Advertencias del pronóstico)**, grouped as History,
Data, Calendar, Stockouts and Accuracy, with the affected products listed (at most 60, then a
count).

### Business rules and rationale (Reglas y razón de negocio)

- **History:** **Ventas registradas por menos tiempo**, **Historia menor a un año** (no product gets
  the yearly season), **Productos con poca historia** (under 180 days: no yearly season or calendar
  effects), **Productos con pocas ventas** (fewer than 8 days with sales).
- **Data:** **Ventas sin sede**, **El stock cambió mientras se leía** (reload the page), **Stock
  negativo** (sold without stock on record: receipts registered late or missing), **Productos sin
  categoría** (they share no weekly or yearly shape with similar products), **Productos vendidos fuera
  del catálogo** (forecast under their ID).
- **Calendar:** **Años sin feriados**, **Sin días de pago**, **Sin eventos de la empresa**.
- **Stockouts:** **Sin stock ahora** (days without stock up to yesterday) and **Quiebres frecuentes**
  (without stock on 20% or more of the days: sales understate demand).
- **Accuracy:** **Peor que repetir la semana pasada**.
- With nothing to report the panel says "Nada que reportar: los datos se ven completos.".

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué el pronóstico tiene advertencias? ¿Qué significa "Stock negativo" o "Ventas sin sede"?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No alcanza para pronosticar":** fewer than 56 days of sales; wait until more sales are recorded.
- **Calendar warnings:** add the holidays, paydays and campaigns in **Commercial Calendar (Calendario
  Comercial)**.
- **Productos sin categoría:** assign a category to those products, so they share the weekly and
  yearly shape of similar products.
- **Stock negativo:** register the stock receipts that are missing or late, so stockout days are
  recognized.
- **A forecast in red confidence:** review its parameters or its data, as the accuracy warning says.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Commercial Calendar (Calendario Comercial):** holidays, paydays, gratificación and commercial dates
  the forecast learns from.
- **Sales Planning (Proyección Ventas):** a manual weekly quantity plan per product with seasonality
  curves; it is entered by hand and is not this forecast.
- **Sales Report (Reporte Ventas)** and **Sales Charts (Gráficos Ventas):** the recorded sales
  themselves.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.read-forecast, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.read-forecast]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [page-purpose]
  - path: frontend/routes/sales/sales-forecast/+page.svelte
    role: page
    hash: sha256:aa81532f43e3bd021397d8472b7e9110ad2bdc6c1f6655db0592f6ffb0c6a823
    supports: [page-purpose, concepts, capability.read-forecast, capability.tune-parameters, capability.warnings]
  - path: frontend/routes/sales/sales-forecast/sales-forecast.svelte.ts
    role: frontend-service
    hash: sha256:0cb127773179e6d1e0ec0bc580ee1c7b215a0f44151ef112a44c1b592c585e64
    supports: [page-purpose, capability.read-forecast, capability.tune-parameters, capability.warnings]
  - path: frontend/routes/sales/sales-forecast/sales-forecast.ts
    role: business-logic
    hash: sha256:358eebaf98bc71d874b22b11257982dc874353e0b403997f824d603e94d436ba
    supports: [concepts, capability.read-forecast, capability.tune-parameters, capability.warnings, troubleshooting]
  - path: frontend/routes/sales/sales-forecast/sales-forecast.worker.ts
    role: business-logic
    hash: sha256:f2875e5e8e5b5a3919c8a13b4bf8a4dbdb8bacdb8362ebe9e1a20ca57c99488f
    supports: [concepts]
  - path: frontend/services/sales/sale-quantity.ts
    role: frontend-service
    hash: sha256:a23bd48d91d603b6677a08b0baec40d4e92651f04a4a0d77104c2ca8c9a9a238
    supports: [page-purpose]
  - path: forecast/src/engine.rs
    role: business-logic
    hash: sha256:f98c9e5b881df939a9dc15bf96966cf838c0097085e33b921ac3475fc49718df
    supports: [concepts]
  - path: frontend/services/sales/calendar-events.svelte.ts
    role: frontend-service
    hash: sha256:30ef9695b92696527fe97d7c21a3cbfef46c8dc5658b51389297f782a756cfc8
    supports: [page-purpose, related-pages]
  - path: frontend/routes/sales/sale_planning/SalesPlanningMantainer.svelte
    role: user-interface
    hash: sha256:ac954470858e2dd7c63f80297a2ae8d46b50f48f1faa61446c5d50bf4f2282b8
    supports: [page-purpose, related-pages]
```
