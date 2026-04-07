# ORM Indexes Refactor Plan

## Goal

Refactor the ORM schema API so all index-like declarations use one field:

```go
// One schema entry point for local/global indexes, views, view tables, and index-group writes.
Indexes []Index
```

The old schema fields should be removed, not kept for compatibility:

- `LocalIndexes`
- `HashIndexes`
- `IndexesDeprecated`
- `GlobalIndexes`
- `Views`
- `ViewTables`
- `IndexGroups`

This is a pre-alpha codebase, so the plan assumes direct migration with no backward-compatibility layer.

## Target API

### TableSchema

Replace the current mixed schema surface with a single index declaration list:

```go
// Only one public schema field should describe all index/view strategies.
type TableSchema struct {
    Keyspace          string
    Name              string
    Keys              []Coln
    Partition         Coln
    Indexes           []Index
    SequenceColumn    Coln
    CounterColumn     Coln
    UseSequences      bool
    SequencePartCol   Coln
    KeyConcatenated   []Coln
    KeyIntPacking     []Coln
    AutoincrementPart Coln
    SaveCacheVersion  bool
    UseUpdateCounter  Coln
    DisableUpdateCounter bool
}
```

### Index type

Keep `Index` as the public descriptor:

```go
const (
    // Explicit global secondary index.
    TypeGlobalIndex int8 = 1
    // Explicit local secondary index.
    TypeLocalIndex int8 = 2
    // 3 stays reserved for hash-style runtime indexes if we later expose it publicly.
    // Materialized view.
    TypeView int8 = 6
    // Table-backed derived view.
    TypeViewTable int8 = 9
)

// One struct covers physical indexes, views, and index-group write tracking.
type Index struct {
    Type int8
    Keys []Coln
    Cols []Coln
    KeepPart bool
    UseHash bool
    UseIndexGroup bool
}
```

### Type ID alignment

The constants above should match the actual runtime IDs already used by the ORM today.

Current runtime values in `viewInfo.Type` are effectively:

- `1`: global index
- `2`: local index
- `3`: hash index
- `6`: materialized/simple view
- `7`: hash view
- `8`: range/radix view
- `9`: view table

So the refactor should not introduce a second public numbering scheme like:

- local = `1`
- global = `2`
- view = `3`
- view table = `4`

because that would make `Index.Type` inconsistent with the compiler, planner, logs, and `indexTypes` diagnostics.

Recommendation:

- public explicit constants should reuse the existing runtime IDs
- specialized IDs like `3`, `7`, and `8` can remain internal unless you want to expose them later

## Inference rules

If `Index.Type == 0`, the ORM should infer the strategy.

### Proposed inference order

1. If `Type` is explicitly set, obey it.
2. If `UseIndexGroup` is true, compile with the index-group strategy.
3. If `Type == 0` and `len(Cols) > 0`, infer `TypeView`.
4. If `Type == 0` and `len(Keys) == 1`, infer `TypeLocalIndex`.
5. If `Type == 0` and generated storage is `set<int>`, infer `TypeGlobalIndex`.
6. If `Type == 0` and `len(Keys) >= 2`, infer the multi-column virtual-index path.

Important clarification:

- `hash index` is an internal compilation strategy, not necessarily a new public `Type` constant.
- Public API can stay with `TypeGlobalIndex`, `TypeLocalIndex`, `TypeView`, and `TypeViewTable`.
- Internally, the compiler can still classify one `Index` as:
  - direct local index
  - packed local/global index
  - composite hash/set index
  - materialized view
  - table-backed view
  - index-group tracked index

## UseIndexGroup behavior

`UseIndexGroup` replaces the separate `IndexGroups` field.

Example target schema:

```go
Indexes: []db.Index{
    // Single-column entry: ORM infers a local index and enables index-group tracking.
    {Keys: []db.Coln{e.Fecha}, UseIndexGroup: true},
    // Multi-column scalar entry: ORM generates a virtual hash column plus tracked hashes.
    {Keys: []db.Coln{e.Fecha.StoreAsWeek(), e.Status}, UseIndexGroup: true},
    // Multi-column scalar entry: same index-group strategy, different source tuple.
    {Keys: []db.Coln{e.Fecha.StoreAsWeek(), e.ClientID}, UseIndexGroup: true},
    // Slice-backed entry: ORM generates set<int> storage and tracked hash fanout.
    {Keys: []db.Coln{e.Fecha.StoreAsWeek(), e.ClientID, e.DetailProductsIDs}, UseIndexGroup: true},
}
```

Requested semantics from the earlier design still apply:

- one key:
  - create a local index directly on the real column
- two or more scalar keys:
  - create one virtual `int32` hash column
  - create one index on that virtual column
- any integer slice key:
  - create one virtual `set<int>` hash column
  - create one global index on `VALUES(virtual_col)`
- `StoreAsWeek()`:
  - write fanout includes both raw fecha and derived week code
- writes:
  - maintain `<table>__index_updated`
  - store `partition_id`, `index_hash`, `update_counter`

## Views and ViewTables migration

`Views` and `ViewTables` should also move into `Indexes []Index`.

### Proposed schema style

```go
Indexes: []db.Index{
    // Explicit MV because it has projected columns and MV semantics.
    {
        Type: db.TypeView,
        Keys: []db.Coln{e.StatusTrace.Int32(), e.Updated.DecimalSize(8)},
        Cols: []db.Coln{e.ClientID, e.Updated, e.DetailProductsIDs, e.Status},
        KeepPart: true,
    },
    // Explicit table-backed view because maintenance semantics differ from MV.
    {
        Type: db.TypeViewTable,
        Keys: []db.Coln{e.DetailProductsIDs, e.Fecha},
        Cols: []db.Coln{e.Updated},
        KeepPart: true,
    },
}
```

Recommendation:

- infer `TypeView` when `Cols` is set and `Type == 0`
- require `TypeViewTable` explicitly

Reason:

- `View` and `ViewTable` have very different deploy and maintenance behavior
- implicit `ViewTable` inference would be brittle

## Main refactor design

### 1. One normalization step

Before any compilation, normalize `schema.Indexes` into one internal representation.

Suggested internal shape:

```go
// This is compile-time metadata after type inference and validation.
type normalizedIndexConfig struct {
    typeResolved     int8
    keys             []Coln
    cols             []Coln
    keepPart         bool
    useHash          bool
    useIndexGroup    bool
    sourceConfig     Index
}
```

This normalization should:

- resolve `Type` when omitted
- reject invalid combinations early
- centralize all schema validation in one place

### 2. Split compilers by internal strategy, not by schema field

After normalization, route each entry to one compiler:

- `compileLocalIndex(...)`
- `compileGlobalIndex(...)`
- `compilePackedOrVirtualIndex(...)`
- `compileIndexGroup(...)`
- `compileView(...)`
- `compileViewTable(...)`

This keeps `reflect.go` smaller and removes the current field-by-field branching.

### 3. Keep one runtime registry

`ScyllaTable` already stores:

- `indexes map[string]*viewInfo`
- `views map[string]*viewInfo`
- `packedIndexes []*packedIndexInfo`
- `compositeBucketIndexes []compositeBucketIndex`

The refactor should preserve the runtime registries that the planner already uses, but the source of truth becomes only `schema.Indexes`.

That means:

- no planner rewrite is needed at the public API level
- only the compile pipeline changes
- select capability generation can keep using runtime metadata

### 4. Remove old schema fields completely

Do not keep dual compile paths for:

- `LocalIndexes`
- `HashIndexes`
- `IndexesDeprecated`
- `GlobalIndexes`
- `Views`
- `ViewTables`
- `IndexGroups`

Reason:

- the old fields are exactly what make `reflect.go` fragmented now
- keeping both APIs doubles test surface and migration bugs
- pre-alpha status allows a clean cut

## Compilation rules by feature

### A. Local index

Cases:

- explicit `TypeLocalIndex`
- inferred single-key `Index`

Compile result:

- real-column local secondary index
- same capability generation as today

### B. Global index

Cases:

- explicit `TypeGlobalIndex`
- inferred virtual `set<int>` index-group/global-hash storage

Compile result:

- direct global secondary index or global virtual-column index

### C. Multi-column virtual index

Cases:

- inferred multi-key scalar index
- explicit `TypeLocalIndex` or `TypeGlobalIndex` with multiple keys

Compile result:

- ORM chooses packed numeric column vs. hash column depending on key metadata
- existing helpers like `DecimalSize()`, `Int32()`, `CompositeBucketing()` still decide storage strategy

Important point:

- the refactor should not force every multi-column index into one storage style
- packed range indexes and set/hash indexes are different internal strategies

### D. Index group

Cases:

- any `Index` with `UseIndexGroup == true`

Compile result:

- physical index artifact based on key count and slice usage
- side-table metadata for `<table>__index_updated`
- write fanout/hash generation
- update validation that all source keys must be updated together

### E. Materialized view

Cases:

- explicit `TypeView`
- inferred `TypeView` when `Cols` is set and `Type == 0`

Compile result:

- current MV compiler path
- same `UseHash`, `KeepPart`, and packed-view behavior

### F. Table-backed view

Cases:

- explicit `TypeViewTable`

Compile result:

- current derived-table compiler path
- same fanout maintenance and maintenance-index behavior

## Migration examples

### Before

```go
return db.TableSchema{
    Name:         "sale_order",
    Partition:    e.CompanyID,
    Keys:         []db.Coln{e.ID.Autoincrement(2)},
    LocalIndexes: []db.Coln{e.Updated},
    Views: []db.Index{
        {Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
    },
    ViewTables: []db.Index{
        {
            Keys: []db.Coln{e.DetailProductsIDs, e.Fecha},
            Cols: []db.Coln{e.Updated},
            KeepPart: true,
        },
    },
    IndexGroups: [][]db.Coln{
        {e.Fecha},
        {e.Fecha.StoreAsWeek(), e.Status},
    },
}
```

### After

```go
return db.TableSchema{
    Name:      "sale_order",
    Partition: e.CompanyID,
    Keys:      []db.Coln{e.ID.Autoincrement(2)},
    Indexes: []db.Index{
        // Direct local index.
        {Type: db.TypeLocalIndex, Keys: []db.Coln{e.Updated}},
        // Inferred local index + index-group tracking.
        {Keys: []db.Coln{e.Fecha}, UseIndexGroup: true},
        // Inferred tracked virtual hash index.
        {Keys: []db.Coln{e.Fecha.StoreAsWeek(), e.Status}, UseIndexGroup: true},
        // Explicit MV.
        {Type: db.TypeView, Keys: []db.Coln{e.Status.Int32(), e.Updated.DecimalSize(8)}, KeepPart: true},
        // Explicit table-backed view.
        {
            Type: db.TypeViewTable,
            Keys: []db.Coln{e.DetailProductsIDs, e.Fecha},
            Cols: []db.Coln{e.Updated},
            KeepPart: true,
        },
    },
}
```

## Affected files

Main files expected to change:

- `backend/db/main.go`
  - remove old schema fields
  - keep only `Indexes []Index`
  - keep `Index` public type
  - align exported `Type*` constants with existing runtime IDs
- `backend/db/reflect.go`
  - replace field-by-field compilation with one normalized `Indexes` loop
  - move compile logic into smaller helpers
- `backend/db/packed_indexes.go`
  - adapt callers to the new normalized config flow
- `backend/db/view_tables.go`
  - keep runtime behavior, change only schema entry path
- `backend/db/select_compute.go`
  - verify capability generation still works with unified compilation output
- `backend/db/deploy.go`
  - no schema-field assumptions, just runtime metadata
- `backend/db/insert-update.go`
  - keep index-group write maintenance under the unified `Indexes` flow
- `backend/docs/ORM_DATABASE_QUERY.md`
  - rewrite schema examples to use `Indexes []Index`

## Implementation sequence

1. Add the new plan doc and freeze further expansion of old schema fields.
2. Align `Type*` constants in `main.go` with current runtime `viewInfo.Type` IDs.
3. Implement a normalizer for `schema.Indexes`.
4. Move current compile logic into helper functions keyed by normalized strategy.
5. Migrate `Views` compiler to accept only normalized `TypeView` entries.
6. Migrate `ViewTables` compiler to accept only normalized `TypeViewTable` entries.
7. Migrate current local/global/packed index compilers to normalized `Indexes`.
8. Migrate index-group logic from the separate plan into `UseIndexGroup`.
9. Delete old schema fields from `TableSchema`.
10. Update all schema declarations in application code.
11. Update docs and tests.

## Tests

### Schema normalization tests

- explicit `TypeLocalIndex` stays local
- explicit `TypeGlobalIndex` stays global
- `Cols != nil` with `Type == 0` becomes `TypeView`
- `TypeViewTable` stays explicit
- single-key `UseIndexGroup` normalizes correctly
- invalid combinations fail fast with descriptive errors

### Compilation tests

- one-key inferred index builds a local index
- multi-key scalar index builds the expected virtual index strategy
- slice-backed key builds `set<int>` plus `VALUES(...)` index
- `TypeView` compiles to MV metadata
- `TypeViewTable` compiles to derived-table metadata

### Planner tests

- capability generation is unchanged for equivalent schemas after migration
- packed indexes still match equality and range correctly
- view-backed queries still route correctly

### Write-path tests

- `UseIndexGroup` writes `<table>__index_updated`
- partial update of tracked keys fails
- `StoreAsWeek()` still expands raw date + week code

### Migration tests

- representative schemas like `sale_order` compile with only `Indexes []Index`
- no test uses removed fields

## Non-goals

- backward compatibility for old schema fields
- supporting both old and new APIs in parallel
- redesigning the query DSL
- changing the runtime planner contracts unless the unified compiler requires a small fix

## Open points

1. Should `TypeViewTable` always be explicit, or do you also want inference for it?
2. For multi-key indexes without `UseIndexGroup`, what should be the default internal strategy:
   - packed numeric
   - hash virtual column
   - choose by key metadata
3. Should `UseHash` remain public on `Index`, or should it become an internal compiler detail after the unification?
