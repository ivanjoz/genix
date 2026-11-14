import { formatQuantity, quantityDivisorOf } from '#core/quantity.ts'
import type { IProductionOrder } from '#services/production/production-orders.ts'

// Mirrors backend/production/daily_close.go and production/types/daily_close.go: the daily
// production close (docs/PRODUCTION_PLAN.md §5). A run close is a production order of Kind 2
// (docs/MANUFACTURING_PLAN.md §3).

// A day of the warehouse with sales still waiting for the close.
export interface IDailyClosePendingDay {
  Date: number
  SaleCount: number
  Units: number
}

export interface IDailyCloseProduct {
  ProductID: number
  Units: number
  CostSource: number
  // Cents. In a preview it is the estimate; in a saved close, what the entry was booked at.
  Value: number
  // A supply it needed was short, so it consumed less than its recipe.
  ShortSupplies?: boolean
}

// Quantities in sub-units of the supply's divisor (grams of a kg supply).
export interface IDailyCloseSupply {
  SupplyID: number
  SubDivisor: number
  Needed: number
  Available: number
  Consumed: number
}

export interface IDailyClosePreview {
  WarehouseID: number
  SuppliesWarehouseID: number
  Date: number
  MovementDate: number
  IsStrict?: boolean
  Products?: IDailyCloseProduct[]
  Supplies?: IDailyCloseSupply[]
  BlockingReasons?: string[]
  SaleOrderIDs?: number[]
  PreviewHash: string
}

// The units a run close produced of each product, its presentations summed, in the order they entered.
export const closeOrderProductUnits = (closeOrder: IProductionOrder): { ProductID: number, Units: number }[] => {
  const unitsByProductID = new Map<number, number>()
  for (const line of closeOrder.Lines || []) {
    unitsByProductID.set(line.ProductID, (unitsByProductID.get(line.ProductID) || 0) + (line.ProducedUnits || 0))
  }
  return [...unitsByProductID].map(([ProductID, Units]) => ({ ProductID, Units }))
}

export const closeOrderTotalValue = (closeOrder: IProductionOrder): number =>
  (closeOrder.Lines || []).reduce((total, line) => total + (line.Value || 0), 0)

// Only the oldest pending day can be closed: the backend refuses a day while an earlier one waits.
export const oldestPendingDay = (pendingDays: IDailyClosePendingDay[]): IDailyClosePendingDay | undefined =>
  pendingDays.reduce<IDailyClosePendingDay | undefined>(
    (oldest, day) => (!oldest || day.Date < oldest.Date ? day : oldest), undefined)

// Sub-units rendered the way the supply is counted: "2 + 250 g" of a kg supply, "3" of a whole one.
export const formatSupplySubUnits = (subUnits: number, subDivisor: number, subUnitName?: string): string => {
  const divisor = quantityDivisorOf(subDivisor)
  return formatQuantity({ units: Math.floor(subUnits / divisor), sub: subUnits % divisor }, divisor, subUnitName)
}

// What the close lacks of a supply, in its sub-units: 0 when the stock covers the recipes.
export const supplyShortage = (supply: IDailyCloseSupply): number =>
  Math.max(0, (supply.Needed || 0) - (supply.Available || 0))

export const closeTotalValue = (products: IDailyCloseProduct[] = []): number =>
  products.reduce((total, product) => total + (product.Value || 0), 0)

// A day left open past yesterday books its movements on yesterday, never earlier.
export const isMovementDateShifted = (preview: IDailyClosePreview): boolean =>
  preview.MovementDate !== preview.Date

export const canRunClose = (preview: IDailyClosePreview | undefined): boolean =>
  !!preview && !!preview.PreviewHash && !(preview.BlockingReasons || []).length
