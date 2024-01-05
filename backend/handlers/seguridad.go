package handlers

import (
	"app/aws"
	"app/core"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"
)

/* ACCESOS */
func MakeAccesoTable() aws.DynamoTableRecords[s.SeguridadAcceso] {
	return aws.DynamoTableRecords[s.SeguridadAcceso]{
		TableName:      core.Env.DYNAMO_TABLE,
		PK:             "acceso",
		UseCompression: true,
		GetIndexKeys: func(e s.SeguridadAcceso, idx uint8) string {
			switch idx {
			case 0: // SK (Sort Key)
				return core.Concatn(e.ID)
			case 2: // ix2
				return core.Concatn(e.Status, e.Updated)
			case 3: // ix3
				return core.Concatn(e.Updated)
			}
			return ""
		},
	}
}

func PostAcceso(req *core.HandlerArgs) core.HandlerResponse {

	body := s.SeguridadAcceso{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if body.ID < 1 {
		counter, err := aws.GetDynamoCounter("empresa")
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
	}

	body.Updated = time.Now().Unix()
	accesosTable := MakeAccesoTable()
	err = accesosTable.PutItem(&body, 1)

	if err != nil {
		return req.MakeErr("Error al actualizar el acceso: " + err.Error())
	}

	return req.MakeResponse(body)
}

func GetAccesos(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	query := aws.DynamoQueryParam{Index: "ix3", GreaterThan: "0"}
	query.GreaterThan = fmt.Sprintf("%v", updated)

	dynamoTable := MakeAccesoTable()
	records, err := dynamoTable.QueryBatch([]aws.DynamoQueryParam{query})
	if err != nil {
		panic(err)
	}

	for i := range records {
		e := &records[i]
		if e.Updated == 0 {
			e.Updated = 1
		}
	}

	core.Log("Accesos obtenidos:: ", len(records))

	return core.MakeResponse(req, &records)
}

/* PERFILES */
func MakePerfilTable(empresaID int32) aws.DynamoTableRecords[s.Perfil] {
	return aws.DynamoTableRecords[s.Perfil]{
		TableName:      core.Env.DYNAMO_TABLE,
		PK:             core.Concatn("perf", empresaID),
		UseCompression: true,
		GetIndexKeys: func(e s.Perfil, idx uint8) string {
			switch idx {
			case 0: // SK (Sort Key)
				return core.Concatn(e.ID)
			case 2: // ix2
				return core.Concatn(e.Status, e.Updated)
			case 3: // ix3
				return core.Concatn(e.Updated)
			}
			return ""
		},
	}
}

func GetPerfiles(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	query := aws.DynamoQueryParam{Index: "ix3", GreaterThan: "0"}
	if updated > 0 {
		query.GreaterThan = fmt.Sprintf("%v", updated)
	} else {
		query.Index = "ix2"
		query.GreaterThan = fmt.Sprintf("1_%v", updated)
	}

	dynamoTable := MakePerfilTable(req.Usuario.EmpresaID)
	records, err := dynamoTable.QueryBatch([]aws.DynamoQueryParam{query})
	if err != nil {
		panic(err)
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

	if body.ID < 1 {
		counter, err := aws.GetDynamoCounter(core.Concatn("perfiles", req.Usuario.EmpresaID))
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
	}

	core.Print(body)

	body.Updated = time.Now().Unix()
	accesosTable := MakePerfilTable(req.Usuario.EmpresaID)
	err = accesosTable.PutItem(&body, 1)

	if err != nil {
		return req.MakeErr("Error al actualizar el acceso: " + err.Error())
	}

	return req.MakeResponse(body)
}
