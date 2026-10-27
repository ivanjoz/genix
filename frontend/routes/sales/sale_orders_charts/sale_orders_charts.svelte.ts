import { GETFrameFiles } from '#libs/ui-runtime.svelte.ts';
import { getFechaUnix } from '@genix/ui/utilities';
import { saleSummaryRecordsOfFrameFiles, type ISaleSummaryRecord } from './sale_orders_charts.ts';

// The days the charts read: the last 8 weeks, today included.
const SALE_SUMMARY_DAYS = 8 * 7;

export class SaleOrdersChartsService {
	records: ISaleSummaryRecord[] = $state([]);

	// The "sale-summary" frame's day files, kept in the browser: each visit only receives the days
	// that changed since the held copies.
	async fetch() {
		const today = getFechaUnix();
		const { files } = await GETFrameFiles('sale-summary', 'sale-summary',
			{ fromKey: today - SALE_SUMMARY_DAYS + 1, toKey: today });
		this.records = saleSummaryRecordsOfFrameFiles(files);
		console.debug('[SaleOrdersChartsService] records:', this.records.length);
	}

	constructor() {
		this.fetch();
	}
}
