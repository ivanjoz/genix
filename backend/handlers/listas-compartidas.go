package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetListasCompartidas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	listasIDs := req.GetQueryIntSlice("ids")

	if len(listasIDs) == 0 {
		return req.MakeErr("No se enviaron los ids de las listas a consultar.")
	}

	listasRegistrosMap := map[int32]*[]s.ListaCompartidaRegistro{}
	for _, listaID := range listasIDs {
		listasRegistrosMap[listaID] = &[]s.ListaCompartidaRegistro{}
	}
	errGroup := errgroup.Group{}

	for _, listaID := range listasIDs {
		errGroup.Go(func() error {
			query := db.Query(listasRegistrosMap[listaID])
			query.Select().
				EmpresaID.Equals(req.Usuario.EmpresaID).
				ListaID.Equals(listaID)
			if updated > 0 {
				query.Updated.GreaterThan(updated)
			} else {
				query.Status.Equals(1)
			}
			return query.Exec()
		})
	}

	err := errGroup.Wait()
	if err != nil {
		return req.MakeErr(err)
	}

	listasRegistros := []s.ListaCompartidaRegistro{}
	for _, registros := range listasRegistrosMap {
		listasRegistros = append(listasRegistros, *registros...)
	}

	core.Log("Listas compartidas registros obtenidos::", len(listasRegistros))
	/*
		type Result struct {
			Registros []s.ListaCompartidaRegistro `json:"registros"`
		}
	*/
	return core.MakeResponse(req, &listasRegistros)
}

func GetListasCompartidas2(req *core.HandlerArgs) core.HandlerResponse {
	listasIDs := req.GetQueryIntSlice("ids")

	if len(listasIDs) == 0 {
		return req.MakeErr("No se enviaron los ids de las listas a consultar.")
	}

	listaRegistrosMap := map[int32]*[]s.ListaCompartidaRegistro{}
	for _, listaID := range listasIDs {
		listaRegistrosMap[listaID] = &[]s.ListaCompartidaRegistro{}
	}
	eg := errgroup.Group{}

	for _, listaID := range listasIDs {
		updated := req.GetQueryInt64(fmt.Sprintf("id_%v", listaID))

		eg.Go(func() error {
			query := db.Query(listaRegistrosMap[listaID])
			query.Select().
				EmpresaID.Equals(req.Usuario.EmpresaID).
				ListaID.Equals(listaID)
			if updated > 0 {
				query.Updated.GreaterThan(updated)
			} else {
				query.Status.Equals(1)
			}
			return query.Exec()
		})
	}

	if err := eg.Wait(); err != nil {
		return req.MakeErr(err)
	}

	response := map[string]*[]s.ListaCompartidaRegistro{}
	for id, registros := range listaRegistrosMap {
		response[fmt.Sprintf("id_%v", id)] = registros

		core.Log("Listas Compartidas Registros::", id, "|", len(*registros))
	}

	return core.MakeResponse(req, &response)
}

func PostListasCompartidas(req *core.HandlerArgs) core.HandlerResponse {

	records := []s.ListaCompartidaRegistro{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	createCounter := 0
	for _, e := range records {
		if e.ID <= 0 {
			createCounter++
		}
		if len(e.Nombre) < 4 || e.ListaID == 0 {
			return req.MakeErr("Faltan propiedades de en uno de los registros.")
		}
	}

	var counter int64
	if createCounter > 0 {
		counter, err = records[0].GetCounter(createCounter, req.Usuario.EmpresaID)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
	}

	nowTime := time.Now().Unix()
	newIDs := []s.NewIDToID{}

	for i := range records {
		e := &records[i]
		if e.ID <= 0 {
			id := int32(counter)
			newIDs = append(newIDs, s.NewIDToID{NewID: id, TempID: e.ID})
			e.ID = id
			e.Updated = nowTime
			e.UpdatedBy = req.Usuario.ID
			e.Status = 1
			counter++
		} else {
			e.UpdatedBy = req.Usuario.ID
		}
		e.EmpresaID = req.Usuario.EmpresaID
		e.Updated = nowTime
	}

	core.Print(records)

	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al actualizar / insertar el registro de lista compartida: " + err.Error())
	}

	return req.MakeResponse(newIDs)
}
