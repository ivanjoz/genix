package logistics

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.productos-stock":            PostAlmacenStock,
	"GET.warehouse-movimientos":         GetAlmacenMovimientos,
	"GET.warehouse-movimientos-grouped": GetAlmacenMovimientosGrouped,
	"GET.warehouse-product-stock":     GetWarehouseProductStock,
	"GET.products-stock":              GetProductsStock,
	"GET.product-stock-lots-by-ids":   GetProductStockLotsByIDs,
	"GET.product-supply":              GetProductSupply,
	"POST.product-supply":             PostProductSupply,
	"GET.supply-material":             GetSupplyMaterials,
	"POST.supply-material":            PostSupplyMaterial,
	"GET.purchase-orders":             GetPurchaseOrders,
	"GET.purchase-orders-query":       GetPurchaseOrdersQuery,
	"POST.purchase-orders":            PostPurchaseOrder,
	"PUT.purchase-orders":             PutPurchaseOrder,
	"POST.purchase-order-entry":       PostPurchaseOrderEntry,
}
