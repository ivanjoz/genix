---
schema: 1
page_id: accounting.books
route: /accounting/books
title: Accounting Books (Libros Contables)
status: implemented
visibility: tenant
description_en: >-
  SUNAT accounting books for a month: the Registro de Ventas built from the issued comprobantes
  and the Registro de Compras built from the supplier comprobantes of purchase orders, expenses
  and fixed assets. Review the month or a single day, see sales and purchases without a
  comprobante, export the book to Excel, and download the SIRE replacement TXT once the period
  has nothing blocking it.
description_es: >-
  Libros contables SUNAT de un mes: el Registro de Ventas armado con los comprobantes emitidos y
  el Registro de Compras armado con los comprobantes de proveedor de órdenes de compra, gastos y
  activos fijos. Revisar el mes o un día, ver ventas y compras sin comprobante, exportar el libro a
  Excel y descargar el TXT de reemplazo para SIRE cuando el periodo no tiene nada que lo impida.
---

# Accounting Books (Libros Contables)

<!-- DOC-ID: page-purpose -->
## Page purpose

Accounting Books (`Libros Contables`, menu entry **Libros**) shows the company's SUNAT tax
registers for a month (`periodo`): the sales register (**Registro de Ventas**, the RVIE of SIRE)
and the purchases register (**Registro de Compras**, the RCE of SIRE). It answers the month-end
questions *¿cuánto vendí y cuánto IGV debo declarar?*, *¿cuánto crédito fiscal tengo este mes?*
and *¿ya puedo presentar el libro?*, and it produces the files to hand over: an **Excel** of the
book and the **TXT SUNAT** replacement file (`archivo de reemplazo`) for SIRE.

The page **only reads**. It never creates, edits or annuls a comprobante, a sale or a purchase.
Every row comes from records registered elsewhere:

- sales rows are the electronic comprobantes (facturas and boletas) that Genix issued with each
  sale, the same documents listed in **Facturación**;
- purchase rows are the supplier comprobantes (`comprobante del proveedor`) registered on a
  purchase order (**Órdenes Compra**), an expense (**Gastos**) or a fixed asset (**Activos**).

Genix does **not** upload anything to SUNAT from this page: the TXT is downloaded to the user's
computer. The third book in the selector, **Diario Simplificado**, is not available yet.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Libro / registro**: a monthly register of comprobantes in the shape SUNAT asks for. A book
  is a register of **documents**, not of operations: a sale or a purchase without a comprobante
  is never a book row. Genix lists those apart, only so the month can be reviewed.
- **Periodo**: the month a book belongs to, written as SUNAT writes it, `YYYYMM` (for example
  `202609`), and shown next to its label (`SET 2026`). A sales comprobante belongs to the month
  of its issue date (`fecha de emisión`). A purchase belongs to the month of the **supplier
  comprobante's issue date**, not the date of the order or the expense; a purchase without a
  comprobante is placed by its own date.
- **Estado** of a sales row (`estado de validez`): what SUNAT answered for the comprobante,
  read from its state in **Facturación**:
  - **Activo** — SUNAT accepted it (*Aceptado* or *Aceptado con observaciones*). It declares
    its amounts.
  - **Baja** — the comprobante was voided (*Anulado* in Facturación). It keeps its row with
    amounts at zero.
  - **Rechazado** — SUNAT rejected it. It keeps its row with amounts at zero.
  - **Pendiente SUNAT** — SUNAT has not answered yet (*Pendiente de envío*, *Enviando* or
    *Error de envío* in Facturación). This is not a SUNAT state: Genix shows it so the user can
    see the month is not ready, and the period cannot be exported to TXT while any row is in it.
- **Boletas consolidadas**: boletas under S/ 700 of the same day and series are declared as one
  row covering a range of numbers (`Nro inicial — Nro final`) instead of one row each (see
  *Registro de Ventas* below).
- **Crédito fiscal**: the IGV on purchases that the company can deduct. Only a purchase with a
  supplier comprobante is in the Registro de Compras; a purchase without one gives no crédito
  fiscal.
- **Archivo de reemplazo (TXT SUNAT)**: the flat text file that replaces SUNAT's proposal
  (`propuesta`) of the RVIE or the RCE in SIRE. Genix writes it from exactly the rows shown on
  screen.

<!-- DOC-ID: capability.read-book -->
## Read a book for a month or a day (Consultar un libro)

### User intention (Intención del usuario)

Open the sales or purchases register of a month to review it before declaring, or look at a
single day to check what was registered that day.

### Where to find it (Dónde encontrarlo)

**Finanzas y Contabilidad → Contabilidad → Libros** (`/accounting/books`). The page title is
**Libros Contables**. The toolbar has three selectors and three buttons:

- **Libro**: **Ventas**, **Compras** or **Diario Simplificado**.
- **Fecha**: a single day. It is disabled while **Mes** holds a month.
- **Mes**: **Día específico** followed by the last 24 months, newest first (`SET 2026`,
  `AGO 2026`, …). It opens on the current month.
- **Consultar** reads the book; **Excel** and **TXT SUNAT** export it.

To the right, **Periodo** shows the period the rows belong to, as code and label
(`202609 SET 2026`).

### Required information and prerequisites (Requisitos previos)

A book (**Ventas** or **Compras**) and either a month in **Mes**, or **Día específico** in
**Mes** plus a day in **Fecha**. Nothing is loaded when the page opens: the book appears only
after pressing **Consultar**.

### Business rules and rationale (Reglas y razón de negocio)

- Choosing a month locks **Fecha**, because the book is the whole month. To look at one day,
  pick **Día específico** in **Mes**; that re-enables **Fecha**.
- A single day is a **working view, not a book that can be filed**: the rows are filtered to
  that day, **Periodo** still shows the month the day belongs to, and the TXT export is refused
  for it (*Un día suelto no es un libro presentable — elige un mes.*).
- While reading, the page shows *Leyendo el libro...*. Pressing **Consultar** again re-reads the
  month from the server, so changes made since the last read are included.
- After every read, if anything prevents the period from being filed, Genix sends **one yellow
  message** titled *El periodo SET 2026 todavía no se puede presentar*, with the subtitle
  *Libros Contables · Registro de Ventas* (or *Registro de Compras*) and one line per problem
  to fix (see *Exportar el TXT SUNAT*). The message is kept in the notifications panel, tab
  **Mensajes**, and pressing **Consultar** sends it again.

### Result and side effects (Resultado y efectos)

The table and the totals of the chosen book are shown. Nothing is written.

### Limitations (Limitaciones)

- Selecting **Diario Simplificado** shows *Este libro aún no está construido.*, and
  **Consultar** answers *Este libro aún no está disponible.* There is no Libro Diario on this
  page yet.
- With **Día específico** and no date, **Consultar** answers *Elige un mes o una fecha.*
- Changing **Libro**, **Mes** or **Fecha** does not reload the table by itself; press
  **Consultar** again.
- There is no range of several months and no annual book: one month (or one day) per read.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde veo el registro de ventas del mes?` En **Libros**, elija **Ventas**, el mes en
  **Mes** y presione **Consultar**.
- `¿Cómo veo solo las ventas de un día?` En **Mes** elija **Día específico**, luego la
  **Fecha**, y presione **Consultar**.
- `¿Por qué no puedo elegir la fecha?` Porque hay un mes elegido; cambie **Mes** a **Día
  específico**.
- Search terms: `libro de ventas`, `registro de ventas`, `RVIE`, `libro de compras`,
  `registro de compras`, `RCE`, `SIRE`, `PLE`, `libros electrónicos`, `periodo`, `mes`,
  `cierre de mes`, `declaración mensual`, `libro diario`.

<!-- DOC-ID: capability.sales-register -->
## Review the sales register (Revisar el Registro de Ventas)

### User intention (Intención del usuario)

Check the month's issued comprobantes and the amounts that go into the sales declaration: base
imponible, IGV and total, and see which documents are still waiting for SUNAT.

### Where to find it (Dónde encontrarlo)

**Libro** = **Ventas**, then **Consultar**. The view strip has two options; **Contabilizadas**
is the book itself (the other, **Sin Comprobante**, is described below). The table columns are
**Comprobante**, **Tipo**, **Emisión**, **Cliente**, **Doc.**, **Base imp.**, **IGV**,
**Total**, **Modifica** and **Estado**. Above the table, at the right, the totals strip shows
**Comprobantes**, **Base imp.**, **IGV** and **Total** for the rows of the book.

### Required information and prerequisites (Requisitos previos)

Comprobantes issued in that month. The series code and the document type of each row come from
the company's **Series de Facturación** (Mi Empresa → Configuración, tab **Facturación**); the
buyer's name and document come from the customer identity recorded on the comprobante.

### Business rules and rationale (Reglas y razón de negocio)

- **Every comprobante of the month has a row**, including voided and rejected ones. A **Baja**
  or **Rechazado** row keeps its number but shows its amounts at **0.00**, because its
  correlativo was used and SUNAT expects to see it, while it declares no sale.
- **Pendiente SUNAT** rows keep their amounts and are counted in the totals, so the month is not
  understated, but they block the TXT until SUNAT answers.
- **Daily consolidation of boletas (`consolidación diaria`)**: within the same day and series,
  boletas that are **under S/ 700** and **Activo** and have **consecutive numbers** are merged
  into one row. Its **Comprobante** shows the range (`B001-120 — 135`), its **Estado** says how
  many boletas it holds (`Activo · 16 blt.`), and **Cliente** and **Doc.** are left empty. A
  range is cut, and the boleta stays on its own row, by:
  - a boleta of **S/ 700 or more**, which SUNAT requires declared individually with its buyer;
  - a boleta that is **Baja**, **Rechazado** or **Pendiente SUNAT**, which must appear on its
    own;
  - a **gap in the numbering**.
  A run of a single boleta stays a normal row. Consolidation is always applied; there is no
  option to turn it off. A boleta with an identified buyer (DNI or RUC) under S/ 700 is still
  consolidated: the identity stays on the sale and on the comprobante, not in the book.
- A boleta issued **without identifying the customer** shows **Cliente** `CLIENTES VARIOS` and
  **Doc.** `00000000`, exactly what the comprobante itself declared.
- **Comprobantes** in the totals strip counts each boleta inside a consolidated row, not the row.
- **Modifica** is only filled for a **Nota de Crédito** or **Nota de Débito**: the series and
  number of the comprobante it corrects, even when that document is from an earlier month. It is
  empty for facturas and boletas.
- Rows are ordered by issue date, then series, then number.

### Result and side effects (Resultado y efectos)

Reading only. Nothing changes on the comprobantes or the sales.

### Limitations (Limitaciones)

- Genix does not issue notas de crédito or débito from a sale today (a sale cannot be registered
  with a note series), so **Modifica** is normally empty.
- The amounts are shown as stored, without converting dollars to soles. A dollar comprobante
  blocks the TXT export (see *Exportar el TXT SUNAT*).
- The book does not answer *¿quién compró en esta boleta?* for a consolidated row; open the sale
  or the comprobante in **Facturación** instead.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué varias boletas aparecen en una sola fila?` Son boletas menores a S/ 700 del mismo
  día y serie con números consecutivos; SUNAT permite declararlas como un rango.
- `¿Por qué una factura aparece con importe cero?` Está en **Baja** o **Rechazado**: se anota
  con 0.00 pero no se omite.
- `¿Qué significa Pendiente SUNAT?` Que SUNAT aún no respondió ese comprobante; revíselo en
  **Facturación**.
- `¿Cuánto IGV vendí este mes?` Es el **IGV** de la franja de totales de **Contabilizadas**.
- Search terms: `boletas consolidadas`, `rango de boletas`, `resumen diario`, `clientes varios`,
  `base imponible`, `IGV ventas`, `débito fiscal`, `anulado`, `baja`, `rechazado`,
  `nota de crédito`, `Formulario 621`, `PDT 621`.

<!-- DOC-ID: capability.uninvoiced-sales -->
## Review sales without a comprobante (Ventas sin comprobante)

### User intention (Intención del usuario)

Answer *¿qué vendí este mes sin emitir comprobante?* during the month-end review.

### Where to find it (Dónde encontrarlo)

**Libro** = **Ventas**, **Consultar**, then the **Sin Comprobante** option of the view strip.
The table has the columns **Venta** (the sale's id), **Fecha** and **Total**, with the sum of the
list below it.

### Required information and prerequisites (Requisitos previos)

Sales of the month registered without an invoicing series, for example generated with **SIN
COMPROBANTE** at the Point of Sale.

### Business rules and rationale (Reglas y razón de negocio)

The view lists the sales of the chosen month (or day) that were never registered for electronic
issuance. Annulled sales are not listed. The page states it above the table: *No forman parte del
libro — el Registro de Ventas registra comprobantes. Se muestran para revisar el periodo.*

### Result and side effects (Resultado y efectos)

Reading only.

### Limitations (Limitaciones)

These sales are **not part of the Registro de Ventas**: they are not in the book's totals, in
the Excel or in the TXT. Their amounts are not split into base and IGV here. This page cannot
issue a comprobante for them.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Dónde veo las ventas que no tienen boleta ni factura?` En **Ventas → Sin Comprobante**.
- `¿Las ventas sin comprobante entran al libro?` No; solo se muestran para revisar el mes.
- Search terms: `ventas sin comprobante`, `ventas no facturadas`, `sin boleta`, `nota de venta`,
  `ventas internas`.

<!-- DOC-ID: capability.purchases-register -->
## Review the purchases register (Revisar el Registro de Compras)

### User intention (Intención del usuario)

Check the supplier comprobantes of the month and the crédito fiscal (IGV de compras) they give,
before declaring.

### Where to find it (Dónde encontrarlo)

**Libro** = **Compras**, then **Consultar**. The table columns are **Comprobante** (series and
number), **Tipo**, **Emisión**, **Proveedor**, **RUC**, **Base imp.**, **IGV**, **No grav.**,
**Total** and **Origen**. The totals strip shows **Comprobantes**, **Base imp.**, **IGV** and
**Total (S/)**.

### Required information and prerequisites (Requisitos previos)

The supplier's comprobante must have been registered on the purchase, in its **Comprobante**
fields (type, **Serie**, **Número**, **Fecha de Emisión**, amounts, currency): on a purchase
order in **Órdenes Compra**, on an expense in **Gastos**, or on a fixed asset in **Activos**.
The supplier's name and document come from the supplier identity recorded on that purchase.

### Business rules and rationale (Reglas y razón de negocio)

- **Tipo** is one of the purchase comprobantes Genix accepts: **Factura**, **Recibo por
  Honorarios**, **Boleta**, **Ticket** or **Recibo de Servicios Públicos**.
- A purchase enters the month of its **comprobante's issue date**. A factura dated in August
  but entered in September is in the August book.
- **Origen** says where each row comes from, with its id: **Orden de compra**, **Gasto** or
  **Activo fijo**. **One comprobante registered on several records is one row**: records with
  the same supplier, type, series and number are added together and **Origen** lists all of
  them, so the factura is declared once (for example five serialized assets bought on one
  factura, or a factura split between an order and an expense).
- Only live purchases count: canceled purchase orders, removed expenses and asset depreciation
  entries are not purchases and never appear. An asset received without a comprobante and
  without a purchase amount (a donation) is not listed either.
- **Total** shows each row in its own currency, with `USD` after a dollar amount. The totals
  strip is in **soles**: each dollar row is converted at the exchange rate registered on its
  comprobante.
- Rows are ordered by issue date, then series, then number.

### Result and side effects (Resultado y efectos)

Reading only. Nothing changes on the purchases.

### Limitations (Limitaciones)

- **Supplier notas de crédito and débito cannot be registered yet**, so the book never has a
  negative row for a supplier note.
- All crédito fiscal is booked as destined to taxed sales. A company that also makes exonerated
  or unaffected sales cannot split the IGV of a purchase between taxed and untaxed operations
  here.
- A supplier comprobante that was never entered in Genix is not in this book, even if it
  appears in SUNAT's proposal.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cuánto crédito fiscal tengo este mes?` Es el **IGV** de la franja de totales del libro
  **Compras**, en soles.
- `¿Por qué mi factura de compra sale en otro mes?` Porque el libro usa la fecha de emisión del
  comprobante del proveedor, no la fecha de la orden o del gasto.
- `¿Por qué una factura sale una sola vez si la registré en varios activos?` Porque el libro
  agrupa el mismo comprobante y suma sus partes; **Origen** muestra todos los registros.
- Search terms: `libro de compras`, `registro de compras`, `RCE`, `crédito fiscal`,
  `IGV compras`, `factura de proveedor`, `recibo por honorarios`, `recibo de luz`,
  `servicios públicos`, `ticket`, `compras en dólares`.

<!-- DOC-ID: capability.undocumented-purchases -->
## Review purchases without a comprobante (Compras sin comprobante)

### User intention (Intención del usuario)

Find the purchases of the month that were registered without the supplier's comprobante, so it
can be added before declaring.

### Where to find it (Dónde encontrarlo)

**Libro** = **Compras**, **Consultar**, then tick the checkbox **Compras sin comprobante (N)**
(with a warning icon) above the table. A second table appears under the book, with the columns
**Origen**, **Fecha**, **Compra**, **Proveedor** and **Monto**. The number in the checkbox label
is how many there are in the period.

### Required information and prerequisites (Requisitos previos)

None. The list is hidden until the checkbox is ticked.

### Business rules and rationale (Reglas y razón de negocio)

A purchase without a comprobante is placed in the month of its own date (the order, expense or
asset date). The page explains above the list: *No forman parte del libro — estas compras no
tienen comprobante del proveedor, así que no dan crédito fiscal. Regístralo en la compra para
anotarla.*

### Result and side effects (Resultado y efectos)

Reading only. Once the comprobante is registered on the purchase (in **Órdenes Compra**,
**Gastos** or **Activos**), pressing **Consultar** again shows it in the book of the month of
the comprobante's issue date.

### Limitations (Limitaciones)

These purchases are not in the book, its totals, the Excel or the TXT, and they do **not** block
the TXT export.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué mi compra no aparece en el registro de compras?` Probablemente no tiene
  comprobante; marque **Compras sin comprobante** para verla.
- Search terms: `compras sin factura`, `compras sin comprobante`, `gastos sin comprobante`,
  `sin crédito fiscal`.

<!-- DOC-ID: capability.export-excel -->
## Export the book to Excel (Exportar a Excel)

### User intention (Intención del usuario)

Hand the book to the accountant (`contador`), or check it column by column against the SUNAT
format.

### Where to find it (Dónde encontrarlo)

The green **Excel** button in the toolbar, after **Consultar**.

### Required information and prerequisites (Requisitos previos)

A read book with at least one row; otherwise the page answers *No hay nada que exportar.*

### Business rules and rationale (Reglas y razón de negocio)

- The file is named `registro-ventas-YYYYMM.xlsx` or `registro-compras-YYYYMM.xlsx`, with the
  title *Registro de Ventas — SET 2026* (or *Registro de Compras — …*).
- Columns follow SUNAT's field order and the header carries the field number, for example
  `5 · Fecha emisión`, `15 · Base imponible`, `17 · IGV`, `26 · Total`. Amounts are numbers, so
  they can be summed.
- The sales sheet adds three columns that are not SUNAT fields: **Estado**, **Boletas
  consolidadas** (how many boletas a range holds) and **Comprobante** (the type's name). The
  purchases sheet adds **Comprobante** and includes `6 · Vencimiento` and
  `27 · Tipo de cambio`; its amounts stay in each document's currency.
- The Excel exports the rows exactly as shown, including consolidated boleta ranges and
  zero-amount rows. It can be exported for a single day too.

### Result and side effects (Resultado y efectos)

An `.xlsx` file is downloaded. Nothing changes in Genix.

### Limitations (Limitaciones)

Only the book rows are exported: never the **Sin Comprobante** sales or the **Compras sin
comprobante** list. The Excel is not a file SUNAT accepts; use **TXT SUNAT** for that.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo le paso el registro de ventas a mi contador?` Consulte el mes y presione **Excel**.
- Search terms: `exportar`, `Excel`, `xlsx`, `descargar libro`, `reporte de ventas`,
  `reporte de compras`.

<!-- DOC-ID: capability.export-sunat-txt -->
## Export the SUNAT TXT file (Exportar el TXT SUNAT)

### User intention (Intención del usuario)

Get the file that replaces SUNAT's proposal of the Registro de Ventas or Compras in SIRE, so the
book SUNAT receives is exactly what Genix holds.

### Where to find it (Dónde encontrarlo)

The **TXT SUNAT** button in the toolbar. When the period can be exported, a line under the view
selector shows **Archivo de reemplazo** followed by the file name that will be downloaded.

### Required information and prerequisites (Requisitos previos)

- A **whole month** read with **Consultar**.
- The company's **RUC** (11 digits) and **Razón Social**, from Mi Empresa → Configuración, tab
  **Mi Empresa**. When the Razón Social is empty, the company **Nombre** is used instead.
- No blocking problem in the period (below).

### Business rules and rationale (Reglas y razón de negocio)

The button stays **disabled** while anything would make SUNAT reject the file. Every reason is
listed at once in the yellow message sent after **Consultar**, one per line, so a single read
tells what to fix:

- both books:
  - *Un día suelto no es un libro presentable — elige un mes.*
  - *La empresa no tiene un RUC válido (campo 1).*
- Registro de Ventas:
  - *N comprobantes siguen esperando la respuesta de SUNAT.* — rows in **Pendiente SUNAT**
    would be declared as sales SUNAT may still reject.
  - *N comprobantes no están en PEN y el ERP no guarda tipo de cambio (campo 28).* — SUNAT
    requires the exchange rate for a non-soles comprobante, and Genix stores none for sales.
  - *N filas no tienen serie — revisa la configuración de series de la empresa.* — the
    comprobante's series is not found in the company's **Series de Facturación**, so the row has
    no type and no series.
- Registro de Compras:
  - *N comprobantes no tienen la identidad del proveedor (campos 12-14) — guarda el proveedor
    en CRM.* — the supplier's document and name would go out blank.
  - *N comprobantes en dólares no tienen tipo de cambio (campo 27).*
  - *N recibos de servicios públicos no tienen fecha de vencimiento (campo 6).*

A month **with no comprobantes is not a blocker**: it exports as a book "sin información", with
an empty file whose name carries that flag. Sales and purchases without a comprobante are not
blockers either.

The file name follows SUNAT's mask: `LE` + RUC + period + `00` + book code (`140400` for the
Registro de Ventas, `080400` for the Registro de Compras) + `02` (replaces the proposal) + `1`
(company operating) + `1` with information or `0` without + `1` (soles) + `2` + `.TXT`. For
example `LE2012345678920260900140400021112.TXT`.

The file holds one line per book row, the same rows as the screen, so consolidated boleta ranges
and zero-amount **Baja**/**Rechazado** rows are in it. It is written in the encoding SUNAT reads
(ISO-8859-1), so `Ñ` and accented letters are kept; the characters `|`, `/` and `\` are removed
from names, because SUNAT forbids them in free text. In the Registro de Compras each amount
stays in the document's currency, with the exchange rate on dollar rows.

### Result and side effects (Resultado y efectos)

The `.TXT` file is downloaded. Nothing changes in Genix and **nothing is sent to SUNAT**: the
user uploads the file to SIRE.

### Limitations (Limitaciones)

- No TXT for a single day, and none while a blocking problem remains.
- The file **replaces** SUNAT's proposal: a comprobante missing from Genix (for example a
  supplier factura never entered) is missing from the filed book.
- Genix does not upload the file to SIRE nor read SUNAT's answer to it.
- There is no TXT for the Diario Simplificado.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Por qué el botón TXT SUNAT está deshabilitado?` Lea el mensaje amarillo que llega al
  presionar **Consultar** (también en **Mensajes** de las notificaciones): lista cada problema
  del periodo.
- `¿Puedo presentar un mes sin ventas?` Sí; se exporta como libro sin información.
- `¿Genix envía el libro a SUNAT?` No; descarga el archivo para subirlo a SIRE.
- Search terms: `TXT`, `archivo plano`, `SIRE`, `RVIE`, `RCE`, `propuesta`, `reemplazo`,
  `presentar libro`, `libro electrónico`, `sin información`, `LE`, `140400`, `080400`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- **The screen, the Excel and the TXT are the same rows.** What is filed is exactly what the
  table shows: consolidation, zeroing of voided or rejected comprobantes and grouping of shared
  purchase comprobantes happen once, before anything is displayed or exported.
- **A book is monthly.** The single-day view is only for checking; exports to TXT need a month.
- **Sales and purchases without a comprobante are never book rows.** They are shown apart, they
  are not counted in the totals, and they are not exported.
- **The exports use the last read.** Excel and TXT take the rows of the last **Consultar**, so
  after changing **Libro**, **Mes** or **Fecha**, press **Consultar** again before exporting.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **The table is empty after opening the page.** Nothing loads by itself; press **Consultar**.
- **TXT SUNAT is disabled.** Check the yellow message (notifications panel, tab **Mensajes**) or
  press **Consultar** again to receive it; fix each line and read the month again.
- **"N comprobantes siguen esperando la respuesta de SUNAT".** Open **Facturación**, find the
  comprobantes in *Pendiente de envío*, *Enviando* or *Error de envío* and send them with
  **Enviar ahora**; then press **Consultar** here.
- **"N filas no tienen serie".** The row's **Comprobante** and **Tipo** are empty because its
  series is missing from **Series de Facturación** in Mi Empresa → Configuración, tab
  **Facturación**.
- **"La empresa no tiene un RUC válido".** Fill in **RUC** (11 digits) in Mi Empresa →
  Configuración, tab **Mi Empresa**, and save.
- **"N comprobantes no están en PEN…".** Genix stores no exchange rate for sales comprobantes,
  so a month with a dollar comprobante cannot be exported to TXT from this page.
- **"N comprobantes no tienen la identidad del proveedor".** The purchase rows show empty
  **Proveedor** and **RUC**. The message asks to save the supplier.
  <!-- DOCUMENTATION_GAP: Confirm whether saving the supplier fixes purchases already registered
  without a supplier identity, or whether the purchase itself must be saved again (the identity
  is pinned on the purchase when it is saved). This comment is excluded from RAG. -->
- **A purchase is missing from the Registro de Compras.** Tick **Compras sin comprobante**: if
  it is there, register its comprobante on the purchase. If not, check the month of the
  comprobante's issue date.
- **The exported file names the wrong month.** The exports follow the current **Mes**
  selection; press **Consultar** after changing it so the rows and the period match.
- **The book could not be read.** The page shows *No se pudo leer el libro.* or the server's
  error; try **Consultar** again.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Facturación** (`/accounting/invoicing`) lists each sales comprobante with its SUNAT state and
  sends the pending ones with **Enviar ahora**. Use it when a row is **Pendiente SUNAT** or to
  see a single document's detail.
- **Mi Empresa → Configuración** (`/company/configuration`): tab **Mi Empresa** for **RUC** and
  **Razón Social** (fields 1 and 2 of the TXT), tab **Facturación** for **Series de
  Facturación**.
- **Órdenes Compra** (`/logistics/purchase-orders`), **Gastos** (`/finance/expenses`) and
  **Activos** (`/accounting/assets`) are where the supplier's comprobante of a purchase is
  registered; that is what puts it in the Registro de Compras.
- **Proveedores** (`/logistics/suppliers`) holds the supplier whose identity the purchase rows
  print.
- **Punto de Venta** (`/sales/sale_order_create`) is where a sale is created with or without a
  comprobante (**SIN COMPROBANTE**), which decides whether it appears in the Registro de Ventas
  or in **Sin Comprobante**.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.read-book, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.read-book, related-pages]
  - path: frontend/routes/accounting/books/+page.svelte
    role: page
    hash: sha256:25cb3fccc7b5614cf71c6f65348ef56e8424e69eccf1a308445059888a8aaa45
    supports: [page-purpose, capability.read-book, capability.sales-register, capability.uninvoiced-sales, capability.purchases-register, capability.undocumented-purchases, capability.export-excel, capability.export-sunat-txt, rules, troubleshooting]
  - path: frontend/routes/accounting/books/books.svelte.ts
    role: frontend-service
    hash: sha256:46dadddd03a8d6410327cfa1dfc73a12301c5a8e7e8ed1ea7c3d6d6801d1ea1f
    supports: [capability.read-book, capability.sales-register, troubleshooting]
  - path: frontend/routes/accounting/books/books.ts
    role: business-logic
    hash: sha256:7e26d19318088cd3acf2be59f90ba0bebfe851efa2279d6b149fdf403a8a6555
    supports: [concepts, capability.read-book, capability.sales-register, capability.uninvoiced-sales, rules]
  - path: frontend/routes/accounting/books/books.purchases.ts
    role: business-logic
    hash: sha256:24b3356c896e612766a1bf5b03e87d850515d79e6241d71c31fd027ba5f393ff
    supports: [concepts, capability.purchases-register, capability.undocumented-purchases, capability.export-sunat-txt, troubleshooting]
  - path: frontend/routes/accounting/books/books.txt.ts
    role: business-logic
    hash: sha256:312020a4d91602152f9c33f5ae7c0ad615cbb492f447d417bb6189c1162127c3
    supports: [capability.export-sunat-txt, troubleshooting, rules]
  - path: frontend/routes/accounting/books/books.purchases.txt.ts
    role: business-logic
    hash: sha256:495d1589487ed2f017f40aaae4e2f9822a1664d3d7a53e348a44df78434a7373
    supports: [capability.purchases-register, capability.export-sunat-txt]
  - path: frontend/routes/accounting/books/books.excel.ts
    role: frontend-service
    hash: sha256:ed394ef6cc0cfa0848cf8bc2fc2e52e3b511e9959974d79c06d02681edbd1a49
    supports: [capability.export-excel]
  - path: frontend/routes/accounting/books/books.utils.ts
    role: frontend-service
    hash: sha256:0ad7bb6c036d249bcc1ee3f10a32eacdee4fef3907abf61076461b65e41474a0
    supports: [capability.export-sunat-txt]
  - path: frontend/routes/accounting/books/RATIONALE.md
    role: reference-document
    hash: sha256:db6df336ca662a80f78bf7aa25dacf381f0a6aae4873706d796429ad4009861c
    supports: [capability.sales-register, capability.purchases-register, capability.export-sunat-txt, capability.read-book]
  - path: backend/accounting/sales_book_api.go
    role: backend-handler
    hash: sha256:fdbb182059ebafae2578a42af5e93bb40ceceeada7faacbde755f5f1ebcdea95
    supports: [capability.read-book, capability.sales-register, capability.uninvoiced-sales]
  - path: backend/accounting/purchases_book_api.go
    role: backend-handler
    hash: sha256:12920785965e9598eb546cafcd5dd9fe9c255c86005cc454d2bf51b7d99012ef
    supports: [capability.purchases-register, capability.undocumented-purchases]
  - path: backend/accounting/accounting_operations_api.go
    role: backend-handler
    hash: sha256:c8fe220fc549d253728fe36977a078132b61a95dcaa094f7116d4ce810907569
    supports: [capability.sales-register, capability.uninvoiced-sales, capability.purchases-register]
  - path: backend/invoicing/types/invoice_document.go
    role: data-model
    hash: sha256:a32b018a853cbd763186e4043cbfc3bcb350c2550c77f7b73f01507bd9b73a66
    supports: [concepts, capability.sales-register]
  - path: backend/sales/types/sale_order_id.go
    role: data-model
    hash: sha256:67d8313090863dda02ea1d57b3b363ebdd55e233deb6d0019ff0e17539e622b7
    supports: [capability.uninvoiced-sales]
  - path: backend/sales/saleissue/types/sale_issue_series.go
    role: business-logic
    hash: sha256:6c210d7117e055a98c571f9bd17d0cd04f55d9458365735de9b8970e7550255d
    supports: [capability.sales-register]
  - path: backend/finance/types/purchase_document.go
    role: business-logic
    hash: sha256:ec949ba6c381199a92b8310f78d4b02a3ba79ea02f84653c4a4020a52cb0be3a
    supports: [concepts, capability.purchases-register, capability.undocumented-purchases, capability.export-sunat-txt]
  - path: backend/finance/types/expenses.go
    role: data-model
    hash: sha256:6bb15089e1434be15645e06909f8f7ef73172b4d46ba58c58786ad11caba7c72
    supports: [concepts, capability.purchases-register]
  - path: backend/logistics/types/purchase_order.go
    role: data-model
    hash: sha256:c13fc922ef507f06ccba0b7dbc257311d6edf9cc8162b13ecd56221e14a7c23a
    supports: [concepts, capability.purchases-register, capability.undocumented-purchases]
  - path: frontend/core/sunat-doc-type.ts
    role: shared-domain
    hash: sha256:fe287043f4274bc1dac8f6e5da48e4c7456e73f8fc0189ed4bd812923a786a1d
    supports: [capability.sales-register]
  - path: frontend/core/purchase-document.ts
    role: shared-domain
    hash: sha256:6a9178bf99be21baf75b2c106b241fb24e410bfca9e3c42ad5fab840e7480cb0
    supports: [capability.purchases-register]
  - path: frontend/routes/accounting/invoicing/invoicing.ts
    role: shared-domain
    hash: sha256:e2096d8ad3016afcf31748fd12d3debcda9a26c38a0b85b58831342524954e5a
    supports: [concepts, troubleshooting]
  - path: frontend/core/notifications.svelte.ts
    role: user-interface
    hash: sha256:2fba79fb04e8033b3df76475066a5cb2810332d5d5d2331edba696f6c5dd8918
    supports: [capability.read-book, capability.export-sunat-txt, troubleshooting]
  - path: frontend/domain-components/NotificationsButton.svelte
    role: user-interface
    hash: sha256:25129b3a97ebbb5e68871bf87a147f54e0cc172cee380e8923fd7c693b439455
    supports: [capability.read-book, troubleshooting]
  - path: frontend/libs/helpers.ts
    role: frontend-service
    hash: sha256:d286f290f196366c67fd50a45c109cf441eceea3666c0922d7a548cf9a5b461c
    supports: [capability.export-sunat-txt]
  - path: frontend/domain-components/PurchaseDocumentFields.svelte
    role: user-interface
    hash: sha256:fc3a46a9311ac0be2c7f42996646c958c521df7e7807065590b0a6dda9fdf75e
    supports: [capability.purchases-register, capability.undocumented-purchases]
  - path: frontend/routes/logistics/purchase-orders/PurchaseOrderReport.svelte
    role: user-interface
    hash: sha256:668c15b4ead42dca23f11f785f8f12af0e029ddef87295904109f7796c463180
    supports: [capability.undocumented-purchases, related-pages]
  - path: frontend/routes/finance/expenses/ExpensesRegister.svelte
    role: user-interface
    hash: sha256:d321929caabc8bafafe601a9692222322eba6a62c78f4620f063f5717e121f10
    supports: [capability.purchases-register, related-pages]
  - path: frontend/routes/accounting/assets/AssetForm.svelte
    role: user-interface
    hash: sha256:3c75246ce1225a3caf3398314964c0fa9b9e208a43da61a8d22989b16781baea
    supports: [capability.purchases-register, related-pages]
  - path: frontend/routes/accounting/invoicing/+page.svelte
    role: user-interface
    hash: sha256:a0c983ada56235534fe2aa79a7536d63296191a73e1adf352be070c39d99a82a
    supports: [troubleshooting, related-pages]
  - path: frontend/routes/sales/sale_order_create/+page.svelte
    role: user-interface
    hash: sha256:d28216a645b9623316a8d8e2fdd6c0b7a2e8d2fc6228b19cf589b8d586edd4f7
    supports: [capability.uninvoiced-sales, related-pages]
  - path: frontend/routes/logistics/suppliers/+page.svelte
    role: user-interface
    hash: sha256:4ea248941db02acab88071cd6510da75d4496aa1b4630c8eec158e93b6819850
    supports: [related-pages]
  - path: frontend/routes/company/configuration/+page.svelte
    role: user-interface
    hash: sha256:46cd5a24d7dac4d50583401399892a830ceb8085ade7e3ecae766df0c0820e42
    supports: [capability.export-sunat-txt, troubleshooting, related-pages]
  - path: frontend/routes/company/configuration/CompanyTab.svelte
    role: user-interface
    hash: sha256:ac0fe4c47cbb0b09d800c1e9c49639e66e32d2752b4fcc95246ab71d8503539e
    supports: [capability.export-sunat-txt, troubleshooting, related-pages]
  - path: frontend/routes/company/configuration/InvoiceSeriesTable.svelte
    role: user-interface
    hash: sha256:e31ade2c20c6feab4bdd59ae6a89acc3abb81f89dc58932ebeee85b85ce0ee08
    supports: [capability.sales-register, troubleshooting, related-pages]
```
