import { QUANTITY_DIVISOR_NONE, packQuantityLine } from '#core/quantity.ts'
import { LINE_STATUS, SHAPE_KIND, TABLE_ORDER_STATUS, lineAmount, tableLabelOf, type IDiningLayout, type ITableOrder, type ITableOrderLine } from '../dining.ts'

// The floor-service page's rules (docs/DINING_PLAN.md §2, §5): what a table owes, the dishes to
// deliver and the round being composed. Pure, so they are tested without a component or a request.

// One product of a layout's menu (backend diningMenuProduct), priced in soles.
export interface IDiningMenuProduct {
  ID: number
  Name: string
  Price: number
  SubPrice?: number
  SubDivisor?: number
  CategoryIDs?: number[]
  StationID: number
  Presentations: { ID?: number, Name?: string, Sellable: number }[]
}

// What the table owes so far: every line not cancelled, at the price it was ordered at.
export const orderTotal = (order: ITableOrder) =>
  order.Lines.filter((line) => line.Status !== LINE_STATUS.CANCELLED).reduce((total, line) => total + lineAmount(line), 0)

export interface IBillLine { key: string, ProductID: number, Quantity: number, SubDivisor: number, Amount: number }

// The bill as "Pagar" lists it, PoS style: one row per product, presentation and price across every
// round, cancelled lines left out. Packed quantities add up (units × scale + sub).
export function billLines(order: ITableOrder): IBillLine[] {
  const billLineByKey = new Map<string, IBillLine>()
  for (const line of order.Lines.filter((candidate) => candidate.Status !== LINE_STATUS.CANCELLED)) {
    const key = `${line.ProductID}_${line.PresentationID || 0}_${line.Price}_${line.SubPrice || 0}`
    const billLine = billLineByKey.get(key)
    if (billLine) {
      billLine.Quantity += line.Quantity
      billLine.Amount += lineAmount(line)
    } else {
      billLineByKey.set(key, { key, ProductID: line.ProductID, Quantity: line.Quantity, SubDivisor: line.SubDivisor || 1, Amount: lineAmount(line) })
    }
  }
  return [...billLineByKey.values()]
}

export const tableOrderStatusLabels: Record<number, string> = {
  [TABLE_ORDER_STATUS.ANNULLED]: 'Annulled|Anulado',
  [TABLE_ORDER_STATUS.OPEN]: 'Open|En curso',
  [TABLE_ORDER_STATUS.CHARGED]: 'Charged|Cobrado',
}

export interface IReadyItem { orderID: number, tableLabel: string, line: ITableOrderLine }
export interface IPickupGroup { stationID: number, stationName: string, items: IReadyItem[] }

// "Listos para recoger" (§5): the ready lines of every table, grouped by the station whose pickup
// point they wait at, the longest-waiting first in each group and across groups.
export function readyToPickupGroups(orders: ITableOrder[], layout: IDiningLayout): IPickupGroup[] {
  const groupByStation = new Map<number, IPickupGroup>()
  for (const order of orders) {
    for (const line of order.Lines.filter((candidate) => candidate.Status === LINE_STATUS.READY)) {
      let group = groupByStation.get(line.StationID)
      if (!group) {
        const stationName = (layout.Stations || []).find((station) => station.ID === line.StationID)?.Name || ''
        group = { stationID: line.StationID, stationName, items: [] }
        groupByStation.set(line.StationID, group)
      }
      group.items.push({ orderID: order.ID, tableLabel: tableLabelOf(layout, order.TableShapeID), line })
    }
  }
  const groups = [...groupByStation.values()]
  const readyAt = (item: IReadyItem) => item.line.ReadyAt || 0
  for (const group of groups) group.items.sort((left, right) => readyAt(left) - readyAt(right))
  return groups.sort((left, right) => readyAt(left.items[0]) - readyAt(right.items[0]))
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

export const roundDraftTotal = (draft: IRoundDraftLine[]) => draft.reduce((total, line) => total + line.Units * line.Price, 0)

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
