package business

import (
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
	"encoding/json"

	"golang.org/x/sync/errgroup"
)

// initialDataBody carries the "Datos Iniciales" bootstrap: the first Site, Warehouse and CashBank
// of a company. SiteID > 0 reuses an existing site instead of creating one, which happens when the
// company already has a site but is missing the warehouse or the cash bank.
type initialDataBody struct {
	SiteID        int32
	SiteName      string
	SiteAddress   string
	CityID        int32
	WarehouseName string
	CashBankName  string
}

// PostInitialData creates the minimum records a company needs to operate. It lives in a single
// handler because Warehouse and CashBank both need the Site's autoincremented ID, so the insert
// order has to be decided server-side.
func PostInitialData(req *core.HandlerArgs) core.HandlerResponse {

	body := initialDataBody{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.WarehouseName) < 4 {
		return req.MakeErr("El nombre del almacén debe poseer al menos 4 caracteres.")
	}
	if len(body.CashBankName) < 4 {
		return req.MakeErr("El nombre de la caja debe poseer al menos 4 caracteres.")
	}
	// The site is only validated when it has to be created: an existing one is checked against the
	// company's rows further down instead.
	if body.SiteID == 0 {
		if len(body.SiteName) < 4 {
			return req.MakeErr("El nombre de la sede debe poseer al menos 4 caracteres.")
		}
		if len(body.SiteAddress) < 4 {
			return req.MakeErr("La dirección de la sede debe poseer al menos 4 caracteres.")
		}
		if body.CityID <= 0 {
			return req.MakeErr("Debe seleccionar una ciudad válida para la sede.")
		}
	}

	// The endpoint is reachable by URL at any time, so it re-reads what already exists and only
	// inserts what is missing. Delta(0, 1) is the first-sync form: it pins Status to 1.
	sites := []businessTypes.Site{}
	warehouses := []businessTypes.Warehouse{}
	cashBanks := []financeTypes.CashBank{}

	errGroup := errgroup.Group{}

	errGroup.Go(func() error {
		query := db.Query(&sites)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(0, 1)
		return query.Exec()
	})
	errGroup.Go(func() error {
		query := db.Query(&warehouses)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(0, 1)
		return query.Exec()
	})
	errGroup.Go(func() error {
		query := db.Query(&cashBanks)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(0, 1)
		return query.Exec()
	})

	if err := errGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los datos iniciales de la empresa:", err)
	}

	core.Log("PostInitialData:: sedes:", len(sites), "almacenes:", len(warehouses), "cajas:", len(cashBanks))

	// Nothing to bootstrap: the company can already operate.
	if len(warehouses) > 0 && len(cashBanks) > 0 {
		return req.MakeErr("La empresa ya posee un almacén y una caja registrados.")
	}

	nowTime := core.SUnixTime()
	siteID := body.SiteID

	if siteID > 0 {
		// Never trust the client with a foreign key: the site must belong to this company.
		siteExists := false
		for _, site := range sites {
			if site.ID == siteID {
				siteExists = true
				break
			}
		}
		if !siteExists {
			return req.MakeErr("La sede seleccionada no existe en la empresa.")
		}
	} else {
		newSites := []businessTypes.Site{{
			CompanyID: req.User.CompanyID,
			Name:      body.SiteName,
			Address:   body.SiteAddress,
			CityID:    body.CityID,
			Status:    1,
			Updated:   nowTime,
			UpdatedBy: req.User.ID,
			Created:   nowTime,
			CreatedBy: req.User.ID,
		}}
		// The ORM fills the autoincremented ID on the record itself during handlePreInsert.
		if err := db.Insert(&newSites); err != nil {
			return req.MakeErr("Error al insertar la sede:", err)
		}
		siteID = newSites[0].ID
		core.Log("PostInitialData:: sede creada con ID:", siteID)
	}

	warehouseID := int32(0)
	if len(warehouses) == 0 {
		newWarehouses := []businessTypes.Warehouse{{
			CompanyID: req.User.CompanyID,
			SiteID:    siteID,
			Name:      body.WarehouseName,
			Status:    1,
			Updated:   nowTime,
			UpdatedBy: req.User.ID,
			Created:   nowTime,
			CreatedBy: req.User.ID,
		}}
		if err := db.Insert(&newWarehouses); err != nil {
			return req.MakeErr("Error al insertar el almacén:", err)
		}
		warehouseID = newWarehouses[0].ID
		core.Log("PostInitialData:: almacén creado con ID:", warehouseID)
	}

	cashBankID := int32(0)
	if len(cashBanks) == 0 {
		newCashBanks := []financeTypes.CashBank{{
			CompanyID: req.User.CompanyID,
			SiteID:    siteID,
			Name:      body.CashBankName,
			// Type 1 = "Caja" and CurrencyType 1 = PEN, the defaults of the cash-banks page.
			Type:         1,
			CurrencyType: 1,
			Status:       1,
			Updated:      nowTime,
			UpdatedBy:    req.User.ID,
			Created:      nowTime,
			CreatedBy:    req.User.ID,
		}}
		if err := db.Insert(&newCashBanks); err != nil {
			return req.MakeErr("Error al insertar la caja:", err)
		}
		cashBankID = newCashBanks[0].ID
		core.Log("PostInitialData:: caja creada con ID:", cashBankID)
	}

	response := map[string]any{
		"SiteID":      siteID,
		"WarehouseID": warehouseID,
		"CashBankID":  cashBankID,
	}

	return req.MakeResponse(response)
}
