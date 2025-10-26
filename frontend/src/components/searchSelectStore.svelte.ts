/**
 * Shared store for SearchSelect components
 * Controls which mobile search layer is currently open
 */
export let openSearchLayer = $state(0);

export function setOpenSearchLayer(value: number) {
  openSearchLayer = value;
}

