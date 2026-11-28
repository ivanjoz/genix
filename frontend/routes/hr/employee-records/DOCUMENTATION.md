---
schema: 1
page_id: hr.employee-records
route: /hr/employee-records
title: Employee Records (Registros Empleados)
status: implemented
visibility: tenant
description_en: >-
  Per worker on payroll, register what happened in the month and the payroll pays or deducts:
  overtime 25% and 35%, unpaid absence days, and amounts of company concepts such as bonuses or
  extraordinary payments. Grant staff loans from a cash or bank account, split into monthly
  installments the payslips recover, and move a pending installment to another month.
description_es: >-
  Por cada trabajador en planilla, registro de lo ocurrido en el mes que la planilla paga o
  descuenta: horas extras 25% y 35%, días de inasistencia y montos de conceptos de la empresa como
  bonos o pagos extraordinarios. Préstamos al personal entregados desde una caja o banco, divididos
  en cuotas mensuales que las boletas descuentan, y traslado de una cuota pendiente a otro mes.
---

# Employee Records (Registros Empleados)

<!-- DOC-ID: page-purpose -->
## Page purpose

Employee Records (`Registros Empleados`, `novedades`) holds, per worker, what happened in a given
month that the payroll must pay or deduct: overtime (`horas extras`), unpaid absences
(`inasistencias`) and amounts of the company's own concepts (`bonos`, `pagos extraordinarios`,
other deductions). It also holds the loans the company grants to its workers (`préstamos al
personal`) and their monthly installments (`cuotas`).

The page does not calculate the payroll; opening Payroll (`Planilla`) recalculates every open month
whose records changed. Recurring monthly amounts are fixed concepts in Employees & Contracts,
vacations and leaves are in Time Off, and weekly or biweekly advances are in Payroll → Weekly
payments. Bank loans the company itself owes are in Finance → Loans (`Préstamos`), not here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **record** (`registro`) has the **Date (Fecha)** it happened and the **Payroll month (Mes de
  planilla)** that pays it, which is normally the date's month and can be a later month of the same
  year for a late report.
- **Overtime 25% (Horas extras 25%)** and **Overtime 35% (Horas extras 35%)** are registered in
  hours; the 25% hours are the first two overtime hours of a day.
- **Unpaid absence (days) (Inasistencia (días))** reduces the paid days of the month.
- A **company concept** record (`concepto de la empresa`) is an amount in soles of a concept created
  in HR Settings: an earning or a deduction. A concept marked to be paid on a separate payslip is an
  extraordinary payment (`pago extraordinario`).
- A record is **Open (Abierto)** until a closed payroll uses it; then it is **Settled
  (Liquidado)**.
- A **staff loan** (`préstamo al personal`) is money handed to the worker and recovered from his
  payslips in equal monthly installments; an installment is **Pending (Pendiente)** or **Recovered
  (Descontada)**.

<!-- DOC-ID: capability.find-employee -->
## Open a worker's records (Elegir trabajador)

### User intention (Intención del usuario)

Find the worker whose month or loans need to be registered.

### Where to find it (Dónde encontrarlo)

**Company (Empresa) → HR (RRHH) → Employee Records (Registros Empleados)** at
`/hr/employee-records`. The list shows **ID**, **Name (Nombre)**, **Document (Documento)** and
**Status (Estado)**, searchable by name or document; **Show terminated (Mostrar cesados)** adds
terminated workers. Clicking a worker opens a side panel with the tabs **Records (Registros)** and
**Loans (Préstamos)**.

### Business rules and rationale (Reglas y razón de negocio)

Only workers engaged on **Planilla** are listed, because the records feed the payroll. Terminated
workers stay reachable because they can still have a pending installment.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué no aparece un trabajador? → he is engaged by fee receipt or as temporary off payroll, or
  he is terminated and **Show terminated** is not ticked.

<!-- DOC-ID: capability.register-entry -->
## Register overtime, absences and monthly amounts (Registrar novedades)

### User intention (Intención del usuario)

Tell the payroll about this month's overtime, unpaid absences, a bonus, a commission or another
one-off amount for one worker.

### Where to find it (Dónde encontrarlo)

Worker panel → tab **Records (Registros)** → **Add (Agregar)** opens **New record (Nuevo
registro)**. Clicking an Open record opens **Edit record (Editar registro)**, which can also delete
it. The table shows **Date (Fecha)**, **Month (Mes)**, **Concept (Concepto)**, **Detail
(Detalle)**, **Note (Nota)** and **Status (Estado)**.

### Required information and prerequisites (Requisitos previos)

- **Date (Fecha)**, **Payroll month (Mes de planilla)** and **Concept (Concepto)**: the overtime and
  absence concepts first, then the company concepts from HR Settings.
- The concept decides the input: **Hours (Horas)** for overtime, **Days (Días)** for an absence,
  **Amount (S/) (Monto (S/))** for a company concept. **Note (Nota)** is optional.
- Changing the date moves the payroll month to the date's month; the user can then pick another
  month of that year.

### Business rules and rationale (Reglas y razón de negocio)

- The payroll month must belong to the same year as the date.
- Absence days are a whole number from 1 to 31; overtime hours are above 0 and at most 200; a
  company concept amount must be above zero.
- The months the record touched before and after the change must not have a closed payroll.
- A record can only move between concepts of this same screen.

### Result and side effects (Resultado y efectos)

The payroll month's payslip pays the overtime on the hourly value of salary plus family allowance,
removes the absence days from the paid days, and adds the company concept as an earning or a
deduction with its treatment. An extraordinary-payment concept is calculated with its own pension,
5ta and EsSalud. When that month's payroll is closed, its records become **Settled (Liquidado)**.

### Limitations (Limitaciones)

- A Settled record cannot be opened or changed; reopen the month in Payroll first.
- The payroll is not recalculated on this page; open Payroll to update the month.
- Vacations, medical leave and unpaid leave are registered in Time Off, not as absences here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro horas extras? ¿Dónde descuento una falta? ¿Cómo le pago un bono este mes?
- ¿Cómo registro una comisión o un pago extraordinario? ¿Puedo pagar en noviembre unas horas extras
  de octubre?
- Novedades, faltas, tardanzas en días, sobretiempo, bono, comisión.

<!-- DOC-ID: capability.grant-loan -->
## Grant a staff loan (Otorgar un préstamo)

### User intention (Intención del usuario)

Lend money to a worker and recover it from his payslips in monthly installments.

### Where to find it (Dónde encontrarlo)

Worker panel → tab **Loans (Préstamos)** → the green **+** button opens **New loan (Nuevo
préstamo)**. The loans table shows **Amount (Monto)**, **Installments (Cuotas)**, **First
installment (Primera cuota)** and **Note (Nota)**; below it, **Installments (Cuotas)** lists each
installment's **Month (Mes)**, **Amount (Monto)** and **Status (Estado)**.

### Required information and prerequisites (Requisitos previos)

**Amount (S/) (Monto (S/))**, **Installments (Cuotas)**, **Grant date (Fecha del préstamo)**,
**First installment (Primera cuota)** (a year before to a year after the current month), **Handed
out from (Entregado desde)** (an active cash register or bank account in soles) and an optional
**Note (Nota)**.

### Business rules and rationale (Reglas y razón de negocio)

- Between 1 and 60 installments, as a whole number.
- The first installment cannot be in a month before the grant date's month.
- None of the installment months can have a closed payroll.
- The installments are equal monthly amounts; the last one absorbs the rounding. The loan carries no
  interest field.
- The account must be in soles.

### Result and side effects (Resultado y efectos)

The loan and all its installments are saved together, and a **Pago Planilla** cash movement for the
whole amount leaves the chosen account on the grant date against staff loans (1411). Each month's
payslip deducts that month's installment as **Loan deduction (Descuento de préstamo)**; closing the
month marks it **Recovered (Descontada)**. If the worker is terminated, his settlement recovers the
installments due after the termination month.

### Limitations (Limitaciones)

A granted loan is never edited: delete it and grant it again, or move an installment.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo le presto dinero a un trabajador? ¿Cómo se descuenta el préstamo en la boleta? ¿Puedo
  cobrar intereses?
- Préstamo al personal, adelanto grande, descuento por planilla, cuotas.

<!-- DOC-ID: capability.manage-loan -->
## Move an installment or delete a loan (Mover cuota o eliminar préstamo)

### User intention (Intención del usuario)

Postpone one month's installment (for example a month with a low net pay), or remove a loan
registered by mistake.

### Where to find it (Dónde encontrarlo)

- Click a **Pending (Pendiente)** installment: **Move installment (Mover cuota)** asks **Recovered in
  (Se descuenta en)**.
- Click a loan row: a confirmation **Delete loan (Eliminar préstamo)** appears.

### Business rules and rationale (Reglas y razón de negocio)

- Only a pending installment can move, only its month changes, and it cannot go before the loan's
  month. Neither the old nor the new month can have a closed payroll.
- A loan can only be deleted while none of its installments was recovered by a closed payroll, and
  while its installment months are open.

### Result and side effects (Resultado y efectos)

Deleting cancels the loan and its installments and returns the full amount to the account it left
from, dated on the day of the deletion.

### Limitations (Limitaciones)

The amount or count of installments cannot be changed; a recovered installment cannot be moved.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo postergo una cuota del préstamo? ¿Por qué no puedo eliminar el préstamo? → an installment
  was already deducted in a closed payroll.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"La planilla del periodo … ya está cerrada; reábrala para hacer cambios":** reopen that month in
  Payroll, then register the change.
- **"El mes de planilla debe ser del mismo año que la fecha del registro":** choose a month of the
  date's year.
- **"Los días de inasistencia deben ser un número entero entre 1 y 31" / "Las horas extras deben
  estar entre 0 y 200":** correct the quantity.
- **"El registro ya fue usado por una planilla cerrada":** the record is settled; reopen the month.
- **"Indique el monto y entre 1 y 60 cuotas":** check the loan amount and installments.
- **"La primera cuota no puede ser anterior al mes del préstamo":** choose a later first month.
- **"Ya se descontaron cuotas del préstamo: no se puede eliminar":** the loan stays; move the
  remaining installments instead.
- **"La planilla se paga en soles: elija una caja o cuenta en soles":** pick an account in soles.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Payroll (Planilla):** recalculates the month with these records; its **Weekly payments (Pagos
  semanales)** tab registers advances.
- **Employees & Contracts (Empleados & Contratos):** fixed monthly concepts and contracts.
- **Time Off (Vacaciones y Licencias):** vacations, medical leave and leaves.
- **HR Settings (Configuración RRHH):** creates the company concepts offered here.
- **Cash & Banks (Cajas & Bancos):** the accounts loans are handed out from.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.find-employee]
  - path: frontend/routes/hr/employee-records/+page.svelte
    role: page
    hash: sha256:e71d0cf235a9bfd55e262bc7eb506f2cc0d085a7ca2de7d81180faaf5a4f4fa7
    supports: [page-purpose, capability.find-employee]
  - path: frontend/routes/hr/employee-records/EmployeeEntries.svelte
    role: user-interface
    hash: sha256:f3634d6d0fea1463863f727ffea59b9462fb0b8b83b3cfd3f4a5321f0b08fcb9
    supports: [concepts, capability.register-entry]
  - path: frontend/routes/hr/employee-records/EmployeeLoans.svelte
    role: user-interface
    hash: sha256:7102404fcc38fe2c584db826d95029ef2c1bac023cc9ccf4bd8e04e9be61ad92
    supports: [concepts, capability.grant-loan, capability.manage-loan]
  - path: frontend/routes/hr/employee-records/employee-records.ts
    role: business-logic
    hash: sha256:12f9355646e7889d7b23da8496e7b1120faf5007babceb3a3d334da2d8b53639
    supports: [concepts, capability.register-entry, capability.grant-loan]
  - path: frontend/routes/hr/employees/employees.ts
    role: business-logic
    hash: sha256:590c1af5be318be15985ecee8eba31897d03d23728093762326e929cf1748672
    supports: [capability.find-employee]
  - path: frontend/routes/hr/payroll/payroll.ts
    role: business-logic
    hash: sha256:58c3885b397125a11a9b2922a8cabd0023f1f0e74e39b94935fac0037090e15e
    supports: [capability.grant-loan]
  - path: frontend/services/hr/employee-records.svelte.ts
    role: frontend-service
    hash: sha256:597e465444a4ee69165e90378ddd5de17a447d61f2f303c6ace78716944bf460
    supports: [capability.register-entry, capability.grant-loan, capability.manage-loan]
  - path: frontend/services/hr/employee-records.ts
    role: frontend-service
    hash: sha256:26b3c9022220b0288e8bb9c9afe47009dee0270373dc649d37b75ee67906fd71
    supports: [concepts]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts, capability.register-entry]
  - path: backend/hr/employee_records.go
    role: backend-handler
    hash: sha256:74522e025c98f021b1005c567fa08e4b81aa124fc89b8c8940c974c418106bbd
    supports: [capability.register-entry, capability.grant-loan, capability.manage-loan, troubleshooting]
  - path: backend/hr/payroll_cash.go
    role: backend-handler
    hash: sha256:b72e8c59a1c51b961c2229cd7e1e20391e26bf5b70c30fef746abc16777c1e13
    supports: [capability.grant-loan, capability.manage-loan, troubleshooting]
  - path: backend/hr/payroll_runs.go
    role: backend-handler
    hash: sha256:5843b5ec5a3f0f5b719e4bac6900382b026471d763631fd7fee37684db901722
    supports: [concepts, capability.register-entry, capability.grant-loan]
  - path: backend/hr/types/employee_records.go
    role: data-model
    hash: sha256:e5bea6740b8ae271994c455035ae734b86bbadcc66cfa8f5380753ea1270ef0a
    supports: [concepts, capability.grant-loan]
  - path: backend/hr/types/legal_concepts.go
    role: business-logic
    hash: sha256:0ec0c2c15ba57d378f4d6243a80fb1d3e90c54325354aea02501032fb76e38fd
    supports: [concepts, capability.register-entry, capability.grant-loan]
  - path: backend/hr/types/payroll_engine.go
    role: business-logic
    hash: sha256:781443cdd93f5953057a93225c36b9fd1bf2e27efdfae924c887ced25c11d2b7
    supports: [capability.register-entry, capability.grant-loan]
  - path: backend/hr/types/payroll_benefits.go
    role: business-logic
    hash: sha256:8c4b8f6432edf7193dae2949986696057cb3c190c5a342afb273990cc4fc669b
    supports: [capability.register-entry, capability.grant-loan]
  - path: backend/finance/types/cash_movement_account.go
    role: shared-domain
    hash: sha256:2c462a5021a979a7ce37405eb667f99e784b699f75e024c419c184e239a93eba
    supports: [capability.grant-loan]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: shared-domain
    hash: sha256:051c429f6af7fbc3031301c0b052febfdbd06b5b71a689d695cedbc538e4a1c7
    supports: [capability.grant-loan]
```
