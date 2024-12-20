package handlers

import (
	"app/core"
	"app/db"
	"app/shared"
	s "app/types"
	"encoding/json"
)

func GetCajas(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.UnixToSunix(req.GetQueryInt64("upd"))

	cajas := db.Select(func(q *db.Query[s.Caja], col s.Caja) {
		q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
		if updated > 0 {
			q.Where(col.Updated_().GreaterEqual(updated))
		} else {
			q.Where(col.Status_().Equals(1))
		}
	})

	if cajas.Err != nil {
		return req.MakeErr("Error al obtener las cajas:", cajas.Err)
	}

	//TODO: Eliminar luego
	for i := range cajas.Records {
		e := &cajas.Records[i]
		if e.Updated == 0 {
			e.Updated = 1
		}
	}

	response := map[string]any{
		"Cajas": cajas.Records,
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

	err = db.InsertOrUpdate(
		&[]s.Caja{body},
		func(e *s.Caja) bool { return e.Created == nowTime },
		[]db.Coln{body.CuadreFecha_(), body.CuadreSaldo_(), body.SaldoCurrent_()},
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

	movimientos := db.Select(func(q *db.Query[s.CajaMovimiento], col s.CajaMovimiento) {
		q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
		if lastRegistros > 0 {
			q.Where(col.ID_().Between(
				core.SunixUUIDx2FromID(cajaID, 0), core.SunixUUIDx2FromID(cajaID+1, 0)))
		} else {
			q.Where(col.ID_().Between(
				core.SunixUUIDx2FromID(cajaID, fechaHInicio),
				core.SunixUUIDx2FromID(cajaID, fechaHFin+1)))
		}
		q.OrderDescending()
	})

	if movimientos.Err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", movimientos.Err)
	}

	core.Log("Movimientos obtenidos::", len(movimientos.Records))

	usuarios, err := shared.GetUsuarios(req.Usuario.EmpresaID,
		core.Map(movimientos.Records, func(e s.CajaMovimiento) int32 { return e.CreatedBy }))

	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	response := map[string]any{
		"movimientos": movimientos.Records,
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

	cuadres := db.Select(func(q *db.Query[s.CajaCuadre], col s.CajaCuadre) {
		q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
		if lastRegistros > 0 {
			q.Where(col.ID_().Between(core.SunixUUIDx2FromID(cajaID, 0), core.SunixUUIDx2FromID(cajaID+1, 0)))
		} else {
			//TODO: completar?
		}
		q.OrderDescending().Limit(lastRegistros)
	})

	if cuadres.Err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", cuadres.Err)
	}

	usuarios, err := shared.GetUsuarios(req.Usuario.EmpresaID,
		core.Map(cuadres.Records, func(e s.CajaCuadre) int32 { return e.CreatedBy }))

	if err != nil {
		return req.MakeErr("Error al obtener los usuarios.", err)
	}

	response := map[string]any{
		"usuarios": usuarios, "cuadres": cuadres,
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
	statements := db.MakeInsertStatement(&[]s.CajaCuadre{record})

	// Guarda la caja
	caja.CuadreSaldo = record.SaldoReal
	caja.CuadreFecha = nowTime
	caja.SaldoCurrent = record.SaldoReal
	caja.Updated = nowTime
	caja.UpdatedBy = req.Usuario.ID

	statements = append(statements, db.MakeUpdateStatements(
		&[]s.Caja{caja}, caja.CuadreFecha_(), caja.CuadreSaldo_(), caja.SaldoCurrent_(),
		caja.Updated_(), caja.UpdatedBy_())...)

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
		db.MakeInsertStatement(&[]s.CajaMovimiento{movimiento})...)

	if err := db.QueryExecStatements(statements); err != nil {
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

	statements := db.MakeInsertStatement(&[]s.CajaMovimiento{record})

	// Statement para actualizar la caja
	caja.SaldoCurrent = record.SaldoFinal
	caja.Updated = nowTime
	caja.UpdatedBy = req.Usuario.ID
	statements = append(statements, db.MakeUpdateStatements(
		&[]s.Caja{caja}, caja.CuadreFecha_(), caja.CuadreSaldo_(), caja.SaldoCurrent_(),
		caja.Updated_(), caja.UpdatedBy_())...)

	core.Log(statements)

	if err := db.QueryExecStatements(statements); err != nil {
		core.Log("Error ScyllaDB: ", err)
		return req.MakeErr("Error al registrar el cuadre:", err)
	}

	return req.MakeResponse(&record)
}
