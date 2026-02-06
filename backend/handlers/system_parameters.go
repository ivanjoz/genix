package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"time"
)

func GetSystemParameters(req *core.HandlerArgs) core.HandlerResponse {
	empresaID := req.Usuario.EmpresaID
	updated := core.Coalesce(req.GetQueryInt64("upd"), req.GetQueryInt64("updated"))

	records := []s.SystemParameters{}
	q := db.Query(&records)
	q.EmpresaID.Equals(empresaID)

	if updated > 0 {
		q.Updated.GreaterThan(updated)
	}

	if 	err := q.Exec(); err != nil {
		return req.MakeErr("Error al obtener los parámetros del sistema.", err)
	}
	
	core.Print(records)

	return core.MakeResponse(req, &records)
}

func PostSystemParameters(req *core.HandlerArgs) core.HandlerResponse {
	empresaID := req.Usuario.EmpresaID

	records := []s.SystemParameters{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}
	
	if len(records) == 0 {
		return req.MakeErr("No se enviaron registros a guardar.")
	}

	now := time.Now().Unix()
	for i := range records {
		e := &records[i]
		if e.ID == 0 {
			return req.MakeErr("No se envió el parámetro ID.")
		}
		e.EmpresaID = empresaID
		e.Updated = now
		e.UpdatedBy = req.Usuario.ID
	}

	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al insertar los registros", err)
	}

	return req.MakeResponse(records)
}
