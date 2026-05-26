# Migrate ORM writes to prepared statements

## Why

Today the ORM has two parallel write implementations:

- **Prepared path** (safe): `Insert`, `Update`, `InsertUpdate`, `InsertUpdateInclude`, `InsertUpdateExclude` → `executeInsertUpdateBatch` → `appendInsertQueriesToBatch` / `appendUpdateQueriesToBatch` → `session.ExecuteBatch(batch)` with `?` placeholders. gocql encodes and escapes values.
- **Raw-string path** (unsafe): `UpdateExclude` → `makeUpdateStatementsBase` → `getNormalizedWriteLiteral` → `QueryExec(string)` with concatenated CQL literals. Hand-rolled string formatters lacked single-quote escaping (the bug fixed today).

The raw-string path will keep producing CQL parse failures and SQL-injection-shaped bugs every time someone touches `makeScyllaValue`, `makeStringCollectionLiteral`, or a virtual column's `getValue`. The fix is to delete the raw-string write path entirely.

## Scope

In scope (entry points that need migration):

1. **`UpdateExclude`** (`insert-update.go:785`) — production hot path. Called by `Merge` and `InsertOrUpdate`.
2. **`GetCounter`** UPDATE (`main.go:886`) — inlines `name` with `'%v'`. Internal-only callers, so practically safe but inconsistent.
3. **`RecalcVirtualColumns`** (`deploy.go:111`) — admin-only, but iterates user data through raw string formatters.
4. **`RecalcGroupIndexHashes`** DELETE (`deploy.go:324`) — admin-only, only inlines a numeric partition value. Low risk but trivial to fix.

Kept as-is:

- **`MakeInsertStatement`** / **`MakeUpdateStatements`** (insert-update.go:400, 757) — public, but used only by tests in `update_counter_test.go` to assert managed-column wiring. Tests will be rewritten to assert against prepared-statement output (see step 5). The functions themselves can stay if any external tooling depends on them; flag for deletion after confirming no callers outside tests.
- **`saveCacheVersionsByPackedID`** (`cache_version.go:167`) — *already* uses `?` placeholders for values; `fmt.Sprintf` only templates the keyspace name. No change needed.
- All view-table / index-group write paths — already use `batch.Query(stmt, values...)`.

Out of scope:

- DDL statements (CREATE TABLE / CREATE INDEX) — keyspace and column names are not user data.
- SELECT statements — those are bound through `Compute` already.

## Migration steps

### Step 1 — Migrate `UpdateExclude` to the prepared batch path

`UpdateExclude` does almost everything `executeInsertUpdateBatch` does for the update branch already. Replace its body:

```go
func UpdateExclude[T ...](records *[]T, columnsToExclude ...Coln) error {
    if records == nil || len(*records) == 0 {
        return nil
    }
    empty := []T{}
    return executeInsertUpdateBatch(&empty, records, false, columnsToExclude)
}
```

Verify the existing `executeInsertUpdateBatch` invariants carry over:
- `runSelfParseIfDefined(recordsForUpdate)` is already called (line 905).
- `syncIndexGroupsAfterWrite` is called (line 991).
- `syncTableBackedViews` for the update branch is called (line 1004).
- Text-search sync is called (line 1025).
- `updateCacheVersionsAfterWrite` is called (line 1048).

Compare against `UpdateExclude`'s current sync calls (lines 808-837) to make sure nothing is dropped:
- ✓ `syncIndexGroupsAfterWrite`
- ✓ `syncTableBackedViews` with `collectAffectedColumnsForExclude`
- ✓ text-search sync (includes the status-only branch)
- ✓ `updateCacheVersionsAfterWrite`

All present. The migration is a pure delete-and-redirect.

### Step 2 — Migrate `GetCounter`

```go
queryUpdate := fmt.Sprintf("UPDATE %v.sequences SET current_value = current_value + ? WHERE name = ?", keyspace)
if err := getScyllaConnection().Query(queryUpdate, counterIncrement, name).Exec(); err != nil { ... }
```

`QueryExec` is replaced by a direct prepared `Query(...).Exec()` because `QueryExec` does its own retry-on-disconnect — preserve that by wrapping in the same `no hosts available` reconnect retry block from `connection.go:112-127`, or extract that retry block into a helper `QueryExecPrepared(stmt, values...)` and call it here.

### Step 3 — Migrate `RecalcVirtualColumns`

Currently builds a list of `[]string` UPDATEs and ships them through `QueryExecStatements` → `BEGIN BATCH ... APPLY BATCH`. Replace with a `gocql.Batch`:

```go
for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
    batch := session.NewBatch(gocql.UnloggedBatch)
    for _, updateToApply := range updatesToApply[fromIndex:toIndex] {
        setClauses, setValues := buildSetClause(updateToApply.changedVirtualColumns, recordPointer)
        whereClauses, whereValues := buildWhereClause(whereColumns, recordPointer)
        stmt := fmt.Sprintf("UPDATE %v SET %v WHERE %v", scyllaTable.GetFullName(), setClauses, whereClauses)
        batch.Query(stmt, append(setValues, whereValues...)...)
    }
    if err := session.ExecuteBatch(batch); err != nil { ... }
}
```

Value extraction reuses `getNormalizedWriteValue` (the prepared-statement version), not `getNormalizedWriteLiteral`. The `buildSetClause` / `buildWhereClause` helpers can be lifted from `appendUpdateQueriesToBatch` (or refactored to share that function).

### Step 4 — Migrate `RecalcGroupIndexHashes` DELETE

Single trivial change at `deploy.go:324`:

```go
deleteStatement := fmt.Sprintf("DELETE FROM %v.%v WHERE partition_id = ?", scyllaTable.keyspace, scyllaTable.indexUpdatedTable.name)
if err := getScyllaConnection().Query(deleteStatement, partValue).Exec(); err != nil { ... }
```

### Step 5 — Update tests in `update_counter_test.go`

The three tests at lines 213, 247, 299 assert on raw CQL substrings (`"nombre = 'nuevo'"`, `"updated = 77"`, etc.). Rewrite to:

- Build a `*gocql.Batch` via the same compiler helpers (`collectInsertColumns`, `appendInsertQueriesToBatch`).
- Iterate `batch.Entries` and assert on `Entry.Stmt` (the prepared statement template) and `Entry.Args` (the bound values).

Example assertion: `Entry.Stmt` contains `"nombre = ?"` and `Entry.Args` contains `"nuevo"` at the position of the `nombre` column.

This makes the tests resilient to literal-encoding changes (and exercises the path that production actually uses).

### Step 6 — Delete dead code

After steps 1-5 land and tests pass:

- `makeUpdateStatementsBase` (only caller after step 1 is `MakeUpdateStatements`).
- `getNormalizedWriteLiteral` and `normalizeEmptyStringWriteLiteral` — only caller after migration is `MakeInsertStatement` / `MakeUpdateStatements`. If those are kept for tests, keep the helpers; otherwise delete them.
- The CQL literal branches in `makeScyllaValue` (case 1, 11, 9, etc.) that exist *only* to emit string-CQL — verify these aren't used in any read path (`column.GetValue` outside select projection compilation) before deleting.

If `MakeInsertStatement` / `MakeUpdateStatements` survive (kept for debugging), they keep their current implementation but should grow a `// Test-only: do not use for production writes` comment.

## Rollout / verification

1. **Step 1 alone** unblocks the production bug (`UpdateExclude` is what blew up). Land it first behind no flag — the change is internal and the new path is what other writes already use.
2. Run existing test suite + the renamed counter-test assertions.
3. Smoke-test the `PostProducts` 10k-record flow that triggered the original report.
4. Steps 2-4 can land in a follow-up PR — they're low-traffic paths.
5. Step 6 (deletion) lands after a soak period.

## Risk

- **Low** for step 1: the prepared-update path already executes for every `Update()` / `InsertUpdate()` call in production. Step 1 is just routing `UpdateExclude`'s callers (`Merge`, `InsertOrUpdate`) through that same path.
- **Low** for steps 2 and 4: trivial single-statement swaps.
- **Medium** for step 3: `RecalcVirtualColumns` is admin-triggered and exercises the value-extraction path for every column type, including virtuals. Needs explicit testing per column type (string, slice, complex/cbor, virtual).
- **Medium** for step 5: tests depend on string-substring assertions; the rewrite needs to preserve the *intent* of each assertion (that managed columns are emitted with the right values).

## Estimated size

- Step 1: ~15 LoC change, deletes ~60 LoC.
- Step 2: ~10 LoC, deletes ~5.
- Step 3: ~50 LoC change (mostly factoring out the SET/WHERE builder).
- Step 4: ~3 LoC.
- Step 5: ~100 LoC of test rewrites.
- Step 6: deletes ~200 LoC across `insert-update.go`, `converter.go`, `reflect_accessors.go`.

Net: code shrinks. One source of truth for writes.
