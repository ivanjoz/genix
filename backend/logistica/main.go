package logistica

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.productos-stock":            PostAlmacenStock,
	"GET.almacen-movimientos":         GetAlmacenMovimientos,
	"GET.almacen-movimientos-grouped": GetAlmacenMovimientosGrouped,
	"GET.warehouse-product-stock":     GetWarehouseProductStock,
	"GET.products-stock":              GetProductsStock,
	"GET.product-stock-lots-by-ids":   GetProductStockLotsByIDs,
	"GET.product-supply":              GetProductSupply,
	"POST.product-supply":             PostProductSupply,
}
