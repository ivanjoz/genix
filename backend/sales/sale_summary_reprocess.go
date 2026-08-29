package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
)

func SaleOrderReprocess(companyID int32, date int16) {
	// Load every persisted sale state for the requested scope so the rebuild ignores transient action history.
	sales := []types.SaleOrder{}
	query := db.Query(&sales)
	query.Select(query.ID, query.Status, query.Date, query.DetailProductsIDs,
		query.DetailPrices, query.DetailSubPrices, query.DetailQuantities, query.DetailSubDivisor).
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
			packedQuantity := sale.DetailQuantities[lineIndex]
			if productID <= 0 || packedQuantity <= 0 {
				continue
			}

			// The order line is packed; the summary accumulates, so split it back apart.
			lineQuantity := core.UnpackQuantityLine(packedQuantity)
			lineDivisor := core.GetIndex(sale.DetailSubDivisor, lineIndex)
			if lineDivisor <= 0 {
				lineDivisor = core.QuantityDivisorNone
			}
			// Reconstructed from what was charged on the line, not from the product's price
			// today: a reprocess must reproduce the original amounts.
			lineAmount := core.QuantityAmount(lineQuantity,
				core.GetIndex(sale.DetailPrices, lineIndex), core.GetIndex(sale.DetailSubPrices, lineIndex))

			pendingDelivery := core.Quantity{}
			if !hasDeliveryCompleted {
				// Reprocess derives pending delivery directly from the persisted sale status.
				pendingDelivery = lineQuantity
			}
			totalDebtAmount := int32(0)
			if !hasPaymentCompleted {
				// Reprocess derives pending debt directly from the persisted sale status.
				totalDebtAmount = lineAmount
			}
			summaryChanges = append(summaryChanges, ProductSummaryChange{
				productID: productID,
				SaleOrderProductStats: types.SaleOrderProductStats{
					Quantity: lineQuantity.Units,
					// A packed line always unpacks normalized, so Sub is already below the
					// divisor and fits the narrow field.
					SubQuantity:                int16(lineQuantity.Sub),
					QuantityPendingDelivery:    pendingDelivery.Units,
					SubQuantityPendingDelivery: int16(pendingDelivery.Sub),
					SubDivisor:                 lineDivisor,
					TotalAmount:                lineAmount,
					TotalDebtAmount:            totalDebtAmount,
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
