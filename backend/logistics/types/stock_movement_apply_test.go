package types

import (
	"app/core"
	"testing"
)

// A stock row's DetailSubQuantity is the absolute sum of its detail rows, so refining the
// row without refining every detail under it would leave the sum adding sixths to twelfths.
func TestRefineStockDivisorConvertsDetailsAtomically(t *testing.T) {
	stock := &ProductStock{
		ID:                21,
		SubDivisor:        6,
		Quantity:          21,
		SubQuantity:       4,
		DetailQuantity:    3,
		DetailSubQuantity: 5,
	}
	details := []*ProductStockDetail{
		{ProductStockID: 21, Quantity: 2, SubQuantity: 3},
		{ProductStockID: 21, Quantity: 1, SubQuantity: 2},
	}

	if err := refineStockDivisor(stock, details, 12, 555, 7); err != nil {
		t.Fatalf("unexpected refinement error: %v", err)
	}

	if stock.SubDivisor != 12 {
		t.Fatalf("expected the row to record divisor 12, got %v", stock.SubDivisor)
	}
	// 21 + 4/6 is 21 + 8/12; whole units never move.
	if stock.Quantity != 21 || stock.SubQuantity != 8 {
		t.Fatalf("expected free bucket {21,8}, got {%v,%v}", stock.Quantity, stock.SubQuantity)
	}
	if stock.DetailQuantity != 3 || stock.DetailSubQuantity != 10 {
		t.Fatalf("expected detail bucket {3,10}, got {%v,%v}", stock.DetailQuantity, stock.DetailSubQuantity)
	}

	expectedDetails := []core.Quantity{{Units: 2, Sub: 6}, {Units: 1, Sub: 4}}
	for i, detail := range details {
		if detail.Quantity != expectedDetails[i].Units || detail.SubQuantity != expectedDetails[i].Sub {
			t.Fatalf("detail %v: expected %+v, got {%v,%v}",
				i, expectedDetails[i], detail.Quantity, detail.SubQuantity)
		}
		// An unstamped detail is skipped by the write step and would stay at the old divisor.
		if detail.Updated != 555 || detail.UpdatedBy != 7 {
			t.Fatalf("detail %v was not stamped dirty: Updated=%v UpdatedBy=%v",
				i, detail.Updated, detail.UpdatedBy)
		}
	}

	// The rolled-up detail bucket must still equal the sum of the converted details.
	summedSub := int32(0)
	summedUnits := int32(0)
	for _, detail := range details {
		summedUnits += detail.Quantity
		summedSub += detail.SubQuantity
	}
	if summedUnits != stock.DetailQuantity || summedSub != stock.DetailSubQuantity {
		t.Fatalf("detail rollup drifted: sum {%v,%v} vs row {%v,%v}",
			summedUnits, summedSub, stock.DetailQuantity, stock.DetailSubQuantity)
	}
}

func TestRefineStockDivisorRejectsCoarsening(t *testing.T) {
	stock := &ProductStock{ID: 1, SubDivisor: 12, Quantity: 1, SubQuantity: 9}
	if err := refineStockDivisor(stock, nil, 6, 1, 1); err == nil {
		t.Fatal("expected 12 -> 6 to be refused: 9/12 is 4.5 sixths")
	}
	// The row must be left untouched when the conversion is refused.
	if stock.SubDivisor != 12 || stock.SubQuantity != 9 {
		t.Fatalf("a refused refinement mutated the row: %+v", stock)
	}
}

// A row that has never carried sub-units adopts whatever the next movement brings, and
// converting to the divisor it already holds is a no-op rather than an error.
func TestRefineStockDivisorAdoptsAndNoOps(t *testing.T) {
	fresh := &ProductStock{ID: 1, Quantity: 5}
	if err := refineStockDivisor(fresh, nil, 6, 1, 1); err != nil {
		t.Fatalf("unexpected error adopting a divisor: %v", err)
	}
	if fresh.SubDivisor != 6 || fresh.Quantity != 5 || fresh.SubQuantity != 0 {
		t.Fatalf("adoption changed the value: %+v", fresh)
	}

	same := &ProductStock{ID: 1, SubDivisor: 6, Quantity: 2, SubQuantity: 3}
	if err := refineStockDivisor(same, nil, 6, 1, 1); err != nil {
		t.Fatalf("unexpected error on a same-divisor refinement: %v", err)
	}
	if same.Quantity != 2 || same.SubQuantity != 3 {
		t.Fatalf("same-divisor refinement changed the value: %+v", same)
	}
}

// Guessing a divisor would be silently wrong for exactly the products this feature exists
// for, so a movement carrying sub-units without one is refused.
func TestMovementDivisorOf(t *testing.T) {
	if _, err := movementDivisorOf(&InternalMovement{SubQuantity: 4}); err == nil {
		t.Fatal("expected sub-units with no divisor to be refused")
	}

	wholeUnitsOnly, err := movementDivisorOf(&InternalMovement{Quantity: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wholeUnitsOnly != core.QuantityDivisorNone {
		t.Fatalf("expected a whole-unit movement to report divisor %v, got %v",
			core.QuantityDivisorNone, wholeUnitsOnly)
	}

	if _, err := movementDivisorOf(&InternalMovement{SubQuantity: 1, SubDivisor: core.MaxQuantityDivisor + 1}); err == nil {
		t.Fatal("expected an out-of-range divisor to be refused")
	}
}

func TestStockDivisorOfDefaultsToNone(t *testing.T) {
	if divisor := stockDivisorOf(&ProductStock{}); divisor != core.QuantityDivisorNone {
		t.Fatalf("expected %v for a row that never carried sub-units, got %v",
			core.QuantityDivisorNone, divisor)
	}
	if divisor := stockDivisorOf(&ProductStock{SubDivisor: 1000}); divisor != 1000 {
		t.Fatalf("expected 1000, got %v", divisor)
	}
}
