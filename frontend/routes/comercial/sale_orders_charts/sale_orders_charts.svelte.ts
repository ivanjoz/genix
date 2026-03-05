import { GetHandler } from '$libs/http.svelte';

export interface ISaleSummaryRecord {
	EmpresaID: number;
	Fecha: number;
	ProductIDs_32: number[];
	Quantity_32: number[];
	QuantityPendingDelivery_32: number[];
	TotalAmount_32: number[];
	TotalDebtAmount_32: number[];
	upd: number;
}

export class SaleOrdersChartsService extends GetHandler {
	route = 'sale_summary';
	// Keep cache short for chart pages while preserving delta behavior.
	useCache = { min: 0.2, ver: 1 };
	// Backend key for delta merge when records don't expose `ID`.
	keyID = 'Fecha';

	records: ISaleSummaryRecord[] = $state([]);

	handler(result: ISaleSummaryRecord[]): void {
		const nextRecords = Array.isArray(result)
			? result
			: result
				? [result]
				: [];

		// Keep charts stable by always rendering records in date order.
		this.records = nextRecords
			.filter((summaryRecord) => (summaryRecord?.Fecha || 0) > 0)
			.sort((leftRecord, rightRecord) => leftRecord.Fecha - rightRecord.Fecha);

		console.debug('[SaleOrdersChartsService] records:', this.records.length);
	}

	constructor() {
		super();
		this.fetch();
	}
}
