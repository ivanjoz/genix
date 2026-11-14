import { describe, expect, it } from 'vitest'
import {
  estimateProductionUnitCost, clearProductionFields, copyProductionFields, findProductionError, makeEmptyRecipeLabor, makeEmptyRecipeSupply,
  pickRecipeSupply, recipeSupplyCost, setRecipeSupplyQuantity, setWastePercent, supplyAverageCosts,
} from './product-production.ts'

const RICE_ID = 10
const SALT_ID = 11
const SHOEMAKER_ID = 1

describe('supplyAverageCosts', () => {
  it('divides the summed value by the summed units over every warehouse, sub-units included', () => {
    const averageCosts = supplyAverageCosts([
      // 2 kg + 500 g at divisor 1000 worth 1000 cents, and 1.5 kg worth 600 cents: 1600 / 4 kg.
      { ProductID: RICE_ID, Quantity: 2, SubQuantity: 500, SubDivisor: 1000, StockValue: 1000 },
      { ProductID: RICE_ID, Quantity: 1, SubQuantity: 0, DetailQuantity: 0, DetailSubQuantity: 500, SubDivisor: 1000, StockValue: 600 },
    ])
    expect(averageCosts.get(RICE_ID)).toBe(400)
  })

  it('leaves out a supply with no units or no value', () => {
    const averageCosts = supplyAverageCosts([
      { ProductID: RICE_ID, Quantity: 0, SubQuantity: 0, StockValue: 0 },
      { ProductID: SALT_ID, Quantity: 3, SubQuantity: 0 },
    ])
    expect(averageCosts.size).toBe(0)
  })
})

describe('recipe lines', () => {
  it('normalizes sub-units into units and keeps the units when the supply changes', () => {
    const recipeSupply = makeEmptyRecipeSupply()
    pickRecipeSupply(recipeSupply, RICE_ID, 1000)
    setRecipeSupplyQuantity(recipeSupply, { units: 0, sub: 1500 })
    expect(recipeSupply.Quantity).toBe(1500) // 1 kg + 500 g
    pickRecipeSupply(recipeSupply, SALT_ID, 0)
    expect(recipeSupply).toEqual({ SupplyID: SALT_ID, Quantity: 1000, SubDivisor: 1, WasteBasisPoints: 0 })
  })

  it('values a line at the supply average, or not at all without one', () => {
    const recipeSupply = { SupplyID: RICE_ID, Quantity: 1250, SubDivisor: 1000 }
    expect(recipeSupplyCost(recipeSupply, 400)).toBe(500)
    expect(recipeSupplyCost(recipeSupply, undefined)).toBeUndefined()
  })

  it('adds the waste to the line value', () => {
    const recipeSupply = { SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1000 }
    setWastePercent(recipeSupply, 12.5)
    expect(recipeSupply).toMatchObject({ WasteBasisPoints: 1250 })
    expect(recipeSupplyCost(recipeSupply, 400)).toBe(450)
  })
})

describe('estimateProductionUnitCost', () => {
  const averageCosts = new Map([[RICE_ID, 400]])
  const noLaborRates = new Map<number, number>()
  const recipe = {
    ProductionYieldPortions: 5, ProductionExtraCostPerUnit: 30, ProductionStandardCost: 0,
    ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1250, SubDivisor: 1000 }], ProductionLabor: [],
  }

  it('spreads the recipe over its yield and adds the extra cost', () => {
    expect(estimateProductionUnitCost(recipe, averageCosts, noLaborRates)).toEqual({ unitCost: 130, isStandardCost: false, isIncomplete: false })
  })

  it('flags a recipe with an uncosted supply as incomplete', () => {
    const withSalt = { ...recipe, ProductionSupplies: [...recipe.ProductionSupplies, { SupplyID: SALT_ID, Quantity: 1000, SubDivisor: 1 }] }
    expect(estimateProductionUnitCost(withSalt, averageCosts, noLaborRates)?.isIncomplete).toBe(true)
  })

  it('lets the standard cost override the recipe', () => {
    expect(estimateProductionUnitCost({ ...recipe, ProductionStandardCost: 900 }, averageCosts, noLaborRates))
      .toEqual({ unitCost: 900, isStandardCost: true, isIncomplete: false })
  })

  it('adds the labor at its type rate, spread over the yield', () => {
    // 60 min of a 1500-cent hour for 5 units: 300 per unit on top of the 130.
    const withLabor = { ...recipe, ProductionLabor: [{ LaborTypeID: SHOEMAKER_ID, Minutes: 60 }] }
    expect(estimateProductionUnitCost(withLabor, averageCosts, new Map([[SHOEMAKER_ID, 1500]]))?.unitCost).toBe(430)
  })

  it('estimates a labor-only recipe (pure assembly)', () => {
    const assembly = { ...recipe, ProductionExtraCostPerUnit: 0, ProductionSupplies: [], ProductionLabor: [{ LaborTypeID: SHOEMAKER_ID, Minutes: 30 }] }
    expect(estimateProductionUnitCost(assembly, averageCosts, new Map([[SHOEMAKER_ID, 1000]]))?.unitCost).toBe(100)
  })

  it('has no estimate without a recipe or a standard cost', () => {
    expect(estimateProductionUnitCost({ ...clearProductionFields({}), ProductionExtraCostPerUnit: 50 }, averageCosts, noLaborRates)).toBeUndefined()
  })
})

describe('findProductionError', () => {
  it('accepts an empty configuration and a complete recipe', () => {
    expect(findProductionError(copyProductionFields({}))).toBeUndefined()
    expect(findProductionError({ ...clearProductionFields({}), ProductionYieldPortions: 4, ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1000 }] })).toBeUndefined()
  })

  it.each([
    ['negative cost', { ProductionExtraCostPerUnit: -1 }, 'negativos'],
    ['no yield', { ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1 }] }, 'unidades'],
    ['labor without yield', { ProductionLabor: [{ LaborTypeID: SHOEMAKER_ID, Minutes: 10 }] }, 'unidades'],
    ['waste over 100%', { ProductionYieldPortions: 1, ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1, WasteBasisPoints: 10001 }] }, 'merma'],
    ['unpicked labor type', { ProductionYieldPortions: 1, ProductionLabor: [makeEmptyRecipeLabor()] }, 'tipo de trabajo'],
    ['repeated labor type', {
      ProductionYieldPortions: 1,
      ProductionLabor: [{ LaborTypeID: SHOEMAKER_ID, Minutes: 10 }, { LaborTypeID: SHOEMAKER_ID, Minutes: 5 }],
    }, 'repetido'],
    ['labor without minutes', { ProductionYieldPortions: 1, ProductionLabor: [{ LaborTypeID: SHOEMAKER_ID, Minutes: 0 }] }, 'minutos'],
    ['unpicked supply', { ProductionYieldPortions: 1, ProductionSupplies: [makeEmptyRecipeSupply()] }, 'Elija'],
    ['repeated supply', {
      ProductionYieldPortions: 1,
      ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1 }, { SupplyID: RICE_ID, Quantity: 2000, SubDivisor: 1 }],
    }, 'repetido'],
    ['no quantity', { ProductionYieldPortions: 1, ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 0, SubDivisor: 1 }] }, 'cantidad'],
  ])('refuses %s', (_, override, expectedText) => {
    expect(findProductionError({ ...copyProductionFields({}), ...override })).toContain(expectedText)
  })
})

describe('production fields of the form', () => {
  it('copies with zero defaults and its own recipe lines, so editing the copy leaves the record alone', () => {
    const storedProduct = { ProductionYieldPortions: 4, ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1000 }] }
    const formFields = copyProductionFields(storedProduct)
    expect(formFields).toEqual({ ...storedProduct, ProductionExtraCostPerUnit: 0, ProductionStandardCost: 0, ProductionLabor: [] })
    formFields.ProductionSupplies[0].Quantity = 2000
    expect(storedProduct.ProductionSupplies[0].Quantity).toBe(1000)
  })

  it('clears every field of a product that is no longer made', () => {
    const product = { Name: 'Lomo', ProductionStandardCost: 900, ProductionSupplies: [{ SupplyID: RICE_ID, Quantity: 1000, SubDivisor: 1 }] }
    expect(clearProductionFields(product)).toEqual({
      Name: 'Lomo', ProductionYieldPortions: 0, ProductionSupplies: [], ProductionExtraCostPerUnit: 0, ProductionStandardCost: 0,
      ProductionLabor: [],
    })
  })
})
