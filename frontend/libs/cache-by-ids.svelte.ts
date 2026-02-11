import { GET } from './http.svelte'

/**
 * Goal:
 * Resolve records by IDs using a 3-layer strategy:
 * 1) in-memory map, 2) IndexedDB persistent cache, 3) server delta sync.
 * It sends `ids`, `cached`, and `ccv` so backend returns only new/changed records.
 */
import { concatenateInts } from "./funcs/parsers"
import { readRecordsFromIDBByIDs, upsertRecordsIntoIDB } from "./cache-by-ids.idb"

const CACHE_TIME = 5

export interface IMinimalRecord {
	ID: number /* ID f the record */ 
	ccv: number /* cache version a number from 0 to 255 */
	ss: number /* status: 1 active, 0 deleted */
	_fch: number /* fetched: when the record was last fetched (in seconds) */
}

const tableIDRecordCache: Map<string,Map<number,IMinimalRecord>> = new Map()

const LOG_PREFIX = "[cache-by-ids]"

export type CacheByIDsFetchFromServer = <T extends IMinimalRecord>(
	apiRoute: string,
	uriParams: string,
) => Promise<T[]>

// Configure this from your app when the backend endpoint is ready.
let fetchFromServer: CacheByIDsFetchFromServer = async <T extends IMinimalRecord>(
	apiRoute: string,
	uriParams: string,
): Promise<T[]> => {
	try {
		// `apiRoute` is treated as the backend route (example: `productos-ids`).
		const responsePayload = await GET({ route: `${apiRoute}?${uriParams}` })

		// Support both raw array responses and wrapped `{ records: [...] }` payloads.
		if (Array.isArray(responsePayload)) return responsePayload as T[]
		if (Array.isArray(responsePayload?.records)) return responsePayload.records as T[]

		console.warn(`${LOG_PREFIX} Unexpected server response shape. Returning empty list.`, {
			apiRoute,
			uriParams,
			responsePayload,
		})
		return []
	} catch (serverFetchError) {
		console.warn(`${LOG_PREFIX} Failed to fetch records from server. Returning empty list.`, {
			apiRoute,
			uriParams,
			serverFetchError,
		})
		return []
	}
}

export const configureCacheByIDs = (fetcher: CacheByIDsFetchFromServer) => {
	fetchFromServer = fetcher
}

const nowSeconds = () => Math.floor(Date.now() / 1000)

const getOrCreateTableCache = (apiRoute: string): Map<number, IMinimalRecord> => {
	const existingTableCache = tableIDRecordCache.get(apiRoute)
	if (existingTableCache) return existingTableCache
	const createdTableCache = new Map<number, IMinimalRecord>()
	tableIDRecordCache.set(apiRoute, createdTableCache)
	return createdTableCache
}

const normalizeIDs = (ids: number[]): number[] => {
	const uniqueIDs = new Set<number>()
	for (const rawID of ids) {
		const numericID = Number(rawID)
		if (!Number.isFinite(numericID)) continue
		if (numericID <= 0) continue
		uniqueIDs.add(numericID)
	}
	return Array.from(uniqueIDs).sort((a, b) => a - b)
}

const summarizeIDs = (ids: number[]) => {
	if (ids.length <= 10) return ids.join(",")
	return `${ids.slice(0, 10).join(",")} ... (+${ids.length - 10})`
}

export const getRecordsByIDs = async <T extends IMinimalRecord>(apiRoute: string, ids: number[]): Promise<T[]> => {
	// Normalize early to avoid duplicated work and to keep deterministic order.
	const normalizedSortedIDs = normalizeIDs(ids)
	const currentTimeSeconds = nowSeconds()

	console.debug(`${LOG_PREFIX} getRecordsByIDs start.`, {
		apiRoute,
		idsCount: normalizedSortedIDs.length,
		ids: summarizeIDs(normalizedSortedIDs),
	})

	const tableCache = getOrCreateTableCache(apiRoute)

	// Classify each requested ID into:
	// - missing cache (needs IDB lookup),
	// - cached entries (send ID + ccv for server-side delta validation),
	// - stale cache count (forces revalidation call).
	const idsMissingFromMemoryCache: number[] = []
	const recordsWitoutCache: number[] = []
	const recordsCachedIDs: number[] = []
	const recordsCachedUpdatedGroupsIDs: number[] = []

	// Count cached records that exceeded CACHE_TIME.
	// These records may still be unchanged (same ccv), but we must revalidate with backend.
	let staleCachedRecordsCount = 0

	for (const id of normalizedSortedIDs) {
		const cachedRecord = tableCache.get(id) as T | undefined
		if (!cachedRecord) {
			idsMissingFromMemoryCache.push(id)
			continue
		}

		// Tombstone: never fetch again, and never return as "active" record.
		if (cachedRecord.ss === 0) continue

		recordsCachedIDs.push(id)
		recordsCachedUpdatedGroupsIDs.push(cachedRecord.ccv || 0)

		const recordFetchAgeSeconds = currentTimeSeconds - (cachedRecord._fch || 0)
		if (recordFetchAgeSeconds > CACHE_TIME) {
			staleCachedRecordsCount++
		}
	}

	// Resolve all memory misses from persistent cache in one batch read.
	if (idsMissingFromMemoryCache.length > 0) {
		console.debug(`${LOG_PREFIX} Reading from IndexedDB.`, {
			apiRoute,
			idsCount: idsMissingFromMemoryCache.length,
			ids: summarizeIDs(idsMissingFromMemoryCache),
		})

		const idbRecordsByID = await readRecordsFromIDBByIDs<T>(apiRoute, idsMissingFromMemoryCache)

		for (const id of idsMissingFromMemoryCache) {
			const idbRecord = idbRecordsByID.get(id)
			if (!idbRecord) {
				recordsWitoutCache.push(id)
				continue
			}

			// Promote from IndexedDB to memory cache.
			tableCache.set(id, idbRecord)

			// Tombstone: never fetch again, and never return as "active" record.
			if (idbRecord.ss === 0) continue

			recordsCachedIDs.push(id)
			recordsCachedUpdatedGroupsIDs.push(idbRecord.ccv || 0)

			const recordFetchAgeSeconds = currentTimeSeconds - (idbRecord._fch || 0)
			if (recordFetchAgeSeconds > CACHE_TIME) {
				staleCachedRecordsCount++
			}
		}
	}

	// Build delta-validation payload:
	// - `ids`: records with no local cache.
	// - `cached`: records that exist locally and can be checked by backend.
	// - `ccv`: local update-group values aligned by position with `cached`.
	const uriParams = [
		recordsWitoutCache.length > 0 && `ids=${concatenateInts(recordsWitoutCache)}`,
		recordsCachedIDs.length > 0 && `cids=${concatenateInts(recordsCachedIDs)}`,
		recordsCachedUpdatedGroupsIDs.length > 0 && `ccv=${concatenateInts(recordsCachedUpdatedGroupsIDs)}`,
	]
	.filter(Boolean)
	.join("&")

	// Why not only `uriParams.length > 0`?
	// Because `cached` can be non-empty even when data is still fresh. In that case, network
	// call is unnecessary. We fetch only when:
	// 1) there are cache misses (`recordsWitoutCache`), or
	// 2) cached entries became stale and must be revalidated (`staleCachedRecordsCount`).
	// `uriParams.length > 0` is still required to avoid calling backend with empty query.
	const shouldFetchFromServer =
		(recordsWitoutCache.length > 0 || staleCachedRecordsCount > 0) && uriParams.length > 0

	console.debug(`${LOG_PREFIX} Cache partitioned.`, {
		apiRoute,
		recordsWitoutCacheCount: recordsWitoutCache.length,
		recordsCachedCount: recordsCachedIDs.length,
		staleCachedRecordsCount,
		uriParams,
	})

	let updatedOrNewRecordsFromServer: T[] = []
	if (shouldFetchFromServer) {
		try {
			console.log(`${LOG_PREFIX} Fetching from server.`, { apiRoute, uriParams })
			updatedOrNewRecordsFromServer = await fetchFromServer<T>(apiRoute, uriParams)
			console.log(`${LOG_PREFIX} Server response received.`, {
				apiRoute,
				recordsCount: updatedOrNewRecordsFromServer.length,
			})
		} catch (error) {
			console.warn(`${LOG_PREFIX} Server fetch failed. Using local cache only.`, error)
			updatedOrNewRecordsFromServer = []
		}
	}

	if (updatedOrNewRecordsFromServer.length > 0) {
		// Merge server delta into memory map and stamp fresh fetch time.
		for (const record of updatedOrNewRecordsFromServer) {
			if (!record || typeof record.ID !== "number") {
				console.warn(`${LOG_PREFIX} Invalid record received from server. Skipping.`, record)
				continue
			}
			// Always refresh fetch timestamp for records the server returned (new or changed).
			record._fch = currentTimeSeconds

			// Ensure required fields exist to keep cache invariant.
			if (typeof record.ccv !== "number") record.ccv = 0
			if (typeof record.ss !== "number") record.ss = 1

			tableCache.set(record.ID, record)
		}

		// Persist merged changes so next page load can hit IDB before network.
		await upsertRecordsIntoIDB<T>(apiRoute, updatedOrNewRecordsFromServer)
	}

	// Final read is stable by requested ID order; tombstones remain hidden.
	const resolvedRecords: T[] = []
	for (const id of normalizedSortedIDs) {
		const record = tableCache.get(id) as T | undefined
		if (!record) continue
		if (record.ss === 0) continue
		resolvedRecords.push(record)
	}

	console.debug(`${LOG_PREFIX} getRecordsByIDs done.`, {
		apiRoute,
		requestedCount: normalizedSortedIDs.length,
		returnedCount: resolvedRecords.length,
	})

	return resolvedRecords
}

const buffetMaxTime = 80 // milliseconds
const recordIDsBufferByTable: Map<string, Set<number>> = new Map()
const bufferedPromiseByTableAndID: Map<string, Promise<any>> = new Map()
const bufferedResolversByTableAndID: Map<
	string,
	{ resolve: (value: any) => void; reject: (reason?: any) => void }
> = new Map()
let bufferStarTime = 0
let bufferFlushTimer: ReturnType<typeof setTimeout> | null = null

// Runs once per buffer window and resolves every pending getRecordByID Promise.
// It batches IDs per table, does one request per table, then fans out each result by ID.
const flushBufferedRequests = async () => {
	const bufferedStartTime = bufferStarTime
	bufferStarTime = 0
	if (bufferFlushTimer) {
		clearTimeout(bufferFlushTimer)
		bufferFlushTimer = null
	}

	// Snapshot and clear current buffer so new calls can start a fresh window.
	const idsByTableToFetch: Array<{ apiRoute: string; ids: number[] }> = []
	for (const [apiRoute, idsSet] of recordIDsBufferByTable) {
		if (idsSet.size === 0) continue
		idsByTableToFetch.push({ apiRoute, ids: Array.from(idsSet) })
		idsSet.clear()
	}

	if (idsByTableToFetch.length === 0) return

	console.debug(`${LOG_PREFIX} Flushing buffered getRecordByID requests.`, {
		bufferedForMs: bufferedStartTime ? Date.now() - bufferedStartTime : 0,
		tablesCount: idsByTableToFetch.length,
	})

	for (const { apiRoute, ids } of idsByTableToFetch) {
		try {
			// One batched call per table, then fan out by ID.
			const records = await getRecordsByIDs<any>(apiRoute, ids)
			const recordsByID = new Map<number, any>(records.map((record) => [record.ID, record]))

			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const pending = bufferedResolversByTableAndID.get(key)
				if (!pending) continue
				bufferedResolversByTableAndID.delete(key)
				bufferedPromiseByTableAndID.delete(key)
				pending.resolve(recordsByID.get(id) || null)
			}
		} catch (error) {
			console.warn(`${LOG_PREFIX} Buffered fetch failed. Rejecting pending promises.`, { apiRoute }, error)
			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const pending = bufferedResolversByTableAndID.get(key)
				if (!pending) continue
				bufferedResolversByTableAndID.delete(key)
				bufferedPromiseByTableAndID.delete(key)
				pending.reject(error)
			}
		}
	}
}

export const getRecordByID = async <T extends IMinimalRecord>(
	apiRoute: string,
	id: number,
): Promise<T | null> => {
	const numericID = Number(id)
	if (!Number.isFinite(numericID) || numericID <= 0) return null

	const promiseKey = `${apiRoute}:${numericID}`
	const existingPromise = bufferedPromiseByTableAndID.get(promiseKey)
	if (existingPromise) return existingPromise as Promise<T | null>

	if (!bufferStarTime) bufferStarTime = Date.now()

	let idsBuffer = recordIDsBufferByTable.get(apiRoute)
	if (!idsBuffer) {
		idsBuffer = new Set()
		recordIDsBufferByTable.set(apiRoute, idsBuffer)
	}
	idsBuffer.add(numericID)

	const createdPromise = new Promise<T | null>((resolve, reject) => {
		bufferedResolversByTableAndID.set(promiseKey, { resolve, reject })
	})
	bufferedPromiseByTableAndID.set(promiseKey, createdPromise)

	// Start one timer per active buffer window; all requests inside this window are batched.
	if (!bufferFlushTimer) {
		bufferFlushTimer = setTimeout(() => {
			void flushBufferedRequests()
		}, buffetMaxTime)
	}

	return createdPromise
}
