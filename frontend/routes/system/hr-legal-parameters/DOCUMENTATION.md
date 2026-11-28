---
schema: 1
page_id: system.hr-legal-parameters
route: /system/hr-legal-parameters
title: Labor Parameters (Parámetros Laborales)
status: implemented
visibility: saas
description_en: >-
  SaaS-only maintenance of the Peruvian legal payroll values every company's payroll uses: RMV, UIT,
  EsSalud and ONP rates, the maximum insurable remuneration, the family allowance rate and the
  gratificación bonus rates, each dated from when it applies, plus the monthly SBS rates of each
  AFP.
description_es: >-
  Mantenimiento, solo para la empresa SaaS, de los valores legales de planilla que usan todas las
  empresas: RMV, UIT, tasas de EsSalud y ONP, remuneración máxima asegurable, asignación familiar y
  bonificación extraordinaria, cada uno con su fecha de vigencia, y las tasas mensuales SBS de cada
  AFP.
---

# Labor Parameters (Parámetros Laborales)

<!-- DOC-ID: page-purpose -->
## Page purpose

Labor Parameters (`Parámetros Laborales`) stores the legal values of Peruvian payroll (`valores
legales`) and the monthly AFP rates published by the SBS (`tasas AFP`). Every company's payroll
reads them, so they are maintained once by the SaaS operator, not by each company.

Companies do not see this page. They see the same values read only in HR Settings (`Configuración
RRHH`) → Legal values, and they feel their effect in their payslips.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **legal value** (`valor legal`) has a **Parameter (Parámetro)**, a **Valid from (Vigente desde)**
  date and a **Value (Valor)**. It applies from its date until the next value of the same parameter.
  Parameters: **Minimum wage (RMV) (Remuneración mínima vital (RMV))**, **Tax unit (UIT) (Unidad
  impositiva tributaria (UIT))**, **EsSalud rate (Tasa EsSalud)**, **EsSalud rate with EPS (Tasa
  EsSalud con EPS)**, **ONP rate (Tasa ONP)**, **Max. insurable salary (AFP premium) (Remuneración
  máxima asegurable)**, **Family allowance (% of RMV) (Asignación familiar (% de la RMV))**,
  **Gratificación bonus (Ley 30334) (Bonificación extraordinaria (Ley 30334))** and **Gratificación
  bonus with EPS (Bonificación extraordinaria con EPS)**.
- Amounts are in soles; rates are percentages.
- The **AFP rates of a period** (`tasas AFP del periodo`) are, for each AFP (Habitat, Integra, Prima,
  Profuturo) and month (YYMM): fund contribution (`aporte al fondo`), insurance premium (`prima de
  seguro`), flow commission (`comisión sobre el flujo`) and balance commission (`comisión sobre el
  saldo`). The balance commission is kept for reference; payroll does not withhold it.

<!-- DOC-ID: capability.legal-value -->
## Add or correct a legal value (Valor legal)

### User intention (Intención del usuario)

Load a new RMV, UIT or rate when the law changes, or fix a mistyped value.

### Where to find it (Dónde encontrarlo)

**System → Platform (Plataforma) → Labor Parameters (Parámetros Laborales)** at
`/system/hr-legal-parameters`, only in the SaaS company. In **Legal values (Valores legales)**,
**New value (Nuevo valor)** opens **New legal value (Nuevo valor legal)** with **Parameter
(Parámetro)**, **Valid from (Vigente desde)** and **Amount (S/) (Monto (S/))** or **Rate (%) (Tasa
(%))** depending on the parameter. Clicking a row opens **Edit legal value (Editar valor legal)**,
which can also delete it.

### Business rules and rationale (Reglas y razón de negocio)

- The date is required and the value must be above zero; a rate cannot exceed 100%.
- A parameter holds one value per date. A change of law is a new row with its own date, never an
  edit of the old value, so earlier months keep the value that applied to them.

### Result and side effects (Resultado y efectos)

Every company's payroll uses, for each month, the value in force on that date. Open payrolls
calculated before the change become stale and are recalculated the next time each company opens
Payroll. Closed payslips keep the values they were calculated with.

### Limitations (Limitaciones)

Values are typed by hand; there is no import from SUNAT, the SBS or the official gazette.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde actualizo la RMV? ¿Cómo cargo la UIT del nuevo año? ¿Cambió la tasa de EsSalud?
- Sueldo mínimo, remuneración mínima vital, UIT, tope asegurable, porcentaje de asignación
  familiar.

<!-- DOC-ID: capability.afp-rates -->
## Load the AFP rates of a month (Tasas AFP del periodo)

### User intention (Intención del usuario)

Load the SBS rates of each AFP for a month, so the companies' payrolls can withhold AFP
contributions.

### Where to find it (Dónde encontrarlo)

**AFP rates (SBS) (Tasas AFP (SBS))** → **Period rates (Tasas del periodo)** opens **AFP rates of the
period (Tasas AFP del periodo)**: **Period (YYMM) (Periodo (AAMM))** and, for each AFP, **Fund %
(Fondo %)**, **Premium % (Prima %)**, **Flow % (Flujo %)** and **Balance % (Saldo %)**. Clicking a
row opens that period's four AFPs. The table lists **Period (Periodo)**, **AFP**, **Fund (Fondo)**,
**Premium (Prima)**, **Flow comm. (Com. flujo)** and **Balance comm. (Com. saldo)**.

### Required information and prerequisites (Requisitos previos)

A new period proposes 10.00% fund and 1.37% premium for every AFP; adjust them to the SBS table.

### Business rules and rationale (Reglas y razón de negocio)

- The period must be a real month in YYMM format, for example 2611 for November 2026.
- Every rate is between 0% and 100%; the fund contribution and the premium are required.
- Saving a period that already exists overwrites its rates, so re-entering a month corrects it.

### Result and side effects (Resultado y efectos)

The payroll of that month withholds, for each AFP affiliate, the fund contribution, the premium on
pay capped by the maximum insurable remuneration, and the flow commission when the worker's
commission type is on flow. Open payrolls are recalculated the next time each company opens
Payroll.

### Limitations (Limitaciones)

AFP rates cannot be deleted from the page; correct them by saving the period again.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde cargo las tasas de la AFP del mes? ¿Cómo corrijo la comisión de Prima?
- Comisión sobre el flujo, comisión mixta, prima de seguros, aporte obligatorio, SBS.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Ya existe un valor de … vigente desde esa fecha; edítelo en lugar de crear otro":** open that
  row instead of adding one.
- **"La tasa de … no puede superar el 100%" / "Las tasas de AFP deben estar entre 0% y 100%":** type
  the percentage, not a fraction.
- **"Ingrese un periodo AAMM" / "El periodo … no es válido (formato AAMM)":** type the year's last two
  digits and the month.
- **"Indique el aporte al fondo y la prima de seguro de la AFP":** fill both for every AFP.
- **A company's payslips show `HR-003` (missing AFP rates) or `HR-004` (missing legal value):** load
  the month's AFP rates or the missing legal value here.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **HR Settings (Configuración RRHH)** of each company shows these values read only.
- **Payroll (Planilla)** of each company calculates with them.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.legal-value]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [page-purpose]
  - path: backend/main-handlers.go
    role: permissions
    hash: sha256:c859f2b1554396e71b39049ce914574878636a57c916bf3e41d1d3b74c414ac6
    supports: [page-purpose]
  - path: frontend/routes/system/hr-legal-parameters/+page.svelte
    role: page
    hash: sha256:f9daa723c88f37e1cb9eb6280607c97f2d58236139b372c31f9b4c00e8df91c9
    supports: [page-purpose, capability.legal-value, capability.afp-rates]
  - path: frontend/domain-components/hr/LegalParametersPanel.svelte
    role: user-interface
    hash: sha256:d17a5353537b8eedc47ec1a073d0df3578062feae8cbfc3c16b3c0cc55dd499b
    supports: [concepts, capability.legal-value, capability.afp-rates, troubleshooting]
  - path: frontend/services/hr/legal-parameters.svelte.ts
    role: frontend-service
    hash: sha256:1f6523e1f052c24e31935206d4072c34d5871bf0dbfa90d09407c373dd810b0c
    supports: [capability.legal-value, capability.afp-rates]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts]
  - path: backend/hr/legal_parameters.go
    role: backend-handler
    hash: sha256:25fd910adc618ba82d819d38a522cd0c75cf245ca9282b8e9b2fd620c95b8e13
    supports: [capability.legal-value, capability.afp-rates, troubleshooting]
  - path: backend/hr/types/legal_parameters.go
    role: data-model
    hash: sha256:be11e0dddc5dfcf924bef4556f55326c166d7e1ad8bdb399caf26903165decd6
    supports: [concepts, capability.legal-value]
  - path: backend/hr/types/payroll_data.go
    role: business-logic
    hash: sha256:bae9967e9ae7ba1e9060dde44697664aa5e30226407bf8a0b105380e21833e95
    supports: [capability.legal-value, capability.afp-rates]
  - path: backend/hr/types/payroll_engine.go
    role: business-logic
    hash: sha256:781443cdd93f5953057a93225c36b9fd1bf2e27efdfae924c887ced25c11d2b7
    supports: [capability.afp-rates, troubleshooting]
  - path: backend/hr/types/payroll.go
    role: data-model
    hash: sha256:77649e81bc52c64e4c8982a5cbda388fed530b579684aeded39a2eb44bca4e4b
    supports: [capability.legal-value]
  - path: frontend/routes/hr/hr-settings/+page.svelte
    role: shared-domain
    hash: sha256:4d99cfb83c751eca6d1cf43333d9d93ff9b24f7b76063c7736122f9e9405586e
    supports: [page-purpose, related-pages]
```
