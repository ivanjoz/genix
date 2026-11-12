package finance

import (
	"app/core"
	"app/db"
	"app/finance/types"
	"encoding/json"
)

func GetCashBanks(req *core.HandlerArgs) core.HandlerResponse {
	cajas := []types.CashBank{}
	query := db.Query(&cajas)
	// Delta() keeps only active rows on a first sync and every status afterwards, so the frontend
	// can evict deleted ones from its cache.
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las cajas:", err)
	}

	response := map[string]any{
		"Cajas": cajas,
	}
	return core.MakeResponse(req, &response)
}

func PostCashBanks(req *core.HandlerArgs) core.HandlerResponse {

	body := types.CashBank{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if body.ID == 0 || len(body.Name) == 0 || body.Type == 0 || body.SiteID == 0 {
		core.Print(body)
		return req.MakeErr("Faltan parámetros a enviar: (ID, Name, Type o SiteID)")
	}

	// A new cash bank arrives with a negative ID; the ORM replaces it with the autoincremented one
	// during handlePreInsert. Discriminating by ID is the only reliable signal, since the client
	// does not send the audit columns.
	isNewCashBank := body.ID < 0

	// Only one account may collect the web store's Culqi charges, or the checkout could not tell
	// which one to book them into.
	if body.ReceivesCulqiPayments {
		culqiCashBank, err := types.FindCulqiCashBank(req.User.CompanyID)
		if err != nil {
			return req.MakeErr(err)
		}
		if culqiCashBank != nil && culqiCashBank.ID != body.ID {
			return req.MakeErr(`La cuenta "` + culqiCashBank.Name + `" ya recibe los pagos de Culqi. Desmárquela antes de marcar otra.`)
		}
	}

	nowTime := core.SUnixTime()
	body.CompanyID = req.User.CompanyID
	body.Updated = nowTime
	body.UpdatedBy = req.User.ID
	if isNewCashBank {
		body.Created = nowTime
		body.CreatedBy = req.User.ID
	}

	cajas := &[]types.CashBank{body}

	if isNewCashBank {
		err = db.Upsert(cajas)
	} else {
		// The balance and audit columns are never sent by the client, so they are excluded from the
		// update: writing the zero values would wipe the reconciled amounts and the creation stamp.
		q1 := db.TableOf[types.CashBank]()
		err = db.UpdateExclude(cajas, q1.ReconciliationDate, q1.ReconciliationAmount,
			q1.CurrentAmount, q1.Created, q1.CreatedBy)
	}

	if err != nil {
		return req.MakeErr("Error al insertar/actualizar registros:", err)
	}

	return req.MakeResponse((*cajas)[0])
}

// GetCashBankMovements reads a cash bank's movements of a date range (?caja-id=&date-inicio=&date-fin=)
// as its (cash bank, day) cache groups, so a range read again downloads only the days that moved.
// ?last-registros=N instead reads the cash bank's latest N movements, uncached (a group with
// IndexID -1, which the browser passes through).
func GetCashBankMovements(req *core.HandlerArgs) core.HandlerResponse {
	cashBankID := req.GetQueryInt("caja-id")
	if cashBankID == 0 {
		return req.MakeErr("No se envió la CashBank-ID")
	}
	movementGroups := []db.RecordGroup[types.CashBankMovement]{}

	if lastMovementsLimit := req.GetQueryInt("last-registros"); lastMovementsLimit > 0 {
		latestMovements := []types.CashBankMovement{}
		query := db.Query(&latestMovements)
		query.Select().CompanyID.Equals(req.User.CompanyID).CashBankID.Equals(cashBankID).
			OrderDesc().Limit(min(lastMovementsLimit, 1000))
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
		}
		movementGroups = append(movementGroups, db.RecordGroup[types.CashBankMovement]{IndexID: -1, Records: latestMovements})
		return core.MakeResponse(req, &movementGroups)
	}

	dateStart := req.GetQueryInt16("date-inicio")
	dateEnd := req.GetQueryInt16("date-fin")
	if dateStart <= 0 || dateEnd <= 0 {
		return req.MakeErr("Debe enviar una fecha de inicio y fin.")
	}
	if dateEnd < dateStart {
		return req.MakeErr("La fecha final no puede ser menor a la fecha inicial.")
	}
	// Each day is one group to probe and maybe fetch: cap the range.
	if dateEnd-dateStart > 120 {
		return req.MakeErr("Sólo se pueden consultar hasta 120 días a la vez.")
	}

	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}
	query := db.QueryIndexGroup(&movementGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.CashBankID.Equals(cashBankID).Date.Between(dateStart, dateEnd)
	if err := query.Exec(); err != nil {
		core.Log("Error querying cash bank movement groups:", err)
		return req.MakeErr("Error al obtener los movimientos de las cajas.")
	}
	// Users are resolved client-side via cache-by-ids (apiRoute "users-ids").
	return core.MakeResponse(req, &movementGroups)
}

// GetCashBankMovementByID returns the movements tied to one document (?document-id=, e.g. an
// Expense.ID) as that document's single cache group: the browser downloads it again only after a
// new movement of it. DocumentID is shared across document kinds, so the caller filters by Type.
func GetCashBankMovementByID(req *core.HandlerArgs) core.HandlerResponse {
	documentID := req.GetQueryInt64("document-id")
	if documentID <= 0 {
		return req.MakeErr("Debe enviar un document-id.")
	}
	heldGroups, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}
	movementGroups := []db.RecordGroup[types.CashBankMovement]{}
	query := db.QueryIndexGroup(&movementGroups).CompanyID.Equals(req.User.CompanyID)
	for _, heldGroup := range heldGroups {
		query.IncludeCachedGroup(heldGroup.GroupHash, heldGroup.LastWrite)
	}
	query.DocumentID.Between(documentID, documentID)
	if err := query.Exec(); err != nil {
		core.Log("Error querying document cash movement group:", err)
		return req.MakeErr("Error al obtener los movimientos.")
	}
	return core.MakeResponse(req, &movementGroups)
}

// GetCashReconciliation syncs one cash bank's physical counts (?caja-id=) through its delta log.
func GetCashReconciliation(req *core.HandlerArgs) core.HandlerResponse {
	cashBankID := req.GetQueryInt("caja-id")
	if cashBankID == 0 {
		return req.MakeErr("No se envió la CashBank-ID")
	}

	reconciliations := []types.CashReconciliation{}
	query := db.Query(&reconciliations)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), cashBankID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los cuadres de la caja:", err)
	}
	// Users are resolved client-side via cache-by-ids (apiRoute "users-ids").
	return req.MakeResponse(reconciliations)
}

func PostCashReconciliation(req *core.HandlerArgs) core.HandlerResponse {

	nowTime := core.SUnixTime()
	// The account classifies the difference, so it travels with the request and is stored on
	// the movement, not on the reconciliation.
	body := struct {
		types.CashReconciliation
		CounterAccountCode int32
	}{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}
	record := body.CashReconciliation

	if record.CashBankID == 0 {
		return req.MakeErr("Faltan Parámetros: [CashBank-ID]")
	}

	// The till is locked from the balance check to the write: a movement landing in between would
	// otherwise be erased by setting CurrentAmount to the counted amount. Under the lock it either
	// committed before (and the check below asks the client to refresh) or waits for the count.
	staleBalance := int32(0)
	isBalanceStale := false
	err = db.RunTransaction(func(transaction *db.Transaction) error {
		cashBanks := []types.CashBank{{CompanyID: req.User.CompanyID, ID: record.CashBankID}}
		if err := db.LockByKeys(transaction, &cashBanks); err != nil {
			return core.Err("Error al obtener la caja:", err)
		}
		if len(cashBanks) == 0 {
			return core.Err("No se encontró la cashBank")
		}
		cashBank := cashBanks[0]
		isBalanceStale = record.SystemAmount != cashBank.CurrentAmount
		if isBalanceStale {
			staleBalance = cashBank.CurrentAmount
			return nil
		}
		return reconcileCashBankIn(transaction, req, &record, cashBank, body.CounterAccountCode, nowTime)
	})
	if err != nil {
		return req.MakeErr(err)
	}
	if isBalanceStale {
		re := map[string]any{"NeedUpdateSaldo": staleBalance}
		return req.MakeResponse(&re)
	}
	return req.MakeResponse(&record)
}

// reconcileCashBankIn writes a physical count of the cash bank its caller locked: the reconciliation,
// the difference movement and the counted balance.
func reconcileCashBankIn(transaction *db.Transaction, req *core.HandlerArgs, record *types.CashReconciliation,
	cashBank types.CashBank, counterAccountCode int32, nowTime int32,
) error {
	record.CompanyID = req.User.CompanyID
	record.ID = core.SUnixTimeUUIDConcatID(record.CashBankID)
	record.Created = nowTime
	record.CreatedBy = req.User.ID
	record.DifferenceAmount = record.ActualAmount - cashBank.CurrentAmount

	// The sign is the server's own difference, not the client's, so a shortage can never be
	// filed as a surplus.
	accountCode, err := types.ResolveCashMovementAccount(
		types.CashMovementTypePhysicalCount, record.DifferenceAmount, counterAccountCode)
	if err != nil {
		return err
	}

	// Update the cash bank
	cashBank.ReconciliationAmount = record.ActualAmount
	cashBank.ReconciliationDate = nowTime
	cashBank.CurrentAmount = record.ActualAmount
	cashBank.Updated = nowTime
	cashBank.UpdatedBy = req.User.ID

	// Record the reconciliation movement
	movimiento := types.CashBankMovement{
		ID:         core.SUnixTimeUUIDConcatID(record.CashBankID),
		CompanyID:  req.User.CompanyID,
		CashBankID: record.CashBankID,
		// The Libro Diario reads movements by period, so a count without a date would never
		// reach it.
		Date:               core.FechaUnix(),
		Type:               types.CashMovementTypePhysicalCount,
		CounterAccountCode: accountCode,
		FinalAmount:        record.ActualAmount,
		Amount:             record.DifferenceAmount,
		Created:            nowTime,
		CreatedBy:          req.User.ID,
		Period:             core.PeriodOfDay(core.FechaUnix()),
	}

	// Both IDs are fresh time-based UUIDs: nothing is stored under them.
	if err := db.InsertIn(transaction, &[]types.CashReconciliation{*record}); err != nil {
		return core.Err("Error al registrar el cuadre:", err)
	}
	q1 := db.TableOf[types.CashBank]()
	if err := db.UpdateIn(transaction, &[]types.CashBank{cashBank}, q1.Status, q1.ReconciliationDate, q1.ReconciliationAmount, q1.CurrentAmount, q1.Updated, q1.UpdatedBy); err != nil {
		return core.Err("Error al actualizar la cashBank:", err)
	}
	if err := db.InsertIn(transaction, &[]types.CashBankMovement{movimiento}); err != nil {
		return core.Err("Error al registrar el movimiento:", err)
	}
	return nil
}

func PostCashBankMovement(req *core.HandlerArgs) core.HandlerResponse {

	record := types.CashBankMovement{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body:", err)
	}

	if record.Type == 0 || record.Amount == 0 || record.CashBankID == 0 {
		return req.MakeErr("Hay parámetros faltantes (Type, Amount o CashBank-ID)")
	}
	if err := types.ValidateManualCashMovement(record.Type, record.Amount); err != nil {
		return req.MakeErr(err)
	}

	cashBank, err := types.GetCaja(req.User.CompanyID, record.CashBankID)
	if err != nil {
		return req.MakeErr(err)
	}

	saldoSistema := record.FinalAmount - record.Amount

	if saldoSistema != cashBank.CurrentAmount {
		re := map[string]any{"NeedUpdateSaldo": cashBank.CurrentAmount}
		core.Log("El saldo de la cashBank no coincide con el del movimiento:", saldoSistema, cashBank.CurrentAmount)
		return req.MakeResponse(&re)
	}

	movements := []types.InternalCashMovement{}
	if record.Type == types.CashMovementTypeTransfer {
		if record.CashBankRefID == 0 {
			return req.MakeErr("Las transferencias necesitan especificar una caja de destino.")
		}
		// GetCaja is scoped to the company, so a destination id from another tenant is not found.
		destinationCashBank, err := types.GetCaja(req.User.CompanyID, record.CashBankRefID)
		if err != nil {
			return req.MakeErr(err)
		}
		movements, err = types.BuildCashTransferMovements(
			cashBank, destinationCashBank, record.Amount, record.FinalAmount)
		if err != nil {
			return req.MakeErr(err)
		}
	} else {
		// Only a transfer names a counterpart register.
		record.CashBankRefID = 0
		movements = append(movements, types.InternalCashMovement{
			CashBankID:         record.CashBankID,
			Type:               record.Type,
			CounterAccountCode: record.CounterAccountCode,
			Amount:             record.Amount,
			FinalAmount:        record.FinalAmount,
		})
	}

	// Both sides of a transfer go in one call, so the two balances move together.
	if err := types.ApplyCashBankMovement(req, movements); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(&record)
}
