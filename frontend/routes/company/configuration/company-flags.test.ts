import { describe, expect, it } from 'vitest'
import {
  companyFlagValueSlots, companyFlagsCatalog, getCompanyFlagValue, parseCompanyFlagsCatalog,
  storedCompanyFlagValues, toggleCompanyFlag, valuedCompanyFlags,
} from './company-flags'

describe('parseCompanyFlagsCatalog', () => {
  it('reads sections, their label and their flags', () => {
    const sections = parseCompanyFlagsCatalog([
      '# a comment',
      '[comercial]',
      'label = "Commercial|Comercial"',
      '1 = { name = "Force full payment|Forzar pago completo" }',
      '',
      '[logistics]',
      'label = "Logistics|Logística"',
    ].join('\n'))

    expect(sections).toHaveLength(2)
    expect(sections[0].label).toBe('Commercial|Comercial')
    expect(sections[0].flags).toEqual([{ id: 1, name: 'Force full payment|Forzar pago completo' }])
    expect(sections[1].flags).toEqual([])
  })

  it('refuses a line it does not understand instead of dropping a flag silently', () => {
    expect(() => parseCompanyFlagsCatalog('[comercial]\n1 = "no table"')).toThrow()
  })

  it('reads the decimals of a valued flag and leaves a checkbox without any', () => {
    const [section] = parseCompanyFlagsCatalog([
      '[comercial]',
      'label = "Commercial|Comercial"',
      '1 = { name = "Plain|Simple" }',
      '7 = { name = "Spread|Spread", type = "i:3", bytes = 2 }',
    ].join('\n'))

    expect(section.flags[0].decimals).toBeUndefined()
    expect(section.flags[1].decimals).toBe(3)
  })

  it('refuses a valued flag whose type or width it cannot honour', () => {
    const declare = (entry: string) => () => parseCompanyFlagsCatalog(`[comercial]\n7 = ${entry}`)
    expect(declare('{ name = "x", type = "i:3" }')).toThrow()
    expect(declare('{ name = "x", type = "i:3", bytes = 3 }')).toThrow()
    expect(declare('{ name = "x", type = "f:3", bytes = 2 }')).toThrow()
  })
})

describe('company flag values', () => {
  // Flag 7 of the real catalog is the valued one, so this also guards the file against an edit
  // that would make the panel draw a field the backend refuses to store.
  it('the real catalog declares its valued flags with decimals', () => {
    expect(valuedCompanyFlags.length).toBeGreaterThan(0)
    expect(valuedCompanyFlags.every(flag => typeof flag.decimals === 'number')).toBe(true)
  })

  it('makes one slot per valued flag, carrying the stored number', () => {
    const storedFlagID = valuedCompanyFlags[0].id
    const slots = companyFlagValueSlots([{ ID: storedFlagID, Value: 2500 }])

    expect(slots).toHaveLength(valuedCompanyFlags.length)
    expect(slots.find(slot => slot.ID === storedFlagID)?.Value).toBe(2500)
  })

  // Empty and not zero: Input draws a number it is handed, so a zero would show a "0" over a
  // field the company never filled.
  it('gives an untouched flag an empty slot instead of leaving the input unbound', () => {
    const slots = companyFlagValueSlots([])
    expect(slots).toHaveLength(valuedCompanyFlags.length)
    expect(slots.every(slot => slot.Value === undefined)).toBe(true)
  })

  it('drops the empty slots on the way out so a zero is never stored', () => {
    const storedFlagID = valuedCompanyFlags[0].id
    const slots = companyFlagValueSlots([{ ID: storedFlagID, Value: 2500 }])

    expect(storedCompanyFlagValues(slots)).toEqual([{ ID: storedFlagID, Value: 2500 }])
    expect(storedCompanyFlagValues(companyFlagValueSlots([]))).toEqual([])
  })

  it('reads a stored value back as the number that was typed', () => {
    const valuedFlag = valuedCompanyFlags[0]
    const storedValue = 25 * 10 ** (valuedFlag.decimals || 0)
    expect(getCompanyFlagValue([{ ID: valuedFlag.id, Value: storedValue }], valuedFlag.id)).toBe(25)
    expect(getCompanyFlagValue([], valuedFlag.id)).toBe(0)
  })
})

describe('companyFlagsCatalog', () => {
  // Parses the real backend/company_flags.toml, so a malformed edit to it fails here.
  it('carries the commercial flags and drops the sections with none', () => {
    expect(companyFlagsCatalog.length).toBeGreaterThan(0)
    expect(companyFlagsCatalog.every(section => section.flags.length > 0)).toBe(true)

    const allFlagIDs = companyFlagsCatalog.flatMap(section => section.flags.map(flag => flag.id))
    expect(new Set(allFlagIDs).size).toBe(allFlagIDs.length)
    expect(companyFlagsCatalog[0].flags[0].name).toContain('|')
  })
})

describe('toggleCompanyFlag', () => {
  it('adds a checked flag in id order', () => {
    expect(toggleCompanyFlag([3, 1], 2, true)).toEqual([1, 2, 3])
  })

  it('removes an unchecked flag rather than storing a zero', () => {
    expect(toggleCompanyFlag([1, 2, 3], 2, false)).toEqual([1, 3])
  })

  it('never stores the same flag twice', () => {
    expect(toggleCompanyFlag([1, 2], 2, true)).toEqual([1, 2])
  })

  it('handles a company that has no flags field yet', () => {
    expect(toggleCompanyFlag(undefined as unknown as number[], 4, true)).toEqual([4])
  })
})
