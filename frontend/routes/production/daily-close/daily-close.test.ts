import { describe, expect, it } from 'vitest'
import {
  canRunClose, closeTotalValue, formatSupplySubUnits, isMovementDateShifted, oldestPendingDay, supplyShortage,
  type IDailyClosePreview,
} from './daily-close.ts'

const makePreview = (overrides: Partial<IDailyClosePreview> = {}): IDailyClosePreview =>
  ({ WarehouseID: 1, SuppliesWarehouseID: 1, Date: 20400, MovementDate: 20400, PreviewHash: '123', ...overrides })

describe('daily close', () => {
  it('closes the oldest pending day first', () => {
    expect(oldestPendingDay([{ Date: 20401, SaleCount: 1, Units: 2 }, { Date: 20399, SaleCount: 3, Units: 5 }])?.Date).toBe(20399)
    expect(oldestPendingDay([])).toBeUndefined()
  })

  it('renders supply sub-units as the supply is counted', () => {
    expect(formatSupplySubUnits(2250, 1000, 'g')).toBe('2 + 250 g')
    expect(formatSupplySubUnits(750, 1000, 'g')).toBe('750 g')
    expect(formatSupplySubUnits(3, 1)).toBe('3')
    expect(formatSupplySubUnits(3, 0)).toBe('3')
  })

  it('measures what a supply lacks', () => {
    expect(supplyShortage({ SupplyID: 1, SubDivisor: 1000, Needed: 2250, Available: 1000, Consumed: 1000 })).toBe(1250)
    expect(supplyShortage({ SupplyID: 1, SubDivisor: 1000, Needed: 500, Available: 1000, Consumed: 500 })).toBe(0)
  })

  it('sums the products value', () => {
    expect(closeTotalValue([{ ProductID: 1, Units: 2, CostSource: 2, Value: 1850 }, { ProductID: 2, Units: 1, CostSource: 1, Value: 250 }])).toBe(2100)
    expect(closeTotalValue(undefined)).toBe(0)
  })

  it('runs only a preview without blocking reasons', () => {
    expect(canRunClose(makePreview())).toBe(true)
    expect(canRunClose(makePreview({ BlockingReasons: ['falta receta'] }))).toBe(false)
    expect(canRunClose(makePreview({ PreviewHash: '' }))).toBe(false)
    expect(canRunClose(undefined)).toBe(false)
  })

  it('warns when the movements are dated yesterday instead of the day', () => {
    expect(isMovementDateShifted(makePreview())).toBe(false)
    expect(isMovementDateShifted(makePreview({ Date: 20390, MovementDate: 20399 }))).toBe(true)
  })
})
