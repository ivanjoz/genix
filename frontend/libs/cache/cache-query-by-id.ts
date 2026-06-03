import { GET } from '../http.svelte'
import {
	readQueryCacheRow,
	upsertQueryCacheRow,
	deleteQueryCacheRow,
	type ICacheQueryByIdRow,
} from './cache-by-ids.idb'

/**
 * Goal:
 * Cache a whole list-endpoint response keyed by its `route`, invalidated by a parent
 * record's `updated` watermark. Unlike cache-by-ids (per-record `ccv` delta protocol)
 * or delta-cache (watermark status maps), this is one route -> one stored blob -> one
 * `updated` stamp. Freshness is driven solely by `updated`: when it changes (the parent
 * was modified), the cache misses and the records are re-fetched.
 *
 * Layers: in-memory map (fast path) -> IndexedDB (survives reloads) -> server.
 */

const LOG_PREFIX = '[cache-query-by-id]'

const memoryCache = new Map<string, ICacheQueryByIdRow<any>>()
const inFlightByRoute = new Map<string, Promise<any[]>>()

const nowSeconds = () => Math.floor(Date.now() / 1000)

// Default extractor mirrors cache-by-ids: accept a raw array, else `{ records: [...] }`.
const defaultPick = <T>(payload: any): T[] => {
	if (Array.isArray(payload)) return payload as T[]
	if (Array.isArray(payload?.records)) return payload.records as T[]
	console.warn(`${LOG_PREFIX} unexpected server response shape; returning empty list`, payload)
	return []
}

/**
 * Fetch `route` through the route-keyed cache. Returns the cached records while the
 * stored `updated` strictly equals `updated`; any mismatch re-fetches and overwrites.
 *
 * @param pick optional extractor to pull the list out of a wrapped payload
 *             (e.g. `p => p.movimientos || []`). Defaults to the array/`.records` heuristic.
 */
export const GETCached = async <T = any>(
	route: string,
	updated: number,
	pick: (payload: any) => T[] = defaultPick,
): Promise<T[]> => {
	// 1) Memory hit — same watermark means the cached blob is still valid.
	const memoryRow = memoryCache.get(route)
	if (memoryRow && memoryRow.updated === updated) {
		return memoryRow.records as T[]
	}

	// 2) Share a single fetch across concurrent callers for the same route.
	const inFlight = inFlightByRoute.get(route)
	if (inFlight) return inFlight as Promise<T[]>

	const fetchPromise = (async (): Promise<T[]> => {
		// 3) IndexedDB hit — promote into memory when the watermark still matches.
		const idbRow = await readQueryCacheRow<T>(route)
		if (idbRow && idbRow.updated === updated) {
			memoryCache.set(route, idbRow)
			return idbRow.records
		}

		// 4) Server fetch (plain GET — this helper IS the cache, so it must not double-cache
		//    through the service-worker delta cache).
		const payload = await GET({ route })
		const records = pick(payload)

		// 5) Persist the new blob to both layers, stamped with the caller's watermark.
		const nextRow: ICacheQueryByIdRow<T> = { route, updated, records, _fch: nowSeconds() }
		memoryCache.set(route, nextRow)
		await upsertQueryCacheRow(nextRow)
		return records
	})()

	inFlightByRoute.set(route, fetchPromise)
	try {
		return await fetchPromise
	} finally {
		if (inFlightByRoute.get(route) === fetchPromise) inFlightByRoute.delete(route)
	}
}

// Explicit invalidation for callers that mutate the child list directly.
export const invalidateQueryCache = async (route: string): Promise<void> => {
	memoryCache.delete(route)
	await deleteQueryCacheRow(route)
}

// Drop the in-memory layer; called from clearCacheByIDs() so a global reset also clears this.
// IndexedDB rows live in the cache-by-ids database, which that reset already drops.
export const clearQueryByIdMemoryCache = (): number => {
	const clearedRoutes = memoryCache.size
	memoryCache.clear()
	inFlightByRoute.clear()
	return clearedRoutes
}
