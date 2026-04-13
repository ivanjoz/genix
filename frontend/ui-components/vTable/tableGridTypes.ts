import type { ElementAST } from '$components/Renderer.svelte';
import type { Snippet } from 'svelte';

export type TableGridCellAlign = 'left' | 'center' | 'right';

export interface TableGridMobileCard<TRecord> {
  order: number;
  elementLeft?: string | ElementAST | ElementAST[];
  elementRight?: string | ElementAST | ElementAST[];
  css?: string;
  icon?: string;
  iconCss?: string;
  labelLeft?: string;
  labelTop?: string;
  // Allow mobile cards to override the default grid cell output with a lighter summary.
  render?: (record: TRecord, rowIndex: number) => string | number | ElementAST | ElementAST[];
  if?: (record: TRecord, rowIndex: number) => boolean;
}

export interface TableGridColumn<TRecord> {
  id: string | number;
  header: string;
  width: string;
  hidden?: boolean;
  useCellRenderer?: boolean;
  splitString?: number;
  align?: TableGridCellAlign;
  headerCss?: string;
  cellCss?: string;
  mobile?: TableGridMobileCard<TRecord>;
  // Business value extractor for the default lightweight render path.
  getValue?: (record: TRecord, rowIndex: number) => string | number;
}

export type TableGridCellRendererSnippet<TRecord> = Snippet<[
  record: TRecord,
  columnDefinition: TableGridColumn<TRecord>,
  rowIndex: number,
]>;

export type TableGridHeaderRendererSnippet<TRecord> = Snippet<[
  columnDefinition: TableGridColumn<TRecord>,
  columnIndex: number,
]>;
