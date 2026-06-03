# Expenses Module — Design

## 1. Purpose

The Expenses module lets a company register money it has to pay out and track its
payment state. An expense is one of:

- **One-time expense** — a single bill (e.g. "Office repair, 800 USD").
- **Scheduled expense** — a recurring template (e.g. "Rent 500 USD, every month on
  the 12th") that produces a stream of per-period expenses over time.

Every expense can be **unpaid**, **partially paid**, or **fully paid**. Payment is
recorded against the existing cash/bank movement ledger (see §4) — we do **not**
create a separate Payment table.

### Partial-payment / "Is Fully Paid" rule

A scheduled period can be closed for less than its nominal amount. Example: the
schedule is 500 USD/month; this month the user pays 480 USD and ticks the
**"Is Fully Paid"** checkbox. The expense is then treated as fully paid for that
period even though `PaidAmount (480) < Amount (500)`. The leftover 20 is written off,
not carried forward.

---

## 2. Data model

Two new tables in `backend/finance/types/expenses.go`. They follow the conventions in
the `create-database-tables` skill and mirror the existing finance tables in
`backend/finance/types/cajas.go` (partition on `CompanyID`, amounts stored as **cents
in int32**, dates as **UnixDay int16**, datetimes as **int32 SUnixTime**).

### 2.0 Static expense categories

Categories are a **fixed, code-defined list** (no DB table). Defined once on the
backend and mirrored on the frontend, keyed by a small `int8` `CategoryID`. Labels use
the bilingual `English|Spanish` pipe convention (`tr()` helper). Initial set:

| ID | Label |
| -- | ----- |
| 1  | `Rent\|Alquiler` |
| 2  | `Utilities\|Servicios` (water / electricity / internet) |
| 3  | `Payroll\|Planilla` |
| 4  | `Supplies\|Insumos` |
| 5  | `Taxes\|Impuestos` |
| 6  | `Maintenance\|Mantenimiento` |
| 7  | `Transport\|Transporte` |
| 8  | `Marketing\|Marketing` |
| 9  | `Professional services\|Servicios profesionales` |
| 10 | `Other\|Otros` |

`CategoryID` is stored on both `ExpenseScheduled` and `Expense`. A scheduled expense
copies its `CategoryID` down to each generated period (editable per period).

### 2.0.1 Currency

`CurrencyType int8` (same enum as `CashBank.CurrencyType`), restricted to two values:
**`1` = PEN**, **`2` = USD**. Currency is fixed by the schedule and copied to each
period (see Q5, resolved).

### 2.1 `ExpenseScheduled` — the recurring template

One row per recurring obligation. It does **not** hold payment state; it only
describes the cadence and the default amount. Actual per-period rows live in
`Expense`.

| Field            | Type     | Notes |
| ---------------- | -------- | ----- |
| `CompanyID`      | `int32`  | Partition (tenant). |
| `ID`             | `int32`  | Autoincrement. Referenced by `Expense.ExpenseScheduledID` and by `CashBankMovement.ReferenceID`. |
| `Name`           | `string` | e.g. "Office rent". |
| `Description`    | `string` | Optional. |
| `CategoryID`     | `int8`   | Static expense category (see §2.0). |
| `SupplierID`     | `int32`  | Optional — who is paid. |
| `CurrencyType`   | `int8`   | `1` = PEN, `2` = USD (see §2.0.1). |
| `Amount`         | `int32`  | Default expected amount per period, in cents. |
| `Frequency`      | `int16`  | **Packed cadence code `CDD`** (3 digits): first digit `C` = cadence, last two digits `DD` = day. See below. |
| `StartDate`      | `int16`  | UnixDay the schedule begins generating periods. Also **anchors the month** for N-monthly / yearly cadences. |
| `EndDate`        | `int16`  | UnixDay the schedule stops (0 = open-ended). |
| `Status`         | `int8`   | `json:"ss"` — 1 active / 0 deleted (also used to pause a schedule). |
| `Updated`        | `int32`  | `json:"upd"` — delta-cache watermark. |
| `UpdatedBy`      | `int32`  | |
| `Created`        | `int32`  | |
| `CreatedBy`      | `int32`  | |

**Cadence encoding (`Frequency`, packed `CDD`):** a single `int16` holds the whole
cadence. `Frequency = C*100 + DD`, where the first digit `C` is the cadence (chosen
from a UI dropdown) and the last two digits `DD` are the day. The day's meaning depends
on `C`:

| Cadence (dropdown) | `C` | `DD` meaning           | Example | Reads as |
| ------------------ | --- | ---------------------- | ------- | -------- |
| Weekly             | 1   | weekday `1`=Mon…`7`=Sun| `103`   | weekly, Wednesday |
| Monthly            | 2   | day of month `1`–`31`  | `212`   | monthly, on the 12th |
| Every 2 months     | 3   | day of month           | `315`   | every 2 months, on the 15th |
| Every 3 months     | 4   | day of month           | `401`   | every 3 months, on the 1st |
| Every 4 months     | 5   | day of month           | `520`   | every 4 months, on the 20th |
| Yearly             | 6   | day of month           | `601`   | yearly, on the 1st |

Notes:
- For N-monthly (`C=3,4,5`) and yearly (`C=6`), `StartDate` anchors **which** month(s)
  the cadence lands on; `DD` only sets the day-of-month. (e.g. `Every 3 months` from a
  February start → Feb, May, Aug, Nov.)
- `DD` is clamped to the real month length when generating a period (e.g. `231` in
  February falls on the 28th/29th).
- Decoding: `cadence = Frequency / 100`, `day = Frequency % 100`.

**Schema:**

```go
func (e ExpenseScheduledTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:         "expenses_scheduled",
        Partition:    e.CompanyID,
        UseSequences: true,
        Keys:         []db.Coln{e.ID.Autoincrement(0)},
        Indexes: []db.Index{
            // Delta-cache view: frontend syncs active schedules by watermark.
            {Type: db.TypeView, Keys: []db.Coln{e.Status.DecimalSize(1), e.Updated.DecimalSize(10)}, KeepPart: true},
        },
    }
}
```

### 2.2 `Expense` — a concrete expense (one-time or one period of a schedule)

One row per actual amount owed. For one-time expenses `ExpenseScheduledID = 0`. For
scheduled expenses, one row is generated per period, carrying the (possibly adjusted)
amount for that period plus its payment state.

| Field                | Type     | Notes |
| -------------------- | -------- | ----- |
| `CompanyID`          | `int32`  | Partition. |
| `ID`                 | `int32`  | Autoincrement. Referenced by `CashBankMovement.DocumentID` (int64 — int32 fits). |
| `ExpenseScheduledID` | `int32`  | `0` = one-time. Otherwise → `ExpenseScheduled.ID`. |
| `PeriodDate`         | `int16`  | UnixDay identifying which period this is (only meaningful when scheduled). Used to dedupe period generation. |
| `Name`               | `string` | Defaults from the schedule, editable per period. |
| `Description`        | `string` | |
| `CategoryID`         | `int8`   | Static category (§2.0); copied from the schedule, editable. |
| `SupplierID`         | `int32`  | |
| `CurrencyType`       | `int8`   | `1` = PEN, `2` = USD (§2.0.1). |
| `Date`              | `int16`  | UnixDay the expense was incurred. |
| `DueDate`           | `int16`  | UnixDay payment is due. |
| `Amount`            | `int32`  | Total owed for this expense/period, in cents (the per-period adjusted value). |
| `PaidAmount`        | `int32`  | Sum of payments applied so far, in cents. Maintained server-side when payments are recorded. |
| `Status`            | `int8`   | `json:"ss"` — **payment lifecycle**: `0` removed · `1` created/pending · `2` fully paid. Drives the status tabs (§5/§6). Set by `PostExpensePayment`; the UI "Is Fully Paid" checkbox forces `2` even when `PaidAmount < Amount` (write-off). The list chip (Unpaid/Partial/Paid) is **derived** from `ss` + `PaidAmount` — there is no separate `PaymentStatus` field. |
| `Updated`           | `int32`  | `json:"upd"`. |
| `UpdatedBy`         | `int32`  | |
| `Created`           | `int32`  | |
| `CreatedBy`         | `int32`  | |

**Payment lifecycle (`Status`/`ss`):** `PostExpensePayment` sets `ss` after recomputing
`PaidAmount`: `2` (fully paid) when the "Is Fully Paid" flag was sent or
`PaidAmount >= Amount`, else `1` (pending). New rows are created with `ss = 1`; `0` means
removed. Detail edits (`PostExpenses`) never touch `ss`/`PaidAmount` (excluded from the
update) so the lifecycle is server-authoritative. The list chip distinguishes
Unpaid/Partial purely from `PaidAmount` for `ss = 1` rows.

**Schema:**

```go
func (e ExpenseTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:         "expenses",
        Partition:    e.CompanyID,
        UseSequences: true,
        Keys:         []db.Coln{e.ID.Autoincrement(0)},
        Indexes: []db.Index{
            // Delta-cache view for the Register list.
            {Type: db.TypeView, Keys: []db.Coln{e.Status.DecimalSize(1), e.Updated.DecimalSize(10)}, KeepPart: true},
            // Fetch all periods belonging to a schedule.
            {Type: db.TypeLocalIndex, Keys: []db.Coln{e.ExpenseScheduledID}},
        },
    }
}
```

---

## 3. Relationships

```
ExpenseScheduled (1) ──< (N) Expense        Expense.ExpenseScheduledID → ExpenseScheduled.ID
Expense          (1) ──< (N) CashBankMovement   CashBankMovement.DocumentID  → Expense.ID
ExpenseScheduled (1) ──< (N) CashBankMovement   CashBankMovement.ReferenceID → ExpenseScheduled.ID
```

A payment is an existing **`CashBankMovement`** row (`backend/finance/types/cajas.go:57`)
of an outflow `Type`. Its `DocumentID` points at the `Expense.ID` being paid and its
`ReferenceID` at the originating `ExpenseScheduled.ID` (0 for one-time). Both columns
are already locally indexed, so "show all payments for this expense / schedule" is a
cheap query.

---

## 4. Payment flow

1. User opens an `Expense` and enters a payment (positive amount, date, source
   `CashBankID`, optional "Is Fully Paid" flag).
2. Backend creates a `CashBankMovement` of the new outflow type **`9` =
   `Expense Payment|Pago Gasto`** (added to `cajaMovimientoTipos`; `group: 2`,
   `isNegative: true`) via the existing `ApplyCajaMovimientos` path with
   `DocumentID = Expense.ID`, `ReferenceID = Expense.ExpenseScheduledID`. The movement
   `Amount` is stored **negative** (outflow): `Amount = -payment`.
3. Backend recomputes `Expense.PaidAmount` = Σ of the **absolute** values of its
   movements (queried via the `DocumentID` local index); `PaidAmount` is kept positive.
   It then sets `Status`/`ss` (`2` when fully paid / "Is Fully Paid", else `1`) and saves
   the `Expense` — moving the row between the status tabs (§5).
4. The cash/bank balance is adjusted by the existing movement logic — no new balance
   code needed here.

**Sign convention:** the user enters a positive payment; the ledger movement is
negative (`FinalAmount = previousBalance + Amount`); `Expense.PaidAmount` is the
positive running sum.

**Balance rule:** the resulting cash-bank balance is computed server-side
(`ApplyCajaMovimientos`); the client no longer sends an expected `FinalAmount`. The
only balance constraint is that it can't go negative — `PostExpensePayment` rejects
the payment with a plain error when `CurrentAmount - Amount < 0`.

**Currency match (validation):** `PostExpensePayment` rejects the payment unless
`Expense.CurrencyType == CashBank.CurrencyType` (cannot pay a USD expense from a PEN
register, or vice-versa).

Reusing `CashBankMovement` means expense payments automatically show up in
Cash Movements and Cash Flow reports.

---

## 5. Backend handlers — `backend/finance/expenses.go`

Following `backend/docs/CREATE_API_HANDLERS.md`. Register in
`backend/finance/main.go` `ModuleHandlers`:

| Router key                     | Handler                | Purpose |
| ------------------------------ | ---------------------- | ------- |
| `GET.expenses`                 | `GetExpenses`          | Per-status delta-cache list of `Expense`. One status tab per request via `?status=` (`0` Todos / `1` Pend. Pago / `2` Pagados) + `updated` watermark. Mirrors `GetSaleOrders`: rows that change status are evicted client-side via `records_IDsToRemove`. Todos/Pagados capped at 2000; Pend. Pago uncapped. |
| `POST.expenses`                | `PostExpenses`         | Create/update one-time or per-period expenses. |
| `GET.expenses-scheduled`       | `GetExpensesScheduled` | Delta-cache list of `ExpenseScheduled`. |
| `POST.expenses-scheduled`      | `PostExpensesScheduled`| Create/update a schedule. Does **not** pre-generate periods. |
| `GET.expense-schedule-periods` | `GetExpenseSchedulePeriods` | **Lazy generation:** given a `scheduleID`, materializes any missing `Expense` periods from `StartDate` up to today (deduped by `PeriodDate`), then returns the schedule's periods. Called when the user opens a schedule (§6). |
| `POST.expense-payment`         | `PostExpensePayment`   | Record a payment: writes a `CashBankMovement`, recomputes `Expense.PaidAmount`/status. |

**Lazy period generation (Q2, resolved):** periods are not created on schedule save nor
by a cron job. When the user opens a schedule, `GET.expense-schedule-periods` walks the
cadence from `StartDate` to today, creating one `Expense` per period that doesn't yet
exist (matched on `ExpenseScheduledID` + `PeriodDate`). Already-existing periods (and
their payment state) are untouched.

- **"Today" boundary:** generation emits periods with `PeriodDate <= req.EffectiveFechaUnix()`
  (the server's effective date), stopping early at `EndDate` when it is set (> 0).
- **Weekly anchor (`C=1`):** `DD` is the weekday (1=Mon…7=Sun). The first period is the
  first occurrence of weekday `DD` on/after `StartDate`; subsequent periods step +7 days.
- **Monthly / N-monthly / yearly anchor (`C=2..6`):** `StartDate` anchors which month(s)
  the cadence lands on; `DD` is the day-of-month, clamped to the real month length.

Validation (server, never trust client): positive `Amount`, valid `CurrencyType`
(1/2), valid `CategoryID` (in the static list), well-formed `Frequency` (`C` 1–6,
`DD` in range for the cadence), payment amount > 0 and a valid `CashBankID`.

---

## 6. Frontend — `frontend/routes/finance/expenses/`

Per the `create-page-layout` skill. Three files, plus a service:

### `+page.svelte` (page router)
The draft calls this `Expenses.svelte`; in SvelteKit the route entry must be
`+page.svelte`. It hosts the `Page` top-tab menu with two sections and delegates the
body to the two sub-views:

```svelte
<Page title="Expenses|Gastos"
  options={[{ id: 1, name: "Register|Registro" }, { id: 2, name: "Scheduled|Programados" }]}>
  {#if Core.pageOptionSelected === 1}<ExpensesRegister />{/if}
  {#if Core.pageOptionSelected === 2}<ExpensesSchedule />{/if}
</Page>
```

### `ExpensesRegister.svelte`
- `VTable` of `Expense` rows (columns: name, due date, amount, paid, status chip).
- `FilterInput` (client-side name filter) + 3 status tabs (Todos / Pend. Pago /
  Pagados). Each tab spins up a fresh `ExpensesService(status)` (route `?status=<code>`)
  and re-queries on tab switch — mirrors `sale_orders_status`. The chip (Sin pagar /
  Parcial / Pagado) is derived from `ss` + `PaidAmount`.
- A `Button` "New|Nuevo" opens a side `Layer` (`Core.openSideLayer(1)`) to create a
  one-time expense.
- Row click opens the same side `Layer` to edit and to **record a payment** (amount +
  "Is Fully Paid" checkbox), calling `POST.expense-payment`.

### `ExpensesSchedule.svelte`
- `VTable` of `ExpenseScheduled` rows (name, cadence summary, amount, next due).
- Editing uses a **side `Layer`** (`type="side"`, per the draft: "use a side layer
  component … similar to other modules"). The cadence is one **`SearchSelect`/dropdown**
  picking the cadence (`C` → Weekly / Monthly / Every 2–4 months / Yearly) plus a day
  field; the two combine into the packed `Frequency` (`C*100 + DD`). A second Layer tab
  lists the generated `Expense` periods with their payment state — opening the schedule
  triggers `GET.expense-schedule-periods` to lazily materialize missing periods.
- Reference page for the Page-tabs + side-Layer combination:
  `frontend/routes/negocio/sedes-almacenes/+page.svelte`.

### `expenses.svelte.ts` (service)
Two delta-cached services (`SERVICES_GUIDE.md` / `delta-cache-api` skill):
`ExpensesService` (→ `GET.expenses`) and `ExpensesScheduledService`
(→ `GET.expenses-scheduled`), plus a `postExpensePayment(...)` report-style call.
TS interfaces `IExpense` / `IExpenseScheduled` mirror the Go structs.

---

## 7. Decisions & open questions

**Resolved:**
- **Q1 — Cadence UI.** A dropdown picks the cadence; cadence + day pack into a single
  3-digit `Frequency` (`CDD`) field (§2.1).
- **Q2 — Period generation.** Lazy, on opening a schedule, via
  `GET.expense-schedule-periods` (§5). No cron, no pre-generation on save.
- **Q3 — Categories.** Static code-defined category list, `int8 CategoryID` (§2.0).
- **Q5 — Currency.** Fixed per schedule, copied to each period. Values: PEN / USD
  (§2.0.1).
- **Q4 — Route slug.** Standardized on `/finance/expenses` (matches the folder and the
  English-route convention). The `modules.ts:121` menu entry currently pointing at
  `/finance/gastos` will be updated to `/finance/expenses`.
```
