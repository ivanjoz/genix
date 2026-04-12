import { Notify } from '$libs/helpers';
import { GetHandler } from '$libs/http.svelte';
import { POST } from '$libs/http.svelte';

export interface ISaleOrderTopProduct {
	ProductID: number;
	LineAmount: number;
}

export interface ISaleOrder {
    ID: number;
    EmpresaID: number;
    Fecha: number;
    WarehouseID: number;
    DetailProductsIDs: number[];
    DetailPrices: number[];
    DetailQuantities: number[];
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

// UI groups map 1:1 to backend query params:
// - `pending-status=2` => pending payment
// - `pending-status=3` => pending delivery
// - `order-status=4`   => completed
export const SaleOrderGroup = {
	PENDIENTE_DE_PAGO: 2,
	PENDIENTE_DE_ENTREGA: 3,
	FINALIZADO: 4,
}

export interface ISaleOrdersResult {
	records: ISaleOrder[]
}

export class SaleOrdersService extends GetHandler {
  route = "sale-orders"
  // Route now depends on group; bump version to avoid mixing old cached queries.
  useCache = { min: 0.1, ver: 8 }

	records: ISaleOrder[] = $state([])

	handler(result: ISaleOrdersResult): void {
		console.log("ISaleOrdersResult", result)
		const records = (result?.records || []).filter((saleOrder) => (saleOrder?.ss || 0) > 0)
		records.sort((a,b) => a.Created > b.Created ? -1 : 1)
		
		// Normalize once here so page consumers can reuse the service data without extra passes.
		this.records = records
		console.debug('[SaleOrdersService] fetched rows:', result?.records?.length,"|",this.records.length);
  }

	constructor(group: number) {
		super()

		// Keep cache keys separated by group by embedding query params in `route`.
		if (group === SaleOrderGroup.FINALIZADO) {
			this.route += `?order-status=${SaleOrderGroup.FINALIZADO}`
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
