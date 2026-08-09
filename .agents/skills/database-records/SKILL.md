---
name: database-records
description: Read and write real database records through the ORM from the command line — list tables, inspect a table's schema, query/count rows with filters, insert and update records. Use whenever the task needs actual data.
version: 0.1.0
---

# Database records (`fn-db`)

One command reaches every table through the ORM — no new Go file, no raw CQL:

```bash
cd backend
go run . fn-db '<json request>'      # inline JSON
go run . fn-db tmp/db-request.json   # or a file, for payloads with many records
```

Reads land in **`backend/tmp/db-result.json`** (Read it after the call); stdout only carries a
summary line plus the first 3 rows. Startup logs a lot before it — ignore everything above the
`== db …` line.

## Non-negotiable order of operations

1. `tables` — find the exact table name.
2. `describe` — get column names, the partition, and `query_shapes`.
3. Only then `query` / `count` / `insert` / `update`.

Never guess a column name: they are Spanish in older tables (`nombre`, `empresa_id`) and English in
newer ones (`company_id`), and the two conventions coexist inside a single table.

## Operations

### `tables` — what exists

```bash
go run . fn-db '{"op":"tables"}'
```
42 entries with `name`, `id`, `partition` and `keys`.

### `describe` — before every first query on a table

```bash
go run . fn-db '{"op":"describe","table":"products"}'
```

Two fields matter:

- **`columns[]`** — each one in three vocabularies: `name` (what filters use), `field` (Go),
  `json` (what insert/update payloads use). `is_managed: true` means the ORM writes it — never send
  it.
- **`query_shapes[]`** — the predicate combinations the table can serve through a key, index or
  view. Format: `column|op` pairs, where `=` is equality, `~` a range, `@` a CONTAINS. So
  `company_id|=|status|=` means "filter by company_id and status, both by equality".

**If your filter set is not in `query_shapes`, the query fails** unless you pass
`"allow_filter": true` — which is a full partition scan. Prefer reshaping the filter; if you do use
it, say so in your answer to the user.

### `query` — read rows

```bash
go run . fn-db '{"op":"query","table":"products",
  "filters":[{"col":"company_id","op":"=","value":1},{"col":"status","op":"=","value":1}],
  "columns":["id","nombre","final_price"],
  "limit":20}'
```

- `filters[].op`: `=` `!=` `>` `>=` `<` `<=` `IN` (`"values":[…]`) `BETWEEN` (`"from"`/`"to"`)
  `CONTAINS` (one element of a slice column).
- `columns` is a projection — always use it on wide tables (`products` has 39 columns, several of
  them large blobs).
- `limit` defaults to **50**, hard cap **5000**. `"limit_reached": true` in the result means there
  are more rows; it is a floor, not a total.
- `"order_desc": true` reverses the clustering-key order.

### `count` — how many

```bash
go run . fn-db '{"op":"count","table":"products","filters":[{"col":"company_id","op":"=","value":1}],"limit":5000}'
```
Reads only the key columns. Still bound by `limit`, so check `limit_reached` before reporting a
number as final.

### `insert` / `update` — writes

Records use the **JSON keys** from `describe` (`CompanyID`, `Name`, `ss`), not column names.
Filters keep using column names. Both accept the Go field name as a fallback spelling.

```bash
# 1. dry run first — validates the payload and column names, writes nothing
go run . fn-db '{"op":"insert","table":"zz_demo_struct",
  "records":[{"CompanyID":1,"Name":"prueba","ss":1}]}'

# 2. only after showing the user what will be written
go run . fn-db '{"op":"insert","table":"zz_demo_struct",
  "records":[{"CompanyID":1,"Name":"prueba","ss":1}],"apply":true}'
```

```bash
go run . fn-db '{"op":"update","table":"zz_demo_struct",
  "records":[{"CompanyID":1,"ID":90001,"Name":"nuevo nombre","ListID":0,"ss":1}],
  "update_columns":["nombre","lista_id","status"],"apply":true}'
```

Rules:
- **`apply: true` is required.** Without it the write is validated and reported, never executed.
  `config.toml` normally points at a shared, real database — get the user's go-ahead first.
- `update` needs `update_columns` (column names): the explicit list is what stops a partial payload
  from blanking fields you never read.
- Every record must carry its **partition and key** columns (e.g. `CompanyID` + `ID`), or the write
  lands on the wrong row.
- **Read the row first**, edit it, send it back. That is what the shared JSON shape is for.
- Never send `created`, `updated` or `updated_version` — the ORM assigns them, along with
  autoincrement keys, virtual columns, view fan-out and the text-search index.
- There is **no delete**: the ORM exposes none, and this console adds none.

## Project conventions that apply to written values

- Dates: `UnixDay int16` — days since the unix epoch.
- Datetimes: `SUnixTime int32` — `int32((time.Now().Unix() - 1e9) / 2)`.
- Validate before writing. A record the client sent is never trusted as-is.

## When it fails

| Message | Meaning |
|---|---|
| `table "X" has no column "Y". Available: …` | wrong spelling — the list of valid names is in the error |
| `use ALLOW FILTERING` | the filter set is not in `query_shapes`: reshape it or set `allow_filter` |
| `Error del ORM:: A composit index/view requires the columns "a","b" be updated together` | a view keys on those columns — add them all to `update_columns` |
| `column "X" expects a integer, got …` | wrong JSON type for the column |

Full request/response schema, the operator table and more troubleshooting: `reference.md`.
Model/query documentation for writing Go against the ORM: `backend/docs/ORM_DATABASE_QUERY.md`.
