import { describe, expect, it } from 'vitest'
import { openDetractionOf, remainingDebtAfterPayment } from './sale_order_status.ts'

describe('remainingDebtAfterPayment', () => {
  // A factura of 11800 with a 12 % detracción of 1416, 400 of it already deposited.
  const imposedTaxes = [{ Type: 10, Amount: 1416, SettledAmount: 400 }, { Type: 7, Amount: 500, Line: 1 }]

  it('counts only the detracción not deposited yet', () => {
    expect(openDetractionOf(imposedTaxes)).toBe(1016)
    expect(openDetractionOf(undefined)).toBe(0)
  })
  it('deposits the open detracción only into the detracciones account', () => {
    expect(remainingDebtAfterPayment(11416, imposedTaxes, true)).toBe(10400)
  })
  it('collects everything but the open detracción into any other account', () => {
    expect(remainingDebtAfterPayment(11416, imposedTaxes, false)).toBe(1016)
    expect(remainingDebtAfterPayment(500, imposedTaxes, false)).toBe(500)
    expect(remainingDebtAfterPayment(11800, [], false)).toBe(0)
  })
})
