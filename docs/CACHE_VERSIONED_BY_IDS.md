# Cache Versioned By IDs (Frontend + Backend + ORM)

This guide explains the complete Genix flow for ID-based cache revalidation using ORM cache versions.

Audience: new developers who need to understand implementation, tradeoffs, and extension points.

## 1. Why this exists

Goals:
- Serve records fast from local cache.
- Avoid re-downloading unchanged records.
- Keep UI simple while preserving consistency.

Approach:
- Frontend uses memory + IndexedDB + delta fetch.
- Backend parses compact ID/version payload.
- ORM compares client versions against server versions.
- Backend returns only changed/new rows.

## 2. End-to-end request lifecycle

1. UI asks for a record by ID (`getRecordWithCache`) or several IDs (`getRecordsByIDs`).
2. Frontend resolves local cache first.
3. Frontend builds query params:
- `ids` for local misses.
- `cc-ids` for cached IDs.
- `cc-ver` for cached versions aligned by index with `cc-ids`.
4. Frontend sends request with `cmp` (empresa/partition id).
5. Backend handler calls `ExtractCacheVersionValues`.
6. Backend handler calls `db.QueryCachedIDs`.
7. ORM checks `cache_version` state and fetches only mismatched IDs.
8. Backend returns only changed/new rows.
9. Frontend merges server rows, refreshes timestamps for unchanged cached rows, and persists updates.

## 3. Contract and payload fields

Frontend cache record contract (`IMinimalRecord`):
- `ID`: record id.
- `ccv`: cache version.
- `ss`: status (`1` active, `0` tombstone/deleted).
- `_fch`: local fetch timestamp (unix seconds).

Query params used by backend cache-by-ids handlers:
- `ids`: encoded IDs with no local cache.
- `cc-ids`: encoded IDs with local cache.
- `cc-ver`: encoded versions for `cc-ids`.
- `cmp`: optional company id; backend fallback is authenticated user's `EmpresaID`.

## 4. Compact encoding format

The project does not send IDs as CSV.

Frontend encoder: `frontend/libs/funcs/parsers.ts` -> `concatenateInts(values)`
- Splits numbers into `u8`, `u16`, `u32` buckets.
- Converts each bucket into bytes.
- Base64-url encodes each bucket.
- Joins as `u8.u16.u32`.

Backend decoder: `backend/core/cache.go` -> `parseConcatenatedInts(s)`
- Splits string by `.`.
- Decodes with `base64.RawURLEncoding`.
- Reads little-endian values by section:
- section 0 = `u8`
- section 1 = `u16`
- section 2 = `u32`

Important:
- Order is preserved in each parameter.
- `cc-ver[i]` maps to `cc-ids[i]`.

## 5. Backend handler layer

Main parser: `backend/core/cache.go` -> `ExtractCacheVersionValues(req)`.

Behavior:
- Reads `ids`, `cc-ids`, `cc-ver`, `cmp`.
- Builds `[]db.IDCacheVersion`.
- For each `ids` value: append with `CacheVersion=0`.
- For each `cc-ids` value: append with aligned `cc-ver`, default `0` if missing.
- Returns combined list to one ORM call.

Used by:
- `backend/handlers/productos.go` -> `GetProductosByIDs`.
- `backend/handlers/empresas-usuarios.go` -> `GetUsuariosByIDs`.

Typical handler pattern:
1. Parse request with `ExtractCacheVersionValues`.
2. Validate non-empty IDs.
3. Call `db.QueryCachedIDs(&records, cachedIDs)`.
4. Return fetched rows.

## 6. ORM feature: `SaveCacheVersion: true`

### 6.1 What the flag does

When a table schema enables `SaveCacheVersion: true`, ORM does two things:
- On writes: increments version groups touched by inserted/updated records.
- On reads: assigns current group version (`ccv`) to each returned record.

Examples enabled in project:
- `backend/types/productos.go` (`ProductoTable`).
- `backend/types/users.go` (`UsuarioTable`).

### 6.2 Schema/record requirements

At compile/config time (`backend/db/cache_version.go`), ORM validates:
- Exactly one key column.
- Key type must be `int16|int32|int64`.
- Partition must exist and be `int32|int64`.
- Record must expose a `uint8` cache-version field:
- field named `CacheVersion`, or
- field with json tag `ccv`.

Failure in these checks causes panic during table compilation.

### 6.3 Internal storage model (`cache_version` table)

Table: `cache_version`.

Columns:
- `packed_id bigint` (PK).
- `cached_values blob`.

`packed_id` is:
- High 32 bits: partition (`EmpresaID`).
- Low 32 bits: table hash (`BasicHashInt(tableName)`).

`cached_values` stores compact pairs:
- `[group, version, group, version, ...]`.

### 6.4 Grouping semantics

Cache group is `uint8(recordID)`.

Consequences:
- Only 256 groups per table+partition.
- IDs sharing low 8 bits share one version counter.
- Updating one ID invalidates all IDs in that group.

Version behavior:
- Unseen group defaults to server version `1`.
- Next version increments by 1.
- Overflow wraps (`255 -> 1`).

## 7. ORM hooks in read/write lifecycle

Write hook (`updateCacheVersionsAfterWrite`) is called after successful:
- `Insert`
- `Update`
- `UpdateExclude`

Write algorithm:
1. Collect unique touched groups by packed_id.
2. Load current group-version map from `cache_version`.
3. Increment touched groups once per batch.
4. Save compact map back.
5. Assign resulting `ccv` to in-memory records.

Normal select hook (`assignCacheVersionsAfterSelect`) runs after query scan:
- Ensures partition+key columns are available even if not requested.
- Loads needed cache-version maps.
- Assigns `ccv` on every selected row.

## 8. `QueryCachedIDs` algorithm (delta query path)

`QueryCachedIDs` is a specialized read path for client revalidation.

Phase 1: compare versions without touching main table.
1. Normalize IDs by partition.
2. Load `cache_version` for each partition+table packed key.
3. Compute server group version for each ID.
4. If `clientVersion == serverVersion`, mark as fully cached.
5. Otherwise, mark ID for table fetch.

Phase 2: fetch mismatches only.
1. Build per-partition `IN` query for marked IDs.
2. Fetch rows from main table.
3. Assign `ccv` using already loaded version maps.
4. Return only changed/new rows.

If all are fully cached, returns empty result.

## 9. Frontend implementation details

Main file: `frontend/libs/cache/cache-by-ids.svelte.ts`.

### 9.1 Multi-layer resolution

Order:
1. Memory map per API route.
2. IndexedDB (`cache-by-ids.idb.ts`).
3. Backend delta fetch.

IDB behavior:
- One object store per route/table.
- Key path is `ID`.
- Includes recovery for version mismatch between localStorage and actual IndexedDB version.

### 9.2 Staleness and fetch policy

Staleness rule: `now - _fch > CACHE_TIME`.

Current `CACHE_TIME = 5` seconds.

Important distinction:
- stale means "must revalidate"
- stale does not necessarily mean "changed"

`getRecordsByIDs` only fetches when:
- there are local misses, or
- there are stale cached records

### 9.3 Merge behavior after backend response

After delta response:
- Returned rows get `_fch = now` and are merged into memory.
- Returned rows are upserted into IndexedDB.
- Cached IDs omitted by backend are treated as unchanged and still get `_fch` refreshed.
- Tombstones (`ss=0`) are not returned as active records.

### 9.4 Batching and fan-out

`getRecordByID` buffers requests by table for `buffetMaxTime` (80ms):
- Coalesces many single-ID requests into one table-level fetch.
- Uses one Promise per `table:id` and resolves after batched fetch.
- Prevents list/card UIs from generating N network requests.

### 9.5 `getRecordWithCache` two-step render

`getRecordWithCache(apiRoute, id)` returns `{ record, loading, refresh }`.

Flow:
1. Resolve local record (`memory -> IDB`).
2. If local fresh, return immediately.
3. If stale or missing, call buffered `getRecordByID`.
4. Update reactive record only when changed (`ccv`, `ss`, or optional `upd`).

This gives fast first paint and eventual consistency.

## 10. How to enable for a new entity

Backend steps:
1. Add `uint8` cache-version field in record (`CacheVersion` or json `ccv`).
2. Enable `SaveCacheVersion: true` in schema.
3. Add GET by IDs handler:
- parse with `ExtractCacheVersionValues`
- query with `db.QueryCachedIDs`
4. Ensure response includes at least `ID`, `ss`, `ccv`.

Frontend steps:
1. Ensure type extends `IMinimalRecord`.
2. Use `getRecordByID` or `getRecordWithCache`.
3. Preserve cache-managed fields (`ID`, `ss`, `ccv`, `_fch`).

## 11. Operational and debugging notes

- Multi-tenancy correctness depends on partition (`cmp`/`EmpresaID`).
- `cc-ver` and `cc-ids` must remain index-aligned.
- Grouping by `uint8(ID)` is compact but coarse by design.
- Logging is intentionally verbose for diagnosis:
- frontend uses `console.debug` / `console.warn`
- backend uses `Log` / `fmt.Println`
