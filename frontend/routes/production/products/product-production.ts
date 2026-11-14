import {
  packQuantityLine, quantityDivisorOf, unpackQuantityLine, type Quantity,
} from '#core/quantity.ts'
import type { IProduct, IProductionLabor, IProductionSupply } from '#services/production/products.svelte.ts'

// The production configuration of a made product or a semi-finished supply (docs/MANUFACTURING_PLAN.md
// §2), its Production* fields: a recipe of supplies (with their waste) and labor minutes for
// ProductionYieldPortions units, an overhead cost per unit, and an optional standard cost that
// overrides the calculated one. Costs are PEN cents.

export type IProductionFields = Pick<IProduct,
  'ProductionYieldPortions' | 'ProductionSupplies' | 'ProductionExtraCostPerUnit' | 'ProductionStandardCost' | 'ProductionLabor'>

// 100% in basis points: a waste of 1500 is 15%.
export const BASIS_POINTS_WHOLE = 10000

// A copy of a product's production fields, 0 / [] where unset (the server omits zero fields). Each
// recipe line is copied too, so the form editing the copy never touches the cached record.
export const copyProductionFields = (product: Partial<IProductionFields>): IProductionFields => ({
  ProductionYieldPortions: product.ProductionYieldPortions || 0,
  ProductionSupplies: (product.ProductionSupplies || []).map((recipeSupply) => ({ ...recipeSupply })),
  ProductionExtraCostPerUnit: product.ProductionExtraCostPerUnit || 0,
  ProductionStandardCost: product.ProductionStandardCost || 0,
  ProductionLabor: (product.ProductionLabor || []).map((recipeLabor) => ({ ...recipeLabor })),
})

// Clears the production fields of a product: a new one, or one that is no longer made.
export const clearProductionFields = <T extends Partial<IProductionFields>>(product: T): T =>
  Object.assign(product, copyProductionFields({}))

// A recipe line starts on no supply; picking one sets its divisor.
export const makeEmptyRecipeSupply = (): IProductionSupply => ({ SupplyID: 0, Quantity: 0, SubDivisor: 1, WasteBasisPoints: 0 })

export const makeEmptyRecipeLabor = (): IProductionLabor => ({ LaborTypeID: 0, Minutes: 0 })

// The parts of a stock row the supply average reads (IProductStock of products-stock).
export interface ISupplyStockRow {
  ProductID: number
  Quantity: number
  SubQuantity: number
  DetailQuantity?: number
  DetailSubQuantity?: number
  SubDivisor?: number
  StockValue?: number
}

// The average cost of one whole unit of each supply over every warehouse: Σ StockValue ÷ Σ units
// held. A supply with no units or no value has no entry, so its recipe lines show "sin costo". The
// close values the real consumption at each row's own value; this is only the form's estimate.
export const supplyAverageCosts = (stockRows: ISupplyStockRow[]): Map<number, number> => {
  const totalsBySupplyID = new Map<number, { units: number, value: number }>()
  for (const stockRow of stockRows) {
    // Every row carries its own divisor, so a refined sub-unit reads correctly on old rows.
    const divisor = quantityDivisorOf(stockRow.SubDivisor)
    const units = (stockRow.Quantity || 0) + (stockRow.DetailQuantity || 0)
      + ((stockRow.SubQuantity || 0) + (stockRow.DetailSubQuantity || 0)) / divisor
    const totals = totalsBySupplyID.get(stockRow.ProductID) || { units: 0, value: 0 }
    totals.units += units
    totals.value += stockRow.StockValue || 0
    totalsBySupplyID.set(stockRow.ProductID, totals)
  }
  const averageCostBySupplyID = new Map<number, number>()
  for (const [supplyID, totals] of totalsBySupplyID) {
    if (totals.units > 0 && totals.value > 0) averageCostBySupplyID.set(supplyID, totals.value / totals.units)
  }
  return averageCostBySupplyID
}

export const recipeSupplyQuantity = (recipeSupply: IProductionSupply): Quantity =>
  unpackQuantityLine(recipeSupply.Quantity)

// Picking another supply keeps the whole units and drops the sub-units: they were in the previous
// supply's sub-unit.
export const pickRecipeSupply = (recipeSupply: IProductionSupply, supplyID: number, supplySubQuantity?: number) => {
  recipeSupply.SupplyID = supplyID
  recipeSupply.SubDivisor = quantityDivisorOf(supplySubQuantity)
  recipeSupply.Quantity = packQuantityLine({ units: recipeSupplyQuantity(recipeSupply).units, sub: 0 }, recipeSupply.SubDivisor)
}

// Units and sub-units are edited apart; packing normalizes, so 1500 g typed on a kg supply becomes
// 1 kg + 500 g.
export const setRecipeSupplyQuantity = (recipeSupply: IProductionSupply, quantity: Quantity) => {
  recipeSupply.Quantity = packQuantityLine({ units: Math.max(quantity.units, 0), sub: Math.max(quantity.sub, 0) }, recipeSupply.SubDivisor)
}

// The value of a recipe line at the supply's average, waste included, or undefined when the supply
// has no cost yet.
export const recipeSupplyCost = (recipeSupply: IProductionSupply, averageUnitCost: number | undefined): number | undefined => {
  if (averageUnitCost === undefined) return undefined
  const quantity = recipeSupplyQuantity(recipeSupply)
  const wasteFactor = 1 + (recipeSupply.WasteBasisPoints || 0) / BASIS_POINTS_WHOLE
  return (quantity.units + quantity.sub / recipeSupply.SubDivisor) * wasteFactor * averageUnitCost
}

// The waste is edited as a percentage with up to two decimals; it is stored in basis points.
export const wastePercentOf = (recipeSupply: IProductionSupply) => (recipeSupply.WasteBasisPoints || 0) / 100
export const setWastePercent = (recipeSupply: IProductionSupply, percent: number) => {
  recipeSupply.WasteBasisPoints = Math.round((percent || 0) * 100)
}

// The value of a labor line at its type's estimate rate: minutes × CostPerHour ÷ 60.
export const recipeLaborCost = (recipeLabor: IProductionLabor, costPerHour: number | undefined): number =>
  (recipeLabor.Minutes || 0) * (costPerHour || 0) / 60

export interface IProductionCostEstimate {
  // Cost of one unit of the product, PEN cents.
  unitCost: number
  isStandardCost: boolean
  // Some supply of the recipe has no cost yet, so unitCost leaves it out.
  isIncomplete: boolean
}

// The estimated cost of one unit (docs/MANUFACTURING_PLAN.md §2): the standard cost when set (it
// overrides everything, P3), else
//   Σ supplies (quantity × (1 + waste) × average cost) ÷ yield
//   + Σ labor (minutes × the type's CostPerHour) ÷ yield
//   + the overhead per unit.
// undefined when there is no cost source, which production refuses.
export const estimateProductionUnitCost = (
  product: IProductionFields, averageCostBySupplyID: Map<number, number>, costPerHourByLaborTypeID: Map<number, number>,
): IProductionCostEstimate | undefined => {
  if (product.ProductionStandardCost > 0) {
    return { unitCost: product.ProductionStandardCost, isStandardCost: true, isIncomplete: false }
  }
  const recipeSupplies = product.ProductionSupplies.filter((recipeSupply) => recipeSupply.SupplyID > 0)
  const recipeLabor = (product.ProductionLabor || []).filter((labor) => labor.LaborTypeID > 0)
  if ((recipeSupplies.length === 0 && recipeLabor.length === 0) || !(product.ProductionYieldPortions > 0)) return undefined

  let recipeValue = 0
  let isIncomplete = false
  for (const recipeSupply of recipeSupplies) {
    const lineCost = recipeSupplyCost(recipeSupply, averageCostBySupplyID.get(recipeSupply.SupplyID))
    if (lineCost === undefined) isIncomplete = true
    else recipeValue += lineCost
  }
  for (const labor of recipeLabor) {
    recipeValue += recipeLaborCost(labor, costPerHourByLaborTypeID.get(labor.LaborTypeID))
  }
  return {
    unitCost: Math.round(recipeValue / product.ProductionYieldPortions + (product.ProductionExtraCostPerUnit || 0)),
    isStandardCost: false,
    isIncomplete,
  }
}

// The first reason the server would refuse this configuration that the form can tell on its own,
// so the user fixes it on the tab instead of reading a save error. The server still validates all.
export const findProductionError = (product: IProductionFields): string | undefined => {
  if (product.ProductionExtraCostPerUnit < 0 || product.ProductionStandardCost < 0) {
    return 'Production costs cannot be negative.|Los costos de producción no pueden ser negativos.'
  }
  const recipeLabor = product.ProductionLabor || []
  if ((product.ProductionSupplies.length > 0 || recipeLabor.length > 0) && !(product.ProductionYieldPortions > 0)) {
    return 'Enter how many units the recipe yields.|Indique cuántas unidades rinde la receta.'
  }
  const seenSupplyIDs = new Set<number>()
  for (const recipeSupply of product.ProductionSupplies) {
    if (!recipeSupply.SupplyID) return 'Pick the supply of every recipe line.|Elija el insumo de cada línea de la receta.'
    if (seenSupplyIDs.has(recipeSupply.SupplyID)) return 'A supply is repeated in the recipe.|Un insumo está repetido en la receta.'
    seenSupplyIDs.add(recipeSupply.SupplyID)
    if (recipeSupply.Quantity <= 0) return 'Every recipe line needs a quantity.|Cada línea de la receta necesita una cantidad.'
    const waste = recipeSupply.WasteBasisPoints || 0
    if (waste < 0 || waste > BASIS_POINTS_WHOLE) return 'Waste must be between 0% and 100%.|La merma debe estar entre 0% y 100%.'
  }
  const seenLaborTypeIDs = new Set<number>()
  for (const labor of recipeLabor) {
    if (!labor.LaborTypeID) return 'Pick the labor type of every labor line.|Elija el tipo de trabajo de cada línea de mano de obra.'
    if (seenLaborTypeIDs.has(labor.LaborTypeID)) return 'A labor type is repeated in the recipe.|Un tipo de trabajo está repetido en la receta.'
    seenLaborTypeIDs.add(labor.LaborTypeID)
    if (!(labor.Minutes > 0)) return 'Every labor line needs minutes.|Cada línea de mano de obra necesita minutos.'
  }
  return undefined
}
