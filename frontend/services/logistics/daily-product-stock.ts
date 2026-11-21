import { GETFrameFiles } from '#libs/ui-runtime.svelte.ts'
import type { FrameFilesResult } from '@genix/ui/cache'

// GET.daily-product-stock reads at most 400 days per request (backend logistics/daily_product_stock.go).
const DAILY_PRODUCT_STOCK_MAX_DAYS = 400

/**
 * fetchDailyProductStockFiles reads the "daily-product-stock" frame (each day's stock inflow and
 * outflow per site and product) over [fromDay, toDay], one request per 400 days. The files stay in the
 * browser's frame cache, so a later read only receives the days that changed. snapshot is the oldest
 * of the requests'. With GET.products-stock it feeds closingStockSeries (#core/stock-closing.ts).
 */
export const fetchDailyProductStockFiles = async (fromDay: number, toDay: number): Promise<FrameFilesResult> => {
	const result: FrameFilesResult = { snapshot: 0, files: [] }
	for (let chunkFromDay = fromDay; chunkFromDay <= toDay; chunkFromDay += DAILY_PRODUCT_STOCK_MAX_DAYS) {
		const chunkToDay = Math.min(chunkFromDay + DAILY_PRODUCT_STOCK_MAX_DAYS - 1, toDay)
		const chunk = await GETFrameFiles('daily-product-stock', 'daily-product-stock', { fromKey: chunkFromDay, toKey: chunkToDay })
		result.snapshot = result.snapshot === 0 ? chunk.snapshot : Math.min(result.snapshot, chunk.snapshot)
		result.files.push(...chunk.files)
	}
	return result
}
