package handlers

import (
	"app/core"
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
		query := core.DBSelect(listasRegistrosMap[listaID], "empresa_id").
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		/*
			base := s.ListaCompartidaRegistro{ListaID: listaID, Updated: updated, Status: 1}
			base.SelfParse()
		*/

		if updated > 0 {
			query = query.Where("lista_id", "updated").GreatEq(listaID, updated)
		} else {
			query = query.Where("lista_id", "status").Equals(listaID, int8(1))
		}

		errGroup.Go(func() error {
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

	for i := range records {
		e := &records[i]
		if e.ID <= 0 {
			e.ID = int32(counter)
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

	err = core.DBInsert(&records)
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar el registro de lista compartida: " + err.Error())
	}

	return req.MakeResponse(records)
}
