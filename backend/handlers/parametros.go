package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
)

func GetParametros(req *core.HandlerArgs) core.HandlerResponse {
	grupoID := req.GetQueryInt("grupo")

	if grupoID == 0 {
		return req.MakeErr("No se envió el ID del grupo.")
	}

	records := []s.Parametros{}
	q := db.Query(&records)
	err := q.Exclude(q.UpdatedBy).Exec()
	if err != nil {
		return req.MakeErr("Error al obtener los parámetros.", err)
	}

	return core.MakeResponse(req, &records)
}

func PostParametros(req *core.HandlerArgs) core.HandlerResponse {

	records := []s.Parametros{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	groupIDs := core.SliceSet[int32]{}
	for _, e := range records {
		groupIDs.AddIf(e.Grupo)
	}

	if len(groupIDs.Values) != 1 {
		return req.MakeErr("Debe enviar al menos 1 grupo, no mayor a 1 grupo")
	}

	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al insertar los registros", err)
	}

	return req.MakeResponse(records)
}
