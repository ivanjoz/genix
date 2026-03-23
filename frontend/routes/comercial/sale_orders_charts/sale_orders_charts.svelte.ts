import { GetHandler } from '$libs/http.svelte';

export interface ISaleSummaryRecord {
	EmpresaID: number;
	Fecha: number;
	ProductIDs: number[];
	Quantity: number[];
	QuantityPendingDelivery: number[];
	TotalAmount: number[];
	TotalDebtAmount: number[];
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
		result = result || []
		
		// Keep charts stable by always rendering records in date order.
		this.records = result
			.filter((summaryRecord) => (summaryRecord?.Fecha || 0) > 0)
			.sort((leftRecord, rightRecord) => leftRecord.Fecha - rightRecord.Fecha);

		console.debug('[SaleOrdersChartsService] records:', this.records.length);
	}

	constructor() {
		super();
		this.fetch();
	}
}
