package logistica

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.productos-stock":    PostAlmacenStock,
	"GET.almacen-movimientos": GetAlmacenMovimientos,
	"GET.productos-stock":     GetProductosStock,
	"GET.product-supply":      GetProductSupply,
	"POST.product-supply":     PostProductSupply,
}
