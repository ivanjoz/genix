import { GET } from '../http.svelte'

/**
 * Goal:
 * Resolve records by IDs using a 3-layer strategy:
 * 1) in-memory map, 2) IndexedDB persistent cache, 3) server delta sync.
 * It sends `ids`, `cc-ids`, and `cc-ver` so backend returns only new/changed records.
 */
import { concatenateInts } from "../funcs/parsers"
import { readRecordsFromIDBByIDs, upsertRecordsIntoIDB } from "./cache-by-ids.idb"
import { Env } from '$core/env'

const CACHE_TIME = 5

export interface IMinimalRecord {
	ID: number /* ID f the record */ 
	ccv?: number /* cache version a number from 0 to 255 */
	ss: number /* status: 1 active, 0 deleted */
	_fch?: number /* fetched: when the record was last fetched (in seconds) */
	upd: number /* for used in getRecordByIDUpdated */
}

const cacheRecordIdTable: Map<string,Map<number,IMinimalRecord>> = new Map()

const LOG_PREFIX = "[cache-by-ids]"
const CACHE_DEBUG_PREFIX = `${LOG_PREFIX}:stale-debug`

export type CacheByIDsFetchFromServer = <T extends IMinimalRecord>(
	apiRoute: string,
	ids: number[],
	ccIDs: number[],
	ccVer: number[],
) => Promise<T[]>

const buildFetchUriParams = (
	ids: number[],
	ccIDs: number[],
	ccVer: number[],
): string => {
	return [
		ids.length > 0 && `ids=${concatenateInts(ids)}`,
		ccIDs.length > 0 && `cc-ids=${concatenateInts(ccIDs)}`,
		ccVer.length > 0 && `cc-ver=${concatenateInts(ccVer)}`,
	]
		.filter(Boolean)
		.join("&")
}

// Configure this from your app when the backend endpoint is ready.
let fetchFromServer: CacheByIDsFetchFromServer = async <T extends IMinimalRecord>(
	apiRoute: string,
	ids: number[],
	ccIDs: number[],
	ccVer: number[],
): Promise<T[]> => {
	const uriParams = buildFetchUriParams(ids, ccIDs, ccVer)
	try {
		// `apiRoute` is treated as the backend route (example: `p-productos-ids`).
		const responsePayload = await GET({
			route: `${apiRoute}?${uriParams}&cmp=${Env.getEmpresaID()}`
		})

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
	const existingTableCache = cacheRecordIdTable.get(apiRoute)
	if (existingTableCache) return existingTableCache
	const createdTableCache = new Map<number, IMinimalRecord>()
	cacheRecordIdTable.set(apiRoute, createdTableCache)
	return createdTableCache
}

const mergeFetchedRecordsIntoCache = async <T extends IMinimalRecord>(
	apiRoute: string,
	records: T[],
	fetchedAtSeconds: number,
): Promise<Map<number, T>> => {
	const tableCache = getOrCreateTableCache(apiRoute)
	const mergedRecords = new Map<number, T>()

	// Normalize server rows before writing so memory and IDB keep the same invariant.
	for (const fetchedRecord of records) {
		if (!fetchedRecord || typeof fetchedRecord.ID !== "number") {
			console.warn(`${LOG_PREFIX} Invalid record received from server. Skipping.`, fetchedRecord)
			continue
		}
		fetchedRecord._fch = fetchedAtSeconds
		if (typeof fetchedRecord.ccv !== "number") fetchedRecord.ccv = 0
		if (typeof fetchedRecord.ss !== "number") fetchedRecord.ss = 1
		tableCache.set(fetchedRecord.ID, fetchedRecord)
		mergedRecords.set(fetchedRecord.ID, fetchedRecord)
	}

	if (mergedRecords.size > 0) {
		await upsertRecordsIntoIDB<T>(apiRoute, Array.from(mergedRecords.values()))
	}

	return mergedRecords
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

export const getRecordsByIDs = async <T extends IMinimalRecord>(
	apiRoute: string,
	ids: number[],
	cacheHintsByID?: Map<number, T | false>,
): Promise<T[]> => {
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
		// Optional hint path: caller already knows local cache state for this ID.
		if (cacheHintsByID && cacheHintsByID.has(id)) {
			const hintedRecord = cacheHintsByID.get(id)
			if (hintedRecord === false) {
				console.debug(`${CACHE_DEBUG_PREFIX} hint says local miss`, { apiRoute, id })
				recordsWitoutCache.push(id)
				continue
			}
			if (!hintedRecord) {
				console.debug(`${CACHE_DEBUG_PREFIX} hint undefined, treat as miss`, { apiRoute, id })
				recordsWitoutCache.push(id)
				continue
			}
			// Tombstone: never fetch again, and never return as "active" record.
			if (hintedRecord.ss === 0) continue

			recordsCachedIDs.push(id)
			recordsCachedUpdatedGroupsIDs.push(hintedRecord.ccv || 0)

			const recordFetchAgeSeconds = currentTimeSeconds-(hintedRecord._fch || 0)
			if (recordFetchAgeSeconds > CACHE_TIME) {
				staleCachedRecordsCount++
				console.debug(`${CACHE_DEBUG_PREFIX} hint says stale`, {
					apiRoute,
					id,
					recordFetchAgeSeconds,
					cacheTimeSeconds: CACHE_TIME,
					fetchedAt: hintedRecord._fch || 0,
					now: currentTimeSeconds,
				})
			} else {
				console.debug(`${CACHE_DEBUG_PREFIX} hint says fresh`, {
					apiRoute,
					id,
					recordFetchAgeSeconds,
					cacheTimeSeconds: CACHE_TIME,
					fetchedAt: hintedRecord._fch || 0,
					now: currentTimeSeconds,
				})
			}
			// Keep memory cache aligned with caller-provided local state.
			tableCache.set(id, hintedRecord)
			continue
		}

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
			console.debug(`${CACHE_DEBUG_PREFIX} memory says stale`, {
				apiRoute,
				id,
				recordFetchAgeSeconds,
				cacheTimeSeconds: CACHE_TIME,
				fetchedAt: cachedRecord._fch || 0,
				now: currentTimeSeconds,
			})
		} else {
			console.debug(`${CACHE_DEBUG_PREFIX} memory says fresh`, {
				apiRoute,
				id,
				recordFetchAgeSeconds,
				cacheTimeSeconds: CACHE_TIME,
				fetchedAt: cachedRecord._fch || 0,
				now: currentTimeSeconds,
			})
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
				console.debug(`${CACHE_DEBUG_PREFIX} idb says stale`, {
					apiRoute,
					id,
					recordFetchAgeSeconds,
					cacheTimeSeconds: CACHE_TIME,
					fetchedAt: idbRecord._fch || 0,
					now: currentTimeSeconds,
				})
			} else {
				console.debug(`${CACHE_DEBUG_PREFIX} idb says fresh`, {
					apiRoute,
					id,
					recordFetchAgeSeconds,
					cacheTimeSeconds: CACHE_TIME,
					fetchedAt: idbRecord._fch || 0,
					now: currentTimeSeconds,
				})
			}
		}
	}

	// Build delta-validation payload:
	// - `ids`: records with no local cache.
	// - `cc-ids`: records that exist locally and can be checked by backend.
	// - `cc-ver`: local update-group values aligned by position with `cc-ids`.
	const uriParams = buildFetchUriParams(
		recordsWitoutCache,
		recordsCachedIDs,
		recordsCachedUpdatedGroupsIDs,
	)

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
		shouldFetchFromServer,
	})
	if (!shouldFetchFromServer) {
		console.debug(`${CACHE_DEBUG_PREFIX} skip server fetch`, {
			apiRoute,
			reason: "no miss and no stale cached records (or empty uriParams)",
			recordsWitoutCacheCount: recordsWitoutCache.length,
			staleCachedRecordsCount,
			uriParams,
		})
	}

	let updatedOrNewRecordsFromServer: T[] = []
	if (shouldFetchFromServer) {
		try {
			console.log(`${LOG_PREFIX} Fetching from server.`, { apiRoute, uriParams })
			updatedOrNewRecordsFromServer = await fetchFromServer<T>(
				apiRoute,
				recordsWitoutCache,
				recordsCachedIDs,
				recordsCachedUpdatedGroupsIDs,
			)
			console.log(`${LOG_PREFIX} Server response received.`, {
				apiRoute,
				recordsCount: updatedOrNewRecordsFromServer.length,
			})
		} catch (error) {
			console.warn(`${LOG_PREFIX} Server fetch failed. Using local cache only.`, error)
			updatedOrNewRecordsFromServer = []
		}
	}

	// Backend delta response includes only changed/new cached IDs.
	// Any cached ID not returned is validated as unchanged and should refresh `_fch`.
	if (shouldFetchFromServer && recordsCachedIDs.length > 0) {
		const returnedUpdatedIDs = new Set<number>()
		for (const updatedRecord of updatedOrNewRecordsFromServer) {
			if (!updatedRecord || typeof updatedRecord.ID !== "number") continue
			returnedUpdatedIDs.add(updatedRecord.ID)
		}
		for (const cachedID of recordsCachedIDs) {
			if (returnedUpdatedIDs.has(cachedID)) continue
			const existingCachedRecord = tableCache.get(cachedID)
			if (!existingCachedRecord) continue
			existingCachedRecord._fch = currentTimeSeconds
			tableCache.set(cachedID, existingCachedRecord)
		}
	}

	if (updatedOrNewRecordsFromServer.length > 0) {
		await mergeFetchedRecordsIntoCache<T>(apiRoute, updatedOrNewRecordsFromServer, currentTimeSeconds)
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
const bufferedCacheHintByTableAndID: Map<string, IMinimalRecord | false> = new Map()
const bufferedResolversByTableAndID: Map<
	string,
	{ resolve: (value: any) => void; reject: (reason?: any) => void }
> = new Map()
const updatedRecordIDsBufferByTable: Map<string, Set<number>> = new Map()
const bufferedUpdatedPromiseByTableAndID: Map<string, Promise<any>> = new Map()
const bufferedUpdatedResolversByTableAndID: Map<
	string,
	{ resolve: (value: any) => void; reject: (reason?: any) => void }
> = new Map()
let bufferStarTime = 0
let bufferFlushTimer: ReturnType<typeof setTimeout> | null = null
let updatedBufferStarTime = 0
let updatedBufferFlushTimer: ReturnType<typeof setTimeout> | null = null

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
			const cacheHintsByID = new Map<number, IMinimalRecord | false>()
			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const bufferedCacheHint = bufferedCacheHintByTableAndID.get(key)
				if (bufferedCacheHint === undefined) continue
				cacheHintsByID.set(id, bufferedCacheHint)
				bufferedCacheHintByTableAndID.delete(key)
			}

			// One batched call per table, then fan out by ID.
			console.debug(`${CACHE_DEBUG_PREFIX} flush table batch`, {
				apiRoute,
				idsCount: ids.length,
				ids: summarizeIDs(ids),
				hintsCount: cacheHintsByID.size,
			})
			const records = await getRecordsByIDs<any>(apiRoute, ids, cacheHintsByID)
			const recordsByID = new Map<number, any>(records.map((record) => [record.ID, record]))

			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const pending = bufferedResolversByTableAndID.get(key)
				if (!pending) continue
				bufferedResolversByTableAndID.delete(key)
				bufferedPromiseByTableAndID.delete(key)
				pending.resolve(recordsByID.get(id))
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

export interface GetRecordByIDOptions<T extends IMinimalRecord> {
	cachedRecord?: T | false
}

const flushBufferedUpdatedRequests = async () => {
	const bufferedStartTime = updatedBufferStarTime
	updatedBufferStarTime = 0
	if (updatedBufferFlushTimer) {
		clearTimeout(updatedBufferFlushTimer)
		updatedBufferFlushTimer = null
	}

	const idsByTableToFetch: Array<{ apiRoute: string; ids: number[] }> = []
	for (const [apiRoute, idsSet] of updatedRecordIDsBufferByTable) {
		if (idsSet.size === 0) continue
		idsByTableToFetch.push({ apiRoute, ids: Array.from(idsSet) })
		idsSet.clear()
	}

	if (idsByTableToFetch.length === 0) return

	console.debug(`${LOG_PREFIX} Flushing buffered getRecordByIDUpdated requests.`, {
		bufferedForMs: bufferedStartTime ? Date.now() - bufferedStartTime : 0,
		tablesCount: idsByTableToFetch.length,
	})

	for (const { apiRoute, ids } of idsByTableToFetch) {
		try {
			console.debug(`${CACHE_DEBUG_PREFIX} flush updated table batch`, {
				apiRoute,
				idsCount: ids.length,
				ids: summarizeIDs(ids),
			})
			const fetchedRecords = await fetchFromServer<any>(apiRoute, ids, [], [])
			const fetchedRecordsByID = await mergeFetchedRecordsIntoCache<any>(apiRoute, fetchedRecords, nowSeconds())

			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const pending = bufferedUpdatedResolversByTableAndID.get(key)
				if (!pending) continue
				bufferedUpdatedResolversByTableAndID.delete(key)
				bufferedUpdatedPromiseByTableAndID.delete(key)
				const fetchedRecord = fetchedRecordsByID.get(id)
				pending.resolve(fetchedRecord && fetchedRecord.ss !== 0 ? fetchedRecord : undefined)
			}
		} catch (error) {
			console.warn(`${LOG_PREFIX} Buffered updated fetch failed. Rejecting pending promises.`, { apiRoute }, error)
			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const pending = bufferedUpdatedResolversByTableAndID.get(key)
				if (!pending) continue
				bufferedUpdatedResolversByTableAndID.delete(key)
				bufferedUpdatedPromiseByTableAndID.delete(key)
				pending.reject(error)
			}
		}
	}
}

export const getRecordByID = async <T extends IMinimalRecord>(
	apiRoute: string,
	id: number,
	options?: GetRecordByIDOptions<T>,
): Promise<T | undefined> => {
	const numericID = Number(id)
	if (!Number.isFinite(numericID) || numericID <= 0) return undefined
	const currentTimeSeconds = nowSeconds()

	// Explicit cache hint path avoids repeating local lookups already done by caller.
	// Still goes through buffered pipeline to prevent parallel per-ID requests.
	if (options && options.cachedRecord !== undefined) {
		const knownCachedRecord = options.cachedRecord
		if (!!knownCachedRecord) {
			if (knownCachedRecord.ss === 0) return undefined
			const cachedRecordIsStale = isRecordStale(knownCachedRecord, currentTimeSeconds)
			console.debug(`${CACHE_DEBUG_PREFIX} getRecordByID cachedRecord option`, {
				apiRoute,
				id: numericID,
				hasCachedRecord: true,
				cachedRecordIsStale,
				fetchedAt: knownCachedRecord._fch || 0,
				cacheTimeSeconds: CACHE_TIME,
			})
			if (!cachedRecordIsStale) {
				console.debug(`${CACHE_DEBUG_PREFIX} getRecordByID returns fresh option record immediately`, {
					apiRoute,
					id: numericID,
				})
				return knownCachedRecord
			}
		}
		if (knownCachedRecord === false) {
			console.debug(`${CACHE_DEBUG_PREFIX} getRecordByID option says local miss`, {
				apiRoute,
				id: numericID,
			})
		}
	}

	// Fast path optimization:
	// return immediately only for fresh memory hits; stale hits must go through buffered revalidation.
	const cacheTable = cacheRecordIdTable.get(apiRoute)
	if (cacheTable) {
		const cachedRecord = cacheTable.get(numericID) as T | undefined
		if (cachedRecord) {
			if (cachedRecord.ss === 0) return undefined
			const cachedRecordAgeSeconds = currentTimeSeconds - (cachedRecord._fch || 0)
			const cachedRecordIsStale = cachedRecordAgeSeconds > CACHE_TIME
			console.debug(`${CACHE_DEBUG_PREFIX} getRecordByID memory fast path hit`, {
				apiRoute,
				id: numericID,
				cachedRecordAgeSeconds,
				cacheTimeSeconds: CACHE_TIME,
				isStale: cachedRecordIsStale,
				fetchedAt: cachedRecord._fch || 0,
			})
			if (!cachedRecordIsStale) {
				return cachedRecord
			}
			console.debug(`${CACHE_DEBUG_PREFIX} getRecordByID memory stale hit, continue to buffered revalidation`, {
				apiRoute,
				id: numericID,
			})
		}
	}

	const promiseKey = `${apiRoute}:${numericID}`
	const existingPromise = bufferedPromiseByTableAndID.get(promiseKey)
	if (existingPromise) return existingPromise as Promise<T | undefined>

	if (!bufferStarTime) bufferStarTime = Date.now()

	let idsBuffer = recordIDsBufferByTable.get(apiRoute)
	if (!idsBuffer) {
		idsBuffer = new Set()
		recordIDsBufferByTable.set(apiRoute, idsBuffer)
	}
	idsBuffer.add(numericID)

	const createdPromise = new Promise<T | undefined>((resolve, reject) => {
		bufferedResolversByTableAndID.set(promiseKey, { resolve, reject })
	})
	bufferedPromiseByTableAndID.set(promiseKey, createdPromise)
	if (options && options.cachedRecord !== undefined) {
		bufferedCacheHintByTableAndID.set(promiseKey, options.cachedRecord)
	}
	console.debug(`${CACHE_DEBUG_PREFIX} getRecordByID queued into buffer`, {
		apiRoute,
		id: numericID,
		hasCachedHint: Boolean(options && options.cachedRecord !== undefined),
		bufferWindowMs: buffetMaxTime,
	})

	// Start one timer per active buffer window; all requests inside this window are batched.
	if (!bufferFlushTimer) {
		bufferFlushTimer = setTimeout(() => {
			void flushBufferedRequests()
		}, buffetMaxTime)
	}

	return createdPromise
}

export const getRecordByIDUpdated = async <T extends IMinimalRecord>(
	apiRoute: string, id: number,	updated: number,
): Promise<T | undefined> => {
	const numericID = Number(id)
	if (!Number.isFinite(numericID) || numericID <= 0) return undefined

	const localResolution = await getLocalRecordByID<T>(apiRoute, numericID)
	if (localResolution.record && localResolution.record.upd >= updated) {
		console.debug(`${CACHE_DEBUG_PREFIX} getRecordByIDUpdated returns local cache`, {
			apiRoute,
			id: numericID,
			updated,
		})
		return localResolution.record
	}

	let returnedRecord: T | undefined
	try {
		console.debug(`${CACHE_DEBUG_PREFIX} getRecordByIDUpdated queued into updated buffer`, {
			apiRoute,
			id: numericID,
			requestedUpdated: updated,
			localUpdated: localResolution.record?.upd || 0,
			bufferWindowMs: buffetMaxTime,
		})
		const promiseKey = `${apiRoute}:${numericID}`
		const existingPromise = bufferedUpdatedPromiseByTableAndID.get(promiseKey)
		if (existingPromise) {
			returnedRecord = await existingPromise as T | undefined
		} else {
			if (!updatedBufferStarTime) updatedBufferStarTime = Date.now()

			let idsBuffer = updatedRecordIDsBufferByTable.get(apiRoute)
			if (!idsBuffer) {
				idsBuffer = new Set()
				updatedRecordIDsBufferByTable.set(apiRoute, idsBuffer)
			}
			idsBuffer.add(numericID)

			const createdPromise = new Promise<T | undefined>((resolve, reject) => {
				bufferedUpdatedResolversByTableAndID.set(promiseKey, { resolve, reject })
			})
			bufferedUpdatedPromiseByTableAndID.set(promiseKey, createdPromise)

			if (!updatedBufferFlushTimer) {
				updatedBufferFlushTimer = setTimeout(() => {
					void flushBufferedUpdatedRequests()
				}, buffetMaxTime)
			}

			returnedRecord = await createdPromise
		}
	} catch (fetchError) {
		console.warn(`${LOG_PREFIX} Failed to refresh record by updated.`, {
			apiRoute,
			id: numericID,
			updated,
		}, fetchError)
		return undefined
	}

	// Once we fetched from server, that row is authoritative even when its `upd` advanced.
	const fetchedRecord = returnedRecord
	if (!fetchedRecord || fetchedRecord.ss === 0) return undefined
	return fetchedRecord
}

const isRecordStale = (record: IMinimalRecord, nowTimeSeconds: number): boolean => {
	const fetchedAtSeconds = record._fch || 0
	const ageSeconds = nowTimeSeconds-fetchedAtSeconds
	const stale = ageSeconds > CACHE_TIME
	console.debug(`${CACHE_DEBUG_PREFIX} stale-check`, {
		id: record.ID,
		fetchedAtSeconds,
		nowTimeSeconds,
		ageSeconds,
		cacheTimeSeconds: CACHE_TIME,
		stale,
	})
	return stale
}

const getLocalRecordByID = async <T extends IMinimalRecord>(
	apiRoute: string,
	id: number,
): Promise<{ record: T | null, stale: boolean }> => {
	const numericID = Number(id)
	if (!Number.isFinite(numericID) || numericID <= 0) {
		return { record: null, stale: false }
	}

	const currentTimeSeconds = nowSeconds()
	const cacheTable = getOrCreateTableCache(apiRoute)
	const memoryRecord = cacheTable.get(numericID) as T | undefined
	if (memoryRecord) {
		if (memoryRecord.ss === 0) return { record: null, stale: false }
		console.debug(`${CACHE_DEBUG_PREFIX} local resolve hit memory`, { apiRoute, id: numericID })
		return { record: memoryRecord, stale: isRecordStale(memoryRecord, currentTimeSeconds) }
	}

	const idbRecordsByID = await readRecordsFromIDBByIDs<T>(apiRoute, [numericID])
	const idbRecord = idbRecordsByID.get(numericID)
	if (!idbRecord) {
		console.debug(`${CACHE_DEBUG_PREFIX} local resolve miss in IDB`, { apiRoute, id: numericID })
		return { record: null, stale: false }
	}

	// Promote IDB hit into memory cache for subsequent synchronous reads.
	cacheTable.set(numericID, idbRecord)
	if (idbRecord.ss === 0) return { record: null, stale: false }
	console.debug(`${CACHE_DEBUG_PREFIX} local resolve hit IDB`, { apiRoute, id: numericID })
	return { record: idbRecord, stale: isRecordStale(idbRecord, currentTimeSeconds) }
}

export interface IRecordRef<T extends IMinimalRecord> {
	readonly record: T | null
	readonly loading: boolean
	refresh: () => Promise<T | null>
}

export const getRecordWithCache = <T extends IMinimalRecord>(
	apiRoute: string,
	id: number,
): IRecordRef<T> => {
	let record = $state<T | null>(null)
	let loading = $state(false)

	const refresh = async (): Promise<T | null> => {
		const numericID = Number(id)
		if (!Number.isFinite(numericID) || numericID <= 0) {
			record = null
			loading = false
			return null
		}

		const localResolution = await getLocalRecordByID<T>(apiRoute, numericID)
		console.debug(`${CACHE_DEBUG_PREFIX} ref.refresh local resolution`, {
			apiRoute,
			id: numericID,
			hasLocalRecord: Boolean(localResolution.record),
			stale: localResolution.stale,
		})
		// Show loading spinner only when we don't have local data for first paint.
		loading = !Boolean(localResolution.record)
		if (localResolution.record) {
			record = localResolution.record
		}

		// Fresh local hit: skip network and complete.
		if (localResolution.record && !localResolution.stale) {
			console.debug(`${CACHE_DEBUG_PREFIX} ref.refresh skip network due fresh local record`, {
				apiRoute,
				id: numericID,
			})
			loading = false
			return localResolution.record
		}

		// Local miss or stale local hit: refresh from server and update ref if changed/new.
		const refreshedRecord = await getRecordByID<T>(apiRoute, numericID, {
			cachedRecord: localResolution.record || false,
		})
		console.debug(`${CACHE_DEBUG_PREFIX} ref.refresh server refresh result`, {
			apiRoute,
			id: numericID,
			hasRefreshedRecord: Boolean(refreshedRecord),
		})
		const isRecordChanged =
			Boolean(refreshedRecord) &&
			(
				!localResolution.record ||
				refreshedRecord!.ccv !== localResolution.record.ccv ||
				refreshedRecord!.ss !== localResolution.record.ss ||
				// Keep this flexible for model payloads that include `upd`.
				(refreshedRecord as unknown as { upd?: number }).upd !==
					(localResolution.record as unknown as { upd?: number }).upd
			)
		if (isRecordChanged && refreshedRecord) {
			record = refreshedRecord
		}

		loading = false
		return refreshedRecord || localResolution.record
	}

	// Kick-off on creation so component only reads state.
	void refresh()

	return {
		get record() {
			return record
		},
		get loading() {
			return loading
		},
		refresh,
	}
}
