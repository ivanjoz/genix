import { Notify } from '$core/helpers';
import { fetchEvent } from '$core/store.svelte';
import { Env } from '$core/env';
import type { IGetCacheSubObject, serviceHttpProps } from '$core/workers/service-worker';
import { setFetchProgress } from '$core/http';

let tempID = parseInt(String(Math.floor(Date.now()/1000)).substring(4))
const serviceWorkerResolverMap: Map<number,((value: any) => void)> = new Map()
const serviceWorkerHandlerMap: Map<number,((value: any) => void)> = new Map()
const successfulResponses: Set<number> = new Set()

const nowTime = Date.now()

export const doInitServiceWorker = (): Promise<number> => {
  if(typeof navigator.serviceWorker === 'undefined'){
    console.log("serviceWorker es undefined")
    return Promise.resolve(0)
  }

  return new Promise((resolve) => {
    navigator.serviceWorker.register(
      Env.serviceWorker, { scope: '/', type: 'module' }
    ).then(() => {
      console.log("Service Worker registrado!")
      return navigator.serviceWorker.ready
    }).then(() => {
      console.log("Service Worker iniciado en: ",Date.now() - nowTime,"ms")
      // Listen for messages from the service worker
      navigator.serviceWorker.addEventListener('message', ({ data }) => {
        console.log("Mensaje del service worker::", data)

        if(data.__response__ === 5){
          setFetchProgress(data.bytes)
        } else if (data.__response__ > 0 && data.__req__ > 0) {
          if(data.__response__ === 3){
            fetchEvent(data.__req__, 0)
          }
          successfulResponses.add(data.__req__)
          if(serviceWorkerResolverMap.get(data.__req__)){
            serviceWorkerResolverMap.get(data.__req__)?.(data)
            serviceWorkerResolverMap.delete(data.__req__)
          } else {
            console.log("No se envió el request ID!")
          }
        } else {
          console.log("Mensaje del service worker no reconocido:", data)
        }
      })
      resolve(1)
    })
  })
}

export const registerServiceHandler = async (id: number, handler: (e: any) => void) => {
  serviceWorkerHandlerMap.set(id, handler)
}

export const sendServiceMessage = async (accion: number, content: any): Promise<any> => {

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
      status.updated = 0

      if(status.tryCount > 0){
        console.log(`Intentando fech por ${status.tryCount}º vez...`)
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
        console.log(`respuesta del service worker (${info}):`, res)
        status.id = 2
        status.updated = Date.now()
      })
      .catch(err => {
        console.log(`Error en la respuesta, intentando (${info}):`, err)
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
    Notify.failure(errMessage)
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
