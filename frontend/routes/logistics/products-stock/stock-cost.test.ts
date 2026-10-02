import { describe, expect, it } from 'vitest'
import { StockMovementType } from '#core/stock-movement-type.ts'
import { findManualStockRefusal, manualStockChangeType, type IManualStockChange } from './stock-cost'

const change = (previousQuantity: number, nextQuantity: number, unitCost = 0): IManualStockChange =>
  ({ productName: 'Aceite', detailName: '', previousQuantity, nextQuantity, unitCost })

const { MANUAL_ENTRY, OPENING_STOCK, SHRINKAGE } = StockMovementType

describe('findManualStockRefusal', () => {
  it('lets decreases through even when manual entries are blocked', () => {
    expect(findManualStockRefusal([change(10, 4), change(3, 0)], true, true, MANUAL_ENTRY)).toBeNull()
  })

  it('refuses any increase when manual entries are blocked', () => {
    expect(findManualStockRefusal([change(10, 12, 500)], true, false, MANUAL_ENTRY))
      .toEqual({ reason: 'inbound-blocked', productName: 'Aceite' })
  })

  it('refuses an increase with no cost when the cost is required', () => {
    expect(findManualStockRefusal([change(0, 5)], false, true, MANUAL_ENTRY))
      .toEqual({ reason: 'cost-missing', productName: 'Aceite' })
  })

  it('accepts an increase with its cost, and any increase when nothing is enforced', () => {
    expect(findManualStockRefusal([change(0, 5, 1250)], false, true, MANUAL_ENTRY)).toBeNull()
    expect(findManualStockRefusal([change(0, 5)], false, false, MANUAL_ENTRY)).toBeNull()
  })

  it('lets opening stock past the block, but never without its cost', () => {
    expect(findManualStockRefusal([change(0, 5, 900)], true, false, OPENING_STOCK)).toBeNull()
    expect(findManualStockRefusal([change(0, 5)], false, false, OPENING_STOCK))
      .toEqual({ reason: 'cost-missing', productName: 'Aceite' })
  })
})

describe('manualStockChangeType', () => {
  it('gives each change the reason chosen for its direction', () => {
    expect(manualStockChangeType(change(2, 8), OPENING_STOCK, SHRINKAGE)).toBe(OPENING_STOCK)
    expect(manualStockChangeType(change(8, 2), OPENING_STOCK, SHRINKAGE)).toBe(SHRINKAGE)
  })
})
