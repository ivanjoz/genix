---
schema: 1
page_id: finance.loans
route: /finance/loans
title: Loans (Préstamos)
status: implemented
visibility: tenant
description_en: >-
  Register loans the company owes to banks and other lenders, with the lender's installment
  schedule. Propose the schedule with a TEA calculator (French, German or bullet), edit it to match
  the bank, register the disbursement into a cash or bank account, and pay each installment, fully or
  in part. A month calendar shows paid, due and overdue installments.
description_es: >-
  Registro de préstamos que la empresa debe a bancos y otras entidades, con su cronograma de cuotas.
  Calculadora de cuotas con TEA (francés, alemán o al vencimiento), edición del cronograma según el
  banco, registro del desembolso en una caja o banco y pago de cada cuota, total o parcial.
  Calendario mensual de cuotas pagadas, por vencer y vencidas.
---

# Loans (Préstamos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Loans (`Préstamos`) keeps the debts the company took from a bank or another lender (`entidad que
presta`): how much was borrowed (`monto del préstamo`), when the money arrived (`desembolso`), and
the installment schedule (`cronograma de cuotas`) the company repays. Each installment payment
(`pago de cuota`) leaves the money from a cash or bank account (`caja o banco`).

The page does not lend money to workers: staff loans and advances (`préstamos al personal`,
`adelantos`) belong to HR. It also does not create the cash forecast itself; the Cash Flow page
(`Flujo de Caja`) reads the pending installments from here.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A **loan** (`préstamo`) belongs to one lender, which must be registered as a provider
  (`proveedor`), and to one cash or bank account in the loan's currency.
- The **disbursement** (`desembolso`) is the borrowed money arriving in that account. It can be
  written as a cash movement **Loan disbursement (Desembolso Préstamo)** when the loan is created,
  or left out when the money arrived before the company used Genix.
- An **installment** (`cuota`) has a due date (`vencimiento`), principal (`capital` or
  `amortización`), interest (`interés`) and fees (`comisiones`, such as `desgravamen`). Its total is
  the sum of the three.
- **TEA** (`tasa efectiva anual`) is the annual effective rate. It is reference information: the
  installments carry the interest the lender actually charges.
- A loan is **Active (Vigente)** until every installment received its total; it then becomes
  **Paid off (Cancelado)**.

<!-- DOC-ID: capability.register-loan -->
## Register a loan (Registrar un préstamo)

### User intention (Intención del usuario)

Record a new bank loan or a loan already running, so its installments appear in the calendar and
in the cash flow forecast, and so each payment can be booked against it.

### Where to find it (Dónde encontrarlo)

**Finance & Accounting (Finanzas y Contabilidad) → Treasury (Tesorería) → Loans (Préstamos)** at
`/finance/loans`. Use **New loan (Nuevo préstamo)**; the side panel opens on the **Loan
(Préstamo)** tab. Save with the panel's save button.

### Required information and prerequisites (Requisitos previos)

- **Name (Nombre):** how the company identifies the loan.
- **Lender (Entidad que presta):** an active provider. Register the bank as a provider first.
- **Cash / bank account (Caja o banco):** where the money arrives and the default account
  installments are paid from. The loan takes the account's currency.
- **Amount (Monto)** and **Disbursement (Desembolso)** date.
- Optional: **TEA (%)**, **Method (Método)**, **Grace months (Meses de gracia)** and **Notes
  (Notas)**.
- **Register the disbursement in the account (Registrar el desembolso en la caja o banco):** only
  offered on a new loan, ticked by default.
- The installment schedule (`cronograma`): at least one installment, proposed by the calculator or
  entered by hand in the **Installments (Cuotas)** tab.

### Business rules and rationale (Reglas y razón de negocio)

- Installments are numbered 1 to n in order, each due strictly after the disbursement and after
  the previous installment.
- The principal of all installments must add up exactly to the loan amount. The Installments tab
  shows **Principal of the installments (Capital de las cuotas)** against the loan amount, in red
  while they differ.
- No amount can be negative, and the TEA and grace months cannot be negative.
- The loan currency must match the selected account's currency.

### Result and side effects (Resultado y efectos)

With the disbursement option ticked, the account receives a **Loan disbursement (Desembolso
Préstamo)** movement for the full amount, dated on the disbursement date, booked against loans
received (account 451). Without it, no cash moves. The pending installments start appearing in
the calendar and in the Cash Flow forecast.

### Limitations (Limitaciones)

- The disbursement can only be registered when the loan is created, not later.
- A loan cannot be entered as a manual **Cobro** in Cash & Banks: loans received (451) are not
  offered there, so the loan always carries its schedule. Older Cobros booked against 451 are
  flagged by Cash Flow.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo registro un préstamo del banco? ¿Dónde pongo el crédito del BCP?
- ¿Cómo ingreso un préstamo que ya tenía antes de usar Genix? → leave the disbursement option
  unticked.
- Préstamo bancario, crédito, financiamiento, pagaré, capital de trabajo.

<!-- DOC-ID: capability.calculate-schedule -->
## Propose the installments with the calculator (Calcular las cuotas)

### User intention (Intención del usuario)

Get a first schedule from the amount, the TEA and the term, then adjust it to the lender's
official schedule (`cronograma del banco`).

### Where to find it (Dónde encontrarlo)

In the loan panel, **Loan (Préstamo)** tab, box **Installment calculator (Calculadora de
cuotas)**: **Months (Meses)**, **First due date (Primera cuota)**, **Fee per installment
(Comisión por cuota)**, then **Calculate (Calcular)**.

### Required information and prerequisites (Requisitos previos)

Amount, disbursement date and months are required. The first due date defaults to one month after
the disbursement. The method and grace months come from the loan form.

### Business rules and rationale (Reglas y razón de negocio)

- Interest compounds the TEA by the actual days between installments over a 360-day year, as the
  calculator's note says.
- **French (Francés):** constant installment. **German (Alemán):** constant principal.
  **Bullet (Al vencimiento):** interest only, all principal in the last installment.
- Grace months charge interest only before amortization starts.
- The fee is added to every installment. The last installment absorbs rounding so the principals
  add up to the loan.
- Calculating replaces the current installments after a confirmation. It is refused once any
  installment has payments; edit the installments instead.

### Result and side effects (Resultado y efectos)

Only the form changes; nothing is saved until the loan is saved.

### Limitations (Limitaciones)

The lender's schedule always prevails: the calculator only proposes. Review the Installments tab
against the bank's document before saving.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo calculo la cuota de un préstamo? ¿Qué es el método francés o alemán?
- ¿Por qué mi cuota no coincide con la del banco? → adjust the installments by hand.

<!-- DOC-ID: capability.edit-installments -->
## Edit the installment schedule (Editar las cuotas)

### User intention (Intención del usuario)

Match the schedule exactly to the lender's: due dates, principal, interest and fees.

### Where to find it (Dónde encontrarlo)

Loan panel → **Installments (Cuotas)** tab. Cells are edited in place; the due date is typed as
dd-mm-yyyy. **Installment (Cuota)** adds an empty installment one month after the last one; the
delete icon removes one.

### Required information and prerequisites (Requisitos previos)

The rules of the loan apply on save: increasing due dates, principals adding up to the amount.

### Business rules and rationale (Reglas y razón de negocio)

- An installment that received a payment is locked: its amounts and due date cannot be edited and
  it cannot be removed.
- Once the disbursement is registered, the amount, account, currency and disbursement date can no
  longer change.
- What each installment has paid is kept by Genix from the payments; it is not typed.

### Result and side effects (Resultado y efectos)

Saving updates the calendar and the Cash Flow forecast. No cash moves.

### Limitations (Limitaciones)

There is no automatic rescheduling (`reprogramación`) after a prepayment; edit the remaining
installments to the lender's new schedule.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Puedo cambiar la fecha de una cuota? ¿Cómo agrego una cuota? ¿Por qué no puedo editar una
  cuota pagada?

<!-- DOC-ID: capability.pay-installment -->
## Pay an installment (Pagar una cuota)

### User intention (Intención del usuario)

Record that the company paid the lender an installment, fully or in part.

### Where to find it (Dónde encontrarlo)

Open a saved loan → **Installments (Cuotas)** tab → **Pay (Pagar)** on the installment. The
**Pay installment (Pagar cuota)** dialog asks **Pay from (Pagar desde)**, **Amount (Monto)** and
**Date (Fecha)**, then **Pay (Pagar)**.

### Required information and prerequisites (Requisitos previos)

The amount defaults to what the installment still owes, the account to the loan's account and
the date to today.

### Business rules and rationale (Reglas y razón de negocio)

- A partial payment is allowed. Each payment is applied to fees first, then interest, then
  principal, as a lender applies a short payment.
- A payment cannot exceed what the installment still owes.
- The paying account must be in the loan's currency, and its balance cannot go negative.
- The payment date cannot be earlier than the disbursement.
- The loan must be active.

### Result and side effects (Resultado y efectos)

One **Loan installment payment (Pago Cuota Préstamo)** movement leaves the account. That single
movement is divided across the accounts it settles: fees (671), interest (673) and principal
(451), so the accounting journal posts each part. The installment shows its paid amount and, when
complete, **Paid (Pagada)**. When every installment is complete, the loan becomes **Paid off
(Cancelado)**.

### Limitations (Limitaciones)

A payment is not undone from this page.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo pago la cuota del préstamo? ¿Puedo pagar una parte de la cuota? ¿Pago a cuenta de una
  cuota? ¿Cómo se separa el interés del capital?

<!-- DOC-ID: capability.calendar -->
## See upcoming installments in a calendar (Calendario de cuotas)

### User intention (Intención del usuario)

See which installments fall due this month across every active loan.

### Where to find it (Dónde encontrarlo)

Page tab **Calendar (Calendario)**, with arrows for the previous and next month.

### Business rules and rationale (Reglas y razón de negocio)

Green is paid, blue is due and red is overdue, as the legend says. Paid-off loans are not shown.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Qué cuotas vencen este mes? ¿Tengo cuotas vencidas?

<!-- DOC-ID: capability.delete-loan -->
## Delete a loan (Eliminar un préstamo)

### User intention (Intención del usuario)

Remove a loan entered by mistake.

### Where to find it (Dónde encontrarlo)

Open the loan → the panel's delete button, then confirm.

### Business rules and rationale (Reglas y razón de negocio)

A loan can only be deleted while it has moved no cash: no registered disbursement and no payments.
Otherwise those movements would remain in the account explaining nothing.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Por qué no puedo eliminar el préstamo? → its disbursement or a payment is already in an account.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **"La entidad que presta debe ser un proveedor activo":** register or reactivate the bank as a
  provider in Suppliers (Proveedores).
- **"El capital de las cuotas suma … y el préstamo es de …":** adjust the installments until the
  principal indicator matches the amount.
- **"La moneda del préstamo no coincide con la de la caja o banco":** choose an account in the
  loan's currency.
- **"El desembolso ya se registró en caja…":** the amount, account, currency and disbursement date
  are fixed after the disbursement.
- **"El pago supera lo pendiente de la cuota…":** pay at most what the installment still owes.
- **"El saldo de la caja no puede quedar negativo":** pay from another account or record the
  missing income first.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Cash Flow (Flujo de Caja)** projects every pending installment on its due date, and warns about
  overdue installments and loans without a rate.
- **Cash & Banks (Cajas & Bancos)** shows the disbursement and payment movements in each account.
- **Suppliers (Proveedores)** is where the lender is registered.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.register-loan]
  - path: frontend/routes/finance/loans/+page.svelte
    role: page
    hash: sha256:92ed31a3b734fd87eec98f92777759bb44ce66a9e639a0fa4545c31653ae98bd
    supports: [page-purpose, concepts, capability.register-loan, capability.pay-installment, capability.calendar, capability.delete-loan]
  - path: frontend/routes/finance/loans/LoanForm.svelte
    role: user-interface
    hash: sha256:8bd81b2eb3e6a46638fa44548d56206e0b9664a5bd4ddb3fe54d68b51cd7d927
    supports: [capability.register-loan, capability.calculate-schedule, capability.edit-installments]
  - path: frontend/routes/finance/loans/LoanInstallmentsTable.svelte
    role: user-interface
    hash: sha256:7d5f2b2e08d6508ddd61568cb72bddb8ed552c6502887664a9b74d4566ab2fc6
    supports: [capability.edit-installments, capability.pay-installment]
  - path: frontend/routes/finance/loans/LoansCalendar.svelte
    role: user-interface
    hash: sha256:9f9ba80b95f456f4a142ff6630e338f6330276a3761d426cee4f61d40a9e16f1
    supports: [capability.calendar]
  - path: frontend/routes/finance/loans/loans.ts
    role: business-logic
    hash: sha256:a9bd0405f41c66f7f229ce6b2e52b900a7db6e1a82626741cd45b286a5725b30
    supports: [capability.calculate-schedule, capability.calendar]
  - path: frontend/services/finance/loans.svelte.ts
    role: frontend-service
    hash: sha256:cfec3ec16cca88770256f1dbfdb6c8aa1df92ba798efa9c0f5cdc26ad0f55ead
    supports: [capability.register-loan, capability.pay-installment]
  - path: backend/finance/loans.go
    role: backend-handler
    hash: sha256:f7e42f93f0b1de80656535e137fc0e9b4850d0522dcd1aed2b6a49c05bebdf42
    supports: [capability.register-loan, capability.edit-installments, capability.pay-installment, capability.delete-loan, troubleshooting]
  - path: backend/finance/types/loans.go
    role: data-model
    hash: sha256:255986f9410b44df446ee5e8f29d1e9bfe47939c370fc7039c12956e5f030505
    supports: [concepts, capability.register-loan, capability.edit-installments, capability.pay-installment, troubleshooting]
  - path: backend/finance/types/cash_movement_account.go
    role: business-logic
    hash: sha256:2c462a5021a979a7ce37405eb667f99e784b699f75e024c419c184e239a93eba
    supports: [capability.register-loan, capability.pay-installment]
```
