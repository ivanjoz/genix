---
schema: 1
page_id: accounting.accounts-consolidation
route: /accounting/accounts-consolidation
title: Consolidation & Accounts (Consolidado & Cuentas)
status: implemented
visibility: tenant
description_en: >-
  Every movement of a month turned into a journal entry on the PCGE chart: see which operations
  are classified, unclassified or incomplete, pick the accounting account of expenses, supply
  purchases, fixed assets and manual cash movements one by one or in bulk, and review the
  management journal and the period's trial balance per account.
description_es: >-
  Todos los movimientos de un mes convertidos en asientos del PCGE: ver qué operaciones están
  clasificadas, sin clasificar o incompletas, elegir la cuenta contable de gastos, compras de
  insumos, activos fijos y movimientos de caja manuales uno por uno o en bloque, y revisar el
  Diario de gestión y el balance de comprobación del periodo por cuenta.
---

# Consolidation & Accounts (Consolidado & Cuentas)

<!-- DOC-ID: page-purpose -->
## Page purpose

Consolidation & Accounts (`Consolidado & Cuentas`) is where a month's business activity becomes
accounting. Genix reads every movement of the chosen period — sales and their comprobantes, credit
notes, purchase orders, expenses, warehouse receptions, sale deliveries, production, fixed assets,
depreciation, cash and bank movements, and closed payslips — and turns each one into a journal
entry (`asiento contable`) on the Peruvian chart of accounts (`Plan Contable General Empresarial`,
PCGE 2019). Amounts are shown in soles (`S/`).

The page answers two questions: *¿qué tan cuadrado está el mes?* (how many operations are
classified, unclassified or incomplete, and whether Debe equals Haber) and *¿a qué cuenta va
esto?* — the user picks the accounting account (`cuenta contable`) only where a business decision
is needed, and Genix derives every other line.

It shows the same period three ways, as three tabs: **Movimientos** (the operations to review and
classify), **Diario de gestión** (the management journal, one entry per operation) and **Plan de
cuentas** (the period's balance per account).

This page **does not create operations**. Sales, purchases, expenses, assets, cash movements and
payroll are registered on their own pages; this page only reads them and, when classifying, writes
the chosen account back onto the original record. It has no manual journal entry
(`asiento manual`), no opening or closing entry (`apertura`, `cierre`), and no export.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Operation (`Operación`)**: one source record of the period seen as one journal entry — a sale,
  an expense, a reception, a cash movement, a payslip, and so on. Each operation has a **CUO**
  (`Código Único de Operación`), a code made of a prefix for its kind and the source record's id,
  for example `V-5501` for a sale or `G-120` for an expense.
- **Nature line (`línea de naturaleza`)**: the line that says *what* the money was for — the
  expense account of an expense (63x, 62x…), the account of a supply bought on a purchase order,
  the 33x of a fixed asset, the counter-account of a manual cash movement. **Only these lines can
  be classified by the user.**
- **Derived line (`línea derivada`)**: every other line — IGV (40111), suppliers payable (4212),
  receivables (1212), cash (101) or bank (1041), sales revenue (7012 / 7022), cost of sales,
  depreciation pairs. Genix decides them from the operation's type and they are **locked**: in the
  entry panel they carry a padlock icon and cannot be edited.
- **Operation status (`estado`)**:
  - **Classified (`Clasificada`)** — every line has an account and the entry balances.
  - **Unclassified (`Sin clasificar`)** — at least one line has no account yet, normally a nature
    line waiting for the user. The missing account appears as a red warning sign.
  - **Incomplete (`Incompleta`)** — the source data has a problem Genix cannot resolve by
    classifying: a dollar amount without an exchange rate, a sale or purchase that cannot be found,
    a delivery or reception without cost, an account without its stock or depreciation pair, a
    sale whose stock is still to regularize, or an entry whose Debe and Haber do not match. An
    incomplete operation shows its accounts in red; hovering them shows the problem.
- **Official Diario vs management journal**: every entry belongs to the management journal
  (`Diario de gestión`), which includes movements with and without comprobante. An entry also
  belongs to the official Diario (the SUNAT book) unless it is a sale, purchase or asset purchase
  **without comprobante**, or a payment, collection or cost of sales that follows one. Internal entries —
  depreciation, transfers, disposals, payroll, production — are in the official Diario. Entries
  outside it are marked **Fuera del Diario oficial** in the entry panel and **sin comprobante** in
  the Diario de gestión.
- **Period (`Periodo`)**: always a whole calendar month. The page never reads a single day.

<!-- DOC-ID: capability.read-period -->
## Read a period and its status (Consultar un periodo)

### User intention (Intención del usuario)

See how a month stands before relying on its financial statements: *¿cuántas
operaciones faltan clasificar en octubre?*, *¿cuadra el Debe con el Haber?*

### Where to find it (Dónde encontrarlo)

**Finanzas y Contabilidad → Contabilidad → Consolidado & Cuentas** (`/accounting/accounts-consolidation`).
The top row holds the **Periodo** selector and the **Consultar** button.

### Required information and prerequisites (Requisitos previos)

None. The page opens on the current month and reads it automatically. **Periodo** lists the
current month and the 23 before it, newest first (for example `NOV 2026`); there is no
*Día específico* option here, unlike the books page.

### Business rules and rationale (Reglas y razón de negocio)

Changing **Periodo** does not reload by itself: press **Consultar** to read the chosen month.
After a successful read a status line appears above the table:

- **Clasificadas**: count of classified operations and its percentage of all operations (by count).
- **Sin clasificar**: count of unclassified operations.
- **Incompletas**: count of incomplete operations.
- **Debe − Haber**: a green check when the period's total Debe equals its total Haber; otherwise
  the difference in red.
- **Sin comprobante**: how many operations are outside the official Diario and their total in
  `S/`. This includes the collections, payments and cost of sales of undocumented sales and
  purchases, not only the documents themselves.

Dollar amounts are converted to soles at the official sell rate of their day — the company's own
rate from **Tipo de Cambio** if it loaded one, otherwise the published BCRP rate — taking the most
recent rate up to 7 days back. A purchase comprobante (and an asset bought with one) uses the
exchange rate written on the document instead.

### Result and side effects (Resultado y efectos)

Reading writes nothing. The three tabs show the operations of the period just read.

### Limitations (Limitaciones)

The page does not lock or close a period. The balance shown is only the period's movement: there
are no opening balances from earlier months.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo sé si ya clasifiqué todo el mes?` Consulte el periodo: *Sin clasificar* e *Incompletas*
  deben estar en 0 y *Debe − Haber* debe mostrar el check verde.
- `¿Por qué no cambia la tabla al elegir otro mes?` Hay que presionar **Consultar**.
- `¿En qué tipo de cambio convierte los dólares?` El tipo de cambio venta oficial del día (el de la empresa o el del BCRP); una
  compra con comprobante usa el tipo de cambio del comprobante.
- Search terms: `periodo`, `mes`, `cierre de mes`, `cuadre contable`, `debe`, `haber`,
  `consolidado`, `clasificadas`, `pendientes de clasificar`.

<!-- DOC-ID: capability.review-operations -->
## Review the period's operations (Revisar movimientos)

### User intention (Intención del usuario)

Find what still needs an account or has a data problem, and check how each operation was booked.

### Where to find it (Dónde encontrarlo)

Tab **Movimientos**. Beside the period, **Mostrar** filters by status and a search box (magnifier
icon) filters by text.

### Required information and prerequisites (Requisitos previos)

A period read with **Consultar**.

### Business rules and rationale (Reglas y razón de negocio)

Columns: a tick column, **Fecha**, **Operación** (the kind, for example *Venta*, *Gasto*,
*Ingreso a almacén*, *Movimiento de caja*), **Descripción**, **Nº Doc**, and **Debe** and
**Haber**, each split into **Cuenta** and **Monto**. Each side stacks one line per account.

**Mostrar** offers *Todas*, *Sin clasificar*, *Incompletas* and *Sin comprobante* (the operations
outside the official Diario). The search matches the description, the operation kind and the
document numbers; it does not match account codes or CUOs here.

**Similar operations are grouped into one row**: operations of the same day, kind, description,
status, official-Diario flag and the same accounts on the same sides read as a single row — so a
day's sales usually show as one line. The amounts are summed and **Nº Doc** shows the number of
operations in parentheses, for example `(12)`. A row of one operation shows its document number,
or a **sin comprobante** chip when the operation carries no document number (this also happens on
internal entries such as depreciation or cash movements).

Clicking a row opens a side panel: a single operation opens its entry directly; a grouped row
first lists its operations (**CUO**, **Nº Doc**, **Monto (S/)**), and clicking one opens its entry,
with **← Volver a la lista** to go back.

### Result and side effects (Resultado y efectos)

Filtering and opening rows write nothing. The search text is shared with the Diario de gestión tab
and stays applied when switching between them.

### Limitations (Limitaciones)

Grouped rows cannot be ticked for bulk classification; open them to work on each operation.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo veo solo lo que falta clasificar?` En **Mostrar** elija *Sin clasificar*.
- `¿Por qué varias ventas aparecen en una sola fila?` Porque son del mismo día y van a las mismas
  cuentas; haga clic en la fila para ver cada una.
- `¿Qué significa el triángulo rojo en la cuenta?` Que esa línea está sin clasificar.
- Search terms: `movimientos`, `operaciones`, `filtrar`, `sin clasificar`, `incompletas`,
  `sin comprobante`, `agrupado`.

<!-- DOC-ID: capability.classify-operation -->
## Classify one operation (Clasificar una operación)

### User intention (Intención del usuario)

Give an operation its accounting account — *¿a qué cuenta va este gasto?* — or correct one that
was set wrong.

### Where to find it (Dónde encontrarlo)

Tab **Movimientos** → click the row (and, for a grouped row, the operation). The side panel is
titled with the operation kind and its CUO. It shows the kind, the date, the document number (or
`—`), the **Fuera del Diario oficial** chip when it applies, a yellow notice with the problem when
the operation is incomplete, and the entry's lines (**Cuenta**, **Debe**, **Haber**). Below, the
**Clasificación contable** section holds one selector per thing that can be classified, and the
panel's **Guardar** button saves.

### Required information and prerequisites (Requisitos previos)

What can be classified depends on the operation:

| Operation | What you pick | Accounts offered |
| --- | --- | --- |
| **Gasto** (simple expense) | the expense's account, labelled with the expense's name | 62, 63, 64, 651–654, 656, 658, 659, 67 and 40 |
| **Gasto** of inventory (supplies) | the supply account | 602, 603, 604, 656 and 33 |
| **Compra** (purchase order) | one selector per supply line (**Línea 1**, **Línea 2**…) | 602, 603, 604, 656 and 33 |
| **Activo fijo**, its **Depreciación** or its **Baja de activo** | the asset's account, labelled with the asset's name | 32, 33 and 34 |
| **Movimiento de caja** manual: sin especificar, Cuadre Físico, Retiro, Pérdida or Cobro | **Tipo y cuenta** | the options of the movement's type (below) |

Only detail (postable) accounts under those prefixes are listed. Every selector is required:
saving with one empty shows *Seleccione todas las cuentas.* or *Seleccione el tipo y la cuenta.*

For cash movements, **Tipo y cuenta** offers:

- an unspecified movement (`sin especificar`) that **took money out**: *Retiro* (142 retiro del
  socio o accionista, 441 pago de dividendos) or *Pérdida* (659 otro gasto, 1419 cargo a un
  trabajador);
- an unspecified movement that **brought money in**: *Cobro* (759 otro ingreso, 142 devolución del
  socio o accionista, 1419 devolución de un trabajador, 501 aporte de capital);
- a **Cuadre Físico** shortage: 659 otro gasto or 1419 cargo al cajero; a surplus: 759 otro ingreso;
- a **Retiro**, **Pérdida** or **Cobro** already typed: only the accounts of its own type.

When the operation has nothing to classify the panel says *Todas las cuentas de esta operación se
derivan de su tipo; no hay nada que clasificar.* and has no **Guardar** button. That is the case for
sales, credit notes, receptions, cost of sales, production, payroll, transfers, document payments
and collections, loans, and purchase orders that only carry products.

### Business rules and rationale (Reglas y razón de negocio)

- The account is written **on the original record** — the expense, the purchase-order line, the
  asset or the cash movement — not on a separate journal. Genix rebuilds the entries from those
  records every time the period is read.
- **Classifying an asset classifies its acquisition, every depreciation and its disposal at once**,
  because all three take their accounts from the asset's account (and its depreciation pair). That
  is why a Depreciación or Baja de activo row offers the asset's account.
- A product line of a purchase order always posts to **6011 Mercaderías** and cannot be classified;
  only supply lines can.
- Classifying a supply line of a purchase order, or an inventory expense, also updates its
  warehouse receptions, so the **Ingreso a almacén** entry moves to the stock and variation pair of
  the new account (for example 6032 → Dr 252 · Cr 6132). A purchase account without a stock pair
  (656 or a 33x) never produces a reception entry.
- An unspecified cash movement is **retyped** by the option chosen: picking *Retiro · 142…* turns it
  into a Retiro. A money-out movement can only become a Retiro or Pérdida and a money-in movement
  only a Cobro. Once retyped, its type can no longer be changed here — only its account within that
  type.
- The server checks every account against the operation's allowed list and refuses an expense that
  was removed, a removed asset, or an account that is not a detail account of the right kind
  (*La cuenta contable no corresponde a este tipo de operación, o no es una cuenta de detalle*).

### Result and side effects (Resultado y efectos)

On success the page shows *Clasificación guardada.*, closes the panel and re-reads the period, so
the status line and every derived entry are rebuilt from what was saved. Because the account is
stored on the record itself, the **Cuenta contable** of the expense and asset forms shows it too, and
a retyped cash movement shows its new type in **Cajas & Bancos**.

### Limitations (Limitaciones)

Derived lines (IGV, 4212, 1212, cash, revenue, cost of sales…) can never be changed here. Amounts,
dates and documents cannot be edited here either: an **Incompleta** operation is fixed on its source
record (the sale, purchase, expense, asset or exchange rate), not by classifying.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo le pongo cuenta a un gasto?` Abra el gasto en **Movimientos**, elija la cuenta en
  *Clasificación contable* y presione **Guardar**.
- `¿Por qué no puedo cambiar la cuenta del IGV o de proveedores?` Son líneas derivadas, bloqueadas
  con candado.
- `¿Por qué no aparece la cuenta 60 en mi gasto?` Un gasto simple solo admite las cuentas de gasto
  (62, 63, 64, 65x, 67, 40); las 60x son para insumos.
- `¿Cómo clasifico un retiro de caja sin tipo?` Elija en **Tipo y cuenta** *Retiro* o *Pérdida* con
  su cuenta.
- Search terms: `clasificar`, `clasificación contable`, `cuenta contable`, `asignar cuenta`,
  `PCGE`, `plan contable`, `naturaleza`, `reclasificar`, `cambiar cuenta`.

<!-- DOC-ID: capability.bulk-classify -->
## Apply one account to many operations (Aplicar cuenta en bloque)

### User intention (Intención del usuario)

Classify many similar operations at once — for example every utility expense of the month to the
same 63x account.

### Where to find it (Dónde encontrarlo)

Tab **Movimientos**: click the tick box in the first column of each row to mark it. A bar appears
above the table with *N marcadas · M registros a clasificar*, the **Cuenta a aplicar** selector,
**Aplicar**, and **Limpiar** to untick everything.

### Required information and prerequisites (Requisitos previos)

Only rows of a single operation have a tick box; clicking that column on a grouped row opens its
list instead. The marked operations must share one account kind — expenses, supplies or assets —
or the bar says *Marque operaciones de un solo tipo (gastos, insumos o activos) para aplicar una
cuenta.* Supplies here means inventory expenses and purchase-order supply lines together.

### Business rules and rationale (Reglas y razón de negocio)

Operations without anything to classify, and **all cash movements**, are left out and counted as
*sin cuenta clasificable* — each cash type has its own options, so they are classified one by one.
On a purchase order, the account is applied to all its supply lines. **Aplicar** asks for
confirmation (*Aplicar cuenta*, showing the account and the number of records).

The save is **all or nothing**: every record is validated first and nothing is written unless all
of them pass, so a bulk apply never leaves the period half classified.

### Result and side effects (Resultado y efectos)

*Cuenta aplicada.* — the accounts are written on the records and the period is re-read, which also
clears the ticks. Reading another month clears them too.

### Limitations (Limitaciones)

Grouped rows cannot be ticked; open them and classify each operation in the side panel.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Puedo clasificar varios gastos a la vez?` Sí: márquelos en **Movimientos**, elija la
  **Cuenta a aplicar** y presione **Aplicar**.
- `¿Por qué no me deja aplicar la cuenta?` Marcó operaciones de tipos distintos (por ejemplo un
  gasto y un activo).
- Search terms: `masivo`, `en bloque`, `en lote`, `varios a la vez`, `aplicar cuenta`, `marcar`.

<!-- DOC-ID: capability.management-journal -->
## Review the management journal (Diario de gestión)

### User intention (Intención del usuario)

Read the month as a journal (`libro diario`): every entry with its CUO, glosa and accounts.

### Where to find it (Dónde encontrarlo)

Tab **Diario de gestión**. The search box of the top row filters it.

### Required information and prerequisites (Requisitos previos)

A period read with **Consultar**.

### Business rules and rationale (Reglas y razón de negocio)

One row per operation, never grouped: **Fecha**, **CUO**, **Glosa**, **Nº Doc**, and **Debe** and
**Haber** with **Cuenta** and **Monto**. It holds **every** entry, with and without comprobante;
those outside the official Diario carry a **sin comprobante** chip next to the glosa. Unclassified
lines show as a red warning, not hidden. Here the search also matches the CUO and the account codes.

### Result and side effects (Resultado y efectos)

Read only.

### Limitations (Limitaciones)

Entries cannot be classified from this tab; use **Movimientos**. The official SUNAT Diario is not
produced here: in **Libros** the *Diario Simplificado* option still answers *Este libro aún no está
disponible.*

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde veo el libro diario?` En la pestaña **Diario de gestión**.
- `¿El diario incluye las ventas sin comprobante?` Sí, marcadas *sin comprobante*.
- Search terms: `diario`, `libro diario`, `asientos`, `glosa`, `CUO`, `diario de gestión`.

<!-- DOC-ID: capability.trial-balance -->
## See the balance per account (Plan de cuentas)

### User intention (Intención del usuario)

See how much moved on each account in the month: *¿cuánto se cargó a 63 este mes?*

### Where to find it (Dónde encontrarlo)

Tab **Plan de cuentas**.

### Required information and prerequisites (Requisitos previos)

A period read with **Consultar**.

### Business rules and rationale (Reglas y razón de negocio)

A trial balance (`balance de comprobación`) of the period as a PCGE tree: **Cuenta**, **Debe**,
**Haber** and **Saldo** (Debe − Haber). Only accounts with movement in the period are listed. Each
line also adds up into every parent account (40111 adds to 4011, 401 and 40); grouping accounts
are in bold and each level is indented. Lines still unclassified appear last as
**— Sin clasificar**. It includes every entry, with and without comprobante.

### Result and side effects (Resultado y efectos)

Read only.

### Limitations (Limitaciones)

**Saldo is the period's movement only**, not a closing balance: there are no opening balances from
earlier months. Accounts without movement are not listed.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde veo el saldo de cada cuenta?` En **Plan de cuentas**; es el saldo del mes, sin saldo
  inicial.
- Search terms: `balance de comprobación`, `plan de cuentas`, `saldo por cuenta`, `mayor`,
  `cuentas contables`, `sumas y saldos`.

<!-- DOC-ID: entry-rules -->
## How Genix books each operation (Cómo se arma cada asiento)

| Operación (CUO) | Entry | Official Diario |
| --- | --- | --- |
| **Venta** (`V-`) | Dr 1212 · Cr 7012 (merchandise) / 7022 (made products), split by the value of the sale's lines · Cr 40111 IGV | only with comprobante |
| **Nota de crédito** (`NC-`) | the sale's entry reversed | yes |
| **Compra** (`OC-`) | Dr 6011 per product line, Dr the line's account per supply line · Dr 40111 · Cr 4212 (with comprobante) or 4699 (without) | only with comprobante |
| **Gasto** (`G-`) | Dr the expense's account · Dr 40111 · Cr 4212 or 4699 | only with comprobante |
| **Ingreso a almacén** (`IA-`) | Dr the stock account · Cr the 61x variation, from the purchase account's pair | follows its purchase |
| **Costo de ventas** (`CV-`) | Dr 6912 · Cr 2011 merchandise; Dr 6922 · Cr 2111 made product; Dr 6912 · Cr the supply's stock account; a return reversed | follows its sale |
| **Producción** (`PR-`) | consumed supply: Dr its variation · Cr its stock; produced: Dr 2111 · Cr 7111 (semi-finished: 231 / 713) | yes |
| **Ajuste de mano de obra** (`LA-`) | Dr 7111 · Cr 2111 on the last day | yes |
| **Activo fijo** (`AF-`) | Dr the asset's account (32x–34x) · Dr 40111 · Cr 4654; donated: Dr the asset's account · Cr 501 at its value | with comprobante, or donated |
| **Depreciación** (`DEP-`) | Dr 6841x · Cr 3952x, the depreciation pair of the asset's account | yes |
| **Baja de activo** (`BAJA-`) | Dr 3952x accumulated depreciation · Dr 6551 the rest · Cr the asset's account at its value | yes |
| **Movimiento de caja** (`CJ-`, `CJD-`) | 101 (caja) or 1041 (banco) against the counter-account | a collection or payment follows the sale, purchase, expense or asset it settles; the rest yes |
| **Planilla** (`PL-`), **Provisiones de planilla** (`PP-`) | from the accounts stamped on each payslip line, on the last day of the month | yes |

Details that change what the user sees:

- **IGV is separated only when the purchase comprobante grants crédito fiscal.** Without a
  comprobante the whole total is cost and is owed on **4699** instead of 4212, because 4212 means an
  issued invoice and no IGV can be credited without one. Its payment uses the same rule, so the
  payable returns to zero.
- A purchase is dated by its comprobante's issue date (or its own date when it has none); a fixed
  asset by its comprobante's issue date, else its acquisition date. Payroll is dated on the last day
  of the month, and only **closed or paid** payslips appear. A payslip line that carries no account
  stays *Sin clasificar*: Genix never guesses it, and this page cannot classify it.
- A payment that settled its document **on the document's own date** has no movement row of its
  own: Genix infers it from the document and gives it a `CJD-` CUO. It cannot be classified.
- A **transfer** between cash accounts is booked once, from its outgoing side (Dr destination · Cr
  origin).
- A voided or rejected comprobante, an annulled sale, a canceled purchase order and a removed
  expense produce no entry.
- The **Ajuste de mano de obra** appears only once the month has payslips: when the labor booked to
  products by a payroll worker is higher than that worker's actual payroll cost, the excess is taken
  back from the stored production.
- Collections and payments of a sale or purchase **without comprobante**, and the cost of an
  undocumented sale, stay out of the official Diario with it — otherwise the official book would show
  receivables, payables or stock changes that no document there explains.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- **The user decides the nature, Genix decides the rest.** Only nature lines are classifiable;
  derived lines are locked on every tab and the server refuses to classify any other kind of
  operation (*sus cuentas son derivadas*).
- **The entries are rebuilt on every read** from the source records, so a change made on another
  page (an expense's amount, a new exchange rate, a reception) shows here after **Consultar**.
- **Status precedence**: any data problem makes the operation *Incompleta*, even if it also lacks an
  account; otherwise a missing account makes it *Sin clasificar*.
- **Stock movements other than purchase receptions, inventory-expense receptions, sale deliveries
  and returns, and production produce no entry**: stock adjustments (`ajustes de stock`), opening
  stock (`stock inicial`), internal consumption and shrinkage (`mermas`) are not booked here.
- The **Estados Financieros** page reads the period exactly the same way. Its income statement
  leaves out the unclassified lines and an incomplete entry may be missing amounts, so it sends the
  user back here (*Revíselas en Consolidado & Cuentas*) before trusting the figures.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

Open the operation in **Movimientos** to read its problem in the yellow notice, or hover its red
accounts.

- **Un monto en dólares no tiene tipo de cambio.** No company or BCRP sell rate exists within the 7
  days up to that date (or a dollar purchase comprobante carries no exchange rate), so the amount
  counts as 0. Load the rate in **Tipo de Cambio**, or correct the purchase, and press **Consultar**
  again.
- **Venta con stock por regularizar: su costo de ventas se registra en el cierre de producción.** A
  sale of made products was delivered before the daily production close. Run the close (**Cierre
  del día**) and read the period again.
- **No se encontró la venta de este comprobante.** / **No se encontró la compra de este ingreso.**
  The sale or purchase behind the record could not be read; check it on its own page.
- **Una entrega salió del almacén sin costo.** / **Un ingreso no tiene costo.** / **Un movimiento de
  producción no tiene costo.** The warehouse movement was recorded with no cost value.
- **La cuenta de un insumo entregado no tiene cuenta de existencias.** / **La cuenta de un insumo
  consumido no tiene cuenta de existencias.** The supply's account has no stock pair (for example
  656 Suministros). Change its **Cuenta contable por defecto** in **Insumos & Materiales** to a 60x
  stock account if it should be held in stock.
- **La cuenta N no tiene cuentas de depreciación.** The asset's account has no depreciation pair;
  classify the asset (on its Activo fijo, Depreciación or Baja de activo row) with an account that
  has one.
- **El Debe y el Haber no cuadran.** The entry's sides differ, usually because the source record's
  amounts are inconsistent; correct the record.
- **Tipo de movimiento de almacén no contemplado.** A stock movement of a type no entry handles
  reached the page; it is shown as *Movimiento de almacén* so it is not lost silently.
- **No se pudo leer el periodo.** / **No se pudo guardar la clasificación.** Retry. When a save is
  refused, the message explains what the server rejected, and nothing of that save was written.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Gastos** (`/finance/expenses`): register expenses with their **Cuenta contable**. An expense saved
  with an account — picked, or its category's default — arrives here already classified; one saved
  without any arrives *Sin clasificar*.
- **Activos** (`/accounting/assets`): acquire, pay and dispose of fixed assets; the asset form has
  the same account selector.
- **Insumos & Materiales** (`/production/supplies-materials`): the **Cuenta contable por defecto**
  of each supply, which its purchase lines and inventory expenses start from.
- **Cajas & Bancos** (`/finance/cash-banks`): register cash movements; a retyped movement shows its
  new type there.
- **Tipo de Cambio** (`/finance/exchange-rate`): load the company's rates when a dollar operation is
  incomplete.
- **Cierre del día** (`/production/daily-close`): settles sales with stock to regularize.
- **Libros** (`/accounting/books`): the SUNAT sales and purchases registers.
- **Estados Financieros** (`/accounting/financial-statements`): the income statement and cash flow
  built from these same entries, in a *Gestión* or *Oficial* view.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.read-period]
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.read-period, related-pages]
  - path: frontend/routes/accounting/accounts-consolidation/+page.svelte
    role: page
    hash: sha256:7fe758db2766c328e302fecad357e9a32cc9307ef549de09de22309a56c0efa2
    supports: [page-purpose, capability.read-period, capability.review-operations, capability.management-journal, capability.trial-balance]
  - path: frontend/routes/accounting/accounts-consolidation/OperationsTable.svelte
    role: user-interface
    hash: sha256:c5b349f8a98509e8345c5700907fa5ce33d394c2946d2c3faf2be24c5c3808d2
    supports: [capability.review-operations, capability.classify-operation, capability.bulk-classify, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/OperationEntryLayer.svelte
    role: user-interface
    hash: sha256:6eff3423c23ec13f61987b7ce290710c0ce35f75f4ff5fd82ac03caf782da902
    supports: [concepts, capability.classify-operation, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/JournalTable.svelte
    role: user-interface
    hash: sha256:1cf2413b63087827f19059a3373b3f1fa036e3a25e3084f4f93a3b0ac31a9d99
    supports: [concepts, capability.management-journal]
  - path: frontend/routes/accounting/accounts-consolidation/ChartOfAccounts.svelte
    role: user-interface
    hash: sha256:9a7f31dd66f85ca61620d98b542e2a14789541267d666d2703c5959cddcd8457
    supports: [capability.trial-balance]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.cells.ts
    role: user-interface
    hash: sha256:a81ebba97480e4adcc51b50091c7d4959267a568cb6e110a36d29bc6c4924a9b
    supports: [concepts, capability.review-operations, capability.management-journal, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.ts
    role: business-logic
    hash: sha256:7cdc814a6eb91a7ecb6038243e7e7c6dafa176029d1c48eaf11dbf8188f6bf6c
    supports: [concepts, capability.read-period, capability.review-operations, capability.classify-operation, capability.bulk-classify, capability.trial-balance, entry-rules, rules, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.svelte.ts
    role: frontend-service
    hash: sha256:63800ebe3f249db9471e2aed07f2e0523b07d966e7765a616ca0aa1d897acb23
    supports: [capability.read-period, capability.classify-operation, troubleshooting]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.utils.ts
    role: business-logic
    hash: sha256:6155814e126e855e85682b0dff8bfd1d9b5371238a20002a78464e2b7b01580b
    supports: [capability.read-period, entry-rules]
  - path: frontend/routes/accounting/accounts-consolidation/RATIONALE.md
    role: reference-document
    hash: sha256:9f87c7ccd0b7b97713d0524736522219aaed15d772ed9344312e508c5c3b8a55
    supports: [capability.classify-operation, capability.bulk-classify, capability.trial-balance, entry-rules]
  - path: frontend/routes/accounting/accounts-consolidation/PLAN.md
    role: reference-document
    hash: sha256:a7c54c99411ccd899772091e05d1900027f11c95773aa26906cec0be5ddeef7e
    supports: [page-purpose, concepts]
  - path: backend/accounting/accounting_operations_api.go
    role: backend-handler
    hash: sha256:c8fe220fc549d253728fe36977a078132b61a95dcaa094f7116d4ce810907569
    supports: [page-purpose, capability.read-period, entry-rules, rules]
  - path: backend/accounting/accounting_classification_api.go
    role: backend-handler
    hash: sha256:6005f435b2594659c0a85a0cd7462abf881e59e1105ce7a224218e8b2c514d6b
    supports: [capability.classify-operation, capability.bulk-classify, rules]
  - path: backend/accounting/purchases_book_api.go
    role: backend-handler
    hash: sha256:12920785965e9598eb546cafcd5dd9fe9c255c86005cc454d2bf51b7d99012ef
    supports: [entry-rules]
  - path: backend/accounting/types/pcge.go
    role: shared-domain
    hash: sha256:b1eb5ab7555176b1940ee05020c241940c3e0a82dc6c33b06276a0209e6af753
    supports: [capability.classify-operation]
  - path: frontend/core/pcge.ts
    role: shared-domain
    hash: sha256:26947ab13508fceeaed84d577ecb117a530e0df9a9ec314dbcb253d31c322118
    supports: [capability.classify-operation, capability.trial-balance, troubleshooting]
  - path: frontend/domain-components/PCGEAccountSelect.svelte
    role: user-interface
    hash: sha256:37de375271ebb515d3d626237b578d8dd1d0590be1542233f9f7698fef22caf7
    supports: [capability.classify-operation, capability.bulk-classify]
  - path: backend/finance/types/cash_movement_account.go
    role: business-logic
    hash: sha256:2c462a5021a979a7ce37405eb667f99e784b699f75e024c419c184e239a93eba
    supports: [capability.classify-operation]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:051c429f6af7fbc3031301c0b052febfdbd06b5b71a689d695cedbc538e4a1c7
    supports: [capability.classify-operation]
  - path: backend/logistics/types/product-stock-movement.go
    role: business-logic
    hash: sha256:ea5c66a445dfbb7aac4e969165014af8e46cd6ca0f47bc633508bd0f5c049a6d
    supports: [capability.classify-operation]
  - path: backend/logistics/types/purchase_order.go
    role: business-logic
    hash: sha256:c13fc922ef507f06ccba0b7dbc257311d6edf9cc8162b13ecd56221e14a7c23a
    supports: [capability.classify-operation]
  - path: backend/logistics/types/stock_movement_type.go
    role: data-model
    hash: sha256:ee09b510d7ad0e5bc4011e348ef2bcc30f8241fc8892a6f970589eebb00d8127
    supports: [rules]
  - path: frontend/routes/accounting/books/books.ts
    role: shared-domain
    hash: sha256:7e26d19318088cd3acf2be59f90ba0bebfe851efa2279d6b149fdf403a8a6555
    supports: [capability.read-period, capability.management-journal]
  - path: frontend/routes/accounting/books/+page.svelte
    role: user-interface
    hash: sha256:25cb3fccc7b5614cf71c6f65348ef56e8424e69eccf1a308445059888a8aaa45
    supports: [capability.management-journal]
  - path: frontend/routes/finance/exchange-rate/exchange-rate.ts
    role: business-logic
    hash: sha256:5a5b23729c283c270bb18dfea2b17758572f9405c2fa9a01cbdf96d98d6c0cba
    supports: [capability.read-period, troubleshooting]
  - path: frontend/routes/accounting/financial-statements/+page.svelte
    role: user-interface
    hash: sha256:2213254c95698ac854c2cae42dc0bddc427c2ee4b02c197e3816b2375e9953f9
    supports: [rules, related-pages]
  - path: frontend/routes/accounting/financial-statements/financial-statements.ts
    role: business-logic
    hash: sha256:653b463bac2d450c5b856a5afbc24459021435741a545abe09b67f7f7052d014
    supports: [related-pages]
  - path: backend/finance/expenses.go
    role: business-logic
    hash: sha256:88afa304e7aa3b3b96f3c02055efd98fe1a9da60b869dfd08de6d2f10780fd05
    supports: [related-pages]
  - path: backend/logistics/purchase-order-management.go
    role: business-logic
    hash: sha256:833a7dade609a4b174f3ffc3bb264a563437ff1dc37c5345efba6541e04ccf00
    supports: [related-pages]
  - path: frontend/routes/finance/expenses/ExpensesRegister.svelte
    role: user-interface
    hash: sha256:d321929caabc8bafafe601a9692222322eba6a62c78f4620f063f5717e121f10
    supports: [capability.classify-operation, related-pages]
  - path: frontend/routes/accounting/assets/AssetForm.svelte
    role: user-interface
    hash: sha256:3c75246ce1225a3caf3398314964c0fa9b9e208a43da61a8d22989b16781baea
    supports: [capability.classify-operation, related-pages]
  - path: frontend/routes/production/supplies-materials/+page.svelte
    role: user-interface
    hash: sha256:a40d5e4f4bf55ffdf9db385897029e4b2c41671104724dc06d526c063cfdfb73
    supports: [troubleshooting, related-pages]
  - path: frontend/packages/genix-ui/layers/Layer.svelte
    role: user-interface
    hash: sha256:613370cc48ba043dbf0ef6c77680c54f9b9703a07e949a5900a660a71b56cb52
    supports: [capability.classify-operation]
```
