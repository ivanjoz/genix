import Dexie from 'dexie'
import { Env } from '$core/env'
import type { ICacheDebugRow } from './cache-debug.types'

const GROUP_CACHE_DB_VERSION = 2
const LOG_PREFIX = '[group-cache:idb]'
const groupCacheDatabasesByName = new Map<string, GroupCacheDatabase>()

export interface IGroupCacheRecord<T = any> {
	// Mirrors db.RecordGroup JSON: ig identifies the backend index group that produced this bucket.
	ig: number
	// GroupHash is signed int32 in Go; it is converted to uint32 only when sent as cc-gh.
	id: number
	// Backend planner values are the stable identity for a bucket inside one query shape.
	igVal: number[]
	// The actual payload is stored only on the primary row, never in metadata indexes.
	records: T[]
	// UpdateCounter is compared with backend freshness metadata to skip unchanged buckets.
	upc: number
}

export interface IGroupCacheRow<T = any> extends IGroupCacheRecord<T> {
	// Route plus sorted param names; values are excluded so equivalent query shapes share metadata.
	queryShape: string
	// String form of igVal used as the second primary-key part for exact bucket lookup.
	key: string
	// Local timestamp is only diagnostic for now, but helps inspect cache age in DevTools.
	fetchTime: number
}

export interface IGroupCacheMetadata {
	// These fields are duplicated in an index so cc-gh/cc-upc can be built without loading records.
	id: number
	upc: number
}

class GroupCacheDatabase extends Dexie {
	groupRows!: Dexie.Table<IGroupCacheRow<any>, [string, string]>

	constructor(databaseName: string) {
		super(databaseName)

		// Indexed metadata lets GETWithGroupCache send freshness keys without loading record JSON.
		this.version(1).stores({
			groupRows: '[queryShape+key],[queryShape+id+upc],queryShape',
		})
		this.version(GROUP_CACHE_DB_VERSION).stores({
			// Primary key fetches exact groups; secondary index streams all freshness pairs by shape.
			groupRows: '[queryShape+key],[queryShape+id+upc],queryShape',
		}).upgrade(async (transaction) => {
			// Version bumps can discard report caches; the backend remains the source of truth.
			await transaction.table('groupRows').clear()
		})
	}
}

const makeGroupCacheDatabaseName = (): string => {
	// Company and API endpoint live in the database name, so rows do not need partition fields.
	return `${Env.getEmpresaID()}_group_cache_${Env.enviroment || 'main'}`
}

const getGroupCacheDatabase = (): GroupCacheDatabase => {
	const databaseName = makeGroupCacheDatabaseName()
	const cachedDatabase = groupCacheDatabasesByName.get(databaseName)
	if (cachedDatabase) return cachedDatabase

	const createdDatabase = new GroupCacheDatabase(databaseName)
	groupCacheDatabasesByName.set(databaseName, createdDatabase)
	return createdDatabase
}

const deleteCachedGroupDatabaseReference = (databaseName: string) => {
	// Removing the in-memory instance prevents stale handles after cache cleanup.
	groupCacheDatabasesByName.delete(databaseName)
}

export const makeGroupCacheKey = (groupRecord: Pick<IGroupCacheRecord, 'igVal'>): string => {
	if (!Array.isArray(groupRecord.igVal) || groupRecord.igVal.length === 0) {
		return ''
	}
	// igVal order is produced by the backend planner, so this key is stable for a specific bucket.
	return groupRecord.igVal.join('_')
}

export const makeGroupQueryShape = (route: string, uriParamNames: string[]): string => {
	// Values are intentionally omitted: all requests with the same filter shape can share freshness metadata.
	return [route, ...uriParamNames.filter(Boolean).sort()].join('|')
}

export const readGroupCacheMetadata = async (queryShape: string): Promise<IGroupCacheMetadata[]> => {
	try {
		// keys() reads only the compound index values; it avoids deserializing the cached records payload.
		const indexKeys = await getGroupCacheDatabase().groupRows
			.where('[queryShape+id+upc]')
			.between(
				[queryShape, Dexie.minKey, Dexie.minKey],
				[queryShape, Dexie.maxKey, Dexie.maxKey],
			)
			.keys()

		const metadataRows: IGroupCacheMetadata[] = []
		for (const indexKey of indexKeys) {
			const [, id, upc] = indexKey as unknown as [string, number, number]
			if (typeof id !== 'number' || typeof upc !== 'number') {
				console.warn(`${LOG_PREFIX} Invalid metadata index key.`, indexKey)
				continue
			}
			metadataRows.push({ id, upc })
		}

		console.debug(`${LOG_PREFIX} Metadata read.`, { queryShape, groups: metadataRows.length })
		return metadataRows
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to read metadata.`, { queryShape }, error)
		return []
	}
}

export const readGroupCacheRows = async <T>(
	queryShape: string,
	keys: string[],
): Promise<Map<string, IGroupCacheRow<T>>> => {
	// The backend response tells us exactly which igVal keys belong to the current request.
	const uniqueKeys = [...new Set(keys.filter(Boolean))]
	if (uniqueKeys.length === 0) return new Map()

	try {
		// bulkGet uses the primary key [queryShape, key], so unchanged groups can be restored directly.
		const rows = await getGroupCacheDatabase().groupRows.bulkGet(
			uniqueKeys.map((key) => [queryShape, key] as [string, string])
		) as (IGroupCacheRow<T> | undefined)[]

		const rowsByKey = new Map<string, IGroupCacheRow<T>>()
		for (const row of rows) {
			if (row) rowsByKey.set(row.key, row)
		}

		console.debug(`${LOG_PREFIX} Rows read.`, { queryShape, requested: uniqueKeys.length, found: rowsByKey.size })
		return rowsByKey
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to read rows.`, { queryShape, keys: uniqueKeys.length }, error)
		return new Map()
	}
}

export const upsertGroupCacheRows = async <T>(
	queryShape: string,
	groupRecords: IGroupCacheRecord<T>[],
): Promise<void> => {
	if (groupRecords.length === 0) return

	const fetchTime = Math.floor(Date.now() / 1000)
	const rows: IGroupCacheRow<T>[] = []
	for (const groupRecord of groupRecords) {
		// Records without igVal cannot be matched with backend metadata-only responses later.
		const key = makeGroupCacheKey(groupRecord)
		if (!key) {
			console.warn(`${LOG_PREFIX} Skipping group without igVal key.`, groupRecord)
			continue
		}

		rows.push({
			queryShape,
			key,
			ig: Number(groupRecord.ig || 0),
			id: Number(groupRecord.id || 0),
			igVal: [...groupRecord.igVal],
			records: Array.isArray(groupRecord.records) ? groupRecord.records : [],
			upc: Number(groupRecord.upc || 0),
			fetchTime,
		})
	}

	if (rows.length === 0) return

	try {
		// bulkPut replaces changed groups and leaves unrelated groups in the same query shape untouched.
		await getGroupCacheDatabase().groupRows.bulkPut(rows)
		console.debug(`${LOG_PREFIX} Rows upserted.`, { queryShape, rows: rows.length })
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to upsert rows.`, { queryShape, rows: rows.length }, error)
	}
}

export const listGroupCacheStats = async (): Promise<ICacheDebugRow[]> => {
	try {
		// Primary keys expose queryShape without deserializing the records payload stored in each row.
		const primaryKeys = await getGroupCacheDatabase().groupRows
			.orderBy('queryShape')
			.primaryKeys() as [string, string][]

		const rowsCountByQueryShape = new Map<string, number>()
		for (const primaryKey of primaryKeys) {
			const queryShape = String(primaryKey?.[0] || '')
			if (!queryShape) { continue }
			rowsCountByQueryShape.set(queryShape, (rowsCountByQueryShape.get(queryShape) || 0) + 1)
		}

		const statsRows = [...rowsCountByQueryShape.entries()].map(([queryShape, recordsCount]) => ({
			source: 'group' as const,
			baseRoute: queryShape.split('|')[0] || '(sin ruta)',
			apiRoute: queryShape,
			recordsCount,
		}))

		console.debug(`${LOG_PREFIX} Stats listed.`, {
			queryShapes: statsRows.length,
			rows: primaryKeys.length,
		})
		return statsRows
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to list stats.`, error)
		return []
	}
}

export const clearGroupCache = async (): Promise<number> => {
	const databaseName = makeGroupCacheDatabaseName()
	const database = getGroupCacheDatabase()

	try {
		// Row count is returned so the UI can report how much cached grouped data was removed.
		const deletedRows = await database.groupRows.count()
		database.close()
		deleteCachedGroupDatabaseReference(databaseName)
		await Dexie.delete(databaseName)
		console.debug(`${LOG_PREFIX} Cache cleared.`, { databaseName, deletedRows })
		return deletedRows
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to clear cache.`, { databaseName }, error)
		throw error
	}
}
