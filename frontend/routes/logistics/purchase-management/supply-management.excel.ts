import { downloadExcel, ExcelBuilder, type ExcelTableColumn } from '@genix/ui/excel'
import { normalizeStringN } from '@genix/ui/utilities'
import type { IProductSupplyProviderRow, IProductSupplyRow } from './supply-management.svelte'

// The sheet is one row per product: the domain record plus the columns that only exist to be read
// by whoever edits the file (product name, current stock) and the raw provider cells.
export interface ISupplyExcelRow extends IProductSupplyRow {
  _productName: string
  _currentStock: number
  _providerCells: ISupplyProviderCell[]
  _updatedFields?: string[]
}

export interface ISupplyProviderCell {
  name: string
  capacity: number
  deliveryTime: number
  price: number
}

// Minimum provider groups the sheet always carries, even when no product has that many suppliers.
const MINIMUM_PROVIDER_GROUPS = 3

const makeEmptyProviderCell = (): ISupplyProviderCell => ({
  name: '', capacity: 0, deliveryTime: 0, price: 0,
})

const providerSlotFieldKey = (slotIndex: number) => `provider-${slotIndex}`

const readProviderCell = (row: ISupplyExcelRow, slotIndex: number): ISupplyProviderCell => {
  return row._providerCells?.[slotIndex] || makeEmptyProviderCell()
}

const writeProviderCell = (
  rowDraft: Partial<ISupplyExcelRow>,
  slotIndex: number,
  patch: Partial<ISupplyProviderCell>,
) => {
  const providerCells = rowDraft._providerCells || (rowDraft._providerCells = [])
  while (providerCells.length <= slotIndex) providerCells.push(makeEmptyProviderCell())
  Object.assign(providerCells[slotIndex], patch)
}

// Prices are persisted in cents; the sheet and the side panel both show currency units.
const priceToSheet = (priceInCents: number) => Number(((priceInCents || 0) / 100).toFixed(2))
const priceFromSheet = (sheetPrice: number) => Math.round((sheetPrice || 0) * 100)

export const countProviderGroups = (supplyRows: IProductSupplyRow[]): number => {
  let widestProviderCount = MINIMUM_PROVIDER_GROUPS
  for (const supplyRow of supplyRows) {
    const providerCount = (supplyRow.ProviderSupply || []).length
    if (providerCount > widestProviderCount) widestProviderCount = providerCount
  }
  return widestProviderCount
}

// buildSupplyExcelRows turns the page's table rows into sheet rows, resolving the names the file
// has to carry because an ID means nothing to whoever edits it.
export const buildSupplyExcelRows = (
  supplyRows: IProductSupplyRow[],
  productNameByID: Map<number, string>,
  providerNameByID: Map<number, string>,
  currentStockByProductID: Map<number, number>,
): ISupplyExcelRow[] => {
  return supplyRows.map((supplyRow) => ({
    ...supplyRow,
    _productName: productNameByID.get(supplyRow.ProductID) || '',
    _currentStock: currentStockByProductID.get(supplyRow.ProductID) || 0,
    _providerCells: (supplyRow.ProviderSupply || []).map((providerSupplyRow) => ({
      name: providerNameByID.get(providerSupplyRow.ProviderID) || '',
      capacity: providerSupplyRow.Capacity || 0,
      deliveryTime: providerSupplyRow.DeliveryTime || 0,
      price: priceToSheet(providerSupplyRow.Price || 0),
    })),
  }))
}

const updatedFieldCss = (row: ISupplyExcelRow, fieldKey: string): string | undefined => {
  return row._updatedFields?.includes(fieldKey) ? 'bg-purple-100' : undefined
}

// makeSupplyExcelColumns drives export, import and the preview table from one definition, so the
// three can never drift apart. providerGroupCount widens to the product with the most suppliers.
export const makeSupplyExcelColumns = (providerGroupCount: number): ExcelTableColumn<ISupplyExcelRow>[] => {
  const columns: ExcelTableColumn<ISupplyExcelRow>[] = [
    {
      id: 'product-name',
      header: 'Product|Producto',
      width: '280px',
      css: 'whitespace-normal leading-[1.15]',
      getValue: (row) => row._productName || '',
      setCellCss: (row) => updatedFieldCss(row, 'ProductID'),
      excel: { type: 'string', width: 46, importField: '_productName' },
    },
    {
      id: 'current-stock',
      header: 'Current Stock|Stock Actual',
      width: '90px',
      align: 'right',
      getValue: (row) => row._currentStock || 0,
      // Reconstructed from warehouse movements, so it maps to no field: the importer reads the
      // cell and assigns nothing. Exported because it is the number the buyer decides against.
      excel: { type: 'number' },
    },
    {
      id: 'minimum-stock',
      header: 'Minimum Stock|Stock Mínimo',
      width: '90px',
      align: 'right',
      field: 'MinimunStock',
      getValue: (row) => row.MinimunStock || 0,
      setCellCss: (row) => updatedFieldCss(row, 'MinimunStock'),
      excel: { type: 'number' },
    },
    {
      id: 'sales-per-day',
      header: 'Estimated Sales / Day|Ventas / Día Estimadas',
      width: '110px',
      align: 'right',
      field: 'SalesPerDayEstimated',
      getValue: (row) => row.SalesPerDayEstimated || 0,
      setCellCss: (row) => updatedFieldCss(row, 'SalesPerDayEstimated'),
      excel: { type: 'number' },
    },
  ]

  for (let slotIndex = 0; slotIndex < providerGroupCount; slotIndex++) {
    const slotFieldKey = providerSlotFieldKey(slotIndex)

    columns.push({
      id: `provider-group-${slotIndex + 1}`,
      header: `Supplier ${slotIndex + 1}|Proveedor ${slotIndex + 1}`,
      subcols: [
        {
          id: `provider-name-${slotIndex + 1}`,
          header: 'Supplier|Proveedor',
          width: '200px',
          css: 'whitespace-normal leading-[1.15]',
          getValue: (row) => readProviderCell(row, slotIndex).name,
          setCellCss: (row) => updatedFieldCss(row, slotFieldKey),
          excel: {
            type: 'string',
            width: 34,
            setValue: (rowDraft, _parsedValue, rawValue) => {
              writeProviderCell(rowDraft, slotIndex, { name: rawValue })
            },
          },
        },
        {
          id: `provider-capacity-${slotIndex + 1}`,
          header: 'Capacity|Capacidad',
          width: '80px',
          align: 'right',
          getValue: (row) => readProviderCell(row, slotIndex).capacity,
          setCellCss: (row) => updatedFieldCss(row, slotFieldKey),
          excel: {
            type: 'number',
            setValue: (rowDraft, parsedValue) => {
              writeProviderCell(rowDraft, slotIndex, { capacity: Number(parsedValue || 0) })
            },
          },
        },
        {
          id: `provider-delivery-${slotIndex + 1}`,
          header: 'Delivery|Entrega',
          width: '80px',
          align: 'right',
          getValue: (row) => readProviderCell(row, slotIndex).deliveryTime,
          setCellCss: (row) => updatedFieldCss(row, slotFieldKey),
          excel: {
            type: 'number',
            setValue: (rowDraft, parsedValue) => {
              writeProviderCell(rowDraft, slotIndex, { deliveryTime: Number(parsedValue || 0) })
            },
          },
        },
        {
          id: `provider-price-${slotIndex + 1}`,
          header: 'Price|Precio',
          width: '90px',
          align: 'right',
          getValue: (row) => readProviderCell(row, slotIndex).price,
          setCellCss: (row) => updatedFieldCss(row, slotFieldKey),
          excel: {
            type: 'number',
            format: '0.00',
            setValue: (rowDraft, parsedValue) => {
              writeProviderCell(rowDraft, slotIndex, { price: Number(parsedValue || 0) })
            },
          },
        },
      ],
    })
  }

  return columns
}

export const exportSupplyToExcel = async (
  columns: ExcelTableColumn<ISupplyExcelRow>[],
  rows: ISupplyExcelRow[],
): Promise<void> => {
  await downloadExcel({
    fileName: 'abastecimiento.xlsx',
    creator: 'Genix',
    includeTitleRow: true,
    includeGroupedHeaders: true,
    headerRowIndex: 2,
    sheet: {
      sheetName: 'Abastecimiento',
      title: 'Abastecimiento',
      columns,
      records: rows,
    },
  })
}

// AMBIGUOUS_MATCH marks a name that two different records normalize to: resolving it either way
// would write the configuration onto the wrong record, so the row is rejected instead.
const AMBIGUOUS_MATCH = -1

const buildNormalizedNameIndex = (
  records: { ID: number, Name?: string }[],
): Map<string, number> => {
  const idByNormalizedName = new Map<string, number>()
  for (const record of records) {
    const normalizedName = normalizeStringN(record.Name || '')
    if (!normalizedName || !record.ID) continue
    const existingID = idByNormalizedName.get(normalizedName)
    if (existingID !== undefined && existingID !== record.ID) {
      idByNormalizedName.set(normalizedName, AMBIGUOUS_MATCH)
      continue
    }
    idByNormalizedName.set(normalizedName, record.ID)
  }
  return idByNormalizedName
}

const normalizeProviderRowForDiff = (providerSupplyRow: IProductSupplyProviderRow) => {
  return [
    providerSupplyRow.ProviderID || 0,
    providerSupplyRow.Capacity || 0,
    providerSupplyRow.DeliveryTime || 0,
    providerSupplyRow.Price || 0,
  ].join('|')
}

export interface ISupplyImportResult {
  rows: ISupplyExcelRow[]
  errors: string[]
  mappedColumns: string[]
  ignoredHeaders: string[]
  unchangedCount: number
}

// processSupplyImportFile parses the sheet, resolves every name to an ID and returns only the rows
// whose configuration actually differs from what is persisted. A file exported and re-uploaded
// untouched therefore yields nothing to save.
export const processSupplyImportFile = async (
  columns: ExcelTableColumn<ISupplyExcelRow>[],
  source: File,
  products: { ID: number, Name?: string }[],
  providers: { ID: number, Name?: string }[],
  supplyRecordsByProductID: Map<number, IProductSupplyRow>,
): Promise<ISupplyImportResult> => {
  const builder = new ExcelBuilder<ISupplyExcelRow>()
    .setColumns(columns)
    // Grouped headers occupy two rows: the group titles and the four sub-columns under each.
    .setHeaderRows([2, 3])

  await builder.loadFile(source)

  const productIDByName = buildNormalizedNameIndex(products)
  const providerIDByName = buildNormalizedNameIndex(providers)

  const importResult = builder.extractRecords((row) => {
    const currentRow = row as ISupplyExcelRow
    const validationErrors: string[] = []

    const productName = (currentRow._productName || '').trim()
    if (!productName) {
      validationErrors.push('el producto es obligatorio')
    } else {
      const resolvedProductID = productIDByName.get(normalizeStringN(productName))
      if (resolvedProductID === undefined) {
        validationErrors.push(`producto no encontrado "${productName}"`)
      } else if (resolvedProductID === AMBIGUOUS_MATCH) {
        validationErrors.push(`hay más de un producto llamado "${productName}"`)
      } else {
        currentRow.ProductID = resolvedProductID
      }
    }

    const providerSupplyRows: IProductSupplyProviderRow[] = []
    const usedProviderIDs = new Set<number>()

    for (const providerCell of currentRow._providerCells || []) {
      const providerName = (providerCell?.name || '').trim()
      // A slot with a blank name but numbers filled in is a half-edited row, not an empty slot.
      if (!providerName) {
        if (providerCell?.capacity || providerCell?.deliveryTime || providerCell?.price) {
          validationErrors.push('hay una columna de proveedor con datos pero sin nombre')
        }
        continue
      }

      const resolvedProviderID = providerIDByName.get(normalizeStringN(providerName))
      if (resolvedProviderID === undefined) {
        validationErrors.push(`proveedor no encontrado "${providerName}"`)
        continue
      }
      if (resolvedProviderID === AMBIGUOUS_MATCH) {
        validationErrors.push(`hay más de un proveedor llamado "${providerName}"`)
        continue
      }
      if (usedProviderIDs.has(resolvedProviderID)) {
        validationErrors.push(`el proveedor "${providerName}" está repetido en la fila`)
        continue
      }
      usedProviderIDs.add(resolvedProviderID)

      providerSupplyRows.push({
        ProviderID: resolvedProviderID,
        Capacity: Math.round(providerCell.capacity || 0),
        DeliveryTime: Math.round(providerCell.deliveryTime || 0),
        Price: priceFromSheet(providerCell.price || 0),
      })
    }

    currentRow.ProviderSupply = providerSupplyRows
    currentRow.MinimunStock = Math.round(currentRow.MinimunStock || 0)
    currentRow.SalesPerDayEstimated = Math.round(currentRow.SalesPerDayEstimated || 0)

    if (currentRow.MinimunStock < 0) validationErrors.push('el stock mínimo no puede ser negativo')
    if (currentRow.SalesPerDayEstimated < 0) validationErrors.push('las ventas / día no pueden ser negativas')

    return validationErrors.length > 0 ? validationErrors : undefined
  })

  const changedRows: ISupplyExcelRow[] = []
  let unchangedCount = 0

  for (const importedRow of importResult.rowsWithoutErrors as ISupplyExcelRow[]) {
    // A product without a saved configuration diffs against an empty one, so every filled cell
    // shows up as a change and a row of zeros is correctly treated as nothing to save.
    const savedSupplyRecord = supplyRecordsByProductID.get(importedRow.ProductID)
    const updatedFieldKeys: string[] = []

    if ((savedSupplyRecord?.MinimunStock || 0) !== importedRow.MinimunStock) {
      updatedFieldKeys.push('MinimunStock')
    }
    if ((savedSupplyRecord?.SalesPerDayEstimated || 0) !== importedRow.SalesPerDayEstimated) {
      updatedFieldKeys.push('SalesPerDayEstimated')
    }

    const savedProviderRows = savedSupplyRecord?.ProviderSupply || []
    const providerSlotCount = Math.max(savedProviderRows.length, importedRow.ProviderSupply.length)
    for (let slotIndex = 0; slotIndex < providerSlotCount; slotIndex++) {
      const savedProviderRow = savedProviderRows[slotIndex]
      const importedProviderRow = importedRow.ProviderSupply[slotIndex]
      const savedKey = savedProviderRow ? normalizeProviderRowForDiff(savedProviderRow) : ''
      const importedKey = importedProviderRow ? normalizeProviderRowForDiff(importedProviderRow) : ''
      if (savedKey !== importedKey) {
        updatedFieldKeys.push(providerSlotFieldKey(slotIndex))
      }
    }

    if (updatedFieldKeys.length === 0) {
      unchangedCount++
      continue
    }

    importedRow._updatedFields = updatedFieldKeys
    changedRows.push(importedRow)
  }

  console.log('[supply-import] processed:', {
    parsedRows: importResult.rows.length,
    changedRows: changedRows.length,
    unchangedRows: unchangedCount,
    errors: importResult.errors.length,
    ignoredHeaders: importResult.ignoredHeaders,
  })

  return {
    rows: changedRows,
    errors: importResult.errors,
    mappedColumns: importResult.mappedColumns,
    ignoredHeaders: importResult.ignoredHeaders,
    unchangedCount,
  }
}
