import { goto } from '$app/navigation'
import { Env } from '$core/env'
import {
  deleteRouteData,
  getRouteData,
  makeDeltaCacheDatabaseName,
  putRouteData,
} from './delta-cache.idb'

// Resolves the active company+environment-scoped IndexedDB name once per call;
// keeping it small avoids stale captures when the user switches company at runtime.
const getActiveDbName = (): string => {
  return makeDeltaCacheDatabaseName(Env.getEmpresaID(), Env.enviroment || 'main')
}

export interface IRouteRecordResult<T> {
  // record/err are mutually exclusive: when the URL has no query param, both are undefined.
  record?: T
  err?: string
}

// Persists `value` under `key` in the routeData KV store so a subsequent navigation
// (driven by a `?<paramName>=<key>` URL param) can read it back.
export const saveRouteRecord = async (key: string, value: unknown): Promise<void> => {
  const dbName = getActiveDbName()
  console.debug('[route-data] put', { dbName, key })
  await putRouteData(dbName, key, value)
  console.debug('[route-data] put:done', { key })
}

// Reads `?<paramName>=` from the current URL and resolves the stored record.
// - param missing: returns {} (no error, no record).
// - param present + record found: returns { record }.
// - param present + record missing: strips the param from the URL and returns { err: 'Not Found' }.
export const loadRouteRecordFromQueryParam = async <T = any>(
  paramName: string = 'rec',
): Promise<IRouteRecordResult<T>> => {
  if (typeof window === 'undefined') {
    console.debug('[route-data] load:ssr-skip')
    return {}
  }
  const url = new URL(window.location.href)
  const key = url.searchParams.get(paramName)
  console.debug('[route-data] load:start', { paramName, key, href: url.href })
  if (!key) { return {} }

  const dbName = getActiveDbName()
  const record = await getRouteData<T>(dbName, key)
  if (record === undefined) {
    console.warn('[route-data] load:not-found — clearing url param', { paramName, key, dbName })
    // Strip the dangling param so refreshing the page doesn't re-trigger the lookup.
    url.searchParams.delete(paramName)
    await goto(url.pathname + (url.search || '') + url.hash, { noScroll: true, replaceState: true, keepFocus: true })
    return { err: 'Not Found' }
  }
  console.debug('[route-data] load:ok', { paramName, key })
  return { record }
}

// Adds or updates `?<paramName>=<key>` on the current URL without a full page reload.
export const setRouteRecordQueryParam = async (paramName: string, key: string): Promise<void> => {
  if (typeof window === 'undefined') { return }
  const url = new URL(window.location.href)
  url.searchParams.set(paramName, key)
  const target = url.pathname + url.search + url.hash
  console.debug('[route-data] setQueryParam', { paramName, key, target })
  await goto(target, { noScroll: true, replaceState: false, keepFocus: true })
  console.debug('[route-data] setQueryParam:done', { href: window.location.href })
}

// Removes `?<paramName>=` from the URL after the consumer has finished reading it,
// so the form can be edited freely without the param re-firing on refresh.
export const clearRouteRecordQueryParam = async (paramName: string = 'rec'): Promise<void> => {
  if (typeof window === 'undefined') { return }
  const url = new URL(window.location.href)
  if (!url.searchParams.has(paramName)) { return }
  url.searchParams.delete(paramName)
  const target = url.pathname + (url.search || '') + url.hash
  console.debug('[route-data] clearQueryParam', { paramName, target })
  await goto(target, { noScroll: true, replaceState: true, keepFocus: true })
}

// Re-export the IDB-level delete in case a caller needs to forget a stored entry explicitly.
export const removeRouteRecord = async (key: string): Promise<void> => {
  await deleteRouteData(getActiveDbName(), key)
}
