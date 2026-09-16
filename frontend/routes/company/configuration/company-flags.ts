// The flag catalog is backend/company_flags.toml, imported as text and parsed here. There is no
// endpoint and no second copy: the backend embeds that same file to validate what this tab saves,
// so a flag added to it shows up on both sides at once.
import companyFlagsTomlContent from '../../../../backend/company_flags.toml?raw';

export interface ICompanyFlag {
  id: number
  // "English|Español", fed straight to T / tr like every other catalog string.
  name: string
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

    const flagMatch = /^(\d+)\s*=\s*\{\s*name\s*=\s*"(.*)"\s*\}$/.exec(trimmedLine)
    if (!flagMatch) {
      throw new Error(`Unsupported company-flags TOML at line ${lineIndex + 1}: ${trimmedLine}`)
    }
    activeSection.flags.push({ id: Number(flagMatch[1]), name: flagMatch[2] })
  }

  return parsedSections
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
