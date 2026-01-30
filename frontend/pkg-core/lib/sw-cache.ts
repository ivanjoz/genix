import { Notify } from '$core/helpers';
import { fetchEvent } from '$core/store.svelte';
import { Env } from '$core/env';
import type { IGetCacheSubObject, serviceHttpProps } from '$core/workers/service-worker';
import { setFetchProgress } from '$core/http';

let tempID = parseInt(String(Math.floor(Date.now()/1000)).substring(4))
const serviceWorkerResolverMap: Map<number,((value: any) => void)> = new Map()
const serviceWorkerHandlerMap: Map<number,((value: any) => void)> = new Map()
const successfulResponses: Set<number> = new Set()

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

export const doInitServiceWorker = (): Promise<number> => {
  if(typeof navigator.serviceWorker === 'undefined'){
    console.log("serviceWorker es undefined")
    return Promise.resolve(0)
  }

  if (swReadyPromise) return Promise.resolve(1);

  // 1. Register listener IMMEDIATELY to avoid missing early messages
  navigator.serviceWorker.addEventListener('message', ({ data }) => {
    console.log(`[${getTS()}] [SW-Cache] Message from Service Worker:`, data);

    if(data.__response__ === 5){
      setFetchProgress(data.bytes)
    } else if (data.__response__ === 40){
      // Handle incoming WebRTC signals
      const handler = serviceWorkerHandlerMap.get(40)
      if(handler){
        handler(data)
      } else {
        console.log("No handler registered for action 40 (WebRTC signal)")
      }
    } else if (data.__response__ > 0 && data.__req__ > 0) {
      if(data.__response__ === 3){
        fetchEvent(data.__req__, 0)
      }
      successfulResponses.add(data.__req__)
      if(serviceWorkerResolverMap.get(data.__req__)){
        serviceWorkerResolverMap.get(data.__req__)?.(data)
        serviceWorkerResolverMap.delete(data.__req__)
      }
    }
  });

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
  serviceWorkerHandlerMap.set(id, handler)
}

export const sendServiceMessage = async (accion: number, content: any): Promise<any> => {
  // Ensure SW is initialized before sending
  if (!swIsInitialized) {
    await doInitServiceWorker();
  }

  const reqID = tempID
  let info = ""
  if(accion === 3){
    info = [content.route, content.cacheMode, reqID].join(" | ")
    if(content.cacheMode !== 'offline'){
      fetchEvent(reqID, { url: content.route })
    }
  }

  tempID++

  return new Promise((resolve) => {
    serviceWorkerResolverMap.set(reqID, resolve)
    const status = { id: 0, updated: 0, tryCount: 0 }

    const doFetch = () => {
      status.id = 0
      status.updated = Date.now() // Initialize with now to prevent immediate retry

      if(status.tryCount > 0){
        console.log(`[${getTS()}] [SW-Cache] Retrying fetch (Action: ${accion}, Attempt: ${status.tryCount})`);
      } else {
        console.log(`[${getTS()}] [SW-Cache] First fetch attempt (Action: ${accion})`);
      }

      let route = `${window.location.origin}/_sw_?accion=${accion}&req=${reqID}&env=${Env.enviroment}`
      if(content.route){
        route += "&r=" + content.route.replaceAll("?","_").replaceAll("&","_")
      }

      status.tryCount++
      fetch(route,{
        method: 'POST',
        body: JSON.stringify(content),
        headers: {
          'Content-Type': 'application/json' // Indicate body type
        },
      })
      .then(res => res.json())
      .then(res => {
        console.log(`[${getTS()}] [SW-Cache] Fetch response received (Action: ${accion}):`, res);
        status.id = 2
        status.updated = Date.now()
      })
      .catch(err => {
        console.log(`[${getTS()}] [SW-Cache] Fetch error (Action: ${accion}):`, err);
        status.id = 3
        status.updated = Date.now()
      })
    }

    const interval = setInterval(() => {
      if(successfulResponses.has(reqID)){
        clearInterval(interval)
        return
      }
      if(!status.updated){ return }
      if(status.id === 3){
        doFetch()
      } else {
        // Revisa si han pasado más de 2 segundos
        if(Date.now() - status.updated >= 2000){
          doFetch()
        }
      }
    },1000)

    doFetch()
  })
}

export const getCacheSubObject = async (args: IGetCacheSubObject): Promise<any[]> => {
  return await sendServiceMessage(15, args)
}

export const fetchCache = async (args: serviceHttpProps): Promise<FetchCacheResponse> => {
  args.routeParsed = Env.makeRoute(args.route)
  args.__version__ = args.useCache?.ver || 1
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
