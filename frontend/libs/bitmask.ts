// A bitmask column stores a set of option IDs as one integer: option ID n is bit n-1, the same
// convention SelectMultiple (useBitmask) saves with and the backend's core.BitmaskHas reads.

/** Whether optionID is set in mask. */
export function bitmaskHas(mask: number, optionID: number): boolean {
  return (mask & (1 << (optionID - 1))) !== 0
}

/** mask with optionID set. */
export function bitmaskAdd(mask: number, optionID: number): number {
  return mask | (1 << (optionID - 1))
}
