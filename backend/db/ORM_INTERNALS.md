# Genix ORM Internals: Current Architecture

This document describes the current internal architecture of the Genix ORM in `backend/db`, focused on performance-critical execution paths for ScyllaDB.

---

## 1. Core Runtime Model

The ORM separates immutable metadata from per-query state.

- **Immutable metadata**: schema-derived table/column/index/view structures, column type mappings, and compiled accessors.
- **Per-query state**: `TableInfo` (`WHERE` statements, selected/excluded columns, order, limit, ref slice).

This separation keeps setup overhead low while preserving the fluent API.

### 1.1 Main Types

- **`ScyllaTable`** (`main.go`): table runtime descriptor (keys, partition, columns, maps, views/indexes, capabilities, cache-version metadata).
- **`columnInfo` / `colInfo`** (`reflect_accessors.go`): column runtime metadata + getter/setter function pointers.
- **`TableInfo`** (`main.go`): mutable query builder state.
- **`ColumnStatement`** (`main.go`): normalized predicate unit used by planner and query execution.

---

## 2. Metadata Caching and Initialization

### 2.1 Struct Field Metadata Cache

`table_cache.go` provides `structFieldMetadataCache` keyed by record `reflect.Type`.

Cached items include:
- field name/index/type
- `xunsafe.Field` pointer for direct memory access
- inferred ORM type (`colType`)
- cache-version response field index

This avoids rebuilding reflection metadata for each call to `initStructTable`.

### 2.2 Table Compilation Cache

`table_cache.go` provides `scyllaTableCache` keyed by schema type (`pkgpath.TypeName`) and guarded by `sync.Once`.

Each cache entry stores one compiled `ScyllaTable`:
- all columns and maps
- generated virtual columns
- configured indexes/views
- computed query capabilities
- precompiled accessors

### 2.3 `initStructTable` Binding Flow

`initStructTable` (`main.go`) now:
1. fetches cached field metadata,
2. binds each fluent column field to schema/table info,
3. copies immutable metadata into per-column runtime state,
4. assigns per-call `TableInfo`.

The function still validates schema struct shape and preserves existing fluent behavior.

---

## 3. Memory Access and Reflection Elimination

### 3.1 `xunsafe` Field Access

Instead of `reflect.Value.Set` / `Interface` in hot loops, the ORM uses `xunsafe.Field` typed methods and pointer arithmetic.

Benefits:
- lower allocation pressure
- reduced interface boxing
- predictable scalar assignment costs

### 3.2 Column Accessor Compilation

`compileFastAccessors` (`reflect_accessors.go`) configures function pointers once per column.

Compiled categories:
- scalar: `string`, `int*`, `float*`, `bool`
- hot slices: `[]string`, `[]int64`, `[]int32`, `[]int16`, `[]int8`
- pointer scalars: `*string`, `*int*`, `*float*`
- pointer slices: `*[]string`, `*[]int64`, `*[]int32`, `*[]int16`, `*[]int8`

### 3.3 Setter Semantics

For hot slice and pointer-slice types, fast setters are exact-type only:
- accepted: `[]T` or `*[]T` for the target type
- fallback: generic converter path for non-exact inputs

This keeps fast paths simple while preserving compatibility for edge inputs.

---

## 4. Type System and Conversion Layer

`converter.go` defines the type matrix used by schema inference and runtime assignment.

### 4.1 Internal Type IDs

Representative mappings:
- `1`: `string` <-> `text`
- `2`: `int64` <-> `bigint`
- `3`: `int32` <-> `int`
- `9`: `[]byte` / complex blob
- `11`: `[]string` <-> `set<text>` (default; can be overridden with db tag flags)
- `22`: `*int64` <-> nullable bigint

### 4.2 Conversion Paths

- **Fast path**: precompiled accessors in `columnInfo`.
- **Fallback path**: `assingValue` for uncommon or mismatched assignments.
- **Statement serialization**: `GetValue` / `GetStatementValue` use either compiled functions or `makeScyllaValue`.

### 4.3 Unsigned Blob Compatibility

Unsigned primitive/slice compatibility is handled via:
- `encodeUnsignedValueToBlob`
- `decodeUnsignedValueFromBlob`

This supports backward-compatible blob encoding for unsupported CQL unsigned native types.

### 4.4 CBOR for Complex Types

Complex fields that do not map directly to CQL types are persisted as blob using `fxamacker/cbor`.

- write: marshal to bytes
- read: in-place unmarshal into field memory

---

## 5. Fallback Telemetry and Diagnostics

`converter.go` tracks fallback usage by `colType`.

Available APIs:
- `GetAssignFallbackUsageByType()`
- `ResetAssignFallbackUsageByType()`

Behavior:
- every `assingValue` call increments per-type atomic counters
- optional first-hit debug log per type when full logging is enabled

Purpose:
- identify hotspots still missing fast accessors
- guide incremental converter simplification safely

---

## 6. Query Builder State Machine

Fluent query methods append statements to `TableInfo`.

- `Equals(v)` -> `Operator: "="`
- `In(...v)` -> `Operator: "IN"`, `Values` populated
- `Between(v1,v2)` -> `Operator: "BETWEEN"`, `From`/`To` statements

Additional state:
- include/exclude projections
- order and limit
- allow-filter flag

`TableInfo` is ephemeral and never cached globally.

---

## 7. Capability-Based Query Routing

### 7.1 Signature Generation

At table compile time, `ComputeCapabilities()` creates normalized signatures:
- format: `column|operator|column|operator...`

Each signature ties a predicate pattern to a source:
- base table keys
- index
- materialized/virtual view

### 7.2 Matching Strategy

`MatchQueryCapability` selects the best source by:
1. checking required predicates/operators,
2. scoring candidates by specificity,
3. preferring exact/high-selectivity paths.

This is the primary mechanism used to avoid accidental `ALLOW FILTERING` queries.

---

## 8. Indexes, Views, and Virtual Columns

### 8.1 Local and Global Secondary Indexes

`TableSchema` supports:
- `Indexes []Index`

Inference rules:
- one key, no explicit type: local secondary index
- two or more numeric keys, no explicit type: packed local index
- `Type: TypeGlobalIndex`: global secondary index
- `Cols != nil`: materialized view payload declaration
- `Type: TypeViewTable`: derived table with write-side maintenance
- `UseIndexGroup: true`: write-maintained hash group metadata

### 8.2 Packed Indexes

Packed indexes concatenate numeric components into one sortable number.

Rules:
- first component width is inferred
- trailing components require `DecimalSize`
- values exceeding slot width are truncated by rule
- `.Int32()` allows packed value storage in `int32` with post-filter exactness when needed

### 8.3 Views

- **Hash views**: equality/IN routing via computed hash columns.
- **Range (radix) views**: multi-column range routing via radix-weighted composite values.

### 8.4 Composite Bucketing

For hash indexes with range constraints:
- bucket IDs are generated for one designated numeric column
- tuple hash sets are materialized into virtual set columns
- select planner chooses bucket coverage and emits indexed statements
- post-filter is applied to guarantee exact semantics after controlled overfetch

---

## 9. Primary Key Compression Strategies

### 9.1 KeyConcatenated

Multiple fields are flattened into one string key using Base62-compatible concatenation.

### 9.2 KeyIntPacking

Multiple numeric fields are packed into one `int64` key with decimal slot sizing.

Supports autoincrement placeholders as packing components.

---

## 10. Write Pipeline

### 10.1 Insert and Update Execution

- inserts use `gocql.UnloggedBatch`
- updates generate deterministic `SET` and `WHERE` clauses
- struct values are extracted with column accessors

### 10.2 Pre-Insert Autoincrement

`handlePreInsert`:
- groups rows by partition/autoincrement partition
- requests counters in bulk
- fills autoincrement field (optional random suffix)
- applies key packing when configured

### 10.3 Virtual Consistency Checks

Update path enforces dependency integrity:
- if a virtual index/view depends on source columns,
- partial updates that would break derived values panic early.

---

## 11. Cache-Version Integration

Cache-version support is precomputed during table build.

Stored runtime metadata includes:
- cache-version response field index
- partition column used for group key
- key column used for group/version mapping

Read/write flow:
- write operations increment version groups after successful commit
- select paths ensure required ID/partition columns are available for version assignment

---

## 12. Schema Deployment and Homologation

`deploy.go` compares declared schema with live Scylla metadata.

Capabilities:
- add missing table columns
- create/manage indexes and materialized views
- generate required `IS NOT NULL` clauses for view key parts

---

## 13. Parallel Query Execution

When a logical query fans out into multiple statements (e.g. IN-based expansions):
- execution uses `errgroup`
- subqueries run concurrently
- result sets merge into final output
- reconnection logic retries on transient no-host conditions

---

## 14. Current Performance Profile Summary

1. `xunsafe` typed access avoids reflection overhead in hot paths.
2. Struct metadata cache removes repeated reflection setup costs.
3. Table compile cache removes repeated index/view/capability construction.
4. Compiled accessors cover high-frequency scalar/slice/pointer types.
5. Capability matching routes queries to index/view-aware plans.
6. Packed/hash/radix strategies support complex predicate patterns.
7. Post-filtering enforces exactness when optimized plans overfetch.
8. Fallback telemetry provides measurable guidance for next optimizations.
