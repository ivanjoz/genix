import accessCatalogTomlContent from '../../../../backend/access.toml?raw';
import { parseTomlTables, splitCatalogRoutes } from '#libs/toml-tables.utils.ts';

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
  // Parallel arrays, equal length, ids 2..13. Absent on the accesses that declare no sub-access,
  // which is most of them. Id 1 is reserved for "Todos" and is never declared here.
  sub_accesses_ids?: number[]
  sub_accesses_names?: string[]
}

export interface IAccessListCatalogPayload {
  groups: IAccessGroupCatalogEntry[]
  access: IAccessListCatalogEntry[]
}

let accessListCatalog: IAccessListCatalogPayload | null = null
const accessEntriesByRoute = new Map<string, IAccessListCatalogEntry[]>()

function parseAccessCatalog(tomlContent: string): IAccessListCatalogPayload {
  return parseTomlTables(tomlContent, ['groups', 'access'], 'access-catalog') as unknown as IAccessListCatalogPayload
}

// Split the catalog's compact comma list once so every consumer uses the same routes.
export const normalizeAccessFrontendRoutes = splitCatalogRoutes

function indexAccessEntries(payload: IAccessListCatalogPayload): void {
  accessEntriesByRoute.clear()

  // One route can be unlocked by multiple access IDs.
  for (const accessEntry of payload.access || []) {
    for (const normalizedRoute of normalizeAccessFrontendRoutes(accessEntry.frontend_routes)) {
      const matchedAccessEntries = accessEntriesByRoute.get(normalizedRoute) || []
      matchedAccessEntries.push(accessEntry)
      accessEntriesByRoute.set(normalizedRoute, matchedAccessEntries)
    }
  }
}

export async function fetchAccessListCatalog(): Promise<IAccessListCatalogPayload> {
  if (!accessListCatalog) {
    console.debug('[access-list] Parsing access catalog')
    accessListCatalog = parseAccessCatalog(accessCatalogTomlContent)
    indexAccessEntries(accessListCatalog)

    console.info('[access-list] Access catalog ready', {
      accessGroupCount: accessListCatalog.groups.length,
      accessEntryCount: accessListCatalog.access.length
    })
  }

  return accessListCatalog
}

export function getAccessEntriesByRouteMap(): Map<string, IAccessListCatalogEntry[]> {
  console.info('[access-list] Returning route access map', {
    routeCount: accessEntriesByRoute.size
  })

  return accessEntriesByRoute
}

export function getAccessEntriesForRoute(routeValue: string | undefined | null): IAccessListCatalogEntry[] {
  const routeSegments = String(routeValue || "").trim().replace(/^\//, "").split("/").filter(Boolean)

  // Match by whole segments, longest first: "webpage-builder/gallery" beats "webpage-builder" for
  // the gallery, and the page editor "/webpage-builder/<pageID>" still inherits "webpage-builder".
  // Comparing segment by segment (and not by string prefix) keeps "finance/cash-banks" from
  // swallowing "finance/cash-banks-movements".
  for (let segmentCount = routeSegments.length; segmentCount > 0; segmentCount--) {
    const matchedAccessEntries = accessEntriesByRoute.get(routeSegments.slice(0, segmentCount).join("/"))
    if (matchedAccessEntries) { return matchedAccessEntries }
  }

  return []
}

// Exported for the tests: the parser is the only thing standing between the catalog file and every
// access check in the app, so its rejections are worth asserting directly.
export const parseAccessCatalogForTest = parseAccessCatalog
