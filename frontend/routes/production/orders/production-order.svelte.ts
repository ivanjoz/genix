import { productoUnidadOptions } from '#core/products-lists.ts'
import { formatQuantity } from '#core/quantity.ts'
import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
import { ProductStockSimpleService } from '#routes/logistics/products-stock/stock-movement.ts'
import { copyProductionFields, estimateProductionUnitCost, supplyAverageCosts } from '#routes/production/products/product-production.ts'
import {
  SUPPLY_ORIGIN_SEMI_FINISHED, SupplyMaterialService,
} from '#routes/production/supplies-materials/supply-material.svelte.ts'
import { ClientProviderService } from '#services/crm/client-provider.svelte.ts'
import { PRODUCT_ORIGIN_MADE, ProductsService } from '#services/production/products.svelte.ts'
import { ProductionOrdersService, ProductionTemplatesService } from '#services/production/production-orders.svelte.ts'
import { ProductionLaborTypesService, ProductionStagesService } from '#services/production/production-settings.svelte.ts'
import type { IRecipeProduct } from './production-order.ts'

// The catalogs the orders and templates pages read: the orders and templates themselves, what can be
// made (made products and semi-finished supplies, M11), the supplies, labor types, stages,
// warehouses, and the stock rows the supplies' average cost comes from.
export class ProductionCatalogs {
  orders: ProductionOrdersService | undefined
  templates = new ProductionTemplatesService(true)
  products = new ProductsService(true)
  supplies = new SupplyMaterialService(true)
  laborTypes = new ProductionLaborTypesService(true)
  stages = new ProductionStagesService(true)
  warehouses = new WarehousesService()
  stockRows = new ProductStockSimpleService(true)

  // An order line is a made product (Origin 2) or a semi-finished supply (Status 2 + Origin 2).
  recipeProductsByID = $derived(new Map<number, IRecipeProduct>([
    ...this.products.records.filter((product) => product.Origin === PRODUCT_ORIGIN_MADE)
      .map((product) => [product.ID, product] as [number, IRecipeProduct]),
    ...this.supplies.records.filter((supply) => supply.Origin === SUPPLY_ORIGIN_SEMI_FINISHED)
      .map((supply) => [supply.ID, supply] as [number, IRecipeProduct]),
  ]))
  recipeProductOptions = $derived([...this.recipeProductsByID.values()].sort((left, right) => left.Name.localeCompare(right.Name)))
  averageCostBySupplyID = $derived(supplyAverageCosts(this.stockRows.records))
  costPerHourByLaborTypeID = $derived(new Map(this.laborTypes.records.map((laborType) => [laborType.ID, laborType.CostPerHour])))

  // The providers (client_provider Type 2): a subcontracted order's workshop (§8).
  workshops: ClientProviderService | undefined

  constructor(withOrders: boolean) {
    // The templates page needs neither the orders nor the workshops.
    if (withOrders) {
      this.orders = new ProductionOrdersService(true)
      this.workshops = new ClientProviderService(2, true)
    }
  }

  workshopNameOf(subcontractorID: number): string {
    return this.workshops?.recordsMap.get(subcontractorID)?.Name || `#${subcontractorID}`
  }

  productNameOf(productID: number): string {
    return this.recipeProductsByID.get(productID)?.Name || this.products.recordsMap.get(productID)?.Name || `#${productID}`
  }

  presentationNameOf(productID: number, presentationID: number): string {
    if (!presentationID) return ''
    const product = this.recipeProductsByID.get(productID) || this.products.recordsMap.get(productID)
    return product?.Presentations?.find((presentation) => presentation.id === presentationID)?.nm || `#${presentationID}`
  }

  warehouseNameOf(warehouseID: number): string {
    return this.warehouses.AlmacenesMap.get(warehouseID)?.Name || ''
  }

  // "2 + 500 g" for sub-units of a supply at its divisor, with the unit's name.
  formatSupplyQuantity(supplyID: number, subUnits: number, subDivisor: number): string {
    const supply = this.supplies.recordsMap.get(supplyID)
    const unitName = productoUnidadOptions.find((unitOption) => unitOption.i === supply?.UnitID)?.v || ''
    const quantity = formatQuantity({ units: 0, sub: subUnits || 0 }, Math.max(subDivisor || 1, 1), supply?.SbuUnit)
    return unitName ? `${quantity} · ${unitName}` : quantity
  }

  // The estimated cost of one unit (the recipe at the supplies' average, or the standard cost),
  // undefined without a cost source.
  estimatedUnitCostOf(productID: number): number | undefined {
    const product = this.recipeProductsByID.get(productID)
    if (!product) return undefined
    return estimateProductionUnitCost(copyProductionFields(product), this.averageCostBySupplyID, this.costPerHourByLaborTypeID)?.unitCost
  }
}
