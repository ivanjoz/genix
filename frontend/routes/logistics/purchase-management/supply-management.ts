import type { IWarehouseProductMovement } from '../warehouse-movements/warehouse-movements.svelte'

export interface IDateProductMovements {
  Date: number
  DetailProductsIDs: number[]
  DetailInflows: number[]
  DetailOutflows: number[]
}

/**
 * Folds raw ledger movements into one row per day with the whole-unit inflow and outflow of each
 * product. The sums run per (Date, ProductID, Type, SubDivisor) first, so sub-units are only
 * added within one divisor; each sum is then truncated to whole units (supply thresholds are
 * whole-unit) and split by sign. A product moved only in sub-units still shows its whole units
 * instead of reading as dead stock.
 */
export const aggregateDailyProductMovements = (movements: IWarehouseProductMovement[]): IDateProductMovements[] => {
  const sumsByGroupKey = new Map<string, { date: number, productID: number, subDivisor: number, units: number, sub: number }>()
  for (const movement of movements) {
    const date = movement.Date || 0
    const productID = movement.ProductID || 0
    const subDivisor = movement.SubDivisor || 0
    const groupKey = `${date}_${productID}_${movement.Type || 0}_${subDivisor}`
    const groupSum = sumsByGroupKey.get(groupKey) || { date, productID, subDivisor, units: 0, sub: 0 }
    groupSum.units += movement.Quantity || 0
    groupSum.sub += movement.SubQuantity || 0
    sumsByGroupKey.set(groupKey, groupSum)
  }

  const dayRowsByDate = new Map<number, IDateProductMovements>()
  const productIndexByDate = new Map<number, Map<number, number>>()
  for (const groupSum of sumsByGroupKey.values()) {
    let dayRow = dayRowsByDate.get(groupSum.date)
    if (!dayRow) {
      dayRow = { Date: groupSum.date, DetailProductsIDs: [], DetailInflows: [], DetailOutflows: [] }
      dayRowsByDate.set(groupSum.date, dayRow)
      productIndexByDate.set(groupSum.date, new Map())
    }
    const productIndexes = productIndexByDate.get(groupSum.date)!
    let productIndex = productIndexes.get(groupSum.productID)
    if (productIndex === undefined) {
      productIndex = dayRow.DetailProductsIDs.length
      productIndexes.set(groupSum.productID, productIndex)
      dayRow.DetailProductsIDs.push(groupSum.productID)
      dayRow.DetailInflows.push(0)
      dayRow.DetailOutflows.push(0)
    }

    const wholeUnits = groupSum.sub !== 0 && groupSum.subDivisor > 0
      ? Math.trunc((groupSum.units * groupSum.subDivisor + groupSum.sub) / groupSum.subDivisor)
      : groupSum.units
    if (wholeUnits > 0) {
      dayRow.DetailInflows[productIndex] += wholeUnits
    } else if (wholeUnits < 0) {
      dayRow.DetailOutflows[productIndex] += -wholeUnits
    }
  }

  return [...dayRowsByDate.values()].sort((a, b) => a.Date - b.Date)
}
