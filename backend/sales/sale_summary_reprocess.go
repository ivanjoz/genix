package sales

import (
	"app/sales/types"
	"app/core"
	"app/db"
)

func SaleOrderReprocess(companyID int32, date int16) {
	// Load every persisted sale state for the requested scope so the rebuild ignores transient action history.
	sales := []types.SaleOrder{}
	query := db.Query(&sales)
	query.Select(query.ID, query.Status, query.Date, query.DetailProductsIDs, query.DetailPrices, query.DetailQuantities).
		CompanyID.Equals(companyID)

	if date > 0 {
		query.Date.Equals(date)
	}

	if err := query.AllowFilter().Exec(); err != nil {
		core.Log("SaleOrderReprocess query error:", "companyID", companyID, "date", date, "error", err)
		return
	}

	changesByFecha := map[int16][]ProductSummaryChange{}
	for _, sale := range sales {
		if sale.Date <= 0 {
			core.Log("SaleOrderReprocess skipped invalid date:", "saleID", sale.ID, "date", sale.Date)
			continue
		}
		if len(sale.DetailProductsIDs) != len(sale.DetailQuantities) || len(sale.DetailProductsIDs) != len(sale.DetailPrices) {
			core.Log("SaleOrderReprocess invalid sale detail lengths:", "saleID", sale.ID, "status", sale.Status)
			continue
		}

		// Status encodes whether payment and delivery were completed for the persisted sale.
		hasPaymentCompleted := sale.Status == 2 || sale.Status == 4
		hasDeliveryCompleted := sale.Status == 3 || sale.Status == 4
		if sale.Status < 1 || sale.Status > 4 {
			core.Log("SaleOrderReprocess skipped invalid status:", "saleID", sale.ID, "status", sale.Status)
			continue
		}

		summaryChanges := []ProductSummaryChange{}
		for lineIndex, productID := range sale.DetailProductsIDs {
			quantity := sale.DetailQuantities[lineIndex]
			if productID <= 0 || quantity <= 0 {
				continue
			}

			lineAmount := core.MultiplyInt32Saturated(sale.DetailPrices[lineIndex], quantity)
			quantityPendingDelivery := int32(0)
			if !hasDeliveryCompleted {
				// Reprocess derives pending delivery directly from the persisted sale status.
				quantityPendingDelivery = quantity
			}
			totalDebtAmount := int32(0)
			if !hasPaymentCompleted {
				// Reprocess derives pending debt directly from the persisted sale status.
				totalDebtAmount = lineAmount
			}
			summaryChanges = append(summaryChanges, ProductSummaryChange{
				productID: productID,
				SaleOrderProductStats: types.SaleOrderProductStats{
					Quantity:                quantity,
					QuantityPendingDelivery: quantityPendingDelivery,
					TotalAmount:             lineAmount,
					TotalDebtAmount:         totalDebtAmount,
				},
			})
		}
		changesByFecha[sale.Date] = append(changesByFecha[sale.Date], summaryChanges...)
	}

	for targetFecha, summaryChanges := range changesByFecha {
		core.Log("SaleOrderReprocess applying summary rows", "companyID", companyID, "date", targetFecha, "changes", len(summaryChanges))
		if err := applyChangesToSaleSumary(companyID, targetFecha, summaryChanges, true); err != nil {
			core.Log("SaleOrderReprocess apply changes error:", "companyID", companyID, "date", targetFecha, "error", err)
		}
	}
}

func SaleOrderReprocessHandler(args *core.ExecArgs) core.FuncResponse {
	return core.FuncResponse{}
}
