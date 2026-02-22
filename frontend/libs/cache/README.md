# Cache By IDs

This folder groups the ID-based cache stack used by product search cards and any feature that resolves records by `ID`.

## Why this exists

- Reduce repeated API calls for the same IDs.
- Support fast first paint from local data (memory or IndexedDB).
- Revalidate stale entries in background with delta requests.
- Batch many `getRecordByID` calls into one backend request per table.

## Files

- `cache-by-ids.svelte.ts`
  - Public API for cache reads.
  - 3-layer read flow: memory -> IndexedDB -> server delta.
  - Buffer window (`buffetMaxTime`) that batches per-table ID lookups.
  - Includes `getRecordWithCache(apiRoute, id)` for transparent two-step render.
- `cache-by-ids.idb.ts`
  - IndexedDB adapter wrapper on top of `typed-idb`.
  - Creates one object store per table (`tableName`, keyPath `ID`).
  - Handles DB version recovery when local version metadata is behind actual IDB version.

## Core behavior

1. Local resolution:
- Try memory map first.
- If miss, try IndexedDB and promote hit to memory.

2. Staleness:
- Record is stale when `now - _fch > CACHE_TIME`.
- Fresh local records return immediately.

3. Server revalidation:
- For stale/missing records, requests include:
  - `ids` for local misses.
  - `cids` + `ccv` for cached records to validate delta.
- Backend returns only new/changed records.
- Cached IDs omitted by backend are treated as unchanged and their `_fch` is refreshed.

4. Buffering:
- `getRecordByID` queues IDs for up to `buffetMaxTime` ms.
- A single flush executes one `getRecordsByIDs` per table, then fans out results by `ID`.
- This avoids parallel per-card network requests.

## Two-step render pattern

- `getRecordWithCache` returns `{ record, loading, refresh }`.
- Step 1: render local record immediately if present.
- Step 2: if stale/missing, revalidate through buffered pipeline and update only when values changed.

This keeps components simple: card UI consumes a reactive ref and does not manage cache internals.
