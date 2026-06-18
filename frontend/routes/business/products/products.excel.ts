import {
  downloadExcel,
  ExcelBuilder,
  type ResolvedLeafColumn,
  type ExcelTableColumn,
} from '$libs/excel/excelBuilder';
import {
  getOptionByName,
  PRODUCT_OPTION_LIST_MONEDA_ID,
  PRODUCT_OPTION_LIST_UNIDAD_ID,
  PRODUCT_SHARED_LIST_CATEGORIA_ID,
  PRODUCT_SHARED_LIST_MARCA_ID,
} from '$core/products-lists';
import type { ISharedListRecord, SharedListsService } from '$services/business/shared-lists.svelte';
import { normalizeComparableValue, normalizeStringN } from '$libs/helpers';
import type { IProduct, ProductsService } from './products.svelte';

// Centralizes Productos Excel export so the page only triggers the action.
export const exportProductosToExcel = async (
  columns: ExcelTableColumn<IProduct>[],
  records: IProduct[],
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
  columns: ExcelTableColumn<IProduct>[], source: File,
): Promise<ExcelBuilder<IProduct>> => {
  const builder = new ExcelBuilder<IProduct>()
    .setColumns(columns)
    .setHeaderRows([2]);

  await builder.loadFile(source);
  return builder;
};

interface ProductoImportProcessResult {
  rows: IProduct[];
  errors: string[];
  mappedColumns: string[];
  ignoredHeaders: string[];
}

const IMPORT_FIELD_TO_PRODUCT_FIELD: Record<string, keyof IProduct> = {
  _categoriasNames: 'CategoryIDs',
  _marcaNombre: 'BrandID',
  _unidadNombre: 'UnidadID',
  _monedaNombre: 'MonedaID',
};

const collectComparableFieldKeys = (
  mappedLeafColumns: ResolvedLeafColumn<IProduct>[],
): string[] => {
  const collectedKeys: string[] = [];
  const usedKeys = new Set<string>();

  for (const mappedLeafColumn of mappedLeafColumns) {
    const comparableFieldKey = mappedLeafColumn.field
      || (mappedLeafColumn.importField
        ? IMPORT_FIELD_TO_PRODUCT_FIELD[mappedLeafColumn.importField]
        : undefined);
    if (!comparableFieldKey || usedKeys.has(comparableFieldKey)) continue;
    usedKeys.add(comparableFieldKey);
    collectedKeys.push(comparableFieldKey);
  }

  console.log('[productos-import] collected comparable field keys:', collectedKeys);
  return collectedKeys;
};

// Executes the complete import flow (parse + resolve + error shaping) for the Productos page.
export const processProductosImportFile = async (
  columns: ExcelTableColumn<IProduct>[],
  source: File,
  listasService: SharedListsService,
  productosService: ProductsService,
): Promise<ProductoImportProcessResult> => {
	const builder = await importProductosFromExcel(columns, source);
  
  const importResult = builder.extractRecords((row) => {
    const currentRow = row as IProduct;
    const validationErrors: string[] = [];

    if ((currentRow._categoriasNames || '').length > 0) {
      const categoriasIDs: number[] = [];
      for (const categoriaNameRaw of (currentRow._categoriasNames || '').split(',')) {
        const categoriaName = categoriaNameRaw.trim();
        if (!categoriaName) continue;
        const categoriaExistente = listasService.getByName({ ListID: PRODUCT_SHARED_LIST_CATEGORIA_ID, Name: categoriaName });
        if (categoriaExistente?.ID) {
          categoriasIDs.push(categoriaExistente.ID);
          continue;
        }

        const categoriaTemporal: ISharedListRecord = {
          ID: 0,
          ListID: PRODUCT_SHARED_LIST_CATEGORIA_ID,
          Name: categoriaName,
          ss: 1,
          upd: 0,
        };
        listasService.addTempRecord(categoriaTemporal);
        categoriasIDs.push(categoriaTemporal.ID);
      }
      currentRow.CategoryIDs = categoriasIDs;
    }

    if ((currentRow._marcaNombre || '').length > 0) {
      const marcaNombre = (currentRow._marcaNombre || '').trim();
      if (marcaNombre.length > 0) {
        const marcaExistente = listasService.getByName({ ListID: PRODUCT_SHARED_LIST_MARCA_ID, Name: marcaNombre });
        if (marcaExistente?.ID) {
          currentRow.BrandID = marcaExistente.ID;
        } else {
          const marcaTemporal: ISharedListRecord = {
            ID: 0,
            ListID: PRODUCT_SHARED_LIST_MARCA_ID,
            Name: marcaNombre,
            ss: 1,
            upd: 0,
          };
          listasService.addTempRecord(marcaTemporal);
          currentRow.BrandID = marcaTemporal.ID;
        }
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

    const matchedByID = isValidPositiveID(currentRow.ID) ? productosService.recordsMap.get(currentRow.ID) : undefined;
    if (matchedByID) {
      currentRow.ID = matchedByID.ID;
    } else if (isValidPositiveID(currentRow.ID)) {
      validationErrors.push(`ID de producto no encontrado "${currentRow.ID}"`);
    } else {
      currentRow.ID = productosService.getByName({ Name: currentRow.Name })?.ID || 0;
    }

    return validationErrors.length > 0 ? validationErrors : undefined;
	});

  console.log("importResult",importResult)

  const comparableFieldKeys = collectComparableFieldKeys(importResult.mappedLeafColumns);
  console.log('[productos-import] comparable fields for diff:', comparableFieldKeys);

	const rowsWithUpdatedFields: IProduct[] = [];
  
  for (const importedProducto of (importResult.rowsWithoutErrors as IProduct[])) {
    const existingProducto = productosService.recordsMap.get(importedProducto.ID)
    if (!existingProducto) {
      importedProducto._updatedFields = ['ID'];
      rowsWithUpdatedFields.push(importedProducto);
      continue;
    }

    const updatedFieldKeys: string[] = [];
    for (const fieldKey of comparableFieldKeys) {
      const importedValue = normalizeComparableValue(
        (importedProducto as unknown as Record<string, unknown>)[fieldKey],
      );
      const existingValue = normalizeComparableValue(
        (existingProducto as unknown as Record<string, unknown>)[fieldKey],
      );
      if (!Object.is(importedValue, existingValue)) {
        updatedFieldKeys.push(fieldKey);
      }
		}
    
    // if (updatedFieldKeys.length === 0) continue;
    importedProducto._updatedFields = updatedFieldKeys;
    rowsWithUpdatedFields.push(importedProducto);
  }
  
  console.log("rowsWithUpdatedFields", rowsWithUpdatedFields)

  return {
    rows: rowsWithUpdatedFields,
    errors: importResult.errors,
    mappedColumns: importResult.mappedColumns,
    ignoredHeaders: importResult.ignoredHeaders,
  };
};

const isValidPositiveID = (value: unknown): value is number => {
  return typeof value === 'number' && Number.isInteger(value) && value > 0;
};
