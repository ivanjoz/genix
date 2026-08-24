# RATIONALE — accounting

Design decisions behind the asset register and depreciation, newest first.
Full design in `PLAN.md`.

## A depreciation entry stores no PeriodDate either, because it dedupes elsewhere

**Context** — `Expense.PeriodDate` is the key the scheduled-expense generator matches on to avoid
materializing a period twice (`finance/expenses.go:384`). Depreciation was writing it too, always
equal to `Date`.

**Decision** — Type-4 rows carry only `Date`. `PeriodDate` stays for scheduled expenses, which is
its only reader.

**Rationale** — Depreciation does not dedupe against the expenses table at all: `PendingDepreciation
Periods` compares the computed schedule against `Asset.LastDepreciationDate`. So the column was
write-only on a Type-4 row, and could never diverge from `Date` anyway — `PostExpenses` refuses to
edit a row at `ExpenseStatusPosted`. `Date` is the column kept because it is the accounting date
every date-ranged report filters on, and depreciation is a real non-cash P&L expense. The cost: if
posting ever moves off the first of the month, the two stop being the same fact and `PeriodDate`
has to come back.

## A depreciation entry stores no Name, because the label is presentation

**Context** — `PostAssetDepreciationRun` persisted `"Depreciación 7/60"` into `Expense.Name`. That
put a rendering decision in the database: the string is Spanish-only in a bilingual app, and it
freezes `DepreciationMonths` at whatever it was the day the row was written.

**Decision** — The Type-4 expense is written with no `Name`. The Activos page composes the label
from `PeriodDate` and the asset's own `DepreciationMonths` (`depreciationSchedule` in
`frontend/routes/accounting/assets/assets.ts`).

**Rationale** — Nothing else reads that column for a Type-4 row: `belongsToTab` in the Gastos
register only admits `ss` 1 and 2, and depreciation is posted at `ss` 3, so the schedule in the
asset panel is its only reader. The cost is that rows written before this change keep a dead
`Name` in the database — harmless, since no code reads it any more.

## Every asset update writes Status, even when it does not change it

**Context** — `PostAssetPayment` updated `PaidAmount`, `PaymentStatus`, `Updated` and `UpdatedBy`,
which is exactly the set of columns a payment changes. It panicked on the first real call:

    Table "accounting_asset": A composit index/view requires the columns "status",
    "updated_version" be updated together. Not Included: status

**Decision** — Every `db.Update` on an asset includes `assetTable.Status`, whether or not the
lifecycle moved. Same for `PutAssetTransfer`. `finance/expenses.go` carries the identical note for
the same reason.

**Rationale** — The delta index keys on `(status, updated_version)`, and the ORM refuses to write
half of a composite key. Writing only the columns that changed is the natural instinct and it is
wrong for any table with a delta index — which is most of them. Worth remembering: the failure is a
panic at runtime, not a compile error, so it only shows up when the endpoint is actually called.

## A payment is not atomic, and a mid-way failure leaves the cash moved

**Context** — `PostAssetPayment` writes the cash-bank outflow first, then recomputes and stores
`PaidAmount`. While the composite-key bug above was live, three test payments wrote their outflow
and then panicked on the update: 75,000 left the register with nothing on the asset recording it.

**Decision** — Left as is, and recorded here. The recompute-from-ledger design is what limits the
damage: `PaidAmount` is derived by summing the movements, never incremented, so the next successful
payment reconciles the orphans instead of compounding them — which is exactly what happened.

**Rationale** — Making the pair atomic needs a transaction the ORM does not offer across tables.
The alternative order (asset first, cash second) is worse: it would show money as paid that never
left the register. `PostExpensePayment` has the same shape and the same exposure, so this is a
property of the payment design rather than of this handler.

## An asset is not an expense, so it writes no expense row

**Context** — The first cut recorded an asset acquisition as an `Expense` of Type 3, carrying a
`Value` alongside `Amount` so book value could differ from cash paid. It kept the money in one
table, but it made the asset a kind of expense, which it is not.

**Decision** — Acquiring an asset writes **nothing** to `expenses`. `PurchaseAmount`, `PaidAmount`,
`PaymentStatus`, `DueDate` and `SupplierID` all live on `accounting.Asset`, and `POST.asset-payment`
settles it against a cash register directly. `ExpenseTypeAsset` and `Expense.Value` are gone. The
only row an asset ever contributes to `expenses` is its monthly Type-4 depreciation.

**Rationale** — Buying a fixed asset is cash turning into a balance-sheet item, not a cost. Putting
it in the expense register meant every reader of that table had to know that some of its rows were
not really expenses, and `Value` existed solely to carry a number that had no meaning for the other
types. Now the register contains only things that are costs, and the asset owns its own money.

The cost is a second payment path: `PostAssetPayment` duplicates the shape of `PostExpensePayment`
(load, validate against the pending balance, check the register's currency and balance, write the
outflow, recompute from the ledger). They are kept separate rather than generalized because they
resolve different lifecycle columns on different tables.

**The collision this creates, and how it is handled:** `CashBankMovement.DocumentID` is now shared
between expense IDs and asset IDs, and both are small autoincrements — asset 5 and expense 5 exist
at once. `Type` is the discriminator: expense payments are type 9, asset payments type 10, and
`PostAssetPayment` filters on it when summing what has been paid. Summing by `DocumentID` alone
would silently mix the two.

## The inventory expense lives here, not in finance, because of an import cycle

**Context** — An inventory purchase has to write two things: an `Expense` and a stock movement.
The obvious home was `finance.PostExpenses`, next to the other expense handlers. It cannot go there:
`logistics` already imports `finance` (`purchase-order-management.go:391` calls
`ApplyCashBankMovement`), so a `finance` → `logistics` edge closes a cycle and the package will not
compile.

**Decision** — `POST.expense-inventory` lives in `accounting/inventory_expense.go`. `accounting` sits
above both and already imports both. `POST.expenses` stays in finance and now refuses Type 2 without
a `ProductID`, and refuses Type 4 outright.

**Rationale** — The alternative was pushing `ApplyMovimientos` down into a lower package purely to
break the cycle, which moves the stock engine somewhere it does not belong to satisfy a compiler.
The cost of this choice is that "create an expense" is now two endpoints rather than one, split by
whether the purchase touches stock. The split is real, though: one of them writes to the warehouse
and the other does not.

## Depreciation is a Status, not a Type filter

**Context** — Type-4 depreciation entries are `Expense` rows, and the expense tabs (Todos, Pend.
Pago, Pagados) query by `Status`. If depreciation rows carried `Status = 1` they would appear as
unpaid bills — for money that will never be paid, because none is owed.

**Decision** — Depreciation entries get `Status = 3` (`ExpenseStatusPosted`). `GetExpenses` maps its
tabs onto statuses 1 and 2 only, so they are excluded by the query that already exists.

**Rationale** — The alternative was adding `Type != 4` to every expense query. That is a filter to
remember in every new query written from now on, and forgetting it produces a wrong number rather
than an error. Making it a status means the existing queries exclude them without knowing that
depreciation exists. `PostExpensePayment` still rejects them explicitly, because a client can name
any ID it likes.

## The asset overlay is a separate table, and the stock row is not its identity

**Context** — The natural reading of "an asset is a supply" is that the asset *is* its stock row:
`ProductStock` when grouped, `ProductStockDetail` when serial-tracked. That is right about existence
and wrong about identity.

**Decision** — `accounting.Asset` (table 52) has its own autoincrement ID and points at the stock row
through `(ProductID, SerialNumber)`, with `WarehouseID` as a mutable mirror.

**Rationale** — Three properties of the stock schema forced it:

- `ProductStockDetail` has no monetary or temporal columns at all (`product-stock.go:100-115`).
  `Created` is a write timestamp, which is the wrong date for a donated or backdated asset.
- `ProductStockID` packs `WarehouseID` (`product-stock.go:75-79`), so transferring a laptop between
  warehouses changes the detail row's key. An accounting identity cannot move when the thing moves.
- `ProductStock.SelfParse` sets `Status = 0` at quantity 0 (`product-stock.go:39-41`). Disposal
  would therefore delete the row the balance sheet still has to read.

The cost is one table and one join's worth of indirection. `ProductStock` is the highest-traffic
table in the system, and this keeps four rarely-read accounting columns off it.

## Purchase and value are different columns, because a donated asset has no price

**Context** — An asset contributed by the owner cost the business nothing and is still worth
something. One number cannot say both.

**Decision** — `Asset.PurchaseAmount` is cash owed; `Asset.AcquisitionValue` is book value. A
donation has `PurchaseAmount = 0` and `PaymentStatus = AssetPaymentNone`, so it is never payable,
and it depreciates on its value like any other asset.

**Rationale** — Deriving "is this payable?" from the amount alone would make a zero-cost asset
indistinguishable from one whose price had not been entered yet. An explicit `PaymentStatus` says
which it is, and `canPayAsset` refuses a payment against a donation rather than accepting one that
can never be reconciled.

## Depreciation is generated lazily, and the ledger is written before the watermark

**Context** — Periods have to be materialized somehow. A cron is one more moving part to deploy and
monitor for a calculation nobody sees until they open the page.

**Decision** — `POST.asset-depreciation-run` posts everything due, and the Activos page calls it on
load. Each period is filtered against the asset's own `LastDepreciationDate`, so a second call the
same day writes nothing. The ledger entries are inserted *before* the assets are updated.

**Rationale** — The write order is the part worth remembering. If the asset update ran first and the
insert then failed, the watermark would have advanced past periods that were never written and they
would be lost silently. In this order a failure leaves the watermark behind, and the next run
re-posts them — a visible duplicate-free retry rather than an invisible gap.

## Depreciation starts the month after acquisition

**Context** — An asset bought on the 28th could earn a full month of depreciation, a prorated three
days, or nothing.

**Decision** — Nothing. The first period is the month following `AcquisitionDate`, and the final
period absorbs the integer-division remainder so the schedule sums to the acquisition value exactly.

**Rationale** — Full-month-on-acquisition overstates the first period; day-prorating means every
schedule depends on month lengths and produces amounts that do not repeat. Starting the month after
is the common convention and keeps every period identical but the last. Both rules are covered by
`depreciation_test.go`; the remainder rule is what makes the book value land on 0 rather than a few
cents either side.
