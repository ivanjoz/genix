import Dexie, { type EntityTable } from 'dexie'
import type { ICacheDebugRow } from './cache-debug.types'
import type { IGroupCacheRow } from './group-cache.idb'
import type {
  CacheRecordID,
  ICacheRecordRow,
  ICacheRecordRowMulti,
  ICacheRecordRowSingle,
  ICacheRouteRow,
  IDeltaCacheRouteRef,
  ILastSync,
  IRequestLogRow,
} from './delta-cache.types'

const CACHE_DB_VERSION = 4

const deltaCacheDatabasesByName = new Map<string, DeltaCacheDatabase>()
const routeMemoryByLookupKey = new Map<string, ICacheRouteRow>()

class DeltaCacheDatabase extends Dexie {
  cacheRoutes!: EntityTable<ICacheRouteRow, 'id'>
  cacheRecords!: Dexie.Table<ICacheRecordRowMulti, [number, number, CacheRecordID]>
  cacheRecordsSingle!: Dexie.Table<ICacheRecordRowSingle, [number, CacheRecordID]>
  requestLogs!: EntityTable<IRequestLogRow, 'id'>
  groupRows!: Dexie.Table<IGroupCacheRow<any>, [string, string]>

  constructor(databaseName: string) {
    super(databaseName)

    // The route table owns cache metadata, while records are persisted row-by-row by route ID.
    // Multi-key routes live in `cacheRecords` keyed `[_r+_k+ID]`; array routes use
    // `cacheRecordsSingle` keyed `[_r+ID]` since their responseKey is always `_default`.
    this.version(CACHE_DB_VERSION).stores({
      cacheRoutes: '++id,&routeLookupKey,module',
      cacheRecords: '[_r+_k+ID],[_r+ss]',
      cacheRecordsSingle: '[_r+ID],[_r+ss]',
      requestLogs: '&id,route',
      groupRows: '[queryShape+key],[queryShape+id+upc],queryShape',
    })
  }
}

export const makeCacheDatabaseName = (companyID: number, env: string): string => {
  return `${companyID || 0}_cache_${env || 'main'}`
}

export const makeDeltaCacheDatabaseName = (companyID: number, env: string): string => {
  return makeCacheDatabaseName(companyID, env)
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

const forgetRoutesByDatabaseName = (dbName: string) => {
  // Full environment cleanup must evict stale memory rows even if IndexedDB is already empty.
  for (const routeMemoryKey of routeMemoryByLookupKey.keys()) {
    if (!routeMemoryKey.startsWith(`${dbName}::`)) { continue }
    routeMemoryByLookupKey.delete(routeMemoryKey)
  }
}

const getRecordsTable = (database: DeltaCacheDatabase, routeRow: Pick<ICacheRouteRow, 'isSingle'>) => {
  return routeRow.isSingle ? database.cacheRecordsSingle : database.cacheRecords
}

const deleteRouteRecords = async (routeRow: ICacheRouteRow): Promise<void> => {
  // Dexie exposes `_r` as a virtual prefix index from `[_r+ss]`.
  await getRecordsTable(getDeltaCacheDatabase(routeRow.dbName), routeRow)
    .where('_r')
    .equals(routeRow.id as number)
    .delete()
}

const countRouteRecords = async (routeRow: ICacheRouteRow): Promise<number> => {
  return await getRecordsTable(getDeltaCacheDatabase(routeRow.dbName), routeRow)
    .where('_r')
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
  const recordsTable = getRecordsTable(database, routeRow)
  await database.transaction('rw', database.cacheRoutes, recordsTable, async () => {
    await deleteRouteRecords(routeRow)
    const preservedIsSingle = routeRow.isSingle
    Object.assign(routeRow, makeEmptyLastSync(version), {
      responseKeys: [],
      isSingle: preservedIsSingle,
    })
    await database.cacheRoutes.put(routeRow)
  })

  return rememberRouteRow(routeRow)
}

export const listRouteRecordRows = async (routeRow: ICacheRouteRow): Promise<ICacheRecordRow[]> => {
  // Full snapshot reads are rebuilt from row storage only when the caller explicitly asks for them.
  return await getRecordsTable(getDeltaCacheDatabase(routeRow.dbName), routeRow)
    .where('_r')
    .equals(routeRow.id as number)
    .toArray()
}

export const bulkPutRouteRecordRows = async (
  routeRow: Pick<ICacheRouteRow, 'dbName' | 'isSingle'>, rows: ICacheRecordRow[]
): Promise<void> => {
  if (rows.length === 0) { return }
  await getRecordsTable(getDeltaCacheDatabase(routeRow.dbName), routeRow).bulkPut(rows as any)
}

export const bulkDeleteRouteRecordRowsMulti = async (
  dbName: string,
  rowKeys: [number, number, CacheRecordID][]
): Promise<void> => {
  if (rowKeys.length === 0) { return }
  await getDeltaCacheDatabase(dbName).cacheRecords.bulkDelete(rowKeys)
}

export const bulkDeleteRouteRecordRowsSingle = async (
  dbName: string,
  rowKeys: [number, CacheRecordID][]
): Promise<void> => {
  if (rowKeys.length === 0) { return }
  await getDeltaCacheDatabase(dbName).cacheRecordsSingle.bulkDelete(rowKeys)
}

export const replaceRouteRecordRows = async (
  routeRow: ICacheRouteRow, responseKeys: string[], rows: ICacheRecordRow[]
): Promise<void> => {
  // Initial syncs replace the entire logical snapshot for the route in one transaction.
  const database = getDeltaCacheDatabase(routeRow.dbName)
  const recordsTable = getRecordsTable(database, routeRow)
  await database.transaction('rw', database.cacheRoutes, recordsTable, async () => {
    await deleteRouteRecords(routeRow)
    if (rows.length > 0) {
      await recordsTable.bulkPut(rows as any)
    }
    routeRow.responseKeys = [...responseKeys]
    await database.cacheRoutes.put(routeRow)
  })

  rememberRouteRow(routeRow)
}

export const addRequestLogRow = async (dbName: string, requestLogRow: Omit<IRequestLogRow, 'id'>): Promise<IRequestLogRow> => {
  const database = getDeltaCacheDatabase(dbName)
  let nextRequestLogRow = {} as IRequestLogRow

  await database.transaction('rw', database.requestLogs, async () => {
    const createdAtMilliseconds = Date.now()
    const lastLogRow = await database.requestLogs.orderBy('id').last()

    // Keep IDs monotonic even when multiple requests finish inside the same millisecond.
    nextRequestLogRow = {
      ...requestLogRow,
      id: Math.max(createdAtMilliseconds, (lastLogRow?.id || 0) + 1),
    }

    await database.requestLogs.add(nextRequestLogRow)
  })

  return nextRequestLogRow
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

export const stripRowMeta = (row: ICacheRecordRow): any => {
  // Flatten-back: the row *is* the record, minus internal indexing fields.
  const { _r, _k, ...record } = row as ICacheRecordRowMulti
  void _r; void _k
  return record
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

  if (routeRow.isSingle) {
    // Single routes always resolve to `_default`; the stored rows have no `_k`.
    const bucket = responseContent._default || (responseContent._default = [])
    for (const recordRow of recordRows) {
      bucket.push(stripRowMeta(recordRow))
    }
  } else {
    const responseKeys = routeRow.responseKeys || []
    for (const recordRow of recordRows) {
      const keyIndex = (recordRow as ICacheRecordRowMulti)._k
      const responseKey = responseKeys[keyIndex - 1]
      if (!responseKey) { continue }
      if (!responseContent[responseKey]) {
        responseContent[responseKey] = []
      }
      responseContent[responseKey].push(stripRowMeta(recordRow))
    }
  }

  responseContent.__version__ = routeRow.__version__
  return responseContent
}

export const getRecordRowsByResponseKey = async (
  routeRow: ICacheRouteRow, responseKey: string
): Promise<ICacheRecordRow[]> => {
  const database = getDeltaCacheDatabase(routeRow.dbName)

  if (routeRow.isSingle) {
    // Single routes only ever expose `_default`; scan the whole route partition.
    if (responseKey !== '_default') { return [] }
    return await database.cacheRecordsSingle
      .where('[_r+ID]')
      .between(
        [routeRow.id as number, Dexie.minKey],
        [routeRow.id as number, Dexie.maxKey]
      )
      .toArray()
  }

  const keyIndex = (routeRow.responseKeys || []).indexOf(responseKey)
  if (keyIndex === -1) { return [] }
  return await database.cacheRecords
    .where('[_r+_k+ID]')
    .between(
      [routeRow.id as number, keyIndex + 1, Dexie.minKey],
      [routeRow.id as number, keyIndex + 1, Dexie.maxKey]
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
  // The database name already scopes environment and company, so we can safely wipe cache data and request logs.
  const database = getDeltaCacheDatabase(dbName)
  const routeRows = await database.cacheRoutes.toArray()

  await database.transaction(
    'rw',
    [database.cacheRoutes, database.cacheRecords, database.cacheRecordsSingle, database.requestLogs, database.groupRows],
    async () => {
      // Clearing both record stores also removes orphan record rows that no longer have route metadata.
      await database.cacheRecords.clear()
      await database.cacheRecordsSingle.clear()
      await database.cacheRoutes.clear()
      await database.requestLogs.clear()
      await database.groupRows.clear()
    }
  )

  for (const routeRow of routeRows) {
    forgetRouteRow(routeRow)
  }
  forgetRoutesByDatabaseName(dbName)

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
  await database.transaction('rw', database.cacheRoutes, database.cacheRecords, database.cacheRecordsSingle, async () => {
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

export const listEnvironmentCacheRouteStats = async (dbName: string): Promise<ICacheDebugRow[]> => {
  // Route-level stats power the cache inspector UI without rebuilding full response payloads.
  const routeRows = await getDeltaCacheDatabase(dbName).cacheRoutes.toArray()
  const routeStats = await Promise.all(routeRows.map(async (routeRow) => {
    rememberRouteRow(routeRow)
    return {
      source: 'delta' as const,
      baseRoute: String(routeRow.route || '').split('?')[0] || '(sin ruta)',
      apiRoute: routeRow.route || '',
      recordsCount: await countRouteRecords(routeRow),
      // The current inspector uses the persisted fetch payload bytes as the only available size metric.
      sizeMB: Number(routeRow.fetchedBytes || 0) / (1024 * 1024),
    }
  }))

  return routeStats.sort((leftRow, rightRow) => {
    if (leftRow.baseRoute === rightRow.baseRoute) {
      return leftRow.apiRoute.localeCompare(rightRow.apiRoute)
    }
    return leftRow.baseRoute.localeCompare(rightRow.baseRoute)
  })
}

export const listRecentRequestLogRows = async (dbName: string, limit = 200): Promise<IRequestLogRow[]> => {
  const normalizedLimit = Math.max(1, Math.min(500, Math.round(limit || 200)))

  try {
    // Reverse by monotonic id so the UI can inspect the newest requests first without scanning the full store.
    return await getDeltaCacheDatabase(dbName).requestLogs
      .orderBy('id')
      .reverse()
      .limit(normalizedLimit)
      .toArray()
  } catch (error) {
    console.warn('[DeltaCache] Failed to list recent request logs.', { dbName, normalizedLimit }, error)
    return []
  }
}
