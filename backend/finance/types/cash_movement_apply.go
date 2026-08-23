// Cash ledger writer. This is business logic in a `types` package on purpose: every module
// that moves money — sales collections, purchase-order supplier payments, expense payments,
// asset payments — has to write through it, and a module body may not import another module
// body (see backend/docs/MODULE_BOUNDARIES.md). Living here is what lets `accounting`,
// `logistics` and `sales` call it while importing only `app/finance/types`.
//
// It depends on nothing but core, db and this package's own tables, so it stays a leaf.

package types

import (
	"app/core"
	"app/db"
)

func GetCaja(companyID, cashBankID int32) (CashBank, error) {
	cajas := []CashBank{}
	query := db.Query(&cajas)
	query.Select().
		CompanyID.Equals(companyID).
		ID.Equals(cashBankID)

	if err := query.Exec(); err != nil {
		return CashBank{}, core.Err("Error al obtener información de la cashBank:", err)
	}
	if len(cajas) == 0 {
		return CashBank{}, core.Err("No se encontró la cashBank")
	}
	return cajas[0], nil
}

func ApplyCashBankMovement(req *core.HandlerArgs, movimientos []InternalCashMovement) error {
	if len(movimientos) == 0 {
		return nil
	}

	nowTime := core.SUnixTime()
	dateUnix := core.FechaUnix()

	// Group movements by CashBankID to track running balances.
	cashBankIDs := core.SliceSet[int32]{}
	for _, m := range movimientos {
		cashBankIDs.Add(m.CashBankID)
	}

	cashBankMap := make(map[int32]CashBank)
	for _, id := range cashBankIDs.Values {
		cashBank, err := GetCaja(req.User.CompanyID, id)
		if err != nil {
			return core.Err("Error al obtener la cashBank:", id, err)
		}
		cashBankMap[id] = cashBank
	}

	records := []CashBankMovement{}
	cashBanksToUpdate := []CashBank{}

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

		record := CashBankMovement{
			ID:            core.SUnixTimeUUIDConcatID(m.CashBankID, core.SUnixTimeUUID()),
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

	q := db.TableOf[CashBank]()
	// Status is part of the delta-view key and must be written with the managed UpdatedVersion.
	if err := db.Update(&cashBanksToUpdate, q.Status, q.CurrentAmount, q.Updated, q.UpdatedBy); err != nil {
		return core.Err("Error al actualizar saldo de las cajas:", err)
	}

	return nil
}
