<script lang="ts">
import ButtonLayer from '$components/buttons/ButtonLayer.svelte'
import DateInput from '$components/form/DateInput.svelte'
import Input from '$components/form/Input.svelte'
import Layer from '$components/layers/Layer.svelte'
import Modal from '$components/layers/Modal.svelte'
import SearchSelect from '$components/form/SearchSelect.svelte'
import FilterInput from '$components/form/FilterInput.svelte'
import KeyValueStrip from '$components/misc/KeyValueStrip.svelte'
import LabelText from '$components/form/LabelText.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { ConfirmWarn, formatN, formatTime, Loading, Notify } from '$libs/helpers'
import { saveRouteRecord, setRouteRecordQueryParam } from '$libs/cache/route-data'
import { CajasService } from '$routes/finanzas/cajas/cajas.svelte'
import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import { ProductosService, type IProducto } from '$routes/negocio/productos/productos.svelte'
import { AlmacenesService } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'
import { onDestroy } from 'svelte'
import PurchaseOrderForm from './PurchaseOrderForm.svelte'
import {
  PurchaseOrderAction,
  PurchaseOrderEditableStatuses,
  PurchaseOrdersService,
  PurchaseOrderStatus,
  purchaseOrderStatusOptions,
  queryPurchaseOrders,
  updatePurchaseOrder,
  type IPurchaseOrder,
} from './purchase_order.svelte'

const providersService = new ClientProviderService(ClientProviderType.PROVIDER, true)
const productosService = new ProductosService(true)
const purchaseOrdersService = new PurchaseOrdersService(PurchaseOrderStatus.PENDING, true)
// Warehouses are loaded lazily because the report only needs them when the edit modal is opened.
const almacenesService = new AlmacenesService()
// Cajas are needed only when the user opens the Pagar modal; the service auto-fetches on construction.
const cajasService = new CajasService()
const EDIT_PURCHASE_ORDER_MODAL_ID = 31
const PAY_PURCHASE_ORDER_MODAL_ID = 32

// Patch sent to PUT /purchase-orders?action=2; ProviderID is read-only in edit mode but kept for the form display.
let editForm = $state<Partial<IPurchaseOrder>>({})
// Payment form for PUT /purchase-orders?action=3. Monto is stored in cents (Input baseDecimals=2 handles the display).
let payForm = $state({ CajaID: 0, Monto: 0 })
// Live preview of the debt that will remain after the current Monto is applied; recalculates as the user types.
const remainingDebtAfterPayment = $derived(
  Math.max((selectedPurchaseOrder?.DebtAmount || 0) - (payForm.Monto || 0), 0),
)

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
let rowRerender: (() => void) | undefined = undefined

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
  const quantities = purchaseOrder.DetailProductQuantity || []
  const prices = purchaseOrder.DetailProductPrice || []
  const presentationIDs = purchaseOrder.DetailProductPresentationIDs || []

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

// Opens the edit modal preloaded with the selected order's editable fields.
// Editing is restricted to Pending/Confirmed orders (mirrors the backend allow-list).
const openEditPurchaseOrderModal = () => {
  if (!selectedPurchaseOrder) { return }
  if (!PurchaseOrderEditableStatuses.includes(selectedPurchaseOrder.ss || 0)) {
    Notify.failure('Solo se pueden editar órdenes en estado Pendiente o Confirmada.')
    return
  }
  
  editForm = {...selectedPurchaseOrder}
  Core.openModal(EDIT_PURCHASE_ORDER_MODAL_ID)
}

// Persists the edited fields via PUT action=2; backend re-validates state and ignores protected fields.
const saveEditPurchaseOrder = async () => {
  if ((editForm.ID||0) <= 0) { return }
  Loading.standard('Actualizando orden de compra...')
  const selectedRecord = visibleRecords.find(x => x.ID === selectedPurchaseOrder?.ID)
  if(!selectedRecord){
 		Notify.failure("No se encontró el registro seleccionado. (¿?)")
  }
  
  try {
    // Send only the fields the backend accepts to keep payloads minimal and intent explicit.
    await updatePurchaseOrder(editForm.ID||0, PurchaseOrderAction.EDIT, editForm)
    Object.assign(selectedRecord as IPurchaseOrder, editForm)
    
    Notify.success(`La orden Nº ${editForm.ID} fue actualizada.`)
    Core.closeModal(EDIT_PURCHASE_ORDER_MODAL_ID)
    rowRerender?.()
  } catch (error) {
    console.error('[purchase-orders-report] edit error', error)
  } finally {
    Loading.remove()
  }
}

// Opens the payment modal pre-filled with the remaining DebtAmount and the first available caja.
// Backend only accepts payments while the order is Confirmada; mirror that constraint here for fast feedback.
const openPayPurchaseOrderModal = () => {
  if (!selectedPurchaseOrder) { return }
  if (selectedPurchaseOrder.ss !== PurchaseOrderStatus.CONFIRMED) {
    Notify.failure('Solo se pueden pagar órdenes en estado Confirmada.')
    return
  }
  const remainingDebt = selectedPurchaseOrder.DebtAmount || 0
  if (remainingDebt <= 0) {
    Notify.failure('La orden no tiene deuda pendiente.')
    return
  }

  // Monto starts at 0 so the user explicitly types the amount; "Deuda pendiente" reflects the order's full debt.
  payForm = {
    CajaID: cajasService.Cajas[0]?.ID || 0,
    Monto: 0,
  }
  Core.openModal(PAY_PURCHASE_ORDER_MODAL_ID)
}

// Submits the payment: backend creates the caja movimiento (Tipo=6) and decrements DebtAmount atomically.
const submitPurchaseOrderPayment = async () => {
  if (!selectedPurchaseOrder) { return }
  if (payForm.CajaID <= 0) {
    Notify.failure('Seleccione una caja.')
    return
  }
  const remainingDebt = selectedPurchaseOrder.DebtAmount || 0
  if (payForm.Monto <= 0) {
    Notify.failure('Ingrese un monto mayor a 0.')
    return
  }
  if (payForm.Monto > remainingDebt) {
    Notify.failure('El monto excede la deuda pendiente.')
    return
  }

  const orderID = selectedPurchaseOrder.ID
  Loading.standard('Registrando pago...')
  try {
    const updated = await updatePurchaseOrder(orderID, PurchaseOrderAction.PAY, {
      CajaID: payForm.CajaID,
      Monto: payForm.Monto,
    })
    // Sync the in-memory record so the layer/table reflect the new debt without a round-trip.
    const newDebt = updated?.DebtAmount ?? (remainingDebt - payForm.Monto)
    selectedPurchaseOrder.DebtAmount = newDebt
    const selectedRecord = visibleRecords.find((r) => r.ID === orderID)
    if (selectedRecord) { selectedRecord.DebtAmount = newDebt }

    rowRerender?.()
    Notify.success(`Pago de ${formatN(payForm.Monto / 100, 2)} registrado en la orden Nº ${orderID}.`)
    Core.closeModal(PAY_PURCHASE_ORDER_MODAL_ID)
  } catch (error) {
    console.error('[purchase-orders-report] pay error', { orderID, error })
  } finally {
    Loading.remove()
  }
}

// Stashes the selected order in the routeData KV store and navigates to the create tab
// with `?rec=oc_<ID>`; PurchaseOrderCreate reads the param on mount and populates a copy.
// Order matters: IDB write -> URL update -> tab switch. Switching the tab last guarantees
// PurchaseOrderCreate's onMount sees the `?rec=` param already on the URL.
const generarCopiaPurchaseOrder = async () => {
  if (!selectedPurchaseOrder) {
    console.warn('[generar-copia] no purchase order selected')
    return
  }
  const orderID = selectedPurchaseOrder.ID
  const routeDataKey = `oc_${orderID}`
  console.debug('[generar-copia] start', { orderID, routeDataKey })
  try {
    const snapshot = $state.snapshot(selectedPurchaseOrder)
    await saveRouteRecord(routeDataKey, snapshot)
    console.debug('[generar-copia] saved to IDB', {
      routeDataKey,
      detailCount: snapshot.DetailProductIDs?.length || 0,
    })

    Core.hideSideLayer()
    selectedPurchaseOrder = null
    detailRows = []

    // Push the URL param BEFORE flipping the tab so the create view's onMount sees `?rec=` on first read.
    await setRouteRecordQueryParam('rec', routeDataKey)
    console.debug('[generar-copia] URL updated', { href: window.location.href })

    Core.pageOptionSelected = 1
    console.debug('[generar-copia] switched to "Órdenes" tab')
  } catch (error) {
    console.error('[generar-copia] error', { orderID, error })
    Notify.failure('No se pudo generar la copia de la orden.')
  }
}

// Annuls the selected order (Pendiente/Confirmada -> Cancelada). Asks for confirmation first
// because the operation flips the order to status=0 (CANCELED) and is not reversible from the UI.
const annulSelectedPurchaseOrder = () => {
  if (!selectedPurchaseOrder) { return }
  if (!PurchaseOrderEditableStatuses.includes(selectedPurchaseOrder.ss || 0)) {
    Notify.failure('Solo se pueden anular órdenes en estado Pendiente o Confirmada.')
    return
  }
  const orderID = selectedPurchaseOrder.ID
  ConfirmWarn(
    'Anular Órden de Compra',
    `¿Desea anular la órden de compra Nº ${orderID}?`,
    'SI',
    'NO',
    async () => {
      Loading.standard('Anulando orden de compra...')
      try {
        await updatePurchaseOrder(orderID, PurchaseOrderAction.ANNUL)
        const selectedRecord = visibleRecords.find((r) => r.ID === orderID)
        if (selectedRecord) { selectedRecord.ss = PurchaseOrderStatus.CANCELED }
        if (selectedPurchaseOrder) { selectedPurchaseOrder.ss = PurchaseOrderStatus.CANCELED }
        rowRerender?.()
        Notify.success(`La orden Nº ${orderID} fue anulada.`)
      } catch (error) {
        console.error('[purchase-orders-report] annul error', { orderID, error })
      } finally {
        Loading.remove()
      }
    },
  )
}

// Confirms the selected order (Pendiente -> Cumplida); backend rejects the call if the state is not Pending.
const confirmSelectedPurchaseOrder = async () => {
  if (!selectedPurchaseOrder) { return }
  if (selectedPurchaseOrder.ss !== PurchaseOrderStatus.PENDING) {
    Notify.failure('Solo se pueden confirmar órdenes en estado Pendiente.')
    return
  }
  const orderID = selectedPurchaseOrder.ID
  Loading.standard('Confirmando orden de compra...')
  try {
    await updatePurchaseOrder(orderID, PurchaseOrderAction.CONFIRM)
    const selectedRecord = visibleRecords.find(x => x.ID === selectedPurchaseOrder?.ID)
    if(selectedRecord){ selectedRecord.ss = PurchaseOrderStatus.CONFIRMED }
    
    rowRerender?.()
    Notify.success(`La orden Nº ${orderID} fue confirmada.`)
  } catch (error) {
    console.error('[purchase-orders-report] confirm error', { orderID, error })
  } finally {
    Loading.remove()
  }
}

const reporteColumns: ITableColumn<IPurchaseOrder>[] = [
  {
    header: 'ID',
    width: '70px',
    align: 'right',
    css: 'ff-mono text-right',
    getValue: (r) => r.ID,
  },
  {
    header: 'Fecha Generación',
    width: '130px',
    getValue: (r) => (r.Date ? (formatTime(r.Date, 'd-m-Y') as string) : ''),
  },
  {
    header: 'Estado',
    width: '110px',
    align: 'center',
    // ss=2 (Completada) → green, ss=1 (Pendiente) → amber, ss=0 (Cancelada) → red.
    getValue: (r) => purchaseOrderStatusOptions.find((s) => s.ID === (r.ss || 0))?.Nombre || '',
    render: (r) => {
      const status = r.ss || 0
      const name = purchaseOrderStatusOptions.find((s) => s.ID === status)?.Nombre || '—'
      const palette =
        status === PurchaseOrderStatus.FULFILLED ? 'bg-green-50 text-green-700 border-green-200' :
        status === PurchaseOrderStatus.PENDING ? 'bg-amber-50 text-amber-700 border-amber-200' :
        'bg-red-50 text-red-600 border-red-200'
      return `<span class="inline-block px-8 py-2 rounded-md border ${palette} text-xs font-medium">${name}</span>`
    },
  },
  {
    header: 'Fecha Entrega',
    width: '130px',
    getValue: (r) => (r.DeliveryDate ? (formatTime(r.DeliveryDate, 'd-m-Y') as string) : ''),
  },
  {
    header: 'Fecha Pago',
    width: '130px',
    getValue: (r) => (r.PaymentDate ? (formatTime(r.PaymentDate, 'd-m-Y') as string) : ''),
  },
  {
    header: 'Proveedor',
    width: 'minmax(160px, 1.5fr)',
    highlight: true,
    getValue: (r) => providerNameOf(r.ProviderID),
  },
  {
    header: 'Factura',
    width: 'minmax(110px, 0.8fr)',
    getValue: (r) => r.InvoiceNumber || '',
  },
  {
    header: 'Monto Total',
    width: '120px',
    align: 'right',
    css: 'ff-mono text-right',
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
    align: 'right',
    getValue: (row) => row.detailPosition + 1,
  },
  {
    id: 'product',
    header: 'Producto', css: "py-4 leading-[1]",
    // The actual cell uses cellRenderer to stack productName + presentation/SKU on a second line.
    getValue: (row) => row.productName,
  },
  {
    header: 'Cant.',
    align: 'right',
    getValue: (row) => row.quantity,
  },
  {
    header: 'Precio',
    align: 'right',
    headerCss: "w-80",
    getValue: (row) => formatN((row.unitPrice || 0) / 100, 2),
  },
  {
    header: 'Subtotal',
    align: 'right',
    headerCss: "w-80",
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
        label="Opens the search filter for purchase orders."
      >
        <div class="w-full grid grid-cols-24 gap-12 p-12" aria-label="Purchase orders search filter with date range, provider, product, and status">
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
    <FilterInput
      css="col-start-2 row-start-1 self-start w-full max-w-224 ml-auto md:mr-16 md:w-224"
      placeholder="Buscar proveedor..."
      throttle={150}
      icon="icon-search"
      bind:value={filterText}
    />
  </div>

  <VTable
    columns={reporteColumns}
    data={visibleRecords}
    estimateSize={38}
    maxHeight="calc(100vh - var(--header-height) - 72px)"
    selected={selectedPurchaseOrder?.ID || 0}
    isSelected={(purchaseOrder, selectedID) => purchaseOrder.ID === selectedID}
    onRowClick={(purchaseOrder,_,rerender) => {
    	rowRerender = rerender
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
    actionsButton={{ name: "Acciones", icon: "icon-menu", css: "" }}
    actions={[
    	{ id: 1,
     		name: "Confirmar", icon: "icon-ok text-green-500",
       	label: "Confirms and approves the selected purchase order.",
       	handler: () => { void confirmSelectedPurchaseOrder() }
     	},
     	{ id: 5,
    		name: "Editar", icon: "icon-pencil text-blue-500",
       	label: "Opens the modal to edit the selected purchase order.",
       	handler: () => { openEditPurchaseOrderModal() }
     	},
     	{ id: 2,
    		name: "Pagar", icon: "icon-tag",
       	label: "Opens the payment form for the selected purchase order.",
       	handler: () => { openPayPurchaseOrderModal() }
     	},
     	{ id: 3,
    		name: "Anular", icon: "icon-cancel",
       	label: "Cancels and annuls the selected purchase order.",
       	handler: () => { annulSelectedPurchaseOrder() }
     	},
     	{ id: 4,
    		name: "Generar Copia", icon: "text-xs icon-plus",
       	label: "Creates a copy of the selected purchase order.",
       	handler: () => { void generarCopiaPurchaseOrder() }
     	},
    ]}
  >
    {#if selectedPurchaseOrder}
      <div class="flex flex-col gap-10 mt-8" aria-label="Purchase order detail with provider info, amounts, notes, and product list">
        <div class="grid grid-cols-24 gap-8 text-13 md:text-14" aria-label="Purchase order summary fields">
          <LabelText
            css="col-span-8"
            label="Proveedor"
            text={providerNameOf(selectedPurchaseOrder.ProviderID)}
          />
          <LabelText
            css="col-span-4"
            label="Estado"
            text={purchaseOrderStatusOptions.find((status) => status.ID === (selectedPurchaseOrder?.ss || 0))?.Nombre || 'Desconocido'}
          />
          <LabelText
            css="col-span-6"
            label="Fecha Entrega"
            text={selectedPurchaseOrder.DeliveryDate ? (formatTime(selectedPurchaseOrder.DeliveryDate, 'd-m-Y') as string) : '—'}
          />
          <LabelText
            css="col-span-6"
            label="Fecha Pago"
            text={selectedPurchaseOrder.PaymentDate ? (formatTime(selectedPurchaseOrder.PaymentDate, 'd-m-Y') as string) : '—'}
          />
          <LabelText
            css="col-span-7"
            label="Factura"
            text={selectedPurchaseOrder.InvoiceNumber || '—'}
          />
          <LabelText
            css="col-span-5"
            label="Total"
            contentCss="ff-mono"
            text={formatN((selectedPurchaseOrder.TotalAmount || 0) / 100, 2)}
          />
          <LabelText
            css="col-span-6"
            label="Pagado"
            contentCss="ff-mono"
            text={formatN(((selectedPurchaseOrder.TotalAmount || 0) - (selectedPurchaseOrder.DebtAmount || 0)) / 100, 2)}
          />
          <LabelText
            css="col-span-6"
            label="Entregado"
            contentCss="ff-mono"
            text={""}
          />
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
        >
          {#snippet cellRenderer(detailRow: IPurchaseOrderDetailRow, col: ITableColumn<IPurchaseOrderDetailRow>)}
            {#if col.id === 'product'}
              <!-- Stack producto in the first line; presentation (blue) + SKU (gray mono) below when present. -->
              <div class="leading-[1.2]">{detailRow.productName}</div>
              {#if detailRow.presentationName || detailRow.sku}
                <!-- items-baseline aligns the differently-sized texts on their text baseline (presentation is regular size, SKU is text-xs). -->
                <div class="flex items-baseline gap-6">
                  {#if detailRow.presentationName}
                    <span class="text-blue-600 ff-bold text-sm">{detailRow.presentationName}</span>
                  {/if}
                  {#if detailRow.sku}
                    <span class="text-gray-400 ff-mono text-xs">{detailRow.sku}</span>
                  {/if}
                </div>
              {/if}
            {/if}
          {/snippet}
        </VTable>
      </div>
    {/if}
  </Layer>

  <!-- Edit modal: only the metadata fields are mutable; Proveedor is locked because changing it
       on an existing order would invalidate the cart pricing assumptions. -->
  <Modal id={EDIT_PURCHASE_ORDER_MODAL_ID}
    size={5}
    bodyCss="px-16 py-12"
    isEdit={true}
    title={`Editar Orden #${editForm.ID}`}
    onSave={() => { void saveEditPurchaseOrder() }}
  >
    <PurchaseOrderForm
      bind:form={editForm}
      providers={providersService.records}
      almacenes={almacenesService.Almacenes}
      disableProvider={true}
    />
  </Modal>

  <!-- Payment modal: registers a Pago Proveedor (movimiento Tipo=6) and decrements DebtAmount.
       Monto is bound in cents (baseDecimals=2 displays the value divided by 100). -->
  <Modal id={PAY_PURCHASE_ORDER_MODAL_ID}
    size={3}
    bodyCss="px-16 py-12"
    isEdit={true}
    saveButtonLabel="Registrar Pago"
    title={`Pagar Orden #${selectedPurchaseOrder?.ID || ''}`}
    onSave={() => { void submitPurchaseOrderPayment() }}
  >
    <div class="grid grid-cols-2 gap-8" aria-label="Purchase order payment form with cash register and amount">
      <SearchSelect
        css="col-span-2"
        label="Caja"
        keyId="ID"
        keyName="Nombre"
        options={cajasService.Cajas}
        selected={payForm.CajaID}
        onChange={(caja) => { payForm.CajaID = caja?.ID || 0 }}
      />
      <Input
        css="col-span-1"
        inputCss="text-center ff-mono"
        label="Monto"
        type="number"
        baseDecimals={2}
        bind:saveOn={payForm}
        save="Monto"
      />
      <!-- Gray box mirrors the visual weight of the Input next to it; recalculates as the user types Monto.
           The remaining amount turns red while it is still > 0 to highlight that the debt is not fully covered. -->
      <div class="col-span-1 flex items-center justify-between gap-8 px-12 py-8 bg-gray-100 border border-gray-200 rounded-md text-13 text-gray-700">
        <span>Deuda pendiente</span>
        <span class={`ff-mono ff-bold ${remainingDebtAfterPayment > 0 ? 'text-red-600' : 'text-gray-700'}`}>
          {formatN(remainingDebtAfterPayment / 100, 2)}
        </span>
      </div>
    </div>
  </Modal>
</div>
