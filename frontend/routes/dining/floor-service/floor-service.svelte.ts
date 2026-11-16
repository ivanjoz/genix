import { GET, POST } from '#libs/ui-runtime.svelte.ts'
import type { ITableOrder } from '../dining.ts'
import type { IDiningMenuProduct, IRoundDraftLine, makeRoundRequest } from './floor-service.ts'
import type { ITableChargeRequest } from './table-charge.ts'

// The round being composed for the selected table, shared by "Nuevo pedido" (RoundComposer, which
// restarts it each time it mounts) and the menu picker the page shows beside the table's layer.
export class RoundDraftState {
  menu = $state<IDiningMenuProduct[]>([])
  isMenuLoading = $state(false)
  draft = $state<IRoundDraftLine[]>([])
  // True while "Nuevo pedido" is on screen: the menu picker shows only then.
  isComposing = $state(false)
}

// What can be ordered now in a layout, priced in soles, with the units left of each presentation.
export const getDiningMenu = async (layoutID: number) => {
  const result = await GET({ route: `dining-menu?layout-id=${layoutID}` }) as { Products?: IDiningMenuProduct[] }
  return result?.Products || []
}

// The orders a table received since yesterday's midnight, newest first ("Historial").
export const getTableOrderHistory = async (layoutID: number, tableShapeID: number) => {
  const orders = await GET({ route: `dining-table-order-history?layout-id=${layoutID}&table-shape-id=${tableShapeID}` }) as ITableOrder[]
  return orders || []
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
