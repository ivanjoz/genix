// Production reservations (docs/MANUFACTURING_PLAN.md §5): what released production orders hold of a
// supply's stock row and have not consumed yet. They live on ProductStock so the stock engine checks
// them with the balance they limit. Same concurrency rule as the engine: rows are locked in key order
// before they are read.

package types

import (
	"app/core"
	"app/db"
	"cmp"
	"fmt"
	"slices"
)

// ProductionReservationChange moves the reservation of a supply in a warehouse by signed SubUnits,
// counted at SubDivisor (the supply's own sub-unit, as on the order). Supplies are stocked in
// presentation 0.
type ProductionReservationChange struct {
	WarehouseID int32
	ProductID   int32
	SubUnits    int32
	SubDivisor  int16
}

// ApplyProductionReservationIn adds the changes to their stock rows. An increase is refused unless
// the row's balance minus its blocked lots and what is already reserved covers it: nothing may
// reserve more than is available (M17). A decrease below zero is refused: it means a release without its reservation, and
// the counters must be rebuilt (fn-recompute-production-reservation) rather than silently clamped.
func ApplyProductionReservationIn(transaction *db.Transaction, companyID int32, changes []ProductionReservationChange) error {
	type mergedChange struct {
		change   ProductionReservationChange
		subUnits int64 // at change.SubDivisor
	}
	changeByStockID := map[int64]*mergedChange{}
	for _, change := range changes {
		if change.SubUnits == 0 {
			continue
		}
		if change.WarehouseID <= 0 || change.ProductID <= 0 {
			return core.Err(fmt.Sprintf("Reserva de producción inválida: almacén %v producto %v.", change.WarehouseID, change.ProductID))
		}
		change.SubDivisor = max(change.SubDivisor, core.QuantityDivisorNone)
		stockID := PackProductStockID(change.WarehouseID, change.ProductID, 0)
		merged := changeByStockID[stockID]
		if merged == nil {
			changeByStockID[stockID] = &mergedChange{change: change, subUnits: int64(change.SubUnits)}
			continue
		}
		if merged.change.SubDivisor != change.SubDivisor {
			return core.Err(fmt.Sprintf("Reserva de producción del producto %v con divisores distintos.", change.ProductID))
		}
		merged.subUnits += int64(change.SubUnits)
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

	updatedStocks := make([]ProductStock, 0, len(changeByStockID))
	for stockID, merged := range changeByStockID {
		change := merged.change
		stock, exists := stockByID[stockID]
		if !exists {
			// Nothing to reserve from, and nothing reserved to release.
			return core.Err(fmt.Sprintf("El producto %v no tiene stock en el almacén %v para reservar.", change.ProductID, change.WarehouseID))
		}
		// The row keeps its own divisor; the change converts to it. A row still at a coarser divisor
		// than the supply's cannot hold a finer amount until a movement refines it.
		stockDivisor := stockDivisorOf(stock)
		changeQuantity, err := core.Quantity{Units: int32(merged.subUnits / int64(change.SubDivisor)), Sub: int32(merged.subUnits % int64(change.SubDivisor))}.
			ConvertToDivisor(change.SubDivisor, stockDivisor)
		if err != nil {
			return core.Err(fmt.Sprintf("El stock del producto %v en el almacén %v está en una sub-unidad anterior (divisor %v): registre un movimiento para actualizarlo antes de reservar.",
				change.ProductID, change.WarehouseID, stockDivisor))
		}
		changeSubUnits := changeQuantity.TotalSubUnits(stockDivisor)
		reservedSubUnits := core.Quantity{Units: stock.ReservedProductionQuantity, Sub: stock.ReservedProductionSubQuantity}.TotalSubUnits(stockDivisor)
		// Blocked lots (Cuarentena, Rechazado) cannot be consumed, so they cannot be reserved either.
		unblockedSubUnits := core.Quantity{Units: stock.Quantity, Sub: stock.SubQuantity}.
			Add(core.Quantity{Units: stock.DetailQuantity, Sub: stock.DetailSubQuantity}).
			Add(core.Quantity{Units: stock.BlockedQuantity, Sub: stock.BlockedSubQuantity}.Negate()).TotalSubUnits(stockDivisor)

		nextReservedSubUnits := reservedSubUnits + changeSubUnits
		if changeSubUnits > 0 && unblockedSubUnits-reservedSubUnits < changeSubUnits {
			return core.Err(fmt.Sprintf("No hay stock libre del producto %v en el almacén %v para reservar: libre %v, se piden %v (en sub-unidades de %v).",
				change.ProductID, change.WarehouseID, unblockedSubUnits-reservedSubUnits, changeSubUnits, stockDivisor))
		}
		if nextReservedSubUnits < 0 {
			return core.Err(fmt.Sprintf("Reserva de producción negativa: almacén %v producto %v. Ejecute fn-recompute-production-reservation.",
				change.WarehouseID, change.ProductID))
		}
		stock.ReservedProductionQuantity = int32(nextReservedSubUnits / int64(stockDivisor))
		stock.ReservedProductionSubQuantity = int32(nextReservedSubUnits % int64(stockDivisor))
		updatedStocks = append(updatedStocks, *stock)
	}

	stockTable := db.TableOf[ProductStock]()
	// Written with WarehouseID and Status, the delta index's pinned columns. The ORM adds Updated, so
	// the browsers' stock caches pick the change up.
	if err := db.UpdateIn(transaction, &updatedStocks, stockTable.ReservedProductionQuantity, stockTable.ReservedProductionSubQuantity,
		stockTable.Status, stockTable.WarehouseID); err != nil {
		return core.Err("Error al guardar la reserva de producción:", err)
	}
	return nil
}
