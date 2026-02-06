import { unmarshall } from '$libs/funcs/unmarshall';
import { CACHE_APP, getCacheRecord, HandlersMap, hasCacheKey, parseObject, sendClientMessage, setCacheRecord } from "./service-worker-cache"
import { connectAppSync, sendSignalToAppSync, disconnectAppSync, type AppSyncWebRTCConfig } from "./service-worker-webrtc"

export type CacheMode = 'offline' | 'updateOnly' | 'refresh' | 'fetchOnly'

export type serviceHttpProps = {
  __enviroment__: string
  __accion__: number
  __client__: number
  __req__?: number
  __version__?: number /* version del caché */
  route: string
  module?: string
  routeParsed?: string
  headers?: { [e: string]: string } | Headers
  keyID?: string | string[]
  keysIDs?: { [e: string]: string | string[] }
  fields?: string[]
  keyFilterIfEmpty?: string
  keyForUpdated?: string
  cacheMode?: CacheMode
  contentLength?: number
  partition?: { 
    key: string, value: string | number, param?: string
  }
  status?: { code: number, message: string }
  updatedStatus?: { [e: string]: string }
  cacheSyncTime?: number
  useCache?: { 
    min: number, /* minutos del caché */
    ver: number  /* versión del caché */
  },
  useCacheStatic?: { 
    min: number, /* minutos del caché */
    ver: number  /* versión del caché */
  },
}

type CacheContent = { __version__?: number } & { [k: string]: any[] }

// Parsea los headers de la respuesta crear un reader
const parseResponseAsStream = async (
  fetchResponse: Response, props: serviceHttpProps
) => {

  const contentType = fetchResponse.headers.get("Content-Type")

  if (fetchResponse.status && props.status) {
    props.status.code = fetchResponse.status
    props.status.message = fetchResponse.statusText
  }

  if (fetchResponse.status === 200) { 
    const reader = fetchResponse.body?.getReader() as ReadableStreamDefaultReader<Uint8Array<ArrayBuffer>>
    const stream = new ReadableStream({
      start(controller) {
        // fetchOnCourse++
        return pump()
        function pump(): Promise<void> {
          return reader.read().then(({ done, value }): Promise<void> => {
            // When no more data needs to be consumed, close the stream
            if (done) {
              controller.close()
              return Promise.resolve()
            }
            sendClientMessage(props.__client__, { __response__: 5, bytes: value.length })
            // Enqueue the next data chunk into our target stream
            controller.enqueue(value)
            return pump()
          })
        }
      },
    })
    const responseStream = new Response(stream)
    const responseText = await responseStream.text()
    // console.log('responseText::', responseText)
    if(props){ props.contentLength = responseText.length }
    return Promise.resolve(JSON.parse(responseText))
  }
  else if (fetchResponse.status === 401) {
    // document.dispatchEvent(new Event('userLogout'))
    console.warn('Error 401, la sesión ha expirado.')
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

const extractUpdated = (
  obj: {[k: string]: any[]}, useMin?: boolean
) => {
  const updatedStatus: {[k: string]: number } = {}

  for(const [key, values] of Object.entries(obj)){
    if(values && !Array.isArray(values)){
      continue
    }
    let maxOrMin = 0
    for(const v of values||[]){
      const updated = v.updated || v.upd || 0
      if(useMin){
        if(maxOrMin === 0 || updated < maxOrMin){ maxOrMin = updated }
      } else {
        if(updated > maxOrMin){ maxOrMin = updated }
      }
    }
    updatedStatus[key] = maxOrMin
  }
  return updatedStatus
}

const addToRoute = (route: string, key: string, value: (string|number)) => {
  const sign = route.includes('?') ? "&" : "?"
  if(typeof value === 'string'){
    value = value.replace("?","&")
  }
  return route + `${sign}${key}=${value}`
}

// fecha para forzar el cache
let forceFetch = false
let forcedFetchRequests: Set<string> = new Set()

HandlersMap.set(11, async ()=> {
  forceFetch = true
  forcedFetchRequests = new Set()
  setTimeout(() => { forceFetch = false },8000)
  return { ok: 1 }
})

HandlersMap.set(3, async (args: serviceHttpProps) => {
  return await fetchCache(args)
})

// WebRTC Handlers
HandlersMap.set(30, async (args: AppSyncWebRTCConfig) => {
  return await connectAppSync(args)
})

HandlersMap.set(31, async (args: { __client__: number; action: string; data: string }) => {
  return await sendSignalToAppSync(args)
})

HandlersMap.set(32, async (args: { __client__: number }) => {
  return await disconnectAppSync(args)
})

const makeKey = (args: serviceHttpProps): [string, string] => {
  const key = [args.route, args.partition?.value||"0"].join("_")
  const cacheName = [args.__enviroment__||"main", args.module||"a"].join("_")
  return [key, cacheName]
}

// Obtiene los updated
HandlersMap.set(12, async (args: serviceHttpProps)=> {
  const [key, cacheName] = makeKey(args)
  const lastSync: ILastSync = await getCacheRecord(key+"_updated", cacheName)
  console.log("lastSync obtenido (1):",args, args.route, key, lastSync)
  const updatedStatus = lastSync?.updatedStatus || {}
  return { updatedStatus, updated: lastSync?.fetchTime || 0  }
})

interface ILastSync {
  fetchTime: number
  updatedStatus: {[k: string]: number }
  forceNetwork?: boolean
  __version__: number
}

const getRecordKeys = (args: serviceHttpProps, field: string): string |string[] => {
  const keyIDs = args.keysIDs && Object.hasOwn(args.keysIDs, field)
    ? args.keysIDs[field]
    : args.keyID || "id"
  return keyIDs
}

const getKeyValue = (record: any, keyIDs: string |string[]): (string|number) => {
  if(typeof keyIDs === 'string'){
    return record[keyIDs] || record.ID || 0
  } else if(Array.isArray(keyIDs)){
    const arr: (string|number)[] = []
    for(const key of keyIDs){
      if(record[key]){
        arr.push(record[key])
      } else {
        return 0
      }
    }
    return arr.join("_")
  } else {
    return 0
  }
}

export interface IResponse { [k: string]: any[] }

const handleFetchResponse = async (
  args: serviceHttpProps, lastSync: ILastSync, response: CacheContent, nowTime: number
): Promise<any> => {

  if(Array.isArray(response)){ response = { _default: response } }

  const updatedStatusDelta = extractUpdated(response)
  const updatedMinDelta = extractUpdated(response, true)

  let hasChanged = false
  for(const [key, updated] of Object.entries(updatedStatusDelta)){
    if(!updated){ continue }
    const prevUpdated = lastSync.updatedStatus[key]
    if(updated !== prevUpdated){
      lastSync.updatedStatus[key] = updated
      hasChanged = true
    }
    if(updatedMinDelta[key] < prevUpdated){
      console.warn(`Cache Error: En "${args.route}" [${key}] se están obteniendo registros con [updated] menor que el caché (${(response[key]||[]).length} recibidos)`)
    }
  }

  console.log("Fetch cache ha cambiado?:", hasChanged,"|",args.route,"|", updatedStatusDelta)
  const [key, cacheName] = makeKey(args)
  const keyUpdated = key+"_updated"
  
  if(!hasChanged && args.cacheMode !== 'updateOnly'){
    response = (await getCacheRecord(key, cacheName)) as { [k: string]: any[] }
  } else if(hasChanged){
    const prevResponse = (await getCacheRecord(key, cacheName)) as IResponse
    console.log("cache obtenido:",key,cacheName, args)

    if(self._isLocal){
      console.log("prevResponse (old)", args.route, args.cacheMode, {...prevResponse})
    }

    const usedKeysCount: {[k: string]: number } = {}

    for(const [respKey, newRecords] of Object.entries(response as IResponse)){
      if((newRecords||[]).length === 0){ continue }

      const keyIDs = getRecordKeys(args,respKey)

      let missingCount = 0

      usedKeysCount[String(keyIDs)] = (usedKeysCount[String(keyIDs)]||0) + newRecords.length
      // Combina los registros basados en el ID
      // ***
      const makeKeyID = (r: any) => {
        const value = getKeyValue(r, keyIDs)
        if(!value){ missingCount++ }
        return value
      }
      
      const prevRecords = prevResponse[respKey] || []
      if(!Array.isArray(prevRecords)){ continue }

      const recordsMap: Map<number|string,any> = new Map()

      for(const e of prevRecords){ recordsMap.set(makeKeyID(e), e) }
      for(const e of newRecords){ recordsMap.set(makeKeyID(e), e) }

      if(missingCount > 0){
        console.warn(`Cache Error: En "${args.route}" (${respKey}) hay ${missingCount} registros sin la key: ${keyIDs}`)
      }
      console.log("Cache usedKeysCount", usedKeysCount)

      const mergedRecords: any[] = []
      for(const e of recordsMap.values()){
        // if(args.keyFilterIfEmpty && !e[args.keyFilterIfEmpty]){ continue }
        mergedRecords.push(e)
      }

      prevResponse[respKey] = mergedRecords
    }

    if(self._isLocal){
      console.log("prevResponse (new)", args.route, args.cacheMode, {...prevResponse})
    }
    setCacheRecord(key, prevResponse, cacheName)
    response = prevResponse
  } else {
    response = null as unknown as CacheContent
  }

  lastSync.fetchTime = nowTime
  setCacheRecord(keyUpdated, lastSync, cacheName)
  return response
}

// Confirmación del client que ha obtenido la respuesta enviada
const acknowledgeResponses: Set<number> = new Set()

HandlersMap.set(21, async (args: serviceHttpProps)=> {
  const reqID = (args.__req__||0) * 1000 + args.__client__
  acknowledgeResponses.add(reqID)
})

const fetchCache = async(args: serviceHttpProps) => {
  console.log("Obteniendo fetch service worker:",args.route,"|", args.cacheMode,"|",args.__req__,"|", args.__version__)

  const [key, cacheName] = makeKey(args)
  const keyUpdated = key+"_updated"

  const makeStats = (content: any): string[] => {
    if(!content || Object.keys(content).length === 0){ return ["sin registros"] }
    // Revisa la cantidad de registros por key
    const stats: string[] = []
    for(const key of Object.keys(content)){
      stats.push(`${key}=${(content[key]||[]).length}`)
    }
    return stats
  }

  const lastSyncEmpty = { 
    fetchTime: 0, updatedStatus: {}, __version__: args.__version__ 
  } as ILastSync

  let lastSync: ILastSync = (await getCacheRecord(keyUpdated, cacheName)) || lastSyncEmpty

  // Si la version ha cambiado, limpia el caché
  if(lastSync.fetchTime && lastSync.__version__ !== args.__version__){
    console.log(`Linmpiando caché, difererente versión ${args.__version__} > ${lastSync.__version__}. Route ${args.route}`)
    lastSync = lastSyncEmpty
    await setCacheRecord(keyUpdated, lastSync, cacheName)
    await setCacheRecord(key, null, cacheName)
  } else {
    const isCachePresent = await hasCacheKey(key, cacheName)

    // Limpia el lastfech si no se encontró el registro base
    if(lastSync.fetchTime && !isCachePresent){
      await setCacheRecord(keyUpdated, lastSyncEmpty, cacheName)
      lastSync = lastSyncEmpty
    // Si se encontró el registro base pero no el lastSync, entonces limpia el registro base
    } else if(isCachePresent && !lastSync.fetchTime){
      await setCacheRecord(key, null, cacheName)
    }
  }

  if(args.cacheMode === 'offline'){
    const content = await getCacheRecord(key, cacheName) as CacheContent
    if(content && content.__version__ === args.__version__){
      const updatedStatus = extractUpdated(content)

      // Revisa si son diferentes
      for(const [key, updated] of Object.entries(updatedStatus)){
        let reSave = false
        if(lastSync.updatedStatus[key] !== updated){
          console.log("El updatedStatus difiere:", args.route,"|",key,"|",lastSync.updatedStatus[key]," vs ", updated)
          lastSync.updatedStatus[key] = updated
          reSave = true
        }
        if(reSave){
          setCacheRecord(keyUpdated, lastSync, cacheName)
        }
      }
    }

    console.log("Enviando fetch response (offline):", args.route)
    console.log(`${args.route}: Retornando registros "${args.cacheMode||"normal"}". ${makeStats(content).join(" | ")}`)

    return { content: content?._default ? content._default : content  }
  }

  const fetchTime = Math.floor(Date.now()/1000)
  args.status = { code: 200, message: "" }

  try {

    let route = args.routeParsed||args.route
    if(args.partition && args.partition.value){
      const param = args.partition.param || args.partition.key
      route = addToRoute(route, param, args.partition.value)
    }  

    const hasCache = lastSync && lastSync.updatedStatus && lastSync.fetchTime
    console.log("hasCache", args.route, lastSync)

    const fields = args.fields || []
    for(const field of fields){
      const updated = lastSync.updatedStatus[field] || 0
      if(!updated){ route = addToRoute(route, field, 0) }
    }

    if(hasCache){
      if(lastSync.updatedStatus._default){
        route = addToRoute(route, "updated", lastSync.updatedStatus._default as number)
      } else {
        let minUpdated = 0
        for(const [key, updated] of Object.entries(lastSync.updatedStatus)){
          if(minUpdated === 0 || updated < minUpdated){ minUpdated = updated }
          if(fields.length > 0 && !fields.includes(key)){ continue }
          route = addToRoute(route, key, updated as number)
        }
        route = addToRoute(route, "updated", minUpdated)
      }

      // Revisa si es necesario realizar una nuevo fetch
      const cacheSyncTime = args.cacheSyncTime || args.useCache?.min || 0
      const fetchNextTime = lastSync.fetchTime + (cacheSyncTime * 60)
      const remainig = fetchNextTime - fetchTime

      let doFetch = args.cacheMode === "refresh"
      if(lastSync.forceNetwork){
        console.log("Forzando fetch por flag forceNetwork en ILastSync:",args.route)
        doFetch = true
        lastSync.forceNetwork = false // Resetear el flag
        await setCacheRecord(keyUpdated, lastSync, cacheName) // Persistir el reseteo
      } else if(forceFetch && !forcedFetchRequests.has(key)){
        console.log("Forzando fetch por flag global:",args.route)
        doFetch = true
      } else if(remainig <= 0){
        console.log("Preparando fetch: ",args.route," | Last: ", lastSync.fetchTime,"| Remainig:", remainig)
        doFetch = true
      }

      for(const field of args.fields || []){
        const updated = lastSync.updatedStatus[field] || 0
        if(!updated){ doFetch = true }
      }

      if(!doFetch){
        const remainig = fetchNextTime - fetchTime
        console.log(`Obviando sync fech "${key}". Quedan ${remainig}s`)
        if(args.cacheMode === 'updateOnly'){
          console.log(args.route, "Retornando null por 'updated only'")
          return { content: null }
        } else {
          const content = await getCacheRecord(key, cacheName)
          console.log(`${args.route}: Retornando registros "${args.cacheMode||"normal"}`, parseObject(content))
          return {
            content: content?._default ? content._default : content
          }
        }
      }
    }

    console.log(`Realizando fetch (${route})...`)
    const preResponse = await self.fetch(route, { headers: args.headers }) 

    if(preResponse.status && preResponse.status !== 200){
      const responseText = await preResponse.text()
      return { error: responseText }
    }

    let response = ((await parseResponseAsStream(preResponse, args))||{}) as CacheContent
    console.log("response pre-unmarshall", response)
    response = unmarshall(response)
    console.log("response post-unmarshall", response)

    if(Array.isArray(response.response) && typeof response.message === 'string'){
      response = response.response as any
    }

    if(Array.isArray(response)){ response = { _default: response } }
    response.__version__ = args.__version__

    console.log(`Fetch response recibida! (${route}) | Has-caché: ${hasCache}`)
    // Revisa si hay data a actualizar
    if(hasCache){
      console.log("prevResponse (hasCache)", args.route, args.cacheMode, {...response}) 
      response = await handleFetchResponse(args, lastSync, response, fetchTime)
    } else {
      const updatedStatus = extractUpdated(response)
      Object.assign(lastSync, { fetchTime, updatedStatus })
      setCacheRecord(keyUpdated, lastSync, cacheName)
      console.log("prevResponse (recent)", args.route, args.cacheMode, {...response}) 
      setCacheRecord(key, response, cacheName)
    }

    console.log(`${args.route}: Retornando registros "${args.cacheMode||"normal"}". ${makeStats(response).join(" | ")}`)

    return { content: response }
  } catch (error) {
    console.log("Fetch Error::", error)
    return { error: error }
  }
}

// Handler para actualizar el caché luego que un POST/PUT ha obtenido información delta como si fuera un GET
interface ICacheSyncUpdate {
  args: serviceHttpProps
  response: any
  __enviroment__: string
}

HandlersMap.set(13, async (args: ICacheSyncUpdate)=> {
  const [keyUpdated, cacheName] = makeKey(args.args)+"_updated"
  const lastFech = ((await getCacheRecord(keyUpdated, cacheName)) || {
    fetchTime: 0, updatedStatus: {} }) as ILastSync
  const nowTime = Math.floor(Date.now()/1000) - 5
  args.args.__enviroment__ = args.__enviroment__
  console.log("Guardando external fech response en caché:", args.args.route)
  handleFetchResponse(args.args, lastFech, args.response, nowTime)
})

HandlersMap.set(14, async (args: serviceHttpProps) => {
  const [key, cacheName] = makeKey(args);
  const keyUpdated = key + "_updated";
  const lastSync: ILastSync = (await getCacheRecord(keyUpdated, cacheName)) || { fetchTime: 0, updatedStatus: {} };
  lastSync.forceNetwork = true;
  await setCacheRecord(keyUpdated, lastSync, cacheName);
  console.log(`ForceNetwork set to true for key: ${keyUpdated}`);
  return { ok: 1 };
});

export interface IGetCacheSubObject {
  route: string
  module: string
  partValue?: string | number
  propInResponse?: string /* por defecto _default */
  filter?: string /* ejemplo: id=1,order=2 */
}

HandlersMap.set(15, async (args: IGetCacheSubObject)=> {
  const keyArgs = {
    route: args.route, module: args.module, partition: { value: args.partValue }
  } as serviceHttpProps

  const [key, cacheName] = makeKey(keyArgs)
  const response = await getCacheRecord(key, cacheName)

  if(!response){ return [] }
  const records = response[args.propInResponse||"_default"]
  if(!records){ return [] }

  if(args.filter){
    const filterKeyValues = args.filter.split(",").filter(x => x).map(x => {
      const filter: (string|number)[] = x.split("=")
      if(!isNaN(filter[1] as unknown as number)){
        filter.push(parseInt(filter[1] as string))
      } else {
        filter.push(0)
      }
      return filter
    })
    console.log("keyValues filters:", filterKeyValues)

    const filtered: any[] = []
    for(const r of records){
      for(const fil of filterKeyValues){
        const value = r[fil[0]]
        if(value === fil[1] || value === fil[2]){
          filtered.push(r)
        }
      }
    }
    return filtered
  } else {
    return response
  }
})

// Obtiene la cantidad de espacio de los registros obtenidos
HandlersMap.set(22, async (args: { __enviroment__: string })=> {
  console.log("obteniendo cantidad de registros obtenidos")
  const cacheStores = await caches.keys()
  console.log("cache stores::", cacheStores, "| Env:", args.__enviroment__)
  const cacheStats: { module: string, name: string, size?: number }[] = []

  for(const name of cacheStores){
    if(!name.includes("_")){ continue }
    const [envirotment, module] = name.split("_")
    if(envirotment !== args.__enviroment__){ continue }
    cacheStats.push({ name, module, size: 0 })
  }

  await Promise.all(cacheStats.map(e => {
    return new Promise((resolve, reject) => {
      let cache: Cache
      caches.open(e.name)
      .then(_cache => {
        cache = _cache
        return cache.keys()
      })
      .then(requests => {
        return Promise.all(requests.map(async (request) => {
          try {
            // 4. Match the Request object to get the corresponding Response object
            const response = await cache.match(request);
            if (response) {
              // 5. Get Content-Length from the Response headers
              // Fallback to response.headers.get('content-length') for case-insensitivity
              const contentLength = response.headers.get('Content-Length') || response.headers.get('content-length');
              if(!e.size){ e.size = 0 }
              e.size += parseInt(contentLength || '0', 10)
            } else {
              // Handle case where a request might not have a matching response (unlikely for cache.keys)
              console.warn(`Service Worker: No matching response found for request in cache '${e.name}':`, request.url)
            }
          } catch (itemError) {
            console.error(`Service Worker: Error matching request or getting size for ${request.url}:`, itemError)
          }
        }))
      })
      .then(() => {
        resolve(0)
      })
      .catch(err => {
        console.log("Error al obtener informacion del caché:", err)
        reject(err)
      })
    })
  }))

  console.log("cacheStats", cacheStats)

  return { cacheStats }
})

//Función para eliminar caché en base a un cacheName
HandlersMap.set(23, async (args: { __enviroment__: string, cacheName: string })=> {

  console.log(`Eliminando caché "${args.cacheName}" (Enviroment ${args.__enviroment__})...`)

  await caches.delete(args.cacheName)

  console.log(`Caché "${args.cacheName}" eliminado! (Enviroment ${args.__enviroment__})...`)
  
  return { ok: 1 }
})


//Función para eliminar todo el cache
HandlersMap.set(26, async (args: { __enviroment__: string })=> {
  console.log("Eliminando caché...")

  const cacheNames = await caches.keys();
  for (const name of cacheNames) {
    if (name.startsWith(args.__enviroment__)) {
      console.log(`Eliminando caché por ambiente "${args.__enviroment__}": ${name}`);
      await caches.delete(name);
    }
  }
  
  console.log("Caché eliminado.")
  return { ok: 1 }
})

//Función para refrescar caché async
interface IRefreshCache {
  __enviroment__: string, module: string, routes: string[] 
}
HandlersMap.set(24, async (args: IRefreshCache)=> {
  const cacheName = [args.__enviroment__||"main", args.module||"a"].join("_")
  console.log("Setting ForceNerwork for routes: ",args.routes)

  const cache = await caches.open(cacheName + "_"+ CACHE_APP)
  const requests = await cache.keys()
  const routesUpdated: Set<string> = new Set()

  console.log("cache:", cache,"| keys::",  requests)
  
  for (const request of requests) {
    const cachedRoute = request.url.split("/")[3]
    if(!cachedRoute){ continue }
    console.log("buscando request::", cachedRoute)
    for(const route of args.routes){
      if(cachedRoute.startsWith(route) && cachedRoute.endsWith("_updated") ){
        const lastSync: ILastSync = (await getCacheRecord(cachedRoute, cacheName))
        lastSync.forceNetwork = true
        await setCacheRecord(cachedRoute, lastSync, cacheName)
        routesUpdated.add(route)
        break
			} else {
				console.log("no es refresh::", route, cachedRoute)
      }
    }
  }

  console.log(`ForceNetwork = true for routes: ${[...routesUpdated].join(", ")} | In ${requests.length} (v2)`)
  return { ok: 1 }
})
