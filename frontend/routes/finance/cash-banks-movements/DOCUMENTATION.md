---
schema: 1
page_id: finance.cash-banks-movements
route: /finance/cash-banks-movements
title: Cash Movements (Cajas Movimientos)
status: implemented
visibility: tenant
description_en: >-
  Cash and bank movement history report. Select one cash register or bank account and a date
  range to list every recorded movement with its date, type, amount, resulting balance, linked
  document, and responsible user; filter the loaded results by movement type.
description_es: >-
  Reporte de historial de movimientos de caja y banco. Seleccionar una caja o cuenta bancaria y
  un rango de fechas para listar cada movimiento registrado con su fecha, tipo, monto, saldo
  resultante, documento vinculado y usuario responsable; filtrar los resultados cargados por
  tipo de movimiento.
---

# Cash Movements (Cajas Movimientos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Cash Movements (`Cajas Movimientos`) is a read-only report: it lists the ledger movements
(`movimientos de caja`) already recorded for one chosen cash register or bank account
(`caja o cuenta bancaria`) over a date range (`rango de fechas`) the user picks. It is the
dedicated history/report route for this data — it does not create, edit, or delete a
movement, does not configure an account, and does not perform a reconciliation (`cuadre` or
`arqueo`). Those actions live on **Cash & Banks (Cajas & Bancos)**; see
Related pages below for exactly when to use each page.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **movement (`movimiento`)** is one ledger entry: it carries a type, an amount, the
  resulting balance (`saldo final`) right after that entry, an optional linked document/
  reference id, when it was created, and which user created it. Movement types shown here
  come from the same shared catalog used on Cash & Banks: `Cuadre Físico`, `Transferencia`,
  `Retiro`, `Pérdida`, `Pago Proveedor`, `Cobro`, `Cobro (Venta)`, and `Pago Gasto`. Outflow
  types (`Transferencia`, `Retiro`, `Pérdida`, `Pago Proveedor`, `Pago Gasto`) are stored and
  shown as negative amounts, rendered in red on this page.
- This page's rows are historical facts already produced elsewhere (manual entry, a paid
  purchase order, an expense payment, a sale collection, or a reconciliation); it only reads
  and displays them for a chosen account and period, it does not recompute the account
  balance itself.

<!-- DOC-ID: capability.query-movements -->
## Query an account's movement history (Consultar historial de movimientos)

### User intention (Intención del usuario)

Review everything that happened to one cash register or bank account's balance during a
specific period — for example to audit a week's cash activity or to trace a payment by date.

### Where to find it (Dónde encontrarlo)

Open **Finance (Finanzas) → Cash Movements (Cajas Movimientos)** at
`/finance/cash-banks-movements`. Choose the account in the **Cash & Banks (Cajas &
Bancos)** selector, adjust **Start Date (Fecha Inicio)** / **End Date (Fecha Fin)**, and
click the search (magnifying glass) button to run the query.

### Required information and prerequisites (Requisitos previos)

- An existing cash/bank account must be selected; the page pre-selects the first account
  returned by the account list once it loads, but nothing is queried automatically — the
  table stays empty until the user presses the search button, and it does not
  auto-refresh when the account or dates are changed afterward.
- Both **Start Date** and **End Date** are required. They default to the last 7 days up to
  today when the page opens.
- If the account or either date is missing when search is pressed, Genix shows "Please
  select a cash register and a date range.|Debe seleccionar una caja y un rango de dates."
  and does not call the server. The server independently rejects a missing account
  ("No se envió la CashBank-ID") or a missing date range ("Debe enviar una date de inicio
  y fin."), so those errors should not normally appear given the frontend check.

### Business rules and rationale (Reglas y razón de negocio)

Dates are whole-day values (no time-of-day component). Results are returned newest first.
Because this report always queries by date range rather than by a fixed row count, it has
no page-imposed limit on how many movements come back for a wide range — unlike the
account detail panel on **Cash & Banks**, which caps its own Movements tab at the latest
200 rows regardless of date. Selecting a very wide date range on a busy account can
therefore return a large result set in one response.

### Result and side effects (Resultado y efectos)

The table lists, per movement: **Date & Time (Fecha Hora)**, **Movement Type (Tipo
Mov.)** (resolved from the shared type catalog), **Amount (Monto)** (negative amounts in
red), **Final Balance (Saldo Final)** (the balance immediately after that movement),
**Document # (Nº Documento)** (the linked document/reference id as plain text, not a
clickable link, shown only when the movement carries one), and **User (Usuario)** (the
username of whoever created the movement, resolved by id). Querying only reads data; no
record is created, changed, or removed by this page.

### Limitations (Limitaciones)

- Entirely read-only: there is no create, edit, delete, export, or print action on this
  page.
- Document # is not a link; to see the originating sale, expense, or purchase order, open
  that record from its own page using the shown identifier.
- No result count or pagination indicator is shown, and no maximum-rows warning is shown
  either, even though the query itself has no row cap.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo veo todos los movimientos de una caja en un rango de fechas?`
- `¿Por qué la tabla está vacía si ya elegí la caja y las fechas?` Falta presionar el
  botón de buscar (la lupa); la consulta no se ejecuta sola.
- `¿Puedo ver más de 200 movimientos aquí?` Sí — a diferencia de la pestaña de
  movimientos de Cajas & Bancos, este reporte no limita la cantidad de registros por
  rango de fechas.
- Search terms: `movimientos de caja`, `historial de caja`, `reporte de movimientos`,
  `cajas movimientos`, `saldo final`, `tipo de movimiento`.

<!-- DOC-ID: capability.filter-results -->
## Filter the loaded results (Buscar dentro de los resultados)

### User intention (Intención del usuario)

Narrow an already-loaded result set down to movements of one type, for example to isolate
every `Pago Proveedor` in the queried period.

### Where to find it (Dónde encontrarlo)

The search box at the top-right of the page, above the results table.

### Business rules and rationale (Reglas y razón de negocio)

Typing filters the rows already loaded in the browser; it does not query the server
again. Matching is case-insensitive and only checks the movement's type name (for
example `Pago Proveedor`, `Cobro`, `Retiro`) — it does not match against the amount,
final balance, document number, or user.

### Limitations (Limitaciones)

Cannot filter by amount, document number, date, or user; change the account/date range
and re-run the search above for that.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo busco sólo los retiros o sólo los pagos a proveedor en la lista?` Escribe el tipo
  de movimiento en el buscador de la derecha.
- `¿Por qué no encuentro un movimiento buscando el nombre del usuario o el monto?` El
  buscador de esta página sólo filtra por tipo de movimiento.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- The "Cajas Movimientos" access entry in the access catalog controls whether the menu
  item and route are reachable in the app, but the underlying read endpoint has no
  backend API mapped to it in the catalog, the same pattern documented for other
  view-only lists in Genix: an unmapped `GET` is open to any authenticated session of the
  company, so this page's data is not further access-restricted at the server beyond
  normal login.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Please select a cash register and a date range.|Debe seleccionar una caja y un
  rango de dates.":** press search without an account or without both dates chosen.
- **The table stays empty right after opening the page:** expected — press the search
  (magnifying glass) button after confirming the account and dates.
- **Changing the account or the dates does not refresh the table:** search must be
  pressed again; the page never re-queries automatically.
- **A "Document #" value with nothing to click:** open the related sale, expense, or
  purchase-order page directly using that identifier; this report does not link to it.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash & Banks (Cajas & Bancos)** at `/finance/cash-banks`: create/configure accounts,
  register a manual movement, reconcile (`cuadre`/`arqueo`) an account, and see one
  account's own recent Movements/Reconciliations tabs (capped at the latest 200 rows,
  not date-range driven). Use that page for configuration, manual entry, or
  reconciliation; use this page instead when the task needs an explicit date range or a
  history that may exceed 200 rows.
- **Purchase Orders (Órdenes de Compra)**, expense payment workflows, and sales
  collections are what actually create the `Pago Proveedor`, `Pago Gasto`, `Cobro`, and
  `Cobro (Venta)` movements listed here; open those pages to act on the originating
  document.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.query-movements, rules]
  - path: frontend/routes/finance/cash-banks-movements/+page.svelte
    role: page
    hash: sha256:3047a9ebe93ae77f732b4e981d876c2e2345df7e784b29a1b100a58262eb35cc
    supports: [page-purpose, concepts, capability.query-movements, capability.filter-results, troubleshooting]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: frontend-service
    hash: sha256:57abecc19b2da874e32d06acbf244f6bcd786470c746551be381ca02070e6887
    supports: [concepts, capability.query-movements, rules]
  - path: frontend/packages/genix-ui/misc/RecordByIDText.svelte
    role: user-interface
    hash: sha256:9dab49a6a9b4007f7a5ccffd4c1471bc06d37764e112646f3fff84de750ee6f6
    supports: [capability.query-movements]
  - path: frontend/packages/genix-ui/vTable/VTable.svelte
    role: user-interface
    hash: sha256:1926ae5dd0323d8b1352873aa4bbc8e727c39e8835b786abcf677a1c97eda9b6
    supports: [capability.filter-results]
  - path: backend/finance/cash_banks.go
    role: backend-handler
    hash: sha256:d3bd7b24ab258a52e5d404fee29c8b796d91507f84fb2caf2c0f44330504e36f
    supports: [capability.query-movements, rules, troubleshooting]
  - path: backend/finance/types/cash_banks.go
    role: data-model
    hash: sha256:82ac985de5ca9fe560af0386a7bb6c35c8b65352c3a0b006cce8620fc5b84957
    supports: [concepts, capability.query-movements]
  - path: backend/access_list.yml
    role: permissions
    hash: sha256:0c00cfb3e7af9a918eb753846874ac7d213f1eacb3e46bded4049016e1c57951
    supports: [rules]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:2474ede3472c063c1e28ea584cd6265dc4b7b8437231cfa94aa5671e66b62330
    supports: [rules]
```
