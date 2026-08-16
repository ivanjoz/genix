---
schema: 1
page_id: finance.cash-banks
route: /finance/cash-banks
title: Cash & Banks (Cajas & Bancos)
status: implemented
visibility: tenant
---

# Cash & Banks (Cajas & Bancos)

<!-- DOC-ID: page-purpose -->
## Page purpose

Cash & Banks (`Cajas & Bancos`) manages the financial accounts where Genix records
cash inflows (`ingresos`), outflows (`egresos`), and the running available balance
(`saldo actual`). An account may represent a physical cash register (`caja`) or a bank
account (`cuenta bancaria`).

Use this page to configure those accounts, inspect their latest movements, register
manual movements, and reconcile the system balance with money actually counted. This
page does not create sales, supplier purchases, or expenses; those workflows may create
their own linked cash movements in the selected account.

<!-- DOC-ID: concepts -->
## Business concepts (Conceptos del negocio)

- A cash or bank account (`caja o banco`) is a ledger plus its current balance. Changing
  its name or branch does not change that balance.
- A movement (`movimiento de caja`) is an immutable-looking ledger entry with an amount
  and the resulting balance. Positive amounts increase the balance; withdrawals,
  losses, supplier payments, and expense payments are stored as negative amounts.
- A reconciliation (`cuadre`, also called `arqueo de caja`) compares the system balance
  (`saldo del sistema`) with the amount physically found (`saldo real`). The difference
  may be positive or negative.
- Currency (`moneda`) identifies the account as PEN or USD. Genix keeps each account's
  amount in its selected currency; this page does not perform currency conversion
  (`tipo de cambio`).

<!-- DOC-ID: capability.configure-account -->
## Create or configure an account (Crear o configurar una caja o banco)

### User intention (Intención del usuario)

Create a separate account for each place or financial channel whose balance must be
tracked, such as `Caja Principal`, `Caja Chica`, or a bank account. Edit it when its
descriptive configuration changes.

### Where to find it (Dónde encontrarlo)

Open **Finance (Finanzas) → Cash & Banks (Cajas & Bancos)** at
`/finance/cash-banks`. Use the create button for a new account. Select an existing row,
open **Config.**, and use the edit action to change it.

### Required information and prerequisites (Requisitos previos)

- **Type (Tipo):** `Caja` or `Cuenta Bancaria`.
- **Name (Nombre):** a user-recognizable account name.
- **Branch (Sede):** an existing branch/site to which the account belongs.
- **Currency (Moneda):** PEN or USD in the current interface.
- **Description (Descripción):** optional operational context.

Name, type, and branch are validated before saving. The currency selector is displayed
as required by the form, although the current server validation does not reject a
missing currency value.

### Business rules and rationale (Reglas y razón de negocio)

Editing configuration deliberately preserves the current balance, last reconciliation,
and creation audit fields. A configuration edit must not silently rewrite financial
history or reset a reconciled balance (`saldo cuadrado`).

### Result and side effects (Resultado y efectos)

A new account becomes selectable by workflows that use a `caja` or bank account. An
edit changes descriptive settings only; it creates no movement and does not recalculate
the balance.

### Limitations (Limitaciones)

- There is no implemented delete action on this page.
- The visible **Negative Balance (Saldo Negativo)** checkbox is not connected to a
  supported account rule. Do not rely on it to allow or prevent overdrafts.
- Selecting PEN or USD does not convert existing amounts or consolidate currencies.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo creo una caja chica o una cuenta bancaria?`
- `¿Dónde configuro la sede y moneda de una caja?`
- `How do I rename a cash register without changing its balance?`
- Search terms: `caja`, `banco`, `cuenta bancaria`, `caja chica`, `saldo`, `PEN`, `USD`.

<!-- DOC-ID: capability.manual-movement -->
## Register a manual movement (Registrar ingreso o egreso manual)

### User intention (Intención del usuario)

Use a manual movement when money changes outside an automated document workflow and the
account ledger must reflect it—for example a withdrawal (`retiro`), loss (`pérdida`),
general collection (`cobro`), or a supplier/expense payment registered manually.

### Where to find it (Dónde encontrarlo)

Select the account, stay in **Movements (Movimientos)**, and use the add movement action.
Choose a visible movement type and enter the amount.

### Required information and prerequisites (Requisitos previos)

An existing account, a non-zero amount, and a movement type are required. For
`Transferencia`, the server additionally requires a destination reference.

### Business rules and rationale (Reglas y razón de negocio)

The interface turns an entered positive value into a negative amount for an outflow
type. The server confirms that `previous balance + movement = final balance` before
saving. If another operation changed the account first, Genix returns the current
balance instead of applying a movement calculated from stale information.

### Result and side effects (Resultado y efectos)

Genix adds one ledger movement and updates the selected account's current balance. The
movement records its type, amount, resulting balance, date, and user.

### Limitations (Limitaciones)

The current `Transferencia` behavior records an outflow and a destination reference,
but it does not automatically create the matching inflow in the destination account.
Therefore it is not a complete two-sided bank transfer (`traspaso entre cajas`). The
current destination selector also does not reliably list actual accounts.

The server currently has no general rule that prevents a movement from leaving a
negative balance (`saldo en rojo`).

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo registro un retiro, pérdida, cobro o pago manual?`
- `¿Por qué cambió el saldo antes de guardar mi movimiento?`
- `Does Transferencia credit the destination caja automatically?` No, not currently.
- Search terms: `movimiento`, `ingreso`, `egreso`, `retiro`, `pérdida`, `cobro`,
  `transferencia`, `pago proveedor`, `pago gasto`.

<!-- DOC-ID: capability.reconcile -->
## Reconcile the account (Cuadrar o hacer arqueo de caja)

### User intention (Intención del usuario)

Use a reconciliation (`cuadre físico` or `arqueo`) after counting the actual cash or
checking the bank balance, to record whether it matches Genix.

### Where to find it (Dónde encontrarlo)

Select an account and open **Reconciliations (Cuadres)**. Start a new reconciliation,
review the displayed system balance, and enter the actual amount found.

### Required information and prerequisites (Requisitos previos)

The account must exist. Enter the complete observed balance, not only the shortage or
surplus. Genix calculates `difference = actual balance - system balance`.

### Business rules and rationale (Reglas y razón de negocio)

The server compares the system balance shown in the form with the latest stored balance.
If they differ, the reconciliation is not applied and the page asks the user to review
the updated balance. This protects the `cuadre` from overwriting a movement registered
after the counting form was opened.

### Result and side effects (Resultado y efectos)

An accepted reconciliation:

- stores the system amount, actual amount, difference, date, and responsible user;
- sets the account's current balance to the actual amount;
- records a `Cuadre Físico` movement for the signed difference; and
- updates the last reconciliation amount and date displayed in the account list.

### Limitations (Limitaciones)

A reconciliation is an adjustment, not only an observation. Entering an incorrect actual
amount changes the ledger balance. This page does not expose an undo or reversal action
for a completed reconciliation.

### Common questions and vocabulary (Preguntas y vocabulario)

- `¿Cómo hago un cuadre o arqueo de caja?`
- `¿Qué significa diferencia de caja?`
- `¿Por qué Genix me pide recalcular el cuadre?`
- `Does reconciliation adjust the balance?` Yes, it sets it to the observed amount and
  records the difference.

<!-- DOC-ID: capability.inspect-history -->
## Inspect movements and reconciliations (Revisar historial)

Selecting an account opens its detail panel. **Movements (Movimientos)** loads up to the
latest 200 movements in descending order and shows the movement type, amount, resulting
balance, linked document when available, date/time, and user. **Reconciliations
(Cuadres)** loads up to the latest 200 reconciliation records.

The lists are operational history, not a general accounting report. Use the linked sales,
expenses, or purchase-order workflows when the question concerns the originating
document rather than only its cash effect.

<!-- DOC-ID: rules -->
## Cross-capability business rules (Reglas generales)

- Each movement carries a resulting balance, allowing users to understand the account
  sequence rather than only a collection of unrelated amounts.
- Manual movement and reconciliation saves reject a stale starting balance instead of
  silently overwriting a concurrent change.

<!-- DOC-ID: troubleshooting -->
## Common problems (Problemas comunes)

- **“Nombre, Tipo y Sede son obligatorios”:** complete those three fields before saving.
- **The balance was updated while entering a movement or reconciliation:** review the new
  system balance and submit again; another operation changed the ledger first.
- **A transfer did not increase the destination balance:** current transfers are not
  two-sided; inspect both accounts and register the necessary destination operation using
  the business procedure approved for the company.
- **A payment appears in the movement list:** open the related purchase order, expense, or
  sale when a document identifier is present to understand its origin.

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Purchase Orders (Órdenes de Compra)** can create a `Pago Proveedor` movement when a
  confirmed purchase order is paid from an account.
- Expense workflows can create `Pago Gasto` movements.
- Sales collections can appear as `Cobro` or `Cobro (Venta)` movements.
- **Cash Movements (Cajas Movimientos)** is a separate finance route intended for a
  broader movement-oriented view; use this page when the task is account configuration,
  reconciliation, or one account's recent history.

### FILES

```yaml
# Exact source hashes captured after claim-by-claim review.
schema: 1
hash_algorithm: sha256
files:
  - path: frontend/core/modules.ts
    role: user-interface
    hash: sha256:0839d4ae72db6d7a902b99dae286edd5be0a16d691543ee7c1643e61fb4bf014
    supports: [page-purpose, capability.configure-account, related-pages]
  - path: frontend/routes/finance/cash-banks/+page.svelte
    role: page
    hash: sha256:43642169cc81a486395bd8fa91da8227a3baf24814c214df006983abc5e0848c
    supports: [page-purpose, concepts, capability.configure-account, capability.manual-movement, capability.reconcile, capability.inspect-history, troubleshooting]
  - path: frontend/routes/finance/cash-banks/CajaForm.svelte
    role: user-interface
    hash: sha256:885c56fa5832c39a9135ef3b129236abe43c30e9765864ce360aac40c4b1bae9
    supports: [capability.configure-account]
  - path: frontend/routes/finance/cash-banks/cajas.svelte.ts
    role: frontend-service
    hash: sha256:57abecc19b2da874e32d06acbf244f6bcd786470c746551be381ca02070e6887
    supports: [concepts, capability.configure-account, capability.manual-movement, capability.reconcile, capability.inspect-history]
  - path: backend/finance/cash_banks.go
    role: backend-handler
    hash: sha256:d3bd7b24ab258a52e5d404fee29c8b796d91507f84fb2caf2c0f44330504e36f
    supports: [capability.configure-account, capability.manual-movement, capability.reconcile, capability.inspect-history, rules, troubleshooting]
  - path: backend/finance/cash_bank_movement.go
    role: business-logic
    hash: sha256:dced7592fe610decf9352eb1705b09a46b75a2be73c438cff81d4be7f03a4775
    supports: [concepts, capability.manual-movement, capability.reconcile, rules]
  - path: backend/finance/types/cash_banks.go
    role: data-model
    hash: sha256:82ac985de5ca9fe560af0386a7bb6c35c8b65352c3a0b006cce8620fc5b84957
    supports: [concepts, capability.configure-account, capability.manual-movement, capability.reconcile, rules]
```
