package dining

import "app/core"

// ModuleHandlers are the routes of the restaurant module (docs/DINING_PLAN.md). Route names carry
// the "dining-" prefix.
var ModuleHandlers = core.AppRouterType{
	"GET.dining-layouts":             GetDiningLayouts,
	"POST.dining-layout":             PostDiningLayout,
	"GET.dining-table-orders":        GetDiningTableOrders,
	"GET.dining-table-order-history": GetDiningTableOrderHistory,
	"POST.dining-table-order-round":  PostDiningTableOrderRound,
	"POST.dining-table-order-lines":  PostDiningTableOrderLines,
	"POST.dining-table-order-charge": PostDiningTableOrderCharge,
	"POST.dining-table-order-annul":  PostDiningTableOrderAnnul,
}
