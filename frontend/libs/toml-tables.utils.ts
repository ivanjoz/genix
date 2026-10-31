// Parser for the controlled TOML shape of the catalogs under db/ (access.toml, pages.toml,
// modules.toml): arrays of tables holding scalar or single-line-array fields. Hand-written rather
// than pulled from a library because the catalogs are imported as text and must not drag a parser
// into the bundle.
//
// A dotted section (`[[modules.menus]]`) is a child array of tables: it is appended to the closest
// record of its parent section above it, exactly as TOML reads it. Indentation is ignored.

export type TomlTableValue = string | number | boolean | (string | number)[] | TomlTableRecord[]
export type TomlTableRecord = { [fieldName: string]: TomlTableValue }

export function parseTomlTables(
  tomlContent: string,
  // Every section the caller accepts, dotted children included. Anything else is a typo.
  allowedSections: string[],
  // Names the file in error messages, e.g. "access-catalog".
  catalogLabel: string,
): Record<string, TomlTableRecord[]> {
  const parsedTables: Record<string, TomlTableRecord[]> = {}
  for (const sectionName of allowedSections) {
    if (!sectionName.includes('.')) { parsedTables[sectionName] = [] }
  }
  // The last record opened in each section: a child section attaches to its parent's.
  const lastRecordBySection = new Map<string, TomlTableRecord>()
  let activeRecord: TomlTableRecord | null = null

  for (const [lineIndex, sourceLine] of tomlContent.split(/\r?\n/).entries()) {
    const trimmedLine = sourceLine.trim()
    if (!trimmedLine || trimmedLine.startsWith('#')) { continue }

    const sectionMatch = /^\[\[([a-z_.]+)\]\]$/.exec(trimmedLine)
    if (sectionMatch) {
      const sectionName = sectionMatch[1]
      if (!allowedSections.includes(sectionName)) {
        throw new Error(`Unsupported ${catalogLabel} section "${sectionName}" at line ${lineIndex + 1}`)
      }
      activeRecord = {}
      lastRecordBySection.set(sectionName, activeRecord)

      const parentSeparatorIndex = sectionName.lastIndexOf('.')
      if (parentSeparatorIndex < 0) {
        parsedTables[sectionName].push(activeRecord)
        continue
      }
      const parentRecord = lastRecordBySection.get(sectionName.slice(0, parentSeparatorIndex))
      if (!parentRecord) {
        throw new Error(`${catalogLabel} section "${sectionName}" comes before any parent record (line ${lineIndex + 1})`)
      }
      const childFieldName = sectionName.slice(parentSeparatorIndex + 1)
      const childRecords = (parentRecord[childFieldName] || []) as TomlTableRecord[]
      childRecords.push(activeRecord)
      parentRecord[childFieldName] = childRecords
      continue
    }

    const fieldMatch = /^([a-z_]+)\s*=\s*(.*)$/.exec(trimmedLine)
    if (!fieldMatch) {
      throw new Error(`Unsupported ${catalogLabel} TOML at line ${lineIndex + 1}: ${trimmedLine}`)
    }
    if (!activeRecord) {
      throw new Error(`${catalogLabel} field found before a [[section]] at line ${lineIndex + 1}`)
    }

    const [, fieldName, rawValue] = fieldMatch
    activeRecord[fieldName] = parseTomlValue(rawValue, catalogLabel, lineIndex + 1)
  }

  return parsedTables
}

// A single-line TOML array of integers or double-quoted strings is already valid JSON, so the whole
// value parser is one JSON.parse plus the two TOML-only spellings it would choke on.
function parseTomlValue(rawValue: string, catalogLabel: string, lineNumber: number): string | number | boolean | (string | number)[] {
  if (rawValue.startsWith('[')) {
    if (!rawValue.endsWith(']')) {
      throw new Error(`${catalogLabel} arrays must fit on one line (line ${lineNumber})`)
    }
    if (/,\s*\]$/.test(rawValue)) {
      throw new Error(`${catalogLabel} arrays must not carry a trailing comma (line ${lineNumber})`)
    }
    return JSON.parse(rawValue)
  }
  if (rawValue.startsWith('"')) { return JSON.parse(rawValue) }
  if (rawValue === 'true' || rawValue === 'false') { return rawValue === 'true' }
  if (/^-?\d+$/.test(rawValue)) { return Number(rawValue) }
  throw new Error(`${catalogLabel} values must be integers, booleans, quoted strings or arrays (line ${lineNumber})`)
}

// Splits a catalog's compact comma list ("a/b, /c/d") into routes without the leading "/".
export function splitCatalogRoutes(routeList: string | string[] | undefined | null): string[] {
  const rawRoutes = Array.isArray(routeList) ? routeList : [routeList || ""]

  return rawRoutes
    .flatMap((routeValue) => String(routeValue || "").split(','))
    .map((routeValue) => routeValue.trim().replace(/^\//, ""))
    .filter((routeValue) => routeValue.length > 0)
}
