# ORM IndexGroups Plan

## Goal

Implement `TableSchema.IndexGroups` as a write-maintained indexing feature with two outputs:

1. The physical Scylla index artifacts needed to query the base table efficiently.
2. A new side table `<table_name>__index_updated` that stores which index-group hashes changed for each write operation.

The requested example is `sale_order`, which should produce `sale_order__index_updated`.

## Requested behavior

`IndexGroups` should behave like a constrained hash-index builder:

- If the group has 1 column, create a local index directly on that column.
- If the group has 2 or more columns, create one virtual hash column and one index for that virtual column.
- If any column in the group is an integer slice (`[]int`, `[]int8`, `[]int16`, `[]int32`, `[]int64`), the virtual column must be `set<int>` and the index must be global on `VALUES(virtual_col)`.
- If all columns in the group are scalar integers, the virtual column can be a scalar `int32` and the index can be global on that virtual column.
- `StoreAsWeek()` means the source fecha contributes two values during hash fanout:
  - the raw unix-day value
  - the week code (`YYWW`, e.g. `2603`)

For write maintenance, every insert or update must compute all group combinations, hash them as `int32`, and insert them into `<table_name>__index_updated` with:

- `partition_id`
- `index_hash`
- `update_counter`

`update_counter` must be the same managed counter already assigned to the whole write operation.

## Current state

Relevant findings from the current ORM:

- `IndexGroups` already exists in `TableSchema` but is not compiled yet.
- `StoreAsWeek()` currently does nothing.
- Managed `updated` and `update_counter` assignment already happens centrally in `applyWriteManagedColumns`.
- Existing virtual-index patterns already exist in:
  - `HashIndexes` with `set<int>` virtual columns
  - `Indexes` / `GlobalIndexes` with packed virtual columns
- Deploy flow already knows how to create:
  - base tables
  - secondary indexes
  - view-backed tables

This means the feature should be added as a new ORM compilation + write-maintenance path, not as ad hoc logic in `sales.go`.

## Main design decisions

### 1. Add dedicated metadata for IndexGroups

Add a new runtime metadata structure in `backend/db` for compiled index groups.

Suggested shape:

```go
type indexGroupInfo struct {
    name                    string
    sourceColumns           []IColInfo
    sourceColumnNames       []string
    generatedColumn         IColInfo
    generatedColumnIsSet    bool
    generatedColumnIsScalar bool
    usesStoreAsWeek         bool
    writeHashes             func(ptr unsafe.Pointer) []int32
}
```

`ScyllaTable` should keep:

```go
indexGroups []indexGroupInfo
indexUpdatedTable *indexUpdatedTableInfo
```

### 2. Do not reuse `isWeek` for `StoreAsWeek`

`isWeek` is already used by `CompositeBucketing` to mean "the stored value is already week-coded".

`StoreAsWeek()` has a different meaning:

- source field is still a unix-day date
- write fanout must include both raw fecha and derived week code

So the ORM should add a new metadata flag, for example:

```go
storeAsWeek bool
```

and `StoreAsWeek()` should set that flag.

### 3. Copy week helpers into `db`

To avoid import cycles, copy the minimum required week conversion logic from `backend/core/time-helpers.go` into `backend/db`.

Recommended new file:

- `backend/db/week_helpers.go`

It should expose small cached helpers such as:

- `makeWeekCodeFromUnixDay(fechaUnix int16) int16`
- `makeUnixDayFromWeekCode(weekCode int16) int16`

Cache requirements:

- `fechaUnix -> weekCode`
- `weekCode -> mondayFechaUnix`

Use the same week semantics as `core.MakeSemanaFromFechaUnix(..., false)`, because the requested week code format is `YYWW` (`Code`, not `Id`).

### 4. Hash fanout algorithm

For each `IndexGroups` entry:

1. Resolve each source column into a list of candidate `int64` values.
2. If the column uses `StoreAsWeek()`, expand one fecha into two values:
   - raw fecha
   - derived week code
3. If the column is an integer slice, expand into one value per slice element.
4. Build the cartesian product across all columns.
5. Hash each combination with `HashInt64(...)`.
6. Deduplicate and sort the hashes for deterministic writes.

Example:

```go
{e.Fecha.StoreAsWeek(), e.ClientID, e.DetailProductsIDs}
```

If `DetailProductsIDs` has 3 elements, the fanout is:

- `2` values from `Fecha.StoreAsWeek()`
- `1` value from `ClientID`
- `3` values from `DetailProductsIDs`

Total: `2 * 1 * 3 = 6` hashes.

### 5. Physical index generation rules

Compilation rules for `IndexGroups`:

- 1 column:
  - create a local index directly on the column
  - do not create a virtual column
- 2+ scalar columns:
  - create one virtual `int32` column
  - create one global secondary index on that column
- 2+ columns with any integer slice:
  - create one virtual `set<int>` column
  - create one global secondary index on `VALUES(virtual_col)`

Recommended naming:

- local direct index:
  - `<table>__<column>_igroup_index_1`
- scalar virtual column:
  - `zz_ig_<col1>_<col2>...`
- set virtual column:
  - `zz_igs_<col1>_<col2>...`
- global index on virtual column:
  - `<table>__<virtual_col>_index_0`

### 6. New side table for write tracking

Each base table with at least one `IndexGroups` entry should compile a side-table descriptor:

- name: `<table>__index_updated`
- columns:
  - `partition_id bigint` or the normalized integer type used by the base partition
  - `index_hash int`
  - `update_counter int`

Before implementation, one schema detail needs confirmation:

- If reads will scan "all changed hashes since counter X", the best PK is likely `PRIMARY KEY ((partition_id), update_counter, index_hash)`.
- If reads will lookup one hash and then inspect counters, the best PK is likely `PRIMARY KEY ((partition_id, index_hash), update_counter)`.

The current request only defines the 3 columns, not the primary-key order, so the plan should keep this as a pre-implementation decision.

### 7. Write path integration

The side-table write hook should run after the base-table insert/update succeeds and use the same `update_counter` already assigned to each record.

Recommended flow:

1. `applyWriteManagedColumns(...)` assigns `updated` and `update_counter`.
2. Base-table insert/update executes.
3. ORM computes `IndexGroups` hashes for every written record.
4. ORM inserts deduplicated `(partition_id, index_hash, update_counter)` rows into `<table>__index_updated`.
5. Cache-version updates run after all write-maintenance steps succeed.

Deduplication rule:

- Deduplicate identical tuples inside the current write batch to avoid redundant side-table inserts.

### 8. Update validation rules

`IndexGroups` must follow the same consistency rule already used by other virtual indexes:

- if an update touches any source column of one group, it must include all source columns of that group

Otherwise the ORM should panic with a descriptive message.

This check belongs in the same update planning phase that already validates:

- composite packed indexes
- composite bucket indexes

### 9. Debug logging

Add concise debug logs guarded by existing debug flags.

Useful logs:

- `IndexGroup compiled: table=sale_order group=fecha_client_id_detail_products_ids mode=set hashes=fanout`
- `IndexGroup write hashes: table=sale_order partition=7 update_counter=41 groups=5 tuples=18 unique_hashes=14`
- `IndexUpdated write batch: table=sale_order rows=14`

## Affected files

Expected main touch points:

- `backend/db/main.go`
  - add metadata field for `StoreAsWeek()`
- `backend/db/reflect_accessors.go`
  - extend `columnInfo`
- `backend/db/reflect.go`
  - compile `IndexGroups`
  - register virtual columns and indexes
  - register side-table metadata
- `backend/db/insert-update.go`
  - compute and persist `__index_updated` rows after base writes
  - validate updates touching `IndexGroups`
- `backend/db/deploy.go`
  - create `<table>__index_updated` when missing
- `backend/db/week_helpers.go`
  - copied week conversion cache from `core`
- `backend/docs/ORM_DATABASE_QUERY.md`
  - document `IndexGroups` and `StoreAsWeek()`

## Implementation sequence

1. Add `storeAsWeek` metadata and make `StoreAsWeek()` set it.
2. Copy minimal week conversion helpers into `backend/db` with caches for both directions.
3. Add compiled runtime metadata for `IndexGroups`.
4. Implement `IndexGroups` compilation in `reflect.go`.
5. Add side-table metadata and deploy support for `<table>__index_updated`.
6. Add write-hash generation helpers with deterministic deduplication.
7. Hook side-table inserts into `Insert`, `Update`, and `UpdateExclude`.
8. Add update validation so partial group updates fail fast.
9. Add tests.
10. Update docs.

## Tests

Add focused tests before wiring the feature into application tables.

### Compile-time tests

- single-column group creates one local index and no virtual column
- two scalar columns create one virtual `int32` column and one global index
- group with integer slice creates one virtual `set<int>` column and `VALUES(...)` index
- `StoreAsWeek()` marks the correct metadata without colliding with `isWeek`

### Fanout tests

- raw scalar group hashes one tuple
- `StoreAsWeek()` hashes both raw fecha and week code
- slice + `StoreAsWeek()` produces the requested cartesian count
- duplicate slice elements are deduplicated in final hash output

### Write-path tests

- insert writes one `update_counter` value and the matching side-table rows
- update writes side-table rows only after base update succeeds
- identical hashes across many records in the same batch are deduplicated
- partial update of one group fails with a descriptive error

### Deploy tests

- table with `IndexGroups` creates `<table>__index_updated`
- re-deploy does not try to recreate it

## Non-goals for the first iteration

- new query API on top of `<table>__index_updated`
- backfilling old rows automatically
- non-integer array support
- string/date mixed hashing beyond what `IndexGroups` currently needs
- backward-compatibility shims for deprecated index definitions

## Open questions

1. What is the primary-key order for `<table>__index_updated`?
2. Should single-column `StoreAsWeek()` also create a week-derived physical index, or only store both hashes in the side table?
3. Should side-table maintenance be append-only, or do you also want cleanup/rebuild tooling for recomputing hashes from existing data?
