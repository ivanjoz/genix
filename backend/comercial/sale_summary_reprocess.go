package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
)

func SaleOrderReprocess(companyID int32, fecha int16) {
	// Step 1: load every sale for the requested company and day.
	sales := []types.SaleOrder{}
	query := db.Query(&sales)
	query.Select(query.ID, query.Status, query.DetailProductsIDs, query.DetailPrices, query.DetailQuantities).
		EmpresaID.Equals(companyID).Fecha.Equals(fecha)

	if err := query.AllowFilter().Exec(); err != nil {
		core.Log("SaleOrderReprocess query error:", "companyID", companyID, "fecha", fecha, "error", err)
		return
	}

	// Step 2: build one summary row per product and keep its index for O(1) updates.
	summary := types.SaleSummary{
		EmpresaID: companyID,
		Fecha:     fecha,
	}
	productIndexByID := map[int32]int{}

	for _, sale := range sales {
		if len(sale.DetailProductsIDs) != len(sale.DetailQuantities) || len(sale.DetailProductsIDs) != len(sale.DetailPrices) {
			core.Log("SaleOrderReprocess invalid sale detail lengths:", "saleID", sale.ID, "status", sale.Status)
			continue
		}

		// Status encodes whether payment and delivery were completed for the sale.
		hasPaymentCompleted := sale.Status == 2 || sale.Status == 4
		hasDeliveryCompleted := sale.Status == 3 || sale.Status == 4
		if sale.Status < 1 || sale.Status > 4 {
			continue
		}

		for lineIndex, productID := range sale.DetailProductsIDs {
			quantity := sale.DetailQuantities[lineIndex]
			if productID <= 0 || quantity <= 0 {
				continue
			}

			summaryIndex, exists := productIndexByID[productID]
			if !exists {
				// Append aligned slices once and reuse the saved index for later lines.
				summary.ProductIDs = append(summary.ProductIDs, productID)
				summary.Quantity = append(summary.Quantity, 0)
				summary.QuantityPendingDelivery = append(summary.QuantityPendingDelivery, 0)
				summary.TotalAmount = append(summary.TotalAmount, 0)
				summary.TotalDebtAmount = append(summary.TotalDebtAmount, 0)
				summaryIndex = len(summary.ProductIDs) - 1
				productIndexByID[productID] = summaryIndex
			}

			lineAmount := core.MultiplyInt32Saturated(sale.DetailPrices[lineIndex], quantity)
			summary.Quantity[summaryIndex] += quantity
			summary.TotalAmount[summaryIndex] += lineAmount
			if !hasDeliveryCompleted {
				summary.QuantityPendingDelivery[summaryIndex] += quantity
			}
			if !hasPaymentCompleted {
				summary.TotalDebtAmount[summaryIndex] += lineAmount
			}
		}
	}

	// Step 3: save the rebuilt daily snapshot in one write.
	summary.Updated = core.SUnixTime()
	summary.ReprocessUpdated = summary.Updated
	if err := db.InsertOne(summary); err != nil {
		core.Log("SaleOrderReprocess save error:", "companyID", companyID, "fecha", fecha, "error", err)
	}
}


func SaleOrderReprocessHandler(args *core.ExecArgs) core.FuncResponse {
	
	return core.FuncResponse{}
}
