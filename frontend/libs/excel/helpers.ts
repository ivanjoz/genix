import { base } from '$app/paths';
import { init, type Style } from 'excelize-wasm';
import type {
  ExcelColumnConfig,
  ExcelTableColumn,
  HeaderCellLayout,
  ResolvedLeafColumn,
  ResolvedTreeColumn,
} from './types';

const XLSX_BLOB_TYPE = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
const EXCELIZE_WASM_VENDOR_URL = `${base}/vendor/excelize.wasm.bin`;

export type ExcelizeModule = Awaited<ReturnType<typeof init>>;

let excelizeModulePromise: Promise<ExcelizeModule> | null = null;

// Lazily loads and caches the WASM runtime so repeated imports/exports are fast.
export async function getExcelize(): Promise<ExcelizeModule> {
  if (!excelizeModulePromise) {
    console.log('[excel-builder] Initializing excelize-wasm runtime:', EXCELIZE_WASM_VENDOR_URL);
    excelizeModulePromise = init(EXCELIZE_WASM_VENDOR_URL);
  }
  return excelizeModulePromise;
}

export function normalizeHeader(value: string): string {
  return (value || '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
    .toLowerCase();
}

export function resolveHeaderText<T>(column: ExcelTableColumn<T>): string {
  if (column.excel?.header) return column.excel.header;
  return typeof column.header === 'function' ? column.header() : column.header;
}

export function isColumnCandidate<T>(column: ExcelTableColumn<T>): boolean {
  if (column.excel?.ignoreOnExport) return false;
  if ((column.subcols || []).length > 0) return true;
  if (column.excel?.exportValue) return true;
  if (column.getValue) return true;
  if (column.field) return true;
  return false;
}

export function buildResolvedTree<T>(columns: ExcelTableColumn<T>[], prefix = ''): ResolvedTreeColumn<T>[] {
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
        importField: column.excel?.importField,
        validationList: column.excel?.validationList,
      },
    });
  }

  return resolved;
}

export function treeDepth<T>(nodes: ResolvedTreeColumn<T>[]): number {
  let maxDepth = 1;
  for (const node of nodes) {
    if (node.children.length > 0) {
      maxDepth = Math.max(maxDepth, 1 + treeDepth(node.children));
    }
  }
  return maxDepth;
}

export function countLeafColumns<T>(node: ResolvedTreeColumn<T>): number {
  if (node.children.length === 0) return 1;
  return node.children.reduce((sum, child) => sum + countLeafColumns(child), 0);
}

export function flattenLeafColumns<T>(nodes: ResolvedTreeColumn<T>[]): ResolvedLeafColumn<T>[] {
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

export function buildHeaderLayout<T>(
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

export function assertExcelCall(opName: string, error: string | null | undefined): void {
  if (error) {
    throw new Error(`[excel-builder] ${opName} failed: ${error}`);
  }
}

export function toCellName(excelize: ExcelizeModule, col: number, row: number): string {
  const ret = excelize.CoordinatesToCellName(col, row);
  assertExcelCall('CoordinatesToCellName', ret.error);
  return ret.cell;
}

export function safeValue(raw: unknown): string | number | boolean {
  if (raw === undefined || raw === null) return '';
  if (raw instanceof Date) {
    const yyyy = raw.getFullYear();
    const mm = String(raw.getMonth() + 1).padStart(2, '0');
    const dd = String(raw.getDate()).padStart(2, '0');
    return `${yyyy}-${mm}-${dd}`;
  }
  if (typeof raw === 'string' || typeof raw === 'number' || typeof raw === 'boolean') return raw;
  return String(raw);
}

export function getLeafColumnValue<T>(
  column: ResolvedLeafColumn<T>,
  record: T,
  rowIndex: number,
): string | number | boolean {
  if (column.exportValue) {
    return safeValue(column.exportValue(record, rowIndex));
  }
  if (column.getValue) {
    return safeValue(column.getValue(record, rowIndex));
  }
  if (column.field) {
    return safeValue((record as Record<string, unknown>)[column.field]);
  }
  return '';
}

export function resolveAutoColumnWidth<T>(column: ResolvedLeafColumn<T>, records: T[]): number {
  if (column.width && column.width > 0) return column.width;

  let largestCellLength = column.header.length;
  for (let rowIndex = 0; rowIndex < records.length; rowIndex++) {
    const rowValue = getLeafColumnValue(column, records[rowIndex], rowIndex);
    const valueLength = String(rowValue ?? '').trim().length;
    if (valueLength > largestCellLength) {
      largestCellLength = valueLength;
    }
  }

  const widthWithPadding = largestCellLength + (column.type === 'number' ? 1 : 2);
  return Math.min(60, Math.max(10, widthWithPadding));
}

export function makeHeaderStyle(): Style {
  return {
    Font: { Bold: true, Color: 'FFFFFF' },
    Fill: { Type: 'pattern', Pattern: 1, Color: ['1E6096'] },
    Alignment: { Horizontal: 'center', Vertical: 'center', WrapText: true },
  };
}

export function makeTitleStyle(): Style {
  return {
    Font: { Bold: true, Color: 'FFFFFF', Size: 12 },
    Fill: { Type: 'pattern', Pattern: 1, Color: ['0F4C75'] },
    Alignment: { Horizontal: 'center', Vertical: 'center' },
  };
}

export function makeBodyStyle<T>(column: ResolvedLeafColumn<T>): Style | null {
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

export function downloadBuffer(fileName: string, buffer: Uint8Array | ArrayBuffer): void {
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

export async function readSourceBuffer(source: File | Uint8Array | ArrayBuffer): Promise<Uint8Array> {
  if (source instanceof Uint8Array) return source;
  if (source instanceof ArrayBuffer) return new Uint8Array(source);
  const arr = await source.arrayBuffer();
  return new Uint8Array(arr);
}

export function parseDefaultByType(type: ExcelColumnConfig<unknown>['type'], raw: string): unknown {
  if (raw === '') return undefined;
  if (!type || type === 'string') return raw;

  if (type === 'number') {
    const parsed = Number(raw);
    if (Number.isFinite(parsed)) return parsed;

    // Fallback for decorated numeric strings like "23%" or "S/ 19.90".
    const numberToken = String(raw).trim().match(/[-+]?\d+(?:[.,]\d+)?/)?.[0];
    if (!numberToken) return undefined;

    const normalizedToken = numberToken.includes(',') && !numberToken.includes('.')
      ? numberToken.replace(',', '.')
      : numberToken;
    const parsedToken = Number.parseFloat(normalizedToken);
    return Number.isFinite(parsedToken) ? parsedToken : undefined;
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
