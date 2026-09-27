## The purchase document rules live in finance/types

**Context** — Purchase orders (logistics), expenses (finance) and assets (accounting) carry the same
supplier-comprobante columns. `accounting/types` was the natural home, but the rules need finance's
currency and exchange-rate constants while `Expense` needs the struct: that home is an import cycle.
**Decision** — `PurchaseDocument` and `NormalizePurchaseDocument` are in
`finance/types/purchase_document.go`. `PurchaseDocTypeOptions` is a catalog synced to
`frontend/core/purchase-document.ts`, and the generator now also scans `frontend/core`.
**Rationale** — finance already owns the currency and exchange-rate rules the document is checked
against. The ORM cannot embed a struct in a table, so each table has
`PurchaseDocument()`/`SetPurchaseDocument()` to copy the ten columns in and out.

## A purchase without comprobante stores its own date as DocIssueDate

**Context** — The book lists purchases without a comprobante behind a checkbox, so it has to find
them by period. The three tables had no common date index.
**Decision** — `NormalizePurchaseDocument` with DocType 0 clears the document and sets
DocIssueDate to the record's date (order Date, expense Date, AcquisitionDate). One index group on
DocIssueDate per table serves both lists. `fn-backfill-purchase-documents` fixes the existing rows.
**Rationale** — The alternative, a second date index on every table, doubles the reads for no gain.
The cost: for DocType 0 the column means "booking date", not "issue date", so the forms blank it
when opening an undocumented record.

## Boletas and recibos por honorarios must be entirely untaxed

**Context** — Neither document grants crédito fiscal (LIGV art. 19 requires a factura), so their
IGV cannot go to fields 15-16.
**Decision** — The backend rejects a base or IGV on types 02/03 (`CreditsTax = 0`). The form hides
those two inputs, and the split button puts the whole total in "No gravado" (field 21).
**Rationale** — Filing a boleta's IGV as crédito fiscal overstates the credit. ⚠️ This is common
practice, not verified against an annex line. Confirm it with the accountant.

## A comprobante must add up to what the purchase costs

**Context** — `Expense.Amount`, `Asset.PurchaseAmount` and `PurchaseOrder.TotalAmount` are the bill.
**Decision** — The document total must equal Amount (expense), PurchaseAmount (asset, per unit on
create) or TotalAmount (purchase order).
**Rationale** — Two figures for one purchase would book two different purchases. An invoice that
differs from the order is registered after editing the order to match it.

## Expenses and assets reuse CurrencyType and DueDate; both gain ProviderSnapshotID

**Context** — PLAN §2.2 listed `Currency` as a new column and a new due date for expenses. Both
tables already had `CurrencyType` and `DueDate`. Fields 12-14 need a frozen supplier identity,
which only purchase orders pinned.
**Decision** — The document shares the record's `CurrencyType`, and field 6 reads `DueDate` (order
`PaymentDate`). The order gains `CurrencyType` under the same name. Expense and Asset gain
`ProviderSnapshotID`, resolved from `SupplierID` on every save through `crm.ProviderSnapshotID`.
**Rationale** — One currency per record, one name across the three tables. A second currency
column could disagree with the first.

## A paid expense still accepts its comprobante

**Context** — `PostExpenses` refused any edit to a paid expense, but a bill often arrives after it
was paid. Without the bill the purchase can never reach the book.
**Decision** — On a paid expense, `savePaidExpenseDocument` writes only the supplier, the snapshot
and the document columns, validated against the stored Amount. Everything else stays locked.
**Rationale** — It closes the gap without reopening the amount or the dates of a settled expense.

## A transfer only goes to another register in the same currency

**Context** — The destination picker listed the register *types* (Caja / Cuenta Bancaria)
instead of the registers. The form also sent `CajaRefID` while the backend reads
`CashBankRefID`, so every transfer was rejected.

**Decision** — The picker lists the other active registers with the source's currency.
`PostCashBankMovement` refuses a destination that is the same register, belongs to another
company, or has a different currency.

**Rationale** — A cross-currency transfer is a currency exchange. It needs a rate, and the
movement has nowhere to store one. The cost: a PEN→USD move between tills cannot be recorded
as a transfer until that exists.

## Undated movements are reported by the backfill, not dated

**Context** — The reconciliation handler inserted its movement with `Date = 0`. The movement ID
packs `CashBankID + Date + autoincrement`, and period queries become ranges over that ID, so an
undated row sits under day 0 and no period ever reads it.

**Decision** — The handler now sets `Date` before inserting. The backfill only counts rows with
`Date = 0` and never writes them. On 2026-09-24 there were none: company 1's 883 movements are all
types 6, 7, 8 and 10.

**Rationale** — Updating only the column would leave the key and the column disagreeing. The
real fix is to re-key the row, and the ORM has no delete. Not worth building while there is
nothing to repair.

## "Charged to an employee" posts to 1419, not 1413

**Context** — The approved proposal named 1413 for a cash loss or shortage charged to a worker.
In the PCGE, 1413 is *Entregas a rendir cuenta*: money handed over to be accounted for later.
That is not a debt the worker owes.

**Decision** — `PCGEOtherStaffReceivables = 1419`, *Otras cuentas por cobrar al personal*.

**Rationale** — The Diario would otherwise show the shortage as an advance awaiting receipts.
Changing it is one constant plus a backfill, if an accountant prefers otherwise.

## The counter-account is validated in `ApplyCashBankMovement`, and a single option needs no choice

**Context** — Withdrawals, losses and physical counts now carry `AccountCode`. There are two
writers: `ApplyCashBankMovement`, the path every module's payments take, and the reconciliation
handler, which inserts its movement directly.

**Decision** — `ResolveCashMovementAccount` runs in both. It refuses a missing or foreign account
when the type has several options, and refuses any account on a type that has none. When the type
has exactly one option (a count surplus → 759), 0 resolves to it, so the form shows no picker. The
account is stored on the movement, not on `CashReconciliation`, because the movement is what the
Diario posts.

**Rationale** — A check placed in each handler would be missed by the next writer. Resolving the
single option on the server means the UI does not ask a question with one answer. It also keeps
the sign authoritative, since the reconciliation's difference is computed server-side.

## The account backfill uses each list's first option, and leaves "Unspecified" at 0

**Context** — `fn-backfill-cash-movement-accounts` stamps the rows written before the column
existed. The user's actual choice was never recorded.

**Decision** — Each row gets the first matching option in `CashMovementAccountOptions`:
withdrawal → 142, loss and shortage → 659, surplus → 759. Type-1 "Unspecified" movements are only
counted. The movement form does not offer that type (it is in group 1).

**Rationale** — The first option is the one the list presents as the default, so the backfill and
the list cannot drift. No account is correct for a movement that never said why money moved.
Guessing one would put a false entry in the Diario, so the count in the report is what decides
whether those rows need a manual fix.

## The sale's BCRP fallback is cached an hour and fails open to the company's rates

**Context** — A sale's rate falls back to the BCRP interbank series for the days the company did
not load, read by the backend from public-business-data.un.pe (one gzipped file per year). That
is an outbound call on the sale path.

**Decision** — `bcrpSellRatesByDay` keeps each year in memory. A past year is kept for good; the
current year is re-read after an hour. A year that cannot be downloaded is logged and skipped, so
the sale still uses the company's own rates, and is refused only if the whole window is empty.

**Rationale** — The current-year file gains one day per working day, so an hour of staleness never
changes today's rate in practice, and at most one download per hour per instance is added. Failing
closed would make every dollar sale depend on a third-party host even when the company has its
own rates.

# RATIONALE — finance

Design decisions for cash banks and expenses, newest first.

## The exchange rate month is its own id

**Context** — A month of exchange rates is 31 numbers that always arrive and are read together, and
the month it belongs to already identifies it. An autoincremented id would have added a second
identity that every write then has to resolve back to the month before it can update the right row.

**Decision** — `exchange_rates` is keyed by `ID int16` holding the month in YYMM form: 2601 is
January 2026. The rates live in two parallel `[]int32` arrays, `DetailBuyRate` and
`DetailSellRate`, where the position is the day (index 0 is day 1) and the value is the rate x 1000.
`PostExchangeRates` merges by that key, so saving a month is an upsert with no lookup of its own.

**Rationale** — The client can name the row it wants before it has ever seen it, which is what lets
the page build its 12-column grid from the year alone and post only the months that were touched.
The cost is that rewriting a single day rewrites the month's arrays; at one row per month that is
around 60 rows per company for five years, so the write amplification never shows up.

A rate of 0 means the day has no published rate — a weekend, a holiday, or a day nobody filled in —
and trailing zeros are trimmed before the row is stored, so a month loaded up to the 12th keeps 12
numbers. The handler rejects a rate above 1000.000 because at that magnitude it is a misplaced
decimal separator, and accounting would otherwise take it at face value.

## Expense.Type: what the money became

**Context** — Every purchase was an expense, and every expense went to the P&L. That is wrong for
two of them. Buying a laptop is not a cost — it is cash turning into an asset, and the cost arrives
later as depreciation. Buying inventory is cash turning into stock, which becomes cost when the
stock is consumed or sold. Recording either as a plain expense counts the same money twice: once at
purchase and again when it depreciates or sells.

**Decision** — `Expense.Type`: 1 Simple (P&L), 2 Inventory, 4 Depreciation. `AssetID` /
`ProductID` / `WarehouseID` / `Quantity` say what was bought and where it went.

There is deliberately no type for an asset acquisition: an asset is not an expense and writes no
row here at all. Its cost and payment state live on `accounting.Asset` — see
`accounting/RATIONALE.md`. 3 is left unused rather than renumbering 4, because "type 4 is
depreciation" is the shape the design was specified in.

**Rationale** — One `int8` is what lets Estados Financieros and Balance separate the P&L from the
balance sheet when they are built. Without it the split has to be reconstructed from category names,
which are free text chosen by the user. `PostExpenses` accepts only Types 1 and 2 — depreciation is
generated by the server, so accepting it here would let a client post a depreciation entry with no
asset behind it.

The Type-2 path is not in this package: it has to write a stock movement, and `logistics` already
imports `finance`, so it lives in `accounting` (see `accounting/RATIONALE.md`).

## Depreciation entries are Status 3, not a Type the queries have to know about

**Context** — Type-4 rows are `Expense` rows, and the register's tabs query by `Status`. At
`Status = 1` they would show as unpaid bills for money nobody owes.

**Decision** — `ExpenseStatusPosted = 3`. The tabs map to statuses 1 and 2, so they exclude
depreciation through the query that already exists. `PostExpensePayment` rejects it explicitly as
well, and `PostExpenses` refuses to edit it.

**Rationale** — Encoding it in `Type` would mean remembering `Type != 4` in every expense query
written from here on, where forgetting produces a wrong total rather than an error. A status the
tabs already filter on cannot be forgotten. The explicit rejections exist because a client can name
any ID it likes.
