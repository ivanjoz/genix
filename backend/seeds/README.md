# genix-seed

Every dummy-data generator, split by domain, in its own binary. It writes through the production
handlers in its own process, so every validation and side effect applies as it does from the
frontend, and it backdates by freezing the process clock. The production binary carries none of it.

There is no production data anywhere: every database is dummy data, and nothing guards against
seeding one.

```text
cmd/genix-seed/   the CLI, plus wipe
runner/           Run (the shared context), config and flags, day loop, resume state, abort, lock
base/             always first: legal parameters, warehouses, tills + bank, invoice series,
                  clients/providers, products.xlsx, supplies catalog
logistics/        replenishment settings, purchase orders → supplier factura → payment → reception
sales/            sales planning, shipping costs, daily sales, annulments
finance/          expenses and their payments, till → bank deposits, month-end till counts
accounting/       fixed assets (bought, donated, part paid) and the month-end depreciation run
hr/               staff, concepts, loans, advances, vacations, novedades, the monthly payroll run
projects/         agile projects lived sprint by sprint, checked after every write
crm/              custom fields, pipelines, prospects, a year of opportunities
loadtest/         drives a running server over HTTP with the same flows (see loadtest/plan.md)
```

Each package embeds its own data files (`*.json`, `*.psv`, `products.xlsx`), so the binary is
self-contained.

## Running it

Locally, against the database of `config.toml`:

```bash
./app.sh seed list                               # modules and their requirements
./app.sh seed run all --dry-run                  # print the plan, write nothing
./app.sh seed run all                            # the whole demo company, last 60 days
./app.sh seed run sales,hr --days=5              # base, logistics, sales and hr; 5 days
./app.sh seed status                             # the resume point an aborted run left
./app.sh seed wipe --apply                       # empty company 1's ERP data
```

(`./app.sh seed …` is `cd scripts && go run . seed …`, which is `cd backend && go run ./seeds/cmd/genix-seed …`.)

On a VPS, next to its database: each write is a local round trip instead of one across the internet,
and files land in the server's own file store.

```bash
./app.sh seed_remote run all --days=60           # build, upload, start, follow
./app.sh seed_remote --server=1.2.3.4 run hr     # several [[servers]]: pick one
./app.sh seed_remote logs | status | abort
```

`seed_remote` cross-compiles `genix-seed` for the `[[servers]]` entry's architecture, uploads it to
`/usr/local/bin/genix/genix_seed`, and starts it as a transient unit (`systemd-run
--unit=genix-seed`) with the `genix` service's own user, working directory and `GENIX_*`
environment, so it reads the same `config.toml`. The run survives the SSH session. Ctrl+C while
following only stops the tail; a menu then offers to abort the run, follow again or detach.

## Modules

`run` takes a comma-separated list or `all`. Requirements are added, and `base` is always included.

| Module | Requires | Setup (once per invocation, idempotent) | Day steps |
|---|---|---|---|
| base | | legal parameters (per country), warehouses 1–2, a till per store (the first keeps the company's first till, the others are `Caja Tienda N`) + the bank `BCP Cta. Cte. Soles`, default F001/B001, providers and clients from `erp_history_*.json`, `products.xlsx`, 100 supply providers, 100 supply materials, product pool | |
| logistics | | replenishment settings for the first N products (min stock, sales/day, 2 providers each) | purchase orders |
| sales | logistics | sales planning (skipped if present), shipping costs for 20 Lima districts | sales, annulments |
| finance | | expense schedules and suppliers | expenses, till transfers, till counts |
| accounting | | 100 assets from `assets.psv` (skipped if the company has 100) | month-end depreciation |
| hr | | staff (skipped if the company has employees), concepts, recurring items, a loan | payroll |
| projects | | the whole projects timeline (skipped if the company has projects) | |
| crm | | the whole CRM history (skipped if the company has an active pipeline) | |

Setups run with the clock frozen on the first simulated morning (06:00), so what they write
predates the history. projects and crm keep their own clocks: their timelines (sprints back from
today, a year of opportunities) do not fit the day loop.

### One day loop

A cash movement stores its account's balance **at write time** (`FinalAmount`). Modules replayed one
after another over the same days would leave the running balances out of order, so the selected
modules' steps are merged into one loop and every day runs them in this order, the clock frozen on
each write (`core.SetHistoricalUnix`):

| Order | Step | Time | Route |
|---|---|---|---|
| 10 | Purchase order per store → confirm → supplier factura (base + 18% IGV) → payment from the bank → reception (free, lot and per-unit serial stock) | 08:00 | `POST/PUT.purchase-orders` (actions 1, 5, 3), `POST.purchase-order-entry` |
| 20 | Expense payments due today, then the day's expenses, paid now or queued | 10:15–10:30 | `POST.expense-payment`, `POST.expenses`, `POST.expense-inventory` |
| 30 | 200–400 sales into the store's till; 1 in 8 a factura to a RUC client, the rest boletas (S/ 700+ gets a DNI client); some unpaid, some undelivered | 11–20h | `POST.sale-order` |
| 40 | 2–5 annulments, refunded from the till | 20:30 | `POST.sale-order-annul` |
| 50 | Each till's takings minus a S/ 500 float, deposited in the bank | 21:00 | `POST.cash-banks-movement` (type 3) |
| 60 | The HR day, paid from the bank | 21:15 | `POST.hr-*` |
| 70 | Month end: till counts (usually exact, sometimes short → 1419 or over → 759) | 21:30 | `POST.cash-banks-reconciliation` |
| 80 | Month end: the depreciation run | 22:00 | `POST.asset-depreciation-run` |

A capital contribution (type 7, account 501) tops up an account that can't cover a payment. When
the run ends, the bank and every till get their running balances recomputed in date order
(`RebalanceCashBank`), which also covers a module run over days that already had movements.

Stock: the run keeps an in-memory ledger per warehouse with the three buckets the backend validates
apart (free, lot, serial), and every sale line names its bucket. On a stock refusal the ledger is
reloaded and the sale rebuilt with other products, up to 3 times.

### Expenses

`expenseTemplates` in `finance/expenses.go`, suppliers in `finance/expense_suppliers.json`.

| Expense | PCGE | Document | When |
|---|---|---|---|
| Rent, store 1 / store 2 | 6352 | Factura | schedule, day 1 |
| Electricity / water | 6361 / 6363 | Utility bill (14) | schedule, day 8 / 12, paid at +12 / +14 days |
| Internet / mobile phone | 6365 / 6364 | Factura | schedule, day 5 / 18 |
| External accountant | 6323 | Fee receipt (2), no IGV | schedule, day 28 |
| Security / cleaning | 638 | Factura | schedule, day 28 |
| POS software license | 639 | Factura | schedule, day 3 |
| Advertising / refrigeration maintenance | 6371 / 6343 | Factura | ~7% / ~4% of days |
| Freight | 6311 | Boleta | Saturdays |
| Supplies into stock | the supply's account | Factura | ~30% of days |
| Bank fees / ITF | 639 / 6412 | none | last day of the month |
| Municipal fees | 643 | none | quarterly, day 28 |
| IGV for the previous month / income-tax prepayment | 40111 / 40171 | none | day 15 |

IGV is 18/118 of the previous month's gross sales minus the IGV of its purchase and expense
documents; the prepayment is 1.5% of net sales. The first simulated month pays neither. 85% of
expenses are paid in full, 10% half, 5% left pending.

### HR

| Day | Action |
|---|---|
| 1st | Two workers with a year of service book vacation for the 10th–19th |
| 7th, 14th, 21st, 28th | The two weekly-paid workers get the week (S/ 250 each) |
| 20th | About one month in three, a worker resigns |
| 25th | The month's novedades: overtime 25% and 35%, absences, bonuses |
| Month end | The month's single run (gratificación in July/December, CTS in May/November, settlements), calculated, closed and paid |
| Last day | The current month's run, calculated and left open |

About 23 employees in 4 departments, mixing ONP and the four AFPs (flow and mixed commission), EPS,
part-time, fixed-term contracts, raises and dependents. Each day's randomness is the seed plus the
day, and every action first checks what is stored, so the same days never write twice. Not seeded:
paying the payroll contributions, profit sharing. AFP rates exist for 2026 only.

### Accounting

`assets.psv` holds 100 acquisitions of depreciable materials from `base/supply_materials.psv` (40
equipment rows with `DepreciationMonths > 0`, 60 consumables). A row with a serial is one tracked
asset; one without is a grouped asset of N units. The first 50 are bought (about 25 paid, 15 part
paid, 10 unpaid, from the bank), the last 50 donated. `AcquisitionDate` is spread 30–1080 days back
through the payload, so the schedules have posted periods. A test keeps the two files consistent.

### Projects

Simulates projects end to end and checks every step, so it is also a regression test of the
module: after each write the project is read back through the browser's GET routes and checked for
the step's outcome and every project invariant (`checkProjectInvariants`). The first failure stops
the run and names the step. It creates the catalog's 4 people (password `genix2026`). In dev, every
write takes the project's lock in fareward: run it with `FAREWARD_ADDRESS=127.0.0.1:14013` (the dev
stack's daemon), or the first lock answers 503.

### CRM

3 custom fields, 4 lost reasons, 2 pipelines with members, a team and goals, 8 prospects, and 300
opportunities over the last 365 days, each living 2–30 days: moved stage by stage, sometimes back,
renegotiated, handed over, won, lost, reopened, or moved to the other pipeline.

## Flags

One surface for every module, so the same command line resumes the same plan.

| Flag | Default | |
|---|---|---|
| `--company=N` | 1 | |
| `--days=N` | 60 | days back, today included |
| `--seed=N` | 20260101 | the random seed |
| `--warehouses=1,2` | 1,2 | one purchase order and one till per store |
| `--products=N` | 400 | the pool the day loop buys and sells |
| `--po-products` / `--lot-products` / `--serial-products` | 150 / 15 / 10 | per purchase order |
| `--sales-min` / `--sales-max` | 200 / 400 | sales per day |
| `--sale-lines-min` / `--sale-lines-max` | 3 / 8 | lines per sale |
| `--unpaid-*` / `--undelivered-*` / `--annul-*` | 5–20 / 5–20 / 2–5 | per day |
| `--supply-products` / `--providers-per-product` | 2000 / 2 | replenishment settings |
| `--min-stock-*` / `--sales-per-day-*` | 2–50 / 0–20 | |
| `--supply-materials` / `--assets` | 100 / 100 | also the "already seeded" threshold; 0 skips |
| `--projects` / `--sprints` / `--stories` / `--sprint-days` | 2 / 4 / 20 / 14 | |
| `--dry-run` | | print the plan and the resume point, write nothing |
| `--reset` | | drop the resume point and start over |
| `--yes` | | skip the confirmation (required without a terminal) |

Ranges are validated up front, so a bad ratio fails before the first write.

## Resume, abort, lock

- The resume point is `<file store>/seed-state/c<company>_genix_seed_state.json`, rewritten (by
  atomic rename) after every day step and inside a step after each purchase order, sale, expense
  and annulment. The history is not idempotent, so the point is saved **after** each write.
- It carries the plan's signature (modules + flags, minus `--reset/--dry-run/--yes`): resuming with
  other arguments is refused rather than mixed. A finished run deletes it; `wipe` does too.
- Ctrl+C or SIGTERM stops the run at its next checkpoint: rerun the same command to resume. A second
  signal exits at once.
- One run per company at a time (a `flock` beside the state file).
- Reproducible: the same arguments on the same real date produce the same data. A resumed run
  restarts the random sequence where it resumed.
- The seed turns off the invoice emit sweep (`invoicing.EmitSweepDisabled`), so the running server
  never sends the seeded documents to SUNAT. The `genix` service can keep running.

## Wipe

`genix-seed wipe` empties one company's ERP data: every company-partitioned table except users,
profiles, parameters, secrets, metering and webpage tables, with their delta rows, snapshots, text
index, DataFrames and counters (and the `cpe_<id>_*` correlativos). Then it seeds the site,
warehouse 1 and till 1 `fn-init` would and republishes the company config. Dry run unless
`--apply`; `--only=hr_` limits it to tables by prefix. Browsers keep their delta caches: clear site
data.

## Requirements

- `fn-init` ran (company, admin user 1, cities, first site/warehouse/till) and the schema is
  current (`fn-homologate`).
- `git lfs pull` before building: without it `products.xlsx` is the LFS pointer and base stops.
