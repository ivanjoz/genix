import { describe, expect, it } from 'vitest'
import type { IRecipeProduct } from '#routes/production/orders/production-order.ts'
import { estimateSeriesCost, findTemplateError, makeEmptyTemplate } from './production-template.ts'

const SANDAL_ID = 1
const SOLE_ID = 2
const NO_RECIPE_ID = 3
const LEATHER_ID = 10

const recipeProduct = (product: Partial<IRecipeProduct> & { ID: number }): IRecipeProduct => ({
  Name: `#${product.ID}`, ProductionYieldPortions: 1, ProductionSupplies: [], ProductionExtraCostPerUnit: 0,
  ProductionStandardCost: 0, ProductionLabor: [], ...product,
})

const productsByID = new Map<number, IRecipeProduct>([
  // One pair: 2 m² of leather at 500 cents each, plus 100 cents of overhead = 1100.
  [SANDAL_ID, recipeProduct({ ID: SANDAL_ID, ProductionSupplies: [{ SupplyID: LEATHER_ID, Quantity: 2000, SubDivisor: 1 }], ProductionExtraCostPerUnit: 100 })],
  [SOLE_ID, recipeProduct({ ID: SOLE_ID, ProductionStandardCost: 300 })],
  [NO_RECIPE_ID, recipeProduct({ ID: NO_RECIPE_ID })],
])

describe('estimateSeriesCost', () => {
  it('sums the units of one series × each product estimated unit cost', () => {
    const estimate = estimateSeriesCost([
      { ProductID: SANDAL_ID, Run: [{ PresentationID: 1, Units: 1 }, { PresentationID: 2, Units: 2 }] },
      { ProductID: SOLE_ID, Run: [{ PresentationID: 0, Units: 4 }] },
    ], productsByID, new Map([[LEATHER_ID, 500]]), new Map())
    expect(estimate).toEqual({ totalCost: 3 * 1100 + 4 * 300, isIncomplete: false })
  })

  it('flags a product without a cost source or a supply without cost', () => {
    expect(estimateSeriesCost([{ ProductID: NO_RECIPE_ID, Run: [{ PresentationID: 0, Units: 1 }] }], productsByID, new Map(), new Map()))
      .toEqual({ totalCost: 0, isIncomplete: true })
    expect(estimateSeriesCost([{ ProductID: SANDAL_ID, Run: [{ PresentationID: 1, Units: 1 }] }], productsByID, new Map(), new Map()))
      .toEqual({ totalCost: 100, isIncomplete: true })
  })
})

describe('findTemplateError', () => {
  it('asks for a name, some product and units in every product', () => {
    const template = makeEmptyTemplate()
    expect(findTemplateError(template)).toMatch(/name/)
    template.Name = 'Zapatos Verano'
    expect(findTemplateError(template)).toMatch(/product/)
    template.Lines = [{ ProductID: 0, Run: [] }]
    expect(findTemplateError(template)).toMatch(/Pick the product/)
    template.Lines = [{ ProductID: SANDAL_ID, Run: [] }]
    expect(findTemplateError(template)).toMatch(/units/)
    template.Lines[0].Run = [{ PresentationID: 1, Units: 2 }]
    expect(findTemplateError(template)).toBeUndefined()
  })
})
