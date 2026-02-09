# Stage 2 Simplification Plan: Split `reflect.go` / `makeTable()` Into Phases

## Goal
Make `backend/db/reflect.go` smaller and easier to reason about by:
- Splitting `makeTable()` into named phases.
- Moving schema-compilation strategies (indexes/views) into focused files.
- Keeping behavior identical (no query-plan or deploy behavior changes).

Stage 1 already extracted packed index code into `backend/db/packed_indexes.go`. Stage 2 builds on that.

## Current Pain Points
- `makeTable()` mixes 3 concerns:
  - struct reflection + column mapping
  - schema compilation (indexes, virtual columns, views)
  - finalization (columns slice, indexViews, capability computation)
- Many strategy blocks are long (views, hash/composite bucketing), and the ordering is hard to verify.

## Constraints / Non-Goals
- No behavioral changes (same generated column names, index names, and view names).
- No changes to Scylla deploy logic beyond where it reads `dbTable.indexes`/`dbTable.views`.
- Keep extensive debug logs (but move them with the strategy code).

## Proposed Refactor Structure

### 1) Split `makeTable()` Into Small Internal Functions
File: `backend/db/reflect.go`

Turn `makeTable()` into a pipeline of calls:

1. `func newScyllaTableFromSchema(structRefValue reflect.Value, schema TableSchema) ScyllaTable[any]`
   - Initializes `ScyllaTable` base fields/maps and `_maxColIdx`.

2. `func mapSchemaStructColumns(dbTable *ScyllaTable[any], structRefValue reflect.Value, schema TableSchema)`
   - Current loop over struct fields implementing `Coln`.
   - Applies sequence/counter column type override.
   - Sets `Idx` and fills `columnsMap`.

3. `func applyPartitionAndKeys(dbTable *ScyllaTable[any], schema TableSchema)`
   - Sets `partKey`, keys, keysIdx, autoincrementCol/autoincrementPart wiring.

4. `func applySmartKeyPacking(dbTable *ScyllaTable[any], schema TableSchema)`
   - Moves KeyIntPacking and KeyConcatenated logic (including placeholder handling).

5. `func registerIndexesAndViews(dbTable *ScyllaTable[any], schema TableSchema, idxCount *int8)`
   - Calls into strategy-specific helpers in other files (see below).
   - Ordering preserved.

6. `func finalizeTable(dbTable *ScyllaTable[any])`
   - Populates `dbTable.columns` and `columnsIdxMap`.
   - Builds `indexViews`.
   - Computes `columnsIdx` for views/indexes.
   - Calls `ComputeCapabilities()`.

`makeTable()` becomes a short orchestration function that preserves existing ordering.

### 2) Move Strategy Blocks Out of `reflect.go`
Create focused files in `backend/db/`:

- `schema_local_indexes.go`
  - `func registerLocalIndexes(dbTable *ScyllaTable[any], schema TableSchema, idxCount *int8)`
  - Extracts the current `schema.LocalIndexes` loop.

- `schema_global_indexes.go`
  - `func registerSimpleGlobalIndexes(dbTable *ScyllaTable[any], schema TableSchema, idxCount *int8)`
  - Extracts the `schema.GlobalIndexes` single-column branch (including VALUES(slice) handling).
  - Composite packed global indexes stay handled by `registerPackedIndex` (stage 1).

- `schema_hash_indexes.go`
  - `func registerCompositeBucketingHashIndexes(dbTable *ScyllaTable[any], schema TableSchema, idxCount *int8)`
  - Moves the current `schema.HashIndexes` + CompositeBucketing implementation.
  - Keeps helper functions it relies on in a nearby file (see next bullet).

- `composite_bucketing.go`
  - Move composite bucketing helper functions currently in `reflect.go`:
    - `normalizeCompositeBucketSizes`
    - `isCompositeNumericFieldType`
    - `flattenCompositeInt64Values`
    - `getCompositeBucketValues`
    - `computeCompositeHashSet`
  - Keep names unchanged to avoid churn.

- `schema_views_deprecated.go`
  - `func registerViewsDeprecated(dbTable *ScyllaTable[any], schema TableSchema)`
  - Moves the whole ViewsDeprecated compilation (Type 6/7/8 viewInfo creation + getCreateScript + getStatement logic).
  - This is a large block and the best win after packed-index extraction.

Packed indexes are already moved:
- `backend/db/packed_indexes.go`

### 3) Ensure Logs and Panics Stay Equivalent
- Preserve panic messages as much as possible (or keep them descriptive).
- Keep the existing `fmt.Printf(...)` log lines, but move them with their strategy.

### 4) Testing / Verification
Compilation:
- `cd backend && go test ./...`

Behavioral smoke checks:
- Run existing `backend/exec/test_selects.go` and verify:
  - The same “registered” logs appear.
  - Generated query strings match for:
    - local packed index range
    - global packed equality
    - composite bucketing query plan
    - ViewsDeprecated routing

Optional diff tool:
- Add a temporary `DebugSchemaDump` helper that prints:
  - sorted column names
  - index names and create scripts
  - view names and create scripts
  - computed capabilities signatures
- Use it before/after refactor to confirm no diffs, then remove it.

## Incremental Execution Plan (Least Risk First)
1. Extract `schema.LocalIndexes` loop to `schema_local_indexes.go`.
2. Extract `schema.GlobalIndexes` single-column branch to `schema_global_indexes.go`.
3. Move composite bucketing helpers + hash index registration to `composite_bucketing.go` + `schema_hash_indexes.go`.
4. Move ViewsDeprecated to `schema_views_deprecated.go` (largest block).
5. Finally, split `makeTable()` into named phases (pure refactor), using the moved helpers.

## Acceptance Criteria
- `reflect.go` no longer contains long strategy blocks; it mostly orchestrates.
- No changes in:
  - generated column/index/view names
  - CQL create scripts
  - query routing / chosen capabilities (except where already fixed for GSI range)
- `go test ./...` compiles, and existing manual tests keep working.
