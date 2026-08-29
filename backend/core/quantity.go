package core

import (
	"fmt"
	"math"
)

// Quantity is the (whole units, sub-units) pair every product quantity is expressed in.
// See docs/QUANTITY_SCALE_PLAN.md for the full rationale.
//
// The real value is Units + Sub/Divisor, an exact rational. 0.750 kg at divisor 1000 is
// {0, 750}; one box plus four candies at divisor 6 is {1, 4}; a product with no sub-unit
// is {N, 0} at any divisor.
//
// Units is NEVER scaled. All fractional information lives in Sub/Divisor, so there is no
// per-product "is this discrete or continuous" classification to make at creation time and
// get wrong later — continuous goods are simply divisor 1000.
//
// Sub is deliberately NOT kept normalized in storage: it may exceed the divisor and it may
// be negative. That is what makes addition closed without carrying —
// {a1,b1} + {a2,b2} = {a1+a2, b1+b2} — so the stock engine accumulates with a plain
// integer + and Scylla can still SUM() both columns independently. Normalization happens
// only at validation and display boundaries.
//
// This is a computation type, not a column type: the tables keep two flat int32 columns
// (Quantity, SubQuantity) plus the divisor, and code assembles a Quantity from them.
type Quantity struct {
	Units int32
	Sub   int32
}

// QuantityDivisorNone is the divisor of a product that has no sub-unit configured.
const QuantityDivisorNone int16 = 1

// QuantityLineScale is the packing slot for a document line. A sale order stores a line
// quantity as Units*QuantityLineScale + Sub in one int32, so 1001 at divisor 6 is one box
// plus one candy. Documents are written once and read back whole — they never need the
// component-wise accumulation or the Scylla-side SUM() that the movement ledger does, so
// they pay one field instead of two.
const QuantityLineScale int32 = 1000

// MaxQuantityDivisor follows directly: a normalized Sub must fit the packed slot, which
// holds 0..999. A kilogram sold by the gram is exactly divisor 1000 and exactly fills it.
const MaxQuantityDivisor int16 = int16(QuantityLineScale)

// MaxQuantityLineUnits is the whole-unit ceiling the packing imposes on a single document
// line. Aggregates never pay it: stock, the ledger and the summaries keep the two halves in
// separate columns, so only a per-line value is bounded here.
const MaxQuantityLineUnits int32 = math.MaxInt32 / QuantityLineScale

// Add is the closed component-wise addition described above. No carry, no borrow, and
// order-independent, so a sequence of movements can be accumulated in any order and
// normalized once at the end.
func (quantity Quantity) Add(other Quantity) Quantity {
	return Quantity{Units: quantity.Units + other.Units, Sub: quantity.Sub + other.Sub}
}

// Negate flips the sign of both components, for posting an outbound movement.
func (quantity Quantity) Negate() Quantity {
	return Quantity{Units: -quantity.Units, Sub: -quantity.Sub}
}

// IsZero reports whether the pair holds nothing, without needing a divisor. Used by the
// stock Status predicates, where a row with 0 whole units but a non-zero sub-unit balance
// still holds stock and must not be evicted.
func (quantity Quantity) IsZero() bool {
	return quantity.Units == 0 && quantity.Sub == 0
}

// TotalSubUnits expresses the whole value in sub-units. This is the workhorse for every
// comparison: int64 makes it overflow-proof (int32 units × int16 divisor tops out around
// 7e13), so IsNegative and CompareQuantities never need to normalize first and never fail.
func (quantity Quantity) TotalSubUnits(divisor int16) int64 {
	return int64(quantity.Units)*int64(divisor) + int64(quantity.Sub)
}

// IsNegative reports whether the real value is below zero. A plain Units < 0 check misses
// the case that matters — {0, -4} at divisor 6 is -4/6 of a unit, i.e. oversold, while its
// Units field is exactly zero.
func (quantity Quantity) IsNegative(divisor int16) bool {
	return quantity.TotalSubUnits(divisor) < 0
}

// CompareQuantities returns -1, 0 or 1. Both operands must already be expressed at the
// same divisor; convert with ConvertToDivisor first if they are not.
func CompareQuantities(quantityA Quantity, quantityB Quantity, divisor int16) int {
	totalA, totalB := quantityA.TotalSubUnits(divisor), quantityB.TotalSubUnits(divisor)
	switch {
	case totalA < totalB:
		return -1
	case totalA > totalB:
		return 1
	}
	return 0
}

// Normalize returns the canonical form, where Sub lands in [0, divisor) and the carry or
// borrow moves into Units. {1,-4} at divisor 6 normalizes to {0,2}; {2,-8} to {0,4};
// {0,-4} to {-1,2}. Storage never requires this — it is for display, and for any check
// that wants a canonical pair rather than a comparison.
func (quantity Quantity) Normalize(divisor int16) (Quantity, error) {
	if err := ValidateQuantityDivisor(divisor); err != nil {
		return Quantity{}, err
	}

	// Euclidean division, so the remainder is non-negative even for a negative total.
	// Go's / truncates toward zero and % takes the sign of the dividend, which would
	// yield {0,-4} for -4/6 instead of the canonical {-1,2}.
	totalSubUnits := quantity.TotalSubUnits(divisor)
	normalizedUnits := totalSubUnits / int64(divisor)
	normalizedSub := totalSubUnits % int64(divisor)
	if normalizedSub < 0 {
		normalizedUnits--
		normalizedSub += int64(divisor)
	}

	// An unnormalized Sub can grow far past the divisor before anyone normalizes, so the
	// carry into Units is the one place this can leave int32 range. Report it rather than
	// saturate: a silently clamped stock balance is unrecoverable.
	if normalizedUnits > math.MaxInt32 || normalizedUnits < math.MinInt32 {
		return Quantity{}, Err(fmt.Sprintf(
			"La cantidad normalizada excede el rango permitido: %v unidades (divisor %v).",
			normalizedUnits, divisor))
	}

	return Quantity{Units: int32(normalizedUnits), Sub: int32(normalizedSub)}, nil
}

// ValidateQuantityDivisor rejects the divisors no arithmetic here can work with.
func ValidateQuantityDivisor(divisor int16) error {
	if divisor < 1 {
		return Err(fmt.Sprintf("El divisor de sub-unidad debe ser mayor a 0. Se recibió: %v.", divisor))
	}
	if divisor > MaxQuantityDivisor {
		return Err(fmt.Sprintf("El divisor de sub-unidad no puede superar %v. Se recibió: %v.",
			MaxQuantityDivisor, divisor))
	}
	return nil
}

// IsQuantityDivisorRefinement reports whether a divisor change is exact. A divisor may only
// be refined by an integer factor: 6 → 12 splits every sixth into two twelfths and loses
// nothing, while 12 → 6 (9/12 becomes 4.5 sixths) and 6 → 5 (4/6 becomes 3.33 fifths) do
// not divide and would silently round away real stock.
func IsQuantityDivisorRefinement(fromDivisor int16, toDivisor int16) bool {
	if ValidateQuantityDivisor(fromDivisor) != nil || ValidateQuantityDivisor(toDivisor) != nil {
		return false
	}
	return toDivisor%fromDivisor == 0
}

// ConvertToDivisor re-expresses the pair at a finer divisor. Units are untouched because
// only the fractional part changes granularity: {21,4} at divisor 6 becomes {21,8} at
// divisor 12, since 4/6 and 8/12 are the same value.
//
// This is what lets a divisor change stay lazy — a stock row records the divisor it was
// last written under and converts on the fly when a movement arrives at a finer one, so no
// bulk rewrite of history is ever needed.
func (quantity Quantity) ConvertToDivisor(fromDivisor int16, toDivisor int16) (Quantity, error) {
	if err := ValidateQuantityDivisor(fromDivisor); err != nil {
		return Quantity{}, err
	}
	if err := ValidateQuantityDivisor(toDivisor); err != nil {
		return Quantity{}, err
	}
	if fromDivisor == toDivisor {
		return quantity, nil
	}
	if !IsQuantityDivisorRefinement(fromDivisor, toDivisor) {
		return Quantity{}, Err(fmt.Sprintf(
			"El divisor de sub-unidad sólo puede refinarse a un múltiplo. No se puede convertir de %v a %v.",
			fromDivisor, toDivisor))
	}

	refinementFactor := int64(toDivisor / fromDivisor)
	convertedSub := int64(quantity.Sub) * refinementFactor
	if convertedSub > math.MaxInt32 || convertedSub < math.MinInt32 {
		return Quantity{}, Err(fmt.Sprintf(
			"La conversión de divisor %v a %v desborda las sub-unidades: %v.",
			fromDivisor, toDivisor, convertedSub))
	}

	return Quantity{Units: quantity.Units, Sub: int32(convertedSub)}, nil
}

// QuantityAmount is the line total in cents for a sale priced with an explicit per-sub-unit
// price. Charging sub-units at their own price is exact, where prorating the unit price
// would not be: four candies from a 5000-cent box of six is 3333.33 cents prorated, but an
// exact multiple of the candy's own price.
//
// The caller is responsible for rejecting a sub-unit line whose subUnitPrice is unset —
// that validation belongs with the product, not here.
func QuantityAmount(quantity Quantity, unitPrice int32, subUnitPrice int32) int32 {
	unitsAmount := int64(quantity.Units) * int64(unitPrice)
	subAmount := int64(quantity.Sub) * int64(subUnitPrice)
	return saturateInt32(unitsAmount + subAmount)
}

// QuantityValueAtUnitPrice values a quantity at a single per-unit price, prorating the
// sub-unit part. This is for cost valuation on the movement ledger, where there is only a
// purchase price and no separate sub-unit price to charge against.
func QuantityValueAtUnitPrice(quantity Quantity, divisor int16, unitPrice int32) (int32, error) {
	if err := ValidateQuantityDivisor(divisor); err != nil {
		return 0, err
	}
	// Prorate from the total in sub-units so the whole and fractional parts round once
	// together, instead of rounding the fraction separately and drifting.
	value := quantity.TotalSubUnits(divisor) * int64(unitPrice) / int64(divisor)
	return saturateInt32(value), nil
}

// PackQuantityLine folds a quantity into the single int32 a document line stores. It
// normalizes first, so the packed value always reads back as the same rational.
func PackQuantityLine(quantity Quantity, divisor int16) (int32, error) {
	normalized, err := quantity.Normalize(divisor)
	if err != nil {
		return 0, err
	}
	if normalized.Units < 0 {
		return 0, Err(fmt.Sprintf(
			"La cantidad de una línea no puede ser negativa: %v unidades (divisor %v).",
			normalized.Units, divisor))
	}
	if normalized.Units > MaxQuantityLineUnits {
		return 0, Err(fmt.Sprintf(
			"La cantidad de una línea excede el máximo de %v unidades. Se recibió: %v.",
			MaxQuantityLineUnits, normalized.Units))
	}
	return normalized.Units*QuantityLineScale + normalized.Sub, nil
}

// UnpackQuantityLine splits a stored line quantity back into the pair. The result is
// already normalized, because PackQuantityLine only ever writes a normalized value.
func UnpackQuantityLine(packedQuantity int32) Quantity {
	return Quantity{
		Units: packedQuantity / QuantityLineScale,
		Sub:   packedQuantity % QuantityLineScale,
	}
}

func saturateInt32(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
