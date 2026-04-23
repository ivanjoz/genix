<script lang="ts">
import Input from '$components/Input.svelte'
import LayerStatic from '$components/LayerStatic.svelte'
import SearchSelect from '$components/SearchSelect.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import Page from '$domain/Page.svelte'
import { formatN } from '$libs/helpers'
import { ProductSupplyService } from '$routes/logistica/products-stock/supply-management.svelte'
import { ProductStockSimpleService } from '$routes/logistica/products-stock/stock-movement'
import { ClientProviderService, ClientProviderType } from '$routes/negocio/clientes/clientes-proveedores.svelte'
import { ProductosService } from '$routes/negocio/productos/productos.svelte'
import { AlmacenesService, type IAlmacen } from '$routes/negocio/sedes-almacenes/sedes-almacenes.svelte'
import { untrack } from 'svelte'
import ProductCardSearch from './ProductCardSearch.svelte'
import { PurchaseOrderState, type PurchaseOrderItem } from './purchase_order.svelte'

  const formatMo = (n: number) => formatN(n / 100, 2)

  const productosService = new ProductosService(true)
  const providersService = new ClientProviderService(ClientProviderType.PROVIDER, true)
  const productSupplyService = new ProductSupplyService(true)
  const productStockService = new ProductStockSimpleService(true)
  const almacenesService = new AlmacenesService()

  const orderState = new PurchaseOrderState()

  const stockByProductID = $derived.by(() => {
    const stockMap = new Map<number, number>()
    for (const stock of productStockService.records) {
      stockMap.set(stock.ProductID, (stockMap.get(stock.ProductID) || 0) + Math.max(0, stock.Quantity || 0))
    }
    return stockMap
  })

  let almacenSelected = $state(0)

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
      cellCss: 'text-right',
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
      cellCss: 'text-right',
      inputCss: 'text-right',
      getValue: (item) => formatN((item.precio || 0) / 100, 2),
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
      cellCss: 'text-right font-mono',
      getValue: (item) => formatN(((item.precio || 0) * (item.cantidad || 0)) / 100, 2),
    },
    {
      id: 'actions',
      header: '',
      width: '36px',
      cellCss: 'text-center',
      buttonDeleteHandler: (item) => orderState.removeItem(item.key),
    },
  ]

  async function handleSave() {
    const ok = await orderState.postPurchaseOrder()
    if (ok) {
      orderState.reset()
    }
  }
</script>

<Page title="Órdenes de Compra">
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
            <span>Detalle de Compra</span>
          </div>
          <div class="flex items-center gap-6">
            <div class="bg-gray-50 flex flex-1 p-6 rounded-md items-center gap-6 border border-gray-100 shadow-sm">
              <div class="text-[10px] leading-[1] text-gray-500 font-bold tracking-wider uppercase">
                <div>Sub</div>
                <div>Total</div>
              </div>
              <div class="leading-[1] text-gray-800 text-[16px] ml-auto">
                {formatMo(orderState.form.TotalAmount - orderState.form.TaxAmount)}
              </div>
            </div>

            <div class="bg-blue-50 items-center flex flex-1 p-6 rounded-md gap-6 border border-blue-100 shadow-sm">
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

      <div class="flex-1 min-h-0 px-8 pb-8">
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
</Page>
