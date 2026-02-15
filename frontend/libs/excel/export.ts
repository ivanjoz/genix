import type { DataValidation } from 'excelize-wasm';
import type { ExcelBuildOptions, ExcelDownloadOptions, ExcelTableColumn } from './types';
import {
  assertExcelCall,
  buildHeaderLayout,
  buildResolvedTree,
  downloadBuffer,
  flattenLeafColumns,
  getExcelize,
  getLeafColumnValue,
  makeBodyStyle,
  makeHeaderStyle,
  makeTitleStyle,
  resolveAutoColumnWidth,
  toCellName,
} from './helpers';

export function toExcelColumns<T>(columns: ExcelTableColumn<T>[]) {
  const tree = buildResolvedTree(columns);
  const leaves = flattenLeafColumns(tree);
  return { tree, leaves };
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

    const resolvedWidth = resolveAutoColumnWidth(leaf, sheet.records);
    assertExcelCall(
      'SetColWidth',
      file.SetColWidth(sheet.sheetName, colNameRet.col, colNameRet.col, resolvedWidth).error,
    );
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
      const leaf = leafColumns[colIndex];
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
