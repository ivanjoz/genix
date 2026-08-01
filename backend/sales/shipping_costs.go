package sales

import (
	"app/core"
	"app/db"
	s "app/sales/types"
	"encoding/json"
	"strconv"
)

func GetShippingCosts(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("upv")

	shippingCosts := []s.ShippingCost{}
	query := db.Query(&shippingCosts)
	// No status to filter here, so Delta() constrains nothing but the watermark.
	query.CompanyID.Equals(req.User.CompanyID).Delta(updatedSince)

	core.Log("GetShippingCosts:", "companyID=", req.User.CompanyID, "upv_since=", updatedSince)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener costos de envío:", err)
	}

	core.Log("GetShippingCosts result:", len(shippingCosts))
	return req.MakeResponse(shippingCosts)
}

func PostShippingCosts(req *core.HandlerArgs) core.HandlerResponse {
	shippingCostsPayload := []s.ShippingCost{}
	if err := json.Unmarshal([]byte(*req.Body), &shippingCostsPayload); err != nil {
		core.Log("PostShippingCosts deserialization error:", err)
		return req.MakeErr("Error al deserializar costos de envío:", err)
	}

	recordsToSave := []s.ShippingCost{}
	nowTime := core.SUnixTime()
	seenCityIDs := map[int32]struct{}{}
	for recordIndex := range shippingCostsPayload {
		record := shippingCostsPayload[recordIndex]
		if !record.HasUpdated {
			continue
		}
		if record.CityID <= 0 {
			return req.MakeErr("El costo de envío en posición " + strconv.Itoa(recordIndex) + " debe tener CityID válido.")
		}
		if record.FlatCost < 0 || record.CostPerKg < 0 {
			return req.MakeErr("El costo de envío en posición " + strconv.Itoa(recordIndex) + " no puede tener costos negativos.")
		}
		if _, exists := seenCityIDs[record.CityID]; exists {
			return req.MakeErr("Se envió más de un costo de envío para la misma ciudad.")
		}

		// Tenant and audit fields are server-owned; the client only chooses city and costs.
		record.CompanyID = req.User.CompanyID
		record.Updated = nowTime
		record.UpdatedBy = req.User.ID
		record.HasUpdated = false
		seenCityIDs[record.CityID] = struct{}{}
		recordsToSave = append(recordsToSave, record)
	}

	if len(recordsToSave) == 0 {
		core.Log("PostShippingCosts no-op:", "payload=", len(shippingCostsPayload))
		return req.MakeResponse(recordsToSave)
	}

	shippingCostTable := db.TableOf[s.ShippingCost]()
	if err := db.Merge(&recordsToSave,
		db.Cols(shippingCostTable.Created, shippingCostTable.CreatedBy),
		func(prev, current *s.ShippingCost) bool {
			// Merge avoids write churn; unchanged cost rows keep their previous Updated watermark.
			current.Created = prev.Created
			current.CreatedBy = prev.CreatedBy
			return current.FlatCost != prev.FlatCost || current.CostPerKg != prev.CostPerKg
		},
		func(current *s.ShippingCost) {
			current.Created = nowTime
			current.CreatedBy = req.User.ID
		},
	); err != nil {
		core.Log("PostShippingCosts merge error:", err)
		return req.MakeErr("Error al guardar costos de envío:", err)
	}

	core.Log("PostShippingCosts saved:", len(recordsToSave))
	return req.MakeResponse(recordsToSave)
}
