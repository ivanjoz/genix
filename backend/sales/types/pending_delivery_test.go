package types

import (
	"app/core"
	"testing"
)

func packedLine(t *testing.T, units, sub int32, divisor int16) int32 {
	t.Helper()
	packed, err := core.PackQuantityLine(core.Quantity{Units: units, Sub: sub}, divisor)
	if err != nil {
		t.Fatal(err)
	}
	return packed
}

func TestPendingDeliveryChangesRoundsSubUnitsUpAndPicksTheCounter(t *testing.T) {
	lines := []SaleOrderLine{
		{ProductID: 7, Quantity: packedLine(t, 2, 0, 1), SubDivisor: 1},
		// Three candies of a box of six hold one whole box.
		{ProductID: 8, PresentationID: 3, Quantity: packedLine(t, 1, 3, 6), SubDivisor: 6},
		{ProductID: 9, Quantity: packedLine(t, 0, 3, 6), SubDivisor: 6},
		{ProductID: 0, Quantity: packedLine(t, 5, 0, 1)},
	}

	tillRelease := PendingDeliveryChanges(lines, 4, false, -1)
	if len(tillRelease) != 3 {
		t.Fatalf("expected 3 changes (the line without product skipped), got %v", tillRelease)
	}
	expectedStore := []int32{-2, -2, -1}
	for index, change := range tillRelease {
		if change.WarehouseID != 4 || change.Online != 0 || change.Store != expectedStore[index] {
			t.Fatalf("line %v: got %+v, want Store %v on warehouse 4", index, change, expectedStore[index])
		}
	}
	if tillRelease[1].PresentationID != 3 {
		t.Fatalf("presentation lost: %+v", tillRelease[1])
	}

	webReserve := PendingDeliveryChanges(lines, 4, true, 1)
	if webReserve[0].Online != 2 || webReserve[0].Store != 0 {
		t.Fatalf("a web order reserves on Online: %+v", webReserve[0])
	}
}

func TestIsDeliveryPending(t *testing.T) {
	for status, expected := range map[int8]bool{
		OrderStatusAnnulled: false, OrderStatusPending: true, OrderStatusPaid: true,
		OrderStatusDelivered: false, OrderStatusCompleted: false,
	} {
		sale := SaleOrder{Status: status}
		if sale.IsDeliveryPending() != expected {
			t.Fatalf("status %v: IsDeliveryPending = %v", status, !expected)
		}
	}
}
