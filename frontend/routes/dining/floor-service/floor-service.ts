import { QUANTITY_DIVISOR_NONE, packQuantityLine, quantityAmount, unpackQuantityLine } from '#core/quantity.ts'
import { LINE_STATUS, SHAPE_KIND, type IDiningLayout, type ITableOrder, type ITableOrderLine } from '../dining.ts'

// The floor-service page's rules (docs/DINING_PLAN.md §2, §5): what a waiter can do with a line, how
// an order reads, the ready-to-pick-up strip and the round being composed. Pure, so they are tested
// without a component or a request.

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

// How long the kitchen can take back a "ready" tap: 2 minutes of 2-second ticks (backend readyUndoWindowTicks).
const READY_UNDO_TICKS = 60

export interface ILineAction { status: number, name: string, color: 'green' | 'blue' | 'red' | undefined }

// What can be done with a line now (§2.1). A waiter may do all of it, "ready" included: a
// restaurant without a kitchen display marks dishes ready from here.
export function lineActions(line: ITableOrderLine, nowTime: number): ILineAction[] {
  const cancel: ILineAction = { status: LINE_STATUS.CANCELLED, name: 'Cancel|Anular', color: 'red' }
  switch (line.Status) {
    case LINE_STATUS.TO_CONFIRM:
      return [{ status: LINE_STATUS.ORDERED, name: 'Confirm|Confirmar', color: 'blue' }, cancel]
    case LINE_STATUS.ORDERED:
      return [{ status: LINE_STATUS.READY, name: 'Ready|Listo', color: 'blue' }, cancel]
    case LINE_STATUS.READY: {
      const actions: ILineAction[] = [{ status: LINE_STATUS.SERVED, name: 'Served|Servido', color: 'green' }]
      if (nowTime - (line.ReadyAt || 0) <= READY_UNDO_TICKS) {
        actions.push({ status: LINE_STATUS.ORDERED, name: 'Back to kitchen|Devolver a cocina', color: undefined })
      }
      return actions
    }
    default:
      return []
  }
}

// The order's rounds, oldest first, each with its lines.
export function orderRounds(order: ITableOrder): { round: number, lines: ITableOrderLine[] }[] {
  const linesByRound = new Map<number, ITableOrderLine[]>()
  for (const line of order.Lines) linesByRound.set(line.Round, [...(linesByRound.get(line.Round) || []), line])
  return [...linesByRound.entries()].sort(([left], [right]) => left - right).map(([round, lines]) => ({ round, lines }))
}

export const lineAmount = (line: ITableOrderLine) =>
  quantityAmount(unpackQuantityLine(line.Quantity), line.Price, line.SubPrice || 0)

// What the table owes so far: every line not cancelled, at the price it was ordered at.
export const orderTotal = (order: ITableOrder) =>
  order.Lines.filter((line) => line.Status !== LINE_STATUS.CANCELLED).reduce((total, line) => total + lineAmount(line), 0)

export const tableLabelOf = (layout: IDiningLayout, tableShapeID: number) =>
  (layout.Shapes || []).find((shape) => shape.ShapeID === tableShapeID)?.Label || `#${tableShapeID}`

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

// One option per sellable product presentation, keyed so the pick names both.
export const menuOptionKey = (productID: number, presentationID: number) => productID * 65536 + presentationID

export function menuOptions(menu: IDiningMenuProduct[]): { ID: number, Name: string }[] {
  return menu.flatMap((product) => product.Presentations
    .filter((presentation) => presentation.Sellable > 0)
    .map((presentation) => ({
      ID: menuOptionKey(product.ID, presentation.ID || 0),
      Name: `${product.Name}${presentation.Name ? ` · ${presentation.Name}` : ''} (${presentation.Sellable})`,
    })))
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
