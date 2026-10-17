# Plan: phase 5 — the scylla driver on dynamo's cache protocols, and genix-ui `main`

Detail for phase 5 of `PLAN_COLBIN_0_6.md`. Phases 1–4 are done (genix `95187cee`, fareward
v0.2.1, genix-s3 `28094b0`, genix-orm `86bdea2`).

## What exists today

**Scylla:**
- **By-IDs:** 8 tables have `SaveUpdatedVersion`: client_provider, client_provider_snapshot,
  product, invoice_summary, company_secrets, invoice_document, asset and users. The handlers
  read them through `ExtractUpdatedVersionValues` and `QueryCachedIDs`, or
  `QueryCachedGenericByIDs`. There are 8 routes.
- **Delta:** 26 `TypeDelta` declarations across 20 files, read by 21 `.Delta(upv, statuses...)`
  calls in 14 handlers. Each is a packed view on (keys…, `updated_version`).
- **The `updated_version` sequence:** reserved once per partition on every write. Three things
  use it: the delta views, the slot table, and the index groups' `update_counter` (8 tables use
  `UseIndexGroup`).
- **`updated`:** stamped only when the record's value is zero (fill-if-zero). dynamo overwrites
  it on every write.
- **Writes:** the base write is blind. The only read before a write is in `Merge`. There is no
  public Delete.

**genix-ui `main`:**
- **By-IDs:** the request is `?cc=` and the response is `[{p, r}]`. Records carry `ID`, `upd`
  (the slot tick) and `ss`.
- **Delta:**
  - The client sends `?<key>=<bucket>.<bytes>...`, and nothing at all on a first sync.
  - Records carry `ver`.
  - The response ends with `mdt` records, Safe85 of the metadata and the file.
  - The old `up=<upv>.<upd>` watermark is no longer sent.
- **Frontend changes are small:**
  - Every GetHandler re-syncs once (IndexedDB v8).
  - 4 `upv` type fields and 3 test fixtures change.
  - `roosterjs` leaves `package.json`, the tiptap packages come in, and `bun.lock` is
    refreshed.
  - The stale `webpage/static/sw.js` is rebuilt.
  - No Button or HTMLEditor call site breaks.

## Decisions I need from you

**D1. How scylla gets a per-record `Version`.** Scylla has no atomic increment on a regular
column; dynamo raises `ver` inside the write.
- **(a) Recommended:** before each write to a table with a delta index, read the stored
  `version` of the written keys: one `SELECT key, version … WHERE part = ? AND key IN (…)` per
  partition. Then write `stored + 1`.
  - Cost: one extra round trip per write call.
  - Two concurrent writers of one record can store the same version. The client only skips a
    log entry whose version equals the one it holds, so the risk is that one write is not
    re-applied until the next change to that record.
- (b) LWT (`IF version = ?`): exact, but it costs a Paxos round per row and needs batching
  workarounds.
- (c) Fold the write's SUnixTime into 13 bits: no read, but two writes in the same 2-second
  tick get the same version.

**D2. `updated` is always stamped.** Both protocols need it to be the time of the write.
Recommended: overwrite it on every write, as dynamo does. Fill-if-zero goes away.

**D3. Index groups lose `updated_version`.** Recommended: their `update_counter` becomes the
write's `updated` (SUnixTime). That is also what `RecalcGroupIndexHashes` reads.

**D4. How a first sync, or a replace, lists the view.** Today it reads the packed delta view.
- **(a) Recommended:** compile each `TypeDelta` into the same packed range view minus the
  `updated_version` slot. It is keyed on the pinned keys and the sync-filter column, and a
  first sync reads it with `status IN (…)` through the existing select planner.
- (b) Scan the partition and filter in Go. This needs no view, but every first sync reads
  inactive rows too.

**D5. The meaning of the keys.** dynamo reads them as: the last key is the sync filter, and the
others are pinned (each pinned value gets its own log). Three genix declarations list Status
first, so they are reordered:
- `client_provider`: `Cols(Status, Type)` → `Cols(Type, Status)`
- `generales`: `Cols(Status, ListID)` → `Cols(ListID, Status)`
- the `Cols(WarehouseID, Status)` pair in product-stock is already in order.

Keyless `{Type: TypeDelta}` (4 tables) stays legal in scylla: the log is per partition and
there is no sync filter. dynamo keeps refusing it.

**D6. Routes that are not backed by a delta log.**
- genix-ui no longer sends the `up` watermark, so a `useCache` route that isn't backed by
  `.Delta()` gets a first sync every time.
- That route must answer `db.DeltaReplaceMetadata()`, or deleted records never leave the
  browser.
- Recommended: add that metadata record to every such route. They return their whole list on
  every sync, as they effectively do now.
- I will inventory them, about 18 routes.

**D7. Where the file store lives for genix.** It needs a `[file_store]` bucket.
- **Recommended:** the genix-s3 bucket under root `orm/`, so the wipe in phase 6 empties one
  bucket.
- Alternative: a new bucket.

**D8. Read consistency.** The cluster reads at `LocalOne`. A by-IDs or delta read that follows a
slot/log write could hit a replica that hasn't caught up. Recommended: read the records at
`LocalQuorum` on these two paths only.

## Steps (once D1–D8 are settled)

1. **genix-orm: new module `genix-orm/cache`, which depends only on `utils`.**
   - Moved from `dynamo/`:
     - the year cycle and ticks, and the slot-file read and If-Match raise;
     - `DecodeCachedIDsRequest`, `CachedID`, `CachedIDsRequest` and `CachedPartition`;
     - the delta-log codec and its vectors;
     - the bucket, seal and replace rules, and the cursor plan (which buckets to read, and
       whether it is a replace);
     - parallel appends of `fileKey → entries`, and reading the files;
     - `deltaMetadataBytes` and `DeltaReplaceMetadata`, `DeltaCursor`;
     - the `FileStore` holder and the `Now` clock.
   - dynamo keeps its own keys, pinned folders, log IDs, BatchGetItem and write hooks, and
     re-exports what berryapps imports, so berryapps' code does not change.
   - dynamo's tests pass unchanged.
   - Then berryapps' `backend/genix-orm` is updated, and `go build ./...` plus the dynamo
     tests run in berryapps **before** genix-orm is pushed.
2. **Scylla schema:**
   - `CacheByIDs` replaces `SaveUpdatedVersion`.
   - Managed columns `updated` (always stamped, D2) and `version` smallint (D1).
   - A virtual `Metadata *string json:"mdt,omitempty"` on delta tables.
   - `TypeDelta` compiles to a listing view (D4), plus log metadata: tag, pinned columns,
     sync filter, log ID columns and scale.
   - Deleted: the `cache_updated_version` table and files, `index_delta_view.go`'s version slot,
     the `updated_version` sequence, `IDUpdatedVersion`, `UpdatedVersionCol`, `db.Delta(upv…)`
     and `maxDeltaVersionValue`.
3. **Scylla write path:**
   - Order: stamp `updated`, read and raise `version` (D1), write the CQL.
   - After the existing phase 1 syncs, in phase 2: raise the slot files and append the delta
     logs. A status-0 write appends a deleted entry. A failed append fails the write.
   - Index groups take `updated` (D3).
4. **Scylla read path:**
   - `QueryCachedIDs(request) ([]CachedPartition[E], error)` and `QueryCachedGenericByIDs`
     with the same request: read the slot files, select the changed IDs by
     `part = ? AND key IN (…)`, and stamp `upd` with the slot tick.
   - `.Delta(cursor, syncFilter...)`: on a first sync or a replace, list the view; otherwise
     read the logs past the cursor and select the named IDs. Append the `mdt` records.
5. **genix backend:**
   - `core/filestore` on S3, installed at boot (D7), from berryapps' Express/compatibility
     model.
   - `req.GetDeltaCursor(key)` and `req.DecodeCachedIDs()`.
   - Port the 8 by-IDs routes and the 14 delta handlers.
   - Add the replace metadata to the non-delta routes (D6).
   - Record structs: drop `UpdatedVersion`/`upv`; add `Version int16 json:"ver"` and
     `Metadata`; reorder the D5 keys.
   - Delete the `cc-ids`/`cc-ver` parsing and `ExtractUpdatedVersionValues`.
   - Regenerate `exec/controllers.generated.go` and the frontend `//STRUCT` mirrors.
6. **genix frontend:**
   - Move genix-ui to `main` (`f7525df`).
   - Drop `roosterjs`, run `bun install`, fix the `upv` types and fixtures, rebuild `sw.js`.
   - Run `bun run check`.
7. **Tests:**
   - The shared package's vectors, and dynamo's suite (in genix-orm and in berryapps).
   - Scylla unit tests for compile, stamping, version and the entries appended (with
     `MemoryFileStore`).
   - A live ScyllaDB integration test behind `GENIX_ORM_SCYLLA_TEST`: write → by-IDs → delta
     past a cursor → soft delete → replace.

Phases 6 (wipe) and 7 (verification) follow as written in `PLAN_COLBIN_0_6.md`.
