// Pending-delivery counters: the whole units sold and not delivered yet, per stock row. They live
// on ProductStock so the sale checks read them with the stock they limit. Same concurrency rule as
// the movement engine (stock_movement_apply.go): the caller's transaction locks the rows before
// reading them, and a missing row is created with a plain INSERT that reruns on a duplicate key.

package types

import (
	"app/core"
	"app/db"
	"cmp"
	"fmt"
	"slices"
)

// PendingDeliveryChange moves the counters of one stock row by a signed number of whole units:
// positive when a sale or web order is registered undelivered, negative when it is delivered or
// annulled.
type PendingDeliveryChange struct {
	WarehouseID    int32
	ProductID      int32
	PresentationID int16
	Store          int32
	Online         int32
}

// PendingDeliveryOf sums the changes by stock row (PackProductStockID), both counters together.
func PendingDeliveryOf(changes []PendingDeliveryChange) map[int64]int32 {
	unitsByStockID := map[int64]int32{}
	for _, change := range changes {
		unitsByStockID[PackProductStockID(change.WarehouseID, change.ProductID, change.PresentationID)] += change.Store + change.Online
	}
	return unitsByStockID
}

// ApplyPendingDeliveryIn adds the changes to their stock rows, creating the rows that do not exist.
// A counter that would go below zero is refused: it means a release without its reservation, and the
// counters must be recomputed (fn-recompute-pending-delivery) rather than silently clamped.
func ApplyPendingDeliveryIn(transaction *db.Transaction, companyID int32, userID int32, changes []PendingDeliveryChange) error {
	changeByStockID := map[int64]*PendingDeliveryChange{}
	for _, change := range changes {
		if change.Store == 0 && change.Online == 0 {
			continue
		}
		if change.WarehouseID <= 0 || change.ProductID <= 0 {
			return core.Err(fmt.Sprintf("Reserva de stock inválida: almacén %v producto %v.", change.WarehouseID, change.ProductID))
		}
		stockID := PackProductStockID(change.WarehouseID, change.ProductID, change.PresentationID)
		if merged := changeByStockID[stockID]; merged != nil {
			merged.Store += change.Store
			merged.Online += change.Online
		} else {
			changeCopy := change
			changeByStockID[stockID] = &changeCopy
		}
	}
	if len(changeByStockID) == 0 {
		return nil
	}

	// In key order, so two writers of overlapping rows queue instead of deadlocking.
	lockedStocks := make([]ProductStock, 0, len(changeByStockID))
	for stockID := range changeByStockID {
		lockedStocks = append(lockedStocks, ProductStock{CompanyID: companyID, ID: stockID})
	}
	slices.SortFunc(lockedStocks, func(a, b ProductStock) int { return cmp.Compare(a.ID, b.ID) })
	if err := db.LockByKeys(transaction, &lockedStocks); err != nil {
		return core.Err("Error al obtener el stock a reservar:", err)
	}
	stockByID := make(map[int64]*ProductStock, len(lockedStocks))
	for index := range lockedStocks {
		stockByID[lockedStocks[index].ID] = &lockedStocks[index]
	}

	nowTime := core.SUnixTime()
	newStocks, updatedStocks := []ProductStock{}, []ProductStock{}
	for stockID, change := range changeByStockID {
		stock, exists := stockByID[stockID]
		if !exists {
			stock = &ProductStock{
				ID: stockID, CompanyID: companyID, WarehouseID: change.WarehouseID, ProductID: change.ProductID,
				PresentationID: change.PresentationID, Created: nowTime, CreatedBy: userID,
			}
		}
		stock.PendingDeliveryStore += change.Store
		stock.PendingDeliveryOnline += change.Online
		if stock.PendingDeliveryStore < 0 || stock.PendingDeliveryOnline < 0 {
			return core.Err(fmt.Sprintf("Reserva de stock negativa: almacén %v producto %v presentación %v "+
				"(tienda %v, online %v). Ejecute fn-recompute-pending-delivery.", change.WarehouseID, change.ProductID,
				change.PresentationID, stock.PendingDeliveryStore, stock.PendingDeliveryOnline))
		}
		if exists {
			updatedStocks = append(updatedStocks, *stock)
		} else {
			newStocks = append(newStocks, *stock)
		}
	}

	if len(newStocks) > 0 {
		if err := db.InsertIn(transaction, &newStocks); err != nil {
			return core.Err("Error al crear el stock reservado:", err)
		}
	}
	if len(updatedStocks) > 0 {
		stockTable := db.TableOf[ProductStock]()
		// The counters, and the Status SelfParse derives from them with the locked row's quantities (a
		// row with nothing on hand that owes deliveries is Status 1), written with WarehouseID, its
		// delta-index partner. The ORM adds Updated, so the browsers' stock caches pick the change up.
		if err := db.UpdateIn(transaction, &updatedStocks, stockTable.PendingDeliveryStore, stockTable.PendingDeliveryOnline,
			stockTable.Status, stockTable.WarehouseID); err != nil {
			return core.Err("Error al guardar el stock reservado:", err)
		}
	}
	return nil
}
