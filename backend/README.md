# Golang Genix Lambda Compatible Backend
This is the backend for the Genix ERP + Ecommerce project

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

1. **Base Struct**: Embeds `db2.TableStruct` and contains your data fields
2. **Table Struct**: Defines columns as `db2.Col` or `db2.ColSlice` for query building

```go
// Base struct - holds the actual data
type ListaCompartidaRegistro struct {
    db2.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
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
    db2.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
    EmpresaID   db2.Col[ListaCompartidaRegistroTable, int32]
    ID          db2.Col[ListaCompartidaRegistroTable, int32]
    ListaID     db2.Col[ListaCompartidaRegistroTable, int32]
    Nombre      db2.Col[ListaCompartidaRegistroTable, string]
    Images      db2.ColSlice[ListaCompartidaRegistroTable, string]  // For slices
    Descripcion db2.Col[ListaCompartidaRegistroTable, string]
    Status      db2.Col[ListaCompartidaRegistroTable, int8]
    Updated     db2.Col[ListaCompartidaRegistroTable, int64]
    UpdatedBy   db2.Col[ListaCompartidaRegistroTable, int32]
}

// GetSchema defines the table structure and keys
func (e ListaCompartidaRegistroTable) GetSchema() db2.TableSchema {
    return db2.TableSchema{
        Name:      "lista_compartida_registro",
        Partition: e.EmpresaID,              // Partition key
        Keys:      []db2.Coln{e.ID},         // Clustering key(s)
        Views: []db2.View{
            {Cols: []db2.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
            {Cols: []db2.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
        },
    }
}
```

### Column Definition Rules

1. **Field Names**: Column names are inferred from field names converted to snake_case, or explicitly set with the `db` tag
2. **Primary Keys**: Use `db:"column_name,pk"` to mark partition/clustering keys
3. **Column Types**:
   - Use `db2.Col[TableStruct, Type]` for single values
   - Use `db2.ColSlice[TableStruct, Type]` for slices (stored as sets)
4. **Both structs must have matching field names** to map columns correctly

## Connection Setup

```go
db2.MakeScyllaConnection(db2.ConnParams{
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
err := db2.Insert(&records)

// Insert excluding specific columns
err := db2.Insert(&records, q1.UpdatedBy)

// Insert single record
err := db2.InsertOne(records[0])
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
q1 := db2.Table[types.ListaCompartidaRegistro]()

// Update specific columns only
err := db2.Update(&[]types.ListaCompartidaRegistro{recordToUpdate},
    q1.Status, q1.ListaID, q1.Nombre, q1.Images, q1.Updated)

// Update single record
err := db2.UpdateOne(recordToUpdate, q1.Nombre, q1.Status)

// Update all fields except specified
err := db2.UpdateExclude(&records, q1.UpdatedBy, q1.Created)
```

### Select/Query

```go
registros := []types.ListaCompartidaRegistro{}

// Build and execute query
query := db2.Query(&registros)
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
db2.TableSchema{
    Name:      "table_name",
    Partition: e.CompanyID,           // Partition key
    Keys:      []db2.Coln{e.ID},      // Clustering keys
}
```

### Views

Create materialized views for alternative query patterns:

```go
Views: []db2.View{
    // Simple view with partition key
    {Cols: []db2.Coln{e.ListaID, e.Status}, KeepPart: true},
    
    // Concatenated integer view (for range queries)
    {Cols: []db2.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
    {Cols: []db2.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
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
- **Slices**: Stored as ScyllaDB sets (use `db2.ColSlice`)
- **Complex types**: Structs automatically serialized to CBOR and stored as blobs
