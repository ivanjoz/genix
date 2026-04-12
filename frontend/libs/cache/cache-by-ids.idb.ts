/**
 * Goal:
 * Provide a minimal IndexedDB adapter for cache-by-ids using `typed-idb`,
 * auto-creating one objectStore per table (`tableName`) with keyPath `ID`.
 */
import { IndexedDBRepository, type DatabaseInformation, type IDatabase } from "../typed-idb"
import { Env } from '$core/env'

const LOG_PREFIX = "[cache-by-ids:idb]"
const CACHE_BY_IDS_DB_PREFIX = "cached_ids"

export const makeCacheByIDsDatabaseName = (companyID: number, env: string): string => {
	// Keep cache-by-ids partitioned per company/environment like the other offline caches.
	return `${companyID || 0}_${CACHE_BY_IDS_DB_PREFIX}_${env || 'main'}`
}

const makeCacheByIDsVersionStorageKey = (databaseName: string): string => {
	return `${databaseName}__version`
}

const readLocalStorageNumber = (key: string, fallback: number): number => {
	try {
		if (typeof localStorage === "undefined") return fallback
		const value = localStorage.getItem(key)
		if (!value) return fallback
		const parsedValue = Number(value)
		return Number.isFinite(parsedValue) && parsedValue > 0 ? parsedValue : fallback
	} catch {
		return fallback
	}
}

const writeLocalStorageNumber = (key: string, value: number) => {
	try {
		if (typeof localStorage === "undefined") return
		localStorage.setItem(key, String(value))
	} catch {
		// Ignore: Private mode / quota / SSR.
	}
}

class CacheByIDsDatabase implements IDatabase {
	private cachedDBByName = new Map<string, IDBDatabase>()
	private cachedVersionByName = new Map<string, number>()

	private getCurrentDatabaseName(): string {
		return makeCacheByIDsDatabaseName(Env.getEmpresaID(), Env.enviroment || 'main')
	}

	private getCurrentVersion(databaseName: string): number {
		const version = readLocalStorageNumber(makeCacheByIDsVersionStorageKey(databaseName), 1)
		return version
	}

	private getCachedDatabase(databaseName: string): IDBDatabase | null {
		return this.cachedDBByName.get(databaseName) || null
	}

	private getCachedVersion(databaseName: string): number {
		return this.cachedVersionByName.get(databaseName) || 0
	}

	private rememberDatabase(databaseName: string, database: IDBDatabase) {
		this.cachedDBByName.set(databaseName, database)
		this.cachedVersionByName.set(databaseName, database.version)
	}

	closeCachedDatabase(databaseName: string) {
		const cachedDatabase = this.cachedDBByName.get(databaseName)
		if (cachedDatabase) {
			cachedDatabase.close()
		}
		this.cachedDBByName.delete(databaseName)
		this.cachedVersionByName.delete(databaseName)
	}

	private async openWithVersion(
		databaseName: string,
		version: number,
		storesToEnsure: string[],
	): Promise<IDBDatabase> {
		return await new Promise<IDBDatabase>((resolve, reject) => {
			const openRequest = indexedDB.open(databaseName, version)

			openRequest.onupgradeneeded = () => {
				const db = openRequest.result
				for (const storeName of storesToEnsure) {
					if (db.objectStoreNames.contains(storeName)) continue
					// Cache-by-ids always keys by `ID`.
					db.createObjectStore(storeName, { keyPath: "ID" })
					console.debug(`${LOG_PREFIX} Created objectStore.`, { storeName, version })
				}
			}

			openRequest.onsuccess = () => resolve(openRequest.result)
			openRequest.onerror = () => reject(openRequest.error)
		})
	}

	private async openLatestVersion(databaseName: string): Promise<IDBDatabase> {
		return await new Promise<IDBDatabase>((resolve, reject) => {
			const openRequest = indexedDB.open(databaseName)
			openRequest.onsuccess = () => resolve(openRequest.result)
			openRequest.onerror = () => reject(openRequest.error)
		})
	}

	private isVersionError(error: unknown): boolean {
		if (!error || typeof error !== "object") return false
		const errorName = (error as { name?: string }).name
		return errorName === "VersionError"
	}

	private async openWithRecoveredVersion(
		databaseName: string,
		version: number,
		storesToEnsure: string[],
	): Promise<IDBDatabase> {
		try {
			return await this.openWithVersion(databaseName, version, storesToEnsure)
		} catch (openError) {
			if (!this.isVersionError(openError)) {
				throw openError
			}
			// Recover when localStorage version lags behind actual IndexedDB version.
			const latestDatabase = await this.openLatestVersion(databaseName)
			const actualVersion = latestDatabase.version
			writeLocalStorageNumber(makeCacheByIDsVersionStorageKey(databaseName), actualVersion)
			console.warn(`${LOG_PREFIX} Recovered IndexedDB version mismatch.`, {
				databaseName,
				requestedVersion: version,
				actualVersion,
			})
			latestDatabase.close()
			const recoveredVersion = storesToEnsure.length > 0 ? actualVersion + 1 : actualVersion
			const recoveredDatabase = await this.openWithVersion(databaseName, recoveredVersion, storesToEnsure)
			writeLocalStorageNumber(
				makeCacheByIDsVersionStorageKey(databaseName),
				recoveredDatabase.version,
			)
			return recoveredDatabase
		}
	}

	private async ensureStoresExist(databaseName: string, storeNames: string[]): Promise<IDBDatabase> {
		const currentVersion = this.getCurrentVersion(databaseName)
		const cachedDatabase = this.getCachedDatabase(databaseName)

		// Reuse cached connection if it's the correct version.
		if (cachedDatabase && this.getCachedVersion(databaseName) === currentVersion) {
			const missingStores = storeNames.filter((storeName) => !cachedDatabase.objectStoreNames.contains(storeName))
			if (missingStores.length === 0) return cachedDatabase
			// Need upgrade: close and reopen with bumped version.
			this.closeCachedDatabase(databaseName)
		}

		// Open current DB version first; upgrade only when requested stores are missing.
		let db = await this.openWithRecoveredVersion(databaseName, currentVersion, [])
		const missingStores = storeNames.filter((storeName) => !db.objectStoreNames.contains(storeName))
		if (missingStores.length === 0) {
			this.rememberDatabase(databaseName, db)
			return db
		}

		// Upgrade path: bump version by 1 and create only missing stores.
		db.close()
		const upgradedVersion = currentVersion + 1
		db = await this.openWithRecoveredVersion(databaseName, upgradedVersion, missingStores)
		writeLocalStorageNumber(makeCacheByIDsVersionStorageKey(databaseName), db.version)

		this.rememberDatabase(databaseName, db)
		return db
	}

	async openConnection(): Promise<IDBDatabase> {
		return await this.openScopedConnection(this.getCurrentDatabaseName())
	}

	async openScopedConnection(databaseName: string): Promise<IDBDatabase> {
		if (typeof indexedDB === "undefined") {
			throw new Error("IndexedDB is not available in this environment.")
		}

		// Caller might call this without store context; open the DB as-is.
		const currentVersion = this.getCurrentVersion(databaseName)
		const cachedDatabase = this.getCachedDatabase(databaseName)
		if (cachedDatabase && this.getCachedVersion(databaseName) === currentVersion) return cachedDatabase

		const db = await this.openWithRecoveredVersion(databaseName, currentVersion, [])
		this.rememberDatabase(databaseName, db)
		return db
	}

	async executeTransaction<T>(
		storeNames: string[],
		mode: IDBTransactionMode,
		operation: (tx: IDBTransaction) => Promise<T>,
	): Promise<T> {
		if (typeof indexedDB === "undefined") {
			throw new Error("IndexedDB is not available in this environment.")
		}

		const db = await this.ensureStoresExist(this.getCurrentDatabaseName(), storeNames)
		const transaction = db.transaction(storeNames, mode)

		return await new Promise<T>((resolve, reject) => {
			try {
				const operationResult = operation(transaction)
				operationResult.then(resolve).catch(reject)
				transaction.onerror = (event) => reject(event)
			} catch (error) {
				reject(error)
			}
		})
	}

	async getInfo(): Promise<DatabaseInformation> {
		const db = await this.openConnection()
		return {
			name: db.name,
			version: db.version,
			objectStores: Array.from(db.objectStoreNames),
		}
	}
}

const database = new CacheByIDsDatabase()
const repositoryByStore: Map<string, IndexedDBRepository<any>> = new Map()

const getRepositoryForStore = (storeName: string): IndexedDBRepository<any> => {
	const cached = repositoryByStore.get(storeName)
	if (cached) return cached

	// `typed-idb` repositories constrain `T` to include an `id` field. Our records are
	// keyed by `ID`, and the repository implementation doesn't actually read `id`,
	// so we use `any` to keep our records shape unchanged.
	const created = new IndexedDBRepository<any>(database)
	repositoryByStore.set(storeName, created)
	return created
}

export const readRecordsFromIDBByIDs = async <T extends { ID: number }>(
	tableName: string,
	ids: number[],
): Promise<Map<number, T>> => {
	if (ids.length === 0) return new Map()
	try {
		const repo = getRepositoryForStore(tableName)
		// `typed-idb` returns an array; map by `ID` for O(1) joins in caller.
		const records = (await repo.getByIds(tableName, ids)) as T[]
		return new Map(records.map((record) => [record.ID, record]))
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to read from IndexedDB.`, { tableName, idsCount: ids.length }, error)
		return new Map()
	}
}

export const upsertRecordsIntoIDB = async <T extends { ID: number }>(
	tableName: string,
	records: T[],
): Promise<void> => {
	if (records.length === 0) return
	try {
		const repo = getRepositoryForStore(tableName)
		// `updateMany` uses `put`, so this handles insert + update in one call.
		await repo.updateMany(tableName, records as any[])
	} catch (error) {
		console.warn(`${LOG_PREFIX} Failed to upsert into IndexedDB.`, { tableName, recordsCount: records.length }, error)
	}
}

const deleteIndexedDBDatabase = async (databaseName: string): Promise<void> => {
	await new Promise<void>((resolve, reject) => {
		const deleteRequest = indexedDB.deleteDatabase(databaseName)
		deleteRequest.onsuccess = () => resolve()
		deleteRequest.onerror = () => reject(deleteRequest.error)
		deleteRequest.onblocked = () => reject(new Error(`Deletion blocked for ${databaseName}`))
	})
}

export const clearCacheByIDsDatabase = async (
	companyID: number,
	env: string,
): Promise<{ databaseName: string }> => {
	if (typeof indexedDB === "undefined") {
		throw new Error("IndexedDB is not available in this environment.")
	}

	const databaseName = makeCacheByIDsDatabaseName(companyID, env)
	database.closeCachedDatabase(databaseName)
	repositoryByStore.clear()
	await deleteIndexedDBDatabase(databaseName)

	try {
		if (typeof localStorage !== "undefined") {
			localStorage.removeItem(makeCacheByIDsVersionStorageKey(databaseName))
		}
	} catch {
		// Ignore: private mode or restricted storage should not block cache cleanup.
	}

	console.debug(`${LOG_PREFIX} Cache database cleared.`, { databaseName })
	return { databaseName }
}
