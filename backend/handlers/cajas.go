package handlers

import (
	"app/core"
	s "app/types"
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

	body := s.Caja{}
	if body.ID == 0 || len(body.Nombre) == 0 || body.Tipo == 0 || body.SedeID == 0 {
		return req.MakeErr("Faltan parámetros a enviar: (ID, Nombre, Tipo o SedeID)")
	}

	nowTime := time.Now().Unix()
	body.Updated = nowTime
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

	core.DBUpdateInsert(
		&[]s.Caja{body},
		func(e s.Caja) bool { return e.Created == nowTime },
		[]string{"fecha_cuadre", "monto_cuadre", "monto_current"},
	)

	return req.MakeResponse(body)
}
