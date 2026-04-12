import { GET } from '../http.svelte'

/**
 * Goal:
 * Resolve records by IDs using a 3-layer strategy:
 * 1) in-memory map, 2) IndexedDB persistent cache, 3) server delta sync.
 * It sends `ids`, `cc-ids`, and `cc-ver` so backend returns only new/changed records.
 */
import { concatenateInts } from "../funcs/parsers"
import { clearCacheByIDsDatabase, readRecordsFromIDBByIDs, upsertRecordsIntoIDB } from "./cache-by-ids.idb"
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
const inFlightRecordsBatchPromiseByTable: Map<
	string,
	Map<number, Promise<Map<number, IMinimalRecord>>>
> = new Map()

const LOG_PREFIX = "[cache-by-ids]"
const DEBUG_CACHE_RECORD_IDS = new Set<number>([26])

const shouldDebugCacheRecord = (id: number): boolean => {
	return DEBUG_CACHE_RECORD_IDS.has(id)
}

const logDebugCacheRecord = (
	stage: string,
	apiRoute: string,
	id: number,
	record: Partial<IMinimalRecord> | null | undefined,
	extra?: Record<string, unknown>,
) => {
	if (!shouldDebugCacheRecord(id)) return
	console.debug(`${LOG_PREFIX} debug ${apiRoute} | ${stage}`, {
		id,
		ccv: record?.ccv,
		_fch: record?._fch,
		ss: record?.ss,
		upd: record?.upd,
		...extra,
	})
}

export type CacheByIDsFetchFromServer = <T extends IMinimalRecord>(
	apiRoute: string,
	ids: number[],
	ccIDs: number[],
	ccVer: number[],
) => Promise<T[]>

const buildFetchUriParams = (
	ids: number[], ccIDs: number[], ccVer: number[],
): string => {
	const cachedRecordsU8IDs: number[] = []
	const cachedRecordsU8Versions: number[] = []
	const cachedRecordsU16IDs: number[] = []
	const cachedRecordsU16Versions: number[] = []
	const cachedRecordsU32IDs: number[] = []
	const cachedRecordsU32Versions: number[] = []

	for (let index = 0; index < ccIDs.length; index++) {
		const cachedID = ccIDs[index]
		const cachedVersion = ccVer[index] || 0
		if (cachedVersion < 0 || cachedVersion > 255) {
			// Cache-version protocol is uint8 on backend; fail loudly before corrupting alignment.
			throw new Error(`${LOG_PREFIX} invalid cc-ver for ${cachedID}: ${cachedVersion}`)
		}

		// Keep `cc-ids` and `cc-ver` in the same bucket order the compact encoder emits.
		if (cachedID >= 0 && cachedID <= 255) {
			cachedRecordsU8IDs.push(cachedID)
			cachedRecordsU8Versions.push(cachedVersion)
			continue
		}
		if (cachedID >= 0 && cachedID <= 65535) {
			cachedRecordsU16IDs.push(cachedID)
			cachedRecordsU16Versions.push(cachedVersion)
			continue
		}
		cachedRecordsU32IDs.push(cachedID)
		cachedRecordsU32Versions.push(cachedVersion)
	}

	const alignedCachedIDs = [...cachedRecordsU8IDs, ...cachedRecordsU16IDs, ...cachedRecordsU32IDs]
	const alignedCachedVersions = [...cachedRecordsU8Versions,...cachedRecordsU16Versions,...cachedRecordsU32Versions]

	return [
		ids.length > 0 && `ids=${concatenateInts(ids)}`,
		alignedCachedIDs.length > 0 && `cc-ids=${concatenateInts(alignedCachedIDs)}`,
		alignedCachedVersions.length > 0 && `cc-ver=${concatenateInts(alignedCachedVersions)}`,
	].filter(Boolean).join("&")
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

		console.warn(`${LOG_PREFIX} ${apiRoute} unexpected server response shape; returning empty list`, responsePayload)
		return []
	} catch (serverFetchError) {
		console.warn(`${LOG_PREFIX} ${apiRoute} failed to fetch records from server; returning empty list`, serverFetchError)
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

const getOrCreateInFlightRecordsTable = (
	apiRoute: string,
): Map<number, Promise<Map<number, IMinimalRecord>>> => {
	const existingInFlightTable = inFlightRecordsBatchPromiseByTable.get(apiRoute)
	if (existingInFlightTable) return existingInFlightTable
	const createdInFlightTable = new Map<number, Promise<Map<number, IMinimalRecord>>>()
	inFlightRecordsBatchPromiseByTable.set(apiRoute, createdInFlightTable)
	return createdInFlightTable
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
		logDebugCacheRecord("server record merged into memory", apiRoute, fetchedRecord.ID, fetchedRecord)
		mergedRecords.set(fetchedRecord.ID, fetchedRecord)
	}

	if (mergedRecords.size > 0) {
		await upsertRecordsIntoIDB<T>(apiRoute, Array.from(mergedRecords.values()))
		for (const mergedRecord of mergedRecords.values()) {
			if (!shouldDebugCacheRecord(mergedRecord.ID)) continue
			const idbRecord = (await readRecordsFromIDBByIDs<T>(apiRoute, [mergedRecord.ID])).get(mergedRecord.ID)
			logDebugCacheRecord("record read back from indexedDB after upsert", apiRoute, mergedRecord.ID, idbRecord)
		}
	}

	return mergedRecords
}

const summarizeIDs = (ids: number[]) => {
	if (ids.length <= 10) return ids.join(",")
	return `${ids.slice(0, 10).join(",")} ... (+${ids.length - 10})`
}

const normalizePositiveIDs = (ids: number[]): number[] => {
	const normalizedIDs: number[] = []
	const seenIDs = new Set<number>()

	for (const rawID of ids) {
		const numericID = Number(rawID)
		if (!Number.isFinite(numericID)) continue
		if (numericID <= 0) continue
		if (seenIDs.has(numericID)) continue
		seenIDs.add(numericID)
		normalizedIDs.push(numericID)
	}

	return normalizedIDs
}

export const doGetRecordsByIDs = async <T extends IMinimalRecord>(
	apiRoute: string,
	ids: number[],
	cacheHintsByID?: Map<number, T | false>,
): Promise<Map<number, T>> => {
	// Normalize early to avoid duplicated work and to keep deterministic order.
	const normalizedSortedIDs = normalizePositiveIDs(ids)
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
	const idsCachedOnMemory: number[] = []
	const idsCachedFromIndexedDB: number[] = []
	const staleIDsToFetch: number[] = []
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
				recordsWitoutCache.push(id)
				continue
			}
			if (!hintedRecord) {
				recordsWitoutCache.push(id)
				continue
			}
			// Tombstone: never fetch again, and never return as "active" record.
			if (hintedRecord.ss === 0) continue

			idsCachedOnMemory.push(id)
			recordsCachedIDs.push(id)
			recordsCachedUpdatedGroupsIDs.push(hintedRecord.ccv || 0)

			const recordFetchAgeSeconds = currentTimeSeconds-(hintedRecord._fch || 0)
			if (recordFetchAgeSeconds > CACHE_TIME) {
				staleCachedRecordsCount++
				staleIDsToFetch.push(id)
			}
			
			// Keep memory cache aligned with caller-provided local state.
			tableCache.set(id, hintedRecord)
			continue
		}

		const cachedRecord = tableCache.get(id) as T | undefined
		if (!cachedRecord) {
			logDebugCacheRecord("memory miss before indexedDB lookup", apiRoute, id, null)
			idsMissingFromMemoryCache.push(id)
			continue
		}
		logDebugCacheRecord("memory hit before stale check", apiRoute, id, cachedRecord)

		// Tombstone: never fetch again, and never return as "active" record.
		if (cachedRecord.ss === 0) continue

		idsCachedOnMemory.push(id)
		recordsCachedIDs.push(id)
		recordsCachedUpdatedGroupsIDs.push(cachedRecord.ccv || 0)

		const recordFetchAgeSeconds = currentTimeSeconds - (cachedRecord._fch || 0)
		if (recordFetchAgeSeconds > CACHE_TIME) {
			staleCachedRecordsCount++
			staleIDsToFetch.push(id)
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
				logDebugCacheRecord("indexedDB miss", apiRoute, id, null)
				recordsWitoutCache.push(id)
				continue
			}
			logDebugCacheRecord("indexedDB hit before promotion", apiRoute, id, idbRecord)

			// Promote from IndexedDB to memory cache.
			tableCache.set(id, idbRecord)
			logDebugCacheRecord("indexedDB record promoted to memory", apiRoute, id, idbRecord)

			// Tombstone: never fetch again, and never return as "active" record.
			if (idbRecord.ss === 0) continue

			idsCachedFromIndexedDB.push(id)
			recordsCachedIDs.push(id)
			recordsCachedUpdatedGroupsIDs.push(idbRecord.ccv || 0)

			const recordFetchAgeSeconds = currentTimeSeconds - (idbRecord._fch || 0)
			if (recordFetchAgeSeconds > CACHE_TIME) {
				staleCachedRecordsCount++
				staleIDsToFetch.push(id)
			}
		}
	}

	console.debug(`${LOG_PREFIX} ${apiRoute} memory cache hits`, idsCachedOnMemory)
	console.debug(`${LOG_PREFIX} ${apiRoute} indexedDB cache hits`, idsCachedFromIndexedDB)
	console.debug(`${LOG_PREFIX} ${apiRoute} stale IDs to fetch`, staleIDsToFetch)
	console.debug(`${LOG_PREFIX} ${apiRoute} missing IDs to fetch`, recordsWitoutCache)

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

	for (const id of normalizedSortedIDs) {
		if (!shouldDebugCacheRecord(id)) continue
		logDebugCacheRecord("request payload version snapshot", apiRoute, id, {
			ID: id,
			ccv: recordsCachedIDs.includes(id)
				? recordsCachedUpdatedGroupsIDs[recordsCachedIDs.indexOf(id)]
				: undefined,
			_fch: (tableCache.get(id) as T | undefined)?._fch,
			ss: (tableCache.get(id) as T | undefined)?.ss,
			upd: (tableCache.get(id) as T | undefined)?.upd,
		}, {
			isMissing: recordsWitoutCache.includes(id),
			isStale: staleIDsToFetch.includes(id),
			willFetch: shouldFetchFromServer,
		})
	}

	console.debug(
		`${LOG_PREFIX} ${apiRoute} cache partitioned` +
		` | memory=${idsCachedOnMemory.length}` +
		` | indexedDB=${idsCachedFromIndexedDB.length}` +
		` | stale=${staleCachedRecordsCount}` +
		` | missing=${recordsWitoutCache.length}` +
		` | fetch=${shouldFetchFromServer}`,
	)
	if (!shouldFetchFromServer) {
		console.debug(`${LOG_PREFIX} ${apiRoute} skip server fetch | stale=${staleCachedRecordsCount} | missing=${recordsWitoutCache.length}`)
	}

	let updatedOrNewRecordsFromServer: T[] = []
	if (shouldFetchFromServer) {
		try {
			console.log(
				`${LOG_PREFIX} ${apiRoute} fetching from server` +
				` | stale=${staleCachedRecordsCount}` +
				` | missing=${recordsWitoutCache.length}` +
				` | cached=${recordsCachedIDs.length}`,
			)
			updatedOrNewRecordsFromServer = await fetchFromServer<T>(
				apiRoute,
				recordsWitoutCache,
				recordsCachedIDs,
				recordsCachedUpdatedGroupsIDs,
			)
			for (const fetchedRecord of updatedOrNewRecordsFromServer) {
				if (!fetchedRecord || typeof fetchedRecord.ID !== "number") continue
				logDebugCacheRecord("server response record", apiRoute, fetchedRecord.ID, fetchedRecord)
			}
			console.log(`${LOG_PREFIX} ${apiRoute} server response received | records=${updatedOrNewRecordsFromServer.length}`)
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
			logDebugCacheRecord("unchanged cached record refreshed in memory", apiRoute, cachedID, existingCachedRecord)
		}
	}

	if (updatedOrNewRecordsFromServer.length > 0) {
		await mergeFetchedRecordsIntoCache<T>(apiRoute, updatedOrNewRecordsFromServer, currentTimeSeconds)
	}

	// Final read is stable by requested ID order; tombstones remain hidden.
	const resolvedRecordsByID = new Map<number, T>()
	for (const id of normalizedSortedIDs) {
		const record = tableCache.get(id) as T | undefined
		if (!record) continue
		if (record.ss === 0) continue
		resolvedRecordsByID.set(id, record)
	}

	console.debug(`${LOG_PREFIX} getRecordsByIDs done.`, {
		apiRoute,
		requestedCount: normalizedSortedIDs.length,
		returnedCount: resolvedRecordsByID.size,
	})

	return resolvedRecordsByID
}

export const getRecordsByID = async <T extends IMinimalRecord>(
	apiRoute: string, ids: number[],
): Promise<Map<number, T>> => {
	// Keep one canonical pass so duplicated/invalid IDs do not create duplicated waits.
	const normalizedIDs = normalizePositiveIDs(ids)
	if (normalizedIDs.length === 0) return new Map<number, T>()

	const currentTimeSeconds = nowSeconds()
	const tableCache = cacheRecordIdTable.get(apiRoute)
	const resolvedRecordsByID = new Map<number, T>()

	const inFlightTable = getOrCreateInFlightRecordsTable(apiRoute)
	const existingBatchPromisesByID = new Map<number, Promise<Map<number, T>>>()
	const idsWithoutPromise: number[] = []

	for (const id of normalizedIDs) {
		const cachedRecord = tableCache?.get(id) as T | undefined
		if (cachedRecord) {
			if (cachedRecord.ss === 0) continue

			const cachedRecordAgeSeconds = currentTimeSeconds - (cachedRecord._fch || 0)
			const cachedRecordIsStale = cachedRecordAgeSeconds > CACHE_TIME

			// Fresh memory rows are authoritative for this call and must not re-enter batching.
			if (!cachedRecordIsStale) {
				resolvedRecordsByID.set(id, cachedRecord)
				continue
			}
		}

		const existingBatchPromise = inFlightTable.get(id) as Promise<Map<number, T>> | undefined
		if (existingBatchPromise) {
			existingBatchPromisesByID.set(id, existingBatchPromise)
			continue
		}
		idsWithoutPromise.push(id)
	}

	console.debug(
		`${LOG_PREFIX} ${apiRoute} getRecordsByID partition` +
		` | requested=${normalizedIDs.length}` +
		` | freshMemory=${resolvedRecordsByID.size}` +
		` | inFlight=${existingBatchPromisesByID.size}` +
		` | fetch=${idsWithoutPromise.length}`,
	)

	let createdFetchPromise: Promise<Map<number, T>> | null = null
	if (idsWithoutPromise.length > 0) {
		// Assign one shared batch promise per missing ID to avoid creating per-ID derived promises.
		createdFetchPromise = doGetRecordsByIDs<T>(apiRoute, idsWithoutPromise)
		for (const id of idsWithoutPromise) {
			inFlightTable.set(
				id,
				createdFetchPromise as Promise<Map<number, IMinimalRecord>>,
			)
			existingBatchPromisesByID.set(id, createdFetchPromise)
		}
		void createdFetchPromise.finally(() => {
			const currentInFlightTable = inFlightRecordsBatchPromiseByTable.get(apiRoute)
			if (!currentInFlightTable) return
			for (const id of idsWithoutPromise) {
				const currentPromise = currentInFlightTable.get(id)
				if (currentPromise !== createdFetchPromise) continue
				currentInFlightTable.delete(id)
			}
			if (currentInFlightTable.size === 0) {
				inFlightRecordsBatchPromiseByTable.delete(apiRoute)
			}
		})
	}

	const resolvedEntries = await Promise.all(
		normalizedIDs.map(async (id) => {
			const pendingBatchPromise = existingBatchPromisesByID.get(id)
			if (!pendingBatchPromise) return null
			const recordsByID = await pendingBatchPromise
			const record = recordsByID.get(id)
			if (!record) return null
			if (record.ss === 0) return null
			return [id, record] as const
		}),
	)

	for (const resolvedEntry of resolvedEntries) {
		if (!resolvedEntry) continue
		resolvedRecordsByID.set(resolvedEntry[0], resolvedEntry[1])
	}

	return resolvedRecordsByID
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

const resetPendingBufferState = (
	pendingResolvers: Map<string, { resolve: (value: any) => void; reject: (reason?: any) => void }>,
	pendingPromises: Map<string, Promise<any>>,
	bufferedIDsByTable: Map<string, Set<number>>,
	bufferedHints?: Map<string, IMinimalRecord | false>,
) => {
	for (const pending of pendingResolvers.values()) {
		pending.reject(new Error(`${LOG_PREFIX} cache cleared while request was pending`))
	}
	pendingResolvers.clear()
	pendingPromises.clear()
	bufferedIDsByTable.clear()
	bufferedHints?.clear()
}

export const clearCacheByIDs = async (): Promise<{ databaseName: string; clearedMemoryTables: number }> => {
	// Memory maps are global in this module, so clear them before dropping persisted IndexedDB state.
	const clearedMemoryTables = cacheRecordIdTable.size
	cacheRecordIdTable.clear()
	inFlightRecordsBatchPromiseByTable.clear()

	resetPendingBufferState(
		bufferedResolversByTableAndID,
		bufferedPromiseByTableAndID,
		recordIDsBufferByTable,
		bufferedCacheHintByTableAndID,
	)
	resetPendingBufferState(
		bufferedUpdatedResolversByTableAndID,
		bufferedUpdatedPromiseByTableAndID,
		updatedRecordIDsBufferByTable,
	)

	if (bufferFlushTimer) {
		clearTimeout(bufferFlushTimer)
		bufferFlushTimer = null
	}
	if (updatedBufferFlushTimer) {
		clearTimeout(updatedBufferFlushTimer)
		updatedBufferFlushTimer = null
	}
	bufferStarTime = 0
	updatedBufferStarTime = 0

	const clearedDatabase = await clearCacheByIDsDatabase(
		Env.getEmpresaID(),
		Env.enviroment || 'main',
	)
	console.debug(`${LOG_PREFIX} Cache cleared.`, {
		databaseName: clearedDatabase.databaseName,
		clearedMemoryTables,
	})
	return {
		databaseName: clearedDatabase.databaseName,
		clearedMemoryTables,
	}
}

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

	console.debug(
		`${LOG_PREFIX} flush buffered getRecordByID` +
		` | bufferedMs=${bufferedStartTime ? Date.now() - bufferedStartTime : 0}` +
		` | tables=${idsByTableToFetch.length}`,
	)

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
			console.debug(`${LOG_PREFIX} ${apiRoute} flush table batch | ids=${ids.length} | hints=${cacheHintsByID.size}`)
			const recordsByID = await doGetRecordsByIDs<any>(apiRoute, ids, cacheHintsByID)

			for (const id of ids) {
				const key = `${apiRoute}:${id}`
				const pending = bufferedResolversByTableAndID.get(key)
				if (!pending) continue
				bufferedResolversByTableAndID.delete(key)
				bufferedPromiseByTableAndID.delete(key)
				pending.resolve(recordsByID.get(id))
			}
		} catch (error) {
			console.warn(`${LOG_PREFIX} ${apiRoute} buffered fetch failed; rejecting pending promises`, error)
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

	console.debug(
		`${LOG_PREFIX} flush buffered getRecordByIDUpdated` +
		` | bufferedMs=${bufferedStartTime ? Date.now() - bufferedStartTime : 0}` +
		` | tables=${idsByTableToFetch.length}`,
	)

	for (const { apiRoute, ids } of idsByTableToFetch) {
		try {
			console.debug(`${LOG_PREFIX} ${apiRoute} flush updated table batch | ids=${ids.length}`)
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
			console.warn(`${LOG_PREFIX} ${apiRoute} buffered updated fetch failed; rejecting pending promises`, error)
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
			if (!cachedRecordIsStale) {
				return knownCachedRecord
			}
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
			if (!cachedRecordIsStale) {
				return cachedRecord
			}
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
		return localResolution.record
	}

	let returnedRecord: T | undefined
	try {
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
	return ageSeconds > CACHE_TIME
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
		return { record: memoryRecord, stale: isRecordStale(memoryRecord, currentTimeSeconds) }
	}

	const idbRecordsByID = await readRecordsFromIDBByIDs<T>(apiRoute, [numericID])
	const idbRecord = idbRecordsByID.get(numericID)
	if (!idbRecord) {
		return { record: null, stale: false }
	}

	// Promote IDB hit into memory cache for subsequent synchronous reads.
	cacheTable.set(numericID, idbRecord)
	if (idbRecord.ss === 0) return { record: null, stale: false }
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
		// Show loading spinner only when we don't have local data for first paint.
		loading = !Boolean(localResolution.record)
		if (localResolution.record) {
			record = localResolution.record
		}

		// Fresh local hit: skip network and complete.
		if (localResolution.record && !localResolution.stale) {
			loading = false
			return localResolution.record
		}

		// Local miss or stale local hit: refresh from server and update ref if changed/new.
		const refreshedRecord = await getRecordByID<T>(apiRoute, numericID, {
			cachedRecord: localResolution.record || false,
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
