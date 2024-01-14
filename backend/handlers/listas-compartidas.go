package handlers

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetListasCompartidas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	listasRegistros := []s.ListaCompartidaRegistro{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := core.DBSelect(&listasRegistros, "empresa_id").
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		if updated > 0 {
			query = query.Where("updated").GreatEq(updated)
		} else {
			query = query.Where("status").Equals(1)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener las listas compartidas: %v", err)
		}
		return err
	})

	err := errGroup.Wait()
	if err != nil {
		return req.MakeErr(err)
	}

	core.Log("Listas compartidas registros obtenidos::", len(listasRegistros))

	return core.MakeResponse(req, &listasRegistros)
}

func PostListasCompartidas(req *core.HandlerArgs) core.HandlerResponse {

	records := []s.ListaCompartidaRegistro{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	createCounter := 0
	for _, e := range records {
		if e.ID == "" {
			createCounter++
		}
		if len(e.Nombre) < 4 || e.ListaID == 0 {
			return req.MakeErr("Faltan propiedades de en uno de los registros.")
		}
	}

	var counter int64
	if createCounter > 0 {
		key := core.Concatn("lista_registros", req.Usuario.EmpresaID, records[0].ListaID)
		counter, err = core.GetCounter(key, createCounter)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
	}

	nowTime := time.Now().Unix()

	for i := range records {
		e := &records[i]
		if e.ID == "" {
			e.ID = core.Concat(".", e.ListaID, counter)
			e.Created = nowTime
			e.CreatedBy = req.Usuario.ID
			e.Status = 1
			counter++
		} else {
			e.UpdatedBy = req.Usuario.ID
		}
		e.EmpresaID = req.Usuario.EmpresaID
		e.Updated = nowTime
	}

	err = core.DBInsert(&records)
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar el almacén: " + err.Error())
	}

	return req.MakeResponse(records)
}
