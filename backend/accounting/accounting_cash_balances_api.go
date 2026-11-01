// The cash banks' balances at a period's opening and closing, for the direct cash flow of the
// Estados Financieros page.
//
// The entries only carry the period's movements: there are no opening accounting balances yet.
// Every cash movement stores the running balance it left its cash bank at (FinalAmount), so a
// cash bank's balance at a day is the FinalAmount of its last movement up to that day. The
// browser restates a USD cash bank at the day's sell rate.

package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"

	"golang.org/x/sync/errgroup"
)

// CashBankPeriodBalance is a cash bank's balance, in its own currency, before the period's
// first day and at the end of its last day.
type CashBankPeriodBalance struct {
	CashBankID     int32
	OpeningBalance int32
	ClosingBalance int32
}

// GetAccountingCashBalances reads, per cash bank, the FinalAmount of its last movement before
// the period and of its last movement within it. A cash bank with no movement yet is at 0.
func GetAccountingCashBalances(req *core.HandlerArgs) core.HandlerResponse {
	firstDay, lastDay, boundsError := types.PeriodBounds(req.Query["period"])
	if boundsError != nil {
		return req.MakeErr(boundsError.Error())
	}
	companyID := req.User.CompanyID

	cashBanks := []finance.CashBank{}
	cashBanksQuery := db.Query(&cashBanks)
	cashBanksQuery.Select(cashBanksQuery.ID).CompanyID.Equals(companyID)
	if err := cashBanksQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener las cajas:", err)
	}

	// lastFinalAmount is the balance the last movement on or before the day left the cash bank at.
	lastFinalAmount := func(cashBankID int32, day int16) (int32, error) {
		movements := []finance.CashBankMovement{}
		query := db.Query(&movements)
		query.Select(query.FinalAmount).CompanyID.Equals(companyID).CashBankID.Equals(cashBankID).
			Date.LessEqual(day).OrderDesc().Limit(1)
		if err := query.Exec(); err != nil {
			return 0, err
		}
		if len(movements) == 0 {
			return 0, nil
		}
		return movements[0].FinalAmount, nil
	}

	balances := make([]CashBankPeriodBalance, len(cashBanks))
	readGroup := errgroup.Group{}
	for cashBankIndex, cashBank := range cashBanks {
		readGroup.Go(func() error {
			openingBalance, err := lastFinalAmount(cashBank.ID, firstDay-1)
			if err != nil {
				return err
			}
			closingBalance, err := lastFinalAmount(cashBank.ID, lastDay)
			if err != nil {
				return err
			}
			balances[cashBankIndex] = CashBankPeriodBalance{
				CashBankID: cashBank.ID, OpeningBalance: openingBalance, ClosingBalance: closingBalance,
			}
			return nil
		})
	}
	if err := readGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los saldos de las cajas:", err)
	}

	return req.MakeResponse(map[string]any{"CashBalances": balances})
}
