package logistics

import (
	"app/core"
	"app/db"
	"app/logistics/types"
	"math"
)

// purchaseOrderSummaryMaxDays bounds one read of the "purchase-order-summary" frame: a file per day and
// site. The browser splits a longer range into several reads.
const purchaseOrderSummaryMaxDays = 400

// GetPurchaseOrderSummary answers the files of the "purchase-order-summary" frame (committed purchase
// orders per day and site, in soles) over a day range (from/to, UnixDay) the browser does not hold
// current (g/h). The cash flow forecast learns the merchandise purchases from it
// (plan/CASH_FLOW_FORECAST_PLAN.md §5).
func GetPurchaseOrderSummary(req *core.HandlerArgs) core.HandlerResponse {
	fromDate, toDate := req.GetQueryInt16("from"), req.GetQueryInt16("to")
	if fromDate <= 0 || toDate < fromDate {
		return req.MakeErr("El rango de fechas no es válido.")
	}
	if toDate-fromDate >= purchaseOrderSummaryMaxDays {
		return req.MakeErr("Se admite como máximo", purchaseOrderSummaryMaxDays, "días por consulta.")
	}
	orderTable := db.TableOf[types.PurchaseOrder]()
	body, err := db.QueryFrame[types.PurchaseOrder](req.User.CompanyID, "purchase-order-summary").
		Between(orderTable.Date, int64(fromDate), int64(toDate)).
		Between(orderTable.SiteID, 0, math.MaxInt16).
		ExecChanged(req.Query["g"], req.Query["h"])
	if err != nil {
		return req.MakeErr("Error al obtener el resumen de órdenes de compra.", err)
	}
	return req.MakeBinaryResponse(body)
}
