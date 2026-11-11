import { describe, expect, it } from 'vitest'
import { aggregateDailyProductMovements } from './supply-management'

describe('aggregateDailyProductMovements', () => {
  it('splits each day and product into whole-unit inflows and outflows', () => {
    const dayRows = aggregateDailyProductMovements([
      { ID: 1, Date: 20001, ProductID: 7, Type: 3, Quantity: 10 },
      { ID: 2, Date: 20001, ProductID: 7, Type: 1, Quantity: -4 },
      { ID: 3, Date: 20001, ProductID: 7, Type: 1, Quantity: -2 },
      { ID: 4, Date: 20000, ProductID: 8, Type: 1, Quantity: -1 },
    ])
    expect(dayRows).toEqual([
      { Date: 20000, DetailProductsIDs: [8], DetailInflows: [0], DetailOutflows: [1] },
      { Date: 20001, DetailProductsIDs: [7], DetailInflows: [10], DetailOutflows: [6] },
    ])
  })

  it('folds sub-units within one divisor and truncates them to whole units', () => {
    const dayRows = aggregateDailyProductMovements([
      // 0 units + 9 sixths and 0 units + 4 sixths sold: 13 sixths = 2 whole units out.
      { ID: 1, Date: 20000, ProductID: 7, Type: 1, Quantity: 0, SubQuantity: -9, SubDivisor: 6 },
      { ID: 2, Date: 20000, ProductID: 7, Type: 1, Quantity: 0, SubQuantity: -4, SubDivisor: 6 },
      // A different divisor is summed apart: 5 twelfths truncate to 0.
      { ID: 3, Date: 20000, ProductID: 7, Type: 1, Quantity: 0, SubQuantity: -5, SubDivisor: 12 },
    ])
    expect(dayRows).toEqual([
      { Date: 20000, DetailProductsIDs: [7], DetailInflows: [0], DetailOutflows: [2] },
    ])
  })
})
