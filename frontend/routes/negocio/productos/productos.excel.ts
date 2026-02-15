import {
  downloadExcel,
  ExcelBuilder,
  type ExcelImportResult,
  type ExcelTableColumn,
} from '$libs/excel/excelBuilder';
import {
  getOptionByName,
  PRODUCT_OPTION_LIST_MONEDA_ID,
  PRODUCT_OPTION_LIST_UNIDAD_ID,
  PRODUCT_SHARED_LIST_CATEGORIA_ID,
  PRODUCT_SHARED_LIST_MARCA_ID,
} from '$core/products-lists';
import type { ListasCompartidasService } from '$services/negocio/listas-compartidas.svelte';
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

// Keeps Productos import parsing centralized and aligned with export headers.
export const importProductosFromExcel = async (
  columns: ExcelTableColumn<IProducto>[], source: File,
): Promise<ExcelImportResult<IProducto>> => {
  const builder = new ExcelBuilder<IProducto>()
    .setColumns(columns)
    .setSource(source)
    .setHeaderRows([2]);

  return builder.import();
};

interface ProductoImportResolvedRows {
  processedRows: IProducto[];
  validationErrors: string[];
}

interface ProductoImportProcessResult {
  rows: IProducto[];
  errors: string[];
  mappedColumns: string[];
  ignoredHeaders: string[];
}

// Resolves labels imported from Excel into IDs expected by the backend.
export const resolveImportRows = (
  rawRows: Partial<IProducto>[],
  listasService: ListasCompartidasService,
): ProductoImportResolvedRows => {
  const processedRows: IProducto[] = [];
  const validationErrors: string[] = [];

  for (const [index, rawRow] of rawRows.entries()) {
    const rowNumber = index + 1;
    const processedRow = { ...rawRow } as IProducto;

    if ((processedRow._categoriasNames || '').trim().length > 0) {
      const categoriasIDs: number[] = [];
      for (const categoriaName of (processedRow._categoriasNames || '').split(',')) {
        const categoria = listasService.getByName(PRODUCT_SHARED_LIST_CATEGORIA_ID, categoriaName);
        if (!categoria?.ID) {
          validationErrors.push(`Fila ${rowNumber}: categoría no encontrada "${categoriaName.trim()}"`);
          continue;
        }
        categoriasIDs.push(categoria.ID);
      }
      processedRow.CategoriasIDs = categoriasIDs;
    }

    if ((processedRow._marcaNombre || '').trim().length > 0) {
      const marca = listasService.getByName(PRODUCT_SHARED_LIST_MARCA_ID, processedRow._marcaNombre || '');
      if (!marca?.ID) {
        validationErrors.push(`Fila ${rowNumber}: marca no encontrada "${processedRow._marcaNombre}"`);
      } else {
        processedRow.MarcaID = marca.ID;
      }
    }

    if ((processedRow._unidadNombre || '').trim().length > 0) {
      const unidad = getOptionByName(PRODUCT_OPTION_LIST_UNIDAD_ID, processedRow._unidadNombre || '');
      if (!unidad?.i) {
        validationErrors.push(`Fila ${rowNumber}: unidad no válida "${processedRow._unidadNombre}"`);
      } else {
        processedRow.UnidadID = unidad.i;
      }
    }

    if ((processedRow._monedaNombre || '').trim().length > 0) {
      const moneda = getOptionByName(PRODUCT_OPTION_LIST_MONEDA_ID, processedRow._monedaNombre || '');
      if (!moneda?.i) {
        validationErrors.push(`Fila ${rowNumber}: moneda no válida "${processedRow._monedaNombre}"`);
      } else {
        processedRow.MonedaID = moneda.i;
      }
    }

    processedRows.push(processedRow);
  }

  return { processedRows, validationErrors };
};

// Executes the complete import flow (parse + resolve + error shaping) for the Productos page.
export const processProductosImportFile = async (
  columns: ExcelTableColumn<IProducto>[],
  source: File,
  listasService: ListasCompartidasService,
): Promise<ProductoImportProcessResult> => {
  const importResult = await importProductosFromExcel(columns, source);
  const { processedRows, validationErrors } = resolveImportRows(importResult.rows, listasService);

  return {
    rows: processedRows,
    errors: [
      ...importResult.errors.map((error) => `Fila ${error.row} (${error.column}): ${error.message}`),
      ...validationErrors,
    ],
    mappedColumns: importResult.mappedColumns,
    ignoredHeaders: importResult.ignoredHeaders,
  };
};
