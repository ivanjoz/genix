import { browser } from "$app/environment";
import { Env } from '$core/env';
import { fetchEvent } from '$core/store.svelte';
import { unmarshall } from '@genix/ui/utilities';
import { formatN, normalizeStringN, Notify } from '$libs/helpers';
import { fetchCacheParsed, sendServiceMessage } from '@genix/ui/service-worker';
import {
  createHttpClient,
  type httpProps,
} from '@genix/ui/http';
import { concatenateInts } from '@genix/ui/utilities';
import {
  makeGroupCacheKey,
  makeGroupQueryShape,
  readGroupCacheMetadata,
  readGroupCacheRows,
  upsertGroupCacheRows,
  type IGroupCacheRecord,
} from '@genix/ui/cache';
import { getRecordsByID, type IMinimalRecord } from '@genix/ui/cache';
import type {
  CacheConversions,
  CacheMode,
  serviceHttpProps,
} from '@genix/ui/cache';

export type {
  AxiosProgressEvent,
  IHttpStatus,
  httpProps,
} from '@genix/ui/http';

const genixHttpClient = createHttpClient({
  makeRoute: Env.makeRoute,
  getToken: Env.getToken,
  transformResponse: unmarshall,
  notify: Notify,
  onUnauthorized: () => {
    Env.clearAccesos?.();
    if (browser) {
      document.dispatchEvent(new Event('userLogout'));
    }
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.');
  },
  startRequest: (route) => {
    if (!browser) { return 0; }
    const requestId = fetchEvent(0, 0) as number;
    if (requestId > 0) {
      fetchEvent(requestId, { url: route });
    }
    return requestId;
  },
  finishRequest: (requestId) => {
    fetchEvent(requestId, 0);
  },
  fetchCached: fetchCacheParsed,
  refreshRoutes: (routes) => sendServiceMessage(24, { routes }),
  verifyRouteMemoryState: () => Env.DELTA_CACHE_VERIFY_ROUTE_MEMORY,
});

export const {
  buildHeaders,
  GET,
  POST,
  PUT,
  POST_XMLHR,
} = genixHttpClient;

let progressTimeStart = 0
let progressBytes = 0

export const setFetchProgress = (bytesLen: number) => {
  const nowTime = Date.now()
  if(!progressBytes){
    progressTimeStart = nowTime
  }

  progressBytes += bytesLen

  let mbps = 0
  const kb = progressBytes/1000
  const elapsed = nowTime - progressTimeStart

  if(elapsed > 50){
    mbps = kb / elapsed
  }

  let msg = `Descargando... ${formatN(kb)} kb`
  if(mbps){
    if(mbps > 10){ mbps = 10 }
    msg += ` (${formatN(mbps,2)} MB/s)`
  }

  const loadingMsgDiv = document.getElementById("NotiflixLoadingMessage")
  if(loadingMsgDiv){
    let nextElement = loadingMsgDiv.nextElementSibling
    if(!nextElement && loadingMsgDiv.parentNode){
      nextElement = document.createElement("div")
      nextElement.setAttribute("id","NotifyProgressMessage")
      loadingMsgDiv.parentNode.insertBefore(nextElement, loadingMsgDiv.nextSibling)
    }
    if(nextElement){
      nextElement.innerHTML = msg
    }
  }
}


const normalizeGroupCacheResponse = <T>(responsePayload: any): IGroupCacheRecord<T>[] => {
  if(Array.isArray(responsePayload)){ return responsePayload as IGroupCacheRecord<T>[] }
  if(Array.isArray(responsePayload?.records)){ return responsePayload.records as IGroupCacheRecord<T>[] }
  if(Array.isArray(responsePayload?.response)){ return responsePayload.response as IGroupCacheRecord<T>[] }
  return []
}

const makeGroupCacheRoute = (route: string, uriParams: {[k: string]: string}) => {
  const requestParams = new URLSearchParams()
  for(const paramName of Object.keys(uriParams).sort()){
    requestParams.set(paramName, uriParams[paramName])
  }
  const queryString = requestParams.toString()
  return queryString ? `${route}?${queryString}` : route
}

export const GETWithGroupCache = async <T = any>(
  route: string,
  uriParams: {[k: string]: string},
): Promise<IGroupCacheRecord<T>[]> => {
  if(!browser){
    return normalizeGroupCacheResponse<T>(await GET({ route: makeGroupCacheRoute(route, uriParams) }))
  }

  // The database name already scopes company and API endpoint, so the row key only needs the route shape.
  const queryShape = makeGroupQueryShape(route, Object.keys(uriParams))
  const cachedMetadataRows = await readGroupCacheMetadata(queryShape)
  const requestParams = new URLSearchParams()
  for(const paramName of Object.keys(uriParams).sort()){
    requestParams.set(paramName, uriParams[paramName])
  }

  if(cachedMetadataRows.length > 0){
    // Group ids stay signed in IndexedDB and are converted to uint32 only for compact transport.
    requestParams.set("cc-gh", concatenateInts(cachedMetadataRows.map((row) => row.id >>> 0)))
    requestParams.set("cc-upc", concatenateInts(cachedMetadataRows.map((row) => row.upc)))
  }

  const routeWithCacheParams = requestParams.toString() ? `${route}?${requestParams.toString()}` : route
  console.debug("[GETWithGroupCache] Fetching grouped route.", {
    route,
    queryShape,
    cachedGroups: cachedMetadataRows.length,
  })

  const responseGroups = normalizeGroupCacheResponse<T>(await GET({ route: routeWithCacheParams }))
  const responseKeys = responseGroups.map(makeGroupCacheKey).filter(Boolean)
  const cachedRowsByKey = await readGroupCacheRows<T>(queryShape, responseKeys)
  const rowsToPersist: IGroupCacheRecord<T>[] = []
  const mergedGroups: IGroupCacheRecord<T>[] = []

  for(const responseGroup of responseGroups){
    const responseRecords = Array.isArray(responseGroup.records) ? responseGroup.records : []

    // ig === -1 marks direct-lookup results (e.g. SerialNumber/LotID/DocumentID) that do not belong
    // to any grouped index, so they must pass through untouched instead of being dropped or cached.
    if(responseGroup.ig === -1){
      mergedGroups.push({ ...responseGroup, records: responseRecords })
      continue
    }

    const key = makeGroupCacheKey(responseGroup)
    if(!key){
      console.warn("[GETWithGroupCache] Ignoring grouped response without igVal.", responseGroup)
      continue
    }

    const cachedRow = cachedRowsByKey.get(key)
    const canUseCachedRecords = cachedRow && cachedRow.upc === responseGroup.upc && responseRecords.length === 0

    if(canUseCachedRecords){
      mergedGroups.push({
        ig: responseGroup.ig,
        id: responseGroup.id,
        igVal: responseGroup.igVal,
        records: cachedRow.records || [],
        upc: responseGroup.upc,
      })
      continue
    }

    const freshGroup = { ...responseGroup, records: responseRecords }
    rowsToPersist.push(freshGroup)
    mergedGroups.push(freshGroup)
  }

  await upsertGroupCacheRows(queryShape, rowsToPersist)
  console.debug("[GETWithGroupCache] Grouped cache merged.", {
    route,
    queryShape,
    responseGroups: responseGroups.length,
    persistedGroups: rowsToPersist.length,
    mergedGroups: mergedGroups.length,
  })

  return mergedGroups
}

export interface INewIDToID {
  ID: number;
  TempID: number;
}

export class GetHandler<T extends { ID: number, ss?: number } = any> {

  route = ""
	routeParsed = ""
  routeByID = ""
  module = "a"
  keyID: string | string[] = ""
	keysIDs: { [e: string]: string | string[] } = {}
	columnarIDField = ""
  combineColumnarValuesOnFields: string[] = []

  useCache: { min: number, ver: number  } | undefined = undefined
	headers: { [k: string]: string } | undefined = undefined
	conversion: CacheConversions | undefined = undefined
  // CDN snapshot bootstrap (optional): absolute URL to a .db file + its per-section column schema.
  // When set, the first sync seeds the cache from the file instead of the full API list.
  fileRoute = ""
  fileSchema: { [section: string]: string[] } | undefined = undefined


	handler(e: any) { }
  isReady = $state(0)

  makeProps(cacheMode?: CacheMode): serviceHttpProps {
    // Forward cache merge metadata so the service worker can apply
    // columnar delta updates without each service reimplementing it.
    const props = {
      routeParsed: Env.makeRoute(this.route),
      route: this.route,
      useCache: this.useCache,
      module: this.module,
      headers: buildHeaders('json'),
      cacheMode,
      // The verification pass is useful for debugging corruption, not for the default mobile path.
      verifyRouteMemoryState: Env.DELTA_CACHE_VERIFY_ROUTE_MEMORY,
      keyID: this.keyID,
      keysIDs: this.keysIDs,
      columnarIDField: this.columnarIDField,
      combineColumnarValuesOnFields: this.combineColumnarValuesOnFields,
      conversion: this.conversion,
      fileRoute: this.fileRoute || undefined,
      fileSchema: this.fileSchema,
    } as serviceHttpProps
    return props
  }

	async fetchOnline() {
  	if(!browser){ return }
    if(this.route.length === 0){
      Notify.failure("No se especificó el route en productos.")
      return
		}

		if (!Env.canUserAccessRoute(Env.getPathname())) {
   		console.error(`Servicio "${this.route}" no cargado debido a acceso.`)
      return
    }
    
		const cachedResponse = await fetchCacheParsed(this.makeProps('refresh'))
	  if(cachedResponse){
      delete cachedResponse.__version__
      this.handler(cachedResponse)
		}
		this.isReady++
	}

	// Like fetchOnline but honors the useCache.min TTL (normal mode): returns cached
	// content while fresh and only delta-syncs once it expires. Awaitable; calls handler
	// once. Writes that set forceNetwork (POST refreshRoutes) still bypass the TTL.
	async fetchCached() {
		if(!browser){ return }
		if(this.route.length === 0){
			Notify.failure("No se especificó el route.")
			return
		}

		if (!Env.canUserAccessRoute(Env.getPathname())) {
			console.error(`Servicio "${this.route}" no cargado debido a acceso.`)
			return
		}

		const response = await fetchCacheParsed(this.makeProps())
		if(response){
			delete response.__version__
			this.handler(response)
		}
		this.isReady++
	}

	async syncIDs(ids: number[]) {
		if (!this.routeByID) {
			Notify.failure("[syncIDs] Missing routeByID in: " + this.route)
			return
		}

		// Resolve only valid, not-yet-loaded IDs so we keep the merge minimal and deterministic.
		const missingIDs = [...new Set(
			ids.filter((recordID) => recordID > 0 && !this.recordsMap.has(recordID))
		)]
		console.debug(`[GetHandler] syncIDs:start route=${this.route} byID=${this.routeByID} requested=${ids.length} missing=${missingIDs.length}`)
		if (missingIDs.length === 0) {
			console.debug(`[GetHandler] syncIDs:skip route=${this.route} reason=no-missing-ids`)
			return
		}

		console.debug("[GetHandler] syncIDs fetching missing records:", this.route, {
			routeByID: this.routeByID,
			requestedIDs: ids.length,
			missingIDs,
		})

		try {
			const fetchedRecordsByID = await getRecordsByID<T & IMinimalRecord>(this.routeByID, missingIDs)
			const fetchedRecords = [...fetchedRecordsByID.values()]
			console.debug(`[GetHandler] syncIDs:fetched route=${this.route} byID=${this.routeByID} fetched=${fetchedRecords.length}`)
			if (fetchedRecords.length === 0) {
				console.debug(`[GetHandler] syncIDs:empty route=${this.route} byID=${this.routeByID}`)
				return
			}

			// Reuse the standard merge path so `records`, `recordsMap`, and name indexes stay aligned.
			this.addSavedRecords(...fetchedRecords)
			console.debug(`[GetHandler] syncIDs:end route=${this.route} byID=${this.routeByID} merged=${fetchedRecords.length}`)
		} catch (syncIDsError) {
			console.error(`[GetHandler] syncIDs:error route=${this.route} byID=${this.routeByID}`)
			console.error(syncIDsError)
			throw syncIDsError
		}
	}
  
  fetch(){
    if(!browser){ return }
    if(this.route.length === 0){
      Notify.failure("No se especificó el route en productos.")
      return
    }

    if (!Env.canUserAccessRoute(Env.getPathname())) {
   		console.error(`Servicio "${this.route}" no cargado debido a acceso.`)
      return
    }

    fetchCacheParsed(this.makeProps('offline'))
    .then(cachedResponse => {
      if(cachedResponse){
        delete cachedResponse.__version__
        this.handler(cachedResponse)
			}
      this.isReady++
      // Delta-only online sync: return null when server has no updates, so handler is not called twice.
      return fetchCacheParsed(this.makeProps('updateOnly'))
    })
    .then(fetchedResponse => {
      if(fetchedResponse){
        delete fetchedResponse.__version__
				this.handler(fetchedResponse)
        this.isReady++
      }
    })
	}
  
	// Post Method
	routePost = ""
	refreshRoutes: string[] = []
	loadingMessage = "Enviando Registros..."
	tempToNewID: Map<number, number> = new Map()	
	nextTempID = -1
	recordsMap: Map<number, T> = $state(new Map())
	nameToRecordMap: Map<string,T> = new Map()
	records: T[] = $state([])
	prependOnSave?: boolean
	inferRemoveFromStatus?: boolean
	
	makeName(record: Partial<T>){ return "" }
	onTempRecordAdded(_record: T) {}
	onTempRecordSynced(_record: T, _tempID: number, _newID: number) {}
	afterSaveRecords(...records: T[]) {}

	addSavedRecords(...records: T[]) {
		console.log("addSavedRecords", this.route, records.length)
		
		const recordsToKeep: T[] = []
		for (const rec of records) {
			// Always keep the ID lookup map updated, even for ss=0 records.
			this.recordsMap.set(rec.ID, rec)

			const shouldRemoveByStatus = this.inferRemoveFromStatus && rec.ss === 0
			const normalizedRecordName = normalizeStringN(this.makeName(rec) || "")
			if (shouldRemoveByStatus) {
				if (normalizedRecordName) {
					this.nameToRecordMap.delete(normalizedRecordName)
				}
				// Also clear any stale normalized-name keys pointing to the same record.
				for (const [normalizedNameKey, indexedRecord] of this.nameToRecordMap.entries()) {
					if (indexedRecord.ID === rec.ID) {
						this.nameToRecordMap.delete(normalizedNameKey)
					}
				}
				continue
			}

			if (normalizedRecordName) {
				this.nameToRecordMap.set(normalizedRecordName, rec)
			}
			recordsToKeep.push(rec)
		}

		const currentIDs = new Set(this.records.map((existingRecord) => existingRecord.ID))
		const incomingIDs = new Set(records.map((incomingRecord) => incomingRecord.ID))
		const incomingRecordByID = new Map(recordsToKeep.map((incomingRecord) => [incomingRecord.ID, incomingRecord]))

		// Preserve order of existing rows; replace in-place when an updated version exists.
		const updatedExistingRecords: T[] = []
		for (const existingRecord of this.records) {
			if (!incomingIDs.has(existingRecord.ID)) {
				updatedExistingRecords.push(existingRecord)
				continue
			}
			const updatedRecord = incomingRecordByID.get(existingRecord.ID)
			// Skip if the incoming record is filtered out (e.g. ss=0 with inferRemoveFromStatus).
			if (updatedRecord) { updatedExistingRecords.push(updatedRecord) }
		}

		// Append/prepend only brand-new records; never reorder existing ones.
		const newRecords = recordsToKeep.filter((incomingRecord) => !currentIDs.has(incomingRecord.ID))
		this.records = this.prependOnSave
			? [...newRecords, ...updatedExistingRecords]
			: [...updatedExistingRecords, ...newRecords]
	}

	setTempID(record: T): number {
		if (!record.ID) { record.ID = this.nextTempID-- }
		if(record.ID <= 0){ this.tempToNewID.set(record.ID, 0) }
		return record.ID
	}
	
	addTempRecord(record: T) {
		const existingRecord = this.getByName(record)
		if (existingRecord) {
			record.ID = existingRecord.ID
			return existingRecord
		}

		if (record.ID > 0) { return record }

		// Use negative IDs in-memory so the UI can reference unsaved records safely.
		this.setTempID(record)
		if (!record.ss) { record.ss = 1 }

		this.recordsMap.set(record.ID, record)
		
		const name = this.makeName(record)
		if (name) {
			this.nameToRecordMap.set(normalizeStringN(name), record)
		}
		this.onTempRecordAdded(record)
		console.log("[GetHandler] temp record created:", this.route, record.ID)
		return record
	}
	
 	get(id: number) {
	   return this.recordsMap.get(id);
	}
	
	getByName(record: Partial<T>): T | undefined {
		const name = normalizeStringN(this.makeName(record))
		// console.log("Comparando nombre::", name, [...this.nameToRecordMap.keys()])
		return name ? this.nameToRecordMap.get(name) as T : undefined
	}
	
	getTempRecords(): T[] {
		const pendingRecords: T[] = []
		for (const record of this.recordsMap.values()) {
			if (record.ID < 0) { pendingRecords.push(record) }
		}
		return pendingRecords
	}

	clearTempRecords(tempIDs?: Set<number>) {
    for (const [recordID] of this.recordsMap.entries()) {
      if (recordID >= 0) continue;
			if (tempIDs && !tempIDs.has(recordID)) { continue }
      this.recordsMap.delete(recordID);
    }
		for (const [normalizedName, record] of this.nameToRecordMap.entries()) {
			if (record.ID >= 0) { continue }
			if (tempIDs && !tempIDs.has(record.ID)) { continue }
			this.nameToRecordMap.delete(normalizedName)
		}
  }
	
	async post(records: T[], reqWrapper?: { [e: string]: any }): Promise<INewIDToID[]> {
		if (reqWrapper) { reqWrapper.records = records }
		const data = reqWrapper ? reqWrapper : records
		const routeToPost = this.routePost || this.route
		
		let response: INewIDToID[] = []
		try {
		  response = await POST({
		    data,
		    route: routeToPost,
		    refreshRoutes: [this.route].concat(this.refreshRoutes||[])
		  })			
		} catch(err) {
			console.log("[GetHandler] Error al hacer POST:", err)
			return []
		}

		return response
	}

	async postAndSync(records: T[], reqWrapper?: { [e: string]: any }): Promise<Map<number, number>> {
		for (const record of records) { this.setTempID(record) }

		const idMappings = await this.post(records, reqWrapper)
		const tempToNewIDs = new Map<number, number>()

		for (const mapping of idMappings) {
			if (!mapping || mapping.ID <= 0 || mapping.TempID === 0) { continue }
			tempToNewIDs.set(mapping.TempID, mapping.ID)
			this.tempToNewID.set(mapping.TempID, mapping.ID)

			for (const record of records) {
				if (record.ID !== mapping.TempID) { continue }
				if (mapping.TempID < 0) { this.recordsMap.delete(mapping.TempID) }
				record.ID = mapping.ID
				this.onTempRecordSynced(record, mapping.TempID, mapping.ID)
			}
		}

		this.addSavedRecords(...records)
		this.afterSaveRecords(...records)
		return tempToNewIDs
	}

	async syncTempRecords(reqWrapper?: { [e: string]: any }): Promise<Map<number, number>> {
		const pendingRecords = this.getTempRecords()
		if (pendingRecords.length === 0) { return new Map() }

		console.log("[GetHandler] syncing temp records:", this.route, pendingRecords.length)
		const tempToNewIDs = await this.postAndSync(pendingRecords, reqWrapper)

		// Remove only synced temp entries so unsynced rows are preserved for retries.
		const syncedTempIDs = new Set<number>([...tempToNewIDs.keys()].filter((tempID) => tempID < 0))
		this.clearTempRecords(syncedTempIDs)
		console.log("[GetHandler] temp sync completed:", this.route, tempToNewIDs.size)
		return tempToNewIDs
	}
}
