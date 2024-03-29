package handlers

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"time"
)

func GetCajas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

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

	nowTime := time.Now().Unix()
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

	lastRegistros := req.GetQueryInt("last-registros")
	lastRegistros = core.If(lastRegistros > 1000, 1000, lastRegistros)

	movimientos := []s.CajaMovimiento{}
	query := core.DBSelect(&movimientos).
		Where("empresa_id").Equals(req.Usuario.EmpresaID)

	baseRecord := s.CajaMovimiento{CajaID: cajaID}
	if lastRegistros > 0 {
		baseRecord.SetID(9_999_999_999_999)
		query = query.Where("id").LessEq(baseRecord.ID)
		baseRecord.SetID(0)
		query = query.Where("id").GreatEq(baseRecord.ID)
		query.Limit(lastRegistros)
	} else {

	}

	err := query.OrderDescending().Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los movimientos de las cajas:", err)
	}

	usuariosIDs := core.SliceInclude[int32]{}
	for _, e := range movimientos {
		usuariosIDs.Add(e.CreatedBy)
	}

	usuarios := []s.Usuario{}

	if !usuariosIDs.IsEmpty() {
		err := core.DBSelect(&usuarios).
			Where("empresa_id").Equals(req.Usuario.EmpresaID).
			Where("id").In(usuariosIDs.ToAny()).Exec()

		if err != nil {
			return req.MakeErr("Error al obtener los usuarios.", err)
		}
	}

	response := map[string]any{
		"movimientos": movimientos,
		"usuarios":    usuarios,
	}

	return core.MakeResponse(req, &response)
}

func PostCajaCuadre(req *core.HandlerArgs) core.HandlerResponse {
	core.Env.LOGS_DEBUG = true

	nowTimeMill := time.Now().UnixMilli()
	nowTime := nowTimeMill / 1000

	record := s.CajaCuadre{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if record.CajaID == 0 {
		return req.MakeErr("Faltan Parámetros: [Caja-ID]")
	}

	caja_ := []s.Caja{}
	err = core.DBSelect(&caja_).
		Where("empresa_id").Equals(req.Usuario.EmpresaID).
		Where("id").Equals(record.CajaID).Exec()

	if err != nil {
		return req.MakeErr("Error al obtener información de la caja.", err)
	}
	caja := caja_[0]

	if record.SaldoSistema != caja.SaldoCurrent {
		re := map[string]any{"NeedUpdateSaldo": caja_[0].SaldoCurrent}
		return req.MakeResponse(&re)
	}

	// Guarda el cuadre
	record.EmpresaID = req.Usuario.EmpresaID
	record.SetID(nowTimeMill)
	record.Created = nowTime
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
		EmpresaID:  req.Usuario.EmpresaID,
		CajaID:     record.CajaID,
		Tipo:       2, // Cuadre de caja
		SaldoFinal: record.SaldoReal,
		Monto:      record.SaldoDiferencia,
		Created:    nowTime,
		CreatedBy:  req.Usuario.ID,
	}
	movimiento.SetID(nowTimeMill)

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
