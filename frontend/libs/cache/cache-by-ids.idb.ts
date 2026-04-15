/**
 * Goal:
 * Persist cache-by-ids rows in Dexie using one shared table instead of dynamic object stores.
 */
import Dexie from 'dexie'

const LOG_PREFIX = '[cache-by-ids:idb]'
const CACHE_BY_IDS_DB_PREFIX = 'cached_ids'
const CACHE_BY_IDS_DB_VERSION = 1

export interface DatabaseInformation {
	name: string
	version: number
	objectStores: string[]
}

interface ICacheByIDsRow<T = any> {
	storeName: string
	ID: number
	record: T
}

class CacheByIDsDatabase extends Dexie {
	cacheByIDs!: Dexie.Table<ICacheByIDsRow<any>, [string, number]>

	constructor(databaseName: string) {
		super(databaseName)

		// One compound-key table avoids schema upgrades per route and keeps reads partitioned by storeName.
		this.version(CACHE_BY_IDS_DB_VERSION).stores({
			cacheByIDs: '[storeName+ID],storeName',
		})
	}
}

const cacheByIDsDatabasesByName = new Map<string, CacheByIDsDatabase>()

export const makeCacheByIDsDatabaseName = (companyID: number, env: string): string => {
	// Match the existing cache naming convention: [empresa_id]_cached_ids_[environment-hash].
	return `${companyID || 0}_${CACHE_BY_IDS_DB_PREFIX}_${env || '000000'}`
}

const getCacheByIDsDatabase = (databaseName: string): CacheByIDsDatabase => {
	const cachedDatabase = cacheByIDsDatabasesByName.get(databaseName)
	if (cachedDatabase) return cachedDatabase

	const createdDatabase = new CacheByIDsDatabase(databaseName)
	cacheByIDsDatabasesByName.set(databaseName, createdDatabase)
	return createdDatabase
}

const forgetCacheByIDsDatabase = (databaseName: string) => {
	const cachedDatabase = cacheByIDsDatabasesByName.get(databaseName)
	if (cachedDatabase) {
		cachedDatabase.close()
	}
	cacheByIDsDatabasesByName.delete(databaseName)
}

const getCurrentDatabaseName = (): string => {
	return makeCacheByIDsDatabaseName(Env.getEmpresaID(), Env.enviroment || 'main')
}

import { Env } from '$core/env'

export const readRecordsFromIDBByIDs = async <T extends { ID: number }>(
	tableName: string,
	ids: number[],
): Promise<Map<number, T>> => {
	const uniqueIDs = [...new Set(ids.filter((id) => Number.isFinite(id) && id > 0))]
	if (uniqueIDs.length === 0) return new Map()

	try {
		console.debug(`${LOG_PREFIX} read:start store=${tableName} ids=${uniqueIDs.length}`)
		const rows = await getCacheByIDsDatabase(getCurrentDatabaseName()).cacheByIDs.bulkGet(
			uniqueIDs.map((id) => [tableName, id] as [string, number])
		)

		const recordsByID = new Map<number, T>()
		for (const row of rows) {
			if (!row?.record || typeof row.ID !== 'number') continue
			recordsByID.set(row.ID, row.record as T)
		}

		console.debug(`${LOG_PREFIX} read:end store=${tableName} ids=${uniqueIDs.length} hits=${recordsByID.size}`)
		return recordsByID
	} catch (error) {
		console.error(`${LOG_PREFIX} read:error store=${tableName} ids=${uniqueIDs.length}`)
		console.warn(`${LOG_PREFIX} Failed to read from IndexedDB.`, { tableName, idsCount: uniqueIDs.length }, error)
		return new Map()
	}
}

export const upsertRecordsIntoIDB = async <T extends { ID: number }>(
	tableName: string,
	records: T[],
): Promise<void> => {
	const normalizedRecords = records.filter((record) => record && typeof record.ID === 'number' && record.ID > 0)
	if (normalizedRecords.length === 0) return

	try {
		console.debug(`${LOG_PREFIX} write:start store=${tableName} rows=${normalizedRecords.length}`)
		await getCacheByIDsDatabase(getCurrentDatabaseName()).cacheByIDs.bulkPut(
			normalizedRecords.map((record) => ({
				storeName: tableName,
				ID: record.ID,
				record,
			}))
		)
		console.debug(`${LOG_PREFIX} write:end store=${tableName} rows=${normalizedRecords.length}`)
	} catch (error) {
		console.error(`${LOG_PREFIX} write:error store=${tableName} rows=${normalizedRecords.length}`)
		console.warn(`${LOG_PREFIX} Failed to upsert into IndexedDB.`, { tableName, recordsCount: normalizedRecords.length }, error)
	}
}

export const clearCacheByIDsDatabase = async (
	companyID: number,
	env: string,
): Promise<{ databaseName: string }> => {
	const databaseName = makeCacheByIDsDatabaseName(companyID, env)
	forgetCacheByIDsDatabase(databaseName)
	await Dexie.delete(databaseName)
	console.debug(`${LOG_PREFIX} clear db=${databaseName}`)
	return { databaseName }
}

export const getCacheByIDsDatabaseInfo = async (
	companyID: number,
	env: string,
): Promise<DatabaseInformation> => {
	const database = getCacheByIDsDatabase(makeCacheByIDsDatabaseName(companyID, env))
	await database.open()
	return {
		name: database.name,
		version: database.verno,
		objectStores: database.tables.map((table) => table.name),
	}
}
