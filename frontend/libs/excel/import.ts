import type { ExcelImportOptions, ExcelImportResult } from './types';
import {
  assertExcelCall,
  buildResolvedTree,
  flattenLeafColumns,
  getExcelize,
  normalizeHeader,
  parseDefaultByType,
  readSourceBuffer,
} from './helpers';

function normalizeCompositeKey(parts: string[]): string {
  const normalizedParts = parts
    .map((part) => normalizeHeader(part))
    .filter((part) => part.length > 0);
  return normalizedParts.join('_');
}

function collectLeafKeyMap<T>(columns: ExcelImportOptions<T>['columns']) {
  const tree = buildResolvedTree(columns);
  const leaves = flattenLeafColumns(tree);
  const leafMap = new Map<string, (typeof leaves)[number]>();

  const walk = (nodes: typeof tree, parentHeader = '') => {
    for (const node of nodes) {
      if (node.children.length > 0) {
        walk(node.children, node.header);
        continue;
      }
      if (!node.leaf) continue;

      const leaf = node.leaf;
      const directKey = normalizeHeader(leaf.header);
      const composedKey = normalizeCompositeKey([parentHeader, leaf.header]);
      const internalKey = normalizeHeader(leaf.key).replace(/\s+/g, '_');

      for (const candidate of [composedKey, directKey, internalKey]) {
        if (!candidate || leafMap.has(candidate)) continue;
        leafMap.set(candidate, leaf);
      }
    }
  };

  walk(tree);
  return { leaves, leafMap };
}

function sanitizeHeaderRows(headerRows: number[] | undefined): number[] {
  const rows = (headerRows && headerRows.length > 0 ? headerRows : [2])
    .map((row) => Math.floor(row))
    .filter((row) => row > 0);

  if (rows.length === 0) return [2];
  if (rows.length > 2) {
    throw new Error('[excel-builder] headerRows supports a maximum of 2 rows');
  }
  return [...new Set(rows)].sort((a, b) => a - b);
}

function formatImportError(row: number, message: string): string {
  const normalizedMessage = String(message || '').trim();
  const accentNormalizedMessage = normalizedMessage.replace(/\bno valido\b/gi, 'no válido');
  const finalMessage = accentNormalizedMessage
    ? accentNormalizedMessage.charAt(0).toUpperCase() + accentNormalizedMessage.slice(1)
    : 'Error de validacion';
  return `Fila ${row}: ${finalMessage}`;
}

export async function parseExcelFile<T>(options: ExcelImportOptions<T>): Promise<ExcelImportResult<T>> {
  const { columns, source, sheetName } = options;
  const headerRows = sanitizeHeaderRows(options.headerRows);

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
  const lastHeaderRow = headerRows[headerRows.length - 1];
  if (rows.length < lastHeaderRow) {
    return {
      rows: [],
      rowsWithoutErrors: [],
      rowNumbers: [],
      errors: [],
      mappedLeafColumns: [],
      mappedColumns: [],
      ignoredHeaders: [],
      sheetName: selectedSheet,
    };
  }

  const { leaves, leafMap } = collectLeafKeyMap(columns);

  const importFieldColumns = leaves.filter((leaf) => leaf.importField);
  if (importFieldColumns.length > 0) {
    console.log(
      `[excel-builder] Import will capture raw values for columns: ${importFieldColumns
        .map((leaf) => `${leaf.header}->${leaf.importField}`)
        .join(', ')}`,
    );
  }

  const headerRowsValues = headerRows.map((headerRow) => rows[headerRow - 1] || []);
  const maxHeaderCols = Math.max(...headerRowsValues.map((headerRow) => headerRow.length), 0);
  const expandedHeaderRows = headerRowsValues.map((headerRowValues) => {
    const expandedValues: string[] = [];
    let lastHeaderValue = '';
    for (let index = 0; index < maxHeaderCols; index++) {
      const headerValue = String(headerRowValues[index] || '').trim();
      if (headerValue) {
        lastHeaderValue = headerValue;
      }
      expandedValues[index] = lastHeaderValue;
    }
    return expandedValues;
  });

  const mappedIndexes = new Map<number, (typeof leaves)[number]>();
  const ignoredHeaders: string[] = [];

  for (let index = 0; index < maxHeaderCols; index++) {
    const headerParts = expandedHeaderRows
      .map((headerRowValues) => String(headerRowValues[index] || '').trim())
      .filter((headerValue) => headerValue.length > 0);

    const candidateKeys = [
      normalizeCompositeKey(headerParts),
      ...headerParts.map((headerPart) => normalizeHeader(headerPart)),
    ].filter((candidateKey, candidateIndex, arr) => {
      return candidateKey.length > 0 && arr.indexOf(candidateKey) === candidateIndex;
    });

    const match = candidateKeys.map((key) => leafMap.get(key)).find((leaf) => !!leaf);
    if (match) {
      mappedIndexes.set(index, match);
    } else {
      const ignoredHeaderText = headerParts.join(' / ');
      if (ignoredHeaderText.length > 0) {
        ignoredHeaders.push(ignoredHeaderText);
      }
    }
  }

  const parsedRows: Partial<T>[] = [];
  const parsedRowsWithoutErrors: Partial<T>[] = [];
  const parsedRowNumbers: number[] = [];
  const errors: string[] = [];
  const rowNumbersWithErrors = new Set<number>();

  for (let rowIndex = lastHeaderRow; rowIndex < rows.length; rowIndex++) {
    const row = rows[rowIndex] || [];
    const draft: Partial<T> = {};
    let hasAnyValue = false;

    for (const [columnIndex, leaf] of mappedIndexes.entries()) {
      const rawCellValue = row[columnIndex] ?? '';
      const rawValue = String(rawCellValue).trim();
      if (rawValue !== '') hasAnyValue = true;

      const saveError = (message: string) => {
        rowNumbersWithErrors.add(rowIndex + 1);
        errors.push(formatImportError(rowIndex + 1, message));
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
      if (leaf.importField) {
        (draft as Record<string, unknown>)[leaf.importField] = rawValue;
      }
    }

    if (hasAnyValue) {
      parsedRows.push(draft);
      parsedRowNumbers.push(rowIndex + 1);
      if (!rowNumbersWithErrors.has(rowIndex + 1)) {
        parsedRowsWithoutErrors.push(draft);
      }
    }
  }

  const mappedLeafColumns = [...mappedIndexes.values()];

  return {
    rows: parsedRows,
    rowsWithoutErrors: parsedRowsWithoutErrors,
    rowNumbers: parsedRowNumbers,
    errors,
    mappedLeafColumns,
    mappedColumns: mappedLeafColumns.map((leaf) => leaf.header),
    ignoredHeaders,
    sheetName: selectedSheet,
  };
}
