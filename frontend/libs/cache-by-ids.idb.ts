/**
 * Goal:
 * Provide a minimal IndexedDB adapter for cache-by-ids using `typed-idb`,
 * auto-creating one objectStore per table (`tableName`) with keyPath `ID`.
 */
import { IndexedDBRepository, type DatabaseInformation, type IDatabase } from "./typed-idb"

const LOG_PREFIX = "[cache-by-ids:idb]"
const IDB_DB_NAME = "cached_ids"
const IDB_DB_VERSION_KEY = `${IDB_DB_NAME}__version`

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
	private cachedDB: IDBDatabase | null = null
	private cachedVersion = 0

	private getCurrentVersion(): number {
		const version = readLocalStorageNumber(IDB_DB_VERSION_KEY, 1)
		return version
	}

	private async openWithVersion(version: number, storesToEnsure: string[]): Promise<IDBDatabase> {
		return await new Promise<IDBDatabase>((resolve, reject) => {
			const openRequest = indexedDB.open(IDB_DB_NAME, version)

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

	private async ensureStoresExist(storeNames: string[]): Promise<IDBDatabase> {
		const currentVersion = this.getCurrentVersion()

		// Reuse cached connection if it's the correct version.
		if (this.cachedDB && this.cachedVersion === currentVersion) {
			const missingStores = storeNames.filter((storeName) => !this.cachedDB!.objectStoreNames.contains(storeName))
			if (missingStores.length === 0) return this.cachedDB
			// Need upgrade: close and reopen with bumped version.
			this.cachedDB.close()
			this.cachedDB = null
			this.cachedVersion = 0
		}

		// Open current DB version first; upgrade only when requested stores are missing.
		let db = await this.openWithVersion(currentVersion, [])
		const missingStores = storeNames.filter((storeName) => !db.objectStoreNames.contains(storeName))
		if (missingStores.length === 0) {
			this.cachedDB = db
			this.cachedVersion = db.version
			return db
		}

		// Upgrade path: bump version by 1 and create only missing stores.
		db.close()
		const upgradedVersion = currentVersion + 1
		db = await this.openWithVersion(upgradedVersion, missingStores)
		writeLocalStorageNumber(IDB_DB_VERSION_KEY, upgradedVersion)

		this.cachedDB = db
		this.cachedVersion = db.version
		return db
	}

	async openConnection(): Promise<IDBDatabase> {
		if (typeof indexedDB === "undefined") {
			throw new Error("IndexedDB is not available in this environment.")
		}

		// Caller might call this without store context; open the DB as-is.
		const currentVersion = this.getCurrentVersion()
		if (this.cachedDB && this.cachedVersion === currentVersion) return this.cachedDB

		const db = await this.openWithVersion(currentVersion, [])
		this.cachedDB = db
		this.cachedVersion = db.version
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

		const db = await this.ensureStoresExist(storeNames)
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
