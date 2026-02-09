# ORM Plan: Packed Int Global Secondary Indexes via `TableSchema.GlobalIndexes`

## Objective
Add support for composite “packed int” **global** secondary indexes (GSI) declared as:

```go
GlobalIndexes: [][]db.Coln{
  {e.Status.Int32(), e.Updated.DecimalSize(8)},
},
```

This should:
- Create a stored virtual packed column (ex: `zz_gixp_status_updated`).
- Create a **global** secondary index: `CREATE INDEX ... ON table (zz_gixp_...)`.
- Rewrite queries expressed on source columns (`Status`, `Updated`, ...) into predicates on the packed column.
- Support:
  - prefix equality / IN on all columns except the last
  - range/equality on the last column (`BETWEEN`, `>`, `>=`, `<`, `<=`, `=`)
- Apply a post-filter when truncation can overfetch results.

Notes:
- This is parallel to the already-implemented packed **local** index feature (`TableSchema.Indexes` => `CREATE INDEX ... ((pk), packed)`).
- Backfill is explicitly not required: existing rows won’t be discoverable via this index until they are inserted/updated (same as local packed index).

## Current Status (as of today in repo)
- `backend/db/main.go`: `TableSchema.GlobalIndexes [][]Coln` exists; legacy `GlobalIndexesDeprecated []Coln` still exists.
- `backend/db/reflect.go`: only processes `schema.GlobalIndexesDeprecated` (single-column global index) and does **not** process `schema.GlobalIndexes`.
- Packed local index implementation is already present:
  - `schema.Indexes` processing in `backend/db/reflect.go`
  - local index is registered in `dbTable.indexes` with `Type=2` and has `getStatement` that emits packed predicates + supports `IN` fan-out.
  - query path uses `RequiresPostFilter` to enforce exact semantics (`backend/db/select.go`).
- `backend/db/select_compute.go`: capabilities for regular global index `Type=1` currently match by the indexed column name only; this won’t match composite packed indexes because users query the **source** columns.

## Proposed Semantics for `TableSchema.GlobalIndexes`
Each entry `[]db.Coln{c1, c2, ..., cn}` defines a *single* packed global index where:
- `c1` is the “prefix” component:
  - MUST NOT use `DecimalSize()`
  - may use `.Int32()` to select packed output storage type `int32` (otherwise default `int64`)
- `c2..cn` are the remaining components:
  - MUST all have `DecimalSize(n)` set
- Packing is computed as int64, then (if `.Int32()` was set) trimmed/cast at the end:
  - `packedTrimmed := trimRightToDigitsNonNegative(packed, 9)`
- Negative component values are not supported; panic if encountered.

Digit budget checks (same as local packed indexes):
- If `int32` packed: require `sum(DecimalSize(c2..cn)) <= 8` (panic otherwise)
- If `int64` packed: require `sum(DecimalSize(c2..cn)) <= 18` (panic otherwise)
- First column uses `remainingDigits = totalDigits - sumTrailingDigits` (overflow accepted per requirement).

## Implementation Plan

### 1. Reflection: build packed global index columns + register global index
File: `backend/db/reflect.go`

Add a new loop after `schema.Indexes` (or near the current `GlobalIndexesDeprecated` section) to process:
- `schema.GlobalIndexes`

For each global index definition:
1. Validate inputs using the same rules as packed local indexes:
   - len >= 2
   - all columns are scalar integer types (no slices, no complex types)
   - first column has no DecimalSize; remaining columns have DecimalSize
   - enforce digit budgets as above
2. Create stored virtual packed column:
   - name: `zz_gixp_<col1>_<col2>_...`
   - type: `int` if `.Int32()` was set, else `bigint`
   - `getRawValue(ptr)`:
     - read all component values as `int64`
     - compute packed int64
     - if int32 mode: trim packed to 9 digits then cast to `int32`
3. Create a global index `viewInfo` entry:
   - `Type=1`
   - name: `<table>__<packed_col>_index_0`
   - `getCreateScript()` => `CREATE INDEX ... ON table (<packed_col>)`
4. Attach a `getStatement(statements...) []string` on the index:
   - Accept statements for source columns, not for the packed column.
   - Produce one or more WHERE clauses that predicate on the packed column only.
   - Support prefix `IN` by expanding into multiple WHERE clauses (fan-out).
5. Mark `RequiresPostFilter=true`:
   - Any truncation via `DecimalSize()` or final int32 trimming can overfetch.
   - The caller must post-filter using original source statements.

Concise log line:
- `Packed Global Index registered: table=... index=... packedCol=... isInt32=... slotDigits=...`

### 2. Capability computation: match queries on source columns
File: `backend/db/select_compute.go`

Current problem:
- Default global index capability uses `idx.column.GetName()` (the packed column), but user queries include source columns (`Status`, `Updated`, ...).

Add a new capability generator step similar to the existing “packed local indexes” block:
Implementation detail:
- Reuse the existing packed index metadata (`packedIndexInfo`) and store both local and global packed indexes in a single slice.
- Differentiate local vs global via `packedIndexInfo.partitionColumnName` (empty means global).

For each packed global index:
- Build signatures:
  - `<c1>|=|<c2>|=|...|<c(n-1)>|=|<cn>|=`
  - `<c1>|=|<c2>|=|...|<c(n-1)>|=|<cn>|~`
Notes:
- `IN` is normalized to `"="` in `capabilityOpForStatement`, so `Status IN (...)` matches `Status|=` signatures automatically.
- Priority should be lower than packed local indexes (since local index is more selective when partition is provided).

### 3. Query execution: ensure post-filter works for packed global index fan-out
File: `backend/db/select.go`

This is already implemented generically:
- If selected `viewOrIndex.RequiresPostFilter` is true:
  - set `usePostFilter=true`
  - set `postFilterStatements` to the set of statements used by the index
  - enable deduplication when multiple WHERE statements are emitted

For global packed index:
- Ensure `postFilterStatements` includes the original source predicates so the in-memory filter can remove false positives.
- The existing `recordMatchesPostFilter` supports `IN`, `BETWEEN`, and range ops.

### 4. Deploy: create the GSI
File: `backend/db/deploy.go`

No deploy changes needed if:
- the packed global index is registered in `dbTable.indexes`
- `getCreateScript()` returns `CREATE INDEX ... ON table (packed_col)`

## Testing / Verification
Extend `backend/exec/test_selects.go` with a new test that forces a packed **global** index selection:
- Query without partition equality, e.g.:
  - `Status.Equals(1).Updated.Between(...)`
  - (Optionally keep `EmpresaID.Equals(1)` as a remaining filter to confirm it’s appended correctly.)
- Verify the printed SQL includes:
  - `WHERE zz_gixp_* >= ... AND zz_gixp_* <= ...`
  - no `((empresa_id), ...)` in the CREATE INDEX script (ensure it’s global, not local)
- Validate correctness:
  - Insert rows that collide due to truncation and ensure post-filter removes them.

## Questions
1. For `GlobalIndexes`, do you want the “prefix columns” to allow `IN` on *any* prefix column (not just the first), or only support `IN` on the first column for v1?
   - The local packed index implementation already supports `IN` on any prefix column by building a cartesian expansion.
2. Should we allow `GlobalIndexes` entries with only 1 column (simple GSI), or keep that strictly in `GlobalIndexesDeprecated`?
   - Plan assumes `GlobalIndexes` is only for packed/composite (len >= 2).
