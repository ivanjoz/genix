package sales

import (
	"app/core"
	"app/db"
	"app/libs"
	"app/sales/types"
	"slices"
)

type ProductSummaryChange struct {
	productID int32
	types.SaleOrderProductStats
}

// The summary row stores its pairs normalized, which is what keeps the Sub halves — and the
// pending whole-unit count — inside int16. These four accessors are the only places that
// widen or narrow them, so the normalization can never be skipped by accident.

func statsDivisorOf(stats *types.SaleOrderProductStats) int16 {
	if stats.SubDivisor > 0 {
		return stats.SubDivisor
	}
	return core.QuantityDivisorNone
}

func statsSoldQuantity(stats *types.SaleOrderProductStats) core.Quantity {
	return core.Quantity{Units: stats.Quantity, Sub: int32(stats.SubQuantity)}
}

func statsPendingQuantity(stats *types.SaleOrderProductStats) core.Quantity {
	return core.Quantity{
		Units: stats.QuantityPendingDelivery,
		Sub:   int32(stats.SubQuantityPendingDelivery),
	}
}

func setStatsSoldQuantity(stats *types.SaleOrderProductStats, quantity core.Quantity) error {
	normalized, err := quantity.Normalize(statsDivisorOf(stats))
	if err != nil {
		return err
	}
	stats.Quantity = normalized.Units
	stats.SubQuantity = int16(normalized.Sub)
	return nil
}

func setStatsPendingQuantity(stats *types.SaleOrderProductStats, quantity core.Quantity) error {
	normalized, err := quantity.Normalize(statsDivisorOf(stats))
	if err != nil {
		return err
	}
	stats.QuantityPendingDelivery = normalized.Units
	stats.SubQuantityPendingDelivery = int16(normalized.Sub)
	return nil
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
		packedQuantity := sale.DetailQuantities[lineIndex]
		if productID <= 0 || packedQuantity <= 0 {
			continue
		}

		// The order stores lines packed; the summary accumulates, so it works on the split
		// pair. 1004 at divisor 6 is one box plus four candies.
		lineQuantity := core.UnpackQuantityLine(packedQuantity)
		lineDivisor := core.GetIndex(sale.DetailSubDivisor, lineIndex)
		if lineDivisor <= 0 {
			lineDivisor = core.QuantityDivisorNone
		}
		// Sub-units are charged at their own price, so the line total is exact rather than
		// a prorated fraction of the whole-unit price.
		lineAmount := core.QuantityAmount(lineQuantity,
			core.GetIndex(sale.DetailPrices, lineIndex), core.GetIndex(sale.DetailSubPrices, lineIndex))

		currentChange := changesByProduct[productID]
		currentChange.productID = productID
		alignedQuantity, err := alignSummaryChangeDivisor(&currentChange, lineQuantity, lineDivisor)
		if err != nil {
			core.LogError("Resumen de ventas: divisores de sub-unidad incompatibles.", err)
			continue
		}

		soldQuantity := statsSoldQuantity(&currentChange.SaleOrderProductStats)
		pendingQuantity := statsPendingQuantity(&currentChange.SaleOrderProductStats)

		if includeSale {
			// Sale creation adds units and total amount once.
			soldQuantity = soldQuantity.Add(alignedQuantity)
			currentChange.TotalAmount += lineAmount
			// When delivery is still pending, the full quantity remains pending.
			if !includeDelivery {
				pendingQuantity = pendingQuantity.Add(alignedQuantity)
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
			pendingQuantity = pendingQuantity.Add(alignedQuantity.Negate())
		}

		if err := setStatsSoldQuantity(&currentChange.SaleOrderProductStats, soldQuantity); err != nil {
			core.LogError("Resumen de ventas: cantidad vendida fuera de rango.", err)
			continue
		}
		if err := setStatsPendingQuantity(&currentChange.SaleOrderProductStats, pendingQuantity); err != nil {
			core.LogError("Resumen de ventas: cantidad pendiente fuera de rango.", err)
			continue
		}
		changesByProduct[productID] = currentChange
	}

	for _, summaryChange := range changesByProduct {
		changes = append(changes, summaryChange)
	}
	return changes
}

// alignSummaryChangeDivisor brings an accumulating change and an incoming line to one
// divisor before their sub-units are added together. Lines of the same product normally
// share a divisor, but a refinement between two sales of the same day would otherwise have
// the row adding sixths to twelfths.
func alignSummaryChangeDivisor(change *ProductSummaryChange, lineQuantity core.Quantity,
	lineDivisor int16) (core.Quantity, error) {

	changeDivisor := change.SubDivisor
	if changeDivisor <= 0 {
		change.SubDivisor = lineDivisor
		return lineQuantity, nil
	}
	if changeDivisor == lineDivisor {
		return lineQuantity, nil
	}

	// A finer line pulls the accumulator forward; a coarser one converts up to it.
	if core.IsQuantityDivisorRefinement(changeDivisor, lineDivisor) {
		refined, err := statsSoldQuantity(&change.SaleOrderProductStats).
			ConvertToDivisor(changeDivisor, lineDivisor)
		if err != nil {
			return core.Quantity{}, err
		}
		refinedPending, err := statsPendingQuantity(&change.SaleOrderProductStats).
			ConvertToDivisor(changeDivisor, lineDivisor)
		if err != nil {
			return core.Quantity{}, err
		}
		change.SubDivisor = lineDivisor
		if err := setStatsSoldQuantity(&change.SaleOrderProductStats, refined); err != nil {
			return core.Quantity{}, err
		}
		if err := setStatsPendingQuantity(&change.SaleOrderProductStats, refinedPending); err != nil {
			return core.Quantity{}, err
		}
		return lineQuantity, nil
	}
	return lineQuantity.ConvertToDivisor(lineDivisor, changeDivisor)
}

func updateSaleSummaryForChange(sale types.SaleOrder, actions ...int8) error {
	summaryChanges := MakeSummaryChangeFromOSaleOrder(sale, actions...)
	return applyChangesToSaleSumary(sale.CompanyID, sale.Date, summaryChanges, false)
}

// annulSaleSummary takes back everything the sale ever put into its day summary. It rebuilds
// the change set the sale would produce today and negates it, rather than composing a rollback
// by hand: whatever MakeSummaryChangeFromOSaleOrder added is exactly what has to come off, so
// deriving the two from one function is what keeps them from drifting apart.
func annulSaleSummary(sale types.SaleOrder) error {
	summaryChanges := negateSummaryChanges(
		MakeSummaryChangeFromOSaleOrder(sale, summaryActionsOfStatus(sale.Status)...))
	return applyChangesToSaleSumary(sale.CompanyID, sale.Date, summaryChanges, false)
}

// summaryActionsOfStatus reads back the actions a sale's persisted status implies. Status is the
// only surviving record of what happened to it — ActionsIncluded is a per-request field, not a
// history — so this is how an annulment learns whether the sale was ever paid or delivered.
func summaryActionsOfStatus(status int8) []int8 {
	actions := []int8{1}
	if status == types.OrderStatusPaid || status == types.OrderStatusCompleted {
		actions = append(actions, 2)
	}
	if status == types.OrderStatusDelivered || status == types.OrderStatusCompleted {
		actions = append(actions, 3)
	}
	return actions
}

// negateSummaryChanges flips every counter so the change subtracts what it used to add.
// SubDivisor is deliberately untouched: it is the unit the other fields are counted in, not a
// quantity, and negating it would make the change unmergeable with the stored row.
func negateSummaryChanges(changes []ProductSummaryChange) []ProductSummaryChange {
	for index := range changes {
		stats := &changes[index].SaleOrderProductStats
		stats.Quantity = -stats.Quantity
		stats.SubQuantity = -stats.SubQuantity
		stats.QuantityPendingDelivery = -stats.QuantityPendingDelivery
		stats.SubQuantityPendingDelivery = -stats.SubQuantityPendingDelivery
		stats.TotalAmount = -stats.TotalAmount
		stats.TotalDebtAmount = -stats.TotalDebtAmount
	}
	return changes
}

func loadSaleSummaryRowsByProducts(companyID int32, date int16, summaryChanges []ProductSummaryChange) (map[int32]types.ProductSaleSummary, error) {
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
	query.CompanyID.Equals(companyID).Date.Equals(date).ProductID.In(productIDs...)
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

func loadSaleSummaryRowsByDay(companyID int32, date int16) ([]types.ProductSaleSummary, error) {
	summaries := []types.ProductSaleSummary{}
	query := db.Query(&summaries)
	query.CompanyID.Equals(companyID).Date.Equals(date)
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

	// Bring the stored row to the change's divisor before the sub-units are added, so a
	// refinement between two sales of the same day never mixes sixths with twelfths.
	if err := alignStatsToChangeDivisor(summaryStats, summaryChange); err != nil {
		core.LogError("Resumen de ventas: no se pudo alinear el divisor de sub-unidad.", err)
		return
	}

	// Clamp every counter to zero because the compact serializer only stores unsigned values.
	// A pair is clamped as one value: zeroing the halves independently would turn {0,-4}
	// into {0,0} on the sub side while leaving a positive whole count, inventing stock.
	divisor := statsDivisorOf(summaryStats)
	sold := clampQuantityToZero(
		statsSoldQuantity(summaryStats).Add(statsSoldQuantity(&summaryChange.SaleOrderProductStats)), divisor)
	pending := clampQuantityToZero(
		statsPendingQuantity(summaryStats).Add(statsPendingQuantity(&summaryChange.SaleOrderProductStats)), divisor)

	if err := setStatsSoldQuantity(summaryStats, sold); err != nil {
		core.LogError("Resumen de ventas: cantidad vendida fuera de rango.", err)
		return
	}
	if err := setStatsPendingQuantity(summaryStats, pending); err != nil {
		core.LogError("Resumen de ventas: cantidad pendiente fuera de rango.", err)
		return
	}
	summaryStats.TotalAmount = core.ClampInt32ToZero(summaryStats.TotalAmount + summaryChange.TotalAmount)
	summaryStats.TotalDebtAmount = core.ClampInt32ToZero(summaryStats.TotalDebtAmount + summaryChange.TotalDebtAmount)
}

func clampQuantityToZero(quantity core.Quantity, divisor int16) core.Quantity {
	if quantity.IsNegative(divisor) {
		return core.Quantity{}
	}
	return quantity
}

// alignStatsToChangeDivisor refines a stored summary row up to the incoming change's
// divisor. Only that direction is possible: a divisor may never be coarsened, so a change
// arriving at a coarser divisor than the row is a real inconsistency, not a conversion.
func alignStatsToChangeDivisor(summaryStats *types.SaleOrderProductStats, summaryChange ProductSummaryChange) error {
	changeDivisor := summaryChange.SubDivisor
	storedDivisor := summaryStats.SubDivisor
	if changeDivisor <= 0 {
		return nil
	}
	if storedDivisor <= 0 {
		summaryStats.SubDivisor = summaryChange.SubDivisor
		return nil
	}
	if storedDivisor == changeDivisor {
		return nil
	}

	sold, err := statsSoldQuantity(summaryStats).ConvertToDivisor(storedDivisor, changeDivisor)
	if err != nil {
		return err
	}
	pending, err := statsPendingQuantity(summaryStats).ConvertToDivisor(storedDivisor, changeDivisor)
	if err != nil {
		return err
	}

	summaryStats.SubDivisor = summaryChange.SubDivisor
	if err := setStatsSoldQuantity(summaryStats, sold); err != nil {
		return err
	}
	return setStatsPendingQuantity(summaryStats, pending)
}

func applyChangesToSaleSumary(companyID int32, date int16, changes []ProductSummaryChange, replaceCurrentValues bool) error {
	if companyID <= 0 || date <= 0 {
		return core.Err("invalid sale summary scope")
	}
	if len(changes) == 0 {
		core.Log("applyChangesToSaleSumary skipped: no changes", "companyID", companyID, "date", date, "replaceCurrentValues", replaceCurrentValues)
		return nil
	}

	mergedChanges := map[int32]ProductSummaryChange{}
	for _, summaryChange := range changes {
		if summaryChange.productID <= 0 {
			continue
		}
		currentChange := mergedChanges[summaryChange.productID]
		currentChange.productID = summaryChange.productID
		// Align first, then add through the accessors so the result is normalized. Adding the
		// raw halves would let Sub climb past its field on a product with many changes in
		// one day — a few hundred lines at ~999 sub-units each is enough.
		if err := alignStatsToChangeDivisor(&currentChange.SaleOrderProductStats, summaryChange); err != nil {
			core.LogError("Resumen de ventas: no se pudo alinear el divisor al fusionar cambios.", err)
			continue
		}
		mergedSold := statsSoldQuantity(&currentChange.SaleOrderProductStats).
			Add(statsSoldQuantity(&summaryChange.SaleOrderProductStats))
		mergedPending := statsPendingQuantity(&currentChange.SaleOrderProductStats).
			Add(statsPendingQuantity(&summaryChange.SaleOrderProductStats))
		// No clamping here: a merge of changes is signed, and the floor is applied when the
		// change lands on the stored row.
		if err := setStatsSoldQuantity(&currentChange.SaleOrderProductStats, mergedSold); err != nil {
			core.LogError("Resumen de ventas: cantidad vendida fuera de rango al fusionar.", err)
			continue
		}
		if err := setStatsPendingQuantity(&currentChange.SaleOrderProductStats, mergedPending); err != nil {
			core.LogError("Resumen de ventas: cantidad pendiente fuera de rango al fusionar.", err)
			continue
		}
		currentChange.TotalAmount += summaryChange.TotalAmount
		currentChange.TotalDebtAmount += summaryChange.TotalDebtAmount
		mergedChanges[summaryChange.productID] = currentChange
	}
	if len(mergedChanges) == 0 {
		core.Log("applyChangesToSaleSumary skipped: merged changes empty", "companyID", companyID, "date", date, "replaceCurrentValues", replaceCurrentValues)
		return nil
	}

	mergedChangesList := make([]ProductSummaryChange, 0, len(mergedChanges))
	for _, summaryChange := range mergedChanges {
		mergedChangesList = append(mergedChangesList, summaryChange)
	}

	summaryByProductID, err := loadSaleSummaryRowsByProducts(companyID, date, mergedChangesList)
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
				Date:      date,
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

	// Persist one sentinel row per date so clients can query date=-1 and discover which day buckets changed.
	summaryRows = append(summaryRows, types.ProductSaleSummary{
		CompanyID: companyID,
		Date:      -1,
		ProductID: int32(date),
		Updated:   summaryUpdated,
	})

	core.Log("applyChangesToSaleSumary saving rows", "companyID", companyID, "date", date, "rows", len(summaryRows), "replaceCurrentValues", replaceCurrentValues)
	if err := db.Insert(&summaryRows); err != nil {
		return core.Err("error saving sale summary rows:", err)
	}
	return nil
}

func makeSaleSummaryStatsFromChange(summaryChange ProductSummaryChange) types.SaleOrderProductStats {
	// Rebuild rows use the change struct as the final summary payload for the product.
	divisor := statsDivisorOf(&summaryChange.SaleOrderProductStats)
	rebuiltStats := types.SaleOrderProductStats{
		SubDivisor:      divisor,
		TotalAmount:     core.ClampInt32ToZero(summaryChange.TotalAmount),
		TotalDebtAmount: core.ClampInt32ToZero(summaryChange.TotalDebtAmount),
	}
	// Clamped as whole values, so a negative pair zeroes both halves together.
	sold := clampQuantityToZero(statsSoldQuantity(&summaryChange.SaleOrderProductStats), divisor)
	pending := clampQuantityToZero(statsPendingQuantity(&summaryChange.SaleOrderProductStats), divisor)
	if err := setStatsSoldQuantity(&rebuiltStats, sold); err != nil {
		core.LogError("Resumen de ventas: cantidad vendida fuera de rango al reconstruir.", err)
	}
	if err := setStatsPendingQuantity(&rebuiltStats, pending); err != nil {
		core.LogError("Resumen de ventas: cantidad pendiente fuera de rango al reconstruir.", err)
	}
	return rebuiltStats
}
