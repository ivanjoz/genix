import { QUANTITY_DIVISOR_NONE, QUANTITY_LINE_SCALE, packQuantityLine } from '#core/quantity.ts'
import { webPresentationIDs, webSellableStock } from '#core/web-sellable.ts'
import type { IProductStock } from '#routes/logistics/products-stock/stock-movement.ts'
import { restaurantPriceInSoles } from '#core/dining-menu.ts'
import { PRODUCT_CHANNEL_RESTAURANT, isProductSoldIn } from '#core/product-channels.ts'
import type { IProduct } from '#services/production/products.svelte.ts'
import { LINE_STATUS, SHAPE_KIND, TABLE_ORDER_STATUS, TABLE_STATE, lineAmount, tableLabelOf, tableState, type IDiningLayout, type ITableOrder, type ITableOrderLine } from '../dining.ts'

// The floor-service page's rules (docs/DINING_PLAN.md §2, §5): what a table owes, the dishes to
// deliver and the round being composed. Pure, so they are tested without a component or a request.

// One product of a layout's menu, priced in soles.
export interface IDiningMenuProduct {
  ID: number
  Name: string
  Price: number
  Presentations: { ID?: number, Name?: string, Sellable: number }[]
}

// The layout's menu (§3), built from the delta-cached catalog and stock so it paints from IndexedDB
// at once: the products sold in the restaurant that can be ordered now, with what the web store's
// sellable rule leaves of each presentation on the layout's warehouse alone. A USD-priced product
// needs penRate (the effective rate into soles); without one it cannot be ordered and is left out.
// The server checks each round with the same rules (dining/table_orders.go), so a stale cache can
// only get a round refused, never booked wrong.
export function diningMenu(products: IProduct[], stockRows: IProductStock[], warehouseID: number, penRate: number): IDiningMenuProduct[] {
  const stockRowsByProduct = new Map<number, IProductStock[]>()
  for (const stockRow of stockRows) {
    if (stockRow.WarehouseID !== warehouseID) continue
    stockRowsByProduct.set(stockRow.ProductID, [...(stockRowsByProduct.get(stockRow.ProductID) || []), stockRow])
  }
  const menu: IDiningMenuProduct[] = []
  for (const product of products) {
    if (!isProductSoldIn(product, PRODUCT_CHANNEL_RESTAURANT)) continue
    const priceInSoles = restaurantPriceInSoles(product, penRate)
    if (priceInSoles === null) continue
    const presentationIDs = webPresentationIDs(product)
    const sellableByPresentation = webSellableStock(stockRowsByProduct.get(product.ID) || [], presentationIDs, product.WebExtraStock)
    // Only what can be ordered now: a round of it would be refused.
    if (!presentationIDs.some((presentationID) => sellableByPresentation.get(presentationID)! > 0)) continue
    menu.push({
      ID: product.ID, Name: product.Name,
      Price: priceInSoles,
      Presentations: presentationIDs.map((presentationID) => ({
        ID: presentationID, Sellable: sellableByPresentation.get(presentationID)!,
        Name: product.Presentations?.find((presentation) => presentation.id === presentationID)?.nm || '',
      })),
    })
  }
  return menu
}

// What the table owes so far: every line not cancelled, at the price it was ordered at.
export const orderTotal = (order: ITableOrder) =>
  order.Lines.filter((line) => line.Status !== LINE_STATUS.CANCELLED).reduce((total, line) => total + lineAmount(line), 0)

export interface IBillLine { key: string, ProductID: number, Quantity: number, SubDivisor: number, Price: number, SubPrice: number }

// The bill as "Pagar" lists it, PoS style: one line per product, presentation and price across every
// round, cancelled lines left out. Packed quantities add up (units × scale + sub). Its amounts come
// from the pack engine (packGroupRows), which prices it as the charge will.
export function billLines(order: ITableOrder): IBillLine[] {
  const billLineByKey = new Map<string, IBillLine>()
  for (const line of order.Lines.filter((candidate) => candidate.Status !== LINE_STATUS.CANCELLED)) {
    const key = `${line.ProductID}_${line.PresentationID || 0}_${line.Price}_${line.SubPrice || 0}`
    const billLine = billLineByKey.get(key)
    if (billLine) {
      billLine.Quantity += line.Quantity
    } else {
      billLineByKey.set(key, {
        key, ProductID: line.ProductID, Quantity: line.Quantity, SubDivisor: line.SubDivisor || 1,
        Price: line.Price, SubPrice: line.SubPrice || 0,
      })
    }
  }
  return [...billLineByKey.values()]
}

export const tableOrderStatusLabels: Record<number, string> = {
  [TABLE_ORDER_STATUS.ANNULLED]: 'Annulled|Anulado',
  [TABLE_ORDER_STATUS.OPEN]: 'Open|En curso',
  [TABLE_ORDER_STATUS.CHARGED]: 'Charged|Cobrado',
}

// A dish the table is still owed: to confirm (QR), in the kitchen, or ready at the pickup point.
const UNSERVED_LINE_STATUSES: number[] = [LINE_STATUS.TO_CONFIRM, LINE_STATUS.ORDERED, LINE_STATUS.READY]

export interface ITableToServe {
  order: ITableOrder, tableLabel: string, unservedLines: ITableOrderLine[], waitTicks: number
  // The map's "Retrasado" state (§9), so the card and the panel's badge agree.
  isLate: boolean
}

// "Por entregar" (§5): every table still owed a dish, the longest-waiting first. A table waits since
// the oldest of its dishes in the kitchen or ready was ordered: a ready dish keeps counting until it
// is served. A QR line not confirmed yet has no OrderedAt, so a table with only those waits 0.
// Its dishes come oldest first, the ones never ordered last.
export function tablesToServe(orders: ITableOrder[], layout: IDiningLayout, nowTime: number): ITableToServe[] {
  const tables: ITableToServe[] = []
  for (const order of orders) {
    const unservedLines = order.Lines.filter((line) => UNSERVED_LINE_STATUSES.includes(line.Status))
    if (!unservedLines.length) continue
    unservedLines.sort((left, right) => (left.OrderedAt || nowTime) - (right.OrderedAt || nowTime))
    tables.push({
      order, tableLabel: tableLabelOf(layout, order.TableShapeID), unservedLines,
      waitTicks: nowTime - (unservedLines[0].OrderedAt || nowTime),
      isLate: tableState(order, layout, nowTime) === TABLE_STATE.LATE,
    })
  }
  return tables.sort((left, right) => right.waitTicks - left.waitTicks || left.tableLabel.localeCompare(right.tableLabel))
}

export const isTableShape = (kind: number) => kind === SHAPE_KIND.TABLE

// ── The round being composed ──────────────────────────────────────────────────────────────────

export interface IRoundDraftLine {
  key: number
  ProductID: number
  PresentationID: number
  Name: string
  Units: number
  Note: string
  Price: number
}

// One pick per sellable product presentation, keyed so the pick names both.
export const menuOptionKey = (productID: number, presentationID: number) => productID * 65536 + presentationID

export interface IMenuPickItem { ID: number, Name: string, Price: number, Sellable: number, searchText: string }

// The menu picker's list: every presentation with units left, as the round will name it.
export function menuPickItems(menu: IDiningMenuProduct[]): IMenuPickItem[] {
  return menu.flatMap((product) => product.Presentations
    .filter((presentation) => presentation.Sellable > 0)
    .map((presentation) => {
      const name = `${product.Name}${presentation.Name ? ` · ${presentation.Name}` : ''}`
      return {
        ID: menuOptionKey(product.ID, presentation.ID || 0), Name: name, Price: product.Price,
        Sellable: presentation.Sellable, searchText: name.toLowerCase(),
      }
    }))
}

// The picks whose name holds every word of the filter, in any order. An empty filter keeps them all.
export function filterMenuPickItems(items: IMenuPickItem[], filterText: string): IMenuPickItem[] {
  const filterWords = filterText.toLowerCase().split(' ').filter(Boolean)
  return filterWords.length ? items.filter((item) => filterWords.every((word) => item.searchText.includes(word))) : items
}

// Adds one unit of the picked presentation: to its line when it has one without a note, else as a new line.
export function addToRoundDraft(draft: IRoundDraftLine[], menu: IDiningMenuProduct[], optionKey: number): IRoundDraftLine[] {
  const productID = Math.floor(optionKey / 65536), presentationID = optionKey % 65536
  const product = menu.find((candidate) => candidate.ID === productID)
  if (!product) return draft
  const sameLine = draft.find((line) => line.ProductID === productID && line.PresentationID === presentationID && !line.Note)
  if (sameLine) return draft.map((line) => line === sameLine ? { ...line, Units: line.Units + 1 } : line)
  const presentationName = product.Presentations.find((presentation) => (presentation.ID || 0) === presentationID)?.Name
  return [...draft, {
    key: Math.max(0, ...draft.map((line) => line.key)) + 1, ProductID: productID, PresentationID: presentationID,
    Name: presentationName ? `${product.Name} · ${presentationName}` : product.Name, Units: 1, Note: '', Price: product.Price,
  }]
}

// The draft as the pack engine reads it: whole units, packed (#core/quantity.ts).
export const roundPackLines = (draft: IRoundDraftLine[]) =>
  draft.map((line) => ({ ...line, Quantity: line.Units * QUANTITY_LINE_SCALE }))

// The table splits a line the packs took part of, so a row edits only the units it shows: the rest
// of its line stays. An edit keeps the line at 0 units; a delete drops it once nothing is left.
export const setRoundRowUnits = (draft: IRoundDraftLine[], lineKey: number, shownUnits: number, newUnits: number) =>
  draft.map((line) => line.key === lineKey ? { ...line, Units: Math.max(0, line.Units - shownUnits + newUnits) } : line)

export const removeRoundRowUnits = (draft: IRoundDraftLine[], lineKey: number, shownUnits: number) =>
  setRoundRowUnits(draft, lineKey, shownUnits, 0).filter((line) => line.key !== lineKey || line.Units > 0)

// The request of a round: whole units (the floor sells dishes, not fractions), each line's note.
export function makeRoundRequest(layoutID: number, tableShapeID: number, guests: number, draft: IRoundDraftLine[]) {
  return {
    LayoutID: layoutID, TableShapeID: tableShapeID, Guests: guests,
    Lines: draft.filter((line) => line.Units > 0).map((line) => ({
      ProductID: line.ProductID, PresentationID: line.PresentationID, Note: line.Note.trim(),
      Quantity: packQuantityLine({ units: line.Units, sub: 0 }, QUANTITY_DIVISOR_NONE),
    })),
  }
}
