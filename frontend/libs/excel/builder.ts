import type { ITableColumn } from '$components/vTable/types';
import { base } from '$app/paths';
import { init, type DataValidation, type Style } from 'excelize-wasm';

const XLSX_BLOB_TYPE = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
const EXCELIZE_WASM_VENDOR_URL = `${base}/vendor/excelize.wasm.bin`;

type ExcelizeModule = Awaited<ReturnType<typeof init>>;

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
}

export type ExcelTableColumn<T> = ITableColumn<T> & {
  excel?: ExcelColumnConfig<T>;
  subcols?: ExcelTableColumn<T>[];
};

interface ResolvedLeafColumn<T> {
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
}

interface ResolvedTreeColumn<T> {
  header: string;
  key: string;
  column: ExcelTableColumn<T>;
  children: ResolvedTreeColumn<T>[];
  leaf?: ResolvedLeafColumn<T>;
}

interface HeaderCellLayout {
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
  headerRowIndex?: number;
}

export interface ExcelImportError {
  row: number;
  column: string;
  message: string;
}

export interface ExcelImportResult<T> {
  rows: Partial<T>[];
  errors: ExcelImportError[];
  mappedColumns: string[];
  ignoredHeaders: string[];
  sheetName: string;
}

let excelizeModulePromise: Promise<ExcelizeModule> | null = null;

// Lazily loads and caches the WASM runtime so repeated imports/exports are fast.
async function getExcelize(): Promise<ExcelizeModule> {
  if (!excelizeModulePromise) {
    console.log('[excel-builder] Initializing excelize-wasm runtime:', EXCELIZE_WASM_VENDOR_URL);
    excelizeModulePromise = init(EXCELIZE_WASM_VENDOR_URL);
  }
  return excelizeModulePromise;
}

function normalizeHeader(value: string): string {
  return (value || '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .toLowerCase();
}

function resolveHeaderText<T>(column: ExcelTableColumn<T>): string {
  if (column.excel?.header) return column.excel.header;
  return typeof column.header === 'function' ? column.header() : column.header;
}

function isColumnCandidate<T>(column: ExcelTableColumn<T>): boolean {
  if (column.excel?.ignoreOnExport) return false;
  if ((column.subcols || []).length > 0) return true;
  if (column.excel?.exportValue) return true;
  if (column.getValue) return true;
  if (column.field) return true;
  return false;
}

function buildResolvedTree<T>(columns: ExcelTableColumn<T>[], prefix = ''): ResolvedTreeColumn<T>[] {
  const resolved: ResolvedTreeColumn<T>[] = [];

  for (const column of columns) {
    if (!column || !isColumnCandidate(column)) continue;

    const header = resolveHeaderText(column);
    const keyBase = prefix ? `${prefix}_${normalizeHeader(header)}` : normalizeHeader(header);
    const children = buildResolvedTree(column.subcols || [], keyBase);

    if (children.length > 0) {
      resolved.push({
        header,
        key: keyBase,
        column,
        children,
      });
      continue;
    }

    resolved.push({
      header,
      key: keyBase,
      column,
      children: [],
      leaf: {
        key: keyBase,
        header,
        width: column.excel?.width,
        type: column.excel?.type,
        format: column.excel?.format,
        field: column.field,
        exportValue: column.excel?.exportValue,
        getValue: column.getValue,
        parseValue: column.excel?.parseValue,
        setValue: column.excel?.setValue,
        validationList: column.excel?.validationList,
      },
    });
  }

  return resolved;
}

function treeDepth<T>(nodes: ResolvedTreeColumn<T>[]): number {
  let maxDepth = 1;
  for (const node of nodes) {
    if (node.children.length > 0) {
      maxDepth = Math.max(maxDepth, 1 + treeDepth(node.children));
    }
  }
  return maxDepth;
}

function countLeafColumns<T>(node: ResolvedTreeColumn<T>): number {
  if (node.children.length === 0) return 1;
  return node.children.reduce((sum, child) => sum + countLeafColumns(child), 0);
}

function flattenLeafColumns<T>(nodes: ResolvedTreeColumn<T>[]): ResolvedLeafColumn<T>[] {
  const flat: ResolvedLeafColumn<T>[] = [];
  const walk = (items: ResolvedTreeColumn<T>[]) => {
    for (const item of items) {
      if (item.children.length > 0) {
        walk(item.children);
      } else if (item.leaf) {
        flat.push(item.leaf);
      }
    }
  };
  walk(nodes);
  return flat;
}

function buildHeaderLayout<T>(
  tree: ResolvedTreeColumn<T>[],
  headerRowIndex: number,
  includeGroupedHeaders: boolean,
): { cells: HeaderCellLayout[]; depth: number; leafColumns: ResolvedLeafColumn<T>[] } {
  const depth = includeGroupedHeaders ? treeDepth(tree) : 1;
  const leafColumns = flattenLeafColumns(tree);
  const cells: HeaderCellLayout[] = [];

  if (!includeGroupedHeaders) {
    leafColumns.forEach((column, index) => {
      cells.push({
        title: column.header,
        rowStart: headerRowIndex,
        rowEnd: headerRowIndex,
        colStart: index + 1,
        colEnd: index + 1,
      });
    });
    return { cells, depth, leafColumns };
  }

  // Builds merged header cells by recursively assigning grid coordinates.
  const walk = (nodes: ResolvedTreeColumn<T>[], level: number, startColumn: number): number => {
    let currentColumn = startColumn;

    for (const node of nodes) {
      if (node.children.length === 0) {
        cells.push({
          title: node.header,
          rowStart: headerRowIndex + level,
          rowEnd: headerRowIndex + depth - 1,
          colStart: currentColumn,
          colEnd: currentColumn,
        });
        currentColumn += 1;
        continue;
      }

      const leafCount = countLeafColumns(node);
      cells.push({
        title: node.header,
        rowStart: headerRowIndex + level,
        rowEnd: headerRowIndex + level,
        colStart: currentColumn,
        colEnd: currentColumn + leafCount - 1,
      });

      currentColumn = walk(node.children, level + 1, currentColumn);
    }

    return currentColumn;
  };

  walk(tree, 0, 1);
  return { cells, depth, leafColumns };
}

function assertExcelCall(opName: string, error: string | null | undefined): void {
  if (error) {
    throw new Error(`[excel-builder] ${opName} failed: ${error}`);
  }
}

function toCellName(excelize: ExcelizeModule, col: number, row: number): string {
  const ret = excelize.CoordinatesToCellName(col, row);
  assertExcelCall('CoordinatesToCellName', ret.error);
  return ret.cell;
}

function safeValue(raw: ExcelValue): string | number | boolean {
  if (raw === undefined || raw === null) return '';
  if (raw instanceof Date) {
    const yyyy = raw.getFullYear();
    const mm = String(raw.getMonth() + 1).padStart(2, '0');
    const dd = String(raw.getDate()).padStart(2, '0');
    return `${yyyy}-${mm}-${dd}`;
  }
  return raw;
}

function getLeafColumnValue<T>(column: ResolvedLeafColumn<T>, record: T, rowIndex: number): string | number | boolean {
  if (column.exportValue) {
    return safeValue(column.exportValue(record, rowIndex));
  }
  if (column.getValue) {
    return safeValue(column.getValue(record, rowIndex));
  }
  if (column.field) {
    return safeValue((record as Record<string, unknown>)[column.field] as ExcelValue);
  }
  return '';
}

function makeHeaderStyle(): Style {
  return {
    Font: { Bold: true, Color: 'FFFFFF' },
    Fill: { Type: 'pattern', Pattern: 1, Color: ['1E6096'] },
    Alignment: { Horizontal: 'center', Vertical: 'center', WrapText: true },
  };
}

function makeTitleStyle(): Style {
  return {
    Font: { Bold: true, Color: 'FFFFFF', Size: 12 },
    Fill: { Type: 'pattern', Pattern: 1, Color: ['0F4C75'] },
    Alignment: { Horizontal: 'center', Vertical: 'center' },
  };
}

function makeBodyStyle(column: ResolvedLeafColumn<unknown>): Style | null {
  const style: Style = {};
  if (column.format) {
    style.CustomNumFmt = column.format;
  }
  if (column.type === 'number') {
    style.Alignment = { Horizontal: 'right', Vertical: 'center' };
  }
  if (column.type === 'boolean') {
    style.Alignment = { Horizontal: 'center', Vertical: 'center' };
  }
  if (column.type === 'date' && !column.format) {
    style.CustomNumFmt = 'yyyy-mm-dd';
  }
  return Object.keys(style).length > 0 ? style : null;
}

function downloadBuffer(fileName: string, buffer: Uint8Array | ArrayBuffer): void {
  const binaryBytes = buffer instanceof Uint8Array ? buffer : new Uint8Array(buffer);
  const binaryBuffer = new ArrayBuffer(binaryBytes.byteLength);
  new Uint8Array(binaryBuffer).set(binaryBytes);
  const blob = new Blob([binaryBuffer], { type: XLSX_BLOB_TYPE });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName;
  anchor.click();
  setTimeout(() => URL.revokeObjectURL(url), 2500);
}

async function readSourceBuffer(source: File | Uint8Array | ArrayBuffer): Promise<Uint8Array> {
  if (source instanceof Uint8Array) return source;
  if (source instanceof ArrayBuffer) return new Uint8Array(source);
  const arr = await source.arrayBuffer();
  return new Uint8Array(arr);
}

function parseDefaultByType(type: ExcelColumnConfig<unknown>['type'], raw: string): unknown {
  if (raw === '') return undefined;
  if (!type || type === 'string') return raw;

  if (type === 'number') {
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? parsed : undefined;
  }

  if (type === 'boolean') {
    const value = normalizeHeader(raw);
    if (value === 'true' || value === '1' || value === 'si' || value === 'yes') return true;
    if (value === 'false' || value === '0' || value === 'no') return false;
    return undefined;
  }

  const parsedDate = new Date(raw);
  if (!Number.isNaN(parsedDate.getTime())) return parsedDate;
  return undefined;
}

export function toExcelColumns<T>(columns: ExcelTableColumn<T>[]): {
  tree: ResolvedTreeColumn<T>[];
  leaves: ResolvedLeafColumn<T>[];
} {
  const tree = buildResolvedTree(columns);
  return {
    leaves: flattenLeafColumns(tree),
    tree,
  };
}

export async function buildExcelBuffer<T>(options: ExcelBuildOptions<T>): Promise<Uint8Array> {
  const {
    creator = '',
    includeTitleRow = true,
    includeGroupedHeaders = true,
    headerRowIndex = includeTitleRow ? 2 : 1,
    sheet,
  } = options;

  const excelize = await getExcelize();
  const file = excelize.NewFile();

  assertExcelCall('NewFile', file.error);
  if (creator) {
    const creatorRes = file.SetAppProps({ Company: creator, Application: 'Genix' });
    assertExcelCall('SetAppProps', creatorRes.error);
  }

  const defaultSheetName = 'Sheet1';
  if (sheet.sheetName !== defaultSheetName) {
    const addSheetRes = file.NewSheet(sheet.sheetName);
    assertExcelCall('NewSheet', addSheetRes.error);
    const deleteRes = file.DeleteSheet(defaultSheetName);
    assertExcelCall('DeleteSheet', deleteRes.error);
  }

  const { tree } = toExcelColumns(sheet.columns);
  const { cells, depth: headerDepth, leafColumns } = buildHeaderLayout(tree, headerRowIndex, includeGroupedHeaders);
  if (leafColumns.length === 0) {
    throw new Error('[excel-builder] No exportable columns were found');
  }

  console.log('[excel-builder] Preparing export', {
    sheetName: sheet.sheetName,
    columns: leafColumns.length,
    records: sheet.records.length,
  });

  const headerStyleRes = file.NewStyle(makeHeaderStyle());
  assertExcelCall('NewStyle(header)', headerStyleRes.error);

  if (includeTitleRow && sheet.title) {
    const titleStyleRes = file.NewStyle(makeTitleStyle());
    assertExcelCall('NewStyle(title)', titleStyleRes.error);

    const firstTitleCell = toCellName(excelize, 1, 1);
    const lastTitleCell = toCellName(excelize, leafColumns.length, 1);

    assertExcelCall('SetCellValue(title)', file.SetCellValue(sheet.sheetName, firstTitleCell, sheet.title).error);
    assertExcelCall('MergeCell(title)', file.MergeCell(sheet.sheetName, firstTitleCell, lastTitleCell).error);
    assertExcelCall(
      'SetCellStyle(title)',
      file.SetCellStyle(sheet.sheetName, firstTitleCell, lastTitleCell, titleStyleRes.style).error,
    );
  }

  for (const headerCell of cells) {
    const topLeft = toCellName(excelize, headerCell.colStart, headerCell.rowStart);
    const bottomRight = toCellName(excelize, headerCell.colEnd, headerCell.rowEnd);

    assertExcelCall('SetCellValue(header)', file.SetCellValue(sheet.sheetName, topLeft, headerCell.title).error);
    if (topLeft !== bottomRight) {
      assertExcelCall('MergeCell(header)', file.MergeCell(sheet.sheetName, topLeft, bottomRight).error);
    }
    assertExcelCall(
      'SetCellStyle(header)',
      file.SetCellStyle(sheet.sheetName, topLeft, bottomRight, headerStyleRes.style).error,
    );
  }

  for (let colIndex = 0; colIndex < leafColumns.length; colIndex++) {
    const leaf = leafColumns[colIndex];
    const colNumber = colIndex + 1;
    const colNameRet = excelize.ColumnNumberToName(colNumber);
    assertExcelCall('ColumnNumberToName', colNameRet.error);

    if (leaf.width && leaf.width > 0) {
      assertExcelCall(
        'SetColWidth',
        file.SetColWidth(sheet.sheetName, colNameRet.col, colNameRet.col, leaf.width).error,
      );
    }
  }

  const bodyStartRow = options.bodyRowIndex || (headerRowIndex + headerDepth);
  for (let rowIndex = 0; rowIndex < sheet.records.length; rowIndex++) {
    const record = sheet.records[rowIndex];
    const rowValues = leafColumns.map((leaf) => getLeafColumnValue(leaf, record, rowIndex));
    const rowStartCell = toCellName(excelize, 1, bodyStartRow + rowIndex);

    assertExcelCall('SetSheetRow(body)', file.SetSheetRow(sheet.sheetName, rowStartCell, rowValues).error);
  }

  const bodyEndRow = bodyStartRow + sheet.records.length - 1;
  if (sheet.records.length > 0) {
    for (let colIndex = 0; colIndex < leafColumns.length; colIndex++) {
      const leaf = leafColumns[colIndex] as ResolvedLeafColumn<unknown>;
      const style = makeBodyStyle(leaf);
      if (!style) continue;

      const styleRes = file.NewStyle(style);
      assertExcelCall(`NewStyle(body:${leaf.key})`, styleRes.error);

      const top = toCellName(excelize, colIndex + 1, bodyStartRow);
      const bottom = toCellName(excelize, colIndex + 1, bodyEndRow);
      assertExcelCall(
        `SetCellStyle(body:${leaf.key})`,
        file.SetCellStyle(sheet.sheetName, top, bottom, styleRes.style).error,
      );
    }
  }

  // Adds list data validation directly on each data column when provided.
  for (let colIndex = 0; colIndex < leafColumns.length; colIndex++) {
    const leaf = leafColumns[colIndex];
    if (!leaf.validationList || sheet.records.length === 0) continue;

    const values = leaf.validationList().filter((entry) => typeof entry === 'string' && entry.length > 0);
    if (values.length === 0) continue;

    const top = toCellName(excelize, colIndex + 1, bodyStartRow);
    const bottom = toCellName(excelize, colIndex + 1, bodyEndRow);

    const validation: DataValidation = {
      Type: 'list',
      Sqref: `${top}:${bottom}`,
      Formula1: `"${values.join(',')}"`,
      ShowDropDown: true,
      ShowErrorMessage: true,
    };

    assertExcelCall(
      `AddDataValidation(${leaf.key})`,
      file.AddDataValidation(sheet.sheetName, validation).error,
    );
  }

  const output = file.WriteToBuffer();
  assertExcelCall('WriteToBuffer', output.error);

  console.log('[excel-builder] Export completed', {
    bytes: (output.buffer as Uint8Array).byteLength || 0,
    rows: sheet.records.length,
  });

  return output.buffer as Uint8Array;
}

export async function downloadExcel<T>(options: ExcelDownloadOptions<T>): Promise<void> {
  const buffer = await buildExcelBuffer(options);
  downloadBuffer(options.fileName, buffer);
}

export async function parseExcelFile<T>(options: ExcelImportOptions<T>): Promise<ExcelImportResult<T>> {
  const { columns, source, sheetName, headerRowIndex = 2 } = options;

  const excelize = await getExcelize();
  const buffer = await readSourceBuffer(source);
  const file = excelize.OpenReader(buffer);

  assertExcelCall('OpenReader', file.error);

  const sheets = file.GetSheetList().list;
  const selectedSheet = sheetName || sheets[0];
  if (!selectedSheet) {
    throw new Error('[excel-builder] Could not find a sheet to parse');
  }

  const rowsRes = file.GetRows(selectedSheet);
  assertExcelCall('GetRows', rowsRes.error);

  const rows = rowsRes.result;
  if (rows.length < headerRowIndex) {
    return {
      rows: [],
      errors: [],
      mappedColumns: [],
      ignoredHeaders: [],
      sheetName: selectedSheet,
    };
  }

  const { leaves } = toExcelColumns(columns);
  const leafMap = new Map<string, ResolvedLeafColumn<T>>();
  for (const leaf of leaves) {
    leafMap.set(normalizeHeader(leaf.header), leaf);
  }

  const headerRow = rows[headerRowIndex - 1] || [];
  const mappedIndexes = new Map<number, ResolvedLeafColumn<T>>();
  const ignoredHeaders: string[] = [];

  for (let index = 0; index < headerRow.length; index++) {
    const rawHeader = headerRow[index] || '';
    const match = leafMap.get(normalizeHeader(rawHeader));
    if (match) {
      mappedIndexes.set(index, match);
    } else if (rawHeader.trim().length > 0) {
      ignoredHeaders.push(rawHeader);
    }
  }

  const parsedRows: Partial<T>[] = [];
  const errors: ExcelImportError[] = [];

  for (let rowIndex = headerRowIndex; rowIndex < rows.length; rowIndex++) {
    const row = rows[rowIndex] || [];
    const draft: Partial<T> = {};
    let hasAnyValue = false;

    for (const [columnIndex, leaf] of mappedIndexes.entries()) {
      const rawValue = row[columnIndex] ?? '';
      if (rawValue !== '') hasAnyValue = true;

      const saveError = (message: string) => {
        errors.push({
          row: rowIndex + 1,
          column: leaf.header,
          message,
        });
      };

      let parsed: unknown;
      if (leaf.parseValue) {
        parsed = leaf.parseValue(rawValue, saveError, draft);
      } else {
        parsed = parseDefaultByType(leaf.type, rawValue);
        if (rawValue !== '' && parsed === undefined) {
          saveError(`Invalid value \"${rawValue}\" for type ${leaf.type || 'string'}`);
        }
      }

      if (leaf.setValue) {
        leaf.setValue(draft, parsed, rawValue);
      } else if (leaf.field && parsed !== undefined) {
        (draft as Record<string, unknown>)[leaf.field] = parsed;
      }
    }

    if (hasAnyValue) {
      parsedRows.push(draft);
    }
  }

  return {
    rows: parsedRows,
    errors,
    mappedColumns: [...mappedIndexes.values()].map((leaf) => leaf.header),
    ignoredHeaders,
    sheetName: selectedSheet,
  };
}
