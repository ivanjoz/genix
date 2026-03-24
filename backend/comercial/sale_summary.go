package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
	"app/libs"
	"slices"
)

type ProductSummaryChange struct {
	productID int32
	types.SaleOrderProductStats
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
			currentChange.Quantity += quantity
			currentChange.TotalAmount += lineAmount
			// When delivery is still pending, the full quantity remains pending.
			if !includeDelivery {
				currentChange.QuantityPendingDelivery += quantity
			}
			// When payment is still pending, the full line amount remains as debt.
			if !includePayment {
				currentChange.TotalDebtAmount += lineAmount
			}
		}
		if includePayment && !includeSale {
			// Pure payment updates only reduce pending debt.
			currentChange.TotalDebtAmount -= lineAmount
		}
		if includeDelivery && !includeSale {
			// Pure delivery updates only reduce pending quantity.
			currentChange.QuantityPendingDelivery -= quantity
		}
		changesByProduct[productID] = currentChange
	}

	for _, summaryChange := range changesByProduct {
		changes = append(changes, summaryChange)
	}
	return changes
}

func updateSaleSummaryForChange(sale types.SaleOrder, actions ...int8) error {
	summaryChanges := MakeSummaryChangeFromOSaleOrder(sale, actions...)
	return applyChangesToSaleSumary(sale.EmpresaID, sale.Fecha, summaryChanges, false)
}

func loadSaleSummaryRowsByProducts(companyID int32, fecha int16, summaryChanges []ProductSummaryChange) (map[int32]types.ProductSaleSummary, error) {
	productIDs := make([]int32, 0, len(summaryChanges))
	seenProducts := map[int32]struct{}{}

	// Build the target product list once so the query only fetches touched rows.
	for _, summaryChange := range summaryChanges {
		if summaryChange.productID <= 0 {
			continue
		}
		if _, exists := seenProducts[summaryChange.productID]; exists {
			continue
		}
		seenProducts[summaryChange.productID] = struct{}{}
		productIDs = append(productIDs, summaryChange.productID)
	}

	if len(productIDs) == 0 {
		return map[int32]types.ProductSaleSummary{}, nil
	}

	summaries := []types.ProductSaleSummary{}
	query := db.Query(&summaries)
	query.CompanyID.Equals(companyID).Fecha.Equals(fecha).ProductID.In(productIDs...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("error querying sale summary rows:", err)
	}

	summaryByProductID := make(map[int32]types.ProductSaleSummary, len(summaries))
	for _, summary := range summaries {
		if summary.ProductID > 0 {
			summaryByProductID[summary.ProductID] = summary
		}
	}
	return summaryByProductID, nil
}

func loadSaleSummaryRowsByDay(companyID int32, fecha int16) ([]types.ProductSaleSummary, error) {
	summaries := []types.ProductSaleSummary{}
	query := db.Query(&summaries)
	query.CompanyID.Equals(companyID).Fecha.Equals(fecha)
	if err := query.Exec(); err != nil {
		return nil, core.Err("error querying daily sale summary rows:", err)
	}
	return summaries, nil
}

func encodeSaleSummaryStats(summaryStats types.SaleOrderProductStats) []byte {
	// The summary blob must stay in the compact int30 format shared by incremental and rebuild flows.
	return libs.SerializeInt30Struct(summaryStats)
}

func decodeSaleSummaryStats(encodedStats []byte) (types.SaleOrderProductStats, error) {
	summaryStats := types.SaleOrderProductStats{}
	if len(encodedStats) == 0 {
		return summaryStats, nil
	}

	// Empty rows decode to zero-value stats, while malformed payloads fail fast for diagnosis.
	if err := libs.DeserializeInt30Struct(encodedStats, &summaryStats); err != nil {
		return types.SaleOrderProductStats{}, err
	}
	return summaryStats, nil
}

func applySummaryChangeToStats(summaryStats *types.SaleOrderProductStats, summaryChange ProductSummaryChange) {
	if summaryStats == nil {
		return
	}

	// Clamp every counter to zero because the compact serializer only stores unsigned values.
	summaryStats.Quantity = core.ClampInt32ToZero(summaryStats.Quantity + summaryChange.Quantity)
	summaryStats.QuantityPendingDelivery = core.ClampInt32ToZero(summaryStats.QuantityPendingDelivery + summaryChange.QuantityPendingDelivery)
	summaryStats.TotalAmount = core.ClampInt32ToZero(summaryStats.TotalAmount + summaryChange.TotalAmount)
	summaryStats.TotalDebtAmount = core.ClampInt32ToZero(summaryStats.TotalDebtAmount + summaryChange.TotalDebtAmount)
}

func applyChangesToSaleSumary(companyID int32, fecha int16, changes []ProductSummaryChange, replaceCurrentValues bool) error {
	if companyID <= 0 || fecha <= 0 {
		return core.Err("invalid sale summary scope")
	}
	if len(changes) == 0 {
		core.Log("applyChangesToSaleSumary skipped: no changes", "companyID", companyID, "fecha", fecha, "replaceCurrentValues", replaceCurrentValues)
		return nil
	}

	mergedChanges := map[int32]ProductSummaryChange{}
	for _, summaryChange := range changes {
		if summaryChange.productID <= 0 {
			continue
		}
		currentChange := mergedChanges[summaryChange.productID]
		currentChange.productID = summaryChange.productID
		currentChange.Quantity += summaryChange.Quantity
		currentChange.QuantityPendingDelivery += summaryChange.QuantityPendingDelivery
		currentChange.TotalAmount += summaryChange.TotalAmount
		currentChange.TotalDebtAmount += summaryChange.TotalDebtAmount
		mergedChanges[summaryChange.productID] = currentChange
	}
	if len(mergedChanges) == 0 {
		core.Log("applyChangesToSaleSumary skipped: merged changes empty", "companyID", companyID, "fecha", fecha, "replaceCurrentValues", replaceCurrentValues)
		return nil
	}

	mergedChangesList := make([]ProductSummaryChange, 0, len(mergedChanges))
	for _, summaryChange := range mergedChanges {
		mergedChangesList = append(mergedChangesList, summaryChange)
	}

	summaryByProductID, err := loadSaleSummaryRowsByProducts(companyID, fecha, mergedChangesList)
	if err != nil {
		return err
	}

	summaryRows := make([]types.ProductSaleSummary, 0, len(mergedChangesList))
	summaryUpdated := core.SUnixTime()

	for _, summaryChange := range mergedChangesList {
		currentRow, exists := summaryByProductID[summaryChange.productID]
		if !exists {
			currentRow = types.ProductSaleSummary{
				CompanyID: companyID,
				Fecha:     fecha,
				ProductID: summaryChange.productID,
			}
		}

		var nextStats types.SaleOrderProductStats
		if replaceCurrentValues {
			// Reprocess sends final product totals, so the stored payload is replaced instead of adjusted.
			nextStats = makeSaleSummaryStatsFromChange(summaryChange)
			if exists {
				currentStats, err := decodeSaleSummaryStats(currentRow.Stats)
				if err != nil {
					return core.Err("error decoding sale summary stats:", err)
				}
				if currentStats == nextStats {
					// Reprocess can skip rows that already have the target totals.
					continue
				}
			}
		} else {
			// Incremental updates reuse current counters and only apply the requested deltas.
			nextStats, err = decodeSaleSummaryStats(currentRow.Stats)
			if err != nil {
				return core.Err("error decoding sale summary stats:", err)
			}
			applySummaryChangeToStats(&nextStats, summaryChange)
		}

		currentRow.Stats = encodeSaleSummaryStats(nextStats)
		currentRow.Updated = summaryUpdated
		summaryRows = append(summaryRows, currentRow)
	}

	// Persist one sentinel row per fecha so clients can query fecha=-1 and discover which day buckets changed.
	summaryRows = append(summaryRows, types.ProductSaleSummary{
		CompanyID: companyID,
		Fecha:     -1,
		ProductID: int32(fecha),
		Updated:   summaryUpdated,
	})

	core.Log("applyChangesToSaleSumary saving rows", "companyID", companyID, "fecha", fecha, "rows", len(summaryRows), "replaceCurrentValues", replaceCurrentValues)
	if err := db.Insert(&summaryRows); err != nil {
		return core.Err("error saving sale summary rows:", err)
	}
	return nil
}

func makeSaleSummaryStatsFromChange(summaryChange ProductSummaryChange) types.SaleOrderProductStats {
	// Rebuild rows use the change struct as the final summary payload for the product.
	return types.SaleOrderProductStats{
		Quantity:                core.ClampInt32ToZero(summaryChange.Quantity),
		QuantityPendingDelivery: core.ClampInt32ToZero(summaryChange.QuantityPendingDelivery),
		TotalAmount:             core.ClampInt32ToZero(summaryChange.TotalAmount),
		TotalDebtAmount:         core.ClampInt32ToZero(summaryChange.TotalDebtAmount),
	}
}
