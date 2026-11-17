import { describe, expect, it } from 'vitest'
import { webPresentationIDs, webSellableStock } from './web-sellable.ts'

describe('the web sellable rule', () => {
  it('names presentation 0 for a product without active presentations', () => {
    expect(webPresentationIDs({ Presentations: [] })).toEqual([0])
    expect(webPresentationIDs({ Presentations: [{ id: 1, ss: 0 }, { id: 2, ss: 1 }] as any })).toEqual([2])
  })

  it('nets blocked and reserved units, and shares what the oversold presentations leave of the margin', () => {
    const sellable = webSellableStock([
      { PresentationID: 1, Quantity: 5, DetailQuantity: 3, BlockedQuantity: 1, PendingDeliveryStore: 2 },
      { PresentationID: 2, Quantity: 1, PendingDeliveryOnline: 3 },
      { PresentationID: 7, Quantity: 40 },
    ], [1, 2], 5)
    // 1: 5+3-1-2 = 5. 2: 1-3 = -2, oversold by 2, so 3 of the margin of 5 are left for each.
    expect([...sellable]).toEqual([[1, 8], [2, 3]])
  })
})
