import { GetHandler } from '#libs/ui-runtime.svelte.ts'
import {
  PRODUCTION_ORDER_KIND, type IProductionLaborEntry, type IProductionOrder, type IProductionReceiptRequest, type IProductionTemplate,
} from './production-orders.ts'

// The manual production orders (Kind 1), newest first. The route also syncs the daily closes (Kind 2,
// shown by the daily-close page); an annulled order leaves the delta as a removed ID.
export class ProductionOrdersService extends GetHandler<IProductionOrder> {
  route = 'production-orders'
  useCache = { min: 0.5, ver: 1 }

  handler(result: { records?: IProductionOrder[] }): void {
    const orders = (result?.records || []).filter((order) => order.ss && order.Kind === PRODUCTION_ORDER_KIND.MANUAL)
    orders.sort((leftOrder, rightOrder) => rightOrder.ID - leftOrder.ID)
    this.records = orders
    this.recordsMap = new Map(orders.map((order) => [order.ID, order]))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export class ProductionTemplatesService extends GetHandler<IProductionTemplate> {
  route = 'production-templates'
  useCache = { min: 5, ver: 1 }

  handler(result: IProductionTemplate[]): void {
    const templates = (result || []).filter((template) => template.ss === 1)
    templates.sort((leftTemplate, rightTemplate) => leftTemplate.Name.localeCompare(rightTemplate.Name))
    this.records = templates
    this.recordsMap = new Map(templates.map((template) => [template.ID, template]))
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

// The writes below answer the saved order and bring the orders list up to date from the delta the
// write carries (GetHandler.writeAndSync). Release, receipts, completion and annulment move the
// supplies' reservations or stock, so the stock list is marked for a fresh read too.
const STOCK_ROUTES = ['products-stock']

// Creates (ID ≤ 0) or edits a draft: its lines, warehouses, dates, stage and supply adjustment.
export const postProductionOrder = (orders: ProductionOrdersService, order: IProductionOrder) =>
  orders.writeAndSync<IProductionOrder>('production-order', order)

// Explodes the recipes, freezes them on the order and reserves its supplies. Refused, listing the
// shortages, when a supply is short.
export const postProductionOrderRelease = (orders: ProductionOrdersService, orderID: number) =>
  orders.writeAndSync<IProductionOrder>('production-order-release', { OrderID: orderID }, STOCK_ROUTES)

export const postProductionOrderReceipt = (orders: ProductionOrdersService, receipt: IProductionReceiptRequest) =>
  orders.writeAndSync<IProductionOrder>('production-order-receipt', receipt, STOCK_ROUTES)

// Finishes an order in progress without another receipt; what it still reserves is released.
export const postProductionOrderComplete = (orders: ProductionOrdersService, orderID: number) =>
  orders.writeAndSync<IProductionOrder>('production-order-complete', { OrderID: orderID }, STOCK_ROUTES)

// Annuls a draft or released order nothing was received of; its reservations are released.
export const postProductionOrderAnnul = (orders: ProductionOrdersService, orderID: number) =>
  orders.writeAndSync<IProductionOrder>('production-order-annul', { OrderID: orderID }, STOCK_ROUTES)

// Moves a released or in-progress order to another board column (0 = no stage). Information only.
export const postProductionOrderStage = (orders: ProductionOrdersService, orderID: number, stageID: number) =>
  orders.writeAndSync<IProductionOrder>('production-order-stage', { OrderID: orderID, StageID: stageID })

// Saves one template; ss = 0 deletes it.
export const postProductionTemplate = (templates: ProductionTemplatesService, template: IProductionTemplate) =>
  templates.writeAndSync<IProductionTemplate>('production-template', template)


// One order's labor entries, delta-synced: the order is part of the route, so each order keeps its
// own cache key and cursor.
export class ProductionLaborEntriesService extends GetHandler<IProductionLaborEntry> {
  route = 'production-labor-entries'
  useCache = { min: 0.2, ver: 1 }

  handler(result: { records?: IProductionLaborEntry[] }): void {
    const entries = (result?.records || []).filter((entry) => entry.ss === 1)
    entries.sort((leftEntry, rightEntry) => rightEntry.Date - leftEntry.Date || rightEntry.ID - leftEntry.ID)
    this.records = entries
    this.recordsMap = new Map(entries.map((entry) => [entry.ID, entry]))
  }

  constructor(orderID: number) {
    super()
    this.route += `?order-id=${orderID}`
    this.fetch()
  }
}

// Logs, edits or (ss = 0) deletes an entry; the order's recorded labor changes with it.
export const postProductionLaborEntry = async (orders: ProductionOrdersService, entries: ProductionLaborEntriesService,
  entry: IProductionLaborEntry) => {
  const savedEntry = await entries.writeAndSync<IProductionLaborEntry>('production-labor-entry', entry)
  await orders.fetchOnline()
  return savedEntry
}

// Sends a subcontracted order's supplies to its workshop, or (isReturn) brings the leftovers back.
// SubUnits at each order supply's divisor.
export const postProductionOrderMaterial = (orders: ProductionOrdersService, orderID: number, isReturn: boolean,
  lines: { SupplyID: number, SubUnits: number }[]) =>
  orders.writeAndSync<IProductionOrder>('production-order-material', { OrderID: orderID, IsReturn: isReturn, Lines: lines }, STOCK_ROUTES)
