package business

import (
	businessTypes "app/business/types"
	"app/core"
	"app/db"
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

	// Load the selected district plus its province and department from numeric ubigeo IDs.
	ubigeosSlice := core.SliceSet[int32]{}

	for _, e := range sedes {
		if e.CityID >= 10000 {
			ubigeosSlice.Add(e.CityID)
			ubigeosSlice.Add(e.CityID / 100)
			ubigeosSlice.Add(e.CityID / 10000)
		}
	}

	paisCiudades := []businessTypes.CityLocation{}

	if !ubigeosSlice.IsEmpty() {
		query := db.Query(&paisCiudades)
		query.Select().
			CountryID.Equals(604).
			ID.In(ubigeosSlice.Values...)

		err := query.Exec()
		if err != nil {
			return req.MakeErr("Error al obtener las ciudades:", err)
		}

		paisCiudadesMap := core.SliceToMapK(paisCiudades,
			func(e businessTypes.CityLocation) int32 { return e.ID })

		for _, pc := range paisCiudadesMap {
			if pc.Hierarchy != 3 {
				continue
			}
			provincia := paisCiudadesMap[pc.ParentID]
			if provincia != nil {
				pc.Province = provincia
				departamento := paisCiudadesMap[provincia.ParentID]
				if departamento != nil {
					pc.Department = departamento
				}
			}
		}

		for i := range sedes {
			site := &sedes[i]
			if pc, ok := paisCiudadesMap[site.CityID]; ok {
				if pc.Department != nil {
					site.City = core.Concat("|", pc.Name, pc.Province.Name, pc.Department.Name)
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

	if len(body.Name) < 4 || len(body.Description) < 4 {
		return req.MakeErr("Faltan propiedades de la site")
	}
	if body.CityID <= 0 {
		return req.MakeErr("Debe seleccionar una ciudad válida para la sede.")
	}

	// Autoincrement is handled automatically by the ORM via handlePreInsert
	body.CompanyID = req.User.CompanyID
	body.Updated = core.SUnixTime()
	body.Created = core.SUnixTime()
	body.CreatedBy = req.User.ID

	records := []businessTypes.Site{body}
	if err = db.Insert(&records); err != nil {
		return req.MakeErr("Error al actualizar / insertar la site: " + err.Error())
	}

	return req.MakeResponse(records[0])
}

func GetPaisCiudades(req *core.HandlerArgs) core.HandlerResponse {
	paisID := req.GetQueryInt("pais-id")
	updated := req.GetQueryInt("upd")

	paisCiudades := []businessTypes.CityLocation{}
	query := db.Query(&paisCiudades)
	query.Select().
		CountryID.Equals(int32(paisID))

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

	core.Log("registros obtenidos:: ", len(paisCiudades))

	return core.MakeResponse(req, &paisCiudades)
}

func PostAlmacen(req *core.HandlerArgs) core.HandlerResponse {

	body := businessTypes.Warehouse{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Name) < 4 || body.SiteID == 0 {
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

	almacenes := []businessTypes.Warehouse{body}
	if err := db.Insert(&almacenes); err != nil {
		return req.MakeErr("Error al actualizar / insertar el almacén: " + err.Error())
	}

	return req.MakeResponse(almacenes[0])
}
