import {
  downloadExcel,
  ExcelBuilder,
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
import { normalizeStringN } from '$libs/helpers';
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

// Step 1: load file and return a ready ExcelBuilder instance.
export const importProductosFromExcel = async (
  columns: ExcelTableColumn<IProducto>[], source: File,
): Promise<ExcelBuilder<IProducto>> => {
  const builder = new ExcelBuilder<IProducto>()
    .setColumns(columns)
    .setHeaderRows([2]);

  await builder.loadFile(source);
  return builder;
};

interface ProductoImportProcessResult {
  rows: IProducto[];
  errors: string[];
  mappedColumns: string[];
  ignoredHeaders: string[];
}

interface ExistingProductosLookup {
  byID: Map<number, IProducto>;
  normalizedNameToID: Map<string, number>;
}

// Executes the complete import flow (parse + resolve + error shaping) for the Productos page.
export const processProductosImportFile = async (
  columns: ExcelTableColumn<IProducto>[],
  source: File,
  listasService: ListasCompartidasService,
  existingProductos: ExistingProductosLookup,
): Promise<ProductoImportProcessResult> => {
	const builder = await importProductosFromExcel(columns, source);
  
  const importResult = builder.extractRecords((row) => {
    const currentRow = row as IProducto;
    const validationErrors: string[] = [];

    if ((currentRow._categoriasNames || '').length > 0) {
      const categoriasIDs: number[] = [];
      for (const categoriaNameRaw of (currentRow._categoriasNames || '').split(',')) {
        const categoriaName = categoriaNameRaw;
        if (!categoriaName) continue;
        const categoria = listasService.getByName(PRODUCT_SHARED_LIST_CATEGORIA_ID, categoriaName);
        if (!categoria?.ID) {
          validationErrors.push(`categoría no encontrada "${categoriaName}"`);
          continue;
        }
        categoriasIDs.push(categoria.ID);
      }
      currentRow.CategoriasIDs = categoriasIDs;
    }

    if ((currentRow._marcaNombre || '').length > 0) {
      const marca = listasService.getByName(PRODUCT_SHARED_LIST_MARCA_ID, currentRow._marcaNombre || '');
      if (!marca?.ID) {
        validationErrors.push(`marca no encontrada "${currentRow._marcaNombre}"`);
      } else {
        currentRow.MarcaID = marca.ID;
      }
    }

    if ((currentRow._unidadNombre || '').length > 0) {
      const unidad = getOptionByName(PRODUCT_OPTION_LIST_UNIDAD_ID, currentRow._unidadNombre || '');
      if (!unidad?.i) {
        validationErrors.push(`unidad no válida "${currentRow._unidadNombre}"`);
      } else {
        currentRow.UnidadID = unidad.i;
      }
    }

    if ((currentRow._monedaNombre || '').length > 0) {
      const moneda = getOptionByName(PRODUCT_OPTION_LIST_MONEDA_ID, currentRow._monedaNombre || '');
      if (!moneda?.i) {
        validationErrors.push(`moneda no válida "${currentRow._monedaNombre}"`);
      } else {
        currentRow.MonedaID = moneda.i;
      }
    }

    const matchedByID = isValidPositiveID(currentRow.ID) ? existingProductos.byID.get(currentRow.ID) : undefined;
    if (matchedByID) {
      currentRow.ID = matchedByID.ID;
    } else if (isValidPositiveID(currentRow.ID)) {
      validationErrors.push(`ID de producto no encontrado "${currentRow.ID}"`);
    } else {
      const normalizedName = normalizeStringN(currentRow.Nombre || '');
      const matchedByNameID = existingProductos.normalizedNameToID.get(normalizedName);
      currentRow.ID = matchedByNameID || 0;
    }

    return validationErrors.length > 0 ? validationErrors : undefined;
  });

  return {
    rows: importResult.rowsWithoutErrors as IProducto[],
    errors: importResult.errors,
    mappedColumns: importResult.mappedColumns,
    ignoredHeaders: importResult.ignoredHeaders,
  };
};

const isValidPositiveID = (value: unknown): value is number => {
  return typeof value === 'number' && Number.isInteger(value) && value > 0;
};
