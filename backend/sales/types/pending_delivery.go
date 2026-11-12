package types

import (
	"app/core"
	logistics "app/logistics/types"
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
