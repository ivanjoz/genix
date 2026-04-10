import { GETWithGroupCache } from '$libs/http.svelte';
import { Notify } from '$libs/helpers';

export interface ISaleOrder {
	ID: number;
	ClientID: number;
	Fecha: number;
	Created: number;
	DetailProductsIDs: number[];
	DetailPrices: number[];
	DetailQuantities: number[];
	TotalAmount: number;
	DebtAmount: number;
	ss: number;
}

export interface ISaleOrderGroupRecord {
	id: number;
	igVal: number[];
	records: ISaleOrder[];
	upc: number;
}

export interface ISaleOrderReportForm {
	fechaInicio: number;
	fechaFin: number;
	status: number;
	productID: number;
	clientID: number;
}

export const saleOrderStatusOptions = [
	{ ID: 0, Nombre: 'Todos' },
	{ ID: 1, Nombre: 'Generado' },
	{ ID: 2, Nombre: 'Pagado' },
	{ ID: 3, Nombre: 'Entregado' },
	{ ID: 4, Nombre: 'Finalizado' },
];

export const querySaleOrderReport = async (filters: ISaleOrderReportForm): Promise<ISaleOrder[]> => {
	if (!filters.fechaInicio || !filters.fechaFin) {
		throw new Error('Debe especificar la fecha inicial y final.');
	}

	const queryParams = new URLSearchParams();
	queryParams.set('fecha-start', String(filters.fechaInicio));
	queryParams.set('fecha-end', String(filters.fechaFin));

	if (filters.status > 0) {
		queryParams.set('status', String(filters.status));
	}
	if (filters.productID > 0) {
		queryParams.set('product-id', String(filters.productID));
	}
	if (filters.clientID > 0) {
		queryParams.set('client-id', String(filters.clientID));
	}

	const route = 'sale-order-query';
	const uriParams = Object.fromEntries(queryParams.entries());
	console.debug('[sale_order_report] querying route', { route, uriParams });

	let result: ISaleOrderGroupRecord[];
	try {
		result = await GETWithGroupCache<ISaleOrder>(route, uriParams);
	} catch (error) {
		Notify.failure(String(error || 'No se pudo consultar el reporte de ventas.'));
		throw error;
	}

	const saleOrders = (result || [])
		.flatMap((groupRecord) => groupRecord.records || [])
		.filter((saleOrderRecord) => (saleOrderRecord?.ID || 0) > 0);

	// Keep the report stable across mixed grouped responses.
	saleOrders.sort((leftSaleOrder, rightSaleOrder) => {
		if ((rightSaleOrder.Created || 0) !== (leftSaleOrder.Created || 0)) {
			return (rightSaleOrder.Created || 0) - (leftSaleOrder.Created || 0);
		}
		return (rightSaleOrder.ID || 0) - (leftSaleOrder.ID || 0);
	});

	console.debug('[sale_order_report] fetched rows', {
		groups: (result || []).length,
		saleOrders: saleOrders.length,
	});

	return saleOrders;
};
