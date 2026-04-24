package logistica

import (
	"app/core"
	"app/db"
	logisticaTypes "app/logistica/types"
)

// GetPurchaseOrdersQuery powers the optional report view over purchase orders.
// Indexes on PurchaseOrder are grouped by Week (not Date), so the incoming day range
// is converted into a week range (YYWW Code) before the QueryIndexGroup plan runs.
// The client post-filters by exact Date afterwards because week boundaries may spill
// a couple of days outside the requested range.
func GetPurchaseOrdersQuery(req *core.HandlerArgs) core.HandlerResponse {
	fechaStart := req.GetQueryInt16("fecha-start")
	fechaEnd := req.GetQueryInt16("fecha-end")
	providerID := int32(req.GetQueryInt("provider-id"))
	productID := int32(req.GetQueryInt("product-id"))
	status := int8(req.GetQueryInt("status"))

	if fechaStart <= 0 || fechaEnd <= 0 {
		return req.MakeErr("Debe especificar el rango de fechas.")
	}
	if fechaEnd < fechaStart {
		return req.MakeErr("La fecha final no puede ser menor a la fecha inicial.")
	}
	if fechaEnd-fechaStart > 366 {
		return req.MakeErr("Sólo se puede consultar hasta 1 año a la vez.")
	}

	// Convert day-unix range to the YYWW Code range used by the grouped indexes.
	weekStart := core.MakeSemanaFromFechaUnix(fechaStart, false).Code
	weekEnd := core.MakeSemanaFromFechaUnix(fechaEnd, false).Code

	cacheGroupHashes, err := core.ExtractGroupIndexCacheValues(req)
	if err != nil {
		return req.MakeErr(err)
	}

	records := []db.RecordGroup[logisticaTypes.PurchaseOrder]{}
	query := db.QueryIndexGroup(&records).
		CompanyID.Equals(req.Usuario.EmpresaID)

	for _, cacheGroup := range cacheGroupHashes {
		query.IncludeCachedGroup(cacheGroup.GroupHash, cacheGroup.UpdateCounter)
	}

	// Week is the single BETWEEN required by QueryIndexGroup; remaining equality filters
	// drive index selection toward the most specific compatible grouped index.
	query.Week.Between(weekStart, weekEnd)

	switch {
	case status > 0 && productID > 0:
		// Uses the raw group: Week + Status + DetailProductIDs.
		query.Status.Equals(status).DetailProductIDs.Contains(productID)
	case status > 0 && providerID > 0:
		// Uses the raw group: Week + Status + ProviderID.
		query.Status.Equals(status).ProviderID.Equals(providerID)
	case productID > 0:
		// Uses the raw group: Week + DetailProductIDs.
		query.DetailProductIDs.Contains(productID)
	case providerID > 0:
		// Uses the raw group: Week + ProviderID.
		query.ProviderID.Equals(providerID)
	}
	// Status-only case falls back to the base Week group and the client post-filters.
	// No Week+Status grouped index exists, so adding Status here would fail index selection.

	if err := query.Exec(); err != nil {
		core.Log("Error querying purchase order groups:", err)
		return req.MakeErr("Error al obtener las órdenes de compra.")
	}

	return core.MakeResponse(req, &records)
}
