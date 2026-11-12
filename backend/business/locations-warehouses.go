package business

import (
	"app/business/types"
	"app/cloud"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"
)

func GetLocationsWarehouses(req *core.HandlerArgs) core.HandlerResponse {
	// Each table has its own delta log, so the two response keys carry independent cursors; the
	// frontend sends one query param per key, named after it.
	almacenes := []types.Warehouse{}
	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&almacenes)
		// A first sync lists the active rows; later syncs read the log, whose deleted entries evict.
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Almacenes"), 1)

		if err := query.Exec(); err != nil {
			return fmt.Errorf("error al obtener los almacenes: %v", err)
		}
		return nil
	})

	sedes := []types.Site{}
	errGroup.Go(func() error {
		query := db.Query(&sedes)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Sedes"), 1)

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

	paisCiudades := []types.CityLocation{}

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
			func(e types.CityLocation) int32 { return e.ID })

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

func PostSite(req *core.HandlerArgs) core.HandlerResponse {

	body := types.Site{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Name) < 4 {
		return req.MakeErr("El nombre debe poseer al menos 3 caracteres.")
	}
	if body.CityID <= 0 {
		return req.MakeErr("Debe seleccionar una ciudad válida para la sede.")
	}

	// Autoincrement is handled automatically by the ORM via handlePreInsert
	body.CompanyID = req.User.CompanyID
	body.Updated = core.SUnixTime()
	body.Created = core.SUnixTime()
	body.CreatedBy = req.User.ID

	records := []types.Site{body}
	if err = db.Upsert(&records); err != nil {
		return req.MakeErr("Error al actualizar / insertar la site: " + err.Error())
	}

	// The sites travel in the cached company config (invoicing reads them, checkout validates a
	// pickup site there) and the pickup ones in the storefront's public file. Republishing on every
	// save is two small writes, cheaper than diffing which fields a pickup site changed.
	cloud.StoreCompanyConfigAsync(req.User.CompanyID)
	if err := cloud.PublishCompanyPub(req.User.CompanyID); err != nil {
		core.Log("PostSite:: no se pudo publicar el archivo público de la empresa", err)
	}

	return req.MakeResponse(records[0])
}

func GetCountryCities(req *core.HandlerArgs) core.HandlerResponse {
	paisID := req.GetQueryInt("pais-id")
	paisCiudades := []types.CityLocation{}
	query := db.Query(&paisCiudades)
	// Keyless delta index: cities have no status to filter.
	query.Select().
		CountryID.Equals(int32(paisID)).
		Delta(req.GetDeltaCursor())
	err := query.Exec()

	if err != nil {
		err := fmt.Errorf("error al obtener los paises - ciudades: %v", err)
		core.Print(err)
		return req.MakeErr(err)
	}

	core.Log("registros obtenidos:: ", len(paisCiudades))

	return core.MakeResponse(req, &paisCiudades)
}

func PostWarehouse(req *core.HandlerArgs) core.HandlerResponse {

	body := types.Warehouse{}
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

	// Opening or closing a warehouse to online sales moves every product's web stock without touching
	// a product or stock row, so the web catalog needs a new generation. A new warehouse holds no
	// stock yet: its first stock rows reach the catalog through the delta.
	isOnlineStateChanged := false
	if body.ID > 0 {
		storedWarehouses := []types.Warehouse{}
		query := db.Query(&storedWarehouses)
		query.Select(query.ID, query.Status, query.ExcludeFromOnlineSales).
			CompanyID.Equals(body.CompanyID).ID.Equals(body.ID)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener el almacén:", err)
		}
		if len(storedWarehouses) == 0 {
			return req.MakeErr("No se encontró el almacén a actualizar.")
		}
		isOnlineStateChanged = storedWarehouses[0].IsOpenToOnlineSales() != body.IsOpenToOnlineSales()
	}

	almacenes := []types.Warehouse{body}
	if err := db.Upsert(&almacenes); err != nil {
		return req.MakeErr("Error al actualizar / insertar el almacén: " + err.Error())
	}
	if isOnlineStateChanged {
		if err := bumpEcommerceCatalogGeneration(body.CompanyID); err != nil {
			return req.MakeErr(err)
		}
	}

	return req.MakeResponse(almacenes[0])
}
