package finance

import (
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
)

func ApplyCashBankMovement(req *core.HandlerArgs, movimientos []financeTypes.InternalCashMovement) error {
	if len(movimientos) == 0 {
		return nil
	}

	nowTime := req.EffectiveSUnixTime()
	dateUnix := req.EffectiveFechaUnix()

	// Group movements by CashBankID to track running balances.
	cashBankIDs := core.SliceSet[int32]{}
	for _, m := range movimientos {
		cashBankIDs.Add(m.CashBankID)
	}

	cashBankMap := make(map[int32]financeTypes.CashBank)
	for _, id := range cashBankIDs.Values {
		cashBank, err := GetCaja(req.User.CompanyID, id)
		if err != nil {
			return core.Err("Error al obtener la cashBank:", id, err)
		}
		cashBankMap[id] = cashBank
	}

	records := []financeTypes.CashBankMovement{}
	cashBanksToUpdate := []financeTypes.CashBank{}

	// Track current balance per cash bank across multiple movements in the same batch.
	currentAmounts := make(map[int32]int32)
	for id, cashBank := range cashBankMap {
		currentAmounts[id] = cashBank.CurrentAmount
	}

	for _, m := range movimientos {
		cashBank := cashBankMap[m.CashBankID]
		previousAmount := currentAmounts[m.CashBankID]

		// If FinalAmount is provided, validate it matches the calculated result.
		if m.FinalAmount != 0 {
			calculatedAmount := previousAmount + m.Amount
			if m.FinalAmount != calculatedAmount {
				return core.Err("El saldo final enviado no coincide con el sistema para la cashBank:", m.CashBankID)
			}
		} else {
			m.FinalAmount = previousAmount + m.Amount
		}

		// Honor a caller-supplied movement date (e.g. an expense payment date); else use the request date.
		movementDate := dateUnix
		if m.Date != 0 {
			movementDate = m.Date
		}

		record := financeTypes.CashBankMovement{
			ID:            core.SUnixTimeUUIDConcatID(m.CashBankID, req.EffectiveSUnixTimeUUID()),
			CompanyID:     req.User.CompanyID,
			CashBankID:    m.CashBankID,
			CashBankRefID: m.CashBankRefID,
			DocumentID:    m.DocumentID,
			ReferenceID:   m.ReferenceID,
			Date:          movementDate,
			Type:          m.Type,
			Amount:        m.Amount,
			FinalAmount:   m.FinalAmount,
			Created:       nowTime,
			CreatedBy:     req.User.ID,
		}

		records = append(records, record)
		currentAmounts[m.CashBankID] = m.FinalAmount

		cashBank.CurrentAmount = m.FinalAmount
		cashBank.Updated = nowTime
		cashBank.UpdatedBy = req.User.ID
		cashBankMap[m.CashBankID] = cashBank
	}

	if err := db.Insert(&records); err != nil {
		return core.Err("Error al insertar movimientos de cashBank:", err)
	}

	for _, id := range cashBankIDs.Values {
		cashBanksToUpdate = append(cashBanksToUpdate, cashBankMap[id])
	}

	q := db.Table[financeTypes.CashBank]()
	if err := db.Update(&cashBanksToUpdate, q.CurrentAmount, q.Updated, q.UpdatedBy); err != nil {
		return core.Err("Error al actualizar saldo de las cajas:", err)
	}

	return nil
}
