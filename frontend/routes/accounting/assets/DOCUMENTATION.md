---
schema: 1
page_id: accounting.assets
route: /accounting/assets
title: Assets (Activos)
status: implemented
visibility: tenant
description_en: >-
  Fixed-asset register. Acquire an asset from a depreciable supply/material, track its book value
  and straight-line depreciation, correct its acquisition data, pay what is owed for it from a
  cash or bank account, and dispose of it.
description_es: >-
  Registro de activos fijos. Adquirir un activo a partir de un insumo/material depreciable,
  controlar su valor en libros y su depreciación lineal, corregir sus datos de adquisición, pagar
  lo que se adeuda por él desde una caja o banco, y darlo de baja.
---

# Assets (Activos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Assets (`Activos`, or `activos fijos`) is the register of the things the business **owns and uses**
rather than sells: computers (`equipos de cómputo`), vehicles (`vehículos`), machinery
(`maquinaria`), furniture (`muebles`), buildings (`edificios`). For each one the page holds what it
was worth when acquired, what it is worth now after depreciation (`valor en libros`), what is still
owed for it (`adeudado`), and whether it is still in service.

The page owns four operations: acquire (`Adquirir`), edit the acquisition data (`Editar`), register
a payment (`Registrar Pago`), and dispose (`Dar de Baja`).

**An asset is not an expense (`un activo no es un gasto`).** Buying one is cash turning into
something the business owns, not a cost, so acquiring an asset writes **nothing** in Expenses
(`Gastos`) and the money owed for it is settled here, not in the expense register. The only cost an
asset ever produces is its monthly depreciation.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Acquisition value (Valor de Adquisición):** what the asset was worth when it entered service.
  This is the amount that depreciates.
- **Book value (Valor en Libros):** the acquisition value minus everything depreciated so far. It
  never goes below zero.
- **Purchase amount (Monto de Compra):** the cash owed for the asset. It is a **different number**
  from the acquisition value. A donated or contributed asset (`activo donado o aportado`) has a
  purchase amount of 0 and still has a book value and still depreciates.
- **Owed (Adeudado):** purchase amount minus what has been paid. It never shows as negative.
- **Depreciation (Depreciación):** the monthly loss of value, straight-line
  (`depreciación lineal`) over the asset's term. **It starts the month *after* acquisition** — an
  asset bought on the 28th earns no depreciation for those three days.
- **Depreciation ledger (Asientos de Depreciación):** the list of monthly entries already posted
  for one asset, shown in its detail panel as `Depreciación n/N`.
- **Payment states:** **Donated (Donado)** — nothing was ever owed; **Pending (Pendiente)** — owed
  in full or in part; **Paid (Pagado)** — settled.
- **Lifecycle states:** **Active (Activo)** — in service and still depreciating;
  **Fully depreciated (Totalmente depreciado)** — book value reached zero but the business still
  owns it and it is still in the warehouse; **Disposed (Dado de baja)** — retired, out of the
  warehouse, no longer depreciating, but still on the register with its history.

<!-- DOC-ID: capability.acquire -->
## Acquire an asset (Adquirir un activo)

### User intention (Intención del usuario)

Put a newly bought, donated, or contributed asset on the register so it starts depreciating and so
what is owed for it is tracked.

### Where to find it (Dónde encontrarlo)

Open **Accounting (Contabilidad) → Assets (Activos)** at `/accounting/assets` and press
**Acquire (Adquirir)**. The form opens as a side panel titled **Nuevo Activo**.

### Required information and prerequisites (Requisitos previos)

**The asset must already exist as a supply/material with a depreciation term.** The
**Material** selector only lists supplies/materials (`insumos y materiales`) whose depreciation is
set to something other than `No depreciable`. If the dropdown is empty, nothing has been set up as
depreciable yet; the form links to **Production (Producción) → Supplies & Materials
(Insumos & Materiales)** with *"Regístrelo aqui"*, where the **Depreciation (Depreciación)** field
offers 4 years for computers, 5 for vehicles, 10 for machinery, 20 for furniture, and 33 for
buildings. The term is copied onto the asset at acquisition, so changing the material's
depreciation later never rewrites the schedule of an asset already in service.

Also required: a **Warehouse (Almacén)**, and a **Book Value (Valor en Libros)** greater than 0.
The currency must be `PEN` or `USD`. Supplier (`Proveedor`), acquisition date, and payment due date
are optional — an empty acquisition date means today, and an empty due date follows the acquisition
date.

### Business rules and rationale (Reglas y razón de negocio)

On this form the money is **per unit** (`por unidad`) and Genix multiplies it by the quantity.

**Serial numbers decide the granularity.** The **Serial Numbers (Números de Serie)** box takes one
serial per line. Entering serials produces **one independent asset per serial**; leaving it empty
produces **one grouped asset** carrying the whole lot in its quantity. The presence of serials *is*
the answer to "do you track these individually?", which is why the form does not ask separately.
Buying five identical laptops is legitimately either five assets or one, depending on whether the
business tags them.

Leaving **Purchase Amount (Monto de Compra)** at 0 is the donated case: nothing is owed, the asset
is never payable, but it is still on the books and still depreciates.

Genix rejects a missing material, a material that is not a supply/material, a material without a
depreciation term, a missing warehouse, a quantity of zero with no serials, a book value of zero or
less, a negative purchase amount, and an invalid currency.

### Result and side effects (Resultado y efectos)

Genix creates the asset row (or one per serial) in **Active (Activo)** status and **moves the units
into the selected warehouse** through the ordinary stock movement engine — an asset's existence
*is* its stock, so it appears in warehouse quantities like anything else, and a serial-tracked unit
lands on its own serial-scoped stock row.

It does **not** create an expense, does **not** move cash, and does **not** post any depreciation
yet. A purchased asset starts as **Pending (Pendiente)**; a donated one as **Donated (Donado)**.

### Limitations (Limitaciones)

- The form has no name or description field; the asset takes the material's name.
- The depreciation term cannot be overridden from this page — it comes from the material.
- Acquiring an asset does not register the payment. Use **Registrar Pago** afterwards.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro una computadora o un vehículo como activo fijo?`
- `El selector de Material está vacío, ¿por qué?` Nothing has a depreciation term yet.
- `¿Cómo registro un activo donado o aportado por el dueño?` Leave `Monto de Compra` at 0.
- `¿Cómo registro 5 laptops iguales?` One serial per line for five separate assets; empty for one
  grouped lot of 5.
- Search terms: `activo fijo`, `adquirir activo`, `número de serie`, `valor en libros`,
  `depreciación`, `almacén`, `activo donado`.

<!-- DOC-ID: capability.review -->
## Review an asset and its depreciation (Consultar un activo y su depreciación)

Select any row in the register to open its detail panel. It shows the acquisition value, the
current book value, the percentage depreciated, the months remaining (`Meses Restantes`), and — for
a purchased asset — the **Compra / Pagado / Adeudado** boxes. A donated asset shows *"Donado o
aportado: no se adeuda nada por este activo"* instead of those boxes.

Below that, **Asientos de Depreciación** lists every monthly entry already posted, each labelled
`Depreciación n/N` with its date and amount. If it says *"Aún no hay asientos registrados"*, no
month has come due yet — remember depreciation starts the month **after** acquisition.

**Depreciation is generated when this page loads.** Genix has no background job for it: opening
Assets posts every monthly entry that has come due for every active asset, and reports how many
were created. This is safe to repeat — visiting the page twice in one day posts nothing the second
time. It also means book values in the register are only as current as the last visit to this page.

**Depreciation entries are the one cost an asset produces, and they are visible only here.** They
are non-cash postings, so they never appear in the Expenses (`Gastos`) register list and never show
up as something to pay. They are filed under the `Maintenance (Mantenimiento)` expense category.

<!-- DOC-ID: capability.edit -->
## Edit an asset's acquisition data (Editar un activo)

### User intention (Intención del usuario)

Correct a mistake made when the asset was registered: a mistyped amount, the wrong acquisition
date, a missing or wrong serial number. This is a **correction (`corrección de datos`), not a
revaluation (`revaluación`)** — the register ends up looking as if the asset had been entered
correctly the first time.

### Where to find it (Dónde encontrarlo)

Select the asset in the register, then press **Edit (Editar)** in its detail panel, next to
**Registrar Pago** and **Dar de Baja**. The form opens as a **dialog (modal)** on top of the panel,
not as a side panel like acquisition, so the balances and the depreciation ledger the edit is about
stay visible behind it.

### Required information and prerequisites (Requisitos previos)

The asset must be **Active (Activo)** or **Fully depreciated (Totalmente depreciado)**. The
**Editar** button does not appear on a **Disposed (Dado de baja)** asset.

### Business rules and rationale (Reglas y razón de negocio)

**Only five fields can be changed:** Purchase Amount (`Monto de Compra`), Book Value
(`Valor en Libros`), Acquisition Date (`Fecha de Adquisición`), Payment Due (`Vencimiento del
Pago`), and Serial Number (`Número de Serie`).

**Everything else appears greyed out and locked:** Material, Warehouse (`Almacén`), Supplier
(`Proveedor`), Currency (`Moneda`), and Quantity (`Cantidad`). They are the asset's identity — the
material is its catalog row, the currency is what payments are matched against, the quantity is how
many units the stock movement carried. They are shown locked rather than hidden so the user can
still see they are looking at the right asset.

**In this dialog the money is the asset's total, not per unit.** The `(por unidad)` suffix that the
acquisition form carries is gone. For a grouped asset of five, the value shown is the value of all
five, which is also what the register and the detail panel display. Dividing a stored total by the
quantity would not round-trip — a lot of three at 1,000 has no exact per-unit figure.

Three consequences are the reason this operation is not a simple field update:

**1. Changing the book value or the acquisition date rewrites the depreciation already posted.**
The monthly schedule is calculated from the acquisition value, the acquisition date, and the term,
so changing either of the first two makes every entry already posted describe a schedule that no
longer exists. Genix therefore rewrites the ledger period by period: existing entries are updated
in place with their new dates and amounts, months that now exist but were never posted are added,
and months that no longer happen — because the acquisition date moved forward — are removed from
the ledger. The accumulated depreciation, the book value, and the months remaining are all
recalculated from the new numbers. The dialog states this before saving. **This changes the
depreciation cost recorded in past months, including months already closed.** If the acquisition
date moves into the current month, the ledger empties completely and the asset starts depreciating
from zero again.

**2. The purchase amount cannot go below what has already been paid.** Genix rejects it and names
the amount already paid: *"El monto de compra no puede ser menor a lo ya pagado (X). Revierta el
pago antes de reducirlo."* Reducing it would leave an over-paid asset with no record of why, and
reversing cash is a cash operation that an asset edit does not perform implicitly. Reverse the
payment first. Raising the amount, or lowering it to anything at or above what was paid, is
accepted, and the payment state is recalculated: setting it to 0 makes the asset **Donated
(Donado)** — possible only when nothing has been paid — and paying the new amount in full makes it
**Paid (Pagado)**.

**3. Changing the serial number moves the units in the warehouse.** A serial is part of the identity
of the stock row it points at, so it cannot simply be relabelled. Genix takes the units out of the
old serial and puts them into the new one through the same movement engine acquisition and disposal
use, which leaves two entries in the movement ledger (`kardex`). Clearing a serial, or adding one to
a grouped asset, works the same way. A serial another live asset of the same material already
carries is rejected: *"El número de serie ya está registrado en otro activo."*

Genix also rejects a book value of zero or less, a negative purchase amount, a missing acquisition
date, and a payment due date earlier than the acquisition date.

### Result and side effects (Resultado y efectos)

The asset row is updated, the depreciation ledger is rewritten when the value or date changed, the
stock moves when the serial changed, and the payment state is recalculated. The detail panel and
its ledger refresh immediately so the numbers on screen match what was saved.

A value correction can move an asset **back from Fully depreciated to Active** — raising the book
value of an asset that had reached zero gives it something left to depreciate.

The lifecycle status is never edited directly; it is always derived from the asset's own numbers.

### Limitations (Limitaciones)

- **A disposed asset cannot be edited at all.** Its units are already out of the warehouse, so
  there is nothing for a serial change to move.
- The material, warehouse, supplier, currency, and quantity cannot be changed here. To move an
  asset to a different warehouse there is a separate transfer operation, not exposed as a button on
  this page. To fix the wrong material, dispose of the asset and acquire the correct one.
- The depreciation term cannot be changed, so an asset entered against the wrong material keeps
  that material's term.
- Editing does not create an adjusting entry (`asiento de ajuste`) documenting the correction. The
  old amounts are overwritten, not preserved alongside the new ones. There is no undo — to get back
  to the previous figures, edit again with the old values.
- The value recorded on the original stock movement is not corrected, so a stock valuation report
  reading movement prices may still show the old figure. The book value and the depreciation
  entries — what the balance sheet reads — are correct.

### Common questions and vocabulary (Preguntas y vocabulario)

- `Me equivoqué en el monto del activo, ¿cómo lo corrijo?`
- `¿Qué pasa con las depreciaciones ya registradas si cambio el valor en libros?` They are
  rewritten with the new amounts, including past months.
- `¿Puedo cambiar la fecha de adquisición de un activo?` Yes, and it recalculates the whole
  schedule.
- `¿Por qué no puedo bajar el monto de compra?` Because a payment already covers more than that.
- `¿Puedo agregar el número de serie después?` Yes; it moves the units to the serial in the
  warehouse.
- `¿Por qué está bloqueado el almacén / el material / la moneda?`
- `No veo el botón Editar.` The asset is disposed (`dado de baja`).
- Search terms: `editar activo`, `corregir activo`, `modificar valor en libros`,
  `cambiar número de serie`, `rehacer depreciación`, `depreciación retroactiva`.

<!-- DOC-ID: capability.pay -->
## Register a payment for an asset (Registrar el pago de un activo)

### User intention (Intención del usuario)

Pay a supplier for an asset, in full or in part (`abono parcial`), and have the money leave the
cash register or bank account it actually came from.

### Where to find it (Dónde encontrarlo)

Select the asset and press **Register Payment (Registrar Pago)**. The dialog opens **pre-filled
with the outstanding balance**, which is the payment made in almost every case.

### Required information and prerequisites (Requisitos previos)

The asset must have a purchase amount above 0 and something still owed — the button does not appear
on a donated or fully settled asset. A cash or bank account (`caja o banco`) must exist, and its
currency must match the asset's.

### Business rules and rationale (Reglas y razón de negocio)

The payment must be greater than 0 and **no greater than what is still owed**, which also blocks
any payment on an already settled asset. The account's currency must equal the asset's currency,
and **the account balance may not go negative** — unlike some other payment paths in Genix, this one
enforces that. The paid total is recalculated from the cash movements each time rather than added
up, so a retry or a simultaneous payment cannot drift the figure.

**Is Fully Paid (Pagado Completo)** forces the asset to **Paid (Pagado)** even when the cash does
not cover the balance. This is the write-off case (`castigo` / condonación de saldo), the one
situation the amounts alone cannot decide.

### Result and side effects (Resultado y efectos)

The selected account gets an outgoing movement linked to the asset and its balance falls by the
payment amount. The asset's paid and owed figures update, and its payment state becomes **Paid
(Pagado)** once covered.

The payment writes **no expense row** — an asset is not a cost — so it does not appear in Expenses
(`Gastos`).

### Limitations (Limitaciones)

- A payment cannot be reversed or deleted from this page.
- The payment is not a single atomic step: if it fails after the cash has moved, the money is out of
  the account while the asset still shows the old balance. Check the account movements before
  retrying.
- Paying an asset does not change its lifecycle status or its depreciation. Payment and depreciation
  are independent — an asset can be fully paid and barely depreciated, or fully depreciated and
  still unpaid.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo pago un activo desde una caja o banco?`
- `¿Puedo hacer un abono parcial de un activo?`
- `La caja no aparece en la lista.` Its currency does not match the asset's.
- `¿Qué significa Pagado Completo?` A write-off: it settles the balance without the cash.
- Search terms: `pagar activo`, `abono`, `caja`, `banco`, `adeudado`, `pagado completo`.

<!-- DOC-ID: capability.dispose -->
## Dispose of an asset (Dar de baja un activo)

### User intention (Intención del usuario)

Retire an asset the business no longer holds — sold, scrapped, lost, or written off (`baja de
activo`).

### Where to find it (Dónde encontrarlo)

Select the asset and press **Dispose (Dar de Baja)**, then confirm. The confirmation names the asset
and warns that it leaves the warehouse and stops depreciating.

### Required information and prerequisites (Requisitos previos)

The asset must be **Active (Activo)** or **Fully depreciated (Totalmente depreciado)** — a fully
depreciated laptop is still a laptop the business owns, so it remains a candidate. An asset can only
be disposed once.

### Business rules and rationale (Reglas y razón de negocio)

The disposal date cannot be earlier than the acquisition date. Disposal **cuts the depreciation
schedule off**: a month is only posted if the asset was still held when that month began.

### Result and side effects (Resultado y efectos)

The units leave the warehouse through the ordinary movement engine, the asset moves to **Disposed
(Dado de baja)**, and it **stays on the register** with its full history — this is exactly why the
accounting record is kept separately from the stock record, so retiring an asset does not erase the
history the balance sheet still needs.

### Limitations (Limitaciones)

- **Disposal cannot be undone from this page.** After it, the asset can no longer be edited or
  paid.
- Disposal does not cancel or refund what is still owed, and it does not reverse payments already
  registered. Settle or check the balance before disposing.
- Disposal records no sale price or gain/loss on disposal (`ganancia o pérdida en venta de activo`).
  If the asset was sold, record the income separately.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo doy de baja un activo que vendí o se dañó?`
- `¿Puedo revertir una baja?` No, not from this page.
- `¿Un activo totalmente depreciado se da de baja automáticamente?` No; it stays Active until
  disposed.
- Search terms: `dar de baja`, `baja de activo`, `retirar activo`, `activo dado de baja`.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- **Purchase and value are independent.** What was paid never affects the book value, and
  depreciation never affects what is owed.
- **Status is always derived, never chosen.** An asset is Disposed if it has a disposal date, Fully
  depreciated if its accumulated depreciation reached its acquisition value, otherwise Active. No
  operation on this page sets it directly.
- **The register keeps everything.** Active, fully depreciated, and disposed assets all stay in the
  list; the filter box searches by asset name and serial number.
- **Stock and accounting are two records of one asset.** The warehouse holds the units; this page
  holds the money, dates, and lifecycle. Any operation that changes where units are — acquisition,
  a serial change, disposal — moves them through the same engine every other warehouse movement
  uses, so warehouse quantities always agree with the register.
- **Only depreciation reaches Expenses.** Acquiring, editing, and paying an asset write no expense
  row at all.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **The Material dropdown is empty.** No supply/material has a depreciation term. Set one in
  Production (Producción) → Supplies & Materials (Insumos & Materiales).
- **"El insumo no tiene un esquema de depreciación configurado."** The chosen material's
  depreciation is `No depreciable`.
- **The depreciation ledger says "Aún no hay asientos registrados".** Depreciation starts the month
  *after* acquisition, so a recently acquired asset has nothing posted yet.
- **Book values look a month behind.** Depreciation is posted when this page loads. Open Assets to
  bring the register up to date. If posting fails, the page stays readable with slightly stale
  numbers and says nothing — the values are not wrong, just not yet updated.
- **"El monto de compra no puede ser menor a lo ya pagado."** Reverse the payment before lowering
  the purchase amount.
- **"El número de serie ya está registrado en otro activo."** Another live asset of the same
  material holds that serial.
- **"La moneda de la caja no coincide con la del activo."** Pay from an account in the asset's
  currency.
- **"El saldo de la caja no puede quedar negativo."** The account does not hold enough for the
  payment.
- **The Editar button is missing.** The asset is disposed.
- **Depreciation amounts changed for months already closed.** Someone edited the asset's book value
  or acquisition date; that rewrites the whole posted ledger by design.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Production (Producción) → Supplies & Materials (Insumos & Materiales)** — where an asset must
  first exist as a material with a depreciation term. The list marks each one `Asset (Activo)` or
  `Consumable (Consumible)` depending on that term.
- **Finance (Finanzas) → Cash & Banks (Cajas & Bancos)** — the accounts an asset payment leaves
  from. **Cash Movements (Cajas Movimientos)** shows the resulting outgoing movement.
- **Finance (Finanzas) → Expenses (Gastos)** — for costs that are *not* assets. An asset purchase
  does not belong here. Depreciation entries exist in this table but are deliberately kept out of
  its list, so the asset detail panel is the only place to see them.
- **Logistics (Logística) → Movements Report (Rep. Movimientos)** — the movement ledger
  (`kardex`) where asset acquisitions, serial changes, and disposals appear as stock movements.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:f60ca2c4d17f36984b5994230c5df7fbe056e5646d2f1817707ce6827e4cab25
    supports: [page-purpose, capability.acquire, related-pages]
  - path: frontend/routes/accounting/assets/+page.svelte
    role: page
    hash: sha256:1f77cc2f37360353971654b28a7204e24a8db8f36681c51ac2ce209e86953b14
    supports: [page-purpose, capability.acquire, capability.review, capability.edit, capability.pay, capability.dispose, rules, troubleshooting]
  - path: frontend/routes/accounting/assets/AssetForm.svelte
    role: user-interface
    hash: sha256:b67edea7dc90f2c4d226dc33ce35e4f7d9344d9c7afe83107da147e28764ee31
    supports: [capability.acquire, capability.edit, troubleshooting]
  - path: frontend/routes/accounting/assets/assets.svelte.ts
    role: frontend-service
    hash: sha256:a881ca046ba5c4cbbcd97e41dab35b3c039aaa87045ef09b6517d60afa62fa41
    supports: [capability.acquire, capability.edit, capability.pay, capability.dispose, rules]
  - path: frontend/routes/accounting/assets/assets.ts
    role: business-logic
    hash: sha256:0991c8fb4bc86cbe6ad9f4160644f02fa62235b9cccc43b5e674387f12856a32
    supports: [concepts, capability.review, capability.edit, capability.pay, capability.dispose, rules]
  - path: backend/accounting/asset_api.go
    role: backend-handler
    hash: sha256:4a8e102832b2c17ec02d76c5c726818782d5fb99295be33ad7d45875fa4ca2c0
    supports: [capability.acquire, capability.dispose, rules, troubleshooting]
  - path: backend/accounting/asset_edit.go
    role: backend-handler
    hash: sha256:04c5f07ed07681cbe21e9894536eec31b568ea8d478c59a647de1d73b4c6aafa
    supports: [capability.edit, rules, troubleshooting]
  - path: backend/accounting/asset_payment.go
    role: backend-handler
    hash: sha256:a554b9a3401289ff0bb4aff947d3e79e6e7a9990adcfb1bc8c53e71c433a84ac
    supports: [capability.pay, rules, troubleshooting]
  - path: backend/accounting/depreciation.go
    role: business-logic
    hash: sha256:f993fedaccd1c101ae5b973cb720b158ffebbc284bf907742cbb39bb2b26533b
    supports: [concepts, capability.review, capability.edit, capability.dispose, rules]
  - path: backend/accounting/depreciation_run.go
    role: business-logic
    hash: sha256:aff1a1767d105bc28ea00a90f562bbca966aa97347921985e66a67b07b8a5007
    supports: [capability.review, capability.edit, troubleshooting]
  - path: backend/accounting/main.go
    role: business-logic
    hash: sha256:671571ebfebad1f8b63217dfaf0c8c5fcb46b48ca6c921dfd3bbb5eb1ab44b30
    supports: [capability.review]
  - path: backend/accounting/types/asset.go
    role: data-model
    hash: sha256:94163e84d5adb3731609275575add2251b1a04d41d3511245e1dc73e33d3a34c
    supports: [concepts, capability.acquire, capability.edit, capability.pay, capability.dispose, rules]
  - path: backend/finance/types/expenses.go
    role: data-model
    hash: sha256:a8024fe7354fe2ec8cccc613e9dd11d00ee652472784067f768763b11f02bb04
    supports: [capability.review, capability.edit, rules, related-pages]
  - path: backend/finance/types/cash_banks.go
    role: data-model
    hash: sha256:7811811ae13762b2c52610de373bfbee46e821adb5b87fda3fce797497db4759
    supports: [capability.pay]
  - path: backend/logistics/types/product-stock.go
    role: data-model
    hash: sha256:3850fdfcf3eb9ecd628c56d3a04730d4dc8d038915fe7410d2b076572d6a7667
    supports: [capability.acquire, capability.edit, rules]
  - path: frontend/routes/production/supplies-materials/supply-material.svelte.ts
    role: shared-domain
    hash: sha256:f93d9770f34a17342cd2624ae22660a3ecf09ca2351e3ef7f2f2a52ba7c5c47b
    supports: [capability.acquire, related-pages]
```
