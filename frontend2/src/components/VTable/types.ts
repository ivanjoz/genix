/**
 * Type definitions for Svelte 5 Virtual Scroller
 */

import type { ElementAST } from '$components/micro/Renderer.svelte';
import type { Snippet } from 'svelte';

export interface ITableColumn<T> {
  id?: number | string
  header: string | (() => string)
  headerCss?: string
  headerStyle?: Record<string, string>
  cellStyle?: Record<string, string>
  css?: string
	inputCss?: string
	cellCss?: string
  cardCss?: string
  field?: string
  subcols?: ITableColumn<T>[]
  cardColumn?: [number, (1|2|3)?]
	cellOptions?: any[]
	onCellEdit?: (e:T, value: string|number) => void
	onCellSelect?: (e:T, value: string|number) => void
  cardRender?: (e: T, idx: number) => (any)
  getValue?: (e: T, idx: number) => (string|number)
	
  render?: (e: T, idx: number) => string | ElementAST | ElementAST[]
  _colspan?: number
	highlight?: boolean
	/* Buttons */
	buttonEditHandler?: (e:T) => void
	buttonDeleteHandler?: (e:T) => void
}

/**
 * Cell renderer function type for global cell overrides (legacy - use CellRendererSnippet for Svelte components)
 */
export type CellRendererFn<T> = (
  record: T,
  columnId: string | number | undefined,
  columnField: string | undefined,
  rowIndex: number,
  defaultContent: any
) => any | null;

/**
 * Cell renderer snippet type for rendering Svelte components in cells
 */
export type CellRendererSnippet<T> = Snippet<[
  T,                              // record
  ITableColumn<T>,    						// column
  any,                            // defaultContent
	number,                         // rowIndex
]>;

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
