<script lang="ts">
import Checkbox from '$components/Checkbox.svelte'
import Layer from '$components/Layer.svelte'
import SearchSelect from '$components/SearchSelect.svelte'
import LoadingBar from '$components/micro/LoadingBar.svelte'
import TableGrid from '$components/vTable/TableGrid.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { formatN, Loading, Notify, throttle } from '$libs/helpers'
import { getStaticRecordsByID } from '$libs/cache/cache-by-ids.svelte'
import { SvelteMap } from 'svelte/reactivity'
import { untrack } from 'svelte'
import { ProductosService } from '../../negocio/productos/productos.svelte'
import { AlmacenesService } from '../../negocio/sedes-almacenes/sedes-almacenes.svelte'
import {
  getWarehouseProductStock,
  makeStockID,
  postProductosStock,
  type IPostProductoStockItem,
  type IProductStockDetail,
  type IProductStockLot,
  type IProductoStock,
} from './stock-movement'
import type { ElementAST } from '$components/Renderer.svelte'

type IProductStockDetailRow = IProductStockDetail & {
  _cantidadPrev?: number
  _isNew?: boolean
  _hasUpdated?: boolean
  _panelType?: 'serial' | 'lot'
}

type IProductoStockDisplay = {
  base: IProductoStock
  groupKey: string
  lots: IProductStockDetailRow[]
  serialNumbers: IProductStockDetailRow[]
  serialNumbersCount: number
  serialNumbersCountNew: number
  stockSimple: number
  stockLoteado: number
  stockSerialNumbers: number
  stockSimpleNew: number
  stockLoteadoNew: number
  stockSerialNumbersNew: number
  computed?: boolean
}

const almacenes = new AlmacenesService()
const productos = new ProductosService(true)

let stockFilters = $state({ warehouseID: 0, showTodosProductos: false })
let stockFilterText = $state('')
let almacenStock = $state([] as IProductoStock[])
let almacenStockGetted = [] as IProductoStock[]
let selectedSerialNumberGroup = $state<IProductoStockDisplay | null>(null)
let selectedSerialNumbers = $state<IProductStockDetailRow[]>([])
let selectedLotGroup = $state<IProductoStockDisplay | null>(null)
let selectedLots = $state<IProductStockDetailRow[]>([])
let isSerialNumberLayerLoadingLots = $state(false)
let isLotLayerLoadingLots = $state(false)
let serialNumberLayerLotsRequestVersion = 0
let lotLayerLotsRequestVersion = 0
const newSerialNumberRowsByProductStockID = new Map<number, IProductStockDetailRow[]>()
const newLotRowsByProductStockID = new Map<number, IProductStockDetailRow[]>()
let pendingDetailRowsVersion = $state(0)

// Backing store for lot metadata resolved via getStaticRecordsByID.
// Reactive so the rendered lot names update once a fetch for the opened layer resolves.
const lotsByID = new SvelteMap<number, IProductStockLot>()

const getLotNameByID = (lotID?: number) => {
  if (!lotID) { return '' }
  const lotRecord = lotsByID.get(lotID)
  // Fall back to a placeholder while the static-cache fetch is in flight.
  return lotRecord?.Name || `LOT-${lotID}`
}

// Collect unique LotIDs from a slice of stock details and ensure their records are in `lotsByID`.
// The fetch is cache-first (memory → IDB → server), so repeat calls for already-loaded lots are free.
const ensureLotsLoadedForDetails = async (stockDetails: IProductStockDetailRow[]) => {
  const missingLotIDs: number[] = []
  const seenLotIDs = new Set<number>()
  for (const stockDetail of stockDetails) {
    const lotID = stockDetail.LotID || 0
    if (lotID <= 0) { continue }
    if (seenLotIDs.has(lotID)) { continue }
    seenLotIDs.add(lotID)
    if (lotsByID.has(lotID)) { continue }
    missingLotIDs.push(lotID)
  }
  if (missingLotIDs.length === 0) { return }

  const fetchedLotsByID = await getStaticRecordsByID<IProductStockLot>(
    'product-stock-lots-by-ids',
    missingLotIDs,
  )
  for (const [lotID, lotRecord] of fetchedLotsByID) {
    lotsByID.set(lotID, lotRecord)
  }
}

const hasMissingLotsForDetails = (stockDetails: IProductStockDetailRow[]) => {
  const seenLotIDs = new Set<number>()
  for (const stockDetail of stockDetails) {
    const lotID = stockDetail.LotID || 0
    if (lotID <= 0) { continue }
    if (seenLotIDs.has(lotID)) { continue }
    seenLotIDs.add(lotID)
    if (!lotsByID.has(lotID)) { return true }
  }
  return false
}

// Keep the layer open while lot metadata resolves, but avoid stale async completions
// toggling the loader for a newer request.
const loadLotsForSerialNumberLayer = async (stockDetails: IProductStockDetailRow[]) => {
  if (!hasMissingLotsForDetails(stockDetails)) {
    isSerialNumberLayerLoadingLots = false
    return
  }
  const requestVersion = ++serialNumberLayerLotsRequestVersion
  isSerialNumberLayerLoadingLots = true
  try {
    await ensureLotsLoadedForDetails(stockDetails)
  } finally {
    if (requestVersion === serialNumberLayerLotsRequestVersion) {
      isSerialNumberLayerLoadingLots = false
    }
  }
}

// Lot panels also depend on resolved lot metadata before the table can render names consistently.
const loadLotsForLotLayer = async (stockDetails: IProductStockDetailRow[]) => {
  if (!hasMissingLotsForDetails(stockDetails)) {
    isLotLayerLoadingLots = false
    return
  }
  const requestVersion = ++lotLayerLotsRequestVersion
  isLotLayerLoadingLots = true
  try {
    await ensureLotsLoadedForDetails(stockDetails)
  } finally {
    if (requestVersion === lotLayerLotsRequestVersion) {
      isLotLayerLoadingLots = false
    }
  }
}

// hasLotAssignment: true when the row is bound to a lot — either a server-assigned
// LotID (existing rows) or a user-entered LotCode (new rows awaiting backend resolution).
const hasLotAssignment = (stockDetail: IProductStockDetailRow) =>
  (stockDetail.LotID || 0) > 0 || !!stockDetail.LotCode

// Canonical per-row lot identity string used for uniqueness / row-id keys.
// New rows live in the "c:" namespace (LotCode); existing rows in "i:" (LotID).
const getLotIdentity = (stockDetail: IProductStockDetailRow): string => {
  if (stockDetail.LotCode) { return `c:${stockDetail.LotCode}` }
  return `i:${stockDetail.LotID || 0}`
}

const getLotDisplay = (stockDetail: IProductStockDetailRow): string => {
  if (stockDetail.LotCode) { return stockDetail.LotCode }
  return getLotNameByID(stockDetail.LotID)
}

const getCantPrevia = (record: { Quantity: number, _cantidadPrev?: number }) => {
  if (record._cantidadPrev === -1) { return 0 }
  return typeof record._cantidadPrev === 'number' && record._cantidadPrev !== -1 ? record._cantidadPrev : record.Quantity
}

const hasPendingDetailContent = (stockDetail: IProductStockDetailRow) => {
  return !!stockDetail.SerialNumber || hasLotAssignment(stockDetail) || (stockDetail.Quantity || 0) > 0 || (stockDetail.SubQuantity || 0) > 0
}

const shouldPersistNewDetailRow = (stockDetail: IProductStockDetailRow) => {
  // New serial rows only count as pending when they already define a quantity to persist.
  if (stockDetail.SerialNumber) {
    return (stockDetail.Quantity || 0) > 0 || (stockDetail.SubQuantity || 0) > 0
  }
  return hasPendingDetailContent(stockDetail)
}

const getNewDetailRowsMap = (panelType: 'serial' | 'lot') => {
  return panelType === 'serial' ? newSerialNumberRowsByProductStockID : newLotRowsByProductStockID
}

const getNewDetailRows = (productStockRecord: IProductoStock, panelType: 'serial' | 'lot') => {
  return getNewDetailRowsMap(panelType).get(productStockRecord.ID) || []
}

// Keep editable rows grouped at the top of the layer so the user always sees
// recently created entries first, then the empty draft row, and finally the locked server rows.
const orderLayerDetailRows = (
  stockDetails: IProductStockDetailRow[],
  panelType: 'serial' | 'lot',
) => {
  const editableFilledRows = stockDetails.filter((stockDetail) =>
    stockDetail._isNew && (panelType === 'serial' ? !!stockDetail.SerialNumber : hasLotAssignment(stockDetail)),
  )
  const editableEmptyRows = stockDetails.filter((stockDetail) =>
    stockDetail._isNew && (panelType === 'serial' ? !stockDetail.SerialNumber : !hasLotAssignment(stockDetail)),
  )
  const persistedRows = stockDetails.filter((stockDetail) => !stockDetail._isNew)

  return [...editableFilledRows, ...editableEmptyRows, ...persistedRows]
}

const touchPendingDetailRows = (reason: string, productStockID?: number) => {
  pendingDetailRowsVersion++
}

const persistNewDetailRows = (productStockRecord: IProductoStock, panelType: 'serial' | 'lot') => {
  const stockDetails = (panelType === 'serial' ? selectedSerialNumbers : selectedLots)
    .filter((stockDetail) => stockDetail._isNew && shouldPersistNewDetailRow(stockDetail))

  const detailRowsMap = getNewDetailRowsMap(panelType)
  if (stockDetails.length > 0) {
    detailRowsMap.set(productStockRecord.ID, stockDetails)
  } else {
    detailRowsMap.delete(productStockRecord.ID)
  }
  touchPendingDetailRows(`persist-${panelType}`, productStockRecord.ID)
}

const getSerialNumberRows = (productStockRecord: IProductoStock) => {
  return [
    ...(productStockRecord.StockDetails as IProductStockDetailRow[])
      .filter((stockDetail) => !!stockDetail.SerialNumber),
    ...getNewDetailRows(productStockRecord, 'serial')
      .filter((stockDetail) => !!stockDetail.SerialNumber),
  ]
}

const getLotRows = (productStockRecord: IProductoStock) => {
  return [
    ...(productStockRecord.StockDetails as IProductStockDetailRow[])
      .filter((stockDetail) => !stockDetail.SerialNumber && hasLotAssignment(stockDetail)),
    ...getNewDetailRows(productStockRecord, 'lot')
      .filter((stockDetail) => !stockDetail.SerialNumber && hasLotAssignment(stockDetail)),
  ]
}

const createPendingStockDetail = (
  productStockRecord: IProductoStock,
  panelType: 'serial' | 'lot',
): IProductStockDetailRow => ({
  ProductStockID: productStockRecord.ID,
  LotID: 0,
  LotCode: '',
  SerialNumber: '',
  WarehouseID: productStockRecord.WarehouseID,
  ProductID: productStockRecord.ProductID,
  Quantity: 0,
  SubQuantity: 0,
  _isNew: true,
  _panelType: panelType,
})

const addPendingSerialNumberRow = () => {
  if (!selectedSerialNumberGroup) { return }
  if (selectedSerialNumbers.some((stockDetail) => stockDetail._isNew && !stockDetail.SerialNumber)) { return }

  const pendingStockDetail = createPendingStockDetail(selectedSerialNumberGroup.base, 'serial')
  const newDetailRows = getNewDetailRows(selectedSerialNumberGroup.base, 'serial')
  newSerialNumberRowsByProductStockID.set(selectedSerialNumberGroup.base.ID, [...newDetailRows, pendingStockDetail])
  selectedSerialNumbers = orderLayerDetailRows([...selectedSerialNumbers, pendingStockDetail], 'serial')
  selectedSerialNumberGroup.computed = false
  touchPendingDetailRows('add-pending-serial', selectedSerialNumberGroup.base.ID)
}

const addPendingLotRow = () => {
  if (!selectedLotGroup) { return }
  if (selectedLots.some((stockDetail) => stockDetail._isNew && !hasLotAssignment(stockDetail))) { return }

  const pendingStockDetail = createPendingStockDetail(selectedLotGroup.base, 'lot')
  const newDetailRows = getNewDetailRows(selectedLotGroup.base, 'lot')
  newLotRowsByProductStockID.set(selectedLotGroup.base.ID, [...newDetailRows, pendingStockDetail])
  selectedLots = orderLayerDetailRows([...selectedLots, pendingStockDetail], 'lot')
  selectedLotGroup.computed = false
  touchPendingDetailRows('add-pending-lot', selectedLotGroup.base.ID)
}

const openDetailLayer = (
  panelType: 'serial' | 'lot',
  productStockDisplay: IProductoStockDisplay,
  rerender: () => void,
) => {
  rerenderHandler = rerender

  if (panelType === 'serial') {
    selectedSerialNumberGroup = productStockDisplay
    selectedSerialNumbers = orderLayerDetailRows([
      ...(productStockDisplay.base.StockDetails as IProductStockDetailRow[]).filter((stockDetail) => !!stockDetail.SerialNumber),
      ...getNewDetailRows(productStockDisplay.base, 'serial'),
    ], 'serial')
    addPendingSerialNumberRow()
    void loadLotsForSerialNumberLayer(selectedSerialNumbers)
    Core.openSideLayer(2)
    return
  }

  selectedLotGroup = productStockDisplay
  selectedLots = orderLayerDetailRows([
    ...(productStockDisplay.base.StockDetails as IProductStockDetailRow[]).filter((stockDetail) => !stockDetail.SerialNumber && hasLotAssignment(stockDetail)),
    ...getNewDetailRows(productStockDisplay.base, 'lot'),
  ], 'lot')
  addPendingLotRow()
  void loadLotsForLotLayer(selectedLots)
  Core.openSideLayer(3)
}

const closeSerialNumberLayer = () => {
  serialNumberLayerLotsRequestVersion++
  isSerialNumberLayerLoadingLots = false
  if (selectedSerialNumberGroup) {
    persistNewDetailRows(selectedSerialNumberGroup.base, 'serial')
    selectedSerialNumberGroup.computed = false
    rerenderHandler?.()
  }
  selectedSerialNumbers = []
  selectedSerialNumberGroup = null
}

const closeLotLayer = () => {
  lotLayerLotsRequestVersion++
  isLotLayerLoadingLots = false
  if (selectedLotGroup) {
    persistNewDetailRows(selectedLotGroup.base, 'lot')
    selectedLotGroup.computed = false
    rerenderHandler?.()
  }
  selectedLots = []
  selectedLotGroup = null
}

const displayStock = $derived.by((): IProductoStockDisplay[] => {
  almacenStock
  pendingDetailRowsVersion

  const productStockByGroup = new Map<string, IProductoStockDisplay>()
  const result: IProductoStockDisplay[] = []

  untrack(() => {
    for (const productStockRecord of almacenStock) {
      const groupKey = [productStockRecord.ProductID, productStockRecord.PresentationID || 0].join('_')
      productStockByGroup.set(groupKey, {
        base: productStockRecord,
        groupKey,
        lots: getLotRows(productStockRecord),
        serialNumbers: getSerialNumberRows(productStockRecord),
        serialNumbersCount: 0,
        serialNumbersCountNew: 0,
        stockSimple: 0,
        stockLoteado: 0,
        stockSerialNumbers: 0,
        stockSimpleNew: 0,
        stockLoteadoNew: 0,
        stockSerialNumbersNew: 0,
      })
    }

    for (const productStockDisplay of productStockByGroup.values()) {
      result.push(productStockDisplay)
    }
  })

  return result
})

const selectedSerialNumberGroupTitle = $derived.by(() => {
  if (!selectedSerialNumberGroup) { return 'Seriales' }
  const productName = productos.recordsMap.get(selectedSerialNumberGroup.base.ProductID)?.Nombre || ''
  const count = selectedSerialNumbers.filter((stockDetail) => stockDetail.SerialNumber).length
  return `${productName} — ${count} Serial${count !== 1 ? 'es' : ''}`
})

const selectedLotGroupTitle = $derived.by(() => {
  if (!selectedLotGroup) { return 'Lotes' }
  const productName = productos.recordsMap.get(selectedLotGroup.base.ProductID)?.Nombre || ''
  const count = selectedLots.filter(hasLotAssignment).length
  return `${productName} — ${count} Lote${count !== 1 ? 's' : ''}`
})

const validateSerialNumberUnique = (
  stockDetailRecord: IProductStockDetailRow,
  checkSerialNumber: string,
  checkLotIdentity: string,
): boolean => {
  if (!checkSerialNumber) { return true }

  const duplicate = selectedSerialNumbers.some((stockDetail) =>
    stockDetail !== stockDetailRecord
    && (stockDetail.SerialNumber || '') === checkSerialNumber
    && getLotIdentity(stockDetail) === checkLotIdentity,
  )
  if (duplicate) { Notify.warning('La combinación Lote + Serial ya existe.') }
  return !duplicate
}

const validateLotUnique = (stockDetailRecord: IProductStockDetailRow, checkLotCode: string): boolean => {
  if (!checkLotCode) { return true }

  const duplicate = selectedLots.some((stockDetail) =>
    stockDetail !== stockDetailRecord
    && (stockDetail.LotCode || '') === checkLotCode,
  )
  if (duplicate) { Notify.warning('El Lote ya existe.') }
  return !duplicate
}

const updateStockDetailQuantity = (stockDetailRecord: IProductStockDetailRow, quantity: number) => {
  stockDetailRecord._cantidadPrev = typeof stockDetailRecord._cantidadPrev === 'number'
    ? stockDetailRecord._cantidadPrev
    : stockDetailRecord._isNew ? -1 : stockDetailRecord.Quantity
  stockDetailRecord.Quantity = quantity
  stockDetailRecord._hasUpdated = true
}

const serialNumberColumns: ITableColumn<IProductStockDetailRow>[] = [
  {
    header: 'Serial', cellCss: 'ff-mono',
    getValue: (stockDetail) => stockDetail.SerialNumber || '',
    disableCellInteractions: (stockDetail) => !stockDetail._isNew,
    onBeforeCellChange: (stockDetail, value) => {
      return validateSerialNumberUnique(stockDetail, String(value || '').trim(), getLotIdentity(stockDetail))
    },
    onCellEdit: (stockDetail, value) => {
      const nextSerialNumber = String(value || '').trim()
      const hadSerialNumber = !!stockDetail.SerialNumber
      stockDetail.SerialNumber = nextSerialNumber
      // The first time a new serial is defined, default its quantity to 1 unless the user already set one.
      if (stockDetail._isNew && !hadSerialNumber && nextSerialNumber && !(stockDetail.Quantity > 0 || stockDetail.SubQuantity > 0)) {
        stockDetail.Quantity = 1
      }
      stockDetail._hasUpdated = true
      if (stockDetail._isNew && stockDetail.SerialNumber) {
        addPendingSerialNumberRow()
      }
      if (selectedSerialNumberGroup) {
        selectedSerialNumberGroup.computed = false
      }
    },
    render: (stockDetail) => {
      if (!stockDetail.SerialNumber) { return { css: 'text-xs c-red', text: 'NUEVO SERIAL...' } }
      if (stockDetail._isNew) { return { css: 'text-blue-800', text: stockDetail.SerialNumber } }
      return stockDetail.SerialNumber
    },
  },
  {
    header: 'Lote',
    getValue: (stockDetail) => getLotDisplay(stockDetail),
    disableCellInteractions: (stockDetail) => !stockDetail._isNew,
    onBeforeCellChange: (stockDetail, value) => {
      const nextLotCode = String(value || '').trim()
      const nextLotIdentity = nextLotCode ? `c:${nextLotCode}` : 'i:0'
      return validateSerialNumberUnique(stockDetail, stockDetail.SerialNumber || '', nextLotIdentity)
    },
    onCellEdit: (stockDetail, value) => {
      stockDetail.LotCode = String(value || '').trim()
      stockDetail._hasUpdated = true
      if (selectedSerialNumberGroup) {
        selectedSerialNumberGroup.computed = false
      }
    },
    render: (stockDetail) => {
      const lotDisplay = getLotDisplay(stockDetail)
      if (stockDetail._isNew && lotDisplay) { return { css: 'text-blue-800', text: lotDisplay } }
      return lotDisplay
    },
  },
  {
    header: 'Cant.', css: 'justify-end', inputCss: 'text-right pr-6',
    getValue: (stockDetail) => stockDetail.Quantity ?? '',
    cellInputType: 'number',
    onCellEdit: (stockDetail, value) => {
      updateStockDetailQuantity(stockDetail, parseInt(String(value || '0')))
      if (selectedSerialNumberGroup) {
        selectedSerialNumberGroup.computed = false
      }
    },
    render: (stockDetail) => {
      if (stockDetail._cantidadPrev && stockDetail._cantidadPrev !== stockDetail.Quantity) {
        return {
          css: 'flex items-center',
          children: [
            { text: String(stockDetail._cantidadPrev > 0 ? stockDetail._cantidadPrev : 0) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: stockDetail.Quantity, css: 'text-red-500' },
          ],
        }
      }
      return { css: `text-right ${stockDetail._isNew ? 'text-blue-800' : ''}`.trim(), text: stockDetail.Quantity || '' }
    },
  },
]

const lotColumns: ITableColumn<IProductStockDetailRow>[] = [
  {
    header: 'Lote',
    getValue: (stockDetail) => getLotDisplay(stockDetail),
    disableCellInteractions: (stockDetail) => !stockDetail._isNew,
    onBeforeCellChange: (stockDetail, value) => validateLotUnique(stockDetail, String(value || '').trim()),
    onCellEdit: (stockDetail, value) => {
      stockDetail.LotCode = String(value || '').trim()
      stockDetail._hasUpdated = true
      if (stockDetail._isNew && stockDetail.LotCode) {
        addPendingLotRow()
      }
      if (selectedLotGroup) {
        selectedLotGroup.computed = false
      }
    },
    render: (stockDetail) => {
      if (!hasLotAssignment(stockDetail)) { return { css: 'text-xs c-red', text: 'NUEVO LOTE...' } }
      const lotDisplay = getLotDisplay(stockDetail)
      if (stockDetail._isNew) { return { css: 'text-blue-800', text: lotDisplay } }
      return lotDisplay
    },
  },
  {
    header: 'Cant.', css: 'justify-end', inputCss: 'text-right pr-6',
    getValue: (stockDetail) => stockDetail.Quantity ?? '',
    cellInputType: 'number',
    onCellEdit: (stockDetail, value) => {
      updateStockDetailQuantity(stockDetail, parseInt(String(value || '0')))
      if (selectedLotGroup) {
        selectedLotGroup.computed = false
      }
    },
    render: (stockDetail) => {
      if (stockDetail._cantidadPrev && stockDetail._cantidadPrev !== stockDetail.Quantity) {
        return {
          css: 'flex items-center',
          children: [
            { text: String(stockDetail._cantidadPrev > 0 ? stockDetail._cantidadPrev : 0) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: stockDetail.Quantity, css: 'text-red-500' },
          ],
        }
      }
      return { css: `text-right ${stockDetail._isNew ? 'text-blue-800' : ''}`.trim(), text: stockDetail.Quantity || '' }
    },
  },
]

const computeStockDisplay = (productStockDisplay: IProductoStockDisplay) => {
  if (productStockDisplay.computed) { return }

  productStockDisplay.serialNumbersCount = (productStockDisplay.base.StockDetails as IProductStockDetailRow[])
    .filter((stockDetail) => !!stockDetail.SerialNumber)
    .length
  productStockDisplay.serialNumbersCountNew = productStockDisplay.serialNumbers.length

  productStockDisplay.stockSimple = getCantPrevia(productStockDisplay.base)
  productStockDisplay.stockSimpleNew = productStockDisplay.base.Quantity || 0

  productStockDisplay.stockLoteado = 0
  productStockDisplay.stockLoteadoNew = 0
  for (const stockDetail of productStockDisplay.lots) {
    productStockDisplay.stockLoteado += getCantPrevia(stockDetail) || 0
    productStockDisplay.stockLoteadoNew += stockDetail.Quantity || 0
  }

  productStockDisplay.stockSerialNumbers = 0
  productStockDisplay.stockSerialNumbersNew = 0
  for (const stockDetail of productStockDisplay.serialNumbers) {
    productStockDisplay.stockSerialNumbers += getCantPrevia(stockDetail) || 0
    productStockDisplay.stockSerialNumbersNew += stockDetail.Quantity || 0
  }

  productStockDisplay.computed = true
}

const stockColumns: ITableColumn<IProductoStockDisplay>[] = [
  {
    header: 'Producto', highlight: true,
    mobile: { order: 1, css: 'col-span-24' },
    getValue: (productStockDisplay) => {
      const productRecord = productos.recordsMap.get(productStockDisplay.base.ProductID)?.Nombre
      return productRecord || `Producto-${productStockDisplay.base.ProductID}`
    },
    render: (productStockDisplay) => {
      const productRecord = productos.recordsMap.get(productStockDisplay.base.ProductID)?.Nombre || `Producto-${productStockDisplay.base.ProductID}`
      return productRecord
    },
  },
  {
    header: 'Presentación',
    mobile: { order: 2, css: 'col-span-24' },
    getValue: (productStockDisplay) => {
      if (!productStockDisplay.base.PresentationID) { return '' }
      const productRecord = productos.recordsMap.get(productStockDisplay.base.ProductID)
      const presentationRecord = productRecord?.Presentaciones?.find((presentationOption) => presentationOption.id === productStockDisplay.base.PresentationID)
      return presentationRecord?.nm || `Tipo-${productStockDisplay.base.PresentationID}`
    },
  },
  {
    header: 'Seriales',
    getValue: (productStockDisplay) => {
      computeStockDisplay(productStockDisplay)
      return productStockDisplay.serialNumbersCountNew
    },
    mobile: { 
    	order: 3, 
     	css: 'col-span-12', 
     	labelLeft: "Seriales",
      contentCss: "flex items-center justify-end pr-4 bg-purple-light",
    },
    showEditIcon: true,
    onCellClick: (productStockDisplay, index, rerender) => openDetailLayer('serial', productStockDisplay, rerender),
    render: (productStockDisplay) => {
      computeStockDisplay(productStockDisplay)
      if (productStockDisplay.serialNumbersCountNew !== productStockDisplay.serialNumbersCount) {
        return {
          css: 'flex items-center justify-end',
          children: [
            { text: String(productStockDisplay.serialNumbersCount) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: productStockDisplay.serialNumbersCountNew, css: 'text-red-500' },
          ],
        }
      }
      return { css: 'text-right', text: productStockDisplay.serialNumbersCountNew || '' }
    },
  },
  {
    header: 'Stock Simple', css: 'justify-end px-6', inputCss: 'text-right',
    headerCss: 'w-150',
    mobile: { order: 4, css: 'col-span-12', labelLeft: "Simple" },
    showEditIcon: true,
    cellInputType: 'number',
    getValue: (productStockDisplay) => {
      computeStockDisplay(productStockDisplay)
      return productStockDisplay.stockSimpleNew
    },
    onCellEdit: (productStockDisplay, value, rerender) => {
      productStockDisplay.computed = false
      updateStockQuantity(productStockDisplay.base, parseInt(String(value || '0')))
      rerender()
    },
    render: (productStockDisplay) => {
      if (productStockDisplay.stockSimpleNew !== productStockDisplay.stockSimple) {
        return {
          css: 'flex items-center justify-end',
          children: [
            { text: String(productStockDisplay.stockSimple) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: productStockDisplay.stockSimpleNew, css: 'text-red-500' },
          ],
        }
      }
      return { css: 'text-right', text: productStockDisplay.stockSimpleNew || '' }
    },
  },
  {
    header: 'Stock Loteado',
    showEditIcon: true,
    mobile: { 
    	order: 5, 
     	css: 'col-span-12', 
     	labelLeft: "Loteado",
      contentCss: "flex items-center justify-end bg-purple-light pr-4",
    },
    onCellClick: (productStockDisplay, index, rerender) => openDetailLayer('lot', productStockDisplay, rerender),
    getValue: (productStockDisplay) => {
      computeStockDisplay(productStockDisplay)
      return productStockDisplay.stockLoteadoNew
    },
    render: (productStockDisplay) => {
      computeStockDisplay(productStockDisplay)
      if (productStockDisplay.stockLoteadoNew !== productStockDisplay.stockLoteado) {
        return {
          css: 'flex justify-end items-center',
          children: [
            { text: String(productStockDisplay.stockLoteado) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: productStockDisplay.stockLoteadoNew, css: 'text-red-500' },
          ],
        }
      }
      return { css: 'text-right', text: productStockDisplay.stockLoteadoNew || '' }
    },
  },
  {
    header: 'Stock Total', css: 'justify-end text-right',
    mobile: { order: 6, css: 'col-span-12 pr-4', labelLeft: "Total" },
    getValue: (productStockDisplay) => {
      computeStockDisplay(productStockDisplay)
      return formatN(productStockDisplay.stockLoteadoNew + productStockDisplay.stockSimpleNew + productStockDisplay.stockSerialNumbersNew)
    },
  },
]

const onChangeAlmacen = async () => {
  if (!stockFilters.warehouseID) { return }

  Loading.standard()
  try {
    const result = await getWarehouseProductStock(stockFilters.warehouseID)
    newSerialNumberRowsByProductStockID.clear()
    newLotRowsByProductStockID.clear()
    touchPendingDetailRows('warehouse-change-clear')
    almacenStock = result || []
    almacenStockGetted = result || []
  } finally {
    Loading.remove()
  }
}

const guardarRegistros = async () => {
  const seenDetailKeys = new Set<string>()
  for (const productStockRecord of almacenStock) {
    const stockDetails = [
      ...(productStockRecord.StockDetails as IProductStockDetailRow[]),
      ...getNewDetailRows(productStockRecord, 'serial'),
      ...getNewDetailRows(productStockRecord, 'lot'),
    ]
    for (const stockDetail of stockDetails) {
      if (stockDetail._isNew && !shouldPersistNewDetailRow(stockDetail)) { continue }
      if (!stockDetail.SerialNumber && !hasLotAssignment(stockDetail)) { continue }

      const detailKey = [productStockRecord.ID, getLotIdentity(stockDetail), stockDetail.SerialNumber || ''].join('_')
      if (seenDetailKeys.has(detailKey)) {
        Notify.failure('Hay detalles duplicados (mismo Lote + Serial). Verifica antes de guardar.')
        return
      }
      seenDetailKeys.add(detailKey)
    }
  }

  const recordsForUpdate: IPostProductoStockItem[] = []

  for (const productStockRecord of almacenStock) {
    if (productStockRecord._hasUpdated) {
      recordsForUpdate.push({
        WarehouseID: productStockRecord.WarehouseID,
        ProductID: productStockRecord.ProductID,
        PresentationID: productStockRecord.PresentationID || 0,
        Quantity: productStockRecord.Quantity || 0,
        SubQuantity: productStockRecord.SubQuantity || 0,
      })
    }

    const stockDetails = [
      ...(productStockRecord.StockDetails as IProductStockDetailRow[]),
      ...getNewDetailRows(productStockRecord, 'serial'),
      ...getNewDetailRows(productStockRecord, 'lot'),
    ]
    for (const stockDetail of stockDetails) {
      if (stockDetail._isNew && !shouldPersistNewDetailRow(stockDetail)) { continue }
      if (!stockDetail._hasUpdated && !(stockDetail._isNew && hasPendingDetailContent(stockDetail))) { continue }
      if (!stockDetail.SerialNumber && !hasLotAssignment(stockDetail)) { continue }

      recordsForUpdate.push({
        WarehouseID: productStockRecord.WarehouseID,
        ProductID: productStockRecord.ProductID,
        PresentationID: productStockRecord.PresentationID || 0,
        Quantity: stockDetail.Quantity || 0,
        SubQuantity: stockDetail.SubQuantity || 0,
        SerialNumber: stockDetail.SerialNumber || '',
        LotID: stockDetail.LotID || 0,
        LotCode: stockDetail.LotCode || '',
      })
    }
  }

  if (recordsForUpdate.length === 0) {
    Notify.failure('No hay registros a actualizar.')
    return
  }

  Loading.standard('Enviando registros...')
  try {
    await postProductosStock(recordsForUpdate)

    for (const productStockRecord of almacenStock) {
      // After a successful save, the current quantity becomes the new baseline for diff rendering.
      productStockRecord._cantidadPrev = undefined
      productStockRecord._hasUpdated = false
      productStockRecord._isNew = false

      const persistedStockDetails = [
        ...(productStockRecord.StockDetails as IProductStockDetailRow[]),
        ...getNewDetailRows(productStockRecord, 'serial'),
        ...getNewDetailRows(productStockRecord, 'lot'),
      ]
        .filter((stockDetail) => stockDetail.SerialNumber || hasLotAssignment(stockDetail))
      productStockRecord.StockDetails = persistedStockDetails

      for (const stockDetail of persistedStockDetails) {
        // Clear pending diff state so persisted detail rows no longer render as changed.
        stockDetail._cantidadPrev = undefined
        stockDetail._hasUpdated = false
        stockDetail._isNew = false
        stockDetail._panelType = undefined
      }

      newSerialNumberRowsByProductStockID.delete(productStockRecord.ID)
      newLotRowsByProductStockID.delete(productStockRecord.ID)
      touchPendingDetailRows('save-clear', productStockRecord.ID)
    }

    rerenderHandler?.()
  } finally {
    Loading.remove()
  }
}

const fillAllProductos = () => {
  const productStockMap = new Map<number, IProductoStock>(
    almacenStockGetted.map((productStockRecord) => [productStockRecord.ID, productStockRecord]),
  )

  for (const productRecord of productos.records) {
    const presentationIDs = productRecord.Presentaciones?.length > 0
      ? productRecord.Presentaciones.map((presentationOption) => presentationOption.id)
      : [0]

    for (const presentationID of presentationIDs) {
      const stockID = makeStockID({
        WarehouseID: stockFilters.warehouseID,
        ProductID: productRecord.ID,
        PresentationID: presentationID,
      })

      if (productStockMap.has(stockID)) {
        const existingProductStock = productStockMap.get(stockID) as IProductoStock
        existingProductStock.PresentationID = presentationID
        continue
      }

      productStockMap.set(stockID, {
        ID: stockID,
        WarehouseID: stockFilters.warehouseID,
        ProductID: productRecord.ID,
        PresentationID: presentationID,
        Quantity: 0,
        SubQuantity: 0,
        StockDetails: [],
      })
    }
  }

  almacenStock = [...productStockMap.values()]
}

const updateStockQuantity = (productStockRecord: IProductoStock, quantity: number) => {
  const existingProductStock = almacenStock.find((rowRecord) => rowRecord.ID === productStockRecord.ID)
    || almacenStockGetted.find((rowRecord) => rowRecord.ID === productStockRecord.ID)

  if (existingProductStock) {
    existingProductStock._hasUpdated = true
    existingProductStock._cantidadPrev = typeof existingProductStock._cantidadPrev === 'number'
      ? existingProductStock._cantidadPrev
      : existingProductStock.Quantity || -1
    existingProductStock.Quantity = quantity
  } else {
    productStockRecord.ID = makeStockID(productStockRecord)
    productStockRecord.StockDetails = productStockRecord.StockDetails || []
    productStockRecord._cantidadPrev = -1
    productStockRecord._hasUpdated = true
    almacenStockGetted.unshift(productStockRecord)
    almacenStock.unshift(productStockRecord)
  }
}

$effect(() => {
  if (stockFilters.showTodosProductos) {
    untrack(() => { fillAllProductos() })
  } else {
    untrack(() => { almacenStock = almacenStockGetted })
  }
})

let rerenderHandler: ((() => void) | undefined) = undefined
</script>

<div class="grid grid-cols-24 gap-8 mb-8 items-center md:flex">
  <div class="col-span-14 md:col-span-5 min-w-0 mr-8">
    <SearchSelect options={almacenes?.Almacenes || []} keyId="ID" keyName="Nombre"
      bind:saveOn={stockFilters} save="warehouseID" placeholder="ALMACÉN ::"
      css="w-full md:w-240" id={1} useCache
      onChange={() => {
        onChangeAlmacen()
      }}
    />
  </div>

  <div class="col-span-10 md:col-span-5 md:order-3 min-w-0 flex justify-end gap-8 md:ml-auto">
    {#if stockFilters.warehouseID > 0}
      <button class="bx-blue shrink-0" onclick={() => {
        guardarRegistros()
      }}>
        <i class="icon-floppy"></i>Guardar
      </button>
    {/if}
  </div>

  {#if !stockFilters.warehouseID}
    <div class="col-span-24 text-red-500"><i class="icon-attention"></i>Debe seleccionar un almacén.</div>
  {:else}
    <div class="col-span-12 md:col-span-4 md:order-1 min-w-0 i-search w-full md:w-180">
      <div><i class="icon-search"></i></div>
      <input type="text" onkeyup={(event) => {
        const value = String((event.target as HTMLInputElement).value || '')
        throttle(() => { stockFilterText = value }, 150)
      }}>
    </div>
    <Checkbox label="Todos los Productos" bind:saveOn={stockFilters} save="showTodosProductos"
      css="col-span-12 md:col-span-5 md:order-2 min-w-0 self-center" />
  {/if}
</div>

<VTable columns={stockColumns} data={displayStock}
  filterText={stockFilterText}
  useFilterCache={true}
  getFilterContent={(productStockDisplay) => {
    const productRecord = productos.recordsMap.get(productStockDisplay.base.ProductID)
    const serialText = productStockDisplay.serialNumbers.map((stockDetail) => stockDetail.SerialNumber).filter(Boolean).join(' ')
    const lotText = productStockDisplay.lots.map(getLotDisplay).filter(Boolean).join(' ')
    return [productRecord?.Nombre, serialText, lotText].filter((value) => value).join(' ').toLowerCase()
  }}
/>

<Layer id={2} type="side" sideLayerSize={740} title={selectedSerialNumberGroupTitle} titleCss="h2"
  css="px-12 py-10"
  onClose={() => {
    closeSerialNumberLayer()
  }}
>
  {#if isSerialNumberLayerLoadingLots}
    <div class="mt-6 p-8 min-h-240 w-full fx-c rounded-md bg-gray-50">
      <LoadingBar label="Cargando Lotes..." />
    </div>
  {:else}
    <TableGrid columns={serialNumberColumns} data={selectedSerialNumbers} height="calc(100vh - 14rem)"
      cellCss="px-6" headerCss="px-6 flex items-center min-h-32" css="mt-6"
      getRowId={(stockDetail) => `${getLotIdentity(stockDetail)}_${stockDetail.SerialNumber || 'pending'}_${stockDetail._panelType || ''}`}
    />
  {/if}
</Layer>

<Layer id={3} type="side" sideLayerSize={620} title={selectedLotGroupTitle} titleCss="h2"
  css="px-12 py-10"
  onClose={() => {
    closeLotLayer()
  }}
>
  {#if isLotLayerLoadingLots}
    <div class="mt-6 p-8 min-h-240 w-full fx-c rounded-md bg-gray-50">
      <LoadingBar label="Cargando Lotes..." />
    </div>
  {:else}
    <TableGrid columns={lotColumns} data={selectedLots} height="calc(100vh - 14rem)"
      cellCss="px-6" headerCss="px-6 flex items-center min-h-32" css="mt-6"
      getRowId={(stockDetail) => `${getLotIdentity(stockDetail)}_${stockDetail._panelType || ''}`}
    />
  {/if}
</Layer>
