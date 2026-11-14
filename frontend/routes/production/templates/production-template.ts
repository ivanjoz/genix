import type { IProductionTemplate, IProductionTemplateLine } from '#services/production/production-orders.ts'
import { copyProductionFields, estimateProductionUnitCost } from '#routes/production/products/product-production.ts'
import { runTotalUnits, type IRecipeProduct } from '#routes/production/orders/production-order.ts'

// The rules of a production template (docs/MANUFACTURING_PLAN.md §4.1): what one series costs and
// what the form can refuse before the server does.

export const makeEmptyTemplate = (): IProductionTemplate => ({
  ID: 0, Name: '', Lines: [], DefaultSeries: 1, WarehouseID: 0, SuppliesWarehouseID: 0, Notes: '', ss: 1, upd: 0,
})

export interface ISeriesCostEstimate {
  // One series of every line, PEN cents.
  totalCost: number
  // Some product has no cost source, or a supply of its recipe has no cost yet: totalCost leaves it out.
  isIncomplete: boolean
}

// Σ units of one series × the estimated unit cost of each product (§2's estimate).
export const estimateSeriesCost = (
  templateLines: IProductionTemplateLine[], productsByID: Map<number, IRecipeProduct>,
  averageCostBySupplyID: Map<number, number>, costPerHourByLaborTypeID: Map<number, number>,
): ISeriesCostEstimate => {
  let totalCost = 0
  let isIncomplete = false
  for (const templateLine of templateLines) {
    const product = productsByID.get(templateLine.ProductID)
    const unitEstimate = product
      ? estimateProductionUnitCost(copyProductionFields(product), averageCostBySupplyID, costPerHourByLaborTypeID)
      : undefined
    if (!unitEstimate) { isIncomplete = true; continue }
    isIncomplete = isIncomplete || unitEstimate.isIncomplete
    totalCost += unitEstimate.unitCost * runTotalUnits(templateLine)
  }
  return { totalCost, isIncomplete }
}

// The first reason the server would refuse the template that the form can tell on its own.
export const findTemplateError = (template: IProductionTemplate): string | undefined => {
  if (template.Name.trim().length < 2) return 'The name needs at least 2 characters.|El nombre debe tener al menos 2 caracteres.'
  if (template.DefaultSeries < 0) return 'The default series cannot be negative.|Las series por defecto no pueden ser negativas.'
  if (!template.Lines.length) return 'Add at least one product.|Agregue al menos un producto.'
  if (template.Lines.some((templateLine) => !(templateLine.ProductID > 0))) {
    return 'Pick the product of every row.|Elija el producto de cada fila.'
  }
  if (template.Lines.some((templateLine) => runTotalUnits(templateLine) <= 0)) {
    return 'Every product needs units in some size.|Cada producto necesita unidades en alguna talla.'
  }
  return undefined
}
