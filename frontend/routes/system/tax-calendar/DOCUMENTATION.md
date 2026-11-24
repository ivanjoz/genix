---
schema: 1
page_id: system.tax-calendar
route: /system/tax-calendar
title: Tax Calendar (Calendario Tributario)
status: implemented
visibility: saas
description_en: >-
  SaaS-only maintenance of SUNAT's monthly due-date schedule: for each tax period, the due date of
  the monthly obligations for each RUC last digit (0 to 9) and for good taxpayers. Every company's
  cash flow forecast dates its IGV, income tax and payroll contributions with it.
description_es: >-
  Mantenimiento, solo para la empresa SaaS, del cronograma mensual de vencimientos de SUNAT: por cada
  periodo tributario, la fecha de vencimiento de las obligaciones mensuales según el último dígito del
  RUC (0 al 9) y para buenos contribuyentes. El flujo de caja de todas las empresas fecha con él su
  IGV, renta y aportes de planilla.
---

# Tax Calendar (Calendario Tributario)

<!-- DOC-ID: page-purpose -->
## Page purpose

Tax Calendar (`Calendario Tributario`, `cronograma de vencimientos SUNAT`) stores, for each tax
period (`periodo tributario`), the day the monthly obligations fall due: one date per RUC last
digit (`último dígito del RUC`, 0 to 9) and one for good taxpayers (`buenos contribuyentes`). The
dates are transcribed from SUNAT's published schedule, never computed, because decreed non-working
days move them.

The schedule is shared by every company and only the SaaS operator maintains it. Companies do not
see this page; they only feel its effect in Cash Flow (`Flujo de Caja`). Whether a company is a good
taxpayer is set by each company in its configuration, not here.

<!-- DOC-ID: capability.add-period -->
## Add a period's due dates (Registrar un periodo)

### User intention (Intención del usuario)

Load the due dates of a new month as soon as SUNAT publishes its schedule, so the cash flow
forecasts date that month's taxes correctly.

### Where to find it (Dónde encontrarlo)

**System → Platform (Plataforma) → Tax Calendar (Calendario Tributario)** at
`/system/tax-calendar`, only in the SaaS company. **New period (Nuevo periodo)** opens a dialog
with **Period (YYMM) (Periodo (AAMM))** and eleven dates: **RUC ending in 0 … 9 (RUC terminado en
0 … 9)** and **Good taxpayers (Buenos contrib.)**.

### Required information and prerequisites (Requisitos previos)

- The period as YYMM, for example 2612 for December 2026. It proposes the month after the newest
  stored period.
- All eleven due dates.

### Business rules and rationale (Reglas y razón de negocio)

- The period must be a real month in YYMM format.
- Every due date must fall after the period's last day, because a month's tax is declared once the
  month has closed.
- Saving a period that already exists overwrites its dates.

### Result and side effects (Resultado y efectos)

The period appears as a row with its period and the day and month of each column. From then on,
the Cash Flow of every company dates that period's IGV, income tax prepayment, Nuevo RUS and
contributions on the column of its RUC's last digit, or on the good taxpayers column if the company
is marked as one.

### Limitations (Limitaciones)

There is no import from SUNAT: dates are typed by hand from the official page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Dónde cargo el cronograma de SUNAT? ¿Cómo registro los vencimientos de diciembre?
- Cronograma de obligaciones mensuales, vencimiento por dígito, fecha de declaración, buen
  contribuyente, UESP.

<!-- DOC-ID: capability.edit-period -->
## Correct or delete a period (Editar o eliminar un periodo)

### User intention (Intención del usuario)

Fix a mistyped date, follow a SUNAT change (for example a new holiday), or remove a wrong period.

### Where to find it (Dónde encontrarlo)

Click the period's row: the **Edit period (Editar periodo)** dialog opens with its dates. Save, or
use the delete button and confirm.

### Business rules and rationale (Reglas y razón de negocio)

- The period itself cannot be changed when editing; to move dates to another month, create that
  period and delete the wrong one.
- The same date rules as when adding apply.

### Result and side effects (Resultado y efectos)

The forecasts use the corrected dates the next time they are computed. After a delete, the period's
taxes and contributions fall due on the 15th of the following month, and Cash Flow warns that the
SUNAT calendar is missing for it.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo cambio una fecha del cronograma? ¿SUNAT movió el vencimiento, cómo lo actualizo?

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Ingrese un periodo AAMM" / "no es válido (formato AAMM)":** type the year's last two digits
  and the month, for example 2701.
- **"Ingrese todas las fechas de vencimiento":** fill all eleven dates.
- **"El vencimiento N del periodo … debe caer después del cierre del periodo":** a date is inside
  the period itself; due dates belong to the following month or later.
- **A company's Cash Flow shows "Sin calendario SUNAT para estos periodos":** load those periods
  here.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash Flow (Flujo de Caja)** of each company uses these dates and warns when a period is
  missing.
- **Company configuration (Configuración de la empresa)** is where a company sets its tax regime
  and whether it is a good taxpayer (`buen contribuyente`).

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.add-period]
  - path: backend/access.toml
    role: permissions
    hash: sha256:1e71e45906aed3d07f42a65ad2ad13b38ad2055938fd5098547be4d4898f769a
    supports: [page-purpose]
  - path: frontend/routes/system/tax-calendar/+page.svelte
    role: page
    hash: sha256:5c68d4bbfd6c302dd173e139f5ec3902cf0d1eb7817060f99990bcc4fcf7aaa5
    supports: [page-purpose, capability.add-period, capability.edit-period]
  - path: frontend/routes/system/tax-calendar/tax-calendar.ts
    role: business-logic
    hash: sha256:fe9dbc8df8c5f45a7dea545ecf06622d36883c4501b8c732014710ed1cb83452
    supports: [capability.add-period, troubleshooting]
  - path: frontend/services/accounting/tax-due-calendar.svelte.ts
    role: frontend-service
    hash: sha256:e2630cf1dc29fb63a5eda91756d00f020be1acec863bf42dc6f4986ed8e16b67
    supports: [capability.add-period, capability.edit-period]
  - path: backend/accounting/tax_due_calendar.go
    role: backend-handler
    hash: sha256:dd09694bcaa8a21c4a63e8d785dff86c4ec2cfe1d463f13371729f95278dfc3a
    supports: [page-purpose, capability.add-period, capability.edit-period]
  - path: backend/accounting/types/tax_due_calendar.go
    role: data-model
    hash: sha256:8e905655ac1b016b4ba2f41352c4ed2f3b08fdd29425811666852bbe307b4c7f
    supports: [capability.add-period, capability.edit-period, troubleshooting]
  - path: backend/finance/cash_flow_commitments.go
    role: shared-domain
    hash: sha256:f07f8d3b96c303bd871b2266ca59bfb2a7b0cdce7f522b2aec85dafb4ca71394
    supports: [capability.add-period, capability.edit-period, troubleshooting]
  - path: frontend/routes/company/configuration/CompanyTab.svelte
    role: shared-domain
    hash: sha256:8b1af029a0ca90629ede659f5ea7bcbc04101329c2d778fd09e6f6c9af8c7ac9
    supports: [related-pages]
```
