// Mirror of backend/core/quantity.go. A product quantity is the pair
// `{ units, sub }` read at a divisor, where the real value is `units + sub / divisor`.
//
// Two storage shapes, and which one you get depends on what the row is for:
//  - Aggregates (stock, the movement ledger, sale summaries) keep the halves in separate
//    fields, because they are accumulated and summed.
//  - Documents (sale-order lines) pack them into one number as `units * 1000 + sub`, so
//    1001 at divisor 6 is one box plus one candy.

export const QUANTITY_LINE_SCALE = 1000

// A normalized `sub` has to fit the packed slot's 0..999, which is what caps the divisor.
// A kilogram sold by the gram is exactly 1000 and fills it.
export const MAX_QUANTITY_DIVISOR = QUANTITY_LINE_SCALE

// The divisor of a product with no sub-unit configured.
export const QUANTITY_DIVISOR_NONE = 1

export interface Quantity {
  units: number
  sub: number
}

export const quantityDivisorOf = (subQuantityPerUnit?: number): number =>
  subQuantityPerUnit && subQuantityPerUnit > QUANTITY_DIVISOR_NONE ? subQuantityPerUnit : QUANTITY_DIVISOR_NONE

export const hasSubUnit = (product: { SbuQuantity?: number; SbuUnit?: string }): boolean =>
  (product.SbuQuantity || 0) > QUANTITY_DIVISOR_NONE && !!product.SbuUnit

export const totalSubUnits = (quantity: Quantity, divisor: number): number =>
  quantity.units * divisor + quantity.sub

// Canonical form, with `sub` in [0, divisor). Uses floored division so a negative total
// borrows correctly: -4 at divisor 6 is `{ units: -1, sub: 2 }`, not `{ units: 0, sub: -4 }`.
export const normalizeQuantity = (quantity: Quantity, divisor: number): Quantity => {
  const total = totalSubUnits(quantity, divisor)
  const units = Math.floor(total / divisor)
  return { units, sub: total - units * divisor }
}

export const addQuantity = (a: Quantity, b: Quantity): Quantity =>
  ({ units: a.units + b.units, sub: a.sub + b.sub })

export const isQuantityZero = (quantity: Quantity): boolean =>
  quantity.units === 0 && quantity.sub === 0

// Packs a quantity into the single number a document line carries. Normalizes first, so an
// over-full `sub` carries into `units` instead of corrupting the slot.
export const packQuantityLine = (quantity: Quantity, divisor: number): number => {
  const normalized = normalizeQuantity(quantity, divisor)
  return normalized.units * QUANTITY_LINE_SCALE + normalized.sub
}

export const unpackQuantityLine = (packedQuantity: number): Quantity => ({
  units: Math.floor(packedQuantity / QUANTITY_LINE_SCALE),
  sub: packedQuantity % QUANTITY_LINE_SCALE,
})

// The line total: whole units at their price, sub-units at theirs. Charging sub-units at
// their own price is exact, where prorating the unit price is not — four candies from a
// 5000-cent box of six is 3333.33 prorated.
export const quantityAmount = (quantity: Quantity, unitPrice: number, subUnitPrice: number): number =>
  quantity.units * unitPrice + quantity.sub * subUnitPrice

// Renders a quantity the way the operator entered it: "2" for whole units, "2 + 3 g" when
// there is a sub-unit part. Normalizes first so an accumulated stock row reads canonically.
export const formatQuantity = (
  quantity: Quantity,
  divisor: number,
  subUnitName?: string,
): string => {
  const normalized = normalizeQuantity(quantity, divisor)
  if (!normalized.sub) return String(normalized.units)
  const subLabel = subUnitName ? ` ${subUnitName}` : ''
  if (!normalized.units) return `${normalized.sub}${subLabel}`
  return `${normalized.units} + ${normalized.sub}${subLabel}`
}
