# SEEDS PLAN — dummy data generators as their own binary

## Goal

Every seed and data generator lives in `backend/seeds/`, split by domain, and ships as its own
binary (`genix-seed`) that is uploaded to the server next to the database and run there. Modules
run separately, always on top of a `base` seed that holds the cross-module catalog. A remote run
can be watched, detached from, and aborted without corrupting what was written.

There is no production data anywhere: every database is dummy data, so no environment guard.

## Constraints the design follows

1. **Seeds run in-process, never over HTTP.** Every generator calls the production handlers
   directly (`send` → `handler(&request)`) and backdates by freezing the whole process clock
   (`core.SetHistoricalUnix`). A running server cannot freeze its clock per request, so seeding
   needs its own process with the handlers linked in. The binary is therefore about the backend's
   size; what is gained is that the production binary stops carrying seeds and their data.
2. **Modules that move cash share one day loop.** A cash movement stores its register's balance
   at write time. Purchases, sales, expenses, payroll and asset payments all move cash, so running
   them one after another over the same days leaves the running balances out of order (today
   `fn-seed-hr` alone has to `rebalanceCashBank` afterwards). Selecting modules therefore merges
   their day steps into one loop, in a fixed global order.
3. **The resume state, `--dry-run`, `--reset`, `--company` and the fixed random seed stay.**

## Layout

```
backend/seeds/
  cmd/genix-seed/main.go   CLI: list | run | status | wipe | loadtest
  runner/                  day loop, clock freeze, resume state, abort, lock, args, send()
  base/                    always first, idempotent
  logistics/               replenishment settings, purchase orders, receptions
  sales/                   sales, annulments, shipping costs, sales planning
  finance/                 expenses, till → bank transfers, month-end till counts
  accounting/              fixed assets + depreciation run
  hr/                      staff + payroll days
  projects/                agile projects and sprints
  crm/                     pipelines, custom fields, opportunities
  loadtest/                fn-load-test (drives a running server over HTTP with the same payloads)
```

Each domain embeds its own data files (`*.json`, `*.psv`, `products.xlsx`).

### `base`

Global: the Peruvian legal payroll parameters. Per company: warehouses 1–2, one till per store and
the bank current account, the F001/B001 default invoice series, clients and providers, the product
catalog (categories, brands, products from `products.xlsx`) and the supplies catalog.

`fn-init` stays in the production binary: it is installation (schema deploy, companies, admin
users, cities, the first site/warehouse/till), and `base` runs after it. Cities stay in `fn-init`
because a real install needs them too.

### Module contract

```go
type Module struct {
    Name     string
    Requires []string                                     // "base" is implicit
    Setup    func(run *runner.Run) ([]runner.DayStep, error) // idempotent; returns its day steps
}
```

`runner.Run` replaces `erpHistoryGenerator`: clock, random source, company, user token, tills,
parties, invoice series, product pool and prices, stock buckets, resume state. Domains are plain
functions over `*runner.Run`; each module keeps its own section of the resume state.

`genix-seed run sales,hr --days=60` resolves `Requires`, runs every `Setup` (clock on the first
simulated morning), then one day loop whose steps run in the global order: purchases → expenses →
sales → annulments → till transfers → payroll → till counts → depreciation. One module alone is a
selection of one; `run all` replaces `fn-generate-demo-data`. No sale-frame rebuild is needed: the
frames are maintained on write (cron action 2 is gone).

## Remote run and abort

- `genix-seed` is built for linux/<arch> (same `[[servers]]` entry as `deploy_vps`), zstd'd,
  uploaded to `/usr/local/bin/genix/genix_seed`, and reads the server's `config.toml`.
- `./app.sh seed_remote [--server=X] run all --days=60` starts it under
  `systemd-run --unit=genix-seed` with the `genix` unit's user, working directory and `GENIX_*`
  environment, so an SSH drop does not kill it, and follows `journalctl -u genix-seed -f`. Ctrl+C
  stops the tail; a menu offers abort at the next checkpoint / follow again / detach.
  `seed_remote status|logs|abort` work later from any terminal.
- SIGTERM/SIGINT cancel the run's context. Every checkpoint (`SaveState`) persists the resume
  point and then returns if the run was cancelled, so the run stops between writes and says
  "rerun the same command to resume". A second signal exits at once; the state file is written by
  atomic rename.
- A lock per company refuses two seeds writing the same company. A pre-flight prints host,
  database, company and whether it already holds data, and asks for confirmation (`--yes` skips).
- The `genix` service keeps running. The seed process turns off the invoice emit sweep
  (`ScheduleEmitPendingSweep` writes no cron row), so the server never sends the seeded invoices to
  SUNAT and marks them failed.

## Production binary

Loses `fn-generate-*`, `fn-seed-*`, `fn-load-test`, `fn-wipe-company-data` and
`backend/exec/sample_records.go`. `wipe` moves into `genix-seed`.

## Steps

Each step compiles and is checked with a seed run before the next.

1. Move `tests/sample_records`, `exec/wipe_company_data.go` and the load test into `backend/seeds/`
   as they are; add `genix-seed` dispatching to them; remove them from `exec`, `scripts/main.go`
   and the loadtest shell scripts' binary.
2. Extract `runner` and split into domain packages; define `base`; module registry and the merged
   day loop; `run all` reproduces today's demo data.
3. Abort (context + signals), per-company lock, pre-flight, emit sweep off, sale frames rebuild.
4. `seed_remote` in `scripts/`.
5. One `backend/seeds/README.md` replaces `scripts/GENERATE_*.md`, `SEED_*.md`,
   `WIPE_COMPANY_DATA.md`.

All five steps are implemented (2026-11-11); see `backend/seeds/README.md` and `RATIONALE.md`.

Verification: `run all --days=5` on a wiped company completes, balances reconcile and the reports
show the history; an abort followed by a resume finishes the same plan. Exact totals no longer
match the old `fn-generate-demo-data`: supply data now shares the run's random source.
