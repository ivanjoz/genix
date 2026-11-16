package types

import (
	"app/core"
	production "app/production/types"
	"slices"
)

// The pack pricing engine (docs/PACKS_PLAN.md §3): it finds the company's packs in a sale's lines
// and reprices the lines that fill them, so they sum to the pack price.

// A pack's relevance is the sum of its slots' weights: a slot naming one product is a deliberate
// offer and outranks a slot open to a whole category. "Ceviche + [cat] Entradas" (7) beats
// "[cat] Entradas + [cat] Fondos + [cat] Postres" (6) when an order fills both.
const (
	PackRelevanceProductSlot  = int32(5)
	PackRelevanceCategorySlot = int32(2)
)

// PackRelevance is the weight that decides between packs competing for the same units.
func PackRelevance(pack production.ProductPack) int32 {
	relevance := int32(0)
	for _, component := range pack.Components {
		relevance += core.If(component.ProductID > 0, PackRelevanceProductSlot, PackRelevanceCategorySlot)
	}
	return relevance
}

// PackPrice is what the filled units cost as a pack (§3.1), from catalogSum, the sum of their unit
// prices. A fixed price ignores the rounding: it was chosen as it is.
func PackPrice(pack production.ProductPack, catalogSum int32) int32 {
	packPrice := catalogSum
	switch pack.PriceType {
	case production.PackPriceFixed:
		return pack.PriceValue
	case production.PackPricePercentOff:
		packPrice = int32(int64(catalogSum) * int64(10000-pack.PriceValue) / 10000)
	case production.PackPriceAmountOff:
		packPrice = catalogSum - pack.PriceValue
	}
	// Rounding is to the whole sol: the prices are in cents.
	switch pack.RoundingMode {
	case production.PackRoundingUp:
		packPrice = (packPrice + 99) / 100 * 100
	case production.PackRoundingNearest:
		packPrice = (packPrice + 50) / 100 * 100
	case production.PackRoundingDown:
		packPrice = packPrice / 100 * 100
	}
	return packPrice
}

// packUnit is one whole unit of a sale line, the grain packs are filled with.
type packUnit struct {
	lineIndex int
	productID int32
	unitPrice int32
	// packID and packedPrice are set once a pack takes the unit; applicationIndex orders the
	// split lines by the pack application that priced them.
	packID           int32
	packedPrice      int32
	applicationIndex int
}

// packFill is one pack filled from the free units: the indexes into the unit list it would take.
type packFill struct {
	pack        production.ProductPack
	relevance   int32
	unitIndexes []int
	catalogSum  int32
	packPrice   int32
}

// ApplySalePacks reprices the lines that fill the active packs. Lines come priced at catalog
// (PriceSaleOrderLine); only their whole units take part. A line whose units end up at different
// prices is split: one line per (pack, prorated price), in the order the packs were applied, then
// the line's unpacked units and sub-units at catalog price. productByID needs the lines' products
// with their CategoryIDs.
//
// Matching is greedy and deterministic: while some pack can be filled with a positive saving, apply
// the one with the highest relevance (PackRelevance); a tie goes to the larger saving, then to the
// lower pack ID. A pack applies as many times as the order fills it.
func ApplySalePacks(lines []SaleOrderLine, productByID map[int32]production.Product, packs []production.ProductPack) []SaleOrderLine {
	activePacks := slices.DeleteFunc(slices.Clone(packs), func(pack production.ProductPack) bool { return pack.Status != 1 })
	if len(activePacks) == 0 {
		return lines
	}

	units := []packUnit{}
	for lineIndex, line := range lines {
		for range core.UnpackQuantityLine(line.Quantity).Units {
			units = append(units, packUnit{lineIndex: lineIndex, productID: line.ProductID, unitPrice: line.Price})
		}
	}
	// A slot takes the most expensive eligible unit first: the largest saving for the customer.
	slices.SortStableFunc(units, func(left, right packUnit) int { return int(right.unitPrice - left.unitPrice) })

	for applicationIndex := 1; ; applicationIndex++ {
		var bestFill *packFill
		for _, pack := range activePacks {
			fill, isFilled := fillPack(pack, units, productByID)
			if !isFilled || fill.packPrice <= 0 || fill.packPrice >= fill.catalogSum {
				continue
			}
			if bestFill == nil || isBetterPackFill(fill, *bestFill) {
				bestFill = &fill
			}
		}
		if bestFill == nil {
			break
		}
		prorateFillPrice(*bestFill, units, applicationIndex)
	}
	return splitPackedLines(lines, units)
}

func isBetterPackFill(candidate packFill, current packFill) bool {
	if candidate.relevance != current.relevance {
		return candidate.relevance > current.relevance
	}
	candidateSaving, currentSaving := candidate.catalogSum-candidate.packPrice, current.catalogSum-current.packPrice
	if candidateSaving != currentSaving {
		return candidateSaving > currentSaving
	}
	return candidate.pack.ID < current.pack.ID
}

// fillPack takes the free units a pack needs, product slots before category slots: a slot naming the
// product must not lose its unit to a slot that would take any product of a category.
func fillPack(pack production.ProductPack, units []packUnit, productByID map[int32]production.Product) (packFill, bool) {
	fill := packFill{pack: pack, relevance: PackRelevance(pack)}
	takenUnits := map[int]bool{}
	takeUnits := func(component production.ProductPackComponent) bool {
		missingUnits := int(component.Quantity)
		for unitIndex, unit := range units {
			if missingUnits == 0 {
				break
			}
			if unit.packID > 0 || takenUnits[unitIndex] {
				continue
			}
			isEligible := unit.productID == component.ProductID
			if component.CategoryID > 0 {
				isEligible = slices.Contains(productByID[unit.productID].CategoryIDs, component.CategoryID)
			}
			if isEligible {
				takenUnits[unitIndex] = true
				fill.unitIndexes = append(fill.unitIndexes, unitIndex)
				fill.catalogSum += unit.unitPrice
				missingUnits--
			}
		}
		return missingUnits == 0
	}
	for _, isProductPass := range []bool{true, false} {
		for _, component := range pack.Components {
			if (component.ProductID > 0) == isProductPass && !takeUnits(component) {
				return fill, false
			}
		}
	}
	fill.packPrice = PackPrice(pack, fill.catalogSum)
	return fill, true
}

// prorateFillPrice spreads the pack price over its units in proportion to their catalog price
// (§3.2). The cents the floors leave go to the most expensive unit, so the units sum to the pack
// price exactly.
func prorateFillPrice(fill packFill, units []packUnit, applicationIndex int) {
	assignedTotal, mostExpensiveIndex := int32(0), fill.unitIndexes[0]
	for _, unitIndex := range fill.unitIndexes {
		unit := &units[unitIndex]
		unit.packID, unit.applicationIndex = fill.pack.ID, applicationIndex
		unit.packedPrice = int32(int64(unit.unitPrice) * int64(fill.packPrice) / int64(fill.catalogSum))
		assignedTotal += unit.packedPrice
		if unit.unitPrice > units[mostExpensiveIndex].unitPrice {
			mostExpensiveIndex = unitIndex
		}
	}
	units[mostExpensiveIndex].packedPrice += fill.packPrice - assignedTotal
}

// splitPackedLines turns the priced units back into lines (§3.3), keeping the lines' order.
func splitPackedLines(lines []SaleOrderLine, units []packUnit) []SaleOrderLine {
	unitsByLine := make([][]packUnit, len(lines))
	for _, unit := range units {
		if unit.packID > 0 {
			unitsByLine[unit.lineIndex] = append(unitsByLine[unit.lineIndex], unit)
		}
	}
	packedLines := make([]SaleOrderLine, 0, len(lines))
	for lineIndex, line := range lines {
		lineUnits := unitsByLine[lineIndex]
		if len(lineUnits) == 0 {
			packedLines = append(packedLines, line)
			continue
		}
		slices.SortStableFunc(lineUnits, func(left, right packUnit) int { return left.applicationIndex - right.applicationIndex })
		firstSplitIndex := len(packedLines)
		for _, unit := range lineUnits {
			// Units of this line taken by the same pack at the same price stay on one line.
			if lastIndex := len(packedLines) - 1; lastIndex >= firstSplitIndex &&
				packedLines[lastIndex].PackID == unit.packID && packedLines[lastIndex].Price == unit.packedPrice {
				packedLines[lastIndex].Quantity += core.QuantityLineScale
				continue
			}
			packedLine := line
			packedLine.Quantity, packedLine.Price, packedLine.PackID = core.QuantityLineScale, unit.packedPrice, unit.packID
			packedLines = append(packedLines, packedLine)
		}
		// What no pack took stays at catalog price: the leftover whole units and the sub-units.
		if remainderQuantity := line.Quantity - int32(len(lineUnits))*core.QuantityLineScale; remainderQuantity > 0 {
			remainderLine := line
			remainderLine.Quantity = remainderQuantity
			packedLines = append(packedLines, remainderLine)
		}
	}
	return packedLines
}
