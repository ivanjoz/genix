import { describe, expect, it } from 'vitest'
import { findManualStockRefusal, type IManualStockChange } from './stock-cost'

const change = (previousQuantity: number, nextQuantity: number, unitCost = 0): IManualStockChange =>
  ({ productName: 'Aceite', previousQuantity, nextQuantity, unitCost })

describe('findManualStockRefusal', () => {
  it('lets decreases through even when manual entries are blocked', () => {
    expect(findManualStockRefusal([change(10, 4), change(3, 0)], true, true)).toBeNull()
  })

  it('refuses any increase when manual entries are blocked', () => {
    expect(findManualStockRefusal([change(10, 12, 500)], true, false))
      .toEqual({ reason: 'inbound-blocked', productName: 'Aceite' })
  })

  it('refuses an increase with no cost when the cost is required', () => {
    expect(findManualStockRefusal([change(0, 5)], false, true))
      .toEqual({ reason: 'cost-missing', productName: 'Aceite' })
  })

  it('accepts an increase with its cost, and any increase when nothing is enforced', () => {
    expect(findManualStockRefusal([change(0, 5, 1250)], false, true)).toBeNull()
    expect(findManualStockRefusal([change(0, 5)], false, false)).toBeNull()
  })
})
