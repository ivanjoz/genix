# Genix ORM Internals: Architectural Deep Dive

This document provides an exhaustive technical analysis of the Genix ORM, a high-performance database abstraction layer specifically engineered for ScyllaDB. It prioritizes low-latency execution, efficient memory management, and advanced query optimization.

---

## 1. Memory Model and Reflection Engine

At its core, the Genix ORM is designed to eliminate the performance bottlenecks typically associated with Go's standard `reflect` package during high-volume data operations.

### 1.1 The `xunsafe` Integration
Instead of using `reflect.Value.Set()` or `reflect.Value.Interface()` during row scanning—which involves significant heap allocations and interface boxing—the ORM utilizes the `viant/xunsafe` library.
- **Field Offsets**: During the one-time table initialization (`initStructTable`), the ORM calculates the `uintptr` offset of every field in the destination struct.
- **Direct Pointer Access**: During `Scan()`, the ORM calculates the memory address of a field by adding its offset to the struct's base pointer.
- **Zero-Allocation Mapping**: Values are written directly to memory locations, bypassing the reflection layer entirely for primitive types.

### 1.2 Type Mapping Matrix
The ORM maintains a strict mapping between Go types and CQL (Cassandra Query Language) types, defined in `converter.go`:

| Go Type | CQL Type | Internal ID | Notes |
| :--- | :--- | :--- | :--- |
| `string` | `text` | 1 | |
| `int64` | `bigint` | 2 | |
| `int32` | `int` | 3 | |
| `[]byte` | `blob` | 9 | Also used for CBOR complex types |
| `[]string` | `set<text>` | 11 | |
| `*int64` | `bigint` | 22 | Handles NULL values |
| `struct` | `blob` | 9 | Serialized via CBOR |

---

## 2. The Query Builder State Machine

The query interface uses a fluent API that populates a `TableInfo` structure. This structure acts as a state machine for the upcoming execution.

### 2.1 Statement Tracking
Every method call (`Equals`, `GreaterEqual`, `In`, `Between`) appends a `ColumnStatement` to the `statements` slice. 
- **`Equals(v)`**: Sets `Operator: "="`.
- **`In(...v)`**: Populates the `Values` slice and sets `Operator: "IN"`.
- **`Between(v1, v2)`**: Creates a composite statement with `From` and `To` sub-statements.

### 2.2 Select and Exclude
The ORM supports partial fetching to reduce network I/O.
- **`Select(cols...)`**: Populates `columnsInclude`. Only these columns are added to the `SELECT` clause.
- **`Exclude(cols...)`**: Populates `columnsExclude`. The ORM calculates the complement set from the table's full column list.

---

## 3. The Optimization Engine: Capability Matching

This is the most critical part of the ORM, allowing it to bypass ScyllaDB's `ALLOW FILTERING` requirement by intelligently routing queries.

### 3.1 Signature Generation
At table initialization, the ORM calls `ComputeCapabilities()`, which generates a list of "Query Signatures". A signature is a string representing a valid query path.
- **Format**: `column_name|operator|column_name|operator...`
- **Example**: An index on `(EmpresaID, SKU)` generates `empresa_id|=|sku|=`.

### 3.2 The Matching Algorithm (`MatchQueryCapability`)
When `Exec()` is called:
1. The ORM builds a map of columns and operators present in the current query.
2. It iterates through all `QueryCapability` objects.
3. A match is found if the query contains every column in the signature with the required operator.
4. **Priority Scoring**:
    - Exact equality matches on Materialized Views score highest (~30-40 points).
    - Range queries on clustering keys score lower (~15-20 points).
    - Primary Key lookups are prioritized if no specialized view covers more columns.

---

## 4. Virtual Views and Indexing Strategies

The ORM implements "Virtual" indexes that don't exist natively in ScyllaDB but are simulated through Materialized Views and calculated columns.

### 4.1 Hash Views (Types 3 & 7)
Used for equality across multiple columns without a defined hierarchy.
- **Initialization**: Creates a virtual column `zz_column1_column2_index`.
- **Runtime**: `HashInt(val1, val2)` is called to generate a 32-bit FNV hash.
- **CQL**: `SELECT * FROM table_view WHERE zz_hash = ?`.

### 4.2 Range / Radix Views (Type 8)
This solves the problem of range-querying multiple numeric columns simultaneously.
- **Mathematical Formula**: `CombinedValue = Sum(Value[i] * 10^Radix[i])`.
- **Example**: To query `AlmacenID` (max 100,000) and `Updated` (Unix timestamp):
    - `Radix[Updated] = 0`
    - `Radix[AlmacenID] = 10`
    - `Combined = (AlmacenID * 10,000,000,000) + Updated`.
- **Range Logic**: A range query on `Updated` becomes a range query on the combined column while keeping the `AlmacenID` prefix constant.

### 4.3 Packed Indexes (Local vs Global)
Packed indexes are a numeric "digit packing" strategy that creates a stored virtual integer column (computed on write) and indexes it.

They are declared via `TableSchema`:
- `Indexes: [][]db.Coln` creates a **local** secondary index on `((partition), packed_col)`.
- `GlobalIndexes: [][]db.Coln` creates:
  - simple **global** indexes for single-column entries, and
  - **packed global** indexes for composite entries (2+ columns).

#### 4.3.1 Packing Rules
Given source columns `[c1, c2, ..., cn]`:
- `c1` (first) MUST NOT set `DecimalSize()`; its digit width is derived from remaining budget.
- `c2..cn` MUST set `DecimalSize(n)`; these define fixed digit widths.
- All components must be non-negative; negative values are programmer errors and trigger a panic.
- Truncation rule: if a value has more digits than its slot, the ORM drops least-significant digits (right-trim) to fit.

`.Int32()` is a schema hint meaning "store the packed column as `int32`":
- The packed value is computed as `int64` first.
- The final stored value is trimmed to 9 digits and then cast to `int32`.
- This can overfetch on reads, so the query engine applies a post-filter to guarantee exact semantics.

#### 4.3.2 Query Routing and Safety
Packed **local** indexes are intended for the pattern:
- partition equality + prefix equality/IN + range on the last column

Packed **global** indexes are intended for:
- equality lookups (and small fan-out with `IN` on the first column)

Important: global secondary indexes are not reliable for general range scans in ScyllaDB.
To avoid accidental `ALLOW FILTERING` behavior, the ORM does not advertise range (`~`) capabilities for packed global indexes.

---

## 5. Smart Primary Keys (KeyConcatenated & KeyIntPacking)

For tables where the primary key is a composite value, the ORM provides "Smart Key" logic to flatten multiple fields into a single column.

### 5.1 KeyConcatenated (Base62)
Fields are concatenated using a `_` separator. Integers are converted to **Base62** (`0-9a-zA-Z`) to ensure the key is compact and URL-safe.
- `Concat62(100, "abc")` -> `"1C_abc"`

### 5.2 KeyIntPacking (Mathematical Concatenation)
This mechanism packs multiple numeric fields into a single `int64` (bigint). It allocates slots within a 19-digit space (the limit of a signed 64-bit integer).
- **Width Control**: Each component's width is defined by `DecimalSize(n)`.
- **Calculation**: `Sum(Value[i] * 10^RemainingDigits)`.
- **Placeholder Support**: Allows using an `Autoincrement` value as one of the packing components.

---

## 6. Automated Autoincrement and Sequences

The ORM automates the retrieval of unique IDs from a central `sequences` counter table before execution.

### 6.1 Partitioned Counters
By defining `AutoincrementPart`, the ORM uses a specific column's value to partition the counter.
- **Counter Key**: `table_name + "_" + partition_value`.
- **Efficiency**: The ORM groups records by partition and performs a single `GetCounter(N)` call for all records in the batch.

### 6.2 Concurrent Collision Avoidance
The `Autoincrement(randSize)` method can append a random numeric suffix to the retrieved counter value.
- **Formula**: `CounterValue * 10^randSize + Random(10^randSize)`.
- This ensures that even if two processes retrieve the same counter value (e.g., during manual resets or extreme concurrency), the final IDs remain unique.

---

## 7. Write Operations and Consistency

### 6.1 Batching Strategy
The `Insert` function (`insert-update.go`) uses `gocql.UnloggedBatch`.
- **Efficiency**: Reduces the number of round-trips to the cluster.
- **Statement Preparation**: Statements are prepared once and bound with values from the struct pointers using `xunsafe`.

### 6.2 The Virtual Column Update Constraint
If a view depends on a virtual column (like a hash or radix), updating any of the source columns *must* also trigger an update of the virtual column.
- **Internal Check**: The `Update` logic identifies dependencies and panics if the user attempts a partial update that would leave a virtual index in an inconsistent state.

---

## 7. Schema Evolution and Deployment

The `deploy.go` logic ensures that the application and database stay in sync.

### 7.1 Automated Homologation
- **Metadata Fetching**: Queries `system_schema.columns` and `system_schema.indexes`.
- **Diffing**: Compares the live schema with the `TableSchema` defined in Go code.
- **Auto-Correction**: Executes `ALTER TABLE ADD` for new fields.
- **View Management**: Automatically generates the complex `CREATE MATERIALIZED VIEW` statements, including the required `IS NOT NULL` clauses for all primary key components.

---

## 8. Data Serialization: CBOR Integration

For complex Go structures (slices of structs, maps, etc.) that don't map to CQL types:
- **Encoding**: Uses `fxamacker/cbor` for binary serialization.
- **Storage**: Saved as a `blob` in ScyllaDB.
- **Decoding**: During `Scan`, the ORM detects the blob type and performs an in-place unmarshal into the struct field memory.

---

## 9. Parallel Query Execution

For queries that result in multiple `WHERE` statements (e.g., an `IN` operator on a partition key or a complex view statement):
- **ErrGroup**: Uses `golang.org/x/sync/errgroup` to spawn multiple goroutines.
- **Concurrency**: Each sub-query runs in parallel, and results are merged into the final result slice.
- **Resource Management**: Automatically handles reconnections if a host becomes unavailable during a parallel scan.

---

## 10. Summary of Performance Features
1. **Zero-Reflection Scanning**: Via `xunsafe` pointer arithmetic.
2. **Signature Matching**: $O(N)$ source selection where $N$ is the number of indices.
3. **Radix Range Clustering**: Allowing range queries on multi-column views.
4. **Base62 Key Compression**: Reducing the size of string-based primary keys.
5. **KeyIntPacking**: Mathematical concatenation of primary key components into single `int64`.
6. **Automated Partitioned Counters**: Zero-effort unique ID generation with collision avoidance.
7. **CBOR Serialization**: Faster and more compact than JSON for complex data.
8. **Packed Indexes (Local/Global)**: Digit-packing strategy for composite predicates via secondary indexes, with post-filtering when truncation is used.
