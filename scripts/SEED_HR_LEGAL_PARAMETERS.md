# seed_hr_legal_parameters

`fn-seed-hr-legal-parameters` loads the Peruvian legal values the payroll engine reads
(`hr_legal_parameters`, `hr_afp_rates`). Both tables are partitioned by country (604), so every
company shares them. In production the SaaS admin keeps them up to date on
`system/hr-legal-parameters`; this script only loads the starting values for a fresh database.

```bash
cd scripts && go run . seed_hr_legal_parameters
```

The tables must exist first (`fn-homologate`). Values are upserted by key and date (legal values)
or by AFP and period (rates), so running it twice does not duplicate them.

## What it loads

| Value | Seeded | Verify against |
|---|---|---|
| RMV | S/ 1,130 from 2025-01-01, S/ 1,230 from 2026-10-01 | El Peruano |
| UIT | S/ 5,350 (2025), S/ 5,500 (2026) | MEF |
| EsSalud, ONP, family allowance, grati bonus (9 % / 6.75 % with EPS) | Rates in basis points | — |
| RMA (AFP premium cap) | S/ 12,672.65 from 2026-01-01, S/ 12,732.70 from 2026-10-01 | **SBS: the first value stands in for Q1–Q3 2026 and must be checked** |
| AFP rates 2601–2612 | Fund 10 %, premium 1.37 %, flow / mixed commission per AFP | SBS monthly commissions |

The AFP commissions are the 2026 values gathered for the plan; check them against the SBS table
before using the data for anything other than demos.
