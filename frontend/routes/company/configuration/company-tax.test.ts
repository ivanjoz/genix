import { describe, expect, it } from 'vitest'
import {
  defaultIncomeTaxPrepaymentRate, TAX_REGIME_GENERAL, TAX_REGIME_MYPE, TAX_REGIME_NRUS, TAX_REGIME_SPECIAL,
} from './company-tax.ts'

describe('company tax setup', () => {
  it('starts each regime from its renta rate', () => {
    expect(defaultIncomeTaxPrepaymentRate(TAX_REGIME_GENERAL)).toBe(150)
    expect(defaultIncomeTaxPrepaymentRate(TAX_REGIME_MYPE)).toBe(100)
    expect(defaultIncomeTaxPrepaymentRate(TAX_REGIME_SPECIAL)).toBe(150)
    expect(defaultIncomeTaxPrepaymentRate(TAX_REGIME_NRUS)).toBe(0)
    expect(defaultIncomeTaxPrepaymentRate(0)).toBe(0)
  })
})
