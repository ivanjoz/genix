# seed_hr

The HR and payroll history of the demo company (company 1), lived one day at a time through the
production handlers of `app/hr` with the clock frozen on each day. Code:
`backend/tests/sample_records/seed_hr.go` (staff in `generate_hr_data.go`).

It runs in two ways:

- **Inside `generate_demo_data`**, as the `payroll` step of every simulated day. This is the
  path for a full refill: wipe the ERP data, then run `generate_demo_data`.
- **On its own**, over a database that already has the ERP data:

```bash
cd scripts && go run . seed_hr               # the last 7 months, up to today
cd scripts && go run . seed_hr --months=3
cd scripts && go run . seed_hr --reset       # drop an interrupted run's state and start over
```

## What a day does

| Day | Action | Route |
|---|---|---|
| First day | Legal parameters; the staff as it stands that day (skipped if the company has employees); company concepts; movilidad for one in three and a judicial withholding for one; a S/ 900 loan in 3 installments | `POST.hr-legal-parameters`, `POST.hr-employees`…, `POST.hr-concepts`, `POST.hr-employee-concepts`, `POST.hr-loans` |
| 1st | Two workers with a year of service book vacation for the 10th–19th | `POST.hr-employee-records` |
| 7th, 14th, 21st, 28th | The last two workers on the list are paid the week (S/ 250 each) | `POST.hr-pay-advances` |
| 20th | About one month in three, a worker resigns | `POST.hr-employees` |
| 25th | The month's novedades: overtime 25 % and 35 %, absences, productivity and one-off bonuses | `POST.hr-employee-records` |
| Month end | The month's single run (salary, plus the gratificación in July and December, the CTS in May and November, and the settlement of anyone who left), calculated, closed and paid | run inserted, then `POST.hr-payroll-calculate` → `-close` → `-paid` |
| Last day | The current month's run, calculated and left open (unless it is a month end) | |

Every payment comes out of `BCP Cta. Cte. Soles`. When the bank can't cover one, a capital
contribution (type 7, account 501) tops it up first.

## Reproducible and safe to repeat

- The staff is drawn from `sampleDataSeed`, and each day's randomness from the seed plus the day.
  The same window always produces the same history.
- Each action first checks what is stored: the run of that type and period (and resumes it if it
  stopped halfway), the period's novedades, the week's advance, an overlapping time off, a leaver
  this month, an existing loan. Running the same days again writes nothing twice.
- A run is resumable like the demo: the state is `tmp/genix_seed_hr_state.json`.

## Writing over existing data

On its own, the seed writes payments into days the bank account already has movements for. Each
movement stores the balance at write time (`FinalAmount`), so that trail ends up out of order. At the
end, `rebalanceCashBank` fixes it:

1. It walks the account's movements in ID order (account, date, write order within the day).
2. It starts from the opening balance (the current balance minus every movement).
3. It recomputes each running balance and rewrites only the ones that changed.
4. If the trail dips below zero, it books a capital contribution on the day before the window and
   recomputes once more.

## Not seeded

- Payment of the payroll contributions (EsSalud, ONP, AFP, 5ta, the judicial withholding). They
  stay owed in the accounting.
- Profit sharing and extraordinary runs.
- AFP rates are loaded for 2026 only (`seed_hr_legal_parameters`). A window that reaches 2025
  or 2027 fails at its first run there.

## The staff

About 23 employees in 4 departments, with invented Peruvian identities. The mix exercises the
payroll engine's branches:

| Case | How it is seeded |
|---|---|
| ONP vs AFP | One in four in the ONP; the rest spread over the four AFPs with a 12-character CUSPP |
| AFP commission | Mostly mixed, one in four on flow |
| EPS | Some workers earning S/ 3,000 or more |
| Part-time | One in eight shop-floor workers at 3 hours/day (no CTS, reduced vacations) |
| Fixed-term contracts | Recent hires, one-year contracts |
| Contract versions | Half of the indefinite workers with more than a year get a 5–12 % raise one year after hiring |
| Asignación familiar | About half of the workers over 25 have children: minors, adults, and adults still studying |
| Terminated before the window | One in twelve, by resignation or end of contract |

Salaries are drawn from each position's band and never go below the RMV of S/ 1,230.

## Requirements

`fn-homologate`, company 1 with user 1, and warehouses 1 and 2. The bank account is created if it
is missing.
