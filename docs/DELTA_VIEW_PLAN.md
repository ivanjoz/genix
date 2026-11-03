# Delta sync on a full-record view, with a client-checked window hash (Scylla driver)

Status: **implemented, not deployed** (ORM, backend, genix-ui). Supersedes, for the
Scylla driver only, the S3 delta log (`genix-orm/db/cache/delta_log.go`, the former
`genix-orm/scylla/delta_log.go`). The DynamoDB driver (berryapps) keeps the S3 log unchanged, and
genix-ui keeps speaking it. §10 lists where the implementation departs from this plan.

## 1. Why

- Every write to a `TypeDelta` table appends to an S3 file (Express: HEAD + PUT; Standard: GET +
  conditional PUT). That round trip, its retries under concurrency, and the S3 dependency go away.
- A materialized view on `updated` alone loses late writes (clock skew, a stalled Lambda, view lag).
  A hash of the recent window, checked by the client, detects them and falls back to a full resend.

## 2. Decisions taken

| Topic | Decision |
|---|---|
| Change source | A materialized view per `TypeDelta` holding every stored column, keyed by the partition + pinned `Cols` + `updated` |
| Read | The records come straight from the view: no by-ID read. The planner also serves plain reads on the pinned keys from it |
| Replace threshold | More than **100** changed IDs → send the whole partition. Configurable per index |
| Margin | **10 s**: rows with `updated ≥ C − 10s` are always resent |
| Hash window | **15 min**: `[C − 15min, C − 10s)` |
| Who compares | **The client.** The server returns its hash; the client compares after excluding the returned IDs, and on a mismatch re-requests as a first sync |
| Status vs hash | Status is a pinned key like any other: the server reads and hashes only the call's prefixes and sends the last key's field and values; the client hashes its held records with one of them |
| Hard deletes | Not allowed on `TypeDelta` tables |
| Pinned columns | May change. A change writes one row to `<table>__delta_moves` under the **old** prefix; a sync of that prefix returns the ID as a removal (§3a) |
| Dynamo | Unchanged (S3 log) |

**Pinned columns** are every `Keys` entry of a `TypeDelta` index, the last one included
(`[status][updated]`, as `3115775^:scylla/index_delta_view.go` did). `Delta()`'s values name the
last key's prefixes; a key with no equality and no value fans out over its `FixedValues`, one
parallel read per value. A status change is a move like any other. Real moves today, besides status: `AssigneeID` (story reassignment),
`Period` (expenses and purchase orders, when the comprobante moves the accounting date),
`SameDayMovement` (cash movements, `RestampDocumentPayments`).

## 3. Protocol

**Request.** `<param>=<C>`: `C` is the highest `upd` the client holds for that response key, a
plain integer. The S3 cursor always contains dots, so the two never collide. No param = first sync.

**Server.**
1. Per prefix, in parallel: read the changed records, `updated ≥ C − 10s`, whole from the view, with
   `LIMIT threshold + 1`; read the moves since `C − 15min` with the same limit.
2. More than the threshold of either, summed across prefixes, or a first sync → **replace**: each
   prefix read with `LIMIT <Limit>`, merged newest first and cut at `Limit`.
3. Otherwise the changed records are the answer.
4. `serverHash` = hash of `(ID, version)` over the prefixes' view rows with `updated` in
   `[C − 15min, C − 10s)` (a slim read of ID, updated, version).
5. Append one Metadata record (below).

### 3a. Moves

```sql
CREATE TABLE <keyspace>.<table>__delta_moves (
  partition_id <partition type>, index_tag text, pinned_prefix bigint,
  updated int, id <key type>,
  PRIMARY KEY ((partition_id, index_tag, pinned_prefix), updated, id)
);   -- no TTL: rows never expire (§10)
```

- `pinned_prefix` is the pinned values packed exactly as in the view key (the packed column without
  its `updated` slot). Created only for tables with a `TypeDelta` index that pins a column.
- **Write:** the pre-write read already fetches the stored pinned values. For every record whose
  pinned prefix changes, one row `(old prefix, updated of the write, id)` goes to the moves table,
  after the record rows land (a row published first would remove a record that didn't move if the
  write then failed).
- **Read:** the sync reads the moves slice `updated ≥ C − 15min` of its prefix. Removals = those IDs
  minus the IDs present in the view slice it already read (a record that moved back is in the view
  again, with a newer `updated`). They go out as `<key>_IDsToRemove`, which genix-ui already applies.
  The client excludes them from its hash like any returned ID.
- More than the threshold of moves since `C − 15min` → replace. The cursor's age never is one.
- Known gap, as with the S3 log today: a crash between the rows and the move row leaves the record
  in the old prefix's clients until their next replace.

**Metadata record.** Same envelope as today (negative ID, `mdt` = Safe85), new layout, told apart by
flag bit **8**:

```
[flags u8]                 2 replace · 8 window protocol
[nextCursor uvarint]       highest updated the response leaves the client at
[windowFrom uvarint]       C − 15min (absent on a replace)
[windowTo uvarint]         C − 10s, exclusive
[serverHash u32 LE]
[filterField len u8 + UTF-8]   json name of the sync-filter column ("ss", "state", …); empty = no filter
[filterValue count uvarint][zigzag varint] × count
```

**Client.**
1. Apply records as today (replace drops what the response doesn't bring).
2. Unless it was a replace: `clientHash` = hash of `(ID, ver)` over the held records with `upd` in
   `[windowFrom, windowTo)`, field `filterField` in the values, **excluding the IDs the response
   brought**. Filter the in-memory route snapshot (`readCachedContent`); no IndexedDB index needed.
3. `clientHash ≠ serverHash` → re-request without a cursor (first sync → replace).
4. Store `nextCursor` as the key's cursor.

The flag makes the response self-describing: genix-ui picks the protocol from what the server
answers, so berryapps (Dynamo, no flag 8) keeps working with no config.

**Why the client compares.** A record edited inside the window and again after `C` has moved out of
the server's window but is still in the client's at its old `upd`. Only the client knows which IDs
the response replaced, so only the client can exclude them.

## 4. Hash

Per record: `h(ID, version)` = the murmur3 32-bit finalizer over `lo32(ID) ^ rotl(hi32(ID), 16) ^
version·0x9E3779B1` (exact definition fixed by the shared vectors). Window hash = sum of `h` mod 2³².
Order-independent, BigInt-free in JS. Revives the shape of the old fingerprint (genix-ui `70fcf9c`
`delta-cache.watermark.ts`), over `(ID, version)` instead of `upd`.

Shared vectors in `genix-orm/db/cache/testdata/delta_window_vectors.json`. The Go test and the genix-ui
test both **load the file** (today's genix-ui test copies hex by hand).

## 5. ORM changes (`backend/genix-orm`)

- **View** (`scylla/index_delta_view.go`, `index_view_compile.go`): `TypeDelta` compiles a packed
  range view `PRIMARY KEY ((partition, packed(pinned…, updated)), id)`, payload `version`, `status`
  (the sync-filter column), `CLUSTERING ORDER BY DESC`, lower `gc_grace_seconds` (1 day). The
  trailing slot is `updated` (32 bits, so the packed column is `bigint` when pinned keys exist);
  `planDeltaSlots` already sizes pinned slots from `FixedValues`. A keyless `TypeDelta` is
  `((partition), updated, id)`. The packing keeps a single non-key base column in the view key,
  as the pre-`3115775` delta view did (verify Scylla's current limit on the live cluster first).
- **Write path** (`scylla/insert-update.go`, `delta_log.go`): drop `appendDeltaLogs` and the log
  appends. Keep the pre-write read and `version` stamping. Write the move rows (§3a); deploy
  creates `<table>__delta_moves`.
- **Read** (`execDelta`): the protocol in §3 and §3a. `Index.DeltaReplaceThreshold` (0 → 100).
- **`db`**: `DeltaRead{CursorUpdated int32, SyncFilterValues}` replaces `cache.DeltaCursor`;
  `TableStruct.Delta(cursorUpdated int32, values...)`. `db/schema.go` docs for `TypeDelta` rewritten.
- **`db/cache`**: `DeltaWindowHash`, the Metadata encoder, vectors. `delta_log.go` and its codec
  stay (Dynamo).
- Docs: `scylla/ORM_INTERNALS.md`, `scylla/RATIONALE.md` (window protocol, moves table).

## 6. Backend changes (`backend`)

- `core/responses.go` `GetDeltaCursor` → parses `C` (int32). The 19 `.Delta(` call sites change
  mechanically.
- Remove the `delta-log/` prefix of genix's bucket after deploy.

## 7. genix-ui changes (`frontend/packages/genix-ui`, shared with berryapps)

- `cache/delta-log.wire.ts`: decode the flag-8 Metadata.
- `cache/delta-cache.cursor.ts` / `delta-cache.fetch.ts`: on flag 8, compute `clientHash`, store the
  cursor as a number, re-request on mismatch. The S3 path stays as is.
- `cache/delta-cache.types.ts`: the cursor becomes `DeltaCursor | number`.
- Tests load the shared vectors. `cache/DELTA_CACHE.md`, `cache/RATIONALE.md` updated.
- Commit and push inside the submodule; berryapps picks it up when it bumps.

## 8. Migration

Pre-alpha, no production data: no compatibility code of any kind. Deploy drops and recreates every
`TypeDelta` view; if anything stored doesn't fit, wipe the keyspace and reseed. Browsers' IndexedDB
caches are wiped (bump the route `ver`).

## 9. Implementation order

1. ORM: view compile, write path (moves), `execDelta`, `db` API, `db/cache` hash + Metadata + vectors.
2. Backend: `GetDeltaCursor`, call sites, deploy, reseed.
3. genix-ui: flag-8 branch, hash, tests, docs.

## 10. As implemented

- **View key.** Scylla allows one non-key base column in a view's key (checked live), so the view is
  `PRIMARY KEY ((part), zz_dlt_<pinned>, id)` `DESC`, where the virtual bigint packs the pinned
  slots over 31 bits of `updated`; keyless: `updated` alone (`zz_dlt_updated`). Name
  `<table>__dlt_<keys>_view`. Every write rewrites the packed column, from the stored pinned values a
  partial update omits.
- **Full records, in the planner.** The view holds every stored column, so `Delta()` reads records
  from it and never by ID. It is a planner range view on `[part, keys…, updated]`, a key without an
  equality fanning out over its `FixedValues`; no `ALLOW FILTERING`. Epics, stories and sprints
  (`ProjectID`) and the 7 tables reading by Status (or State) dropped their explicit `TypeView`.
- **Every key pinned.** An intermediate version treated the last key as a filter outside the packed
  key (whole-prefix reads, records returned in every status, a client "leaving by filter" rule, a
  replace on every sync of the Anuladas tab). Restored to the original layout: the last key is a
  slot too, `Delta()`'s values are prefixes read in parallel (`resolveDeltaPrefixes`), and a status
  change is a move. `GetSaleOrders`/`GetExpenses` dropped their `records_IDsToRemove` post-filter.
- **Removed IDs travel in the Metadata**, not as `<key>_IDsToRemove`: one more field after the
  filter values, `[removed count uvarint][ID delta uvarint] × count`, ascending.
- **Limit.** A replace keeps the newest `updated` first, up to the query's `Limit`. A `Select`
  projection is honoured on the view reads.
- **Vectors.** genix-ui is its own repo, so `delta-window.wire.test.ts` copies the vectors of
  `delta_window_vectors.json` (generated by an independent Python implementation) instead of
  loading the file.
- **Mismatch.** The key loses its cursor and the route is marked `forceNetwork`; `syncDeltaCache`
  re-requests at once. An external delta (`applyExternalDeltaResponse`) leaves it to the next read.
- **No age rule.** The moves table has no TTL, so a cursor of any age syncs incrementally; a replace
  comes only from a first sync or more than the threshold of changed or moved records. The changed
  rows and the moves are read with `LIMIT threshold + 1`, so the decision costs at most that many
  rows. Cost: one moves row per pinned-value change, kept forever (rare, a few bytes each).
- **Pending:** wipe and reseed the keyspace (old views and the old delta-log S3 prefix are not
  dropped by deploy), bump the route `ver`, delete the `delta-log/` prefix.
