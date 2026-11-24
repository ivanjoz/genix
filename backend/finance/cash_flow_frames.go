package finance

import (
	"app/core"
	"app/db"
	"app/finance/types"
	"math"
)

// cashFlowFrameMaxDays bounds one read of a cash flow history frame, as the sale-summary read does: a
// file per day and key. The browser splits a longer range into several reads.
const cashFlowFrameMaxDays = 400

// GetCashDelay answers the files of the "cash-delay" frame (what each cash bank collected or paid per
// day, by how many days after its document) over a day range (from/to, UnixDay) the browser does not
// hold current (g/h: its generation and held files). The cash flow forecast builds its delay curves
// from it (plan/CASH_FLOW_FORECAST_PLAN.md §4).
func GetCashDelay(req *core.HandlerArgs) core.HandlerResponse {
	fromDate, toDate, err := cashFlowFrameRange(req)
	if err != nil {
		return req.MakeErr(err)
	}
	movementTable := db.TableOf[types.CashBankMovement]()
	body, err := db.QueryFrame[types.CashBankMovement](req.User.CompanyID, "cash-delay").
		Between(movementTable.Date, int64(fromDate), int64(toDate)).
		Between(movementTable.CashBankID, 0, math.MaxInt32).
		ExecChanged(req.Query["g"], req.Query["h"])
	if err != nil {
		return req.MakeErr("Error al obtener los plazos de cobro y pago.", err)
	}
	return req.MakeBinaryResponse(body)
}

// GetCashDocuments answers the files of the shared "cash-documents" frame (every sale, expense,
// purchase order and asset purchase per day, site, Kind and tax group, open or settled) over a day
// range, as GetCashDelay does (plan/CASH_DOCUMENTS_FRAME_PLAN.md). Any member table reads the whole
// frame; Expense is the one this module owns.
func GetCashDocuments(req *core.HandlerArgs) core.HandlerResponse {
	fromDate, toDate, err := cashFlowFrameRange(req)
	if err != nil {
		return req.MakeErr(err)
	}
	expenseTable := db.TableOf[types.Expense]()
	body, err := db.QueryFrame[types.Expense](req.User.CompanyID, types.CashDocumentsFrame.Name).
		Between(expenseTable.Date, int64(fromDate), int64(toDate)).
		Between(expenseTable.SiteID, 0, math.MaxInt16).
		ExecChanged(req.Query["g"], req.Query["h"])
	if err != nil {
		return req.MakeErr("Error al obtener los documentos de caja.", err)
	}
	return req.MakeBinaryResponse(body)
}

// cashFlowFrameRange reads and bounds the from/to day range of a cash flow frame read.
func cashFlowFrameRange(req *core.HandlerArgs) (int16, int16, error) {
	fromDate, toDate := req.GetQueryInt16("from"), req.GetQueryInt16("to")
	if fromDate <= 0 || toDate < fromDate {
		return 0, 0, core.Err("El rango de fechas no es válido.")
	}
	if toDate-fromDate >= cashFlowFrameMaxDays {
		return 0, 0, core.Err("Se admite como máximo", cashFlowFrameMaxDays, "días por consulta.")
	}
	return fromDate, toDate, nil
}
