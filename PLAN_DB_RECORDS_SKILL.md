# PLAN — Generic data-access skill (list / query / insert / update)

Goal: let the agent **list tables, query records with filters (including `AllowFilter`), insert and
update** without writing new Go code every time, with **everything going through the ORM** — no raw
CQL.

Confirmed decisions: records travel as the record struct's JSON; writes are gated behind
`apply: true`; the CLI bridge is an `exec` handler (`fn-db`), not a separate binary.

---

## 1. What already exists

| Piece | Location | What it gives |
|---|---|---|
| Table name registry | `genix-orm/db/registry.go` — `RegisterTableFactory` / `ResolveTableByName` | name → `db.Table` (compiled metadata, **no** query capability) |
| Generated registry | `backend/exec/controllers.generated.go` | 44 `RegisterTableFactory(...)` + `MakeScyllaControllers() []db.Controller` |
| `db.Controller` | `genix-orm/db/registry.go:86` | non-generic admin surface per table: `GetRecords(partValue, limit, lastKey)`, CSV, gob, recalc, counters… |
| Implementation | `genix-orm/scylla/deploy.go` — `ScyllaController[T,E]` | the **only** implementor of `db.Controller` (the `dynamo` one is a separate interface) |
| Dynamic predicates | `TableQueryInterface[T]` (`db/interfaces.go`) + `SetWhere(col, op, value)` / `SetWhereIn` | already filter **by column name at runtime** — this is what `GetRecords` uses |
| CLI bridge | `backend/main.go:281-380` | `cd backend && go run . fnXXX <rest of args>` runs a handler from `exec.ExecHandlers`, with the Scylla connection already open |
| Query capabilities | `ScyllaTable.capabilities` (`select_compute.go`) | signatures like `company_id\|=\|status\|=` → **exactly which query shapes avoid `ALLOW FILTERING`** |

## 2. The gap

1. `Controller` can only fetch a **whole partition**: no filters, no column projection, no
   `AllowFilter`, no useful limit.
2. There is **no** generic write path: `Insert`/`Update` are Go generics and need `T` known at
   compile time.
3. There is no way to resolve `table name → Controller` (only → `db.Table`).
4. There is no serializable schema description (columns, types, keys, supported query shapes) the
   agent can read before querying.
5. Values arrive from JSON as `float64`, and `db.ToInt64` returns **0** for `float64` — so a filter
   `{"col":"id","value":123}` would silently become `id = 0`. Explicit coercion to the column's real
   type is required.

## 3. Design decisions

- **Everything through the ORM.** No hand-written CQL. Writes go through `scylla.Insert` /
  `scylla.Update`, so these come for free: key autoincrement, `created` / `updated`,
  `updated_version`, virtual columns, views, packed indexes and the text-search index.
- **The generic surface hangs off `db.Controller`** (the interface you pointed at). It has a single
  real implementor, so extending it breaks nothing.
- **Filters by column name; records in the record's JSON shape.** Filters speak the engine's
  language (`company_id`, `nombre`); records use the struct's JSON (`CompanyID`, `Name`) — the same
  contract already sent to the frontend, so a record that was read can be edited and sent straight
  back. To remove the ambiguity, the column resolver accepts **both** forms (column name or Go field
  name) and `describe` publishes the mapping.
- **One CLI bridge**, `go run . fn-db`, instead of a new binary: it reuses `main.go`'s startup
  (config.toml, Scylla connection, GenixSearch, logging) with zero duplication.
- **Result to a file**, summary to stdout. `core.Print` pretty-prints and startup emits plenty of
  noise; a JSON file is parseable and does not flood the terminal with 10,000 rows.

---

## 4. ORM changes (`backend/genix-orm`, submodule → its own commit)

### 4.1 `db/generic_access.go` (new, ~140 lines)

Serializable types plus name resolution and coercion. Nothing here knows about Scylla.

```go
// FilterSpec is one predicate expressed with strings and JSON values, for callers that resolve a
// table by name and cannot hold its Go types.
type FilterSpec struct {
	Column   string // column name ("company_id") or Go field name ("CompanyID")
	Operator string // = != > >= < <= IN BETWEEN CONTAINS
	Value    any
	Values   []any // IN
	From, To any   // BETWEEN
}

type QuerySpec struct {
	Filters     []FilterSpec
	Columns     []string // projection; empty = all
	Limit       int32
	AllowFilter bool
	OrderDesc   bool
}

// TableDescription is one table's schema in serializable form: what to read before writing a filter.
type TableDescription struct {
	Name, Namespace    string
	ID                 int16
	Partition          string
	Keys               []string
	Columns            []ColumnDescription
	QueryShapes        []string // QueryCapability signatures: shapes that need no ALLOW FILTERING
	SaveUpdatedVersion bool
}

type ColumnDescription struct {
	Name, FieldName, JSONKey, GoType string
	IsPartition, IsKey, IsVirtual, IsManaged bool
}

// ResolveColumn looks a column up by column name and, failing that, by Go field name.
func ResolveColumn(table Table, nameOrField string) (IColInfo, error)

// ColnByName returns the column as the Coln handle Insert/Update require.
func ColnByName(table Table, nameOrField string) (Coln, error)

// CoerceToColumn converts a value straight out of JSON (float64, string, bool, []any) into the
// column's real Go type. Without this a numeric id arrives as float64 and ToInt64 flattens it to 0.
func CoerceToColumn(column IColInfo, value any) (any, error)
```

`ColnByName` needs a tiny adapter, because `Coln` asks for `GetInfo() ColumnInfo` while `IColInfo`
exposes `GetInfo() *ColInfo`:

```go
type namedCol struct{ info ColumnInfo }
func (c namedCol) GetInfo() ColumnInfo { return c.info }
func (c namedCol) GetName() string     { return c.info.Name }
```

`GetName()` is enough: `resolveUpdateColumnsForWrite` (insert-update.go:1150) and
`collectAffectedColumnsForInclude` only use the name to look the compiled column back up.

### 4.2 `db/registry.go` — extend `Controller`

```go
type Controller interface {
	// … the current 12 methods, untouched …

	// DescribeTable publishes the schema in serializable form.
	DescribeTable() TableDescription
	// QueryRecordsJSON runs a read described only with strings and returns the records as a JSON
	// array plus their count.
	QueryRecordsJSON(spec QuerySpec) (payload []byte, count int, err error)
	// InsertRecordsJSON inserts a JSON array of records. Returns how many were written.
	InsertRecordsJSON(payload []byte, excludeColumns []string) (int, error)
	// UpdateRecordsJSON updates only includeColumns on each record of the JSON array.
	UpdateRecordsJSON(payload []byte, includeColumns []string) (int, error)
}
```

`genix-orm/dynamo` declares its **own** `Controller` (`dynamo/controller.go:31`) and does not
implement this one, so the change only obliges `ScyllaController`.

### 4.3 `db/tablestruct.go` + `db/interfaces.go` — the missing runtime predicates

```go
// SetBetween adds a BETWEEN from runtime values, mirroring Col.Between for callers that only know
// the column by name.
func (e *TableStruct[D, T, E]) SetBetween(colname string, from, to any)
```

Plus a new interface (`TableQueryInterface` stays untouched so `deploy.go` keeps its footing):

```go
// TableGenericQuery is the surface a read built at runtime needs.
type TableGenericQuery[T any] interface {
	TableQueryInterface[T]
	SetWhereIn(string, []any)
	SetBetween(string, any, any)
	Select(...Coln) *T
	OrderDesc() *T
}
```

### 4.4 `scylla/controller_generic.go` (new, ~200 lines)

The four methods on `ScyllaController[T,E]`:

- **`DescribeTable`** — built from `e.Table` (`TableCore` + `capabilities`, same package) and
  `e.Schema`. Flags `CreatedCol` / `UpdatedCol` / `UpdatedVersionCol` as *managed*.
- **`QueryRecordsJSON`** — `records := []T{}`; `any(Query(&records)).(TableGenericQuery[E])`; per
  filter: `ResolveColumn` → `CoerceToColumn` → `SetWhere` / `SetWhereIn` / `SetBetween`; then
  `Select` (projection), `Limit`, `AllowFilter`, `OrderDesc`, `Exec`, `json.Marshal(records)`.
- **`InsertRecordsJSON`** — `json.Unmarshal(payload, &records)` into `[]T` → `Insert(&records, cols…)`.
- **`UpdateRecordsJSON`** — same → `Update(&records, cols…)`.

Unmarshalling into `[]T` is what makes complex fields (`[]ProductProperties`, CBOR blobs) work
without a per-type decoder; `encoding/json` ignores `TableStruct`'s unexported fields.

### 4.5 `db/registry.go` — resolve a Controller by name

```go
func RegisterControllerFactory(tableName string, makeController func() Controller)
func ResolveControllerByName(tableName string) (Controller, error)
```

Populated from `backend/exec` by walking `MakeScyllaControllers()` and keying on `GetTableName()` —
**no change to the code generator**, so `controllers.generated.go` needs no regeneration.

---

## 5. CLI bridge: `backend/exec/db_console.go` (new, ~150 lines)

Registered in `exec/main.go` as `"fn-db": DbConsole`.

```bash
cd backend
go run . fn-db '{"op":"tables"}'          # inline JSON (starts with "{")
go run . fn-db tmp/db-request.json        # or a file path, for large payloads
```

`main.go` already detects any argument starting with `fn`, joins the rest into `args.Message` and
runs it with the Scylla connection open.

### Request

```jsonc
{
  "op": "tables" | "describe" | "query" | "count" | "insert" | "update",
  "table": "products",

  // query / count
  "filters": [
    {"col": "company_id", "op": "=",       "value": 1},
    {"col": "status",     "op": "IN",      "values": [1, 2]},
    {"col": "updated",    "op": "BETWEEN", "from": 1700000000, "to": 1800000000}
  ],
  "columns": ["id", "nombre"],   // projection
  "limit": 50,                   // default 50
  "allow_filter": false,         // only when describe.query_shapes does not cover the shape
  "order_desc": false,

  // insert / update
  "records": [ { "CompanyID": 1, "Name": "X" } ],
  "update_columns": ["Name", "Status"],   // required on update
  "exclude_columns": [],                  // optional on insert
  "apply": false,                         // without true the write is validated but NOT executed

  "out": "tmp/db-result.json"    // default
}
```

### Response

File at `out`:
```json
{ "op":"query", "table":"products", "count":12, "records":[ … ] }
```
stdout: one summary line plus the first 3 records, so thousands of rows are never dumped.

Errors: a descriptive message in `FuncResponse.Error` and `os.Exit(1)` (already handled by `main.go`).

---

## 6. Skill: `.claude/skills/database-records/`

`SKILL.md` — name `database-records`, description triggered by: *"list the records in table X"*,
*"query the DB"*, *"insert this record"*, *"update field Y on record Z"*, *"how many records are in
…"*.

Contents:
1. **Mandatory protocol**: `tables` → `describe <table>` → only then `query`. Never guess column
   names.
2. **How to pick a filter**: compare against `query_shapes`; if the shape is missing, either add the
   partition or turn on `allow_filter` (warning that it is a full scan).
3. Operator table and the column ↔ Go field ↔ JSON key mapping.
4. **Write rules**: read first, always filter by the partition, explicit `update_columns`,
   `apply:true` only with the human's go-ahead.
5. **Project conventions** (from AGENTS.md): dates as `UnixDay int16`, datetimes as `SUnixTime
   int32`; `created`/`updated`/`updated_version` are set by the ORM and must not be sent.
6. Copy-paste recipes (list, filtered query, count, insert, single-field update).

`reference.md` — the full request/response schema, the operator table, and common errors
(`use ALLOW FILTERING`, unknown column, type coercion) with their fixes.

Also: a new section **§14 Generic access by table name** in
`backend/docs/ORM_DATABASE_QUERY.md`.

---

## 7. Safety

`config.toml` points at a **real remote database** (`149.104.66.239:14008`, confirmed reachable).
Therefore:

- `insert` / `update` do nothing without `"apply": true`; without it they are validated and the tool
  reports what would be written.
- No `delete`: the Scylla driver exposes no delete operation at all, and none will be added here.
- `limit` defaults to 50, with a hard cap of 5,000 per call.
- The skill requires filtering by the partition column (`company_id`) unless explicitly told
  otherwise.
- The result file lands in `backend/tmp/` (already git-ignored) and may contain real data.

## 8. Validation

```bash
cd backend && go build ./...                                    # compiles with the patched submodule
go run . fn-db '{"op":"tables"}'                                # 44 tables
go run . fn-db '{"op":"describe","table":"products"}'
go run . fn-db '{"op":"query","table":"products","filters":[{"col":"company_id","op":"=","value":1}],"limit":5}'
go run . fn-db '{"op":"count","table":"products","filters":[{"col":"company_id","op":"=","value":1}]}'
# writes against the demo table, never against business data:
go run . fn-db '{"op":"insert","table":"zz_demo_struct","records":[{…}],"apply":true}'
go run . fn-db '{"op":"update","table":"zz_demo_struct","records":[{…}],"update_columns":["…"],"apply":true}'
cd ../scripts && go run . check_tables                          # schema still valid
```

## 9. Out of scope

- Deleting records.
- Generic `GROUP BY`, `Delta()`, text search and `QueryCachedIDs` (they can follow the same pattern
  later; the goal now is read/write).
- Touching the `scripts/controllers/controllers_generator.go` generator.
- The `cloud` ORM (DynamoDB / D1).

## 10. Files

| File | Action | Approx. |
|---|---|---|
| `genix-orm/db/generic_access.go` | new | 140 |
| `genix-orm/db/registry.go` | +4 methods on `Controller`, + controller registry | 25 |
| `genix-orm/db/tablestruct.go` | +`SetBetween` | 8 |
| `genix-orm/db/interfaces.go` | +`TableGenericQuery` | 10 |
| `genix-orm/scylla/controller_generic.go` | new | 200 |
| `backend/exec/db_console.go` | new | 150 |
| `backend/exec/main.go` | +1 entry in `ExecHandlers` | 1 |
| `backend/docs/ORM_DATABASE_QUERY.md` | +§14 | 40 |
| `.claude/skills/database-records/SKILL.md` + `reference.md` | new | 180 |

Two commits: one in the `genix-orm` submodule, one in `genix`.
