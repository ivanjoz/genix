package config

import (
	"app/cloud"
	"app/config/types"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
)

func PostEmpresa(req *core.HandlerArgs) core.HandlerResponse {
	body := types.Company{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Email) < 4 || len(body.Name) < 5 {
		return req.MakeErr("Faltan parámetros para la company a crear/actualizar.")
	}

	body.Updated = core.SUnixTime()
	empresasToSave := []types.Company{body}
	if err = db.Insert(&empresasToSave); err != nil {
		return req.MakeErr("Error guardar la company en ScyllaDB.", err)
	}

	body = empresasToSave[0]
	if err = cloud.Insert([]types.Company{body}); err != nil {
		return req.MakeErr("Error guardar la company en cloud.", err)
	}

	return req.MakeResponse(body)
}

func GetEmpresas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	if updated < 0 {
		updated = 0
	}

	records := []types.Company{}
	if err := cloud.Select(&records).Where("updated").GreaterEqual(updated).Exec(); err != nil {
		return req.MakeErr("Error al obtener las empresas.", err)
	}

	core.Log("Empresas obtenidas:: ", len(records))

	return core.MakeResponse(req, &records)
}

func GetEmpresaParametros(req *core.HandlerArgs) core.HandlerResponse {
	if req.User.ID != 1 {
		return req.MakeErr("No está autorizado para realizar esta solicitud.")
	}

	record, err := cloud.GetByID(types.Company{ID: req.User.CompanyID})
	if err != nil {
		return req.MakeErr("Error al obtener la company.", err)
	}
	if record == nil {
		return req.MakeErr("No se encontró la company solicitada.")
	}

	records := []types.Company{*record}
	return core.MakeResponse(req, &records)
}

func PostEmpresaParametros(req *core.HandlerArgs) core.HandlerResponse {
	record := types.Company{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(record.Name) == 0 || len(record.RUC) == 0 ||
		len(record.LegalName) == 0 || len(record.Email) == 0 {
		return req.MakeErr("Falta alguno de los siguiente parámetros: Nombre, Razon-Social, RUC, Email.")
	}

	if req.User.ID != 1 {
		return req.MakeErr("No está autorizado para realizar esta solicitud.")
	}

	record.ID = req.User.CompanyID
	if len(record.FormApiKey) == 0 {
		record.FormApiKey = core.MakeRandomBase36String(18)
	}

	record.Updated = core.SUnixTime()
	empresasToSave := []types.Company{record}
	if err = db.Insert(&empresasToSave); err != nil {
		return req.MakeErr("Error al guardar el registro de la company en ScyllaDB:", err)
	}

	record = empresasToSave[0]
	if err = cloud.Insert([]types.Company{record}); err != nil {
		return req.MakeErr("Error al guardar el registro de la company en cloud:", err)
	}

	// Save the public company file used by ecommerce/public clients.
	empresaPublic := types.CompanyPub{
		ID:            record.ID,
		Name:          record.Name,
		CulqiLlave:    record.CulqiConfig.PubKeyDev,
		CulqiRsaKey:   record.CulqiConfig.RsaKey,
		CulqiRsaKeyID: record.CulqiConfig.RsaKeyID,
	}

	empresaPublicBytes, _ := json.Marshal(empresaPublic)
	core.Log("Guardando::", string(empresaPublicBytes))

	cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "empresas",
		FileContent: empresaPublicBytes,
		Name:        fmt.Sprintf("e-%v.json", req.User.CompanyID),
	})

	core.Print(empresaPublic)
	return core.MakeResponse(req, &record)
}
