import { GET, POST } from '#libs/ui-runtime.svelte.ts'
import type { ITableOrder } from '../dining.ts'
import type { IDiningMenuProduct, makeRoundRequest } from './floor-service.ts'
import type { ITableChargeRequest } from './table-charge.ts'

// What can be ordered now in a layout, priced in soles, with the units left of each presentation.
export const getDiningMenu = async (layoutID: number) => {
  const result = await GET({ route: `dining-menu?layout-id=${layoutID}` }) as { Products?: IDiningMenuProduct[] }
  return result?.Products || []
}

// Adds a round to a table, opening its order when it has none. The answer is the whole order.
export const postTableOrderRound = (request: ReturnType<typeof makeRoundRequest>) => {
  return POST({
    route: 'dining-table-order-round',
    data: request,
    refreshRoutes: ['dining-table-orders'],
  }) as Promise<ITableOrder>
}

// Closes the bill into a sale. The order comes back charged (ss 2), with its SaleOrderID.
export const postTableOrderCharge = (request: ITableChargeRequest) => {
  return POST({
    route: 'dining-table-order-charge',
    data: request,
    refreshRoutes: ['dining-table-orders'],
  }) as Promise<{ Order: ITableOrder, SaleOrder: { ID: number } }>
}

// Annuls an open bill, releasing its reservations. The order comes back annulled (ss 0).
export const postTableOrderAnnul = (request: { OrderID: number, Reason: string }) => {
  return POST({
    route: 'dining-table-order-annul',
    data: request,
    refreshRoutes: ['dining-table-orders'],
  }) as Promise<ITableOrder>
}
