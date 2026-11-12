# RATIONALE — seeds

Design decisions for the dummy-data generators, newest first.

## Seeds are their own binary, split into modules over one shared Run (2026-11-11)
**Context** — the generators lived in `tests/sample_records` behind `fn-generate-*` commands of the
production binary, one god-struct (`erpHistoryGenerator`) per flow, and a remote database meant
every write crossed the internet.
**Decision** — `backend/seeds/` builds `genix-seed`. One `runner.Run` holds the shared context and
one `State` the resume point; each domain package is plain functions over `*runner.Run` and exports
a `Module` whose `Setup` returns its day steps. `run` merges the selected modules' steps into one
day loop ordered by `runner.Order*`. `seed_remote` uploads the binary and runs it as a transient
systemd unit beside the database.
**Rationale** — cash movements store their running balance at write time, so modules that move cash
cannot be replayed one after another; merging their steps keeps one loop without coupling the
packages. A Setup that returns its steps lets a step close over what its setup loaded instead of
parking it in the shared struct.

## base.Setup is the legal parameters plus an exported SetupCompany
**Context** — the load test onboards many companies concurrently over HTTP. The HR legal parameters
are per country, written in-process, and the load test never runs a payroll.
**Decision** — `base.Setup` writes the legal parameters, then calls `SetupCompany`; the load test
calls only `SetupCompany`. Everything a concurrent onboarding touches reads the company from
`run.UserToken`, never the process-wide `runner.CompanyID`.
**Rationale** — concurrent onboardings would race to insert the same legal parameter rows.

## The seed turns the invoice emit sweep off in-process
**Context** — every seeded sale enqueues `ScheduleEmitPendingSweep`, a cron row the running `genix`
service executes; it would send backdated dummy documents to SUNAT and mark them failed. The old
workaround was stopping the service while seeding.
**Decision** — `invoicing.EmitSweepDisabled`, set by `genix-seed` at startup, makes the enqueue a no-op.
**Rationale** — the service keeps serving while a remote seed runs, and the flag is one branch in the
enqueue rather than a change to the sweep.

## No sale-frame rebuild at the end of a run
**Context** — the plan had the runner rebuild the sale DataFrames of the affected days, replacing
the old `unixDaysToReprocess` hint for cron action 2.
**Decision** — dropped. Cron action 2 no longer exists: the sale frames are maintained on write, and
`fn-rebuild-sale-frames` is only for a frame version change.
**Rationale** — a rebuild would redo work the writes already did.

## Behavior changes from the fn-generate-* era
- `--dry-run` prints the plan and the resume point; it no longer builds first-day payloads.
- Supply data shares `run.Random` instead of its own source, so totals differ from old runs with the
  same seed.
- The HR module's `--months` is gone: every module uses `--days`.
- The single-register mode is gone: every run uses one till per store plus the bank.
- projects and crm are modules of `run all`; each skips when its data already exists.

## nosync_preload.c sits in loadtest/nosync/
**Context** — the load test's Go code and its benchmark scripts share `loadtest/`, and cgo refuses a
`.c` file inside a package built without cgo.
**Decision** — the OceanBase redo shim moved to `loadtest/nosync/`; `run_seekdb.sh` and
`deploy_oceanbase.sh` point there.

## minijson decodes into a typed mirror by JSON field name (v0.3.2)
**Context** — `reloadWarehouseLedger` decodes `GET.productos-stock`, whose `ProductStockDetail` field is
`[]any`, into a local struct where that field is typed. minijson took field types from the payload's
source type, so the typed mirror panicked in `reflect.Set`.
**Decision** — In minijson, when the destination struct differs from the encoded one, fields are matched
by JSON name and decoded with the destination's types; values with no matching field are skipped.
**Rationale** — Keeps the generator decoding through the same wire format the frontend reads instead of
adding a typed endpoint; the extra cost is one name map per destination type per decode.

## The asset seeds are guarded by a count, and each row produces exactly one asset

**Context** — the request was 100 supplies with a count guard, and 100 assets split 50 bought / 50
donated. Two things it did not say: whether the assets get the same guard, and what "100 assets"
means when `POST.asset` turns one acquisition with three serial numbers into three asset rows.

**Decision** — assets get the same guard as supplies (skip when the company already has 100 or
more), and every row of `assets.psv` produces exactly one asset: a row with a serial is one
individually tracked unit, a row without one is a single grouped row of N units. The guard
threshold is `>=`, not the literal `> 100` that was asked for.

**Rationale** — neither `POST.supply-material` nor `POST.asset` matches on anything: both always
insert. Without a guard on assets, a second run would leave 200 of them, and with a `> 100`
threshold the run that had just written exactly 100 supplies would write 100 more. Keeping one
asset per row is what makes "50 bought and 50 donated" land exactly, and mixing the two
granularities still exercises both branches of `PostAsset`.

## Acquisitions are backdated through the payload, not the clock

**Context** — the point of the asset seeds is to check that amortizations and payments appear.
`PostAssetDepreciationRun` only posts periods that have already come due, so assets acquired today
produce an empty schedule. `generate_erp_history` solves its own dating problem by freezing the
process clock.

**Decision** — the clock is left alone; `AcquisitionDate` is set directly on the payload, spread
between 30 and 1080 days back.

**Rationale** — `AcquisitionDate` exists precisely because it is not `Created`: the type's own
comment says a donated or backdated asset is registered long after it was acquired. Freezing the
clock would additionally backdate the audit columns and the cash movements of the payments, which
is a lie about when the data was seeded. Three years of spread is what produces assets with a
couple of posted periods, others with thirty, and a few already fully depreciated.

## The assets' currency is the bank's, and the bank pays them

**Context** — assets carry a `CurrencyType`, and `PostAssetPayment` refuses a payment whose currency
differs from the cash bank's. The first version read the lowest-ID active till.

**Decision** — the assets are paid from the bank account base creates (`run.BankCashBankID`) and
take its currency.

**Rationale** — the constraint is between the asset and the account that pays it, and with one till
per store plus the bank, the bank is what pays suppliers, expenses and payroll. It also keeps the
donated half consistent with the bought half, even though nothing is ever paid on it.

## Replenishment settings have no resume file

**Context** — the day loop checkpoints its progress; the replenishment settings run in the
logistics Setup.

**Decision** — no checkpoint. (They are now written with the Setup clock, the first simulated
morning, like every setup.)

**Rationale** — the resume file exists because purchase orders and sales are not idempotent —
`product_supply` is keyed by `ProductID` and written through `db.Merge`, so rerunning overwrites the
same rows. A checkpoint would guard against a duplication that cannot happen.

## The 50-client seed survives `generate_sale_orders`, renamed after its only remaining reader

**Context** — `generate_sale_orders` was deleted whole, but it owned the `//go:embed` of
`sale_order_clients.json`, and `generate_erp_history` reads that variable to seed its clients.
Deleting the file with the generator would have left the surviving generator without clients.

**Decision** — the JSON stays, renamed to `erp_history_clients.json` alongside
`erp_history_providers.json`; its embed now lives in `base/parties.go`.

**Rationale** — a data file named after a script that no longer exists is the same dead reference
the deletion was meant to remove.

## The deployer has no seed button

**Context** — the deployer TUI's Generators group once had a demo-data button.

**Decision** — none; `./app.sh seed` and `seed_remote` are the entry points.

**Rationale** — a run takes many flags and a resume file; a one-click button with defaults invites
an accidental 60-day write against whatever database the deployer is pointed at. A button can be
added later with an `argumentsHint`, as the other parameterized scripts do.
