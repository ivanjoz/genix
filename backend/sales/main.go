package sales

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"POST.sale-order":       PostSaleOrder,
	"GET.sale-orders":       GetSaleOrders,
	"GET.sale-order-query":  SaleOrderQuery,
	"GET.sale-summary":      GetSaleSummary,
	"GET.sale-order-by-ids": GetSaleOrderByIDs,
	"GET.shipping-costs":    GetShippingCosts,
	"POST.shipping-costs":   PostShippingCosts,
}

func init() {
	core.RegisterActionHandler(2, "Reprocesar Resumen de Ventas", SaleOrderReprocessHandler)
}
