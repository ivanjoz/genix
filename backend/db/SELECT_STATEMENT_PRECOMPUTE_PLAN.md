# Select Statement Precompute Plan

## Goal

Reduce the cost of repeated `SELECT` execution in `backend/db/select.go` by caching the **query shape**, not the query values.

The cache should skip repeated planner work for identical logical queries such as:

```go
query.EmpresaID.Equals(companyID).StatusTrace.Equals(statusTrace)
query.Updated.GreaterEqual(updated)
```

For the example above, the cached artifact should already know:

- which source is used: base table, view, packed view, or composite bucket plan
- which predicates belong to the selected source
- which predicates remain as direct `WHERE` filters
- whether post-filter or deduplication is required
- how to assemble the final CQL quickly from runtime values
- whether one logical query binds into one CQL statement or many

## Current Behavior

`selectExec()` currently does these steps on every call:

1. Build selected columns and `SELECT ... FROM ...` prefix.
2. Copy `tableInfo.statements` into local planner state.
3. Run planner logic:
   - native group-by planning
   - composite bucket planning
   - `MatchQueryCapability()`
   - key-concat / key-int-packing fallback transforms
4. Convert `ColumnStatement` values into string clauses with `buildRemainingWhereClauses()`.
5. Build the full CQL string.
6. Execute with `queryValues := []any{}`.

Important observation:

- The hot planning work is already separated from row scanning.
- The current select path mostly interpolates values into strings.
- `scanSelectQueryRows()` already accepts `queryValues`, so the select path can evolve toward bind placeholders without changing the scan API first.
- some view/index `getStatement()` implementations already expand one logical query into many physical `WHERE` clauses:
  - packed indexes fan out over prefix `IN` groups in [packed_indexes.go](/home/ivanjoz/projects/genix/backend/db/packed_indexes.go)
  - radix/range views can emit multiple `WHERE` clauses from prefix groups in [reflect.go](/home/ivanjoz/projects/genix/backend/db/reflect.go)

## Recommendation

Do **not** cache the raw final CQL string as the first step.

Instead, cache a **compiled select shape** that stores:

- resolved source table/view name
- preselected scan columns / select expressions
- a tokenized `WHERE` template
- runtime variable slots
- flags for post-filter, dedup, and allow-filter behavior
- optional fan-out expansion metadata when one logical query becomes many physical statements

This keeps the cache useful even when values change every call.

## Scope For Phase 1

Limit the first implementation to standard `SELECT` queries without:

- `GroupBy()`
- composite bucket fan-out
- dynamic packed/radix range expansion into many statements

Reason:

- those paths are the most correctness-sensitive
- they already create multiple physical statements
- they should be added only after the single-statement cache is stable

Phase 1 should still support:

- base-table queries
- direct view/index routing chosen by `MatchQueryCapability()`
- key-concat rewrites
- key-int-packing rewrites
- include/exclude/select column projections
- post-filter flags for packed indexes

## Two Complexity Stages

The compiled path should be designed as two explicit stages.

### Stage 1: Single-Statement Compilation

Support only shapes that bind into exactly one final CQL statement.

Examples:

- base table equality/range query
- direct view with one generated `WHERE`
- hash view that compiles to one `IN (...)`
- key-concat and key-int-packing rewrites that still end as one statement

### Stage 2: Multi-Statement Compilation

Support shapes where one logical query expands into multiple physical statements based on runtime values.

Current examples in the codebase:

- packed indexes that fan out over prefix `IN` combinations in [packed_indexes.go](/home/ivanjoz/projects/genix/backend/db/packed_indexes.go)
- radix/range views that fan out over value groups in [reflect.go](/home/ivanjoz/projects/genix/backend/db/reflect.go)
- composite bucket plans in [select.go](/home/ivanjoz/projects/genix/backend/db/select.go)

The key point is:

- the **shape** is still precomputable
- the **statement count** is not always fixed

So the cache should precompute the expansion algorithm, not only a final token slice.

## Shape Hash

The cache key must represent the **statement shape**, excluding values.

Recommended inputs for the shape hash:

- base table name
- selected projection shape
- each predicate as:
  - column index
  - normalized operator
  - cardinality kind: scalar / `IN(n)` / `BETWEEN`
- presence of:
  - `ORDER BY`
  - `LIMIT`
  - `ALLOW FILTERING`
  - post-filter mode
  - group-by mode

Important:

- do not hash only the original fluent predicates
- hash the **normalized planner input**
- if the planner rewrites to key-concat or key-int-packing, the compiled result must represent the rewritten shape

## Proposed Compiled Types

`select_helpers.go` should hold a compiled artifact closer to this model:

```go
type selectVariableKind int8

const (
	selectVariableScalar selectVariableKind = iota
	selectVariableInList
	selectVariableBetweenFrom
	selectVariableBetweenTo
)

type selectVariableSlot struct {
	ID          int16
	ColumnIdx   int16
	Operator    string
	ValueIndex  int16
	VariableKind selectVariableKind
}

type SelectStatement struct {
	Hash                   uint64
	SourceTableName        string
	SelectExpressions      []string
	ScanColumns            []selectScanColumn
	WhereTokens            []int16
	VariableSlots          []selectVariableSlot
	UsesPostFilter         bool
	PostFilterStatements   []ColumnStatement
	RequiresDeduplication  bool
	OrderByFormat          string
	LimitEnabled           bool
	AllowFilteringEnabled  bool
}
```

Notes:

- prefer storing `SourceTableName string`, not `*View`
- prefer storing planner output, not schema input
- keep `ColumnStatement` only where exact post-filter logic still needs it
- fix the typo `syllaTable` -> `scyllaTable` before the helper grows further

## Token Strategy

The current token idea in `select_helpers.go` is valid, but it should be constrained.

Use tokens only for **stable SQL structure**:

- `SELECT `
- `, `
- `FROM `
- ` WHERE `
- ` AND `
- `>=`
- `<=`
- `=`
- `IN`
- `(`
- `)`
- `ALLOW FILTERING`

Use variable slots for dynamic values.

Do not store runtime values in `map[int16]any` inside the compiled statement.

That map would mix immutable compiled state with per-call data and remove most of the benefit.

For Stage 2, tokenization alone is not enough.

The compiled artifact also needs precomputed expansion metadata such as:

- which variable slots are scalar
- which variable slots are repeatable groups
- which variable slots participate in cartesian fan-out
- which clause template is reused per expanded group

## Build Flow

Add a new builder with explicit stages:

1. `buildSelectShape(tableInfo, scyllaTable)`  
   Extract a normalized, value-free query shape.
2. `compileSelectStatement(shape, tableInfo, scyllaTable)`  
   Run planner logic once and emit a compiled statement template.
3. `bindSelectStatement(compiled, tableInfo)`  
   Convert current runtime values into `queryStr` and optionally `queryValues`.

For Stage 2, `bindSelectStatement()` should be allowed to return many statement strings:

```go
func bindSelectStatement(compiled SelectBinder, tableInfo *TableInfo) []BoundSelectStatement
```

Where each bound result contains:

- final `WHERE` clause or full query string
- bind values for that concrete statement
- post-filter flags inherited from the compiled plan

This separation matters:

- shape building answers cache identity
- compilation answers planner reuse
- binding answers execution reuse

## Integration In `selectExec`

Recommended integration order:

1. Keep current `selectExec()` behavior as fallback.
2. Add a small cache lookup near the start of `selectExec()`.
3. On hit:
   - bind runtime values into the compiled template
   - execute without running planner selection again
4. On miss:
   - run the existing planner
   - compile the result into `SelectStatement`
   - store in cache
   - execute

The cache should be attached to immutable table metadata, not global mutable query state.

Best location:

- per-`ScyllaTable` cache map keyed by shape hash

Reason:

- query shapes are table-local
- metadata lifetime already matches `ScyllaTable`
- this avoids a noisy process-wide cache keyed by string table names

## Binding Strategy

Phase 1 can still emit a final string directly if that is the fastest safe change.

But the compiled format should be designed so Phase 2 can move to placeholders:

- scalar: `column = ?`
- range: `column >= ?`
- between: `column >= ? AND column <= ?`
- `IN`: `column IN (?, ?, ?)`

Recommended approach:

- in Phase 1, `bindSelectStatement()` renders values with existing quoting rules
- in Phase 2, `bindSelectStatement()` fills `queryValues` and reuses `scanSelectQueryRows()`

This avoids blocking the cache work on a full prepared-statement migration.

## Multi-Statement Precompute Model

The placeholder `Compute(variables []any) []string` in [select_helpers.go](/home/ivanjoz/projects/genix/backend/db/select_helpers.go) is directionally correct, but it needs one more abstraction level.

Do not model every cached entry as one `SelectStatement`.

Instead, model a compiled select plan that can bind into one or many statements:

```go
type BoundSelectStatement struct {
	QuerySuffix string
	QueryValues []any
}

type CompiledSelectPlan interface {
	Hash() uint64
	Bind(tableInfo *TableInfo) []BoundSelectStatement
}

type CompiledSingleSelect struct {
	Template SelectStatement
}

type CompiledFanoutSelect struct {
	BaseTemplate       SelectStatement
	ExpansionVariables []selectExpansionVariable
	Expand             func(tableInfo *TableInfo) []BoundSelectStatement
}
```

Rationale:

- `CompiledSingleSelect` keeps the simple path minimal
- `CompiledFanoutSelect` precomputes how expansion works without storing runtime values
- the caller of `selectExec()` can keep the same outer execution loop over `queryWhereStatements`

## What To Precompute For Fan-Out

For multi-statement views and indexes, precompute these immutable pieces:

- source table/view chosen by the planner
- exact source columns used by the expansion
- expansion mode:
  - prefix cartesian product
  - range split
  - bucket expansion
- fixed clause fragments before and after each variable
- whether the expansion emits:
  - many `... = value`
  - many `... >= from AND ... < to`
  - one `... IN (...)`

Examples from the current code:

- packed index:
  - precompute source column order
  - precompute which columns may fan out from `IN`
  - precompute packed-slot digits and final clause kind for the last source column
- radix/range view:
  - precompute ordered view columns
  - precompute radix slot digits
  - precompute whether the path emits `IN (...)` or multiple prefix ranges
- composite bucket:
  - precompute handled columns and virtual target column
  - bind-time still computes bucket selections from runtime range values

The expensive part you want to avoid repeating is deciding **how** to expand, not the arithmetic on the current values.

## Special Cases

These should be deferred or explicitly marked unsupported until Phase 2 or 3:

- `GroupBy()` plans from `buildNativeGroupByPlan()`
- composite bucketing from `tryBuildCompositeBucketPlan()` in Phase 1 only
- queries that fan out into multiple CQL statements in Phase 1 only
- paths where `whereStatements` length is greater than 1 in Phase 1 only

Those cases need a separate compiled type such as:

```go
type CompiledSelectPlan struct {
	Statements []SelectStatement
}
```

Do not fold those into the first implementation.

For Phase 2, bring them back under `CompiledSelectPlan` instead of extending `SelectStatement` until it mixes template state and expansion state.

## Logging

Add debug logs around the cache path:

- `select cache lookup`
- `select cache hit`
- `select cache miss`
- `select compile source selected`
- `select bind completed`

Recommended fields:

- table name
- shape hash
- source table/view name
- statement count
- uses post-filter
- uses dedup

This is necessary to validate correctness and hit rate during rollout.

## Tests

Add targeted tests before broad rollout.

Minimum test matrix:

1. Same shape, different values => same shape hash, different bound CQL.
2. Same columns, different operators => different shape hash.
3. Same predicates, different projection => different shape hash.
4. View-routed query => compiled source is the expected view.
5. Key-concat rewrite => compiled source keeps rewritten predicate layout.
6. Packed view with post-filter => compiled plan preserves `UsesPostFilter`.
7. Cache miss then hit => second execution skips planner stage.
8. Packed index with prefix `IN` => one compiled fan-out shape binds into multiple statements.
9. Radix/range view with many prefix groups => one compiled fan-out shape binds into the expected statement count.
10. Same fan-out shape, different runtime cardinality => same shape hash, different bound statement count.

Prefer unit tests around:

- shape hashing
- statement compilation
- value binding

Keep execution tests small and explicit.

## Rollout Order

1. Normalize and document the compiled select model.
2. Implement shape hashing and cache storage on `ScyllaTable`.
3. Compile only single-statement select plans.
4. Bind runtime values using the compiled template.
5. Add logs and tests for hit/miss behavior.
6. Add `CompiledFanoutSelect` for packed/radix view expansion.
7. Extend to composite bucket plans only if profiling still shows planner overhead is relevant.

## Non-Goals

Do not include these in the first pass:

- backwards compatibility for multiple competing cache formats
- prepared statement pooling
- group-by compilation
- composite bucket compilation
- cross-table global select cache

## Concrete First Patch

The first code patch should be intentionally small:

1. Replace the placeholder `SelectStatement` with a compiled-template struct.
2. Add a shape-hash function from `TableInfo` + projection metadata.
3. Add a table-local cache on `ScyllaTable`.
4. Compile only the path where `len(whereStatements) <= 1`.
5. Leave current runtime behavior unchanged for unsupported paths.

That gives you a measurable win without trying to freeze the entire planner in one refactor.

## Concrete Second Patch

After the single-statement path is stable:

1. Introduce `CompiledSelectPlan` with single-select and fan-out variants.
2. Move packed-index fan-out logic out of ad-hoc `getStatement()` string building into reusable bind helpers.
3. Do the same for radix/range fan-out views.
4. Keep `selectExec()` execution flow unchanged: it should still receive `[]BoundSelectStatement` and run them concurrently as today.
5. Only then evaluate whether composite bucket fan-out belongs in the same abstraction.
