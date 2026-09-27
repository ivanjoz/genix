// The company's manual-stock policy (backend/company_flags.toml). The backend enforces both on
// every adjustment; the page checks them first so the user sees which product broke the rule.
export const COMPANY_FLAG_BLOCK_MANUAL_STOCK_INBOUND = 6
export const COMPANY_FLAG_REQUIRE_MANUAL_STOCK_COST = 9

export interface IManualStockChange {
  productName: string
  previousQuantity: number
  nextQuantity: number
  unitCost: number
}

export type ManualStockRefusal = { reason: 'inbound-blocked' | 'cost-missing', productName: string }

// The first change the policy refuses, or null. Only an increase can break it: a decrease leaves
// at the current average cost, so it needs no purchase cost and is allowed even when manual
// entries are blocked — that is how shrinkage and physical counts are recorded.
export function findManualStockRefusal(
  changes: IManualStockChange[], blockInbound: boolean, requireCost: boolean,
): ManualStockRefusal | null {
  for (const change of changes) {
    if (change.nextQuantity <= change.previousQuantity) { continue }
    if (blockInbound) { return { reason: 'inbound-blocked', productName: change.productName } }
    if (requireCost && !(change.unitCost > 0)) { return { reason: 'cost-missing', productName: change.productName } }
  }
  return null
}
