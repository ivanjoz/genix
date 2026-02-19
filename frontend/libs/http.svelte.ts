import { normalizeStringN, Notify } from '$libs/helpers';
import axios, { type AxiosProgressEvent } from 'axios';
import { formatN } from '$libs/helpers';
import type { CacheMode, serviceHttpProps } from '$libs/workers/service-worker';
import { accessHelper, getToken } from '$core/security';
import { fetchCache, fetchCacheParsed, sendServiceMessage } from '$libs/sw-cache';
import { browser } from "$app/environment";
import { unmarshall } from '$libs/funcs/unmarshall';
import { Env } from '$core/env';
import {  ConcatenateIntsTest } from "./funcs/parsers"

ConcatenateIntsTest()

export interface IHttpStatus { code: number, message: string }

export interface httpProps {
  data?: any
  route: string
  apiName?: string
  headers?: {[key: string]: string}
  successMessage?: string
  errorMessage?: string
  module?: string
  onUploadProgress?: (e: AxiosProgressEvent) => void
  status?: IHttpStatus
  refreshRoutes?: string[]
  keysIDs?: { [e: string]: string | string[] }
  keyID?: string | string[]
  cacheMode?: CacheMode
  useCache?: {
    min: number, /* minutos del caché */
    ver: number  /* versión del caché */
  },
  useCacheStatic?: {
    min: number, /* minutos del caché */
    ver: number  /* versión del caché */
  },
}

export const buildHeaders = (contentType?: string) => {
  const cTs: {[e: string]: string } = { "json": "application/json" }
  /*
  const fromHeader = [
    location.pathname,
    (new Date()).getTimezoneOffset(),
    Env.language || "",
    [`${Env.screen.height}x${Env.screen.width}`,
    `ram:${Env.deviceMemory || 0}`,
    ].join("_"),
  ].join("|")
  */
  const headers = {} as {[k:string]: string}

  if (contentType && cTs[contentType]) {
    headers['Content-Type'] = cTs[contentType]
  }
  headers['Authorization'] = `Bearer ${getToken()}`
  // headers.append('x-api-key', fromHeader)
  return headers
}

const extractError = (result: any): string => {
  let errorJson
  let errorString = ""

  if(typeof result === 'string'){
    errorString = result.trim()
    if(errorString[0] === "{" || errorString[0] === "["){
      try {
        errorJson = JSON.parse(errorString)
      } catch {}
    }
  } else {
    errorJson = result
  }
  if(errorJson){
    if(Array.isArray(errorJson)){
      errorJson = errorJson[0]
    }
    if(errorJson.message || errorJson.error || errorJson.errorMessage){
      errorJson = errorJson.message || errorJson.error || errorJson.errorMessage
    }
    errorString = typeof errorJson === 'string'
      ?  errorJson
      : JSON.stringify(errorJson)
  }
  return errorString
}

const checkErrorResponse = (result: any, status: IHttpStatus) => {
  if (!status.code || status.code !== 200 || result.errorMessage) {
    console.warn(result)
    Notify.failure(extractError(result))
    return false
  } else {
    return true
  }
}

// Parsea los headers de la respuesta antes parseado el body
const parsePreResponse = (res: any, status: IHttpStatus): Promise<any> => {
  const contentType = res.headers.get("content-type")
  if (res.status) {
    status.code = res.status
    status.message = res.statusText
  }
  if (res.status === 200) { return res.json() }
  else if (res.status === 401) {
    accessHelper.clearAccesos?.()
    console.warn('Error 401, la sesión ha expirado.')
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
  }
  else if (res.status !== 200) {
    if (!contentType || contentType.indexOf("/json") === -1) {
      return res.text()
    } else {
      return res.json()
    }
	}
	return Promise.resolve()
}

// Parsea el body de la respuesta
function parseResponseBody(res: any, props: httpProps, status: IHttpStatus) {
  if (!res) { res = "Hubo un error desconocido en el servidor" }
  // Revisa si es un objeto
  else if (typeof res === 'string') { try { res = JSON.parse(res) } catch { } }

  // Revisa el Status Code
  if (!checkErrorResponse(res, status)) return false

  if (props.successMessage) Notify.success(props.successMessage)
  return true
}

const POST_PUT = (props: httpProps, method: string): Promise<any> => {
  const data = props.data
  if (typeof data !== 'object') {
    const err = 'The data provided is not a JSON'
    console.error(err)
    return Promise.reject(err)
  }

  const apiRoute = Env.makeRoute(props.route)

  if((props.refreshRoutes||[]).length > 0){
    sendServiceMessage(24, { routes: props.refreshRoutes })
	}
	const status: IHttpStatus = { code: 200, message: "" }

  return new Promise((resolve, reject) => {
    console.log(`Fetching ${method} : ` + props.route)

    fetch(apiRoute, {
      method: method,
      headers: buildHeaders('json'),
      body: JSON.stringify(data)
    })
    .then(res => parsePreResponse(res, status))
    .then(res => {
      res = unmarshall(res)
      parseResponseBody(res, props, status) ? resolve(res) : reject(res)
    })
    .catch(error => {
      console.log('error::', error)
      if (props.errorMessage) {
        Notify.failure(props.errorMessage)
      } else {
        Notify.failure(String(error))
      }
      reject(error)
    })
  })
}

export function POST(props: httpProps) {
  return POST_PUT(props, 'POST')
}

export function PUT(props: httpProps) {
  return POST_PUT(props, 'PUT')
}

// Crea una solicitud HTTP request
export const POST_XMLHR = (props: httpProps): Promise<any> => {
  const data = props.data
  if (typeof data !== 'object') {
    const err = 'The data provided is not a JSON'
    console.error(err)
    return Promise.reject(err)
  }
  props.status = { code: 200, message: "" }
const apiRoute = Env.makeRoute(props.route)

  return new Promise((resolve, reject) => {
    axios.post(apiRoute, data, {
      onUploadProgress: props.onUploadProgress,
      headers: { 'authorization': `Bearer ${getToken()}` },
    })
    .then(result => {
      const data = unmarshall(result.data)
      if (result.status !== 200) {
        let message = data.message || data.error || data.errorMessage
        if (!message) message = String(data)
        Notify.failure(data)
        reject(data)
      } else {
        resolve(data)
      }
    })
    .catch(error => {
      if (error.response && error.response.data) error = error.response.data
      const message = error.message || error.error || error.errorMessage
      Notify.failure(String(message || error))
      reject(error)
    })
  })
}

let progressLastTime = 0
let progressTimeStart = 0
let progressBytes = 0
let fetchOnCourse = 0

export const setFetchProgress = (bytesLen: number) => {
  const nowTime = Date.now()
  if(!progressBytes){
    progressTimeStart = nowTime
  }

  progressLastTime = nowTime
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
    if(!nextElement){
      nextElement = document.createElement("div")
      nextElement.setAttribute("id","NotifyProgressMessage")
      loadingMsgDiv.parentNode.insertBefore(nextElement, loadingMsgDiv.nextSibling)
    }
    nextElement.innerHTML = msg
  }
}


// Parsea los headers de la respuesta crear un reader
const parseResponseAsStream = async (fetchResponse: Response, status: any, props?: httpProps) => {

  const contentType = fetchResponse.headers.get("Content-Type")

  if (fetchResponse.status) {
    status.code = fetchResponse.status
    status.message = fetchResponse.statusText
  }

  if (fetchResponse.status === 200) {
    const reader = fetchResponse.body.getReader()
    const stream = new ReadableStream({
      start(controller) {
        fetchOnCourse++
        return pump()
        function pump() {
          return reader.read().then(({ done, value }) => {
            // When no more data needs to be consumed, close the stream
            if (done) {
              controller.close()
              fetchOnCourse--
              if(fetchOnCourse <= 0){ progressBytes = 0 }
              return
            }
            // console.log("chunk obtenido:: ", value.length)
            setFetchProgress(value.length)
            // Enqueue the next data chunk into our target stream
            controller.enqueue(value)
            return pump()
          })
        }
      },
    })
    const responseStream = new Response(stream)
    const responseText = await responseStream.text()
    if(props){ props.contentLength = responseText.length }
    return Promise.resolve(JSON.parse(responseText))
  }
  else if (fetchResponse.status === 401) {
    document.dispatchEvent(new Event('userLogout'))
    console.warn('Error 401, la sesión ha expirado.')
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
    Notify.remove()
  }
  else if (fetchResponse.status !== 200) {
    console.log(fetchResponse)
    if (!contentType || contentType.indexOf("/json") === -1) {
      console.log('Parseando como texto')
      return fetchResponse.text()
    } else {
      console.log('parseando como JSON')
      return fetchResponse.json()
    }
  }
}

export const GET = (props: httpProps): Promise<any> => {
  const status: IHttpStatus = { code: 200, message: "" }
  const routeParsed = Env.makeRoute(props.route)

  if(props.useCache){
    const args = {
      routeParsed,
      route: props.route,
      useCache: props.useCache,
      module: props.module || "a",
      headers: buildHeaders('json'),
      cacheMode: props.cacheMode,
    } as serviceHttpProps

    return new Promise((resolve, reject) => {
      fetchCacheParsed(args)
      .then(cachedResponse => {
        resolve(cachedResponse)
      })
      .catch(err => { reject(err) })
    })
  } else {
    return new Promise((resolve, reject) => {
      console.log("realizando fetch::", props)
      fetch(routeParsed, { headers: buildHeaders() })
      .then(res => parsePreResponse(res, status))
      .then(res => {
        res = unmarshall(res)
        return parseResponseBody(res, props, status) ? resolve(res) : reject(res)
      })
      .catch(error => {
        console.warn(error)
        if (props.errorMessage) { Notify.failure(props.errorMessage) }
        reject(error)
      })
    })
  }
}

export interface INewIDToID {
  NewID: number;
  TempID: number;
}

export class GetHandler<T extends { ID: number, ss?: number } = any> {

  route = ""
  routeParsed = ""
  module = "a"
  keyID: string | string[] = ""
  keysIDs: { [e: string]: string | string[] } = {}

  useCache: { min: number, ver: number  } | undefined = undefined
  headers: { [k: string]: string } | undefined = undefined

	handler(e: any) { }
  isReady = $state(0)

  makeProps(cacheMode?: CacheMode): serviceHttpProps {
    const props = {
      routeParsed: Env.makeRoute(this.route),
      route: this.route,
      useCache: this.useCache,
      module: this.module,
      headers: buildHeaders('json'),
      cacheMode,
      keyID: this.keyID,
      keysIDs: this.keysIDs,
    } as serviceHttpProps
    return props
  }

	fetchOnline() {
  	if(!browser){ return }
    if(this.route.length === 0){
      Notify.failure("No se especificó el route en productos.")
      return
		}
    
  	fetchCacheParsed(this.makeProps('refresh'))
    .then(cachedResponse => {
      if(cachedResponse){
        delete cachedResponse.__version__
        this.handler(cachedResponse)
      }
      this.isReady++
      return fetchCacheParsed(this.makeProps())
    })
	}
  
  fetch(){
    if(!browser){ return }
    if(this.route.length === 0){
      Notify.failure("No se especificó el route en productos.")
      return
    }

    fetchCacheParsed(this.makeProps('offline'))
    .then(cachedResponse => {
      if(cachedResponse){
        delete cachedResponse.__version__
        this.handler(cachedResponse)
			}
      this.isReady++
      return fetchCacheParsed(this.makeProps())
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
		
		const ids = records.map(x => x.ID)
		const recordsWithoutCurrent = this.records.filter((existingRecord) => !ids.includes(existingRecord.ID))
		
		this.records = this.prependOnSave
			? [...recordsToKeep, ...recordsWithoutCurrent]
			: [...recordsWithoutCurrent, ...recordsToKeep]
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
		const name = this.makeName(record)
		return name ? this.nameToRecordMap.get(normalizeStringN(name)) as T : undefined
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
			if (!mapping || mapping.NewID <= 0 || mapping.TempID === 0) { continue }
			tempToNewIDs.set(mapping.TempID, mapping.NewID)
			this.tempToNewID.set(mapping.TempID, mapping.NewID)

			for (const record of records) {
				if (record.ID !== mapping.TempID) { continue }
				if (mapping.TempID < 0) { this.recordsMap.delete(mapping.TempID) }
				record.ID = mapping.NewID
				this.onTempRecordSynced(record, mapping.TempID, mapping.NewID)
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
