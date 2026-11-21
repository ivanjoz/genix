package logistics

import (
	"app/core"
	"app/db"
	"app/logistics/types"
	"math"
)

// dailyProductStockMaxDays bounds one read of the "daily-product-stock" frame, as the sale-summary
// read does: a file per day and site.
const dailyProductStockMaxDays = 400

// GetDailyProductStock answers the files of the "daily-product-stock" frame (each day's stock inflow
// and outflow per site and product) over a day range (from/to, UnixDay) that the browser does not hold
// current (g/h: its generation and held files, genix-ui FRAME_CACHE.md). With GET.products-stock (the
// current stock) it is what the browser rebuilds every day's closing stock from, backwards from today
// (plan/SALES_FORECAST_PLAN.md §10.3).
func GetDailyProductStock(req *core.HandlerArgs) core.HandlerResponse {
	fromDate, toDate := req.GetQueryInt16("from"), req.GetQueryInt16("to")
	if fromDate <= 0 || toDate < fromDate {
		return req.MakeErr("El rango de fechas del stock diario no es válido.")
	}
	if toDate-fromDate >= dailyProductStockMaxDays {
		return req.MakeErr("El stock diario admite como máximo", dailyProductStockMaxDays, "días por consulta.")
	}

	movementTable := db.TableOf[types.WarehouseProductMovement]()
	// Every site: the SQL driver selects a Keys range inside the Part's rows, so the whole int16 range
	// costs nothing.
	body, err := db.QueryFrame[types.WarehouseProductMovement](req.User.CompanyID, "daily-product-stock").
		Between(movementTable.Date, int64(fromDate), int64(toDate)).
		Between(movementTable.SiteID, 0, math.MaxInt16).
		ExecChanged(req.Query["g"], req.Query["h"])
	if err != nil {
		return req.MakeErr("Error al obtener el stock diario.", err)
	}
	return req.MakeBinaryResponse(body)
}
