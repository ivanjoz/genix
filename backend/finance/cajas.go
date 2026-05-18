package finance

import (
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
	"encoding/json"
)

func GetCajas(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.UnixToSunix(req.GetQueryInt64("upd"))

	cajas := []financeTypes.CashBank{}
	query := db.Query(&cajas)
	query.Select().CompanyID.Equals(req.User.CompanyID)
	if updated > 0 {
		query.Updated.GreaterEqual(updated)
	} else {
		query.Status.Equals(1)
	}

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las cajas:", err)
	}

	//TODO: Eliminar luego
	for i := range cajas {
		e := &cajas[i]
		if e.Updated == 0 {
			e.Updated = 1
		}
	}

	response := map[string]any{
		"Cajas": cajas,
	}
	return core.MakeResponse(req, &response)
}

func PostCajas(req *core.HandlerArgs) core.HandlerResponse {
	core.Env.LOGS_DEBUG = true

	body := financeTypes.CashBank{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if body.ID == 0 || len(body.Nombre) == 0 || body.Tipo == 0 || body.SedeID == 0 {
		core.Print(body)
		return req.MakeErr("Faltan parámetros a enviar: (ID, Nombre, Tipo o SedeID)")
	}

	nowTime := core.SUnixTime()
	body.Updated = nowTime
	body.CompanyID = req.User.CompanyID

	// Autoincrement is handled automatically by the ORM via handlePreInsert
	body.CreatedBy = req.User.ID
	body.Created = nowTime
	body.UpdatedBy = req.User.ID
	cajas := &[]financeTypes.CashBank{body}

	// Insert or Update using db2
	if body.Created == nowTime {
		// New record - insert
		err = db.Insert(cajas)
	} else {
		// Existing record - update excluding specific fields
		q1 := db.Table[financeTypes.CashBank]()
		err = db.UpdateExclude(cajas, q1.CuadreFecha, q1.CuadreSaldo, q1.SaldoCurrent)
	}

	if err != nil {
		return req.MakeErr("Error al insertar/actualizar registros:", err)
	}

	return req.MakeResponse((*cajas)[0])
}

func GetCajaMovimientos(req *core.HandlerArgs) core.HandlerResponse {
	cashBankID := req.GetQueryInt("cashBank-id")

	if cashBankID == 0 {
		return req.MakeErr("No se envió la CashBank-ID")
	}

	lastRegistrosLimit := req.GetQueryInt("last-registros")

	movimientos := []financeTypes.CashBankMovement{}
	query := db.Query(&movimientos)
	query.Select().CompanyID.Equals(req.User.CompanyID).CajaID.Equals(cashBankID)

	// KeyIntPacking: CajaID (5) + Date (5) + Autoincrement (9)
	// Base formula: CajaID * 10^14 + Date * 10^9 + Autoincrement
	if lastRegistrosLimit > 0 {
		query.Limit(lastRegistrosLimit)
	} else {
		dateInicio := req.GetQueryInt16("date-inicio")
		dateFin := req.GetQueryInt16("date-fin")
		if dateInicio == 0 || dateFin == 0 {
			return req.MakeErr("Debe enviar una date de inicio y fin.")
		}

		query.Date.Between(dateInicio, dateFin)
	}
	query.OrderDesc()

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	core.Log("Movimientos obtenidos::", len(movimientos))

	// Usuarios resolved client-side via cache-by-ids (apiRoute "usuarios-ids") to keep payload small.
	response := map[string]any{
		"movimientos": movimientos,
	}

	return core.MakeResponse(req, &response)
}

func GetCajaCuadres(req *core.HandlerArgs) core.HandlerResponse {
	cashBankID := req.GetQueryInt("cashBank-id")
	if cashBankID == 0 {
		return req.MakeErr("No se envió la CashBank-ID")
	}

	lastRegistros := req.GetQueryInt("last-registros")
	lastRegistros = core.If(lastRegistros > 1000, 1000, lastRegistros)

	cuadres := []financeTypes.CashReconciliation{}
	query := db.Query(&cuadres)
	query.Select().CompanyID.Equals(req.User.CompanyID)
	if lastRegistros > 0 {
		query.ID.Between(core.SUnixTimeUUIDConcatID(cashBankID, 0), core.SUnixTimeUUIDConcatID(cashBankID+1, 0))
	} else {
		//TODO: completar?
	}
	query.OrderDesc().Limit(lastRegistros)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	// Usuarios resolved client-side via cache-by-ids (apiRoute "usuarios-ids") to keep payload small.
	response := map[string]any{
		"cuadres": cuadres,
	}

	return core.MakeResponse(req, &response)
}

func PostCajaCuadre(req *core.HandlerArgs) core.HandlerResponse {

	nowTime := core.SUnixTime()
	record := financeTypes.CashReconciliation{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if record.CajaID == 0 {
		return req.MakeErr("Faltan Parámetros: [CashBank-ID]")
	}

	cashBank, err := GetCaja(req.User.CompanyID, record.CajaID)
	if err != nil {
		return req.MakeErr(err)
	}

	if record.SaldoSistema != cashBank.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": cashBank.SaldoCurrent}
		return req.MakeResponse(&re)
	}

	// Guarda el cuadre
	record.CompanyID = req.User.CompanyID
	record.ID = core.SUnixTimeUUIDConcatID(record.CajaID)
	record.Created = nowTime
	record.CreatedBy = req.User.ID
	record.SaldoDiferencia = record.SaldoReal - cashBank.SaldoCurrent

	// Guarda la cashBank
	cashBank.CuadreSaldo = record.SaldoReal
	cashBank.CuadreFecha = nowTime
	cashBank.SaldoCurrent = record.SaldoReal
	cashBank.Updated = nowTime
	cashBank.UpdatedBy = req.User.ID

	// Guarda el movimiento
	movimiento := financeTypes.CashBankMovement{
		ID:         core.SUnixTimeUUIDConcatID(record.CajaID),
		CompanyID:  req.User.CompanyID,
		CajaID:     record.CajaID,
		Tipo:       2, // Cuadre de cashBank
		SaldoFinal: record.SaldoReal,
		Monto:      record.SaldoDiferencia,
		Created:    nowTime,
		CreatedBy:  req.User.ID,
	}

	// Insert records using db2
	if err := db.Insert(&[]financeTypes.CashReconciliation{record}); err != nil {
		core.Log("Error ScyllaDB inserting cuadre: ", err)
		return req.MakeErr("Error al registrar el cuadre:", err)
	}

	q1 := db.Table[financeTypes.CashBank]()
	if err := db.Update(&[]financeTypes.CashBank{cashBank}, q1.CuadreFecha, q1.CuadreSaldo, q1.SaldoCurrent, q1.Updated, q1.UpdatedBy); err != nil {
		core.Log("Error ScyllaDB updating cashBank: ", err)
		return req.MakeErr("Error al actualizar la cashBank:", err)
	}

	if err := db.Insert(&[]financeTypes.CashBankMovement{movimiento}); err != nil {
		core.Log("Error ScyllaDB inserting movimiento: ", err)
		return req.MakeErr("Error al registrar el movimiento:", err)
	}

	return req.MakeResponse(&record)
}

func PostMovimientoCaja(req *core.HandlerArgs) core.HandlerResponse {

	record := financeTypes.CashBankMovement{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body:", err)
	}

	if record.Tipo == 0 || record.Monto == 0 || record.CajaID == 0 {
		return req.MakeErr("Hay parámetros faltantes (Tipo, Monto o CashBank-ID)")
	}

	if record.Tipo == 3 && record.CajaRefID == 0 {
		return req.MakeErr("Las trasferencias necesitan especificar una cashBank de destino.")
	}

	cashBank, err := GetCaja(req.User.CompanyID, record.CajaID)
	if err != nil {
		return req.MakeErr(err)
	}

	saldoSistema := record.SaldoFinal - record.Monto

	if saldoSistema != cashBank.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": cashBank.SaldoCurrent}
		core.Log("El saldo de la cashBank no coincide con el del movimiento:", saldoSistema, cashBank.SaldoCurrent)
		return req.MakeResponse(&re)
	}

	movimientoInterno := financeTypes.InternalCashMovement{
		CajaID:     record.CajaID,
		CajaRefID:  record.CajaRefID,
		Tipo:       record.Tipo,
		Monto:      record.Monto,
		SaldoFinal: record.SaldoFinal,
	}

	if err := ApplyCajaMovimientos(req, []financeTypes.InternalCashMovement{movimientoInterno}); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(&record)
}
