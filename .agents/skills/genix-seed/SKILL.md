---
name: genix-seed
description: Fill a company with dummy data (catalog, purchases, sales, expenses, payroll, projects, CRM). Locally or on a VPS.
version: 0.1.0
---

# Seeding dummy data (`genix-seed`)

Every dummy-data generator lives in `backend/seeds/` and ships as its own binary, `genix-seed`.
It writes **through the production handlers, in its own process** (no HTTP), so every validation
and side effect applies as from the frontend, and it backdates by freezing the process clock
(`core.SetHistoricalUnix`). Every database is dummy data: nothing stops you from seeding one.

The full reference (module table, day order, expenses, HR calendar, every flag) is
`backend/seeds/README.md`; the design decisions are in `backend/seeds/RATIONALE.md`. Read the README
before changing a module.

## Before the first run

1. `fn-init` has run on the database (company, admin user 1, cities, first site/warehouse/till) and
   the schema is current: `cd backend && go run . fn-homologate`.
2. `git lfs pull` — otherwise `base/products.xlsx` is an LFS pointer and `base` fails.
3. Projects module in dev: export `FAREWARD_ADDRESS=127.0.0.1:14013` (the dev stack's daemon), or
   the first project lock answers 503.

## Commands

From the repo root (`./app.sh seed …` = `cd backend && go run ./seeds/cmd/genix-seed …`):

```bash
./app.sh seed list                          # modules and what they require
./app.sh seed run all --dry-run             # pre-flight + resume point, writes nothing
./app.sh seed run all                       # whole demo company, last 60 days (asks to confirm)
./app.sh seed run sales,hr --days=5 --yes   # base + logistics + sales + hr, 5 days, no prompt
./app.sh seed status [--company=N]          # resume point an aborted run left
./app.sh seed wipe [--company=N]            # dry run: what would be deleted
./app.sh seed wipe --apply [--only=hr_]     # empty the company's ERP data (optionally by table prefix)
./app.sh seed loadtest [--flag=value …]     # HTTP load test against a running server (seeds/loadtest/plan.md)
```

**Running non-interactively (as an agent): always pass `--yes`.** Without a terminal the
confirmation fails with "sin terminal para confirmar: agrega --yes". Do a `--dry-run` first and
read the banner: host, `DB_NAME`, company, whether it already has sales, date range, and whether
the run is new or resumes.

Seeding is slow (hundreds of sales per day through the real handlers). For a quick check use
`--days=2..5` and small volumes, e.g. `--sales-min=20 --sales-max=40 --products=100 --po-products=40
--unpaid-max=5 --undelivered-max=5`. Run long seeds with `run_in_background`.

## Modules

`run` takes a comma list or `all`; requirements are added and `base` is always first.

| Module | Requires | What it writes |
|---|---|---|
| base | — | legal payroll params, warehouses 1–2, tills + bank, F001/B001 series, clients/providers, `products.xlsx` catalog, supplies |
| logistics | — | replenishment settings; daily purchase orders → factura → payment → reception |
| sales | logistics | sales planning, shipping costs; daily sales and annulments |
| finance | — | expense schedules; daily expenses, till → bank deposits, month-end till counts |
| accounting | — | 100 fixed assets; month-end depreciation run |
| hr | — | staff, concepts, loans; the payroll calendar |
| projects | — | agile projects sprint by sprint, checking invariants after every write (also a regression test) |
| crm | — | custom fields, pipelines, prospects, a year of opportunities |

Setups are idempotent and skip what already exists (e.g. hr skips if the company has employees,
crm if it has an active pipeline). The day steps of all selected modules run in **one** day loop in
a fixed order (purchases → expenses → sales → annulments → till transfers → payroll → till counts →
depreciation), because a cash movement stores its account balance at write time. At the end every
touched cash account is rebalanced in date order.

## Common flags

| Flag | Default | |
|---|---|---|
| `--company=N` | 1 | |
| `--days=N` | 60 | days back, today included |
| `--seed=N` | 20260101 | same args + same real date → same data |
| `--warehouses=1,2` | 1,2 | one PO and one till per store |
| `--products` / `--po-products` | 400 / 150 | product pool / lines per purchase order |
| `--sales-min` / `--sales-max` | 200 / 400 | sales per day |
| `--dry-run` / `--reset` / `--yes` | | plan only / drop resume point / skip confirmation |

Ranges are validated before the first write (`runner/config.go` `ParseArgs`); the full list is in
the README.

## Resume, abort, lock

- The resume point is `<file store>/seed-state/c<company>_genix_seed_state.json`, saved after every
  write. **Rerun the exact same command to resume.** It stores the plan's signature (modules +
  flags): different arguments are refused — use `--reset` to start over, or `status` to see the plan.
- Ctrl+C / SIGTERM stops at the next checkpoint (exit 130, "abortado; repite el mismo comando para
  reanudar"). A second signal exits at once.
- One run per company: "otra siembra está corriendo sobre la empresa N" means a `flock` is held by
  another `genix-seed` process.
- "el estado guardado … deja el rango vacío; usa --reset": the saved day is outside the current
  `--days` window (e.g. the real date moved on) — rerun with `--reset`.
- The seed turns off the invoice emit sweep, so the running `genix` server never sends the seeded
  documents to SUNAT. The server can stay up.

## Wipe

`wipe` empties one company's ERP data (every company-partitioned table except users, profiles,
parameters, secrets, metering and webpage tables) with its delta rows, snapshots, text index,
DataFrames and counters, then recreates the site/warehouse 1/till 1 that `fn-init` makes and
republishes the company config. It also deletes the resume point. Dry run unless `--apply`.
After a wipe, tell the user to clear the browser's site data — delta caches survive it.

Typical fresh demo: `./app.sh seed wipe --apply` then `./app.sh seed run all --yes`.

## On a VPS (`seed_remote`)

Runs next to the database (one local round trip per write instead of one across the internet):

```bash
./app.sh seed_remote run all --days=60          # build, upload, start, follow logs
./app.sh seed_remote --server=1.2.3.4 run hr    # pick one when config.toml has several [[servers]]
./app.sh seed_remote logs | status | abort
```

It cross-compiles for the `[[servers]]` entry's arch, uploads to `/usr/local/bin/genix/genix_seed`
and starts a transient `systemd-run --unit=genix-seed` with the `genix` unit's user, working dir and
`GENIX_*` env (it adds `--yes` itself). The run survives SSH drops; Ctrl+C only stops the log tail
and opens a menu (abort / follow again / detach). `abort` sends SIGTERM → stops at the next
checkpoint. Confirm with the user before seeding or wiping a remote server.

## Adding or changing a seed

- One package per domain under `backend/seeds/<domain>/`, data files embedded next to it.
- Export a `runner.Module{Name, Requires, Setup}` and register it in `seedModules` in
  `backend/seeds/cmd/genix-seed/main.go` (after its requirements; `base` stays first).
- `Setup(run *runner.Run) ([]runner.DayStep, error)` runs on **every** invocation (including
  resumes) with the clock on the first simulated morning: it must be idempotent — check what is
  stored before writing.
- Return day steps with an `Order` from the `runner.Order*` constants in `runner/module.go`. Anything
  that moves cash **must** be a day step in that loop, not a separate pass.
- Write through the production handlers: `run.MakeRequest(route, query, body)` then
  `run.Send(request, handler)` and `runner.DecodeResponse` (`runner/run.go`; `runner.CallAs` to act
  as another user). Never insert rows directly, and never use `time.Now()` for a date — the frozen
  clock is `core.Now()`.
- Draw randomness only from `run.Random` so runs stay reproducible; save state (`run.SaveState()`)
  after each non-idempotent write inside a long step.
- New flags go in `runner.Config`, `DefaultConfig` and `ParseArgs` (with validation), and in the
  README's flag table.
- Verify with `go build ./... && go test ./seeds/...` from `backend/`, then a short real run
  (`--days=2 --yes` on a wiped company) and, for cash-moving steps, an abort + resume.
