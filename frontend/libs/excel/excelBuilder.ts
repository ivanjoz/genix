import { buildExcelBuffer, downloadExcel } from './export';
import { parseExcelFile } from './import';
import type { ExcelBuildOptions, ExcelImportOptions, ExcelImportResult, ExcelTableColumn } from './types';

export * from './types';
export * from './export';
export * from './import';

export class ExcelBuilder<T> {
  private columns: ExcelTableColumn<T>[] = [];
  private records: T[] = [];
  private source: File | Uint8Array | ArrayBuffer | null = null;
  private exportSheetName = 'Sheet1';
  private importSheetName: string | undefined = undefined;
  private title: string | undefined = undefined;
  private creator = '';
  private includeTitleRow = true;
  private includeGroupedHeaders = true;
  private headerRowIndex = 2;
  private importHeaderRows: number[] = [2];
  private bodyRowIndex: number | undefined = undefined;

  setColumns(columns: ExcelTableColumn<T>[]): this {
    this.columns = columns;
    return this;
  }

  setRecords(records: T[]): this {
    this.records = records;
    return this;
  }

  setSource(source: File | Uint8Array | ArrayBuffer): this {
    this.source = source;
    return this;
  }

  setSheetName(sheetName: string): this {
    this.exportSheetName = sheetName;
    this.importSheetName = sheetName;
    return this;
  }

  setExportSheet(sheetName: string, title?: string): this {
    this.exportSheetName = sheetName;
    this.title = title;
    return this;
  }

  setImportSheet(sheetName?: string): this {
    this.importSheetName = sheetName;
    return this;
  }

  setHeaderRowIndex(headerRowIndex: number): this {
    this.headerRowIndex = headerRowIndex;
    this.importHeaderRows = [headerRowIndex];
    return this;
  }

  setHeaderRows(headerRows: number[]): this {
    this.importHeaderRows = [...headerRows];
    return this;
  }

  setBodyRowIndex(bodyRowIndex?: number): this {
    this.bodyRowIndex = bodyRowIndex;
    return this;
  }

  setCreator(creator: string): this {
    this.creator = creator;
    return this;
  }

  setIncludeTitleRow(includeTitleRow: boolean): this {
    this.includeTitleRow = includeTitleRow;
    return this;
  }

  setIncludeGroupedHeaders(includeGroupedHeaders: boolean): this {
    this.includeGroupedHeaders = includeGroupedHeaders;
    return this;
  }

  private resolveExportOptions(overrides?: Partial<ExcelBuildOptions<T>>): ExcelBuildOptions<T> {
    return {
      creator: overrides?.creator ?? this.creator,
      includeTitleRow: overrides?.includeTitleRow ?? this.includeTitleRow,
      includeGroupedHeaders: overrides?.includeGroupedHeaders ?? this.includeGroupedHeaders,
      headerRowIndex: overrides?.headerRowIndex ?? this.headerRowIndex,
      bodyRowIndex: overrides?.bodyRowIndex ?? this.bodyRowIndex,
      sheet: {
        sheetName: overrides?.sheet?.sheetName || this.exportSheetName,
        title: overrides?.sheet?.title ?? this.title,
        columns: overrides?.sheet?.columns || this.columns,
        records: overrides?.sheet?.records || this.records,
      },
    };
  }

  private resolveImportOptions(overrides?: Partial<ExcelImportOptions<T>>): ExcelImportOptions<T> {
    const source = overrides?.source ?? this.source;
    if (!source) {
      throw new Error('[excel-builder] Missing import source. Use setSource() or pass source in import().');
    }
    return {
      columns: overrides?.columns || this.columns,
      source,
      sheetName: overrides?.sheetName ?? this.importSheetName,
      headerRows: overrides?.headerRows ?? this.importHeaderRows,
    };
  }

  async export(overrides?: Partial<ExcelBuildOptions<T>>): Promise<Uint8Array> {
    return buildExcelBuffer(this.resolveExportOptions(overrides));
  }

  async download(fileName: string, overrides?: Partial<ExcelBuildOptions<T>>): Promise<void> {
    await downloadExcel({
      ...this.resolveExportOptions(overrides),
      fileName,
    });
  }

  async import(overrides?: Partial<ExcelImportOptions<T>>): Promise<ExcelImportResult<T>> {
    return parseExcelFile(this.resolveImportOptions(overrides));
  }

  extractRecords(importResult: ExcelImportResult<T>): Partial<T>[] {
    return importResult.rows;
  }
}
