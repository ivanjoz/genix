---
schema: 1
page_id: hr.hr-settings
route: /hr/hr-settings
title: HR Settings (Configuración RRHH)
status: implemented
visibility: tenant
description_en: >-
  The payroll concepts: the legal ones Genix calculates, shown read only with their treatment, and
  the company's own earnings and deductions (bonuses, movilidad, judicial withholdings) with their
  PLAME code, contributions, 5th category treatment and accounting account. A second tab shows,
  read only, the legal values and AFP rates the payroll uses.
description_es: >-
  Los conceptos de planilla: los legales que Genix calcula, solo de lectura con su tratamiento, y los
  ingresos y descuentos propios de la empresa (bonos, movilidad, retenciones judiciales) con su
  código PLAME, aportes, tratamiento de quinta categoría y cuenta contable. Otra pestaña muestra, solo
  de lectura, los valores legales y las tasas AFP que usa la planilla.
---

# HR Settings (Configuración RRHH)

<!-- DOC-ID: page-purpose -->
## Page purpose

HR Settings (`Configuración RRHH`) defines the payroll concepts (`conceptos de planilla`) of the
company and shows the legal values the payroll is calculated with. The company creates its own
earnings and deductions here (a productivity bonus, `movilidad`, a judicial withholding) and decides
how each one is treated: whether it pays contributions, how it enters the 5th category income tax,
which PLAME code reports it and which account it posts to.

The page does not assign concepts to workers: fixed monthly amounts are set in Employees &
Contracts, one-off amounts in Employee Records. The legal values (RMV, UIT, rates, AFP table) are
only shown here; the SaaS operator maintains them.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Legal concepts** (`conceptos legales`, IDs 1–200) are defined by Genix: salary, family allowance,
  overtime, vacation pay, gratificación, CTS, AFP, ONP, 5ta, EsSalud, provisions and the records
  registered on workers (absences, advances, loans, time off). They cannot be edited.
- **Company concepts** (`conceptos de la empresa`, IDs from 201) are an **Earning (Ingreso)** or a
  **Deduction (Descuento)** created by the company.
- **Remunerative (Remunerativo)**: the earning counts as remuneration. Only a remunerative earning
  can **Pays EsSalud (Aporta a EsSalud)**, **Pays pension (Aporta a pensiones)** or be **CTS / grati
  base (Computable CTS / grati)**.
- **Income tax 5th cat. (Renta de 5ta)**: **Not taxed (Inafecto)**, **Ordinary (projected)
  (Ordinaria (proyectada))**, projected over the year, or **Extraordinary (Extraordinaria)**, taxed
  in the month it is paid.
- **Paid on a separate payslip (Se paga en boleta aparte)**: an extraordinary payment whose pension,
  5ta and EsSalud are calculated apart from the monthly salary's.

<!-- DOC-ID: capability.browse-concepts -->
## Browse the payroll concepts (Ver conceptos)

### User intention (Intención del usuario)

Check how a concept is treated: whether it pays EsSalud or pension, its PLAME code, its 5ta
treatment and its accounts.

### Where to find it (Dónde encontrarlo)

**Company (Empresa) → HR (RRHH) → HR Settings (Configuración RRHH)** at `/hr/hr-settings`, tab
**Concepts (Conceptos)**. The table shows **ID**, **Concept (Concepto)**, **Type (Tipo)**,
**PLAME**, **Remunerative (Remunerativo)**, **EsSalud**, **Pension (Pensión)**, **5ta**, **Account
(Cuenta)** and **Separate payslip (Boleta aparte)**. It can be searched by ID, name or PLAME code,
and the toggle **Legal (Legales)** / **Company's (De la empresa)** chooses which concepts show (at
least one).

### Business rules and rationale (Reglas y razón de negocio)

Clicking a legal concept opens its treatment read only, with the note that it is defined by the
system and cannot be edited.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿La gratificación aporta a EsSalud? ¿Qué cuenta usa la CTS? ¿Qué código PLAME tiene la
  asignación familiar?
- Concepto remunerativo, no remunerativo, afecto, inafecto.

<!-- DOC-ID: capability.manage-company-concept -->
## Create, edit or delete a company concept (Concepto de la empresa)

### User intention (Intención del usuario)

Create a bonus, allowance or deduction the payroll must pay or withhold, with the correct treatment.

### Where to find it (Dónde encontrarlo)

Tab **Concepts (Conceptos)** → **New (Nuevo)** opens **New concept (Nuevo concepto)**; clicking a
company concept opens it for editing or deleting.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre)**, **Type (Tipo)** (**Earning (Ingreso)** or **Deduction (Descuento)**) and
  **PLAME code (table 22) (Código PLAME (tabla 22))**, which only lists the codes of that type.
- Earning: **Income tax 5th cat. (Renta de 5ta)**, **Remunerative (Remunerativo)** and, when
  remunerative, **Pays EsSalud (Aporta a EsSalud)**, **Pays pension (Aporta a pensiones)** and **CTS
  / grati base (Computable CTS / grati)**; **Expense account (Cuenta de gasto)** from the personnel
  expense accounts (62); and **Paid on a separate payslip (extraordinary payment) (Se paga en boleta
  aparte (pago extraordinario))**.
- Deduction: **Owed to (account) (Cuenta por pagar)**, a detail account of 40, 41, 46 or 14.
- A new concept starts as a remunerative earning that pays EsSalud and pension, ordinary 5ta,
  account 622.

### Business rules and rationale (Reglas y razón de negocio)

- The name needs at least 2 characters.
- The PLAME code must exist in SUNAT table 22 and match the concept's type, because the PLAME file
  refuses other codes and a code of another chapter would report the line in the wrong column.
- A non-remunerative earning cannot pay EsSalud or pension or be computable for benefits; those
  options are cleared.
- An earning must post to a personnel expense account (62).
- A deduction needs its payable account (for example 419 for judicial withholdings), cannot be paid
  on a separate payslip, and carries no treatment of its own: it lowers the net pay, not the bases.
- A concept cannot be deleted while it is assigned to workers as a fixed concept.

### Result and side effects (Resultado y efectos)

The concept is offered in Employees & Contracts → **Fixed concepts (Conceptos fijos)** and in
Employee Records → **Records (Registros)**. Open payrolls are recalculated with its treatment the
next time Payroll is opened.

### Limitations (Limitaciones)

Closed payslips keep the treatment and accounts they were calculated with; editing a concept never
rewrites a closed month.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo creo un bono de productividad? ¿Cómo registro la movilidad? ¿Dónde creo un descuento por
  retención judicial?
- ¿Cómo hago un bono que no aporte a AFP? → an earning not marked remunerative.
- Bonificación, movilidad, condición de trabajo, comisión, retención judicial, descuento.

<!-- DOC-ID: capability.legal-values -->
## See the legal values in force (Valores legales)

### User intention (Intención del usuario)

Check the RMV, UIT, EsSalud, ONP and AFP rates the payroll is using.

### Where to find it (Dónde encontrarlo)

Tab **Legal values (Valores legales)**: the **Legal values (Valores legales)** table (**Parameter
(Parámetro)**, **Valid from (Vigente desde)**, **Value (Valor)**) and **AFP rates (SBS) (Tasas AFP
(SBS))** (**Period (Periodo)**, **AFP**, **Fund (Fondo)**, **Premium (Prima)**, **Flow comm. (Com.
flujo)**, **Balance comm. (Com. saldo)**).

### Business rules and rationale (Reglas y razón de negocio)

The payroll reads each value in force on its date; a new value of the law is a new dated row.

### Limitations (Limitaciones)

Read only: the company cannot add or change these values. They are shared by every company and
maintained by the SaaS operator in Labor Parameters (`Parámetros Laborales`).

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuál es la RMV que usa la planilla? ¿Dónde cambio la tasa de la AFP? ¿Cuánto es la UIT?
- Sueldo mínimo, remuneración mínima vital, UIT, tasas SBS.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"El código PLAME … no existe en la tabla 22 de SUNAT" / "no corresponde al tipo del
  concepto":** choose a code of the concept's type.
- **"Un ingreso no remunerativo no aporta a EsSalud ni a pensiones…":** untick those options or mark
  it remunerative.
- **"Un ingreso debe registrarse en una cuenta de gastos de personal (62)":** choose a 62 account.
- **"Indique la cuenta por pagar del descuento…":** choose the payable account.
- **"El concepto está asignado a trabajadores; quíteselo antes de eliminarlo":** remove it from the
  workers' fixed concepts first.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Employees & Contracts (Empleados & Contratos):** assigns company concepts as fixed concepts.
- **Employee Records (Registros Empleados):** registers one-off amounts of company concepts.
- **Payroll (Planilla):** applies the treatment when calculating payslips.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.browse-concepts]
  - path: frontend/routes/hr/hr-settings/+page.svelte
    role: page
    hash: sha256:4d99cfb83c751eca6d1cf43333d9d93ff9b24f7b76063c7736122f9e9405586e
    supports: [page-purpose, capability.browse-concepts, capability.legal-values]
  - path: frontend/routes/hr/hr-settings/ConceptsTab.svelte
    role: user-interface
    hash: sha256:5820bdc12257237a360468b2a18a009bb0e4a83ea2acc3548cef74595e647c7a
    supports: [concepts, capability.browse-concepts, capability.manage-company-concept]
  - path: frontend/routes/hr/hr-settings/hr-settings.ts
    role: business-logic
    hash: sha256:7e480489a9e89257b421ecfb3c092156e6e66a6d6a9cda4355767c7560d58385
    supports: [concepts, capability.browse-concepts, capability.manage-company-concept]
  - path: frontend/domain-components/hr/LegalParametersPanel.svelte
    role: user-interface
    hash: sha256:d17a5353537b8eedc47ec1a073d0df3578062feae8cbfc3c16b3c0cc55dd499b
    supports: [capability.legal-values]
  - path: frontend/services/hr/payroll.svelte.ts
    role: frontend-service
    hash: sha256:1a876ca2e211cb45cf7a27a985305a1fe03e87c23a6cdca8b8b411699e110b11
    supports: [capability.manage-company-concept]
  - path: frontend/services/hr/legal-parameters.svelte.ts
    role: frontend-service
    hash: sha256:1f6523e1f052c24e31935206d4072c34d5871bf0dbfa90d09407c373dd810b0c
    supports: [capability.legal-values]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts, capability.browse-concepts, capability.legal-values]
  - path: backend/hr/concepts.go
    role: backend-handler
    hash: sha256:fb26b24d8d3794198b82e3b5dd5a9dca0398b5cb88e290334f824df1e7ebf48d
    supports: [capability.manage-company-concept, troubleshooting]
  - path: backend/hr/legal_parameters.go
    role: backend-handler
    hash: sha256:25fd910adc618ba82d819d38a522cd0c75cf245ca9282b8e9b2fd620c95b8e13
    supports: [capability.legal-values]
  - path: backend/hr/types/concepts.go
    role: data-model
    hash: sha256:05941839ef9464fdc0e260950dad2eff2e1baa9d881b1c5839a837160be97082
    supports: [concepts, capability.manage-company-concept]
  - path: backend/hr/types/legal_concepts.go
    role: business-logic
    hash: sha256:0ec0c2c15ba57d378f4d6243a80fb1d3e90c54325354aea02501032fb76e38fd
    supports: [concepts]
  - path: backend/hr/types/legal_parameters.go
    role: data-model
    hash: sha256:be11e0dddc5dfcf924bef4556f55326c166d7e1ad8bdb399caf26903165decd6
    supports: [capability.legal-values]
  - path: backend/hr/types/payroll_data.go
    role: business-logic
    hash: sha256:bae9967e9ae7ba1e9060dde44697664aa5e30226407bf8a0b105380e21833e95
    supports: [capability.manage-company-concept]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:c859f2b1554396e71b39049ce914574878636a57c916bf3e41d1d3b74c414ac6
    supports: [capability.legal-values]
```
