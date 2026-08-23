package invoicing

import (
	"app/core"
	"app/db"
	"app/invoicing/types"
	"encoding/json"
	"strings"
)

// GetInvoiceSeries lists the series a company issues under.
func GetInvoiceSeries(req *core.HandlerArgs) core.HandlerResponse {
	updatedVersion := req.GetQueryInt("upv")

	series := []types.InvoiceSeries{}
	query := db.Query(&series)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(updatedVersion, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las series:", err)
	}
	return core.MakeResponse(req, &map[string]any{"InvoiceSeries": series})
}

// PostInvoiceSeries creates or updates a series.
//
// A series code is fixed by SUNAT's format and by what the company registered
// with them, so the validation here is not a preference: the wrong letter means
// every document numbered under it is rejected.
func PostInvoiceSeries(req *core.HandlerArgs) core.HandlerResponse {
	body := types.InvoiceSeries{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	body.SeriesCode = strings.ToUpper(strings.TrimSpace(body.SeriesCode))
	if err := validateSeries(&body); err != nil {
		return req.MakeErr(err.Error())
	}

	companyID := req.User.CompanyID
	existing := []types.InvoiceSeries{}
	query := db.Query(&existing)
	query.Select().CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al leer las series:", err)
	}

	packedID := types.PackDocTypeSeries(body.DocType, body.SeriesID)
	isNew := true
	for index := range existing {
		current := &existing[index]
		if current.ID == packedID {
			isNew = false
			continue
		}
		// The code has to be unique: two series numbering into the same code
		// would produce duplicate document numbers at SUNAT.
		if current.SeriesCode == body.SeriesCode && current.Status == 1 {
			return req.MakeErr("Ya existe una serie con el código " + body.SeriesCode)
		}
	}

	now := core.SUnixTime()
	body.CompanyID = companyID
	body.ID = packedID
	body.Updated = now
	body.UpdatedBy = req.User.ID
	if body.Status == 0 {
		body.Status = 1
	}
	if isNew {
		body.Created = now
		body.CreatedBy = req.User.ID
	}

	records := &[]types.InvoiceSeries{body}
	var err error
	if isNew {
		err = db.Insert(records)
	} else {
		table := db.TableOf[types.InvoiceSeries]()
		err = db.UpdateExclude(records, table.Created, table.CreatedBy)
	}
	if err != nil {
		return req.MakeErr("Error al guardar la serie:", err)
	}
	return req.MakeResponse((*records)[0])
}

// validateSeries checks what SUNAT checks: four characters, and a first letter
// that matches the family of document the series numbers.
func validateSeries(series *types.InvoiceSeries) error {
	switch series.DocType {
	case types.DocTypeFactura, types.DocTypeBoleta,
		types.DocTypeCreditNote, types.DocTypeDebitNote:
	default:
		return core.Err("El tipo de comprobante no es válido.")
	}

	if len(series.SeriesCode) != 4 {
		return core.Err("El código de serie debe tener 4 caracteres, por ejemplo F001.")
	}
	prefix := series.SeriesCode[0]
	if prefix != 'F' && prefix != 'B' {
		return core.Err("El código de serie debe empezar con F o con B.")
	}
	if series.DocType == types.DocTypeFactura && prefix != 'F' {
		return core.Err("Una factura necesita una serie que empiece con F.")
	}
	if series.DocType == types.DocTypeBoleta && prefix != 'B' {
		return core.Err("Una boleta necesita una serie que empiece con B.")
	}

	if series.SeriesID <= 0 || series.SeriesID > 999 {
		return core.Err("El número interno de la serie debe estar entre 1 y 999.")
	}
	if series.SiteID == 0 {
		return core.Err("Debe indicar la sede que emite con esta serie.")
	}
	return nil
}
