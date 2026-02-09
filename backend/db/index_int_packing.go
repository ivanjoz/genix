package db

import "fmt"

// packedIndexInfo stores schema-time metadata for TableSchema.Indexes packed local indexes.
// It is used for capability generation and for query WHERE rewriting.
type packedIndexInfo struct {
	indexName         string
	packedColumnName  string
	sourceColumnNames []string
	// partitionColumnName is non-empty when the packed index is local (partition + packed).
	// Empty means "global index on packed column only".
	partitionColumnName string
	// slotDigitsPerColumn aligns with sourceColumnNames (same length).
	slotDigitsPerColumn []int64
	// totalDigits is 9 for int32 packed and 19 for int64 packed.
	totalDigits int64
	// isInt32Packed indicates the packed column is CQL int (Go int32).
	isInt32Packed bool
}

func countBase10DigitsNonNegative(value int64) int64 {
	if value < 0 {
		panic(fmt.Sprintf("countBase10DigitsNonNegative: negative value %d", value))
	}
	if value == 0 {
		return 1
	}
	digits := int64(0)
	for value > 0 {
		value /= 10
		digits++
	}
	return digits
}

func trimRightToDigitsNonNegative(value int64, maxDigits int64) int64 {
	if value < 0 {
		panic(fmt.Sprintf("trimRightToDigitsNonNegative: negative value %d", value))
	}
	if maxDigits <= 0 {
		panic(fmt.Sprintf("trimRightToDigitsNonNegative: invalid maxDigits=%d", maxDigits))
	}
	digits := countBase10DigitsNonNegative(value)
	if digits <= maxDigits {
		return value
	}
	trimDigits := digits - maxDigits
	return value / Pow10Int64(trimDigits)
}

func computePackedInt64ValueNonNegative(componentValues []int64, slotDigitsPerColumn []int64) int64 {
	if len(componentValues) != len(slotDigitsPerColumn) {
		panic("computePackedInt64ValueNonNegative: componentValues and slotDigitsPerColumn length mismatch")
	}

	// Precompute suffix digit shifts from right to left:
	// shift[i] = sum(slotDigitsPerColumn[i+1:]).
	shiftDigits := make([]int64, len(slotDigitsPerColumn))
	suffix := int64(0)
	for i := len(slotDigitsPerColumn) - 1; i >= 0; i-- {
		shiftDigits[i] = suffix
		suffix += slotDigitsPerColumn[i]
	}

	var packed int64
	for i, value := range componentValues {
		if value < 0 {
			panic(fmt.Sprintf("computePackedInt64ValueNonNegative: negative component value %d", value))
		}
		slotDigits := slotDigitsPerColumn[i]
		valueTrimmed := trimRightToDigitsNonNegative(value, slotDigits)
		packed += valueTrimmed * Pow10Int64(shiftDigits[i])
	}
	return packed
}
