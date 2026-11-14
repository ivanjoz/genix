package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
)

// clientWeeksShown is how many weeks the customers list draws, the current one last.
const clientWeeksShown = 26

// lastClientWeeks are the ISO weeks (YYWW) of the customers list, the oldest first.
func lastClientWeeks(today int16) []any {
	weeks := make([]any, clientWeeksShown)
	for weekIndex := range clientWeeksShown {
		weeks[clientWeeksShown-1-weekIndex] = core.MakeSemanaFromFechaUnix(today-int16(7*weekIndex), false).Code
	}
	return weeks
}

// GetClientWeekSales syncs the week rows of every client of the last 26 weeks (delta cache).
func GetClientWeekSales(req *core.HandlerArgs) core.HandlerResponse {
	rows := []types.ClientWeekSales{}
	query := db.Query(&rows)
	query.Select().
		CompanyID.Equals(req.User.CompanyID).
		Delta(req.GetDeltaCursor(), lastClientWeeks(core.FechaUnix())...)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las ventas semanales de los clientes:", err)
	}
	return req.MakeResponse(rows)
}
