package comercial

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"POST.sale_order":  PostSaleOrder,
	"GET.sale_orders":  GetSaleOrders,
	"GET.sale_summary": GetSaleSummary,
}