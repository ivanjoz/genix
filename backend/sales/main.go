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
	// Pedidos Web: the web store's orders, before and after they become sales.
	"GET.online-sale-orders":        GetOnlineSaleOrders,
	"POST.online-sale-order-settle": PostOnlineSaleOrderSettle,
	"POST.online-sale-order-annul":  PostOnlineSaleOrderAnnul,
	// The web store. Public ("p-"): each route checks the web customer token itself.
	"POST.p-web-customer-code":     PostWebCustomerCode,
	"POST.p-web-customer-verify":   PostWebCustomerVerify,
	"POST.p-web-customer-login":    PostWebCustomerLogin,
	"POST.p-web-checkout-identify": PostWebCheckoutIdentify,
	"GET.p-web-customer":           GetWebCustomer,
	"POST.p-web-customer":          PostWebCustomer,
	"POST.p-web-customer-password": PostWebCustomerPassword,
	"POST.p-web-customer-address":  PostWebCustomerAddress,
	"GET.p-web-customer-orders":    GetWebCustomerOrders,
	"GET.p-web-order-pdf":          GetWebOrderPurchasePDF,
	"POST.p-web-checkout":          PostWebCheckout,
	"GET.p-web-shipping-amount":    GetWebShippingAmount,
}
