package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
	"math"
)

// GetSaleClientDebt answers the files of the "sale-client-debt" frame (what each client still owes on
// its open sales) that the browser does not hold current (g/h). It takes every client bucket: only
// clients with open sales have rows. The cash flow forecast joins it with the delta-cached clients to
// raise client-credit-days-missing (plan/CASH_DOCUMENTS_FRAME_PLAN.md §12.7.3).
func GetSaleClientDebt(req *core.HandlerArgs) core.HandlerResponse {
	body, err := db.QueryFrame[types.SaleOrder](req.User.CompanyID, "sale-client-debt").
		BetweenField("ClientBucket", 0, math.MaxInt32>>types.SaleClientDebtBucketBits).
		ExecChanged(req.Query["g"], req.Query["h"])
	if err != nil {
		return req.MakeErr("Error al obtener la deuda por cliente.", err)
	}
	return req.MakeBinaryResponse(body)
}
