package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"slices"
	"time"
)

// GetSaleSummary returns day-level summaries reconstructed from per-product rows.
func GetSaleSummary(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	fechaInicio := req.GetQueryInt16("fecha-inicio")
	if fechaInicio == 0 {
		fechaInicio = core.TimeToFechaUnix(time.Now()) - (8*7)
	}
	
	fechaFin := req.GetQueryInt16("fecha-fin")
	if fechaFin == 0 {
		fechaFin = core.TimeToFechaUnix(time.Now())
	}

	// Debug trace to diagnose sync ranges and filters from clients.
	core.Log("GetSaleSummary params:", "empresaID", req.Usuario.EmpresaID, "updated", updated, "fechaInicio", fechaInicio, "fechaFin", fechaFin)

	if fechaInicio > 0 && fechaFin > 0 && fechaInicio > fechaFin {
		return req.MakeErr("La fecha de inicio no puede ser mayor que la fecha final.")
	}

	fechasToInclude := []int16{}

	if updated == 0 {
		for i := fechaInicio; i <= fechaFin; i++ {
			fechasToInclude = append(fechasToInclude, i)
		}
	} else {
		// Delta sync resolves changed fechas from sentinel rows stored at fecha=-1.
		metadataRows := []types.ProductSaleSummary{}
		query := db.Query(&metadataRows)
		query.CompanyID.Equals(req.Usuario.EmpresaID).Fecha.Equals(-1).Updated.GreaterThan(updated)
		if fechaInicio > 0 && fechaFin > 0 {
			query.ProductID.Between(int32(fechaInicio), int32(fechaFin))
		}
		if err := query.AllowFilter().Exec(); err != nil {
			core.Log("GetSaleSummary metadata query error:", err)
			return req.MakeErr("Error al obtener el resumen de ventas.", err)
		}

		for _, metadataRow := range metadataRows {
			fecha := int16(metadataRow.ProductID)
			if fecha <= 0 {
				continue
			}
			if !slices.Contains(fechasToInclude,fecha) {
				fechasToInclude = append(fechasToInclude, fecha)
			}
		}
	}

	if len(fechasToInclude) == 0 {
		return req.MakeResponse([]types.SaleSummary{})
	}

	productRows := []types.ProductSaleSummary{}
	query := db.Query(&productRows)
	query.CompanyID.Equals(req.Usuario.EmpresaID).Fecha.In(fechasToInclude...)
	
	if updated > 0 {
		query.Updated.GreaterEqual(updated).AllowFilter()
	}
	
	if err := query.Exec(); err != nil {
		core.Log("GetSaleSummary query error:", err)
		return req.MakeErr("Error al obtener el resumen de ventas.", err)
	}

	summaries := buildSaleSummariesFromProductRows(productRows)
	core.Log("GetSaleSummary response count:", len(summaries))
	return req.MakeResponse(summaries)
}

func buildSaleSummariesFromProductRows(productRows []types.ProductSaleSummary) []types.SaleSummary {
	summaryByFecha := map[int16]*types.SaleSummary{}
	orderedFechas := make([]int16, 0)

	for _, productRow := range productRows {
		// Sentinel rows are metadata only and must not appear inside the reconstructed day payload.
		if productRow.Fecha <= 0 || productRow.ProductID <= 0 {
			continue
		}

		productStats, err := decodeSaleSummaryStats(productRow.Stats)
		if err != nil {
			core.Log("buildSaleSummariesFromProductRows decode error:", "companyID", productRow.CompanyID, "fecha", productRow.Fecha, "productID", productRow.ProductID, "error", err)
			continue
		}

		summary := summaryByFecha[productRow.Fecha]
		if summary == nil {
			summary = &types.SaleSummary{
				EmpresaID: productRow.CompanyID,
				Fecha:     productRow.Fecha,
			}
			summaryByFecha[productRow.Fecha] = summary
			orderedFechas = append(orderedFechas, productRow.Fecha)
		}

		summary.ProductIDs = append(summary.ProductIDs, productRow.ProductID)
		summary.Quantity = append(summary.Quantity, productStats.Quantity)
		summary.QuantityPendingDelivery = append(summary.QuantityPendingDelivery, productStats.QuantityPendingDelivery)
		summary.TotalAmount = append(summary.TotalAmount, productStats.TotalAmount)
		summary.TotalDebtAmount = append(summary.TotalDebtAmount, productStats.TotalDebtAmount)
		if productRow.Updated > summary.Updated {
			summary.Updated = productRow.Updated
		}
	}

	summaries := make([]types.SaleSummary, 0, len(orderedFechas))
	for _, fecha := range orderedFechas {
		summaries = append(summaries, *summaryByFecha[fecha])
	}
	return summaries
}
