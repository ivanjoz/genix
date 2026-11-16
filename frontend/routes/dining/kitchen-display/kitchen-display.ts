import {
  LINE_STATUS, lineActions, tableLabelOf, waitMinutes,
  type IDiningLayout, type ILineAction, type ITableOrder, type ITableOrderLine,
} from '../dining.ts'

// The kitchen's rules (docs/DINING_PLAN.md §5): which tickets it has to prepare, in which order,
// and where the selection goes as they are prepared. Pure, so they are tested without a component.

// One ticket: a round of a table, for one station. The kitchen and the bar each see their own lines.
export interface IKitchenTicket {
  key: string
  orderID: number
  tableLabel: string
  round: number
  stationID: number
  stationName: string
  // The lines the station works on: in the kitchen and already ready. QR lines still to confirm
  // have not reached it; served and cancelled ones are no longer its business.
  lines: ITableOrderLine[]
  pendingLineIDs: number[]
  // When the oldest line still in the kitchen was ordered (SUnixTime).
  waitingSince: number
}

const STATIONS_ALL = 0

// The tickets still to prepare (at least one line in the kitchen), the longest-waiting first. A
// ticket leaves once all its lines are ready. stationID 0 shows every station.
export function kitchenTickets(orders: ITableOrder[], layout: IDiningLayout, stationID: number, nowTime: number): IKitchenTicket[] {
  const ticketByKey = new Map<string, IKitchenTicket>()
  for (const order of orders) {
    for (const line of order.Lines) {
      if (line.Status !== LINE_STATUS.ORDERED && line.Status !== LINE_STATUS.READY) continue
      if (stationID !== STATIONS_ALL && line.StationID !== stationID) continue
      const key = `${order.ID}-${line.Round}-${line.StationID}`
      let ticket = ticketByKey.get(key)
      if (!ticket) {
        ticket = {
          key, orderID: order.ID, tableLabel: tableLabelOf(layout, order.TableShapeID), round: line.Round,
          stationID: line.StationID, stationName: (layout.Stations || []).find((station) => station.ID === line.StationID)?.Name || '',
          lines: [], pendingLineIDs: [], waitingSince: nowTime,
        }
        ticketByKey.set(key, ticket)
      }
      ticket.lines.push(line)
      if (line.Status === LINE_STATUS.ORDERED) {
        ticket.pendingLineIDs.push(line.LineID)
        ticket.waitingSince = Math.min(ticket.waitingSince, line.OrderedAt || nowTime)
      }
    }
  }
  return [...ticketByKey.values()]
    .filter((ticket) => ticket.pendingLineIDs.length > 0)
    .sort((left, right) => left.waitingSince - right.waitingSince || left.orderID - right.orderID
      || left.round - right.round || left.stationID - right.stationID)
}

export type TWaitLevel = 'ok' | 'warn' | 'late'

// How late a ticket runs, by the layout's thresholds — the same ones that color the floor map.
export function ticketWaitLevel(ticket: IKitchenTicket, layout: IDiningLayout, nowTime: number): TWaitLevel {
  const waitedMinutes = waitMinutes(nowTime - ticket.waitingSince)
  if (waitedMinutes >= layout.WaitLateMinutes) return 'late'
  if (waitedMinutes >= layout.WaitWarnMinutes) return 'warn'
  return 'ok'
}

// The ticket to keep selected when the list changes: the same one while it is there, else the one
// that took its place (a prepared ticket leaves, the next one moves up into its row).
export function resolveSelectedTicketKey(tickets: IKitchenTicket[], selectedKey: string, selectedIndex: number): string {
  if (tickets.some((ticket) => ticket.key === selectedKey)) return selectedKey
  return tickets[Math.min(Math.max(selectedIndex, 0), tickets.length - 1)]?.key || ''
}

// What the kitchen may do with a line (backend IsKitchenTransition): mark it ready, or take that
// tap back within its window. Serving, cancelling and confirming belong to the floor.
export function kitchenLineActions(line: ITableOrderLine, nowTime: number): ILineAction[] {
  return lineActions(line, nowTime).filter((action) =>
    (line.Status === LINE_STATUS.ORDERED && action.status === LINE_STATUS.READY) ||
    (line.Status === LINE_STATUS.READY && action.status === LINE_STATUS.ORDERED))
}

// The lines Enter marks ready: the selected ones still in the kitchen, else the card the keyboard is on.
export function linesToMarkReady(ticket: IKitchenTicket, selectedLineIDs: number[], focusedLineID: number): number[] {
  const selectedPending = ticket.pendingLineIDs.filter((lineID) => selectedLineIDs.includes(lineID))
  if (selectedPending.length) return selectedPending
  return ticket.pendingLineIDs.includes(focusedLineID) ? [focusedLineID] : []
}

// The card to put the keyboard on once lineIDs are marked ready: the next one still in the
// kitchen after the last of them, then from the top; 0 when none is left (the ticket leaves).
export function pendingLineAfter(ticket: IKitchenTicket, lineIDs: number[]): number {
  const lastPosition = Math.max(-1, ...lineIDs.map((lineID) => ticket.lines.findIndex((line) => line.LineID === lineID)))
  const ordered = [...ticket.lines.slice(lastPosition + 1), ...ticket.lines.slice(0, lastPosition + 1)]
  return ordered.find((line) => !lineIDs.includes(line.LineID) && line.Status === LINE_STATUS.ORDERED)?.LineID || 0
}

// The detail lays its cards on this many columns; the arrows move by it.
export const DETAIL_COLUMNS = 2

// Where an arrow key takes the keyboard in a grid of cardCount cards: the next card's index, the
// same one at an edge, or -1 for ← on the first column (back to the ticket list).
export function gridStep(index: number, key: string, cardCount: number, columns = DETAIL_COLUMNS): number {
  switch (key) {
    case 'ArrowLeft': return index % columns === 0 ? -1 : index - 1
    case 'ArrowRight': return Math.min(index + 1, cardCount - 1)
    case 'ArrowUp': return index - columns >= 0 ? index - columns : index
    case 'ArrowDown': return index + columns < cardCount ? index + columns : index
    default: return index
  }
}
