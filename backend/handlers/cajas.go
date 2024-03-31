package handlers

import (
	"app/core"
	"app/shared"
	s "app/types"
	"encoding/json"
)

func GetCajas(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.UnixToSunix(req.GetQueryInt64("upd"))

	cajas := []s.Caja{}
	query := core.DBSelect(&cajas).
		Where("empresa_id").Equals(req.Usuario.EmpresaID)

	if updated > 0 {
		query = query.Where("updated").GreatEq(updated)
	} else {
		query = query.Where("status").Equals(1)
	}
	query.Exec()

	response := map[string]any{
		"Cajas": cajas,
	}

	return core.MakeResponse(req, &response)
}

func PostCajas(req *core.HandlerArgs) core.HandlerResponse {
	core.Env.LOGS_DEBUG = true

	body := s.Caja{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if body.ID == 0 || len(body.Nombre) == 0 || body.Tipo == 0 || body.SedeID == 0 {
		core.Print(body)
		return req.MakeErr("Faltan parámetros a enviar: (ID, Nombre, Tipo o SedeID)")
	}

	nowTime := core.SunixTime()
	body.Updated = nowTime
	body.EmpresaID = req.Usuario.EmpresaID

	if body.ID < 0 {
		counter, err := core.GetCounter("cajas", 1, req.Usuario.EmpresaID)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
		body.CreatedBy = req.Usuario.ID
		body.Created = nowTime
	} else {
		body.UpdatedBy = req.Usuario.ID
	}

	core.Print(body)

	err = core.DBUpdateInsert(
		&[]s.Caja{body},
		func(e s.Caja) bool { return e.Created == nowTime },
		[]string{"fecha_cuadre", "monto_cuadre", "monto_current"},
	)

	if err != nil {
		return req.MakeErr("Error al insertar/actualizar registros:", err)
	}

	return req.MakeResponse(body)
}

func GetCajaMovimientos(req *core.HandlerArgs) core.HandlerResponse {
	cajaID := req.GetQueryInt("caja-id")
	if cajaID == 0 {
		return req.MakeErr("No se envió la Caja-ID")
	}

	fechaHInicio := req.GetQueryInt64("fecha-hora-inicio")
	fechaHFin := req.GetQueryInt64("fecha-hora-fin")
	lastRegistros := req.GetQueryInt("last-registros")
	lastRegistros = core.If(lastRegistros > 1000, 1000, lastRegistros)

	movimientos := []s.CajaMovimiento{}
	query := core.DBSelect(&movimientos).
		Where("empresa_id").Equals(req.Usuario.EmpresaID)

	if lastRegistros > 0 {
		query = query.Where("id").GreatEq(core.SunixUUIDx2FromID(cajaID, 0)).
			Where("id").LessEq(core.SunixUUIDx2FromID(cajaID+1, 0))
		query.Limit(lastRegistros)
	} else if fechaHInicio > 0 && fechaHFin > 0 {
		query = query.Where("id").GreatEq(core.SunixUUIDx2FromID(cajaID, fechaHInicio)).
			Where("id").LessEq(core.SunixUUIDx2FromID(cajaID, fechaHFin+1))
	}

	err := query.OrderDescending().Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	core.Log("Movimientos obtenidos::", len(movimientos))

	usuarios, err := shared.GetUsuarios(req.Usuario.EmpresaID,
		core.Map(movimientos, func(e s.CajaMovimiento) int32 { return e.CreatedBy }))

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

	cuadres := []s.CajaCuadre{}
	query := core.DBSelect(&cuadres).
		Where("empresa_id").Equals(req.Usuario.EmpresaID)

	if lastRegistros > 0 {
		query = query.Where("id").GreatEq(core.SunixUUIDx2FromID(cajaID, 0)).
			Where("id").LessEq(core.SunixUUIDx2FromID(cajaID+1, 0))
		query.Limit(lastRegistros)
	} else {

	}

	err := query.OrderDescending().Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	usuarios, err := shared.GetUsuarios(req.Usuario.EmpresaID,
		core.Map(cuadres, func(e s.CajaCuadre) int32 { return e.CreatedBy }))

	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	response := map[string]any{
		"usuarios": usuarios,
		"cuadres":  cuadres,
	}

	return core.MakeResponse(req, &response)
}

func PostCajaCuadre(req *core.HandlerArgs) core.HandlerResponse {

	nowTime := core.SunixTime()
	record := s.CajaCuadre{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if record.CajaID == 0 {
		return req.MakeErr("Faltan Parámetros: [Caja-ID]")
	}

	caja, err := shared.GetCaja(req.Usuario.EmpresaID, record.CajaID)
	if err != nil {
		return req.MakeErr(err)
	}

	if record.SaldoSistema != caja.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": caja.SaldoCurrent}
		return req.MakeResponse(&re)
	}

	// Guarda el cuadre
	record.EmpresaID = req.Usuario.EmpresaID
	record.ID = core.SunixUUIDx2FromID(record.CajaID)
	record.Created = nowTime
	record.CreatedBy = req.Usuario.ID
	record.SaldoDiferencia = record.SaldoReal - caja.SaldoCurrent
	statements := core.MakeInsertQuery(&[]s.CajaCuadre{record})

	// Guarda la caja
	caja.CuadreSaldo = record.SaldoReal
	caja.CuadreFecha = nowTime
	caja.SaldoCurrent = record.SaldoReal
	caja.Updated = nowTime
	caja.UpdatedBy = req.Usuario.ID
	statements = append(statements, core.MakeUpdateQuery(
		&[]s.Caja{caja}, "cuadre_fecha", "cuadre_saldo", "saldo_current", "updated",
		"updated_by")...)

	// Guarda el movimiento
	movimiento := s.CajaMovimiento{
		ID:         core.SunixUUIDx2FromID(record.CajaID),
		EmpresaID:  req.Usuario.EmpresaID,
		CajaID:     record.CajaID,
		Tipo:       2, // Cuadre de caja
		SaldoFinal: record.SaldoReal,
		Monto:      record.SaldoDiferencia,
		Created:    nowTime,
		CreatedBy:  req.Usuario.ID,
	}

	statements = append(statements,
		core.MakeInsertQuery(&[]s.CajaMovimiento{movimiento})...)

	statement := core.MakeQueryStatement(statements)
	core.Log(statement)

	if err := core.ScyllaConnect().Query(statement).Exec(); err != nil {
		core.Log("Error ScyllaDB: ", err)
		return req.MakeErr("Error al registrar el cuadre:", err)
	}

	return req.MakeResponse(&record)
}

func PostMovimientoCaja(req *core.HandlerArgs) core.HandlerResponse {

	nowTime := core.SunixTime()
	record := s.CajaMovimiento{}
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

	caja, err := shared.GetCaja(req.Usuario.EmpresaID, record.CajaID)
	if err != nil {
		return req.MakeErr(err)
	}

	saldoSistema := record.SaldoFinal - record.Monto

	if saldoSistema != caja.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": caja.SaldoCurrent}
		core.Log("El saldo de la caja no coincide con el del movimiento:", saldoSistema, caja.SaldoCurrent)
		return req.MakeResponse(&re)
	}

	// Statement para guardar el movimiento
	record.EmpresaID = req.Usuario.EmpresaID
	record.Created = nowTime
	record.CreatedBy = req.Usuario.ID
	record.ID = core.SunixUUIDx2FromID(record.CajaID)

	statements := core.MakeInsertQuery(&[]s.CajaMovimiento{record})

	// Statement para actualizar la caja
	caja.SaldoCurrent = record.SaldoFinal
	caja.Updated = nowTime
	caja.UpdatedBy = req.Usuario.ID
	statements = append(statements, core.MakeUpdateQuery(
		&[]s.Caja{caja}, "cuadre_fecha", "cuadre_saldo", "saldo_current", "updated",
		"updated_by")...)

	statement := core.MakeQueryStatement(statements)
	core.Log(statement)

	if err := core.ScyllaConnect().Query(statement).Exec(); err != nil {
		core.Log("Error ScyllaDB: ", err)
		return req.MakeErr("Error al registrar el cuadre:", err)
	}

	return req.MakeResponse(&record)
}
