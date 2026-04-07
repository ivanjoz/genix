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
    CacheV    uint8    `db:"-" json:"ccv,omitempty"`
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
}
```

### Rules

- Field names must match between record and table structs.
- Column name is inferred from `db` tag or snake_case field name.
- Partition/key columns are defined in `GetSchema()` (not by tag alone).
- For cached delta APIs, record should expose `ID`, partition field, and cache-version response field (`uint8`).

---

## 3. Defining `GetSchema()`

```go
// Purpose: Declare partitioning, keys, and optional acceleration structures.
// Rationale: Query routing and write behavior are derived from this contract.
func (e ProductTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "product",
        Partition: e.EmpresaID,
        Keys:      []db.Coln{e.ID},

        // Optional: enable per-record cache-version support.
        SaveCacheVersion: true,
    }
}
```

---

## 4. Connection Setup

```go
// Purpose: Configure one Scylla session used by ORM operations.
// Rationale: Keyspace and credentials are shared by query/insert/update flows.
db.MakeScyllaConnection(db.ConnParams{
    Host:     "localhost",
    Port:     9042,
    User:     "cassandra",
    Password: "cassandra",
    Keyspace: "genix",
})
```

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
        Keys:      []db.Coln{e.ID},
        KeyIntPacking: []db.Coln{
            e.StoreID.DecimalSize(5),
            e.DayCode.DecimalSize(5),
            e.Autoincrement(3),
        },
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
        Keys:          []db.Coln{e.ID}, // string key field
        KeyConcatenated: []db.Coln{e.CustomerID, e.Year, e.Serial},
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
        Keys:      []db.Coln{e.ID},
        Indexes: []db.Index{
            {
                Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)},
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
        Keys: []db.Coln{e.Email},
    },
}
```

```go
// Purpose: Add packed global index for composite equality-oriented filters.
// Rationale: Supports compact global lookup shape on multiple numeric fields.
Indexes: []db.Index{
    {
        Type: db.TypeGlobalIndex,
        Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)},
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
        Type:     db.TypeView,
        Keys:     []db.Coln{e.CustomerID, e.Status},
        KeepPart: true,
    },
    {
        Type: db.TypeView,
        Keys: []db.Coln{e.StoreID.Int32(), e.Updated.DecimalSize(10)},
    },
}
```

### 7.6 Composite-Bucket Indexes

Use for range + multi-field membership scenarios over numeric dimensions.

```go
// Purpose: Create hash-set virtual indexes with bucketed range support.
// Rationale: Efficient for tuple-style filters plus bounded time/week ranges.
Indexes: []db.Index{
    {
        Type: db.TypeGlobalIndex,
        Keys: []db.Coln{
            e.ProductID,
            e.ChannelID,
            e.WeekCode.CompositeBucketing(1, 2, 4).StoreAsWeek(),
        },
    },
}
```

Rules:
- Each composite-bucket index supports 2 to 3 numeric source columns.
- Exactly one source column must define `CompositeBucketing(...)`.
- `StoreAsWeek()` enables week-based normalization for bucket/range math.

---

## 8. Cache-Version Delta Queries

If `SaveCacheVersion` is enabled in schema, you can fetch only changed records.

```go
// Purpose: Return only rows whose server cache-version changed.
// Rationale: Reduces payload for sync endpoints with client-side caches.
changed := []Product{}
cached := []db.IDCacheVersion{
    {PartitionID: 1, ID: 101, CacheVersion: 5},
    {PartitionID: 1, ID: 102, CacheVersion: 2},
}
err := db.QueryCachedIDs(&changed, cached)
```

Requirements:
- `SaveCacheVersion: true` in `GetSchema()`
- one partition field and one key field resolvable by ORM
- response model includes cache-version output field (`uint8`) if you expose it to client

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
    []db.Coln{q.Updated},
)
```

### 9.2 `Merge`

```go
// Purpose: Compare incoming rows against DB and apply insert/update selectively.
// Rationale: Reduces write churn when only changed rows should be updated.
q := db.Table[Product]()
err := db.Merge(
    &rows,
    []db.Coln{q.Updated},
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
