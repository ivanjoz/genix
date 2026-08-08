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
- Optional per-record cache-version selective fetch (`QueryCachedIDs`)

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
    Status    int8     `db:"status"`
    Updated   int64    `db:"updated"`
    Tags      []string `db:"tags"`
    // Required on any table that is synced incrementally. See §3.1.
    UpdatedVersion int32 `json:"upv,omitempty"`
}

// Purpose: Table struct exposes typed fluent query fields.
// Rationale: Compile-time safety for query operators and schema references.
type ProductTable struct {
    db.TableStruct[ProductTable, Product]
    EmpresaID db.Col[ProductTable, int32]
    ID        db.Col[ProductTable, int64]
    Nombre    db.Col[ProductTable, string]
    Status    db.Col[ProductTable, int8]
    Updated   db.Col[ProductTable, int64]
    Tags      db.ColSlice[ProductTable, string]
    UpdatedVersion db.Col[ProductTable, int32]
}
```

### Rules

- Field names must match between record and table structs.
- Column name is inferred from `db` tag or snake_case field name.
- Partition/key columns are defined in `GetSchema()` (not by tag alone).
- For cached delta APIs, record must expose `ID`, the partition field, and `UpdatedVersion int32` with the json tag `upv`.

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

        // Optional: enable the by-IDs slot-version cache.
        SaveUpdatedVersion: true,
    }
}
```

### 3.1 `TableSchema.ID` and `updated_version`

Every table declares an `ID` in `1..16383`, unique across the project. It is packed into the
partition key of `cache_updated_version` alongside the tenant, so reusing or changing one silently
repoints a table's cached slot versions. `scripts` `check_tables` enforces presence, range and
uniqueness statically; the ORM panics at table-compile time as a backstop. ORM-internal tables claim
IDs from the top of the range (`sequences` holds `16383`).

`updated_version` is a managed column, like `created` and `updated`: the ORM assigns it on every
write from a per-partition sequence — the same counter call that hands out autoincrement IDs. Unlike
a timestamp it is strictly increasing and never collides, which is what makes both caches exact.

A table must expose it to the client when it declares `SaveUpdatedVersion: true` **or** a
`db.TypeDelta` index; the ORM panics otherwise. Exposing it means both structs:

```go
// record struct — the json tag is the frontend contract
UpdatedVersion int32 `json:"upv,omitempty"`
// table struct — without this the column is DB-only and never selected
UpdatedVersion db.Col[ProductTable, int32]
```

The field name snake-cases to `updated_version`, so no `db` tag is needed. Tables without a
`SaveUpdatedVersion`, `db.TypeDelta`, or declared `UpdatedVersion` field omit it automatically,
saving the counter read. To also omit the default DB-only `created` and `updated` timestamp
columns, set `DisableDefaultColumns: true` in `GetSchema()`.

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
    {EmpresaID: 1, ID: 101, Nombre: "A", Status: 1, Updated: time.Now().Unix()},
    {EmpresaID: 1, ID: 102, Nombre: "B", Status: 1, Updated: time.Now().Unix()},
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
row := Product{EmpresaID: 1, ID: 101, Nombre: "A+", Status: 2, Updated: time.Now().Unix()}
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

`Delta` replaces the hand-written watermark branch a delta-cache handler used to carry. It requires
a `db.TypeDelta` index (§7.5b).

```go
// Purpose: Read everything the client has not seen since its watermark.
// Rationale: One call covers both halves of the sync and routes to the packed delta view.
query.Select().
    CompanyID.Equals(req.User.CompanyID).
    Type.Equals(int8(requestedType)).
    Delta(updatedSince, 1)
```

- `updatedSince` is an `updated_version`, not a timestamp — the client's highest received value.
- `updatedSince > 0` → `updated_version >= updatedSince+1`, fanned out over **every** declared value
  of the filter column, so rows that flipped to an inactive value still reach the client for eviction.
- `updatedSince == 0` → a first sync: the filter column is pinned to the arguments and the whole
  version slot is scanned (versions start at 1, so nothing is excluded).
- **The filter column is inferred, never named**: it is `Keys[0]` of the first `db.TypeDelta` index —
  usually `Status`, but any low-cardinality column works.
- Pass no value to constrain nothing but the watermark.
- The variadic is `...int64`. Untyped literals (`Delta(w, 1)`) are fine; a *typed* `int8` constant
  needs `int64(…)`.
- Position in the chain does not matter.
- It panics on a declaration bug — no `TypeDelta` index, no `FixedValues` for the filter column, or
  several `TypeDelta` indexes disagreeing on `Keys[0]`. The message names the fix.

The bound is exact: because `updated_version` is a sequence rather than a timestamp, two writes in
the same second are distinguishable and the boundary rows are not re-sent on every poll.

**Known limitation.** Versions are reserved before the write commits, so two overlapping writers can
commit out of order: writer A reserves 100, writer B reserves 101 and commits first, a client polls
and stores watermark 101, then A commits. A's rows are never delivered until they are written again.
The window is the few milliseconds between two concurrent writes on the same tenant and table. This
is accepted, not fixed.

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
            e.StoreID.DecimalSize(5),
            e.DayCode.DecimalSize(5),
            e.Autoincrement(3),
        ),
        AutoincrementPart: e.DayCode,
    }
}
```

Rules:
- Use exactly one key column (`int64`) in `Keys`.
- First packing component must not define `DecimalSize()`.
- Remaining components should define `DecimalSize()`.
- `Autoincrement(size)` may be used as a packed component placeholder.

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
                Keys: db.Cols(e.Status.Int32(), e.Updated.DecimalSize(8)),
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
        Keys: db.Cols(e.Status.Int32(), e.Updated.DecimalSize(8)),
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
        Keys: db.Cols(e.StoreID.Int32(), e.Updated.DecimalSize(10)),
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

Use for any table the frontend syncs incrementally. One declaration replaces the
`[Filter, Updated]` + `[Filter, Status]` view pair the delta pattern used to need, and pairs with
`query.Delta()` (§4.7).

```go
// Purpose: Serve both halves of a delta-cache sync from one packed view.
// Rationale: The declared value ranges size each digit slot, so the engine — not the developer —
// decides whether the packed column fits an int.
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
            // updated_version is appended implicitly. Keys[0] is the column Delta() filters on.
            {Type: db.TypeDelta, Keys: db.Cols(t.Status, t.Type)},
        },
    }
}
```

Rules:
- Every declared key needs a `FixedValues` entry (`Values` list, or `Min`/`Max`). That range sizes
  its digit slot; the schema panics without one.
- Do **not** list `UpdatedVersion` — it is implicit and takes the trailing slot.
- No `.DecimalSize()` or `.Int32()` needed. Both remain as escape hatches; a forced `.Int32()` that
  cannot hold the declared ranges panics with the computed maximum.
- The packed column is `int` when the maximum packed value fits `2,147,483,647`, else `bigint`.
  `updated_version` gets 8 digits in the `int` case and 10 in the `bigint` case, since the digits are
  already paid for. 8 digits caps the table at 10^8 write calls per partition; a write past that
  fails loudly rather than silently truncating the packed key.
- **`Keys[0]` decides the fit** — it is the most significant slot. `[Status{0,1}, Type{1,2}]` reaches
  `1_2_99999999` → `int`; reversed it reaches `2_1_99999999` → `bigint`. Declare the most tightly
  bounded column first. When `bigint` is chosen the compiler logs which order would have fit.
- `Keys[0]` is also `Delta()`'s filter column, so the two roles are coupled. If they conflict, put
  the filter column first and accept `bigint`.
- Writing a row outside its declared range panics rather than corrupting the packed key.
- `Partition` may relocate the view partition, same as `TypeView`.

A delta view doubles as an ordinary index, because a packed base-10 key is prefix-searchable:

| Query | Plan |
|---|---|
| `Status = 1` | range over the whole `Status = 1` block |
| `Status = 1 AND Type = 2` | narrower range inside it |
| `Status IN (0,1)` | one range per value |
| `Status = 1 AND Type = 2 AND Updated > W` | the full delta shape |

A gap is refused rather than silently dropped: `Type = 2` without `Status` cannot be one range, so
the view declines and the planner looks elsewhere. Partial prefixes rank below a local index, so
`Status = 1 AND RegistryNumber = 'X'` uses the `registry_number` index instead.

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

## 8. Cache-Version Delta Queries

If `SaveUpdatedVersion` is enabled in schema, you can fetch only changed records.

```go
// Purpose: Return only rows whose server cache-version changed.
// Rationale: Reduces payload for sync endpoints with client-side caches.
changed := []Product{}
cached := []db.IDUpdatedVersion{
    {PartitionID: 1, ID: 101, UpdatedVersion: 5},
    {PartitionID: 1, ID: 102, UpdatedVersion: 2},
}
err := db.QueryCachedIDs(&changed, cached)
```

Requirements:
- `SaveUpdatedVersion: true` in `GetSchema()`
- one partition field and one key field resolvable by ORM
- response model includes cache-version output field (`uint8`) if you expose it to client

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
- `GenericRecord` **requires** `SaveUpdatedVersion: true` (panics at table build otherwise), which is
  also what guarantees the key is a single integer column.

Query it by name:

```go
records, err := db.QueryCachedGenericByIDs("products", cachedIDs)
// -> []db.GenericRecord{{ID, Name, S1, S2, N1, N2, Status, UpdatedVersion}}
```

Slot-version filtering is identical to `QueryCachedIDs`: IDs whose client `upv` still matches the
server are never read from the table.

Notes:
- A table with no `GenericRecord` is **rejected** — the config is the exposure allowlist.
- Name resolution uses a registry populated by generated code. After adding a table, run
  `./app.sh generate_controllers`.
- Column lookup, the SELECT projection and the per-column scan/assign closures are precompiled once
  per table, so the read loop does no reflection or type switching per row.
- HTTP route: `GET records-by-ids?table=<name>&ids=…&cc-ids=…&cc-ver=…` (authenticated only).

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
    func(r *Product) { r.Updated = time.Now().Unix() },
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
- **Complex structs/maps/slices**: persisted as CBOR `blob`

---

## 12. Common Errors and Fixes

- **"use ALLOW FILTERING"**: query shape is not covered by your keys/indexes/views.
  - Fix: add suitable `Indexes` entries or adjust predicates.

- **Packed index overfetch concerns**:
  - Fix: keep `DecimalSize` design coherent and rely on ORM post-filter exactness.

- **Autoincrement/key packing panic**:
  - Fix: verify `KeyIntPacking` rules (single bigint key, decimal sizes, non-negative domain).

- **Composite bucketing config panic**:
  - Fix: ensure exactly one `CompositeBucketing(...)` column in each composite-bucket `Indexes` entry.

---

## 13. Assigned Table IDs

Every `GetSchema()` declares a unique `TableSchema.ID` (§3.1). New tables take the next free number;
IDs are never reused or renumbered, because they are packed into `cache_updated_version`'s key.
`check_tables` fails the build on a duplicate or a missing one.

| ID | Table | ID | Table |
|---:|---|---:|---|
| 1 | client_provider | 22 | products |
| 2 | agent_messages | 23 | profiles |
| 3 | cache | 24 | purchase_order |
| 4 | cache_global | 25 | sale_order |
| 5 | cash_bank_movements | 26 | sale_summary |
| 6 | cash_banks | 27 | sales_planning |
| 7 | cash_reconciliations | 28 | seasonality_curve |
| 8 | city_locations | 29 | shared_list_records |
| 9 | companies | 30 | shipping_costs |
| 10 | cron_actions | 31 | sites |
| 11 | delivery_order_note | 32 | supply_material |
| 12 | ecommerce_page_content | 33 | system_parameters |
| 13 | expenses | 34 | usage_log |
| 14 | expenses_scheduled | 35 | users |
| 15 | gallery_images | 36 | warehouse_product_movement |
| 16 | image_assets | 37 | warehouse_product_stock |
| 17 | image_assets_category | 38 | warehouse_product_stock_detail |
| 18 | parameters | 39 | warehouses |
| 19 | product_sale_summary | 40 | webpages |
| 20 | product_stock_lot | 41 | zz_demo_struct |
| 21 | product_supply | 16383 | sequences (ORM-internal) |

**Next free ID: 42.**
