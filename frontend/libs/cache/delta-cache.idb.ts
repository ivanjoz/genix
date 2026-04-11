import Dexie, { type EntityTable } from 'dexie'
import type {
  CacheRecordID,
  ICacheRecordRow,
  ICacheRouteRow,
  IDeltaCacheRouteRef,
  ILastSync,
} from './delta-cache.types'

const CACHE_DB_VERSION = 2

const deltaCacheDatabasesByName = new Map<string, DeltaCacheDatabase>()
const routeMemoryByLookupKey = new Map<string, ICacheRouteRow>()

class DeltaCacheDatabase extends Dexie {
  cacheRoutes!: EntityTable<ICacheRouteRow, 'id'>
  cacheRecords!: Dexie.Table<ICacheRecordRow, [number, string, CacheRecordID]>

  constructor(databaseName: string) {
    super(databaseName)

    // The route table owns cache metadata, while records are persisted row-by-row by route ID.
    this.version(CACHE_DB_VERSION).stores({
      cacheRoutes: '++id,&routeLookupKey,module',
      cacheRecords: '[cR+rK+ID],[cR+ss]',
    })
  }
}

export const makeDeltaCacheDatabaseName = (companyID: number, env: string): string => {
  return `${companyID || 0}_delta_cache_${env || 'main'}`
}

const getDeltaCacheDatabase = (dbName: string): DeltaCacheDatabase => {
  const cachedDatabase = deltaCacheDatabasesByName.get(dbName)
  if (cachedDatabase) return cachedDatabase

  const createdDatabase = new DeltaCacheDatabase(dbName)
  deltaCacheDatabasesByName.set(dbName, createdDatabase)
  return createdDatabase
}

const getRouteMemoryKey = (routeRow: Pick<ICacheRouteRow, 'dbName' | 'routeLookupKey'>): string => {
  // The same service worker can serve multiple scoped databases, so memory keys include dbName.
  return [routeRow.dbName, routeRow.routeLookupKey].join('::')
}

const rememberRouteRow = (routeRow: ICacheRouteRow) => {
  // Route metadata is cached in memory because every fetch path resolves route -> cacheRouteId first.
  routeMemoryByLookupKey.set(getRouteMemoryKey(routeRow), routeRow)
  return routeRow
}

const forgetRouteRow = (routeRow: Pick<ICacheRouteRow, 'dbName' | 'routeLookupKey'>) => {
  routeMemoryByLookupKey.delete(getRouteMemoryKey(routeRow))
}

const deleteRouteRecords = async (routeRow: ICacheRouteRow): Promise<void> => {
  // Dexie exposes `cR` as a virtual prefix index from `[cR+ss]`.
  await getDeltaCacheDatabase(routeRow.dbName).cacheRecords
    .where('cR')
    .equals(routeRow.id as number)
    .delete()
}

const countRouteRecords = async (routeRow: ICacheRouteRow): Promise<number> => {
  return await getDeltaCacheDatabase(routeRow.dbName).cacheRecords
    .where('cR')
    .equals(routeRow.id as number)
    .count()
}

const getPersistedRouteRow = async (routeRef: IDeltaCacheRouteRef): Promise<ICacheRouteRow | undefined> => {
  return await getDeltaCacheDatabase(routeRef.dbName).cacheRoutes
    .where('routeLookupKey')
    .equals(routeRef.routeLookupKey)
    .first()
}

export const makeEmptyLastSync = (version: number): ILastSync => ({
  fetchTime: 0,
  updatedStatus: {},
  fetchedRecordsCount: 0,
  fetchedBytes: 0,
  __version__: version,
})

export const getCacheRouteRow = async (routeRef: IDeltaCacheRouteRef): Promise<ICacheRouteRow | undefined> => {
  // Fast path: the SW keeps only route metadata in memory, not the full snapshots.
  const cachedRouteRow = routeMemoryByLookupKey.get(getRouteMemoryKey(routeRef))
  if (cachedRouteRow) {
    return cachedRouteRow
  }

  const routeRow = await getPersistedRouteRow(routeRef)

  if (!routeRow) { return undefined }
  return rememberRouteRow(routeRow)
}

export const ensureCacheRouteRow = async (routeRef: IDeltaCacheRouteRef): Promise<ICacheRouteRow> => {
  // Route rows are created lazily, only when a network response needs to be persisted.
  const existingRouteRow = await getCacheRouteRow(routeRef)
  if (existingRouteRow) {
    return existingRouteRow
  }

  const nextRouteRow: ICacheRouteRow = {
    ...makeEmptyLastSync(routeRef.version),
    routeLookupKey: routeRef.routeLookupKey,
    dbName: routeRef.dbName,
    env: routeRef.env,
    module: routeRef.module,
    route: routeRef.route,
    partitionValue: routeRef.partitionValue,
    cacheKey: routeRef.cacheKey,
    responseKeys: [],
  }

  nextRouteRow.id = await getDeltaCacheDatabase(routeRef.dbName).cacheRoutes.add(nextRouteRow)
  return rememberRouteRow(nextRouteRow)
}

export const saveCacheRouteRow = async (routeRow: ICacheRouteRow): Promise<ICacheRouteRow> => {
  // Every metadata write keeps the memory index in sync with IndexedDB.
  await getDeltaCacheDatabase(routeRow.dbName).cacheRoutes.put(routeRow)
  return rememberRouteRow(routeRow)
}

export const resetCacheRouteRow = async (
  routeRow: ICacheRouteRow, version: number
): Promise<ICacheRouteRow> => {
  // A cache version bump discards every persisted row for that route in one transaction.
  const database = getDeltaCacheDatabase(routeRow.dbName)
  await database.transaction('rw', database.cacheRoutes, database.cacheRecords, async () => {
    await deleteRouteRecords(routeRow)
    Object.assign(routeRow, makeEmptyLastSync(version), {
      responseKeys: [],
    })
    await database.cacheRoutes.put(routeRow)
  })

  return rememberRouteRow(routeRow)
}

export const listRouteRecordRows = async (routeRow: ICacheRouteRow): Promise<ICacheRecordRow[]> => {
  // Full snapshot reads are rebuilt from row storage only when the caller explicitly asks for them.
  return await getDeltaCacheDatabase(routeRow.dbName).cacheRecords
    .where('cR')
    .equals(routeRow.id as number)
    .toArray()
}

export const bulkGetRouteRecordRows = async (
  dbName: string,
  rowKeys: [number, string, CacheRecordID][]
): Promise<(ICacheRecordRow | undefined)[]> => {
  if (rowKeys.length === 0) { return [] }
  return await getDeltaCacheDatabase(dbName).cacheRecords.bulkGet(rowKeys)
}

export const bulkPutRouteRecordRows = async (dbName: string, rows: ICacheRecordRow[]): Promise<void> => {
  if (rows.length === 0) { return }
  await getDeltaCacheDatabase(dbName).cacheRecords.bulkPut(rows)
}

export const bulkDeleteRouteRecordRows = async (
  dbName: string,
  rowKeys: [number, string, CacheRecordID][]
): Promise<void> => {
  if (rowKeys.length === 0) { return }
  await getDeltaCacheDatabase(dbName).cacheRecords.bulkDelete(rowKeys)
}

export const replaceRouteRecordRows = async (
  routeRow: ICacheRouteRow, responseKeys: string[], rows: ICacheRecordRow[]
): Promise<void> => {
  // Initial syncs replace the entire logical snapshot for the route in one transaction.
  const database = getDeltaCacheDatabase(routeRow.dbName)
  await database.transaction('rw', database.cacheRoutes, database.cacheRecords, async () => {
    await deleteRouteRecords(routeRow)
    if (rows.length > 0) {
      await database.cacheRecords.bulkPut(rows)
    }
    routeRow.responseKeys = [...responseKeys]
    await database.cacheRoutes.put(routeRow)
  })

  rememberRouteRow(routeRow)
}

export const extendRouteResponseKeys = async (
  routeRow: ICacheRouteRow, responseKeys: string[]
): Promise<ICacheRouteRow> => {
  // Delta responses may introduce a response key before any record exists for it.
  const nextResponseKeys = new Set(routeRow.responseKeys || [])
  for (const responseKey of responseKeys) {
    nextResponseKeys.add(responseKey)
  }
  routeRow.responseKeys = [...nextResponseKeys]
  return await saveCacheRouteRow(routeRow)
}

export const readCachedRouteResponse = async (routeRow: ICacheRouteRow): Promise<any | undefined> => {
  // The public worker contract still expects grouped arrays by response key.
  const recordRows = await listRouteRecordRows(routeRow)
  if (recordRows.length === 0 && (!routeRow.responseKeys || routeRow.responseKeys.length === 0)) {
    return undefined
  }

  const responseContent = {} as { [key: string]: any[] } & { __version__?: number }
  for (const responseKey of routeRow.responseKeys || []) {
    responseContent[responseKey] = []
  }

  for (const recordRow of recordRows) {
    if (!responseContent[recordRow.rK]) {
      responseContent[recordRow.rK] = []
    }
    responseContent[recordRow.rK].push(recordRow.E)
  }

  responseContent.__version__ = routeRow.__version__
  return responseContent
}

export const getRecordRowsByResponseKey = async (
  routeRow: ICacheRouteRow, responseKey: string
): Promise<ICacheRecordRow[]> => {
  // Sub-object reads use the primary key prefix `(cR, rK, *)`.
  return await getDeltaCacheDatabase(routeRow.dbName).cacheRecords
    .where('[cR+rK+ID]')
    .between(
      [routeRow.id as number, responseKey, Dexie.minKey],
      [routeRow.id as number, responseKey, Dexie.maxKey]
    )
    .toArray()
}

export const setRouteForceNetwork = async (
  routeRef: IDeltaCacheRouteRef, forceNetwork: boolean
): Promise<boolean> => {
  const routeRow = await getCacheRouteRow(routeRef)
  if (!routeRow) { return false }
  routeRow.forceNetwork = forceNetwork
  await saveCacheRouteRow(routeRow)
  return true
}

export const verifyRouteMemoryState = async (routeRef: IDeltaCacheRouteRef): Promise<void> => {
  // Reconcile the in-memory fast path with IndexedDB after the current request finishes.
  const memoryRouteRow = routeMemoryByLookupKey.get(getRouteMemoryKey(routeRef))
  if (!memoryRouteRow) { return }

  const persistedRouteRow = await getPersistedRouteRow(routeRef)
  if (!persistedRouteRow) {
    forgetRouteRow(memoryRouteRow)
    return
  }

  rememberRouteRow(persistedRouteRow)
  if (!persistedRouteRow.fetchTime) { return }

  const recordsCount = await countRouteRecords(persistedRouteRow)
  if (recordsCount > 0) { return }

  console.warn(`[DeltaCache] Resetting corrupted route cache: ${persistedRouteRow.cacheKey}`)
  await resetCacheRouteRow(persistedRouteRow, persistedRouteRow.__version__ || routeRef.version)
}

export const refreshRoutesByPrefix = async (
  dbName: string, module: string, routes: string[]
): Promise<number> => {
  // Prefix matching preserves the existing refreshRoutes contract used by POST handlers.
  const database = getDeltaCacheDatabase(dbName)
  const routeRows = await database.cacheRoutes
    .where('module')
    .equals(module)
    .toArray()

  let updatedRoutesCount = 0
  for (const routeRow of routeRows) {
    if (!routes.some((routePrefix) => routeRow.route.startsWith(routePrefix))) {
      continue
    }
    routeRow.forceNetwork = true
    await database.cacheRoutes.put(routeRow)
    rememberRouteRow(routeRow)
    updatedRoutesCount++
  }

  return updatedRoutesCount
}

export const clearEnvironmentCache = async (dbName: string): Promise<number> => {
  // The database name already scopes environment and company, so cleanup drops every route in it.
  const database = getDeltaCacheDatabase(dbName)
  const routeRows = await database.cacheRoutes.toArray()

  if (routeRows.length === 0) { return 0 }

  const routeIDs = routeRows.map((routeRow) => routeRow.id as number)
  await database.transaction('rw', database.cacheRoutes, database.cacheRecords, async () => {
    for (const routeRow of routeRows) {
      await deleteRouteRecords(routeRow)
    }
    await database.cacheRoutes.bulkDelete(routeIDs)
  })

  for (const routeRow of routeRows) {
    forgetRouteRow(routeRow)
  }

  return routeRows.length
}

export const clearModuleCache = async (dbName: string, module: string): Promise<number> => {
  // Module cleanup is useful for debug operations that want narrower invalidation.
  const database = getDeltaCacheDatabase(dbName)
  const routeRows = await database.cacheRoutes
    .where('module')
    .equals(module)
    .toArray()

  if (routeRows.length === 0) { return 0 }

  const routeIDs = routeRows.map((routeRow) => routeRow.id as number)
  await database.transaction('rw', database.cacheRoutes, database.cacheRecords, async () => {
    for (const routeRow of routeRows) {
      await deleteRouteRecords(routeRow)
    }
    await database.cacheRoutes.bulkDelete(routeIDs)
  })

  for (const routeRow of routeRows) {
    forgetRouteRow(routeRow)
  }

  return routeRows.length
}

export const listEnvironmentCacheStats = async (dbName: string) => {
  // Stats are grouped by module to replace the old cache-storage based debug helper.
  const routeRows = await getDeltaCacheDatabase(dbName).cacheRoutes.toArray()

  const moduleStats = new Map<string, { module: string, routes: number, records: number }>()

  for (const routeRow of routeRows) {
    const moduleStat = moduleStats.get(routeRow.module) || { module: routeRow.module, routes: 0, records: 0 }
    moduleStat.routes++
    moduleStat.records += await countRouteRecords(routeRow)
    moduleStats.set(routeRow.module, moduleStat)
    rememberRouteRow(routeRow)
  }

  return [...moduleStats.values()]
}
