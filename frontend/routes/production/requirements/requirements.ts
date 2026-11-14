import { quantityDivisorOf } from '#core/quantity.ts'
import { explodeProductionLines, makeEmptyOrder, type IPlannedLine, type IRecipeProduct } from '#routes/production/orders/production-order.ts'
import type { IProductionOrder } from '#services/production/production-orders.ts'

// The pure rules of the purchase requirements page (docs/MANUFACTURING_PLAN.md §7, MRP): explode what
// is planned into supplies through every recipe level, net each supply against the warehouse, and turn
// the shortfalls into purchase lines (bought supplies) and draft production orders (semi-finished ones).
// All quantities are sub-units of the supply's current divisor unless a name says "Units".

// A semi-finished supply (§2.1) has this Origin: the company makes it, so it is never bought.
export const SEMI_FINISHED_ORIGIN = 2
// Purchase orders in this status are promised by the provider and not received yet.
export const PURCHASE_ORDER_CONFIRMED = 2
// Recipes are acyclic and at most 5 deep (§2.1); the walk stops one level further as a safety net.
const MAX_RECIPE_DEPTH = 6
// Keeps a float sum like 1500.0000000002 from asking a whole extra sub-unit, as the release does.
const FLOAT_TOLERANCE = 1e-6

// A supply as the explosion reads it: its divisor and, when semi-finished, its own recipe.
export type IRequirementSupply = IRecipeProduct & { SbuQuantity?: number, Origin?: number }

// The parts of a ProductStock row the netting reads. Blocked* are optional: rows written before the
// lot-status work carry none.
export interface IRequirementStockRow {
  WarehouseID: number
  ProductID: number
  Quantity: number
  SubQuantity: number
  DetailQuantity?: number
  DetailSubQuantity?: number
  SubDivisor?: number
  ReservedProductionQuantity?: number
  ReservedProductionSubQuantity?: number
  BlockedQuantity?: number
  BlockedSubQuantity?: number
}

export interface IRequirementPurchaseOrder {
  WarehouseID: number
  ss: number
  Lines?: { ProductID: number, Quantity: number }[]
}

// What is to be produced: the lines of one draft order (with its supply adjustment) or of a template ×
// series (no adjustment).
export interface IDemandBatch {
  lines: Pick<IPlannedLine, 'ProductID' | 'PlannedUnits'>[]
  adjustmentBasisPoints: number
}

export interface IRequirementsInput {
  batches: IDemandBatch[]
  warehouseID: number
  // Made products and semi-finished supplies by ID: whatever an order line can name.
  recipeProductsByID: Map<number, IRecipeProduct>
  suppliesByID: Map<number, IRequirementSupply>
  stockRows: IRequirementStockRow[]
  purchaseOrders: IRequirementPurchaseOrder[]
  // Whole units per supply ("reponer al mínimo"); undefined when the box is off.
  minimumStockBySupplyID?: Map<number, number>
}

export interface ISupplyRequirement {
  SupplyID: number
  SubDivisor: number
  IsSemiFinished: boolean
  // The recipe level the supply first fully settles at: 1 = asked by the planned products directly.
  Level: number
  Need: number
  MinimumStock: number
  Stock: number
  Reserved: number
  Blocked: number
  OnOrder: number
  // need + minimum − (stock − reserved − blocked) − on order, never below 0.
  Shortfall: number
  // The shortfall in whole units, rounded up: a purchase line and an order line carry no sub-units.
  ShortfallUnits: number
}

// A row's (units, sub) pair at its own divisor, in sub-units of the supply's current divisor. A row
// written before the supply's sub-unit was refined still reads correctly.
const toSupplySubUnits = (units: number, sub: number, rowDivisor: number, supplyDivisor: number): number =>
  (units || 0) * supplyDivisor + (sub || 0) * supplyDivisor / rowDivisor

const isSemiFinished = (supply: IRequirementSupply | undefined): boolean => supply?.Origin === SEMI_FINISHED_ORIGIN

// The deepest recipe level each supply is reached at. A supply used both directly and inside a
// semi-finished one settles at the deeper level, so every parent's shortfall has been added to its
// need before it is netted (the "low-level code" of MRP).
const supplyLevels = (rootProductIDs: number[], input: IRequirementsInput): Map<number, number> => {
  const levelBySupplyID = new Map<number, number>()
  const visit = (productID: number, level: number) => {
    if (level > MAX_RECIPE_DEPTH) return
    const recipeProduct = input.recipeProductsByID.get(productID) || input.suppliesByID.get(productID)
    for (const recipeSupply of recipeProduct?.ProductionSupplies || []) {
      if (!recipeSupply.SupplyID) continue
      if ((levelBySupplyID.get(recipeSupply.SupplyID) || 0) < level) levelBySupplyID.set(recipeSupply.SupplyID, level)
      if (isSemiFinished(input.suppliesByID.get(recipeSupply.SupplyID))) visit(recipeSupply.SupplyID, level + 1)
    }
  }
  for (const productID of new Set(rootProductIDs)) visit(productID, 1)
  return levelBySupplyID
}

// Explodes lines through one recipe level and adds what they ask to grossNeedBySupplyID, converted to
// each supply's current divisor.
const addExplodedNeed = (
  grossNeedBySupplyID: Map<number, number>, batch: IDemandBatch, input: IRequirementsInput,
) => {
  const recipeProductsByID = new Map([...input.recipeProductsByID, ...input.suppliesByID])
  for (const exploded of explodeProductionLines(batch.lines, batch.adjustmentBasisPoints, recipeProductsByID).supplies) {
    const supplyDivisor = quantityDivisorOf(input.suppliesByID.get(exploded.SupplyID)?.SbuQuantity)
    const subUnits = exploded.Planned * supplyDivisor / quantityDivisorOf(exploded.SubDivisor)
    grossNeedBySupplyID.set(exploded.SupplyID, (grossNeedBySupplyID.get(exploded.SupplyID) || 0) + subUnits)
  }
}

// §7 steps 2, 3 and 5. Every supply the plan reaches gets a row, short or not, so the page can show
// why nothing is suggested. Sorted by level, then supply ID.
export const computeRequirements = (input: IRequirementsInput): ISupplyRequirement[] => {
  const grossNeedBySupplyID = new Map<number, number>()
  for (const batch of input.batches) addExplodedNeed(grossNeedBySupplyID, batch, input)

  const levelBySupplyID = supplyLevels(input.batches.flatMap((batch) => batch.lines.map((line) => line.ProductID)), input)
  const supplyIDsByLevel = [...levelBySupplyID.keys()]
    .sort((leftID, rightID) => (levelBySupplyID.get(leftID)! - levelBySupplyID.get(rightID)!) || leftID - rightID)

  const requirements: ISupplyRequirement[] = []
  for (const supplyID of supplyIDsByLevel) {
    const supply = input.suppliesByID.get(supplyID)
    const subDivisor = quantityDivisorOf(supply?.SbuQuantity)
    const need = grossNeedBySupplyID.get(supplyID) || 0
    const minimumStock = (input.minimumStockBySupplyID?.get(supplyID) || 0) * subDivisor
    // A semi-finished supply no parent is short of asks nothing: it is not part of this plan.
    if (need <= 0 && minimumStock <= 0) continue

    let stock = 0, reserved = 0, blocked = 0
    for (const stockRow of input.stockRows) {
      if (stockRow.WarehouseID !== input.warehouseID || stockRow.ProductID !== supplyID) continue
      const rowDivisor = quantityDivisorOf(stockRow.SubDivisor)
      stock += toSupplySubUnits((stockRow.Quantity || 0) + (stockRow.DetailQuantity || 0),
        (stockRow.SubQuantity || 0) + (stockRow.DetailSubQuantity || 0), rowDivisor, subDivisor)
      reserved += toSupplySubUnits(stockRow.ReservedProductionQuantity || 0, stockRow.ReservedProductionSubQuantity || 0, rowDivisor, subDivisor)
      blocked += toSupplySubUnits(stockRow.BlockedQuantity || 0, stockRow.BlockedSubQuantity || 0, rowDivisor, subDivisor)
    }

    // A confirmed order with no warehouse yet may still be received here, so it counts.
    let onOrder = 0
    for (const purchaseOrder of input.purchaseOrders) {
      if (purchaseOrder.ss !== PURCHASE_ORDER_CONFIRMED) continue
      if (purchaseOrder.WarehouseID && purchaseOrder.WarehouseID !== input.warehouseID) continue
      for (const line of purchaseOrder.Lines || []) {
        if (line.ProductID === supplyID) onOrder += (line.Quantity || 0) * subDivisor
      }
    }

    const shortfall = Math.max(Math.ceil(need + minimumStock - (stock - reserved - blocked) - onOrder - FLOAT_TOLERANCE), 0)
    const shortfallUnits = Math.ceil(shortfall / subDivisor)
    requirements.push({
      SupplyID: supplyID, SubDivisor: subDivisor, IsSemiFinished: isSemiFinished(supply), Level: levelBySupplyID.get(supplyID)!,
      Need: Math.ceil(need - FLOAT_TOLERANCE), MinimumStock: minimumStock, Stock: stock, Reserved: reserved, Blocked: blocked,
      OnOrder: onOrder, Shortfall: shortfall, ShortfallUnits: shortfallUnits,
    })

    // A semi-finished shortfall is made, in whole units, so its own recipe asks for them. Its
    // supplies sit at deeper levels and are netted after this.
    if (isSemiFinished(supply) && shortfallUnits > 0) {
      addExplodedNeed(grossNeedBySupplyID, { lines: [{ ProductID: supplyID, PlannedUnits: shortfallUnits }], adjustmentBasisPoints: 0 }, input)
    }
  }
  return requirements
}

// ---------------------------------------------------------------------------------------------------
// To buy
// ---------------------------------------------------------------------------------------------------

export interface IProviderPrice {
  ProviderID: number
  // PEN cents per whole unit.
  Price: number
}

// One suggested purchase line. Provider and price start at the cheapest offer and are editable.
export interface IPurchaseSuggestion {
  SupplyID: number
  Units: number
  ProviderID: number
  Price: number
}

// The provider with the lowest known price; one with no price only when none has a price.
export const cheapestProviderOffer = (offers: IProviderPrice[] | undefined): IProviderPrice | undefined => {
  const knownProviders = (offers || []).filter((offer) => offer.ProviderID > 0)
  const pricedOffers = knownProviders.filter((offer) => offer.Price > 0).sort((left, right) => left.Price - right.Price)
  return pricedOffers[0] || knownProviders[0]
}

export const buildPurchaseSuggestions = (
  requirements: ISupplyRequirement[], providerOffersBySupplyID: Map<number, IProviderPrice[]>,
): IPurchaseSuggestion[] => requirements
  .filter((requirement) => !requirement.IsSemiFinished && requirement.ShortfallUnits > 0)
  .map((requirement) => {
    const cheapestOffer = cheapestProviderOffer(providerOffersBySupplyID.get(requirement.SupplyID))
    return {
      SupplyID: requirement.SupplyID, Units: requirement.ShortfallUnits,
      ProviderID: cheapestOffer?.ProviderID || 0, Price: cheapestOffer?.Price || 0,
    }
  })

// The price a provider offers for a supply: picking another provider re-prices the line.
export const providerPriceOf = (offers: IProviderPrice[] | undefined, providerID: number): number =>
  (offers || []).find((offer) => offer.ProviderID === providerID)?.Price || 0

export const findPurchaseSuggestionsError = (suggestions: IPurchaseSuggestion[]): string | undefined => {
  if (!suggestions.some((suggestion) => suggestion.Units > 0)) return 'There is nothing to buy.|No hay nada que comprar.'
  if (suggestions.some((suggestion) => suggestion.Units > 0 && !(suggestion.ProviderID > 0))) {
    return 'Pick the provider of every line.|Elija el proveedor de cada línea.'
  }
  return undefined
}

// The body of one POST purchase-orders: a supply line has presentation 0 and whole units.
export interface IPurchaseOrderDraft {
  ProviderID: number
  WarehouseID: number
  Lines: { ProductID: number, PresentationID: number, Quantity: number, Price: number }[]
}

// "Crear órdenes de compra": one order per provider, lines with no units left out.
export const groupPurchaseOrdersByProvider = (suggestions: IPurchaseSuggestion[], warehouseID: number): IPurchaseOrderDraft[] => {
  const ordersByProviderID = new Map<number, IPurchaseOrderDraft>()
  for (const suggestion of suggestions) {
    if (!(suggestion.Units > 0) || !(suggestion.ProviderID > 0)) continue
    const order = ordersByProviderID.get(suggestion.ProviderID) || { ProviderID: suggestion.ProviderID, WarehouseID: warehouseID, Lines: [] }
    order.Lines.push({ ProductID: suggestion.SupplyID, PresentationID: 0, Quantity: suggestion.Units, Price: suggestion.Price })
    ordersByProviderID.set(suggestion.ProviderID, order)
  }
  return [...ordersByProviderID.values()]
}

// ---------------------------------------------------------------------------------------------------
// To make
// ---------------------------------------------------------------------------------------------------

// The workshop of the newest order of the product that was subcontracted (§8: a subcontracted product
// keeps its last workshop), or undefined when it was always made in-house.
export const lastSubcontractOf = (
  productID: number, orders: Pick<IProductionOrder, 'ID' | 'SubcontractorID' | 'SubcontractPricePerUnit' | 'Lines'>[],
): { SubcontractorID: number, SubcontractPricePerUnit: number } | undefined => {
  const lastOrder = [...orders].sort((left, right) => right.ID - left.ID).find((order) =>
    order.SubcontractorID > 0 && (order.Lines || []).some((line) => line.ProductID === productID))
  return lastOrder && { SubcontractorID: lastOrder.SubcontractorID, SubcontractPricePerUnit: lastOrder.SubcontractPricePerUnit || 0 }
}

// "Crear órdenes": one draft per semi-finished shortfall. A supply has no presentations (0), and the
// supplies come from and the product enters the same warehouse.
export const buildSemiFinishedDrafts = (
  requirements: ISupplyRequirement[], warehouseID: number, previousOrders: IProductionOrder[],
): IProductionOrder[] => requirements
  .filter((requirement) => requirement.IsSemiFinished && requirement.ShortfallUnits > 0)
  .map((requirement) => ({
    ...makeEmptyOrder(),
    WarehouseID: warehouseID, SuppliesWarehouseID: warehouseID,
    ...lastSubcontractOf(requirement.SupplyID, previousOrders),
    Lines: [{
      ProductID: requirement.SupplyID, PresentationID: 0, PlannedUnits: requirement.ShortfallUnits,
      ProducedUnits: 0, ScrappedUnits: 0, Value: 0, CostSource: 0,
    }],
  }))
