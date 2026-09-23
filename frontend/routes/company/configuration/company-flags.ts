// The flag catalog is backend/company_flags.toml, imported as text and parsed here. There is no
// endpoint and no second copy: the backend embeds that same file to validate what this tab saves,
// so a flag added to it shows up on both sides at once.
import companyFlagsTomlContent from '../../../../backend/company_flags.toml?raw';

export interface ICompanyFlag {
  id: number
  // "English|Español", fed straight to T / tr like every other catalog string.
  name: string
  // Set only on a flag the catalog declares with a `type`: it holds a number instead of an
  // on/off, and `decimals` is the scale it is stored at — `type = "i:3"` stores 2.5 as 2500,
  // which is exactly what Input's `baseDecimals` does, so the panel hands this straight to it.
  decimals?: number
}

// One valued flag's number on the company record. Mirrors core.CompanyFlagValue field for field:
// the POST body is unmarshalled straight into it.
export interface ICompanyFlagValue {
  ID: number
  // Already scaled by the flag's decimals. Nothing outside this file and Input's baseDecimals
  // should read it as the number the user typed.
  //
  // Undefined, not zero, while the company has set nothing: Input renders a number it is given
  // as text, so a zero here would draw a "0" the user never typed over a field that is empty.
  // storedCompanyFlagValues drops it either way, and the backend drops a zero too.
  Value?: number
}

export interface ICompanyFlagSection {
  label: string
  flags: ICompanyFlag[]
}

// Parses the controlled shape of company_flags.toml — `[section]`, a `label`, and one
// `<id> = { name = "..." }` per flag. Hand-written for the same reason the access catalog's parser
// is: the file arrives as text and must not drag a TOML parser into the bundle.
export function parseCompanyFlagsCatalog(tomlContent: string): ICompanyFlagSection[] {
  const parsedSections: ICompanyFlagSection[] = []
  let activeSection: ICompanyFlagSection | null = null

  for (const [lineIndex, sourceLine] of tomlContent.split(/\r?\n/).entries()) {
    const trimmedLine = sourceLine.trim()
    if (!trimmedLine || trimmedLine.startsWith('#')) { continue }

    const sectionMatch = /^\[([a-z_]+)\]$/.exec(trimmedLine)
    if (sectionMatch) {
      activeSection = { label: sectionMatch[1], flags: [] }
      parsedSections.push(activeSection)
      continue
    }
    if (!activeSection) {
      throw new Error(`Company-flags entry found before a [section] at line ${lineIndex + 1}`)
    }

    const labelMatch = /^label\s*=\s*"(.*)"$/.exec(trimmedLine)
    if (labelMatch) {
      activeSection.label = labelMatch[1]
      continue
    }

    const flagMatch = FLAG_LINE_PATTERN.exec(trimmedLine)
    if (!flagMatch) {
      throw new Error(`Unsupported company-flags TOML at line ${lineIndex + 1}: ${trimmedLine}`)
    }
    const [, flagID, flagName, declaredType, declaredBytes] = flagMatch
    const flag: ICompanyFlag = { id: Number(flagID), name: flagName }
    if (declaredType) {
      // Refused rather than ignored, for the same reason the backend loader refuses it: a type
      // this parser cannot scale would be drawn as a plain field and save the wrong number.
      flag.decimals = parseCompanyFlagDecimals(declaredType, declaredBytes, lineIndex + 1)
    }
    activeSection.flags.push(flag)
  }

  return parsedSections
}

// `<id> = { name = "...", type = "i:3", bytes = 2 }` — `type` and `bytes` only on a valued flag.
const FLAG_LINE_PATTERN =
  /^(\d+)\s*=\s*\{\s*name\s*=\s*"([^"]*)"\s*(?:,\s*type\s*=\s*"([^"]*)"\s*)?(?:,\s*bytes\s*=\s*(\d+)\s*)?\}$/

// The catalog's only valued type is "i" or "i:<decimals>", and `bytes` (2 or 4) is mandatory
// beside it — it is the width the backend bounds a save against, so it is never inferred here.
function parseCompanyFlagDecimals(declaredType: string, declaredBytes: string, lineNumber: number): number {
  const [integerPart, decimalsPart] = declaredType.split(':')
  const decimals = decimalsPart === undefined ? 0 : Number(decimalsPart)
  if (integerPart !== 'i' || !Number.isInteger(decimals) || decimals < 0 || decimals > 9) {
    throw new Error(`Unsupported company-flag type "${declaredType}" at line ${lineNumber}`)
  }
  if (declaredBytes !== '2' && declaredBytes !== '4') {
    throw new Error(`A company flag with a type must declare bytes = 2 or 4, at line ${lineNumber}`)
  }
  return decimals
}

// A section with no flags declared yet would draw a heading over nothing, so it is dropped rather
// than rendered empty — the file keeps the section so the next flag has somewhere to go.
export const companyFlagsCatalog: ICompanyFlagSection[] =
  parseCompanyFlagsCatalog(companyFlagsTomlContent).filter(section => section.flags.length > 0)

// Only the checked ids are stored, so turning a flag off removes it instead of writing a zero.
// Sorted because the saved list is compared as a whole and a reordered array reads as a change.
export function toggleCompanyFlag(companyFlags: number[], flagID: number, isChecked: boolean): number[] {
  const remainingFlags = (companyFlags || []).filter(storedFlagID => storedFlagID !== flagID)
  if (!isChecked) { return remainingFlags }
  return [...remainingFlags, flagID].sort((a, b) => a - b)
}

export const valuedCompanyFlags: ICompanyFlag[] =
  companyFlagsCatalog.flatMap(section => section.flags).filter(flag => flag.decimals !== undefined)

// The record held in memory carries one entry per valued flag whether or not the company set it,
// because each `Input` writes into its entry object directly and only re-reads that object when
// its identity changes. Building the slots once, when the record arrives, is what makes the field
// show the stored number without an effect syncing two copies of it.
export function companyFlagValueSlots(storedValues: ICompanyFlagValue[]): ICompanyFlagValue[] {
  return valuedCompanyFlags.map(flag => ({
    ID: flag.id,
    Value: (storedValues || []).find(stored => stored.ID === flag.id)?.Value || undefined,
  }))
}

// The inverse, run on the way out: the empty slots are dropped so an untouched field is absent
// rather than stored as a zero — the backend refuses nothing here, it drops the same zeros, but
// sending them would make every company's record carry a value it never typed.
export function storedCompanyFlagValues(valueSlots: ICompanyFlagValue[]): ICompanyFlagValue[] {
  return (valueSlots || []).filter(slot => !!slot.Value)
}

// The number the company typed for a valued flag, unscaled by the flag's decimals; 0 when it set
// none. Mirrors core.GetCompanyFlagValue, so a rule never divides by the scale itself.
export function getCompanyFlagValue(storedValues: ICompanyFlagValue[] | undefined, flagID: number): number {
  const valuedFlag = valuedCompanyFlags.find((flag) => flag.id === flagID)
  const storedValue = (storedValues || []).find((flagValue) => flagValue.ID === flagID)?.Value || 0
  return valuedFlag ? storedValue / 10 ** (valuedFlag.decimals || 0) : 0
}
