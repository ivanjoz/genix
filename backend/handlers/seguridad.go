package handlers

import (
	"app/cloud"
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"fmt"
)

func GetPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	records := []s.Perfil{}

	var err error
	if updated > 0 {
		companyUpdated := fmt.Sprintf("%d_%020d", req.Usuario.EmpresaID, updated)
		err = cloud.Select(&records).Where("empresa_id").Equals(req.Usuario.EmpresaID).Where("company_updated").GreaterEqual(companyUpdated).Exec()
	} else {
		activeProfiles := fmt.Sprintf("%d_%d_%020d", req.Usuario.EmpresaID, 1, updated)
		err = cloud.Select(&records).Where("empresa_id").Equals(req.Usuario.EmpresaID).Where("company_status_updated").GreaterEqual(activeProfiles).Exec()
	}
	if err != nil {
		return req.MakeErr("Error al obtener perfiles.", err)
	}

	core.Log("Perfiles obtenidos:: ", len(records))
	core.Print(records)

	return core.MakeResponse(req, &records)
}

func PostPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	body := s.Perfil{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	body.EmpresaID = req.Usuario.EmpresaID
	core.Print(body)

	body.Updated = core.SUnixTime()
	body.PrepareCloudSync()
	perfilesToSave := []s.Perfil{body}
	if err = db.Insert(&perfilesToSave); err != nil {
		return req.MakeErr("Error al actualizar el perfil en ScyllaDB: " + err.Error())
	}

	body = perfilesToSave[0]
	body.PrepareCloudSync()
	if err = cloud.Insert([]s.Perfil{body}); err != nil {
		return req.MakeErr("Error al actualizar el perfil en cloud: " + err.Error())
	}

	return req.MakeResponse(body)
}
