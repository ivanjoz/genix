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

let accessListCatalog: IAccessListCatalogPayload | null = null
const accessEntriesByRoute = new Map<string, IAccessListCatalogEntry[]>()

// Parse only the controlled access-list YAML shape: top-level lists of scalar records.
function parseAccessListCatalog(yamlContent: string): IAccessListCatalogPayload {
  const parsedCatalog: IAccessListCatalogPayload = { access_groups: [], access_list: [] }
  let activeRecords: Record<string, string | number>[] | null = null
  let activeRecord: Record<string, string | number> | null = null

  for (const [lineIndex, sourceLine] of yamlContent.split(/\r?\n/).entries()) {
    const trimmedLine = sourceLine.trim()
    if (!trimmedLine || trimmedLine.startsWith('#')) { continue }

    const sectionMatch = /^([a-z_]+):$/.exec(trimmedLine)
    if (sourceLine === trimmedLine && sectionMatch) {
      const sectionName = sectionMatch[1] as keyof IAccessListCatalogPayload
      if (!(sectionName in parsedCatalog)) {
        throw new Error(`Unsupported access-list section "${sectionName}" at line ${lineIndex + 1}`)
      }
      activeRecords = parsedCatalog[sectionName] as unknown as Record<string, string | number>[]
      activeRecord = null
      continue
    }

    if (!activeRecords) {
      throw new Error(`Access-list field found before a section at line ${lineIndex + 1}`)
    }

    const fieldLine = trimmedLine.startsWith('- ') ? trimmedLine.slice(2) : trimmedLine
    if (trimmedLine.startsWith('- ')) {
      activeRecord = {}
      activeRecords.push(activeRecord)
    }
    if (!activeRecord) {
      throw new Error(`Access-list field found before a record at line ${lineIndex + 1}`)
    }

    const fieldMatch = /^([a-z_]+):\s*(.*)$/.exec(fieldLine)
    if (!fieldMatch) {
      throw new Error(`Unsupported access-list YAML at line ${lineIndex + 1}: ${trimmedLine}`)
    }

    const [, fieldName, rawValue] = fieldMatch
    if (/^-?\d+$/.test(rawValue)) {
      activeRecord[fieldName] = Number(rawValue)
    } else if (rawValue.startsWith('"') && rawValue.endsWith('"')) {
      activeRecord[fieldName] = JSON.parse(rawValue)
    } else {
      throw new Error(`Access-list values must be integers or quoted strings at line ${lineIndex + 1}`)
    }
  }

  return parsedCatalog
}

// Normalize catalog routes once so every consumer uses the same matching rule.
export function normalizeAccessFrontendRoutes(frontendRoutes: string | string[] | undefined | null): string[] {
  const rawRoutes = Array.isArray(frontendRoutes) ? frontendRoutes : [frontendRoutes || ""]

  return rawRoutes
    .map((routeValue) => String(routeValue || "").trim().replace(/^\//, ""))
    .filter((routeValue) => routeValue.length > 0)
}

function indexAccessEntries(payload: IAccessListCatalogPayload): void {
  accessEntriesByRoute.clear()

  // One route can be unlocked by multiple access IDs.
  for (const accessEntry of payload.access_list || []) {
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
    accessListCatalog = parseAccessListCatalog(accessListYamlContent)
    indexAccessEntries(accessListCatalog)

    console.info('[access-list] Access catalog ready', {
      accessGroupCount: accessListCatalog.access_groups.length,
      accessEntryCount: accessListCatalog.access_list.length
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
  const normalizedRoute = String(routeValue || "").trim().replace(/^\//, "")
  return accessEntriesByRoute.get(normalizedRoute) || []
}
