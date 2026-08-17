---
schema: 1
page_id: sales.sales-report
route: /sales/sales-report
title: Sales Report (Reporte Ventas)
status: implemented
visibility: tenant
description_en: >-
  Ad-hoc sales order report. Filter existing sale orders by date range (up to 120 days), client,
  product, and status, run the query on demand, and search the loaded results by free text.
description_es: >-
  Reporte de órdenes de venta bajo demanda. Filtra órdenes existentes por rango de fechas (hasta
  120 días), cliente, producto y estado, ejecuta la consulta cuando quieras y busca en los
  resultados cargados por texto libre.
---

# Sales Report (Reporte Ventas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Sales Report (`Reporte Ventas`) at **Commercial (Comercial) → Sales Report** is a read-only
query tool over already-created sale orders (`órdenes de venta`, `pedidos`). It answers "what
did we sell, to whom, and for how much" over a chosen date window, without letting the user
create, edit, pay, or deliver an order from here. Nothing loads automatically when the page
opens; the user must open the filter panel and run the query (`Buscar`) before any row appears.

This page does not register payments (`pagos`) or deliveries (`entregas`) — that happens on
**Sales Management (Gestión Ventas)** — and it does not create new orders — that happens on
**Point of Sale (Punto de Venta)**.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **sale order (orden de venta / pedido)** on this report shows its ID, date and time, a
  paid/delivered indicator, total amount, outstanding debt (`Deuda`), the client, and its top
  products.
- **Status (`Estado`)** is the order's lifecycle value. The filter selector offers **All
  (Todos)**, **Generated (Generado)**, **Paid (Pagado)**, **Delivered (Entregado)**, and
  **Finished (Finalizado)**. Internally the order model also has a status `0` meaning voided
  (`Anulado`), and status `4` ("Finalizado" in this selector) is stored as "paid + delivered."
  The selector has no option to isolate voided orders specifically — choosing "Todos" is the
  only way to include them, and it also includes every other status.
- The **Delivered/Paid (Entregado/Pagado)** column shows two independent letter markers per row:
  a green **P** lights up once the order has been paid (status 2 or 4), and a blue **E** lights
  up once it has been delivered (status 3 or 4). Both can be lit at once.
- **Top Products (Top Productos)** ranks the up-to-three highest-value products inside one
  order (unit price × quantity, ties broken by product ID) and shows quantity and line amount for
  each; a "(N more...)" note appears on mobile when an order has more than three distinct
  products.

<!-- DOC-ID: capability.run-report -->
## Filter and run the report (Filtrar y ejecutar el reporte)

### User intention (Intención del usuario)

Pull the set of sale orders that match a date range plus optional client, product, and status,
to audit or review sales history for a period.

### Where to find it (Dónde encontrarlo)

On `/sales/sales-report`, click the purple search-icon button (its accessible label is "Opens
the search filter for the sales report") to open the filter panel. Inside it, set **Start Date
(Fecha Inicio)**, **End Date (Fecha Fin)**, **Client (Cliente)**, **Product (Producto)**, and
**Status (Estado)**, then click **Search (Buscar)**. The panel closes automatically after the
click and the summary strip above the table always reflects the current filter values.

### Required information and prerequisites (Requisitos previos)

- **Start Date** and **End Date** are required; the page pre-fills them to the last 7 days
  (today and 6 days back) the first time it renders, but nothing is queried until the user
  presses **Buscar**.
- **Client**, **Product**, and **Status** are optional and default to "all" (`0`/`Todos`).
  The client selector searches by name and registry number (`RUC`/`Nº Documento`) together and
  only lists clients that are currently active; the product selector lists every cached product
  regardless of its own active/inactive status.

### Business rules and rationale (Reglas y razón de negocio)

- The date range cannot exceed **120 days**, and the end date cannot be earlier than the start
  date; both are enforced by the server.
- Only one secondary filter is actually applied together with the date range, in this priority
  order: **Client + Product** (if both are set) → **Product only** (if only a product is set) →
  **Client only** (if only a client is set) → **Status only** (if status is the only filter set
  besides dates). In other words, whenever **Client** or **Product** has a value, **Status is
  silently ignored** by the query even though the field still shows the chosen option — the
  server picks "the most specific compatible grouped index from the remaining equality filters"
  and only one such index is queried per request. Status filtering only actually narrows the
  result when Client and Product are both left as "Todos."
- The results table itself never applies this filtering again on the client side, so whatever
  the server returned is exactly what is shown (subject only to the free-text search below).
- Repeating an unchanged query reuses sale-order data already downloaded to the browser instead
  of re-fetching it; only new or changed groups of orders are downloaded again, which is why
  searching the same period twice can return sooner than the first search.

### Result and side effects (Resultado y efectos)

Running the report only reads data; it creates or updates nothing. It replaces whatever rows
were previously shown with the new result set (there is no "append" or infinite-scroll
behavior across searches).

### Limitations (Limitaciones)

- The summary strip (start/end date, client, product, status) reflects whatever is currently
  selected in the filter fields, even before the next **Buscar**. Changing a filter and not
  pressing **Buscar** leaves the table showing the previous search's rows while the strip above
  it already shows the new, not-yet-applied selection.
- There is no way to combine **Status** with **Client** or **Product** in a single query; see
  the rule above.
- No auto-refresh: nothing here polls for newly created orders; run **Buscar** again to see
  recent activity.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué el filtro de Estado no hace nada si también elegí un cliente o producto?` Porque el
  servidor sólo aplica un filtro secundario a la vez, y Cliente/Producto tienen prioridad sobre
  Estado.
- `¿Cómo veo únicamente los pedidos anulados?` No existe una opción exclusiva para "Anulado" en
  este selector; sólo "Todos" incluye esos pedidos junto con el resto.
- `¿Por qué puedo consultar sólo 120 días?` Es el límite máximo de rango de dates que acepta el
  servidor para este reporte.
- Search terms: `reporte de ventas`, `historial de ventas`, `filtrar por fecha`, `filtrar por
  cliente`, `filtrar por producto`, `filtrar por estado`, `rango de dates`, `120 días`.

<!-- DOC-ID: capability.review-results -->
## Read and search the loaded results (Revisar y buscar los resultados cargados)

### User intention (Intención del usuario)

Scan the orders returned by the last search and locate a specific one by client, product, or
order ID without re-running the filter panel.

### Where to find it (Dónde encontrarlo)

The free-text search box (`Buscar...`) sits at the top right of the page, independent from the
filter panel.

### Required information and prerequisites (Requisitos previos)

A search has to have already returned rows; typing here does not query the server.

### Business rules and rationale (Reglas y razón de negocio)

Free-text search matches locally, against the client name and the product names already loaded
for the visible rows, plus the order ID; it does not reach into fields that were not requested
(for example it cannot find an order by document/RUC number even though the client filter
selector can).

### Result and side effects (Resultado y efectos)

Filtering by text only changes which already-loaded rows are visible; it never issues another
request and never changes the underlying result set from the last **Buscar**.

### Limitations (Limitaciones)

- Clicking a row does nothing on this page — there is no side panel, no navigation to the order,
  and no way to pay or deliver from here. Use **Sales Management (Gestión Ventas)** for that.
- Only up to three top products display per order even when a matched product is not among the
  top three by value; the row still appears (because the underlying order matched), but the
  matching product may not be visible among the shown product cards without expanding on mobile.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Puedo hacer clic en un pedido para editarlo?` No; este reporte es de solo lectura.
- `¿Cómo busco un pedido por número?` Escribe el ID en el cuadro de búsqueda de texto libre.
- Search terms: `buscar pedido`, `buscar por cliente`, `buscar por producto`, `buscar por ID`,
  `filtro de texto`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- This page's own catalog entry ("Reporte Ventas", part of the Commercial/Comercial access
  group) only controls whether the menu link and route are visible; it declares no backend API
  of its own. Because the query endpoint behind this report (`sale-order-query`) is not mapped
  to any access in the catalog, any authenticated user of the company can call it — the
  page-specific access only gates navigation, not the underlying data query.
- Money values (`TotalAmount`, `DebtAmount`, per-product line amounts) are shown already divided
  to their normal display scale by the page; no page-specific rounding or currency conversion is
  applied here beyond that.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **“Sólo se pueden consultar hasta 120 días a la vez”:** shorten the Start/End Date range to 120
  days or less.
- **“La date final no puede ser menor a la date inicial”:** the End Date is earlier than the
  Start Date; swap or correct them (the exact wording mixes English "date" into the Spanish
  message — this is the real text the server returns, not a translation error on this page).
- **“Debe especificar el rango de dates” / “Debe especificar la date inicial y final”:** one of
  the two date fields is empty; both fire before the request even reaches the server for the
  second phrasing.
- **The table stays empty after opening the page:** expected — no query runs until the filter
  panel is opened and **Buscar** is pressed at least once.
- **Choosing a Status does not seem to filter anything:** confirm Client and Product are both
  set back to "Todos"; otherwise Status is not applied (see the rule above).
- **A row I expect to see is missing:** confirm the order's date falls inside the selected
  range and that Client/Product, if set, matches it; remember Status is ignored whenever
  Client or Product has a value.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Point of Sale (Punto de Venta)** at `/sales/sale_order_create`: creates the sale orders that
  later appear in this report.
- **Sales Management (Gestión Ventas)** at `/sales/sale_orders_status`: the page to actually
  register a payment (selecting a `caja`) or a delivery (selecting a `almacén`) for an order,
  and the only sales-order table on which clicking a row opens an action layer.
- **Sales Charts (Gráficos Ventas)** at `/sales/sale_orders_charts`: aggregated by-product and
  daily-summary charts over sales, instead of this page's per-order row list.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/routes/sales/sales-report/+page.svelte
    role: page
    hash: sha256:fef5330fa9857404fd28f0c6840b6cb2f00e85fe5b3f2fe7c3a54a9878b3498b
    supports: [page-purpose, concepts, capability.run-report, capability.review-results]
  - path: frontend/routes/sales/sales-report/sale_order_report.svelte.ts
    role: frontend-service
    hash: sha256:b0c03f8c5ac54e2a8119cf66a180841dd926dba73bc7685e13ffe7ba280831d2
    supports: [concepts, capability.run-report, troubleshooting]
  - path: frontend/routes/sales/SaleOrdersTable.svelte
    role: user-interface
    hash: sha256:187483386db7e604710e8893046b160bc0d523d81540137b0e3c482733466b1f
    supports: [concepts, capability.review-results, related-pages]
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, related-pages]
  - path: frontend/packages/genix-ui/cache/group-cache.fetch.ts
    role: frontend-service
    hash: sha256:c018a9646819280b1dcba31a2e7d97f050820b333fcd3c113be8fa145f505cf9
    supports: [capability.run-report]
  - path: frontend/routes/business/customers/customers.svelte.ts
    role: shared-domain
    hash: sha256:d42d73f9ef8b3ecd5e7fec9e83cbb2e5cddf8d276c4ac6c37c6babdb1d5081d3
    supports: [capability.run-report]
  - path: frontend/routes/business/products/products.svelte.ts
    role: shared-domain
    hash: sha256:77bb3c75bd2663b000da54b9e84385f92c2a09dcc20b44234899388f51cc49d6
    supports: [capability.run-report]
  - path: backend/sales/sale_order_query.go
    role: backend-handler
    hash: sha256:fdf3a3095fe94d28b9edc94d099c2b0cbad94a3ee6eeabf87876ad226e70fdd4
    supports: [capability.run-report, rules, troubleshooting]
  - path: backend/sales/types/sales.go
    role: data-model
    hash: sha256:937666309631867c1693fd6935a17e43f2f68eed0d39577537248dae75fa6cbc
    supports: [concepts, capability.run-report, rules]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
  - path: backend/core/usuario-accesos.go
    role: permissions
    hash: sha256:cb8466840048ae5305e86bba91ee2f507a238f4f25002479ac23f014eaaa9f58
    supports: [rules]
```
