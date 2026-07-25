# PLAN — Generic cached-by-IDs query (`db.QueryCachedGenericByIDs`)

## 1. Goal

Today every table that needs "resolve these IDs to a display label" pays for a dedicated
backend handler + a full record payload. `backend/business/products.go:71` (`GetProductsByIDs`)
is the template: it calls `db.QueryCachedIDs(&products, cachedIDs)`, which selects **every
non-virtual column** of `products`.

We want a table-agnostic path that returns only the few fields a label/reference needs:

```go
db.QueryCachedGenericByIDs(tableName string, cachedIDs []db.IDCacheVersion) ([]db.GenericRecord, error)
```

The per-table mapping of *which* columns fill that shape is declared once in `GetSchema()`.

Constraint (from the request): only tables whose single key column is an integer are supported —
which is already what `SaveCacheVersion` enforces (`backend/db/cache_version.go:72-92`: exactly one
key, `int16|int32|int64`, integer partition).

## 2. What exists today (verified)

| Piece | Location | Note |
|---|---|---|
| `IDCacheVersion{ID int64, PartitionID int32, CacheVersion uint8}` | `db/cache_version.go:332` | wire input, unchanged |
| `QueryCachedIDs[T,E]` | `db/cache_version.go:400` | phase 1 = compare `cache_version`; phase 2 = batched `IN` select |
| `TableSchema` | `db/main.go:201` | where `GenericRecord` gets added |
| `ScyllaTable` (private struct) | `db/main.go:39` | where resolved `genericRecord` metadata gets cached |
| `makeTable` wiring | `db/reflect.go:185`, `:197` (`saveCacheVersion`) | pattern to copy |
| Raw iterator scan pattern | `db/view_tables.go:212-239` | `iter.RowData()` + `scanner.Scan(rowValues...)` + `normalizeScannedValue` |
| Table list codegen | `scripts/controllers/controllers_generator.go` → `backend/exec/controllers.generated.go` | AST scan for `db.TableStruct[XTable, X]` |
| Route table | `backend/business/main.go` etc. | `"GET.p-products-ids": GetProductsByIDs` |

**Blocking design fact:** `QueryCachedIDs` is generic over `[T, E]` and resolves metadata at
compile time via `MakeScyllaTable[T, E]()`. A `tableName string` parameter therefore needs a
**runtime name → ScyllaTable registry**, which does not exist in `db` today. `scyllaTableCache`
(`db/table_cache.go:74`) is keyed by `PkgPath.TypeName`, not by table name.

## 3. Decisions taken (confirmed with the user)

1. **Registry** — extend the existing codegen to emit a lazy `name → func() ScyllaTable` map.
2. **API scope** — ORM function + **one** authenticated generic handler. Existing per-table
   `*-ids` handlers stay untouched in this pass.
3. **Returned struct** — carries `ccv` and `ss` in addition to the requested fields, so the
   frontend `IMinimalRecord` contract (`frontend/libs/cache/cache-by-ids.svelte.ts:15`) holds.

## 4. Design

### 4.1 Schema config — `TableSchema.GenericRecord`

```go
// db/main.go
// GenericRecord maps a table's columns onto the flat shape returned by
// QueryCachedGenericByIDs, so one endpoint can resolve labels for any table.
type GenericRecordSchema struct {
    Name Coln // string  — the display label; required
    S1   Coln // string  — optional secondary text (e.g. SKU)
    S2   Coln // string  — optional
    N1   Coln // integer — optional numeric (e.g. BrandID, Price)
    N2   Coln // integer — optional
}

type TableSchema struct {
    ...
    GenericRecord GenericRecordSchema
}
```

Declared on `ProductTable.GetSchema()` (`backend/business/types/productos.go:137`):

```go
GenericRecord: db.GenericRecordSchema{Name: e.Name, S1: e.SKU, N1: e.FinalPrice, N2: e.BrandID},
```

`ID` and `ss` are **not** listed — they are always the table's single key column and the
`status` column, resolved automatically (same convention as
`db/text_search_index.go:50-56`, which looks up `columnsMap["status"]`).

### 4.2 Returned struct

```go
// db/cache_version_generic.go
type GenericRecord struct {
    ID           int64  `json:"ID"`            // always the table's single integer key
    Name         string `json:"nm,omitempty"`
    S1           string `json:"s1,omitempty"`
    S2           string `json:"s2,omitempty"`
    N1           int64  `json:"n1,omitempty"`
    N2           int64  `json:"n2,omitempty"`
    Status       int8   `json:"ss,omitempty"`  // 0 = tombstone, client caches it
    CacheVersion uint8  `json:"ccv,omitempty"` // required or the client re-fetches forever
}
```

`ID` keeps its capitalised JSON name because `cache-by-ids.svelte.ts` reads `record.ID`.
Everything else uses short tags to keep the payload small — that is the whole point of the
endpoint.

### 4.3 Resolved metadata on `ScyllaTable`

```go
type genericRecordInfo struct {
    nameCol, s1Col, s2Col, n1Col, n2Col, statusCol IColInfo
}
// ScyllaTable gains: genericRecord *genericRecordInfo
```

Built in `makeTable` (`db/reflect.go`) by a new `configureGenericRecordFields`, mirroring
`configureCacheVersionFields`. Validation panics at startup (fail-fast, consistent with the
rest of the ORM):

- `GenericRecord.Name` set but `SaveCacheVersion` false → panic (the function needs `ccv`).
- `Name`/`S1`/`S2` column not `string` → panic.
- `N1`/`N2` column not `int8|int16|int32|int64` → panic.
- `GenericRecord` empty → `genericRecord` stays `nil`; the table is simply not exposed.

`nil` genericRecord is the **allowlist**: `QueryCachedGenericByIDs` refuses any table that
did not opt in, so the one generic endpoint cannot be pointed at arbitrary tables.

### 4.4 Refactor of `QueryCachedIDs` — shared phase 1

`db/cache_version.go:400-589` splits into three pieces so the typed and generic paths share
the cache-version comparison verbatim (no duplicated logic):

```go
type cachedIDsPlan struct {
    idsToFetchByPartition  map[int32][]int64
    cacheVersionByPackedID map[int64]map[uint8]uint8
}

// Phase 1, extracted as-is from the current body (incl. the DebugFull collision/mismatch logs).
func planCachedIDsFetch(scyllaTable ScyllaTable, cachedIDs []IDCacheVersion) (cachedIDsPlan, error)

// Phase 2 helper: run the batched "part = ? AND key IN (...)" selects.
func forEachCachedIDsBatch(plan cachedIDsPlan, scyllaTable ScyllaTable, columnNames []string,
    runBatch func(queryString string, queryValues []any) error) error
```

`QueryCachedIDs` keeps its exact current behaviour and signature; its body becomes
`planCachedIDsFetch` → `forEachCachedIDsBatch` with `scanSelectQueryRows` inside the callback
→ `assignCacheVersionsToRecords`. Net line count of `cache_version.go` goes **down**.

### 4.5 `QueryCachedGenericByIDs`

```go
func QueryCachedGenericByIDs(tableName string, cachedIDs []IDCacheVersion) ([]GenericRecord, error)
```

1. `scyllaTable, err := resolveTableByName(tableName)` — registry lookup (§4.6).
2. Reject if `scyllaTable.genericRecord == nil` (table not opted in) or
   `!shouldUseCacheVersionFeature(scyllaTable)`.
3. `planCachedIDsFetch` — identical `cache_version` comparison; unchanged IDs are skipped
   exactly as today, and an all-cached request returns `nil, nil` with no table read.
4. Build the column list from the resolved `genericRecordInfo` (key, name, s1, s2, n1, n2,
   status — only the ones configured), then `forEachCachedIDsBatch`.
5. Scan raw via `iter.RowData()` + `scanner.Scan(...)` + `normalizeScannedValue` (the
   `view_tables.go:212` pattern — `scanSelectQueryRows` needs a mapped record type, which we
   deliberately do not have here). Convert with the existing `convertToInt64` /
   `convertToInt32` helpers in `db/converter.go`.
6. Set `ccv` per record from the phase-1 `cacheVersionByPackedID` using the same
   `uint8(recordID)` grouping as `assignCacheVersionsToRecords` — the grouping rule lives in
   one small helper both call, so the two paths can't drift.

### 4.6 Name registry (codegen)

`scripts/controllers/controllers_generator.go` currently AST-scans for
`db.TableStruct[XTable, X]`. Extend it to also read the `Name:` string literal out of
`func (e XTable) GetSchema()` and emit a second block into the same generated file:

```go
// backend/exec/controllers.generated.go  (generated, appended block)
func init() {
    db.RegisterTableFactory("products", func() db.ScyllaTable { return db.MakeScyllaTable[businessTypes.Product]() })
    db.RegisterTableFactory("client_provider", func() db.ScyllaTable { return db.MakeScyllaTable[businessTypes.ClientProvider]() })
    // ... one line per discovered table
}
```

- Cold start cost is a ~44-entry map of closures — **no reflection** until a name is actually
  requested. `MakeScyllaTable` then hits the existing `scyllaTableCache`, so the first generic
  query compiles one table and later ones are free.
- `main.go` already imports `app/exec`, so `init()` runs in both the HTTP server and the Lambda
  path. Nothing to call manually.
- If `GetSchema().Name` is not a plain string literal for some table, the generator **errors
  loudly** rather than silently skipping it.
- `db` side: a plain `map[string]func() ScyllaTable` behind a mutex + `RegisterTableFactory` /
  `resolveTableByName`, in the new `db/cache_version_generic.go`.

### 4.7 Company scoping — enforced in the router, before any handler runs

`ExtractCacheVersionValues` (`backend/core/cache.go:106`) resolves the partition as
`Coalesce(req.GetQueryInt("cmp"), req.User.CompanyID)` — the query parameter *wins*. That is
required by the **public** `p-products-ids` route (ecommerce, no token, company must come from
the query) but it means any authenticated caller could aim a private route at another company's
partition.

Fixed once in the request pipeline rather than in each handler. `main-handlers.go:126` already
has the single gate for private routes; right after `CheckUser` succeeds:

```go
// Private routes are always scoped to the token's company. Drop any client-sent "cmp" so a
// query parameter can never widen the request to another company's partition; Coalesce in
// ExtractCacheVersionValues then falls through to args.User.CompanyID.
delete(args.Query, "cmp")
```

Verified safe:

- `"cmp"` is read in **exactly one place** in the whole backend (`core/cache.go:106`).
- The frontend **never sends it** — no occurrence anywhere in `frontend/`.
- Public `p-` routes take the `else` branch, so `p-products-ids` behaves **exactly as before**.

Handlers therefore carry no company logic at all, and every existing private `*-ids` handler
(`usuarios-ids`, `client-provider-ids`, …) gets the same protection for free.

### 4.8 The generic handler

New authenticated route — proposed `"GET.generic-ids"` in `backend/system` (it is cross-module,
so it does not belong to `business`):

```go
func GetTableRecordsByIDs(req *core.HandlerArgs) core.HandlerResponse {
    tableName := req.GetQuery("table")
    if tableName == "" {
        return req.MakeErr("No se envió la tabla a consultar.")
    }
    // Company scoping already enforced by the router (§4.7) — nothing to validate here.
    cachedIDs := req.ExtractCacheVersionValues()
    if len(cachedIDs) == 0 {
        return req.MakeErr("No se enviaron ids a buscar.")
    }
    records, err := db.QueryCachedGenericByIDs(tableName, cachedIDs)
    if err != nil {
        return req.MakeErr("Error al obtener los registros.", err)
    }
    return core.MakeResponse(req, &records)
}
```

Wire protocol is unchanged from §6 of the `fetch-record-by-id-api` skill, plus `&table=<name>`:
`GET generic-ids?table=products&ids=…&cc-ids=…&cc-ver=…`

Registered **without** the `p-` prefix, so it is never reachable unauthenticated.

## 5. Which tables get `GenericRecord`

Rule: **exactly the tables that declare `SaveCacheVersion: true`** — the two features are
inseparable, since the generic query needs `ccv` to work at all. Enforced both ways in
`configureGenericRecordFields`:

- `GenericRecord` set + `SaveCacheVersion` false → **panic** at startup.
- `SaveCacheVersion` true + `GenericRecord` empty → allowed (opt-in stays incremental), the table
  is simply not resolvable by name.

Backend-wide there are exactly **three** such tables today:

| Table | File | `Name` | `S1` | `S2` | `N1` | `N2` |
|---|---|---|---|---|---|---|
| `products` | `business/types/productos.go:137` | `e.Name` | `e.SKU` | — | `e.FinalPrice` | `e.BrandID` |
| `client_provider` | `business/types/client_provider.go:65` | `t.Name` | `t.RegistryNumber` | — | `t.Type` | — |
| `users` | `core/types/users.go:61` | `t.User` | `t.FirstName` | `t.LastName` | — | — |

Notes:

- `users` has no single display column, so the login handle goes in `Name` and the two name parts
  in `S1`/`S2` for the client to compose. `Email` and `DocumentNumber` are **deliberately not
  mapped** — a label lookup doesn't need them.
- This is a **net reduction** in exposure, not an increase: `usuarios-ids` and
  `client-provider-ids` currently return the *entire* record via `QueryCachedIDs` (for `users`
  that includes `Email`, `DocumentNumber`, `ProfileIDs`, `AccessLevelIDs`, `AccesosComputed`).
  The generic route returns at most six scalar fields.
- Any table that later adds `SaveCacheVersion` also gets a `GenericRecord` block in the same edit,
  per this rule.

## 6. Work breakdown

| # | File | Change |
|---|---|---|
| 1 | `backend/db/main.go` | `GenericRecordSchema` type + `TableSchema.GenericRecord`; `ScyllaTable.genericRecord` |
| 2 | `backend/db/cache_version.go` | extract `planCachedIDsFetch` + `forEachCachedIDsBatch`; `QueryCachedIDs` reuses them (behaviour identical) |
| 3 | `backend/db/cache_version_generic.go` *(new)* | `GenericRecord`, `genericRecordInfo`, `configureGenericRecordFields`, registry, `QueryCachedGenericByIDs` |
| 4 | `backend/db/reflect.go` | call `configureGenericRecordFields` from `makeTable` |
| 5 | `scripts/controllers/controllers_generator.go` | read `GetSchema().Name`; emit `RegisterTableFactory` block |
| 6 | `backend/exec/controllers.generated.go` | regenerate |
| 7 | `business/types/productos.go`, `business/types/client_provider.go`, `core/types/users.go` | add `GenericRecord: {...}` per §5 (all three `SaveCacheVersion` tables) |
| 8 | `backend/main-handlers.go` | `delete(args.Query, "cmp")` in the private-route branch (§4.7) |
| 9 | `backend/system/…` + route map + `access_list.yml` | `GetTableRecordsByIDs`, `"GET.generic-ids"` |
| 10 | `backend/db/cache_version_generic_test.go` *(new)* | metadata resolution + column-list build + validation panics (no live Scylla) |
| 11 | `backend/docs/ORM_DATABASE_QUERY.md`, `.claude/skills/fetch-record-by-id-api/SKILL.md` | document the config + endpoint |

Verification: `go build ./...`, `go test ./db/...`, then `scripts/` static table validation
(`static-project-validation` skill) to confirm no schema convention broke.

## 7. Explicitly out of scope for this pass

- Deleting `GetProductsByIDs` / `client-provider-ids` / `usuarios-ids` and repointing the
  frontend (decision 2 keeps them).
- Frontend consumer (`getGenericRecordsByID` in `cache-by-ids.svelte.ts`,
  `RecordByIDText` switching to the generic route) — separate plan once the backend lands.
