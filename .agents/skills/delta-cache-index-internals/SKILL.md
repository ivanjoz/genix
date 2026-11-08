---
name: delta-cache-index-internals
description: Internals of the delta cache across the ORM and genix-ui — the sync cursor, how late writes are detected, how removed records are tracked, and how server snapshots are kept. Read only to debug the cache or analyze its behavior in depth.
version: 0.3.0
---

# Delta cache — internals

> **Driver note (2026-11-07):** the Scylla driver was deleted (`plan/MARIADB_FIRST_REFACTOR_PLAN.md` §7).
> The protocol, units and invariants below are driver-neutral (`genix-orm/db/cache`) and still hold.
> The `scylla/...` file references point at the deleted implementation (git history); the live ones
> are `mariadbdriver/delta.go`, `snapshot.go` and their `pgdriver/` counterparts. Rewriting the per-file sections for the SQL drivers is an open follow-up.

Units: 1 tick = 2 s (SUnixTime). MariaDB and PostgreSQL; Dynamo uses an older delta-log protocol
(`dynamo/delta.go`, `db/cache/delta_log.go`); the `idb` driver is not implemented.

## State

- Record: `upd` (Updated, ORM-stamped), `ver` (Version, +1 per write, wraps at 32767), `ss`.
- Cursor `C.V[.r]` per response key. **C** = max `upd` held. **V** = server time when its last answer
  started. `.r` = retry flag. Zero/absent/malformed cursor = first sync.
- **Prefix** = one combination of a TypeDelta index's Key values (e.g. `Status=1`); a query may cover several.
- Assumption everything rests on: **a write is visible ≤ 16 s after its stamp** (`DeltaWriteLagTicks = 8`).
  `Updated > C` alone would miss a write stamped ≤ C that became visible after the browser's read.

## Invariants

1. V is taken **before** any read (`getManagedUnixTime` at the start of `execDelta`).
2. Only records stamped in the **unsettled window `[V − 8, C]`** can be missing from the browser; the
   server sends a hash of that window instead of its records. Window empty (`V − 8 > C`) → no hash.
3. Server and browser hash the same set: rows with `upd ∈ [from, to)` and filter field ∈ filter
   values, **excluding IDs this answer brought or removed**. Go `DeltaRecordHash` ≡ JS
   `deltaRecordHash`, bit-identical (vectors: `db/cache/testdata/delta_window_vectors.json`).
4. On mismatch the browser keeps **new C + OLD V + `.r`** (the old window is the one that missed a write)
   and re-fetches at once; `.r` makes the server resend the window's records instead of hashing.
5. Only a full view read writes a snapshot; a merged (file + changes) answer is never stored.
6. No hard deletes on TypeDelta tables: the view drops the row and no move is recorded.

## Server algorithm (`scylla/delta_view.go` → `execDelta`)

```
PlanDeltaSync(cursor, now)                        db/cache/delta_window.go
  unsettledFrom = max(V − 8, 0)
  ChangedFrom   = C + 1        | .r → unsettledFrom
  MovesFrom     = min(C + 1, unsettledFrom)
  window        = [unsettledFrom, C + 1) unless settled (unsettledFrom > C) or .r
zero cursor → replace
per prefix (parallel): changed rows  zz_dlt_<tag> > ChangedFrom   limit threshold+1
                       moves         __delta_moves ≥ MovesFrom    limit threshold+1
changed > threshold OR moves > threshold → replace        (threshold default 100)
else: hash (ID, ver) of window rows; moved IDs not re-read → RemovedIDs
append metadata record: ID −1, Metadata = EncodeDeltaWindowMetadata(...)
replace: per prefix snapshot (if valid) or view; sort upd desc; cut to Limit; next C = max upd
```

Consequence: moves are read from the window start, so > 100 soft deletes re-trigger a replace for 16 s.

## Browser algorithm (`frontend/packages/genix-ui/cache/`)

1. Delete removed IDs. 2. Store records (same `ver` → skip). 3. Hash held rows in the window minus IDs
of this answer. 4. Match → store answer's `C.V`; mismatch → `C.oldV.r`, `forceNetwork`, re-fetch once.

- Multi-table: one cursor param per response key (`_default` → `up`), one metadata record per key,
  independent mismatch. The retry sends **every** key's cursor (a missing cursor = full replace).
- Empty view → cursor `''` → next sync is a first sync.

| File | Role |
|---|---|
| `delta-cache.fetch.ts` | URL params, apply/merge, `.r` retry, IndexedDB writes |
| `delta-cache.cursor.ts` | `readDeltaKeyResponse`, hash compare, `C.V` / `C.V.r` formatting |
| `delta-window.wire.ts` | metadata decode, `deltaRecordHash`, `hashHeldDeltaWindow` |
| `delta-cache.idb.ts`, `delta-cache.types.ts` | route row, `deltaCursors[key]`, `CACHE_DB_VERSION = 9` |

## Metadata record

`EncodeDeltaWindowMetadata` (`delta_window.go`) → Safe85 of: flags (8 = window, |2 = replace), next C,
V, window from/to, u32 hash, filter field JSON name, filter values, delta-coded removed IDs. Decoded by
`delta-window.wire.ts`. The filter field name comes from the **table record type**
(`deltaFieldJSONName`), so a projected DTO must use the same JSON names for `upd`/`ver`/`mdt`/ID.

## Snapshots (`scylla/delta_snapshot.go`, `db/cache/delta_snapshot.go`)

- Key `delta-snapshot/<table>/<partition>/<tag>/<prefix|all>.l<limit>.cb`.
- Layout `[fingerprint u64][C uvarint][V uvarint][colbin records]` (`db/cache/record_snapshot.go`).
- On replace: read file → treat it like a browser holding `C.V` (changed ≤ 101, moves ≤ 101, window
  hash only if unsettled) → answer `file − removed + changed`.
- Use the file only if ALL hold, else read the view and rewrite the file:
  - same colbin fingerprint;
  - not expired: ≤ `RecordSnapshotMaxAgeTicks` (3600 = 2 h) past V, and `ReadTime` ≤ now
    (bounds how long a file can carry a write that broke the 16 s rule);
  - changed and moved counts ≤ threshold;
  - window hash matches;
  - no `Limit` gap opened by a removal;
  - no `Select`/`Exclude` projection, no stored `cb:"-"` column.
- Write: `PutFileIfMatch(etag)`; precondition failure ignored. Empty prefix → no file. S3 error → view,
  no write.

## Storage and write path

| Object | Shape | Code |
|---|---|---|
| `<table>__dlt_<tag>_view` | MV, PK `((partition), zz_dlt_<tag>, id)` DESC; `zz_dlt` packs `[pinned keys…][updated 31 bits]`; `gc_grace_seconds 86400` | `delta_view.go` |
| `<table>__delta_moves` | PK `((partition_id, index_tag, pinned_prefix), updated, id)`, no TTL; created only if an index pins a column | `delta_view.go`, `deploy.go` |

Write (`stampDeltaWrites`, `writeDeltaMoves`): read stored `ver` + pinned values at `LocalQuorum` →
stamp `ver = cache.NextVersion`, `upd` → write row → write a move for each record whose pinned prefix changed.

## Constants

| Constant | Value | Where |
|---|---|---|
| `DeltaWriteLagTicks` | 8 (16 s) | `db/cache/delta_window.go` |
| `DefaultDeltaReplaceThreshold` | 100, per table per sync (`Index.DeltaReplaceThreshold`) | `db/cache/delta_window.go` |
| `RecordSnapshotMaxAgeTicks` | 3600 (2 h) | `db/cache/record_snapshot.go` |
| `deltaUpdatedSlotBits` | 31 | `scylla/delta_view.go` |
| `MaxVersion` | 32767 | `db/cache/delta_log.go` |
| `maxFixedValueFanoutQueries` | 8 | `scylla/select_helpers.go` |

## Known weak spots

- 16 s rule is trusted: a later-visible write is missed until rewritten; a snapshot carries it ≤ 2 h.
- A mismatch resends the whole window: every record sharing a bulk write's tick.
- DTOs without `upd`/`ver`/`mdt` (`accounting/accounting_operations_api.go`) hash zero rows → mismatch →
  extra `.r` round trip on every sync near a write.
- Packed int64 IDs > 2⁵³ (`warehouse_product_stock`) are inexact as JS `number` — untested for hash/removed IDs.
- Misleading comments: retry comment in `delta-cache.fetch.ts` (says cursor dropped; sends `.r`);
  `db/cache/filestore.go` header (describes Dynamo's delta-log); `scylla/reflect.go` panic asks for
  `"ver,omitempty"` (records use `"ver"`); handler comments about "the log" / "deleted entries evict".

## Files

| File | What |
|---|---|
| `backend/genix-orm/db/cache/delta_window.go` | cursor, `PlanDeltaSync`, hash, metadata encoding |
| `backend/genix-orm/db/cache/delta_snapshot.go` | snapshot rules, key, held-window hash |
| `backend/genix-orm/db/cache/record_snapshot.go` | file layout, max age (shared with cache groups) |
| `backend/genix-orm/scylla/delta_view.go` | compile checks, view/moves DDL, write stamping, `execDelta` |
| `backend/genix-orm/scylla/delta_snapshot.go` | serving and writing snapshots |
| `backend/genix-orm/scylla/reflect.go` | managed `Updated`/`Version` columns |
| `backend/genix-orm/db/tablestruct.go` | `Delta()` builder |
| `backend/core/responses.go`, `backend/core/cache.go` | `GetDeltaCursor`, `WithDeltaReplace` |
