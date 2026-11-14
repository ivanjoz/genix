package types

import (
	"app/core"
	logistics "app/logistics/types"
	"fmt"
	"maps"
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

// A delivery of a sale with a bought product (7) and a made one (8): the made line's stock exit waits
// for the production close, so it keeps a reservation on the warehouse the sale ships from.
func TestDeliveryPendingChangesKeepTheMadeLinesReserved(t *testing.T) {
	sale := SaleOrder{Lines: []SaleOrderLine{
		{ProductID: 7, Quantity: packedLine(t, 2, 0, 1), SubDivisor: 1},
		{ProductID: 8, Quantity: packedLine(t, 3, 0, 1), SubDivisor: 1},
	}}
	storedLines := sale.Lines
	sale.MarkStockExitPending(map[int32]bool{8: true})
	if sale.PendingStockMovement != 1 || sale.Lines[0].StockExitPending || !sale.Lines[1].StockExitPending {
		t.Fatalf("flags: %+v", sale)
	}
	if storedLines[1].StockExitPending {
		t.Fatal("MarkStockExitPending must not write through to the caller's lines")
	}

	unitsByProduct := func(changes []logistics.PendingDeliveryChange) map[string]int32 {
		units := map[string]int32{}
		for _, change := range changes {
			units[fmt.Sprintf("w%v/p%v", change.WarehouseID, change.ProductID)] += change.Store + change.Online
		}
		return units
	}
	cases := []struct {
		name                  string
		reservedWarehouseID   int32
		deliveryWarehouseID   int32
		expectedUnitsByChange map[string]int32
	}{
		// A till sale delivered on creation held nothing: only the made line reserves.
		{"delivered on creation", 0, 4, map[string]int32{"w4/p8": 3}},
		// A stored sale delivered from where it reserved: the bought line releases, the made one stays.
		{"same warehouse", 4, 4, map[string]int32{"w4/p7": -2}},
		// Delivered from another warehouse: the made line's reservation moves with the sale.
		{"other warehouse", 4, 5, map[string]int32{"w4/p7": -2, "w4/p8": -3, "w5/p8": 3}},
	}
	for _, testCase := range cases {
		got := unitsByProduct(DeliveryPendingChanges(sale.Lines, testCase.reservedWarehouseID, testCase.deliveryWarehouseID, false))
		if !maps.Equal(got, testCase.expectedUnitsByChange) {
			t.Errorf("%s: got %v, want %v", testCase.name, got, testCase.expectedUnitsByChange)
		}
	}

	sale.MarkStockExitPending(map[int32]bool{})
	if sale.PendingStockMovement != 0 {
		t.Fatal("a sale of bought products only has no pending stock movement")
	}
}
