package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
)

// saleSummaryMaxDays bounds one read of the "sale-summary" frame: a file per day, one GET each
// (dataframe.MaxFirstKeys).
const saleSummaryMaxDays = 400

// GetSaleSummary answers the "sale-summary" DataFrame files of a day range (from/to, UnixDay) that
// the browser does not hold current (g/h: its generation and held files, genix-ui FRAME_CACHE.md).
func GetSaleSummary(req *core.HandlerArgs) core.HandlerResponse {
	if frameName := req.Query["f"]; frameName != "sale-summary" {
		return req.MakeErr("El reporte solicitado no existe:", frameName)
	}
	fromDate, toDate := req.GetQueryInt16("from"), req.GetQueryInt16("to")
	if fromDate <= 0 || toDate < fromDate {
		return req.MakeErr("El rango de fechas del resumen de ventas no es válido.")
	}
	if toDate-fromDate >= saleSummaryMaxDays {
		return req.MakeErr("El resumen de ventas admite como máximo", saleSummaryMaxDays, "días por consulta.")
	}

	saleOrderTable := db.TableOf[types.SaleOrder]()
	body, err := db.QueryFrame[types.SaleOrder](req.User.CompanyID, "sale-summary").
		Between(saleOrderTable.Date, int64(fromDate), int64(toDate)).
		ExecChanged(req.Query["g"], req.Query["h"])
	if err != nil {
		return req.MakeErr("Error al obtener el resumen de ventas.", err)
	}
	return req.MakeBinaryResponse(body)
}
