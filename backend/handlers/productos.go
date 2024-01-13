package handlers

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetProductos(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	productos := []s.Producto{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := core.DBSelect(&productos, "temp_id", "empresa_id").
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		if updated > 0 {
			query = query.Where("updated").GreatEq(updated)
		} else {
			query = query.Where("status").Equals(1)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los productos: %v", err)
		}
		return err
	})

	err := errGroup.Wait()
	if err != nil {
		return req.MakeErr(err)
	}

	core.Log("productos obtenidos::", len(productos))

	return core.MakeResponse(req, &productos)
}

func PostProductos(req *core.HandlerArgs) core.HandlerResponse {

	records := []s.Producto{}
	err := json.Unmarshal([]byte(*req.Body), &records)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	createCounter := 0

	for _, e := range records {
		if e.ID < 1 {
			createCounter++
			e.TempID = e.ID
		}
		if len(e.Nombre) < 4 {
			return req.MakeErr("Faltan propiedades de en el producto.")
		}
	}

	var counter int64
	if createCounter > 0 {
		key := core.Concatn("productos", req.Usuario.EmpresaID)
		counter, err = core.GetCounter(key, createCounter)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
	}

	nowTime := time.Now().Unix()

	for i := range records {
		e := &records[i]
		if e.ID < 1 {
			e.ID = int32(counter)
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
		return req.MakeErr("Error al actualizar / insertar la sede: " + err.Error())
	}

	return req.MakeResponse(records)
}
