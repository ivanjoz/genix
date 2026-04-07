# ORM QueryIndexGroup Select Plan

## Goal

Add a new read path:

```go
traceSales := []db.RecordGroup[comercial.SaleOrder]{}
query := db.QueryIndexGroup(&traceSales)

query.IncludeCachedGroup(12340001, 1001)
query.IncludeCachedGroup(12340002, 1002)

query.CompanyID.Equals(1).DetailProductsIDs.Contains(3372).Fecha.Between(20517, 20521)
err := query.Exec()
```

The ORM must:

1. Resolve which `UseIndexGroup` definition can satisfy the query.
2. Expand the single `BETWEEN` range into candidate hash values.
3. Probe `<table>__index_updated` first to find which hashes are present and which ones changed since the client cache.
4. Query only the needed hash groups from the base table.
5. Return grouped results as `[]RecordGroup[T]`.

This plan also fixes the side-table schema generation. The current primary key is wrong for the required lookup pattern.

## Current state

### Working pieces already in the ORM

- `UseIndexGroup` is already compiled in [backend/db/index_groups.go](/home/ivanjoz/projects/genix/backend/db/index_groups.go).
- For multi-column groups, the ORM already creates virtual hash columns like `zz_ig_*`, `zz_igs_*`, `zz_iwk_*`, and `zz_iwks_*`.
- Writes already persist `(partition_id, index_hash, update_counter)` rows into `<table>__index_updated`.
- `selectExec()` already supports multi-query fanout in parallel with `errgroup` in [backend/db/select.go](/home/ivanjoz/projects/genix/backend/db/select.go).

### Missing pieces

- `QueryIndexGroup(...)` is a stub in [backend/db/main.go](/home/ivanjoz/projects/genix/backend/db/main.go#L650).
- `IncludeCachedGroup(...)` is a stub in [backend/db/main.go](/home/ivanjoz/projects/genix/backend/db/main.go#L655).
- `RecordGroup[T]` already exposes `GroupHash`, `IndexGroupValues`, `Records`, and `UpdateCounter` in [backend/db/select_grouped.go](/home/ivanjoz/projects/genix/backend/db/select_grouped.go), but the grouped select execution path that fills those fields does not exist yet.
- The side-table DDL currently uses:

```sql
PRIMARY KEY ((partition_id), update_counter, index_hash)
```

That shape is optimized for scanning by counter, but the requested read path needs exact lookup by `partition_id + index_hash`.

## Required behavior

For a query like:

```go
query.CompanyID.Equals(1).DetailProductsIDs.Contains(3372).Fecha.Between(20517, 20521)
```

and a matching schema entry like:

```go
{
    Keys:          []db.Coln{e.Fecha.StoreAsWeek(), e.DetailProductsIDs},
    UseIndexGroup: true,
}
```

the ORM should:

1. Expand `Fecha.Between(20517, 20521)` into day values `20517..20521`.
2. Combine each day with `DetailProductsIDs.Contains(3372)`.
3. Compute `HashInt64(day, 3372)` for each candidate.
4. Probe `sale_order__index_updated` using `partition_id = 1` and `index_hash IN (...)`.
5. Discard hashes that do not exist in the side table.
6. Discard hashes whose `update_counter` matches the client-provided `IncludeCachedGroup(hash, counter)`.
7. Query only the remaining hashes against the base table through the already generated IndexGroup virtual index.
8. Apply exact post-filtering with the original statements so the final records still honor:
   - `CompanyID = 1`
   - `DetailProductsIDs CONTAINS 3372`
   - `Fecha BETWEEN 20517 AND 20521`

## Schema fix

### Correct side-table primary key

The side table must be generated as:

```sql
CREATE TABLE <keyspace>.<table>__index_updated (
    partition_id int,
    index_hash int,
    update_counter int,
    PRIMARY KEY ((partition_id), index_hash)
)
```

Rationale:

- The read path needs `WHERE partition_id = ? AND index_hash IN (...)`.
- `update_counter` is payload, not a clustering dimension.
- Existence check and freshness check are both resolved from one point lookup row per hash.

### Migration stance

Because this project is pre-alpha and the old schema is not useful for the new query pattern, the implementation should not keep backward compatibility.

Planned stance:

- fix the generated DDL in `getIndexUpdatedTableCreateScript()`
- update tests to assert the new primary key
- when deploy detects the old `__index_updated` schema, prefer dropping and recreating that side table instead of trying to migrate rows in place

## Read-path design

### 1. Query state additions

Extend `TableInfo` with dedicated QueryIndexGroup state instead of overloading normal `Query(...)` behavior.

Suggested fields:

```go
type cachedIndexGroupState struct {
    Hash          int32
    UpdateCounter int32
}

type TableInfo struct {
    statements           []ColumnStatement
    columnsInclude       []columnInfo
    columnsExclude       []columnInfo
    groupByColumns       []columnInfo
    between              ColumnStatement
    orderBy              string
    limit                int32
    allowFilter          bool
    refSlice             any
    indexGroupRefSlice   any
    cachedIndexGroups    map[int32]int32
    useIndexGroupSelect  bool
}
```

Notes:

- keep QueryIndexGroup state explicit so the normal `Query(...).Exec()` path does not get more fragile
- store cached groups in a map keyed by hash for O(1) freshness checks

### 2. API wiring

Update:

- `QueryIndexGroup(...)` to set `useIndexGroupSelect = true` and attach `*[]RecordGroup[T]`
- `IncludeCachedGroup(hash, updateCounter)` to populate `cachedIndexGroups`
- `Exec()` to route QueryIndexGroup calls into a dedicated execution function

Recommended split:

```go
func execIndexGroupQuery[T TableSchemaInterface[T], E any](...) error
```

This is cleaner than trying to make `selectExec()` return both `[]E` and `[]RecordGroup[E]`.

### 3. Result contract

`RecordGroup[T]` should use the current struct contract:

```go
type RecordGroup[T any] struct {
    GroupHash        int32
    IndexGroupValues []int64
    Records          []T
    UpdateCounter    int32
}
```

Rationale:

- the client sends `(hash, updateCounter)` pairs
- the server must return the current `updateCounter` for any fetched group
- otherwise `IncludeCachedGroup(...)` cannot be used correctly on the next round-trip

`IndexGroupValues` should remain optional metadata only if the implementation can recover the exact source values cheaply. If not, keep it empty in the first version and only guarantee:

- `GroupHash`
- `UpdateCounter`
- `Records`

### 4. Planner selection

Add a dedicated matcher for QueryIndexGroup over `scyllaTable.indexGroups`.

A candidate index-group is valid when:

- partition column has `=` restriction
- exactly one source column has a `BETWEEN` restriction
- all remaining source columns have hashable operators:
  - `=`
  - `IN`
  - `CONTAINS`
- all source columns for that index-group are covered by statements

Selection rule:

- prefer the candidate that handles the most source columns
- if tied, prefer the one with the fewest cartesian combinations
- if still tied, prefer the raw-date variant over the week-only variant when the range statement is on unix-day input

Important:

- `StoreAsWeek()` already creates both raw and week-backed virtual columns in [backend/db/index_groups.go](/home/ivanjoz/projects/genix/backend/db/index_groups.go)
- for `Fecha.Between(unixDayStart, unixDayEnd)`, the first version should target the raw-date hash variant unless the query itself is expressed in week codes

### 5. Hash expansion

Build a new helper that mirrors the existing write-side hashing, but works from query statements:

```go
func buildQueryIndexGroupHashes(
    indexGroup indexGroupInfo,
    statements []ColumnStatement,
) ([]int32, error)
```

Rules:

- read the single `BETWEEN` statement from the chosen source column
- expand every value in the inclusive range
- combine it with the equality/IN/CONTAINS values from the other source columns
- hash the cartesian product with `HashInt64(...)`
- dedupe and sort hashes for stable probing

For the example above, `20517..20521` and `3372` produce five hashes.

### 6. Probe side-table first

Before touching the base table, read the side table:

```sql
SELECT index_hash, update_counter
FROM <table>__index_updated
WHERE partition_id = ?
  AND index_hash IN (?, ?, ...)
```

Execution rules:

- chunk the `IN` list if needed, reusing the same query fanout style already used elsewhere in the select code
- build `map[int32]int32` of hashes that actually exist in Scylla
- compare each row against `cachedIndexGroups`
- keep only hashes where:
  - the row exists
  - the client does not have that hash, or has an older `update_counter`

This stage removes both:

- nonexistent groups
- unchanged groups already cached by the client

### 7. Fetch changed groups in parallel

For each hash that survives the side-table probe, run one base-table query that uses the compiled IndexGroup virtual column.

Recommended behavior:

- batch hashes per physical statement when the selected index-group can use `IN (...)`
- otherwise fan out one statement per hash
- reuse the existing `errgroup` execution style
- scan into a temporary `[]T` per hash or per statement
- always apply exact post-filtering with the original fluent statements

The grouped read path must return:

- one `RecordGroup[T]` per surviving hash
- `Records` containing only the records that exactly match the original query

### 8. Group assembly

After the base records are fetched:

1. group them by `index_hash`
2. attach `UpdateCounter` from the side-table probe
3. append one `RecordGroup[T]` per changed hash into the destination slice

Order rule for first version:

- preserve the sorted hash order from the probe stage

This keeps the response deterministic without adding more planner complexity.

## Affected files

Primary files:

- [backend/db/main.go](/home/ivanjoz/projects/genix/backend/db/main.go)
  - wire `QueryIndexGroup`
  - store cached-group state
  - route `Exec()` to a dedicated grouped-select executor
- [backend/db/select_grouped.go](/home/ivanjoz/projects/genix/backend/db/select_grouped.go)
  - extend `RecordGroup`
  - add grouped-select helpers
- [backend/db/index_groups.go](/home/ivanjoz/projects/genix/backend/db/index_groups.go)
  - fix side-table DDL
  - add query-side hash expansion helpers
- [backend/db/select.go](/home/ivanjoz/projects/genix/backend/db/select.go)
  - reuse scan helpers for grouped fetches
  - keep exact post-filtering on grouped reads
- [backend/db/deploy.go](/home/ivanjoz/projects/genix/backend/db/deploy.go)
  - recreate old `__index_updated` tables with the corrected PK
- [backend/db/update_counter_test.go](/home/ivanjoz/projects/genix/backend/db/update_counter_test.go)
  - update DDL assertion
- new focused tests under `backend/db`
  - QueryIndexGroup hash expansion
  - cache-hit skip behavior
  - side-table probe filtering
  - grouped fetch assembly

## Tests

Add targeted tests before wiring more application queries onto the feature.

### DDL tests

- assert `PRIMARY KEY ((partition_id), index_hash)`
- assert `update_counter` is still created as a regular column

### Planner tests

- picks the correct `UseIndexGroup` candidate for:
  - raw date + collection
  - raw date + scalar
  - week-backed variants
- rejects queries with:
  - no partition equality
  - no `BETWEEN`
  - more than one `BETWEEN`
  - missing source-column predicates

### Side-table probe tests

- nonexistent hashes are dropped
- cached `(hash, counter)` matches are dropped
- stale cached counters are fetched

### Execution tests

- one changed hash returns one `RecordGroup`
- multiple changed hashes return deterministic sorted groups
- records are exact-filtered after hash-based fetch
- empty surviving hash set returns no groups and no base-table fetch

## Implementation sequence

1. Extend `TableInfo` and implement `QueryIndexGroup(...)` / `IncludeCachedGroup(...)`.
2. Fix `getIndexUpdatedTableCreateScript()` and its tests.
3. Add a deploy-time recreation path for old `__index_updated` tables.
4. Implement QueryIndexGroup planner selection over compiled `indexGroups`.
5. Implement query-side hash expansion from `BETWEEN` plus equality/collection predicates.
6. Implement side-table probe and cached-group filtering.
7. Implement grouped base-table fetch and response assembly.
8. Populate `GroupHash`, optional `IndexGroupValues`, `Records`, and `UpdateCounter`.
9. Add focused unit tests.
10. Document the feature in [backend/docs/ORM_DATABASE_QUERY.md](/home/ivanjoz/projects/genix/backend/docs/ORM_DATABASE_QUERY.md).

## Assumption to confirm

This plan assumes the first version should choose the raw-date IndexGroup variant for queries like:

```go
Fecha.Between(20517, 20521).DetailProductsIDs.Contains(3372)
```

even when the schema declaration uses `Fecha.StoreAsWeek()`, because the compiler already generates both:

- raw-date hash index
- week-code hash index

If you want the grouped read path to probe both raw-day hashes and week-code hashes for the same query, that should be called out explicitly before implementation because it changes both the planner and the cache contract.
