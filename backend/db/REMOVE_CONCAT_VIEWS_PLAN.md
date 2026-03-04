# Remove `ConcatI64` / `ConcatI32` and Migrate to `DecimalSize()` Plan

## Goal
Eliminate `View.ConcatI64` and `View.ConcatI32` from the ORM API and make range/int-packed views use per-column metadata (`Col.DecimalSize()` + optional `Col.Int32()`) in `View.Cols`, similar to current packed index behavior.

## Scope
- Refactor schema API (`db.View`) to remove `ConcatI64` / `ConcatI32`.
- Refactor view-building internals in `backend/db/reflect.go`.
- Migrate all schema usages in project types and demos.
- Update documentation examples that still reference `ConcatI64` / `ConcatI32`.

## Current State (Observed)
- `View` currently exposes:
  - `ConcatI64 []int8`
  - `ConcatI32 []int8`
- Range view behavior is implemented in `backend/db/reflect.go` and keyed by `len(viewCfg.ConcatI64) > 0 || len(viewCfg.ConcatI32) > 0`.
- Existing `DecimalSize()` metadata is already propagated from `db.Col` into `columnInfo` during `initStructTable`.
- Current schema usages of `ConcatI64` / `ConcatI32` are in:
  - `backend/types/generales.go`
  - `backend/types/productos.go`
  - `backend/exec/demo2.go`
  - docs examples

## Proposed New Contract for Views
- A view with `len(View.Cols) == 1` remains unchanged (simple view).
- A view with `len(View.Cols) > 1` and any `Cols[i].DecimalSize() > 0` (for `i > 0`) is treated as an int-packed/range view.
- Radix definition:
  - Last column is implicit radix `0`.
  - For each previous column, radix comes from `DecimalSize()` values of subsequent columns (same accumulation logic currently used).
- Storage type:
  - Default packed view virtual column type: `int64`.
  - If the first packed column (`Cols[0]`) has `.Int32()`, final virtual value is cast to `int32` (mirrors packed index signal style).

## Implementation Plan

### 1. API change in `db.View`
- File: `backend/db/main.go`
- Remove fields:
  - `ConcatI64 []int8`
  - `ConcatI32 []int8`
- Keep `Cols`, `KeepPart`, `UseHash`, `Project` unchanged.
- Add/adjust comments so view packing is documented as `DecimalSize()`-driven.

### 2. Refactor range-view detection and radix extraction
- File: `backend/db/reflect.go`
- Replace `isRangeView` detection:
  - From `ConcatI64/ConcatI32` presence
  - To checking packing metadata in `viewCfg.Cols` (`decimalSize` and `useInt32Packing`).
- Build radix array from columns metadata rather than `viewCfg.Concat*`.
- Preserve existing validation behavior:
  - numeric-only column types
  - max radix guard (`<= 17`)
  - min 2 columns for packed/range behavior
- Keep `KeepPart=true` auto behavior for packed/range views.

### 3. Define validation rules for new syntax
- File: `backend/db/reflect.go`
- Enforce packed view constraints (analogous to packed indexes):
  - If packed view detected, all columns except the last must have explicit `DecimalSize()` where needed by formula.
  - Last column must not require explicit decimal size (implicit `0`).
  - `.Int32()` allowed only as a storage hint (recommended on first column).
- Panic messages should clearly mention `View.Cols + DecimalSize()` expected usage.

### 4. Migrate schema declarations
- Files:
  - `backend/types/generales.go`
  - `backend/types/productos.go`
  - `backend/exec/demo2.go`
- Convert each old declaration:
  - Before: `{Cols: []db.Coln{a, b}, ConcatI32: []int8{2}}`
  - After: `{Cols: []db.Coln{a.Int32(), b.DecimalSize(2)}}`
- Multi-column `ConcatI64` cases:
  - Map each radix slot to the corresponding following column `DecimalSize()`.
  - Keep equivalent numeric semantics from prior accumulated radix math.

### 5. Documentation updates
- Files:
  - `backend/docs/ORM_DATABASE_QUERY.md`
  - `backend/db/ORM_INTERNALS.md` (range/radix view section)
- Replace `ConcatI64/ConcatI32` examples with `DecimalSize()` + `.Int32()` examples.
- Add one explicit migration snippet “old -> new”.

### 6. Build and behavior verification
- Run compile/tests from backend module (at least):
  - `go test ./...`
- Add focused manual verification checklist:
  - packed view with int64 output
  - packed view with `.Int32()` output
  - simple view unaffected
  - bad config panics (non-numeric cols, invalid/missing decimal sizes)
  - query range behavior remains equivalent for migrated schemas

## File Impact Summary
- Core ORM:
  - `backend/db/main.go`
  - `backend/db/reflect.go`
- Schema consumers:
  - `backend/types/generales.go`
  - `backend/types/productos.go`
  - `backend/exec/demo2.go`
- Docs:
  - `backend/docs/ORM_DATABASE_QUERY.md`
  - `backend/db/ORM_INTERNALS.md`

## Risks and Mitigations
- Risk: Radix mapping mismatch during migration for 3+ columns.
  - Mitigation: Convert with explicit mapping table per view and validate expected packed values with sample tuples.
- Risk: Silent behavior drift from `int64` to `int32`.
  - Mitigation: Require explicit `.Int32()` hint for int32 output and keep int64 default.
- Risk: Existing schemas relying on implicit old concat semantics.
  - Mitigation: Panic with migration-oriented error messages when old pattern cannot be inferred.

## Open Decision Needed Before Coding
1. Should packed views default to `int64` unless `.Int32()` is explicitly set on the first column? (This is the safest and most backward-compatible interpretation.)
2. For packed views, do you want strict validation parity with packed indexes (first slot no `DecimalSize`, trailing slots required), or a more permissive mode that infers missing sizes?
