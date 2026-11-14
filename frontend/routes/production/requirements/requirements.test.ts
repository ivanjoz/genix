import { describe, expect, it } from 'vitest'
import type { IRecipeProduct } from '#routes/production/orders/production-order.ts'
import type { IProductionOrder } from '#services/production/production-orders.ts'
import {
  buildPurchaseSuggestions, buildSemiFinishedDrafts, cheapestProviderOffer, computeRequirements, groupPurchaseOrdersByProvider,
  lastSubcontractOf, type IRequirementsInput, type IRequirementSupply, type ISupplyRequirement,
} from './requirements.ts'

const WAREHOUSE_ID = 1

// A recipe line: units*1000 + sub, at the supply's divisor.
const recipeLine = (SupplyID: number, Quantity: number, SubDivisor = 1, WasteBasisPoints = 0) =>
  ({ SupplyID, Quantity, SubDivisor, WasteBasisPoints })

const makeRecipeProduct = (ID: number, ProductionSupplies: ReturnType<typeof recipeLine>[], yieldPortions = 1): IRecipeProduct => ({
  ID, Name: `P${ID}`, ProductionYieldPortions: yieldPortions, ProductionSupplies, ProductionExtraCostPerUnit: 0,
  ProductionStandardCost: 0, ProductionLabor: [],
})

const makeSupply = (ID: number, SbuQuantity = 0, semiFinishedRecipe?: ReturnType<typeof recipeLine>[], yieldPortions = 1): IRequirementSupply => ({
  ...makeRecipeProduct(ID, semiFinishedRecipe || [], yieldPortions), SbuQuantity, Origin: semiFinishedRecipe ? 2 : 0,
})

const makeInput = (overrides: Partial<IRequirementsInput> & Pick<IRequirementsInput, 'batches'>): IRequirementsInput => ({
  warehouseID: WAREHOUSE_ID, recipeProductsByID: new Map(), suppliesByID: new Map(), stockRows: [], purchaseOrders: [],
  ...overrides,
})

const byID = <T extends { ID: number }>(...records: T[]) => new Map(records.map((record) => [record.ID, record]))
const bySupply = (requirements: ISupplyRequirement[], supplyID: number) => requirements.find((requirement) => requirement.SupplyID === supplyID)

describe('purchase requirements', () => {
  // Shoe: 2 kg leather (g) + 1 sole per 1 pair.
  const leather = makeSupply(10, 1000)
  const sole = makeSupply(11)
  const shoe = makeRecipeProduct(1, [recipeLine(10, 2000, 1000), recipeLine(11, 1000)])

  it('explodes a single level and nets nothing without stock', () => {
    const requirements = computeRequirements(makeInput({
      batches: [{ lines: [{ ProductID: 1, PlannedUnits: 3 }], adjustmentBasisPoints: 0 }],
      recipeProductsByID: byID(shoe), suppliesByID: byID(leather, sole),
    }))
    expect(bySupply(requirements, 10)).toMatchObject({ Need: 6000, Shortfall: 6000, ShortfallUnits: 6, Level: 1 })
    expect(bySupply(requirements, 11)).toMatchObject({ Need: 3, Shortfall: 3, ShortfallUnits: 3 })
  })

  it('applies the recipe waste and the order adjustment', () => {
    // 1 kg at 10% waste, 2 pairs per yield of 1, +5% order adjustment: 2 × 1000 × 1.1 × 1.05 = 2310 g.
    const wasteful = makeRecipeProduct(2, [recipeLine(10, 1000, 1000, 1000)])
    const requirements = computeRequirements(makeInput({
      batches: [{ lines: [{ ProductID: 2, PlannedUnits: 2 }], adjustmentBasisPoints: 500 }],
      recipeProductsByID: byID(wasteful), suppliesByID: byID(leather),
    }))
    expect(bySupply(requirements, 10)).toMatchObject({ Need: 2310, ShortfallUnits: 3 })
  })

  it('nets a semi-finished supply first, then explodes only its shortfall', () => {
    // Shoe uses 1 stitched upper; the upper (semi-finished) takes 500 g leather + 2 thread, yield 1.
    const upper = makeSupply(20, 0, [recipeLine(10, 500, 1000), recipeLine(12, 2000)])
    const thread = makeSupply(12)
    const shoeWithUpper = makeRecipeProduct(3, [recipeLine(20, 1000), recipeLine(10, 1000, 1000)])
    const requirements = computeRequirements(makeInput({
      batches: [{ lines: [{ ProductID: 3, PlannedUnits: 10 }], adjustmentBasisPoints: 0 }],
      recipeProductsByID: byID(shoeWithUpper), suppliesByID: byID(leather, thread, upper),
      stockRows: [{ WarehouseID: WAREHOUSE_ID, ProductID: 20, Quantity: 4, SubQuantity: 0 }],
    }))
    expect(bySupply(requirements, 20)).toMatchObject({ IsSemiFinished: true, Need: 10, Stock: 4, ShortfallUnits: 6, Level: 1 })
    // Leather: 10 kg for the shoes + 6 uppers × 500 g = 13 kg, netted after the upper (deeper level).
    expect(bySupply(requirements, 10)).toMatchObject({ Need: 13000, Level: 2, ShortfallUnits: 13 })
    expect(bySupply(requirements, 12)).toMatchObject({ Need: 12, ShortfallUnits: 12 })
  })

  it('explodes nothing below a semi-finished supply with enough stock', () => {
    const upper = makeSupply(20, 0, [recipeLine(12, 2000)])
    const requirements = computeRequirements(makeInput({
      batches: [{ lines: [{ ProductID: 3, PlannedUnits: 2 }], adjustmentBasisPoints: 0 }],
      recipeProductsByID: byID(makeRecipeProduct(3, [recipeLine(20, 1000)])), suppliesByID: byID(upper, makeSupply(12)),
      stockRows: [{ WarehouseID: WAREHOUSE_ID, ProductID: 20, Quantity: 5, SubQuantity: 0 }],
    }))
    expect(bySupply(requirements, 20)?.ShortfallUnits).toBe(0)
    expect(bySupply(requirements, 12)).toBeUndefined()
  })

  it('nets stock minus reserved and blocked, minus confirmed orders for this warehouse', () => {
    const requirements = computeRequirements(makeInput({
      batches: [{ lines: [{ ProductID: 1, PlannedUnits: 10 }], adjustmentBasisPoints: 0 }],
      recipeProductsByID: byID(shoe), suppliesByID: byID(leather, sole),
      stockRows: [
        // 8 kg free + 2 kg in lots, read at an older divisor of 100: 5 sub = 50 g. 3 kg reserved, 1 kg blocked.
        { WarehouseID: WAREHOUSE_ID, ProductID: 10, Quantity: 8, SubQuantity: 5, DetailQuantity: 2, SubDivisor: 100,
          ReservedProductionQuantity: 3, BlockedQuantity: 1 },
        // Another warehouse does not count.
        { WarehouseID: 2, ProductID: 10, Quantity: 100, SubQuantity: 0 },
      ],
      purchaseOrders: [
        { WarehouseID: WAREHOUSE_ID, ss: 2, Lines: [{ ProductID: 10, Quantity: 4 }] },
        { WarehouseID: 0, ss: 2, Lines: [{ ProductID: 10, Quantity: 1 }] },
        // Pending, or for another warehouse: not on order here.
        { WarehouseID: WAREHOUSE_ID, ss: 1, Lines: [{ ProductID: 10, Quantity: 50 }] },
        { WarehouseID: 2, ss: 2, Lines: [{ ProductID: 10, Quantity: 50 }] },
      ],
    }))
    // need 20000 − (10050 − 3000 − 1000) − 5000 = 8950 g → 9 kg.
    expect(bySupply(requirements, 10)).toMatchObject({
      Need: 20000, Stock: 10050, Reserved: 3000, Blocked: 1000, OnOrder: 5000, Shortfall: 8950, ShortfallUnits: 9,
    })
  })

  it('rounds a sub-unit shortfall up to a whole unit', () => {
    const requirements = computeRequirements(makeInput({
      batches: [{ lines: [{ ProductID: 4, PlannedUnits: 1 }], adjustmentBasisPoints: 0 }],
      recipeProductsByID: byID(makeRecipeProduct(4, [recipeLine(10, 300, 1000)])), suppliesByID: byID(leather),
    }))
    expect(bySupply(requirements, 10)).toMatchObject({ Shortfall: 300, ShortfallUnits: 1 })
  })

  it('adds the minimum stock when replenishing to the minimum', () => {
    const input = makeInput({
      batches: [{ lines: [{ ProductID: 1, PlannedUnits: 1 }], adjustmentBasisPoints: 0 }],
      recipeProductsByID: byID(shoe), suppliesByID: byID(leather, sole),
      stockRows: [{ WarehouseID: WAREHOUSE_ID, ProductID: 11, Quantity: 3, SubQuantity: 0 }],
    })
    expect(bySupply(computeRequirements(input), 11)?.ShortfallUnits).toBe(0)
    const replenished = computeRequirements({ ...input, minimumStockBySupplyID: new Map([[11, 5], [10, 1]]) })
    expect(bySupply(replenished, 11)).toMatchObject({ MinimumStock: 5, ShortfallUnits: 3 })
    expect(bySupply(replenished, 10)).toMatchObject({ MinimumStock: 1000, ShortfallUnits: 3 })
  })

  it('preselects the cheapest provider and groups the lines by provider', () => {
    expect(cheapestProviderOffer([{ ProviderID: 7, Price: 0 }, { ProviderID: 8, Price: 900 }, { ProviderID: 9, Price: 500 }]))
      .toEqual({ ProviderID: 9, Price: 500 })
    expect(cheapestProviderOffer([{ ProviderID: 7, Price: 0 }])).toEqual({ ProviderID: 7, Price: 0 })
    expect(cheapestProviderOffer(undefined)).toBeUndefined()

    const requirements = [
      { SupplyID: 10, IsSemiFinished: false, ShortfallUnits: 9 },
      { SupplyID: 11, IsSemiFinished: false, ShortfallUnits: 3 },
      { SupplyID: 12, IsSemiFinished: false, ShortfallUnits: 0 },
      { SupplyID: 20, IsSemiFinished: true, ShortfallUnits: 6 },
    ] as ISupplyRequirement[]
    const suggestions = buildPurchaseSuggestions(requirements, new Map([
      [10, [{ ProviderID: 8, Price: 2500 }, { ProviderID: 9, Price: 2000 }]],
      [11, [{ ProviderID: 9, Price: 300 }]],
    ]))
    // The semi-finished supply is made, never bought.
    expect(suggestions).toEqual([
      { SupplyID: 10, Units: 9, ProviderID: 9, Price: 2000 },
      { SupplyID: 11, Units: 3, ProviderID: 9, Price: 300 },
    ])
    suggestions[1].ProviderID = 8
    expect(groupPurchaseOrdersByProvider(suggestions, WAREHOUSE_ID)).toEqual([
      { ProviderID: 9, WarehouseID: WAREHOUSE_ID, Lines: [{ ProductID: 10, PresentationID: 0, Quantity: 9, Price: 2000 }] },
      { ProviderID: 8, WarehouseID: WAREHOUSE_ID, Lines: [{ ProductID: 11, PresentationID: 0, Quantity: 3, Price: 300 }] },
    ])
  })

  it('drafts the semi-finished shortfalls with their last workshop', () => {
    const previousOrders = [
      { ID: 5, SubcontractorID: 30, SubcontractPricePerUnit: 150, Lines: [{ ProductID: 20 }] },
      { ID: 9, SubcontractorID: 31, SubcontractPricePerUnit: 180, Lines: [{ ProductID: 20 }] },
      { ID: 12, SubcontractorID: 0, SubcontractPricePerUnit: 0, Lines: [{ ProductID: 20 }] },
    ] as IProductionOrder[]
    expect(lastSubcontractOf(20, previousOrders)).toEqual({ SubcontractorID: 31, SubcontractPricePerUnit: 180 })
    expect(lastSubcontractOf(21, previousOrders)).toBeUndefined()

    const drafts = buildSemiFinishedDrafts([
      { SupplyID: 20, IsSemiFinished: true, ShortfallUnits: 6 },
      { SupplyID: 21, IsSemiFinished: true, ShortfallUnits: 2 },
      { SupplyID: 10, IsSemiFinished: false, ShortfallUnits: 4 },
    ] as ISupplyRequirement[], WAREHOUSE_ID, previousOrders)
    expect(drafts).toHaveLength(2)
    expect(drafts[0]).toMatchObject({ ID: 0, ss: 1, WarehouseID: 1, SuppliesWarehouseID: 1, SubcontractorID: 31, SubcontractPricePerUnit: 180 })
    expect(drafts[0].Lines).toEqual([expect.objectContaining({ ProductID: 20, PresentationID: 0, PlannedUnits: 6 })])
    expect(drafts[1]).toMatchObject({ SubcontractorID: 0, Lines: [expect.objectContaining({ ProductID: 21, PlannedUnits: 2 })] })
  })
})
