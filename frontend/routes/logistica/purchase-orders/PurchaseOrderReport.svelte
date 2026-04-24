<script lang="ts">
import ButtonLayer from '$components/ButtonLayer.svelte'
import DateInput from '$components/DateInput.svelte'
import Layer from '$components/Layer.svelte'
import SearchSelect from '$components/SearchSelect.svelte'
import KeyValueStrip from '$components/micro/KeyValueStrip.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { formatN, formatTime, Loading, throttle } from '$libs/helpers'
import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import { ProductosService, type IProducto } from '$routes/negocio/productos/productos.svelte'
import { onDestroy } from 'svelte'
import {
  PurchaseOrdersService,
  PurchaseOrderStatus,
  purchaseOrderStatusOptions,
  queryPurchaseOrders,
  type IPurchaseOrder,
} from './purchase_order.svelte'

const providersService = new ClientProviderService(ClientProviderType.PROVIDER, true)
const productosService = new ProductosService(true)
const purchaseOrdersService = new PurchaseOrdersService(PurchaseOrderStatus.PENDING, true)

const getCurrentUnixDay = (): number => Math.floor(Date.now() / (1000 * 60 * 60 * 24))

// Default filter window: last 30 days. User can widen up to 1 year (backend cap).
const fechaFinDefault = getCurrentUnixDay()
const fechaInicioDefault = fechaFinDefault - 30

const defaultReportForm = {
  fechaInicio: fechaInicioDefault,
  fechaFin: fechaFinDefault,
  status: 0,
  productID: 0,
  providerID: 0,
}

let reportForm = $state({ ...defaultReportForm })
let reportRecords = $state([] as IPurchaseOrder[])
let isReportMode = $state(false)
let isSearchOpen = $state(false)
let filterText = $state('')
let selectedPurchaseOrder = $state<IPurchaseOrder | null>(null)
let isDetailLayerLoading = $state(false)
let detailRows = $state([] as IPurchaseOrderDetailRow[])

// Swap the table data source depending on whether the report layer is active.
const visibleRecords = $derived(isReportMode ? reportRecords : purchaseOrdersService.records)

interface IPurchaseOrderDetailRow {
  detailPosition: number
  productID: number
  productName: string
  presentationName: string
  sku: string
  quantity: number
  unitPrice: number
  subtotalAmount: number
}

const consultarReporte = async () => {
  Loading.standard('Consultando órdenes de compra...')
  try {
    reportRecords = await queryPurchaseOrders(reportForm)
    isReportMode = true
    isSearchOpen = false
  } catch (error) {
    console.error('[purchase-orders-report] query error', error)
  } finally {
    Loading.remove()
  }
}

// Limpiar returns the view to the default cached "pending orders" list.
const limpiarFiltros = () => {
  reportForm = { ...defaultReportForm }
  reportRecords = []
  isReportMode = false
  isSearchOpen = false
}

const providerNameOf = (providerID: number): string =>
  providersService.recordsMap.get(providerID)?.Name || `#${providerID}`

const getOrderLayerTitle = (purchaseOrder: IPurchaseOrder | null): string => {
  if (!purchaseOrder) { return 'Detalle de orden de compra' }
  return `Orden #${purchaseOrder.ID} · ${formatTime(purchaseOrder.Date, 'd-M-Y')}`
}

const resolvePresentationName = (productRecord: IProducto | undefined, presentationID: number): string => {
  if (!productRecord || !presentationID) { return '' }
  return productRecord.Presentaciones?.find((presentation) => presentation.id === presentationID)?.nm || ''
}

const resolveSku = (productRecord: IProducto | undefined, presentationID: number): string => {
  if (!productRecord) { return '' }
  const presentationSku = productRecord.Presentaciones?.find((presentation) => presentation.id === presentationID)?.sk || ''
  return presentationSku || productRecord.SKU || ''
}

const buildDetailRows = (purchaseOrder: IPurchaseOrder): IPurchaseOrderDetailRow[] => {
  const productIDs = purchaseOrder.DetailProductIDs || []
  const quantities = purchaseOrder.DetailQuantities || []
  const prices = purchaseOrder.DetailPrices || []
  const presentationIDs = purchaseOrder.DetailPresentationIDs || []

  return productIDs.map((rawProductID, detailPosition) => {
    const productID = Number(rawProductID || 0)
    const presentationID = Number(presentationIDs[detailPosition] || 0)
    const quantity = Number(quantities[detailPosition] || 0)
    const unitPrice = Number(prices[detailPosition] || 0)
    const productRecord = productosService.recordsMap.get(productID)
    const productName = productRecord?.Nombre || `Producto #${productID}`

    // The report can be opened from partially synced caches, so each row must survive unresolved products.
    return {
      detailPosition,
      productID,
      productName,
      presentationName: resolvePresentationName(productRecord, presentationID),
      sku: resolveSku(productRecord, presentationID),
      quantity,
      unitPrice,
      subtotalAmount: unitPrice * quantity,
    }
  })
}

const openPurchaseOrderDetailLayer = async (purchaseOrder: IPurchaseOrder) => {
  selectedPurchaseOrder = purchaseOrder
  detailRows = buildDetailRows(purchaseOrder)
  isDetailLayerLoading = true
  Core.openSideLayer(21)

  const productIDs = [...new Set((purchaseOrder.DetailProductIDs || []).map((productID) => Number(productID || 0)).filter((productID) => productID > 0))]
  console.debug('[purchase-orders-report] detail layer open', {
    purchaseOrderID: purchaseOrder.ID,
    productIDsCount: productIDs.length,
  })

  try {
    await productosService.syncIDs(productIDs)
    detailRows = buildDetailRows(purchaseOrder)
    console.debug('[purchase-orders-report] detail layer ready', {
      purchaseOrderID: purchaseOrder.ID,
      detailRowsCount: detailRows.length,
    })
  } catch (detailLayerError) {
    console.error('[purchase-orders-report] detail layer sync error', {
      purchaseOrderID: purchaseOrder.ID,
      productIDs,
      detailLayerError,
    })
  } finally {
    isDetailLayerLoading = false
  }
}

onDestroy(() => {
  // The page swaps report/create views by unmounting this component, so its side layer must not survive the tab change.
  if (Core.showSideLayer === 21) {
    Core.hideSideLayer()
  }
})

const reporteColumns: ITableColumn<IPurchaseOrder>[] = [
  {
    header: 'ID',
    width: '70px',
    align: 'right',
    cellCss: 'ff-mono text-right',
    getValue: (r) => r.ID,
  },
  {
    header: 'Fecha Generación',
    width: '130px',
    getValue: (r) => (r.Date ? (formatTime(r.Date, 'd-m-Y') as string) : ''),
  },
  {
    header: 'Fecha Entrega',
    width: '130px',
    getValue: (r) => (r.DateOfDelivery ? (formatTime(r.DateOfDelivery, 'd-m-Y') as string) : ''),
  },
  {
    header: 'Proveedor',
    width: 'minmax(160px, 1.5fr)',
    highlight: true,
    getValue: (r) => providerNameOf(r.ProviderID),
  },
  {
    header: 'Monto Total',
    width: '120px',
    align: 'right',
    cellCss: 'ff-mono text-right',
    getValue: (r) => formatN((r.TotalAmount || 0) / 100, 2),
  },
  {
    header: 'Nota',
    width: 'minmax(180px, 2fr)',
    getValue: (r) => r.Notes || '',
  },
]

const detailColumns: ITableColumn<IPurchaseOrderDetailRow>[] = [
  {
    header: '#',
    width: '44px',
    align: 'right',
    cellCss: 'ff-mono text-right',
    getValue: (row) => row.detailPosition + 1,
  },
  {
    header: 'Producto',
    width: 'minmax(180px, 1.8fr)',
    getValue: (row) => row.productName,
  },
  {
    header: 'Presentación',
    width: 'minmax(120px, 1fr)',
    getValue: (row) => row.presentationName || '-',
  },
  {
    header: 'SKU',
    width: 'minmax(90px, 0.8fr)',
    getValue: (row) => row.sku || '-',
  },
  {
    header: 'Cant.',
    width: '80px',
    align: 'right',
    cellCss: 'ff-mono text-right',
    getValue: (row) => row.quantity,
  },
  {
    header: 'Precio',
    width: '100px',
    align: 'right',
    cellCss: 'ff-mono text-right',
    getValue: (row) => formatN((row.unitPrice || 0) / 100, 2),
  },
  {
    header: 'Subtotal',
    width: '110px',
    align: 'right',
    cellCss: 'ff-mono text-right',
    getValue: (row) => formatN((row.subtotalAmount || 0) / 100, 2),
  },
]
</script>

<div class="flex flex-col h-full">
  <div class="grid grid-cols-[auto_minmax(0,1fr)] items-start mb-12 gap-12 md:flex md:items-center md:justify-between">
    <div class="contents md:flex md:items-center md:w-full md:gap-12">
      <ButtonLayer buttonClass="bx-purple" bind:isOpen={isSearchOpen}
        horizontalOffset={0} useOutline={true}
        edgeMargin={0} buttonClassOnShow="bx-red"
        layerClass="w-600"
        icon="icon-search" iconOnShow="icon-cancel"
      >
        <div class="w-full grid grid-cols-24 gap-12 p-12">
          <DateInput
            label="Fecha Inicio"
            css="col-span-12"
            save="fechaInicio"
            bind:saveOn={reportForm}
          />
          <DateInput
            label="Fecha Fin"
            css="col-span-12"
            save="fechaFin"
            bind:saveOn={reportForm}
          />
          <SearchSelect
            bind:saveOn={reportForm}
            save="providerID"
            css="col-span-12"
            label="Proveedor"
            keyId="ID"
            keyName="Name"
            options={providersService.records}
            placeholder=""
          />
          <SearchSelect
            bind:saveOn={reportForm}
            save="productID"
            css="col-span-12"
            label="Producto"
            optionsCss="w-480"
            keyId="ID"
            keyName="Nombre"
            options={productosService.records}
            placeholder=""
          />
          <SearchSelect
            bind:saveOn={reportForm}
            save="status"
            css="col-span-12"
            label="Estado"
            keyId="ID"
            keyName="Nombre"
            options={purchaseOrderStatusOptions}
            placeholder=""
          />
          <div class="col-span-12 flex items-center justify-center gap-8">
            <button class="px-16 py-8 bx-purple mt-8 h-44"
              aria-label="Consultar reporte"
              onclick={(ev) => { ev.stopPropagation(); consultarReporte() }}
            >
              Buscar <i class="icon-search"></i>
            </button>
            <button class="px-16 py-8 bx-gray mt-8 h-44"
              aria-label="Limpiar filtros"
              onclick={(ev) => { ev.stopPropagation(); limpiarFiltros() }}
            >
              Limpiar <i class="icon-cancel"></i>
            </button>
          </div>
        </div>
      </ButtonLayer>
      {#if isReportMode}
        <KeyValueStrip
          css="col-span-2 row-start-2 w-full md:w-auto"
          label1="Fec. Inicio"
          value1={reportForm.fechaInicio}
          getContent1={(v) => formatTime(v, 'd-m-Y') as string}
          label2="Fec. Fin"
          value2={reportForm.fechaFin}
          getContent2={(v) => formatTime(v, 'd-m-Y') as string}
          label3="Proveedor"
          value3={reportForm.providerID}
          getContent3={(providerID) =>
            providerID
              ? providersService.recordsMap.get(Number(providerID))?.Name || `Proveedor #${providerID}`
              : 'Todos'}
          label4="Producto"
          value4={reportForm.productID}
          getContent4={(productID) =>
            productID
              ? productosService.recordsMap.get(Number(productID))?.Nombre || `Producto #${productID}`
              : 'Todos'}
          label5="Estado"
          value5={reportForm.status}
          getContent5={(statusID) =>
            purchaseOrderStatusOptions.find((s) => s.ID === Number(statusID))?.Nombre || 'Todos'}
        />
      {/if}
    </div>
    <div class="relative col-start-2 row-start-1 flex items-start self-start w-full max-w-224 ml-auto md:mr-16 md:w-224">
      <div class="absolute left-12 text-gray-400">
        <i class="icon-search"></i>
      </div>
      <input
        class="w-full pl-36 bg-white pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autocomplete="off"
        type="text"
        placeholder="Buscar proveedor..."
        onkeyup={(ev) => {
          ev.stopPropagation()
          throttle(() => {
            filterText = ((ev.target as HTMLInputElement).value || '').toLowerCase().trim()
          }, 150)
        }}
      />
    </div>
  </div>

  <VTable
    columns={reporteColumns}
    data={visibleRecords}
    estimateSize={38}
    maxHeight="calc(100vh - var(--header-height) - 72px)"
    selected={selectedPurchaseOrder?.ID || 0}
    isSelected={(purchaseOrder, selectedID) => purchaseOrder.ID === selectedID}
    onRowClick={(purchaseOrder) => {
      void openPurchaseOrderDetailLayer(purchaseOrder)
    }}
    filterText={filterText}
    getFilterContent={(r) => providerNameOf(r.ProviderID).toLowerCase()}
    useFilterCache={true}
    emptyMessage={isReportMode ? 'Sin resultados para los filtros seleccionados' : 'Sin órdenes de compra pendientes'}
  />

  <Layer
    id={21}
    type="side"
    sideLayerSize={760}
    title={getOrderLayerTitle(selectedPurchaseOrder)}
    titleCss="h2"
    css="px-8 py-8 md:px-16 md:py-10"
    contentCss="px-0"
    onClose={() => {
      selectedPurchaseOrder = null
      detailRows = []
      isDetailLayerLoading = false
    }}
  >
    {#if selectedPurchaseOrder}
      <div class="flex flex-col gap-10 mt-8">
        <div class="grid grid-cols-24 gap-8 text-13 md:text-14">
          <div class="col-span-8">
            <div class="text-gray-500">Proveedor</div>
            <div>{providerNameOf(selectedPurchaseOrder.ProviderID)}</div>
          </div>
          <div class="col-span-8">
            <div class="text-gray-500">Estado</div>
            <div>{purchaseOrderStatusOptions.find((status) => status.ID === (selectedPurchaseOrder?.ss || 0))?.Nombre || 'Desconocido'}</div>
          </div>
          <div class="col-span-8">
            <div class="text-gray-500">Total</div>
            <div class="ff-mono">{formatN((selectedPurchaseOrder.TotalAmount || 0) / 100, 2)}</div>
          </div>
        </div>

        <div class="text-13 text-gray-600">
          {selectedPurchaseOrder.Notes || 'Sin notas.'}
        </div>

        {#if isDetailLayerLoading}
          <div class="min-h-120 rounded-md bg-gray-50 fx-c text-gray-500">
            Sincronizando productos del detalle...
          </div>
        {/if}

        <VTable
          columns={detailColumns}
          data={detailRows}
          estimateSize={38}
          maxHeight="calc(100vh - var(--header-height) - 190px)"
          emptyMessage="Esta orden no tiene productos en el detalle."
        />
      </div>
    {/if}
  </Layer>
</div>
