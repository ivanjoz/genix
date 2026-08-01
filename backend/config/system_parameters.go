package config

import (
	configTypes "app/config/types"
	"app/core"
	"app/db"
	"encoding/json"
)

func GetSystemParameters(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.User.CompanyID
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("upv")

	records := []configTypes.SystemParameters{}
	q := db.Query(&records)
	// No status to filter here, so Delta() constrains nothing but the watermark.
	q.CompanyID.Equals(companyID).Delta(updatedSince)

	if err := q.Exec(); err != nil {
		return req.MakeErr("Error al obtener los parámetros del sistema.", err)
	}

	core.Print(records)

	return core.MakeResponse(req, &records)
}

func PostSystemParameters(req *core.HandlerArgs) core.HandlerResponse {
	companyID := req.User.CompanyID

	records := []configTypes.SystemParameters{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	if len(records) == 0 {
		return req.MakeErr("No se enviaron registros a guardar.")
	}

	now := core.SUnixTime()
	for i := range records {
		e := &records[i]
		if e.ID == 0 {
			return req.MakeErr("No se envió el parámetro ID.")
		}
		e.CompanyID = companyID
		e.Updated = now
		e.UpdatedBy = req.User.ID
	}

	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al insertar los registros", err)
	}

	return req.MakeResponse(records)
}
