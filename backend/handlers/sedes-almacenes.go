package handlers

import (
	"app/core"
	s "app/types"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func GetSedesAlmacenes(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")

	almacenes := []s.Almacen{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := core.DBSelect(&almacenes).
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		if updated > 0 {
			query = query.Where("updated").GreatEq(updated)
		} else {
			query = query.Where("status").Equals(1)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los almacenes: %v", err)
		}
		return err
	})

	sedes := []s.Sede{}
	errGroup.Go(func() error {
		query := core.DBSelect(&sedes).
			Where("empresa_id").Equals(req.Usuario.EmpresaID)

		if updated > 0 {
			query = query.Where("updated").GreatEq(updated)
		} else {
			query = query.Where("status").Equals(1)
		}
		err := query.Exec()
		if err != nil {
			err = fmt.Errorf("error al obtener los almacenes: %v", err)
		}
		return err
	})

	err := errGroup.Wait()
	if err != nil {
		return req.MakeErr(err)
	}

	// Mapea las Sedes con las ciudades
	ubigeosSlice := core.SliceInclude[string]{}

	for _, e := range sedes {
		if len(e.CiudadID) == 6 {
			ubigeosSlice.Add(e.CiudadID)
			ubigeosSlice.Add(e.CiudadID[:4])
			ubigeosSlice.Add(e.CiudadID[:2])
		}
	}

	paisCiudades := []s.PaisCiudad{}
	ubigeosIDs := []any{}
	for _, e := range ubigeosSlice.Values {
		ubigeosIDs = append(ubigeosIDs, any(e))
	}

	if len(ubigeosIDs) > 0 {
		err := core.DBSelect(&paisCiudades).
			Where("pais_id").Equals(604).Where("ciudad_id").IN(ubigeosIDs...).Exec()
		if err != nil {
			return req.MakeErr("Error al obtener las ciudades:", err)
		}
		paisCiudadesMap := core.SliceToMapK(paisCiudades,
			func(e s.PaisCiudad) string { return e.CiudadID })

		for _, pc := range paisCiudadesMap {
			if pc.Jerarquia != 3 {
				continue
			}
			provincia := paisCiudadesMap[pc.PadreID]
			if provincia != nil {
				pc.Provincia = provincia
				departamento := paisCiudadesMap[provincia.PadreID]
				if departamento != nil {
					pc.Departamento = departamento
				}
			}
		}

		for i := range sedes {
			sede := &sedes[i]
			if pc, ok := paisCiudadesMap[sede.CiudadID]; ok {
				if pc.Departamento != nil {
					sede.Ciudad = core.Concat("|", pc.Nombre, pc.Provincia.Nombre, pc.Departamento.Nombre)
				}
			}
		}
	}

	response := map[string]any{
		"Sedes":     sedes,
		"Almacenes": almacenes,
	}

	return core.MakeResponse(req, &response)
}

func PostSedes(req *core.HandlerArgs) core.HandlerResponse {

	body := s.Sede{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Nombre) < 4 || len(body.Descripcion) < 4 {
		return req.MakeErr("Faltan propiedades de la sede")
	}

	if body.ID < 1 {
		key := core.Concatn("sedes", req.Usuario.EmpresaID)
		counter, err := core.GetCounter(key, 1)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
	}

	body.EmpresaID = req.Usuario.EmpresaID
	body.Updated = time.Now().Unix()
	body.Created = time.Now().Unix()
	body.CreatedBy = req.Usuario.ID

	err = core.DBInsert(&[]s.Sede{body})
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar la sede: " + err.Error())
	}

	return req.MakeResponse(body)
}

func GetPaisCiudades(req *core.HandlerArgs) core.HandlerResponse {
	paisID := req.GetQueryInt("pais-id")
	updated := req.GetQueryInt64("upd")

	registros := []s.PaisCiudad{}

	query := core.DBSelect(&registros).
		Where("pais_id").Equals(paisID)

	if updated > 0 {
		query = query.Where("updated").GreatEq(updated)
	}

	err := query.Exec()
	if err != nil {
		err = fmt.Errorf("error al obtener los paises - ciudades: %v", err)
		core.Print(err)
		return req.MakeErr(err)
	}

	core.Log("registros obtenidos:: ", len(registros))

	return core.MakeResponse(req, &registros)
}

func PostAlmacen(req *core.HandlerArgs) core.HandlerResponse {

	body := s.Almacen{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Nombre) < 4 || body.SedeID == 0 {
		return req.MakeErr("Faltan propiedades del almacén")
	}

	if body.ID < 1 {
		key := core.Concatn("almacenes", req.Usuario.EmpresaID)
		counter, err := core.GetCounter(key, 1)
		if err != nil {
			return req.MakeErr("Error al obtener el counter.", counter)
		}
		body.ID = int32(counter)
	}

	for _, e := range body.Layout {
		if e.ID == 0 || len(e.Name) == 0 {
			return req.MakeErr("Hay un layout mal creado.")
		}
	}

	body.EmpresaID = req.Usuario.EmpresaID
	body.Updated = time.Now().Unix()
	body.Created = time.Now().Unix()
	body.CreatedBy = req.Usuario.ID

	err = core.DBInsert(&[]s.Almacen{body})
	if err != nil {
		return req.MakeErr("Error al actualizar / insertar el almacén: " + err.Error())
	}

	return req.MakeResponse(body)
}
