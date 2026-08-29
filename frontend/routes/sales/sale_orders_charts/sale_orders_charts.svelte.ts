import { GetHandler } from '$libs/ui-runtime.svelte';

export interface ISaleSummaryRecord {
	CompanyID: number;
	Date: number;
	ProductIDs: number[];
	// Split pair per product: a summary is accumulated across sales, so it never uses the
	// packed document form. SubDivisor says what the Sub* halves are counted in.
	Quantity: number[];
	SubQuantity: number[];
	QuantityPendingDelivery: number[];
	SubQuantityPendingDelivery: number[];
	SubDivisor: number[];
	TotalAmount: number[];
	TotalDebtAmount: number[];
	upd: number;
}

export class SaleOrdersChartsService extends GetHandler {
	route = 'sale-summary';
	// Keep cache short for chart pages while preserving delta behavior.
	useCache = { min: 0.2, ver: 4 };
	// Backend key for delta merge when records don't expose `ID`.
	keyID = 'Date';
	columnarIDField = "ProductIDs";
	combineColumnarValuesOnFields = [
		"Quantity", "SubQuantity", "QuantityPendingDelivery", "SubQuantityPendingDelivery",
		"SubDivisor", "TotalAmount", "TotalDebtAmount"
	];

	records: ISaleSummaryRecord[] = $state([]);

	handler(result: ISaleSummaryRecord[]): void {
		result = result || []
		
		// Keep charts stable by always rendering records in date order.
		this.records = result
			.filter((summaryRecord) => (summaryRecord?.Date || 0) > 0)
			.sort((leftRecord, rightRecord) => leftRecord.Date - rightRecord.Date);

		console.debug('[SaleOrdersChartsService] records:', this.records.length);
	}

	constructor() {
		super();
		this.fetch();
	}
}
