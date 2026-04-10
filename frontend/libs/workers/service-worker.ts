import {
  applyExternalDeltaResponse,
  clearDeltaEnvironmentCache,
  clearDeltaModuleCache,
  fetchDeltaCache,
  getDeltaCacheStats,
  getDeltaUpdatedStatus,
  readDeltaCacheSubObject,
  refreshDeltaRoutes,
  setDeltaRouteForceNetwork,
  triggerDeltaForceFetchWindow,
} from '$libs/cache/delta-cache.fetch';
import type { ICacheSyncUpdate, IGetCacheSubObject } from '$libs/cache/delta-cache.fetch';
import { HandlersMap } from './service-worker-cache';

export type CacheMode = 'offline' | 'updateOnly' | 'refresh' | 'fetchOnly'

export type serviceHttpProps = {
  __enviroment__: string
  __companyID__?: number
  __accion__: number
  __client__: number
  __req__?: number
  __version__?: number
  route: string
  module?: string
  routeParsed?: string
  headers?: { [key: string]: string } | Headers
  keyID?: string | string[]
  keysIDs?: { [key: string]: string | string[] }
  columnarIDField?: string
  combineColumnarValuesOnFields?: string[]
  fields?: string[]
  keyFilterIfEmpty?: string
  keyForUpdated?: string
  cacheMode?: CacheMode
  contentLength?: number
  partition?: {
    key: string
    value: string | number
    param?: string
  }
  status?: { code: number, message: string }
  updatedStatus?: { [key: string]: string }
  cacheSyncTime?: number
  useCache?: {
    min: number
    ver: number
  }
  useCacheStatic?: {
    min: number
    ver: number
  }
}

export type { ICacheSyncUpdate, IGetCacheSubObject }

// Action 3 is the main delta-cache entrypoint used by cached services.
HandlersMap.set(3, async (args: serviceHttpProps) => {
  return await fetchDeltaCache(args)
})

// Action 11 opens a short force-fetch window for subsequent cache reads.
HandlersMap.set(11, async () => {
  return await triggerDeltaForceFetchWindow()
})

// Action 12 exposes the latest per-response updated watermark for a route.
HandlersMap.set(12, async (args: serviceHttpProps) => {
  return await getDeltaUpdatedStatus(args)
})

// Action 13 applies delta payloads produced by write endpoints into the cache.
HandlersMap.set(13, async (args: ICacheSyncUpdate) => {
  return await applyExternalDeltaResponse(args)
})

// Action 14 marks a route to bypass cache on the next fetch.
HandlersMap.set(14, async (args: serviceHttpProps) => {
  return await setDeltaRouteForceNetwork(args)
})

// Action 15 reads a cached response block or filtered records from a route.
HandlersMap.set(15, async (args: IGetCacheSubObject) => {
  return await readDeltaCacheSubObject(args)
})

// Action 21 is still used by the client to acknowledge delivered responses.
const acknowledgeResponses: Set<number> = new Set()
HandlersMap.set(21, async (args: serviceHttpProps) => {
  const requestID = (args.__req__ || 0) * 1000 + args.__client__
  acknowledgeResponses.add(requestID)
})

// Action 22 returns cache stats grouped by module for the current environment.
HandlersMap.set(22, async (args: { __enviroment__: string, __companyID__?: number }) => {
  return await getDeltaCacheStats(args)
})

// Action 23 clears a whole module cache in the selected environment.
HandlersMap.set(23, async (args: { __enviroment__: string, __companyID__?: number, cacheName: string }) => {
  return await clearDeltaModuleCache(args)
})

// Action 24 marks matching routes for refresh without deleting their rows.
HandlersMap.set(24, async (args: { __enviroment__: string, __companyID__?: number, module: string, routes: string[] }) => {
  return await refreshDeltaRoutes(args)
})

// Action 26 removes the full delta cache for one environment.
HandlersMap.set(26, async (args: { __enviroment__: string, __companyID__?: number }) => {
  return await clearDeltaEnvironmentCache(args)
})
