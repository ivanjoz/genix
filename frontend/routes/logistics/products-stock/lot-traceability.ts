import { StockMovementType } from '#core/stock-movement-type.ts'
import type { IProductionOrder } from '#services/production/production-orders.ts'
import type { IProductStockLot } from './stock-movement'

// A lot's quality status and its traceability through production (docs/MANUFACTURING_PLAN.md §10).
// No new table: the lot names its origin, a production order's receipts name the lots they consumed
// and produced, and the ledger names the orders that consumed a lot.

// Mirrors the LotQuality* values of backend/logistics/types/product-stock.go.
export const LOT_QUALITY = { RELEASED: 1, QUARANTINE: 2, REJECTED: 3 } as const

export const lotQualityOptions = [
  { id: LOT_QUALITY.RELEASED, name: 'Released|Liberado', css: 'text-green-fg' },
  { id: LOT_QUALITY.QUARANTINE, name: 'Quarantine|Cuarentena', css: 'text-orange-fg' },
  { id: LOT_QUALITY.REJECTED, name: 'Rejected|Rechazado', css: 'text-red-fg' },
]

// An expired lot is warned, not blocked (M23): it shows red and production consumes it last.
export const isLotExpired = (lot: IProductStockLot | undefined, today: number): boolean =>
  (lot?.ExpirationDate || 0) > 0 && (lot?.ExpirationDate || 0) < today

// One step of a lot's trace: what a production order's receipt took (backward) or made (forward).
export interface ILotTraceRow {
  OrderID: number
  ReceiptN: number
  ReceiptDate: number
  ProductID: number
  LotID: number
  // Sub-units of SubDivisor: a consumption counts in the supply's sub-unit, a made line in whole units.
  SubUnits: number
  SubDivisor: number
}

// Backward: what the receipt that produced the lot consumed, each supply with the lot it came from.
export const traceLotBackward = (originOrder: IProductionOrder, lotID: number): ILotTraceRow[] =>
  (originOrder.Receipts || [])
    .filter((receipt) => (receipt.Lines || []).some((line) => line.LotID === lotID))
    .flatMap((receipt) => (receipt.Consumptions || []).map((consumption) => ({
      OrderID: originOrder.ID, ReceiptN: receipt.N, ReceiptDate: receipt.Date,
      ProductID: consumption.SupplyID, LotID: consumption.LotID || 0,
      SubUnits: consumption.SubUnits, SubDivisor: consumption.SubDivisor || 1,
    })))

// The production orders whose consumption exits (type 15) took from the lot, per its ledger rows.
export const lotConsumingOrderIDs = (lotMovements: { LotID?: number, Type?: number, DocumentID?: number }[], lotID: number): number[] =>
  [...new Set(lotMovements
    .filter((movement) => movement.LotID === lotID && movement.Type === StockMovementType.PRODUCTION_CONSUMPTION)
    .map((movement) => movement.DocumentID || 0)
    .filter((orderID) => orderID > 0))]

// Forward: what the receipts that consumed the lot produced, each with the lot it entered (0 when the
// product tracks no lots, or the production was a daily close).
export const traceLotForward = (consumingOrders: IProductionOrder[], lotID: number): ILotTraceRow[] =>
  consumingOrders.flatMap((order) => (order.Receipts || [])
    .filter((receipt) => (receipt.Consumptions || []).some((consumption) => consumption.LotID === lotID))
    .flatMap((receipt) => (receipt.Lines || []).filter((line) => line.Units > 0).map((line) => ({
      OrderID: order.ID, ReceiptN: receipt.N, ReceiptDate: receipt.Date,
      ProductID: line.ProductID, LotID: line.LotID || 0, SubUnits: line.Units, SubDivisor: 1,
    }))))
