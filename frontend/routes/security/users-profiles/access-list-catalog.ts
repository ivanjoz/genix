import accessCatalogTomlContent from '../../../../backend/access.toml?raw';

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

type CatalogRecord = Record<string, string | number | (string | number)[]>

// Parse only the controlled access-catalog TOML shape: arrays of tables holding scalar or
// single-line-array fields. Hand-written rather than pulled from a library because the catalog is
// imported as text and must not drag a parser into the bundle.
function parseAccessCatalog(tomlContent: string): IAccessListCatalogPayload {
  const parsedCatalog: IAccessListCatalogPayload = { groups: [], access: [] }
  let activeRecord: CatalogRecord | null = null

  for (const [lineIndex, sourceLine] of tomlContent.split(/\r?\n/).entries()) {
    const trimmedLine = sourceLine.trim()
    if (!trimmedLine || trimmedLine.startsWith('#')) { continue }

    // `[[name]]` is unambiguous, so a record needs no indentation tracking to delimit it.
    const sectionMatch = /^\[\[([a-z_]+)\]\]$/.exec(trimmedLine)
    if (sectionMatch) {
      const sectionName = sectionMatch[1] as keyof IAccessListCatalogPayload
      if (!(sectionName in parsedCatalog)) {
        throw new Error(`Unsupported access-catalog section "${sectionName}" at line ${lineIndex + 1}`)
      }
      activeRecord = {}
      ;(parsedCatalog[sectionName] as unknown as CatalogRecord[]).push(activeRecord)
      continue
    }

    const fieldMatch = /^([a-z_]+)\s*=\s*(.*)$/.exec(trimmedLine)
    if (!fieldMatch) {
      throw new Error(`Unsupported access-catalog TOML at line ${lineIndex + 1}: ${trimmedLine}`)
    }
    if (!activeRecord) {
      throw new Error(`Access-catalog field found before a [[section]] at line ${lineIndex + 1}`)
    }

    const [, fieldName, rawValue] = fieldMatch
    activeRecord[fieldName] = parseCatalogValue(rawValue, lineIndex + 1)
  }

  return parsedCatalog
}

// A single-line TOML array of integers or double-quoted strings is already valid JSON, so the whole
// value parser is one JSON.parse plus the two TOML-only spellings it would choke on.
function parseCatalogValue(rawValue: string, lineNumber: number): string | number | (string | number)[] {
  if (rawValue.startsWith('[')) {
    if (!rawValue.endsWith(']')) {
      throw new Error(`Access-catalog arrays must fit on one line (line ${lineNumber})`)
    }
    if (/,\s*\]$/.test(rawValue)) {
      throw new Error(`Access-catalog arrays must not carry a trailing comma (line ${lineNumber})`)
    }
    return JSON.parse(rawValue)
  }
  if (rawValue.startsWith('"')) { return JSON.parse(rawValue) }
  if (/^-?\d+$/.test(rawValue)) { return Number(rawValue) }
  throw new Error(`Access-catalog values must be integers, quoted strings or arrays (line ${lineNumber})`)
}

// Split the catalog's compact comma list once so every consumer uses the same routes.
export function normalizeAccessFrontendRoutes(frontendRoutes: string | string[] | undefined | null): string[] {
  const rawRoutes = Array.isArray(frontendRoutes) ? frontendRoutes : [frontendRoutes || ""]

  return rawRoutes
    .flatMap((routeValue) => String(routeValue || "").split(','))
    .map((routeValue) => routeValue.trim().replace(/^\//, ""))
    .filter((routeValue) => routeValue.length > 0)
}

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
