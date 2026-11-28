---
schema: 1
page_id: hr.time-off
route: /hr/time-off
title: Time Off (Vacaciones y Licencias)
status: implemented
visibility: tenant
description_en: >-
  Register the vacations, medical leaves, maternity and unpaid leaves of the workers on payroll, by
  date range, already approved. The monthly payroll pays vacation days as vacation pay and leaves
  unpaid or EsSalud-subsidized days out. A second tab shows each worker's vacation balance: days
  earned since hiring, days taken and days left.
description_es: >-
  Registro de vacaciones, descansos médicos, maternidad y licencias sin goce de los trabajadores en
  planilla, por rango de fechas y ya aprobados. La planilla mensual paga los días de vacaciones como
  remuneración vacacional y no paga los días sin goce o subsidiados por EsSalud. Otra pestaña muestra
  el saldo de vacaciones: días ganados desde el ingreso, gozados y pendientes.
---

# Time Off (Vacaciones y Licencias)

<!-- DOC-ID: page-purpose -->
## Page purpose

Time Off (`Vacaciones y Licencias`, `ausencias`) records when a worker on payroll is away for a
whole range of days: vacations (`vacaciones`), medical leave (`descanso médico`), maternity
(`maternidad`) and unpaid leave (`licencia sin goce`). The monthly payroll reads these ranges to
decide which days are paid as salary, as vacation pay, or not at all. The page also shows each
worker's vacation balance (`saldo de vacaciones`).

The page does not handle single unpaid absence days counted by number (`inasistencias`, `faltas`):
those are registered in Employee Records. It has no request and approval workflow: HR registers the
time off already approved.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Vacation (Vacaciones):** the days are paid as **Vacation pay (Remuneración vacacional)** instead
  of salary and count as vacation taken.
- **Medical leave (paid by the company) (Descanso médico (pagado por la empresa)):** the days keep
  being paid as salary.
- **Medical leave (EsSalud subsidy) (Descanso médico (subsidio EsSalud))**, **Maternity
  (Maternidad)** and **Unpaid leave (Licencia sin goce)**: the payroll does not pay those days;
  EsSalud pays the subsidies outside the payroll.
- **Days (Días)** are calendar days from **From (Desde)** to **To (included) (Hasta (incluido))**.
- The **vacation balance** is 30 days a year, earned day by day since the hire date, less the
  vacations registered.

<!-- DOC-ID: capability.register-time-off -->
## Register a vacation or leave (Registrar una ausencia)

### User intention (Intención del usuario)

Record that a worker takes vacation or is on leave between two dates, so the payroll pays it
correctly.

### Where to find it (Dónde encontrarlo)

**Company (Empresa) → HR (RRHH) → Time Off (Vacaciones y Licencias)** at `/hr/time-off`, tab **Time
off (Ausencias)**. **Register (Registrar)** opens **Register time off (Registrar ausencia)**. The
table lists the time off starting in the year chosen in **Starting in (Que empiezan en)**, with
**Employee (Trabajador)**, **Type (Tipo)**, **From (Desde)**, **To (Hasta)**, **Days (Días)** and
**Note (Nota)**.

### Required information and prerequisites (Requisitos previos)

**Employee (Trabajador)** (active workers engaged on Planilla), **Type (Tipo)**, **From (Desde)**,
**To (included) (Hasta (incluido))** and an optional **Note (Nota)**. The form opens with type
Vacation and today's date.

### Business rules and rationale (Reglas y razón de negocio)

- The end date cannot be before the start date.
- One time off covers less than a year; register a longer absence in several ranges.
- The dates must fall within the worker's employment, between hire and termination.
- A worker cannot have two time off records whose dates overlap.
- No month the range touches can have a closed payroll.
- Genix does not switch a paid medical leave to the EsSalud subsidy by itself: the chosen type
  decides who pays.

### Result and side effects (Resultado y efectos)

The record is saved as approved. Every open payroll month the range touches is recalculated the next
time Payroll is opened: vacation days become vacation pay, unpaid and subsidized days are removed
from the paid days. Vacation days reduce the balance. Closing a payroll does not settle time off, but
changes are refused while a month the range touches is closed.

### Limitations (Limitaciones)

Workers engaged by fee receipt or temporary off payroll are not offered, because time off belongs
to the payroll.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro las vacaciones de un trabajador? ¿Dónde pongo un descanso médico? ¿Cómo registro
  la licencia por maternidad?
- ¿Las vacaciones se pagan en la planilla? ¿Quién paga el descanso médico?
- Vacaciones, descanso médico, DM, incapacidad temporal, subsidio, licencia sin goce de haber,
  permiso, ausencia.

<!-- DOC-ID: capability.edit-time-off -->
## Correct or cancel a time off (Editar o anular una ausencia)

### User intention (Intención del usuario)

Change the dates or type of a time off, or cancel it because the worker came back or it was entered
by mistake.

### Where to find it (Dónde encontrarlo)

Click a row of the **Time off (Ausencias)** tab: **Edit time off (Editar ausencia)**. Save the
change, or use the delete button to cancel (**Cancel time off (Anular ausencia)**).

### Business rules and rationale (Reglas y razón de negocio)

- The worker cannot be changed; cancel and register again for another worker.
- The type can only change to another time off type.
- The same date rules apply, and both the old and the new months must not have a closed payroll.

### Result and side effects (Resultado y efectos)

Cancelling gives its days back to the payroll: the open months are recalculated as worked days. A
change of start year moves the record to that year's list.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo anulo unas vacaciones? ¿El trabajador regresó antes, cómo corrijo la fecha?

<!-- DOC-ID: capability.vacation-balance -->
## See vacation balances (Saldos de vacaciones)

### User intention (Intención del usuario)

Know how many vacation days each worker has earned, taken and still has pending.

### Where to find it (Dónde encontrarlo)

Tab **Vacation balances (Saldos de vacaciones)**: **Employee (Trabajador)**, **Hired (Ingreso)**,
**Earned (Ganados)**, **Taken (Gozados)** and **Balance (Saldo)** for every active Planilla worker.

### Business rules and rationale (Reglas y razón de negocio)

- Earned days are 30 per year of service, accrued day by day from the hire date on the payroll's
  30-day month and 360-day year.
- Taken days are the vacation days registered on this page.
- The page reads time off up to 10 years back.

### Limitations (Limitaciones)

- The balance is calculated, never stored or edited by hand.
- It always uses 30 days a year: it does not apply the 6 days a year of a part-time worker (under 4
  daily hours), which the payroll does apply.
- When a worker leaves, the settlement in Payroll pays the vacation earned and not taken.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuántos días de vacaciones le quedan a un trabajador? ¿Cómo se calculan las vacaciones
  pendientes? ¿Vacaciones truncas?
- Récord vacacional, vacaciones pendientes, días gozados.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Se cruza con otra ausencia registrada del trabajador":** the dates overlap another time off;
  correct or cancel that one.
- **"Las fechas deben estar dentro del periodo laboral del trabajador":** check the hire and
  termination dates in Employees & Contracts.
- **"Registre una ausencia de más de un año en varios tramos":** split the range.
- **"La planilla del periodo … ya está cerrada; reábrala para hacer cambios":** reopen that month in
  Payroll first.
- **A worker is missing from the list:** he is not engaged on Planilla or is terminated.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Payroll (Planilla):** pays vacation days and leaves unpaid days out; its settlement pays pending
  vacation.
- **Employee Records (Registros Empleados):** single absence days, overtime and monthly amounts.
- **Employees & Contracts (Empleados & Contratos):** hire and termination dates and the engagement.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.register-time-off]
  - path: frontend/routes/hr/time-off/+page.svelte
    role: page
    hash: sha256:28d7155faf6fee495a1d690249731962dc66bb8e6a1b1c9c9e010c4cf53f6952
    supports: [page-purpose, capability.register-time-off, capability.edit-time-off, capability.vacation-balance]
  - path: frontend/routes/hr/time-off/time-off.ts
    role: business-logic
    hash: sha256:0075f0e0f091c82cb5c9b3892794a2c8665517811bd53216418642aa64b7521a
    supports: [concepts, capability.vacation-balance]
  - path: frontend/routes/hr/employees/employees.ts
    role: business-logic
    hash: sha256:590c1af5be318be15985ecee8eba31897d03d23728093762326e929cf1748672
    supports: [capability.register-time-off]
  - path: frontend/services/hr/employee-records.svelte.ts
    role: frontend-service
    hash: sha256:597e465444a4ee69165e90378ddd5de17a447d61f2f303c6ace78716944bf460
    supports: [capability.register-time-off, capability.edit-time-off]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts]
  - path: backend/hr/employee_records.go
    role: backend-handler
    hash: sha256:74522e025c98f021b1005c567fa08e4b81aa124fc89b8c8940c974c418106bbd
    supports: [capability.register-time-off, capability.edit-time-off, troubleshooting]
  - path: backend/hr/payroll_runs.go
    role: backend-handler
    hash: sha256:5843b5ec5a3f0f5b719e4bac6900382b026471d763631fd7fee37684db901722
    supports: [capability.register-time-off]
  - path: backend/hr/types/employee_records.go
    role: data-model
    hash: sha256:e5bea6740b8ae271994c455035ae734b86bbadcc66cfa8f5380753ea1270ef0a
    supports: [concepts, capability.register-time-off]
  - path: backend/hr/types/legal_concepts.go
    role: business-logic
    hash: sha256:0ec0c2c15ba57d378f4d6243a80fb1d3e90c54325354aea02501032fb76e38fd
    supports: [concepts]
  - path: backend/hr/types/payroll_engine.go
    role: business-logic
    hash: sha256:781443cdd93f5953057a93225c36b9fd1bf2e27efdfae924c887ced25c11d2b7
    supports: [concepts, capability.register-time-off, capability.vacation-balance]
  - path: backend/hr/types/payroll_benefits.go
    role: business-logic
    hash: sha256:8c4b8f6432edf7193dae2949986696057cb3c190c5a342afb273990cc4fc669b
    supports: [capability.vacation-balance]
```
