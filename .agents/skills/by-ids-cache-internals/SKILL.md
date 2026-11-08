---
name: by-ids-cache-internals
description: Internals of the records-by-ID cache across the ORM and genix-ui — how changes are versioned, how requests are encoded, and the read/write data flow. Read only to debug or analyze its behavior in depth.
version: 0.3.0
---

# By-IDs cache — internals

> **Driver note (2026-11-07):** the Scylla driver was deleted (`plan/MARIADB_FIRST_REFACTOR_PLAN.md` §7).
> The protocol, units and invariants below are driver-neutral (`genix-orm/db/cache`) and still hold.
> The `scylla/...` file references point at the deleted implementation (git history); the live ones
> are `mariadbdriver/cache_by_ids.go` and `pgdriver/cache_by_ids.go` (a `_cache_by_ids_slots` table, as Scylla had). Rewriting the per-file sections for the SQL drivers is an open follow-up.

Units: 1 tick = 2 s (SUnixTime). MariaDB, PostgreSQL and Dynamo; the `idb` driver is not implemented
(`QueryCachedIDs` and writes return `errNotImplemented`).

## State

- **Slot** = `uint8(ID)` (JS `id % 256`); 256 slots per (table, partition).
- **Slot version** = highest `Updated` written to any record of the slot. Versions are per slot so a
  request reads one 256-entry row instead of every requested record; a write to one record makes its
  slot-mates re-read (wasted read, never a wrong answer).
- On this path a returned record's `upd` = **its slot's version** (overwritten by the server), the value
  the browser sends back next time — not the record's write time.
- **Year tick** (what the browser holds/sends): `0` unknown/untrusted → always read; `1` slot not
  written this cycle; `≥ 2` = `Updated − cycleStart + 2`. Cycle starts **Jan 1 07:00 UTC**; 3 bytes
  hold a year (15.8 M ticks < 16.7 M). Browser stores `upd = cycleStart + tick − 1`, converts back with
  `upd − cycleStart + 1` (0 if from an earlier cycle). Vectors: `db/cache/testdata/year_tick_vectors.json`.

## Invariants

1. A held version is never above the slot's real one.
2. **Slot raised after the row** (and after views/text index). Raising first could pair a new version
   with an old row, which then matches forever.
3. **Trust window:** a slot version is reported only once its tick ended ≥ 8 s ago
   (`SlotTrustTicks = 4`); before that the record returns with `upd = 0` → re-read next revalidation.
   Covers a slot-mate whose write was stamped with the same tick but landed after the read (it would
   otherwise be held at the current version with old data and never re-read). Trusted rule: a write is
   visible ≤ 8 s after its tick ends, clock skew included.
4. **Slots never go down:** Scylla cells use `USING TIMESTAMP` = stamp time (highest wins = `max()`
   without a read); slot files use `max(stored, new)` + `PutFileIfMatch`.
5. **One tick per slot suffices:** the browser sends the lowest held tick among the slot's requested
   IDs. Lowest == slot → all current; lowest < slot → all read. Lowest 0 → slot omitted → all its IDs read.
6. Cycle change → every held tick converts to 0 → every record re-read once (yearly reset, no job).

## Read path

```
GetCachedIDsRequest (backend/core/cache.go)   decode ?cc=; error on empty, decode failure, partition ≠ 0;
                                              partition := companyID
db.QueryCachedIDs[T] → scylla QueryCachedIDs  scylla/cache_by_ids.go (error if table lacks CacheByIDs)
  slots    SELECT partition_id, slots FROM <ks>.cache_by_ids_slots
           WHERE table_id=? AND partition_id IN ?                 LocalQuorum, 100 per batch
  plan     PlanCachedIDs (db/cache/by_ids.go): dedupe; client cycle ≠ server cycle → held ticks ignored;
           skip ID ⇔ trusted slot tick ≠ 0 AND == held tick
  records  WHERE <partition>=? AND <key> IN (…)                   LocalQuorum, 100 per batch
  upd      HeldUpdated(trusted slot tick)
HideCachedPartitions                          partition → 0
```

Response `[]CachedPartition[T]` = `{p, r}`, partitions ascending.

## Write path (`scylla/insert-update.go`, `executeInsertUpdateBatchErased`)

All writes (Insert, Update, UpdateExclude, InsertUpdate*) pass here; no hard delete (deletion = `ss = 0`).

1. `Updated` stamped from the effective clock; row batches at cluster default `LocalOne`.
2. Phase 1: views + text index (errgroup), barrier.
3. Phase 2: `raiseWrittenSlots` (parallel with delta moves): per (partition, Updated)
   `UPDATE <ks>.cache_by_ids_slots USING TIMESTAMP ? SET slots = slots + ? WHERE table_id=? AND partition_id=?`,
   unlogged batch, `LocalQuorum`.

Slots table (deploy creates it if any table has `CacheByIDs`): `PRIMARY KEY ((table_id smallint,
partition_id bigint))`, `slots map<tinyint, int>` (slot → full `Updated`). Scylla converts full
`Updated` to the current cycle's tick at read time; a slot from an earlier year reads as 1.

**Dynamo** (`dynamo/cache_by_ids.go`): slot file `ids-cache/<cycleYear>/<basePK>`, 768 bytes =
256 × uint24 LE ticks. Write: GET + ETag → max → `PutFileIfMatch`, ≤ 30 attempts, jittered sleep
`rand(attempt × 50 ms)`. Read: one file per partition, 10 concurrent; records via consistent
`BatchGetItem`. Requires `SetFileStore`. Allows zero partition columns (Scylla requires one).

## Wire: `?cc=`

Go `DecodeCachedIDsRequest` (`db/cache/by_ids.go`) ↔ TS `encodeCachedIDsRequest`
(`cache/cache-by-ids.wire.ts`); URL-safe custom base64 (`genix-ui/utilities/parsers.ts`).

```
[cycle year − 2000: u8][partition count: uvarint]
per partition (ascending):
  [partition Δ]
  [slot count][slot u8, tick u24 LE] × n        ascending slots, lowest held tick
  [ID count][ID Δ] × n                          ascending IDs
```

Decoder rejects: non-ascending partitions/IDs/slots, tick 0, > 256 slots, count > remaining payload,
uvarint > 2⁶², trailing bytes.

**Static routes** bypass this: `GetStaticRequestIDs` reads IDs only, the server returns every ID asked,
the browser never revalidates and asks the server only for IDs held nowhere.

## Browser (`frontend/packages/genix-ui/cache/`)

`cache-by-ids.svelte.ts`:
- `resolveRecordsByPartition`: returns at once fresh memory hits (`_fch` < `CACHE_TIME` 5 s) and
  tombstones; waits on IDs in flight from another call.
- `lookUpPartition`: memory → IndexedDB (promoted to memory) → if any remaining ID is missing or stale,
  one request with every remaining ID + held tick.
- Held IDs absent from the response = confirmed unchanged → only `_fch` refreshed.
- Fetch error → warning, answer from local data.
- `getRecordByID` buffers 80 ms per route (`buffetMaxTime`).
- Tombstone = `ss === 0` unless `disableTombstone` for the route: kept, answered locally, never sent.

`cache-by-ids.idb.ts`: Dexie DB `<companyID>_cached_ids_<env>` (v2), table `cacheByIDs` keyed
`[storeName+ID]`, `storeName` = route or `route:partition`; value = record + `upd` + `_fch`. Same DB
holds `cacheQueryById` (`cache-query-by-id.ts`, `GETCached`).

`cache-by-ids.wire.ts`: `yearCycleOf`, `cycleStartUpdated`, `toYearTick`, encoder.

## Constants

| Constant | Value | Where |
|---|---|---|
| Slots | 256 (`uint8(ID)`) | `db/cache/by_ids.go` |
| `SlotTrustTicks` | 4 (8 s) | `db/cache/by_ids.go` |
| Cycle start | Jan 1 07:00 UTC | `db/cache/by_ids.go`, `cache-by-ids.wire.ts` |
| `queryCachedIDsMaxBatchSize` | 100 | `scylla/cache_by_ids.go` |
| `slotPutAttempts` / `slotReadConcurrency` | 30 / 10 | `db/cache/by_ids.go` |
| `CACHE_TIME` / buffer | 5 s / 80 ms | `cache-by-ids.svelte.ts` |

## Known weak spots

- 8 s rule is trusted: a later-visible write is missed until its slot moves again.
- Rows at `LocalOne`, slots at `LocalQuorum`: row-before-slot is covered by the 8 s window, not by
  consistency levels.
- Failed slot raise after the row write → error returned, never repaired.
- Fresh writes are read twice (`upd = 0` for 8 s); everything is re-read at the cycle change.
- Shared slots: re-reads grow with the number of requested IDs; packed IDs cluster (stories: low 23
  bits = per-project sequence → story #N of every project shares a slot).
- `GetCachedIDsRequest` refuses partitions ≠ 0 → the frontend's `{[partition]: ids}` form fails against
  every handler today.
- Memory maps keyed by route, not company: a session change without reload (`clearSession` → `goto`)
  could serve the previous company's records (inferred, untested). IndexedDB is per company.
- Static routes re-request missing IDs on every call; a server error returns `[]` silently.
- Misleading comments: `db/schema.go` (`CacheByIDs` mentions a slot file for Scylla); `scylla/reflect.go`
  panic asks for `upd,omitempty` (records use `upd`); `cache-by-ids.wire.ts` points to
  `dynamo/cache_by_ids.go` for the decoder (it is `db/cache/by_ids.go`).

## Files

| File | What |
|---|---|
| `backend/genix-orm/db/cache/by_ids.go` | slots, ticks, trust window, `PlanCachedIDs`, slot files, decoder |
| `backend/genix-orm/db/cache/by_ids_test.go` | protocol tests, shared vectors |
| `backend/genix-orm/scylla/cache_by_ids.go` | schema checks, slots DDL, raise, reads |
| `backend/genix-orm/scylla/cache_by_ids_generic.go` | `GenericRecord`, `QueryCachedGenericByIDs` |
| `backend/genix-orm/scylla/insert-update.go` | write phases, barrier before slot raise |
| `backend/genix-orm/scylla/reflect.go` | `Updated` column requirement |
| `backend/genix-orm/dynamo/cache_by_ids.go` | Dynamo checks, slot-file raise |
| `backend/core/cache.go` | `GetCachedIDsRequest`, `HideCachedPartitions`, static helpers |
| `frontend/packages/genix-ui/cache/cache-by-ids.svelte.ts` | public API, layers, batching, tombstones |
| `frontend/packages/genix-ui/cache/cache-by-ids.wire.ts` | cycle math, encoder |
| `frontend/packages/genix-ui/cache/cache-by-ids.idb.ts` | IndexedDB naming, clearing |
