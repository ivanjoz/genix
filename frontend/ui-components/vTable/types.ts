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
	contentCss?: string
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
	disableCellInteractions?: (record: T, idx?: number) => boolean
	onCellEdit?: (record: T, value: string | number, rerender: () => void) => void
	onCellSelect?: (record: T, value: string | number, rerender: () => void) => void
	onCellClick?: (record: T, idx: number, rerender: () => void) => void
	showEditIcon?: boolean
  getValue?: (record: T, idx: number) => string | number
  renderPrefix?: (record: T, idx: number) => string | ElementAST | ElementAST[] | false
  render?: (record: T, idx: number) => string | ElementAST | ElementAST[]
  if?: (record: T, idx: number) => boolean
}

export interface ICardButtonDeleteHandler<T> {
  (record: T, idx: number): void
}

export type TableGridCellAlign = 'left' | 'center' | 'right';

export type MobileCardsListVariant = 'compact' | 'cards';

export interface IMobileCardsListCell<T, TSource = unknown> {
  id?: number | string
  source?: TSource
  hidden?: boolean
  field?: string
  type?: string
  css?: string
  inputCss?: string
  contentCss?: string
  label?: string | (() => string)
  labelCss?: string
  itemCss?: string
  cellCss?: string
  setCellCss?: (record: T) => string | undefined
  cellOptions?: any[]
  cellOptionsKeyId?: string
  cellOptionsKeyName?: string
  disableCellInteractions?: (record: T, idx?: number) => boolean
  onCellEdit?: (record: T, value: string | number, rerender: () => void) => void
  onCellSelect?: (record: T, value: string | number, rerender: () => void) => void
  onCellClick?: (record: T, idx: number, rerender: () => void) => void
  showEditIcon?: boolean
  getValue?: (record: T, idx: number) => string | number
  renderPrefix?: (record: T, idx: number) => string | ElementAST | ElementAST[] | false
  render?: (record: T, idx: number) => string | ElementAST | ElementAST[]
  mobileRender?: (record: T, idx: number) => string | number | ElementAST | ElementAST[]
  splitString?: number
  if?: (record: T, idx: number) => boolean
  labelLeft?: string
  labelTop?: string
  icon?: string
  iconCss?: string
  elementLeft?: string | ElementAST | ElementAST[]
  elementRight?: string | ElementAST | ElementAST[]
  useRenderer?: boolean
}

export interface ITableColumn<T> {
  id?: number | string
  header: string | (() => string)
  // Grid mode uses an explicit width track; table mode can ignore it.
  width?: string
  hidden?: boolean
  // Grid mode can render snippet-only cells without affecting VTable defaults.
  useCellRenderer?: boolean
  // Split compact text cells into two lines when the grid layout gets tight.
  splitString?: number
  align?: TableGridCellAlign
  headerCss?: string
  // Class hook for the inner header wrapper (<th><div>...</div></th>)
  headerInnerCss?: string
  headerStyle?: Record<string, string>
  cellStyle?: Record<string, string>
  css?: string
	inputCss?: string
	cellCss?: string
	cellInputType?: 'number'
	// Allow columns to inject runtime cell classes per record
	setCellCss?: (record: T) => string | undefined
  cardCss?: string
  field?: string
  subcols?: ITableColumn<T>[]
	cardColumn?: [number, (1|2|3)?]
	cellOptions?: any[]
	cellOptionsKeyId?: string
	cellOptionsKeyName?: string
	onBeforeCellChange?: (e: T, value: string|number) => boolean
	onCellEdit?: (e: T, value: string|number, rerender: () => void) => void
	disableCellInteractions?: (e:T, idx?: number) => boolean
	onCellSelect?: (e: T, value: string|number, rerender: () => void) => void
	onCellClick?: (e: T, idx: number, rerender: () => void) => void
	showEditIcon?: boolean
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
  boolean,                        // isMobileCardView
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

export type TableGridCellRendererSnippet<T> = Snippet<[
  record: T,
  columnDefinition: ITableColumn<T>,
  rowIndex: number,
]>;

export type TableGridHeaderRendererSnippet<T> = Snippet<[
  columnDefinition: ITableColumn<T>,
  columnIndex: number,
]>;

export type MobileCardsListRendererSnippet<T, TCell> = Snippet<[
  record: T,
  cell: TCell,
  defaultContent: any,
  rowIndex: number,
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
