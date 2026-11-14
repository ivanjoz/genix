import { describe, expect, it } from 'vitest'
import type { IProductionOrderLine, IProductionOrderSupply } from '#services/production/production-orders.ts'
import type { IProductPresentation } from '#services/production/products.svelte.ts'
import {
  expandTemplateToOrderLines, explodeProductionLines, findOrderDraftError, findReceiptError, makeReceiptEntryLines,
  NO_PRESENTATION_COLUMN, orderLinesToProductRuns, orderLinesToTemplateLines, orderProgressPercent, presentationIDAtColumn,
  productRunsToOrderLines, proposeReceiptConsumptions, setRunUnits, sizeGridColumns, summarizeOrderLabor, summarizeOrderProducts,
  summarizeOrderSupplies, unitsByProduct, type IRecipeProduct, materialToSend,
} from './production-order.ts'

const SANDAL_ID = 1
const BOOT_ID = 2
const SOLE_ID = 3 // a semi-finished supply: no presentations
const LEATHER_ID = 10 // m², divisor 100
const GLUE_ID = 11 // no sub-unit
const SHOEMAKER_ID = 1

const size = (id: number, name: string, status = 1) => ({ id, nm: name, ss: status } as IProductPresentation)

const recipeProduct = (product: Partial<IRecipeProduct> & { ID: number }): IRecipeProduct => ({
  Name: `#${product.ID}`, ProductionYieldPortions: 1, ProductionSupplies: [], ProductionExtraCostPerUnit: 0,
  ProductionStandardCost: 0, ProductionLabor: [], ...product,
})

const productsByID = new Map<number, IRecipeProduct>([
  [SANDAL_ID, recipeProduct({
    ID: SANDAL_ID, Presentations: [size(1, '37'), size(2, '38'), size(3, '39', 0)],
    // 2 pairs need 1 m² + 20 of 100 sub-units of leather with 10% waste, 1 glue tube, and 30 minutes of shoemaker.
    ProductionYieldPortions: 2,
    ProductionSupplies: [
      { SupplyID: LEATHER_ID, Quantity: 1020, SubDivisor: 100, WasteBasisPoints: 1000 },
      { SupplyID: GLUE_ID, Quantity: 1000, SubDivisor: 1 },
    ],
    ProductionLabor: [{ LaborTypeID: SHOEMAKER_ID, Minutes: 30 }],
  })],
  [BOOT_ID, recipeProduct({
    ID: BOOT_ID, Presentations: [size(4, '38'), size(5, '40')],
    ProductionSupplies: [{ SupplyID: LEATHER_ID, Quantity: 50, SubDivisor: 100 }],
  })],
  [SOLE_ID, recipeProduct({ ID: SOLE_ID, ProductionStandardCost: 300 })],
])

const orderLine = (productID: number, presentationID: number, planned: number, produced = 0, scrapped = 0, value = 0): IProductionOrderLine => ({
  ProductID: productID, PresentationID: presentationID, PlannedUnits: planned, ProducedUnits: produced, ScrappedUnits: scrapped,
  Value: value, CostSource: 0,
})

const orderSupply = (supplyID: number, planned: number, consumed = 0, subDivisor = 1): IProductionOrderSupply => ({
  SupplyID: supplyID, SubDivisor: subDivisor, Planned: planned, Available: 0, Consumed: consumed, Reserved: 0,
})

describe('size grid', () => {
  it('unites the active sizes by name in first-appearance order, plus a column for products without sizes', () => {
    const columns = sizeGridColumns([
      // A row with no product picked yet adds no column.
      { ProductID: 0, Run: [] }, { ProductID: SANDAL_ID, Run: [] }, { ProductID: BOOT_ID, Run: [] }, { ProductID: SOLE_ID, Run: [] },
    ], productsByID)
    expect(columns.map((column) => column.key)).toEqual(['37', '38', '40', NO_PRESENTATION_COLUMN])
  })

  it('maps a column to each product own presentation, none when it lacks that size', () => {
    expect(presentationIDAtColumn(productsByID.get(BOOT_ID), '38')).toBe(4)
    expect(presentationIDAtColumn(productsByID.get(SANDAL_ID), '40')).toBeUndefined()
    // An inactive size has no cell.
    expect(presentationIDAtColumn(productsByID.get(SANDAL_ID), '39')).toBeUndefined()
    expect(presentationIDAtColumn(productsByID.get(SOLE_ID), NO_PRESENTATION_COLUMN)).toBe(0)
  })

  it('sets and clears the units of one size', () => {
    const productRun = { ProductID: SANDAL_ID, Run: [{ PresentationID: 1, Units: 2 }] }
    setRunUnits(productRun, 2, 3)
    setRunUnits(productRun, 1, 0)
    expect(productRun.Run).toEqual([{ PresentationID: 2, Units: 3 }])
  })

  it('groups order lines by product and flattens them back, dropping empty sizes', () => {
    const productRuns = orderLinesToProductRuns([orderLine(SANDAL_ID, 1, 4), orderLine(BOOT_ID, 4, 2), orderLine(SANDAL_ID, 2, 6)])
    expect(productRuns).toEqual([
      { ProductID: SANDAL_ID, Run: [{ PresentationID: 1, Units: 4 }, { PresentationID: 2, Units: 6 }] },
      { ProductID: BOOT_ID, Run: [{ PresentationID: 4, Units: 2 }] },
    ])
    productRuns[1].Run[0].Units = 0
    productRuns.push({ ProductID: 0, Run: [{ PresentationID: 0, Units: 5 }] })
    expect(productRunsToOrderLines(productRuns).map((line) => [line.ProductID, line.PresentationID, line.PlannedUnits]))
      .toEqual([[SANDAL_ID, 1, 4], [SANDAL_ID, 2, 6]])
  })
})

describe('template ↔ order', () => {
  it('plans Run.Units × series per product and size', () => {
    const lines = expandTemplateToOrderLines([{ ProductID: SANDAL_ID, Run: [{ PresentationID: 1, Units: 1 }, { PresentationID: 2, Units: 2 }] }], 40)
    expect(lines.map((line) => [line.PresentationID, line.PlannedUnits, line.ProducedUnits])).toEqual([[1, 40, 0], [2, 80, 0]])
  })

  it('saves the order back as one series: units ÷ series, rounded and at least 1', () => {
    const templateLines = orderLinesToTemplateLines([orderLine(SANDAL_ID, 1, 40), orderLine(SANDAL_ID, 2, 81), orderLine(BOOT_ID, 4, 3)], 40)
    expect(templateLines).toEqual([
      { ProductID: SANDAL_ID, Run: [{ PresentationID: 1, Units: 1 }, { PresentationID: 2, Units: 2 }] },
      { ProductID: BOOT_ID, Run: [{ PresentationID: 4, Units: 1 }] },
    ])
  })

  it('takes the order as one series when it has none', () => {
    expect(orderLinesToTemplateLines([orderLine(BOOT_ID, 5, 7)], 0)).toEqual([{ ProductID: BOOT_ID, Run: [{ PresentationID: 5, Units: 7 }] }])
  })
})

describe('explodeProductionLines', () => {
  it('mirrors the backend: ceil(Σ units × qty ÷ yield × (1 + waste) × (1 + adjustment)) and labor minutes', () => {
    const exploded = explodeProductionLines([
      { ProductID: SANDAL_ID, PlannedUnits: 10 }, { ProductID: BOOT_ID, PlannedUnits: 3 }, { ProductID: SOLE_ID, PlannedUnits: 5 },
    ], 1000, productsByID)
    // Leather (1020 packed = 1 m² + 20 = 120 sub-units): sandals 10 × 120 ÷ 2 × 1.1 waste = 660, boots
    // 3 × 50 = 150; × 1.1 adjustment = 891. Glue: 10 × 1 ÷ 2 × 1.1 = 5.5 → 6.
    expect(exploded.supplies).toEqual([
      { SupplyID: LEATHER_ID, SubDivisor: 100, Planned: 891 },
      { SupplyID: GLUE_ID, SubDivisor: 1, Planned: 6 },
    ])
    expect(exploded.labor).toEqual([{ LaborTypeID: SHOEMAKER_ID, PlannedMinutes: 150 }])
  })

  it('does not ask an extra sub-unit for a float sum just above a whole number', () => {
    const product = recipeProduct({ ID: 50, ProductionYieldPortions: 3, ProductionSupplies: [{ SupplyID: GLUE_ID, Quantity: 1000, SubDivisor: 1 }] })
    const exploded = explodeProductionLines([{ ProductID: 50, PlannedUnits: 3 }], 0, new Map([[50, product]]))
    expect(exploded.supplies[0].Planned).toBe(1)
  })
})

describe('proposeReceiptConsumptions', () => {
  const lines = [orderLine(SANDAL_ID, 1, 30), orderLine(SANDAL_ID, 2, 70)]
  const supplies = [orderSupply(LEATHER_ID, 783, 300, 100), orderSupply(GLUE_ID, 6, 6)]

  it('proposes Planned × (good + scrapped) ÷ total planned units, rounded', () => {
    const receiptLines = makeReceiptEntryLines(lines)
    receiptLines[0].Units = 20
    receiptLines[1].Units = 4
    receiptLines[1].Scrapped = 1
    // 25 of 100 units: leather 195.75 → 196, glue 1.5 → 2.
    expect(proposeReceiptConsumptions(supplies, lines, receiptLines, false)).toEqual([
      { SupplyID: LEATHER_ID, SubDivisor: 100, SubUnits: 196 },
      { SupplyID: GLUE_ID, SubDivisor: 1, SubUnits: 2 },
    ])
  })

  it('proposes the remainder on the last receipt, never below 0', () => {
    expect(proposeReceiptConsumptions(supplies, lines, makeReceiptEntryLines(lines), true).map((consumption) => consumption.SubUnits))
      .toEqual([483, 0])
  })

  it('refuses a receipt without good units or with negative quantities', () => {
    const receiptLines = makeReceiptEntryLines(lines)
    receiptLines[0].Scrapped = 2
    expect(findReceiptError(receiptLines, [])).toMatch(/good unit/)
    receiptLines[0].Units = 1
    expect(findReceiptError(receiptLines, [{ SupplyID: GLUE_ID, SubDivisor: 1, SubUnits: -1 }])).toMatch(/negative/)
    expect(findReceiptError(receiptLines, [{ SupplyID: GLUE_ID, SubDivisor: 1, SubUnits: 1 }])).toBeUndefined()
  })
})

describe('progress and Resumen', () => {
  const lines = [orderLine(SANDAL_ID, 1, 40, 30, 2, 9000), orderLine(SANDAL_ID, 2, 60, 66, 0, 19800), orderLine(BOOT_ID, 4, 100, 0)]

  it('reads progress as good units over planned units, past 100 on over-production', () => {
    expect(orderProgressPercent(lines)).toBe(48)
    expect(orderProgressPercent([orderLine(BOOT_ID, 4, 10, 12)])).toBe(120)
    expect(orderProgressPercent([])).toBe(0)
  })

  it('sums the sizes per product', () => {
    expect(unitsByProduct(lines)).toEqual([{ ProductID: SANDAL_ID, Units: 100 }, { ProductID: BOOT_ID, Units: 100 }])
  })

  it('compares planned vs produced units and estimated vs booked unit cost per product', () => {
    const rows = summarizeOrderProducts(lines, new Map([[SANDAL_ID, 290]]))
    expect(rows[0]).toEqual({
      ProductID: SANDAL_ID, PlannedUnits: 100, ProducedUnits: 96, ScrappedUnits: 2, yieldPercent: 96,
      estimatedUnitCost: 290, bookedUnitCost: 300,
    })
    expect(rows[1]).toMatchObject({ yieldPercent: 0, estimatedUnitCost: undefined, bookedUnitCost: undefined })
  })

  it('compares consumption against the planned share of what was made', () => {
    // 98 of 200 planned units made (good + scrapped): leather's share is 400 × 98 ÷ 200 = 196.
    const rows = summarizeOrderSupplies({ Lines: lines, Supplies: [orderSupply(LEATHER_ID, 400, 245, 100)] })
    expect(rows).toEqual([{ SupplyID: LEATHER_ID, SubDivisor: 100, Planned: 400, Consumed: 245, expectedForOutput: 196, usagePercent: 125 }])
  })

  it('compares recorded vs planned labor minutes', () => {
    const rows = summarizeOrderLabor([{ LaborTypeID: SHOEMAKER_ID, PlannedMinutes: 120, PlannedCost: 2000, RecordedMinutes: 90, RecordedCost: 1600 }])
    expect(rows[0].minutesPercent).toBe(75)
    expect(summarizeOrderLabor([{ LaborTypeID: 2, PlannedMinutes: 0, PlannedCost: 0, RecordedMinutes: 10, RecordedCost: 0 }])[0].minutesPercent)
      .toBeUndefined()
  })
})

describe('findOrderDraftError', () => {
  const draft = { WarehouseID: 1, SuppliesWarehouseID: 2, SupplyAdjustmentBasisPoints: 0, PlannedStart: 0, PlannedEnd: 0 }

  it('asks for both warehouses, a sane adjustment, ordered dates and some line', () => {
    expect(findOrderDraftError({ ...draft, SuppliesWarehouseID: 0 }, [orderLine(BOOT_ID, 4, 1)])).toMatch(/warehouse/)
    expect(findOrderDraftError({ ...draft, SupplyAdjustmentBasisPoints: -10000 }, [orderLine(BOOT_ID, 4, 1)])).toMatch(/adjustment/)
    expect(findOrderDraftError({ ...draft, PlannedStart: 20000, PlannedEnd: 19999 }, [orderLine(BOOT_ID, 4, 1)])).toMatch(/end/)
    expect(findOrderDraftError(draft, [])).toMatch(/product/)
    expect(findOrderDraftError(draft, [orderLine(BOOT_ID, 4, 1)])).toBeUndefined()
  })
})

describe('materialToSend', () => {
  it('sends the plan minus what is at the workshop and what was consumed, never below 0', () => {
    expect(materialToSend({ Planned: 1000, AtWorkshop: 300, Consumed: 200 })).toBe(500)
    expect(materialToSend({ Planned: 1000, AtWorkshop: 900, Consumed: 200 })).toBe(0)
  })
})
