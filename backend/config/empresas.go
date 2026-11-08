package config

import (
	"app/cloud"
	"app/config/types"
	"app/core"
	"app/db"
	invoicing "app/invoicing/types"
	"encoding/json"
	"fmt"
)

func PostEmpresa(req *core.HandlerArgs) core.HandlerResponse {
	body := types.Company{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Email) < 4 || len(body.Name) < 5 {
		return req.MakeErr("Faltan parámetros para la company a crear/actualizar.")
	}

	// Every company starts able to invoice: the six SUNAT series, two for selling
	// and four for the notes that correct each family.
	if len(body.InvoiceSeries) == 0 {
		body.InvoiceSeries = invoicing.DefaultInvoiceSeries()
	}

	body.Flags, err = core.SanitizeCompanyFlags(body.Flags)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	body.FlagValues, err = core.SanitizeCompanyFlagValues(body.FlagValues)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	body.Updated = core.SUnixTime()
	if body.ID == 0 {
		empresasToSave := []types.Company{body}
		if err = db.Insert(&empresasToSave); err != nil {
			return req.MakeErr("Error guardar la company.", err)
		}
		body = empresasToSave[0]
	} else if body, err = saveCompanyKeepingSeries(body); err != nil {
		return req.MakeErr("Error guardar la company.", err)
	}
	if err = cloud.Insert([]types.Company{body}); err != nil {
		return req.MakeErr("Error guardar la company en cloud.", err)
	}

	cloud.StoreCompanyConfigAsync(body.ID)
	return req.MakeResponse(body)
}

// saveCompanyKeepingSeries writes an existing company whole, except its invoicing series: those
// have their own endpoint (POST.invoice-series), so a company form's Save means what the user sees
// on it, and a payload that omits the series or carries a stale copy cannot wipe one somebody added
// meanwhile. The row is read under its lock, the one the series save takes, so the series kept are
// the current ones.
func saveCompanyKeepingSeries(record types.Company) (types.Company, error) {
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		stored := []types.Company{{ID: record.ID}}
		if err := db.LockByKeys(transaction, &stored); err != nil {
			return err
		}
		if len(stored) == 0 {
			return fmt.Errorf("no se encontró la empresa %d", record.ID)
		}
		record.InvoiceSeries = stored[0].InvoiceSeries
		return db.UpsertIn(transaction, &[]types.Company{record})
	})
	return record, err
}

func GetEmpresas(req *core.HandlerArgs) core.HandlerResponse {
	updated := req.GetQueryInt64("upd")
	if updated < 0 {
		updated = 0
	}

	records, err := getCompaniesUpdatedSince(updated)
	if err != nil {
		return req.MakeErr("Error al obtener las empresas.", err)
	}

	core.Log("Empresas obtenidas:: ", len(records))
	return req.MakeResponse(core.WithDeltaReplace(records))
}

// getCompaniesUpdatedSince keeps global company reads consistent across administrative reports.
func getCompaniesUpdatedSince(updated int64) ([]types.Company, error) {
	records := []types.Company{}
	if cloud.IsDataMirrorEnabled() {
		return records, cloud.Select(&records).Where("updated").GreaterEqual(updated).Exec()
	} else {
		// Companies live in Scylla only when the optional cloud mirror is disabled.
		companyQuery := db.Query(&records)
		companyQuery.Updated.GreaterEqual(int32(updated))
		return records, companyQuery.AllowFilter().Exec()
	}
}

// getCompanyByID keeps cloud-mirrored and self-hosted company lookups consistent.
func getCompanyByID(companyID int32) (*types.Company, error) {
	var record *types.Company
	var err error
	if cloud.IsDataMirrorEnabled() {
		record, err = cloud.GetByID(types.Company{ID: companyID})
	} else {
		// Resolve the company by its Scylla primary key in self-hosted mode.
		records := []types.Company{}
		companyQuery := db.Query(&records)
		companyQuery.ID.Equals(companyID).Limit(1)
		err = companyQuery.Exec()
		if err == nil && len(records) > 0 {
			record = &records[0]
		}
	}
	return record, err
}

type publicCompanyName struct {
	ID   int32
	Name string
}

// GetPublicCompanyNamesByIDs serves immutable cache rows for the pre-login company selector.
func GetPublicCompanyNamesByIDs(req *core.HandlerArgs) core.HandlerResponse {
	companyIDs, err := req.GetStaticRequestIDs()
	if err != nil {
		return req.MakeErr(err)
	}

	companyNames := make([]publicCompanyName, 0, len(companyIDs))
	for _, rawCompanyID := range companyIDs {
		companyID := int32(rawCompanyID)
		if companyID <= 0 {
			continue
		}

		core.Log("Consultando nombre público de empresa::", companyID)
		record, err := getCompanyByID(companyID)
		if err != nil {
			return req.MakeErr("Error al obtener la empresa.", err)
		}
		if record == nil {
			continue
		}

		companyNames = append(companyNames, publicCompanyName{ID: record.ID, Name: record.Name})
	}

	return req.MakeResponse(core.StaticCachedPartition(companyNames))
}

// La autorización la resuelve mainHandler con el catálogo: "GET.company-parametros" está mapeado a
// "Mi Empresa" y a "Punto de Venta" (el POS necesita RUC y razón social para el comprobante).
func GetEmpresaParametros(req *core.HandlerArgs) core.HandlerResponse {
	record, err := getCompanyByID(req.User.CompanyID)
	if err != nil {
		return req.MakeErr("Error al obtener la company.", err)
	}
	if record == nil {
		return req.MakeErr("No se encontró la company solicitada.")
	}

	records := []types.Company{*record}
	return req.MakeResponse(core.WithDeltaReplace(records))
}

func PostEmpresaParametros(req *core.HandlerArgs) core.HandlerResponse {
	record := types.Company{}
	err := json.Unmarshal([]byte(*req.Body), &record)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	// Email is not required: seeded and imported companies have none, and the global email index
	// on the companies table tolerates a blank value. Public sign-up sets it on its own path.
	if len(record.Name) == 0 || len(record.RUC) == 0 || len(record.LegalName) == 0 {
		return req.MakeErr("Falta alguno de los siguiente parámetros: Nombre, Razon-Social, RUC.")
	}

	// The flags tab posts the whole company, so a payload can carry any id it likes: refuse the
	// ones the catalog does not declare instead of storing a rule the browser cannot show back.
	record.Flags, err = core.SanitizeCompanyFlags(record.Flags)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	// Same gate for the flags that carry a number instead of an on/off: an id the catalog does
	// not declare as valued, or a number wider than the declared byte width, is refused rather
	// than truncated into something the company never typed.
	record.FlagValues, err = core.SanitizeCompanyFlagValues(record.FlagValues)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	// Escribir exige "Mi Empresa" en nivel de escritura; mainHandler ya lo validó por el catálogo.
	record.ID = req.User.CompanyID
	if len(record.FormApiKey) == 0 {
		record.FormApiKey = core.MakeRandomBase36String(18)
	}

	record.Updated = core.SUnixTime()
	if record, err = saveCompanyKeepingSeries(record); err != nil {
		return req.MakeErr("Error al guardar el registro de la company:", err)
	}
	if err = cloud.Insert([]types.Company{record}); err != nil {
		return req.MakeErr("Error al guardar el registro de la company en cloud:", err)
	}

	// Save the public company file used by ecommerce/public clients.
	empresaPublic := types.CompanyPub{
		ID:            record.ID,
		Name:          record.Name,
		CulqiLlave:    record.CulqiConfig.PubKeyDev,
		CulqiRsaKey:   record.CulqiConfig.RsaKey,
		CulqiRsaKeyID: record.CulqiConfig.RsaKeyID,
	}

	empresaPublicBytes, _ := json.Marshal(empresaPublic)
	core.Log("Guardando::", string(empresaPublicBytes))

	cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "empresas",
		FileContent: empresaPublicBytes,
		Name:        fmt.Sprintf("e-%v.json", req.User.CompanyID),
	})

	core.Print(empresaPublic)
	cloud.StoreCompanyConfigAsync(record.ID)
	return core.MakeResponse(req, &record)
}
