package sales

import (
	"app/core"
	"app/db"
	s "app/sales/types"
	"encoding/json"
)

func GetSalesPlanning(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))

	records := []s.SalesPlanning{}

	// Initial load: active only. Delta: also fetch Status=0 so the client can evict.
	// The [Status, Updated] view requires Status equality, so we query each status.
	statuses := []int8{1}
	if updated > 0 {
		statuses = append(statuses, 0)
	}

	for _, status := range statuses {
		query := db.Query(&records)
		query.CompanyID.Equals(req.User.CompanyID).
			Status.Equals(status).
			Updated.GreaterThan(updated) // updated=0 on initial → matches all rows

		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener la planificación de ventas:", err)
		}
	}

	return req.MakeResponse(records)
}

func PostSalesPlanning(req *core.HandlerArgs) core.HandlerResponse {
	payload := []s.SalesPlanning{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar la planificación de ventas:", err)
	}

	for i := range payload {
		record := &payload[i]
		if record.ProductID <= 0 {
			return req.MakeErr("Cada planificación debe tener un producto válido.")
		}
		// Preserve incoming ID so the frontend can map TempID -> ID after the merge.
		record.TempID = record.ID
		record.CompanyID = req.User.CompanyID
	}

	nowTime := core.SUnixTime()
	t := s.SalesPlanningTable{}
	err := db.Merge(&payload,
		[]db.Coln{t.Created},
		func(prev, current *s.SalesPlanning) bool {
			current.CompanyID = req.User.CompanyID
			current.Created = prev.Created
			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			return true
		},
		func(current *s.SalesPlanning) {
			current.CompanyID = req.User.CompanyID
			current.Created = nowTime
			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			if current.Status == 0 {
				current.Status = 1
			}
		},
	)
	if err != nil {
		return req.MakeErr("Error al guardar la planificación de ventas:", err)
	}

	return req.MakeResponse(payload)
}

func GetSeasonalityCurve(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))

	records := []s.SeasonalityCurve{}

	// Initial load: active only. Delta: also fetch Status=0 so the client can evict.
	// The [Status, Updated] view requires Status equality, so we query each status.
	statuses := []int8{1}
	if updated > 0 {
		statuses = append(statuses, 0)
	}

	for _, status := range statuses {
		query := db.Query(&records)
		query.CompanyID.Equals(req.User.CompanyID).
			Status.Equals(status).
			Updated.GreaterThan(updated) // updated=0 on initial → matches all rows

		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener las curvas de estacionalidad:", err)
		}
	}

	return req.MakeResponse(records)
}

func PostSeasonalityCurve(req *core.HandlerArgs) core.HandlerResponse {
	payload := []s.SeasonalityCurve{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar las curvas de estacionalidad:", err)
	}

	for i := range payload {
		record := &payload[i]
		if len(record.Name) < 2 {
			return req.MakeErr("Cada curva de estacionalidad debe tener un nombre.")
		}
		record.TempID = record.ID
		record.CompanyID = req.User.CompanyID
	}

	nowTime := core.SUnixTime()
	t := s.SeasonalityCurveTable{}
	err := db.Merge(&payload,
		[]db.Coln{t.Created},
		func(prev, current *s.SeasonalityCurve) bool {
			current.CompanyID = req.User.CompanyID
			current.Created = prev.Created
			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			return true
		},
		func(current *s.SeasonalityCurve) {
			current.CompanyID = req.User.CompanyID
			current.Created = nowTime
			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			if current.Status == 0 {
				current.Status = 1
			}
		},
	)
	if err != nil {
		return req.MakeErr("Error al guardar las curvas de estacionalidad:", err)
	}

	return req.MakeResponse(payload)
}
