package core

import (
	"math"
	"testing"
)

// The whole design rests on addition being closed without a carry: if this stops holding,
// the stock engine can no longer accumulate movements with a plain + and Scylla can no
// longer SUM() the two columns independently.
func TestAddIsClosedAndOrderIndependent(t *testing.T) {
	const divisor int16 = 6

	// The worked example from the plan: +1 box, +1 box, -4 candies, -4 candies.
	movements := []Quantity{{Units: 1}, {Units: 1}, {Sub: -4}, {Sub: -4}}

	forward := Quantity{}
	for _, movement := range movements {
		forward = forward.Add(movement)
	}
	backward := Quantity{}
	for index := len(movements) - 1; index >= 0; index-- {
		backward = backward.Add(movements[index])
	}

	if forward != backward {
		t.Fatalf("accumulation order changed the result: %+v vs %+v", forward, backward)
	}
	// Unnormalized on purpose — 2 whole units and a negative sub balance.
	if forward != (Quantity{Units: 2, Sub: -8}) {
		t.Fatalf("expected {2,-8} before normalization, got %+v", forward)
	}

	// 2 boxes = 12 candies, 8 sold, 4 remain = 4/6 of a box.
	normalized, err := forward.Normalize(divisor)
	if err != nil {
		t.Fatalf("unexpected normalize error: %v", err)
	}
	if normalized != (Quantity{Units: 0, Sub: 4}) {
		t.Fatalf("expected {0,4}, got %+v", normalized)
	}
}

// Go's / truncates toward zero and % takes the sign of the dividend, so a naive
// implementation returns {0,-4} for -4/6 instead of the canonical {-1,2}. Every borrow
// case below is one the stock engine actually produces.
func TestNormalizeBorrowsAcrossZero(t *testing.T) {
	const divisor int16 = 6

	cases := []struct {
		name     string
		input    Quantity
		expected Quantity
	}{
		{"sell four candies from one whole box", Quantity{Units: 1, Sub: -4}, Quantity{Units: 0, Sub: 2}},
		{"two boxes minus eight candies", Quantity{Units: 2, Sub: -8}, Quantity{Units: 0, Sub: 4}},
		{"oversold by four candies", Quantity{Units: 0, Sub: -4}, Quantity{Units: -1, Sub: 2}},
		{"sub exactly one whole unit carries", Quantity{Units: 0, Sub: 6}, Quantity{Units: 1, Sub: 0}},
		{"sub above the divisor carries", Quantity{Units: 1, Sub: 14}, Quantity{Units: 3, Sub: 2}},
		{"already canonical is unchanged", Quantity{Units: 3, Sub: 5}, Quantity{Units: 3, Sub: 5}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			normalized, err := testCase.input.Normalize(divisor)
			if err != nil {
				t.Fatalf("unexpected normalize error: %v", err)
			}
			if normalized != testCase.expected {
				t.Fatalf("expected %+v, got %+v", testCase.expected, normalized)
			}
			// Normalization must never change the value it represents.
			if normalized.TotalSubUnits(divisor) != testCase.input.TotalSubUnits(divisor) {
				t.Fatalf("normalize changed the value: %v became %v",
					testCase.input.TotalSubUnits(divisor), normalized.TotalSubUnits(divisor))
			}
		})
	}
}

// This is the oversell bug the existing stock engine has: a `Quantity < 0` check reads the
// whole-unit field only, so {0,-4} — four candies oversold — passes it untouched.
func TestIsNegativeCatchesSubUnitOversell(t *testing.T) {
	const divisor int16 = 6

	oversold := Quantity{Units: 0, Sub: -4}
	if oversold.Units < 0 {
		t.Fatal("precondition: the whole-unit field is not negative, which is why the old check missed it")
	}
	if !oversold.IsNegative(divisor) {
		t.Fatal("expected {0,-4} at divisor 6 to be negative (-4/6 of a unit)")
	}

	if (Quantity{Units: 0, Sub: 0}).IsNegative(divisor) {
		t.Fatal("zero must not read as negative")
	}
	// A negative sub balance covered by whole units is still positive overall.
	if (Quantity{Units: 2, Sub: -8}).IsNegative(divisor) {
		t.Fatal("{2,-8} at divisor 6 is +4/6 and must not read as negative")
	}
}

// The stock Status predicates need this without a divisor: a row holding no whole units but
// a non-zero sub balance still holds stock and must not be evicted from the delta sync.
func TestIsZeroDistinguishesLooseSubUnits(t *testing.T) {
	if !(Quantity{}).IsZero() {
		t.Fatal("the empty pair must be zero")
	}
	if (Quantity{Units: 0, Sub: 4}).IsZero() {
		t.Fatal("zero boxes with four loose candies still holds stock")
	}
	if (Quantity{Units: 3, Sub: 0}).IsZero() {
		t.Fatal("three whole units is not zero")
	}
}

func TestCompareQuantities(t *testing.T) {
	const divisor int16 = 6

	cases := []struct {
		name     string
		left     Quantity
		right    Quantity
		expected int
	}{
		{"equal canonical", Quantity{Units: 1, Sub: 2}, Quantity{Units: 1, Sub: 2}, 0},
		{"equal across normalization", Quantity{Units: 1, Sub: 2}, Quantity{Units: 0, Sub: 8}, 0},
		{"sub-unit difference only", Quantity{Units: 1, Sub: 1}, Quantity{Units: 1, Sub: 2}, -1},
		{"whole unit outweighs sub", Quantity{Units: 2}, Quantity{Units: 1, Sub: 5}, 1},
		{"negative below zero", Quantity{Sub: -1}, Quantity{}, -1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if result := CompareQuantities(testCase.left, testCase.right, divisor); result != testCase.expected {
				t.Fatalf("expected %v, got %v", testCase.expected, result)
			}
		})
	}
}

// A divisor may only be refined by an integer factor. Coarsening or moving to a
// non-multiple would silently round real stock away, so it has to be refused rather than
// approximated.
func TestDivisorRefinementRules(t *testing.T) {
	cases := []struct {
		from, to int16
		allowed  bool
	}{
		{6, 12, true},     // every sixth splits into two twelfths
		{6, 6, true},      // no change
		{100, 1000, true}, // centigrams refined to grams
		{1, 6, true},      // a product gaining a sub-unit for the first time
		{12, 6, false},    // 9/12 would become 4.5 sixths
		{6, 5, false},     // 4/6 would become 3.33 fifths
		{6, 9, false},     // 9 is not a multiple of 6
		{0, 6, false},     // invalid divisor
		// The packed line slot caps the divisor at 1000, so grams is as fine as a
		// kilogram-based product can get: there is no room left to refine into half-grams.
		{1000, 2000, false},
	}

	for _, testCase := range cases {
		if allowed := IsQuantityDivisorRefinement(testCase.from, testCase.to); allowed != testCase.allowed {
			t.Fatalf("divisor %v -> %v: expected allowed=%v, got %v",
				testCase.from, testCase.to, testCase.allowed, allowed)
		}
	}
}

// The lazy conversion that lets a divisor change avoid rewriting history: the stock row
// converts itself on the fly when a movement arrives at a finer divisor.
func TestConvertToDivisorMatchesTheStockRowExample(t *testing.T) {
	stock := Quantity{Units: 21, Sub: 4} // 21 boxes + 4/6, divisor 6

	converted, err := stock.ConvertToDivisor(6, 12)
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}
	if converted != (Quantity{Units: 21, Sub: 8}) {
		t.Fatalf("expected {21,8} at divisor 12, got %+v", converted)
	}
	// 4/6 and 8/12 must be the same value.
	if stock.TotalSubUnits(6)*2 != converted.TotalSubUnits(12) {
		t.Fatal("conversion changed the value it represents")
	}

	// A movement of two twelfths then lands with a plain add.
	afterMovement := converted.Add(Quantity{Sub: 2})
	if afterMovement != (Quantity{Units: 21, Sub: 10}) {
		t.Fatalf("expected {21,10} at divisor 12, got %+v", afterMovement)
	}
}

func TestConvertToDivisorRejectsCoarsening(t *testing.T) {
	if _, err := (Quantity{Units: 1, Sub: 9}).ConvertToDivisor(12, 6); err == nil {
		t.Fatal("expected 12 -> 6 to be refused: 9/12 is 4.5 sixths")
	}
	if _, err := (Quantity{Units: 1, Sub: 4}).ConvertToDivisor(6, 5); err == nil {
		t.Fatal("expected 6 -> 5 to be refused: 4/6 is 3.33 fifths")
	}
	if _, err := (Quantity{Units: 1}).ConvertToDivisor(0, 6); err == nil {
		t.Fatal("expected a zero source divisor to be refused")
	}
}

// A silently clamped stock balance is unrecoverable, so the carry out of an unnormalized
// Sub reports instead of saturating.
func TestNormalizeReportsOverflowInsteadOfClamping(t *testing.T) {
	overflowing := Quantity{Units: math.MaxInt32, Sub: math.MaxInt32}
	if _, err := overflowing.Normalize(1); err == nil {
		t.Fatal("expected an overflow error rather than a clamped result")
	}

	if _, err := (Quantity{Units: 1}).Normalize(0); err == nil {
		t.Fatal("expected an invalid-divisor error")
	}
}

// Charging sub-units at their own price is exact; prorating the unit price is not.
func TestQuantityAmountChargesSubUnitsAtTheirOwnPrice(t *testing.T) {
	// A box of six costs 5000 cents; a single candy costs 850.
	fourCandies := Quantity{Units: 0, Sub: 4}
	if amount := QuantityAmount(fourCandies, 5000, 850); amount != 3400 {
		t.Fatalf("expected 3400 cents (4 x 850), got %v", amount)
	}

	// Prorating would have produced 3333.33 — the drift this avoids.
	oneBoxAndTwoCandies := Quantity{Units: 1, Sub: 2}
	if amount := QuantityAmount(oneBoxAndTwoCandies, 5000, 850); amount != 6700 {
		t.Fatalf("expected 6700 cents (5000 + 2 x 850), got %v", amount)
	}

	if amount := QuantityAmount(Quantity{Units: math.MaxInt32}, math.MaxInt32, 0); amount != math.MaxInt32 {
		t.Fatalf("expected saturation at MaxInt32, got %v", amount)
	}
}

// Cost valuation on the ledger has only a purchase price, so the sub-unit part is prorated
// from the combined total rather than rounded separately.
func TestQuantityValueAtUnitPrice(t *testing.T) {
	const divisor int16 = 6

	value, err := QuantityValueAtUnitPrice(Quantity{Units: 2, Sub: 3}, divisor, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2.5 boxes at 5000 = 12500.
	if value != 12500 {
		t.Fatalf("expected 12500, got %v", value)
	}

	// Whole units with no sub-unit part are unaffected by the divisor.
	value, err = QuantityValueAtUnitPrice(Quantity{Units: 3}, QuantityDivisorNone, 700)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 2100 {
		t.Fatalf("expected 2100, got %v", value)
	}

	if _, err := QuantityValueAtUnitPrice(Quantity{Units: 1}, 0, 100); err == nil {
		t.Fatal("expected an invalid-divisor error")
	}
}

// A document line packs both halves into one int32: 1001 at divisor 6 is one box plus one
// candy. The ledger keeps them in separate columns instead, because it has to accumulate
// component-wise and be SUM()-able; a document only has to read back as what was written.
func TestPackAndUnpackQuantityLine(t *testing.T) {
	cases := []struct {
		name     string
		quantity Quantity
		divisor  int16
		packed   int32
	}{
		{"one box plus one candy", Quantity{Units: 1, Sub: 1}, 6, 1001},
		{"one box plus four candies", Quantity{Units: 1, Sub: 4}, 6, 1004},
		{"four candies only", Quantity{Units: 0, Sub: 4}, 6, 4},
		{"whole units only", Quantity{Units: 3}, 6, 3000},
		{"750 grams", Quantity{Units: 0, Sub: 750}, 1000, 750},
		{"one and a half kilos", Quantity{Units: 1, Sub: 500}, 1000, 1500},
		// Packing normalizes, so an over-full sub carries instead of corrupting the slot.
		{"sub above the divisor carries", Quantity{Units: 1, Sub: 8}, 6, 2002},
		{"1500 grams becomes one kilo and a half", Quantity{Units: 0, Sub: 1500}, 1000, 1500},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			packed, err := PackQuantityLine(testCase.quantity, testCase.divisor)
			if err != nil {
				t.Fatalf("unexpected pack error: %v", err)
			}
			if packed != testCase.packed {
				t.Fatalf("expected %v, got %v", testCase.packed, packed)
			}
			// The round trip must land on the same value, not just the same pair.
			unpacked := UnpackQuantityLine(packed)
			if unpacked.TotalSubUnits(testCase.divisor) != testCase.quantity.TotalSubUnits(testCase.divisor) {
				t.Fatalf("round trip changed the value: %v became %v",
					testCase.quantity.TotalSubUnits(testCase.divisor),
					unpacked.TotalSubUnits(testCase.divisor))
			}
		})
	}
}

// The divisor ceiling is the packed slot, not a preference: a normalized Sub has to fit in
// 0..999, and divisor 1000 (a kilogram sold by the gram) fills it exactly.
func TestQuantityLineScaleBoundsTheDivisor(t *testing.T) {
	if MaxQuantityDivisor != int16(QuantityLineScale) {
		t.Fatalf("the divisor ceiling must equal the packing slot, got %v vs %v",
			MaxQuantityDivisor, QuantityLineScale)
	}
	if err := ValidateQuantityDivisor(1000); err != nil {
		t.Fatalf("divisor 1000 (grams per kilogram) must be allowed: %v", err)
	}
	if err := ValidateQuantityDivisor(1001); err == nil {
		t.Fatal("a divisor past the packed slot must be refused")
	}

	// A normalized Sub can never collide with the whole-unit part of the packed value.
	maxSub, err := (Quantity{Sub: int32(MaxQuantityDivisor) - 1}).Normalize(MaxQuantityDivisor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if maxSub.Sub >= QuantityLineScale {
		t.Fatalf("a normalized Sub of %v would overflow the packing slot", maxSub.Sub)
	}
}

func TestPackQuantityLineRejectsOutOfRangeLines(t *testing.T) {
	if _, err := PackQuantityLine(Quantity{Sub: -1}, 6); err == nil {
		t.Fatal("expected a negative line quantity to be refused")
	}
	if _, err := PackQuantityLine(Quantity{Units: MaxQuantityLineUnits + 1}, 6); err == nil {
		t.Fatal("expected a line past the packing ceiling to be refused")
	}
	// The ceiling is per line only — aggregates keep the halves in separate columns.
	if _, err := PackQuantityLine(Quantity{Units: MaxQuantityLineUnits}, 6); err != nil {
		t.Fatalf("the ceiling itself must pack: %v", err)
	}
}
