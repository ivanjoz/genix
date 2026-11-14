package types

import (
	accounting "app/accounting/types"
	"app/core"
	logistics "app/logistics/types"
	"fmt"
	"slices"
)

// The daily production close (docs/PRODUCTION_PLAN.md §5): the made products a warehouse delivered
// on a day are produced from their recipes, and their supplies consumed, so the sales' stock exits
// can finally happen. PlanDailyClose is the calculation, pure: the preview shows it, and the close
// recomputes it inside its transaction before writing the movements it lists.

const (
	CloseCostSourceStandard = int8(1) // the product's StandardCost × units
	CloseCostSourceSupplies = int8(2) // what its supplies' consumption was worth + ExtraCostPerUnit × units
)

// ProductionCloseProduct is what a close produced of one made product.
type ProductionCloseProduct struct {
	ProductID  int32 `cb:"1"`
	Units      int32 `cb:"2"`
	CostSource int8  `cb:"3"`
	// Value is the production's cost, PEN cents: estimated by a preview, booked by the close.
	Value int32 `cb:"4"`
	// ShortSupplies: some supply of its recipe was short, so it took less than the recipe asks.
	ShortSupplies bool `cb:"5"`
}

// ProductionCloseSupply is what a close needed, found and consumed of one supply, in sub-units of
// SubDivisor (the supply's own sub-unit).
type ProductionCloseSupply struct {
	SupplyID   int32 `cb:"1"`
	SubDivisor int16 `cb:"2"`
	Needed     int32 `cb:"3"`
	Available  int32 `cb:"4"`
	Consumed   int32 `cb:"5"`
}

// DailyCloseStockBucket is one bucket of a supply's stock row in the supplies warehouse: the free
// bucket (no lot, no serial) or one detail row.
type DailyCloseStockBucket struct {
	LotID        int32
	SerialNumber string
	Quantity     core.Quantity // at the row's SubDivisor
}

// DailyCloseSupplyStock is a supply's stock row in the supplies warehouse.
type DailyCloseSupplyStock struct {
	SubDivisor int16
	StockValue int64
	// Buckets: the free bucket first, then the details oldest lot first. An exit without a lot
	// draws only from the free bucket, so the consumption walks them in this order.
	Buckets []DailyCloseStockBucket
}

type DailyCloseInput struct {
	IsStrict bool
	// Whole units of each made product the pending lines hold (a sub-unit line counts its whole
	// unit, as its reservation did).
	SoldUnitsByProduct map[int32]int32
	ProductsByID       map[int32]Product
	SuppliesByID       map[int32]Product
	SupplyStockByID    map[int32]DailyCloseSupplyStock
}

// DailyCloseConsumption is one consumption exit: what a product took of a supply from one bucket.
type DailyCloseConsumption struct {
	ProductID    int32
	SupplyID     int32
	LotID        int32
	SerialNumber string
	SubUnits     int32
	SubDivisor   int16
	// EstimatedValue is the bucket's share of the row value, as the stock engine will take it.
	EstimatedValue int32
}

type DailyClosePlan struct {
	Products        []ProductionCloseProduct
	Supplies        []ProductionCloseSupply
	Consumptions    []DailyCloseConsumption
	BlockingReasons []string
}

// PlanDailyClose computes a close (§5.3):
//
//	needed[p][s]   = ceil(recipe(p).quantity[s] × sold[p] ÷ recipe(p).yield), sub-units of s
//	available[s]   = the free bucket + the lots of s in the supplies warehouse
//	consumed[p][s] = needed[p][s] when available[s] covers every product, else (not strict) its
//	                 share floor(available × needed[p][s] ÷ needed[s]), the remainder to the largest
//
// A product's value is ProductionStandardCost × units, or the value its consumption takes out of the
// supplies' rows + ProductionExtraCostPerUnit × units. It never fails: what keeps the close from running is listed in
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
		if product.ProductionStandardCost <= 0 && len(product.ProductionSupplies) == 0 {
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
			if pcgeAccount, isKnownAccount := accounting.FindPCGEAccount(supply.AccountCode); !isKnownAccount || pcgeAccount.StockCode == 0 {
				block(fmt.Sprintf(`El insumo "%s" tiene la cuenta %v, que no es de existencias.`, supply.Name, supply.AccountCode))
				continue
			}
			if recipeSupply.SubDivisor != max(supply.SbuQuantity, core.QuantityDivisorNone) {
				block(fmt.Sprintf(`La receta de "%s" usa una sub-unidad anterior del insumo "%s": vuelva a guardarla.`, product.Name, supply.Name))
				continue
			}
			recipeSubUnits := core.UnpackQuantityLine(recipeSupply.Quantity).TotalSubUnits(recipeSupply.SubDivisor)
			yield := int64(max(product.ProductionYieldPortions, 1))
			needed := int32((recipeSubUnits*int64(units) + yield - 1) / yield)
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

		// Every bucket at the supply's divisor. A row still at a coarser one converts exactly.
		bucketSubUnits := make([]int64, len(stock.Buckets))
		balanceSubUnits, availableSubUnits := int64(0), int64(0)
		isConvertible := true
		for bucketIndex, bucket := range stock.Buckets {
			converted, err := bucket.Quantity.ConvertToDivisor(max(stock.SubDivisor, core.QuantityDivisorNone), divisor)
			if err != nil {
				isConvertible = false
				break
			}
			bucketSubUnits[bucketIndex] = converted.TotalSubUnits(divisor)
			balanceSubUnits += bucketSubUnits[bucketIndex]
			availableSubUnits += max(bucketSubUnits[bucketIndex], 0)
		}
		if !isConvertible {
			block(fmt.Sprintf(`El stock del insumo "%s" está en una sub-unidad incompatible con la actual.`, supply.Name))
			continue
		}

		needed := neededBySupply[supplyID]
		consumedByProduct := map[int32]int32{}
		consumingProductIDs := []int32{}
		for _, productID := range productIDs {
			if neededForProduct, uses := neededByProductSupply[productID][supplyID]; uses {
				consumingProductIDs = append(consumingProductIDs, productID)
				consumedByProduct[productID] = neededForProduct
			}
		}
		if availableSubUnits < int64(needed) {
			if input.IsStrict {
				block(fmt.Sprintf(`Falta el insumo "%s": se necesitan %v y hay %v.`, supply.Name,
					formatSubUnits(int64(needed), divisor, supply.SbuUnit), formatSubUnits(availableSubUnits, divisor, supply.SbuUnit)))
			}
			// The available stock is shared by need; the floors' remainder goes to the largest need.
			largestProductID, distributed := int32(0), int64(0)
			for _, productID := range consumingProductIDs {
				share := availableSubUnits * int64(neededByProductSupply[productID][supplyID]) / int64(needed)
				consumedByProduct[productID] = int32(share)
				distributed += share
				shortProductIDs[productID] = true
				if largestProductID == 0 || neededByProductSupply[productID][supplyID] > neededByProductSupply[largestProductID][supplyID] {
					largestProductID = productID
				}
			}
			consumedByProduct[largestProductID] += int32(availableSubUnits - distributed)
		}

		// Each product takes its consumption bucket by bucket, valued as the engine will value it:
		// its share of what the row is worth at that point.
		remainingValue, remainingBalance := stock.StockValue, balanceSubUnits
		bucketIndex := 0
		consumedTotal := int32(0)
		for _, productID := range consumingProductIDs {
			toConsume := int64(consumedByProduct[productID])
			consumedTotal += consumedByProduct[productID]
			for toConsume > 0 && bucketIndex < len(stock.Buckets) {
				taken := min(toConsume, max(bucketSubUnits[bucketIndex], 0))
				if taken > 0 {
					value := logistics.StockOutflowValue(remainingValue, remainingBalance, taken)
					remainingValue -= int64(value)
					remainingBalance -= taken
					bucketSubUnits[bucketIndex] -= taken
					toConsume -= taken
					consumedValueByProduct[productID] += int64(value)
					bucket := stock.Buckets[bucketIndex]
					plan.Consumptions = append(plan.Consumptions, DailyCloseConsumption{
						ProductID: productID, SupplyID: supplyID, LotID: bucket.LotID, SerialNumber: bucket.SerialNumber,
						SubUnits: int32(taken), SubDivisor: divisor, EstimatedValue: value,
					})
				}
				if bucketSubUnits[bucketIndex] <= 0 {
					bucketIndex++
				}
			}
		}
		plan.Supplies = append(plan.Supplies, ProductionCloseSupply{
			SupplyID: supplyID, SubDivisor: divisor, Needed: needed,
			Available: int32(availableSubUnits), Consumed: consumedTotal,
		})
	}

	for _, productID := range productIDs {
		product := input.ProductsByID[productID]
		units := input.SoldUnitsByProduct[productID]
		closeProduct := ProductionCloseProduct{ProductID: productID, Units: units, ShortSupplies: shortProductIDs[productID]}
		if product.ProductionStandardCost > 0 {
			closeProduct.CostSource = CloseCostSourceStandard
			closeProduct.Value = product.ProductionStandardCost * units
		} else {
			closeProduct.CostSource = CloseCostSourceSupplies
			closeProduct.Value = int32(consumedValueByProduct[productID]) + product.ProductionExtraCostPerUnit*units
			// The stock engine refuses an uncosted inflow into a row with no average: a production
			// with no value could not enter. The extra cost per unit alone is value enough.
			if closeProduct.Value <= 0 && len(product.ProductionSupplies) > 0 {
				block(fmt.Sprintf(`"%s" no tiene costo: sus insumos consumidos no tienen valor y no tiene costo estándar ni costo extra.`, product.Name))
			}
		}
		plan.Products = append(plan.Products, closeProduct)
	}
	return plan
}

// formatSubUnits renders sub-units of a divisor the way the operator reads them: "2 + 500 g".
func formatSubUnits(subUnits int64, divisor int16, subUnitName string) string {
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
