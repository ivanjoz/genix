# `fn-db` reference

## Invocation

```bash
cd backend
go run . fn-db '{"op":"tables"}'        # inline JSON: must start with '{', single-quote it
go run . fn-db tmp/db-request.json      # anything else is read as a file path (relative to backend/)
```

Numbers are decoded as `json.Number`, so 64-bit ids keep every digit.

Implementation: `backend/exec/db_console.go` → `db.Controller` (`genix-orm/db/registry.go`) →
`genix-orm/scylla/controller_generic.go`. Name resolution and value coercion live in
`genix-orm/db/generic_access.go`.

## Request

| Field | Type | Applies to | Meaning |
|---|---|---|---|
| `op` | string | all | `tables` \| `describe` \| `query` \| `count` \| `insert` \| `update` |
| `table` | string | all but `tables` | table name as listed by `tables` |
| `filters` | array | `query`, `count` | see below |
| `columns` | []string | `query` | projection, by column name |
| `limit` | int | `query`, `count` | default 50, hard cap 5000 |
| `allow_filter` | bool | `query`, `count` | permit a shape outside `query_shapes` (full scan) |
| `order_desc` | bool | `query` | reverse clustering-key order |
| `records` | array | `insert`, `update` | records in the record struct's JSON shape |
| `update_columns` | []string | `update` | required; columns to write, by column name |
| `exclude_columns` | []string | `insert` | columns to leave out of the insert |
| `apply` | bool | `insert`, `update` | without `true` the write is validated only |
| `out` | string | all | result path, default `tmp/db-result.json` |

### Filters

```jsonc
{"col": "company_id", "op": "=",        "value": 1}
{"col": "status",     "op": "IN",       "values": [0, 1]}
{"col": "updated",    "op": "BETWEEN",  "from": 1700000000, "to": 1800000000}
{"col": "category_ids", "op": "CONTAINS", "value": 12}      // one element of a slice column
```

| Operator | Notes |
|---|---|
| `=` `!=` `>` `>=` `<` `<=` | scalar comparison, value in `value` |
| `IN` | needs a non-empty `values` array |
| `BETWEEN` | needs `from` and `to`, both inclusive |
| `CONTAINS` | slice-backed columns only; the value is one **element** |

`col` accepts the storage column name (`company_id`) or the Go field name (`CompanyID`). Values are
coerced to the column's real Go type; a fractional number on an integer column is rejected rather
than truncated, because a rounded key would silently query the wrong row.

## Response (`tmp/db-result.json`)

```jsonc
{
  "op": "query",
  "table": "products",
  "count": 20,
  "limit_reached": true,     // count is a floor, not a total
  "applied": true,           // writes only: the write actually ran
  "tables":  [ … ],          // op=tables
  "schema":  { … },          // op=describe
  "records": [ … ]           // op=query
}
```

### `schema` (op=describe)

```jsonc
{
  "name": "products", "id": 22, "partition": "company_id", "keys": ["id"],
  "save_updated_version": true,
  "columns": [
    {"name":"company_id","field":"CompanyID","json":"CompanyID","type":"int32","is_partition":true},
    {"name":"id","field":"ID","json":"ID","type":"int32","is_key":true},
    {"name":"nombre","field":"Name","json":"Name","type":"string"},
    {"name":"updated","field":"Updated","json":"upd","type":"int64","is_managed":true}
  ],
  "query_shapes": ["company_id|=", "company_id|=|id|=", "company_id|=|status|=", …]
}
```

- `name` → filters, `columns`, `update_columns`, `exclude_columns`.
- `json` → keys inside `records`. Empty means the field is not serialized (`json:"-"`).
- `is_managed` → written by the ORM (`created`, `updated`, `updated_version`); leave it out of writes.
- `is_virtual` → a column the ORM computes for an index/view; never write it directly.
- `query_shapes` → `column|op` pairs. `=` equality, `~` range, `@` CONTAINS. Sorted and deduplicated.

## Reading query_shapes

A shape is usable when your filters match a prefix of it in order. For `products`:

| Filters | Plan |
|---|---|
| `company_id=1` | matches `company_id\|=` |
| `company_id=1, status=1` | matches `company_id\|=\|status\|=` |
| `company_id=1, updated>N` | matches `company_id\|=\|updated\|~` |
| `status=1` alone | no shape (partition missing) → needs `allow_filter` |
| `company_id=1, final_price BETWEEN` | no shape → needs `allow_filter` |

## Errors

| Message | Fix |
|---|---|
| `table "X" has no column "Y". Available: …` | use one of the listed names |
| `la tabla "X" no tiene un controller registrado` | the name is not in `op:tables`; check spelling |
| `use ALLOW FILTERING` (from Scylla) | reshape the filters to a `query_shape`, or set `allow_filter` |
| `Error del ORM:: A composit index/view requires the columns "a","b" be updated together` | add every named column to `update_columns` |
| `column "X" expects a integer, got 1.5 (float64)` | the value does not fit the column type |
| `update de X sin columnas` | `update_columns` is required |
| `falta "records" para el insert` | `records` must be a non-empty array |
| `unknown operator "≠"` | use one from the operator table |

## Limits

- No delete: the Scylla driver exposes no delete operation.
- No `GROUP BY`, `Delta()`, text search or `QueryCachedIDs` — write Go against the typed API for
  those (`backend/docs/ORM_DATABASE_QUERY.md`).
- One partition per query: every efficient shape starts at the partition column.
- The result file may contain real customer data. It lives in `backend/tmp/`, which is git-ignored.
