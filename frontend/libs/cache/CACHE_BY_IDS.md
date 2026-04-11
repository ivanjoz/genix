# Cache By IDs

This folder contains the `cache_by_ids` flow used to resolve records by `ID` with:

- in-memory cache
- IndexedDB persistence
- backend delta validation using cache-version (`ccv`)

This is the cache used by features that ask for many individual records by `ID` and want to avoid downloading unchanged rows repeatedly.

## Files

- `cache-by-ids.svelte.ts`
  Public API for reads, batching, stale detection, and server delta fetch.
- `cache-by-ids.idb.ts`
  IndexedDB persistence layer keyed by `ID`.

## Frontend Flow

### 1. Caller asks for one or many IDs

Main APIs:

- `getRecordsByID(apiRoute, ids)`
- `getRecordByID(apiRoute, id)`
- `getRecordWithCache(apiRoute, id)`

### 2. Cache checks happen in this order

For each requested ID:

1. memory map
2. IndexedDB
3. backend request only if missing or stale

If a record is found in IndexedDB, it is promoted to memory.

### 3. Stale detection

Each cached record stores:

- `ID`: unique record identifier
- `ccv`: cache-version returned by backend
- `_fch`: fetch timestamp in seconds
- `ss`: status, where `0` means deleted/tombstone
- `upd`: updated value used by `getRecordByIDUpdated`

A record is considered stale when:

```ts
nowSeconds() - _fch > CACHE_TIME
```

Fresh local records return immediately without network.

### 4. Delta request sent to backend

When the frontend needs server validation, it sends:

- `ids`
  IDs that do not exist locally
- `cc-ids`
  IDs that exist locally
- `cc-ver`
  cache-version for each `cc-id`

Important:

- `cc-ids` and `cc-ver` are positional pairs
- they must stay aligned after compact encoding
- `cc-ver` must always fit in `uint8`, so it must be `0..255`

If backend does not return a cached ID, frontend treats that row as unchanged and only refreshes `_fch`.

## Backend Flow

The backend endpoint receives the IDs and calls:

```go
err := db.QueryCachedIDs(&records, cachedIDs)
```

`QueryCachedIDs` compares the client `ccv` against the current server cache-version state.

Behavior:

- matching version: row is omitted from response
- different version: row is selected from the main table and returned

So the response contains only:

- missing records
- changed records

Unchanged cached records are not sent again.

## Requirements To Use This

### Frontend record shape

The frontend record type must include at least:

```ts
export interface IMinimalRecord {
	ID: number
	ccv?: number
	ss: number
	_fch?: number
	upd: number
}
```

Minimum practical requirements:

- `ID`
  Required. Used as cache key.
- `ccv`
  Required for backend delta validation.
- `ss`
  Required. `0` is treated as deleted.
- `_fch`
  Internal frontend timestamp used for stale detection.
- `upd`
  Required only if you use `getRecordByIDUpdated`.

### Backend schema requirements

The backend table schema must enable cache-version support:

```go
func (t ProductoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "productos",
		Partition:        t.EmpresaID,
		SaveCacheVersion: true,
		Keys:             []db.Coln{t.ID.Autoincrement(0)},
	}
}
```

Requirements enforced by the ORM:

- `SaveCacheVersion: true`
- exactly one key column
- key column must be `int16`, `int32`, or `int64`
- table must have a partition column
- partition column must be `int32` or `int64`

### Backend response struct requirements

The response struct must expose a cache-version field:

```go
type Producto struct {
	ID           int32 `json:",omitempty"`
	Status       int8  `json:"ss,omitempty"`
	Updated      int32 `json:"upd,omitempty"`
	CacheVersion uint8 `json:"ccv,omitempty"`
}
```

Requirements:

- field name `CacheVersion` or JSON tag `ccv`
- type must be `uint8`
- `ID` must be present in the response

If `ccv` is missing from the response, frontend cannot validate cached rows correctly.

## Expected Endpoint Pattern

The `*-ids` endpoint usually does this:

1. parse `ids`, `cc-ids`, `cc-ver`
2. build `[]db.IDCacheVersion`
3. call `db.QueryCachedIDs`
4. return only changed/new rows

Example:

```go
func GetProductosByIDs(req *core.HandlerArgs) core.HandlerResponse {
	cachedIDs := req.ExtractCacheVersionValues()
	if len(cachedIDs) == 0 {
		return req.MakeErr("No se enviaron ids a buscar.")
	}

	productos := []negocioTypes.Producto{}
	if err := db.QueryCachedIDs(&productos, cachedIDs); err != nil {
		return req.MakeErr("Error al obtener los productos.", err)
	}

	return core.MakeResponse(req, &productos)
}
```

## IndexedDB Rules

Each route gets its own object store.

- store name = `apiRoute`
- key path = `ID`

IndexedDB stores the full record object, including:

- `ID`
- `ccv`
- `_fch`
- `ss`
- domain fields

## Batching Rules

`getRecordByID` uses a small buffer window (`buffetMaxTime`) so many card/component requests become one backend request per route.

This means:

- many components can ask for records independently
- frontend still sends one batched request per table/route

## Conditions For Correct Behavior

This cache works correctly only if all of these are true:

1. frontend sends `ID` and `ccv` for cached rows
2. backend response includes the correct `ccv`
3. backend schema has `SaveCacheVersion: true`
4. backend response model exposes `CacheVersion uint8` as `ccv`
5. `cc-ver` never exceeds `255`
6. `cc-ids` and `cc-ver` stay aligned in the same order
7. returned records are merged into memory and IndexedDB
8. unchanged cached records refresh `_fch`

If any of these fail, the usual symptom is:

- backend keeps returning the same rows again and again

## Important Limitation

Current backend implementation groups cache-version state by `uint8(id)`.

That means different IDs can share the same cache-version bucket when:

```text
uint8(idA) == uint8(idB)
```

Example:

- `26`
- `282`
- `538`

All share the same group key modulo `256`.

This is compact and fast, but it means unrelated rows can invalidate together.

## Debugging Checklist

If the same rows keep coming back from backend:

1. check frontend request snapshot for `ID`, `ccv`, `_fch`
2. check IndexedDB stored value for the same `ID`
3. check backend received `CacheVersion`
4. check backend response `ccv`
5. confirm `cc-ver` values are `0..255`
6. confirm `cc-ids` and `cc-ver` stay aligned

Typical failure patterns:

- IndexedDB has correct `ccv`, but backend receives another one
  Usually transport ordering/alignment bug.
- backend returns rows with no `ccv`
  Response struct is missing `CacheVersion`.
- every cached row always fetches again
  `_fch` is not refreshed or local rows are always stale.

## Related References

- `backend/docs/ORM_DATABASE_QUERY.md`
- `backend/db/cache_version.go`
