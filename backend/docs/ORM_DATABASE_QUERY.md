# ScyllaDB ORM (db2)

A type-safe, reflection-based ORM for ScyllaDB/Cassandra with fluent query building and automatic schema management.

## Features

- **Type-safe queries** with compile-time checking using Go generics
- **Fluent query API** with method chaining
- **Automatic schema inference** from struct tags
- **Support for views and indexes**
- **Batch operations** for inserts and updates
- **Complex type handling** (structs, slices) with CBOR serialization

## Model Definition

### Basic Structure

Every model requires two structs:

1. **Base Struct**: Embeds `db.TableStruct` and contains your data fields
2. **Table Struct**: Defines columns as `db.Col` or `db.ColSlice` for query building

```go
// Base struct - holds the actual data
type ListaCompartidaRegistro struct {
    db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
    EmpresaID   int32    `db:"empresa_id,pk"`
    ID          int32    `db:"id,pk"`
    ListaID     int32    `db:"lista_id,view.1,view.2"`
    Nombre      string   `json:",omitempty" db:"nombre"`
    Images      []string `json:",omitempty" db:"images"`
    Descripcion string   `json:",omitempty" db:"descripcion"`
    Status      int8     `json:"ss,omitempty" db:"status,view.1"`
    Updated     int64    `json:"upd,omitempty" db:"updated,view.2"`
    UpdatedBy   int32    `json:",omitempty" db:"updated_by"`
}

// Table struct - for type-safe queries
type ListaCompartidaRegistroTable struct {
    db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
    EmpresaID   db.Col[ListaCompartidaRegistroTable, int32]
    ID          db.Col[ListaCompartidaRegistroTable, int32]
    ListaID     db.Col[ListaCompartidaRegistroTable, int32]
    Nombre      db.Col[ListaCompartidaRegistroTable, string]
    Images      db.ColSlice[ListaCompartidaRegistroTable, string]  // For slices
    Descripcion db.Col[ListaCompartidaRegistroTable, string]
    Status      db.Col[ListaCompartidaRegistroTable, int8]
    Updated     db.Col[ListaCompartidaRegistroTable, int64]
    UpdatedBy   db.Col[ListaCompartidaRegistroTable, int32]
}

// GetSchema defines the table structure and keys
func (e CajaMovimientoTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "caja_movimientos",
        Partition: e.EmpresaID,
        Keys:      []db.Coln{e.ID},
        // Pack multiple values into a single int64 ID
        KeyIntPacking: []db.Coln{
            e.CajaID.DecimalSize(5), 
            e.Fecha.DecimalSize(5), 
            e.Autoincrement(3), // Automated counter with 3 random digits
        },
        // Partition the autoincrement sequence by Fecha
        AutoincrementPart: e.Fecha,
    }
}
```

### Column Definition Rules

1. **Field Names**: Column names are inferred from field names converted to snake_case, or explicitly set with the `db` tag
2. **Primary Keys**: Use `db:"column_name,pk"` to mark partition/clustering keys
3. **DecimalSize**: Used for `KeyIntPacking` to define the digit width of a component.
4. **Autoincrement**: Marks a column (or a packing slot) to be automatically filled by the ORM's sequence manager.
5. **Both structs must have matching field names** to map columns correctly

## Connection Setup

```go
db.MakeScyllaConnection(db.ConnParams{
    Host:     "localhost",
    Port:     9042,
    User:     "cassandra",
    Password: "cassandra",
    Keyspace: "genix",
})
```

## CRUD Operations

### Insert

```go
records := []types.ListaCompartidaRegistro{
    {
        ID:          1,
        EmpresaID:   1,
        ListaID:     3,
        Nombre:      "Demo Record",
        Images:      []string{"img1.jpg", "img2.jpg"},
        Descripcion: "Description here",
        Status:      1,
        Updated:     time.Now().Unix(),
        UpdatedBy:   100,
    },
}

// Insert all fields
err := db.Insert(&records)

// Insert excluding specific columns
err := db.Insert(&records, q1.UpdatedBy)

// Insert single record
err := db.InsertOne(records[0])
```

### Update

```go
recordToUpdate := types.ListaCompartidaRegistro{
    ID:          1,
    EmpresaID:   1,
    ListaID:     3,
    Nombre:      "Updated Name",
    Images:      []string{"new1.jpg", "new2.jpg"},
    Status:      1,
    Updated:     time.Now().Unix(),
}

// Get table reference for column selection
q1 := db.Table[types.ListaCompartidaRegistro]()

// Update specific columns only
err := db.Update(&[]types.ListaCompartidaRegistro{recordToUpdate},
    q1.Status, q1.ListaID, q1.Nombre, q1.Images, q1.Updated)

// Update single record
err := db.UpdateOne(recordToUpdate, q1.Nombre, q1.Status)

// Update all fields except specified
err := db.UpdateExclude(&records, q1.UpdatedBy, q1.Created)
```

### Select/Query

```go
registros := []types.ListaCompartidaRegistro{}

// Build and execute query
query := db.Query(&registros)
query.Select().
    EmpresaID.Equals(1).
    ListaID.Equals(2).
    Status.Equals(1).
    AllowFilter()

if err := query.Exec(); err != nil {
    panic(err)
}

// Select specific columns
query.Select(q1.ID, q1.Nombre, q1.Status)

// Exclude specific columns
query.Exclude(q1.Images, q1.Descripcion)

// Additional query options
query.Limit(100).
    OrderDesc().
    AllowFilter()
```

### Query Operators

```go
// Comparison operators
query.ID.Equals(1)
query.Status.In(1, 2, 3)
query.Updated.GreaterThan(timestamp)
query.Updated.GreaterEqual(timestamp)
query.Updated.LessThan(timestamp)
query.Updated.LessEqual(timestamp)
query.Updated.Between(start, end)

// For slice columns
query.Images.Contains("image.jpg")
```

## Schema Definition

### Partition and Clustering Keys

```go
db.TableSchema{
    Name:      "table_name",
    Partition: e.CompanyID,           // Partition key
    Keys:      []db.Coln{e.ID},      // Clustering keys
}
```

### Packed Indexes (Local) - `Indexes`
Use `Indexes` when you need fast queries that include the partition key plus a composite predicate like:
- `Status IN (...)` + `Updated BETWEEN/GT`

Definition example:

```go
db.TableSchema{
    Name:      "sale_order",
    Partition: e.EmpresaID,
    Keys:      []db.Coln{e.ID},
    // Local packed index: creates a stored virtual packed column and a local index:
    // CREATE INDEX ... ON sale_order ((empresa_id), zz_ixp_status_updated)
    Indexes: [][]db.Coln{
        {e.Status.Int32(), e.Updated.DecimalSize(8)},
    },
}
```

Rules:
- First column MUST NOT set `DecimalSize()` (its width is implicit).
- All remaining columns MUST set `DecimalSize()`.
- Packed components must be non-negative integers.
- `.Int32()` means the packed column is stored as CQL `int` (computed as int64 first, trimmed to 9 digits, then cast).
- Truncation (via `DecimalSize`) can overfetch; the ORM will post-filter results to guarantee correctness.

### Global Indexes - `GlobalIndexes`
Use `GlobalIndexes` for:
1. Simple global secondary indexes (single column).
2. Packed composite global indexes (2+ columns) for equality-oriented lookups.

Single-column example:

```go
db.TableSchema{
    Name:      "users",
    Partition: e.EmpresaID,
    Keys:      []db.Coln{e.ID},
    GlobalIndexes: [][]db.Coln{
        {e.Email}, // CREATE INDEX users__email_index_0 ON users (email)
    },
}
```

Packed composite example:

```go
db.TableSchema{
    Name:      "sale_order",
    Partition: e.EmpresaID,
    Keys:      []db.Coln{e.ID},
    GlobalIndexes: [][]db.Coln{
        {e.Status.Int32(), e.Updated.DecimalSize(8)}, // CREATE INDEX ... ON sale_order (zz_gixp_status_updated)
    },
}
```

Important:
- Scylla global secondary indexes are not reliable for general range scans.
- The ORM intentionally does not route range queries (`BETWEEN`, `>`, `<`) through packed global indexes to avoid `ALLOW FILTERING` surprises.

### Views

Create materialized views for alternative query patterns:

```go
Views: []db.View{
    // Simple view with partition key
    {Cols: []db.Coln{e.ListaID, e.Status}, KeepPart: true},
    
    // Concatenated integer view (for range queries)
    {Cols: []db.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
    {Cols: []db.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
}
```

## Best Practices

1. **Always specify columns for Update**: This prevents accidental overwrites and improves performance
2. **Use batch operations**: Insert/Update multiple records at once for better performance
3. **Define views carefully**: Views duplicate data; only create them for essential query patterns
4. **Partition key design**: Choose partition keys that distribute data evenly
5. **Use AllowFilter() sparingly**: It forces full table scans; prefer indexed queries

## Type Support

- **Primitives**: `int8`, `int16`, `int32`, `int64`, `int`, `float32`, `float64`, `string`, `bool`
- **Slices**: Stored as ScyllaDB sets (use `db.ColSlice`)
- **Complex types**: Structs automatically serialized to CBOR and stored as blobs
