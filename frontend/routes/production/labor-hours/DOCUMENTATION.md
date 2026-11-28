---
schema: 1
page_id: production.labor-hours
route: /production/labor-hours
title: Hours vs Payroll (Horas vs Planilla)
status: implemented
visibility: tenant
description_en: >-
  Monthly report comparing the hours each worker logged on production orders with the hours paid in
  the payroll, and the labor cost booked to products with the actual payroll cost. Shows utilization,
  idle hours and the monthly labor adjustment. Workers off payroll are listed apart with their rate.
description_es: >-
  Reporte mensual que compara las horas que cada trabajador registró en las órdenes de producción con
  las horas pagadas en la planilla, y la mano de obra cargada a los productos con el costo real de la
  planilla. Muestra utilización, horas ociosas y el ajuste mensual de mano de obra. Los trabajadores
  sin planilla se listan aparte con su tarifa.
---

# Hours vs Payroll (Horas vs Planilla)

<!-- DOC-ID: page-purpose -->
## Page purpose

Hours vs Payroll (`Horas vs planilla`) is a monthly report for manufacturing companies. For each
worker it compares the hours logged on production orders (`horas registradas`, `mano de obra`) with
the hours the payroll paid (`horas pagadas`), and the labor cost charged to the products made
(`M.O. cargada`) with what the worker's payroll month really cost (`costo planilla`). It shows how
much paid time went into production (`utilización`), the idle hours (`horas ociosas`), and the labor
adjustment of the month.

The page only reads. Hours are logged in the **Labor (Mano de obra)** tab of **Production orders
(Órdenes de producción)**; payslips come from **Payroll (Planilla)**; the adjustment entry is posted by
**Consolidation & Accounts (Consolidado & Cuentas)**, not here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- **Payroll workers (Trabajadores en planilla):** employees whose engagement is **Planilla**. They are
  listed when they logged hours in the month, or when they work in a department whose cost function
  is **Production (Producción)** and were employed some day of the month, even with no hours.
- **Workers off payroll (Trabajadores sin planilla):** engagement **Recibo por honorarios** or
  **Temporal sin planilla**. They appear only when they logged hours, in a separate table, with no
  payroll comparison and no adjustment.
- **Labor booked (M.O. cargada):** the value of the logged hours at the worker's real cost per hour,
  as each entry was saved on the order. A Planilla worker's month never books more than their
  estimated payroll cost; a worker off payroll is not capped and uses the rate of their agreement or,
  without one, the labor type's rate.
- **Hours paid (Horas pagadas):** the payslip's worked days × the **Daily hours (Horas diarias)** of the
  contract in force on the month's last day, plus the overtime (`Horas extras 25%` and `35%`) registered
  for the month.
- **Payroll cost (Costo planilla):** earnings, employer contributions and the month's provisions
  (gratificación, CTS, vacation), leaving out the part of a benefit paid in the month that was already
  provisioned, so nothing is counted twice.
- Only a **closed or paid** payslip counts; an open payroll is still a draft.

<!-- DOC-ID: capability.monthly-report -->
## Compare a month's hours and payroll (Comparar horas y planilla del mes)

### User intention (Intención del usuario)

Check whether production labor was fully used, how many paid hours were idle, and how much labor
cost was charged to products above the real payroll cost.

### Where to find it (Dónde encontrarlo)

**Operations (Operaciones) → Production (Producción) → Hours vs payroll (Horas vs planilla)** at
`/production/labor-hours`. Choose the **Month (Mes)**; it opens on the previous month and offers the
last twelve months and the current one, never a future month.

### Required information and prerequisites (Requisitos previos)

Hours logged on production orders in that month. The payroll columns need the month's payroll closed
in **Payroll (Planilla)**.

### Business rules and rationale (Reglas y razón de negocio)

- Columns of **Trabajadores en planilla:** **Employee (Trabajador)**, **Hours logged (Horas
  registradas)**, **By labor type (h) (Por tipo de labor (h))**, **Hours paid (Horas pagadas)**,
  **Utilization (Utilización)** = hours logged ÷ hours paid, **Idle hours (Horas ociosas)** = hours
  paid − hours logged, **Labor booked (M.O. cargada)**, **Payroll cost (Costo planilla)** and
  **Adjustment (Ajuste)**. A last row adds up the totals.
- **Adjustment** per worker = labor booked − payroll cost, only when positive; it shows in red. A
  shortfall (idle hours) produces no adjustment.
- A worker of a closed month who has no payslip shows **No payslip (Sin boleta)**; with no payroll cost,
  all their booked labor counts as adjustment.
- The box **Adjustment of the month (Dr 7111 · Cr 2111) (Ajuste del mes (Debe 7111 · Haber 2111))**
  shows the sum of the adjustments.
- While the month has no closed payroll, the page warns **Planilla pendiente** and shows only the hours
  and the labor booked; the payroll columns and the adjustment box are hidden.
- **Trabajadores sin planilla** shows **Worker (Trabajador)**, **Hours (Horas)**, **By labor type (h)**,
  **Rate per hour (Tarifa por hora)** (averaged by hours if it changed during the month) and **Labor
  booked (M.O. cargada)**.

### Result and side effects (Resultado y efectos)

Nothing is changed. The same adjustment is what **Consolidation & Accounts (Consolidado & Cuentas)**
posts as "Mano de obra cargada sobre la planilla", debit 7111 and credit 2111, once the month has
payslips.

### Limitations (Limitaciones)

- The report does not edit hours, payslips or the cost of products; correct hours in the production
  order and the payroll in Payroll.
- Stock already produced is not revalued by the adjustment.
- There is no adjustment for workers off payroll.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cuántas horas trabajó cada operario en producción este mes? ¿Cuántas horas pagadas quedaron
  ociosas?
- ¿Por qué hay un ajuste de mano de obra? ¿Qué es la cuenta 7111 / 2111 en producción?
- ¿Por qué no veo las horas pagadas? → the month's payroll is not closed yet.
- Términos: mano de obra directa, horas hombre, costo de planilla, utilización, horas ociosas, ajuste
  de mano de obra.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"Planilla pendiente: … aún no tiene planilla cerrada":** close the month's payroll in **Payroll
  (Planilla)** to see hours paid, cost and adjustment.
- **"Sin boleta" for a worker:** the closed payroll has no payslip for that worker, so their booked
  labor counts entirely as adjustment. Review the payroll or the hours logged.
- **"Ningún trabajador en planilla registró horas ni trabaja en producción.":** no hours were logged
  that month and no Planilla employee belongs to a Production department.
- **Hours paid look too low:** check the worked days of the payslip and the daily hours of the
  worker's contract.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Production orders (Órdenes de producción):** log the hours of each worker in the order's **Mano de
  obra** tab.
- **Payroll (Planilla):** close the month so payslips enter the comparison.
- **Employees & Contracts (Empleados & Contratos):** engagement type, department and the contract's
  daily hours.
- **Employee Records (Registros Empleados):** overtime registered for the month.
- **Departments & Positions (Áreas y Cargos):** the department's cost function **Producción**.
- **Production settings (Configuración de producción):** tab **Labor types (Tipos de trabajo)** and
  their cost per hour.
- **Consolidation & Accounts (Consolidado & Cuentas):** posts the monthly labor adjustment.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/pages.toml
    role: user-interface
    hash: sha256:d435afce0db0eedf924345040be60e0f4d1a15672e0f1f2d9003687920fb3da7
    supports: [page-purpose, capability.monthly-report, related-pages]
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.monthly-report]
  - path: frontend/routes/production/labor-hours/+page.svelte
    role: page
    hash: sha256:8feb3bae2a634b1fce9c04d5a5522f34706029de22b5fe0a11acb46a8cd30a55
    supports: [page-purpose, capability.monthly-report, troubleshooting]
  - path: frontend/routes/production/labor-hours/labor-hours.svelte.ts
    role: frontend-service
    hash: sha256:152a54428e0b494ae16b6abfbc34ec3612841265bfd556dcb9c0d2f2c3c0b4e9
    supports: [page-purpose, capability.monthly-report]
  - path: frontend/routes/production/labor-hours/labor-hours.ts
    role: business-logic
    hash: sha256:4ae9afe3dfae8ece1fc278184ac7e87597a4c917c2fed53ecfba3f9d244826f7
    supports: [concepts, capability.monthly-report, troubleshooting]
  - path: frontend/core/payroll-cost.ts
    role: business-logic
    hash: sha256:028297c06a70125511a05eb454de11898116c8c9c366871a02ed58fe361cc7cd
    supports: [concepts]
  - path: backend/production/production_labor.go
    role: backend-handler
    hash: sha256:8e1de7d581d752f00c10882fd64198bfad95b25e5175486eadb78a1b7732173b
    supports: [concepts, capability.monthly-report, related-pages]
  - path: frontend/routes/hr/payroll/payroll.ts
    role: shared-domain
    hash: sha256:58c3885b397125a11a9b2922a8cabd0023f1f0e74e39b94935fac0037090e15e
    supports: [concepts, capability.monthly-report]
  - path: frontend/services/hr/hr-catalogs.ts
    role: shared-domain
    hash: sha256:989372194f3cbd7cb1190fbe4cd1f46804f697d136e280490820c419d7c79c2d
    supports: [concepts, related-pages]
  - path: frontend/routes/accounting/accounts-consolidation/accounts-consolidation.ts
    role: business-logic
    hash: sha256:7cdc814a6eb91a7ecb6038243e7e7c6dafa176029d1c48eaf11dbf8188f6bf6c
    supports: [page-purpose, capability.monthly-report, related-pages]
  - path: frontend/routes/production/orders/OrderLayer.svelte
    role: user-interface
    hash: sha256:4a4bde050b0a7bbd17913252b2a69fefad3bc25285eca40e6ea66fd3ed29331c
    supports: [page-purpose, related-pages]
  - path: frontend/routes/hr/employees/EmployeeContractsAgreements.svelte
    role: user-interface
    hash: sha256:4dba0ced3aeaf024fbdbb6e2cfe5ca5c9e3a19247ccb2edd9019d55d7f9c0a3f
    supports: [concepts, related-pages]
  - path: frontend/routes/production/settings/+page.svelte
    role: user-interface
    hash: sha256:099df0f6e0ab94e5006c23f904d14a748016c0d8a24841231e65ff8ee194f97e
    supports: [related-pages]
```
