---
schema: 1
page_id: hr.payroll
route: /hr/payroll
title: Payroll (Planilla)
status: implemented
visibility: tenant
description_en: >-
  The monthly payroll of the Planilla workers: one run per month with one payslip per worker,
  calculated automatically from contracts, employee records, time off, loans, weekly payments and the
  legal values. Close it, reopen it, pay the net pay from a cash or bank account, pay the
  contributions and withholdings, and print each payslip. A second tab registers weekly and biweekly
  payments made before the payslip.
description_es: >-
  Planilla mensual de los trabajadores en planilla: una planilla por mes con una boleta por
  trabajador, calculada automáticamente con contratos, registros, ausencias, préstamos, pagos
  semanales y valores legales. Cierre, reapertura, pago del neto desde una caja o banco, pago de
  aportes y retenciones e impresión de boletas. Otra pestaña registra los pagos semanales o
  quincenales hechos antes de la boleta.
---

# Payroll (Planilla)

<!-- DOC-ID: page-purpose -->
## Page purpose

Payroll (`Planilla`) is where the company calculates, closes and pays the monthly payroll of its
workers engaged on **Planilla**. Each month has one payroll run (`planilla del mes`) holding one
payslip (`boleta de pago`) per worker. The page does not ask the user to type pay: Genix calculates
each payslip from the worker's contract, the month's records, time off, loan installments, weekly
payments and the legal values (RMV, UIT, EsSalud, ONP, AFP rates).

The page does not register the facts it calculates from. Contracts and fixed concepts are kept in
Employees & Contracts (`Empleados & Contratos`); overtime, absences, bonuses and staff loans in
Employee Records (`Registros Empleados`); vacations and leaves in Time Off (`Vacaciones y
Licencias`). Workers engaged by fee receipt (`recibo por honorarios`) or as temporary off payroll
(`temporal sin planilla`) never get a payslip here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **payroll run** (`planilla`) is the payroll of one month (`periodo`). Its state is **Draft
  (Borrador)** before the first calculation, **Calculated (Calculada)**, **Closed (Cerrada)** or
  **Paid (Pagada)**.
- A **payslip** (`boleta de pago`) is one worker's result in a run. One payslip holds everything the
  month brings: the salary, the gratificación in July and December, the CTS deposit in May and
  November, the settlement (`liquidación`) when the worker leaves that month, extraordinary
  payments and profit sharing (`utilidades`).
- **Net pay (Neto a pagar)** is earnings minus deductions; the CTS is shown as an earning but is
  excluded from the net pay because it is deposited in the worker's CTS bank account.
- **Employer cost (Costo empleador)** is earnings plus employer contributions such as EsSalud.
- **Observations (Observaciones)** are diagnostics with a code (`HR-001`…). Blocking ones (red)
  prevent closing the run; the others (orange) are warnings.
- A **weekly payment** (`pago semanal`, `adelanto semanal`, `adelanto quincenal`, `adelanto otro`)
  is money paid to a worker before the monthly payslip, which then deducts it as **Advances paid
  (Adelantos pagados)**.

<!-- DOC-ID: capability.monthly-list -->
## See the monthly payrolls and keep them calculated (Planillas del mes)

### User intention (Intención del usuario)

See every month's payroll with its state and net pay, and have the open months recalculated after
contracts, records or legal values changed on other pages.

### Where to find it (Dónde encontrarlo)

**Company (Empresa) → HR (RRHH) → Payroll (Planilla)** at `/hr/payroll`, tab **Payroll
(Planilla)**. The left column **Payrolls (Planillas)** lists the months; each card shows the month,
its state badge or **No payroll (Sin planilla)**, and its **Net pay (Neto)**. Click a month to see
its payslips. The current month opens first.

### Required information and prerequisites (Requisitos previos)

- Workers engaged on Planilla with a contract version in force (Employees & Contracts).
- The legal values in force for the month (shown in HR Settings → Legal values).
- The sub-access **Calcular Planilla** to have the page calculate. Without it the page only lists
  the runs that already exist.

### Business rules and rationale (Reglas y razón de negocio)

- There is no "new payroll" button. Opening the page asks Genix to bring the list up to date: it
  creates the run of each month that has someone to pay and recalculates every open run whose
  inputs changed since its last calculation. While it works the column shows **Updating...
  (Actualizando...)**.
- The list covers from 12 months before the company registered in Genix up to the month after the
  current one.
- A month where nobody is paid has no run: its detail says the month has no workers to pay.
- Closed and paid runs are never recalculated.
- **Recalculate (Recalcular)** recalculates an open run on demand, even when nothing changed.
- A worker who could not be paid in a month is reported in a message with the month, the worker and
  the reason, for example a missing contract.

### Result and side effects (Resultado y efectos)

Each run stores its payslips, its worker count, its total net pay and employer cost. Recalculating
replaces the payslips of the open run; a worker keeps the same payslip across recalculations.

### Limitations (Limitaciones)

- Records, time off and loans are not edited here; changing them on their pages is what makes the
  open month stale and recalculated the next time Payroll is opened.
- Only one payroll per month exists; there is no separate run for gratificación, CTS or
  liquidación.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo genero la planilla del mes? ¿Dónde calculo la planilla? ¿Por qué no hay botón de nueva
  planilla?
- ¿Por qué la planilla de este mes dice "Sin planilla"? ¿Cómo recalculo la planilla?
- Planilla de remuneraciones, nómina, rol de pagos, sueldos del mes.

<!-- DOC-ID: capability.payslip-calculation -->
## What each payslip calculates (Cálculo de la boleta)

### User intention (Intención del usuario)

Understand where each amount of a payslip comes from and why a worker received what he did.

### Where to find it (Dónde encontrarlo)

The payslips table of the selected month: **Employee (Trabajador)**, **Department (Área)**, **Days
(Días)**, **Earnings (Ingresos)**, **Deductions (Descuentos)**, **Net pay (Neto)**, **Employer cost
(Costo empleador)** and **Observations (Observaciones)**. The run header totals **Earnings
(Ingresos)**, **Net pay (Neto a pagar)** and **Employer cost (Costo empleador)**.

### Required information and prerequisites (Requisitos previos)

The contract version in force on the worker's last employed day of the month, and the RMV and UIT in
force on the month's last day. Without a contract in force the month's salary is not calculated;
without RMV or UIT the payslip carries a blocking observation.

### Business rules and rationale (Reglas y razón de negocio)

- **Salary (Remuneración básica):** the monthly pay over a 30-day month, for the days employed less
  unpaid absences, unpaid days off and vacation days. A full month counts 30 days whatever its
  length.
- **Vacation pay (Remuneración vacacional):** vacation days from Time Off are paid at the salary's
  daily value instead of salary.
- Unpaid leave (`licencia sin goce`), medical leave subsidized by EsSalud and maternity are not paid
  by the payroll. Medical leave paid by the company is paid as salary.
- **Family allowance (Asignación familiar):** a percentage of the RMV, paid when a registered child
  gives the right to it on the month's last day.
- **Overtime 25% / 35% (Horas extras 25% / 35%):** the hours from Employee Records times the hourly
  value of salary plus family allowance (monthly value / 30 / daily hours), at 125% or 135%.
- **Company concepts:** fixed concepts of the worker and the month's amounts from Employee Records.
  A concept marked to be paid on a separate payslip is calculated as an extraordinary payment with
  its own pension, 5ta and EsSalud.
- **Pension:** ONP at the ONP rate, or the worker's AFP: fund contribution, insurance premium (on the
  pay capped by the maximum insurable remuneration) and the flow commission only for commission
  type **On flow (Sobre el flujo)**; a mixed commission is not withheld through payroll.
- **Income tax 5th category (Renta de quinta categoría):** each month withholds a share of the
  worker's projected annual tax.
- **Loan installments** of the month, company deductions, and last the **weekly payments** of the
  month.
- **EsSalud** is the employer's contribution, on at least the RMV for a full month of a full-time
  worker, at the EsSalud with EPS rate when the contract has EPS.
- **Gratificación** (July pays January–June, December pays July–December): a sixth of the
  computable remuneration per full month of the semester, plus the **Bonificación extraordinaria
  (Ley 30334)**. Only for workers still employed when the semester ended.
- **CTS** (May pays November–April, November pays May–October): the computable remuneration plus a
  sixth of the gratificación received, per day over 360. Part-time workers (under 4 daily hours)
  have no CTS.
- **Settlement (Liquidación)** in the month of the termination date: truncated gratificación and
  CTS, vacation earned and not taken, the indemnity when the termination reason is **Dismissal
  (Despido)** (1.5 remunerations per year of service, or per remaining month of a fixed-term
  contract, capped at 12), the final 5ta, and the loan installments due after that month.
- The computable remuneration for benefits is salary, family allowance and the variable computable
  earnings received in at least 3 months of the reference period.
- Monthly provisions for gratificación, CTS and vacation are calculated for the accounting; they are
  not printed on the payslip.

### Result and side effects (Resultado y efectos)

Each payslip keeps the legal values it was calculated with, so a later legal change never rewrites
a closed month. It also records the worker's department and that department's cost function.

### Limitations (Limitaciones)

- The calculation cannot be overridden by typing an amount on the payslip; change the input on its
  page and let the run recalculate.
- Only the general labor regime calculation exists here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo calcula Genix la gratificación? ¿Cuándo se paga la CTS? ¿Por qué la CTS no suma al neto?
- ¿Cómo se calculan las horas extras? ¿Por qué no le pagaron la asignación familiar?
- ¿Cómo se calcula la liquidación de un trabajador cesado? ¿Se paga indemnización por despido?
- Boleta, sueldo, haberes, descuentos de ley, AFP, ONP, quinta categoría, EsSalud, grati, CTS,
  liquidación de beneficios sociales.

<!-- DOC-ID: capability.edit-run -->
## Set the pay date and profit sharing (Editar planilla)

### User intention (Intención del usuario)

Change the day the payroll is paid, add a note, or distribute the profit sharing (`participación en
utilidades`) in the month it is paid.

### Where to find it (Dónde encontrarlo)

Select the month, then the pencil (edit) button beside its title. The **Edit payroll (Editar
planilla)** dialog has **Pay date (Fecha de pago)**, **Net income before tax (S/) (Renta neta antes
de impuestos (S/))**, **Sector rate (Porcentaje del sector)** and **Note (Nota)**.

### Required information and prerequisites (Requisitos previos)

- The run must be open (Draft or Calculated).
- A new run's pay date is the month's last day.
- For profit sharing: the previous fiscal year's net income before tax from the annual return, and
  the sector rate: **10 %** (fishing, telecommunications, industry), **8 %** (mining, trade,
  restaurants) or **5 %** (other activities). Leave the income empty when there is no profit
  sharing.

### Business rules and rationale (Reglas y razón de negocio)

- The net income cannot be negative; with an income the rate must be 5 %, 8 % or 10 %.
- The pool (income × rate) is shared among everyone with a payslip in the previous fiscal year,
  former workers included: half by days worked and half by remunerations received.
- A worker's share is capped at 18 monthly remunerations; the excess goes to FONDOEMPLEO and the
  payslip shows a warning observation (`HR-011`).
- Profit sharing is not remunerative; its 5ta is withheld as extraordinary income.

### Result and side effects (Resultado y efectos)

Saving recalculates the run at once. The pay date is printed on the payslips and is the date of the
net pay cash movement.

### Limitations (Limitaciones)

A closed or paid run cannot be edited; reopen a closed one first.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo pago las utilidades? ¿Dónde registro la participación de utilidades?
- ¿Cómo cambio la fecha de pago de la planilla?

<!-- DOC-ID: capability.close-reopen -->
## Close or reopen a payroll (Cerrar o reabrir la planilla)

### User intention (Intención del usuario)

Make the month's payslips final so they can be paid and accounted, or reopen them to correct
something.

### Where to find it (Dónde encontrarlo)

On the selected month: **Close (Cerrar)** when it is Calculated, **Reopen (Reabrir)** when it is
Closed. Both ask for confirmation. They need the sub-access **Cerrar Planilla**.

### Required information and prerequisites (Requisitos previos)

- To close: the run is open and calculated, and no payslip has a blocking observation. **Close
  (Cerrar)** stays disabled and a red line counts the payslips with observations that block
  closing.
- To reopen: the run is Closed and not yet paid.

### Business rules and rationale (Reglas y razón de negocio)

- Closing is refused when anything the payroll reads changed after the last calculation (a record,
  a contract, a legal value): recalculate first.
- Closing settles what the payslips consumed: the month's records from Employee Records of the paid
  workers, the weekly payments and the loan installments they deducted. Those records become read
  only (`Liquidado`, `Descontada`, `Descontado en boleta`).
- Time off is not settled by closing.
- While a month is closed, Genix refuses new or changed records, time off, weekly payments and loan
  installments that touch that month.
- Reopening gives every record settled by the run back to active and takes its payslips out of the
  accounting until the run is closed again.

### Result and side effects (Resultado y efectos)

A closed run's payslips are what the accounting and the Cash Flow forecast read as the month's
known payroll.

### Limitations (Limitaciones)

A paid run cannot be reopened.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cierro la planilla? ¿Por qué no puedo cerrar la planilla? ¿Cómo reabro una planilla
  cerrada?
- Cerrar mes, aprobar planilla, planilla definitiva.

<!-- DOC-ID: capability.pay-net -->
## Pay the net pay (Pagar planilla)

### User intention (Intención del usuario)

Record that the workers were paid the month's net pay.

### Where to find it (Dónde encontrarlo)

On a Closed month, **Pay (Pagar)** opens **Pay payroll (Pagar planilla)**, which shows **Net pay
(Neto a pagar)** and asks **Paid from (Pagado desde)**; confirm with **Pay (Pagar)**. It needs the
sub-access **Pagar Planilla**.

### Required information and prerequisites (Requisitos previos)

A Closed run and an active cash register or bank account in soles; only those are offered.

### Business rules and rationale (Reglas y razón de negocio)

- Payroll is paid in soles: an account in another currency is refused.
- The run is paid once: a second payment finds it already Paid.

### Result and side effects (Resultado y efectos)

One **Pago Planilla** cash movement for the run's whole net pay leaves the chosen account on the
run's pay date, against salaries payable (4111). The run and its payslips become **Paid (Pagada)**.

### Limitations (Limitaciones)

- There is no per-worker payment and no bank payment file: the net pay leaves as one movement.
- The payment cannot be undone from this page, and a paid run cannot be reopened.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro el pago de sueldos? ¿De qué caja sale el pago de la planilla?
- Pago de haberes, abono de sueldos.

<!-- DOC-ID: capability.pay-contributions -->
## Pay contributions and withholdings (Pagar aportes)

### User intention (Intención del usuario)

Record the payment of what the payroll owes to third parties: EsSalud, ONP, 5ta, AFP or judicial
withholdings.

### Where to find it (Dónde encontrarlo)

On a Closed or Paid month, **Contributions (Aportes)** opens **Pay contributions (Pagar aportes)**:
**Payable (Concepto a pagar)**, **Amount (S/) (Monto (S/))**, **Date (Fecha)** and **Paid from
(Pagado desde)**. It needs the sub-access **Pagar Planilla**.

### Required information and prerequisites (Requisitos previos)

- Payables offered: **4031 EsSalud**, **4032 ONP**, **40173 Renta de 5ta categoría**, **417 AFP**
  and **419 Retenciones judiciales**.
- The amount is prefilled with what the run's payslips credited to that payable; adjust it when
  part was already paid.

### Business rules and rationale (Reglas y razón de negocio)

- Only a closed (or paid) run's contributions can be paid.
- The amount cannot exceed what the run still owes on that payable after earlier payments.
- The account must be in soles.

### Result and side effects (Resultado y efectos)

A **Pago Aportes Planilla** cash movement leaves the account on the given date against that
payable. A message shows what is still pending on that payable.

### Limitations (Limitaciones)

Genix only records the cash payment; the declaration itself (PLAME, AFPnet) is filed outside this
page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro el pago de EsSalud? ¿Dónde pago la AFP de la planilla? ¿Cómo pago la retención de
  quinta?
- Aportes, retenciones, pago a SUNAT, tributos laborales.

<!-- DOC-ID: capability.view-print-payslip -->
## View and print a payslip (Ver e imprimir la boleta)

### User intention (Intención del usuario)

Check one worker's detail or give him his printed payslip.

### Where to find it (Dónde encontrarlo)

Click a worker's row: a side panel shows his observations and the payslip preview. **Print
(Imprimir)** opens the browser's print dialog with only the payslip.

### Business rules and rationale (Reglas y razón de negocio)

- The payslip (**BOLETA DE PAGO**) shows the period, pay date, worker name and DNI, pension system,
  **Days worked (Días laborados)**, the **Earnings (Ingresos)** and **Deductions (Descuentos)**
  tables with totals, **Employer contributions (Aportes del empleador)**, **Net pay (Neto a
  pagar)** and signature lines for **Employer (Empleador)** and **Employee (Trabajador)**.
- The CTS line is marked as deposited in the bank (`depositada en el banco`).
- Observations are listed with code and message: red when they block closing, orange when they are
  warnings.

### Limitations (Limitaciones)

There is no PDF file or email sending from this page; the payslip is printed from the browser.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo imprimo la boleta de pago? ¿Dónde veo la boleta de un trabajador?

<!-- DOC-ID: capability.weekly-payments -->
## Register weekly or biweekly payments (Pagos semanales)

### User intention (Intención del usuario)

Record money paid to a worker during the month (weekly, biweekly or other), so the monthly payslip
deducts it.

### Where to find it (Dónde encontrarlo)

Tab **Weekly payments (Pagos semanales)**. **Settles in period (Se descuenta en)** chooses the
month shown. **Register payment (Registrar pago)** opens a dialog with **Employee (Trabajador)**,
**Type (Tipo)** (**Weekly advance (Adelanto semanal)**, **Biweekly advance (Adelanto quincenal)**,
**Other advance (Adelanto otro)**), **Amount (S/) (Monto (S/))**, **Covers from (Cubre desde)**,
**Covers to (Cubre hasta)**, **Paid on (Pagado el)**, **Paid from (Pagado desde)** and **Note
(Nota)**.

### Required information and prerequisites (Requisitos previos)

- Only active Planilla workers are offered.
- The worker needs a contract in force on the **Covers to** date.
- A soles cash register or bank account.

### Business rules and rationale (Reglas y razón de negocio)

- The covered days must fall in a single month; the payment is deducted in that month.
- A worker cannot have two payments covering the same days.
- The month's payroll must not be closed.
- A payment is never edited: void it and register it again. Click a payment with status **Paid
  (Pagado)** to void it (**Void payment (Anular pago)**); one already **Settled in payslip
  (Descontado en boleta)** cannot be voided.

### Result and side effects (Resultado y efectos)

The payment leaves the chosen account on **Paid on** as a **Pago Planilla** movement against
remuneration advances (1412). The month's payslip deducts it as **Advances paid (Adelantos
pagados)**. Voiding returns the money to the same account.

### Limitations (Limitaciones)

If the month's payments exceed the payslip's net, the payslip gets a blocking observation; reduce a
payment or register the money as a loan in Employee Records instead.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro un adelanto de sueldo? ¿Cómo pago semanal a un trabajador de planilla? ¿Cómo
  anulo un adelanto?
- Adelanto, quincena, pago a cuenta, semana.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Only workers whose engagement is **Payroll (Planilla)** enter runs, payslips, profit sharing and
  weekly payments.
- **Recalculate (Recalcular)** needs the sub-access **Calcular Planilla**, **Close (Cerrar)** and
  **Reopen (Reabrir)** need **Cerrar Planilla**, and **Pay (Pagar)** and **Contributions
  (Aportes)** need **Pagar Planilla**; without them the buttons are not shown.
- Every payroll cash movement uses a cash register or bank account in soles.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"No tiene un contrato vigente en el periodo" (`HR-001`):** add or fix the worker's contract
  version in Employees & Contracts.
- **`HR-002` "El contrato no tiene sistema de pensiones":** choose ONP or an AFP in the contract.
- **`HR-003` "Faltan las tasas de la AFP del periodo …":** the month's AFP rates are not loaded yet;
  the SaaS operator loads them in Labor Parameters.
- **`HR-004` missing RMV, UIT or another legal value:** the legal value in force is missing in Labor
  Parameters.
- **`HR-005` "Los adelantos … superan el neto del mes":** void or reduce a weekly payment, or use a
  loan.
- **`HR-006` remuneration below the RMV:** a warning only; it does not block closing.
- **`HR-007` "El concepto … no existe o fue eliminado":** a record or fixed concept points to a
  deleted company concept.
- **`HR-009` "Los descuentos … superan la liquidación":** the loan still owed exceeds the
  settlement; adjust the loan.
- **"Hay datos que cambiaron después del cálculo: vuelva a calcular la planilla":** use
  **Recalculate (Recalcular)** and close again.
- **"La planilla del periodo … ya está cerrada; reábrala para hacer cambios":** a record, time off,
  payment or installment touches a closed month; reopen it here first.
- **"La planilla se paga en soles…":** choose an account in soles.
- **"El monto supera lo pendiente de la cuenta …":** that payable is already partly paid.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Employees & Contracts (Empleados & Contratos):** workers, engagement, contracts, dependents and
  fixed concepts.
- **Employee Records (Registros Empleados):** overtime, absences, company concept amounts and staff
  loans.
- **Time Off (Vacaciones y Licencias):** vacations and leaves the payroll pays or leaves unpaid.
- **HR Settings (Configuración RRHH):** company concepts and the legal values in force.
- **Cash & Banks (Cajas & Bancos):** the accounts the payroll is paid from and their movements.
- **Cash Flow (Flujo de Caja):** projects closed runs as known payroll and open months from the
  current staff.
- **Hours vs payroll (Horas vs planilla):** compares production hours with the month's payslips.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.monthly-list]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [rules]
  - path: frontend/routes/hr/payroll/+page.svelte
    role: page
    hash: sha256:24df1d684ac3ceb6c65f90810493835d3b61f46a8f4037bb3c4b9bae8e20336a
    supports: [page-purpose, capability.monthly-list, capability.weekly-payments]
  - path: frontend/routes/hr/payroll/RunsTab.svelte
    role: user-interface
    hash: sha256:0441a895ea0e67a2bf6ef64a41baf237405e013c19ddf2f48993efa62f062207
    supports: [concepts, capability.monthly-list, capability.payslip-calculation, capability.edit-run, capability.close-reopen, capability.pay-net, capability.pay-contributions, capability.view-print-payslip, rules]
  - path: frontend/routes/hr/payroll/AdvancesTab.svelte
    role: user-interface
    hash: sha256:149cea97ccd50b68784b2063ed3fddc826601959aa7b4c49fb007b31f3dd94f6
    supports: [concepts, capability.weekly-payments]
  - path: frontend/routes/hr/payroll/PayslipBoleta.svelte
    role: user-interface
    hash: sha256:f77d166415311c9d3837de11abb14c0e4e8b632618ab8e9f5716c4dcc1df2a4a
    supports: [capability.view-print-payslip]
  - path: frontend/routes/hr/payroll/payroll.ts
    role: business-logic
    hash: sha256:58c3885b397125a11a9b2922a8cabd0023f1f0e74e39b94935fac0037090e15e
    supports: [concepts, capability.monthly-list, capability.edit-run, capability.close-reopen, capability.pay-net, capability.pay-contributions, rules]
  - path: frontend/routes/hr/employees/employees.ts
    role: business-logic
    hash: sha256:590c1af5be318be15985ecee8eba31897d03d23728093762326e929cf1748672
    supports: [capability.weekly-payments, rules]
  - path: frontend/services/hr/payroll.svelte.ts
    role: frontend-service
    hash: sha256:1a876ca2e211cb45cf7a27a985305a1fe03e87c23a6cdca8b8b411699e110b11
    supports: [capability.monthly-list, capability.edit-run, capability.close-reopen, capability.pay-net, capability.pay-contributions]
  - path: frontend/services/hr/employee-records.svelte.ts
    role: frontend-service
    hash: sha256:597e465444a4ee69165e90378ddd5de17a447d61f2f303c6ace78716944bf460
    supports: [capability.weekly-payments]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts, capability.payslip-calculation, capability.weekly-payments]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:051c429f6af7fbc3031301c0b052febfdbd06b5b71a689d695cedbc538e4a1c7
    supports: [capability.pay-net, capability.pay-contributions, capability.weekly-payments]
  - path: backend/hr/main.go
    role: backend-handler
    hash: sha256:af4765a732931d749fa5e6d11ad4d13d4565cb8695321544a3e66d08e7c8192c
    supports: [page-purpose]
  - path: backend/hr/payroll_calculate.go
    role: backend-handler
    hash: sha256:0cbda12fb8d81d3c6582ca08f9011fb3af283dc22c059653eaf84e24048009b2
    supports: [capability.monthly-list, capability.edit-run, capability.payslip-calculation, troubleshooting]
  - path: backend/hr/payroll_runs.go
    role: backend-handler
    hash: sha256:5843b5ec5a3f0f5b719e4bac6900382b026471d763631fd7fee37684db901722
    supports: [capability.edit-run, capability.close-reopen, capability.pay-net, rules, troubleshooting]
  - path: backend/hr/payroll_cash.go
    role: backend-handler
    hash: sha256:b72e8c59a1c51b961c2229cd7e1e20391e26bf5b70c30fef746abc16777c1e13
    supports: [capability.pay-net, capability.pay-contributions, capability.weekly-payments, rules, troubleshooting]
  - path: backend/hr/employee_records.go
    role: backend-handler
    hash: sha256:74522e025c98f021b1005c567fa08e4b81aa124fc89b8c8940c974c418106bbd
    supports: [capability.close-reopen, capability.weekly-payments, troubleshooting]
  - path: backend/hr/types/payroll.go
    role: data-model
    hash: sha256:77649e81bc52c64e4c8982a5cbda388fed530b579684aeded39a2eb44bca4e4b
    supports: [concepts, capability.payslip-calculation, capability.edit-run]
  - path: backend/hr/types/payroll_data.go
    role: business-logic
    hash: sha256:bae9967e9ae7ba1e9060dde44697664aa5e30226407bf8a0b105380e21833e95
    supports: [page-purpose, capability.monthly-list, capability.close-reopen, rules]
  - path: backend/hr/types/payroll_period.go
    role: business-logic
    hash: sha256:4c640ce7682313c78742b3b5e64d1b160a54cda288bb113c9026556893eaf569
    supports: [concepts, capability.payslip-calculation]
  - path: backend/hr/types/payroll_engine.go
    role: business-logic
    hash: sha256:781443cdd93f5953057a93225c36b9fd1bf2e27efdfae924c887ced25c11d2b7
    supports: [concepts, capability.payslip-calculation, capability.weekly-payments, troubleshooting]
  - path: backend/hr/types/payroll_benefits.go
    role: business-logic
    hash: sha256:8c4b8f6432edf7193dae2949986696057cb3c190c5a342afb273990cc4fc669b
    supports: [capability.payslip-calculation, capability.edit-run, troubleshooting]
  - path: backend/hr/types/legal_concepts.go
    role: business-logic
    hash: sha256:0ec0c2c15ba57d378f4d6243a80fb1d3e90c54325354aea02501032fb76e38fd
    supports: [concepts, capability.payslip-calculation, capability.weekly-payments]
  - path: backend/hr/types/employee_records.go
    role: data-model
    hash: sha256:e5bea6740b8ae271994c455035ae734b86bbadcc66cfa8f5380753ea1270ef0a
    supports: [capability.weekly-payments, capability.close-reopen]
  - path: backend/hr/types/legal_parameters.go
    role: data-model
    hash: sha256:be11e0dddc5dfcf924bef4556f55326c166d7e1ad8bdb399caf26903165decd6
    supports: [capability.payslip-calculation]
  - path: backend/finance/types/cash_movement_account.go
    role: shared-domain
    hash: sha256:2c462a5021a979a7ce37405eb667f99e784b699f75e024c419c184e239a93eba
    supports: [capability.pay-net, capability.pay-contributions, capability.weekly-payments]
  - path: backend/finance/cash_flow_commitments.go
    role: shared-domain
    hash: sha256:642f494dd1491665739525d2db42e59621e75966dc902a9652b58346da0aa7f0
    supports: [capability.close-reopen, related-pages]
```
