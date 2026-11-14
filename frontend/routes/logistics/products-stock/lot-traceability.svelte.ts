import { getStaticRecordsByID } from '@genix/ui/cache'
import { StockMovementType } from '#core/stock-movement-type.ts'
import { GET } from '#libs/ui-runtime.svelte.ts'
import { queryAlmacenMovimientos } from '#routes/logistics/warehouse-movements/warehouse-movements.svelte.ts'
import type { IProductionOrder } from '#services/production/production-orders.ts'
import { lotConsumingOrderIDs, traceLotBackward, traceLotForward, type ILotTraceRow } from './lot-traceability'
import type { IProductStockLot } from './stock-movement'

export interface ILotTrace {
  backward: ILotTraceRow[]
  forward: ILotTraceRow[]
  // The names of the lots the rows name. Names never change, so the immutable by-IDs cache serves them.
  lotNamesByID: Map<number, string>
}

const getProductionOrder = async (orderID: number): Promise<IProductionOrder> =>
  await GET({ route: `production-order?id=${orderID}` }) as IProductionOrder

// Reads a lot's trace (docs/MANUFACTURING_PLAN.md §10). Backward: a produced lot's order receipt and
// what it consumed. Forward: the lot's consumption exits in the ledger, their orders, and what the
// receipts that consumed it produced.
export const loadLotTrace = async (lot: IProductStockLot): Promise<ILotTrace> => {
  const isProducedLot = lot.OriginMovementType === StockMovementType.PRODUCTION_ENTRY && (lot.OriginDocumentID || 0) > 0
  const [backward, lotMovements] = await Promise.all([
    isProducedLot ? getProductionOrder(lot.OriginDocumentID || 0).then((order) => traceLotBackward(order, lot.ID)) : [],
    // The ledger is searched by lot name, which several lots may share: the rows are kept by LotID.
    queryAlmacenMovimientos({ almacenID: 0, dateInicio: 0, dateFin: 0, lotCode: lot.Name || '' }),
  ])
  const consumingOrders = await Promise.all(lotConsumingOrderIDs(lotMovements, lot.ID).map(getProductionOrder))
  const forward = traceLotForward(consumingOrders, lot.ID)

  const tracedLotIDs = [...backward, ...forward].map((row) => row.LotID).filter((lotID) => lotID > 0)
  const tracedLots = await getStaticRecordsByID<IProductStockLot>('product-stock-lots-by-ids', tracedLotIDs)
  const lotNamesByID = new Map([...tracedLots.values()].map((tracedLot) => [tracedLot.ID, tracedLot.Name || '']))
  return { backward, forward, lotNamesByID }
}
