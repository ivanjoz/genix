package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"slices"
)

type ProductSummaryChange struct {
	productID, quantity, quantityPendingDelivery, amount, totalDebtAmount int32
}

func MakeSummaryChangeFromOSaleOrder(sale types.SaleOrder, actions ...int8) []ProductSummaryChange {
	changes := []ProductSummaryChange{}
	if len(sale.DetailProductsIDs) == 0 {
		return changes
	}
	if len(sale.DetailProductsIDs) != len(sale.DetailQuantities) || len(sale.DetailProductsIDs) != len(sale.DetailPrices) {
		return changes
	}

	// Resolve actions once so each line only applies simple numeric updates.
	includeSale := slices.Contains(actions, 1)
	includePayment := slices.Contains(actions, 2)
	includeDelivery := slices.Contains(actions, 3)
	changesByProduct := map[int32]ProductSummaryChange{}

	for lineIndex, productID := range sale.DetailProductsIDs {
		quantity := sale.DetailQuantities[lineIndex]
		if productID <= 0 || quantity <= 0 {
			continue
		}

		lineAmount := core.MultiplyInt32Saturated(sale.DetailPrices[lineIndex], quantity)
		currentChange := changesByProduct[productID]
		currentChange.productID = productID

		if includeSale {
			// Sale creation adds units and total amount once.
			currentChange.quantity += quantity
			currentChange.amount += lineAmount
			// When delivery is still pending, the full quantity remains pending.
			if !includeDelivery {
				currentChange.quantityPendingDelivery += quantity
			}
			// When payment is still pending, the full line amount remains as debt.
			if !includePayment {
				currentChange.totalDebtAmount += lineAmount
			}
		}
		if includePayment && !includeSale {
			// Pure payment updates only reduce pending debt.
			currentChange.totalDebtAmount -= lineAmount
		}
		if includeDelivery && !includeSale {
			// Pure delivery updates only reduce pending quantity.
			currentChange.quantityPendingDelivery -= quantity
		}
		changesByProduct[productID] = currentChange
	}

	for _, summaryChange := range changesByProduct {
		changes = append(changes, summaryChange)
	}
	return changes
}

func UpdateSaleSumary(summary *types.SaleSummary, sale types.SaleOrder, actions ...int8) error {
	if summary == nil {
		return core.Err("sale summary is nil")
	}

	// Step 1: convert the sale plus action set into one delta per product.
	allChanges := MakeSummaryChangeFromOSaleOrder(sale, actions...)

	// Step 2: apply all product deltas against the in-memory summary.
	if len(allChanges) > 0 {
		if err := UpdateProductOnSumary(summary, allChanges...); err != nil {
			return err
		}
	}

	// Step 3: persist the refreshed summary snapshot.
	summary.Updated = core.SUnixTime()
	if err := db.InsertOne(*summary); err != nil {
		return core.Err("error saving sale summary:", err)
	}
	return nil
}

func updateSaleSummaryForChange(sale types.SaleOrder, actions ...int8) error {
	// Load the current day snapshot first, then apply the requested action deltas.
	previousSummary, err := loadSaleSummary(sale.EmpresaID, sale.Fecha)
	if err != nil {
		return err
	}
	return UpdateSaleSumary(&previousSummary, sale, actions...)
}

func loadSaleSummary(empresaID int32, fecha int16) (types.SaleSummary, error) {
	// Read at most one daily summary because the table key is empresa + fecha.
	summaries := []types.SaleSummary{}
	query := db.Query(&summaries)
	query.EmpresaID.Equals(empresaID).Fecha.Equals(fecha).Limit(1)
	if err := query.Exec(); err != nil {
		return types.SaleSummary{}, core.Err("error querying sale summary:", err)
	}

	if len(summaries) > 0 {
		return summaries[0], nil
	}
	return types.SaleSummary{EmpresaID: empresaID, Fecha: fecha}, nil
}

func UpdateProductOnSumary(summary *types.SaleSummary, summaryChanges ...ProductSummaryChange) error {
	if summary == nil || len(summaryChanges) == 0 {
		return nil
	}

	// Step 1: index existing rows so each product update can find its slot in O(1).
	productIndexes := map[int32]int{}
	for index, productID := range summary.ProductIDs {
		if productID > 0 {
			productIndexes[productID] = index
		}
	}

	// Step 2: merge duplicated product deltas so callers can pass raw line changes safely.
	mergedChanges := map[int32]ProductSummaryChange{}
	for _, summaryChange := range summaryChanges {
		if summaryChange.productID <= 0 {
			continue
		}
		currentChange := mergedChanges[summaryChange.productID]
		currentChange.productID = summaryChange.productID
		currentChange.quantity += summaryChange.quantity
		currentChange.quantityPendingDelivery += summaryChange.quantityPendingDelivery
		currentChange.amount += summaryChange.amount
		currentChange.totalDebtAmount += summaryChange.totalDebtAmount
		mergedChanges[summaryChange.productID] = currentChange
	}

	// Step 3: update the existing row or append a new one for each product delta.
	for productID, summaryChange := range mergedChanges {
		index, exists := productIndexes[productID]
		if !exists {
			index = appendSummaryRow(summary, productID)
			productIndexes[productID] = index
		}

		summary.Quantity[index] = core.ClampInt32ToZero(summary.Quantity[index] + summaryChange.quantity)
		summary.QuantityPendingDelivery[index] = core.ClampInt32ToZero(summary.QuantityPendingDelivery[index] + summaryChange.quantityPendingDelivery)
		summary.TotalAmount[index] = core.ClampInt32ToZero(summary.TotalAmount[index] + summaryChange.amount)
		summary.TotalDebtAmount[index] = core.ClampInt32ToZero(summary.TotalDebtAmount[index] + summaryChange.totalDebtAmount)
	}
	return nil
}

func appendSummaryRow(summary *types.SaleSummary, productID int32) int {
	// Keep all summary slices aligned by appending the new product row to each one.
	summary.ProductIDs = append(summary.ProductIDs, productID)
	summary.Quantity = append(summary.Quantity, 0)
	summary.QuantityPendingDelivery = append(summary.QuantityPendingDelivery, 0)
	summary.TotalAmount = append(summary.TotalAmount, 0)
	summary.TotalDebtAmount = append(summary.TotalDebtAmount, 0)
	return len(summary.ProductIDs) - 1
}
