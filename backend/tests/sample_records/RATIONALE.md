# RATIONALE — sample_records

Design decisions for the sample-data generators, newest first.

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

## The assets' currency is taken from the till, not chosen

**Context** — assets carry a `CurrencyType`, and `PostAssetPayment` refuses a payment whose currency
differs from the cash bank's. Picking PEN by hand would fail on a company whose only active till is
in dollars.

**Decision** — the generator resolves the lowest-ID active till first and stamps its currency onto
every asset it creates.

**Rationale** — the constraint is between the asset and the till, so reading it off the till is the
only choice that cannot be wrong. It also keeps the donated half consistent with the bought half,
even though nothing is ever paid on it.

## `generate_supply_data` has no clock override and no resume file

**Context** — the other generator in this package, `generate_erp_history`, does both: it freezes the
process clock on each simulated day and checkpoints its progress to `tmp/`. The obvious move was to
copy that skeleton for the supply generator.

**Decision** — neither. `generate_supply_data` writes at the real clock and keeps no state file.
A `--dry-run` reads existing providers instead of seeding them, so it writes nothing at all.

**Rationale** — a replenishment policy is current state, not history: backdating a `product_supply`
row would only make its audit columns lie. And the resume file exists in the ERP generator purely
because purchase orders and sales are not idempotent — `product_supply` is keyed by `ProductID` and
written through `db.Merge`, so rerunning overwrites the same 2000 rows. A checkpoint would guard
against a duplication that cannot happen.

## The 50-client seed survives `generate_sale_orders`, renamed after its only remaining reader

**Context** — `generate_sale_orders` was deleted whole, but it owned the `//go:embed` of
`sale_order_clients.json`, and `generate_erp_history` reads that variable to seed its clients.
Deleting the file with the generator would have left the surviving generator without clients.

**Decision** — the JSON stays, renamed to `erp_history_clients.json` alongside
`erp_history_providers.json`, and the embed moved into `generate_erp_history.go` as
`erpClientSeedJSON`. `scripts/GENERATE_ERP_HISTORY.md` points at the new name.

**Rationale** — a data file named after a script that no longer exists is the same dead reference
the deletion was meant to remove. The rename costs one doc line; leaving it would cost every future
reader a search for a generator that isn't there.

## The deployer's Generators group loses its demo-data button, and nothing replaces it

**Context** — `generate_sale_orders` was the only `scriptGroupGenerators` entry that produced
sample records. Removing it leaves the TUI with no way to populate a fresh database.

**Decision** — removed the entry and did not add a `generate_erp_history` button in its place. The
`app.sh` usage line now advertises `generate_erp_history`, which is where the TUI used to point.

**Rationale** — `generate_erp_history` takes fifteen flags and a resume file; exposing it as a
one-click button with defaults invites an accidental 15-day write against whatever database the
deployer is pointed at. The scope asked for a removal, not a replacement — a button can be added
later with an `argumentsHint`, which is how the other parameterized scripts already do it.
