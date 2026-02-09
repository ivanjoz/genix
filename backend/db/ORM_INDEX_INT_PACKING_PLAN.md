# ORM Plan: Packed Int Local Indexes via `TableSchema.Indexes`

## Goal
Support schema definitions like:

```go
Indexes: [][]db.Coln{
  {e.Status.Int32(), e.Updated.DecimalSize(8)},
},
```

to generate a stored "virtual" numeric column (example: `zz_virtual`) that packs multiple numeric components into a single sortable integer, plus a *local secondary index* on `(partition, packed)`. This enables Scylla queries for patterns like:

- `EmpresaID = ? AND Status = ? AND Updated > ?` (delta sync)
- `EmpresaID = ? AND Status = ? AND Updated BETWEEN ? AND ?`

while allowing a post-filter when `DecimalSize()` truncation loses precision.

## What Exists Today (Relevant)
- `KeyIntPacking` works only for the table primary key (`backend/db/reflect.go`, `backend/db/select.go`).
- "ViewsDeprecated" already creates stored virtual columns (named `zz_...`) and materialized views (Type 6/7/8) (`backend/db/reflect.go`).
- `TableSchema.Indexes` exists in `backend/db/main.go` but is not processed anywhere yet.
- `.Int32()` currently only sets `columnInfo.useInt32Packing` and is not used. You said it is just a placeholder, but for this feature it will mean: the *packed output column type* is `int32` (else default `int64`).

## Proposed Semantics for `Indexes`
Each entry `[]db.Coln{c1, c2, ...}` defines:
1. A stored virtual packed column `zz_<joined>_pack` (name deterministic from source cols).
2. A *local secondary index* on `((partition_key), zz_<joined>_pack)`.
3. Query translation: user writes predicates on source columns, ORM uses the index capability and rewrites WHERE to use the packed column.
4. Optional post-filter: if any component uses `DecimalSize()` truncation, ORM applies an in-memory filter to guarantee exact semantics.

### Packing Formula (matches your example)
Given:
- totalDigits is derived from output type:
  - `int32` mode: `totalDigits = 9`
  - `int64` mode: `totalDigits = 19`
- each component has a slot width in digits:
  - if component has `DecimalSize(n)`, its slot width is `n`
  - otherwise its slot width is inferred as `remainingDigits`

Rules / constraints:
- Only the *first* component may omit `DecimalSize()` (typical: `Status`).
- If any component (other than the first) omits `DecimalSize()`, `reflect.go` must `panic` with a descriptive error.
- If any component after the first has `DecimalSize()`, then *all* remaining components must have `DecimalSize()` set (enforced in `reflect.go`).
- Packed output is always non-negative. If any component is negative at runtime, `getValue()` must `panic` (programmer error).
- Packed output width constraints:
  - For `int32` packed output: enforce `sum(DecimalSize(components[1:])) <= 8`.
    - Rationale: `int32` max is 10 digits but the first digit cannot exceed `2`; reserving 1 digit for the first component and capping the rest to 8 keeps the packed value safely bounded for most realistic first-component values.
    - The remaining 1 digit (of the 9-digit budget) is implicitly assigned to the first component; if it overflows, that's acceptable per requirements.
  - For `int64` packed output: enforce `sum(DecimalSize(components[1:])) <= 18` for the same reason (leave at least 1 digit for the first component).

Then:
- each component value is reduced to fit its slot width by dropping least-significant digits (right-trim):
  - `trimRightToDigits(v, width)`:
    - if `digits(v) > width`, divide by `10^(digits(v)-width)`
    - otherwise unchanged
  - generic: works whether the input is 6 digits, 9 digits, 11 digits, etc.
  - example: `370598453` with width `8` becomes `37059845`
- packed = sum(component[i] * 10^shift[i]) where shifts are computed from slot widths (like KeyIntPacking).

Example:
- Status=2 (slot=1), Updated=370598453 trimmed to 37059845 (slot=8)
- packed = 2*1e8 + 37059845 = 237059845

### Why Local Index (per requirement)
This feature must generate a *local index* (not a materialized view). The local index is created with:
`CREATE INDEX <table>__<packed_col>_index_1 ON <table> ((<partition_col>), <packed_col>)`

## Concrete Implementation Steps

### 1. Schema/Reflection: build packed columns + local index
File: `backend/db/reflect.go`

Add processing for `schema.Indexes` after `LocalIndexes/HashIndexes` and before `ViewsDeprecated` (order not critical, but keep deterministic).

For each `indexColumns := range schema.Indexes`:
- Validate:
  - length >= 2
  - all columns are numeric scalar types (no slices, no complex types)
  - only the first column may omit `DecimalSize()`; if any later column omits it: panic
  - if any later column has `DecimalSize()` then all later columns must have `DecimalSize()` (enforced)
  - `.Autoincrement()` not allowed here (unless explicitly required later)
- Decide packing output type:
  - if the first column (or any column, by convention) uses `.Int32()`, packed output type is `int32`
  - else default to `int64`
- Create a stored virtual column:
  - name: `zz_ixp_<col1>_<col2>_...` (stable)
  - type: `int` or `bigint` (CQL) based on output
  - `getValue(ptr)` computes packed value using the formula above
- Register a local index (`viewInfo` with `Type=2`) for the packed virtual column:
  - name: `<table>__<packed_col>_index_1`
  - `columns`: `[]string{partition_col, packed_col}` (like other local indexes)
  - `getCreateScript()`: `CREATE INDEX ... ON table ((pk), packed_col)`
  - Add `getStatement(...)` to this index so the ORM can translate predicates on source columns into WHERE clauses on `packed_col` while still querying the base table.

Logging:
- Add a single line per index built: table, index name, packed col name, output type, slot widths.

### 2. Writes: ensure virtual packed column is always persisted
Files:
- `backend/db/insert-update.go`
- (optional) `backend/db/reflect.go` (dependency metadata)

Insert:
- No special change needed if the packed virtual column is in `scyllaTable.columns` and has `getValue` implemented (like existing view virtual columns). Insert already uses `GetStatementValue()` then `GetValue()`.

Update:
- Extend the existing "virtual index dependencies" enforcement:
  - If any source column of an `Indexes` packed index is updated, require all source columns be included OR recompute anyway.
  - Append the packed virtual column to `columnsToUpdate` when any source column is included.
  - This mirrors the existing behavior for `indexViews.column.IsVirtual`.

### 3. Query planning: route to packed local index + rewrite WHERE
Files:
- `backend/db/select_compute.go` (capabilities)
- `backend/db/select.go` (translation + optional post-filter)

Capabilities:
- Ensure the new packed local index contributes signatures like:
  - `empresa_id|=|status|=|updated|~`
  - `empresa_id|=|status|=|updated|=`
  - support `Status IN (...)` by expanding into multiple WHERE statements (fan-out) because the packed clause itself is different per status.

Statement generation (`getStatement`):
- Implement for these cases (minimum set to support your handler):
  - `EmpresaID =` (partition)
  - `Status =` and `Status IN (...)` (prefix)
  - `Updated > / >= / BETWEEN` (range on last component)
- Generate CQL on packed column:
  - `pk = ? AND zz_ixp_* > packed(status, updatedFrom)`
  - For BETWEEN:
    - `pk = ? AND zz_ixp_* >= packed(status, from) AND zz_ixp_* <= packed(status, to)`
  - For `Status IN (...)`:
    - expand to multiple WHERE statements (one per status value) and let existing parallel execution merge.

Post-filter (needed when truncation used):
- If any component uses `DecimalSize()` truncation, range queries can overfetch:
  - After fetching results, filter in-memory with the original predicates on `Status` and `Updated` to remove false positives.
  - Implementation note: `select.go` currently only applies `postFilterStatements` when `useCompositePlan == true`.
    - Change this to a generic `usePostFilter` boolean so packed-index queries can also post-filter.
    - Reuse `matchesCompositeFilter(...)` (it already supports `=`, `IN`, `CONTAINS`, `BETWEEN`).

Logging:
- When chosen: log selected packed index view, packed bounds computed, and whether post-filter is active.

### 4. Deploy: create the local index
File: `backend/db/deploy.go`

No new deploy code is needed if:
- the packed virtual column is part of base table `columns`
- the packed local index is registered in `dbTable.indexes` and has `getCreateScript`

## Tests / Verification Plan
Add a focused test harness (or extend existing) that:
1. Creates a minimal table with `EmpresaID` partition, `ID` key, `Status`, `Updated`, and one `Indexes` packed definition.
2. Inserts rows that differ only in the trimmed digit (to validate post-filter correctness):
   - Updated values that share the same first 8 digits but differ in last digit.
3. Runs queries:
   - `Status=2, Updated BETWEEN ...` expects exact set after post-filter.
   - `Status=2, Updated > ...` expects correct lower bound behavior.
4. Confirms the chosen plan is the packed local index (and no `ALLOW FILTERING` needed for these queries).

## Open Questions (Need Your Answers)
Resolved:
- No backfill is required; existing rows may not participate until updated/rewritten.
- Negative component values are not supported; panic if encountered.
- `Indexes` must support up to 4 components (and generally N components) with the same "only first may omit DecimalSize" rule.
