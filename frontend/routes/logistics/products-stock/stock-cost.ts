import { StockMovementType } from '#core/stock-movement-type.ts'

// The company's manual-stock policy (backend/company_flags.toml). The backend enforces both on
// every adjustment; the page checks them first so the user sees which product broke the rule.
export const COMPANY_FLAG_BLOCK_MANUAL_STOCK_INBOUND = 6
export const COMPANY_FLAG_REQUIRE_MANUAL_STOCK_COST = 9

export interface IManualStockChange {
  productName: string
  detailName: string // the lot or serial the change targets, '' for the plain stock
  previousQuantity: number
  nextQuantity: number
  unitCost: number
}

export type ManualStockRefusal = { reason: 'inbound-blocked' | 'cost-missing', productName: string }

export const isStockIncrease = (change: IManualStockChange) => change.nextQuantity > change.previousQuantity

// The type a change is saved with: the reason chosen for its direction. An unchanged quantity
// writes no movement, so which of the two it gets does not matter.
export const manualStockChangeType = (change: IManualStockChange, increaseType: number, decreaseType: number) =>
  isStockIncrease(change) ? increaseType : decreaseType

// The first change the policy refuses, or null. Only an increase can break it: a decrease leaves
// at the current average cost, so it needs no purchase cost and is allowed even when manual
// entries are blocked. Opening stock is the exception both ways: it is how a company onboards,
// so the block does not apply, and every valuation starts from it, so it always needs its cost.
export function findManualStockRefusal(
  changes: IManualStockChange[], blockInbound: boolean, requireCost: boolean, increaseType: number,
): ManualStockRefusal | null {
  const isOpeningStock = increaseType === StockMovementType.OPENING_STOCK
  for (const change of changes) {
    if (!isStockIncrease(change)) { continue }
    if (blockInbound && !isOpeningStock) { return { reason: 'inbound-blocked', productName: change.productName } }
    if ((requireCost || isOpeningStock) && !(change.unitCost > 0)) {
      return { reason: 'cost-missing', productName: change.productName }
    }
  }
  return null
}
