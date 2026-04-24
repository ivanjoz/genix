<script lang="ts">
import Input from '$components/Input.svelte'
import LayerStatic from '$components/LayerStatic.svelte'
import SearchSelect from '$components/SearchSelect.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { formatN, Notify } from '$libs/helpers'
import { POST } from '$libs/http.svelte'
import { ProductSupplyService } from '$routes/logistica/products-stock/supply-management.svelte'
import { ProductStockSimpleService } from '$routes/logistica/products-stock/stock-movement'
import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import type { IProducto } from '$routes/negocio/productos/productos.svelte'
import { ProductosService } from '$routes/negocio/productos/productos.svelte'
import { AlmacenesService, type IAlmacen } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'
import { untrack } from 'svelte'
import ProductCardSearch, { type IProductCard } from './ProductCardSearch.svelte'

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

let almacenSelected = $state(0)

// Auto-select first warehouse once loaded; untrack prevents feedback loops.
$effect(() => {
  almacenesService.Almacenes
  if (!almacenesService.Almacenes.length || almacenSelected > 0) { return }
  untrack(() => {
    almacenSelected = almacenesService.Almacenes[0].ID
    orderState.form.WarehouseID = almacenSelected
  })
})

const cartColumns: ITableColumn<PurchaseOrderItem>[] = [
  {
    id: 'product',
    header: 'Producto',
    width: 'minmax(160px, 1.5fr)',
    cellCss: 'px-8 py-4 leading-[1.15]',
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
    css: "justify-end",
    inputCss: 'text-right',
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
    css: "justify-end",
    cellCss: 'px-6',
    inputCss: 'text-right',
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
    cellCss: 'font-mono px-6 text-right',
    getValue: (item) => formatN(((item.precio || 0) * (item.cantidad || 0)) / 100, 2),
  },
  {
    id: 'actions',
    header: '',
    width: '36px',
    cellCss: 'text-center px-4',
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
      onSelect={(card, cant) => orderState.addItem(card, cant || 1)}
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

    <div class="grid grid-cols-2 gap-8 px-12 py-8">
      <SearchSelect
        label=""
        keyId="ID"
        keyName="Name"
        options={providersService.records}
        selected={orderState.form.ProviderID}
        placeholder="PROVEEDOR"
        onChange={(provider) => { orderState.form.ProviderID = provider?.ID || 0 }}
      />
      <SearchSelect
        label=""
        keyId="ID"
        keyName="Nombre"
        options={almacenesService.Almacenes}
        selected={almacenSelected}
        placeholder="ALMACÉN"
        onChange={(almacen: IAlmacen) => {
          if (!almacen) { return }
          almacenSelected = almacen.ID
          orderState.form.WarehouseID = almacen.ID
        }}
      />
    </div>

    <div class="px-12 pb-6">
      <Input
        label=""
        saveOn={orderState.form}
        save="Notes"
        placeholder="Notas de la orden..."
      />
    </div>

    <div class="flex-1 min-h-0 px-12 pb-8 mt-4">
      {#if orderState.items.length === 0}
        <div class="flex flex-col items-center justify-center h-192 text-gray-300 gap-8">
          <i class="icon-basket text-4xl"></i>
          <span class="text-sm">Carrito vacío</span>
        </div>
      {:else}
        <VTable
          columns={cartColumns}
          data={orderState.items}
          estimateSize={42}
          maxHeight="calc(100vh - var(--header-height) - 280px)"
          emptyMessage="Sin productos"
        />
      {/if}
    </div>
  </LayerStatic>
</div>
