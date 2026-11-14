package types

import (
	"app/core"
	"fmt"
	"math"
	"slices"
)

// The pure rules of a production order (docs/MANUFACTURING_PLAN.md §3, §4.2, §6): what it may plan,
// what its recipes ask once released, and how a receipt splits its consumption among its products.

// ValidateProductionOrderLines refuses lines that cannot be produced: a product that is not made (a
// finished product or a semi-finished supply with Origin = 2), one without a recipe or standard cost,
// a presentation the product does not have active (presentation 0 when it has none), a repeated
// (product, presentation) or units ≤ 0. productsByID holds the company's rows of every line.
func ValidateProductionOrderLines(lines []ProductionOrderLine, productsByID map[int32]Product) error {
	if len(lines) == 0 {
		return core.Err("La orden de producción necesita al menos un producto.")
	}
	seenLines := map[[2]int32]bool{}
	for _, line := range lines {
		product, isKnown := productsByID[line.ProductID]
		isMadeProduct := isKnown && product.Origin == ProductOriginMade &&
			(product.Status == ProductStatusActive || product.Status == ProductStatusSupply)
		if !isMadeProduct {
			return core.Err(fmt.Sprintf("El producto %v no existe o no es fabricado por la empresa.", line.ProductID))
		}
		if product.ProductionStandardCost <= 0 && len(product.ProductionSupplies) == 0 && len(product.ProductionLabor) == 0 {
			return core.Err(fmt.Sprintf(`"%s" no tiene receta ni costo estándar.`, product.Name))
		}
		activePresentationIDs := []int16{}
		for _, presentation := range product.Presentations {
			if presentation.Status > 0 {
				activePresentationIDs = append(activePresentationIDs, presentation.ID)
			}
		}
		isKnownPresentation := core.If(len(activePresentationIDs) == 0, line.PresentationID == 0,
			slices.Contains(activePresentationIDs, line.PresentationID))
		if !isKnownPresentation {
			return core.Err(fmt.Sprintf(`"%s": la presentación %v no existe o no está activa.`, product.Name, line.PresentationID))
		}
		lineKey := [2]int32{line.ProductID, int32(line.PresentationID)}
		if seenLines[lineKey] {
			return core.Err(fmt.Sprintf(`"%s": la presentación %v está repetida.`, product.Name, line.PresentationID))
		}
		seenLines[lineKey] = true
		if line.PlannedUnits <= 0 {
			return core.Err(fmt.Sprintf(`"%s": las unidades planificadas deben ser mayores a 0.`, product.Name))
		}
	}
	return nil
}

// ExplodeProductionOrder is what a released order freezes (§4.2): per supply, in sub-units of the
// supply's divisor,
//
//	planned[s] = ceil( Σ_lines PlannedUnits × recipe.quantity[s] ÷ recipe.yield × (1 + waste[s]) × (1 + adjustment) )
//
// and per labor type the minutes Σ_lines PlannedUnits × recipe.minutes[t] ÷ recipe.yield, costed at the
// type's estimate rate. A semi-finished supply is consumed from stock, never exploded further. Later
// recipe edits never touch a released order. suppliesByID holds every supply the recipes name.
func ExplodeProductionOrder(lines []ProductionOrderLine, adjustmentBasisPoints int16, productsByID map[int32]Product,
	suppliesByID map[int32]Product, costPerHourByLaborTypeID map[int32]int32) ([]ProductionOrderSupply, []ProductionOrderLabor, error) {
	adjustmentFactor := 1 + float64(adjustmentBasisPoints)/float64(BasisPointsWhole)
	plannedBySupplyID := map[int32]float64{}
	minutesByLaborTypeID := map[int32]float64{}
	for _, line := range lines {
		product := productsByID[line.ProductID]
		yield := float64(max(product.ProductionYieldPortions, 1))
		for _, recipeSupply := range product.ProductionSupplies {
			supply, isKnown := suppliesByID[recipeSupply.SupplyID]
			if !isKnown || supply.Status != ProductStatusSupply {
				return nil, nil, core.Err(fmt.Sprintf(`La receta de "%s" usa el insumo %v, que no existe o no está activo.`, product.Name, recipeSupply.SupplyID))
			}
			if recipeSupply.SubDivisor != max(supply.SbuQuantity, core.QuantityDivisorNone) {
				return nil, nil, core.Err(fmt.Sprintf(`La receta de "%s" usa una sub-unidad anterior del insumo "%s": vuelva a guardarla.`, product.Name, supply.Name))
			}
			recipeSubUnits := float64(core.UnpackQuantityLine(recipeSupply.Quantity).TotalSubUnits(recipeSupply.SubDivisor))
			wasteFactor := 1 + float64(recipeSupply.WasteBasisPoints)/float64(BasisPointsWhole)
			plannedBySupplyID[recipeSupply.SupplyID] += float64(line.PlannedUnits) * recipeSubUnits / yield * wasteFactor * adjustmentFactor
		}
		for _, recipeLabor := range product.ProductionLabor {
			minutesByLaborTypeID[recipeLabor.LaborTypeID] += float64(line.PlannedUnits) * float64(recipeLabor.Minutes) / yield
		}
	}

	orderSupplies := []ProductionOrderSupply{}
	for _, supplyID := range sortedKeys(plannedBySupplyID) {
		// The tolerance keeps a float sum like 1500.0000000002 from asking a whole extra sub-unit.
		planned := int32(math.Ceil(plannedBySupplyID[supplyID] - 1e-6))
		if planned <= 0 {
			continue
		}
		orderSupplies = append(orderSupplies, ProductionOrderSupply{
			SupplyID: supplyID, SubDivisor: max(suppliesByID[supplyID].SbuQuantity, core.QuantityDivisorNone), Planned: planned,
		})
	}
	orderLabor := []ProductionOrderLabor{}
	for _, laborTypeID := range sortedKeys(minutesByLaborTypeID) {
		minutes := int32(math.Round(minutesByLaborTypeID[laborTypeID]))
		orderLabor = append(orderLabor, ProductionOrderLabor{
			LaborTypeID: laborTypeID, PlannedMinutes: minutes,
			PlannedCost: int32(math.Round(float64(minutes) * float64(costPerHourByLaborTypeID[laborTypeID]) / 60)),
		})
	}
	return orderSupplies, orderLabor, nil
}

// ReceiptProductUnits is what a receipt brings of one product, its presentations summed.
type ReceiptProductUnits struct {
	ProductID int32
	Good      int32
	Scrapped  int32
}

// ShareReceiptSupply splits what a receipt consumed of a supply among its products by recipe share:
// each product with good units weighs (good + scrapped) × what its recipe asks of the supply, waste
// included. Scrapped units still used the supply, but a product with no good unit cannot carry value,
// so it weighs nothing and the good units absorb it. When no received product's recipe uses the supply
// (an order line received later does), it is split by good units.
func ShareReceiptSupply(supplyID int32, consumedSubUnits int64, receivedProducts []ReceiptProductUnits,
	productsByID map[int32]Product) []ProductSupplyTake {
	weights := make([]float64, len(receivedProducts))
	hasRecipeWeight := false
	for index, received := range receivedProducts {
		if received.Good <= 0 {
			continue
		}
		product := productsByID[received.ProductID]
		for _, recipeSupply := range product.ProductionSupplies {
			if recipeSupply.SupplyID != supplyID {
				continue
			}
			recipeSubUnits := float64(core.UnpackQuantityLine(recipeSupply.Quantity).TotalSubUnits(recipeSupply.SubDivisor))
			wasteFactor := 1 + float64(recipeSupply.WasteBasisPoints)/float64(BasisPointsWhole)
			weights[index] = float64(received.Good+received.Scrapped) * recipeSubUnits * wasteFactor / float64(max(product.ProductionYieldPortions, 1))
			hasRecipeWeight = hasRecipeWeight || weights[index] > 0
		}
	}
	if !hasRecipeWeight {
		for index, received := range receivedProducts {
			weights[index] = float64(max(received.Good, 0))
		}
	}
	takes := make([]ProductSupplyTake, 0, len(receivedProducts))
	for index, share := range ShareByWeight(consumedSubUnits, weights) {
		if share > 0 {
			takes = append(takes, ProductSupplyTake{ProductID: receivedProducts[index].ProductID, SubUnits: share})
		}
	}
	return takes
}

// ShareReceiptLabor splits a receipt's labor value among its products by what their recipes' labor
// costs for their good units, at the types' estimate rates; by good units when no recipe has labor.
func ShareReceiptLabor(laborValue int64, receivedProducts []ReceiptProductUnits, productsByID map[int32]Product,
	costPerHourByLaborTypeID map[int32]int32) []int64 {
	weights := make([]float64, len(receivedProducts))
	hasRecipeWeight := false
	for index, received := range receivedProducts {
		if received.Good > 0 {
			weights[index] = float64(RecipeLaborStandardCost(productsByID[received.ProductID], received.Good, costPerHourByLaborTypeID))
			hasRecipeWeight = hasRecipeWeight || weights[index] > 0
		}
	}
	if !hasRecipeWeight {
		for index, received := range receivedProducts {
			weights[index] = float64(max(received.Good, 0))
		}
	}
	return ShareByWeight(laborValue, weights)
}

// IsProductionOrderFullyReceived is whether every line has its planned good units: the order then
// finishes on its own. Over-production is allowed; scrapped units do not count.
func IsProductionOrderFullyReceived(lines []ProductionOrderLine) bool {
	for _, line := range lines {
		if line.ProducedUnits < line.PlannedUnits {
			return false
		}
	}
	return len(lines) > 0
}
