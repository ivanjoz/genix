# Plan — `updated_version`: one monotonic sequence for delta cache and by-IDs cache

## 1. What changes and why

Today two unrelated freshness mechanisms coexist:

| Mechanism | Value | Storage | Used by |
|---|---|---|---|
| Cache version (`ccv`) | wrapping `uint8` per 256-record slot | `cache_version.cached_values` — **one blob per (table, partition)** | by-IDs cache |
| `updated` timestamp | `SUnixTime` int32, bucketed to 20 s inside the packed delta key | the table's own `updated` column | delta cache |
| `update_counter` | per-partition autoincrement, DB-only column | the table's `update_counter` column | index-group cache (`upc`) only |

Three problems:

1. **The blob rewrite.** Bumping one slot means re-reading and re-writing the entire
   `cached_values` blob (`cache_version.go:167-174`, `221-265`). Every write is a read-modify-write
   of up to 512 bytes, and two concurrent writers to the same tenant can lose each other's bumps.
2. **`updated` is not deterministic as a watermark.** Two writes inside the same second — or the
   same 20-second bucket after trimming — are indistinguishable. A client that syncs mid-second
   gets a watermark that hides records it never received. `PLAN_CACHE_DELTA.md` was written to work
   around exactly this and stops being necessary here.
3. **The sequence already exists and is already fetched on every write** (`insert-update.go:121-154`,
   concurrently with the autoincrement-ID reservation at `insert-update.go:886-911`), but it is
   only spent on index groups.

The change: **promote that sequence to `updated_version`, persist it on the record, and make it the
single freshness currency for both caches.**

- Delta cache filters `updated_version >= W+1` instead of `updated >= W`. Exact, no bucketing.
- By-IDs cache compares against `cache_updated_version`, a row-per-slot table. A write updates one
  ~10-byte row per touched slot with a **blind write** — no read, no blob.

---

## 2. Decisions taken (confirmed)

| # | Decision |
|---|---|
| D1 | The by-IDs comparison value travels in the **`upv` field itself**. On by-IDs reads the ORM overwrites `upv` with the *slot* version; on every other read `upv` is the record's own version. The `ccv` / `CacheVersion` field is deleted. |
| D2 | `cache_updated_version.updated_version` is **`smallint` (int16)**. It is only ever compared for equality; wrap-around aliasing (1 in 65 536) is accepted. |
| D3 | The concurrent-writer gap (§9) is **accepted and documented**, not mitigated. |
| D4 | Record field is **`UpdatedVersion int32 \`json:"upv,omitempty"\`** — snake-cases to `updated_version` with no `db:` tag. The `UpVersion` field already added to `client_provider.go` gets renamed. |
| D5 | **Every** table declares a unique `TableSchema.ID` in `1..16383`, enforced by the ORM at compile time and by `scripts/validation/check_tables.go` statically. |

---

## 3. Data model

### 3.1 The sequence

Unchanged mechanically. `fetchManagedCounterValues` (`insert-update.go:121`) reserves one value per
(partition, table) per write call from counter `x{partition}_{table}_updated`, in parallel with the
autoincrement-ID reservation. Every record in one `Insert`/`Update`/`InsertUpdate` call for a given
partition shares that value. Only the column it lands in is renamed.

### 3.2 `cache_updated_version`

```sql
CREATE TABLE IF NOT EXISTS <ks>.cache_updated_version (
    partition_table_id  int,       -- (partitionID << 14) | tableID
    id_slot             tinyint,   -- int8(uint8(recordID))
    updated_version     smallint,  -- int16 truncation of the sequence value, never 0
    PRIMARY KEY (partition_table_id, id_slot)
) WITH <makeStatementWith>;
```

- One partition per (tenant, table), at most 256 tiny clustering rows. `rows_per_partition: ALL`
  caching (already in `makeStatementWith`) keeps the whole partition hot.
- **Reads** fetch the whole partition in one query — simpler than an `IN` list, one round trip,
  and it warms the cache for the next request:
  `SELECT id_slot, updated_version FROM cache_updated_version WHERE partition_table_id = ?`
- **Writes** are blind `UPDATE`s, batched, one per touched slot. No read-modify-write.

### 3.3 Packing `partition_table_id`

```go
// 18 bits of partition + 14 bits of table ID pack losslessly into one int32 key.
func makeCacheUpdatedVersionPartitionID(partitionID int32, tableID int16) int32 {
    return int32(uint32(partitionID)<<14 | uint32(tableID))
}
```

Validated once at table-compile time, not per write:

- `partitionID` must be `0 .. 262143` (2^18-1) — checked per write value, error (not panic) when a
  tenant ID exceeds it, since it is data-dependent.
- `tableID` must be `1 .. 16383` (2^14-1) — panics at compile time.

The high bit can be set, so the value is stored as a signed `int` in Scylla via
`int32(uint32(...))`. Both sides use the same cast; the value is opaque.

### 3.4 Slot version encoding

```go
// Slots only ever compare for equality, so int16 truncation is enough. 0 is reserved for
// "the client holds no version", so a real slot never stores it.
slotVersion := uint16(sequenceValue)
if slotVersion == 0 { slotVersion = 1 }
```

- Missing row ⇒ server-side value `0` ⇒ never equal to any client value ⇒ always fetch. Safe default.
- Client with no stored version sends `0` ⇒ same.

---

## 4. Backend ORM changes (`backend/genix-orm`)

### 4.1 `db/schema.go`

- `TableSchema.ID int16` — document it as the stable, hand-assigned, globally unique table ID that
  feeds `partition_table_id`. Never derived from the name.
- `ColumnNameUpdated = "updated"` → add `ColumnNameUpdatedVersion = "updated_version"`. `Delta()`
  switches to the new one; `ColumnNameUpdated` stays only if something else still resolves `updated`
  by name (check at implementation time; delete it if not).
- Rename `DisableUpdateCounter` → `DisableUpdatedVersion`. Delete `UseUpdateCounter` (already dead —
  `reflect.go:106-109` only panics on it).
- `GenericRecord.CacheVersion uint8 \`json:"ccv"\`` → `UpdatedVersion uint16 \`json:"upv,omitempty"\``.
- `IDCacheVersion.CacheVersion uint8` → `UpdatedVersion uint16`. Consider renaming the type to
  `IDUpdatedVersion` for coherence (cheap; it has ~10 call sites).

### 4.2 `db/table.go` — `TableCore`

- Add `ID int16`.
- `UpdateCounterCol` → `UpdatedVersionCol`.
- Delete `CacheVersionFieldIndex`. The slot version is now written through the mapped column
  (`UpdatedVersionCol.SetValue`), so no reflection field path is needed anywhere.
- Keep `SaveCacheVersion`, `CacheVersionPartitionCol`, `CacheVersionKeyCol` but rename to
  `SaveUpdatedVersion` / `UpdatedVersionPartitionCol` / `UpdatedVersionKeyCol`.

### 4.3 `db/metacache.go`, `db/tablestruct.go`

- Delete `findCacheVersionFieldIndex`, `cacheVersionFieldIndex`, `setCacheVersionFieldIndex`,
  `CacheVersionFieldIndex()`, `tableStructCacheMetaSetter`/`Getter`. ~60 lines gone.
- `Delta(updatedSince int32, syncFilterValues ...int64)`:
  ```go
  // The packed view builds its lower bound from the statement value and ignores the operator, so
  // ">= W+1" is how an exclusive delta is expressed. Versions start at 1, so a first sync (W=0)
  // still reads everything.
  e.SetWhere(ColumnNameUpdatedVersion, ">=", updatedSince+1)
  ```
  This is why the boundary tick stops repeating: with an exact sequence there is nothing to re-read.

### 4.4 `scylla/reflect.go`

- `managedUpdateCounterColumnName = "update_counter"` → `managedUpdatedVersionColumnName = "updated_version"`.
- `bindManagedAuditColumns`: bind `UpdatedVersionCol` unless `DisableUpdatedVersion`. Then validate:
  ```
  Table "x": SaveUpdatedVersion / TypeDelta requires the record and table structs to declare
    UpdatedVersion int32 `json:"upv,omitempty"`
  ```
  The check is that `ColumnsMap["updated_version"]` resolves to a **real mapped column** (present in
  the table struct, hence backed by a record field), not the DB-only phantom `ensureManagedIntColumn`
  creates. Tables that need neither feature keep the phantom column exactly as today.
- Store and validate `dbTable.ID = schema.ID` (1..16383, non-zero).

### 4.5 `db/registry.go` — ID uniqueness

`RegisterTableFactory` only stores closures, so it cannot see IDs. Add a second registry populated
at table-compile time (`getOrCompileScyllaTable`): `map[int16]string`, panicking on a collision with
both table names in the message. Compile-time is the right place — every table is compiled on the
first query and by `check_tables`.

### 4.6 `scylla/cache_version.go` → rewrite as `scylla/cache_updated_version.go`

Deleted outright:

| Symbol | Reason |
|---|---|
| `decodeCacheVersions`, `encodeCacheVersions`, `nextCacheVersion` | the blob is gone |
| `getCacheVersionsByPackedID`, `saveCacheVersionsByPackedID` | replaced by row reads/blind writes |
| `makeCacheVersionPackedID` (int64 name-hash packing) | replaced by §3.3 |
| `findCacheVersionFieldIndexInRecordType`, `setRecordCacheVersion`, `tableStructCacheMetaGetter` | column-based stamping |
| **`assignCacheVersionsAfterSelect`** and its call at `select.go:904` | **critical**: it stamps every select on a cache-enabled table; leaving it would overwrite `upv` with the slot version on delta reads and destroy the watermark. Removing it also drops one query from every ordinary select. |
| `ensureCacheVersionColumnsForSelect` (`cache_version.go:307`, `select_helpers.go:151`) | only existed to feed the stamping above |
| `cacheVersionMismatchDebugRow`, `buildCollisionIDsByPartition` | debug scaffolding for the blob model |

Rewritten:

```go
// Write path — no read, one blind UPDATE per touched slot.
func updateSlotVersionsAfterWrite[T any](records *[]T, scyllaTable ScyllaTable) error
```
- Reads each record's own `updated_version` value straight off `UpdatedVersionCol` (already set by
  `applyPrefetchedManagedCounterValues`), so no extra plumbing from the counter prefetch.
- Collects unique `(partitionID, slot)` pairs, keeping the max version per pair.
- Emits one `gocql.UnloggedBatch` of `UPDATE cache_updated_version SET updated_version = ?
  WHERE partition_table_id = ? AND id_slot = ?`.
- Does **not** touch the records — their `upv` is already correct.

```go
// Read path — one partition read per tenant, then equality per requested ID.
func loadSlotVersions(keyspace string, partitionTableID int32) (map[uint8]uint16, error)
```
- `planCachedIDsFetch` keeps its shape but compares `uint16` slot versions instead of `uint8` group
  versions, and carries `map[int32]map[uint8]uint16` keyed by partition.
- After the batched table select, stamp every returned record:
  `scyllaTable.UpdatedVersionCol.SetValue(ptr, int32(slotVersion))` — this is the D1 overwrite.
- `QueryCachedIDs` and `QueryCachedGenericByIDs` (`cache_version_generic.go:201`) keep their
  signatures; only the version type and stamping target change.

`forEachCachedIDsBatch`, `splitIDsIntoBatches`, `prepareCachedIDsTable`, `resolveCacheVersionForID`
survive with renames.

### 4.7 `scylla/init.go` and the deploy path

`InitCacheVersionTable` → `InitCacheUpdatedVersionTable` with the §3.2 DDL, reached through a new
`EnsureInternalTables()` that `Init()` and `DeployScylla()` both call.

This closes a pre-existing gap: `fn-homologate` → `DeployDatabaseSchemas` never called
`scylla.Init()`, so a standalone deploy created the keyspace and every application table but none of
the ORM's own. That was survivable while `sequences` already existed in every live keyspace; it is
not survivable for a brand-new table. `DeployScylla` now ensures them itself rather than trusting
the caller, so every deploy path is covered.

The old `cache_version` table is dropped manually when the data is wiped — homologation never drops
anything, and no migration code is kept.

### 4.8 `scylla/index_delta_view.go`

- The implicit trailing key becomes `UpdatedVersionCol` instead of `UpdatedCol`. Panic message and
  the "remove it from Keys" guard follow.
- Rename `deltaUpdatedDigitsInt32` (8) / `deltaUpdatedDigitsInt64` (10). **Keep the widths.** 8 digits
  is what lets `client_provider` (`status` 1 + `type` 1 + version 8 = max packed 1.3e9) stay inside
  an int32; widening to 9 would push it to ~1.3e10 and force a bigint. The two-pass widening still
  earns its keep: when a bigint is unavoidable anyway, the version slot gets 10 digits of headroom
  for free.
- The comments change meaning: 8 digits no longer buys a 20-second bucket, it buys **10^8 write
  calls per (tenant, table)** before the slot is exhausted.
- **New overflow guard.** `trimRightToDigitsNonNegative` (`index_int_packing.go:33-50`) silently
  drops least-significant digits when a value overruns its slot. That was the *intended* bucketing
  for a timestamp; for a sequence it silently collapses versions in groups of ten and breaks `>`.
  Add an explicit check on the delta version slot at write time and fail loudly:
  `Table "x": updated_version %d exceeded the delta view's %d-digit slot`.

### 4.9 `scylla/insert-update.go`

Rename-only, plus one call swap:
- `updateCounterValues` → `updatedVersionValues`, `managedUpdateCounterColumnName` → new constant,
  `UpdateCounterCol` → `UpdatedVersionCol` (lines 25-66, 121-234, 493, 888, 1109-1143).
- `updateCacheVersionsAfterWrite` (line 1046) → `updateSlotVersionsAfterWrite`.

`Insert` (689) and `Update` (755) go through `InsertUpdateBase`, so both write paths are covered by
the single call site at 1046 — verify during implementation that no path writes rows without
reaching it.

### 4.10 Left alone deliberately

`index_groups.go` and `select_grouped.go` keep the `index_updated.update_counter` column and
`RecordGroup.UpdateCounter` / `upc` / `cc-gh` / `cc-upc`. That is a separate protocol (per index
group, not per record); it is fed by the same sequence and renaming it buys nothing.

---

## 5. Backend application changes

1. **Table IDs.** Add a unique `ID:` to all ~40 `GetSchema()` declarations
   (`business/types/*`, `sales/types/*`, `logistics/types/*`, `finance/types/*`, `core/types/*`,
   `config/types/*`, `security/types/*`, `webpage/types/*`, plus `core/cache.go`, `core/cron-action.go`).
   Keep a numbered list in `backend/docs/ORM_DATABASE_QUERY.md` so the next table picks the next free
   ID.
2. **`UpdatedVersion` fields.** For every table with `SaveCacheVersion: true` or a `TypeDelta` index:
   add `UpdatedVersion int32 \`json:"upv,omitempty"\`` to the record struct **and**
   `UpdatedVersion db.Col[XTable, int32]` to the table struct (both are required — DB columns are
   enumerated from the table struct, `reflect.go:220-277`). Delete the `CacheVersion uint8` field.
3. **`client_provider.go`** (`business/types/`): rename `UpVersion` → `UpdatedVersion`, drop
   `CacheVersion`, add `ID: 1`, add the table-struct column.
4. **Handlers.** Delta endpoints read the watermark from `upv`:
   `core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))` → `req.GetQueryInt("upv")`.
   Affects `business/client_provider.go:16`, `config/system_parameters.go:12`,
   `sales/sales_planning.go:11,81`, `sales/shipping_costs.go:12`, `sales/sale_summary_status.go:13`,
   `logistics/supply-material-management.go:15`, `config/cron-actions-scheduled.go`,
   `security/usuarios.go:106`, `logistics/product-stock-movement.go:270` — one per delta endpoint,
   converted as each table gains its `TypeDelta` index. Endpoints still on plain `updated` views keep
   `updated` until they are migrated.
5. **`core/cache.go:101` `ExtractCacheVersionValues`** → `ExtractUpdatedVersionValues`. `cc-ver` is
   parsed as `uint16` (see §6.3).
6. **`db/db.go`** — re-export the renamed aliases (`FixedValues` alignment typo on line 30 gets
   fixed in passing).

---

## 6. Frontend changes (`frontend/packages/genix-ui/cache`)

### 6.1 `cache-by-ids.svelte.ts`

- `IMinimalRecord`: drop `ccv?: number`, add `upv: number` (`0..65535`, the slot version).
- Every `record.ccv` read (lines 53, 177, 267, 293, 332, 370, 1094) → `record.upv`.
- The validation at line 83 changes from `0..255` to `0..65535`.
- Rename the local `ccVer` / `recordsCachedUpdatedGroupsIDs` variables to say *slot version*.

### 6.2 Document the dual meaning of `upv`

`upv` means the record's own version on delta responses and the slot version on `-ids` responses.
The two stores never share rows (`cacheByIDs` table vs delta route rows), but one seam exists:
**hinted records**. A record seeded into the by-IDs cache from a delta list carries a full int32
version, which can never equal a `uint16` slot version, so it is revalidated once and then correct.
Self-healing, but it must be written down in `CACHE_BY_IDS.md` or it will read as a bug.

### 6.3 `cc-ver` encoding — a real bug this change would introduce

`buildFetchUriParams` (lines 68-109) buckets IDs into u8/u16/u32 groups by magnitude and relies on
`cc-ver` values all landing in the *same* bucket to stay positionally aligned. That holds today only
because every version is ≤ 255. With `uint16` versions, mixed magnitudes would split `cc-ver` across
two buckets and silently misalign it against `cc-ids`.

Fix: emit `cc-ver` as a **single fixed-width u16 array** (one bucket, always), and decode it that way
in `parseConcatenatedInts`' caller. `cc-ids` keeps its magnitude bucketing.

### 6.4 `delta-cache.fetch.ts`

- `getRecordUpdateValue` (line 262): `record?.upc || record?.upd || 0` → `record?.upc || record?.upv || 0`.
- `getNextRouteURL` (lines 530, 543): the `updated` query param becomes `upv`. The per-field params
  in the same loop are field names and are unaffected.
- The `updated < prevUpdated` warning at line 645 stays; it becomes strictly more meaningful now that
  the watermark is exact.

### 6.5 Types

`frontend/core/types/common.ts` and the generated struct mirrors
(`scripts/generators/sync_struct_interfaces.go`) — regenerate after the Go structs change.

---

## 7. Docs

| File | Change |
|---|---|
| `backend/docs/ORM_DATABASE_QUERY.md` | `TableSchema.ID` (required, unique, 1..16383) + the assigned-ID table; `UpdatedVersion` field requirement; `cache_updated_version`. |
| `backend/docs/CREATE_API_HANDLERS.md` | delta handlers read `upv`, not `upd`/`updated`. |
| `backend/genix-orm/scylla/ORM_INTERNALS.md` | replace the cache-version section with the slot-version model. |
| `frontend/packages/genix-ui/cache/CACHE_BY_IDS.md` | `upv` replaces `ccv`; the dual meaning from §6.2. |
| `frontend/packages/genix-ui/cache/DELTA_CACHE.md` | `upv` watermark, exact `>`; the §9 limitation. |
| `scripts/CREATE_EDIT_TABLE.md`, `scripts/CHECK_TABLES_SCRIPT.md` | new required fields. |
| `.claude/skills/create-database-tables`, `delta-cache-api`, `fetch-record-by-id-api` | these skills teach the current conventions and will hand out stale ones otherwise. |
| **Delete `PLAN_CACHE_DELTA.md`** | its entire problem — re-reading the boundary tick under `>=` — disappears with an exact sequence. |
| **Delete `PLAN_TYPE_DELTA.md`** | already implemented (`scylla/index_delta_view.go`, uncommitted). |

---

## 8. Static validation (`scripts/validation/check_tables.go`)

New checks:
1. Every `GetSchema()` declares `ID`, in `1..16383`, unique across the project.
2. `SaveCacheVersion: true` or a `TypeDelta` index ⇒ the record struct has
   `UpdatedVersion int32 \`json:"upv,omitempty"\`` **and** the table struct has the matching `Col`.
3. No record struct still declares `CacheVersion` / `json:"ccv"`.

---

## 9. Known limitation: the concurrent-writer gap (accepted, D3)

Writer A reserves version 100 and writer B reserves 101. B commits first. A client polls, sees
`max(upv) = 101`, stores it. A then commits its rows at 100. The next poll asks for `>= 102` and A's
records are **never delivered** until they are written again.

- Window: the few milliseconds between two overlapping write batches on the *same tenant and table*,
  with a poll landing exactly inside it.
- The current `updated` + `>=` + 20-second bucketing accidentally covers this by re-reading the whole
  boundary bucket, so this is a real regression on that one axis — traded for exactness everywhere else.
- Recorded in `DELTA_CACHE.md`. If it ever bites, the fix is a per-(partition, table) published
  watermark that bounds the delta query above; that needs cross-process in-flight tracking and is
  explicitly out of scope here.

Second accepted limitation: `int16` slot versions alias every 65 536 write calls to a
(tenant, table). A client whose cached copy survives untouched across exactly that many writes can be
served a stale record. Accepted per D2.

---

## 10. Execution order

Each step should compile and pass `go test ./...` in the submodule before the next.

1. **ORM, schema layer** — `TableSchema.ID`, `ColumnNameUpdatedVersion`, renames in
   `db/schema.go`, `db/table.go`; delete the cache-version field-index plumbing from
   `db/metacache.go` / `db/tablestruct.go`; `Delta()` emits `>= W+1`.
2. **ORM, Scylla driver** — rename the managed column, ID registry + validation in `reflect.go`,
   the `UpdatedVersion`-field-required panic, the delta view's trailing key + overflow guard.
3. **ORM, cache table** — `cache_updated_version.go` replacing `cache_version.go`; delete
   `assignCacheVersionsAfterSelect` and `ensureCacheVersionColumnsForSelect`; `init.go` DDL.
4. **ORM tests** — `cache_version_generic_test.go`, `update_counter_test.go`,
   `index_delta_view_test.go`, `executor_test.go`, `select_helpers_test.go` all reference renamed
   symbols; add coverage for the packing helper, the missing-slot ⇒ always-fetch rule, and the
   overflow guard.
5. **Application tables** — IDs on all ~40 schemas; `UpdatedVersion` fields where required; drop
   `CacheVersion`. Run `check_tables` (extended per §8).
6. **Handlers** — `upv` query param, `ExtractUpdatedVersionValues`.
7. **Frontend** — `IMinimalRecord`, the `cc-ver` single-bucket encoding, `getRecordUpdateValue`,
   the route param; regenerate the struct mirrors.
8. **Docs and skills**; delete the two stale plan files.
9. **Wipe the data**, drop `cache_version` by hand (homologation never drops), redeploy schemas
   (`fn-init` or `fn-homologate` — both now create the ORM's internal tables), verify end to end:
   a first sync, a delta sync returning an empty array on the second poll (the boundary tick no
   longer repeats — this is the observable proof the change worked), and a by-IDs request that
   returns nothing when no slot moved.

---

## 11. Open items to confirm during implementation

- Whether anything besides `Delta()` resolves the `updated` column by name (`ColumnNameUpdated`), and
  so whether that constant can be deleted.
- Whether any write path reaches Scylla without passing through `executeInsertUpdateBatch`'s
  `updateSlotVersionsAfterWrite` call — `deploy.go:346-360` (the reindex path) reads
  `update_counter` directly and will need the rename plus a decision on whether it should bump slots.
