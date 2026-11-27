---
schema: 1
page_id: accounting.financial-statements
route: /accounting/financial-statements
title: Financial Statements (Estados Financieros)
status: implemented
visibility: tenant
description_en: >-
  The income statement and the direct cash flow of one month, built from the same journal entries
  as Consolidado & Cuentas: revenue, cost of sales and expenses down to the net result, and the cash
  that came in and went out by operating, investing and financing activities between the opening
  and closing cash balances. Read either every entry (Management) or only the official Diario's
  (Official).
description_es: >-
  El estado de resultados y el flujo de caja directo de un mes, construidos con los mismos asientos
  de Consolidado & Cuentas: ventas, costo de ventas y gastos hasta el resultado neto, y el efectivo
  que entró y salió por actividades de operación, inversión y financiamiento entre el saldo inicial
  y el saldo final de cajas y bancos. Se leen todos los asientos (Gestión) o solo los del Diario
  oficial (Oficial).
---

# Financial Statements (Estados Financieros)

<!-- DOC-ID: page-purpose -->
## Page purpose

Financial Statements (`Estados Financieros`) shows two reports for **one calendar month** (`un
periodo`): the **income statement** (`Estado de resultados`, also called `estado de ganancias y
pérdidas` or `P&L`) and the **cash flow** by the direct method (`Flujo de caja`, `estado de flujo de
efectivo`). It answers questions such as *¿cuánto gané este mes?*, *¿cuál fue mi utilidad bruta?* and
*¿de dónde vino y a dónde se fue el efectivo del mes?*

The page **only reads**. It writes no entry, classifies nothing and changes no balance. Both reports
are built from exactly the same journal entries (`asientos`) that **Consolidado & Cuentas** shows for
that month — sales, purchases, expenses, stock receptions, cost of sales, production, fixed assets,
depreciation, cash movements and payroll — so a figure that looks wrong here is corrected at its
source or classified in Consolidado & Cuentas, never on this page.

It does **not** produce a balance sheet (`balance general`, `estado de situación financiera`): the
menu lists a **Balance** page that is not built yet and shows disabled. It is also not the cash
forecast: the future, day-by-day projection of cash is **Finanzas y Contabilidad → Tesorería → Flujo
de Caja** (`/finance/cash-flow`), while the **Flujo de caja** tab here explains the cash of a month that
already happened (or is in progress).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Period (Periodo):** one whole calendar month, from its first to its last day. There is no
  custom date range, no single day and no year-to-date view.
- **Entry (asiento) and line:** every business operation of the month becomes one journal entry
  whose lines post to **PCGE accounts** (`cuentas del Plan Contable General Empresarial`), each line on
  the Debe or the Haber side. The reports add those lines by account.
- **Unclassified line (línea sin clasificar):** a line whose account has not been chosen yet — for
  example an expense or a cash movement registered without its account. It carries an amount but
  no account. Unclassified lines are assigned in **Consolidado & Cuentas**.
- **Incomplete entry (operación incompleta):** an entry Genix could not build in full — for example
  a dollar amount with no exchange rate (`Un monto en dólares no tiene tipo de cambio`), or a sale
  whose cost of sales is still waiting for the production close. Its amounts may be missing or zero.
- **Management vs. Official entries (Gestión / Oficial):** **Gestión** reads every entry of the
  month, including operations without a comprobante. **Oficial** reads only what the official
  Diario (`Diario oficial`) holds: the entries backed by a comprobante, the entries that follow them
  (the cost of sales and the collections of a sale with comprobante), and the internal ones
  (production, depreciation, payroll, transfers between cajas, and similar).
- **Amounts in soles:** every figure is in soles (PEN), with two decimals. An operation in dollars is
  restated at the official sell rate (`tipo de cambio venta`) of its own day, unless its comprobante
  states its own rate. A **negative amount** — an expense, or cash that went out — is written **in
  parentheses**.

<!-- DOC-ID: capability.select-period-and-view -->
## Choose the month and the entries to read (Elegir periodo y asientos)

### User intention (Intención del usuario)

Pick which month the statements describe, and decide whether they should reflect the whole business
(management figures, `cifras de gestión`) or only what is officially documented (`lo oficial`).

### Where to find it (Dónde encontrarlo)

**Finance & Accounting (Finanzas y Contabilidad) → Accounting (Contabilidad) → Financial Statements
(Estados Financieros)**, at `/accounting/financial-statements`. The top row holds the **Periodo**
selector, the **Consultar** button and the **Asientos** selector (`Gestión` / `Oficial`). The two
reports are the tabs **Estado de resultados** and **Flujo de caja** in the page header.

### Required information and prerequisites (Requisitos previos)

Nothing has to exist first. The page opens on the **current month** with **Gestión** selected and
reads it immediately. **Periodo** offers the last 24 months, newest first, written like `NOV 2026`.

### Business rules and rationale (Reglas y razón de negocio)

Choosing another month in **Periodo** does not reload anything by itself: press **Consultar** to
read it. While it reads, the screen shows *Leyendo el periodo...*. Consultar reads both the month's
entries and the cash balances the cash flow needs.

Switching **Asientos** or switching tabs is immediate and does not read again. **Oficial** leaves
out the sales registered **SIN COMPROBANTE**, the purchases, expenses and fixed assets registered
without a supplier comprobante (a donated asset stays in), and the receptions, cost of sales,
collections and payments that belong to those undocumented operations. According to the code, this
view exists so the statements agree with the books sent to SUNAT.

### Result and side effects (Resultado y efectos)

Only the display changes. Nothing is saved.

### Limitations (Limitaciones)

- Only whole months, only one at a time, and only the last 24. There is no comparison between
  months and no accumulated year (`acumulado anual`).
- There is no export, download or print button on this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo veo el estado de resultados de otro mes?` Elija el mes en **Periodo** y presione
  **Consultar**.
- `¿Qué diferencia hay entre Gestión y Oficial?` Gestión incluye todo, también lo vendido o comprado
  sin comprobante; Oficial solo lo que está en el Diario oficial.
- `¿Puedo ver el resultado del año?` No: un mes a la vez.
- Search terms: `periodo`, `mes`, `consultar`, `asientos`, `gestión`, `oficial`, `sin comprobante`,
  `cifras de gestión`, `estados financieros`, `EEFF`.

<!-- DOC-ID: capability.income-statement -->
## Read the income statement (Estado de resultados)

### User intention (Intención del usuario)

Know whether the month made or lost money, and why: sales, cost of sales, gross margin
(`margen bruto`), operating expenses and the net result (`utilidad o pérdida del mes`).

### Where to find it (Dónde encontrarlo)

The **Estado de resultados** tab of Financial Statements, after the month is read.

### Required information and prerequisites (Requisitos previos)

The month's operations must already be registered in their own pages (sales, purchases, expenses,
assets, payroll, cash). Lines that are still unclassified do not count here, so classify them in
**Consolidado & Cuentas** first for a complete result.

### Business rules and rationale (Reglas y razón de negocio)

Each account's amount is its **Haber minus Debe** over the month, so income is positive and an
expense is negative (shown in parentheses). The report runs top-down, each section with its total
and, under it, the accounts with a non-zero amount, by PCGE code and name in chart order:

| Section | Accounts (PCGE codes starting with) |
|---|---|
| **Ventas netas** (Net sales) | 70, 74 |
| **Costo de ventas** (Cost of sales) | 69, 60, 61 |
| **Utilidad bruta** (Gross profit) | running result |
| **Gastos operativos** (Operating expenses) | 62, 63, 64, 65, 66, 68 |
| **Otros ingresos de gestión** (Other operating income) | 71, 72, 73, 75, 76, 78 |
| **Utilidad operativa** (Operating profit) | running result |
| **Ingresos financieros** (Financial income) | 77 |
| **Gastos financieros** (Financial expenses) | 67 |
| **Resultado antes de impuestos** (Profit before tax) | running result |
| **Impuesto a la renta** (Income tax) | 88 |
| **Resultado neto** (Net profit) | running result |

Every subtotal is the running sum of all the sections above it; a negative subtotal (a loss) shows
in red. A section with nothing posted still shows, with a zero total.

The layout is **hybrid**: revenue and cost of sales first, to show the gross margin, then the
expenses **by nature** (`por naturaleza`: personal, servicios de terceros, tributos,
depreciación…). According to the code, the entries carry no class 9 or account 79 allocation, so a
layout by function (`por función`: costo de producción, gastos de ventas, gastos administrativos) is
not possible.

**Cost of sales adds 69, 60 and 61 together.** A merchandise purchase posts to 60 (`Compras`) and,
when it is received into the warehouse, the reception posts the opposite amount to 61 (`Variación de
inventarios`), so a purchase that entered stock nets to zero there. What remains is the cost of what
was sold (69) and what was bought but never entered stock. Both 60 and 61 accounts still appear as
rows, cancelling each other.

### Result and side effects (Resultado y efectos)

Reading only.

### Limitations (Limitaciones)

- **Unclassified lines are left out** of the income statement; the page warns how many entries
  have them (see *Reglas generales*).
- The page does not calculate income tax (`impuesto a la renta`): that section only adds what entries
  posted to 88 accounts.
- Accounts outside the prefixes above never appear here: the balance-sheet classes 1 to 5, account
  79, class 9, and class 8 except 88.
- No budget, no prior-month comparison and no percentages over sales.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cuánto gané este mes?` Mire **Resultado neto** en **Estado de resultados**.
- `¿Por qué las compras no aparecen como costo?` Porque lo que entró al almacén se compensa entre 60 y
  61; solo es costo lo vendido (69) y lo que nunca entró a stock.
- `¿Dónde está la depreciación?` En **Gastos operativos**, bajo las cuentas 68.
- `¿Por qué mi utilidad sale en rojo / entre paréntesis?` Es una pérdida (un resultado negativo).
- Search terms: `estado de resultados`, `estado de ganancias y pérdidas`, `EGP`, `P&L`, `utilidad`,
  `ganancia`, `pérdida`, `margen bruto`, `utilidad bruta`, `utilidad operativa`, `resultado neto`,
  `costo de ventas`, `gastos operativos`.

<!-- DOC-ID: capability.cash-flow -->
## Read the cash flow of the month (Flujo de caja)

### User intention (Intención del usuario)

See how the cash in the cajas and bancos changed during the month and why: what came in and went out
through operations, through fixed assets, and through loans and owners' money.

### Where to find it (Dónde encontrarlo)

The **Flujo de caja** tab of Financial Statements, after the month is read.

### Required information and prerequisites (Requisitos previos)

The cash and bank accounts and their movements must be registered (**Cajas & Bancos**). A dollar
account (`caja en dólares`) needs an exchange rate for the dates the opening and closing balances are
restated at.

### Business rules and rationale (Reglas y razón de negocio)

The report uses the **direct method** (`método directo`). Every entry that touches a cash account
(PCGE accounts starting with 10: `101 Caja`, `1041` bank accounts) contributes **its other lines**,
each listed under its own account: a sale collected shows on the receivables account (1212), a
supplier invoice paid on the payables account (4212). A positive amount is cash in; parentheses are
cash out. A **transfer between two cajas or bancos** has no other line, so it moves nothing in the report.

Each line is sorted into a section by that other account (the counter-account):

- **Actividades de inversión** (Investing activities): accounts starting with 3 (fixed assets,
  `activos inmovilizados`) and 465 (what is owed for buying them), so paying for an asset is
  investing.
- **Actividades de financiamiento** (Financing activities): accounts starting with 44 (shareholders
  and directors), 45 (financial debt, such as loans) and 5 (equity).
- **Actividades de operación** (Operating activities): everything else, **including the
  unclassified lines**, which show in red as **Sin clasificar**, so the flows always add up to the
  cash that actually moved.

The report reads, top to bottom: **Saldo inicial de efectivo** (Opening cash), the three sections,
**Variación neta del efectivo** (Net change in cash), **Diferencia de cambio y no conciliado**
(Exchange difference and unreconciled) and **Saldo final de efectivo** (Closing cash).

The **opening cash** is the balance every caja and banco had at the end of the day before the month
began, and the **closing cash** the balance at the end of its last day — each taken from the account's
last movement up to that day; an account with no movement yet counts as 0. A **dollar account** is
restated in soles: the opening at the sell rate of the day before the month, the closing at the sell
rate of the month's last day. A rate is the company's own one when it loaded it in **Tipo de Cambio**,
otherwise the published BCRP rate, and Genix accepts one up to 7 days older than the day it needs.

**Diferencia de cambio y no conciliado** is *Saldo final − Saldo inicial − Variación neta*: the cash
the flows do not explain. According to the code it holds the exchange difference of the dollar
accounts plus any cash the selected entries do not contain — an operation whose dollar amount had no
rate, and, in the **Oficial** view, the collections and payments of the operations without
comprobante.

### Result and side effects (Resultado y efectos)

Reading only.

### Limitations (Limitaciones)

- If a dollar account has a balance but no rate for the date it needs, a red notice says *Una caja en
  dólares no tiene tipo de cambio: su saldo cuenta como 0.* and that balance is counted as zero.
- If the cash balances cannot be read, the tab still shows the three sections and the net change,
  without the opening, difference and closing lines.
- The report is a monthly summary: it does not list the individual movements (they are in **Cajas
  Movimientos**) and it does not forecast (that is **Tesorería → Flujo de Caja**).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué mi saldo final no cuadra con las entradas y salidas?` Revise **Diferencia de cambio y no
  conciliado**: diferencia de cambio de las cajas en dólares, operaciones sin tipo de cambio, o (en
  Oficial) lo que no tiene comprobante.
- `¿Por qué no aparece la transferencia entre mi caja y mi banco?` Porque el efectivo solo cambió de
  lugar; no entra ni sale.
- `¿Dónde sale el préstamo recibido o pagado?` El capital del préstamo, en **Actividades de
  financiamiento** (cuentas 45); los intereses y comisiones del pago, según la cuenta de cada uno.
- `¿Qué es Sin clasificar en el flujo?` Movimientos de caja cuya cuenta aún no se eligió; se
  clasifican en Consolidado & Cuentas.
- Search terms: `flujo de caja`, `flujo de efectivo`, `estado de flujos de efectivo`, `método
  directo`, `saldo inicial`, `saldo final`, `variación del efectivo`, `actividades de operación`,
  `inversión`, `financiamiento`, `diferencia de cambio`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- **Same entries as Consolidado & Cuentas.** Both pages read a month the same way, so a figure here
  can always be traced to an entry there. Fixing a number means fixing its source record (the sale,
  the expense, the cash movement) or classifying its line in Consolidado & Cuentas.
- **Unclassified and incomplete entries are announced.** When the selected entries hold any, a yellow
  notice above the report says: *El estado de resultados deja fuera las líneas sin clasificar (el flujo
  de caja las lista como Sin clasificar), y las operaciones incompletas pueden tener montos faltantes.
  Revíselas en Consolidado & Cuentas.* followed by the counts (`Sin clasificar: N · Incompletas: M`).
  The counts are entries, not lines, and they follow the **Asientos** selector.
- **Dollars are restated in soles at the sell rate of their day**, unless the operation's
  comprobante states its own rate. A dollar amount with no rate available is counted as 0 and its
  entry becomes incomplete.
- **Only the month's movements.** The cash balances are the only figures that look before the month;
  there are no accounting opening balances, so the income statement never carries results from
  earlier months.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **I changed the month but the figures did not change.** **Periodo** does not reload on its own;
  press **Consultar**.
- **A yellow notice counts *Sin clasificar* or *Incompletas*.** Open **Consolidado & Cuentas** on the
  same month, set **Mostrar** to *Sin clasificar* or *Incompletas*, and classify the lines or read
  the reason each incomplete entry gives; then press **Consultar** here again.
- **Sales or expenses I registered are missing.** Check the **Asientos** selector: **Oficial** leaves
  out everything without a comprobante. Switch to **Gestión** to see them.
- **Red notice: *Una caja en dólares no tiene tipo de cambio: su saldo cuenta como 0.*** Load the sell
  rate for the day before the month and for its last day in **Tipo de Cambio**. Because a rate may be at
  most 7 days older than the day it is needed for, the **month in progress** can show this notice for a
  dollar account until its last week has a rate.
- **The cash flow has no opening or closing line.** The cash balances could not be read (an error
  such as *No se pudieron leer los saldos de caja.* appears); press **Consultar** again.
- **No report appears and an error such as *No se pudo leer el periodo.* shows.** The month's
  entries could not be read; press **Consultar** again.
- **The net result looks wrong.** Unclassified lines are left out of the income statement, and
  incomplete entries may be missing amounts — check the yellow notice.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Consolidado & Cuentas** (`/accounting/accounts-consolidation`) — the same month's entries one by
  one (**Movimientos**), the **Diario de gestión** and the **Plan de cuentas**; the place to classify
  unclassified lines and read why an entry is incomplete.
- **Libros Contables** (`/accounting/books`) — the period's Registro de Ventas and Registro de
  Compras, the SUNAT books of the comprobantes.
- **Cajas & Bancos** (`/finance/cash-banks`) and **Cajas Movimientos**
  (`/finance/cash-banks-movements`) — the accounts and the individual movements the cash flow sums up.
- **Tipo de Cambio** (`/finance/exchange-rate`) — the company's own daily rates used to restate
  dollar amounts and dollar cash balances.
- **Flujo de Caja** (`/finance/cash-flow`) — the 13-week cash forecast; use it for the future, and
  this page's **Flujo de caja** tab for what already happened.
- **Activos** (`/accounting/assets`) — fixed assets and their depreciation, which appear here as
  investing cash flow and as 68 operating expenses.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.select-period-and-view]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.select-period-and-view, related-pages]
  - path: frontend/routes/accounting/financial-statements/+page.svelte
    role: page
    hash: sha256:2213254c95698ac854c2cae42dc0bddc427c2ee4b02c197e3816b2375e9953f9
    supports: [page-purpose, capability.select-period-and-view, capability.income-statement, capability.cash-flow, rules, troubleshooting]
  - path: frontend/routes/accounting/financial-statements/StatementTable.svelte
    role: user-interface
    hash: sha256:ce944071a6ea6f73e9422769c4feedb5f6e29dcdc97ddfe6ae5af2fcceffb243
    supports: [concepts, capability.income-statement, capability.cash-flow]
  - path: frontend/routes/accounting/financial-statements/financial-statements.ts
    role: business-logic
    hash: sha256:653b463bac2d450c5b856a5afbc24459021435741a545abe09b67f7f7052d014
    supports: [concepts, capability.select-period-and-view, capability.income-statement, capability.cash-flow, rules]
  - path: frontend/routes/accounting/financial-statements/financial-statements.svelte.ts
    role: frontend-service
    hash: sha256:9300732eebe19804a8e91a4f16bbede1e2d6af4145b8f4e6c31a3ff732e13781
    supports: [capability.cash-flow, troubleshooting]
  - path: backend/accounting/accounting_cash_balances_api.go
    role: backend-handler
    hash: sha256:aedb2e003dcf3cb8650464bfb568d36f6cd8c3480abd49dcc4601b57abc68501
    supports: [capability.cash-flow, rules]
  - path: backend/accounting/accounting_operations_api.go
    role: backend-handler
    hash: sha256:c8fe220fc549d253728fe36977a078132b61a95dcaa094f7116d4ce810907569
    supports: [page-purpose, capability.cash-flow]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.svelte.ts
    role: frontend-service
    hash: sha256:63800ebe3f249db9471e2aed07f2e0523b07d966e7765a616ca0aa1d897acb23
    supports: [page-purpose, capability.select-period-and-view, capability.cash-flow, rules, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.ts
    role: business-logic
    hash: sha256:7cdc814a6eb91a7ecb6038243e7e7c6dafa176029d1c48eaf11dbf8188f6bf6c
    supports: [page-purpose, concepts, capability.select-period-and-view, capability.income-statement, capability.cash-flow, rules, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/+page.svelte
    role: user-interface
    hash: sha256:7fe758db2766c328e302fecad357e9a32cc9307ef549de09de22309a56c0efa2
    supports: [troubleshooting, related-pages]
  - path: frontend/routes/accounting/books/books.ts
    role: shared-domain
    hash: sha256:7e26d19318088cd3acf2be59f90ba0bebfe851efa2279d6b149fdf403a8a6555
    supports: [capability.select-period-and-view]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.ts
    role: shared-domain
    hash: sha256:5a5b23729c283c270bb18dfea2b17758572f9405c2fa9a01cbdf96d98d6c0cba
    supports: [capability.cash-flow, troubleshooting]
  - path: frontend/core/pcge.ts
    role: shared-domain
    hash: sha256:26947ab13508fceeaed84d577ecb117a530e0df9a9ec314dbcb253d31c322118
    supports: [concepts, capability.income-statement]
  - path: frontend/routes/accounting/accounts-consolidation/OperationEntryLayer.svelte
    role: user-interface
    hash: sha256:6eff3423c23ec13f61987b7ce290710c0ce35f75f4ff5fd82ac03caf782da902
    supports: [troubleshooting, related-pages]
  - path: frontend/domain-components/Page.svelte
    role: user-interface
    hash: sha256:f303e7168be459d0553cd0449f577b9b5063935b9f0949acd3bea6f4fa8d9761
    supports: [capability.select-period-and-view]
  - path: frontend/routes/finance/cash-flow/cash-flow.ts
    role: shared-domain
    hash: sha256:90cc39538a18192958af5b55f385e22646c806e096d32a629a47f109b8c56e00
    supports: [page-purpose, related-pages]
```
