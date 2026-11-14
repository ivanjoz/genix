import { describe, expect, it } from 'vitest'
import type { IProductionOrder } from '#services/production/production-orders.ts'
import { isLotExpired, lotConsumingOrderIDs, traceLotBackward, traceLotForward } from './lot-traceability'

// Order 12 consumed flour lot 5 and sugar (free bucket) in receipt 1, which entered lot 40;
// receipt 2 consumed flour lot 6 and entered lot 41.
const order12 = {
  ID: 12,
  Receipts: [
    {
      N: 1, Date: 20400,
      Lines: [{ ProductID: 100, PresentationID: 0, Units: 30, Scrapped: 1, LotID: 40, Value: 900 }],
      Consumptions: [
        { SupplyID: 7, LotID: 5, SerialNumber: '', SubUnits: 2500, SubDivisor: 1000, Value: 500 },
        { SupplyID: 8, LotID: 0, SerialNumber: '', SubUnits: 300, SubDivisor: 1000, Value: 90 },
      ],
    },
    {
      N: 2, Date: 20401,
      Lines: [{ ProductID: 100, PresentationID: 0, Units: 10, Scrapped: 0, LotID: 41, Value: 300 }],
      Consumptions: [{ SupplyID: 7, LotID: 6, SerialNumber: '', SubUnits: 1000, SubDivisor: 1000, Value: 200 }],
    },
  ],
} as unknown as IProductionOrder

describe('lot traceability', () => {
  it('traces a produced lot back to what its receipt consumed', () => {
    expect(traceLotBackward(order12, 40)).toEqual([
      { OrderID: 12, ReceiptN: 1, ReceiptDate: 20400, ProductID: 7, LotID: 5, SubUnits: 2500, SubDivisor: 1000 },
      { OrderID: 12, ReceiptN: 1, ReceiptDate: 20400, ProductID: 8, LotID: 0, SubUnits: 300, SubDivisor: 1000 },
    ])
  })

  it('traces a supply lot forward to the lots its consuming receipts produced', () => {
    const movements = [
      { LotID: 5, Type: 3, DocumentID: 77 }, // its purchase reception
      { LotID: 5, Type: 15, DocumentID: 12 },
      { LotID: 5, Type: 15, DocumentID: 12 },
      { LotID: 9, Type: 15, DocumentID: 13 }, // another lot of the same name
    ]
    expect(lotConsumingOrderIDs(movements, 5)).toEqual([12])
    expect(traceLotForward([order12], 5)).toEqual([
      { OrderID: 12, ReceiptN: 1, ReceiptDate: 20400, ProductID: 100, LotID: 40, SubUnits: 30, SubDivisor: 1 },
    ])
  })

  it('warns an expired lot only once its expiry has passed', () => {
    expect(isLotExpired({ ID: 1, ExpirationDate: 20399 }, 20400)).toBe(true)
    expect(isLotExpired({ ID: 1, ExpirationDate: 20400 }, 20400)).toBe(false)
    expect(isLotExpired({ ID: 1, ExpirationDate: 0 }, 20400)).toBe(false)
  })
})
