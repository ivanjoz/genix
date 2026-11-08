---
name: create-database-tables
description: Define new tables with backend ORM — paired `XRecord`/`XRecordTable` structs. `GetSchema()`. Use when creating or editing columns to tables.`.
version: 0.1.0
---

# Create Database Tables

Every table in the genix backend is **two paired Go structs** (`XRecord` + `XRecordTable`) plus an optional `SelfParse()` and a required `GetSchema()`. After writing the file, run the `static-project-validation` skill to catch struct mismatches.

A scaffolding script at `scripts/table/create_edit_table.go` generates boilerplate, but the schema (keys, packing, indexes) is almost always hand-edited because it requires judgment.

---

## 1. The two paired structs

| Struct          | Purpose                                                              |
| --------------- | -------------------------------------------------------------------- |
| `XRecord`       | Plain data — what gets serialized to JSON and stored in a row.       |
| `XRecordTable`  | Column descriptors used by the query builder — `db.Col[Table, T]`.   |

Both embed `db.TableStruct[XRecordTable, XRecord]` so the ORM can resolve them generically.

```go
type Product struct {
    db.TableStruct[ProductTable, Product]
    EmpresaID int32  `json:",omitempty"`
    ID        int32  `json:",omitempty"`
    Nombre    string `json:",omitempty"`
    Status    int8   `json:"ss,omitempty"`
    Updated   int32  `json:"upd,omitempty"`
    UpdatedBy int32  `json:",omitempty"`
}

type ProductTable struct {
    db.TableStruct[ProductTable, Product]
    EmpresaID db.Col[ProductTable, int32]
    ID        db.Col[ProductTable, int32]
    Nombre    db.Col[ProductTable, string]
    Status    db.Col[ProductTable, int8]
    Updated   db.Col[ProductTable, int32]
    UpdatedBy db.Col[ProductTable, int32]
}
```

**Rules:**
- Field names in `*Table` must match the data struct exactly.
- `int32` becomes `db.Col[XTable, int32]`.
- **Slices of primitives** (`[]int32`, `[]string`) must use `db.ColSlice[XTable, ElemType]`. Slices of structs use `db.Col[XTable, []Struct]`. The static validator enforces this.

---

## 2. Conventional fields

| Field       | Type    | JSON tag        | Purpose                                                          |
| ----------- | ------- | --------------- | ---------------------------------------------------------------- |
| `EmpresaID` | `int32` | `,omitempty`    | Tenant partition. Older tables use `CompanyID` instead.          |
| `Status`    | `int8`  | `ss,omitempty`  | `1+` active, `0` deleted. Drives delta-cache evictions.          |
| `Updated`   | `int32` | `upd,omitempty` | Unix timestamp of last write. Drives delta-cache watermarks.     |
| `UpdatedBy` | `int32` | `,omitempty`    | User ID of last writer.                                          |
| `Created`   | `int32` | `,omitempty`    | Optional — Unix timestamp of insert.                             |
| `CreatedBy` | `int32` | `,omitempty`    | Optional — user ID of inserter.                                  |

**JSON tags** are kept short to save bandwidth: `upd`, `ss`, `ccv` (for `CacheVersion`), otherwise plain `json:",omitempty"`.

**db tags** are usually omitted — the ORM derives the column name from the field in snake_case. Add `db:"..."` only when the column name diverges (e.g. `db:"pais_ciudad_id"`) or needs a marker (`,pk` for explicit composite-key parts; `,view` for columns referenced by views).

---

## 3. SelfParse()

`SelfParse()` runs on every record **before insert and before update**. Use it for derived fields: status from quantities, hashes for fast lookup, computed concatenated keys. Pointer receiver, no args. Optional — omit if no derived fields.

```go
// Status from quantity — backend/logistica/types/product-stock.go:33
func (e *ProductStock) SelfParse() {
    e.Status = core.If(e.Quantity == 0 && e.SubQuantity == 0, int8(0), int8(1))
}

// Hash for name lookup — backend/negocio/types/productos.go:118
func (e *Product) SelfParse() {
    e.NombreHash = core.BasicHashInt(core.NormalizeString(&e.Nombre))
}

// Build a KeyConcatenated string — backend/logistica/types/product-stock.go:246
func (e *ProductStockLot) SelfParse() {
    e.Hash = db.MakeKeyConcat(e.Date, e.SupplierID, e.Name)
}
```

---

## 4. GetSchema()

Value receiver on the `*Table` struct returning a `db.TableSchema`. The fields you actually use:

```go
type TableSchema struct {
    Name                 string  // table name in the DB (snake_case)
    Partition            Coln    // partition column (almost always EmpresaID)
    Keys                 []Coln  // clustering columns; with Partition form the PK
    KeyConcatenated      []Coln  // virtual lookup key built by joining columns into a string
    KeyIntPacking        []Coln  // pack multiple ints into the single int64 ID column
    AutoincrementPart    Coln    // partition column for the autoincrement counter
    Indexes              []Index // secondary indexes
    SaveCacheVersion     bool    // maintain a per-row CacheVersion (for the delta cache)
    DisableUpdateCounter bool    // append-only tables that should not maintain UpdateCounter
}
```

Minimum schema:
```go
func (e ProductTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "products",
        Partition: e.EmpresaID,
        Keys:      []db.Coln{e.ID.Autoincrement(0)},
    }
}
```

---

## 5. ID / key patterns

Pick the pattern that matches your access shape.

### 5.1 Single autoincrement int ID (most common)

```go
Keys: []db.Coln{e.ID.Autoincrement(0)},
```

`Autoincrement(N)` adds an `N`-digit random suffix. `0` = clean sequence (1, 2, 3…). Use `Autoincrement(2)` etc. on highly concurrent writers to avoid collisions.

### 5.2 Composite clustering keys

Multiple `Keys` form a clustering composite. Order matters — it's the on-disk sort order and the order required for range queries.

```go
// backend/logistica/types/product-stock.go:215
Keys: []db.Coln{e.ProductStockID, e.LotID, e.SerialNumber},
```

### 5.3 KeyIntPacking — pack multiple ints into one int64 ID

When natural identity is `(WarehouseID, ProductID, PresentationID)` but you want a single `int64 ID`, pack the components into the ID using `Size(bits)` slots. `Size(n)` is a range cap (values below 2^n; a value past it panics on write, it is never truncated). Every slot but the last needs a `Size`; the total must fit the 63 bits of a positive int64.

```go
// backend/logistics/types/product-stock.go
Keys: []db.Coln{e.ID},
KeyIntPacking: []db.Coln{
    e.WarehouseID.Size(17),     // 0..131,071
    e.ProductID.Size(30),       // 0..1,073,741,823
    e.PresentationID.Size(14),  // 0..16,383
},
```

`db.DecodePackedKey[RecordT](id)` splits an ID back into its components when debugging.

Packed components are recoverable — the ID can be queried by any prefix via `TypeCacheGroupFromKey` indexes (§6).

### 5.4 Append-only ledgers — KeyIntPacking + AutoincrementPart

Movement/ledger tables pack a date or bucket column with an autoincrement *within that bucket*. `AutoincrementPart` tells the ORM which column the counter is partitioned by.

```go
// backend/logistica/types/product-stock-movement.go:60
Keys: []db.Coln{e.ID},
KeyIntPacking: []db.Coln{
    e.Fecha.Size(15),                  // a UnixDay
    e.WarehouseID.Size(17),
    e.Autoincrement(8),                // counter<<8 | 8 random bits, takes the remaining bits
},
AutoincrementPart: e.Fecha,            // counter scoped per Fecha
```

### 5.5 KeyConcatenated — virtual string lookup key

When you want to look up rows by a tuple **without** packing it into the ID, declare `KeyConcatenated`. The ORM joins the values into a deterministic string for dedup lookups; you typically also store that string in a field (built by `SelfParse()` via `db.MakeKeyConcat(...)`).

```go
// backend/logistica/types/product-stock.go:71
Keys:            []db.Coln{e.ID},
KeyConcatenated: []db.Coln{e.WarehouseID, e.ProductID, e.PresentationID, e.SKU, e.Lote},
```

### 5.6 String IDs

Some tables use a string ID generated upstream (UUID, slug, external code). Declare the column as `string` and use it directly — no autoincrement.

```go
ID db.Col[XTable, string]
// ...
Keys: []db.Coln{e.ID},
```

### 5.7 No ID at all (partition + clustering only)

Summary tables sometimes use only the partition + a clustering column as the PK:

```go
// backend/comercial/types/sales.go:171
Partition: e.EmpresaID,
Keys:      []db.Coln{e.Fecha},  // (EmpresaID, Fecha) is the PK
```

---

## 6. Indexes

Each `db.Index` has:

```go
type Index struct {
    Type      int8   // see table below
    Keys      []Coln // index key columns (order matters)
    Cols      []Coln // (TypeView only) extra payload columns; empty = SELECT *
    Partition Coln   // overrides the index's partition column; empty keeps the table's
    UseHash   bool   // build a hash for IN-operator support
    DeltaReplaceThreshold int // (TypeDelta) records a sync answers before it answers a replace
}
```

| Constant             | Use when                                                                           |
| -------------------- | ---------------------------------------------------------------------------------- |
| `TypeLocalIndex`     | Look up by a non-key column **within a partition** (cheap; per-partition).         |
| `TypeGlobalIndex`    | Look up by a non-key column **across all partitions** (expensive; sparingly).      |
| `TypeView`           | Composite index with explicit key reordering (on MariaDB a plain secondary index).  |
| `TypeCacheGroupFromKey` | Cache group over a packed-key prefix. Free — a range of the primary key, no index.  |
| `TypeViewTable`      | Separate physical table acting as a view (rare).                                   |
| `TypeUniqueIndex`    | A natural key unique **within a partition** (MariaDB/PostgreSQL only). Use instead of a hash column + lookup. |

`TypeUniqueIndex` rules: create records with `db.Insert`/`db.InsertIn` (or an autoincrement `Upsert`);
an Update, or an Upsert of given keys, may not write its Keys (refused). Inside `db.RunTransaction`
a duplicate reruns the body, so a "find or create" body finds the winner's record on the rerun.

```go
// One lot per (Date, SupplierID, Name) within the company
{Type: db.TypeUniqueIndex, Keys: db.Cols(e.Date, e.SupplierID, e.Name)},
```

### Common patterns

```go
// Lookup by SKU / hash within a company
{Type: db.TypeLocalIndex, Keys: []db.Coln{e.SKU}},

// Delta-cache view (status + updated). The index keeps the table partition (EmpresaID).
{
    Type:     db.TypeView,
    Keys:     []db.Coln{e.Status, e.Updated.Size(31)},
},

// View with column projection — only carry these payload columns
{
    Type:     db.TypeView,
    Keys:     []db.Coln{e.IsWarehouseProductStatus, e.Updated.Size(31)},
    Cols:     []db.Coln{e.WarehouseProductQuantity, e.PresentationID},
},

// Range scans over packed-key prefixes — free, no index
{Type: db.TypeCacheGroupFromKey, Keys: []db.Coln{e.Fecha}},
{Type: db.TypeCacheGroupFromKey, Keys: []db.Coln{e.Fecha, e.WarehouseID}},
```

### Column packing methods (used inside `Keys`)

| Method                       | Effect                                                                  |
| ---------------------------- | ----------------------------------------------------------------------- |
| `.Size(N)`                   | Allocate an `N`-bit slot inside a packed key (values below 2^N).        |
| `.Int32()`                   | Use 32-bit packing instead of 64-bit (smaller keys).                    |
| `.StoreAsWeek()`             | Convert a date int16 to its week bucket inside the key.                 |
| `.IsWeek()`                  | Mark a column as already storing a week value.                          |
| `.CompositeBucketing(a, b…)` | Virtual bucketing across slice columns.                                 |
| `.Autoincrement(N)`          | Inside `KeyIntPacking`, the counter followed by `N` random low bits.    |

**Conventions:** the first key of a packed view takes the bits the others leave, so it carries no `Size`. `Updated.Size(31)` (an int32 SUnixTime), a UnixDay `Size(15)`.

---

## 7. Worked examples

### 7.1 Standard entity (autoincrement, EmpresaID partition, delta-cache view)

```go
type Almacen struct {
    db.TableStruct[AlmacenTable, Almacen]
    EmpresaID int32  `json:",omitempty"`
    ID        int32  `json:",omitempty"`
    Nombre    string `json:",omitempty"`
    Status    int8   `json:"ss,omitempty"`
    Updated   int32  `json:"upd,omitempty"`
    UpdatedBy int32  `json:",omitempty"`
}

type AlmacenTable struct {
    db.TableStruct[AlmacenTable, Almacen]
    EmpresaID db.Col[AlmacenTable, int32]
    ID        db.Col[AlmacenTable, int32]
    Nombre    db.Col[AlmacenTable, string]
    Status    db.Col[AlmacenTable, int8]
    Updated   db.Col[AlmacenTable, int32]
    UpdatedBy db.Col[AlmacenTable, int32]
}

func (e AlmacenTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "almacenes",
        Partition: e.EmpresaID,
        Keys:      []db.Coln{e.ID.Autoincrement(0)},
        Indexes: []db.Index{
            {Type: db.TypeView, Keys: []db.Coln{e.Status, e.Updated.Size(31)}},
        },
    }
}
```

### 7.2 Stock row with packed identity

```go
func (e *ProductStockV2) SelfParse() {
    e.Status = core.If(e.Quantity == 0 && e.DetailQuantity == 0, int8(0), int8(1))
}

func (e ProductStockV2Table) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:                 "warehouse_product_stock",
        Partition:            e.CompanyID,
        Keys:                 []db.Coln{e.ID},
        DisableUpdateCounter: true,
        KeyIntPacking: []db.Coln{
            e.WarehouseID.Size(17),
            e.ProductID.Size(30),
            e.PresentationID.Size(14),
        },
        Indexes: []db.Index{
            {
                Type:     db.TypeView,
                Keys:     []db.Coln{e.WarehouseID, e.Status.Size(4), e.Updated.Size(31)},
            },
        },
    }
}
```

### 7.3 Append-only ledger — packed key + autoincrement-per-bucket

```go
func (e WarehouseProductMovementTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:      "warehouse_product_movement",
        Partition: e.CompanyID,
        Keys:      []db.Coln{e.ID},
        KeyIntPacking: []db.Coln{
            e.Fecha.Size(15),
            e.WarehouseID.Size(17),
            e.Autoincrement(8),
        },
        AutoincrementPart: e.Fecha,
        Indexes: []db.Index{
            {Type: db.TypeCacheGroupFromKey, Keys: []db.Coln{e.Fecha}},
            {Type: db.TypeCacheGroupFromKey, Keys: []db.Coln{e.Fecha, e.WarehouseID}},
            {Type: db.TypeLocalIndex, Keys: []db.Coln{e.SerialNumber}},
        },
    }
}
```

### 7.4 Legacy tuple-keyed table — KeyConcatenated for dedup lookup

```go
func (e ProductStockTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        Name:            "warehouse_product_stock_legacy",
        Partition:       e.CompanyID,
        Keys:            []db.Coln{e.ID},
        KeyConcatenated: []db.Coln{e.WarehouseID, e.ProductID, e.PresentationID, e.SKU, e.Lote},
        Indexes: []db.Index{
            {Type: db.TypeLocalIndex, Keys: []db.Coln{e.SKU}},
            {Type: db.TypeLocalIndex, Keys: []db.Coln{e.Lote}},
        },
    }
}
```

---

## 8. Scaffolding script

`scripts/table/create_edit_table.go` generates the boilerplate. Use it for the initial file; then hand-edit `GetSchema()` for the real keys/indexes.

### Create

```bash
cd scripts/table
go run create_edit_table.go create \
    ../../backend/logistica/types/warehouse_zone.go \
    warehouse_zone \
    nombre:string code:string:key
```

- `<output_path>` — where the file is written.
- `<table_name>` — snake_case; becomes the SQL `Name` and the camelCase Go type.
- `[field:type[:key]]…` — initial fields. Append `:key` to mark a clustering key. Slice types use `[]string` etc.

The script generates both structs with audit fields pre-added and wires `Partition: e.EmpresaID` plus `Keys` from any `:key` fields. It does **not** generate `SelfParse`, `KeyIntPacking`, `KeyConcatenated`, indexes, or autoincrement — add those by hand.

### Edit (add a column)

```bash
cd scripts/table
go run create_edit_table.go edit warehouse_zone descripcion:string
go run create_edit_table.go edit warehouse_zone tags:[]string
go run create_edit_table.go edit warehouse_zone parent_id:int32:key   # also appended to Keys
```

The edit command finds the file containing `type <CamelName> struct`, appends the field to both structs, and (if `:key`) appends the column to `GetSchema().Keys`.

---

## 9. After writing the table

1. Run **`static-project-validation`** (`cd scripts && go run . check_tables`) to catch field mismatches between the two structs and wrong `Col`/`ColSlice` usage.
2. Add **`SelfParse()`** if any field is derived from others (status from quantity, hash from name, KeyConcatenated string from tuple).
3. For any field the frontend filters on, add a `TypeLocalIndex`, a `TypeView` (with `Updated.Size(31)` for delta cache), or — if the field is part of a packed key — a free `TypeCacheGroupFromKey` with `UseIndexGroup: true`.
4. If the table is read by the frontend on every load, pair it with a handler following the **`delta-cache-api`** skill.
