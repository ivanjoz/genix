import { describe, expect, it } from 'vitest'
import { companyFlagsCatalog, parseCompanyFlagsCatalog, toggleCompanyFlag } from './company-flags'

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
