package sales

import (
	"app/sales/types"
	"app/core"
	"app/db"
	"slices"
	"time"
)

// GetSaleSummary returns day-level summaries reconstructed from per-product rows.
func GetSaleSummary(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	dateInicio := req.GetQueryInt16("date-inicio")
	if dateInicio == 0 {
		dateInicio = core.TimeToFechaUnix(time.Now()) - (8*7)
	}
	
	dateFin := req.GetQueryInt16("date-fin")
	if dateFin == 0 {
		dateFin = core.TimeToFechaUnix(time.Now())
	}

	// Debug trace to diagnose sync ranges and filters from clients.
	core.Log("GetSaleSummary params:", "companyID", req.User.CompanyID, "updated", updated, "dateInicio", dateInicio, "dateFin", dateFin)

	if dateInicio > 0 && dateFin > 0 && dateInicio > dateFin {
		return req.MakeErr("La date de inicio no puede ser mayor que la date final.")
	}

	datesToInclude := []int16{}

	if updated == 0 {
		for i := dateInicio; i <= dateFin; i++ {
			datesToInclude = append(datesToInclude, i)
		}
	} else {
		// Delta sync resolves changed dates from sentinel rows stored at date=-1.
		metadataRows := []types.ProductSaleSummary{}
		query := db.Query(&metadataRows)
		query.CompanyID.Equals(req.User.CompanyID).Date.Equals(-1).Updated.GreaterThan(updated)
		if dateInicio > 0 && dateFin > 0 {
			query.ProductID.Between(int32(dateInicio), int32(dateFin))
		}
		if err := query.AllowFilter().Exec(); err != nil {
			core.Log("GetSaleSummary metadata query error:", err)
			return req.MakeErr("Error al obtener el resumen de ventas.", err)
		}

		for _, metadataRow := range metadataRows {
			date := int16(metadataRow.ProductID)
			if date <= 0 {
				continue
			}
			if !slices.Contains(datesToInclude,date) {
				datesToInclude = append(datesToInclude, date)
			}
		}
	}

	if len(datesToInclude) == 0 {
		return req.MakeResponse([]types.SaleSummary{})
	}

	productRows := []types.ProductSaleSummary{}
	query := db.Query(&productRows)
	query.CompanyID.Equals(req.User.CompanyID).Date.In(datesToInclude...)
	
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
		if productRow.Date <= 0 || productRow.ProductID <= 0 {
			continue
		}

		productStats, err := decodeSaleSummaryStats(productRow.Stats)
		if err != nil {
			core.Log("buildSaleSummariesFromProductRows decode error:", "companyID", productRow.CompanyID, "date", productRow.Date, "productID", productRow.ProductID, "error", err)
			continue
		}

		summary := summaryByFecha[productRow.Date]
		if summary == nil {
			summary = &types.SaleSummary{
				CompanyID: productRow.CompanyID,
				Date:     productRow.Date,
			}
			summaryByFecha[productRow.Date] = summary
			orderedFechas = append(orderedFechas, productRow.Date)
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
	for _, date := range orderedFechas {
		summaries = append(summaries, *summaryByFecha[date])
	}
	return summaries
}
