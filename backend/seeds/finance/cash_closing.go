package finance

import (
	"app/core"
	financeModule "app/finance"
	financeTypes "app/finance/types"
	"app/seeds/runner"
	"encoding/json"
	"time"
)

// tillFloat is what each till keeps overnight as change; the rest of the day's takings goes to the bank.
const tillFloat int32 = 50000

// runTillTransfersStep deposits each till's takings in the bank, keeping the change float. It
// re-reads the balance before each transfer, so a resumed step only moves what is still there.
func runTillTransfersStep(run *runner.Run, dayStart time.Time) error {
	transferTime := dayStart.Add(21 * time.Hour)
	for _, warehouseID := range run.Config.WarehouseIDs {
		tillID := run.TillByWarehouse[warehouseID]
		if tillID == run.BankCashBankID {
			continue
		}
		tillBalance, err := run.ReadCashBankBalance(tillID)
		if err != nil {
			return err
		}
		transferAmount := tillBalance - tillFloat
		if transferAmount <= 0 {
			continue
		}
		if err := run.PostManualCashMovement(transferTime, financeTypes.CashBankMovement{
			CashBankID:    tillID,
			CashBankRefID: run.BankCashBankID,
			Type:          financeTypes.CashMovementTypeTransfer,
			Amount:        -transferAmount,
			FinalAmount:   tillFloat,
		}); err != nil {
			return core.Err("no se pudo transferir la caja", tillID, "al banco:", err)
		}
		run.State.Stats.TillTransfers++
		transferTime = transferTime.Add(time.Minute)
	}
	return nil
}

// runTillCountsStep closes the month's tills: each one is counted — usually exact, sometimes a few
// soles short (charged to the cashier) or over (other income).
func runTillCountsStep(run *runner.Run, dayStart time.Time) error {
	if !runner.IsMonthDay(dayStart, 31) {
		return nil
	}
	countTime := dayStart.Add(21*time.Hour + 30*time.Minute)
	for _, warehouseID := range run.Config.WarehouseIDs {
		tillID := run.TillByWarehouse[warehouseID]
		if tillID == run.BankCashBankID {
			continue
		}
		systemAmount, err := run.ReadCashBankBalance(tillID)
		if err != nil {
			return err
		}
		difference, counterAccountCode := int32(0), int32(0)
		switch countOutcome := run.Random.Intn(100); {
		case countOutcome < 20:
			difference, counterAccountCode = -int32(100+run.Random.Intn(1901)), financeTypes.PCGEOtherStaffReceivables
		case countOutcome < 30:
			difference, counterAccountCode = int32(100+run.Random.Intn(701)), financeTypes.PCGEOtherOperatingIncome
		}

		reconciliation := struct {
			financeTypes.CashReconciliation
			CounterAccountCode int32
		}{
			CashReconciliation: financeTypes.CashReconciliation{
				CashBankID: tillID, SystemAmount: systemAmount, ActualAmount: systemAmount + difference,
			},
			CounterAccountCode: counterAccountCode,
		}
		bodyBytes, err := json.Marshal(reconciliation)
		if err != nil {
			return err
		}
		core.SetHistoricalUnix(countTime.Unix())
		request := run.MakeRequest("POST.cash-banks-reconciliation", nil, string(bodyBytes))
		responseBody := map[string]any{}
		if err := runner.DecodeResponse(run.Send(request, financeModule.PostCashReconciliation), &responseBody); err != nil {
			return core.Err("no se pudo cuadrar la caja", tillID, ":", err)
		}
		if storedBalance, isStale := responseBody["NeedUpdateSaldo"]; isStale {
			return core.Err("el saldo de la caja", tillID, "no coincide al cuadrar; saldo guardado:", storedBalance)
		}
		run.State.Stats.Reconciliations++
		countTime = countTime.Add(time.Minute)
	}
	return nil
}
