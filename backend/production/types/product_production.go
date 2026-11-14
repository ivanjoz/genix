package types

import (
	accounting "app/accounting/types"
	"app/core"
	"fmt"
)

// CompanyFlagStrictSupplyClose (company_flags.toml, [production]) blocks the daily production close
// when a supply falls short of what the day's sales need. Without it, the close splits what is
// available among the products and closes anyway.
const CompanyFlagStrictSupplyClose = int16(15)

// BasisPointsWhole is 100%, in basis points: 1500 is 15%.
const BasisPointsWhole = int64(10000)

// MaxRecipeDepth is how many levels of semi-finished supplies a recipe may reach through: a shoe
// made of stitched uppers made of cut leather is depth 2.
const MaxRecipeDepth = 5

// ProductionSupply is one supply of the recipe (Product.ProductionSupplies), for
// ProductionYieldPortions units of the product.
type ProductionSupply struct {
	SupplyID int32 `cb:"1"`
	// Quantity is packed like SaleOrderLine.Quantity (Units*core.QuantityLineScale + Sub), in
	// SubDivisor. The divisor is recorded so a later refinement of the supply never restates it.
	Quantity   int32 `cb:"2"`
	SubDivisor int16 `cb:"3"`
	// WasteBasisPoints is the expected waste on top of Quantity: 1500 = 15% (leather), 0 = none.
	// Every consumption of the recipe, the daily close's included, takes it (M5, M33).
	WasteBasisPoints int16 `cb:"4"`
}

// ProductionLabor is the work ProductionYieldPortions units of the product are expected to need, of
// one labor type (docs/MANUFACTURING_PLAN.md §2).
type ProductionLabor struct {
	LaborTypeID int32 `cb:"1"`
	Minutes     int32 `cb:"2"`
}

// SubUnitsWithWaste is what the recipe line needs for units of the product, in sub-units of its
// divisor, waste included and rounded up: a recipe never consumes less than it asks for.
func (e ProductionSupply) SubUnitsWithWaste(units int32, yield int16) int32 {
	recipeSubUnits := core.UnpackQuantityLine(e.Quantity).TotalSubUnits(e.SubDivisor)
	numerator := recipeSubUnits * int64(units) * (BasisPointsWhole + int64(e.WasteBasisPoints))
	denominator := int64(max(yield, 1)) * BasisPointsWhole
	return int32((numerator + denominator - 1) / denominator)
}

// RecipeLaborStandardCost is the recipe's labor for units of the product at each type's estimate
// rate, PEN cents: Σ minutes × CostPerHour × units ÷ (60 × yield), rounded.
func RecipeLaborStandardCost(product Product, units int32, costPerHourByLaborTypeID map[int32]int32) int64 {
	minuteCents := int64(0)
	for _, recipeLabor := range product.ProductionLabor {
		minuteCents += int64(recipeLabor.Minutes) * int64(costPerHourByLaborTypeID[recipeLabor.LaborTypeID])
	}
	denominator := 60 * int64(max(product.ProductionYieldPortions, 1))
	return (minuteCents*int64(units) + denominator/2) / denominator
}

// IsSemiFinished is whether the row is a supply the company makes with its own recipe
// (docs/MANUFACTURING_PLAN.md §2.1): stocked and consumed like any supply, never sold or bought.
func (e *Product) IsSemiFinished() bool {
	return e.Status == ProductStatusSupply && e.Origin == ProductOriginMade
}

// HasProductionConfig is whether any of the product's Production* columns is set.
func (e *Product) HasProductionConfig() bool {
	return e.ProductionYieldPortions != 0 || len(e.ProductionSupplies) > 0 || len(e.ProductionLabor) > 0 ||
		e.ProductionExtraCostPerUnit != 0 || e.ProductionStandardCost != 0
}

// ValidateProductProduction refuses a production configuration that could not be produced: one on a
// product that is not made, a recipe without a yield, a supply that is not an active supply of the
// company, is repeated, has no quantity or a waste outside 0–100%, disagrees with the supply's
// sub-unit divisor, or has no stock account to credit its consumption to; a labor line of an unknown
// type, repeated or without minutes; or a recipe that reaches itself through its semi-finished
// supplies, or goes deeper than MaxRecipeDepth.
//
// suppliesByID holds the company's rows for every supply the recipe reaches, the semi-finished ones'
// recipes included (their ProductionSupplies). laborTypesByID holds the company's labor types the
// recipe names.
func ValidateProductProduction(product *Product, suppliesByID map[int32]Product, laborTypesByID map[int32]ProductionLaborType) error {
	if !product.HasProductionConfig() {
		return nil
	}
	if product.Origin != ProductOriginMade {
		return core.Err(fmt.Sprintf(`Producto "%s": solo un producto fabricado tiene configuración de producción.`, product.Name))
	}
	if product.ProductionExtraCostPerUnit < 0 || product.ProductionStandardCost < 0 {
		return core.Err(fmt.Sprintf(`Producto "%s": los costos de producción no pueden ser negativos.`, product.Name))
	}
	hasRecipeLines := len(product.ProductionSupplies) > 0 || len(product.ProductionLabor) > 0
	if product.ProductionYieldPortions < 0 || (hasRecipeLines && product.ProductionYieldPortions == 0) {
		return core.Err(fmt.Sprintf(`Producto "%s": la receta debe indicar cuántas unidades rinde.`, product.Name))
	}

	seenSupplyIDs := map[int32]bool{}
	for _, recipeSupply := range product.ProductionSupplies {
		supply, isKnown := suppliesByID[recipeSupply.SupplyID]
		if !isKnown || supply.Status != ProductStatusSupply {
			return core.Err(fmt.Sprintf(`Producto "%s": el insumo %v de la receta no existe o no está activo.`,
				product.Name, recipeSupply.SupplyID))
		}
		if seenSupplyIDs[recipeSupply.SupplyID] {
			return core.Err(fmt.Sprintf(`Producto "%s": el insumo "%s" está repetido en la receta.`, product.Name, supply.Name))
		}
		seenSupplyIDs[recipeSupply.SupplyID] = true

		// The recipe quantity is read in the supply's own sub-unit, so it has to be the divisor
		// the supply has now: a stale one would consume sixths as if they were twelfths.
		supplyDivisor := max(supply.SbuQuantity, core.QuantityDivisorNone)
		if recipeSupply.SubDivisor != supplyDivisor {
			return core.Err(fmt.Sprintf(`Producto "%s": la cantidad del insumo "%s" usa el divisor %v, pero el insumo tiene %v.`,
				product.Name, supply.Name, recipeSupply.SubDivisor, supplyDivisor))
		}
		quantity := core.UnpackQuantityLine(recipeSupply.Quantity)
		if recipeSupply.Quantity <= 0 || quantity.Sub >= int32(supplyDivisor) {
			return core.Err(fmt.Sprintf(`Producto "%s": la cantidad del insumo "%s" es inválida.`, product.Name, supply.Name))
		}
		if recipeSupply.WasteBasisPoints < 0 || int64(recipeSupply.WasteBasisPoints) > BasisPointsWhole {
			return core.Err(fmt.Sprintf(`Producto "%s": la merma del insumo "%s" debe estar entre 0%% y 100%%.`, product.Name, supply.Name))
		}

		// The consumption credits the supply's stock account. A semi-finished supply's is 231 by its
		// origin; a bought one's follows from its purchase account, and an expense or asset account
		// (656, 33x) has none, so that consumption could never be posted.
		if supply.IsSemiFinished() {
			continue
		}
		pcgeAccount, isKnownAccount := accounting.FindPCGEAccount(supply.AccountCode)
		if !isKnownAccount || pcgeAccount.StockCode == 0 {
			return core.Err(fmt.Sprintf(`Producto "%s": el insumo "%s" tiene la cuenta %v, que no es de existencias; no puede consumirse en producción.`,
				product.Name, supply.Name, supply.AccountCode))
		}
	}

	seenLaborTypeIDs := map[int32]bool{}
	for _, recipeLabor := range product.ProductionLabor {
		laborType, isKnown := laborTypesByID[recipeLabor.LaborTypeID]
		if !isKnown || laborType.Status != 1 {
			return core.Err(fmt.Sprintf(`Producto "%s": el tipo de trabajo %v de la receta no existe o no está activo.`,
				product.Name, recipeLabor.LaborTypeID))
		}
		if seenLaborTypeIDs[recipeLabor.LaborTypeID] {
			return core.Err(fmt.Sprintf(`Producto "%s": el tipo de trabajo "%s" está repetido en la receta.`, product.Name, laborType.Name))
		}
		seenLaborTypeIDs[recipeLabor.LaborTypeID] = true
		if recipeLabor.Minutes <= 0 {
			return core.Err(fmt.Sprintf(`Producto "%s": el tipo de trabajo "%s" necesita minutos.`, product.Name, laborType.Name))
		}
	}

	return validateRecipeGraph(product, suppliesByID)
}

// validateRecipeGraph walks the recipe down through its semi-finished supplies: it may not reach the
// product itself (it could never be made) nor go deeper than MaxRecipeDepth. The product's own
// recipe is the one being saved, so a stored copy of it in suppliesByID is never read.
func validateRecipeGraph(product *Product, suppliesByID map[int32]Product) error {
	var walk func(recipe []ProductionSupply, depth int) error
	walk = func(recipe []ProductionSupply, depth int) error {
		for _, recipeSupply := range recipe {
			if recipeSupply.SupplyID == product.ID && product.ID != 0 {
				return core.Err(fmt.Sprintf(`Producto "%s": la receta se contiene a sí misma a través de sus insumos semielaborados.`, product.Name))
			}
			supply := suppliesByID[recipeSupply.SupplyID]
			if !supply.IsSemiFinished() || len(supply.ProductionSupplies) == 0 {
				continue
			}
			if depth >= MaxRecipeDepth {
				return core.Err(fmt.Sprintf(`Producto "%s": la receta pasa por más de %v niveles de semielaborados.`, product.Name, MaxRecipeDepth))
			}
			if err := walk(supply.ProductionSupplies, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(product.ProductionSupplies, 0)
}
