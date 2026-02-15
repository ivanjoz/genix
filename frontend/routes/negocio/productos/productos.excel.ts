import { downloadExcel, type ExcelTableColumn } from '$libs/excel/excelBuilder';
import type { IProducto } from './productos.svelte';

// Centralizes Productos Excel export so the page only triggers the action.
export const exportProductosToExcel = async (
  columns: ExcelTableColumn<IProducto>[],
  records: IProducto[],
): Promise<void> => {
  await downloadExcel({
    fileName: 'productos.xlsx',
    creator: 'Genix',
    includeTitleRow: true,
    includeGroupedHeaders: true,
    headerRowIndex: 2,
    sheet: {
      sheetName: 'Productos',
      title: 'Productos',
      columns,
      records,
    },
  });
};
