import { describe, expect, it } from 'vitest'
import { LINE_STATUS, SHAPE_KIND, rectPoints, type IDiningLayout, type ITableOrder, type ITableOrderLine } from '../dining.ts'
import {
  gridStep, kitchenLineActions, kitchenTickets, linesToMarkReady, pendingLineAfter, resolveSelectedTicketKey, ticketWaitLevel,
} from './kitchen-display.ts'

const MINUTE = 30 // SUnixTime ticks
const NOW = 500_000_000

const layout: IDiningLayout = {
  ID: 1, Name: 'Salón', WarehouseID: 1, GridWidth: 40, GridHeight: 30, WaitWarnMinutes: 15, WaitLateMinutes: 25, ss: 1, upd: 0,
  Shapes: [
    { ShapeID: 1, Kind: SHAPE_KIND.TABLE, Points: rectPoints({ x: 2, y: 2, w: 3, h: 3, quarterTurns: 0 }), Label: 'M1' },
    { ShapeID: 2, Kind: SHAPE_KIND.TABLE, Points: rectPoints({ x: 8, y: 2, w: 3, h: 3, quarterTurns: 0 }), Label: 'M2' },
  ],
  Stations: [{ ID: 1, Name: 'Cocina' }, { ID: 2, Name: 'Bar' }],
}

const makeLine = (fields: Partial<ITableOrderLine>): ITableOrderLine => ({
  LineID: 1, Round: 1, ProductID: 42, Quantity: 1000, Price: 2500, StationID: 1, Status: LINE_STATUS.ORDERED, OrderedAt: NOW, ...fields,
})

const makeOrder = (lines: ITableOrderLine[], fields: Partial<ITableOrder> = {}): ITableOrder => ({
  ID: 1, LayoutID: 1, TableShapeID: 1, WarehouseID: 1, Lines: lines, ss: 1, upd: 0, ...fields,
})

describe('kitchenTickets', () => {
  const orders = [
    makeOrder([
      makeLine({ LineID: 1, Round: 1, StationID: 1, OrderedAt: NOW - 10 * MINUTE }),
      makeLine({ LineID: 2, Round: 1, StationID: 2, OrderedAt: NOW - 10 * MINUTE }),
      makeLine({ LineID: 3, Round: 1, StationID: 1, Status: LINE_STATUS.READY }),
      makeLine({ LineID: 4, Round: 2, StationID: 1, Status: LINE_STATUS.READY }),
      makeLine({ LineID: 5, Round: 2, StationID: 1, Status: LINE_STATUS.TO_CONFIRM }),
      makeLine({ LineID: 6, Round: 1, StationID: 1, Status: LINE_STATUS.SERVED }),
    ]),
    makeOrder([makeLine({ LineID: 1, StationID: 1, OrderedAt: NOW - 20 * MINUTE })], { ID: 2, TableShapeID: 2 }),
  ]

  it('makes one ticket per round and station, the longest-waiting first', () => {
    expect(kitchenTickets(orders, layout, 0, NOW).map((ticket) => [ticket.tableLabel, ticket.round, ticket.stationName]))
      .toEqual([['M2', 1, 'Cocina'], ['M1', 1, 'Cocina'], ['M1', 1, 'Bar']])
  })

  it('keeps the ready lines of a pending ticket, leaves out QR lines to confirm and served ones', () => {
    const ticket = kitchenTickets(orders, layout, 1, NOW).find((candidate) => candidate.orderID === 1)!
    expect(ticket.lines.map((line) => line.LineID)).toEqual([1, 3])
    expect(ticket.pendingLineIDs).toEqual([1])
  })

  it('drops a ticket once all its lines are ready, and filters by station', () => {
    expect(kitchenTickets(orders, layout, 1, NOW).some((ticket) => ticket.round === 2)).toBe(false)
    expect(kitchenTickets(orders, layout, 2, NOW).map((ticket) => ticket.stationName)).toEqual(['Bar'])
  })

  it('colors the wait by the layout thresholds', () => {
    const [m2Ticket, m1Ticket] = kitchenTickets(orders, layout, 1, NOW)
    expect(ticketWaitLevel(m1Ticket, layout, NOW)).toBe('ok')
    expect(ticketWaitLevel(m2Ticket, layout, NOW)).toBe('warn')
    expect(ticketWaitLevel(m2Ticket, layout, NOW + 5 * MINUTE)).toBe('late')
  })
})

describe('kitchenLineActions', () => {
  it('only marks ready, or takes back a recent ready tap', () => {
    const statusesOf = (line: ITableOrderLine) => kitchenLineActions(line, NOW).map((action) => action.status)
    expect(statusesOf(makeLine({ Status: LINE_STATUS.ORDERED }))).toEqual([LINE_STATUS.READY])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.READY, ReadyAt: NOW - MINUTE }))).toEqual([LINE_STATUS.ORDERED])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.READY, ReadyAt: NOW - 3 * MINUTE }))).toEqual([])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.TO_CONFIRM }))).toEqual([])
  })
})

describe('the selection', () => {
  const ticket = kitchenTickets([makeOrder([
    makeLine({ LineID: 1 }), makeLine({ LineID: 2, Status: LINE_STATUS.READY }), makeLine({ LineID: 3 }), makeLine({ LineID: 4 }),
  ])], layout, 0, NOW)[0]

  it('marks ready the selected lines still in the kitchen, else the focused card', () => {
    expect(linesToMarkReady(ticket, [2, 3], 1)).toEqual([3])
    expect(linesToMarkReady(ticket, [], 4)).toEqual([4])
    expect(linesToMarkReady(ticket, [], 2)).toEqual([])
  })

  it('moves the keyboard to the next card still in the kitchen, wrapping to the top', () => {
    expect(pendingLineAfter(ticket, [1])).toBe(3)
    expect(pendingLineAfter(ticket, [4])).toBe(1)
    expect(pendingLineAfter(ticket, [1, 3, 4])).toBe(0)
  })

  it('keeps the selected ticket, or takes the one that moved into its row', () => {
    const tickets = kitchenTickets([
      makeOrder([makeLine({ LineID: 1, OrderedAt: NOW - 2 })]),
      makeOrder([makeLine({ LineID: 1, OrderedAt: NOW - 1 })], { ID: 2 }),
    ], layout, 0, NOW)
    expect(resolveSelectedTicketKey(tickets, '2-1-1', 0)).toBe('2-1-1')
    expect(resolveSelectedTicketKey(tickets, '9-1-1', 1)).toBe('2-1-1')
    expect(resolveSelectedTicketKey(tickets, '9-1-1', 5)).toBe('2-1-1')
    expect(resolveSelectedTicketKey([], '9-1-1', 0)).toBe('')
  })
})

describe('gridStep', () => {
  it('walks a two-column grid and leaves it by the left edge', () => {
    expect(gridStep(0, 'ArrowRight', 5)).toBe(1)
    expect(gridStep(1, 'ArrowDown', 5)).toBe(3)
    expect(gridStep(3, 'ArrowDown', 5)).toBe(3)
    expect(gridStep(4, 'ArrowRight', 5)).toBe(4)
    expect(gridStep(2, 'ArrowUp', 5)).toBe(0)
    expect(gridStep(1, 'ArrowLeft', 5)).toBe(0)
    expect(gridStep(2, 'ArrowLeft', 5)).toBe(-1)
  })
})
