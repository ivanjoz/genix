import { formatQuantity, quantityDivisorOf } from '#core/quantity.ts'

// Mirrors backend/production/daily_close.go and production/types/daily_close.go: the daily
// production close (docs/PRODUCTION_PLAN.md §5).

// Where a produced product's value came from.
export const CLOSE_COST_SOURCE = { STANDARD: 1, SUPPLIES: 2 } as const

// A day of the warehouse with sales still waiting for the close.
export interface IDailyClosePendingDay {
  Date: number
  SaleCount: number
  Units: number
}

export interface IProductionCloseProduct {
  ProductID: number
  Units: number
  CostSource: number
  // Cents. In a preview it is the estimate; in a saved close, what the entry was booked at.
  Value: number
  // A supply it needed was short, so it consumed less than its recipe.
  ShortSupplies?: boolean
}

// Quantities in sub-units of the supply's divisor (grams of a kg supply).
export interface IProductionCloseSupply {
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
  Products?: IProductionCloseProduct[]
  Supplies?: IProductionCloseSupply[]
  BlockingReasons?: string[]
  SaleOrderIDs?: number[]
  PreviewHash: string
}

export interface IProductionClose {
  ID: number
  WarehouseID: number
  SuppliesWarehouseID: number
  Date: number
  IsStrict?: boolean
  Products?: IProductionCloseProduct[]
  Supplies?: IProductionCloseSupply[]
  SaleOrderIDs?: number[]
  Created: number
  CreatedBy: number
  ss: number
  upd: number
}

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
export const supplyShortage = (supply: IProductionCloseSupply): number =>
  Math.max(0, (supply.Needed || 0) - (supply.Available || 0))

export const closeTotalValue = (products: IProductionCloseProduct[] = []): number =>
  products.reduce((total, product) => total + (product.Value || 0), 0)

// A day left open past yesterday books its movements on yesterday, never earlier.
export const isMovementDateShifted = (preview: IDailyClosePreview): boolean =>
  preview.MovementDate !== preview.Date

export const canRunClose = (preview: IDailyClosePreview | undefined): boolean =>
  !!preview && !!preview.PreviewHash && !(preview.BlockingReasons || []).length
