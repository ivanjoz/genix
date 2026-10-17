package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
)

// GetSaleSummary returns day-level summaries reconstructed from per-product rows.
func GetSaleSummary(req *core.HandlerArgs) core.HandlerResponse {
	dateInicio := req.GetQueryInt16("date-inicio")
	if dateInicio == 0 {
		dateInicio = core.FechaUnix() - (8 * 7)
	}

	dateFin := req.GetQueryInt16("date-fin")
	if dateFin == 0 {
		dateFin = core.FechaUnix()
	}

	if dateInicio > 0 && dateFin > 0 && dateInicio > dateFin {
		return req.MakeErr("La date de inicio no puede ser mayor que la date final.")
	}

	datesToInclude := []int16{}
	for i := dateInicio; i <= dateFin; i++ {
		datesToInclude = append(datesToInclude, i)
	}

	productRows := []types.ProductSaleSummary{}
	query := db.Query(&productRows)
	query.CompanyID.Equals(req.User.CompanyID).Date.In(datesToInclude...)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el resumen de ventas.", err)
	}

	// No delta log: the whole range every sync, with the replace Metadata.
	return req.MakeResponse(core.WithDeltaReplace(buildSaleSummariesFromProductRows(productRows)))
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
				Date:      productRow.Date,
			}
			summaryByFecha[productRow.Date] = summary
			orderedFechas = append(orderedFechas, productRow.Date)
		}

		summary.ProductIDs = append(summary.ProductIDs, productRow.ProductID)
		summary.Quantity = append(summary.Quantity, productStats.Quantity)
		summary.SubQuantity = append(summary.SubQuantity, productStats.SubQuantity)
		summary.QuantityPendingDelivery = append(summary.QuantityPendingDelivery, productStats.QuantityPendingDelivery)
		summary.SubQuantityPendingDelivery = append(summary.SubQuantityPendingDelivery, productStats.SubQuantityPendingDelivery)
		summary.SubDivisor = append(summary.SubDivisor, productStats.SubDivisor)
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
