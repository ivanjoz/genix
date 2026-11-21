import type { FrameFileRows } from '@genix/ui/cache'

/** One day of sales as the charts read it: every array is aligned by index with ProductIDs. */
export interface ISaleSummaryRecord {
	Date: number;
	ProductIDs: number[];
	// In units, sub-units as their fraction (3 units and 2 sixths = 3.333333).
	Quantity: number[];
	QuantityPendingDelivery: number[];
	// In cents of soles.
	TotalAmount: number[];
	TotalDebtAmount: number[];
}

// The "sale-summary" frame (backend sales/types SaleSummaryFrame): Keys [Date, SiteID], Rows
// [ProductID], Sums in declaration order.
const KEY_DATE = 0
const ROW_PRODUCT_ID = 0
const SUM_AMOUNT = 0
const SUM_DEBT_AMOUNT = 1
const SUM_SOLD_QUANTITY = 2
const SUM_PENDING_QUANTITY = 3
// A frame file keeps a float sum as round(v × 1e6) (genix-orm dataframe.FloatScale).
const FRAME_FLOAT_SCALE = 1e6

/**
 * saleSummaryRecordsOfFrameFiles turns the files of the "sale-summary" frame, one per day and site,
 * into one record per day for the whole company: the sites of a day add up per product. The files
 * come sorted by keys, so the days stay ascending.
 */
export const saleSummaryRecordsOfFrameFiles = (files: FrameFileRows[]): ISaleSummaryRecord[] => {
	const recordByDate = new Map<number, ISaleSummaryRecord>()
	// Where each product sits in its day's arrays, so a product sold in several sites adds up in one entry.
	const productIndexByDate = new Map<number, Map<number, number>>()
	for (const file of files) {
		const date = file.keys[KEY_DATE] as number
		let record = recordByDate.get(date)
		let productIndexByID = productIndexByDate.get(date)
		if (!record || !productIndexByID) {
			record = { Date: date, ProductIDs: [], Quantity: [], QuantityPendingDelivery: [], TotalAmount: [], TotalDebtAmount: [] }
			productIndexByID = new Map()
			recordByDate.set(date, record)
			productIndexByDate.set(date, productIndexByID)
		}
		const productIDs = file.rows[ROW_PRODUCT_ID] as number[]
		const sumsOf = (sum: number) => file.sums[sum] as number[]
		for (let rowIndex = 0; rowIndex < productIDs.length; rowIndex++) {
			let productIndex = productIndexByID.get(productIDs[rowIndex])
			if (productIndex === undefined) {
				productIndex = record.ProductIDs.push(productIDs[rowIndex]) - 1
				productIndexByID.set(productIDs[rowIndex], productIndex)
				record.Quantity.push(0); record.QuantityPendingDelivery.push(0)
				record.TotalAmount.push(0); record.TotalDebtAmount.push(0)
			}
			// The scaled floats are exact integers: add them scaled, unscale once below.
			record.Quantity[productIndex] += sumsOf(SUM_SOLD_QUANTITY)[rowIndex]
			record.QuantityPendingDelivery[productIndex] += sumsOf(SUM_PENDING_QUANTITY)[rowIndex]
			record.TotalAmount[productIndex] += sumsOf(SUM_AMOUNT)[rowIndex]
			record.TotalDebtAmount[productIndex] += sumsOf(SUM_DEBT_AMOUNT)[rowIndex]
		}
	}
	const records = [...recordByDate.values()]
	for (const record of records) {
		record.Quantity = record.Quantity.map((scaled) => scaled / FRAME_FLOAT_SCALE)
		record.QuantityPendingDelivery = record.QuantityPendingDelivery.map((scaled) => scaled / FRAME_FLOAT_SCALE)
	}
	return records
}
