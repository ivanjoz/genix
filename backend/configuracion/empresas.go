package configuracion

import (
	"app/cloud"
	"app/configuracion/types"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
)

func PostEmpresa(req *core.HandlerArgs) core.HandlerResponse {
	body := types.Empresa{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Email) < 4 || len(body.Nombre) < 5 {
		return req.MakeErr("Faltan parámetros para la empresa a crear/actualizar.")
	}

	body.Updated = core.SUnixTime()
	empresasToSave := []types.Empresa{body}
	if err = db.Insert(&empresasToSave); err != nil {
		return req.MakeErr("Error guardar la empresa en ScyllaDB.", err)
	}

	body = empresasToSave[0]
	if err = cloud.Insert([]types.Empresa{body}); err != nil {
		return req.MakeErr("Error guardar la empresa en cloud.", err)
	}

	return req.MakeResponse(body)
}

func GetEmpresas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	if updated < 0 {
		updated = 0
	}

	records := []types.Empresa{}
	if err := cloud.Select(&records).Where("updated").GreaterEqual(updated).Exec(); err != nil {
		return req.MakeErr("Error al obtener las empresas.", err)
	}

	core.Log("Empresas obtenidas:: ", len(records))

	return core.MakeResponse(req, &records)
}

func GetEmpresaParametros(req *core.HandlerArgs) core.HandlerResponse {
	if req.Usuario.ID != 1 {
		return req.MakeErr("No está autorizado para realizar esta solicitud.")
	}

	record, err := cloud.GetByID(types.Empresa{ID: req.Usuario.EmpresaID})
	if err != nil {
		return req.MakeErr("Error al obtener la empresa.", err)
	}
	if record == nil {
		return req.MakeErr("No se encontró la empresa solicitada.")
	}

	records := []types.Empresa{*record}
	return core.MakeResponse(req, &records)
}

func PostEmpresaParametros(req *core.HandlerArgs) core.HandlerResponse {
	record := types.Empresa{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(record.Nombre) == 0 || len(record.RUC) == 0 ||
		len(record.RazonSocial) == 0 || len(record.Email) == 0 {
		return req.MakeErr("Falta alguno de los siguiente parámetros: Nombre, Razon-Social, RUC, Email.")
	}

	if req.Usuario.ID != 1 {
		return req.MakeErr("No está autorizado para realizar esta solicitud.")
	}

	record.ID = req.Usuario.EmpresaID
	if len(record.FormApiKey) == 0 {
		record.FormApiKey = core.MakeRandomBase36String(18)
	}

	record.Updated = core.SUnixTime()
	empresasToSave := []types.Empresa{record}
	if err = db.Insert(&empresasToSave); err != nil {
		return req.MakeErr("Error al guardar el registro de la empresa en ScyllaDB:", err)
	}

	record = empresasToSave[0]
	if err = cloud.Insert([]types.Empresa{record}); err != nil {
		return req.MakeErr("Error al guardar el registro de la empresa en cloud:", err)
	}

	// Save the public company file used by ecommerce/public clients.
	empresaPublic := types.EmpresaPub{
		ID:            record.ID,
		Nombre:        record.Nombre,
		CulqiLlave:    record.CulquiConfig.LlavePubDev,
		CulqiRsaKey:   record.CulquiConfig.RsaKey,
		CulqiRsaKeyID: record.CulquiConfig.RsaKeyID,
	}

	empresaPublicBytes, _ := json.Marshal(empresaPublic)
	core.Log("Guardando::", string(empresaPublicBytes))

	cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "empresas",
		FileContent: empresaPublicBytes,
		Name:        fmt.Sprintf("e-%v.json", req.Usuario.EmpresaID),
	})

	core.Print(empresaPublic)
	return core.MakeResponse(req, &record)
}
