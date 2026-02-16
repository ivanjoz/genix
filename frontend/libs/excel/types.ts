import type { ITableColumn } from '$components/vTable/types';

export type ExcelValue = string | number | boolean | Date | null | undefined;

export interface ExcelColumnConfig<T> {
  header?: string;
  width?: number;
  type?: 'string' | 'number' | 'date' | 'boolean';
  format?: string;
  exportValue?: (record: T, rowIndex: number) => ExcelValue;
  parseValue?: (
    raw: string,
    saveError: (message: string) => void,
    rowDraft: Partial<T>,
  ) => unknown;
  setValue?: (rowDraft: Partial<T>, parsedValue: unknown, raw: string) => void;
  validationList?: () => string[];
  ignoreOnExport?: boolean;
  /** Target field used to store the raw string coming from the imported Excel cell. */
  importField?: string;
}

export type ExcelTableColumn<T> = ITableColumn<T> & {
  excel?: ExcelColumnConfig<T>;
  subcols?: ExcelTableColumn<T>[];
};

export interface ResolvedLeafColumn<T> {
  key: string;
  header: string;
  width?: number;
  type?: ExcelColumnConfig<T>['type'];
  format?: string;
  field?: string;
  exportValue?: ExcelColumnConfig<T>['exportValue'];
  getValue?: (record: T, rowIndex: number) => string | number;
  parseValue?: ExcelColumnConfig<T>['parseValue'];
  setValue?: ExcelColumnConfig<T>['setValue'];
  validationList?: ExcelColumnConfig<T>['validationList'];
  importField?: ExcelColumnConfig<T>['importField'];
}

export interface ResolvedTreeColumn<T> {
  header: string;
  key: string;
  column: ExcelTableColumn<T>;
  children: ResolvedTreeColumn<T>[];
  leaf?: ResolvedLeafColumn<T>;
}

export interface HeaderCellLayout {
  title: string;
  rowStart: number;
  rowEnd: number;
  colStart: number;
  colEnd: number;
}

export interface ExcelExportSheet<T> {
  sheetName: string;
  title?: string;
  columns: ExcelTableColumn<T>[];
  records: T[];
}

export interface ExcelBuildOptions<T> {
  creator?: string;
  includeTitleRow?: boolean;
  includeGroupedHeaders?: boolean;
  headerRowIndex?: number;
  bodyRowIndex?: number;
  sheet: ExcelExportSheet<T>;
}

export interface ExcelDownloadOptions<T> extends ExcelBuildOptions<T> {
  fileName: string;
}

export interface ExcelImportOptions<T> {
  columns: ExcelTableColumn<T>[];
  source: File | Uint8Array | ArrayBuffer;
  sheetName?: string;
  /** Header row indexes (1-based). Max 2 rows: [header] or [groupHeader, subHeader]. */
  headerRows?: number[];
}

export interface ExcelImportResult<T> {
	rows: Partial<T>[];
  rowsWithoutErrors: Partial<T>[];
  /** Excel row number (1-based) for each parsed row, aligned by index with `rows`. */
  rowNumbers?: number[];
  errors: string[];
  /** Leaf columns that were matched against incoming Excel headers. */
  mappedLeafColumns: ResolvedLeafColumn<T>[];
  mappedColumns: string[];
  ignoredHeaders: string[];
  sheetName: string;
}
