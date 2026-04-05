<script lang="ts">
import Checkbox from '$components/Checkbox.svelte'
import Input from '$components/Input.svelte'
import Layer from '$components/Layer.svelte'
import SearchSelect from '$components/SearchSelect.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { Loading, Notify, throttle } from '$libs/helpers'
import { untrack } from 'svelte'
import { ProductosService } from '../../negocio/productos/productos.svelte'
import { AlmacenesService } from '../../negocio/sedes-almacenes/sedes-almacenes.svelte'
import { getProductosStock, postProductosStock, type IProductoStock } from './stock-movement'

  const almacenes = new AlmacenesService()
  const productos = new ProductosService()

  let stockFilters = $state({ warehouseID: 0, showTodosProductos: false })
  let stockFilterText = $state('')
  let almacenStock = $state([] as IProductoStock[])
  let almacenStockGetted = [] as IProductoStock[]
  let stockForm = $state({} as IProductoStock)

  const stockFormProducto = $derived(productos?.recordsMap?.get(stockForm.ProductID || 0))

  const stockColumns: ITableColumn<IProductoStock>[] = [
    { header: 'Producto', highlight: true,
      getValue: (productStockRecord) => {
        const productRecord = productos.recordsMap.get(productStockRecord.ProductID)?.Nombre
        return productRecord || `Producto-${productStockRecord.ProductID}`
      }
    },
    { header: 'Lote',
      getValue: (productStockRecord) => productStockRecord.Lote || ''
    },
    { header: 'SKU',
      getValue: (productStockRecord) => productStockRecord.SKU || ''
    },
    { header: 'Presentación',
      getValue: (productStockRecord) => {
        if (!productStockRecord.PresentationID) { return '' }
        const productRecord = productos.recordsMap.get(productStockRecord.ProductID)
        const presentationRecord = productRecord?.Presentaciones?.find((presentationOption) => presentationOption.id === productStockRecord.PresentationID)
        return presentationRecord?.nm || `Tipo-${productStockRecord.PresentationID}`
      }
    },
    { header: 'Stock', css: 'justify-end', inputCss: 'text-right pr-6',
      getValue: (productStockRecord) => productStockRecord.Quantity,
      onCellEdit: (productStockRecord, value) => {
        updateStockQuantity(productStockRecord, parseInt(value as string || '0'))
      },
      render: (productStockRecord) => {
        if (productStockRecord._cantidadPrev && productStockRecord._cantidadPrev !== productStockRecord.Quantity) {
          return {
            css: 'flex items-center',
            children: [
              { text: String(productStockRecord._cantidadPrev > 0 ? productStockRecord._cantidadPrev : 0) },
              { text: '→', css: 'ml-2 mr-2' },
              { text: productStockRecord.Quantity, css: 'text-red-500' }
            ]
          }
        }
        return { text: productStockRecord.Quantity || '' }
      }
    },
    { header: 'Costo Un.',
      onCellEdit: (productStockRecord, value) => {
        productStockRecord._hasUpdated = true
        productStockRecord.CostoUn = parseInt(value as string || '0')
      },
    },
  ]

  const onChangeAlmacen = async () => {
    if (!stockFilters.warehouseID) { return }
    Loading.standard()
    try {
      const result = await getProductosStock(stockFilters.warehouseID)
      almacenStock = result || []
      almacenStockGetted = result || []
    } finally {
      Loading.remove()
    }
  }

  const guardarRegistros = async () => {
    const recordsForUpdate = almacenStock.filter((productStockRecord) => productStockRecord._hasUpdated)
    if (recordsForUpdate.length === 0) {
      Notify.failure('No hay registros a actualizar.')
      return
    }

    Loading.standard('Enviando registros...')
    try {
      await postProductosStock(recordsForUpdate)
      // Reset the visual diff marker only after the backend confirms the save.
      for (const productStockRecord of almacenStock) {
        productStockRecord._cantidadPrev = 0
        productStockRecord._hasUpdated = false
      }
    } finally {
      Loading.remove()
    }
  }

  const fillAllProductos = () => {
    const productStockMap = new Map(almacenStockGetted?.map((productStockRecord) => [productStockRecord.ID, productStockRecord]))
    for (const productRecord of productos.records) {
      const presentationIDs = productRecord.Presentaciones?.length > 0 ? productRecord.Presentaciones.map((presentationOption) => presentationOption.id) : [0]
      for (const presentacionID of presentationIDs) {
        const stockID = [stockFilters.warehouseID, productRecord.ID, presentacionID || '', '', ''].join('_')
        if (productStockMap.has(stockID)) {
          const existingProductStock = productStockMap.get(stockID) as IProductoStock
          existingProductStock.PresentationID = presentacionID
          continue
        }

        productStockMap.set(stockID, {
          ID: stockID,
          WarehouseID: stockFilters.warehouseID,
          ProductID: productRecord.ID,
          PresentationID: presentacionID,
          Quantity: 0
        } as IProductoStock)
      }
    }
    almacenStock = [...productStockMap.values()]
  }

  const updateStockQuantity = (productStockRecord: IProductoStock, cantidad: number) => {
    if (!productStockRecord.ID) {
      productStockRecord.ID = [
        productStockRecord.WarehouseID, productStockRecord.ProductID, productStockRecord.PresentationID || 0, productStockRecord.SKU || '', productStockRecord.Lote || ''
      ].join('_')
    }

    const existingProductStock = almacenStock.find((rowRecord) => rowRecord.ID === productStockRecord.ID) || almacenStockGetted.find((rowRecord) => rowRecord.ID === productStockRecord.ID)

    if (existingProductStock) {
      existingProductStock._hasUpdated = true
      existingProductStock._cantidadPrev = existingProductStock._cantidadPrev || existingProductStock.Quantity || -1
      existingProductStock.Quantity = cantidad
    } else {
      productStockRecord._cantidadPrev = -1
      productStockRecord._hasUpdated = true
      almacenStockGetted.unshift(productStockRecord)
      almacenStock.unshift(productStockRecord)
      almacenStock = [...almacenStock]
    }
  }

  $effect(() => {
    if (stockFilters.showTodosProductos) {
      untrack(() => { fillAllProductos() })
    } else {
      untrack(() => { almacenStock = almacenStockGetted })
    }
  })
</script>

<div class="flex items-center mb-8">
  <SearchSelect options={almacenes?.Almacenes || []} keyId="ID" keyName="Nombre"
    bind:saveOn={stockFilters} save="warehouseID" placeholder="ALMACÉN ::"
    css="w-270"
    onChange={() => {
      onChangeAlmacen()
    }}
  />
  {#if !stockFilters.warehouseID}
    <div class="ml-12 text-red-500"><i class="icon-attention"></i>Debe seleccionar un almacén.</div>
  {:else}
    <div class="i-search w-full md:w-200 md:ml-12 col-span-5">
      <div><i class="icon-search"></i></div>
      <input type="text" onkeyup={(event) => {
        const value = String((event.target as HTMLInputElement).value || '')
        throttle(() => { stockFilterText = value }, 150)
      }}>
    </div>
    <Checkbox label="Todos los Productos" bind:saveOn={stockFilters} save="showTodosProductos"
      css="ml-16" />
  {/if}
  <div class="ml-auto">
    {#if stockFilters.warehouseID > 0}
      <button class="bx-blue mr-8" onclick={() => {
        guardarRegistros()
      }}>
        <i class="icon-floppy"></i>Guardar
      </button>
      <button class="bx-green" aria-label="agregar" onclick={() => {
        stockForm = { WarehouseID: stockFilters.warehouseID } as IProductoStock
        Core.openSideLayer(1)
      }}>
        <i class="icon-plus"></i>
      </button>
    {/if}
  </div>
</div>

<VTable columns={stockColumns} data={almacenStock}
  filterText={stockFilterText}
  useFilterCache={true}
  getFilterContent={(productStockRecord) => {
    const productRecord = productos.recordsMap.get(productStockRecord.ProductID)
    return [productRecord?.Nombre, productStockRecord.SKU, productStockRecord.Lote].filter((value) => value).join(' ').toLowerCase()
  }}
/>

<Layer id={1} type="bottom" css="p-12 min-h-360" title="Agregar Stock" titleCss="h2"
  saveButtonName="Agregar" saveButtonIcon="icon-ok" contentOverflow={true}
  onSave={() => {
    if (!stockForm.ProductID || !stockForm.Quantity) {
      Notify.failure('Debe seleccionar un producto y una cantidad.')
      return
    }
    updateStockQuantity(stockForm, stockForm.Quantity)
    Core.hideSideLayer()
  }}
>
  <div class="grid grid-cols-24 gap-10 mt-6 p-4">
    <SearchSelect label="Producto" css="col-span-24" required={true}
      bind:saveOn={stockForm} save="ProductID" options={productos.records || []}
      keyName="Nombre" keyId="ID"
    />
    {#if (stockFormProducto?.Presentaciones?.filter((presentationOption) => presentationOption.ss) || []).length > 0}
      <SearchSelect label="Presentación" css="col-span-24"
        bind:saveOn={stockForm} save="PresentationID" options={stockFormProducto?.Presentaciones || []}
        keyName="nm" keyId="id"
      />
    {/if}
    <Input label="SKU" css="col-span-16"
      bind:saveOn={stockForm} save="SKU"
    />
    <Input label="Cantidad" required={true} css="col-span-8"
      bind:saveOn={stockForm} save="Quantity" type="number"
    />
    <Input label="Lote" css="col-span-16"
      bind:saveOn={stockForm} save="Lote"
    />
    <Input label="Costo x Unidad" css="col-span-8"
      bind:saveOn={stockForm} save="CostoUn" type="number"
    />
  </div>
</Layer>
