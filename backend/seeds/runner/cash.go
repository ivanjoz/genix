package runner

import (
	"app/core"
	"app/db"
	"app/finance"
	financeTypes "app/finance/types"
	"cmp"
	_ "embed"
	"encoding/json"
	"slices"
	"time"
)

// ensureCashBalance tops a cash register up with a capital contribution when it cannot cover an
// outflow: the payment handlers refuse to leave a register negative.
func (run *Run) EnsureCashBalance(cashBankID int32, amountToPay int32, at time.Time) error {
	currentBalance, err := run.ReadCashBankBalance(cashBankID)
	if err != nil {
		return err
	}
	if currentBalance >= amountToPay {
		return nil
	}
	return run.InjectCash(cashBankID, at, currentBalance, amountToPay-currentBalance+cashTopUpPadding)
}

// readCashBankBalance reads the balance from the base row, the same one POST.cash-banks-movement
// validates FinalAmount - Amount against. The GET.cash-banks delta list can answer a stale balance
// right after a backdated movement, which made the next movement fail its balance check.
func (run *Run) ReadCashBankBalance(cashBankID int32) (int32, error) {
	cashBank, err := financeTypes.GetCaja(run.UserToken.CompanyID, cashBankID)
	return cashBank.CurrentAmount, err
}

// postManualCashMovement sends a movement through POST.cash-banks-movement. The handler answers a
// stale balance with a 200 carrying NeedUpdateSaldo instead of an error, so that body is checked.
func (run *Run) PostManualCashMovement(at time.Time, movement financeTypes.CashBankMovement) error {
	payloadBytes, err := json.Marshal(movement)
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(at.Unix())
	request := run.MakeRequest("POST.cash-banks-movement", nil, string(payloadBytes))
	responseBody := map[string]any{}
	if err := DecodeResponse(run.Send(request, finance.PostCashBankMovement), &responseBody); err != nil {
		return err
	}
	if storedBalance, isStale := responseBody["NeedUpdateSaldo"]; isStale {
		return core.Err("el saldo de la caja", movement.CashBankID, "no coincide; saldo guardado:", storedBalance)
	}
	return nil
}

// injectCash registers a type 7 ("Cobro") movement as a capital contribution. The handler rejects
// the write unless FinalAmount - Amount matches the stored balance, so it is sent pre-computed like
// the frontend does.
func (run *Run) InjectCash(cashBankID int32, at time.Time, currentBalance int32, amountToInject int32) error {
	if err := run.PostManualCashMovement(at, financeTypes.CashBankMovement{
		CashBankID:         cashBankID,
		Type:               financeTypes.CashMovementTypeCollection,
		CounterAccountCode: financeTypes.PCGECapital,
		Amount:             amountToInject,
		FinalAmount:        currentBalance + amountToInject,
	}); err != nil {
		return err
	}

	run.State.Stats.CashInjections++
	run.State.Stats.CashInjected += amountToInject
	core.Log("Seed:: efectivo inyectado", "monto", amountToInject, "saldo previo", currentBalance)
	return nil
}

// rebalanceCashBank recomputes the running balance (FinalAmount) of every movement of a cash
// account in ID order — account, date, then write order within the day — and rewrites the ones
// that changed. The opening balance is the account's current balance less every movement. If the
// trail dips below zero somewhere, a capital contribution dated the day before openingDayStart
// covers the deepest point and the trail is recomputed once more.
func (run *Run) RebalanceCashBank(cashBankID int32, openingDayStart time.Time) (int, error) {
	rewritten := 0
	for range 2 {
		movements := []financeTypes.CashBankMovement{}
		query := db.Query(&movements)
		query.CompanyID.Equals(run.UserToken.CompanyID).CashBankID.Equals(cashBankID)
		if err := query.Exec(); err != nil {
			return rewritten, err
		}
		slices.SortFunc(movements, func(left, right financeTypes.CashBankMovement) int { return cmp.Compare(left.ID, right.ID) })
		currentBalance, err := run.ReadCashBankBalance(cashBankID)
		if err != nil {
			return rewritten, err
		}
		running := currentBalance
		for _, movement := range movements {
			running -= movement.Amount
		}
		lowestBalance := running
		changed := []financeTypes.CashBankMovement{}
		for _, movement := range movements {
			running += movement.Amount
			lowestBalance = min(lowestBalance, running)
			if movement.FinalAmount != running {
				movement.FinalAmount = running
				changed = append(changed, movement)
			}
		}
		if len(changed) > 0 {
			table := db.TableOf[financeTypes.CashBankMovement]()
			if err := db.Update(&changed, table.FinalAmount); err != nil {
				return rewritten, err
			}
			rewritten += len(changed)
		}
		if lowestBalance >= 0 {
			return rewritten, nil
		}
		if err := run.InjectCash(cashBankID, openingDayStart.Add(8*time.Hour), currentBalance, -lowestBalance); err != nil {
			return rewritten, err
		}
		core.SetHistoricalUnix(0)
	}
	return rewritten, core.Err("el saldo de la caja", cashBankID, "sigue quedando negativo")
}
