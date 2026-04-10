import { Notify } from '$libs/helpers';
import { fetchEvent } from '$core/store.svelte';
import { Env } from '$core/env';
import type { IGetCacheSubObject, serviceHttpProps } from '$libs/workers/service-worker';
import { setFetchProgress } from '$libs/http.svelte';

let tempID = parseInt(String(Math.floor(Date.now()/1000)).substring(4))

const getTS = () => {
  const d = new Date();
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}.${d.getMilliseconds().toString().padStart(3, '0')}`;
};

const nowTime = Date.now()

export interface FetchCacheResponse {
  content?: any
  error?: string | object
  isEmpty?: boolean
  notUpdated?: boolean
}

let swReadyPromise: Promise<void> | null = null;
let swIsInitialized = false;
const textDecoder = new TextDecoder()

const parseJSONSafe = (textPayload: string) => {
  if (!textPayload) return {}
  try {
    return JSON.parse(textPayload)
  } catch (error) {
    return { error: `Invalid JSON response: ${String(error)}` }
  }
}

const readServiceWorkerResponse = async (
  response: Response, shouldTrackProgress: boolean
) => {
  // If stream is unavailable, parse directly and keep the fallback robust.
  if (!response.body) {
    const textPayload = await response.text()
    return parseJSONSafe(textPayload)
  }

  const reader = response.body.getReader()
  let textPayload = ""

  // Stream chunks so we can update byte progress without postMessage.
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    if (value) {
      if (shouldTrackProgress) {
        setFetchProgress(value.length)
      }
      textPayload += textDecoder.decode(value, { stream: true })
    }
  }
  textPayload += textDecoder.decode()

  return parseJSONSafe(textPayload)
}

export const doInitServiceWorker = (): Promise<number> => {
  if(typeof navigator.serviceWorker === 'undefined'){
    console.log("serviceWorker es undefined")
    return Promise.resolve(0)
  }

  if (swReadyPromise) return Promise.resolve(1);

  swReadyPromise = new Promise((resolve) => {
    navigator.serviceWorker.register(
      Env.serviceWorker, { scope: '/', type: 'module' }
    ).then(() => {
      console.log(`[${getTS()}] [SW-Cache] Service Worker registered!`);
      return navigator.serviceWorker.ready
    }).then(() => {
      console.log(`[${getTS()}] [SW-Cache] Service Worker ready (Iniciado en: ${Date.now() - nowTime}ms)`);
      swIsInitialized = true;
      resolve();
    })
  })

  return swReadyPromise.then(() => 1);
}

export const registerServiceHandler = async (id: number, handler: (e: any) => void) => {
  // Deprecated path: SW now uses direct fetch/response for request-reply actions.
  console.warn(`[SW-Cache] registerServiceHandler(${id}) is deprecated and ignored.`)
  void handler
}

export const sendServiceMessage = async (accion: number, content: any): Promise<any> => {
  // Ensure SW is initialized before sending
  if (!swIsInitialized) {
    await doInitServiceWorker();
  }

  const reqID = tempID
  let requestInfo = ""
  const shouldTrackProgress = accion === 3
  const shouldNotifyFetchLifecycle = accion === 3 && content.cacheMode !== 'offline'
  if(accion === 3){
    requestInfo = [content.route, content.cacheMode, reqID].join(" | ")
    if(shouldNotifyFetchLifecycle){
      fetchEvent(reqID, { url: content.route })
    }
  }

  tempID++

  try {
    if (content && typeof content === 'object') {
      content.__companyID__ = Env.getEmpresaID()
    }

    let route = `${window.location.origin}/_sw_?accion=${accion}&req=${reqID}&env=${Env.enviroment}`
    if(content.route){
      route += "&r=" + content.route.replaceAll("?","_").replaceAll("&","_")
    }
    console.log(`[${getTS()}] [SW-Cache] Fetch request (${accion}):`, route, requestInfo)

    const swResponse = await fetch(route,{
      method: 'POST',
      body: JSON.stringify(content),
      headers: {
        'Content-Type': 'application/json'
      },
    })

    const parsedResponse = await readServiceWorkerResponse(swResponse, shouldTrackProgress)
    console.log(`[${getTS()}] [SW-Cache] Fetch response received (Action: ${accion}):`, parsedResponse)
    return parsedResponse
  } catch (error) {
    const errorMessage = String((error as any)?.message || error)
    console.error(`[${getTS()}] [SW-Cache] Fetch error (Action: ${accion}):`, error)
    return { error: errorMessage, __response__: accion, __req__: reqID }
  } finally {
    // Ensure UI fetch lifecycle is closed even on stream parse/network failures.
    if (shouldNotifyFetchLifecycle) {
      fetchEvent(reqID, 0)
    }
  }
}

export const getCacheSubObject = async (args: IGetCacheSubObject): Promise<any[]> => {
  return await sendServiceMessage(15, args)
}

export const fetchCache = async (args: serviceHttpProps): Promise<FetchCacheResponse> => {
  args.routeParsed = Env.makeRoute(args.route)
  args.__version__ = args.useCache?.ver || 1
  args.__companyID__ = Env.getEmpresaID()
  console.log("fetching cache...", args)

  const response = await sendServiceMessage(3,args)
  console.log("cache response::",response)

  return response as FetchCacheResponse
}

export const fetchCacheParsed = async (args: serviceHttpProps): Promise<any> => {
  const response = await fetchCache(args)

  if(response.error){
    let errMessage = response.error
    if(typeof response.error === 'string'){
      try {
        let errorJson = JSON.parse(response.error)
        errMessage = errorJson.error || JSON.stringify(errorJson)
      } catch (_) { }
    } else {
      errMessage = (response.error as any).error || response.error
    }
    console.log("errMessage", errMessage)
    Notify.failure(String(errMessage))
    return null
  }

  let content = response.content
  // debugger

  if (args.cacheMode === 'offline') {
    if(!content || response.isEmpty){
      return null
    }
  } else if (args.cacheMode === 'updateOnly') {
    if (response.notUpdated) {
      return null
    }
  }

  if(content?._default){ return content._default }
  return content
}
