import { GETFrameFiles } from '#libs/ui-runtime.svelte.ts'
import type { FrameFileRows } from '@genix/ui/cache'

// GET.sale-summary reads at most 400 days per request (backend sales/sale_summary_status.go).
const SALE_SUMMARY_MAX_DAYS = 400

/**
 * fetchSaleQuantityFiles reads the "sale-quantity" frame (whole units sold per day, site and product,
 * derived from sale-summary) over [fromDay, toDay], one request per 400 days. The files stay in the
 * browser's frame cache, so a later read only receives the days that changed.
 */
export const fetchSaleQuantityFiles = async (fromDay: number, toDay: number): Promise<FrameFileRows[]> => {
	const files: FrameFileRows[] = []
	for (let chunkFromDay = fromDay; chunkFromDay <= toDay; chunkFromDay += SALE_SUMMARY_MAX_DAYS) {
		const chunkToDay = Math.min(chunkFromDay + SALE_SUMMARY_MAX_DAYS - 1, toDay)
		const chunk = await GETFrameFiles('sale-summary', 'sale-quantity', { fromKey: chunkFromDay, toKey: chunkToDay })
		files.push(...chunk.files)
	}
	return files
}
