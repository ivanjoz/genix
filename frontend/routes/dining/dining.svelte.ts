import { Env } from '#core/env.ts'
import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'
import type { IDiningLayout, ITableOrder } from './dining.ts'

// The layout this device serves (a tablet stays in its room), remembered across reloads. Per
// company, as layout IDs are.
const servedLayoutStorageKey = () => `dining-served-layout-${Env.getCompanyID()}`
export const readServedLayoutID = () => Number(localStorage.getItem(servedLayoutStorageKey())) || 0
export const saveServedLayoutID = (layoutID: number) => localStorage.setItem(servedLayoutStorageKey(), String(layoutID))

// The company's active layouts, shared by every dining page.
export class DiningLayoutsService extends GetHandler<IDiningLayout> {
  route = 'dining-layouts'
  // ver 2: shapes moved to Points.
  useCache = { min: 0.2, ver: 2 }

  handler(result: { records?: IDiningLayout[] }): void {
    const layouts = (result?.records || []).filter((layout) => layout.ss)
    layouts.sort((leftLayout, rightLayout) => leftLayout.Name.localeCompare(rightLayout.Name))
    this.records = layouts
    this.recordsMap = new Map(layouts.map((layout) => [layout.ID, layout]))
  }

  constructor() {
    super()
    this.fetch()
  }
}

// A layout's open table orders, delta-synced: the layout is part of the route, so each layout keeps
// its own cache key and cursor. A charged or annulled order arrives as a removed ID.
export class TableOrdersService extends GetHandler<ITableOrder> {
  route = 'dining-table-orders'
  useCache = { min: 0.05, ver: 1 }

  handler(result: { records?: ITableOrder[] }): void {
    const orders = (result?.records || []).filter((order) => order.ss === 1)
    this.records = orders
    this.recordsMap = new Map(orders.map((order) => [order.ID, order]))
  }

  constructor(layoutID: number) {
    super()
    this.route += `?layout-id=${layoutID}`
    this.fetch()
  }

  // A write answers the whole order: it replaces the cached one at once (a closed one leaves), and the sync follows.
  applyChangedOrder(order: ITableOrder) {
    const others = this.records.filter((candidate) => candidate.ID !== order.ID)
    this.records = order.ss === 1 ? [...others, order] : others
    this.fetchOnline()
  }
}

// Moves lines of an open order to a new LINE_STATUS (confirm, ready, served, cancel, back to the kitchen).
export const postTableOrderLines = (request: { OrderID: number, LineIDs: number[], Status: number }) => {
  return POST({
    route: 'dining-table-order-lines',
    data: request,
    refreshRoutes: ['dining-table-orders'],
  }) as Promise<ITableOrder>
}

// Saves the whole plan; the answer carries the IDs the backend gave the new shapes. ss = 0 deletes it.
export const postDiningLayout = (layout: IDiningLayout) => {
  return POST({
    route: 'dining-layout',
    data: layout,
    refreshRoutes: ['dining-layouts'],
  }) as Promise<IDiningLayout>
}
