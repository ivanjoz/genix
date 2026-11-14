import { describe, expect, it } from 'vitest'
import {
  LINE_STATUS, SHAPE_KIND, TABLE_STATE, rectPoints, tableState, tableWaitTicks,
  type IDiningLayout, type ITableOrder, type ITableOrderLine,
} from '../dining.ts'
import {
  addToRoundDraft, lineActions, makeRoundRequest, menuOptionKey, menuOptions, orderRounds, orderTotal,
  readyToPickupGroups, type IDiningMenuProduct,
} from './floor-service.ts'

const MINUTE = 30 // SUnixTime ticks
const NOW = 500_000_000

const makeLayout = (): IDiningLayout => ({
  ID: 1, Name: 'Salón', WarehouseID: 1, GridWidth: 40, GridHeight: 30, WaitWarnMinutes: 15, WaitLateMinutes: 25, ss: 1, upd: 0,
  Shapes: [
    { ShapeID: 1, Kind: SHAPE_KIND.TABLE, Points: rectPoints({ x: 2, y: 2, w: 3, h: 3, quarterTurns: 0 }), Label: 'M1' },
    { ShapeID: 2, Kind: SHAPE_KIND.TABLE, Points: rectPoints({ x: 8, y: 2, w: 3, h: 3, quarterTurns: 0 }), Label: 'M2' },
  ],
  Stations: [{ ID: 1, Name: 'Cocina' }, { ID: 2, Name: 'Bar' }],
})

const makeLine = (fields: Partial<ITableOrderLine>): ITableOrderLine => ({
  LineID: 1, Round: 1, ProductID: 42, Quantity: 1000, Price: 2500, StationID: 1, Status: LINE_STATUS.ORDERED, ...fields,
})

const makeOrder = (lines: ITableOrderLine[], fields: Partial<ITableOrder> = {}): ITableOrder => ({
  ID: 1, LayoutID: 1, TableShapeID: 1, WarehouseID: 1, Lines: lines, ss: 1, upd: 0, ...fields,
})

describe('tableState', () => {
  const layout = makeLayout()

  it('is free without an order and eating once everything is served', () => {
    expect(tableState(undefined, layout, NOW)).toBe(TABLE_STATE.FREE)
    expect(tableState(makeOrder([makeLine({ Status: LINE_STATUS.SERVED })]), layout, NOW)).toBe(TABLE_STATE.EATING)
  })

  it('goes from ordered to waiting to late as the oldest kitchen line ages', () => {
    const orderedAt = (minutes: number) => makeOrder([makeLine({ OrderedAt: NOW - minutes * MINUTE })])
    expect(tableState(orderedAt(5), layout, NOW)).toBe(TABLE_STATE.ORDERED)
    expect(tableState(orderedAt(15), layout, NOW)).toBe(TABLE_STATE.WAITING)
    expect(tableState(orderedAt(25), layout, NOW)).toBe(TABLE_STATE.LATE)
  })

  it('shows ready over waiting, but late wins over ready', () => {
    const ready = makeLine({ LineID: 2, Status: LINE_STATUS.READY })
    expect(tableState(makeOrder([makeLine({ OrderedAt: NOW - 16 * MINUTE }), ready]), layout, NOW)).toBe(TABLE_STATE.READY)
    expect(tableState(makeOrder([makeLine({ OrderedAt: NOW - 30 * MINUTE }), ready]), layout, NOW)).toBe(TABLE_STATE.LATE)
  })

  it('measures the wait only on lines still in the kitchen', () => {
    const order = makeOrder([
      makeLine({ LineID: 1, Status: LINE_STATUS.SERVED, OrderedAt: NOW - 40 * MINUTE }),
      makeLine({ LineID: 2, OrderedAt: NOW - 3 * MINUTE }),
    ])
    expect(tableWaitTicks(order, NOW)).toBe(3 * MINUTE)
  })
})

describe('lineActions', () => {
  const statusesOf = (line: ITableOrderLine) => lineActions(line, NOW).map((action) => action.status)

  it('lets a waiter confirm or cancel, mark ready or cancel, and serve', () => {
    expect(statusesOf(makeLine({ Status: LINE_STATUS.TO_CONFIRM }))).toEqual([LINE_STATUS.ORDERED, LINE_STATUS.CANCELLED])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.ORDERED }))).toEqual([LINE_STATUS.READY, LINE_STATUS.CANCELLED])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.SERVED }))).toEqual([])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.CANCELLED }))).toEqual([])
  })

  it('offers sending a ready line back only within two minutes', () => {
    expect(statusesOf(makeLine({ Status: LINE_STATUS.READY, ReadyAt: NOW - 2 * MINUTE }))).toEqual([LINE_STATUS.SERVED, LINE_STATUS.ORDERED])
    expect(statusesOf(makeLine({ Status: LINE_STATUS.READY, ReadyAt: NOW - 2 * MINUTE - 1 }))).toEqual([LINE_STATUS.SERVED])
  })
})

describe('order reading', () => {
  it('groups lines by round, oldest first', () => {
    const order = makeOrder([makeLine({ LineID: 1, Round: 2 }), makeLine({ LineID: 2, Round: 1 }), makeLine({ LineID: 3, Round: 2 })])
    expect(orderRounds(order).map(({ round, lines }) => [round, lines.map((line) => line.LineID)])).toEqual([[1, [2]], [2, [1, 3]]])
  })

  it('totals every line but the cancelled ones, sub-units at their own price', () => {
    const order = makeOrder([
      makeLine({ LineID: 1, Quantity: 2000, Price: 2500 }),
      makeLine({ LineID: 2, Quantity: 1002, Price: 1200, SubPrice: 300, SubDivisor: 6 }),
      makeLine({ LineID: 3, Quantity: 5000, Price: 9999, Status: LINE_STATUS.CANCELLED }),
    ])
    expect(orderTotal(order)).toBe(2 * 2500 + 1200 + 2 * 300)
  })
})

describe('readyToPickupGroups', () => {
  it('groups ready lines by station, the longest-waiting first', () => {
    const orders = [
      makeOrder([makeLine({ LineID: 1, Status: LINE_STATUS.READY, StationID: 2, ReadyAt: NOW - 10 })]),
      makeOrder([
        makeLine({ LineID: 1, Status: LINE_STATUS.READY, StationID: 1, ReadyAt: NOW - 50 }),
        makeLine({ LineID: 2, Status: LINE_STATUS.READY, StationID: 2, ReadyAt: NOW - 90 }),
        makeLine({ LineID: 3, Status: LINE_STATUS.ORDERED, StationID: 1 }),
      ], { ID: 2, TableShapeID: 2 }),
    ]
    const groups = readyToPickupGroups(orders, makeLayout())
    expect(groups.map((group) => group.stationName)).toEqual(['Bar', 'Cocina'])
    expect(groups[0].items.map((item) => item.tableLabel)).toEqual(['M2', 'M1'])
    expect(groups[1].items).toHaveLength(1)
  })
})

describe('the round draft', () => {
  const menu: IDiningMenuProduct[] = [
    { ID: 42, Name: 'Lomo saltado', Price: 3500, StationID: 1, Presentations: [{ Sellable: 8 }] },
    { ID: 50, Name: 'Chicha', Price: 800, StationID: 2, Presentations: [{ ID: 1, Name: 'Vaso', Sellable: 3 }, { ID: 2, Name: 'Jarra', Sellable: 0 }] },
  ]

  it('offers only presentations with units left', () => {
    expect(menuOptions(menu).map((option) => option.Name)).toEqual(['Lomo saltado (8)', 'Chicha · Vaso (3)'])
  })

  it('adds a repeated pick to its line unless that line carries a note', () => {
    let draft = addToRoundDraft([], menu, menuOptionKey(42, 0))
    draft = addToRoundDraft(draft, menu, menuOptionKey(42, 0))
    expect(draft).toHaveLength(1)
    expect(draft[0].Units).toBe(2)
    draft = [{ ...draft[0], Note: 'sin cebolla' }]
    draft = addToRoundDraft(draft, menu, menuOptionKey(42, 0))
    expect(draft.map((line) => [line.Units, line.Note])).toEqual([[2, 'sin cebolla'], [1, '']])
  })

  it('sends whole units and drops lines left at zero', () => {
    const draft = addToRoundDraft(addToRoundDraft([], menu, menuOptionKey(42, 0)), menu, menuOptionKey(50, 1))
    draft[0] = { ...draft[0], Units: 3, Note: '  sin ají ' }
    draft[1] = { ...draft[1], Units: 0 }
    expect(makeRoundRequest(1, 2, 4, draft)).toEqual({
      LayoutID: 1, TableShapeID: 2, Guests: 4,
      Lines: [{ ProductID: 42, PresentationID: 0, Note: 'sin ají', Quantity: 3000 }],
    })
  })
})
