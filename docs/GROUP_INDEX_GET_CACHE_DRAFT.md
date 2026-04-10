# Group Index GET Cache Draft

## Goal

Implement `GETWithGroupCache(route, uriParams)` for endpoints backed by `db.QueryIndexGroup`.

The client keeps grouped query partitions in IndexedDB and sends their freshness metadata to the backend as:

- `cc-gh`: cached group hashes encoded as unsigned `uint32`.
- `cc-upc`: cached update counters aligned by index with `cc-gh`.

The backend uses those values with `query.IncludeCachedGroup(groupHash, updateCounter)`. If the server-side `update_counter` is unchanged, the ORM skips returning records for that group. The frontend then rebuilds the response from IndexedDB for unchanged groups and stores any changed groups returned by the server.

## Backend Contract

`db.RecordGroup[T]` already returns the metadata needed by the client:

```go
type RecordGroup[T any] struct {
	IndexID          int16   `json:"ig"`
	GroupHash        int32   `json:"id"`
	IndexGroupValues []int64 `json:"igVal"`
	Records          []T     `json:"records"`
	UpdateCounter    int32   `json:"upc"`
}
```

Frontend must persist:

- `id` as `groupHash`.
- `upc` as `updateCounter`.
- `igVal.join("_")` as the stable group key for local lookup.
- `records` as the cached payload for that group.

## Backend Parser Fix

`ExtractGroupIndexCacheValues` has two issues:

- `GroupHash` is an `int32`, but the frontend must send it as unsigned `uint32` because compact integer packing does not support negative numbers.
- `UpdateCounter` currently reads `updateCounters[1]`; it should read the aligned value `updateCounters[i]`.

Draft fix:

```go
func ExtractGroupIndexCacheValues(req *HandlerArgs) ([]db.GroupIndexCache, error) {
	groupHashes := parseConcatenatedInts(req.GetQuery("cc-gh"))
	updateCounters := parseConcatenatedInts(req.GetQuery("cc-upc"))

	records := make([]db.GroupIndexCache, 0, len(groupHashes))
	for index, encodedGroupHash := range groupHashes {
		if index >= len(updateCounters) {
			continue
		}

		// The client sends int32 hashes through uint32 packing; this conversion restores the sign.
		groupHash := int32(uint32(encodedGroupHash))
		records = append(records, db.GroupIndexCache{
			GroupHash:     groupHash,
			UpdateCounter: int32(updateCounters[index]),
		})
	}

	Log("records extracted:", len(records))
	return records, nil
}
```

The conversion works for negative hashes. Example: frontend sends `4294967295` for `-1`; Go reads it as `uint32(4294967295)` and `int32(...)` becomes `-1`.

## IndexedDB Shape

Add a small group-cache store beside the existing delta-cache tables.

Database name:

```ts
`${Env.getEmpresaID()}_group_cache_${Env.enviroment}`
```

Example:

```ts
1_group_cache_u4nagj
```

The company and API endpoint partition belong in the database name, not in every row key.
The same rule applies to the delta cache database:

```ts
`${companyID}_delta_cache_${env}`
```

Recommended store:

```ts
interface IGroupCacheRow {
	queryShape: string
	key: string
	id: number
	upc: number
	ig: number
	records: any[]
	fetchTime: number
}
```

Dexie schema:

```ts
groupCache: '[queryShape+key],[queryShape+id+upc],queryShape'
```

Rationale:

- `queryShape` scopes a cache row to the GET shape, not to a specific filter value.
- `key` is `igVal.join("_")`, so a changed backend group maps to exactly one local row.
- `id` and `upc` mirror the backend JSON names and are stored separately from `records`, so freshness params can be built without reading record payloads.
- `id` stays as signed `int32` in IndexedDB. It is converted to unsigned only when building `cc-gh`, because the compact query-param encoder only supports unsigned integers.

## Query Shape

`queryShape` is the plain string identity of the query shape:

```ts
route + "|" + sortedParamNames.join("|")
```

It must not include parameter values.

Example:

```ts
makeGroupQueryShape("sale-order-query", {
	"client-id": "10",
	"fecha-start": "20501",
	"fecha-end": "20515",
})
// sale-order-query|client-id|fecha-end|fecha-start
```

This intentionally avoids hashing. The IndexedDB key is longer, but it is collision-free and easier to inspect in DevTools.

## GETWithGroupCache Flow

1. Validate `route` and `uriParams`.
2. Build `queryShape` from the route and sorted param names.
3. Build the network route from actual `uriParams`.
4. Read only cached metadata rows for `queryShape`.
5. Add `cc-gh` and `cc-upc` to the request:
   - `cc-gh = concatenateInts(id >>> 0)`
   - `cc-upc = concatenateInts(upc)`
6. Fetch the backend route with normal `GET`.
7. Normalize the backend response to `RecordGroup[]`.
8. For each returned group:
   - compute `key = group.igVal.join("_")`
   - replace the local row for `[queryShape, key]`
9. Build final groups:
   - start with cached rows for the requested `queryShape`
   - overlay changed rows returned by the backend
   - return only groups whose `key` belongs to the current query result

Important detail: step 9 needs a way to avoid returning cached groups that are in the same shape but not in the current filter values. Because `queryShape` does not include values, the safest first implementation is:

- send metadata for all rows in the shape
- after the server returns changed groups, merge by key
- filter final rows by the exact candidate keys for the current query

The frontend cannot currently compute candidate `igVal` keys unless it duplicates the backend `QueryIndexGroup` planner. To avoid that duplication, the backend should return zero-record groups for unchanged groups, containing `id`, `igVal`, `ig`, and `upc`. Then the client knows exactly which cached keys belong to this request.

## Backend Response Adjustment

Current ORM behavior returns no group when a cached group is unchanged. For group cache to be exact without duplicating query planning in frontend, the ORM should return metadata-only groups for unchanged cache hits:

```json
{
	"ig": 123,
	"id": -1293912,
	"igVal": [20501, 10],
	"records": [],
	"upc": 44
}
```

Then frontend behavior is deterministic:

- If `records.length > 0`, replace the cached records for that key.
- If `records.length === 0`, read records from IndexedDB for that key.
- If there is no cached row for an unchanged group, treat it as a cache miss and force a refetch without `cc-gh/cc-upc`.

This requires changing `filterIndexGroupFetches` / `execIndexGroupQuery` so unchanged cached groups are preserved as `RecordGroup` metadata instead of being completely skipped.

## Frontend API Draft

```ts
export const GETWithGroupCache = async <T>(
	route: string,
	uriParams: { [key: string]: string },
): Promise<dbRecordGroup<T>[]> => {
	// The shape key ignores values, so all equivalent filter forms share metadata.
	const queryShape = makeGroupQueryShape(route, Object.keys(uriParams))
	const cachedGroups = await readGroupCacheMetadata(queryShape)

	const requestParams = new URLSearchParams(uriParams)
	if (cachedGroups.length > 0) {
		requestParams.set("cc-gh", concatenateInts(cachedGroups.map(group => group.id >>> 0)))
		requestParams.set("cc-upc", concatenateInts(cachedGroups.map(group => group.upc)))
	}

	const responseGroups = await GET({ route: `${route}?${requestParams.toString()}` })
	return await mergeGroupCacheResponse(queryShape, responseGroups)
}
```

## IndexedDB Metadata Reads Without Loading Records

Yes, but only if the values you need are stored as keys or indexed fields.

Current `delta-cache` rows use:

```ts
cacheRecords: '[cR+rK+ID],[cR+ss]'
```

That means IndexedDB can return primary keys like `[cR, rK, ID]` without loading `E`. It cannot return `upd` without loading `E`, because `upd` only exists inside the serialized record payload.

For group cache, store `id` and `upc` on the metadata row itself. Then freshness reads use a key/index-only query and do not deserialize `records`.

For normal delta records, if we need `upd`, `ID`, and `rK` without reading `E`, add metadata outside `E`, for example:

```ts
interface ICacheRecordRow {
	cR: number
	rK: string
	ID: CacheRecordID
	upd: number
	ss: number
	E: any
}
```

Then add an index that includes `upd` in the key if key-only reads are required:

```ts
cacheRecords: '[cR+rK+ID],[cR+rK+upd+ID],[cR+ss]'
```

Without that schema change, Dexie must load each row value to read `E.upd`.

## First Implementation Checklist

- [ ] Fix `ExtractGroupIndexCacheValues` index alignment and signed group hash conversion.
- [ ] Add group-cache Dexie types and store.
- [ ] Add metadata-only read helper for `queryShape`.
- [ ] Add upsert helper keyed by `[queryShape, key]`.
- [ ] Implement `GETWithGroupCache`.
- [ ] Adjust backend ORM to return metadata-only groups for unchanged cached groups.
- [ ] Add tests for negative `GroupHash`, aligned `UpdateCounter`, and unchanged group responses.
