---
name: cache-groups-index-internals
description: Internals of cache groups across the ORM and genix-ui — how writes mark groups as changed, how a query picks its index and which groups it sends, and how groups are kept in S3. Read only to debug the cache or analyze its behavior in depth.
version: 0.3.0
---

# Cache groups — internals

Units: 1 tick = 2 s (SUnixTime). Scylla only; Dynamo and `idb` don't support cache groups.

## State

- **Group** = records sharing one combination of an index's Key values, within one partition.
  Named by `id` = int32 hash of the values; `igVal` = the values; `ig` = int16 index ID.
- **Mark** = row `<table>__index_updated (partition_id int, index_id smallint, index_hash int) → last_write int`.
  `lw` = a group's last write. No mark row → the group does not exist for queries.
- Request: `cc-gh` (held hashes) + `cc-lw` (held `lw` per hash). Response: `[]RecordGroup{ig, id, igVal, records, lw}`.
  Empty `records` + held `lw` = "keep yours". `ig = -1` = ungrouped result, passed through uncached.

## Invariants

1. **Marks before rows.** A write marks (`lw = now`) every group its records end in **and** every group
   they leave, synchronously, then writes rows. Mark failure aborts the write. Rationale: a failed
   or partial write can only cause an extra refetch, never a group labelled changed-free.
2. **Settle rule:** `now − lw < 8 ticks` (16 s) → unsettled: always sent whole with `lw = 0`, never
   matched, never stored in S3. Rationale: rows may land after the mark; `lw = 0` never matches, so
   the first query after settling refetches the complete group.
3. **"Keep yours" ⇔ settled AND held `lw` == stored `lw`.**
4. **S3 files are written only from a full Scylla read of a settled, non-empty group** — never from a
   file, never unsettled; the file holds the whole group (post-filters applied after reading).
5. Unmarked data (written before the group was declared) is invisible until
   `RecalcGroupIndexHashes(partition)` (`scylla/deploy.go`) rebuilds marks from `updated`.

## Compile (`scylla/index_groups.go` → `registerIndexGroup`)

| Declaration | Storage | Fetch predicate |
|---|---|---|
| 1 Key | `TypeLocalIndex` on the column (slice → panic) | `col = ?` |
| 2+ scalar Keys | virtual `zz_ig_<keys>` int32 + local index | `zz_ig = ?` |
| 2+ Keys incl. a slice | virtual `zz_igs_<keys>` `set<int>` + global index; one group per element | `zz_igs CONTAINS ?` |
| `TypeCacheGroupFromKey` | none; Keys = prefix of `KeyIntPacking`, no slices; still marked | packed key range |
| `TypeCacheGroupS3` | as `TypeCacheGroup` + S3 files | as above |

- `indexID` = `BasicHashInt(join(keyNames, "_"))` as int16, collision → `+1`.
- Group hash = `HashInt64(values…)`: FNV-32a over each value as 8 bytes LE + `0x00` (`scylla/helpers.go`).
  Values via `computeIndexGroupHashes` (Cartesian product; empty slice = no group).
- S3 startup check (`recordSnapshotCodecFor`, `scylla/record_snapshot.go`): every stored column must
  round-trip through colbin → a stored `cb:"-"` column panics.

## Query (`scylla/select_grouped.go` → `execIndexGroupQuery`)

```
reject              Select / Exclude / GroupBy / OrderDesc / Limit / AllowFilter
plan                partition '=' required; exactly one BETWEEN in the query
candidate index     every Key has exactly one statement: = | CONTAINS | IN(non-empty) | BETWEEN;
                    one Key carries the BETWEEN
score               keys*100000 − fanout (IN sizes × BETWEEN width); ties → first declared
hashes              Cartesian product; BETWEEN → every integer step from..to (swapped if reversed)
probe               SELECT index_hash, last_write FROM <t>__index_updated
                    WHERE partition_id=? AND index_id=? AND index_hash IN (…)     100 per chunk
split               no row → drop | unsettled → fetch, lw=0 | settled & held == lw → metadata only | else fetch
fetch               errgroup; concurrency cap 16 only for S3 indexes; one failure fails the query
post-filter         all statements re-applied in memory (recordMatchesPostFilter, select.go):
                    = > >= < <= IN CONTAINS BETWEEN; any other operator passes silently
```

- `IncludeCachedGroup(hash, lw)` → map keyed by hash only (not by index).
- Errors: `QueryIndexGroup requires partition equality on …`, `…requires exactly one BETWEEN statement`,
  `…did not find a compatible TypeCacheGroup for table …`, `…did not produce candidate hashes`.

## Write (`scylla/insert-update.go`)

`readStoredWrittenRows` → `collectIndexGroupMarks` (end-in + left groups; omitted columns use stored
values; dedup by partition/index/hash) → `markIndexGroupsBeforeWrite` (unlogged batch, synchronous —
Lambda freezes background work) → row batches. Partial update with some of a group's source columns
and the rest zero in the struct → panic.

## S3 snapshot (`scylla/cache_group_snapshot.go`)

- Key `cache-group/<table>/<partition>/<indexID>/<hash>.cb`.
- Layout `[fingerprint u64 LE][cursor uvarint = lw][readTime uvarint][colbin records]` (`db/cache/record_snapshot.go`).
- Used only for `TypeCacheGroupS3`, `lw ≠ 0`, file store configured.

| State | Action |
|---|---|
| held at current settled `lw` | metadata only; S3 not read |
| unsettled | Scylla; no write |
| file decodes, same fingerprint, `cursor == lw`, age ≤ 3600 ticks, `readTime` ≤ now | serve file, post-filter in memory |
| not found / other label / schema / expired | full Scylla read (no post-filter) → `PutFileIfMatch(etag)` if non-empty → post-filter |
| other S3 error | Scylla; no write; never fails the request |

`ErrFilePreconditionFailed` ignored; other write errors logged. 2 h cap bounds a file that missed a
write. No lifecycle rule: files are never deleted.

## Browser (`frontend/packages/genix-ui/cache/`)

- `group-cache.fetch.ts` (`createGroupCacheGetter` → `GETWithGroupCache`, wired in `http/index.ts`):
  - shape = `[route, ...sortedParamNames].join('|')` (values excluded);
  - sends every held group of the shape: `cc-gh = concatenateInts(id >>> 0)`, `cc-lw = concatenateInts(lw)`;
  - merge per group: `ig === -1` → pass through; cached + `lw ≠ 0` + same `lw` + empty → cached records;
    else store + return.
- `group-cache.idb.ts`: Dexie `groupRows`, keys `[queryShape+key]` (key = `igVal.join('_')`, no `ig`),
  `[queryShape+id+lw]`; DB per company + env; `CACHE_DB_VERSION` bump clears it.
- Server parse: `core.ExtractGroupIndexCacheValues` (`backend/core/cache.go`) pairs lists by position,
  `int32(uint32(hash))`.

## Known bugs and weak spots (verified by reading, not fixed)

1. **`cc-gh`/`cc-lw` mis-pairing.** `concatenateInts` (`genix-ui/utilities/parsers.ts`) splits values into
   u8/u16/u32 buckets, losing order; the server pairs by position. A held `lw = 0` lands in u8 and
   shifts every pair. A shifted `lw` that equals a group's real `lw` (common after bulk writes) →
   server says "keep yours" → browser's held `lw` ≠ response `lw` → **stores the empty group** until the
   next write. Fix: fixed-width u32 for both lists, or never store an empty group for an `lw` not sent.
2. Warehouse movements: frontend sends `almacen-id`/`producto-id` (`warehouse-movements.svelte.ts`),
   handler reads `warehouse-id`/`product-id` (`logistics/product-stock-movement.go`) → filters ignored.
3. Non-group filters are post-filtered, but label and browser key ignore them → a group filtered one
   way can be served for another value.
4. Browser key ignores `ig` → two equal-size groups reachable from one shape can collide.
5. Held groups are never pruned → URL grows ~11 chars per held group of the shape.
6. Rows landing > 16 s after their mark, or replica lag/loss at `LocalOne` → group stored incomplete
   (S3 keeps it ≤ 2 h).
7. Each BETWEEN step is a probe (and, cold, a fetch); non-contiguous codes (YYWW) probe their gaps.
8. Single-Key equality groups (`LotID`, `DocumentID`) need `Between(v, v)` to be queried; unread
   FromKey groups (`credit_history`, `invoice_summary`) still cost a mark per write.

## Constants

| Constant | Value | Where |
|---|---|---|
| `indexGroupSettleTicks` | 8 (16 s) | `scylla/index_groups.go` |
| `RecordSnapshotMaxAgeTicks` | 3600 (2 h) | `db/cache/record_snapshot.go` |
| `cacheGroupSnapshotConcurrency` | 16 (S3 indexes only) | `scylla/cache_group_snapshot.go` |
| Probe chunk | 100 (`getMaxClusteringKeyRestrictionsPerQuery`) | `scylla/select_grouped.go` |

## Files

| File | What |
|---|---|
| `backend/genix-orm/db/schema.go` | type constants, `RecordGroup` |
| `backend/genix-orm/db/tablestruct.go` | `QueryIndexGroup`, `IncludeCachedGroup`, `Exec` routing |
| `backend/genix-orm/scylla/index_groups.go` | compile, hashes, marks, settle constant |
| `backend/genix-orm/scylla/select_grouped.go` | planner, probe, settle split, fetch |
| `backend/genix-orm/scylla/cache_group_snapshot.go` | S3 rules, codec check |
| `backend/genix-orm/scylla/insert-update.go` | marks-first order, partial-update panic |
| `backend/genix-orm/scylla/deploy.go` | `__index_updated` DDL, `RecalcGroupIndexHashes` |
| `backend/genix-orm/scylla/select_grouped_test.go`, `cache_group_snapshot_test.go`, `cache_group_index_test.go` | behaviour specs |
| `backend/core/cache.go` | `ExtractGroupIndexCacheValues`, `parseConcatenatedInts` |
| `frontend/packages/genix-ui/cache/group-cache.fetch.ts`, `group-cache.idb.ts` | browser side |
