/// <reference lib="WebWorker" />
"use-strict"
// Names of the two caches used in this version of the service worker.
// Change to v2, etc. when you update any of the local resources, which will
// in turn trigger the install event again.
const PRECACHE = 'precache-v2'
const CACHE_ASSETS = 'assets-v2'
const CACHE_STATIC = 'static-v2'
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
	const currentCaches = [PRECACHE, CACHE_ASSETS, CACHE_STATIC];
	event.waitUntil(
		caches.keys().then(cacheNames => {
			// The legacy app cache lived in Cache Storage under `app` or `*_app` names.
			return cacheNames.filter(cacheName => (
				cacheName === 'app' ||
				cacheName.endsWith('_app') ||
				!currentCaches.includes(cacheName)
			));
		}).then(cachesToDelete => {
			return Promise.all(cachesToDelete.map(cacheToDelete => {
				console.log('[Service Worker] Removing legacy cache store:', cacheToDelete)
				return caches.delete(cacheToDelete);
			}));
		}).then(() => self.clients.claim())
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

interface IServiceWorkerStructuredRequest {
	__swrpc__: true
	accion: number
	reqID: number
	enviroment: string
	content: any
}

interface IServiceWorkerStructuredAck {
	type: 'ack'
	__response__: number
	__req__: number
}

interface IServiceWorkerStructuredResult {
	type: 'result'
	__response__: number
	__req__: number
	content?: any
	error?: any
}

export const sendClientMessage = (clientID: number, content: any) => {
	const handler = sendHandlers.get(clientID)
	if (!handler) {
		console.log("No se encontró el handler para el client:", clientID, "|", content)
	} else {
		handler(content)
	}
}

const resolveServiceWorkerClientID = (clientIdentifier: string) => {
	// Stable numeric ids keep the duplicate-request guard compatible across fetch and MessageChannel callers.
	if (!clientIDsMap.has(clientIdentifier)) {
		clientIDsMap.set(clientIdentifier, clientIDsMap.size + 1)
	}
	return clientIDsMap.get(clientIdentifier) || 0
}

const buildServiceWorkerRPCResponse = async (
	accion: number,
	reqID: number,
	enviroment: string,
	content: any,
	clientIdentifier: string,
) => {
	const clientID = resolveServiceWorkerClientID(clientIdentifier)
	const clientReqID = reqID * 1000 + clientID
	const usedReqTime = usedRequestIDs.get(clientReqID) || 0

	if (usedReqTime && (Date.now() - usedReqTime) < 1000) {
		const haceMs = Date.now() - usedReqTime
		console.log("El id ", reqID, " está duplicado. | Client:", clientIdentifier, "| Hace:", haceMs, "ms")
		return { error: "ReqID Duplicado.", __response__: accion, __req__: reqID }
	}

	usedRequestIDs.set(clientReqID, Date.now())
	const actionHandler = HandlersMap.get(accion)
	if (!actionHandler) {
		console.warn(`No se encontró el handler para la acción ${accion}`)
		return { error: `No se encontró el handler para la acción ${accion}`, __response__: accion, __req__: reqID }
	}

	content.__enviroment__ = enviroment
	content.__client__ = clientID
	const response = await actionHandler(content)
	const message = { ...response, __response__: accion, __req__: reqID }
	let info = ""
	if (accion === 3) {
		info = [content.route, content.cacheMode, reqID].join(" | ")
	}
	console.log(`Respondiendo Fetch (${info}):`, parseObject(message))
	return message
}

self.addEventListener('message', (event) => {
	const rpcMessage = event.data as IServiceWorkerStructuredRequest | undefined
	if (!rpcMessage?.__swrpc__) { return }

	const responsePort = event.ports?.[0]
	if (!responsePort) { return }

	const sourceClient = event.source as Client | null
	const clientIdentifier = sourceClient?.id || "message-channel"

	event.waitUntil((async () => {
		try {
			const ackMessage: IServiceWorkerStructuredAck = {
				type: 'ack',
				__response__: rpcMessage.accion,
				__req__: rpcMessage.reqID,
			}
			// Ack immediately so the client can distinguish a transition failure from slow cache work.
			responsePort.postMessage(ackMessage)

			const response = await buildServiceWorkerRPCResponse(
				rpcMessage.accion,
				rpcMessage.reqID,
				rpcMessage.enviroment,
				rpcMessage.content || {},
				clientIdentifier,
			)
			const resultMessage: IServiceWorkerStructuredResult = {
				type: 'result',
				...response,
			}
			responsePort.postMessage(resultMessage)
		} catch (error) {
			const resultMessage: IServiceWorkerStructuredResult = {
				type: 'result',
				error: String((error as Error)?.message || error),
				__response__: rpcMessage.accion,
				__req__: rpcMessage.reqID,
			}
			responsePort.postMessage(resultMessage)
		} finally {
			responsePort.close()
		}
	})())
})

self.addEventListener('fetch', (event) => {
	const request = event.request
	const url = new URL(event.request.url)
	//console.log("url recibida:",url, url.pathname)
	// Skip /store routes (handled by store app service worker)
	if (url.pathname.startsWith('/store/')) {
		return;
	}
	
	// Skip requests with X-App-Scope: store header (proxied store requests)
	if (request.headers.get('X-App-Scope') === 'store') {
		return;
	}
	
	console.log("service worker url:", url.href,"|",url.searchParams.get("use-cache"))
	
	if (url.pathname === "/_sw_") {
		event.respondWith((async () => {
			// Handle internal SW RPC requests as direct request/response JSON.
			const accion = parseInt(url.searchParams.get('accion') || "0")
			const reqID = parseInt(url.searchParams.get('req') || "0")
			const enviroment = url.searchParams.get('env') || "main"
			// You MUST clone the request if you intend to read its body
			// AND then potentially pass the original request on to fetch() later.
			// Reading the body consumes the stream.
			const requestClone = event.request.clone()
			const content = await requestClone.json()
			const message = await buildServiceWorkerRPCResponse(
				accion,
				reqID,
				enviroment,
				content || {},
				event.clientId || "fetch-rpc",
			)
			// Keep the fetch fallback for uncontrolled pages and older callers.
			return new Response(JSON.stringify(message), {
				headers: { 'Content-Type': 'application/json' }
			});
		})())
		return
	} else if (request.method === 'GET' && url.searchParams.has("use-cache")) {
		// Generic timed cache for plain GET files (including cross-origin URLs like CDN assets).
		const cacheSecondsRaw = url.searchParams.get('use-cache') || "0";
		const cacheTimeSeconds = parseInt(cacheSecondsRaw);
		if (isNaN(cacheTimeSeconds) || cacheTimeSeconds <= 0) {
			console.warn(`[SW timed cache] Invalid use-cache value "${cacheSecondsRaw}" for ${request.url}. Bypassing timed cache.`);
			return fetch(event.request);
		}
		
		// Canonical cache key removes the control param, so different TTL values reuse the same resource entry.
		url.searchParams.delete('use-cache');
		const canonicalCacheKey = url.toString();
		console.log(`[SW timed cache] Checking cache for ${canonicalCacheKey} with TTL=${cacheTimeSeconds}s`);
		
		event.respondWith((async () => {
			const timedCacheStore = await caches.open('cache_assets');
			const cachedTimedResponse = await timedCacheStore.match(canonicalCacheKey);
			const nowTimestampSeconds = Math.floor(Date.now() / 1000);
			
			if (cachedTimedResponse) {
				const cachedAtSeconds = parseInt(cachedTimedResponse.headers.get('x-use-cache-timestamp') as string);
				const ageSeconds = nowTimestampSeconds - (cachedAtSeconds || 0);
				if (cachedAtSeconds && ageSeconds < cacheTimeSeconds) {
					console.log(`[SW timed cache] Serving cached response for ${canonicalCacheKey}. age=${ageSeconds}s`);
					return cachedTimedResponse;
				}
				console.log(`[SW timed cache] Cache expired for ${canonicalCacheKey}. age=${ageSeconds}s ttl=${cacheTimeSeconds}s`);
			}
			
			try {
				const networkResponse = await fetch(event.request);
				
				if (networkResponse.status === 200) {
					const responseToCache = networkResponse.clone();
					const headers = new Headers(responseToCache.headers);
					headers.set('x-use-cache-timestamp', String(nowTimestampSeconds));
					
					const responseWithCacheMetadata = new Response(await responseToCache.blob(), {
						status: responseToCache.status,
						statusText: responseToCache.statusText,
						headers
					});
					
					await timedCacheStore.put(canonicalCacheKey, responseWithCacheMetadata);
					console.log(`[SW timed cache] Cached response for ${canonicalCacheKey}`);
				} else {
					console.log(`[SW timed cache] Skipping cache due to status=${networkResponse.status} for ${canonicalCacheKey}`);
				}
				
				return networkResponse;
			} catch (networkError) {
				console.error(`[SW timed cache] Network error for ${canonicalCacheKey}`, networkError);
				if (cachedTimedResponse) {
					console.log(`[SW timed cache] Serving stale cached response for ${canonicalCacheKey}`);
					return cachedTimedResponse;
				}
				throw networkError;
			}
		})())
	}
	
	// In development mode, let the server handle all requests without service worker interference
	// This prevents reload loops when using the proxy server setup
	if (self._isLocal) {
		return
	}
	
	if (url.searchParams.has("__cache__")) {
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
				if (networkResponse.status === 200) {
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

export const parseObject = (rec: any) => {
	const newObject = {} as any
	for (const key in rec) {
		const values = rec[key]
		// console.log("v|", values)
		if (typeof values === 'number' || typeof values === 'string') {
			newObject[key] = values
		} else if (Array.isArray(values)) {
			newObject[key] = `[${values.length}]`
		} else if (values && typeof values === 'object') {
			newObject[key] = `{${Object.keys(values).join(", ")}}`
		}
	}
	return newObject
}
