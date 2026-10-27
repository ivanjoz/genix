import type { FrameFileRows } from '@genix/ui/cache'

/** One day of sales as the charts read it: every array is aligned by index with ProductIDs. */
export interface ISaleSummaryRecord {
	Date: number;
	ProductIDs: number[];
	// In units, sub-units as their fraction (3 units and 2 sixths = 3.333…).
	Quantity: number[];
	QuantityPendingDelivery: number[];
	// In cents of soles.
	TotalAmount: number[];
	TotalDebtAmount: number[];
}

// The "sale-summary" frame (backend sales/types SaleSummaryFrame): Keys [Date], Rows [ProductID,
// SubDivisor], Sums in declaration order.
const ROW_PRODUCT_ID = 0
const ROW_SUB_DIVISOR = 1
const SUM_AMOUNT = 0
const SUM_DEBT_AMOUNT = 1
const SUM_SOLD_UNITS = 2
const SUM_SOLD_SUB = 3
const SUM_PENDING_UNITS = 4
const SUM_PENDING_SUB = 5

/**
 * saleSummaryRecordsOfFrameFiles turns the day files of the "sale-summary" frame into one record per
 * day. A product refined mid-day has a row per sub-unit divisor: its rows join into one product
 * entry, each divisor's sub-units added as a fraction of a unit.
 */
export const saleSummaryRecordsOfFrameFiles = (files: FrameFileRows[]): ISaleSummaryRecord[] => {
	const records: ISaleSummaryRecord[] = []
	for (const file of files) {
		const record: ISaleSummaryRecord = {
			Date: file.keys[0] as number,
			ProductIDs: [], Quantity: [], QuantityPendingDelivery: [], TotalAmount: [], TotalDebtAmount: [],
		}
		const productIndexByID = new Map<number, number>()
		const productIDs = file.rows[ROW_PRODUCT_ID] as number[]
		for (let row = 0; row < productIDs.length; row++) {
			const productID = productIDs[row] as number
			const subDivisor = Math.max(1, file.rows[ROW_SUB_DIVISOR]![row] as number)
			const sumOf = (sum: number) => file.sums[sum]![row] as number

			let productIndex = productIndexByID.get(productID)
			if (productIndex === undefined) {
				productIndex = record.ProductIDs.length
				productIndexByID.set(productID, productIndex)
				record.ProductIDs.push(productID)
				record.Quantity.push(0); record.QuantityPendingDelivery.push(0)
				record.TotalAmount.push(0); record.TotalDebtAmount.push(0)
			}
			record.Quantity[productIndex]! += sumOf(SUM_SOLD_UNITS) + sumOf(SUM_SOLD_SUB) / subDivisor
			record.QuantityPendingDelivery[productIndex]! += sumOf(SUM_PENDING_UNITS) + sumOf(SUM_PENDING_SUB) / subDivisor
			record.TotalAmount[productIndex]! += sumOf(SUM_AMOUNT)
			record.TotalDebtAmount[productIndex]! += sumOf(SUM_DEBT_AMOUNT)
		}
		records.push(record)
	}
	return records
}
