import { Notify } from '$libs/helpers';
import { GetHandler, POST } from '$libs/ui-runtime.svelte';

export interface ISaleOrderTopProduct {
	ProductID: number;
	LineAmount: number;
}

export interface ISaleOrder {
    ID: number;
    CompanyID: number;
    Date: number;
    WarehouseID: number;
    DetailProductsIDs: number[];
    DetailPrices: number[];
    // Packed: units * 1000 + sub. Unpack with $core/quantity before displaying.
    DetailQuantities: number[];
    DetailSubDivisor?: number[];
    DetailSubPrices?: number[];
    DetailProductSkus: string[];
    DetailProductPresentations: number[];
    TotalAmount: number;
    TaxAmount: number;
    DebtAmount: number;
    DeliveryStatus: number;
    LastPaymentCajaID: number;
    ActionsIncluded: number[];
    Created: number;
    LastPaymentTime: number;
    LastPaymentUser: number;
    DeliveryTime: number;
    DeliveryUser: number;
		ClientID: number;
    upd: number;
    UpdatedBy: number;
    ss: number;
    // Only set on annulled orders. Updated/UpdatedBy are the "annulled at / by": annulment is
    // terminal, so nothing writes to the record afterwards.
    AnnulReason?: string;
    TopPaidProducts?: ISaleOrderTopProduct[];
}

// API payload for sale-order transitions (payment and delivery updates).
export interface ISaleOrderUpdatePayload {
	ID: number;
	ActionsIncluded: number[];
	LastPaymentCajaID?: number;
	WarehouseID?: number;
	DebtAmount?: number;
}

// The operator chooses where the refund is withdrawn from — it need not be the cash bank that
// collected. The amount is never sent: the backend reads it from the cash ledger.
export interface ISaleOrderAnnulPayload {
	ID: number;
	RefundCashBankID: number;
	Reason: string;
}

// "Gestión Ventas" in backend/access.toml, and its sub-access. Annulling is gated on the
// sub-access on both sides; this check only decides whether the button is offered.
export const SALES_MANAGEMENT_ACCESS_ID = 11
export const ANNUL_SALE_SUB_ACCESS_ID = 2

// UI groups map 1:1 to backend query params:
// - `pending-status=2` => pending payment
// - `pending-status=3` => pending delivery
// - `order-status=4`   => completed
// - `order-status=0`   => annulled
export const SaleOrderGroup = {
	PENDIENTE_DE_PAGO: 2,
	PENDIENTE_DE_ENTREGA: 3,
	FINALIZADO: 4,
	ANULADO: 0,
}

export interface ISaleOrdersResult {
	records: ISaleOrder[]
}

export class SaleOrdersService extends GetHandler {
  route = "sale-orders"
  // Route now depends on group; bump version to avoid mixing old cached queries.
  // v11 stopped filtering out ss=0, so a v10 cache would hide the Anuladas tab's own records.
  useCache = { min: 0.1, ver: 11 }

	records: ISaleOrder[] = $state([])

	handler(result: ISaleOrdersResult): void {
		console.log("ISaleOrdersResult", result)
		// No status filter here: the backend pins the status this group asked for and sends
		// records_IDsToRemove for everything that left it. Dropping ss=0 would empty the
		// Anuladas tab, which is the one group whose records are all ss=0.
		const records = [...(result?.records || [])]
		records.sort((a,b) => a.Created > b.Created ? -1 : 1)

		// Normalize once here so page consumers can reuse the service data without extra passes.
		this.records = records
		console.debug('[SaleOrdersService] fetched rows:', result?.records?.length,"|",this.records.length);
  }

	constructor(group: number) {
		super()

		// Keep cache keys separated by group by embedding query params in `route`.
		if (group === SaleOrderGroup.FINALIZADO || group === SaleOrderGroup.ANULADO) {
			this.route += `?order-status=${group}`
		} else if (group === SaleOrderGroup.PENDIENTE_DE_PAGO || group === SaleOrderGroup.PENDIENTE_DE_ENTREGA) {
			this.route += `?pending-status=${group}`
		} else {
			Notify.failure("El grupo seleccionado es incorrecto")
			return
		}

		console.debug("[SaleOrdersService] route:", this.route)
  }
}

export const postSaleOrderUpdate = (payload: ISaleOrderUpdatePayload) => {
	// Keep route invalidation explicit so all sale-orders views can re-sync after updates.
	return POST({
		route: 'sale-order',
		data: payload,
		refreshRoutes: ['sale-orders'],
	});
};

export const postSaleOrderAnnul = (payload: ISaleOrderAnnulPayload) => {
	// An annulment moves the order out of whichever tab it was in and into Anuladas, so every
	// sale-orders view has to re-sync, not just the current one.
	return POST({
		route: 'sale-order-annul',
		data: payload,
		refreshRoutes: ['sale-orders'],
	});
};
