# PLAN — Shared `db` package for the Scylla + Dynamo ORMs

**Goal:** business code declares tables and runs queries against a single
backend-agnostic `db` package (`db.TableSchema`, `db.Col`, `db.TableStruct`,
`db.Query`, `db.Table`), so ScyllaDB and DynamoDB become interchangeable
drivers behind it.

**Scope of this plan:** phases 0–3. The `db` package, the `ScyllaTable` split,
and the app-wide `scylla.*` → `db.*` rename. `dynamo` is left untouched and
keeps compiling; rewriting it onto `db.Driver` is a separate follow-up plan
(sketched in §7 so the interfaces we design now are the right shape for it).

---

## 1. Current state

| | `scylla` (root module `github.com/ivanjoz/genix-orm`) | `dynamo` (own module `…/genix-orm/dynamo`) |
|---|---|---|
| Size | ~11k lines, 33 files | ~2.5k lines, 11 files |
| Schema type | `TableSchema` (20 fields) | `Schema` (7 fields) |
| Compiled table | `ScyllaTable` (35 fields) | `tableMeta` (7 fields) |
| Column DSL | `Col[T,E]`, `ColSlice[T,E]`, `Coln`, `Cols()` | its **own** `Col[T,E]`, `Coln`, `Keys()` |
| Embedded marker | `TableStruct[T,E]` | its **own** `Model[T,E]` |
| Accessors | `columnInfo` + `compileFastAccessors` (746 lines) | its **own** `colAccessor` (6 closures) |
| Query API | `Query(&slice).Col.Equals(v).Exec()` | `Repo.Query().Eq(col, v).Exec(&out)` |
| App usage | 84 files import it | **zero** — no callers yet |

Two independent ORMs with parallel, incompatible copies of the same four
concepts. Neither shares a line of code.

**Facts that shape the plan:**

1. `dynamo` is a separate Go module *on purpose* — commit `e763ec7` "Split
   dynamo into its own module so its AWS deps stay out of scylla consumers."
   Any shared package must not re-couple gocql and the AWS SDK.
2. The skills and docs **already document `db.TableSchema` / `db.Col` /
   `db.TableStruct`** (`.claude/skills/create-database-tables`,
   `delta-cache-api`, `fetch-record-by-id-api`, `static-project-validation`).
   The code drifted to `scylla.` during the submodule extraction (`85018a36`).
   This rename re-syncs code with its own documentation.
3. The app-facing surface is small — ~55 identifiers, and ten of them are 95%
   of usage: `Col` (483), `Cols` (131), `Query` (118), `TableStruct` (83),
   `TableSchema` (82), `MakeScyllaTable` (42), `Insert` (42), `TypeView` (41),
   `ScyllaTable` (41), `RegisterTableFactory` (41).
4. **Generics are not needed at the driver boundary.** scylla's internals
   already work on `unsafe.Pointer` + `tableInfo.refSlice any`; the `[T,E]`
   type parameters exist only for façade ergonomics. They can live entirely in
   `db`, and the `db.Driver` interface can be non-generic.
5. **The main mechanical cost is field export.** Go has no cross-package
   unexported field access, embedding included. Every field that moves from
   `scylla` into `db` must be exported, and every read site updated:
   `.name` ×274, `.getRawValue` ×67, `.keyspace` ×56, `.columnsMap` ×56,
   `.getStatementValue` ×32, `.setValue` ×30, `.getValue` ×22, `.keys` ×41,
   `.columns` ×37 … ≈ 600 sites inside the ORM. All inside one package, all
   compiler-verified.

**Baseline:** `cd backend && go build ./...` is currently green (verified).
Every phase below must end green.

---

## 2. Target architecture

```
genix-orm/
  db/          NEW module — github.com/ivanjoz/genix-orm/db
               deps: xunsafe, colbin only. No gocql. No AWS SDK.
               ├─ schema.go     TableSchema, Index, Type* consts, GenericRecordSchema
               ├─ column.go     Col[T,E], ColSlice[T,E], Coln, Cols, modifiers
               ├─ colinfo.go    ColInfo, ColumnInfo, ColType, IColInfo
               ├─ accessors.go  compileFastAccessors (moved from scylla)
               ├─ metacache.go  record-type field metadata cache (moved)
               ├─ tablestruct.go TableStruct[T,E], Query, QueryIndexGroup, Table
               ├─ predicate.go  ColumnStatement, TableInfo, operators
               ├─ table.go      Table interface + TableCore struct
               ├─ driver.go     Driver interface + registry
               └─ registry.go   RegisterTableFactory / ResolveTableByName

  (root module github.com/ivanjoz/genix-orm)
  scylla/      requires db. Owns CQL generation, gocql, MVs, views, index
               groups, packed keys, text search, cache-version.
               ScyllaTable = struct { db.TableCore; <scylla-only fields> }
               Registers itself as a db.Driver.

  dynamo/      own module, requires db (phase 4, not this plan)
```

Dependency direction is strictly `scylla → db` and `dynamo → db`. `db` never
imports either. Drivers are wired in by **inversion of control**: the app
imports `scylla` for its side effect of registering, then talks only to `db`.

### 2.1 Module wiring

- `genix-orm/db/go.mod` — `module github.com/ivanjoz/genix-orm/db`, requires
  `xunsafe` + `colbin` only.
- `genix-orm/go.mod` — add `require github.com/ivanjoz/genix-orm/db v0.0.0`
  and `replace github.com/ivanjoz/genix-orm/db => ./db`.
- `backend/go.mod` — add the same require + `replace … => ./genix-orm/db`.
- `genix-orm/dynamo/go.mod` — same, added in phase 4.

gocql stays in the root module; the AWS SDK stays in `dynamo`. Neither leaks
into `db`, so the isolation from `e763ec7` is preserved.

### 2.2 `db.TableSchema` — Scylla-shaped superset

`db.TableSchema` keeps today's `scylla.TableSchema` fields verbatim (the
richer model wins; zero churn for the 82 existing `GetSchema()` bodies). Each
driver reads the subset it understands. When phase 4 lands, the dynamo driver
**projects** it:

| `db.TableSchema` | Dynamo mapping |
|---|---|
| `Partition` | `pk` component |
| `Keys` | `sk` components, `.DecimalSize(n)` → order-preserving Base64 width |
| `Index{Type: TypeView \| TypeLocalIndex, Keys}` | a GSI slot, auto-assigned `N1`…`S3` |
| `UseSequences` + `.Autoincrement(n)` | `UseAutoincrement` + `AutoincrementRandomPadding` |
| `Name` | `Entity` discriminator |
| `KeyIntPacking`, `KeyConcatenated`, `CompositeBucketing`, `UseIndexGroup`, `TextSearchColumn`, `GenericRecord`, `TypeInheritFromKey`, `TypeViewTable` | **compile-time panic** naming the table, the field and the driver |
| >5 indexes | **compile-time panic** — GSI slots exhausted |

Failing loudly at table-compile time (startup, not request time) is the point:
a table silently losing an index under a different driver is the one outcome
worse than not supporting it at all.

### 2.3 `db.Table` — the compiled-table split

`ScyllaTable`'s 35 fields split by whether they mean anything to a
non-Cassandra store:

**→ `db.TableCore` (generic, exported fields):** `Name`, `Namespace`
(was `keyspace`), `Keys`, `PartKey`, `Columns`, `ColumnsMap`, `ColumnsIdxMap`,
`KeysIdx`, `CreatedCol`, `UpdatedCol`, `UpdateCounterCol`, `AutoincrementCol`,
`AutoincrementPart`, `UseSequences`, `SequencePartCol`, `SaveCacheVersion`,
`CacheVersionFieldIndex`, `CacheVersionPartitionCol`, `CacheVersionKeyCol`,
`MaxColIdx`.

**stays in `scylla.ScyllaTable`:** `indexes`, `views`, `indexViews`,
`hasTableBackedViews`, `ViewsExcluded`, `packedIndexes`, `keyConcatenated`,
`keyIntPacking`, `compositeBucketIndexes`, `indexGroups`, `indexGroupIDs`,
`indexUpdatedTable`, `textSearchIndex`, `selectStatementCache`, `capabilities`,
`genericRecordPlan`.

```go
// db
type Table interface {
    GetName() string
    GetFullName() string
    GetColumns() map[string]IColInfo
    GetKeys() []IColInfo
    GetPartKey() IColInfo
    GetPartValue(ptr unsafe.Pointer) int64
    GetKeyValues(ptr unsafe.Pointer) []any
}
type TableCore struct { /* exported fields above */ }  // implements Table

// scylla
type ScyllaTable struct {
    db.TableCore
    // scylla-only fields
}
```

Cross-cutting consumers (`ScyllaControllerInterface`, `RegisterTableFactory`,
backup/restore, admin tooling, the generated controllers) switch to the
`db.Table` **interface**; the driver downcasts internally when it needs its own
fields. That is what lets the 41 `scylla.ScyllaTable` app references become
driver-neutral.

### 2.4 `db.Driver` — the swap point

```go
// db
type Driver interface {
    Name() string
    CompileTable(schemaProvider any) Table
    Select(refSlice any, ti *TableInfo, t Table, scan func(unsafe.Pointer) bool) error
    Insert(records any, ti *TableInfo, t Table, opts WriteOptions) error
    Update(records any, ti *TableInfo, t Table, opts WriteOptions) error
    Delete(records any, ti *TableInfo, t Table) error
    Capabilities(t Table) Capabilities
}

func Use(d Driver)     // called by scylla.Register() / dynamo.Register()
func Active() Driver
```

Non-generic on purpose (per fact #4). `db.TableStruct[T,E].Exec()` resolves
`E`'s record type via the metadata cache, hands the driver a `refSlice any` and
`unsafe.Pointer`-based accessors, and the driver appends rows through them —
exactly what `selectExec` already does internally via
`(tableInfo.refSlice).(*[]E)`.

`db.Capabilities` lets app code ask what the active driver supports rather than
assuming (e.g. `db.Active().Capabilities(t).GroupedIndexes`).

---

## 3. Phase 0 — `db` module skeleton, pure-data types

Move only types with no behaviour and no backend coupling. `scylla` keeps
`type X = db.X` aliases, so **nothing outside the ORM changes and the build
stays green**.

**Move to `db`:**
- `ColumnStatement` + `GetValue()`, `TableInfo`
- `Coln`, `Cols()`, `ColumnSetInfo`, `ColGetInfoPointer`
- `ColInfo` → `db.ColInfo`, `columnInfo` → `db.ColumnInfo`, `colType` →
  `db.ColType` (fields exported)
- `IColInfo`
- `TableSchema`, `Index`, `TypeGlobalIndex` / `TypeLocalIndex` /
  `TypeInheritFromKey` / `TypeView` / `TypeViewTable`, `indexTypes`
- `GenericRecordSchema`, `IDCacheVersion`, `RecordGroup[T]`, `QueryCapability`
- the interface set: `TableSchemaInterface`, `TableBaseInterface`,
  `TableBaseInterfaceWithCounter`, `TableInterface`, `TableQueryInterface`,
  `TableStructInterfaceQuery`

**Naming decisions to make here (they are load-bearing):**
- `colInfo.Name` is a field and `Coln.GetName()` a method — fine, no clash.
- `columnInfo.decimalSize` → **`DecimalDigits`**, *not* `DecimalSize`, because
  `Col[T,E].DecimalSize(n)` is an existing public method and Go forbids a field
  and method sharing a name on the same type. Same treatment for
  `autoincrementRandSize` → `AutoincrementRandDigits` (vs the
  `Autoincrement(n)` method).
- `colType.ColType` (the CQL string, e.g. `set<text>`) → **`DBType`**, and it
  becomes driver-owned: `db` carries the string, `scylla` computes it.
- `ScyllaTable.keyspace` → `TableCore.Namespace` (Dynamo has no keyspaces).

**Verify:** `cd backend && go build ./... && go vet ./...`; `cd genix-orm && go test ./...`.

---

## 4. Phase 1 — column DSL, accessors, driver registry

> **Revised after implementation.** Phase 1 split in two. **1a is done**; **1b is
> not started** and is larger than this plan originally claimed.
>
> The claim in step 5 below — that dropping the `[T,E]` type parameters from
> `selectExec`/`Insert`/`Update` is "mostly deleting type parameters, not
> rewriting logic" — is **wrong**. Reading the code shows the record type flows
> deeper than assumed: `executeBoundSelectQueries` keeps a
> `recordsMap map[int]*[]E`, forks one `*[]E` per parallel bound statement, and
> merges them back in declaration order; `assignCacheVersionsAfterSelect` also
> takes `*[]E`. Only `new(E)` and `append` are genuinely typed per row — the scan
> itself already works through `unsafe.Pointer` and `IColInfo.SetValue` — but the
> fork/merge plumbing is typed throughout.
>
> **1a (done):** everything that does not touch execution. `Col`, `ColSlice`,
> `TableStruct`'s state layer, `InitStructTable`, the record metadata cache, tag
> parsing, `ToSnakeCase`, `db.Table` (interface), and the `Codec` seam extended
> with the two collection-option hooks. `scylla.TableStruct` now embeds
> `db.TableStruct` and adds only the methods needing a live connection
> (`Exec`, `ExecScan`, `Insert*`, `Update*`, `MakeScyllaTable`).
>
> **1b — SUPERSEDED.** The `RowSink` design below was dropped. It traded type
> safety for a non-generic driver interface, and that trade turned out to be
> unnecessary: Go permits **generic interfaces**, so `Executor[TableT, RecordT]`
> keeps the record type all the way into the driver. The driver's methods stay
> generic, which means `select.go`, `select_grouped.go` and `cache_version.go`
> need **no changes at all** — `scylla.Exec` just calls the existing generics.
>
> The replacement design (see §4b) has zero erasure, zero call-site churn, and
> supports both runtime driver selection and two drivers live at once. The
> RowSink text is kept below only to record why it was rejected.
>
> **1b (rejected):** the execution boundary, via a `RowSink` owned by `db`:
>
> ```go
> type RowSink interface {
>     NewRow() unsafe.Pointer      // replaces new(E)
>     Commit(ptr unsafe.Pointer) bool  // replaces append; false = handler consumed it
>     Fork() RowSink               // one branch per parallel bound statement
>     Merge(branches []RowSink)    // fold branches back in order
>     Each(func(unsafe.Pointer))   // for assignCacheVersionsAfterSelect
>     Len() int
> }
> ```
>
> `db` implements it generically over `E` (`sliceSink[E]`), so the driver never
> needs the record type and `Driver.Select/Insert/Update` become plain interface
> methods. Scope: `select.go`, `select_helpers.go`, `select_grouped.go`,
> `cache_version.go`, `merge.go`, `insert-update.go` — the ORM's hottest paths.
> Per-row cost becomes two interface calls in place of a direct `new` + `append`;
> allocations are unchanged.
>
> **Before starting 1b**, add behavioural coverage the current suite lacks: the
> existing tests guard allocation counts and planning, not fork/merge ordering or
> dedup across parallel statements. A silent reordering would pass today.

1. Move `Col[T,E]`, `ColSlice[T,E]` and all modifiers (`DecimalSize`, `Int32`,
   `Autoincrement`, `IsWeek`, `StoreAsWeek`, `CompositeBucketing`, `Sum`,
   `Avg`, `Max`) and all predicate methods (`Equals`, `In`, `Contains`,
   `GreaterThan`, `GreaterEqual`, `LessThan`, `LessEqual`, `Between`,
   `Exclude`) into `db`. These only append `ColumnStatement` — pure data, no
   backend knowledge.
2. Move `TableStruct[T,E]` into `db`, including `initStructTable`,
   `Select`/`Exclude`/`GroupBy`/`Limit`/`OrderDesc`/`AllowFilter`/
   `IncludeCachedGroup`, and the `Insert`/`Update`/`InsertOne`/`UpdateOne`/
   `UpdateExclude` shims.
3. Move the accessor engine: `compileFastAccessors` and the
   `getValue`/`getRawValue`/`getStatementValue`/`setValue`/`getValueString`/
   `fieldsEqual` closures (`reflect_accessors.go`), plus
   `getOrBuildStructFieldMetadata` and the record-type metadata cache
   (`table_cache.go`). Keep the CQL-specific fallbacks (`makeScyllaValue`, the
   `colbin`/blob encoding for `Type == 9`) in `scylla` behind a
   driver-supplied hook on `ColumnInfo`.
4. Add `db/driver.go` (registry) and `db/table.go` (`Table` + `TableCore`).
5. `scylla.Register()` implements `db.Driver`, wrapping the existing
   `selectExec` / `Insert` / `Update` entry points. Refactor their signatures
   from `[T,E]` generics to `(refSlice any, …)` — their bodies already work
   through `refSlice` and `unsafe.Pointer`, so this is mostly deleting type
   parameters, not rewriting logic.
6. `TableStruct.Exec()` / `Insert` / `Update` route through `db.Active()`.
7. `scylla` keeps aliases for everything moved, so app code is still untouched.

**Risk:** this is the one phase with real behavioural risk — the generic →
`any` signature change on `selectExec`/`Insert`/`Update`. Mitigation: the ORM's
own test suite (`group_by_test`, `counter_test`, `update_counter_test`,
`memory_usage_test`, `cache_version_generic_test`, `select_helpers_test`,
`select_debug_test`, `select_grouped_test`) must pass unchanged, and
`memory_usage_test` specifically guards against new per-row allocations.

**Verify:** `go test ./...` in `genix-orm`, `go build ./...` in `backend`.

---

## 4b. Phase 1b (adopted) — generic `Executor`, driver as a typed value

The driver is a **generic interface value**, not a registry entry and not a bare
type parameter. Each table declares a *default* driver as a type argument so
existing call sites keep inferring everything; a query can override it at runtime.

```go
// db — generic, so an Executor[ProductTable, Product] can only ever be handed a *[]Product
type Executor[TableT any, RecordT any] interface {
    Name() string
    Select(schema *TableT, ti *TableInfo, dst *[]RecordT, scan func(*RecordT) bool) error
    SelectGrouped(schema *TableT, ti *TableInfo) error
    Insert(records *[]RecordT, exclude ...Coln) error
    Update(records *[]RecordT, include ...Coln) error
    CompileTable(schema *TableT) Table
}

type TableStruct[D Executor[TableT, RecordT], TableT ..., RecordT ...] struct {
    exec Executor[TableT, RecordT] // nil = use D, the declared default
    ...
}
func (s *TableStruct[D, TableT, RecordT]) Via(exec Executor[TableT, RecordT]) *TableT
```

```go
// scylla — one generic alias keeps every table declaration unchanged
type Exec[TableT ..., RecordT ...] struct{}
type TableStruct[TableT any, RecordT any] = db.TableStruct[Exec[TableT, RecordT], TableT, RecordT]
```

**Verified by compiling probes, not by reasoning:**

| Property | Result |
|---|---|
| `Query(&products)` with no explicit type args | ✓ driver inferred from the record's `GetExecutor() D` |
| Table declarations keep `TableStruct[XTable, X]` | ✓ via the generic alias (Go 1.24+; repo is on 1.26) |
| Switch driver for the whole app | ✓ one alias line |
| Two drivers, one record type, one binary | ✓ two `Executor[…]` variables + `.Via()` |
| Runtime selection from config | ✓ ordinary interface assignment |
| Wrong-typed driver | ✓ compile error (`have Select(*[]Almacen…) want Select(*[]Product…)`) |
| Functions losing type parameters | **0** |
| Changes needed in select.go / select_grouped.go / cache_version.go | **none** |

Cost: one interface dispatch per *query* (not per row), and each driver
implements the `Executor` methods as thin wrappers over its existing generics.

## 5. Phase 2 — split `ScyllaTable`, neutralize the registry

1. Export the ~20 generic `ScyllaTable` fields and move them into
   `db.TableCore`; embed it in `ScyllaTable`. Rename the ≈600 read sites.
   Do it **field by field with `gopls rename`**, not a blanket regex — `.name`
   ×274 also matches `viewInfo.name`, `compositeBucketIndex.name` and
   `indexGroupInfo.name`, which must **not** change.
2. Same for `columnInfo`'s accessor closures and modifier fields (~200 sites).
3. `RegisterTableFactory(name string, f func() db.Table)` and
   `ResolveTableByName(name) (db.Table, error)` move to `db/registry.go`.
4. `ScyllaControllerInterface` → `db.Controller` with `GetTable() db.Table`;
   `ScyllaController[T,E]` stays in `scylla` and implements it. `CSVResult`
   moves to `db`.
5. `MakeScyllaTable[T,E]()` → `db.MakeTable[T,E]() db.Table`, delegating to
   `db.Active().CompileTable`. `MakeSchema[T,E]()` moves as-is.
6. Keep `scylla.MakeScyllaTable` and `scylla.ScyllaTable` as aliases through
   this phase so the build stays green before phase 3 renames the callers.

**Verify:** `go test ./...`; `go build ./...`; run
`static-project-validation` (`cd scripts && go run . check_tables`).

---

## 6. Phase 3 — app-wide rename and doc re-sync

1. Mechanical rename across `backend/` (84 files, ~1100 sites):
   `"github.com/ivanjoz/genix-orm/scylla"` → aliased import of
   `github.com/ivanjoz/genix-orm/db`, and `scylla.X` → `db.X`. Per-identifier
   `gopls rename` where a name changed (`ScyllaTable` → `Table`,
   `MakeScyllaTable` → `MakeTable`, `ScyllaControllerInterface` →
   `Controller`), plain `sd`/`sed` for the package qualifier.
2. Sites that must **keep** importing `scylla` directly, because they are
   genuinely Scylla-specific: `scylla.MakeScyllaConnection`,
   `scylla.SetScyllaConnection`, `scylla.ConnParams`, `scylla.Init`,
   `scylla.CreateKeyspaceIfNotExists`, `scylla.DeployScylla`,
   `scylla.QueryExec` (raw CQL), and the `scylla.Register()` side-effect import
   in `backend/main.go` + `exec/init.go`.
3. Delete the transitional aliases from `scylla`.
4. Update the generator: `scripts/controllers/controllers_generator.go:355`
   emits `db.RegisterTableFactory(%q, func() db.Table { return db.MakeTable[%s]() })`.
   Regenerate `backend/exec/controllers.generated.go`.
5. Update `scripts/validation/check_tables.go` and
   `scripts/table/create_edit_table.go` (they parse and emit `scylla.Col` /
   `scylla.TableSchema` literals today).
6. Docs — these already say `db.`, so this closes the drift:
   `.claude/skills/create-database-tables`, `delta-cache-api`,
   `fetch-record-by-id-api`, `static-project-validation`; plus
   `backend/docs/ORM_DATABASE_QUERY.md`, `backend/docs/CREATE_API_HANDLERS.md`,
   `scripts/CREATE_EDIT_TABLE.md`, `scripts/CHECK_TABLES_SCRIPT.md`,
   `scripts/GENERATE_CONTROLLERS.md`, `AGENTS.md` (§Key Documentation),
   `genix-orm/scylla/ORM_INTERNALS.md`, and a new `genix-orm/db/README.md`
   describing the driver contract.

**Verify:** `go build ./...`; `go vet ./...`; `go test ./...` in both modules;
`scripts` validation green; `rg 'scylla\.' backend/ --glob '!genix-orm/**'`
returns only the connection/deploy/raw-CQL sites from step 2.

---

## 7. Follow-up (not this plan) — phase 4/5, the dynamo rewrite

Recorded so the phase 1–2 interfaces are shaped correctly:

- Delete dynamo's duplicate `Col`/`Coln`/`Keys`/`Schema`/`Model`/`colAccessor`/
  `populateColumnNames`/`buildTableMeta` — roughly 600 of its 2.5k lines.
- `dynamo.tableMeta` becomes `struct { db.TableCore; entity string; slots []indexMeta; … }`.
- Implement `db.Driver`: `CompileTable` performs the §2.2 projection with loud
  panics; `Select` maps `ColumnStatement`s onto Query/Scan with `Eq`/`Gt`/
  `Between`/`BeginsWith`; `Insert`/`Update` map onto `Put`/`PutMany`.
- Keep `Repo[T,E]` as a thin dynamo-native convenience layer, or drop it in
  favour of `db.Query`.
- Add a conformance suite: one shared table definition, the same
  insert/query/update/delete assertions run against both drivers, plus
  `Capabilities` assertions for the features dynamo cannot express.

---

## 8. Honest assessment of the cost

**What genuinely gets shared:** the schema DSL (~700 lines), the accessor and
metadata engine (~900 lines), the predicate AST and query-state builder
(~400 lines), the table registry and controller contract (~150 lines). That is
the ~2.1k lines both ORMs currently duplicate or would duplicate.

**What cannot be shared, and should not be:** CQL/MV generation (`deploy.go`
970 lines, `index_view_compile.go` 813, `view_tables.go` 373), gocql
connection handling, packed-index and index-group planning, text search, and
on the dynamo side the order-preserving key encoding and GSI slot allocation.
Roughly 60% of `scylla` stays Scylla-specific. That is the correct outcome —
the abstraction is at the *schema and record* layer, not the storage layer.

**Where interchangeability will be real:** single-partition equality and range
reads, autoincrement IDs, `Status`/`Updated` delta-cache views, insert/update
of whole records. **Where it will not be:** anything using `KeyIntPacking`,
`UseIndexGroup`, `TypeInheritFromKey`, `CompositeBucketing`, text search, or
`GenericRecord`. A quick audit of current `GetSchema()` bodies before phase 4
will say what fraction of the ~82 tables is portable; my expectation from
reading them is roughly half.

**Biggest risk:** phase 1's generic → `any` signature change on
`selectExec`/`Insert`/`Update`. Second biggest: the ~800 mechanical rename
sites in phases 2–3, where a careless regex silently retargets an unrelated
`.name`. Both are compiler-caught or test-caught, neither is subtle at runtime
— but phase 2 must be done with `gopls rename`, one field at a time.

**No backwards compatibility is kept.** Per `AGENTS.md`, transitional aliases
exist only *within* a phase to keep the build green, and phase 3 deletes them.
