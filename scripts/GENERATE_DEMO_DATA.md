# generate_demo_data and generate_expenses

`fn-generate-demo-data` seeds the whole ERP as if a two-store Lima minimarket had run on it for the
last 60 days. `fn-generate-expenses` is its expense half, runnable on its own.

```bash
cd scripts && go run . generate_demo_data            # 60 days, 200–400 sales/day
cd scripts && go run . generate_demo_data --days=5   # smoke test
cd scripts && go run . generate_expenses --days=60   # expenses only
cd backend && go run . fn-generate-demo-data --dry-run
```

Start from a freshly reseeded database. Nothing here deletes data, and the purchases, sales and
expenses are cumulative.

**Reproducible.** Every generator draws from one fixed seed (`sampleDataSeed`; `--seed=N` picks
another). The same arguments on the same real date produce the same data. Two things break the
match: a different date, since the window ends today, and a resumed run, which restarts the random
sequence at the step where it resumed. The HR step seeds each day by the day itself, so it repeats
exactly either way.

## Why one day loop

Every cash movement stores its register's balance **at write time** (`FinalAmount`). If the ERP
history ran first and the expenses ran afterwards over the same days, the per-movement balance
trail would be out of date order. The demo walks the days once. Each day runs these steps in
timestamp order, with the process clock frozen on each write (`core.SetHistoricalUnix`):

| Time | Step | Route |
|---|---|---|
| 08:00 | Purchase order per store → confirm → **supplier factura** (base + 18% IGV) → payment from the bank → reception (lots/serials) | `POST/PUT.purchase-orders` (actions 1, 5, 3), `POST.purchase-order-entry` |
| 10:15 | Expense payments that fall due today | `POST.expense-payment` |
| 10:30 | The day's expenses, paid now or queued for their due date | `POST.expenses`, `POST.expense-inventory`, `GET.expense-schedule-periods` |
| 11–20h | 200–400 sales, issued as **boletas** or **facturas** into the store's till | `POST.sale-order` with `IssueSeriesID` |
| 20:30 | 2–5 annulments, refunded from the till | `POST.sale-order-annul` |
| 21:00 | Each till's takings (minus a S/ 500 float) deposited in the bank | `POST.cash-banks-movement` (type 3) |
| 21:15 | The HR day, paid from the bank (see `SEED_HR.md`) | `POST.hr-*` |
| 21:30, month end | Till counts (usually exact, sometimes short → 1419, or over → 759) and the depreciation run | `POST.cash-banks-reconciliation`, `POST.asset-depreciation-run` |

A capital contribution (type 7, account 501) tops up the bank when it can't cover a payment.

## One-time setup

Done once per run, recorded in the state file:

1. `fn-generate-supply-data` with the clock on the first simulated morning: products from
   `products.xlsx`, replenishment settings, supplies, and assets with their payments.
2. `fn-generate-projects-flow`, on its own timeline.

Done on every invocation, idempotently:

- **Cash accounts**: the first store keeps the company's first till. The other stores get
  `Caja Tienda N`, and there is a bank account `BCP Cta. Cte. Soles` (type 2). Each is matched by
  name, so a rerun reuses it.
- **Invoice series**: the active default factura and boleta series, or `F001`/`B001` if missing.
  The `invoice_document` is written with the sale; nothing is sent to SUNAT (that is
  `fn-emit-cpe`).
- **Sales planning**: three seasonality curves (steady, Christmas peak, summer) and a 52-week plan
  per pooled product. Skipped if the company already has planning.
- **Shipping costs** for 20 Lima districts. The handler merges by district.

## Sales documents

- One sale in eight goes to a company client (11-digit RUC) and is issued as a **factura**.
- The rest are **boletas**, half of them to a known client.
- A boleta of S/ 700 or more is given a client with a DNI, as SUNAT requires.

The 15 company clients are at the end of `erp_history_clients.json`.

## Expense catalog

`expenseTemplates` in `backend/tests/sample_records/generate_expenses.go`. Suppliers are in
`expense_suppliers.json`.

| Expense | PCGE | Document | When |
|---|---|---|---|
| Rent, store 1 / store 2 | 6352 | Factura | schedule, day 1 |
| Electricity | 6361 | Utility bill (14) | schedule, day 8, paid at +12 days |
| Water | 6363 | Utility bill (14) | schedule, day 12, paid at +14 days |
| Internet / mobile phone | 6365 / 6364 | Factura | schedule, day 5 / 18 |
| External accountant | 6323 | Fee receipt (2), no IGV | schedule, day 28 |
| Security / cleaning | 638 | Factura | schedule, day 28 |
| POS software license | 639 | Factura | schedule, day 3 |
| Advertising | 6371 | Factura | ~7% of days |
| Freight | 6311 | Boleta | Saturdays |
| Refrigeration maintenance | 6343 | Factura | ~4% of days |
| Supplies into stock | the supply's account (6032…) | Factura | ~30% of days, `POST.expense-inventory` |
| Bank fees / ITF | 639 / 6412 | none | last day of the month |
| Municipal fees | 643 | none | quarterly, day 28 |
| IGV for the previous month | 40111 | none | day 15 |
| Income-tax prepayment | 40171 | none | day 15 |

- **Monthly bills** live as expense schedules. On the bill's day, the generator materializes the
  period and attaches the supplier's comprobante and that month's amount.
- **Taxes** are computed from what the run generated in the previous month:
  - IGV = 18/118 of gross sales minus the IGV of purchase and expense documents;
  - income-tax prepayment = 1.5% of net sales.
  The first simulated month has no previous month in the run, so it pays neither.
- **Payment outcome**: 85% paid in full, 10% half paid, 5% left pending. Charges with no document
  are always debited on the spot.
- **Payroll** is the `payroll` day step (`seed_hr.go`): the staff on the first day, then
  advances, vacations, novedades and every run, as described in `SEED_HR.md`. Payroll
  contributions (EsSalud, AFP, ONP, 5ta) are left owed: nothing pays them yet.

## Arguments

The ERP-history flags (see `GENERATE_ERP_HISTORY.md`), with these demo defaults:

| Flag | Demo default |
|---|---|
| `--days` | 60 |
| `--sales-min` / `--sales-max` | 200 / 400 |
| `--po-products` | 150 per store, to restock what 200–400 tickets drain |
| `--annul-min` / `--annul-max` | 2 / 5 |
| `--seed` | `sampleDataSeed` (20260101) |
| `--reset` / `--dry-run` | as in the ERP history |

## Resuming

The state is in `tmp/genix_demo_data_state.json` (`tmp/genix_expenses_state.json` for
`fn-generate-expenses`). Every day is a list of steps, and the state records the finished steps
plus each step's own progress (orders, sales, planned expenses, pending payments and annulments).
Rerunning the same command continues where it stopped.

The state version is 2. A state file from an earlier version of these scripts is refused; use
`--reset`.

## Requirements

- Company 1, user 1, warehouses 1 and 2, and at least one active PEN cash register.
- The database schema must be current (`fn-homologate`). Without it, delta views such as
  `shared_list_records__dlt_list_id_status_view` are missing and the catalog import fails.
- `git lfs pull`, for `products.xlsx`.
- Sales summaries are rebuilt by cron action 2, which does not run in an `fn-…` process. The
  result lists `unixDaysToReprocess`.
