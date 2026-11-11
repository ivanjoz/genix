package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
	"errors"
)

// clientWeeksShown is how many weeks the customers list draws, the current one last.
const clientWeeksShown = 26

// applyClientWeekSaleChangesIn adds what a sale write changed to its client's week rows, in the
// sale's transaction, so the row can never disagree with the sale. The week's first sale of a client
// creates the row; when two sales race to create it, the loser's duplicate key reruns its whole
// transaction, which then increments the winner's row.
func applyClientWeekSaleChangesIn(transaction *db.Transaction, companyID int32, changes []types.ClientWeekSaleShare) error {
	table := db.TableOf[types.ClientWeekSales]()
	for _, change := range changes {
		week, weekday := types.ClientWeekOfDay(change.Date)
		key := types.ClientWeekSales{CompanyID: companyID, Week: week, ClientID: change.ClientID}
		err := db.IncrementIn(transaction, key, table.Purchases.At(weekday).Add(change.Purchase), table.Payments.At(weekday).Add(change.Payment))
		if errors.Is(err, db.ErrIncrementNotStored) {
			err = db.InsertIn(transaction, &[]types.ClientWeekSales{types.NewClientWeekSales(companyID, change)})
		}
		if err != nil {
			return core.Err("Error al acumular las ventas semanales del cliente:", err)
		}
	}
	return nil
}

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
