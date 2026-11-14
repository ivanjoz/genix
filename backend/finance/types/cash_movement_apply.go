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

// FindCulqiCashBank returns the active account marked ReceivesCulqiPayments, or nil when the company
// has none. A company holds a handful of cash banks, so they are read whole instead of indexed.
func FindCulqiCashBank(companyID int32) (*CashBank, error) {
	cashBanks := []CashBank{}
	query := db.Query(&cashBanks)
	query.Select().CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las cajas de la empresa:", err)
	}
	for index := range cashBanks {
		if cashBanks[index].ReceivesCulqiPayments && cashBanks[index].Status > 0 {
			return &cashBanks[index], nil
		}
	}
	return nil, nil
}

// ApplyCashBankMovement writes the movements and the balances they leave in a transaction of their own.
func ApplyCashBankMovement(req *core.HandlerArgs, movimientos []InternalCashMovement) error {
	if len(movimientos) == 0 {
		return nil
	}
	// The balances are read locked and written with the movements in one transaction: two movements
	// of one till at once apply one after the other, never one over the other. A deadlock reruns this
	// body, which recomputes everything from what it reads.
	return db.RunTransaction(func(transaction *db.Transaction) error {
		return ApplyCashBankMovementIn(transaction, req, movimientos)
	})
}

// ApplyCashBankMovementIn is ApplyCashBankMovement inside the caller's transaction, so the movements
// commit with the document that caused them (a sale). It derives everything from what it reads in the
// transaction, so a deadlock retry of the caller's body reruns it cleanly.
func ApplyCashBankMovementIn(transaction *db.Transaction, req *core.HandlerArgs, movimientos []InternalCashMovement) error {
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

	cashBanks := make([]CashBank, 0, len(cashBankIDs.Values))
	for _, id := range cashBankIDs.Values {
		cashBanks = append(cashBanks, CashBank{CompanyID: req.User.CompanyID, ID: id})
	}
	if err := db.LockByKeys(transaction, &cashBanks); err != nil {
		return core.Err("Error al obtener las cajas:", err)
	}
	cashBankMap := make(map[int32]CashBank, len(cashBanks))
	for _, cashBank := range cashBanks {
		cashBankMap[cashBank.ID] = cashBank
	}
	for _, id := range cashBankIDs.Values {
		if _, isFound := cashBankMap[id]; !isFound {
			return core.Err("No se encontró la cashBank:", id)
		}
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

		// Validated here rather than in each handler, so no writer can store a classified type
		// without its counter-account — the Libro Diario would have no other side to post.
		accountCode, accountErr := ResolveCashMovementAccount(m.Type, m.Amount, m.CounterAccountCode)
		if accountErr != nil {
			return accountErr
		}

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
			CompanyID:          req.User.CompanyID,
			CashBankID:         m.CashBankID,
			CashBankRefID:      m.CashBankRefID,
			DocumentID:         m.DocumentID,
			ReferenceID:        m.ReferenceID,
			Date:               movementDate,
			Type:               m.Type,
			CounterAccountCode: accountCode,
			Amount:             m.Amount,
			FinalAmount:        m.FinalAmount,
			Created:            nowTime,
			CreatedBy:          req.User.ID,
			Period:             core.PeriodOfDay(movementDate),
			SameDayMovement:    m.SameDayMovement,
		}

		records = append(records, record)
		currentAmounts[m.CashBankID] = m.FinalAmount

		cashBank.CurrentAmount = m.FinalAmount
		cashBank.Updated = nowTime
		cashBank.UpdatedBy = req.User.ID
		cashBankMap[m.CashBankID] = cashBank
	}

	// Each movement's ID is a fresh time-based UUID: nothing is stored under it.
	if err := db.InsertIn(transaction, &records); err != nil {
		return core.Err("Error al insertar movimientos de cashBank:", err)
	}
	for _, id := range cashBankIDs.Values {
		cashBanksToUpdate = append(cashBanksToUpdate, cashBankMap[id])
	}
	q := db.TableOf[CashBank]()
	if err := db.UpdateIn(transaction, &cashBanksToUpdate, q.Status, q.CurrentAmount, q.Updated, q.UpdatedBy); err != nil {
		return core.Err("Error al actualizar saldo de las cajas:", err)
	}
	return nil
}
