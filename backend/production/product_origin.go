package production

import (
	business "app/business/types"
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	"app/production/types"
	sales "app/sales/types"
	"fmt"
)

// validateProductOriginChanges refuses an Origin change the daily production close could not follow
// (docs/PRODUCTION_PLAN.md §3.5):
//   - made → bought while a delivered sale still owes a stock exit of the product: the close settles
//     only made products, so that exit would never happen;
//   - bought → made while the product has stock: the close would produce on top of units that were
//     bought, and value them as produced.
//
// storedByID holds the stored rows of the edited products.
func validateProductOriginChanges(companyID int32, products []types.Product, storedByID map[int32]types.Product) error {
	madeToBought, boughtToMade := []types.Product{}, []types.Product{}
	for _, product := range products {
		stored, isStored := storedByID[product.ID]
		if !isStored || stored.Origin == product.Origin {
			continue
		}
		if stored.Origin == types.ProductOriginMade {
			madeToBought = append(madeToBought, product)
		} else if product.Origin == types.ProductOriginMade {
			boughtToMade = append(boughtToMade, stored)
		}
	}

	if len(madeToBought) > 0 {
		productIDsWithPendingExit, err := readProductIDsWithPendingStockExit(companyID)
		if err != nil {
			return err
		}
		for _, product := range madeToBought {
			if productIDsWithPendingExit[product.ID] {
				return core.Err(fmt.Sprintf(`"%s" tiene ventas entregadas pendientes del cierre de producción: ciérrelas antes de cambiarlo a comprado.`, product.Name))
			}
		}
	}
	for _, stored := range boughtToMade {
		hasStock, err := productHasStock(companyID, stored)
		if err != nil {
			return err
		}
		if hasStock {
			return core.Err(fmt.Sprintf(`"%s" tiene stock: no puede pasar a fabricado mientras lo tenga.`, stored.Name))
		}
	}
	return nil
}

// readProductIDsWithPendingStockExit are the products of every line still waiting for the production
// close, from the delivered sales flagged PendingStockMovement. That set only holds the sales since
// the last close, so it is read whole.
func readProductIDsWithPendingStockExit(companyID int32) (map[int32]bool, error) {
	pendingSales := []sales.SaleOrder{}
	query := db.Query(&pendingSales)
	query.Select(query.ID, query.Lines).
		CompanyID.Equals(companyID).PendingStockMovement.Equals(1).
		Delta(db.DeltaCursor{}, sales.OrderStatusDelivered, sales.OrderStatusCompleted)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las ventas pendientes del cierre de producción:", err)
	}
	productIDs := map[int32]bool{}
	for _, sale := range pendingSales {
		// A Delta() read ends with its metadata records, which carry a negative ID.
		if sale.ID <= 0 {
			continue
		}
		for _, line := range sales.LinesWithStockExitPending(sale.Lines, true) {
			productIDs[line.ProductID] = true
		}
	}
	return productIDs, nil
}

// productHasStock is whether any warehouse holds units of the product, in any presentation, lots and
// serials included.
func productHasStock(companyID int32, product types.Product) (bool, error) {
	warehouses := []business.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID).CompanyID.Equals(companyID)
	if err := warehouseQuery.Exec(); err != nil {
		return false, core.Err("Error al obtener los almacenes:", err)
	}
	presentationIDs := []int16{0}
	for _, presentation := range product.Presentations {
		presentationIDs = append(presentationIDs, presentation.ID)
	}
	stockIDs := core.SliceSet[int64]{}
	for _, warehouse := range warehouses {
		for _, presentationID := range presentationIDs {
			stockIDs.Add(logistics.PackProductStockID(warehouse.ID, product.ID, presentationID))
		}
	}
	if stockIDs.IsEmpty() {
		return false, nil
	}
	stockRows := []logistics.ProductStock{}
	stockQuery := db.Query(&stockRows)
	stockQuery.Select(stockQuery.Quantity, stockQuery.SubQuantity, stockQuery.DetailQuantity, stockQuery.DetailSubQuantity).
		CompanyID.Equals(companyID).ID.In(stockIDs.Values...)
	if err := stockQuery.Exec(); err != nil {
		return false, core.Err("Error al obtener el stock del producto:", err)
	}
	for _, stockRow := range stockRows {
		if stockRow.Quantity != 0 || stockRow.SubQuantity != 0 || stockRow.DetailQuantity != 0 || stockRow.DetailSubQuantity != 0 {
			return true, nil
		}
	}
	return false, nil
}
