---
schema: 1
page_id: hr.employees
route: /hr/employees
title: Employees & Contracts (Empleados & Contratos)
status: implemented
visibility: tenant
description_en: >-
  Register the company's workers with their identity, department, position, immediate boss,
  engagement type (Planilla, fee receipt or temporary off payroll), hire and termination. Keep each
  worker's dated contract or agreement versions with pay, hours, pension and bank accounts, his
  dependents (which decide the family allowance) and his fixed monthly concepts.
description_es: >-
  Registro de los trabajadores con su documento, área, cargo, jefe inmediato, tipo de vínculo
  (planilla, recibo por honorarios o temporal sin planilla), ingreso y cese. Versiones fechadas de su
  contrato o acuerdo con pago, horas, pensión y cuentas bancarias, sus derechohabientes (que deciden
  la asignación familiar) y sus conceptos fijos mensuales.
---

# Employees & Contracts (Empleados & Contratos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Employees & Contracts (`Empleados & Contratos`, `personal`, `trabajadores`) is the register of the
people who work for the company: their identity, where they sit (department, position, immediate
boss), how they are engaged (`vínculo`), when they started and when they left (`cese`). For each
worker it also keeps the contract or agreement versions (`contratos y acuerdos`) that carry the pay,
the dependents (`derechohabientes`) and the fixed monthly concepts (`conceptos fijos`).

The page does not calculate pay: Payroll (`Planilla`) reads these data. Month-to-month facts such
as overtime, absences, bonuses of the month and staff loans are registered in Employee Records
(`Registros Empleados`), and vacations or leaves in Time Off (`Vacaciones y Licencias`).
Departments and positions are created in Departments & Positions (`Áreas y Cargos`).

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Engagement (Vínculo)** says how the worker is engaged: **Payroll (Planilla)**, **Fee receipt
  (Recibo por honorarios)** or **Temporary, off payroll (Temporal sin planilla)**. Only Planilla
  workers enter the payroll, payslips, time off, employee records and weekly payments. Workers off
  payroll are registered so they can be chosen when production hours are logged.
- A **contract or agreement version** (`versión de contrato o acuerdo`) holds the terms from one
  date (**Valid from (Vigente desde)**). Each raise or change is a new version; the one in force on
  each date applies. A Planilla worker has a contract; a worker off payroll has an agreement without
  payroll fields.
- A **dependent** (`derechohabiente`) is the worker's spouse, cohabitant or child. Children decide
  the family allowance (`asignación familiar`).
- A **fixed concept** (`concepto fijo`) is a company concept paid or deducted every month of a range
  of months, such as a fixed bonus, `movilidad` or a judicial withholding.
- **Status (Estado)** is **Active (Activo)** or **Terminated (Cesado)**. A worker is Terminated as
  soon as a termination date is saved; his records stay.

<!-- DOC-ID: capability.register-employee -->
## Register or edit a worker (Registrar un trabajador)

### User intention (Intención del usuario)

Add a new hire, or correct a worker's personal data, department, position, boss or engagement.

### Where to find it (Dónde encontrarlo)

**Company (Empresa) → HR (RRHH) → Employees & Contracts (Empleados & Contratos)** at
`/hr/employees`. The green **+** button opens **New employee (Nuevo trabajador)**; clicking a row
opens the worker. Tab **Information (Información)**, saved with the panel's save button. The list
can be searched by name, document or email and filtered by **Engagement (Vínculo)**.

### Required information and prerequisites (Requisitos previos)

- **Identity document (Documento de identidad)** with its type (DNI by default; RUC is not offered),
  **First names (Nombres)**, **Paternal surname (Apellido paterno)**, **Birth date (Fecha de
  nacimiento)**, **Sex (Sexo)**, **Department (Área)**, **Engagement (Vínculo)** and **Hire date
  (Fecha de ingreso)** are required.
- Optional: **Maternal surname (Apellido materno)**, **Phone (Teléfono)**, **Email**, **Address
  (Dirección)**, **Position (Cargo)**, **Immediate boss (Jefe inmediato)**.
- The department must exist in Departments & Positions. **Position** only lists the positions of
  the chosen department, and changing the department clears it.

### Business rules and rationale (Reglas y razón de negocio)

- A DNI has exactly 8 digits; any document has at least 6 characters.
- Sex is required because T-Registro requires it.
- The hire date must be after the birth date.
- The document number cannot belong to another registered worker of the company.
- The position must belong to the selected department and not be deleted.
- The immediate boss must be an active worker, not the worker himself, and must not already report
  to him directly or through others, which would close a loop in the org chart.

### Result and side effects (Resultado y efectos)

The worker appears in the list with **ID**, **Name (Nombre)**, **Document (Documento)**,
**Department (Área)**, **Position (Cargo)**, **Engagement (Vínculo)**, **Hire date (Ingreso)**,
**Dependents (Derechohabientes)** and **Status (Estado)**. Once saved, the tabs **Contracts and
agreements (Contratos y acuerdos)**, **Dependents (Derechohabientes)** and **Fixed concepts
(Conceptos fijos)** become available. The immediate boss draws the org chart in Departments &
Positions.

### Limitations (Limitaciones)

- A worker is not paid until he has a contract version (Planilla) in force.
- Changing a worker's data does not change a closed payroll; open months are recalculated the next
  time Payroll is opened.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro un nuevo trabajador? ¿Cómo doy de alta a un empleado? ¿Dónde pongo el jefe
  inmediato?
- ¿Por qué no aparece el cargo? → choose the department first, or create the position there.
- Alta de personal, colaborador, empleado, obrero, ficha del trabajador.

<!-- DOC-ID: capability.terminate-employee -->
## Record a termination or delete a worker (Cese o eliminación)

### User intention (Intención del usuario)

Record that a worker left the company, or remove a worker registered by mistake.

### Where to find it (Dónde encontrarlo)

Tab **Information (Información)**: **Termination date (Fecha de cese)** and **Termination reason
(Motivo de cese)**, then save. Deleting uses the panel's delete button. **Show terminated (Mostrar
cesados)** shows terminated workers in the list.

### Required information and prerequisites (Requisitos previos)

Reasons: **Resignation (Renuncia)**, **Dismissal (Despido)**, **Retirement (Jubilación)**, **Death
(Fallecimiento)**, **End of contract (Término de contrato)**, **Mutual agreement (Mutuo disenso)**.

### Business rules and rationale (Reglas y razón de negocio)

- Termination date and reason go together: one without the other is refused.
- The termination date cannot be before the hire date.
- The delete confirmation recommends using the termination date to record a departure. A
  terminated worker stays in Genix because his payslips, contracts and profit sharing still refer
  to him; a deleted one leaves the list.

### Result and side effects (Resultado y efectos)

The worker becomes **Terminated (Cesado)**. The payroll of the termination month adds his
settlement (`liquidación`); with reason **Dismissal (Despido)** it includes the dismissal
indemnity. A terminated worker can no longer be chosen as immediate boss.

### Limitations (Limitaciones)

A terminated worker is hidden from the list unless **Show terminated (Mostrar cesados)** is ticked.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro la renuncia de un trabajador? ¿Cómo doy de baja a un empleado? ¿Cesar o eliminar?
- Cese, baja, despido, renuncia, liquidación.

<!-- DOC-ID: capability.contracts -->
## Keep contract and agreement versions (Contratos y acuerdos)

### User intention (Intención del usuario)

Record the worker's pay, hours, contract type, pension system and bank accounts, and each raise or
change from the date it applies.

### Where to find it (Dónde encontrarlo)

Open a saved worker → tab **Contracts and agreements (Contratos y acuerdos)**. **New version (Nueva
versión)** opens a version prefilled from the one in force; clicking a row edits it. The table shows
**Valid from (Vigente desde)** (✔ marks the version in force today), **Type (Tipo)**, **Pay
(Pago)**, **Pay day (Día de pago)**, **Hours (Horas)** and **Pension (Pensión)**.

### Required information and prerequisites (Requisitos previos)

- **Valid from (Vigente desde)**: empty means the start date.
- **Start date (Fecha de inicio)**, **End date (Fecha de fin)**, **Daily hours (Horas diarias)**,
  and the pay: **Monthly pay (S/) (Pago mensual (S/))**, **Weekly pay (S/) (Pago semanal (S/))** or
  **Hourly pay (S/) (Pago por hora (S/))** by pay basis.
- Planilla only: **Contract type (Tipo de contrato)** (Plazo indeterminado, Tiempo parcial, Inicio
  o incremento de actividad, Necesidades del mercado, Suplencia, Obra determinada o servicio
  específico, Intermitente, De temporada), **Pension system (Sistema de pensiones)** (ONP, AFP
  Habitat, AFP Integra, AFP Prima, AFP Profuturo), **Has EPS (Tiene EPS)**, and for an AFP the
  **CUSPP** and **AFP commission (Comisión AFP)**: **On flow (Sobre el flujo)** or **Mixed
  (Mixta)**; **CTS bank (Banco CTS)** and **CTS account (Cuenta CTS)**.
- Temporary off payroll only: **Pay basis (Base de pago)**: **Monthly (Mensual)**, **Weekly
  (Semanal)** or **Hourly (Por hora)**; weekly also asks **Pay day (Día de pago)**.
- Optional for everyone: **Pay bank (Banco de pago)** and **Pay account (Cuenta de pago)**.

### Business rules and rationale (Reglas y razón de negocio)

- A Planilla contract is always monthly; a fee-receipt agreement is always hourly.
- The start date cannot be before the worker's hire date.
- Only an indefinite contract (`plazo indeterminado`) has no end date; a fixed-term contract needs an
  end date after the start.
- **Valid from** must fall within the version's start and end dates, and only one version per
  worker can start on the same date.
- Pay must be above zero. Daily hours go from 1 to 12, or 0 to 12 for hourly pay.
- An AFP needs a 12-character CUSPP and the commission type; ONP clears them.
- A bank is optional (cash payment); with a bank the account needs at least 8 characters.
- Fields that do not apply to the engagement are cleared on save.

### Result and side effects (Resultado y efectos)

Payroll uses the version in force on the worker's last employed day of each month. Under 4 daily
hours the worker is part-time: no CTS and 6 vacation days a year. Deleting a version asks for
confirmation: from its date the previous version applies again. The list column **Pay day (Día de
pago)** shows the weekly pay day of the version in force.

### Limitations (Limitaciones)

- This tab, **Fixed concepts (Conceptos fijos)** and the **Pay day** column are only shown to users
  with the **Planilla** access, because contracts carry salaries.
- A change does not alter a closed payroll; reopen that month in Payroll to include it.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro un aumento de sueldo? ¿Dónde cambio la AFP de un trabajador? ¿Cómo registro un
  contrato a plazo fijo? ¿Dónde pongo la cuenta CTS?
- Contrato, renovación, adenda, sueldo básico, remuneración, CUSPP, ONP, AFP, EPS, cuenta sueldo.

<!-- DOC-ID: capability.dependents -->
## Register dependents and the family allowance (Derechohabientes)

### User intention (Intención del usuario)

Record the worker's family members, and so give him the family allowance (`asignación familiar`)
when a child entitles him.

### Where to find it (Dónde encontrarlo)

Open a saved worker → tab **Dependents (Derechohabientes)** → **Add (Agregar)**; click a row to
edit. The header says whether the worker earns the family allowance today.

### Required information and prerequisites (Requisitos previos)

**Relationship (Vínculo)** (**Spouse (Cónyuge)**, **Cohabitant (Concubino(a))**, **Child
(Hijo(a))**), **Identity document (Documento de identidad)**, **First names (Nombres)**, **Paternal
surname (Apellido paterno)**, **Registered from (Alta)**; optional **Maternal surname (Apellido
materno)** and **Removed on (Baja)**. A child also needs **Birth date (Fecha de nacimiento)** and can
be marked **Studying (counts up to 24) (Estudia (cuenta hasta los 24))**.

### Business rules and rationale (Reglas y razón de negocio)

- A child gives the right to the family allowance while under 18, or under 24 while studying,
  within the days between **Alta** and **Baja**.
- The removal date cannot be before the registration date.
- To record that a dependent no longer counts, set **Baja** instead of deleting.

### Result and side effects (Resultado y efectos)

Genix rewrites the worker's dependents count and family allowance days on every save. The list
column **Dependents (Derechohabientes)** shows the count and `Asig. fam.` when it applies today.
Payroll adds the **Family allowance (Asignación familiar)**, a percentage of the RMV, in the months
where a child entitles the worker on the month's last day.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo le doy asignación familiar a un trabajador? ¿Por qué ya no cobra asignación familiar?
  → the child turned 18 (or 24 if studying) or has a **Baja** date.
- Carga familiar, hijos, esposa, conviviente.

<!-- DOC-ID: capability.fixed-concepts -->
## Assign fixed monthly concepts (Conceptos fijos)

### User intention (Intención del usuario)

Pay or deduct a company concept every month of a range, such as a fixed bonus, transport allowance
(`movilidad`) or a judicial withholding.

### Where to find it (Dónde encontrarlo)

Open a saved worker → tab **Fixed concepts (Conceptos fijos)** → the green **+** button; click a row
to edit or delete. Columns: **Concept (Concepto)**, **Monthly (Mensual)**, **From (Desde)**, **To
(Hasta)**.

### Required information and prerequisites (Requisitos previos)

**Concept (Concepto)** (a company concept from HR Settings), **Monthly amount (S/) (Monto mensual
(S/))**, **From (Desde)** and optional **To (optional) (Hasta (opcional))**, chosen from a year
before to a year after the current month; and an optional **Note (Nota)**.

### Business rules and rationale (Reglas y razón de negocio)

- The amount must be above zero.
- The final month cannot be before the first; without it the concept has no end.
- To end it, set **To** rather than deleting, so the months already paid keep it.

### Result and side effects (Resultado y efectos)

Every monthly payslip within the range adds the concept as an earning or a deduction, with the
treatment defined for it in HR Settings. A company concept assigned here cannot be deleted in HR
Settings until it is removed from the workers.

### Limitations (Limitaciones)

Only company concepts are offered; legal concepts (salary, AFP, EsSalud…) are calculated by the
payroll. A one-off amount of a single month belongs in Employee Records, not here.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo le pongo un bono fijo mensual? ¿Dónde registro una retención judicial? ¿Cómo asigno
  movilidad?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"el DNI debe tener 8 dígitos":** check the number or choose another document type.
- **"el documento … ya está registrado para …":** the worker already exists; search him, including
  terminated ones.
- **"el cargo … no pertenece al área seleccionada":** pick a position of that department.
- **"el jefe inmediato elegido ya depende de este trabajador: se formaría un ciclo":** choose
  another boss.
- **"la fecha de cese no puede ser anterior a la de ingreso" / "seleccione el motivo del cese":**
  fill both termination fields consistently.
- **"un contrato a plazo fijo necesita una fecha de fin posterior a la de inicio":** add the end
  date or choose an indefinite contract.
- **"la vigencia de esta versión debe caer dentro de sus fechas":** move **Valid from** inside the
  start and end dates.
- **"ya existe una versión vigente desde esa fecha":** edit that version instead.
- **"el CUSPP de la AFP debe tener 12 caracteres":** type the full CUSPP.
- **No Contracts or Fixed concepts tab:** the worker is not saved yet, or the user lacks the
  Planilla access.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Departments & Positions (Áreas y Cargos):** create departments and positions; see the org chart.
- **Payroll (Planilla):** calculates the payslips from these workers and contracts.
- **Employee Records (Registros Empleados):** overtime, absences, monthly amounts and staff loans.
- **Time Off (Vacaciones y Licencias):** vacations and leaves.
- **HR Settings (Configuración RRHH):** company concepts used as fixed concepts.
- **Hours vs payroll (Horas vs planilla):** production hours of workers on and off payroll.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.register-employee]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [capability.contracts]
  - path: frontend/routes/hr/employees/+page.svelte
    role: page
    hash: sha256:6268d3c42fad85b9d8d15e4842c66573bd6f0d4da08535d6ede5c25514db8ad5
    supports: [page-purpose, capability.contracts]
  - path: frontend/routes/hr/employees/EmployeesTab.svelte
    role: user-interface
    hash: sha256:144a74c35afe8c39abed7fbc6be202e8f3b7d25d665a8505982407c648169a5b
    supports: [concepts, capability.register-employee, capability.terminate-employee, capability.contracts, troubleshooting]
  - path: frontend/routes/hr/employees/EmployeeContractsAgreements.svelte
    role: user-interface
    hash: sha256:4dba0ced3aeaf024fbdbb6e2cfe5ca5c9e3a19247ccb2edd9019d55d7f9c0a3f
    supports: [capability.contracts]
  - path: frontend/routes/hr/employees/EmployeeDependents.svelte
    role: user-interface
    hash: sha256:79025e8145056e3b3d7be819ef7b68418be23c013d3f6438ebd474b6e8321eca
    supports: [capability.dependents]
  - path: frontend/routes/hr/employees/EmployeeFixedConcepts.svelte
    role: user-interface
    hash: sha256:cf6c3676cd55d55b460f6894e141a06fd6e16d3907b64b8f1f239a58f1a366e9
    supports: [concepts, capability.fixed-concepts]
  - path: frontend/routes/hr/employees/employees.ts
    role: business-logic
    hash: sha256:590c1af5be318be15985ecee8eba31897d03d23728093762326e929cf1748672
    supports: [concepts, capability.register-employee, capability.contracts, capability.dependents]
  - path: frontend/routes/hr/payroll/payroll.ts
    role: business-logic
    hash: sha256:58c3885b397125a11a9b2922a8cabd0023f1f0e74e39b94935fac0037090e15e
    supports: [capability.fixed-concepts]
  - path: frontend/services/hr/employees.svelte.ts
    role: frontend-service
    hash: sha256:0f862bfe15e87e3fea8735ed72bbafedc435bda643d3418d5b85a0966cdc47a5
    supports: [capability.register-employee, capability.dependents]
  - path: frontend/services/hr/contracts-agreements.svelte.ts
    role: frontend-service
    hash: sha256:37665e9e9897ff0f911559139bc37cb3fbd80048267cd6d8e54a6ef936877c84
    supports: [capability.contracts]
  - path: frontend/services/hr/payroll.svelte.ts
    role: frontend-service
    hash: sha256:1a876ca2e211cb45cf7a27a985305a1fe03e87c23a6cdca8b8b411699e110b11
    supports: [capability.fixed-concepts]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts, capability.terminate-employee, capability.contracts, capability.dependents]
  - path: frontend/services/crm/identity-doc.ts
    role: shared-domain
    hash: sha256:9923cb088e0f23088291bf802b2459822bcb8335e7277af8a8b72a34cd9da9a8
    supports: [capability.register-employee]
  - path: backend/hr/employees.go
    role: backend-handler
    hash: sha256:e2292f6919d51c2ca7e8f40359b46c6f9218b904b1a3f6217df919e7dc4205f9
    supports: [capability.register-employee, capability.terminate-employee, capability.dependents, troubleshooting]
  - path: backend/hr/contracts_agreements.go
    role: backend-handler
    hash: sha256:c97e27d7ffd6ff1b57880f0fef17b918e132a3d4bce80d2ef3ecac4fe3028e80
    supports: [capability.contracts, troubleshooting]
  - path: backend/hr/employee_records.go
    role: backend-handler
    hash: sha256:74522e025c98f021b1005c567fa08e4b81aa124fc89b8c8940c974c418106bbd
    supports: [capability.fixed-concepts]
  - path: backend/hr/concepts.go
    role: backend-handler
    hash: sha256:fb26b24d8d3794198b82e3b5dd5a9dca0398b5cb88e290334f824df1e7ebf48d
    supports: [capability.fixed-concepts]
  - path: backend/hr/types/employees.go
    role: data-model
    hash: sha256:1c8c7b02fb7bd78600cb14eebf0df454b37260ae14ba938185fa1acc56115a5c
    supports: [concepts, capability.terminate-employee, capability.dependents]
  - path: backend/hr/types/employee_concepts.go
    role: data-model
    hash: sha256:6e3631bd3e397da64fdc65c23c33b62538793643048d8cd011c4d0e7bf3e0a4b
    supports: [concepts, capability.fixed-concepts]
  - path: backend/hr/types/payroll_data.go
    role: business-logic
    hash: sha256:bae9967e9ae7ba1e9060dde44697664aa5e30226407bf8a0b105380e21833e95
    supports: [concepts]
  - path: backend/hr/types/payroll_engine.go
    role: business-logic
    hash: sha256:781443cdd93f5953057a93225c36b9fd1bf2e27efdfae924c887ced25c11d2b7
    supports: [capability.contracts, capability.dependents, capability.fixed-concepts]
  - path: backend/hr/types/payroll_benefits.go
    role: business-logic
    hash: sha256:8c4b8f6432edf7193dae2949986696057cb3c190c5a342afb273990cc4fc669b
    supports: [capability.contracts, capability.terminate-employee]
  - path: backend/hr/types/payroll_period.go
    role: business-logic
    hash: sha256:4c640ce7682313c78742b3b5e64d1b160a54cda288bb113c9026556893eaf569
    supports: [capability.terminate-employee]
```
