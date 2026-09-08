package invoicing

import (
	"app/cloud"
	"app/core"
	"app/invoicing/types"
	"context"
	"encoding/json"
)

// PostInvoiceSeries adds or edits one series.
//
// The series live inline on the company, but they are not written by the company
// form: this is their own endpoint so the two save independently. A user editing
// the company's RUC and a user adding a series never overwrite each other, and the
// Save button on the company panel means what it says — it saves that panel.
//
// It takes one series rather than the whole set, because a set-shaped body makes a
// stale client able to delete series it never saw.
func PostInvoiceSeries(req *core.HandlerArgs) core.HandlerResponse {
	body := types.InvoiceSeries{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body: " + err.Error())
	}

	companyID := req.User.CompanyID

	// The set is read, changed and written back, so two saves have to be ordered
	// or the later one drops whatever the earlier one added.
	lock, lockErr := core.AcquireLock(context.Background(), core.ActionSaveInvoiceSeries, int64(companyID), 2)
	if lockErr != nil {
		return lockErr.Response(req)
	}
	defer lock.Release()

	allSeries, err := LoadCompanySeries(companyID)
	if err != nil {
		return req.MakeErr(err.Error())
	}

	// A new series takes the next id and is active. Ids are never reused, because
	// documents already issued resolve their series by this number.
	//
	// An existing one keeps whatever status the caller sent, which is the only way a
	// series is retired — defaulting that to active here is what would make retiring
	// impossible.
	isNewSeries := body.SeriesID == 0
	if isNewSeries {
		body.SeriesID = types.NextSeriesID(allSeries)
		body.Status = 1
	}
	if body.SeriesID > types.MaxSeriesID {
		return req.MakeErr("La empresa ya no puede tener más series: el máximo es 99.")
	}

	// Editing replaces exactly one entry and leaves the other four alone. A caller
	// naming an id that does not exist is not creating one: ids come from
	// NextSeriesID, and honouring a client-chosen number would let a stale tab burn
	// ids out of the 99 there are, or resurrect one it remembers.
	replaced := false
	for index := range allSeries {
		if allSeries[index].SeriesID == body.SeriesID {
			allSeries[index] = body
			replaced = true
			break
		}
	}
	if !replaced && isNewSeries {
		allSeries = append(allSeries, body)
	} else if !replaced {
		return req.MakeErr("La serie que se intenta editar ya no existe.")
	}

	// The default is a property of the set, so it is settled here rather than by the
	// caller. Sending one series that claims the default would otherwise leave the
	// previous holder claiming it too, and the set would fail validation.
	types.ApplyDefaultSeries(allSeries, body.SeriesID)

	// SaveCompanySeries validates the whole set: ids and active codes unique, one
	// default per document type, and the prefix SUNAT requires for the type.
	if err := SaveCompanySeries(companyID, allSeries); err != nil {
		return req.MakeErr(err.Error())
	}

	cloud.StoreCompanyConfigAsync(companyID)
	return core.MakeResponse(req, &map[string]any{"InvoiceSeries": allSeries})
}
