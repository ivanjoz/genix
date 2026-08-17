---
schema: 1
page_id: sales.sale_orders_charts
route: /sales/sale_orders_charts
title: Sales Charts (Gráficos Ventas)
status: implemented
visibility: tenant
description_en: >-
  Sales analytics comparing product performance and daily sales evolution. Switch between a By
  Product view (paid vs. pending amount or quantity per product, next to its current price) and
  a Daily Summary view (day-by-day received vs. receivable totals, delivery rate, and top
  products); a Weekly Summary view is not implemented yet.
description_es: >-
  Análisis de ventas para comparar el desempeño por producto y la evolución diaria. Alterna entre
  la vista Por Producto (monto o cantidad pagada y pendiente por producto, junto a su precio
  actual) y Resumen Diario (totales de ingresado vs. por cobrar, tasa de entrega y productos top
  por día); la vista Resumen Semanal aún no está implementada.
---

# Sales Charts (Gráficos Ventas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sales Charts, reached from **Commercial (Comercial) → Sales Charts (Gráficos Ventas)** at
`/sales/sale_orders_charts`, is the visual analytics page over the same daily sale-order summary
data used across Sales. Its on-page header actually reads **Gráficos de Ventas** (the side-menu
label is the shorter **Gráficos Ventas**). It turns already-created sale orders (`órdenes de
venta`, `pedidos`) into two chart views: comparing products against each other (By Product / Por
Producto) and tracking the day-by-day evolution of the whole company (Daily Summary / Resumen
Diario). It is a read-only reporting page: it does not create, edit, pay, or deliver an order, and
it does not let the user pick an arbitrary date range — both views work only over the sale-summary
records already loaded into the browser.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- The page reads a **daily sale summary (resumen diario de ventas)**: one row per calendar day per
  company with, for every product sold that day, the quantity sold, the quantity still pending
  delivery, the total invoiced amount, and the amount still unpaid (debt/`por cobrar`). Genix
  builds this summary incrementally from each sale order's create, payment, and delivery events
  (it is not recomputed from scratch every time the page opens), so the numbers shown here should
  match the totals a sale order accumulates over its lifecycle on **Sales Management**.
- **Amount (Monto)** vs. **Quantity (Cantidad)** is a single toggle — **Por Monto Facturado** /
  **Por Cantidad** — shared by both chart views on this page: switching it in one view keeps the
  same selection when the user moves to the other view.
- **Received/Ingresado** vs. **Receivable/Por Cobrar** always refers to the same product line
  split into its paid portion and its still-unpaid (debt) portion; **Delivered/Entregado** always
  refers to the quantity portion of a line whose delivery has completed vs. the quantity still
  pending. In quantity mode, "received" is read as delivered quantity, not payment.
- Sale-summary data currently loaded in the browser typically spans roughly the last 8 weeks (56
  days): the backend endpoint that feeds this page defaults an initial sync to `today − 56 days`
  through `today` when no other date range is given, and the page's own incremental sync
  afterwards only requests changes since the last sync, not a fixed historical window. In
  practice, an old sale order that is paid or delivered today can still surface here as a change
  for its original (older) date, so the exact oldest date present can vary by company.

<!-- DOC-ID: capability.by-product -->
## Compare products (Vista Por Producto)

### User intention (Intención del usuario)

Answer "which products are selling the most (or the least), and is their pending debt or pending
delivery piling up" by comparing every product against the others over the last weeks.

### Where to find it (Dónde encontrarlo)

Default view when opening `/sales/sale_orders_charts` (the first tab of the top options strip,
**By Product (Por Producto)**). Use the metric toggle (**Por Monto Facturado** / **Por Cantidad**)
and the search box to narrow the list; each match highlights the typed words inside the product
name.

### Required information and prerequisites (Requisitos previos)

No form input is required to see the chart; it renders automatically once the page's sale-summary
records and the product catalog are loaded. If a product referenced in the summary can no longer
be resolved against the loaded product catalog, its card falls back to the label
`Producto #<ID>` instead of its real name.

### Business rules and rationale (Reglas y razón de negocio)

- The chart always covers a fixed window of the **last 45 calendar days**, anchored to the latest
  date present in the loaded sale-summary records (or to today if no records are loaded yet) —
  not to the actual calendar "today" when the browser has stale, not-yet-refreshed data.
- Each product card totals its paid/unpaid amount or quantity across that 45-day window and the
  card list is sorted by that same total, from highest to lowest; the search box filters this
  sorted list by product name (every typed word must appear in the name) without changing the
  order.
- Per day, each card draws two stacked bars — **Ventas pagadas** (paid) and **Ventas no pagadas**
  (unpaid/pending) — plus a **Precio** line. That price line is the product's *current* catalog
  price (`FinalPrice`, falling back to `Price`) repeated flat across all 45 days on its own
  independent axis; it is not a historical record of what the product actually sold for on each
  day, so a recent price change will not be reflected on older bars. A product with no price at
  all shows `Sin precio` instead of a price value.

### Result and side effects (Resultado y efectos)

Purely a read: no record is created, updated, or changed by viewing or filtering this chart.

### Limitations (Limitaciones)

- There is no manual refresh button on this page; because the underlying sale-summary service is
  configured with a shorter cache lifetime specifically for chart pages, the numbers refresh
  automatically fairly often rather than needing a user action, but the page cannot force an
  immediate refresh on demand.
- The same message, **"No se encontraron productos con ventas en los últimos 45 días."**, appears
  both when there is genuinely no sales data in the window and when the search box simply matches
  no product name — there is no separate "no results for your search" message.
- The window is always the trailing 45 days from the newest loaded date; it cannot be changed to a
  custom date range, a single day, or a longer historical period from this view.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cuál es el producto que más se vendió?` — ordena las tarjetas de mayor a menor por el total del
  periodo.
- `¿Por qué el precio se ve como una línea recta?` — porque siempre grafica el precio actual del
  producto, no el precio histórico de cada venta.
- Search terms: `gráfico por producto`, `ventas por producto`, `productos más vendidos`, `monto
  facturado`, `cantidad vendida`, `precio del producto`, `sin precio`.

<!-- DOC-ID: capability.daily-summary -->
## Track the daily evolution (Vista Resumen Diario)

### User intention (Intención del usuario)

Answer "how did today (or any recent day) compare with the recent trend," see which products
carried each day, and check how much of each day's sales is still unpaid or undelivered.

### Where to find it (Dónde encontrarlo)

Select the second tab of the top options strip, **Daily Summary (Resumen Diario)**, on
`/sales/sale_orders_charts`. Only the metric toggle (**Por Monto Facturado** / **Por Cantidad**,
shared with the By Product view) is offered here; there is no product search box on this view.

### Required information and prerequisites (Requisitos previos)

No form input is required; the list renders automatically from the same sale-summary records used
by the By Product view. Unlike that view, this list is not fixed at 45 days: it spans every day
from the oldest date present in the currently loaded records through today, so its length depends
on how much sale-summary history the browser has already synced.

### Business rules and rationale (Reglas y razón de negocio)

- Each day is one card, most recent first, always shown even when a day has zero sales (a
  "missing" day in the data still renders as an empty-value card rather than being skipped).
- Every card shows three square indicators: **Received (Ingresado)** — the paid portion —,
  **Receivable (Por Cobrar)** — the still-unpaid portion — and **Delivered (Entregado)** — the
  delivered share of the day's total quantity, shown as a percentage plus the raw
  delivered/total count.
- Two variation indicators compare that day's total against (a) the average of the previous 7
  days, and (b) the same weekday exactly one week before; each only displays a value when the
  comparison base itself had sales greater than zero, otherwise it shows `--` instead of a
  misleading zero-based comparison.
- Below the indicators, the card lists up to the **top 10 products** of that day (by amount or
  quantity, following the shared metric toggle), each drawn as a horizontal bar sized relative to
  that day's largest product value (a minimum width is enforced so small values stay visible). A
  day with sales but no attributable product line shows **Sin Ventas** in that space.

### Result and side effects (Resultado y efectos)

Purely a read: no record is created, updated, or changed by viewing this chart.

### Limitations (Limitaciones)

- The page defines a distinct empty-state message, **"No se encontraron días con ventas en los
  últimos 45 días."**, for when the daily list itself would be empty, but in the current
  implementation at least one day card (even a zero-value placeholder) is always produced whenever
  any sale-summary record is loaded, so this particular message is not expected to appear; an
  empty state is only seen when there are no sale-summary records at all
  ("No hay registros de ventas para mostrar.").
- There is no drill-down from a daily card into the individual sale orders behind it; use **Sales
  Management (Gestión Ventas)** or **Sales Report (Reporte Ventas)** to see or filter the
  underlying orders for a specific day.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo van las ventas de hoy comparadas con la semana pasada?` — la flecha bajo la date compara
  contra el mismo día de la semana anterior y contra el promedio de los últimos 7 días.
- `¿Qué productos vendí hoy?` — la lista de "Top 10 productos" de la tarjeta del día.
- Search terms: `resumen diario`, `ventas del día`, `ingresado vs por cobrar`, `entregado`, `top 10
  productos`, `variación de ventas`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Both views read the exact same sale-summary records and product catalog loaded once for this
  page; switching tabs does not trigger a new fetch, and the Amount/Quantity metric toggle is a
  single shared state between them.
- **Weekly Summary (Resumen Semanal)**, the third tab of the top options strip, is not implemented
  yet: selecting it only shows the placeholder text "Weekly summary pending. / Resumen Semanal
  pendiente."
- Neither chart view lets the user create, edit, cancel, pay, or deliver a sale order; those
  actions live on **Sales Management (Gestión Ventas)**.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No hay registros de ventas para mostrar."** in either view: no sale-summary data has loaded
  yet for this company — this is the same message shown for a brand-new company with no sales and
  for a page that has not finished its initial sync, since the page has no separate loading
  indicator.
- **"No se encontraron productos con ventas en los últimos 45 días."** in By Product: either no
  product had sales in the trailing 45-day window, or the search box matched no product name;
  clear the search box to check which case applies.
- **A product shows `Producto #<ID>` instead of its name:** the product catalog loaded in the
  browser does not currently contain that product ID.
- **The price line on a product card looks wrong or outdated:** it always reflects the product's
  current price, not the price actually charged on past sales.
- **Numbers look slightly out of date:** this page has no manual refresh action; it relies on its
  own short-lived automatic cache to catch up, and syncing can be delayed by a few seconds/minutes.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: the operational queue for
  individual sale orders — use it to open, pay, deliver, or cancel a specific order that this
  chart only shows aggregated.
- **Sales Report (Reporte Ventas)** at `/sales/sales-report`: an ad-hoc, filterable query over
  individual sale orders by date range, client, product, and status — use it when a custom date
  range or per-order detail is needed instead of this page's fixed 45-day charts.
- **Point of Sale (Punto de Venta)** at `/sales/sale_order_create`: where new sale orders are
  created, ultimately feeding the summary data charted here.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/routes/sales/sale_orders_charts/+page.svelte
    role: page
    hash: sha256:093fe34cec96f0c6f5fa86029f5ba0683265b4d55d20dee8cbbbe2abade624a7
    supports: [page-purpose, concepts, capability.by-product, capability.daily-summary, rules]
  - path: frontend/routes/sales/sale_orders_charts/sale_orders_charts.svelte.ts
    role: frontend-service
    hash: sha256:ca2e3a83f32537b7745b925f476e01f4508b18a4e7f3e0ae136d14a4801d7cd7
    supports: [concepts, rules, troubleshooting]
  - path: frontend/routes/sales/sale_orders_charts/SaleOrdersChartsByProduct.svelte
    role: user-interface
    hash: sha256:fd3d3649ce8b8d46097c16e8886e47a18c8d1391c741fac9102733b8ff2e1f97
    supports: [capability.by-product, troubleshooting]
  - path: frontend/routes/sales/sale_orders_charts/SaleOrdersChartsDailySummary.svelte
    role: user-interface
    hash: sha256:66e23ca883a86fa735ccf8c627adb2e0051dbd3e8e804045e0bfa79301a201cc
    supports: [capability.daily-summary, troubleshooting]
  - path: frontend/routes/business/products/products.svelte.ts
    role: shared-domain
    hash: sha256:77bb3c75bd2663b000da54b9e84385f92c2a09dcc20b44234899388f51cc49d6
    supports: [concepts, capability.by-product, troubleshooting]
  - path: frontend/packages/genix-ui/charts/ChartCanvas.svelte
    role: shared-domain
    hash: sha256:53d785220f015d7130aaa7e12cb362c09e212016ae3f647562d6b85d2269a584
    supports: [capability.by-product]
  - path: backend/sales/sale_summary_status.go
    role: backend-handler
    hash: sha256:f34d5ff257ca7daca70f3561a62bf545e49be28c458d7fd3385fdb437f6a09ee
    supports: [concepts, rules]
  - path: backend/sales/sale_summary.go
    role: business-logic
    hash: sha256:6009c242673ce18ffeff7b157154789b459f4c3b55f97be881aec0fbbc718c5d
    supports: [concepts]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:937666309631867c1693fd6935a17e43f2f68eed0d39577537248dae75fa6cbc
    supports: [concepts, capability.by-product, capability.daily-summary]
  - path: backend/sales/main.go
    role: backend-handler
    hash: sha256:b1f47217c39ad2e920ed7916a7a2e104af3ebf3610743c69d58733ef097e8d90
    supports: [concepts]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [page-purpose]
  - path: backend/core/usuario-accesos.go
    role: permissions
    hash: sha256:cb8466840048ae5305e86bba91ee2f507a238f4f25002479ac23f014eaaa9f58
    supports: [page-purpose]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [page-purpose]
```
