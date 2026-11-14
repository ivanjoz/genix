package types

import (
	accounting "app/accounting/types"
	"app/core"
	"fmt"
	"slices"
)

// The daily production close (docs/PRODUCTION_PLAN.md §5): the made products a warehouse delivered
// on a day are produced from their recipes, and their supplies consumed, so the sales' stock exits
// can finally happen. PlanDailyClose is the calculation, pure: the preview shows it, and the close
// recomputes it inside its transaction before writing the movements it lists.

// DailyCloseProduct is what a close produced of one made product.
type DailyCloseProduct struct {
	ProductID  int32 `cb:"1"`
	Units      int32 `cb:"2"`
	CostSource int8  `cb:"3"`
	// Value is the production's cost, PEN cents: estimated by a preview, booked by the close.
	Value int32 `cb:"4"`
	// ShortSupplies: some supply of its recipe was short, so it took less than the recipe asks.
	ShortSupplies bool `cb:"5"`
}

// DailyCloseSupply is what a close needed, found and consumed of one supply, in sub-units of
// SubDivisor (the supply's own sub-unit).
type DailyCloseSupply struct {
	SupplyID   int32 `cb:"1"`
	SubDivisor int16 `cb:"2"`
	Needed     int32 `cb:"3"`
	Available  int32 `cb:"4"`
	Consumed   int32 `cb:"5"`
}

type DailyCloseInput struct {
	IsStrict bool
	// Whole units of each made product the pending lines hold (a sub-unit line counts its whole
	// unit, as its reservation did).
	SoldUnitsByProduct map[int32]int32
	ProductsByID       map[int32]Product
	SuppliesByID       map[int32]Product
	SupplyStockByID    map[int32]SupplyStock
	// CostPerHourByLaborTypeID values the recipes' labor at standard (M26): cooks log no hours.
	CostPerHourByLaborTypeID map[int32]int32
}

type DailyClosePlan struct {
	Products        []DailyCloseProduct
	Supplies        []DailyCloseSupply
	Consumptions    []SupplyConsumption
	BlockingReasons []string
}

// PlanDailyClose computes a close (§5.3):
//
//	needed[p][s]   = ceil(recipe(p).quantity[s] × sold[p] ÷ recipe(p).yield × (1 + waste[s])), sub-units of s
//	available[s]   = the free bucket + the released lots of s in the supplies warehouse − what orders reserved
//	consumed[p][s] = needed[p][s] when available[s] covers every product, else (not strict) its
//	                 share floor(available × needed[p][s] ÷ needed[s]), the remainder to the largest
//
// A product's value is ProductionStandardCost × units, or the value its consumption takes out of the
// supplies' rows + its recipe labor at standard + ProductionExtraCostPerUnit × units. It never fails: what keeps the close from running is listed in
// BlockingReasons.
func PlanDailyClose(input DailyCloseInput) DailyClosePlan {
	plan := DailyClosePlan{}
	block := func(reason string) { plan.BlockingReasons = append(plan.BlockingReasons, reason) }

	productIDs := sortedKeys(input.SoldUnitsByProduct)
	neededByProductSupply := map[int32]map[int32]int32{}
	neededBySupply := map[int32]int32{}
	for _, productID := range productIDs {
		product := input.ProductsByID[productID]
		units := input.SoldUnitsByProduct[productID]
		if product.ProductionStandardCost <= 0 && len(product.ProductionSupplies) == 0 && len(product.ProductionLabor) == 0 {
			block(fmt.Sprintf(`"%s" no tiene receta ni costo estándar.`, product.Name))
			continue
		}
		neededByProductSupply[productID] = map[int32]int32{}
		for _, recipeSupply := range product.ProductionSupplies {
			supply, isKnown := input.SuppliesByID[recipeSupply.SupplyID]
			if !isKnown || supply.Status != ProductStatusSupply {
				block(fmt.Sprintf(`La receta de "%s" usa el insumo %v, que no existe o no está activo.`, product.Name, recipeSupply.SupplyID))
				continue
			}
			// A semi-finished supply's stock account is 231 by its origin; it is consumed from stock,
			// never exploded through its own recipe (PRODUCTION P12).
			if pcgeAccount, isKnownAccount := accounting.FindPCGEAccount(supply.AccountCode); !supply.IsSemiFinished() &&
				(!isKnownAccount || pcgeAccount.StockCode == 0) {
				block(fmt.Sprintf(`El insumo "%s" tiene la cuenta %v, que no es de existencias.`, supply.Name, supply.AccountCode))
				continue
			}
			if recipeSupply.SubDivisor != max(supply.SbuQuantity, core.QuantityDivisorNone) {
				block(fmt.Sprintf(`La receta de "%s" usa una sub-unidad anterior del insumo "%s": vuelva a guardarla.`, product.Name, supply.Name))
				continue
			}
			needed := recipeSupply.SubUnitsWithWaste(units, product.ProductionYieldPortions)
			neededByProductSupply[productID][recipeSupply.SupplyID] = needed
			neededBySupply[recipeSupply.SupplyID] += needed
		}
	}

	consumedValueByProduct := map[int32]int64{}
	shortProductIDs := map[int32]bool{}
	for _, supplyID := range sortedKeys(neededBySupply) {
		supply := input.SuppliesByID[supplyID]
		divisor := max(supply.SbuQuantity, core.QuantityDivisorNone)
		stock := input.SupplyStockByID[supplyID]

		atDivisor, err := ReadSupplyStockAtDivisor(stock, divisor)
		if err != nil {
			block(fmt.Sprintf(`El stock del insumo "%s" está en una sub-unidad incompatible con la actual.`, supply.Name))
			continue
		}
		availableSubUnits := atDivisor.AvailableSubUnits

		needed := neededBySupply[supplyID]
		takes := []ProductSupplyTake{}
		needWeights := []float64{}
		for _, productID := range productIDs {
			if neededForProduct, uses := neededByProductSupply[productID][supplyID]; uses {
				takes = append(takes, ProductSupplyTake{ProductID: productID, SubUnits: int64(neededForProduct)})
				needWeights = append(needWeights, float64(neededForProduct))
			}
		}
		if availableSubUnits < int64(needed) {
			if input.IsStrict {
				block(fmt.Sprintf(`Falta el insumo "%s": se necesitan %v y hay %v.`, supply.Name,
					FormatSubUnits(int64(needed), divisor, supply.SbuUnit), FormatSubUnits(availableSubUnits, divisor, supply.SbuUnit)))
			}
			// The available stock is shared by need; the floors' remainder goes to the largest need.
			for takeIndex, share := range ShareByWeight(availableSubUnits, needWeights) {
				takes[takeIndex].SubUnits = share
				shortProductIDs[takes[takeIndex].ProductID] = true
			}
		}

		// Each product takes its consumption bucket by bucket, valued as the engine will value it.
		consumptions, err := WalkSupplyBuckets(supplyID, divisor, stock, atDivisor, takes)
		if err != nil {
			block(fmt.Sprintf(`No hay stock suficiente del insumo "%s".`, supply.Name))
			continue
		}
		consumedTotal := int32(0)
		for _, consumption := range consumptions {
			consumedTotal += consumption.SubUnits
			consumedValueByProduct[consumption.ProductID] += int64(consumption.EstimatedValue)
		}
		plan.Consumptions = append(plan.Consumptions, consumptions...)
		plan.Supplies = append(plan.Supplies, DailyCloseSupply{
			SupplyID: supplyID, SubDivisor: divisor, Needed: needed,
			Available: int32(availableSubUnits), Consumed: consumedTotal,
		})
	}

	for _, productID := range productIDs {
		product := input.ProductsByID[productID]
		units := input.SoldUnitsByProduct[productID]
		closeProduct := DailyCloseProduct{ProductID: productID, Units: units, ShortSupplies: shortProductIDs[productID]}
		if product.ProductionStandardCost > 0 {
			closeProduct.CostSource = CostSourceStandard
			closeProduct.Value = product.ProductionStandardCost * units
		} else {
			closeProduct.CostSource = CostSourceRecipe
			closeProduct.Value = int32(consumedValueByProduct[productID]) + product.ProductionExtraCostPerUnit*units +
				int32(RecipeLaborStandardCost(product, units, input.CostPerHourByLaborTypeID))
			// The stock engine refuses an uncosted inflow into a row with no average: a production
			// with no value could not enter. The extra cost per unit alone is value enough.
			if closeProduct.Value <= 0 && (len(product.ProductionSupplies) > 0 || len(product.ProductionLabor) > 0) {
				block(fmt.Sprintf(`"%s" no tiene costo: sus insumos consumidos no tienen valor y no tiene costo estándar ni costo extra.`, product.Name))
			}
		}
		plan.Products = append(plan.Products, closeProduct)
	}
	return plan
}

// FormatSubUnits renders sub-units of a divisor the way the operator reads them: "2 + 500 g".
func FormatSubUnits(subUnits int64, divisor int16, subUnitName string) string {
	units, sub := subUnits/int64(divisor), subUnits%int64(divisor)
	if sub == 0 {
		return fmt.Sprint(units)
	}
	return fmt.Sprintf("%v + %v %s", units, sub, subUnitName)
}

func sortedKeys[V any](values map[int32]V) []int32 {
	keys := make([]int32, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// DailyCloseOrderLines are the close's order lines (docs/MANUFACTURING_PLAN.md §3): one per product
// and presentation its receipt entered, planned and produced alike, with the cost source and the
// shortage of the product's plan.
func DailyCloseOrderLines(closeProducts []DailyCloseProduct, receipt ProductionReceipt) []ProductionOrderLine {
	closeProductByID := map[int32]DailyCloseProduct{}
	for _, closeProduct := range closeProducts {
		closeProductByID[closeProduct.ProductID] = closeProduct
	}
	orderLines := make([]ProductionOrderLine, 0, len(receipt.Lines))
	for _, receiptLine := range receipt.Lines {
		closeProduct := closeProductByID[receiptLine.ProductID]
		orderLines = append(orderLines, ProductionOrderLine{
			ProductID: receiptLine.ProductID, PresentationID: receiptLine.PresentationID,
			PlannedUnits: receiptLine.Units, ProducedUnits: receiptLine.Units, Value: receiptLine.Value,
			CostSource: closeProduct.CostSource, ShortSupplies: closeProduct.ShortSupplies,
		})
	}
	return orderLines
}
