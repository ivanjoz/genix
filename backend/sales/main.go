package sales

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"POST.sale-order":        PostSaleOrder,
	"POST.sale-order-annul":  PostSaleOrderAnnul,
	"GET.sale-orders":        GetSaleOrders,
	"GET.sale-order-query":   SaleOrderQuery,
	"GET.sale-summary":       GetSaleSummary,
	"GET.sale-order-by-ids":  GetSaleOrderByIDs,
	"GET.client-week-sales":  GetClientWeekSales,
	"GET.shipping-costs":     GetShippingCosts,
	"POST.shipping-costs":    PostShippingCosts,
	"GET.sales-planning":     GetSalesPlanning,
	"POST.sales-planning":    PostSalesPlanning,
	"GET.seasonality-curve":  GetSeasonalityCurve,
	"POST.seasonality-curve": PostSeasonalityCurve,
}
