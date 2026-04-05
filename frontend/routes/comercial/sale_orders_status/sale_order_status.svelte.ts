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

export class SaleOrdersService extends GetHandler {
    route = "sale-orders"
    // Route now depends on group; bump version to avoid mixing old cached queries.
    useCache = { min: 0.1, ver: 2 }

	records: ISaleOrder[] = $state([])

	handler(result: ISaleOrder[]): void {
		// Keep diagnostics lightweight to avoid blocking the main thread with large object logs.
		const data = Array.isArray(result) ? result : [result];
		console.debug('[SaleOrdersService] fetched rows:', data.length);
		// Never show deleted/canceled records in the UI; cache merge still works via IDs.
		const activeOrders = data.filter((saleOrder) => (saleOrder?.ss || 0) > 0);
		this.records = activeOrders.map((saleOrder) => {
			const topPaidProducts = this.getTopPaidProductsByAmount(saleOrder);
			const normalizedLastPaymentCajaID = saleOrder.LastPaymentCajaID || (saleOrder as any).CajaID_ || 0;
			return {
				...saleOrder,
				LastPaymentCajaID: normalizedLastPaymentCajaID,
				TopPaidProducts: topPaidProducts,
			};
		});
    }

	private getTopPaidProductsByAmount(saleOrder: ISaleOrder): ISaleOrderTopProduct[] {
		const detailCount = Math.min(
			saleOrder.DetailProductsIDs?.length || 0,
			saleOrder.DetailPrices?.length || 0,
			saleOrder.DetailQuantities?.length || 0
		);
		const productAmountMap = new Map<number, ISaleOrderTopProduct>();

		// Aggregate line subtotal by product, then keep top records with tie-aware cutoff.
		for (let detailIndex = 0; detailIndex < detailCount; detailIndex += 1) {
			const productID = saleOrder.DetailProductsIDs[detailIndex];
			const linePrice = saleOrder.DetailPrices[detailIndex] || 0;
			const lineQuantity = saleOrder.DetailQuantities[detailIndex] || 0;
			const lineAmount = linePrice * lineQuantity;
			if (!productID || lineAmount <= 0) { continue; }

			const previousProductData = productAmountMap.get(productID);

			if (previousProductData) {
				previousProductData.LineAmount += lineAmount;
				continue;
			}

			productAmountMap.set(productID, {
				ProductID: productID,
				LineAmount: lineAmount,
			});
		}

		const sortedProducts = Array.from(productAmountMap.values()).sort((leftProduct, rightProduct) => {
			if (rightProduct.LineAmount !== leftProduct.LineAmount) {
				return rightProduct.LineAmount - leftProduct.LineAmount;
			}
			return leftProduct.ProductID - rightProduct.ProductID;
		});
		if (sortedProducts.length <= 3) { return sortedProducts; }

		const topThreeIndex = 2;
		const tieAwareCutoffAmount = sortedProducts[topThreeIndex].LineAmount;
		return sortedProducts.filter(product => product.LineAmount >= tieAwareCutoffAmount);
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
