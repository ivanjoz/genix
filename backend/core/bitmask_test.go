package core

import "testing"

func TestBitmask(t *testing.T) {
	var mask int8
	mask = BitmaskAdd(mask, 1)
	mask = BitmaskAdd(mask, 3)
	if mask != 5 {
		t.Fatalf("mask = %v, want 5 (ids 1 and 3 are bits 0 and 2)", mask)
	}
	if !BitmaskHas(mask, 1) || BitmaskHas(mask, 2) || !BitmaskHas(mask, 3) {
		t.Fatalf("mask %v reports the wrong ids", mask)
	}
	if BitmaskAdd(mask, 3) != mask {
		t.Fatalf("adding a set id must not change the mask")
	}
}
