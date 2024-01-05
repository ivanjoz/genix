package handlers

import (
	"app/aws"
	"app/core"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"
)

func MakeEmpresaTable() aws.DynamoTableRecords[s.Empresa] {
	return aws.DynamoTableRecords[s.Empresa]{
		TableName:      core.Env.DYNAMO_TABLE,
		PK:             "empr",
		UseCompression: true,
		GetIndexKeys: func(e s.Empresa, idx uint8) string {
			switch idx {
			case 0: // SK (Sort Key)
				return core.Concatn(e.ID)
			case 1: // ix1
				return core.Concatn(e.RUC)
			case 2: // ix2
				return core.Concatn(e.Email)
			case 3: // ix3
				return core.Concatn(e.Updated)
			}
			return ""
		},
	}
}

func PostEmpresa(req *core.HandlerArgs) core.HandlerResponse {

	body := s.Empresa{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Email) < 4 || len(body.Nombre) < 5 {
		return req.MakeErr("Faltan parámetros para la empresa a crear/actualizar.")
	}

	// Revisa si hay que crearla
	if body.ID == 0 {
		counter, err := aws.GetDynamoCounter("empresa")
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
	}

	dynamoTable := MakeEmpresaTable()
	err = dynamoTable.PutItem(&body, 2)

	if err != nil {
		return req.MakeErr("Error guardar la empresa.", err)
	}

	return req.MakeResponse(body)
}

func GetEmpresas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	query := aws.DynamoQueryParam{Index: "sk", GreaterThan: "0"}
	if updated > 0 {
		query.GreaterThan = fmt.Sprintf("%v", updated)
	}

	dynamoTable := MakeEmpresaTable()
	records, err := dynamoTable.QueryBatch([]aws.DynamoQueryParam{query})
	if err != nil {
		panic(err)
	}

	core.Log("Empresas obtenidas:: ", len(records))

	return core.MakeResponse(req, &records)
}

// USUARIOS
func MakeUsuarioTable(empresaID int32) aws.DynamoTableRecords[s.Usuario] {
	return aws.DynamoTableRecords[s.Usuario]{
		TableName:      core.Env.DYNAMO_TABLE,
		PK:             core.Concatn("user", empresaID),
		UseCompression: true,
		GetIndexKeys: func(e s.Usuario, idx uint8) string {
			switch idx {
			case 0: // SK (Sort Key)
				return core.Concatn(e.ID)
			case 1: // ix1
				return core.Concatn(e.Usuario)
			case 2: // ix2
				return core.Concatn(e.Email)
			case 3: // ix3
				return core.Concatn(e.Status, e.Updated)
			}
			return ""
		},
	}
}

func GetUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	query := aws.DynamoQueryParam{Index: "ix3", GreaterThan: "0"}
	query.GreaterThan = fmt.Sprintf("1_%v", updated)

	dynamoTable := MakeUsuarioTable(req.Usuario.EmpresaID)
	records, err := dynamoTable.QueryBatch([]aws.DynamoQueryParam{query})

	if err != nil {
		panic(err)
	}

	core.Log("Usuarios obtenidos:: ", len(records))

	return core.MakeResponse(req, &records)
}

func PostUsuarios(req *core.HandlerArgs) core.HandlerResponse {
	body := s.Usuario{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if body.ID != 1 && len(body.PerfilesIDs) == 0 {
		return req.MakeErr("El usuario debe tener al menos 1 permiso")
	}
	if len(body.Usuario) < 5 || len(body.Nombres) < 5 {
		return req.MakeErr("El usuario nombre debe tener al menos 5 caracteres")
	}
	if body.ID == 0 {
		if len(body.Password) < 6 {
			return req.MakeErr("El password debe tener al menos de 6 caracteres")
		}
	}
	if body.ID == 1 {
		body.Usuario = "admin"
	}

	dynamoTable := MakeUsuarioTable(req.Usuario.EmpresaID)

	if body.ID == 0 {
		pk := core.Concatn("usuario", req.Usuario.EmpresaID)
		counter, err := aws.GetDynamoCounter(pk, 2)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
	} else {
		usuario, err := dynamoTable.GetItem(core.Concatn(body.ID))
		if err != nil || usuario == nil {
			return req.MakeErr("No se encontró el usuario a actualizar")
		}
		body.PasswordHash = usuario.PasswordHash
	}

	if len(body.Password) >= 6 {
		passwordConcat := core.Env.SECRET_PHRASE + body.Password
		body.PasswordHash = core.FnvHashString64(passwordConcat, -1, 20)
	}

	body.Password = ""
	body.Updated = time.Now().Unix()
	err = dynamoTable.PutItem(&body, 1)

	if err != nil {
		return req.MakeErr("Error al actualizar el usuario: " + err.Error())
	}

	return req.MakeResponse(body)
}
