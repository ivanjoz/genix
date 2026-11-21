package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	hr "app/hr/types"
	"encoding/json"
)

// GetTaxDueCalendar syncs SUNAT's monthly due-date schedule. Every company reads it (the cash flow
// forecast dates its taxes with it); only the SaaS company writes.
func GetTaxDueCalendar(req *core.HandlerArgs) core.HandlerResponse {
	calendars := []types.TaxDueCalendar{}
	query := db.Query(&calendars)
	query.Select().CountryID.Equals(hr.CountryIDPeru).Delta(req.GetDeltaCursor(""))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el calendario tributario:", err)
	}
	return core.MakeResponse(req, &calendars)
}

// PostTaxDueCalendar saves the schedules of one or more periods (SaaS only, see saasOnlyRoutes).
// The period is the key, so a period saved again is overwritten; Status 0 deletes it.
func PostTaxDueCalendar(req *core.HandlerArgs) core.HandlerResponse {
	calendars := []types.TaxDueCalendar{}
	if err := json.Unmarshal([]byte(*req.Body), &calendars); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if len(calendars) == 0 {
		return req.MakeErr("No se enviaron periodos.")
	}
	for index := range calendars {
		calendar := &calendars[index]
		if calendar.Status != 0 {
			if err := types.ValidateTaxDueCalendar(*calendar); err != nil {
				return req.MakeErr(err)
			}
			calendar.Status = 1
		}
		calendar.CountryID, calendar.UpdatedBy = hr.CountryIDPeru, req.User.ID
	}
	if err := db.Upsert(&calendars); err != nil {
		return req.MakeErr("Error al guardar el calendario tributario:", err)
	}
	return req.MakeResponse(calendars)
}
