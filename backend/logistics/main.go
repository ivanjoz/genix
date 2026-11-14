package logistics

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.productos-stock":          PostAlmacenStock,
	"GET.warehouse-movements":       GetAlmacenMovimientos,
	"GET.warehouse-product-stock":   GetWarehouseProductStock,
	"GET.products-stock":            GetProductsStock,
	"GET.product-stock-lots-by-ids": GetProductStockLotsByIDs,
	"POST.product-stock-lot-status": PostProductStockLotStatus,
	"GET.product-supply":            GetProductSupply,
	"POST.product-supply":           PostProductSupply,
	"GET.purchase-orders":           GetPurchaseOrders,
	"GET.purchase-orders-query":     GetPurchaseOrdersQuery,
	"GET.purchase-order-ids":        GetPurchaseOrdersByIDs,
	"POST.purchase-orders":          PostPurchaseOrder,
	"PUT.purchase-orders":           PutPurchaseOrder,
	"POST.purchase-order-entry":     PostPurchaseOrderEntry,
}
