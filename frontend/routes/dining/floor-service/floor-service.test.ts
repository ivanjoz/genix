import { describe, expect, it } from 'vitest'
import {
  LINE_STATUS, SHAPE_KIND, TABLE_STATE, lineActions, orderRounds, rectPoints, selectionActions, tableState, tableWaitTicks,
  type IDiningLayout, type ITableOrder, type ITableOrderLine,
} from '../dining.ts'
import {
  addToRoundDraft, billLines, diningMenu, filterMenuPickItems, makeRoundRequest, menuOptionKey, menuPickItems, orderTotal,
  removeRoundRowUnits, setRoundRowUnits, tablesToServe, type IDiningMenuProduct,
} from './floor-service.ts'
import { PRODUCT_CHANNEL_PHYSICAL_STORE } from '#core/product-channels.ts'
import type { IProductStock } from '#routes/logistics/products-stock/stock-movement.ts'
import type { IProduct } from '#services/production/products.svelte.ts'

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

describe('selectionActions', () => {
  const actionsOf = (line: ITableOrderLine) => lineActions(line, NOW)
  const lines = [
    makeLine({ LineID: 1, Status: LINE_STATUS.ORDERED }),
    makeLine({ LineID: 2, Status: LINE_STATUS.ORDERED }),
    makeLine({ LineID: 3, Status: LINE_STATUS.READY, ReadyAt: NOW - 10 * MINUTE }),
    makeLine({ LineID: 4, Status: LINE_STATUS.TO_CONFIRM }),
  ]
  const buttonsOf = (selectedLineIDs: number[]) =>
    selectionActions(lines, selectedLineIDs, actionsOf).map((action) => [action.name, action.lineIDs])

  it('offers every action that fits a selected line, each on the lines it fits, cancel first', () => {
    expect(buttonsOf([1, 3])).toEqual([['Cancel|Anular', [1]], ['Ready|Listo', [1]], ['Served|Servido', [3]]])
    expect(buttonsOf([1, 2, 4])).toEqual([['Cancel|Anular', [1, 2, 4]], ['Confirm|Confirmar', [4]], ['Ready|Listo', [1, 2]]])
  })

  it('offers nothing for an empty selection', () => {
    expect(buttonsOf([])).toEqual([])
  })
})

describe('order reading', () => {
  it('groups lines by round, oldest first', () => {
    const order = makeOrder([makeLine({ LineID: 1, Round: 2 }), makeLine({ LineID: 2, Round: 1 }), makeLine({ LineID: 3, Round: 2 })])
    expect(orderRounds(order.Lines).map(({ round, lines }) => [round, lines.map((line) => line.LineID)])).toEqual([[1, [2]], [2, [1, 3]]])
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

describe('billLines', () => {
  it('merges a product across rounds at its price, and leaves cancelled lines out', () => {
    const bill = billLines(makeOrder([
      makeLine({ LineID: 1, Round: 1 }),
      makeLine({ LineID: 2, Round: 2, Quantity: 2000 }),
      makeLine({ LineID: 3, Round: 2, Status: LINE_STATUS.CANCELLED }),
      makeLine({ LineID: 4, Round: 2, Price: 3000 }),
    ]))
    expect(bill.map((billLine) => [billLine.Quantity, billLine.Price])).toEqual([[3000, 2500], [1000, 3000]])
  })
})

describe('round rows over a split line', () => {
  // Arroz con pato × 3, of which a pack took 1: its unpacked row shows 2.
  const draft = [{ key: 1, ProductID: 10, PresentationID: 0, Name: 'Arroz con pato', Units: 3, Note: '', Price: 2500 }]

  it('an edit changes only the units its row shows', () => {
    expect(setRoundRowUnits(draft, 1, 2, 5)[0].Units).toBe(6)
    expect(setRoundRowUnits(draft, 1, 3, 0)[0].Units).toBe(0)
  })

  it("a delete removes the row's units and drops the line once nothing is left", () => {
    expect(removeRoundRowUnits(draft, 1, 1)[0].Units).toBe(2)
    expect(removeRoundRowUnits(draft, 1, 3)).toEqual([])
    expect(removeRoundRowUnits([{ ...draft[0], Units: 0 }], 1, 0)).toEqual([])
  })
})

describe('tablesToServe', () => {
  const layout = makeLayout()

  it('lists the tables owed a dish, the longest-waiting first, a ready dish still counting', () => {
    const orders = [
      makeOrder([makeLine({ LineID: 1, Status: LINE_STATUS.ORDERED, OrderedAt: NOW - 5 * MINUTE })]),
      makeOrder([
        makeLine({ LineID: 1, Status: LINE_STATUS.SERVED, OrderedAt: NOW - 30 * MINUTE }),
        makeLine({ LineID: 2, Status: LINE_STATUS.TO_CONFIRM }),
        makeLine({ LineID: 3, Status: LINE_STATUS.READY, OrderedAt: NOW - 12 * MINUTE }),
      ], { ID: 2, TableShapeID: 2 }),
    ]
    const tables = tablesToServe(orders, layout, NOW)
    expect(tables.map((table) => table.tableLabel)).toEqual(['M2', 'M1'])
    expect(tables[0].waitTicks).toBe(12 * MINUTE)
    // The served dish is left out; the QR line, never ordered, goes last.
    expect(tables[0].unservedLines.map((line) => line.LineID)).toEqual([3, 2])
  })

  it('flags a table late as the map does: a dish in the kitchen past WaitLateMinutes', () => {
    const orders = [
      makeOrder([makeLine({ LineID: 1, Status: LINE_STATUS.ORDERED, OrderedAt: NOW - 30 * MINUTE })]),
      makeOrder([makeLine({ LineID: 1, Status: LINE_STATUS.READY, OrderedAt: NOW - 40 * MINUTE })], { ID: 2, TableShapeID: 2 }),
    ]
    // The ready dish has waited longer, but only a dish still in the kitchen makes a table late.
    expect(tablesToServe(orders, layout, NOW).map((table) => [table.tableLabel, table.isLate])).toEqual([['M2', false], ['M1', true]])
  })

  it('leaves out a table with everything served or cancelled', () => {
    const orders = [makeOrder([
      makeLine({ LineID: 1, Status: LINE_STATUS.SERVED }), makeLine({ LineID: 2, Status: LINE_STATUS.CANCELLED }),
    ])]
    expect(tablesToServe(orders, layout, NOW)).toEqual([])
  })
})

describe('the round draft', () => {
  const menu: IDiningMenuProduct[] = [
    { ID: 42, Name: 'Lomo saltado', Price: 3500, Presentations: [{ Sellable: 8 }] },
    { ID: 50, Name: 'Chicha', Price: 800, Presentations:[{ ID: 1, Name: 'Vaso', Sellable: 3 }, { ID: 2, Name: 'Jarra', Sellable: 0 }] },
  ]

  it('offers only presentations with units left', () => {
    expect(menuPickItems(menu).map((item) => [item.Name, item.Sellable])).toEqual([['Lomo saltado', 8], ['Chicha · Vaso', 3]])
  })

  it('filters the picks by every word of the filter, in any order', () => {
    const items = menuPickItems(menu)
    expect(filterMenuPickItems(items, '  ').map((item) => item.Name)).toEqual(['Lomo saltado', 'Chicha · Vaso'])
    expect(filterMenuPickItems(items, 'VASO chi').map((item) => item.Name)).toEqual(['Chicha · Vaso'])
    expect(filterMenuPickItems(items, 'lomo vaso')).toEqual([])
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

describe('the menu built from the cached catalog and stock', () => {
  const product = (fields: Partial<IProduct>) => ({ Presentations: [], CurrencyID: 1, WebExtraStock: 0, SalesChannels: 0, ...fields }) as IProduct
  const stock = (fields: Partial<IProductStock>) => ({ WarehouseID: 1, PresentationID: 0, Quantity: 0, ...fields }) as IProductStock
  const products = [
    product({ ID: 1, Name: 'Lomo saltado', FinalPrice: 3500 }),
    product({ ID: 2, Name: 'Polo', FinalPrice: 2000, SalesChannels: 1 << (PRODUCT_CHANNEL_PHYSICAL_STORE - 1) }),
    product({ ID: 3, Name: 'Vino', FinalPrice: 1000, CurrencyID: 2 }),
    product({ ID: 4, Name: 'Chicha', FinalPrice: 800, Presentations: [
      { id: 1, nm: 'Vaso', ss: 1 }, { id: 2, nm: 'Jarra', ss: 1 }, { id: 3, nm: 'Botella', ss: 0 },
    ] as IProduct['Presentations'] }),
    product({ ID: 5, Name: 'Ceviche', FinalPrice: 4000 }),
  ]
  const stockRows = [
    stock({ ProductID: 1, Quantity: 5, PendingDeliveryStore: 2 }),
    stock({ ProductID: 1, WarehouseID: 9, Quantity: 50 }),
    stock({ ProductID: 2, Quantity: 5 }),
    stock({ ProductID: 3, Quantity: 5 }),
    stock({ ProductID: 4, PresentationID: 1, Quantity: 4 }),
    stock({ ProductID: 4, PresentationID: 3, Quantity: 9 }),
  ]

  it('keeps the restaurant products with units left on the layout warehouse alone', () => {
    const menu = diningMenu(products, stockRows, 1, 0)
    expect(menu.map((menuProduct) => menuProduct.Name)).toEqual(['Lomo saltado', 'Chicha'])
    expect(menu[0].Presentations).toEqual([{ ID: 0, Name: '', Sellable: 3 }])
    // The inactive Botella is not offered; Jarra is listed with 0 so the picker leaves it out.
    expect(menu[1].Presentations).toEqual([{ ID: 1, Name: 'Vaso', Sellable: 4 }, { ID: 2, Name: 'Jarra', Sellable: 0 }])
  })

  it('restates a USD price in soles, and leaves the product out without a rate', () => {
    expect(diningMenu(products, stockRows, 1, 3750).find((menuProduct) => menuProduct.ID === 3)?.Price).toBe(3750)
    expect(diningMenu(products, stockRows, 1, 0).some((menuProduct) => menuProduct.ID === 3)).toBe(false)
  })
})
