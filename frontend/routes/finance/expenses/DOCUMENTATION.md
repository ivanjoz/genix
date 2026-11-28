---
schema: 1
page_id: finance.expenses
route: /finance/expenses
title: Expenses (Gastos)
status: implemented
visibility: tenant
description_en: >-
  Register the company's expenses and pay them from a cash or bank account. A simple expense goes
  straight to costs; an inventory expense buys a supply and enters it in a warehouse. Each expense
  can carry the supplier's comprobante (factura, boleta, recibo por honorarios, ticket, recibo de
  servicios) and its accounting account. Pay fully or in part. Recurring expenses (rent, services)
  are scheduled with a cadence and generate one expense per period.
description_es: >-
  Registro de los gastos de la empresa y su pago desde una caja o banco. Un gasto simple va directo
  al costo; un gasto de inventario compra un insumo y lo ingresa a un almacén. Cada gasto puede
  llevar el comprobante del proveedor (factura, boleta, recibo por honorarios, ticket, recibo de
  servicios) y su cuenta contable. Pago total o parcial. Los gastos recurrentes (alquiler,
  servicios) se programan con una cadencia y generan un gasto por periodo.
---

# Expenses (Gastos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Expenses (`Gastos`) is where the company records money it owes or spent outside a purchase order
(`orden de compra`) or a sale: rent (`alquiler`), utilities (`luz, agua, internet`), professional
fees (`honorarios`), transport, marketing, small supply purchases (`compra de insumos`), and so on.
Each expense keeps how much is owed, how much was paid (`pagado`) and from which cash or bank
account (`caja o banco`) each payment left.

The page has two views: **Register (Registro)**, the list of individual expenses and their
payments, and **Scheduled (Programados)**, the recurring expenses (`gastos recurrentes` or `gastos
fijos`) that generate one expense per period.

The page does not buy fixed assets (`activos fijos`): an asset is not an expense and is registered
in **Assets (Activos)**, which only adds its monthly depreciation (`depreciación`) to the expense
table; depreciation never appears in these lists. It does not buy sellable merchandise with lots
and reception differences; that is a **Purchase Order (Orden de Compra)**. Salaries are not entered
here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- An **expense** (`gasto`) is one amount owed: a one-time bill, or one period of a scheduled
  expense. It has a name, a category, a currency (PEN or USD), an amount, a date (`Fecha`) and a
  due date (`Vencimiento`).
- **Type (Tipo)** says what the money became:
  - **Simple:** a cost that goes straight to the profit and loss (`gasto del periodo`).
  - **Inventory (Inventario):** a supply or material (`insumo o material`) that enters a warehouse
    as stock; it becomes a cost only when it is consumed or sold.
- **Category (Categoría)** is a fixed list: Alquiler, Servicios, Seguros del personal, Insumos,
  Impuestos, Mantenimiento, Transporte, Marketing, Servicios profesionales, Otros, Compras de
  mercadería. Picking a category proposes its accounting account; Impuestos and Otros propose none.
- **Accounting account (Cuenta contable)** is the PCGE account (`plan contable`) the expense is
  classified under. A simple expense only accepts expense accounts; an inventory expense only
  accepts supply accounts. It may stay empty (unclassified).
- **Comprobante:** the document the supplier issued for the expense (factura, boleta, recibo por
  honorarios, ticket or recibo de servicios públicos). It is what makes the expense a row of the
  purchase register (`Registro de Compras`). An expense can also be saved **Sin comprobante**.
- **Payment status (Estado):** **Unpaid (Sin pagar)**, **Partial (Parcial)** when part was paid,
  and **Paid (Pagado)**.
- A **scheduled expense** (`programación`, `gasto programado`) is a template with a cadence
  (`Cadencia`). It is not paid itself: each of its periods (`Periodos`) is a separate expense
  that is paid on its own.

<!-- DOC-ID: capability.register-simple-expense -->
## Register a simple expense (Registrar un gasto simple)

### User intention (Intención del usuario)

Record a bill or cost the company must pay or already paid (`registrar un gasto`, `anotar una
factura de luz`, `cargar un recibo`), so it shows as pending until it is paid.

### Where to find it (Dónde encontrarlo)

**Finance & Accounting (Finanzas y Contabilidad) → Expenses (Gastos) → Expenses (Gastos)** at
`/finance/expenses`, view **Register (Registro)**. Press **New (Nuevo)**; the **New Expense
(Nuevo Gasto)** panel opens on the **Simple** tab. Save with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)** and **Category (Categoría)**; the category defaults to Otros.
- **Accounting account (Cuenta contable):** proposed from the category, can be changed or left
  empty.
- **Description (Descripción):** optional.
- **Currency (Moneda):** PEN or USD; defaults to PEN.
- **Amount (Monto):** must be greater than 0.
- **Date (Fecha):** defaults to today. **Due Date (Vencimiento):** optional.
- **Supplier (Proveedor):** optional, chosen from the registered suppliers.
- **Site (empty = whole company) (Sede (vacío = toda la empresa)):** the site the expense belongs
  to.
- **Production order (subcontracting) (Orden de producción (maquila)):** only shown when the chosen
  supplier has subcontracted production orders; links a workshop's invoice to the order it is for.
- The **Comprobante** block (see the comprobante section below).

### Business rules and rationale (Reglas y razón de negocio)

- When **Vencimiento** is left empty, Genix sets it to the expense date plus the supplier's
  **Credit days (Días de crédito)**; without a supplier, the due date is the expense date.
- A production order can only be linked to a simple expense, and only to a subcontracted order
  (`maquila`) of that same supplier that is not annulled.
- A simple expense only accepts an expense account from the chart of accounts.

### Result and side effects (Resultado y efectos)

The expense is created **Unpaid (Sin pagar)** and appears in **Todos** and **Pend. Pago**. No cash
moves until a payment is registered. The cash flow forecast reads the pending amount and the due
date.

### Limitations (Limitaciones)

- There is no delete or annul action for an expense on this page.
- An asset purchase or a depreciation entry cannot be registered here: only Simple and Inventario.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro un gasto? ¿Dónde anoto el recibo de luz o el alquiler del local?
- ¿Por qué el vencimiento salió con otra fecha? → it took the supplier's días de crédito.
- ¿Cómo vinculo la factura del taller a la orden de maquila?
- Search terms: `gasto`, `egreso`, `gasto operativo`, `cuenta por pagar`, `recibo`, `factura de
  compra`, `honorarios`, `alquiler`, `servicios`.

<!-- DOC-ID: capability.register-inventory-expense -->
## Buy a supply into a warehouse (Registrar un gasto de inventario)

### User intention (Intención del usuario)

Record the purchase of a supply or material (`insumo`, `material`) and enter the units in a
warehouse in one step, so the money becomes stock instead of a cost.

### Where to find it (Dónde encontrarlo)

**Register (Registro)** → **New (Nuevo)** → tab **Inventory (Inventario)**. This tab exists only for
a new expense.

### Required information and prerequisites (Requisitos previos)

- **Supply / Material (Insumo o Material):** an existing supply, or a new name typed in **…or create
  a new supply (…o crear un insumo nuevo)**, which Genix creates first.
- **Warehouse (Almacén):** where the units enter.
- **Quantity (Cantidad)** and **Unit Price (Precio Unitario)**, both greater than 0, and **Currency
  (Moneda)**. The **Total** is quantity × unit price and becomes the expense amount.
- **Accounting account (Cuenta contable):** proposed from the supply's own account; left empty,
  Genix uses the supply's account.
- Optional: **Name (Nombre)** (defaults to the supply's name), **Category (Categoría)** (defaults
  to Insumos), **Date (Fecha)**, **Due Date (Vencimiento)**, **Supplier (Proveedor)** and the
  comprobante.

### Business rules and rationale (Reglas y razón de negocio)

- The item must be a supply (`insumo`), not a sellable product.
- A supply that tracks lots (`controla lotes`) is refused: it must be bought through a purchase
  order and received with its lot.
- The expense takes the site of the chosen warehouse; the tab does not ask for a site.
- The units enter the stock at their cost in soles, net of the IGV the comprobante recovers as
  crédito fiscal. A boleta or no comprobante keeps the IGV in the cost. A USD purchase is converted
  with the comprobante's exchange rate; without that rate the units enter without a cost.

### Result and side effects (Resultado y efectos)

Genix creates an expense of type **Inventario**, unpaid, and an inbound stock movement **Supplies
purchase (Compra de insumos)** in the warehouse for the quantity bought. No cash moves until the
expense is paid.

### Limitations (Limitaciones)

- After saving, the expense opens with **Detalle** and **Pago** like any other; editing it there
  changes the expense only, not the units or cost already entered in the warehouse.
- Sellable merchandise and lot-tracked supplies are bought with **Purchase Orders (Órdenes
  Compra)**.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo compro insumos y los ingreso al almacén? ¿Cómo registro una compra de materiales sin orden
  de compra?
- ¿Por qué me dice que el insumo controla lotes? → buy it with a purchase order.
- Search terms: `compra de insumos`, `ingreso de materiales`, `gasto de inventario`, `existencias`.

<!-- DOC-ID: capability.supplier-comprobante -->
## Record the supplier's comprobante (Registrar el comprobante del proveedor)

### User intention (Intención del usuario)

Attach the supplier's document (factura, boleta, recibo por honorarios, ticket, recibo de servicios
públicos) to an expense, with its IGV split, so it enters the purchase register.

### Where to find it (Dónde encontrarlo)

At the bottom of the **Simple**, **Inventario** and **Detalle** tabs: **Comprobante** (empty means
**Sin comprobante**), then **Series (Serie)**, **Number (Número)**, **Issue date (Fecha de
Emisión)**, **Taxable base (Base Imponible)**, **IGV**, **Untaxed (No Gravado)**, **Other charges
(Otros Cargos)**, and **Exchange rate (Tipo de cambio)** when the currency is USD. **Split the total
(Desglosar el total)** proposes the amounts from the expense amount.

### Required information and prerequisites (Requisitos previos)

With a comprobante type chosen: the supplier, series and number, and the issue date are required.
A recibo de servicios públicos also needs the expense's **Vencimiento**. A USD comprobante needs
the exchange rate of its issue date.

### Business rules and rationale (Reglas y razón de negocio)

- **Document total (Total del comprobante)** = base + IGV + no gravado + otros cargos, and must
  equal the expense amount; the page shows it in red with **Must equal (Debe ser igual a)** while
  they differ. The comprobante is the bill for that amount, so two different figures are refused.
- A boleta or a recibo por honorarios grants no crédito fiscal: its whole amount goes in **No
  Gravado**.
- IGV needs a taxable base. No amount can be negative.
- The issue date cannot be in the future.
- The series is up to 20 characters: letters, numbers and hyphens.
- **Desglosar el total** splits at the general 18% IGV rate as a proposal; correct it to the
  figures printed on the document.

### Result and side effects (Resultado y efectos)

The expense is booked on the comprobante's issue date (or on its own date when it has none) and is
read by the purchase register in **Books (Libros)**. It also decides the payable its payments
clear: suppliers (4212) with a comprobante, other payables (4699) without one.

### Limitations (Limitaciones)

Credit and debit notes (`notas de crédito`, `notas de débito`) cannot be registered as a
comprobante.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde registro la factura del gasto? ¿Cómo separo el IGV? ¿Por qué no me deja poner IGV en una
  boleta?
- ¿Puedo registrar la factura después de pagar? → yes, see editing below.
- Search terms: `comprobante`, `factura`, `boleta`, `RxH`, `recibo por honorarios`, `ticket`,
  `base imponible`, `IGV`, `crédito fiscal`, `registro de compras`, `tipo de cambio`.

<!-- DOC-ID: capability.edit-expense -->
## Edit an expense (Editar un gasto)

### User intention (Intención del usuario)

Correct an expense, or add its comprobante or account after it was registered or paid.

### Where to find it (Dónde encontrarlo)

**Register (Registro)** → click the expense row → tab **Details (Detalle)** → panel save button.

### Business rules and rationale (Reglas y razón de negocio)

- While not fully paid, every field can change, but the amount cannot go below what was already
  paid.
- Once **Pagado**, the page shows "Este gasto está pagado: sólo se puede registrar su comprobante y
  su cuenta contable". Name, category, description, currency, amount, dates and production order
  are locked; the comprobante, the accounting account, the supplier and the site can still be
  saved. The bill often arrives after the payment, and classifying it does not change the money.

### Limitations (Limitaciones)

Payment status and paid amount are never typed; only payments change them.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué no puedo cambiar el monto de un gasto pagado? ¿Cómo agrego la factura a un gasto ya
  pagado?

<!-- DOC-ID: capability.pay-expense -->
## Pay an expense (Pagar un gasto)

### User intention (Intención del usuario)

Record that the expense was paid from a cash register or bank account, fully or in part (`pago a
cuenta`, `abono`).

### Where to find it (Dónde encontrarlo)

**Register (Registro)** → click the expense → tab **Payment (Pago)**. It shows **Monto**,
**Pagado** and **Pendiente**, then **Source Register (Caja de Origen)**, **Payment Amount (Monto
del Pago)**, **Payment Date (Fecha del Pago)**, **Is Fully Paid (Pagado Completo)** and the button
**Register Payment (Registrar Pago)**.

### Required information and prerequisites (Requisitos previos)

The register defaults to the first one in the expense's currency; the date defaults to today.

### Business rules and rationale (Reglas y razón de negocio)

- The payment must be greater than 0 and cannot exceed the pending amount.
- The register's currency must match the expense's currency.
- The register's balance cannot become negative.
- The expense becomes **Pagado** when payments reach its amount. Ticking **Pagado Completo** closes
  it as **Pagado** even when this payment leaves part unpaid; the remainder is not paid.
- A depreciation entry is never paid.

### Result and side effects (Resultado y efectos)

Each payment writes one **Pago Gasto** outflow in the chosen register and lowers its balance. The
expense's **Pagado** grows; it shows **Parcial** while part is pending and moves to the **Pagados**
tab when fully paid. **Payments made (Pagos realizados)** lists each payment with date, register,
user and amount; on a paid expense, that list replaces the payment form.

### Limitations (Limitaciones)

- A payment cannot be undone or deleted from this page.
- A paid expense accepts no more payments here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo pago un gasto? ¿Puedo pagar una parte? ¿Cómo cierro un gasto que pagué de menos?
- ¿Por qué no me deja pagar desde esta caja? → different currency or not enough balance.
- Search terms: `pagar gasto`, `abono`, `pago parcial`, `pago a cuenta`, `cancelar deuda`, `caja`,
  `banco`.

<!-- DOC-ID: capability.list-expenses -->
## Find expenses by payment status (Ver gastos por estado)

**Register (Registro)** lists expenses with **ID**, **Registered (Registro)**, **Status (Estado)**,
**Name (Nombre)**, **Type (Tipo)**, **Category (Categoría)**, **Due Date (Vencimiento)**, **Amount
(Monto)** and **Paid (Pagado)**. The tabs **All (Todos)**, **Pend. Payment (Pend. Pago)** and
**Paid (Pagados)** filter by payment status; the filter box searches by name. **Todos** and
**Pagados** load the latest 2000 expenses; **Pend. Pago** loads every pending one.

- ¿Qué gastos tengo pendientes de pago? → tab **Pend. Pago**.

<!-- DOC-ID: capability.schedule-recurring-expense -->
## Schedule a recurring expense (Programar un gasto recurrente)

### User intention (Intención del usuario)

Set up a fixed expense that repeats (`alquiler mensual`, `internet`, `vigilancia`) once, instead of
registering it every period.

### Where to find it (Dónde encontrarlo)

View **Scheduled (Programados)** → **New (Nuevo)** opens **New Schedule (Nueva Programación)**,
tab **Details (Detalle)**. Click a schedule row to edit it. The list shows **Name**, **Category**,
**Cadence (Cadencia)**, **Amount** and **Start (Inicio)**.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)**, **Category (Categoría)** (defaults to Alquiler), **Description (Descripción)**
  optional, **Currency (Moneda)**, **Amount (Monto)** greater than 0, **Accounting account (Cuenta
  contable)** (an expense account).
- **Cadence (Cadencia):** Semanal, Mensual, Cada 2 meses, Cada 3 meses, Cada 4 meses or Anual.
  Weekly asks **Weekday (Día de la semana)**; the others ask **Day of month (Día del mes)**, 1 to 31.
- **Start Date (Fecha Inicio)** is required; **End Date (optional) (Fecha Fin (opcional))**.
- **Site (Sede)**, empty for the whole company.

### Business rules and rationale (Reglas y razón de negocio)

- Weekly periods fall on the chosen weekday, the first one on or after the start date.
- Monthly and longer cadences count months from the start date's month; a day beyond the month's
  length falls on its last day (31 → 28 or 30).
- No period is generated after the end date.

### Result and side effects (Resultado y efectos)

Opening a schedule generates its periods up to today, and the **Periods (Periodos)** tab lists them
with **Period (Periodo)**, **Monto**, **Pagado** and **Estado**. Each period is a separate simple
expense in **Register** with the schedule's name, amount, category, account, site and currency,
dated and due on the period date, ready to adjust, add its comprobante and pay. The **Cash Flow
(Flujo de Caja)** forecast projects the upcoming periods of active schedules.

### Limitations (Limitaciones)

- Periods are created only when the schedule is opened in **Programados**; until then they do not
  appear in **Registro**.
- Editing a schedule does not change periods already generated.
- A schedule is not paid; pay each period in **Registro**.
- There is no delete action for a schedule; set **Fecha Fin** to stop it.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo programo el alquiler mensual? ¿Cómo registro un gasto fijo que se repite?
- ¿Por qué no aparece el gasto de este mes en Registro? → open the schedule in Programados.
- ¿Cómo dejo de generar un gasto recurrente? → set its Fecha Fin.
- Search terms: `gasto fijo`, `gasto recurrente`, `programado`, `periodo`, `mensual`, `cadencia`.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El total del comprobante (…) no coincide con el monto del gasto (…)":** fix the comprobante
  amounts or use **Desglosar el total**.
- **"Un comprobante sin crédito fiscal (boleta, recibo por honorarios) se registra entero como
  importe no gravado":** move the amount to **No Gravado**.
- **"Un comprobante de compra debe indicar el proveedor que lo emitió":** choose the **Proveedor**.
- **"La fecha de emisión del comprobante no puede ser futura":** check the issue date.
- **"El monto no puede ser menor a lo ya pagado":** the amount cannot drop below the payments.
- **"La moneda de la caja no coincide con la del gasto":** pay from a register in the expense's
  currency.
- **"El saldo de la caja no puede quedar negativo":** pay from another register or record the
  missing income first in Cash & Banks.
- **"… controla lotes: cómprelo con una orden de compra…":** use a purchase order.
- **"La orden de producción no es de maquila de este proveedor":** choose the workshop as supplier.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash & Banks (Cajas & Bancos)** shows each **Pago Gasto** movement in its register.
- **Cash Flow (Flujo de Caja)** projects pending expenses by due date and the upcoming periods of
  scheduled expenses.
- **Suppliers (Proveedores)** registers the supplier and its **Días de crédito**.
- **Assets (Activos)** registers fixed asset purchases and their depreciation.
- **Purchase Orders (Órdenes Compra)** buy merchandise and lot-tracked supplies with reception.
- **Supplies & Materials (Insumos & Materiales)** keeps the supplies an inventory expense buys.
- **Books (Libros)** reads the comprobantes in the purchase register.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.register-simple-expense]
  - path: frontend/routes/finance/expenses/+page.svelte
    role: page
    hash: sha256:2430f85701840070c210f7132a8d413e328a51bebe24b532422dbcfada3ca7b8
    supports: [page-purpose]
  - path: frontend/routes/finance/expenses/ExpensesRegister.svelte
    role: user-interface
    hash: sha256:d321929caabc8bafafe601a9692222322eba6a62c78f4620f063f5717e121f10
    supports: [concepts, capability.register-simple-expense, capability.register-inventory-expense, capability.edit-expense, capability.pay-expense, capability.list-expenses]
  - path: frontend/routes/finance/expenses/ExpensesSchedule.svelte
    role: user-interface
    hash: sha256:e285c11e87d1f859b763097f8f2010222f600df36d90cdfcdb111483a93bddfa
    supports: [capability.schedule-recurring-expense]
  - path: frontend/routes/finance/expenses/expenses.svelte.ts
    role: frontend-service
    hash: sha256:94bd986f00523e85525d4c24b5f9e279b529764f2ef220fe8bde0015427322aa
    supports: [concepts, capability.register-simple-expense, capability.schedule-recurring-expense, capability.list-expenses]
  - path: frontend/domain-components/PurchaseDocumentFields.svelte
    role: user-interface
    hash: sha256:fc3a46a9311ac0be2c7f42996646c958c521df7e7807065590b0a6dda9fdf75e
    supports: [capability.supplier-comprobante]
  - path: frontend/core/purchase-document.ts
    role: shared-domain
    hash: sha256:6a9178bf99be21baf75b2c106b241fb24e410bfca9e3c42ad5fab840e7480cb0
    supports: [capability.supplier-comprobante]
  - path: frontend/domain-components/ClientProviderMaintainer.svelte
    role: user-interface
    hash: sha256:d4a38161e8ca38c5a2be6a4c691e9c47978d551641b075629b277b3bd32beab0
    supports: [capability.register-simple-expense, related-pages]
  - path: backend/finance/expenses.go
    role: backend-handler
    hash: sha256:88afa304e7aa3b3b96f3c02055efd98fe1a9da60b869dfd08de6d2f10780fd05
    supports: [capability.register-simple-expense, capability.edit-expense, capability.pay-expense, capability.list-expenses, capability.schedule-recurring-expense, troubleshooting]
  - path: backend/accounting/inventory_expense.go
    role: backend-handler
    hash: sha256:7fb866b2329538fffbdb5c9d1ec3875b06d31a735a7880a4addfa7645dd32ad5
    supports: [capability.register-inventory-expense, troubleshooting]
  - path: backend/finance/types/expenses.go
    role: data-model
    hash: sha256:6bb15089e1434be15645e06909f8f7ef73172b4d46ba58c58786ad11caba7c72
    supports: [page-purpose, concepts, capability.register-simple-expense, capability.supplier-comprobante, related-pages]
  - path: backend/finance/types/expense_category.go
    role: data-model
    hash: sha256:0b8afa433f265154f06550bec43fc8b92322c913bb0bf77c2cf91023c1db09cd
    supports: [page-purpose, concepts]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: frontend-service
    hash: sha256:051c429f6af7fbc3031301c0b052febfdbd06b5b71a689d695cedbc538e4a1c7
    supports: [capability.pay-expense, related-pages]
  - path: frontend/routes/accounting/books/books.purchases.ts
    role: business-logic
    hash: sha256:24b3356c896e612766a1bf5b03e87d850515d79e6241d71c31fd027ba5f393ff
    supports: [capability.supplier-comprobante, related-pages]
  - path: backend/finance/types/purchase_document.go
    role: business-logic
    hash: sha256:ec949ba6c381199a92b8310f78d4b02a3ba79ea02f84653c4a4020a52cb0be3a
    supports: [concepts, capability.supplier-comprobante, capability.register-inventory-expense, troubleshooting]
  - path: backend/finance/types/cash_movement_account.go
    role: business-logic
    hash: sha256:2c462a5021a979a7ce37405eb667f99e784b699f75e024c419c184e239a93eba
    supports: [capability.supplier-comprobante]
  - path: backend/crm/types/client_provider.go
    role: business-logic
    hash: sha256:d8f051ef033fae9ba37de0133d6391283f6703310848af61e3f957794b541e87
    supports: [capability.register-simple-expense]
  - path: backend/finance/cash_flow_commitments.go
    role: business-logic
    hash: sha256:642f494dd1491665739525d2db42e59621e75966dc902a9652b58346da0aa7f0
    supports: [capability.schedule-recurring-expense, related-pages]
```
