/// <reference lib="WebWorker" />
"use-strict"

import { recreateObject } from "$core/sharedHelpers"

// Names of the two caches used in this version of the service worker.
// Change to v2, etc. when you update any of the local resources, which will
// in turn trigger the install event again.
const PRECACHE = 'precache-v2'
const CACHE_ASSETS = 'assets-v2'
const CACHE_STATIC = 'static-v2'
export const CACHE_APP = 'app'
export const HandlersMap: Map<number, (input: any) => Promise<any>> = new Map()

// A list of local resources we always want to be cached.
const PRECACHE_URLS: string[] = [
  // 'index.html',
  // './', // Alias for index.html
]

const parseFileExtension = (filename: string) => {
  const ix1 = filename.indexOf("?")
  if (ix1 !== -1) filename = filename.substring(0, ix1)
  const ix2 = filename.indexOf("@")
  if (ix2 !== -1) filename = filename.substring(0, ix2)
  filename = filename.substring(filename.indexOf("/", 8))
  const ix3 = filename.lastIndexOf(".")
  if (filename === "/" || (ix3 === -1 && filename[2]) === "/") {
    return [filename, "*"]
  }
  return [filename, filename.substring(ix3 + 1)]
}

declare const self: ServiceWorkerGlobalScope & {
  __WB_MANIFEST: string[]
  _isLocal: boolean
};

// The install handler takes care of precaching the resources we always need.
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(PRECACHE)
      .then(cache => cache.addAll(PRECACHE_URLS))
      .then(() => self.skipWaiting())
  )
})

// The activate handler takes care of cleaning up old caches.
self.addEventListener('activate', event => {
  const currentCaches = [PRECACHE, CACHE_ASSETS, CACHE_STATIC, CACHE_APP];
  event.waitUntil(
    self.clients.claim()
    /*
    caches.keys().then(cacheNames => {
      return cacheNames.filter(cacheName => !currentCaches.includes(cacheName));
    }).then(cachesToDelete => {
      return Promise.all(cachesToDelete.map(cacheToDelete => {
        return caches.delete(cacheToDelete);
      }));
    }).then(() => self.clients.claim())
    */
  )
})

const extractVersion = (bodyHTML: string) => {
  const idx1 = bodyHTML.indexOf(`name="build-version"`)
  if (idx1 === -1) return null
  const idx2 = bodyHTML.indexOf(`"`, idx1 + 28)
  const buildNumber = bodyHTML.substring(idx2 + 1, idx2 + 7)
  return buildNumber
}

const versionToHour = (buildNumber: string) => {
  const nowTime = Math.floor(Date.now() / 1000)
  const buildTime = parseInt(buildNumber.toLowerCase(), 36) * 10
  const seconds = nowTime - buildTime
  const hours = Math.floor(seconds / 60 / 60)
  const minutes = Math.floor((seconds - (hours * 60 * 60)) / 60)
  const horasText = `${hours} hora${hours === 1 ? '' : 's'}`
  let hace = `${horasText} ${minutes} min`
  if (hours === 0) hace = `${minutes} min`
  if (hours > 12) hace = `más de ${horasText}`
  return hace
}

const VersionInfo = { build: "", hasUpdated: false }
export const ClientIDPort: Map<number, MessagePort> = new Map()

// ping / pong
HandlersMap.set(99, async (message) => {
  return { ...message, response: "pong" }
})

self._isLocal = self.origin.includes("localhost") || self.origin.includes("127.0.0.1")

//Compara las versiones
let lastNewVersionChecked = 0

HandlersMap.set(7, async (message) => {
  const nowTime = Math.floor(Date.now() / 1000)
  if (lastNewVersionChecked && nowTime - lastNewVersionChecked < 30) {
    console.log("Saltando revisión de actualización.")
    return {}
  }

  const versionCurrent = message.version
  if (!versionCurrent) {
    console.log("No se envió la version (build) a comparar.")
    return {}
  }
  const versionHasUpdated = await compareVersionUpdate(versionCurrent)
  lastNewVersionChecked = nowTime
  return { versionHasUpdated }
})

const compareVersionUpdate = async (versionCurrent: string): Promise<string> => {
  VersionInfo.build = versionCurrent

  const headers = new Headers()
  headers.append('pragma', 'no-cache')
  headers.append('cache-control', 'no-cache')

  try {
    const preResp = await fetch(self.location.origin + "/app-version", { method: 'GET', headers })
    const bodyHTML = await preResp.text()
    const versionUpdated = extractVersion(bodyHTML) || ""
    VersionInfo.build = versionUpdated
    console.log('build code compare fetch:: ', versionCurrent, " | ", versionUpdated)

    if (versionCurrent !== versionUpdated) {
      await caches.delete(CACHE_ASSETS)
      return versionToHour(versionUpdated)
    }
  } catch (error) {
    console.warn('Error al obtener la versión nueva (HTML)::', error)
  }
  return ""
}

// The fetch handler serves responses for same-origin resources from a cache.
// If no response is found, it populates the runtime cache with the response from the network before returning it to the page.
const clientIDsMap: Map<string, number> = new Map()
const usedRequestIDs: Map<number, number> = new Map()
const sendHandlers: Map<number, (c: any) => void> = new Map()
export const sendClientMessage = (clientID: number, content: any) => {
  const handler = sendHandlers.get(clientID)
  if (!handler) {
    console.log("No se encontró el handler para el client:", clientID, "|", content)
  } else {
    handler(content)
  }
}

self.addEventListener('fetch', (event) => {
  const request = event.request
  const url = new URL(event.request.url)
  //console.log("url recibida:",url, url.pathname)

  if (url.pathname === "/_sw_") {
    event.respondWith((async () => {
      // 3. Parse incoming data (from URL params or request body)
      const accion = parseInt(url.searchParams.get('accion') || "0")
      const reqID = parseInt(url.searchParams.get('req') || "0")
      const enviroment = url.searchParams.get('env') || "main"

      if (!clientIDsMap.has(event.clientId)) {
        clientIDsMap.set(event.clientId, clientIDsMap.size + 1)
      }
      const clientID = clientIDsMap.get(event.clientId) || 0

      const clientReqID = reqID * 1000 + clientID
      const usedReqTime = usedRequestIDs.get(clientReqID) || 0
      if (usedReqTime && (Date.now() - usedReqTime) < 1000) {
        const haceMs = Date.now() - usedReqTime
        console.log("El id ", reqID, " está duplicado. | Client:", event.clientId, "| Hace:", haceMs, "ms")
        return new Response(JSON.stringify({ "Error": "ReqID Duplicado." }), {
          headers: { 'Content-Type': 'application/json' }
        })
      }
      usedRequestIDs.set(reqID, Date.now())


      const client = await self.clients.get(event.clientId)
      if (!client) {
        console.warn(`No se encontró el client con ID ${event.clientId}`)
        const msg = { error: `No se encontró el client con ID ${event.clientId}` }
        return new Response(JSON.stringify(msg), {
          headers: { 'Content-Type': 'application/json' }
        })
      }

      sendHandlers.set(clientID, (content: any) => {
        client.postMessage(content)
      })

      const handler = HandlersMap.get(accion)
      if (!handler) {
        console.warn(`No se encontró el handler para la acción ${accion}`)
        const msg = { error: `No se encontró el handler para la acción ${accion}` }
        return new Response(JSON.stringify(msg), {
          headers: { 'Content-Type': 'application/json' }
        })
      }

      // You MUST clone the request if you intend to read its body
      // AND then potentially pass the original request on to fetch() later.
      // Reading the body consumes the stream.
      const requestClone = event.request.clone()
      const content = await requestClone.json()

      content.__enviroment__ = enviroment
      content.__client__ = clientID

      const response = await handler(content)
      const message = { ...response, __response__: accion, __req__: reqID }
      let info = ""
      if (accion === 3) {
        info = [content.route, content.cacheMode, reqID].join(" | ")
      }
      // console.log("Respuesta a enviar:", info, response)
      client.postMessage(message)

      // 5. Respond to the fetch request
      console.log(`Respondiendo Fetch (${info}):`, (parseObject(message)))
      return new Response(JSON.stringify({ "ok": 1 }), {
        headers: { 'Content-Type': 'application/json' }
      });
    })())
    return
  } else if (url.searchParams.has("__cache__")) {
    const cacheParam = url.searchParams.get('__cache__') || "";
    const [cacheTimeMinutes_, cacheVersion_, enviroment] = cacheParam.split('.')
    const cacheTimeMinutes = parseInt(cacheTimeMinutes_)
    const cacheVersion = parseInt(cacheVersion_)

    if (isNaN(cacheTimeMinutes) || isNaN(cacheVersion)) {
      console.warn(`Invalid __cache__ parameter format: ${cacheParam}. Bypassing cache.`);
      return fetch(event.request);
    }

    // Create a canonical URL for caching, removing the __cache__ parameter
    url.searchParams.delete('__cache__')
    const cacheKey = url.toString()
    console.log("buscando caché:", cacheKey)

    event.respondWith((async () => {

      const cache = await caches.open(`cache_req_${enviroment}`);
      const cachedResponse = await cache.match(cacheKey);

      const nowTime = Math.floor(Date.now() / 1000)

      if (cachedResponse) {
        const cachedTimestamp = parseInt(cachedResponse.headers.get('x-cache-timestamp') as string)
        const cachedVersion = parseInt(cachedResponse.headers.get('x-cache-version') as string)

        // Check for cache freshness and version
        if (cachedTimestamp && cachedVersion &&
          (nowTime - cachedTimestamp < cacheTimeMinutes * 60) && cachedVersion === cacheVersion) {
          console.log(`Serving from cache: ${cacheKey}`)
          return cachedResponse
        } else {
          console.log(`Cache stale or version mismatch for ${cacheKey}. Fetching new.`)
          // If stale or version mismatch, proceed to fetch
        }
      }

      // If no cached response, or if it's stale/version mismatch, fetch from network
      try {
        const networkResponse = await fetch(event.request)

        // Clone the response because a response can only be consumed once
        if(networkResponse.status === 200){
          const responseToCache = networkResponse.clone()

          // Add custom headers to the response to store timestamp and version
          const headers = new Headers(responseToCache.headers)
          headers.set('x-cache-timestamp', String(nowTime))
          headers.set('x-cache-version', String(cacheVersion))
  
          const responseWithHeaders = new Response(await responseToCache.blob(), {
            status: responseToCache.status,
            statusText: responseToCache.statusText,
            headers: headers
          })
  
          await cache.put(cacheKey, responseWithHeaders)
          console.log(`Fetched and cached: ${cacheKey}`)
        } else {
          console.log(`La consulta falló::`, networkResponse.status)
        }
        return networkResponse;
      } catch (error) {
        console.error(`Fetch failed for ${cacheKey}:`, error)
        // Optionally, you can return the cached response even if stale
        // if a network error occurs (cache-first with network fallback)
        if (cachedResponse) {
          console.log(`Network failed, serving stale cache for ${cacheKey}`)
          return cachedResponse
        }
        // Re-throw or return a network error response
        throw error
      }
    })())
  }

  if (self._isLocal) {
    return
  }
  const contentType = request.headers.get('Content-Type');

  console.log("event URL:: ", request.url, "|", contentType)
  if (!request.url.startsWith(self.location.origin)) return

  const [filename, ext] = parseFileExtension(request.url)
  if (filename === '/app-version') { return }

  // 1. Determine if it's a navigation request for your SPA
  const requestURL = new URL(request.url);
  const isSameOrigin = requestURL.origin === self.location.origin;
  // Request initiated by browser navigation (e.g., direct URL, refresh, link click)
  const isHTMLNavigation = request.mode === 'navigate';
  const hasNoExtension = !requestURL.pathname.includes('.') || requestURL.pathname === '/';


  // como es una SPA todas las URL internas van hacia "/"
  // ext === '*' significa que es una ruta interna del SAP (sin extension) en vez de un archivo 
  if (ext === '*') {
    console.log("Re-routing Service Worker:: ", filename)
  }

  const CACHE_NAME = ['js', 'css', 'html', '*', 'ts', 'mjs', 'tsx'].includes(ext)
    ? CACHE_ASSETS : CACHE_STATIC

  const OFFLINE_URL = "/"

  if (isSameOrigin && isHTMLNavigation && hasNoExtension) {
    event.respondWith(
      caches.open(CACHE_NAME).then(cache => {
        return cache.match(OFFLINE_URL).then(cachedResponse => {
          // If it's in the cache, return it immediately
          if (cachedResponse && !navigator.onLine) {
            console.log(`[Service Worker] Serving cached HTML for ${request.url} from ${OFFLINE_URL}`);
            return cachedResponse;
          }
          // If not in cache, or no specific HTML for the current route was found,
          // go to network for the original request URL
          console.log(`[Service Worker] HTML not in cache, fetching from network: ${request.url}`);
          return fetch(request)
            .then(networkResponse => {
              // Check if we received a valid HTML response
              // We are specifically checking for 'text/html' for HTML caching
              const contentType = networkResponse.headers.get('Content-Type');
              if (networkResponse.ok && contentType && contentType.includes('text/html')) {
                // Clone the response and store it in cache with the OFFLINE_URL key
                // This means all successful HTML navigation responses
                // will be saved as if they were the OFFLINE_URL ('/')
                console.log(`[Service Worker] Caching new HTML response from ${request.url} as ${OFFLINE_URL}`);
                cache.put(OFFLINE_URL, networkResponse.clone());
              }
              return networkResponse; // Return the network response to the browser
            })
            .catch(error => {
              if (cachedResponse) {
                return cachedResponse
              }
              // Network failed for HTML navigation.
              console.error(`[Service Worker] Network failed for HTML navigation: ${request.url}`, error);
              return new Response('<h1>Offline</h1><p>You appear to be offline and this content is not cached.</p>', {
                headers: { 'Content-Type': 'text/html' }
              });
            });
        });
      })
    )
    return
  }

  // Skip cross-origin requests, like those for Google Analytics.
  event.respondWith(
    caches.open("cache_").then(cache => {
      return cache.match(request).then(cachedResponse => {

        if (cachedResponse) {
          const contentType = cachedResponse.headers.get("content-type") || ""
          if (!contentType.includes("/html")) {
            console.log('[Service Worker] Serving from cache:', request.url);
            return cachedResponse
          } else {
            console.log("Es HTML!!", ext)
          }
        }

        return fetch(request)
          .then(response => {
            // Check if we received a valid response
            if (!response || response.status !== 200 || response.type !== 'basic') {
              return response;
            }
            const contentType = request.headers.get('Content-Type');
            console.log("Guardando en caché: ", CACHE_NAME, " | " + request.url, " | ", contentType)

            // Clone the response and put it in the opened cache
            const responseToCache = response.clone();
            console.log('[Service Worker] Caching new asset:', request.url);
            cache.put(request, responseToCache); // Use the specific cache object for putting

            return response;
          })
          .catch(error => {
            console.error('[Service Worker] Fetch failed:', request.url, error)
            return new Response('Network error occurred.',
              { status: 503, statusText: 'Service Unavailable' });
          })
      })
    })
  )
})

const appMemoryCache: Map<string, Map<string, any>> = new Map()

export const hasCacheKey = async (key: string, cache: string): Promise<any> => {
  const cacheName = cache ? cache + "_" + CACHE_APP : CACHE_APP
  const appCache = await caches.open(cacheName)
  const hasCache = await appCache.match(key)
  const memoryCache = appMemoryCache.get(cacheName)
  if (!hasCache && memoryCache) { memoryCache.delete(key) }
  return hasCache
}

export const getCacheRecord = async (key: string, cache: string): Promise<any> => {
  const cacheName = cache ? cache + "_" + CACHE_APP : CACHE_APP
  if (!appMemoryCache.has(cacheName)) { appMemoryCache.set(cacheName, new Map()) }

  const memoryCache = appMemoryCache.get(cacheName)
  if (memoryCache?.has(key)) {
    // Revisa si el cache realmente existe, eso debido a que puede haber sido eliminado y se debe limpiar tambien el que está en memoria
    const cacheExists = await caches.has(cacheName)
    if (cacheExists) {
      return memoryCache.get(key)
    } else {
      appMemoryCache.set(cacheName, new Map())
    }
  }

  const appCache = await caches.open(cacheName)
  const startTime = Date.now()
  const response = await appCache.match(key);
  if (response) {
    let jsonResponse = await response.json(); // Parse the response body as JSON
    if (jsonResponse.__keys__) {
      const keysMap = new Map(jsonResponse.__keys__) as Map<string, string | number>
      // console.log("cache parsedContent 2", jsonResponse.content, keysMap)
      const keysMapReversed = new Map([...keysMap.entries()].map(x => [x[1], x[0]]))

      jsonResponse = recreateObject(jsonResponse.content, keysMapReversed as unknown as Map<string, number>)
    }
    console.log(`Cache response "${key}" in ${Date.now() - startTime}ms (v2)`)
    return jsonResponse
  }
  return undefined; // Or handle as you see fit if the key is not found
};

export const setCacheRecord = async (key: string, content: any, cache: string): Promise<void> => {
  const cacheName = cache ? cache + "_" + CACHE_APP : CACHE_APP
  if (!appMemoryCache.has(cacheName)) { appMemoryCache.set(cacheName, new Map()) }
  const appCache = await caches.open(cacheName)

  const memoryCache = appMemoryCache.get(cacheName)
  if (!content) { // Elimina el caché si se señala que no hay contenido
    memoryCache?.delete(key)
    await appCache.delete(key)
    return
  }
  memoryCache?.set(key, content)

  const startTime = Date.now()
  if (typeof content !== 'string') {
    /*
    const keysMap = new Map()
    const parsedContent = simplifyObject(content, keysMap)
    content = JSON.stringify({ __keys__: [...keysMap.entries()], content: parsedContent }) 
    */
    content = JSON.stringify(content)
  }
  // Create a Response object with the JSON content
  const response = new Response(content, {
    headers: {
      'Content-Type': 'application/json',
      'Content-Length': String(content.length)
    }
  });
  await appCache.put(key, response);
  console.log(`Cache put "${key}" in ${Date.now() - startTime}ms (v2)`)
};

export const parseObject = (rec: any) => {

  const newObject = {} as any

  for(const key in rec){
    const values = rec[key]
    // console.log("v|", values)
    if(typeof values === 'number' || typeof values === 'string'){
      newObject[key] = values
    } else if(Array.isArray(values)){
      newObject[key] = `[${values.length}]`
    } else if(values && typeof values === 'object') {
      newObject[key] = `{${Object.keys(values).join(", ")}}`
    }
  }
  return newObject
}