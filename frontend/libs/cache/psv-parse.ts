// Parses a pipe-separated, ">>>"-section-delimited snapshot file into the multi-table response
// shape the delta cache consumes. The file has NO header row: a line ">>>name" opens a section
// whose name is the response key, and the schema for that key lists columns positionally as
// "FieldName:TYPE". Field names map straight to record fields, so columns named "upd"/"ss" feed
// the cache's watermark/eviction logic with no extra wiring.
//
// Supported type tokens:
//   T  text          N  number         O  JSON object
//   A  array (comma-split, raw strings)
//   AT array of text  AN array of number  AO array of JSON objects

export type PsvSchema = Record<string, string[]>

const splitArray = (raw: string): string[] => {
  return raw ? raw.split(",") : []
}

const coerceColumn = (raw: string, type: string): any => {
  switch(type){
    case "N":  return raw === "" ? 0 : Number(raw)
    case "O":  return raw ? JSON.parse(raw) : null
    case "A":
    case "AT": return splitArray(raw)
    case "AN": return splitArray(raw).map(Number)
    case "AO": return splitArray(raw).map((value) => JSON.parse(value))
    case "T":
    default:   return raw
  }
}

type ColumnSpec = { field: string, type: string }

const parseColumnSpecs = (specs: string[]): ColumnSpec[] => {
  return specs.map((spec) => {
    // Split on the last ":" so field names themselves can't be confused with the type token.
    const separator = spec.lastIndexOf(":")
    return separator === -1
      ? { field: spec, type: "T" }
      : { field: spec.slice(0, separator), type: spec.slice(separator + 1) }
  })
}

export const parsePsvResponse = (content: string, schema: PsvSchema): Record<string, any[]> => {
  const result: Record<string, any[]> = {}
  // Pre-create every declared section so a section missing from the file still yields an empty
  // array — keeps watermark extraction and the downstream merge well-defined.
  for(const sectionKey of Object.keys(schema)){ result[sectionKey] = [] }

  let columns: ColumnSpec[] | null = null
  let section = ""

  for(const rawLine of content.split("\n")){
    const line = rawLine.trim()
    if(!line){ continue }

    if(line.startsWith(">>>")){
      section = line.slice(3).trim()
      const sectionSchema = schema[section]
      // Unknown section → columns stays null and we skip its rows rather than guessing.
      columns = sectionSchema ? parseColumnSpecs(sectionSchema) : null
      if(section && !result[section]){ result[section] = [] }
      continue
    }

    if(!columns){ continue }
    const values = line.split("|")
    const record: Record<string, any> = {}
    for(let i = 0; i < columns.length; i++){
      record[columns[i].field] = coerceColumn(values[i] ?? "", columns[i].type)
    }
    result[section].push(record)
  }

  return result
}
