# ORM Improvement Plan: Composite Bucketing for `CONTAINS + BETWEEN` in ScyllaDB

## 1. Context and Objective
ScyllaDB does not support range indexes over `set<int>` values; `CONTAINS` works for membership, not range.

Target query pattern:
- `DetailProductsIDs.Contains(productID)`
- `Week.Between(weekFrom, weekTo)`

Example that must work:
- Model: `backend/comercial/types/sales.go` (`HashIndexes: {DetailProductsIDs, Week.CompositeBucketing(1,4,5)}`)
- Query: `backend/exec/test_selects.go` (`DetailProductsIDs.Contains(1).Week.Between(2548, 2614)`)

Goal:
Use precomputed composite hashes with multiple bucket sizes so range queries are executed as a small set of indexed `CONTAINS` queries, then post-filtered in memory.

## 2. Current State (Important)
`CompositeBucketing` metadata exists (`columnInfo.compositeBucketing`) but is not wired yet.

Critical gap in current code:
- `backend/db/reflect.go`: `schema.HashIndexes` is only printed (`fmt.Println(indexColumns)`) and does not build virtual columns/indexes/views.

Implication:
The new approach is not active yet even if schema defines `HashIndexes` with `CompositeBucketing`.

## 3. Proposed Data Model Strategy
For a hash index config:
- `{DetailProductsIDs, Week.CompositeBucketing(1,4,5)}`

Create 3 hidden virtual `set<int>` columns:
- `zz_hb_detail_products_ids_week_b1`
- `zz_hb_detail_products_ids_week_b4`
- `zz_hb_detail_products_ids_week_b5`

Each column stores values:
- `HashInt(productID, bucketID)`
- `bucketID = week / bucketSize`

Each generated set column needs a global index on `VALUES(column)`.

Rationale:
- `b1` gives exact week granularity.
- `b4` and `b5` reduce query fan-out for broader ranges.
- Mixed bucket coverage minimizes total DB round-trips.

## 4. Write Path Plan
Files:
- `backend/db/reflect.go`
- `backend/db/insert-update.go`

### 4.1 Reflection/Schema build
When processing `TableSchema.HashIndexes`:
1. Validate shape:
- one slice column (`DetailProductsIDs` style)
- one numeric column with `CompositeBucketing(...)` (`Week` style)
2. Validate bucket sizes are unique and `>0`.
3. Register generated virtual set columns in `dbTable.columnsMap` as `IsVirtual=true`.
4. Register global indexes for each generated virtual set column (`VALUES(...)`).
5. Register internal metadata structure (new): source cols + bucket configs + generated col names.

### 4.2 Insert/Update value materialization
For each record on insert/update:
1. Read week value.
2. Iterate `DetailProductsIDs`.
3. For each bucket size, compute `bucketID := week / size` and `HashInt(productID, bucketID)`.
4. Insert hash in the corresponding virtual set (deduplicate per record).

Update safety:
- If update includes one source column of this composite index (`Week` or `DetailProductsIDs`), enforce both columns are included, same behavior as existing virtual dependency checks.

## 5. Read Path Plan (`Contains + Between`)
Files:
- `backend/db/select.go`
- `backend/db/select_compute.go` (if capability extension is needed)

### 5.1 Query detection rule
Before fallback SQL build, detect pattern:
- partition equality present
- one `CONTAINS` on source slice column
- one `BETWEEN` (or range op) on bucketed numeric column
- matching composite-bucket metadata exists

### 5.2 Coverage planner (core algorithm)
Input:
- week range `[from, to]`
- bucket sizes (descending by size)

Output:
- list of query buckets `{size, bucketID}` minimizing query count.

Selection policy:
1. Prefer larger buckets when fully inside range.
2. Allow bounded overfetch only when it reduces query count.
3. Fill uncovered boundaries with smaller buckets (`b1` fallback).

For each selected bucket:
- hash value = `HashInt(productID, bucketID)`
- SQL fragment = `zz_hb_..._bX CONTAINS <hash>`

### 5.3 Execution and merge
Leverage existing parallel multi-query path in `select.go`:
1. Build one query per selected bucket.
2. Execute concurrently with existing `errgroup` flow.
3. Merge results.
4. Deduplicate by primary key (or full key tuple if needed).
5. Final in-memory filter by exact week range (`week >= from && week <= to`).

This final filter removes overfetch artifacts.

## 6. Capability/Planner Integration
Two implementation options:

Option A (recommended first):
- Special-case in `select.go` before `MatchQueryCapability` for this exact pattern.
- Faster to ship, lower risk.

Option B (later cleanup):
- Extend `ComputeCapabilities/MatchQueryCapability` to support composite-bucket signatures and `CONTAINS` semantics.

Recommendation:
- Start with Option A for one-week delivery.
- Refactor to Option B after validating behavior/performance.

## 7. Example for the Required Test Query
Query:
- `EmpresaID = 1`
- `DetailProductsIDs CONTAINS 1`
- `Week BETWEEN 2548 AND 2614`
- Buckets: `1,4,5`

Planner should output a mixed bucket cover (fewest statements), then run queries against:
- `zz_hb_detail_products_ids_week_b4 CONTAINS HashInt(1, bucketID)`
- `zz_hb_detail_products_ids_week_b5 CONTAINS HashInt(1, bucketID)`
- plus `b1` only where needed for boundaries.

Then dedupe + exact week filter in memory.

## 8. Validation and Debug Logging Plan
Per AGENT protocol, include extensive debug logs:

At reflection/init:
- detected composite hash indexes
- generated virtual columns and bucket sizes

At write path:
- record key, week, productIDs count
- generated hashes per bucket size

At read path:
- detected optimization pattern
- selected coverage buckets
- generated SQL count and statements
- rows fetched before/after dedupe and after week filter

Tests to add/update:
1. Unit test for coverage planner (`from/to + sizes -> expected buckets`).
2. Insert test: generated virtual set values for known record.
3. Query integration test using `sale_order` case in `backend/exec/test_selects.go`.
4. Update consistency test: partial update of only `Week` or only `DetailProductsIDs` should fail with clear error.

## 9. Rollout Sequence (1 week)
Day 1:
- Implement schema/reflection wiring for `HashIndexes + CompositeBucketing`.

Day 2:
- Implement write-path materialization for generated virtual sets.

Day 3:
- Implement read-path special planner in `select.go`.

Day 4:
- Add dedupe + final week filter and full debug logs.

Day 5:
- Add tests (planner, insert/update, integration query).

Day 6:
- Benchmark query-count reduction vs `b1`-only strategy.

Day 7:
- Stabilization and cleanup.

## 10. Risks and Mitigations
Risk: hash collisions (`int32`).
- Mitigation: keep final exact week filter and, if needed, validate product membership in-memory for suspicious cases.

Risk: too many buckets for very large ranges.
- Mitigation: cap query count and optionally fallback to coarser-only plan + stronger post-filtering.

Risk: update inconsistency for virtual sets.
- Mitigation: enforce source-column co-update rule (same pattern already used for virtual dependencies).

## 11. Acceptance Criteria
The feature is complete when:
1. `HashIndexes` with `CompositeBucketing` generates real virtual columns and indexes (not only logs).
2. Insert/update populate virtual bucket hash sets correctly.
3. Query `Contains + Week.Between` uses composite bucket plan automatically.
4. Result set is correct (deduped and exact range filtered).
5. `backend/exec/test_selects.go` sale order range query runs with this strategy and no `ALLOW FILTERING` requirement.
