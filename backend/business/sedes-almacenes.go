package business

import (
	"app/core"
	"app/db"
	businessTypes "app/business/types"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"
)

func GetSedesAlmacenes(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt("upd")

	almacenes := []businessTypes.Warehouse{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&almacenes)
		query.Select().CompanyID.Equals(req.User.CompanyID)

		if updated > 0 {
			query.Updated.GreaterThan(updated)
		} else {
			query.Status.Equals(1)
		}

		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al obtener los almacenes: %v", err)
		}
		return nil
	})

	sedes := []businessTypes.Site{}
	errGroup.Go(func() error {
		query := db.Query(&sedes)
		query.Select().CompanyID.Equals(req.User.CompanyID)

		if updated > 0 {
			query.Updated.GreaterThan(updated)
		} else {
			query.Status.Equals(1)
		}

		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al obtener los sedes: %v", err)
		}
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr(err)
	}

	// Mapea las Sedes con las ciudades
	ubigeosSlice := core.SliceSet[string]{}

	for _, e := range sedes {
		if len(e.CiudadID) == 6 {
			ubigeosSlice.Add(e.CiudadID)
			ubigeosSlice.Add(e.CiudadID[:4])
			ubigeosSlice.Add(e.CiudadID[:2])
		}
	}

	paisCiudades := []businessTypes.CityLocation{}

	if !ubigeosSlice.IsEmpty() {
		// Note: CityLocation still uses old db ORM - this will need to be migrated separately
		query := db.Query(&paisCiudades)
		query.Select().
			PaisID.Equals(604).
			CiudadID.In(ubigeosSlice.Values...)

		err := query.Exec()
		if err != nil {
			return req.MakeErr("Error al obtener las ciudades:", err)
		}

		paisCiudadesMap := core.SliceToMapK(paisCiudades,
			func(e businessTypes.CityLocation) string { return e.CiudadID })

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
			site := &sedes[i]
			if pc, ok := paisCiudadesMap[site.CiudadID]; ok {
				if pc.Departamento != nil {
					site.CiudadID = core.Concat("|", pc.Nombre, pc.Provincia.Nombre, pc.Departamento.Nombre)
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

	body := businessTypes.Site{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Nombre) < 4 || len(body.Descripcion) < 4 {
		return req.MakeErr("Faltan propiedades de la site")
	}

	// Autoincrement is handled automatically by the ORM via handlePreInsert
	body.CompanyID = req.User.CompanyID
	body.Updated = core.SUnixTime()
	body.Created = core.SUnixTime()
	body.CreatedBy = req.User.ID

	if err = db.Insert(&[]businessTypes.Site{body}); err != nil {
		return req.MakeErr("Error al actualizar / insertar la site: " + err.Error())
	}

	return req.MakeResponse(body)
}

func GetPaisCiudades(req *core.HandlerArgs) core.HandlerResponse {
	paisID := req.GetQueryInt("pais-id")
	updated := req.GetQueryInt("upd")

	paisCiudades := []businessTypes.CityLocation{}
	query := db.Query(&paisCiudades)
	query.Select().
		PaisID.Equals(int32(paisID))

	if updated > 0 {
		query.Updated.GreaterEqual(updated)
	}

	query.AllowFilter()
	err := query.Exec()

	if err != nil {
		err := fmt.Errorf("error al obtener los paises - ciudades: %v", err)
		core.Print(err)
		return req.MakeErr(err)
	}

	for i := range  paisCiudades {
		e := &paisCiudades[i]
		e.ID = core.StrToInt(e.CiudadID)
	}

	core.Log("registros obtenidos:: ", len(paisCiudades))

	return core.MakeResponse(req, &paisCiudades)
}

func PostAlmacen(req *core.HandlerArgs) core.HandlerResponse {

	body := businessTypes.Warehouse{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Nombre) < 4 || body.SedeID == 0 {
		return req.MakeErr("Faltan propiedades del almacén")
	}

	// Autoincrement is handled automatically by the ORM via handlePreInsert

	for _, e := range body.Layout {
		if e.ID == 0 || len(e.Name) == 0 {
			return req.MakeErr("Hay un layout mal creado.")
		}
	}

	body.CompanyID = req.User.CompanyID
	body.Updated = core.SUnixTime()
	body.Created = core.SUnixTime()
	body.CreatedBy = req.User.ID

	if err := db.Insert(&[]businessTypes.Warehouse{body}); err != nil {
		return req.MakeErr("Error al actualizar / insertar el almacén: " + err.Error())
	}

	return req.MakeResponse(body)
}
