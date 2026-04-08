package comercial

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"POST.sale-order":       PostSaleOrder,
	"GET.sale-orders":       GetSaleOrders,
	"GET.sale-order-query":  SaleOrderQuery,
	"GET.sale-summary":      GetSaleSummary,
	"GET.sale-order-by-ids": GetSaleOrderByIDs,
}

func init() {
	core.RegisterActionHandler(2, "Reprocesar Resumen de Ventas", SaleOrderReprocessHandler)
}
