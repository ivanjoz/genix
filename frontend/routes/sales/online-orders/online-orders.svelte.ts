import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'
import type { ISaleOrder } from '#routes/sales/sale_orders_status/sale_order_status.svelte.ts'
import type { IOnlineSaleOrder, IOnlineSaleOrderAnnulRequest, IOnlineSaleOrderSettleRequest } from './online-orders.ts'

export interface IOnlineSaleOrderSettleResult {
  Order: IOnlineSaleOrder
  SaleOrder: ISaleOrder
}

// One delta-cached list per status group: the group is part of the route, so each tab keeps its
// own cache key and cursor.
export class OnlineSaleOrdersService extends GetHandler<IOnlineSaleOrder> {
  route = 'online-sale-orders'
  useCache = { min: 0.1, ver: 1 }

  handler(result: { records?: IOnlineSaleOrder[] }): void {
    // The backend reads only this group's statuses; orders that left it arrive as removed IDs.
    const records = [...(result?.records || [])]
    records.sort((leftOrder, rightOrder) => rightOrder.ID - leftOrder.ID)
    this.records = records
  }

  constructor(statusGroup: number) {
    super()
    this.route += `?status-group=${statusGroup}`
  }
}

// Creates the SaleOrder (paid, delivered or both). The order changes tab and a new sale appears
// in Gestión Ventas, so both cached lists re-sync.
export const postOnlineSaleOrderSettle = (request: IOnlineSaleOrderSettleRequest) => {
  return POST({
    route: 'online-sale-order-settle',
    data: request,
    refreshRoutes: ['online-sale-orders', 'sale-orders'],
  }) as Promise<IOnlineSaleOrderSettleResult>
}

export const postOnlineSaleOrderAnnul = (request: IOnlineSaleOrderAnnulRequest) => {
  return POST({
    route: 'online-sale-order-annul',
    data: request,
    refreshRoutes: ['online-sale-orders'],
  }) as Promise<IOnlineSaleOrder>
}
