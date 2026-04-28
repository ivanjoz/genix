<script lang="ts">
import KeyValueStrip from '$components/micro/KeyValueStrip.svelte'
import LayerStatic from '$components/LayerStatic.svelte'
import OptionsStrip from '$components/OptionsStrip.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { formatN, formatTime, Notify } from '$libs/helpers'
import { POST } from '$libs/http.svelte'
import { ProductSupplyService } from '$routes/logistica/products-stock/supply-management.svelte'
import { ProductStockSimpleService } from '$routes/logistica/products-stock/stock-movement'
import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import type { IProducto } from '$routes/negocio/productos/productos.svelte'
import { ProductosService } from '$routes/negocio/productos/productos.svelte'
import { AlmacenesService } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'
import { untrack } from 'svelte'
import ProductCardSearch, { type IProductCard } from './ProductCardSearch.svelte'
import PurchaseOrderForm from './PurchaseOrderForm.svelte'

// Line item shown in the cart; keyed by productID+presentationID composite.
interface PurchaseOrderItem {
  key: string
  productID: number
  presentationID: number
  presentationName: string
  productName: string
  displayName: string
  sku: string
  cantidad: number
  precio: number
  producto: IProducto
}

// Payload shape sent to POST /purchase-orders.
interface IPurchaseOrderForm {
  ID: number
  ProviderID: number
  WarehouseID: number
  TotalAmount: number
  TaxAmount: number
  DetailProductIDs: number[]
  DetailPrices: number[]
  DetailQuantities: number[]
  DetailPresentationIDs: number[]
  Notes: string
  // Dates stored as UnixDay int16 (days since unix-epoch).
  DeliveryDate: number
  PaymentDate: number
  InvoiceNumber: string
}

// Holds the cart state and submits the order; amounts stored as cents (int).
class PurchaseOrderState {
  form = $state({
    ID: 0,
    ProviderID: 0,
    WarehouseID: 0,
    TotalAmount: 0,
    TaxAmount: 0,
    DetailProductIDs: [],
    DetailPrices: [],
    DetailQuantities: [],
    DetailPresentationIDs: [],
    Notes: '',
    DeliveryDate: 0,
    PaymentDate: 0,
    InvoiceNumber: '',
  } as IPurchaseOrderForm)

  items = $state([] as PurchaseOrderItem[])
  errorMessage = $state('')

  // Exposed to ProductCardSearch so cards can show current cart quantity.
  itemsCantMap = $derived.by(() => new Map(this.items.map((item) => [item.key, item.cantidad])))

  addItem(card: IProductCard, cant: number = 1) {
    if (cant <= 0) { return }
    const existing = this.items.find((item) => item.key === card.key)
    if (existing) {
      existing.cantidad += cant
      this.items = [...this.items]
    } else {
      this.items.push({
        key: card.key,
        productID: card.productID,
        presentationID: card.presentationID,
        presentationName: card.presentationName,
        productName: card.productName,
        displayName: card.displayName,
        sku: card.sku,
        cantidad: cant,
        precio: card.price || 0,
        producto: card.producto,
      })
    }
    this.recalcTotals()
    this.errorMessage = ''
  }

  removeItem(key: string) {
    this.items = this.items.filter((item) => item.key !== key)
    this.recalcTotals()
  }

  updateQuantity(key: string, cantidad: number) {
    const item = this.items.find((i) => i.key === key)
    if (!item) { return }
    // Treat zero/negative as removal so the row disappears from the cart.
    if (cantidad <= 0) {
      this.removeItem(key)
      return
    }
    item.cantidad = cantidad
    this.items = [...this.items]
    this.recalcTotals()
  }

  updatePrice(key: string, precio: number) {
    const item = this.items.find((i) => i.key === key)
    if (!item) { return }
    item.precio = Math.max(0, precio)
    this.items = [...this.items]
    this.recalcTotals()
  }

  // Recomputes totals after any item mutation; IGV is 18% inclusive.
  recalcTotals() {
    let total = 0
    for (const item of this.items) {
      total += (item.precio || 0) * (item.cantidad || 0)
    }
    this.form.TotalAmount = total
    const subtotal = Math.floor(total / 1.18)
    this.form.TaxAmount = total - subtotal
  }

  reset() {
    this.items = []
    this.form.ProviderID = 0
    this.form.Notes = ''
    this.form.DeliveryDate = 0
    this.form.PaymentDate = 0
    this.form.InvoiceNumber = ''
    this.recalcTotals()
  }

  async postPurchaseOrder(): Promise<boolean> {
    if (this.items.length === 0) {
      Notify.failure('Agregue al menos un producto a la orden.')
      return false
    }
    if (!this.form.ProviderID) {
      Notify.failure('Seleccione un proveedor.')
      return false
    }
    const unpricedItem = this.items.find((item) => !item.precio)
    if (unpricedItem) {
      Notify.failure(`El producto "${unpricedItem.displayName}" no tiene precio.`)
      return false
    }

    // Flatten cart items into parallel arrays matching the backend schema.
    this.form.DetailProductIDs = this.items.map((item) => item.productID)
    this.form.DetailPrices = this.items.map((item) => item.precio)
    this.form.DetailQuantities = this.items.map((item) => item.cantidad)
    this.form.DetailPresentationIDs = this.items.map((item) => item.presentationID)

    const result = await POST({
      data: $state.snapshot(this.form), route: 'purchase-orders',
      refreshRoutes: ["purchase-orders"]
    })
    // Return the saved order ID so the caller can show a confirmation notification.
    return result?.ID || 0
  }
}

const productosService = new ProductosService(true)
const providersService = new ClientProviderService(ClientProviderType.PROVIDER, true)
const productSupplyService = new ProductSupplyService(true)
const productStockService = new ProductStockSimpleService(true)
const almacenesService = new AlmacenesService()

const orderState = new PurchaseOrderState()

const formatMo = (n: number) => formatN(n / 100, 2)

// Aggregates stock across locations for each product, used by product cards.
const stockByProductID = $derived.by(() => {
  const stockMap = new Map<number, number>()
  for (const stock of productStockService.records) {
    stockMap.set(stock.ProductID, (stockMap.get(stock.ProductID) || 0) + Math.max(0, stock.Quantity || 0))
  }
  return stockMap
})

// Auto-select first warehouse once loaded; untrack prevents feedback loops.
$effect(() => {
  almacenesService.Almacenes
  if (!almacenesService.Almacenes.length || orderState.form.WarehouseID > 0) { return }
  untrack(() => {
    orderState.form.WarehouseID = almacenesService.Almacenes[0].ID
  })
})

const cartColumns: ITableColumn<PurchaseOrderItem>[] = [
  {
    id: 'product',
    header: 'Producto',
    width: 'minmax(160px, 1.5fr)',
    css: 'py-4 leading-[1.15]',
    getValue: (item) => item.displayName,
    render: (item) => {
      const sku = item.sku ? `<span class="text-[11px] font-mono text-gray-400 ml-4">${item.sku}</span>` : ''
      if (!item.presentationName) { return `${item.productName}${sku}` }
      return `${item.productName} <span class="text-blue-600 font-bold">(${item.presentationName})</span>${sku}`
    },
  },
  {
    id: 'cantidad',
    header: 'Cant.',
    width: '80px',
    align: 'right',
    cellInputType: 'number',
    getValue: (item) => item.cantidad,
    onCellEdit: (item, value) => {
      const next = parseInt(String(value || '0'))
      orderState.updateQuantity(item.key, isNaN(next) ? 0 : next)
    },
  },
  {
    id: 'precio',
    header: 'Precio',
    width: '100px',
    align: 'right',
    cellInputType: 'number',
    getValue: (item) => (item.precio || 0) / 100,
    render: (item) => formatN((item.precio || 0) / 100, 2),
    onCellEdit: (item, value) => {
      const parsed = parseFloat(String(value || '0'))
      orderState.updatePrice(item.key, Math.round((isNaN(parsed) ? 0 : parsed) * 100))
    },
  },
  {
    id: 'subtotal',
    header: 'Subtotal',
    width: '100px',
    align: 'right',
    css: 'font-mono',
    getValue: (item) => formatN(((item.precio || 0) * (item.cantidad || 0)) / 100, 2),
  },
  {
    id: 'actions',
    header: '',
    width: '36px',
    css: 'text-center px-4',
    buttonDeleteHandler: (item) => orderState.removeItem(item.key),
  },
]

async function handleSave() {
  const orderID = await orderState.postPurchaseOrder()
  if (orderID) {
    Notify.success(`La orden Nº ${orderID} ha sido generada`)
    orderState.reset()
  }
}

// Two-view detail layer; "info" holds the form fields, "products" the cart table.
type DetailView = 1 | 2
let detailView = $state<DetailView>(1)
const detailViewOptions: [DetailView, string][] = [
  [1, 'Información'],
  [2, 'Productos'],
]

// Lookups used by KeyValueStrip to render IDs as human-readable names.
const providerNameByID = $derived(
  new Map(providersService.records.map((p) => [p.ID, p.Name])),
)
const almacenNameByID = $derived(
  new Map(almacenesService.Almacenes.map((a) => [a.ID, a.Nombre])),
)
const formatDateOrDash = (value: string | number) => {
  const day = Number(value)
  return day ? (formatTime(day, 'd-m-Y') as string) : '—'
}
</script>

<div class="flex h-full gap-20">
  <div class="flex-1 flex flex-col min-w-0 relative">
    <ProductCardSearch
      productosService={productosService}
      providersService={providersService}
      productSupplyService={productSupplyService}
      stockMap={stockByProductID}
      displayProviderFilter
      selectedKeys={orderState.itemsCantMap}
      onSelect={(card, cant) => {
      	orderState.addItem(card, cant || 1)
       	if(detailView !== 2){ detailView = 2 }
      }}
      onRemove={(card) => orderState.removeItem(card.key)}
    />
  </div>

  <LayerStatic
    css="w-[50%] min-w-350 bg-white border-l border-gray-200 flex flex-col h-[calc(100vh-var(--header-height))] shadow-lg md:-m-10"
    mobileLayerTitle="Detalle de Compra"
    useMobileLayerVertical={124}
  >
    {#if orderState.errorMessage}
      <div class="bg-red-50 m-8 text-red-600 p-12 text-sm font-medium border-b border-red-100">
        {orderState.errorMessage}
      </div>
    {/if}

    <div class="px-10 py-8 border-b border-gray-100 flex items-center justify-between bg-gray-50/50">
      <div class="grow mr-16">
        <div class="hidden font-bold mb-2 -mt-2 text-gray-800 mb-4 items-center justify-between md:flex">
          <span>Nueva Órden de Compra</span>
        </div>
        <div class="flex items-center gap-6">
          <div class="bg-gray-100 flex flex-1 p-6 rounded-md items-center gap-6 max-w-200">
            <div class="text-[10px] leading-[1] text-gray-500 font-bold tracking-wider uppercase">
              <div>Sub</div>
              <div>Total</div>
            </div>
            <div class="leading-[1] text-gray-800 text-[16px] ml-auto">
              {formatMo(orderState.form.TotalAmount - orderState.form.TaxAmount)}
            </div>
          </div>

          <div class="bg-blue-50 items-center flex flex-1 p-6 rounded-md gap-6 max-w-200">
            <div class="text-[10px] text-blue-600 uppercase font-bold tracking-wider">Total</div>
            <div class="leading-[1] text-blue-700 font-bold text-[22px] ml-auto">
              {formatMo(orderState.form.TotalAmount)}
            </div>
          </div>
        </div>
      </div>
      <button class="bx-blue shrink-0"
        onclick={handleSave}
        title="Generar orden de compra"
      >
        <span class="hidden md:inline">Generar</span>
        <i class="icon-floppy"></i>
      </button>
    </div>

    <div class="px-12 pt-4 mb-4">
      <OptionsStrip
        options={detailViewOptions}
        selected={detailView}
        onSelect={(opt) => { detailView = opt[0] }}
      />
    </div>

    {#if detailView === 1}
      <!-- Información block: order header form (provider, warehouse, dates, invoice, notes). -->
      <div class="px-12 py-8">
        <PurchaseOrderForm
          bind:form={orderState.form}
          providers={providersService.records}
          almacenes={almacenesService.Almacenes}
        />
      </div>
    {:else}
      <!-- Productos block: read-only summary of the form above the cart table. -->
      <div class="px-12 py-8">
        <KeyValueStrip
          css="gap-x-12 gap-y-4"
          label1="Proveedor"
          value1={orderState.form.ProviderID}
          getContent1={(id) => providerNameByID.get(Number(id)) || '—'}
          label2="Almacén"
          value2={orderState.form.WarehouseID}
          getContent2={(id) => almacenNameByID.get(Number(id)) || '—'}
          label3="Fec. Entrega"
          value3={orderState.form.DeliveryDate}
          getContent3={formatDateOrDash}
          label4="Fec. Pago"
          value4={orderState.form.PaymentDate}
          getContent4={formatDateOrDash}
          label5="Factura"
          value5={orderState.form.InvoiceNumber}
          getContent5={(v) => String(v) || '—'}
          label6="Notas"
          value6={orderState.form.Notes}
          getContent6={(v) => String(v) || '—'}
        />
      </div>

      <div class="flex-1 min-h-0 px-12 pb-8 mt-4">
        <VTable
          columns={cartColumns}
          data={orderState.items}
          estimateSize={42}
          maxHeight="calc(100vh - var(--header-height) - 320px)"
          emptyMessage="Sin productos"
        />
        {#if orderState.items.length === 0}
          <div class="flex flex-col items-center justify-center h-192 text-gray-300 gap-8">
            <i class="icon-basket text-4xl"></i>
            <span class="text-sm">Órden de Compra Vacía</span>
          </div>
        {/if}
      </div>
    {/if}
  </LayerStatic>
</div>
