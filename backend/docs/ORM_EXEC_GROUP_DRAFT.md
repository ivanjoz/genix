# ORM `ExecGroup` Draft

## Goal

Add an opt-in grouped execution path without refactoring `TableInfo` or changing current `db.Col[...]` model declarations.

This design keeps normal queries unchanged:

```go
query.Exec()
```

And adds a grouped variant:

```go
query.ExecGroup(getKey, groupHandler)
```

## Proposed API

```go
func (e *TableStruct[T, E]) ExecGroup(
    getKey func(record *E) string,
    groupHandler func(newRecord *E, groupedRecord *E),
) error
```

## Why This Design

- `TableStruct[T, E]` already knows the record type `E`.
- No need to make `TableInfo` generic.
- No need to change `db.Col[Schema, Field]` declarations across the project.
- Grouping stays strongly typed.
- `Exec()` keeps its current behavior.

## Execution Model

`ExecGroup(...)` is not a database aggregation.

The flow is:

1. Build the CQL query as today.
2. Execute the Scylla iterator as today.
3. Decode each row into `E`.
4. Compute a string key with `getKey`.
5. Use an internal `map[string][]E` accumulator.
6. Call `groupHandler(&record, groupedRecord)` where `groupedRecord` is `nil` for the first record of a key.
7. After scanning finishes, flatten grouped buckets into the destination slice.

## Internal Storage

Use:

```go
map[string]E
```

### Rationale

- A grouped query should keep one aggregated record per key.
- The reducer mutates the existing grouped record when a collision occurs.
- The first record can be initialized by mutating `newRecord` before it is stored in the map.

## Example Usage

```go
query := db.Query(&result.Movimientos)

query.
    EmpresaID.Equals(req.Usuario.EmpresaID).
    WarehouseID.Equals(almacenID).
    Fecha.Between(fechaInicio, fechaFin).
    OrderDesc().
    Limit(1000)

err := query.ExecGroup(
    func(record *negocioTypes.AlmacenMovimiento) string {
        return fmt.Sprintf("%d|%d|%s", record.ProductoID, record.WarehouseID, record.Lote)
    },
    func(newRecord *negocioTypes.AlmacenMovimiento, groupedRecord *negocioTypes.AlmacenMovimiento) {
        if groupedRecord == nil {
            newRecord.Inflows_ = newRecord.Cantidad
            return
        }

        groupedRecord.Inflows_ += newRecord.Cantidad
    },
)
```

## Minimal ORM Changes

### 1. Add `ExecGroup(...)` on `TableStruct`

Keep `Exec()` unchanged.

Add:

```go
func (e *TableStruct[T, E]) ExecGroup(
    getKey func(record *E) string,
    groupHandler func(newRecord *E, groupedRecord *E),
) error
```

### 2. Add grouped execution path below `execQuery`

Suggested shape:

```go
func execQueryGrouped[T TableSchemaInterface[T], E any](
    schemaStruct *T,
    tableInfo *TableInfo,
    getKey func(record *E) string,
    groupHandler func(newRecord *E, groupedRecord *E),
) error
```

This keeps grouped logic separate from plain `execQuery`.

### 3. Add grouped scan helper

Suggested helper:

```go
func scanSelectQueryRowsGrouped[E any](
    queryStr string,
    queryValues []any,
    columnNames []string,
    scyllaTable ScyllaTable[any],
    groupedRecords map[string]E,
    postFilterStatements []ColumnStatement,
    getKey func(record *E) string,
    groupHandler func(newRecord *E, groupedRecord *E),
) error
```

This avoids complicating the current non-grouped path.

## Fanout Query Handling

`selectExec` can generate multiple `WHERE` clauses and run them in parallel.

For grouped execution:

1. Each goroutine uses its own local `map[string][]E`.
2. After `errgroup.Wait()`, merge the local grouped maps.
3. Flatten the merged result into `recordsGetted`.

### Why

- avoids locks
- avoids shared mutable state
- matches current fanout merge strategy

## Merge Rules For Fanout Grouping

When two local maps contain the same key:

1. Take the grouped record from the source map.
2. Call `groupHandler` against the destination record.

This guarantees one grouping rule for both:

- rows grouped inside one query branch
- rows merged across multiple query branches

## Validation Rules

Return descriptive errors when:

- `getKey == nil`
- `groupHandler == nil`
- `getKey(record) == ""`

## `Limit()` Semantics

Keep current meaning:

- `LIMIT` applies to raw DB rows
- grouping happens after scan

So:

- 1000 scanned rows may become 30 grouped rows

## Ordering Semantics

First version should document this clearly:

- scan order follows DB query order per branch
- grouped output order is based on bucket flattening, not guaranteed stable across merged fanout branches

If stable order is required, keep:

```go
groupOrder []string
```

with first-seen keys.

## Benefits Compared To `GroupKeyGetter` + `GroupByHandler` Stored In Query State

- less invasive
- clearer API boundary
- no extra persistent query state
- easier to reason about
- easier to test

## Risks

- duplicates may still need pre-deduplication if fanout paths already overfetch the same primary record
- grouped output ordering must be documented
- some handlers may accidentally create many records per key and increase memory use

## Suggested First Scope

Implement only:

- typed `ExecGroup(...)`
- internal `map[string][]E`
- support for normal and fanout queries
- descriptive validation errors

Do not implement:

- DB-side aggregation
- streaming grouped callbacks
- generic `TableInfo`
- compatibility wrappers for older grouping APIs
