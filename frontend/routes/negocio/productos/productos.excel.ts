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
import type { ListasCompartidasService } from '$services/negocio/listas-compartidas.svelte';
import { normalizeComparableValue, normalizeStringN } from '$libs/helpers';
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

const IMPORT_FIELD_TO_PRODUCT_FIELD: Record<string, keyof IProducto> = {
  _categoriasNames: 'CategoriasIDs',
  _marcaNombre: 'MarcaID',
  _unidadNombre: 'UnidadID',
  _monedaNombre: 'MonedaID',
};

const collectComparableFieldKeys = (
  mappedLeafColumns: ResolvedLeafColumn<IProducto>[],
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
        const categoriaName = categoriaNameRaw.trim();
        if (!categoriaName) continue;
        const categoriaExistente = listasService.getByName(PRODUCT_SHARED_LIST_CATEGORIA_ID, categoriaName);
        if (categoriaExistente?.ID) {
          categoriasIDs.push(categoriaExistente.ID);
          continue;
        }

        const categoriaTemporal = listasService.addNewTemp({
          ListaID: PRODUCT_SHARED_LIST_CATEGORIA_ID,
          Nombre: categoriaName,
          ss: 1,
          upd: 0,
        });
        categoriasIDs.push(categoriaTemporal.ID);
      }
      currentRow.CategoriasIDs = categoriasIDs;
    }

    if ((currentRow._marcaNombre || '').length > 0) {
      const marcaNombre = (currentRow._marcaNombre || '').trim();
      if (marcaNombre.length > 0) {
        const marcaExistente = listasService.getByName(PRODUCT_SHARED_LIST_MARCA_ID, marcaNombre);
        if (marcaExistente?.ID) {
          currentRow.MarcaID = marcaExistente.ID;
        } else {
          const marcaTemporal = listasService.addNewTemp({
            ListaID: PRODUCT_SHARED_LIST_MARCA_ID,
            Nombre: marcaNombre,
            ss: 1,
            upd: 0,
          });
          currentRow.MarcaID = marcaTemporal.ID;
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

  const comparableFieldKeys = collectComparableFieldKeys(importResult.mappedLeafColumns);
  console.log('[productos-import] comparable fields for diff:', comparableFieldKeys);

	const rowsWithUpdatedFields: IProducto[] = [];
  
  for (const importedProducto of (importResult.rowsWithoutErrors as IProducto[])) {
    const existingProducto = isValidPositiveID(importedProducto.ID)
      ? existingProductos.byID.get(importedProducto.ID)
      : undefined;
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
    
    if (updatedFieldKeys.length === 0) continue;
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
