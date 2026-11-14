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

// ProductionSupply is one supply of the recipe (Product.ProductionSupplies), for
// ProductionYieldPortions units of the product.
type ProductionSupply struct {
	SupplyID int32 `cb:"1"`
	// Quantity is packed like SaleOrderLine.Quantity (Units*core.QuantityLineScale + Sub), in
	// SubDivisor. The divisor is recorded so a later refinement of the supply never restates it.
	Quantity   int32 `cb:"2"`
	SubDivisor int16 `cb:"3"`
}

// HasProductionConfig is whether any of the product's Production* columns is set.
func (e *Product) HasProductionConfig() bool {
	return e.ProductionYieldPortions != 0 || len(e.ProductionSupplies) > 0 ||
		e.ProductionExtraCostPerUnit != 0 || e.ProductionStandardCost != 0
}

// ValidateProductProduction refuses a production configuration the daily close could not run:
// one on a product that is not made, a recipe without a yield, or a supply that is not an active
// supply of the company, is repeated, has no quantity, disagrees with the supply's sub-unit
// divisor, or has no stock account to credit its consumption to. suppliesByID holds the
// company's rows for every SupplyID of the configuration.
func ValidateProductProduction(product *Product, suppliesByID map[int32]Product) error {
	if !product.HasProductionConfig() {
		return nil
	}
	if product.Origin != ProductOriginMade {
		return core.Err(fmt.Sprintf(`Producto "%s": solo un producto fabricado tiene configuración de producción.`, product.Name))
	}
	if product.ProductionExtraCostPerUnit < 0 || product.ProductionStandardCost < 0 {
		return core.Err(fmt.Sprintf(`Producto "%s": los costos de producción no pueden ser negativos.`, product.Name))
	}
	if product.ProductionYieldPortions < 0 || (len(product.ProductionSupplies) > 0 && product.ProductionYieldPortions == 0) {
		return core.Err(fmt.Sprintf(`Producto "%s": la receta debe indicar cuántas porciones rinde.`, product.Name))
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

		// The close credits the supply's stock account for what it consumes; an expense or asset
		// account (656, 33x) has none, so that consumption could never be posted.
		pcgeAccount, isKnownAccount := accounting.FindPCGEAccount(supply.AccountCode)
		if !isKnownAccount || pcgeAccount.StockCode == 0 {
			return core.Err(fmt.Sprintf(`Producto "%s": el insumo "%s" tiene la cuenta %v, que no es de existencias; no puede consumirse en producción.`,
				product.Name, supply.Name, supply.AccountCode))
		}
	}
	return nil
}
