import type { FrameFileRows } from '@genix/ui/cache'
import { quantityDivisorOf } from '#core/quantity.ts'

// The end-of-day stock of every site and product, rebuilt backwards from the current stock
// (plan/SALES_FORECAST_PLAN.md §10.3). Nothing stores it: the stock ledger and ProductStock always
// agree, so the stock at the end of a day is today's stock less what moved on the days after it.
//
//   closing(today)  = Σ ProductStock of the site's warehouses
//   closing(d − 1)  = closing(d) − inflow(d) + outflow(d)

/** A ProductStock row as GET.products-stock answers it: the free bucket plus lots/serials, at SubDivisor. */
export interface StockRowOfSite {
	WarehouseID: number
	ProductID: number
	Quantity: number
	SubQuantity: number
	DetailQuantity?: number
	DetailSubQuantity?: number
	SubDivisor?: number
	upd?: number
}

/** One site and product over [fromDay, today]: index i is the day fromDay + i, in whole units. */
export interface SiteProductStockSeries {
	siteID: number
	productID: number
	closing: Float64Array
	inflow: Float64Array
	/** Positive: what left the stock that day. */
	outflow: Float64Array
}

// The "daily-product-stock" frame (backend logistics/types ProductStockDailyFrame): Keys [Date,
// SiteID], Rows [ProductID, SubDivisor], Sums [InUnits, InSub, OutUnits, OutSub].
const KEY_DATE = 0
const KEY_SITE_ID = 1
const ROW_PRODUCT_ID = 0
const ROW_SUB_DIVISOR = 1
const SUM_IN_UNITS = 0
const SUM_IN_SUB = 1
const SUM_OUT_UNITS = 2
const SUM_OUT_SUB = 3

/**
 * closingStockSeries rebuilds the end-of-day stock, inflow and outflow of every site and product over
 * [fromDay, today]. dailyStockFiles must cover every day from fromDay to today: a day missing from them
 * would read as a day nothing moved, and every day before it would be off. A product with stock and
 * no movement keeps its current stock on every day.
 */
export const closingStockSeries = (
	stockRows: StockRowOfSite[], siteIDByWarehouseID: Map<number, number>,
	dailyStockFiles: FrameFileRows[], fromDay: number, today: number,
): SiteProductStockSeries[] => {
	const dayCount = today - fromDay + 1
	const seriesBySiteProduct = new Map<string, SiteProductStockSeries>()
	const seriesOf = (siteID: number, productID: number): SiteProductStockSeries => {
		const key = `${siteID}_${productID}`
		let series = seriesBySiteProduct.get(key)
		if (!series) {
			series = {
				siteID, productID,
				closing: new Float64Array(dayCount), inflow: new Float64Array(dayCount), outflow: new Float64Array(dayCount),
			}
			seriesBySiteProduct.set(key, series)
		}
		return series
	}

	// Today's closing: every warehouse row of the site, sub-units at the row's current divisor.
	for (const stockRow of stockRows) {
		const siteID = siteIDByWarehouseID.get(stockRow.WarehouseID)
		if (siteID === undefined) {
			throw new Error(`closingStockSeries: the warehouse ${stockRow.WarehouseID} has no site`)
		}
		const units = stockRow.Quantity + (stockRow.DetailQuantity || 0)
		const subUnits = stockRow.SubQuantity + (stockRow.DetailSubQuantity || 0)
		seriesOf(siteID, stockRow.ProductID).closing[dayCount - 1] += units + subUnits / quantityDivisorOf(stockRow.SubDivisor)
	}

	// Each day's inflow and outflow; a row of an older divisor converts with its own.
	for (const file of dailyStockFiles) {
		const day = file.keys[KEY_DATE] as number
		if (day < fromDay || day > today) continue
		const dayIndex = day - fromDay
		const siteID = file.keys[KEY_SITE_ID] as number
		const productIDs = file.rows[ROW_PRODUCT_ID] as number[]
		const subDivisors = file.rows[ROW_SUB_DIVISOR] as number[]
		for (let rowIndex = 0; rowIndex < productIDs.length; rowIndex++) {
			const divisor = quantityDivisorOf(subDivisors[rowIndex])
			const series = seriesOf(siteID, productIDs[rowIndex])
			series.inflow[dayIndex] += (file.sums[SUM_IN_UNITS][rowIndex] as number) + (file.sums[SUM_IN_SUB][rowIndex] as number) / divisor
			series.outflow[dayIndex] -= (file.sums[SUM_OUT_UNITS][rowIndex] as number) + (file.sums[SUM_OUT_SUB][rowIndex] as number) / divisor
		}
	}

	// Backwards from today: the day before closed at this day's closing less what this day moved.
	for (const series of seriesBySiteProduct.values()) {
		for (let dayIndex = dayCount - 1; dayIndex > 0; dayIndex--) {
			series.closing[dayIndex - 1] = series.closing[dayIndex] - series.inflow[dayIndex] + series.outflow[dayIndex]
		}
	}
	return [...seriesBySiteProduct.values()]
}

/**
 * isStockNewerThanFrame tells whether a stock row was written after the frame read: the frame must be
 * read again, or that movement would be in today's stock and not in the days, offsetting every day. A
 * movement before the read moved its day file to the read's moment, the latest snapshot of the files.
 */
export const isStockNewerThanFrame = (stockRows: StockRowOfSite[], frameSnapshot: number, dailyStockFiles: FrameFileRows[]): boolean => {
	const frameReadMoment = dailyStockFiles.reduce((latest, file) => Math.max(latest, file.snapshot), frameSnapshot)
	return stockRows.some((stockRow) => (stockRow.upd || 0) > frameReadMoment)
}
