import { unmarshall } from '$libs/funcs/unmarshall';
import {
  bulkDeleteRouteRecordRows,
  bulkGetRouteRecordRows,
  bulkPutRouteRecordRows,
  clearEnvironmentCache,
  clearModuleCache,
  ensureCacheRouteRow,
  extendRouteResponseKeys,
  getCacheRouteRow,
  getRecordRowsByResponseKey,
  listEnvironmentCacheStats,
  makeDeltaCacheDatabaseName,
  makeEmptyLastSync,
  readCachedRouteResponse,
  refreshRoutesByPrefix,
  replaceRouteRecordRows,
  resetCacheRouteRow,
  saveCacheRouteRow,
  setRouteForceNetwork,
  verifyRouteMemoryState,
} from './delta-cache.idb';
import type { CacheRecordID, ICacheRecordRow, ICacheRouteRow, IDeltaCacheRouteRef, ILastSync } from './delta-cache.types';
import type { serviceHttpProps } from '$libs/workers/service-worker';
import { parseObject } from '$libs/workers/service-worker-cache';

type CacheContent = { __version__?: number } & { [key: string]: any[] }

export interface ICacheSyncUpdate {
  args: serviceHttpProps
  response: any
  __enviroment__: string
  __companyID__?: number
}

export interface IGetCacheSubObject {
  route: string
  module: string
  __enviroment__?: string
  __companyID__?: number
  partValue?: string | number
  propInResponse?: string
  filter?: string
}

let forceFetch = false
let forcedFetchRequests: Set<string> = new Set()
const textEncoder = new TextEncoder()
const IDsToRemoveSuffix = "_IDsToRemove"

export const triggerDeltaForceFetchWindow = async () => {
  forceFetch = true
  forcedFetchRequests = new Set()
  setTimeout(() => { forceFetch = false }, 8000)
  return { ok: 1 }
}

const makeRouteReference = (args: serviceHttpProps): IDeltaCacheRouteRef => {
  // Database name scopes company + environment; row keys only need module + logical route + partition.
  const partitionValue = String(args.partition?.value || "0")
  const cacheKey = [args.route, partitionValue].join("_")
  const companyID = args.__companyID__ || 0
  const env = args.__enviroment__ || "main"
  const routeLookupKey = [args.module || "a", cacheKey].join("::")
  return {
    env,
    companyID,
    dbName: makeDeltaCacheDatabaseName(companyID, env),
    module: args.module || "a",
    route: args.route,
    partitionValue,
    cacheKey,
    routeLookupKey,
    version: args.__version__ || 1,
  }
}

const makeScopedDeltaDBName = (args: { __enviroment__?: string, __companyID__?: number }) => {
  return makeDeltaCacheDatabaseName(args.__companyID__ || 0, args.__enviroment__ || "main")
}

const makeCacheKey = (args: serviceHttpProps) => {
  return [args.route, args.partition?.value || "0"].join("_")
}

const normalizeResponse = (response: CacheContent | any[]): CacheContent => {
  if(Array.isArray(response)){ return { _default: response } }
  return response as CacheContent
}

const isRecordsToRemoveFlagKey = (responseKey: string) => {
  return responseKey.endsWith(IDsToRemoveSuffix)
}

const getTargetResponseKeyFromFlag = (responseKey: string) => {
  return responseKey.slice(0, responseKey.length - IDsToRemoveSuffix.length)
}

const isRecordResponseEntry = (entry: [string, unknown]): entry is [string, any[]] => {
  const [responseKey, records] = entry
  return responseKey !== "__version__" && !isRecordsToRemoveFlagKey(responseKey) && Array.isArray(records)
}

const listRecordResponseEntries = (response: CacheContent): [string, any[]][] => {
  return Object.entries(response).filter(isRecordResponseEntry)
}

const listIDsToRemoveByResponseKey = (response: CacheContent) => {
  const idsToRemoveByResponseKey = new Map<string, CacheRecordID[]>()

  for(const [responseKey, idsToRemove] of Object.entries(response)){
    if(!isRecordsToRemoveFlagKey(responseKey) || !Array.isArray(idsToRemove)){ continue }

    const targetResponseKey = getTargetResponseKeyFromFlag(responseKey)
    const validIDs = idsToRemove.filter((recordID): recordID is CacheRecordID => {
      return typeof recordID === 'string' || typeof recordID === 'number'
    })

    if(validIDs.length !== idsToRemove.length){
      console.warn(`Cache Error: En "${targetResponseKey}" se recibió un flag ${responseKey} con IDs inválidos`)
    }

    if(validIDs.length > 0){
      idsToRemoveByResponseKey.set(targetResponseKey, validIDs)
    }
  }

  return idsToRemoveByResponseKey
}

const addToRoute = (route: string, key: string, value: string | number) => {
  const sign = route.includes('?') ? "&" : "?"
  return route + `${sign}${key}=${String(value).replace("?","&")}`
}

const parseResponseAsStream = async (
  fetchResponse: Response,
  args: serviceHttpProps,
) => {
  const contentType = fetchResponse.headers.get("Content-Type")

  if (fetchResponse.status && args.status) {
    args.status.code = fetchResponse.status
    args.status.message = fetchResponse.statusText
  }

  if (fetchResponse.status === 200) {
    const reader = fetchResponse.body?.getReader() as ReadableStreamDefaultReader<Uint8Array<ArrayBuffer>>
    const stream = new ReadableStream({
      start(controller) {
        return pump()
        function pump(): Promise<void> {
          return reader.read().then(({ done, value }): Promise<void> => {
            if (done) {
              controller.close()
              return Promise.resolve()
            }
            controller.enqueue(value)
            return pump()
          })
        }
      },
    })

    const responseText = await new Response(stream).text()
    args.contentLength = textEncoder.encode(responseText).length
    return JSON.parse(responseText)
  }

  if (fetchResponse.status === 401) {
    console.warn('Error 401, la sesión ha expirado.')
    return undefined
  }

  if (!contentType || contentType.indexOf("/json") === -1) {
    return await fetchResponse.text()
  }

  return await fetchResponse.json()
}

const getRecordUpdateValue = (record: any): number => {
  return record?.upc || record?.upd || 0
}

const getRecordStatusValue = (record: any): number => {
  const rawStatus = record?.ss ?? 0
  if(typeof rawStatus === 'number'){ return rawStatus }
  const parsedStatus = parseInt(String(rawStatus || 0))
  return isNaN(parsedStatus) ? 0 : parsedStatus
}

const getRecordKeyFields = (args: serviceHttpProps, responseKey: string): string | string[] => {
  // Response-specific key config wins over route-wide key config, then fallback to `ID`.
  return args.keysIDs?.[responseKey] || args.keyID || "ID"
}

const getRequiredKeyPartValue = (
  record: any,
  keyField: string,
  route: string,
  responseKey: string,
): string | number => {
  const keyPartValue = record?.[keyField]
  if(keyPartValue === undefined || keyPartValue === null || keyPartValue === ""){
    throw new Error(`Cache Error: En "${route}" (${responseKey}) se recibió un registro sin key "${keyField}"`)
  }

  if(typeof keyPartValue === 'string' || typeof keyPartValue === 'number'){
    return keyPartValue
  }

  throw new Error(`Cache Error: En "${route}" (${responseKey}) la key "${keyField}" debe ser string o number`)
}

const getRequiredRecordID = (
  args: serviceHttpProps,
  record: any,
  route: string,
  responseKey: string,
): CacheRecordID => {
  const recordKeyFields = getRecordKeyFields(args, responseKey)
  if(!Array.isArray(recordKeyFields)){
    return getRequiredKeyPartValue(record, recordKeyFields, route, responseKey)
  }

  if(recordKeyFields.length === 0){
    throw new Error(`Cache Error: En "${route}" (${responseKey}) se configuró un keyID compuesto vacío`)
  }

  const keyParts = recordKeyFields.map((keyField) => {
    return getRequiredKeyPartValue(record, keyField, route, responseKey)
  })

  return `cmp:${JSON.stringify(keyParts)}`
}

const buildRecordRow = (
  cacheRouteId: number,
  responseKey: string,
  recordID: CacheRecordID,
  record: any,
): ICacheRecordRow => ({
  cR: cacheRouteId,
  rK: responseKey,
  ID: recordID,
  E: record,
  ss: getRecordStatusValue(record),
})

const buildRouteRowsFromResponse = (
  args: serviceHttpProps, routeRow: ICacheRouteRow, response: CacheContent,
) => {
  const rows: ICacheRecordRow[] = []
  const responseKeys = listRecordResponseEntries(response).map(([responseKey]) => responseKey)

  for(const [responseKey, records] of listRecordResponseEntries(response)){
    for(const record of records){
      rows.push(buildRecordRow(
        routeRow.id as number,
        responseKey,
        getRequiredRecordID(args, record, args.route, responseKey),
        record,
      ))
    }
  }

  return { rows, responseKeys }
}

const extractUpdated = (
  content: { [key: string]: any[] }, useMin?: boolean,
) => {
  const updatedStatus: { [key: string]: number } = {}

  for(const [key, values] of listRecordResponseEntries(content as CacheContent)){

    let maxOrMin = 0
    for(const record of values || []){
      const updated = getRecordUpdateValue(record)
      if(useMin){
        if(maxOrMin === 0 || updated < maxOrMin){ maxOrMin = updated }
      } else if(updated > maxOrMin){
        maxOrMin = updated
      }
    }

    updatedStatus[key] = maxOrMin
  }

  return updatedStatus
}

const countResponseRecords = (response: CacheContent) => {
  let recordsCount = 0
  for(const [, records] of listRecordResponseEntries(response)){
    recordsCount += records.length
  }
  return recordsCount
}

const measureResponseBytes = (response: CacheContent) => {
  // Approximate payload size from the serialized response when byte count was not captured in-stream.
  return textEncoder.encode(JSON.stringify(response)).length
}

const accumulateRouteFetchStats = (
  routeRow: ICacheRouteRow,
  response: CacheContent,
  payloadBytes?: number,
) => {
  // These counters are cumulative and intentionally do not deduplicate repeated IDs.
  routeRow.fetchedRecordsCount = (routeRow.fetchedRecordsCount || 0) + countResponseRecords(response)
  routeRow.fetchedBytes = (routeRow.fetchedBytes || 0) + (payloadBytes || measureResponseBytes(response))
}

const mergeColumnarRecord = (prevRecord: any, nextRecord: any, args: serviceHttpProps) => {
  const columnarIDField = args.columnarIDField || ""
  const combineFields = args.combineColumnarValuesOnFields || []
  const nextColumnarIDs = Array.isArray(nextRecord?.[columnarIDField]) ? nextRecord[columnarIDField] : []
  const prevColumnarIDs = Array.isArray(prevRecord?.[columnarIDField]) ? prevRecord[columnarIDField] : []
  const mergedRecord = { ...prevRecord, ...nextRecord }
  let missingCount = 0

  mergedRecord[columnarIDField] = [...prevColumnarIDs]
  for(const field of combineFields){
    mergedRecord[field] = Array.isArray(prevRecord?.[field]) ? [...prevRecord[field]] : []
  }

  const columnarIndexByID = new Map<string|number, number>()
  for(let index = 0; index < mergedRecord[columnarIDField].length; index++){
    const columnarID = mergedRecord[columnarIDField][index]
    if(columnarID !== undefined && columnarID !== null && columnarID !== ""){
      columnarIndexByID.set(columnarID, index)
    }
  }

  for(let nextIndex = 0; nextIndex < nextColumnarIDs.length; nextIndex++){
    const columnarID = nextColumnarIDs[nextIndex]
    if(columnarID === undefined || columnarID === null || columnarID === ""){
      missingCount++
      continue
    }

    const existingIndex = columnarIndexByID.get(columnarID)
    if(existingIndex === undefined){
      columnarIndexByID.set(columnarID, mergedRecord[columnarIDField].length)
      mergedRecord[columnarIDField].push(columnarID)
      for(const field of combineFields){
        const fieldValues = Array.isArray(nextRecord?.[field]) ? nextRecord[field] : []
        mergedRecord[field].push(fieldValues[nextIndex])
      }
      continue
    }

    for(const field of combineFields){
      const fieldValues = Array.isArray(nextRecord?.[field]) ? nextRecord[field] : []
      mergedRecord[field][existingIndex] = fieldValues[nextIndex]
    }
  }

  return { mergedRecord, missingCount }
}

const makeStats = (content: any) => {
  if(!content || Object.keys(content).length === 0){ return ["sin registros"] }

  const stats: string[] = []
  for(const [key] of listRecordResponseEntries(content as CacheContent)){
    stats.push(`${key}=${(content[key] || []).length}`)
  }
  return stats
}

const readCachedContent = async (routeRow?: ICacheRouteRow): Promise<CacheContent | undefined> => {
  if(!routeRow){ return undefined }
  return await readCachedRouteResponse(routeRow) as CacheContent | undefined
}

const getNextRouteURL = (args: serviceHttpProps, routeRow?: ICacheRouteRow) => {
  // Build the backend delta URL from the current cache watermarks.
  const lastSync = routeRow || makeEmptyLastSync(args.__version__ || 1)
  let route = args.routeParsed || args.route

  if(args.partition?.value){
    route = addToRoute(route, args.partition.param || args.partition.key, args.partition.value)
  }

  for(const field of args.fields || []){
    if(!lastSync.updatedStatus[field]){
      route = addToRoute(route, field, 0)
    }
  }

  if(!routeRow?.fetchTime){
    return { route, lastSync }
  }

  if(lastSync.updatedStatus._default){
    route = addToRoute(route, "updated", lastSync.updatedStatus._default as number)
    return { route, lastSync }
  }

  let minUpdated = 0
  const fields = args.fields || []
  for(const [key, updated] of Object.entries(lastSync.updatedStatus)){
    if(minUpdated === 0 || updated < minUpdated){ minUpdated = updated }
    if(fields.length > 0 && !fields.includes(key)){ continue }
    route = addToRoute(route, key, updated as number)
  }

  route = addToRoute(route, "updated", minUpdated)
  return { route, lastSync }
}

const shouldUseNetwork = async (
  args: serviceHttpProps,
  routeKey: string,
  routeRow: ICacheRouteRow,
  lastSync: ILastSync,
  fetchTime: number,
) => {
  const cacheSyncTime = args.cacheSyncTime || args.useCache?.min || 0
  const fetchNextTime = lastSync.fetchTime + (cacheSyncTime * 60)
  const remaining = fetchNextTime - fetchTime

  let doFetch = args.cacheMode === "refresh"
  if(routeRow.forceNetwork){
    console.log("Forzando fetch por flag forceNetwork:", args.route)
    routeRow.forceNetwork = false
    await saveCacheRouteRow(routeRow)
    return { doFetch: true, remaining }
  }

  if(forceFetch && !forcedFetchRequests.has(routeKey)){
    console.log("Forzando fetch por ventana global:", args.route)
    forcedFetchRequests.add(routeKey)
    return { doFetch: true, remaining }
  }

  if(remaining <= 0){
    console.log("Forzando fetch por expiración:", args.route, remaining)
    return { doFetch: true, remaining }
  }

  for(const field of args.fields || []){
    if(!lastSync.updatedStatus[field]){
      return { doFetch: true, remaining }
    }
  }

  return { doFetch, remaining }
}

const fetchNetworkResponse = async (args: serviceHttpProps, route: string) => {
  console.log(`Realizando fetch (${route})...`)
  const preResponse = await self.fetch(route, { headers: args.headers })

  if(preResponse.status && preResponse.status !== 200){
    throw new Error(await preResponse.text())
  }

  let response = ((await parseResponseAsStream(preResponse, args)) || {}) as CacheContent
  response = unmarshall(response)

  if(Array.isArray((response as any).response) && typeof (response as any).message === 'string'){
    response = (response as any).response as CacheContent
  }

  response = normalizeResponse(response)
  response.__version__ = args.__version__
  return response
}

const handleFetchResponse = async (
  args: serviceHttpProps,
  routeRow: ICacheRouteRow,
  response: CacheContent,
  nowTime: number,
) => {
  response = normalizeResponse(response)

  const updatedStatusDelta = extractUpdated(response)
  const updatedMinDelta = extractUpdated(response, true)
  const idsToRemoveByResponseKey = listIDsToRemoveByResponseKey(response)
  let hasChanged = [...idsToRemoveByResponseKey.values()].some((idsToRemove) => idsToRemove.length > 0)

  for(const [key, updated] of Object.entries(updatedStatusDelta)){
    if(!updated){ continue }
    const prevUpdated = routeRow.updatedStatus[key] || 0
    if(updated !== prevUpdated){
      routeRow.updatedStatus[key] = updated
      hasChanged = true
    }
    if(updatedMinDelta[key] && updatedMinDelta[key] < prevUpdated){
      console.warn(`Cache Error: En "${args.route}" [${key}] se están obteniendo registros con [updated] menor que el caché (${(response[key]||[]).length} recibidos)`)
    }
  }

  console.log("Fetch cache ha cambiado?:", hasChanged, "|", args.route, "|", updatedStatusDelta)

  if(!hasChanged && args.cacheMode !== 'updateOnly'){
    response = (await readCachedRouteResponse(routeRow)) as CacheContent
  } else if(hasChanged){
    const responseKeys = listRecordResponseEntries(response).map(([responseKey]) => responseKey)
    await extendRouteResponseKeys(routeRow, responseKeys)
    console.log("[DeltaCache] aplicando delta:", routeRow.cacheKey, responseKeys)

    const recordRowKeysToDelete: [number, string, CacheRecordID][] = []
    for(const [responseKey, idsToRemove] of idsToRemoveByResponseKey.entries()){
      for(const recordID of idsToRemove){
        // Flags ending in `_IDsToRemove` delete persisted rows before applying incoming deltas.
        recordRowKeysToDelete.push([routeRow.id as number, responseKey, recordID])
      }
    }

    await bulkDeleteRouteRecordRows(routeRow.dbName, recordRowKeysToDelete)

    const nextRecordRows: ICacheRecordRow[] = []
    const allowColumnarMerge = !!(args.columnarIDField && args.combineColumnarValuesOnFields?.length)
    for(const [responseKey, records] of listRecordResponseEntries(response as CacheContent)){
      if(records.length === 0){ continue }

      const rowKeys = records.map((record) => [
        routeRow.id as number,
        responseKey,
        getRequiredRecordID(args, record, args.route, responseKey),
      ] as [number, string, CacheRecordID])

      const prevRecordRows = new Map<CacheRecordID, ICacheRecordRow>()
      if(allowColumnarMerge){
        const currentRows = await bulkGetRouteRecordRows(routeRow.dbName, rowKeys)
        for(const currentRow of currentRows){
          if(currentRow){
            prevRecordRows.set(currentRow.ID, currentRow)
          }
        }
      }

      let missingCount = 0
      for(const record of records){
        const recordID = getRequiredRecordID(args, record, args.route, responseKey)
        const prevRecordRow = prevRecordRows.get(recordID)
        let nextRecord = record

        if(prevRecordRow && allowColumnarMerge){
          const merged = mergeColumnarRecord(prevRecordRow.E, record, args)
          nextRecord = merged.mergedRecord
          missingCount += merged.missingCount
        }

        nextRecordRows.push(buildRecordRow(
          routeRow.id as number,
          responseKey,
          recordID,
          nextRecord,
        ))
      }

      if(missingCount > 0){
        console.warn(`Cache Error: En "${args.route}" (${responseKey}) hay ${missingCount} registros sin key configurada`)
      }
    }

    if(nextRecordRows.length > 0){
      await bulkPutRouteRecordRows(routeRow.dbName, nextRecordRows)
    }

    response = (await readCachedRouteResponse(routeRow)) as CacheContent
  } else {
    response = null as unknown as CacheContent
  }

  accumulateRouteFetchStats(routeRow, normalizeResponse(response || {}), args.contentLength)
  routeRow.fetchTime = nowTime
  routeRow.__version__ = args.__version__ || routeRow.__version__
  await saveCacheRouteRow(routeRow)
  return response
}

const saveInitialSnapshot = async (
  args: serviceHttpProps,
  routeReference: IDeltaCacheRouteRef,
  response: CacheContent,
  fetchTime: number,
) => {
  // First sync writes the full route snapshot as indexed rows.
  const routeRow = await ensureCacheRouteRow(routeReference)
  routeRow.fetchTime = fetchTime
  routeRow.updatedStatus = extractUpdated(response)
  accumulateRouteFetchStats(routeRow, response, args.contentLength)
  routeRow.__version__ = args.__version__ || 1
  routeRow.forceNetwork = false

  const nextRouteRows = buildRouteRowsFromResponse(args, routeRow, response)
  await replaceRouteRecordRows(routeRow, nextRouteRows.responseKeys, nextRouteRows.rows)
  await saveCacheRouteRow(routeRow)
  return response
}

export const getDeltaUpdatedStatus = async (args: serviceHttpProps) => {
  const routeReference = makeRouteReference(args)
  const routeRow = await getCacheRouteRow(routeReference)
  const lastSync = routeRow || makeEmptyLastSync(routeReference.version)
  return { updatedStatus: lastSync.updatedStatus || {}, updated: lastSync.fetchTime || 0 }
}

export const fetchDeltaCache = async (args: serviceHttpProps) => {
  console.log("Obteniendo fetch service worker:", args.route, "|", args.cacheMode, "|", args.__req__, "|", args.__version__)

  const routeKey = makeCacheKey(args)
  const routeReference = makeRouteReference(args)
  let routeRow = await getCacheRouteRow(routeReference)

  if(routeRow?.fetchTime && routeRow.__version__ !== args.__version__){
    routeRow = await resetCacheRouteRow(routeRow, routeReference.version)
  }

  if(args.cacheMode === 'offline'){
    const content = await readCachedContent(routeRow)
    if(routeRow?.fetchTime && !content){
      console.warn(`[DeltaCache] Offline cache vacío, reseteando metadata: ${routeReference.cacheKey}`)
      routeRow = await resetCacheRouteRow(routeRow, routeReference.version)
    }
    console.log("Enviando fetch response (offline):", args.route)
    console.log(`${args.route}: Retornando registros "${args.cacheMode||"normal"}". ${makeStats(content).join(" | ")}`)
    return { content: content?._default ? content._default : content }
  }

  const fetchTime = Math.floor(Date.now()/1000)
  args.status = { code: 200, message: "" }

  try {
    let { route, lastSync } = getNextRouteURL(args, routeRow)
    const hasCache = !!(routeRow && routeRow.fetchTime)
    console.log("hasCache", args.route, lastSync)

    if(hasCache && routeRow){
      const networkDecision = await shouldUseNetwork(args, routeKey, routeRow, lastSync, fetchTime)
      if(!networkDecision.doFetch){
        console.log(`Obviando sync fetch "${routeKey}". Quedan ${networkDecision.remaining}s`)
        if(args.cacheMode === 'updateOnly'){
          console.log(args.route, "Retornando null por updateOnly")
          return { content: null }
        }

        const content = await readCachedContent(routeRow)
        if(!content){
          console.warn(`[DeltaCache] Cache inconsistente, faltan rows para ${routeReference.cacheKey}. Reintentando desde red.`)
          routeRow = await resetCacheRouteRow(routeRow, routeReference.version)
          const rebuiltRequest = getNextRouteURL(args, routeRow)
          route = rebuiltRequest.route
          lastSync = rebuiltRequest.lastSync
        } else {
          console.log(`${args.route}: Retornando registros "${args.cacheMode||"normal"}`, parseObject(content))
          return { content: content?._default ? content._default : content }
        }
      }
    }

    const response = await fetchNetworkResponse(args, route)
    console.log(`Fetch response recibida! (${route}) | Has-caché: ${hasCache}`)
    const content = routeRow && routeRow.fetchTime
      ? await handleFetchResponse(args, routeRow, response, fetchTime)
      : await saveInitialSnapshot(args, routeReference, response, fetchTime)

    console.log(`${args.route}: Retornando registros "${args.cacheMode||"normal"}". ${makeStats(content).join(" | ")}`)
    return { content }
  } catch (error) {
    console.log("Fetch Error::", error)
    return { error }
  } finally {
    // Defer storage verification so the next request can recover from manual tampering.
    void verifyRouteMemoryState(routeReference).catch((error) => {
      console.warn("[DeltaCache] Error verificando memoria vs IndexedDB:", error)
    })
  }
}

export const applyExternalDeltaResponse = async (args: ICacheSyncUpdate) => {
  const nowTime = Math.floor(Date.now()/1000) - 5
  args.args.__enviroment__ = args.__enviroment__
  args.args.__companyID__ = args.__companyID__
  console.log("Guardando external fetch response en caché:", args.args.route)
  args.args.contentLength = measureResponseBytes(normalizeResponse(args.response))
  const routeReference = makeRouteReference(args.args)
  const routeRow = await ensureCacheRouteRow(routeReference)
  return await handleFetchResponse(args.args, routeRow, normalizeResponse(args.response), nowTime)
}

export const setDeltaRouteForceNetwork = async (args: serviceHttpProps) => {
  const routeReference = makeRouteReference(args)
  const ok = await setRouteForceNetwork(routeReference, true)
  console.log(`ForceNetwork = true for key: ${routeReference.cacheKey}`)
  return { ok: ok ? 1 : 0 }
}

export const readDeltaCacheSubObject = async (args: IGetCacheSubObject) => {
  const routeReference = makeRouteReference({
    route: args.route,
    module: args.module,
    __enviroment__: args.__enviroment__,
    __companyID__: args.__companyID__,
    partition: { value: args.partValue },
  } as serviceHttpProps)

  const routeRow = await getCacheRouteRow(routeReference)
  if(!routeRow){ return [] }

  const responseKey = args.propInResponse || "_default"
  const records = (await getRecordRowsByResponseKey(routeRow, responseKey)).map((recordRow) => recordRow.E)
  if(!records.length && !(routeRow.responseKeys || []).includes(responseKey)){ return [] }

  if(!args.filter){
    return { [responseKey]: records }
  }

  const filterKeyValues = args.filter.split(",").filter(Boolean).map((value) => {
    const filter: (string | number)[] = value.split("=")
    filter.push(!isNaN(filter[1] as unknown as number) ? parseInt(filter[1] as string) : 0)
    return filter
  })

  const filtered: any[] = []
  for(const record of records){
    for(const filter of filterKeyValues){
      const value = record[filter[0]]
      if(value === filter[1] || value === filter[2]){
        filtered.push(record)
      }
    }
  }

  return filtered
}

export const getDeltaCacheStats = async (args: { __enviroment__: string, __companyID__?: number }) => {
  console.log("obteniendo cantidad de registros obtenidos")
  return { cacheStats: await listEnvironmentCacheStats(makeScopedDeltaDBName(args)) }
}

export const clearDeltaModuleCache = async (args: { __enviroment__: string, __companyID__?: number, cacheName: string }) => {
  console.log(`Eliminando caché "${args.cacheName}" (Enviroment ${args.__enviroment__})...`)
  const module = args.cacheName.includes("_")
    ? args.cacheName.split("_")[1] || ""
    : args.cacheName
  await clearModuleCache(makeScopedDeltaDBName(args), module)
  console.log(`Caché "${args.cacheName}" eliminado! (Enviroment ${args.__enviroment__})...`)
  return { ok: 1 }
}

export const clearDeltaEnvironmentCache = async (args: { __enviroment__: string, __companyID__?: number }) => {
  console.log("Eliminando caché...")
  await clearEnvironmentCache(makeScopedDeltaDBName(args))
  console.log("Caché eliminado.")
  return { ok: 1 }
}

export const refreshDeltaRoutes = async (args: { __enviroment__: string, __companyID__?: number, module: string, routes: string[] }) => {
  console.log("Setting ForceNetwork for routes:", args.routes)
  const routesUpdated = await refreshRoutesByPrefix(
    makeScopedDeltaDBName(args),
    args.module || "a",
    args.routes || []
  )
  console.log(`ForceNetwork = true for routes: ${args.routes.join(", ")} | In ${routesUpdated} routes`)
  return { ok: 1, routesUpdated }
}
