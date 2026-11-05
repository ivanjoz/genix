import { describe, expect, it } from 'vitest'
import type { IConcept } from '#services/hr/payroll.svelte.ts'
import { DEDUCTION_ACCOUNT_OPTIONS, EARNING_ACCOUNT_OPTIONS, makeEmptyConcept, normalizeConcept } from './hr-settings.ts'

describe('concept accounts', () => {
  it('offers only postable 62x accounts to earnings', () => {
    expect(EARNING_ACCOUNT_OPTIONS.every((option) => String(option.code).startsWith('62'))).toBe(true)
    expect(EARNING_ACCOUNT_OPTIONS.map((option) => option.code)).toContain(622)
    expect(EARNING_ACCOUNT_OPTIONS.map((option) => option.code)).not.toContain(627)
  })
  it('offers payables and worker receivables to deductions', () => {
    const codes = DEDUCTION_ACCOUNT_OPTIONS.map((option) => option.code)
    expect(codes).toContain(419)
    expect(codes).toContain(1411)
    expect(codes).not.toContain(403)
  })
})

describe('normalizeConcept', () => {
  it('strips contributions from a non-remunerative earning', () => {
    const concept = normalizeConcept({ ...makeEmptyConcept(), IsRemunerative: false } as IConcept)
    expect([concept.AffectsEssalud, concept.AffectsPension, concept.ComputableForBenefits]).toEqual([false, false, false])
  })
  it('leaves a deduction only its payable account', () => {
    const concept = normalizeConcept({ ...makeEmptyConcept(), Kind: 2, PayableAccountCode: 419 } as IConcept)
    expect([concept.ExpenseAccountCode, concept.PayableAccountCode, concept.IncomeTax]).toEqual([0, 419, 0])
  })
})
