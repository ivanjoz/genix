import type { Snippet } from 'svelte';

export type TableGridCellAlign = 'left' | 'center' | 'right';

export interface TableGridColumn<TRecord> {
  id: string | number;
  header: string;
  width: string;
  hidden?: boolean;
  useCellRenderer?: boolean;
  align?: TableGridCellAlign;
  headerCss?: string;
  cellCss?: string;
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
