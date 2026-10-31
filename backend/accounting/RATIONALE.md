# RATIONALE — accounting

Design decisions behind the asset register, depreciation and the PCGE chart, newest first.
Full design in `PLAN.md`; the chart's plan is `frontend/routes/accounting/accounts-consolidation/PLAN.md`.

## A record saved with AccountCode 0 gets the default on the server
**Context** — The forms preselect the defaults, but other writers (the express PO entry, imports,
the agent) send no account. The plan says 0 means "unclassified".
**Decision** — Every writer resolves 0 to the default before saving (`ResolveSupplyAccount`,
`ResolveAssetAccount`, the category default). Only Taxes/Other expenses stay 0.
**Rationale** — The default is the same one the form would have proposed, so the result does not
depend on which client wrote the record. Cost: the server cannot tell "user accepted the default"
from "client sent nothing". The page can't show that difference.

## A supply's default account is 3369 when it depreciates, else 6032
**Context** — The plan named typical values but not the fallback for a supply saved without one.
**Decision** — `DefaultSupplyAccountCode`: `DepreciationMonths > 0` → 3369 otros equipos diversos;
otherwise 6032 materiales auxiliares, suministros y repuestos. The backfill uses the same rule.
**Rationale** — The depreciation months already say whether the material is an asset. 6032 is the
broadest stock account for consumables. A wrong guess is one selector change.

## Editing an asset requires an account; creating one does not
**Context** — Decision 8 says an asset with no account defaults to generic 33x.
**Decision** — On acquisition, 0 resolves to the supply's account if it is an asset account, else
3369. On edit the selector is required and validated as an asset account.
**Rationale** — An existing asset was already resolved, so a 0 on edit can only mean the user
cleared it. Rejecting that is safer than silently falling back to 3369.

## The PCGE chart is transcribed from an audited dataset, and the plan's account codes were corrected against it

**Context** — The chart is legal data. Writing ~550 codes and names from memory would put errors
into PLE 5.4. Checking against the official list also showed that several codes in the plan were
wrong: 7011 is *export* sales (local is 7012), 6911/6921 are export cost of sales (local
6912/6922), 4651 is investments (an asset purchase is 4654), and depreciation is 3952x/6841x, not
39x/681x. 6563 does not exist (656).
**Decision** — `pcge_accounts.go` was generated once from `pcge-peru` (github.com/mat-l-dev/pcge-peru,
`data/2019/entries.json`, 1 757 entries extracted from the MEF PDF), then curated: every 2/3-digit
account, plus the 4/5-digit subaccounts of 104 121 141 201 211 333–336 395 401 411 421 465 469
601 603 604 611 613 614 621 631 632 634–637 641 655 684 691 692 701–703 711 and 4011 4017 4018
3952 6841. Names are official except the spacing typo in 6924. The generator script was not
committed: the Go file is the source from now on, and `TestPCGEAccountsAreConsistent` enforces the
rules. The plan's codes were fixed.
**Rationale** — An extracted, checksummed dataset is far more reliable than recall, and keeping
only the output avoids a second source of truth. Cost: the subaccount selection is mine. Anything
missing is one row plus its parent losing `Postable`. Account 565's official name is 121
characters, over PLE 5.4's limit of 100, so the 5.4 writer has to truncate it.

## 501 Capital social is a leaf, and replaces 50 for capital contributions

**Context** — The cash "capital contribution" option stored `50`, a 2-digit element. PLE posts at 3
digits or more. 501's children (5011 Acciones / 5012 Participaciones) depend on the company's
legal form, which the ERP does not record.
**Decision** — 501's children are left out, so 501 is postable. `PCGECapital` is 501, and
`fn-backfill-cash-movement-accounts` rewrites stored 50s to 501 (`retiredCashMovementAccounts`).
**Rationale** — It is correct for every legal form. Cost: a company that wants 5011 vs 5012 needs
those rows added.

## The cash movement's account was renamed to CounterAccountCode to widen it

**Context** — `CashBankMovement.AccountCode` was `int16` (Scylla `smallint`), which cannot hold
5-digit codes such as 40111 (max 32767). CQL cannot `ALTER` a column's type, and the ORM only logs a
type mismatch.
**Decision** — The field became `CounterAccountCode int32`, so it gets a new `int` column.
`fn-rename-columns` gained `columnTypeChanges`, which creates the new column with the new type and
copies the old values in. The option field `CashMovementAccountOption.AccountCode` keeps its name.
The request and TS fields of the movement follow the rename.
**Rationale** — A rename is the only way to change a type, and the old comment already called it
"the account on the other side of the cash". Cost: the migration drops and recreates the table's
views and indexes (`fn-homologate`).

## Which accounts each kind may pick

**Context** — The plan's prefix list was a sketch ("60, 63, 65, 33" for supplies; "62–68, 40" for
expenses).
**Decision** — Supply: 602, 603, 604, 656, 33. Not 63: those are services, not goods. Expense: 62,
63, 64, 651–654, 656, 658, 659, 67, 40. 655 (net cost of a disposal) and 68 (depreciation,
provisions) are derived by the system. 66 (fair-value losses) is not an SME expense. Asset: 32,
33, 34.
**Rationale** — The user should not be able to post something the ledger derives on its own,
which would count it twice. Cost: an unusual expense in 66 or 68 is not pickable.

## finance repeats the cash-movement account codes instead of importing the chart

**Context** — `accounting/types` imports `finance/types`, so finance cannot import the chart back.
**Decision** — The `PCGE*` constants stay in `finance/types/cash_movement_account.go`, and
`TestCashMovementAccountsAreInTheChart` (in accounting) fails if any of them is not a postable
account of the chart.
**Rationale** — No import cycle and no new shared package. The test removes the risk of the copy
drifting.

## Editing an asset rewrites its depreciation ledger in place

**Context** — `DepreciationSchedule` is a pure function of `AcquisitionValue`, `AcquisitionDate`
and `DepreciationMonths`. `PUT.asset` makes the first two editable, so the moment either changes
every Type-4 `Expense` already posted for the asset describes a schedule that no longer exists —
and the ORM has no `Delete`, so those rows cannot simply be dropped.

**Decision** — `rewriteAssetDepreciation` reads the asset's live Type-4 rows, sorts them by `Date`,
and pairs them index-for-index against the schedule the *new* values produce: the first
`min(old, new)` rows have their `Date` and `Amount` overwritten, surplus periods are inserted, and
surplus rows are retired with `Status = 0`. `AccumulatedDepreciation` and `LastDepreciationDate`
then take the last new period's values, or zero when nothing has elapsed under the new dates.
`GetAssetDepreciation` filters `Status != 0` so the retired tail leaves the panel.

**Rationale** — Chosen over retiring the whole ledger and regenerating it, which was the simpler
implementation: with no `Delete`, a wholesale rebuild leaves one dead row per period *per edit* in
`expenses` forever, and shows the expense register a removal followed by an insert rather than an
amount change. Pairing by date index costs about twenty lines more and keeps the row IDs stable.
Chosen over a forward-only adjustment — spreading the difference over the months still to come —
because that needs a basis column on the table and destroys the purity of `DepreciationSchedule`,
and because an edit here is a correction of a data-entry mistake, not a revaluation: the register
should end up looking like the asset was entered correctly the first time. The cost is real, and
deliberate: the P&L of a closed past month changes.

## An asset edit refuses to lower the purchase amount below what was paid

**Context** — `Asset.PaidAmount` is recomputed from the `CashBankMovement` ledger and this handler
never touches it. A new `PurchaseAmount` under it would leave the asset over-paid with no record
of why.

**Decision** — Rejected, with the paid amount named in the message: *"El monto de compra no puede
ser menor a lo ya pagado (X). Revierta el pago antes de reducirlo."*

**Rationale** — The alternative is to clamp and mark the asset `Paid`, which loses the discrepancy
silently. Reversing cash is a cash operation; an asset edit does not get to do it implicitly.

## Changing a serial moves the stock rather than relabelling it

**Context** — `ProductStockDetail`'s key is `(ProductStockID, LotID, SerialNumber)`, so a serial
cannot be updated in place at all. Writing `Asset.SerialNumber` alone would leave the asset naming
a stock detail row that does not exist, with the old serial still holding the units.

**Decision** — A serial change posts two `InternalMovement`s through `ApplyMovimientos`:
`-Quantity` on the old serial, `+Quantity` on the new one. An empty serial is the no-detail bucket,
so adding or clearing a serial is the same pair. A serial change on a *disposed* asset is rejected —
its units are already out of stock, so there is nothing to move — and a serial another live asset
of the same material carries is rejected too, since both would claim one detail row.

**Rationale** — The same engine acquisition and disposal already use, so stock stays consistent
without a second code path. Cost: two rows in the movement ledger per typo corrected.

## The acquisition movement's price is left stale after an edit

**Context** — `PostAsset` writes the inbound stock movement with `Price = AcquisitionValue`.
Correcting the value through `PUT.asset` does not go back and fix it.

**Decision** — Left as it was. Only the serial change re-posts movements, and it carries the new
per-unit value on the inbound leg because it is writing a fresh movement anyway.

**Rationale** — Rewriting a historical movement to correct a price is a larger change than the edit
warrants, and the movement ledger is not what the balance sheet reads — `Asset.AcquisitionValue`
and the depreciation entries are. Recorded here because it is a real inconsistency, not an oversight.


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
