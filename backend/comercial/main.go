package comercial

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"POST.sale-order":  PostSaleOrder,
	"GET.sale-orders":  GetSaleOrders,
	"GET.sale-summary": GetSaleSummary,
}

func init(){
	core.RegisterActionHandler(2,"Reprocesar Resumen de Ventas", SaleOrderReprocessHandler)
}
