package types

import (
	"app/core"
	logistics "app/logistics/types"
	"slices"
)

// IsDeliveryPending: the sale was registered and its goods have not left the warehouse yet.
func (e *SaleOrder) IsDeliveryPending() bool {
	return e.Status == OrderStatusPending || e.Status == OrderStatusPaid
}

// PendingDeliveryChanges is what the lines reserve on warehouseID's stock rows, times sign: +1 when
// the sale or web order is registered undelivered, -1 when it is delivered or annulled. A web
// order's units go to the Online counter, a till sale's to Store. A line with sub-units reserves one
// more whole unit (3 candies of a box of 6 hold the box), so the counters stay in whole units.
func PendingDeliveryChanges(lines []SaleOrderLine, warehouseID int32, isOnline bool, sign int32) []logistics.PendingDeliveryChange {
	changes := make([]logistics.PendingDeliveryChange, 0, len(lines))
	for _, line := range lines {
		if line.ProductID <= 0 || line.Quantity <= 0 {
			continue
		}
		quantity := core.UnpackQuantityLine(line.Quantity)
		units := quantity.Units + core.If(quantity.Sub > 0, int32(1), 0)
		change := logistics.PendingDeliveryChange{WarehouseID: warehouseID, ProductID: line.ProductID, PresentationID: line.PresentationID}
		if isOnline {
			change.Online = sign * units
		} else {
			change.Store = sign * units
		}
		changes = append(changes, change)
	}
	return changes
}

// MarkStockExitPending flags, on delivery, the lines of made products (madeProductIDs): their stock
// leaves at the daily production close instead of now (docs/PRODUCTION_PLAN.md §3). Sets
// PendingStockMovement to match.
func (e *SaleOrder) MarkStockExitPending(madeProductIDs map[int32]bool) {
	// A copy: the caller's sale often shares its Lines with the stored sale or the web order.
	e.Lines = slices.Clone(e.Lines)
	for lineIndex := range e.Lines {
		e.Lines[lineIndex].StockExitPending = madeProductIDs[e.Lines[lineIndex].ProductID]
	}
	e.RefreshPendingStockMovement()
}

// LinesWithStockExitPending are the lines whose StockExitPending equals isPending.
func LinesWithStockExitPending(lines []SaleOrderLine, isPending bool) []SaleOrderLine {
	selectedLines := []SaleOrderLine{}
	for _, line := range lines {
		if line.StockExitPending == isPending {
			selectedLines = append(selectedLines, line)
		}
	}
	return selectedLines
}

// DeliveryPendingChanges is how a delivery moves the reservations of a sale whose lines were already
// flagged by MarkStockExitPending. A line with its exit done releases what it reserved; a pending
// line keeps one until the close, which settles the sale's lines on sale.WarehouseID, so it moves
// there when the delivery ships from another warehouse. reservedWarehouseID is where the sale held
// its reservation, 0 when it held none (a till sale delivered on creation): its pending lines reserve
// now. isOnline names the counter, as in PendingDeliveryChanges.
func DeliveryPendingChanges(lines []SaleOrderLine, reservedWarehouseID int32, deliveryWarehouseID int32, isOnline bool) []logistics.PendingDeliveryChange {
	pendingLines := LinesWithStockExitPending(lines, true)
	if reservedWarehouseID == 0 {
		return PendingDeliveryChanges(pendingLines, deliveryWarehouseID, isOnline, 1)
	}
	changes := PendingDeliveryChanges(LinesWithStockExitPending(lines, false), reservedWarehouseID, isOnline, -1)
	if reservedWarehouseID != deliveryWarehouseID {
		changes = append(changes, PendingDeliveryChanges(pendingLines, reservedWarehouseID, isOnline, -1)...)
		changes = append(changes, PendingDeliveryChanges(pendingLines, deliveryWarehouseID, isOnline, 1)...)
	}
	return changes
}
