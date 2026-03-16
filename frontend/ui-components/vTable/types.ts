/**
 * Type definitions for Svelte 5 Virtual Scroller
 */

import type { ElementAST } from '$components/Renderer.svelte';
import type { Snippet } from 'svelte';

export interface ITableMobileCard<T> {
	order: number,
	elementLeft?: string | ElementAST | ElementAST[] 
	elementRight?: string | ElementAST | ElementAST[]
	css?: string
	icon?: string
	iconCss?: string
	render?: (e: T, idx: number) => string | ElementAST | ElementAST[]
	labelLeft?: string
	labelTop?: string
	if?: (e: T, idx: number) => boolean
}

export interface ICardCell<T> {
  id?: number | string
  label: string | (() => string)
  hidden?: boolean
  field?: string
  type?: string
  css?: string
	inputCss?: string
	contentCss?: string
	labelCss?: string
	itemCss?: string
	cellCss?: string
	// Allow cards to inject runtime field classes per record.
	setCellCss?: (record: T) => string | undefined
	cellOptions?: any[]
	cellOptionsKeyId?: string
	cellOptionsKeyName?: string
	onCellEdit?: (record: T, value: string | number) => void
	onCellSelect?: (record: T, value: string | number) => void
  getValue?: (record: T, idx: number) => string | number
  renderPrefix?: (record: T, idx: number) => string | ElementAST | ElementAST[] | false
  render?: (record: T, idx: number) => string | ElementAST | ElementAST[]
  if?: (record: T, idx: number) => boolean
}

export interface ICardButtonDeleteHandler<T> {
  (record: T, idx: number): void
}

export interface ITableColumn<T> {
  id?: number | string
  header: string | (() => string)
  hidden?: boolean
  headerCss?: string
  // Class hook for the inner header wrapper (<th><div>...</div></th>)
  headerInnerCss?: string
  headerStyle?: Record<string, string>
  cellStyle?: Record<string, string>
  css?: string
	inputCss?: string
	cellCss?: string
	// Allow columns to inject runtime cell classes per record
	setCellCss?: (record: T) => string | undefined
  cardCss?: string
  field?: string
  subcols?: ITableColumn<T>[]
	cardColumn?: [number, (1|2|3)?]
	cellOptions?: any[]
	cellOptionsKeyId?: string
	cellOptionsKeyName?: string
	onCellEdit?: (e:T, value: string|number) => void
	onCellSelect?: (e:T, value: string|number) => void
  cardRender?: (e: T, idx: number) => (any)
  getValue?: (e: T, idx: number) => (string|number)
  // Renders extra visual content before the main cell output without replacing it.
  renderPrefix?: (e: T, idx: number) => string | ElementAST | ElementAST[] | false
  render?: (e: T, idx: number) => string | ElementAST | ElementAST[]
  renderHTML?: (e: T, idx: number) => string
  _colspan?: number
	highlight?: boolean
	/* Buttons */
	buttonEditHandler?: (e:T) => void
	buttonDeleteHandler?: (e:T) => void
	buttonEditIf?: (e:T) => boolean
	buttonDeleteIf?: (e:T) => boolean
	mobile?: ITableMobileCard<T>
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

/**
 * Card renderer snippet type for rendering Svelte components inside card cells.
 */
export type CardRendererSnippet<T> = Snippet<[
  T,                              // record
  ICardCell<T>,                   // cell
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
