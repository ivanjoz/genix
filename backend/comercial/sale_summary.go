package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"math"
)

type ProductSummaryChange struct {
	productID, quantity, quantityPendingDelivery, amount, totalDebtAmount int32
}

func UpdateProductOnSumary(summary *types.SaleSummary, summaryChanges ...ProductSummaryChange) error {
	if summary == nil || len(summaryChanges) == 0 {
		return nil
	}

	// Index existing int32 rows so small deltas for already-promoted products stay on int32.
	existingProductsInt32 := map[int32]int{}
	for index, productID := range summary.ProductIDs_32 {
		if productID > 0 {
			existingProductsInt32[productID] = index
		}
	}

	// Create 2 maps: pending deltas for int16 and int32 storage.
	productosInt16 := map[uint16]ProductSummaryChange{}
	productosInt32 := map[int32]ProductSummaryChange{}

	for _, summaryChange := range summaryChanges {
		if summaryChange.productID <= 0 {
			continue
		}

		mustUseInt32 := summaryChange.productID > math.MaxUint16 ||
			absInt32(summaryChange.quantity) > math.MaxUint16 ||
			absInt32(summaryChange.quantityPendingDelivery) > math.MaxUint16
		if !mustUseInt32 {
			_, existsInInt32 := existingProductsInt32[summaryChange.productID]
			mustUseInt32 = existsInInt32
		}

		if mustUseInt32 {
			if _, exists := productosInt32[summaryChange.productID]; exists {
				return core.Err("duplicated product in summary changes:", summaryChange.productID)
			}
			productosInt32[summaryChange.productID] = summaryChange
			continue
		}

		productID16 := uint16(summaryChange.productID)
		if _, exists := productosInt16[productID16]; exists {
			return core.Err("duplicated product in summary changes:", summaryChange.productID)
		}
		productosInt16[productID16] = summaryChange
	}

	// Update existing int16 rows directly by index and promote overflows to int32 pending map.
	for index, productID16 := range summary.ProductIDs_16 {
		if productID16 == 0 {
			continue
		}

		summaryChange, exists := productosInt16[productID16]
		if !exists {
			continue
		}

		nextQuantity16 := int32(summary.Quantity_16[index]) + summaryChange.quantity
		nextQuantityPendingDelivery16 := int32(summary.QuantityPendingDelivery_16[index]) + summaryChange.quantityPendingDelivery
		nextAmount16 := summary.TotalAmount_16[index] + summaryChange.amount
		nextTotalDebtAmount16 := summary.TotalDebtAmount_16[index] + summaryChange.totalDebtAmount

		if nextQuantity16 > math.MaxUint16 || nextQuantityPendingDelivery16 > math.MaxUint16 {
			promotedChange, existsPromoted := productosInt32[int32(productID16)]
			if existsPromoted {
				return core.Err("duplicated promoted product in summary changes:", int32(productID16))
			}
			promotedChange.productID = int32(productID16)
			promotedChange.quantity += nextQuantity16
			promotedChange.quantityPendingDelivery += nextQuantityPendingDelivery16
			promotedChange.amount += nextAmount16
			promotedChange.totalDebtAmount += nextTotalDebtAmount16
			productosInt32[int32(productID16)] = promotedChange

			summary.ProductIDs_16[index] = 0
			summary.Quantity_16[index] = 0
			summary.QuantityPendingDelivery_16[index] = 0
			summary.TotalAmount_16[index] = 0
			summary.TotalDebtAmount_16[index] = 0
			delete(productosInt16, productID16)
			continue
		}

		summary.Quantity_16[index] = uint16(maxZero(nextQuantity16))
		summary.QuantityPendingDelivery_16[index] = uint16(maxZero(nextQuantityPendingDelivery16))
		summary.TotalAmount_16[index] = maxZero(nextAmount16)
		summary.TotalDebtAmount_16[index] = maxZero(nextTotalDebtAmount16)
		delete(productosInt16, productID16)
	}

	// Append remaining int16 products that did not exist before.
	for productID16, summaryChange := range productosInt16 {
		quantity := maxZero(summaryChange.quantity)
		quantityPendingDelivery := maxZero(summaryChange.quantityPendingDelivery)
		amount := maxZero(summaryChange.amount)
		totalDebtAmount := maxZero(summaryChange.totalDebtAmount)
		if quantity == 0 && quantityPendingDelivery == 0 && amount == 0 && totalDebtAmount == 0 {
			continue
		}

		newIndex := appendSummaryRow16(summary, productID16)
		summary.Quantity_16[newIndex] = uint16(quantity)
		summary.QuantityPendingDelivery_16[newIndex] = uint16(quantityPendingDelivery)
		summary.TotalAmount_16[newIndex] = amount
		summary.TotalDebtAmount_16[newIndex] = totalDebtAmount
	}

	// Update existing int32 rows directly by index.
	for index, productID32 := range summary.ProductIDs_32 {
		if productID32 == 0 {
			continue
		}

		summaryChange, exists := productosInt32[productID32]
		if !exists {
			continue
		}

		summary.Quantity_32[index] = maxZero(summary.Quantity_32[index] + summaryChange.quantity)
		summary.QuantityPendingDelivery_32[index] = maxZero(summary.QuantityPendingDelivery_32[index] + summaryChange.quantityPendingDelivery)
		summary.TotalAmount_32[index] = maxZero(summary.TotalAmount_32[index] + summaryChange.amount)
		summary.TotalDebtAmount_32[index] = maxZero(summary.TotalDebtAmount_32[index] + summaryChange.totalDebtAmount)
		delete(productosInt32, productID32)
	}

	// Append remaining int32 products that did not exist before.
	for productID32, summaryChange := range productosInt32 {
		quantity := maxZero(summaryChange.quantity)
		quantityPendingDelivery := maxZero(summaryChange.quantityPendingDelivery)
		amount := maxZero(summaryChange.amount)
		totalDebtAmount := maxZero(summaryChange.totalDebtAmount)
		if quantity == 0 && quantityPendingDelivery == 0 && amount == 0 && totalDebtAmount == 0 {
			continue
		}

		newIndex := appendSummaryRow32(summary, productID32)
		summary.Quantity_32[newIndex] = quantity
		summary.QuantityPendingDelivery_32[newIndex] = quantityPendingDelivery
		summary.TotalAmount_32[newIndex] = amount
		summary.TotalDebtAmount_32[newIndex] = totalDebtAmount
	}
	return nil
}

func UpdateSaleSumary(summary *types.SaleSummary, newSale types.SaleOrder, prevSale *types.SaleOrder) error {
	if summary == nil {
		return core.Err("sale summary is nil")
	}

	core.Log("UpdateSaleSumary start. EmpresaID:", summary.EmpresaID, " Fecha:", summary.Fecha)

	changesByProduct := map[int32]ProductSummaryChange{}
	if prevSale != nil {
		if err := collectSaleChanges(changesByProduct, *prevSale, -1); err != nil {
			return err
		}
	}
	if len(newSale.DetailProductsIDs) > 0 {
		if err := collectSaleChanges(changesByProduct, newSale, 1); err != nil {
			return err
		}
	}

	allChanges := make([]ProductSummaryChange, 0, len(changesByProduct))
	for _, summaryChange := range changesByProduct {
		allChanges = append(allChanges, summaryChange)
	}
	if len(allChanges) > 0 {
		if err := UpdateProductOnSumary(summary, allChanges...); err != nil {
			return err
		}
	}

	removeEmptySummaryRows(summary)
	summary.Updated = core.SUnixTime()
	if err := db.InsertOne(*summary); err != nil {
		return core.Err("error saving sale summary:", err)
	}
	core.Log("UpdateSaleSumary done. products16:", len(summary.ProductIDs_16), " products32:", len(summary.ProductIDs_32))
	return nil
}

func updateSaleSummaryForChange(newSale types.SaleOrder, prevSale *types.SaleOrder) error {
	if prevSale != nil && (prevSale.EmpresaID != newSale.EmpresaID || prevSale.Fecha != newSale.Fecha) {
		return core.Err("sale summary key mismatch: EmpresaID/Fecha cannot change for the same sale")
	}

	previousSummary, err := loadSaleSummary(newSale.EmpresaID, newSale.Fecha)
	if err != nil {
		return err
	}
	return UpdateSaleSumary(&previousSummary, newSale, prevSale)
}

func loadSaleSummary(empresaID int32, fecha int16) (types.SaleSummary, error) {
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

func collectSaleChanges(changesByProduct map[int32]ProductSummaryChange, sale types.SaleOrder, sign int32) error {
	if len(sale.DetailProductsIDs) == 0 {
		return nil
	}
	if len(sale.DetailProductsIDs) != len(sale.DetailQuantities) || len(sale.DetailProductsIDs) != len(sale.DetailPrices) {
		return core.Err("sale detail slices have inconsistent lengths")
	}

	debtTotal := sale.DebtAmount
	if debtTotal < 0 {
		debtTotal = 0
	}

	// Pending delivery is the full quantity unless the sale is already delivered.
	pendingDeliveryMultiplier := int32(1)
	if sale.Status == 3 || sale.Status == 4 {
		pendingDeliveryMultiplier = 0
	}

	// Split total debt across lines proportionally to line amount.
	allocatedDebt := int32(0)
	lastLineIndex := len(sale.DetailProductsIDs) - 1
	for lineIndex, productID := range sale.DetailProductsIDs {
		quantity := sale.DetailQuantities[lineIndex]
		if productID <= 0 || quantity <= 0 {
			continue
		}

		lineAmount := multiplyInt32(sale.DetailPrices[lineIndex], quantity)
		lineDebtAmount := int32(0)
		if sale.TotalAmount > 0 && debtTotal > 0 {
			if lineIndex == lastLineIndex {
				lineDebtAmount = maxZero(debtTotal - allocatedDebt)
			} else {
				lineDebtAmount = maxZero(int32((int64(lineAmount) * int64(debtTotal)) / int64(sale.TotalAmount)))
				allocatedDebt += lineDebtAmount
			}
		}

		currentSummaryChange := changesByProduct[productID]
		currentSummaryChange.productID = productID
		currentSummaryChange.quantity += sign * quantity
		currentSummaryChange.quantityPendingDelivery += sign * (quantity * pendingDeliveryMultiplier)
		currentSummaryChange.amount += sign * lineAmount
		currentSummaryChange.totalDebtAmount += sign * lineDebtAmount
		changesByProduct[productID] = currentSummaryChange
	}
	return nil
}

func removeEmptySummaryRows(summary *types.SaleSummary) {
	// Compact sparse slices after decrements/promotions to avoid storing dead rows.
	productIDs16 := make([]uint16, 0, len(summary.ProductIDs_16))
	quantities16 := make([]uint16, 0, len(summary.Quantity_16))
	quantitiesPendingDelivery16 := make([]uint16, 0, len(summary.QuantityPendingDelivery_16))
	totalAmounts16 := make([]int32, 0, len(summary.TotalAmount_16))
	totalDebtAmounts16 := make([]int32, 0, len(summary.TotalDebtAmount_16))

	for index, productID := range summary.ProductIDs_16 {
		if productID == 0 {
			continue
		}
		if summary.Quantity_16[index] == 0 && summary.QuantityPendingDelivery_16[index] == 0 && summary.TotalAmount_16[index] == 0 && summary.TotalDebtAmount_16[index] == 0 {
			continue
		}
		productIDs16 = append(productIDs16, productID)
		quantities16 = append(quantities16, summary.Quantity_16[index])
		quantitiesPendingDelivery16 = append(quantitiesPendingDelivery16, summary.QuantityPendingDelivery_16[index])
		totalAmounts16 = append(totalAmounts16, summary.TotalAmount_16[index])
		totalDebtAmounts16 = append(totalDebtAmounts16, summary.TotalDebtAmount_16[index])
	}

	summary.ProductIDs_16 = productIDs16
	summary.Quantity_16 = quantities16
	summary.QuantityPendingDelivery_16 = quantitiesPendingDelivery16
	summary.TotalAmount_16 = totalAmounts16
	summary.TotalDebtAmount_16 = totalDebtAmounts16

	productIDs32 := make([]int32, 0, len(summary.ProductIDs_32))
	quantities32 := make([]int32, 0, len(summary.Quantity_32))
	quantitiesPendingDelivery32 := make([]int32, 0, len(summary.QuantityPendingDelivery_32))
	totalAmounts32 := make([]int32, 0, len(summary.TotalAmount_32))
	totalDebtAmounts32 := make([]int32, 0, len(summary.TotalDebtAmount_32))

	for index, productID := range summary.ProductIDs_32 {
		if productID == 0 {
			continue
		}
		if summary.Quantity_32[index] == 0 && summary.QuantityPendingDelivery_32[index] == 0 && summary.TotalAmount_32[index] == 0 && summary.TotalDebtAmount_32[index] == 0 {
			continue
		}
		productIDs32 = append(productIDs32, productID)
		quantities32 = append(quantities32, summary.Quantity_32[index])
		quantitiesPendingDelivery32 = append(quantitiesPendingDelivery32, summary.QuantityPendingDelivery_32[index])
		totalAmounts32 = append(totalAmounts32, summary.TotalAmount_32[index])
		totalDebtAmounts32 = append(totalDebtAmounts32, summary.TotalDebtAmount_32[index])
	}

	summary.ProductIDs_32 = productIDs32
	summary.Quantity_32 = quantities32
	summary.QuantityPendingDelivery_32 = quantitiesPendingDelivery32
	summary.TotalAmount_32 = totalAmounts32
	summary.TotalDebtAmount_32 = totalDebtAmounts32
}

func appendSummaryRow16(summary *types.SaleSummary, productID uint16) int {
	summary.ProductIDs_16 = append(summary.ProductIDs_16, productID)
	summary.Quantity_16 = append(summary.Quantity_16, 0)
	summary.QuantityPendingDelivery_16 = append(summary.QuantityPendingDelivery_16, 0)
	summary.TotalAmount_16 = append(summary.TotalAmount_16, 0)
	summary.TotalDebtAmount_16 = append(summary.TotalDebtAmount_16, 0)
	return len(summary.ProductIDs_16) - 1
}

func appendSummaryRow32(summary *types.SaleSummary, productID int32) int {
	summary.ProductIDs_32 = append(summary.ProductIDs_32, productID)
	summary.Quantity_32 = append(summary.Quantity_32, 0)
	summary.QuantityPendingDelivery_32 = append(summary.QuantityPendingDelivery_32, 0)
	summary.TotalAmount_32 = append(summary.TotalAmount_32, 0)
	summary.TotalDebtAmount_32 = append(summary.TotalDebtAmount_32, 0)
	return len(summary.ProductIDs_32) - 1
}

func absInt32(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}

func maxZero(value int32) int32 {
	if value < 0 {
		return 0
	}
	return value
}

func multiplyInt32(valueA int32, valueB int32) int32 {
	result64 := int64(valueA) * int64(valueB)
	if result64 > math.MaxInt32 {
		return math.MaxInt32
	}
	if result64 < math.MinInt32 {
		return math.MinInt32
	}
	return int32(result64)
}
