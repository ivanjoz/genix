/**
 * Type definitions for Svelte 5 Virtual Scroller
 */

export interface VirtualItem {
	/** Index of the item in the data array */
	index: number;
	/** Start position in pixels from the top */
	start: number;
	/** Height of the item in pixels */
	size: number;
	/** End position in pixels from the top */
	end: number;
	/** Unique key for the item */
	key: string | number;
}

export interface VirtualizerOptions {
	/** Total number of items to virtualize */
	count: number;
	/** Function that returns the scrollable container element */
	getScrollElement: () => HTMLElement | null;
	/** Function that returns the estimated size for an item at the given index */
	estimateSize: (index: number) => number;
	/** Number of items to render outside of the visible area (default: 3) */
	overscan?: number;
}

export interface VirtualizerStore {
	/** Subscribe to scroll changes */
	subscribe: (callback: () => void) => () => void;
	/** Get the list of virtual items to render */
	getVirtualItems: () => VirtualItem[];
	/** Get the total size of all items */
	getTotalSize: () => number;
	/** Force a refresh/recalculation of virtual items */
	refresh: () => void;
}

