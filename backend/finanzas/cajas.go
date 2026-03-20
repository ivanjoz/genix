package finanzas

import (
	"app/core"
	"app/db"
	finanzasTypes "app/finanzas/types"
	"app/seguridad"
	"encoding/json"
)

func GetCajas(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.UnixToSunix(req.GetQueryInt64("upd"))

	cajas := []finanzasTypes.Caja{}
	query := db.Query(&cajas)
	query.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
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

	body := finanzasTypes.Caja{}
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
	body.EmpresaID = req.Usuario.EmpresaID

	// Autoincrement is handled automatically by the ORM via handlePreInsert
	body.CreatedBy = req.Usuario.ID
	body.Created = nowTime
	body.UpdatedBy = req.Usuario.ID
	cajas := &[]finanzasTypes.Caja{body}

	// Insert or Update using db2
	if body.Created == nowTime {
		// New record - insert
		err = db.Insert(cajas)
	} else {
		// Existing record - update excluding specific fields
		q1 := db.Table[finanzasTypes.Caja]()
		err = db.UpdateExclude(cajas, q1.CuadreFecha, q1.CuadreSaldo, q1.SaldoCurrent)
	}

	if err != nil {
		return req.MakeErr("Error al insertar/actualizar registros:", err)
	}

	return req.MakeResponse((*cajas)[0])
}

func GetCajaMovimientos(req *core.HandlerArgs) core.HandlerResponse {
	cajaID := req.GetQueryInt("caja-id")

	if cajaID == 0 {
		return req.MakeErr("No se envió la Caja-ID")
	}

	lastRegistrosLimit := req.GetQueryInt("last-registros")

	movimientos := []finanzasTypes.CajaMovimiento{}
	query := db.Query(&movimientos)
	query.Select().EmpresaID.Equals(req.Usuario.EmpresaID).CajaID.Equals(cajaID)

	// KeyIntPacking: CajaID (5) + Fecha (5) + Autoincrement (9)
	// Base formula: CajaID * 10^14 + Fecha * 10^9 + Autoincrement
	if lastRegistrosLimit > 0 {
		query.Limit(lastRegistrosLimit)
	} else {
		fechaInicio := req.GetQueryInt16("fecha-inicio")
		fechaFin := req.GetQueryInt16("fecha-fin")
		if fechaInicio == 0 || fechaFin == 0 {
			return req.MakeErr("Debe enviar una fecha de inicio y fin.")
		}

		query.Fecha.Between(fechaInicio, fechaFin)
	}
	query.OrderDesc()

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	core.Log("Movimientos obtenidos::", len(movimientos))

	usuarios, err := seguridad.GetUsuariosList(req.Usuario.EmpresaID,
		core.Map(movimientos, func(e finanzasTypes.CajaMovimiento) int32 { return e.CreatedBy }))

	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	response := map[string]any{
		"movimientos": movimientos,
		"usuarios":    usuarios,
	}

	return core.MakeResponse(req, &response)
}

func GetCajaCuadres(req *core.HandlerArgs) core.HandlerResponse {
	cajaID := req.GetQueryInt("caja-id")
	if cajaID == 0 {
		return req.MakeErr("No se envió la Caja-ID")
	}

	lastRegistros := req.GetQueryInt("last-registros")
	lastRegistros = core.If(lastRegistros > 1000, 1000, lastRegistros)

	cuadres := []finanzasTypes.CajaCuadre{}
	query := db.Query(&cuadres)
	query.Select().EmpresaID.Equals(req.Usuario.EmpresaID)
	if lastRegistros > 0 {
		query.ID.Between(core.SUnixTimeUUIDConcatID(cajaID, 0), core.SUnixTimeUUIDConcatID(cajaID+1, 0))
	} else {
		//TODO: completar?
	}
	query.OrderDesc().Limit(lastRegistros)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	usuarios, err := seguridad.GetUsuariosList(req.Usuario.EmpresaID,
		core.Map(cuadres, func(e finanzasTypes.CajaCuadre) int32 { return e.CreatedBy }))

	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	response := map[string]any{
		"usuarios": usuarios, "cuadres": cuadres,
	}

	return core.MakeResponse(req, &response)
}

func PostCajaCuadre(req *core.HandlerArgs) core.HandlerResponse {

	nowTime := core.SUnixTime()
	record := finanzasTypes.CajaCuadre{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if record.CajaID == 0 {
		return req.MakeErr("Faltan Parámetros: [Caja-ID]")
	}

	caja, err := GetCaja(req.Usuario.EmpresaID, record.CajaID)
	if err != nil {
		return req.MakeErr(err)
	}

	if record.SaldoSistema != caja.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": caja.SaldoCurrent}
		return req.MakeResponse(&re)
	}

	// Guarda el cuadre
	record.EmpresaID = req.Usuario.EmpresaID
	record.ID = core.SUnixTimeUUIDConcatID(record.CajaID)
	record.Created = nowTime
	record.CreatedBy = req.Usuario.ID
	record.SaldoDiferencia = record.SaldoReal - caja.SaldoCurrent

	// Guarda la caja
	caja.CuadreSaldo = record.SaldoReal
	caja.CuadreFecha = nowTime
	caja.SaldoCurrent = record.SaldoReal
	caja.Updated = nowTime
	caja.UpdatedBy = req.Usuario.ID

	// Guarda el movimiento
	movimiento := finanzasTypes.CajaMovimiento{
		ID:         core.SUnixTimeUUIDConcatID(record.CajaID),
		EmpresaID:  req.Usuario.EmpresaID,
		CajaID:     record.CajaID,
		Tipo:       2, // Cuadre de caja
		SaldoFinal: record.SaldoReal,
		Monto:      record.SaldoDiferencia,
		Created:    nowTime,
		CreatedBy:  req.Usuario.ID,
	}

	// Insert records using db2
	if err := db.Insert(&[]finanzasTypes.CajaCuadre{record}); err != nil {
		core.Log("Error ScyllaDB inserting cuadre: ", err)
		return req.MakeErr("Error al registrar el cuadre:", err)
	}

	q1 := db.Table[finanzasTypes.Caja]()
	if err := db.Update(&[]finanzasTypes.Caja{caja}, q1.CuadreFecha, q1.CuadreSaldo, q1.SaldoCurrent, q1.Updated, q1.UpdatedBy); err != nil {
		core.Log("Error ScyllaDB updating caja: ", err)
		return req.MakeErr("Error al actualizar la caja:", err)
	}

	if err := db.Insert(&[]finanzasTypes.CajaMovimiento{movimiento}); err != nil {
		core.Log("Error ScyllaDB inserting movimiento: ", err)
		return req.MakeErr("Error al registrar el movimiento:", err)
	}

	return req.MakeResponse(&record)
}

func PostMovimientoCaja(req *core.HandlerArgs) core.HandlerResponse {

	record := finanzasTypes.CajaMovimiento{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body:", err)
	}

	if record.Tipo == 0 || record.Monto == 0 || record.CajaID == 0 {
		return req.MakeErr("Hay parámetros faltantes (Tipo, Monto o Caja-ID)")
	}

	if record.Tipo == 3 && record.CajaRefID == 0 {
		return req.MakeErr("Las trasferencias necesitan especificar una caja de destino.")
	}

	caja, err := GetCaja(req.Usuario.EmpresaID, record.CajaID)
	if err != nil {
		return req.MakeErr(err)
	}

	saldoSistema := record.SaldoFinal - record.Monto

	if saldoSistema != caja.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": caja.SaldoCurrent}
		core.Log("El saldo de la caja no coincide con el del movimiento:", saldoSistema, caja.SaldoCurrent)
		return req.MakeResponse(&re)
	}

	movimientoInterno := finanzasTypes.CajaMovimientoInterno{
		CajaID:     record.CajaID,
		CajaRefID:  record.CajaRefID,
		Tipo:       record.Tipo,
		Monto:      record.Monto,
		SaldoFinal: record.SaldoFinal,
	}

	if err := ApplyCajaMovimientos(req, []finanzasTypes.CajaMovimientoInterno{movimientoInterno}); err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(&record)
}
