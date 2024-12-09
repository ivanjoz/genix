package handlers

import (
	"app/core"
	"app/db"
	s "app/types"
	"encoding/json"
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
		type r = s.ListaCompartidaRegistro
		errGroup.Go(func() error {
			return db.SelectRef(listasRegistrosMap[listaID], func(q *db.Query[r], col r) {
				q.Where(col.EmpresaID_().Equals(req.Usuario.EmpresaID))
				q.Where(col.ListaID_().Equals(listaID))
				if updated > 0 {
					q.Where(col.Updated_().GreaterThan(updated))
				} else {
					q.Where(col.Status_().Equals(1))
				}
			})
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
		counter, err = core.GetCounter("lista_registros", createCounter, req.Usuario.EmpresaID)
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

	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al actualizar / insertar el registro de lista compartida: " + err.Error())
	}

	return req.MakeResponse(newIDs)
}
