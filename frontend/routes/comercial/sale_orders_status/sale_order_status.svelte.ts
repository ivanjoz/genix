import { Notify } from '$libs/helpers';
import { GetHandler } from '$libs/http.svelte';

export interface ISaleOrderTopProduct {
	ProductID: number;
	LineAmount: number;
}

export interface ISaleOrder {
    ID: number;
    EmpresaID: number;
    Fecha: number;
    AlmacenID: number;
    DetailProductsIDs: number[];
    DetailPrices: number[];
    DetailQuantities: number[];
    DetailProductSkus: string[];
    TotalAmount: number;
    TaxAmount: number;
    DebtAmount: number;
    DeliveryStatus: number;
    CajaID_: number;
    ProcessesIncluded_: number[];
    Created: number;
    upd: number;
    UpdatedBy: number;
    ss: number;
    TopPaidProducts?: ISaleOrderTopProduct[];
}

// Derived states for the requested categories
// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
export const SaleOrderGroup = {
	FINALIZADO: 4, // Pagado + Entregado (Status 4)
	PENDIENTE_DE_PAGO: 7, // Pendiente de Pago: Generado (1) or Entregado (3)
	PENDIENTE_DE_ENTREGA: 8, // Pendiente de Entrega: Generado (1) or Pagado (2)
}

export const SaleOrderGroupMap = new Map([[4,[4]],[7,[1,3]],[8,[1,2]]])

export class SaleOrdersService extends GetHandler {
    route = "sale_orders"
    useCache = { min: 0.1, ver: 1 }

    records: ISaleOrder[] = $state([])

    handler(result: ISaleOrder[]): void {
        const data = Array.isArray(result) ? result : [result];
		this.records = data.map((saleOrder) => {
			const topPaidProducts = this.getTopPaidProductsByAmount(saleOrder);
			return {
				...saleOrder,
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
		
		const status = SaleOrderGroupMap.get(group) || []
		if (status.length === 0) {
			Notify.failure("El grupo seleccionado es incorrecto")
			return
		}

		this.route += `?ss=${status.join(",")}`
		console.log("Servicio Instanciado:", this.route)
  }
}
