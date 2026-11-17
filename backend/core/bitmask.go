package core

import "golang.org/x/exp/constraints"

// A bitmask column stores a set of option IDs as one integer: option ID n is bit n-1, the same
// convention the frontend SelectMultiple (useBitmask) saves with. IDs go from 1 to the column's
// bit width (7 for an int8).

// BitmaskHas reports whether optionID is set in mask.
func BitmaskHas[T constraints.Integer](mask T, optionID T) bool {
	return mask&(1<<(optionID-1)) != 0
}

// BitmaskAdd returns mask with optionID set.
func BitmaskAdd[T constraints.Integer](mask T, optionID T) T {
	return mask | (1 << (optionID - 1))
}
