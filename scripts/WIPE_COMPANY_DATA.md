# wipe_company_data

`fn-wipe-company-data` empties the ERP data of one company, so the sample generators can fill it
again. Code: `backend/exec/wipe_company_data.go`.

```bash
cd scripts && go run . wipe_company_data                       # dry run: lists what it would delete
cd scripts && go run . wipe_company_data --apply               # deletes company 1's ERP data
cd scripts && go run . wipe_company_data --only=hr_ --apply    # only the tables starting with hr_
cd scripts && go run . generate_demo_data                      # refill (ERP + HR)
```

**Dry run by default.** Nothing is deleted without `--apply`. It points at the database in
`config.toml`.

## What it keeps

- The company, its users and profiles, `parameters` and `system_parameters`, `company_secrets`,
  SaaS metering (`credit_*`, `usage_log`) and the webpage (`webpages`, `ecommerce_page_content`,
  `gallery_images`). The list is `wipeKeptTables`.
- Every table not partitioned by company: cities, HR legal parameters and AFP rates, image assets,
  logs, agent messages.

## What it deletes, per table

1. The company's partition of the base table. Scylla removes its materialized views itself.
2. The ORM's side data:
   - the `__delta_moves` rows and `__index_updated` cache-group marks;
   - the by-IDs slots (`cache_by_ids_slots`);
   - the text index;
   - the S3 snapshots (`delta-snapshot/` and `cache-group/`).
   The DataFrames are rebuilt, now empty.
3. The autoincrement counters (`x<company>_<table>_*`) are set back to 0 through fareward, so IDs
   start again from 1. A full wipe also resets the invoice correlativos (`cpe_<company>_*`).

After a full wipe (no `--only`), it seeds what `fn-init` gives a company: site 1, warehouse 1 and
`Caja Principal` (cash register 1). It then republishes the company config blob.
`generate_demo_data` expects exactly that.

## Afterwards

Browsers keep their delta caches, which only learn about changes, not deletions. Clear the site
data (or log out and in) after a wipe.
