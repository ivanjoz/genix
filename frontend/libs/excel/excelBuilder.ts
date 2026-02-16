import { buildExcelBuffer, downloadExcel } from './export';
import { parseExcelFile } from './import';
import type { ExcelBuildOptions, ExcelImportOptions, ExcelImportResult, ExcelTableColumn } from './types';

export * from './types';
export * from './export';
export * from './import';

export type ExcelExtractRecordsHandler<T> = (record: T) => string[] | undefined;

const formatExtractRecordError = (row: number, message: string): string => {
  const normalizedMessage = String(message || '').trim();
  const accentNormalizedMessage = normalizedMessage.replace(/\bno valido\b/gi, 'no válido');
  const finalMessage = accentNormalizedMessage
    ? accentNormalizedMessage.charAt(0).toUpperCase() + accentNormalizedMessage.slice(1)
    : 'Error de validacion';
  return `Fila ${row}: ${finalMessage}`;
};

export class ExcelBuilder<T> {
  private columns: ExcelTableColumn<T>[] = [];
  private records: T[] = [];
  private loadedImportResult: ExcelImportResult<T> | null = null;
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

  private resolveImportOptions(
    source: File | Uint8Array | ArrayBuffer,
    overrides?: Omit<Partial<ExcelImportOptions<T>>, 'source'>,
  ): ExcelImportOptions<T> {
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

  // Step 1: Parse and cache workbook rows so extractRecords can run handlers separately.
  async loadFile(
    source: File | Uint8Array | ArrayBuffer,
    overrides?: Omit<Partial<ExcelImportOptions<T>>, 'source'>,
  ): Promise<this> {
    this.loadedImportResult = await parseExcelFile(this.resolveImportOptions(source, overrides));
    return this;
  }

  // Step 2: Return parsed rows and optionally append row-aware validation errors from handler.
  extractRecords(handler?: ExcelExtractRecordsHandler<T>): ExcelImportResult<T> {
    if (!this.loadedImportResult) {
      throw new Error('[excel-builder] No loaded file found. Call loadFile() before extractRecords().');
    }

    const baseResult = this.loadedImportResult;
    if (!handler) {
      return {
        ...baseResult,
        rows: [...baseResult.rows],
        rowsWithoutErrors: [...baseResult.rowsWithoutErrors],
        errors: [...baseResult.errors],
        mappedLeafColumns: [...baseResult.mappedLeafColumns],
        mappedColumns: [...baseResult.mappedColumns],
        ignoredHeaders: [...baseResult.ignoredHeaders],
        rowNumbers: baseResult.rowNumbers ? [...baseResult.rowNumbers] : undefined,
      };
    }

    const extractedRows: Partial<T>[] = [];
    const extractedRowsWithoutErrors: Partial<T>[] = [];
    const handlerErrors = [...baseResult.errors];
    const baseRowsWithoutErrorsSet = new Set(baseResult.rowsWithoutErrors);

    for (let index = 0; index < baseResult.rows.length; index++) {
      const rowDraft = { ...baseResult.rows[index] } as Partial<T>;
      const hasParserError = !baseRowsWithoutErrorsSet.has(baseResult.rows[index]);
      let validationErrors: string[] | undefined;

      try {
        validationErrors = handler(rowDraft as T);
      } catch (error) {
        validationErrors = [String(error)];
      }

      if (validationErrors?.length) {
        const excelRow = baseResult.rowNumbers?.[index] ?? index + 1;
        for (const message of validationErrors) {
          handlerErrors.push(formatExtractRecordError(excelRow, message));
        }
      } else if (!hasParserError) {
        extractedRowsWithoutErrors.push(rowDraft);
      }

      extractedRows.push(rowDraft);
    }

    return {
      ...baseResult,
      rows: extractedRows,
      rowsWithoutErrors: extractedRowsWithoutErrors,
      errors: handlerErrors,
      mappedLeafColumns: [...baseResult.mappedLeafColumns],
      mappedColumns: [...baseResult.mappedColumns],
      ignoredHeaders: [...baseResult.ignoredHeaders],
      rowNumbers: baseResult.rowNumbers ? [...baseResult.rowNumbers] : undefined,
    };
  }

}
