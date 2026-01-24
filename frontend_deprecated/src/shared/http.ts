import axios from 'axios';
import Dexie from "dexie";
import { Accessor, Setter, createSignal } from "solid-js";
import { Loading, Notify } from "~/core/main";
import { UriMerger, getRecordsFromIDB, httpProps, saveRecordsToIndexDB } from './httpHelpers';
import { formatN } from './main';
import { accessHelper, Env, getToken } from "./security";
import { IsClient, LocalStorage } from '~/env';

export const defaultCacheExp = 20 * 60
export const keyID = 'id'
export const keyUpdated = 'upd'

export const [fetchPending, setfetchPending] = createSignal(new Set() as Set<number>)
export const [fetchOnCourse, setfetchOnCourse] = createSignal(new Map() as Map<number,any>)

export function log1(m: string){ 
  console.log(`%c${m}`, 'background: #222; color: #bada55') }
export function log2(m: string){ 
  console.log(`%c${m}`, 'background: #222; color: #FFE357') }

let progressLastTime = 0
let progressTimeStart = 0
let progressBytes = 0
let fetchStreamOnCourse = 0

const setFetchProgress = (bytesLen?: number) => {
  const nowTime = Date.now()
  if(!progressBytes){ 
    progressTimeStart = nowTime  
  }

  progressLastTime = nowTime
  progressBytes += bytesLen

  let mbps = 0
  const kb = progressBytes/1000
  const elapsed = nowTime - progressTimeStart
  if(elapsed > 50){ mbps = kb / elapsed }
  
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
const parseResponseAsStream = async (fetchResponse: Response, props: IHttpStatus): Promise<any> => {

  if (fetchResponse.status) {
    props.code = fetchResponse.status
    props.message = fetchResponse.statusText
  }

  if (fetchResponse.status === 200) { 
    const reader = fetchResponse.body.getReader()
    const stream = new ReadableStream({
      start(controller) {
        fetchStreamOnCourse++
        function pump(): any {
          return reader.read().then(({ done, value }) => {
            // When no more data needs to be consumed, close the stream
            if (done) {
              controller.close()
              fetchStreamOnCourse--
              if(fetchStreamOnCourse <= 0){ progressBytes = 0 }
              return
            }
            console.log("chunk obtenido:: ", value.length)
            setFetchProgress(value.length)
            // Enqueue the next data chunk into our target stream
            controller.enqueue(value)
            return pump()
          })
        }
        return pump()
      },
    })
    const responseStream = new Response(stream)
    return responseStream.json()
  }
  else if (fetchResponse.status === 401) {
    accessHelper.clearAccesos()
    console.log("sesión expirada::", fetchResponse)
    throw("La sesión ha expirado, vuelva a iniciar sesión.")
  }
  else if (fetchResponse.status !== 200) {
    console.log(fetchResponse)
    let content = ""
    try {
      content = await fetchResponse.text()
    } catch (error) {
      throw(`Error status: ${fetchResponse.status}`)
    }

    throw(extractError(content))
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
    accessHelper.clearAccesos()
    console.warn('Error 401, la sesión ha expirado.')
    Notify.failure('La sesión ha expirado, vuelva a iniciar sesión.')
    Loading.remove()
  }
  else if (res.status !== 200) {
    if (!contentType || contentType.indexOf("/json") === -1) {
      return res.text()
    } else {
      return res.json()
    }
  }
}

const awaitDexieOpen = () => new Promise((resolve) => {
  if (window.DexieDB.isOpen()) resolve(true)
  else { window.DexieDB.open().then(() => resolve(true)) }
})

const checkIdbTables = (props: httpProps) => {
  if (!props.useIndexDBCache) return
  const idbTable = props.useIndexDBCache
  props.idbTableSchema = {}

  const table = window.DexieDB._allTables[idbTable]
  if (!table) {
    const err = `No se encontró la tabla en IndexedDB: ${idbTable}`
    console.warn(err)
    Notify.failure(`Err-1: ${err}`)
    return
  }

  const keyPath: string[] | string = table.schema.primKey.keyPath
  if (typeof keyPath === 'string') {
    props.idbTableSchema[keyPath] = -1
  } else {
    for(let key of keyPath) props.idbTableSchema[key] = -1
  }

  if(props.partition){
    props.idbTableSchema[props.partition.key] = props.partition.value
  }
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

// Parsea el body de la respuesta
function parseResponseBody(res: any, props: httpProps) {
  const handleCache = props.handleCache
  const db = window.DexieDB
  
  if (!res) { res = "Hubo un error desconocido en el servidor" }
  // Revisa si es un objeto
  else if (typeof res === 'string') { try { res = JSON.parse(res) } catch { } }
  // Revisa el Status Code
  if (!checkErrorResponse(res, props.status)) return false

  if (props.successMessage) Notify.success(props.successMessage)
  // Remueve el Caché de la IndexDB
  if (props.removeCacheGroup && db.isOpen()) {
    const group = props.removeCacheGroup
    db.table('cache')
      .where("group").equals(group).delete()
      .then(() => { log1('El grupo fue removido del caché: ' + group) })
  }
  // Agrega el caché a la indexDB
  else if (props.handleCache && db.isOpen()) {
    let cacheGroup = 'general'
    let exp = Date.now() + (defaultCacheExp * 1000)
    if (Array.isArray(handleCache)) {
      if (handleCache[1]) cacheGroup = handleCache[1]
      if (handleCache[2] && typeof handleCache[2] === 'number') {
        exp = (handleCache[2] * 1000)
      }
    }
    const obj = {
      path: props.route,
      group: cacheGroup,
      exp: exp,
      data: JSON.stringify(res)
    }
    db.table('cache').put(obj).then(() => {
      log1('Guardando en Caché DB: ' + props.route)
    })
  }
  return true
}

export const buildHeaders = (props: httpProps, contentType?: string) => {
  const cTs: {[e: string]: string } = { "json": "application/json" }

  const fromHeader = [
    location.pathname,
    (new Date()).getTimezoneOffset(),
    Env.language || "",
    [`${Env.screen.height}x${Env.screen.width}`,
    `ram:${Env.deviceMemory || 0}`,
    //`core:${navigator.hardwareConcurrency || 0}`,
    //window?.navigator['connection']?.effectiveType || "-"
    ].join("_"),
  ].join("|")

  if (props.headers) {
    props.headers['x-api-key'] = fromHeader
    return props.headers
  }
  const headers = new Headers()

  if (contentType && cTs[contentType]) {
    headers.append('Content-Type', cTs[contentType])
  }
  headers.append('Authorization', `Bearer ${getToken()}`)

  if (props.headersExtra) {
    for (let key in props.headersExtra) {
      headers.append(key, props.headersExtra[key])
    }
  }

  headers.append('x-api-key', fromHeader)
  return headers
}

// Busca en el caché de Index DB
const searchOnCache = (props: httpProps) => {
  console.log('buscando en caché indexeddb', props.route)
  const db = window.DexieDB as Dexie

  return new Promise(resolve => {
    if (!db.isOpen()) resolve(null)

    db.table('cache')
    .get({ path: props.route })
    .then(record => {
      if (!record || !record.data) {
        // Resuelve null si no encuentra el caché
        resolve(null)
      } else if (Date.now() > record.exp) {
        // Lo elimina del caché si pasó la fecha de expiración
        db.table('cache').delete(props.route)
        console.log('Eliminando caché (expiró): ' + props.route)
        resolve(null)
      } else {
        // Devuelve el caché
        const len = (record.data.length / 1000).toFixed(2)
        log2(`Enviando desde Caché DB: ${props.route} - ${len}Kb`)
        resolve(JSON.parse(record.data))
      }
    })
  })
}

export const makeRoute = (route: string, apiName?: string) => {
  const apiUrl = apiName ? Env.API_ROUTES[apiName] : Env.API_ROUTES.MAIN
  return route.includes('://') ? route : apiUrl + route
}

const updateFechOnCourse = (props: httpProps, event: 1 | 2) => {
  // 1 = inicio, 2 = fin
  if(event === 1){
    fetchOnCourse().set(props.id,{})
    setfetchOnCourse(new Map(fetchOnCourse()))
  } else if(event === 2){
    fetchOnCourse().delete(props.id)
    setfetchOnCourse(new Map(fetchOnCourse()))
  }
}

const parseResults = (props: httpProps, records: any[]) => {
  //debugger
  if(!Array.isArray(records)){ return records }

  if(!props.collections || Object.keys(props.collections).length === 0){
    return { Records: records }
  } else {
    const map1: Map<number,any[]> = new Map()
    const keyToIdx: Map<number,string> = new Map()
    for(let [k,v] of Object.entries(props.collections)){ 
      map1.set(v,[]) 
      keyToIdx.set(v,k)
    }
    for(let e of records){ map1.get(e._pk)?.push(e) }
    const result = {} as any
    for(let [k,records] of map1){
      result[keyToIdx.get(k)] = records
    }
    return result
  }
}

export function GET(props: httpProps): Promise<any> {
  if(!IsClient()){ return Promise.resolve([]) }

  if(props.mergeRequest && !Env.pendingRequests.includes(props)){ 
    Env.pendingRequests.push(props) 
  }
  console.log('agregando merge:::',[...Env.pendingRequests])

  return new Promise((resolve, reject) => {
    props.status = { code: 200, message: "" }
    props.resolver = resolve
    props.rejecter = reject
    const route = makeRoute(props.route, props.apiName)
    
    // Función Fetch
    const makeFetch = () => {
      updateFechOnCourse(props,1)
      console.log("realizando fetch::", props)
      fetch(route, { headers: buildHeaders(props) })
        .then(res => parseResponseAsStream(res, props.status))
        .then(res => {
          updateFechOnCourse(props,2)
          return parseResponseBody(res, props) ? resolve(res) : reject(res)
        })
        .catch(error => {
          updateFechOnCourse(props,2)
          console.warn(error)
          if (props.errorMessage) { Notify.failure(props.errorMessage) }
          reject(error)
        })
    }
    // Revisa si el contenido puede ser obtenido desde la IndexDB
    if (props.useIndexDBCache) {
      awaitDexieOpen()
        .then(() => searchOnIndexDB(props))
        .then(result => {
          // debugger
          if(props.readyForFetch === 1){ return }
          return result ? resolve(parseResults(props, result)) : makeFetch() 
        })
        .catch(error => reject(error))
    }
    // Revisa si el contenido puede encontrarse en el caché
    else if (props.handleCache) {
      searchOnCache(props)
        .then(result => { result ? resolve(result) : makeFetch() })
    }
    else {
      if (props.mergeRequest) {
        props.readyForFetch = 1
        setTimeout(() => fetchPendingRequests(), 0)
      } else {
        makeFetch()
      }
    }
  })
}

export type GetSignal<T> = [Accessor<T>, Setter<T>]
export type IResult = { Records?: any[] } | { [e: string]: any[] }

export const makeApiGetHandler = <T>(
  props_: httpProps | ((args?: any) => httpProps),
  parseResult?: ((result: IResult, args?: any) => T)
): (GetSignal<T>) => {
  
  const props = typeof props_ === 'function' ? props_() : props_

  if(!props.id){
    props.id = Env.params.fetchID
    Env.params.fetchID++
  }

  const [fetchedRecords, setFetchedRecords] = createSignal(props.emptyValue as T)

  props.cacheMode = props.cacheMode || 'offline'
  
  fetchPending().add(props.id)
  
  const removePending = () => {
    const pending = fetchPending()
    if(pending.size === 0){ return }
    pending.delete(props.id)
    
    console.log("pending...",[...pending])
    if(pending.size === 0){
      setfetchPending(new Set([]))
    }
  }
  // console.log("Realizando GET...",props)
  // Primer GET donde obtiene desde IndexedDB
  GET(props)
  .then((result: T) => {
    // debugger
    props.resultCached = result
    if(parseResult){ result = parseResult(result as Exclude<T,unknown>) }
    setFetchedRecords(result as Exclude<T,unknown>)
    if(props.recordUpdated > 0){ removePending() }
    // Setea el caché mode para realizar un fetch al servidor
    props.cacheMode = 'updateOnly'
    return GET(props)
  })
  .then((result: T) => {
    // console.log("segundo resultado::", result,props)
    if(result && !props.noUpdateReceived){
      if(parseResult){ result = parseResult(result as Exclude<T,unknown>) }
      setFetchedRecords(result as Exclude<T,unknown>)  
    }
    removePending()
  })
  .catch(error => {
    console.warn(error)
    removePending()
  })

  return [fetchedRecords, setFetchedRecords]
}

// ReadyForFetch: 1 = Consultar al servidor, 2 = Se obtuvo desde IndexedDB, no enviar
async function fetchPendingRequests() {
  const pending: httpProps[] = Env.pendingRequests
  if (pending.length === 0) return
  if (pending.some(x => !x.readyForFetch)) return
  console.log("requests to fetch:: ",[...Env.pendingRequests])

  const requests = pending.filter(x => x.readyForFetch === 1)
  Env.pendingRequests = []
  if (requests.length === 0) {
    // Si no hay ningún request a realizar (porque han sido enviados desde IndexDB probablemente)
    return
  }
  const apiReqMap: Map<string, httpProps[]> = new Map()
  for (let req of requests) {
    req.apiName = req.apiName || 'MAIN'
    apiReqMap.has(req.apiName)
      ? apiReqMap.get(req.apiName).push(req)
      : apiReqMap.set(req.apiName, [req])
  }
  // realiza los requests según el api a enviar
  apiReqMap.forEach(apiRequests => {
    doFetchPendingRequests(apiRequests)
  })
}

interface IHttpStatus { code: number, message: string }

interface IMergeResult {
  statusCode: number
  body: string | any
  id: number
  message: string
  route: string
}

const doFetchPendingRequests = async (requests: httpProps[]) => {
  // Realiza los requests  
  console.log('Request a Realizar:::', requests)
  // Crea un query unificado en el GET
  const routes: string[] = requests.map(x => x.route)
  const mergedUri = UriMerger.mergeRoutesInQuery(routes)
  const status = { code: 200, message: "" }
  const cache = Env.cache
  updateFechOnCourse(requests[0],1)

  // Crea la URL y envía el request
  const url = makeRoute('merged?' + mergedUri, requests[0].apiName)
  let results: IMergeResult[]
  let error

  try {
    const re1 = await fetch(url, { headers: buildHeaders(requests[0]) })
    results = await parseResponseAsStream(re1, status)
  } catch (error_) {
    error = error_
  }
  
  updateFechOnCourse(requests[0],2)

  // Si hay un error en la respuesta general, recchaza todas las promesas
  if(error){
    console.warn(error)
    Notify.failure(error as string)
    for (let e of requests) { e.rejecter && e.rejecter(null) }
    Loading.remove()
    return
  }

  results = (results||[]).sort((a,b) => a.id - b.id)

  // Revisa si hay Errores Individuales
  for (let i = 0; i < requests.length; i++) {
    const props = requests[i]
    const routeBase = props.route.split('?')[0]
    let result = results[i]
    if (!result) {
      Notify.failure(`No se encontró el resultado del request a [${routeBase}]`)
      continue
    }
    if (!result.statusCode || result.statusCode !== 200) {
      let message: any = ''
      if (typeof result === 'string') { 
        message = result 
      } else if (result.message) {
        message = result.message
      } else if (result.body) {
        message = result.body
        try {
          message = JSON.parse(message)
          message = message.errorMessage || message.message || message.body
        } catch (_) { }
      }
      if (typeof message !== 'string') {
        message = JSON.stringify(message)
      }
      if (result.statusCode === 401) {
        Notify.warning(`Su sesión ha expirado, vuelva a iniciar sesión. Msg: ${message}`)
        window.dispatchEvent(new Event('userLogout'))
      } else {
        const err = `Hubo un error en la consulta a [${props.route}] ${message}`
        console.warn(err)
        Notify.failure(err)
      }
      // Resuelve por defecto un array vacío
      props.resolver(parseResults(props,[]))
      continue
    }

    // Si todo está ok
    if (props.useIndexDBCache) {
      // console.log('Buscando en IndexDB::', props.useIndexDBCache, result)
      try {
        const re1 = await saveRecordsToIndexDB(props, result.body)
        // Si no hay registro a actualizar y el modo es "offline"
        if (re1 === 0) { props.resolver(parseResults(props,[])); continue }
        // Obtiene los registros de la IndexedBD
        const results = await getRecordsFromIDB(props)
        for (let table of (props.useIndexDBCache || [])){ cache[table] = 1 }
        props.resolver(parseResults(props, results))
      } catch (error) {
        console.warn('Error en IndexDB Merged::', error)
        props.rejecter && props.rejecter(error)
      }
    }
    else {
      props.resolver(result.body || result.message || result)
    }
  }
}

const searchOnIndexDB = async (props: httpProps): Promise<any[]> => {
  const db = window.DexieDB
  const idbTable = props.useIndexDBCache

  checkIdbTables(props)
  const cache = Env.cache

  // Limpia el flag para user el caché
  if (props.clearCache){ delete cache[idbTable] }
  const par = props.partition
  if (par && par.key && !par.value) {
    const err = `La partición ${par.key} no posee un valor para la ruta: ${props.route}. Tabla: ${idbTable}`
    Notify.failure(err)
    throw(err)
  }

  props.localRecordsToDelete = []
  let forceSync = false
  // Revisa si el caché está desactivado por un numero de segundos
  const forceUntil = parseInt(LocalStorage.getItem("force_sync_cache_until") || "0")
  const nowUnixTime = Math.floor((Date.now() / 1000))

  if (nowUnixTime < forceUntil) { forceSync = true }
  else if(props.cacheMode !== 'offline') {
    const partValuesToCheck = ['0']
    if (props.partition) { partValuesToCheck.push(String(props.partition.value)) }
    // Revisa si hay que forzar la sincronización de esa tabla
    for (let partValue of partValuesToCheck) {
      const key1 = 'force_sync_cache_' + idbTable + '_' + partValue
      if (LocalStorage.getItem(key1)) {
        console.log('Forzando Sincronizacion::', key1, idbTable)
        forceSync = true
        props.localRecordsToDelete.push(key1)
      }
    }
  }

  // Busca en la IndexDB
  if (!db.isOpen()) return []
  // Obtiene el objeto donde se guarda la última actualizacion
  // props.[keyUpdated]atedQuery = { [keyID]: -1 }
  // if(mergeKey) props.[keyUpdated]atedQuery[mergeKey] = -1
  const baseObject = {...(props.idbTableSchema||{})}
  // Si hay un valor de particionado
  if (par && par.key && par.value && (!par.include || par.include.includes(idbTable))){
    if (baseObject[par.key]) {
      baseObject[par.key] = par.value
    } else {
      const err = `La clave de particionado [${par.key}] no se encontró en el esquema de la Tabla [${idbTable}]`
      Notify.failure(err)
      console.warn(err)
      return []
    }
  }

  const record = await db.table(idbTable).get(baseObject)
  console.log("record obtenido::", record, props.useIndexDBCache)
  // console.log('Buscando si se debe actualizar::',searchUpdatePromises)
  // console.log('Buscando en Table indexed-db : 1',props.route)      
  // Revisa cuando fue la última ves que se sincronizó
  props.recordUpdated = record ? record[keyUpdated] : 0
  props.collections = record?.collections || {}
  
  // console.log('objeto flag de actualizacion::',record,baseObject)
  props.startTimeMs = Date.now()
  props.startTime = Math.floor(props.startTimeMs / 1000)
  const refreshTime = props.startTime - (60 * 60 * 2)
  let sync = false

  if (cache[idbTable] !== 1 || props.recordUpdated < refreshTime) sync = true
  if (forceSync || props.cacheSyncTime === 0) {
    log1(`Forzando sincronización [${idbTable}]`)
    sync = true
  }
  else if (props.cacheSyncTime) {
    const syncTime = props.cacheSyncTime * 60
    // console.log('calculo de sync:',`${(props.startTime - props.recordUpdated)}`,syncTime)
    if ((props.startTime - props.recordUpdated) < syncTime) {
      sync = false
      const falta = ((props.recordUpdated + syncTime - props.startTime) / 60)
      console.log(`Evitando sincronización [${idbTable}] faltan ${falta.toFixed(2)}min`)
    }
  }
  if (!sync || props.cacheMode === 'offline') {
    if (props.cacheMode === 'updateOnly') { props.noUpdateReceived = true }
    if (props.mergeRequest && props.cacheMode !== 'offline'){ 
      props.readyForFetch = 2
      fetchPendingRequests() 
    }
    
    let result
    try {
      result = await getRecordsFromIDB(props)
    } catch (error) {
      console.warn("Error consultando en IndexedDB.", error)
      return []
    }
    console.log("retornando resultado::", result)
    return result || []
  }

  // Complementa la ruta a consultar
  if (props.route.includes('?')) props.route += `&${keyUpdated}=`
  else props.route += `?${keyUpdated}=`
  
  props.route += String((props.recordUpdated - 20))

  // Agrega el parámetro de la partición al query
  if (par && par.param && par.value) props.route += `&${par.param}=${par.value}`

  if (props.mergeRequest) { // Si se van a juntar los requests
    props.readyForFetch = 1; fetchPendingRequests()
    return []
  }
  props.routeQuery = makeRoute(props.route, props.apiName)

  // Realiza el fetch al servidor
  updateFechOnCourse(props,1)

  try {
    const preResult = await fetch(props.routeQuery || '', { headers: buildHeaders(props) })
    const results = await parsePreResponse(preResult, props.status)
    updateFechOnCourse(props,2)
    if (!checkErrorResponse(results, props.status)) {
      throw(results)
    } else {
      await saveRecordsToIndexDB(props, results)
      if (props.noUpdateReceived && props.cacheMode === 'updateOnly') {
        log2(`Enviando repuesta vacía. No hay registros nuevos IndexedDB Table: ${idbTable}`)
        return []
      } else {
        const resultsGetted = await getRecordsFromIDB(props)
        cache[idbTable] = 1
        return resultsGetted
      }
    }
  } catch (error) {
    updateFechOnCourse(props,2)
    console.warn('Error en IndexDB', error)
    throw(error)
  }
}

interface IForceRefresh {
  refreshIndexDBCache?: string | string[]
  refreshPartition?: (number | string) | (number | string)[]
}

export const setForceRefresh = (props: IForceRefresh) => {
  if (!props.refreshIndexDBCache) return
  if (typeof props.refreshIndexDBCache === 'string') {
    props.refreshIndexDBCache = [props.refreshIndexDBCache]
  }
  const partitions = Array.isArray(props.refreshPartition)
    ? [...props.refreshPartition]
    : [props.refreshPartition || 0]

  if (!partitions.includes(0)) partitions.push(0)

  for (let table of props.refreshIndexDBCache) {
    for (let part of partitions) {
      LocalStorage.setItem(`force_sync_cache_` + table + '_' + part, '1')
    }
  }
}

const POST_PUT = (props: httpProps, method: string): Promise<any> => {
  const data = props.data
  if (typeof data !== 'object') {
    const err = 'The data provided is not a JSON'
    console.error(err)
    return Promise.reject(err)
  }
  props.status = { code: 200, message: "" }
  const apiRoute = makeRoute(props.route, props.apiName)
  setForceRefresh(props)

  return new Promise((resolve, reject) => {
    log1(`Fetching ${method} : ` + props.route)
    updateFechOnCourse(props,1)

    fetch(apiRoute, {
      method: method,
      headers: buildHeaders(props, 'json'),
      body: JSON.stringify(data)
    })
      .then(res => parsePreResponse(res, props.status))
      .then(res => {
        updateFechOnCourse(props,2)
        parseResponseBody(res, props) ? resolve(res) : reject(res)
      })
      .catch(error => {
        console.log('error::', error)
        updateFechOnCourse(props,2)
        if (props.errorMessage) {
          Notify.failure(props.errorMessage)
        } else {
          Notify.failure(String(error))
        }
        reject(error)
      })
  })
}

export function PUT(props: httpProps) {
  return POST_PUT(props, 'PUT')
}

export function POST(props: httpProps) {
  return POST_PUT(props, 'POST')
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
  setForceRefresh(props)
  const apiRoute = makeRoute(props.route, props.apiName)

  return new Promise((resolve, reject) => {
    axios.post(apiRoute, data, {
      onUploadProgress: props.onUploadProgress,
      headers: {
        'authorization': `Bearer ${getToken()}`
      },
    })
    .then(result => {
      const data = result.data
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

