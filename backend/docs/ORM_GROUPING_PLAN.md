# ORM Grouping Plan

## Goal

Add optional in-memory grouping to `db.Query(...)` so query results can be reduced at ORM level after rows are scanned from ScyllaDB.

This is not a database `GROUP BY`.
The query still fetches rows normally, and the ORM groups decoded records in memory before returning them to the caller.

## Requested API

Add two fluent methods on query objects:

```go
query.GroupKeyGetter(func(record *T) string)
query.GroupByHandler(func(newRecord *T, groupedRecords *[]T))
```

### API intent

- `GroupKeyGetter` returns the grouping key for one decoded record.
- `GroupByHandler` receives the incoming record plus the existing grouped records for that key.
- The handler decides how to merge or append records inside the bucket.

## Internal Storage

Use an internal accumulator:

```go
map[string][]T
```

### Rationale

- Supports one-to-many grouping, not only one-to-one reduction.
- Keeps the implementation flexible for future cases where a key may expand into multiple grouped rows.
- Avoids forcing the ORM into a single-record-per-key assumption.

## Query Flow

Current flow:

1. Build CQL query.
2. Execute `Iter()` and `Scanner()`.
3. Decode each row into `T`.
4. Append every record to the result slice.

New grouped flow:

1. Build CQL query.
2. Execute `Iter()` and `Scanner()`.
3. Decode each row into `T`.
4. If grouping is disabled, append as today.
5. If grouping is enabled:
   - compute key with `GroupKeyGetter`
   - fetch `groupedRecords := groupedMap[key]`
   - call `GroupByHandler(&newRecord, &groupedRecords)`
   - store the updated bucket back into `groupedMap[key]`
6. After scan completes, flatten grouped buckets into the destination slice.

## ORM Changes

### 1. Extend `TableInfo`

Add grouping config fields to `TableInfo`.

Suggested shape:

```go
type TableInfo struct {
    statements        []ColumnStatement
    columnsInclude    []columnInfo
    columnsExclude    []columnInfo
    between           ColumnStatement
    orderBy           string
    limit             int32
    allowFilter       bool
    refSlice          any
    groupKeyGetter    any
    groupByHandler    any
}
```

### 2. Add fluent methods on `TableStruct`

Add methods:

```go
func (e *TableStruct[T, E]) GroupKeyGetter(handler func(record *E) string) *T
func (e *TableStruct[T, E]) GroupByHandler(handler func(newRecord *E, groupedRecords *[]E)) *T
```

### 3. Keep type assertions in one place

Inside `execQuery` or `selectExec`, cast the stored handlers once:

```go
groupKeyGetter, _ := tableInfo.groupKeyGetter.(func(*E) string)
groupByHandler, _ := tableInfo.groupByHandler.(func(*E, *[]E))
```

If only one handler is set, return a descriptive error.

## Integration Point

Primary integration point:

- `backend/db/select.go`
- function `scanSelectQueryRows`

Current append:

```go
*refRecords = append(*refRecords, *record)
```

Replace the direct append with:

- normal append when grouping is disabled
- grouped bucket update when grouping is enabled

## Fanout Query Handling

`selectExec` can execute multiple generated queries in parallel and merge them later.

Grouping must not use one shared map across goroutines.

Plan:

1. Each query branch builds its own local `map[string][]T`.
2. After `errgroup.Wait()`, merge those grouped maps in `selectExec`.
3. Flatten once into `recordsGetted`.

### Rationale

- avoids lock contention
- avoids shared mutable state during concurrent scans
- fits the current fanout merge design

## Ordering Rules

Define and document the first version clearly:

- DB `ORDER BY` still applies to scanned rows.
- Grouped output order is bucket-flatten order, not guaranteed DB row order across collisions.

If stable order is needed later, add:

```go
groupOrder []string
```

to track first-seen keys.

## `Limit()` Semantics

Keep current behavior:

- `LIMIT` applies to raw scanned rows from Scylla.
- Grouping happens after scanning.

This must be documented because grouped output count may be less than the query limit.

## Validation Rules

Return descriptive errors for:

- `GroupKeyGetter` set without `GroupByHandler`
- `GroupByHandler` set without `GroupKeyGetter`
- empty key returned by `GroupKeyGetter` if the caller is expected to provide a valid grouping key

## Minimal First Version

Scope the first implementation to the smallest useful behavior:

- support grouping only for standard `Query(...).Exec()`
- support in-memory grouping after row decode
- support fanout queries
- no streaming API changes
- no database-side aggregation
- no special compatibility layer

## Example Usage

```go
query := db.Query(&result.Movimientos)

query.
    GroupKeyGetter(func(record *negocioTypes.AlmacenMovimiento) string {
        return fmt.Sprintf("%d|%d", record.ProductoID, record.CreatedBy)
    }).
    GroupByHandler(func(newRecord *negocioTypes.AlmacenMovimiento, groupedRecords *[]negocioTypes.AlmacenMovimiento) {
        if len(*groupedRecords) == 0 {
            *groupedRecords = append(*groupedRecords, *newRecord)
            return
        }

        (*groupedRecords)[0].Cantidad += newRecord.Cantidad
        (*groupedRecords)[0].SubCantidad += newRecord.SubCantidad
    }).
    EmpresaID.Equals(req.Usuario.EmpresaID).
    WarehouseID.Equals(almacenID).
    Fecha.Between(fechaInicio, fechaFin).
    OrderDesc().
    Limit(1000)
```

## Implementation Sequence

1. Extend `TableInfo` with grouping handlers.
2. Add `GroupKeyGetter` and `GroupByHandler` fluent methods.
3. Add grouped scan accumulator types.
4. Update `scanSelectQueryRows` to optionally group instead of append.
5. Update `selectExec` to merge grouped fanout results.
6. Add debug logs around grouping decisions and bucket sizes.
7. Add docs to `backend/docs/ORM_DATABASE_QUERY.md`.

## Non-Goals

- SQL-like aggregation functions in Scylla
- replacing existing slice-based query API
- automatic numeric summation by column name
- backwards compatibility wrappers beyond the two requested methods
