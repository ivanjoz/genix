package sales

import (
	"app/core"
	"app/db"
	s "app/sales/types"
	"encoding/json"
)

func GetSalesPlanning(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("upv")

	records := []s.SalesPlanning{}

	// Delta() keeps only active rows on a first sync and fans out over both statuses afterwards, so
	// the client can evict deleted ones — it replaces the per-status loop this handler used to run.
	query := db.Query(&records)
	query.CompanyID.Equals(req.User.CompanyID).Delta(updatedSince, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la planificación de ventas:", err)
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
		db.Cols(t.Created),
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
	updatedSince := req.GetQueryInt("upv")

	records := []s.SeasonalityCurve{}

	// Delta() keeps only active rows on a first sync and fans out over both statuses afterwards, so
	// the client can evict deleted ones — it replaces the per-status loop this handler used to run.
	query := db.Query(&records)
	query.CompanyID.Equals(req.User.CompanyID).Delta(updatedSince, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las curvas de estacionalidad:", err)
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
		db.Cols(t.Created),
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
