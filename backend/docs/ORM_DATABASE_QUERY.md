# ScyllaDB ORM Query Guide (`db`)

This guide explains how to model tables, define schema strategies, and execute queries with the Genix ORM from an application developer perspective.

It also documents the cloud ORM used by `app/cloud`, which mirrors the same model metadata for DynamoDB and Cloudflare D1 / SQLite.

---

## 1. What You Get

- Type-safe query building with Go generics
- Fluent query API (`Equals`, `In`, `Between`, etc.)
- Insert/update batch helpers
- Schema declaration in Go (`TableSchema`)
- Packed indexes, views, hash indexes, and smart key strategies
- Delta-log list sync (`Delta`) and slot-tick by-IDs fetch (`QueryCachedIDs`)

---

## 2. Model Structure

Each entity uses two structs:
1. **Record/Base struct**: actual persisted fields
2. **Table struct**: typed query columns (`db.Col`, `db.ColSlice`)

```go
// Purpose: Record struct holds persisted data and JSON payload fields.
// Rationale: Keep runtime data model separate from query builder fields.
type Product struct {
    db.TableStruct[ProductTable, Product]
    EmpresaID int32    `db:"empresa_id"`
    ID        int64    `db:"id"`
    Nombre    string   `db:"nombre"`
    Status    int8     `json:"ss,omitempty"`
    Tags      []string `db:"tags"`
    // Required by CacheByIDs. See §3.1.
    Updated int32 `json:"upd,omitempty"`
    // Required by a db.TypeDelta index. See §3.1.
    Version int16 `json:"ver,omitempty"`
    // Not a column: set only on the log records a Delta() read appends.
    Metadata *string `json:"mdt,omitempty"`
}

// Purpose: Table struct exposes typed fluent query fields.
// Rationale: Compile-time safety for query operators and schema references.
type ProductTable struct {
    db.TableStruct[ProductTable, Product]
    EmpresaID db.Col[*ProductTable, int32]
    ID        db.Col[*ProductTable, int64]
    Nombre    db.Col[*ProductTable, string]
    Status    db.Col[*ProductTable, int8]
    Tags      db.ColSlice[*ProductTable, string]
    Updated   db.Col[*ProductTable, int32]
    Version   db.Col[*ProductTable, int16]
}
```

### Rules

- Field names must match between record and table structs.
- Column name is inferred from `db` tag or snake_case field name.
- Partition/key columns are defined in `GetSchema()` (not by tag alone).
- Cached APIs need the managed fields of §3.1: `Updated` for `CacheByIDs`, `Version` + `Metadata` for `db.TypeDelta`.

---

## 3. Defining `GetSchema()`

```go
// Purpose: Declare partitioning, keys, and optional acceleration structures.
// Rationale: Query routing and write behavior are derived from this contract.
func (e ProductTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        // Required, unique across the whole project, never derived from the name. See §3.1.
        ID:        22,
        Name:      "product",
        Partition: e.EmpresaID,
        Keys:      db.Cols(e.ID),

        // Optional: enable the by-IDs slot cache (§8).
        CacheByIDs: true,
    }
}
```

### 3.1 `TableSchema.ID` and the managed cache columns

Every table declares an `ID` in `1..16383`, unique across the project. It names the table's cache
files in the file store (slot files and delta logs), so reusing or changing one silently repoints
them. `scripts` `check_tables` enforces presence, range and uniqueness statically; the ORM panics at
table-compile time as a backstop. ORM-internal tables claim IDs from the top of the range
(`sequences` holds `16383`).

The ORM stamps `updated` (SUnixTime) on every write. Each cache protocol needs a managed field to
reach the client, so the ORM panics unless both structs declare it:

| Opt-in | Record struct | Table struct |
|---|---|---|
| `CacheByIDs: true` | ``Updated int32 `json:"upd,omitempty"` `` | `Updated db.Col[*XTable, int32]` |
| a `db.TypeDelta` index | ``Version int16 `json:"ver,omitempty"` `` | `Version db.Col[*XTable, int16]` |
| a `db.TypeDelta` index | ``Metadata *string `json:"mdt,omitempty"` `` | — (never a column) |

- `Version` counts the record's writes (1 on insert, +1 per write, back to 1 after 32767). The ORM
  reads the stored version before each write, so only a table with a delta index pays for it. The
  log carries it, and the client skips a logged write whose version it already holds.
- `Metadata` is set only on the log records `Delta()` appends to its result (§6.1).
- A by-IDs read overwrites `Updated` with the slot tick the client must send back (§8).

To omit the default DB-only `created` and `updated` timestamp columns on a table without
`CacheByIDs`, set `DisableDefaultColumns: true` in `GetSchema()`.

---

## 4. Connection Setup

```go
// Purpose: Configure one Scylla session used by ORM operations.
// Rationale: Keyspace and credentials are shared by query/insert/update flows.
scylla.MakeScyllaConnection(scylla.ConnParams{
    Host:             "localhost",
    Port:             9042,
    User:             "cassandra",
    Password:         "cassandra",
    Keyspace:         "genix",
    MaxClusteringKey: 100,
})
```

`MaxClusteringKey` mirrors the node's `max_clustering_key_restrictions_per_query`
(a server-wide setting in `scylla.yaml`, not per keyspace). The backend passes
`db.max_clustering_key` from `config.toml` here; raise both together if you raise
the server's. The ORM splits any wider `In()` fanout into several queries, so a
larger list is never an error — only more round trips. 0 falls back to the
`MAX_CLUSTERING_KEY` env var, then to 100.

---

## 5. CRUD Operations

### 5.1 Insert

```go
// Purpose: Insert many rows in one call.
// Rationale: ORM batches writes for better throughput.
rows := []Product{
    {EmpresaID: 1, ID: 101, Nombre: "A", Status: 1, Updated: core.SUnixTime()},
    {EmpresaID: 1, ID: 102, Nombre: "B", Status: 1, Updated: core.SUnixTime()},
}
err := db.Insert(&rows)
```

```go
// Purpose: Skip selected columns during insert.
// Rationale: Useful when server-managed fields should not be written from input.
q := db.Table[Product]()
err := db.Insert(&rows, q.Updated)
```

```go
// Purpose: Insert one row with the same validation/path as bulk insert.
// Rationale: Single-row helper keeps API uniform.
err := db.InsertOne(rows[0])
```

### 5.2 Update

```go
// Purpose: Update only explicit fields.
// Rationale: Prevents accidental full-row overwrite.
q := db.Table[Product]()
row := Product{EmpresaID: 1, ID: 101, Nombre: "A+", Status: 2, Updated: core.SUnixTime()}
err := db.Update(&[]Product{row}, q.Nombre, q.Status, q.Updated)
```

```go
// Purpose: Update all mutable fields except selected ones.
// Rationale: Useful for broad updates while protecting audit/system columns.
err := db.UpdateExclude(&[]Product{row}, q.Updated)
```

```go
// Purpose: Update one row with explicit include list.
// Rationale: Keeps single-row path consistent with bulk update semantics.
err := db.UpdateOne(row, q.Nombre, q.Status)
```

### 5.3 Query / Select

```go
// Purpose: Build typed query and stream results into a slice.
// Rationale: Fluent API keeps query intent readable and safe.
results := []Product{}
query := db.Query(&results)

err := query.
    EmpresaID.Equals(1).
    Status.In(1, 2).
    Updated.Between(startUnix, endUnix).
    Limit(200).
    Exec()
```

```go
// Purpose: Read only required fields.
// Rationale: Reduces network and decode cost for list endpoints.
q := db.Query(&results)
err := q.Select(q.ID, q.Nombre, q.Updated).EmpresaID.Equals(1).Exec()
```

```go
// Purpose: Exclude expensive/large fields.
// Rationale: Useful when blobs/slices are not needed in current response.
q := db.Query(&results)
err := q.Exclude(q.Tags).EmpresaID.Equals(1).Exec()
```

```go
// Purpose: Process decoded rows while scanning and optionally avoid storing them.
// Rationale: Useful when callers need side effects or custom aggregation with lower memory usage.
q := db.Query(&results)
q.EmpresaID.Equals(1).Status.Equals(1)

latestByID := map[int64]int32{}

err := q.ExecScan(func(record *Product) bool {
    // Keep only the latest Updated value in the external accumulator.
    if record.Updated > latestByID[record.ID] {
        latestByID[record.ID] = record.Updated
    }
    return true
})
```

Rules:
- `ExecScan` runs after row decode and in-memory post-filtering.
- Returning `true` skips storing the row in the destination slice.
- `LIMIT` still applies to raw DB rows before `ExecScan`.

```go
// Purpose: Execute a real Scylla GROUP BY when the schema exposes a compatible key/view path.
// Rationale: Packed views let the ORM group multiple logical columns through one physical grouped key.
q := db.Query(&results)

err := q.
    EmpresaID.Equals(1).
    Fecha.GreaterEqual(startUnixDay).
    GroupBy(q.Fecha, q.ProductoID, q.Cantidad.Sum()).
    Exec()
```

Rules:
- `GroupBy()` emits a real Scylla `GROUP BY`, unlike `ExecScan`.
- `Avg()` requires a `float32` or `float64` destination field.
- Multi-column `GroupBy()` requires a compatible packed view whose key columns match the grouped columns in order.

### 5.4 Cloud Query (`cloud.Select`)

`cloud.Select` is the provider-agnostic query API used for DynamoDB and Cloudflare D1 / SQLite.

Rules:
- `Partition()` was removed from the cloud query API. Do not use it.
- If the model has a logical partition column such as `empresa_id`, every cloud query must include `Where("<partition_column>").Equals(...)`.
- DynamoDB requires exactly:
  - one `Where()` for the partition column using `Equals()`
  - one additional `Where()` over an indexed cloud column
- SQLite / D1 also requires the partition `Where()`, but it can combine multiple `Where()` clauses with `AND`.
- Calling `Where("column")` without an operator such as `Equals()` or `GreaterEqual()` returns an error on `Exec()`.

Example:

```go
// Purpose: Scope the cloud query by empresa_id and then query the sortable synthetic index.
// Rationale: Keeps DynamoDB and SQLite behavior aligned and prevents cross-company reads.
records := []types.Usuario{}
companyStatusUpdated := fmt.Sprintf("%d_%d_%020d", empresaID, 1, updated)

err := cloud.Select(&records).
    Where("empresa_id").Equals(empresaID).
    Where("company_status_updated").GreaterEqual(companyStatusUpdated).
    Exec()
```

Login-style point lookup:

```go
// Purpose: Resolve one user by empresa_id plus indexed company_usuario key.
// Rationale: DynamoDB requires the logical partition and one indexed predicate.
usuarios := []types.Usuario{}
companyUserIndex := fmt.Sprintf("%d_%s", empresaID, usuario)

err := cloud.Select(&usuarios).
    Where("empresa_id").Equals(empresaID).
    Where("company_usuario").Equals(companyUserIndex).
    Exec()
```

Common errors:
- `missing required partition filter Where("empresa_id").Equals(...)`
- `partition column empresa_id must use Equals()`
- `dynamo queries require exactly one indexed Where() in addition to the partition Where()`

---

## 6. Query Operators

```go
// Purpose: Equality and membership filters.
// Rationale: These are index-friendly for most schemas.
q.ID.Equals(1001)
q.Status.In(1, 2, 3)
```

```go
// Purpose: Numeric/time range filters.
// Rationale: Combine with packed indexes/views for efficient range routing.
q.Updated.GreaterThan(ts)
q.Updated.GreaterEqual(ts)
q.Updated.LessThan(ts)
q.Updated.LessEqual(ts)
q.Updated.Between(tsFrom, tsTo)
```

```go
// Purpose: Search item membership in slice-backed set columns.
// Rationale: Uses `CONTAINS` semantics for set-like fields.
q.Tags.Contains("featured")
```

### 6.1 Delta reads (`Delta`)

`Delta` turns a query into a delta-cache read over a `db.TypeDelta` index's change log (§7.5b).

```go
// Purpose: Read everything the client has not applied of the log yet.
// Rationale: One call covers the first sync and every later one.
query.Select().
    CompanyID.Equals(req.User.CompanyID).
    Type.Equals(int8(requestedType)).
    Delta(req.GetDeltaCursor(), int8(1))
```

- The cursor (`db.DeltaCursor`) is what the client holds of the log: `<bucket>.<bytes>` pairs, per
  10-minute log file still open, how many bytes it applied. `req.GetDeltaCursor(responseKey...)`
  parses it from the query param named after the response key (`up` for a bare-array response).
- **Empty cursor** → a first sync: lists the index's view, the sync-filter column pinned to the
  values passed.
- **Cursor** → reads the log files past it and returns the logged records in **any** status, so
  records that left the filter (soft-deleted) reach the clients still caching them.
- **Cursor older than the log** (7 days, or a day of files behind) → a *replace*: a first sync
  flagged so the client drops what the response doesn't bring.
- The result always ends with `Metadata` records — one per log file that moved, ID = −bucket, `mdt` =
  the file in Safe85 — which the client keeps as its next cursor. A handler that post-filters the
  result must keep them (`record.Metadata != nil`).
- **The sync-filter column is inferred, never named**: it is the index's last Key. The values are
  `...any`, so typed constants (`int8(1)`, `types.ProductStatusActive`) are fine. Pass none for a
  keyless index, or to list without a filter.
- The index is picked from the predicates already set: the one whose pinned Keys (all but the last)
  each have an equality. Call `Delta` last.

A cached route that cannot use `Delta` (no integer key, or a listing the log can't serve, such as a
status-0 tab) answers `core.WithDeltaReplace(records)`: the whole list plus one replace `Metadata`
record (`db.DeltaReplaceMetadata()`).

---

## 7. Advanced Schema Strategies

### 7.1 Key Packing (`KeyIntPacking`)

Use when a single `int64` key should encode multiple components.

```go
// Purpose: Compose one bigint key from multiple numeric components.
// Rationale: Keeps key compact while preserving sortable structure.
func (e MovementTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "movement",
        Partition: e.EmpresaID,
        Keys:      db.Cols(e.ID),
        KeyIntPacking: db.Cols(
            e.StoreID.Size(17), // 0..131,071
            e.DayCode.Size(15), // a UnixDay: 0..32,767
            e.Autoincrement(8), // sequence<<8 | 8 random bits
        ),
        AutoincrementPart: e.DayCode,
    }
}
```

Rules:
- Use exactly one key column (`int64`) in `Keys`.
- `.Size(bits)` is a range cap: the column's values must stay below 2^bits. A value that does not
  fit panics on write; it is never truncated.
- Every component but the last must define `.Size()`; the last one (normally `Autoincrement(n)`)
  takes the rest of the 63-bit budget. The ID stays a positive `int64`.
- `Autoincrement(randomBits)` may be used as a packed component placeholder.
- `db.DecodePackedKey[RecordT](id)` splits an ID back into its components for debugging.

### 7.2 Key Concatenation (`KeyConcatenated`)

Use when one string key should encode multiple fields.

```go
// Purpose: Flatten multi-field key into a compact string key.
// Rationale: Enables prefix/range-style key patterns with one PK column.
func (e InvoiceTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:          "invoice",
        Partition:     e.EmpresaID,
        Keys:          db.Cols(e.ID), // string key field
        KeyConcatenated: db.Cols(e.CustomerID, e.Year, e.Serial),
    }
}
```

### 7.3 Local Packed Indexes (`Indexes`)

Use for partition + composite predicate patterns.

```go
// Purpose: Accelerate `partition + status + updated-range` style reads.
// Rationale: Packs multiple numeric predicates into one indexed virtual column.
func (e OrderTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "sale_order",
        Partition: e.EmpresaID,
        Keys:      db.Cols(e.ID),
        Indexes: []db.Index{
            {
                Keys: db.Cols(e.Status, e.Updated.Size(31)),
            },
        },
    }
}
```

### 7.4 Global Indexes

Use for cross-partition equality lookups.

```go
// Purpose: Add a global secondary index for direct equality lookups.
// Rationale: Useful for unique-like lookup fields such as email.
Indexes: []db.Index{
    {
        Type: db.TypeGlobalIndex,
        Keys: db.Cols(e.Email),
    },
}
```

```go
// Purpose: Add packed global index for composite equality-oriented filters.
// Rationale: Supports compact global lookup shape on multiple numeric fields.
Indexes: []db.Index{
    {
        Type: db.TypeGlobalIndex,
        Keys: db.Cols(e.Status, e.Updated.Size(31)),
    },
}
```

Important:
- Do not depend on global indexes for general range scans.
- Prefer local packed indexes or views for robust range workloads.

### 7.5 Views

```go
// Purpose: Materialize alternative query paths.
// Rationale: Duplicate data intentionally for read patterns you must optimize.
Indexes: []db.Index{
    {
        Type: db.TypeView,
        Keys: db.Cols(e.CustomerID, e.Status),
    },
    {
        Type: db.TypeView,
        Keys: db.Cols(e.StoreID, e.Updated.Size(31)),
    },
    // Partition overrides the view partition; without it the view keeps the
    // table partition (company_id) as its own, which is what you normally want.
    {
        Type:      db.TypeView,
        Keys:      db.Cols(e.ID),
        Partition: e.ID,
    },
}
```

### 7.5b Delta Views (`db.TypeDelta`)

Use for any table the frontend syncs as a list. Every write appends `(log ID, Version, deleted)` to a
change-log file in the file store — one per partition, pinned value and 10-minute bucket — which
`query.Delta()` reads past the client's cursor (§6.1).

```go
// Purpose: Log every write so Delta() can answer a sync from the log.
// Rationale: Type is pinned (one log per type, as the route asks for one); Status is the sync filter.
func (t ClientProviderTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        ID:        1,
        Name:      "client_provider",
        Partition: t.CompanyID,
        Keys:      db.Cols(t.ID.Autoincrement(0)),
        FixedValues: []db.FixedValues{
            {Col: t.Status, Values: []int64{0, 1}},
            {Col: t.Type, Min: 1, Max: 2},
        },
        Indexes: []db.Index{
            {Type: db.TypeDelta, Keys: db.Cols(t.Type, t.Status)},
        },
    }
}
```

Rules:
- The table needs an **integer partition** and **exactly one integer key** — the log ID. A table with
  a string or composite key cannot have a `TypeDelta` index; give it a `TypeView` and answer its
  route with `core.WithDeltaReplace(records)` instead.
- The record and table structs must declare `Version` (and the record `Metadata`) — §3.1.
- **The last Key is the sync filter** (usually `Status`): a first sync lists the view filtered on it.
  **Earlier Keys are pinned**: each value has its own log folder, and `Delta()` needs an equality on
  each. Keyless `{Type: db.TypeDelta}` keeps one log per partition with no sync filter.
- A write whose status column (named `status`) is 0 is logged as a soft delete; so is a hard delete,
  or a pinned value that changed (the record leaves that folder's log).
- The ORM compiles the Keys into the listing view a first sync reads: a simple `TypeView` for one
  key, a packed view for several, none for a keyless index.
- For a packed view, `FixedValues` (`Values` list, or `Min`/`Max`) size each key's bit slot; at most
  one key may go without, and it absorbs the remaining bits. The column is `int` when the slots fit,
  else `bigint`. `.Size()` / `.Int32()` remain as escape hatches. Writing a row outside its declared
  range panics rather than corrupting the packed key.
- `Partition` may relocate the view partition, same as `TypeView`.

### 7.6 Composite-Bucket Indexes

Use for range + multi-field membership scenarios over numeric dimensions.

```go
// Purpose: Create hash-set virtual indexes with bucketed range support.
// Rationale: Efficient for tuple-style filters plus bounded time/week ranges.
Indexes: []db.Index{
    {
        Type: db.TypeGlobalIndex,
        Keys: db.Cols(
            e.ProductID,
            e.ChannelID,
            e.WeekCode.CompositeBucketing(1, 2, 4).StoreAsWeek(),
        ),
    },
}
```

Rules:
- Each composite-bucket index supports 2 to 3 numeric source columns.
- Exactly one source column must define `CompositeBucketing(...)`.
- `StoreAsWeek()` enables week-based normalization for bucket/range math.

---

## 8. By-IDs Cache (`CacheByIDs` + `QueryCachedIDs`)

With `CacheByIDs: true`, every write raises its record's slot (`uint8(ID)`, 256 per partition) in a
per-partition slot file in the file store, holding the year tick of the slot's last write. A by-IDs
read sends the ticks the client holds and gets back only the records whose slot moved past them.

```go
// Purpose: Return only the requested records whose slot moved since the client's tick.
// Rationale: Revalidating a cached record costs one slot-file GET, not a row read.
cachedIDsRequest, err := req.GetCachedIDsRequest(req.User.CompanyID) // decodes ?cc=
if err != nil {
    return req.MakeErr(err)
}
partitions, err := db.QueryCachedIDs[types.Product](cachedIDsRequest) // []db.CachedPartition[E]{p, r}
if err != nil {
    return req.MakeErr("Error al obtener los productos.", err)
}
return req.MakeResponse(core.HideCachedPartitions(partitions))
```

Requirements:
- `CacheByIDs: true` in `GetSchema()`, a single integer key, and the `Updated` field of §3.1.
- Genix clients send every ID under partition 0; `GetCachedIDsRequest` substitutes the company and
  refuses any other partition, and `HideCachedPartitions` puts the 0 back on the response.
- Each returned record's `Updated` is overwritten with the slot tick the client sends back next time.
- A static lookup (the record never changes for its ID) uses `req.GetStaticRequestIDs()` on the same
  `cc` param, ignores ticks, and answers `core.StaticCachedPartition(records)`.

### 8.1 Generic By-ID Reads (`QueryCachedGenericByIDs`)

When the caller only needs a **display label** — not the whole record — a table can opt into a
table-agnostic read resolved by table *name*, so one endpoint serves every table instead of one
handler plus one full-record payload each.

Declare the mapping in `GetSchema()`:

```go
// Purpose: expose this table's label fields through the shared generic by-IDs endpoint.
// Rationale: a product reference only needs name/SKU/price, not the full row.
GenericRecord: db.GenericRecordSchema{
    Name: e.Name, S1: e.SKU, N1: e.FinalPrice, N2: e.BrandID,
},
```

- `Name` is required and must be a `string` column. `S1`/`S2` are optional strings; `N1`/`N2` are
  optional integers of any width.
- `ID` and `Status` are **not** declared — they are always the table's single key column and its
  `status` column, resolved automatically.
- `GenericRecord` **requires** `CacheByIDs: true` (panics at table build otherwise), which is
  also what guarantees the key is a single integer column.

Query it by name:

```go
partitions, err := db.QueryCachedGenericByIDs("products", cachedIDsRequest)
// -> []db.CachedPartition[db.GenericRecord]; GenericRecord{ID, Name, S1, S2, N1, N2, Status, Updated}
```

Slot filtering is identical to `QueryCachedIDs`: IDs whose slot did not move past the client's tick
are never read from the table.

Notes:
- A table with no `GenericRecord` is **rejected** — the config is the exposure allowlist.
- Name resolution uses a registry populated by generated code. After adding a table, run
  `./app.sh generate_controllers`.
- Column lookup, the SELECT projection and the per-column scan/assign closures are precompiled once
  per table, so the read loop does no reflection or type switching per row.
- HTTP route: `GET records-by-ids?table=<name>&cc=…` (authenticated only).

---

## 9. Merge and Upsert Helpers

### 9.1 `InsertOrUpdate`

```go
// Purpose: Split a batch into insert/update using a custom predicate.
// Rationale: Reuse one API call for mixed write payloads.
q := db.Table[Product]()
err := db.InsertOrUpdate(
    &rows,
    func(r *Product) bool { return r.ID <= 0 },
    db.Cols(q.Updated),
)
```

### 9.2 `Merge`

```go
// Purpose: Compare incoming rows against DB and apply insert/update selectively.
// Rationale: Reduces write churn when only changed rows should be updated.
q := db.Table[Product]()
err := db.Merge(
    &rows,
    db.Cols(q.Updated),
    func(prev, cur *Product) bool { return prev.Nombre != cur.Nombre || prev.Status != cur.Status },
    func(r *Product) { r.Updated = core.SUnixTime() },
)
```

---

## 10. Best Practices

1. Validate required fields server-side before insert/update.
2. For updates, pass explicit column list whenever possible.
3. Design partition keys for even distribution and practical query locality.
4. Prefer schema-driven routing (indexes/views) over `AllowFilter()`.
5. Use local packed indexes or range views for heavy range workloads.
6. Use `QueryCachedIDs` for sync endpoints with client cache metadata.
7. Keep slice field types consistent across model and usage (`[]int32` with `[]int32`, etc.).
8. Benchmark critical query patterns before and after adding views/indexes.

---

## 11. Supported Field Types

- **Primitives**: `int8`, `int16`, `int32`, `int64`, `int`, `float32`, `float64`, `string`, `bool`
- **Pointers**: pointer equivalents for nullable scalar values
- **Slices / set-backed by default**: `[]string`, numeric slices, and pointer-to-slice variants
- **Table-field default freeze policy**:
  - `db.Col[..., []T]` defaults to `frozen<set<...>>`
  - `db.ColSlice[..., T]` defaults to `set<...>` (not frozen)
- **Collection tag options**:
  - `db:",set"` forces `set<...>`
  - `db:",frozen"` forces `frozen<list<...>>`
  - `db:",frozen,set"` (or `db:",set,frozen"`) forces `frozen<set<...>>`
- **Complex structs/maps/slices**: persisted as a colbin `blob`

---

## 12. Common Errors and Fixes

- **"use ALLOW FILTERING"**: query shape is not covered by your keys/indexes/views.
  - Fix: add suitable `Indexes` entries or adjust predicates.

- **Packed index overfetch concerns**:
  - Fix: keep `Size` design coherent and rely on ORM post-filter exactness.

- **Autoincrement/key packing panic**:
  - Fix: verify `KeyIntPacking` rules (single bigint key, `.Size()` bits that fit 63, every value below its slot's 2^bits).

- **Composite bucketing config panic**:
  - Fix: ensure exactly one `CompositeBucketing(...)` column in each composite-bucket `Indexes` entry.

---

## 12b. Name-Addressed Access (tables resolved at runtime)

Everything above needs the record type at compile time. Tooling that only holds a table *name* — an
admin script, a backup job, the `fn-db` console — reaches the same data through `db.Controller`,
which gained four methods for it:

```go
controller, err := db.ResolveControllerByName("products")

schema := controller.DescribeTable()                          // columns, keys, query shapes
payload, count, err := controller.QueryRecordsJSON(db.QuerySpec{
    Filters: []db.FilterSpec{{Column: "company_id", Operator: "=", Value: 1}},
    Columns: []string{"id", "nombre"},
    Limit:   50,
})
written, err := controller.InsertRecordsJSON(payload, nil)
written, err := controller.UpdateRecordsJSON(payload, []string{"nombre"})
```

- Records cross the boundary as the **record struct's own JSON**, so a row that was read can be
  edited and written straight back. Filters instead name **columns** (`company_id`, `nombre`);
  `ResolveColumn` also accepts the Go field name, so either spelling works.
- Writes run through the ordinary `Insert` / `Update`, so autoincrement keys, `created`/`updated`,
  `version`, the cache slot files and delta logs, virtual columns, views and the text index all still apply.
- `FilterSpec.Value` is coerced to the column's Go type (`db.CoerceToColumn`). This is mandatory,
  not cosmetic: `encoding/json` decodes every number as `float64`, and the accessor engine's
  `ToInt64` returns 0 for a `float64` — an uncoerced `id` filter would silently become `id = 0`.
- `TableDescription.QueryShapes` lists the predicate combinations the compiled table can serve
  through a key, index or view (`column|op` pairs; `=` equality, `~` range, `@` CONTAINS). Anything
  outside it needs `AllowFilter`.
- The controller registry is populated by whoever owns the table list — in this project
  `app/exec`, lazily, from `MakeScyllaControllers()`. It is deliberately not built at startup:
  compiling 40-odd schemas is wasted work on a Lambda cold start.

There is no generic delete, and no generic `GroupBy` / `Delta` / text search: those stay on the
typed API.

Implementation: `genix-orm/db/generic_access.go`, `genix-orm/scylla/controller_generic.go`.
CLI: `cd backend && go run . fn-db '{"op":"describe","table":"products"}'` — see the
`database-records` skill.

---

## 13. Assigned Table IDs

Every `GetSchema()` declares a unique `TableSchema.ID` (§3.1). New tables take the next free number;
IDs are never reused or renumbered, because they name the table's cache files.
`check_tables` fails the build on a duplicate or a missing one.

| ID | Table | ID | Table |
|---:|---|---:|---|
| 1 | client_provider | 27 | sales_planning |
| 2 | agent_messages | 28 | seasonality_curve |
| 3 | cache | 29 | shared_list_records |
| 4 | cache_global | 30 | shipping_costs |
| 5 | cash_bank_movements | 31 | sites |
| 6 | cash_banks | 32 | supply_material |
| 7 | cash_reconciliations | 33 | system_parameters |
| 8 | city_locations | 34 | usage_log |
| 9 | companies | 35 | users |
| 10 | cron_actions | 36 | warehouse_product_movement |
| 11 | delivery_order_note | 37 | warehouse_product_stock |
| 12 | ecommerce_page_content | 38 | warehouse_product_stock_detail |
| 13 | expenses | 39 | warehouses |
| 14 | expenses_scheduled | 40 | webpages |
| 15 | gallery_images | 41 | zz_demo_struct |
| 16 | image_assets | 42 | _retired_ (was credit_usage, split into 50/51) |
| 17 | image_assets_category | 43 | sign_up_requests |
| 18 | parameters | 44 | user_logs |
| 19 | product_sale_summary | 45 | request_errors |
| 20 | product_stock_lot | 46 | contact_messages |
| 21 | product_supply | 47 | server_metrics |
| 22 | products | 48 | company_credit_budget |
| 23 | profiles | 49 | credit_history |
| 24 | purchase_order | 50 | credit_usage_company |
| 25 | sale_order | 51 | credit_usage_user |
| 26 | _retired_ | | |

**Next free ID: 52.**
