import { parse as parseYaml } from 'yaml';
import accessListYamlContent from '../../../../backend/access_list.yml?raw';

export interface IAccessGroupCatalogEntry {
  id: number
  name: string
}

export interface IAccessListCatalogEntry {
  id: number
  name: string
  group: number
  levels: number
  frontend_routes: string | string[]
  backend_apis: string
}

export interface IAccessListCatalogPayload {
  access_groups: IAccessGroupCatalogEntry[]
  access_list: IAccessListCatalogEntry[]
}

// Parse the backend-owned YAML once so every screen reads the same catalog without a generated module.
const accessListCatalogPayload = parseYaml(accessListYamlContent) as IAccessListCatalogPayload
const accessEntriesByRoute = new Map<string, IAccessListCatalogEntry[]>()

// Normalize catalog routes once so every consumer uses the same matching rule.
export function normalizeAccessFrontendRoutes(frontendRoutes: string | string[] | undefined | null): string[] {
  const rawRoutes = Array.isArray(frontendRoutes) ? frontendRoutes : [frontendRoutes || ""]

  return rawRoutes
    .map((routeValue) => String(routeValue || "").trim().replace(/^\//, ""))
    .filter((routeValue) => routeValue.length > 0)
}

// Build a route-to-access list index because one route can be unlocked by multiple access IDs.
for (const accessEntry of accessListCatalogPayload?.access_list || []) {
  for (const normalizedRoute of normalizeAccessFrontendRoutes(accessEntry.frontend_routes)) {
    const matchedAccessEntries = accessEntriesByRoute.get(normalizedRoute) || []
    matchedAccessEntries.push(accessEntry)
    accessEntriesByRoute.set(normalizedRoute, matchedAccessEntries)
  }
}

export async function fetchAccessListCatalog(): Promise<IAccessListCatalogPayload> {
  console.info('[access-list] Loaded access catalog from backend/access_list.yml', {
    accessGroupCount: accessListCatalogPayload?.access_groups?.length || 0,
    accessEntryCount: accessListCatalogPayload?.access_list?.length || 0
  })

  return accessListCatalogPayload
}

export function getAccessEntriesByRouteMap(): Map<string, IAccessListCatalogEntry[]> {
  console.info('[access-list] Returning route access map', {
    routeCount: accessEntriesByRoute.size
  })

  return accessEntriesByRoute
}

export function getAccessEntriesForRoute(routeValue: string | undefined | null): IAccessListCatalogEntry[] {
  const normalizedRoute = String(routeValue || "").trim().replace(/^\//, "")
  return accessEntriesByRoute.get(normalizedRoute) || []
}
