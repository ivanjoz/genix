<script lang="ts">
import Checkbox from '$components/Checkbox.svelte'
import Input from '$components/Input.svelte'
import Layer from '$components/Layer.svelte'
import SearchSelect from '$components/SearchSelect.svelte'
import TableGrid from '$components/vTable/TableGrid.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ITableColumn } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { formatN, Loading, Notify, throttle } from '$libs/helpers'
import { untrack } from 'svelte'
import { ProductosService } from '../../negocio/productos/productos.svelte'
import { AlmacenesService } from '../../negocio/sedes-almacenes/sedes-almacenes.svelte'
import { getProductosStock, makeStockID, postProductosStock, type IProductoStock } from './stock-movement'
    import type { ElementAST } from '$components/Renderer.svelte';

  type IProductoStockDisplay = {
  	base: IProductoStock
    groupKey: string
  	lotes: IProductoStock[]
  	skus: IProductoStock[]
    stockSimple: number
    stockLoteado: number
    stockSkus: number
    stockSimpleNew: number
    stockLoteadoNew: number
    stockSkusNew: number
    computed?: boolean
  }

  const almacenes = new AlmacenesService()
  const productos = new ProductosService(true)

  let stockFilters = $state({ warehouseID: 0, showTodosProductos: false })
  let stockFilterText = $state('')
  let almacenStock = $state([] as IProductoStock[])
  let almacenStockGetted = [] as IProductoStock[]
  let stockForm = $state({} as IProductoStock)
  let selectedSkuGroup = $state<IProductoStockDisplay | null>(null)
  let selectedSkus = $state<IProductoStock[]>([])
  let selectedLoteGroup = $state<IProductoStockDisplay | null>(null)
  let selectedLotes = $state<IProductoStock[]>([])
  const freshPendingSku = (): IProductoStock => ({ _isNew: true, SKU: '', Lote: '', Quantity: 0 } as IProductoStock)
  const freshPendingLote = (): IProductoStock => ({ _isNew: true, Lote: '', Quantity: 0 } as IProductoStock)

  const stockFormProducto = $derived(productos?.recordsMap?.get(stockForm.ProductID || 0))
  const groupToNewStockRecords: Map<string,IProductoStock[]> = new Map()
  const groupToNewLoteRecords: Map<string,IProductoStock[]> = new Map()
  
  const getCantPrevia = (r: IProductoStock) => {
  	if(r._cantidadPrev === -1){ return 0 }
  	return (typeof r._cantidadPrev === 'number' && r._cantidadPrev !== -1) ? r._cantidadPrev : r.Quantity
  }
  
  const displayStock = $derived.by((): IProductoStockDisplay[] => {
  	almacenStock;
   	const result: IProductoStockDisplay[] = []
  	untrack(() => {
		   const recordGroup = new Map<string, IProductoStockDisplay>()
		   for (const record of almacenStock) {
				 const groupKey = [record.ProductID, record.PresentationID||0].join('_')

				 if(!recordGroup.has(groupKey)){
						recordGroup.set(groupKey, {
							groupKey,
								lotes: [] as IProductoStock[],
								skus: [] as IProductoStock[]
						} as IProductoStockDisplay)
				 }

				 const group = recordGroup.get(groupKey) as IProductoStockDisplay

		     if (record.SKU) {
						group.skus.push(record)
					} else if (record.Lote){
						group.lotes.push(record)
					}		else {
						group.base = record
					}
		   }

			 for(const e of recordGroup.values()){
					if(!e.base){
						const record = e.lotes[0] || e.skus[0]
						const baseSeed = {
							WarehouseID: record.WarehouseID,
							ProductID: record.ProductID,
							PresentationID: record.PresentationID,
						} as IProductoStock
						e.base = { ...baseSeed, ID: makeStockID(baseSeed) } as IProductoStock
					}

					result.push(e)
				}
   	})
   	return result
  })

  const selectedSkuGroupTitle = $derived.by(() => {
    if (!selectedSkuGroup) { return 'SKUs' }
    const productName = productos.recordsMap.get(selectedSkuGroup.base.ProductID)?.Nombre || ''
    const count = selectedSkus.filter(s => s.SKU).length
    return `${productName} — ${count} SKU${count !== 1 ? 's' : ''}`
  })

  const selectedLoteGroupTitle = $derived.by(() => {
    if (!selectedLoteGroup) { return 'Lotes' }
    const productName = productos.recordsMap.get(selectedLoteGroup.base.ProductID)?.Nombre || ''
    const count = selectedLotes.filter(s => s.Lote).length
    return `${productName} — ${count} Lote${count !== 1 ? 's' : ''}`
  })

  const validateLoteSkuUnique = (record: IProductoStock, checkSku: string, checkLote: string): boolean => {
    if (!checkSku) return true
    const isDuplicate = (r: IProductoStock) =>
      r !== record && r.SKU === checkSku && (r.Lote || '') === (checkLote || '')
    const duplicate = selectedSkus.some(isDuplicate)
    if (duplicate) { Notify.warning('La combinación Lote + SKU ya existe.') }
    return !duplicate
  }

  const validateLoteUnique = (record: IProductoStock, checkLote: string): boolean => {
    if (!checkLote) return true
    const isDuplicate = (r: IProductoStock) =>
      r !== record && (r.Lote || '') === checkLote
    const duplicate = selectedLotes.some(isDuplicate)
    if (duplicate) { Notify.warning('El Lote ya existe.') }
    return !duplicate
  }

  const skuColumns: ITableColumn<IProductoStock>[] = [
    {
      header: 'SKU', cellCss: 'ff-mono',
      getValue: (sku) => sku.SKU || '',
      disableCellInteractions: (sku) => !sku._isNew,
      onBeforeCellChange: (sku, value) => validateLoteSkuUnique(sku, String(value || ''), sku.Lote || ''),
      onCellEdit: (sku, value) => {
        const newSku = String(value || '')
        sku.SKU = newSku
        if (sku._isNew && newSku && !selectedSkus.some(s => s._isNew && !s.SKU)) {
          const pending = selectedSkus.filter(s => s._isNew)
          const backend = selectedSkus.filter(s => !s._isNew)
          selectedSkus = [...pending, freshPendingSku(), ...backend]
        }
      },
      render: e => {
      	if(!e.SKU){ return { css: "text-xs c-red", text: "NUEVO SKU..." } }
     		return e.SKU as string
      }
    },
    {
      header: 'Lote',
      getValue: (sku) => sku.Lote || '',
      onBeforeCellChange: (sku, value) => validateLoteSkuUnique(sku, sku.SKU || '', String(value || '')),
      onCellEdit: (sku, value) => {
        sku.Lote = String(value || '')
        if (!sku._isNew) {
          sku.ID = makeStockID(sku)
          sku._hasUpdated = true
        }
      }
    },
    {
      header: 'Cant.', css: 'justify-end', inputCss: 'text-right pr-6',
      getValue: (sku) => sku.Quantity ?? '',
      disableCellInteractions: (sku) => !sku._isNew,
      cellInputType: "number",
      onCellEdit: (sku, value) => {
        if (!sku._isNew) return
        sku.WarehouseID = selectedSkuGroup!.base.WarehouseID
        sku.ProductID = selectedSkuGroup!.base.ProductID
        sku.PresentationID = selectedSkuGroup!.base.PresentationID || 0
        sku.Quantity = parseInt(value as string || '0')
        sku._cantidadPrev = -1
        sku._hasUpdated = true
      },
      render: (sku) => {
        if (sku._cantidadPrev && sku._cantidadPrev !== sku.Quantity) {
          return { css: 'flex items-center', children: [
            { text: String(sku._cantidadPrev > 0 ? sku._cantidadPrev : 0) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: sku.Quantity, css: 'text-red-500' }
          ]}
        }
        return { css: "text-right", text: sku.Quantity || '' }
      }
    }
  ]

  const loteColumns: ITableColumn<IProductoStock>[] = [
    {
      header: 'Lote',
      getValue: (lote) => lote.Lote || '',
      disableCellInteractions: (lote) => !lote._isNew,
      onBeforeCellChange: (lote, value) => validateLoteUnique(lote, String(value || '')),
      onCellEdit: (lote, value) => {
        const newLote = String(value || '')
        lote.Lote = newLote
        if (lote._isNew && newLote && !selectedLotes.some(s => s._isNew && !s.Lote)) {
          const pending = selectedLotes.filter(s => s._isNew)
          const backend = selectedLotes.filter(s => !s._isNew)
          selectedLotes = [...pending, freshPendingLote(), ...backend]
        }
      },
      render: e => {
      	if(!e.Lote){ return { css: "text-xs c-red", text: "NUEVO LOTE..." } }
     		return e.Lote as string
      }
    },
    {
      header: 'Cant.', css: 'justify-end', inputCss: 'text-right pr-6',
      getValue: (lote) => lote.Quantity ?? '',
      disableCellInteractions: (lote) => !lote._isNew,
      cellInputType: "number",
      onCellEdit: (lote, value) => {
        if (!lote._isNew) return
        lote.WarehouseID = selectedLoteGroup!.base.WarehouseID
        lote.ProductID = selectedLoteGroup!.base.ProductID
        lote.PresentationID = selectedLoteGroup!.base.PresentationID || 0
        lote.Quantity = parseInt(value as string || '0')
        lote._cantidadPrev = -1
        lote._hasUpdated = true
      },
      render: (lote) => {
        if (lote._cantidadPrev && lote._cantidadPrev !== lote.Quantity) {
          return { css: 'flex items-center', children: [
            { text: String(lote._cantidadPrev > 0 ? lote._cantidadPrev : 0) },
            { text: '→', css: 'ml-2 mr-2' },
            { text: lote.Quantity, css: 'text-red-500' }
          ]}
        }
        return { css: "text-right", text: lote.Quantity || '' }
      }
    }
  ]

  const computeStockDisplay = (e: IProductoStockDisplay) => {
    if (e.computed) return

    e.stockSimple = getCantPrevia(e.base)
    e.stockSimpleNew = e.base.Quantity || 0

    e.stockLoteado = 0; e.stockLoteadoNew = 0
    for (const r of [...e.lotes, ...(groupToNewLoteRecords.get(e.groupKey) || [])]) {
      e.stockLoteado += getCantPrevia(r) || 0
      e.stockLoteadoNew += r.Quantity || 0
    }

    e.stockSkus = 0; e.stockSkusNew = 0
    for (const r of [...e.skus, ...(groupToNewStockRecords.get(e.groupKey) || [])]) {
      e.stockSkus += getCantPrevia(r) || 0
      e.stockSkusNew += r.Quantity || 0
    }

    e.computed = true
  }

  const stockColumns: ITableColumn<IProductoStockDisplay>[] = [
    { header: 'Producto', highlight: true,
   		mobile: { order: 1, css: "col-span-24" },
      getValue: (e) => {
        const productRecord = productos.recordsMap.get(e.base.ProductID)?.Nombre
        return productRecord || `Producto-${e.base.ProductID}`
      },
      render: e => {
        const productRecord = productos.recordsMap.get(e.base.ProductID)?.Nombre || `Producto-${e.base.ProductID}`
      		return {
       	css: "flex",
        children: [
      		{ tagName: "SPAN", text: productRecord  },
         	e.skus.length > 0 && {tagName: "SPAN",  text: "(SKU)", css: "ff-bold c-red ml-6"  },
         ]
        } as ElementAST
      }
    },
    { header: 'Presentación',
    	mobile: { order: 2, css: "col-span-24" },
      getValue: (e) => {
        if (!e.base.PresentationID) { return '' }
        const productRecord = productos.recordsMap.get(e.base.ProductID)
        const presentationRecord = productRecord?.Presentaciones?.find(p => p.id === e.base.PresentationID)
        return presentationRecord?.nm || `Tipo-${e.base.PresentationID}`
      }
    },
    { header: 'SKU',
      getValue: (e) => e.skus.length > 0 ? `${e.skus.length} SKUs` : '',
      mobile: { order: 3, css: "col-span-12" },
      showEditIcon: true,
 	   	onCellClick: (record, index, rerender) => {
	      rerenderHandler = rerender
	      const cached = groupToNewStockRecords.get(record.groupKey) || []
	      selectedSkuGroup = record
	      selectedSkus = [...cached, freshPendingSku(), ...record.skus]
	      Core.openSideLayer(2)
	    },
    },
    { header: 'Stock Simple', css: 'justify-end px-6', inputCss: 'text-right',
   		headerCss: 'w-150',
    	mobile: { order: 4, css: "col-span-12" },
      showEditIcon: true,
      cellInputType: "number",
      getValue: (e) => { computeStockDisplay(e); return e.stockSimpleNew },
      onCellEdit: (e, value, rerender) => {
        e.computed = false
        updateStockQuantity(e.base, parseInt(value as string || '0'))
        rerender()
      },
      render: (e) => {
        if (e.stockSimpleNew !== e.stockSimple) {
          return {
            css: 'flex items-center justify-end',
            children: [
              { text: String(e.stockSimple) },
              { text: '→', css: 'ml-2 mr-2' },
              { text: e.stockSimpleNew, css: 'text-red-500' }
            ]
          }
        }
        return { css: "text-right", text: e.stockSimpleNew || '' }
      }
    },
    { header: 'Stock Loteado',
      showEditIcon: true,
	   	onCellClick: (record, index, rerender) => {
	      rerenderHandler = rerender
	      const cached = groupToNewLoteRecords.get(record.groupKey) || []
	      selectedLoteGroup = record
	      selectedLotes = [...cached, freshPendingLote(), ...record.lotes]
	      Core.openSideLayer(3)
	    },
      getValue: (e) => { computeStockDisplay(e); return e.stockLoteadoNew },
      render: (e) => {
      	computeStockDisplay(e)
        if (e.stockLoteadoNew !== e.stockLoteado) {
          return {
            css: 'flex justify-end items-center',
            children: [
              { text: String(e.stockLoteado) },
              { text: '→', css: 'ml-2 mr-2' },
              { text: e.stockLoteadoNew, css: 'text-red-500' }
            ]
          }
        }
        return { css: "text-right", text: e.stockLoteadoNew || '' }
      }
    },
    { header: 'Stock Total', css: 'justify-end text-right',
      getValue: e => { computeStockDisplay(e); return formatN(e.stockLoteadoNew + e.stockSimpleNew + e.stockSkusNew) }
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
    // Flush pending new SKU/Lote records held in the side-layer maps into almacenStock
    // so they participate in the save. Clear the maps to avoid double-counting in displayStock.
    const pendingNewRecords: IProductoStock[] = []
    for (const newRecords of groupToNewStockRecords.values()) { pendingNewRecords.push(...newRecords) }
    for (const newRecords of groupToNewLoteRecords.values()) { pendingNewRecords.push(...newRecords) }
    groupToNewStockRecords.clear()
    groupToNewLoteRecords.clear()

    for (const r of pendingNewRecords) {
      if (!r.ID) { r.ID = makeStockID(r) }
      r._hasUpdated = true
      almacenStock.push(r)
    }

    const recordsForUpdate = almacenStock.filter((productStockRecord) => productStockRecord._hasUpdated)
    if (recordsForUpdate.length === 0) {
      Notify.failure('No hay registros a actualizar.')
      return
    }

    const seenIDs = new Set<string>()
    for (const r of recordsForUpdate) {
      const id = r.ID || makeStockID(r)
      if (seenIDs.has(id)) {
        Notify.failure('Hay registros con ID duplicado (mismo SKU y Lote). Verifica antes de guardar.')
        return
      }
      seenIDs.add(id)
    }

    Loading.standard('Enviando registros...')
    try {
      await postProductosStock(recordsForUpdate)
      // Reset the visual diff marker only after the backend confirms the save.
      for (const productStockRecord of almacenStock) {
        productStockRecord._cantidadPrev = 0
        productStockRecord._hasUpdated = false
        productStockRecord._isNew = false
      }
      rerenderHandler?.()
    } finally {
      Loading.remove()
    }
  }

  const fillAllProductos = () => {
    const productStockMap = new Map(almacenStockGetted?.map((productStockRecord) => [productStockRecord.ID, productStockRecord]))
    for (const productRecord of productos.records) {
      const presentationIDs = productRecord.Presentaciones?.length > 0 ? productRecord.Presentaciones.map((presentationOption) => presentationOption.id) : [0]
      for (const presentacionID of presentationIDs) {
        const stockID = makeStockID({ WarehouseID: stockFilters.warehouseID, ProductID: productRecord.ID, PresentationID: presentacionID } as IProductoStock)
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
 		// alert("hola")
    if (!productStockRecord.ID) {
      productStockRecord.ID = makeStockID(productStockRecord)
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
    }
  }

  $effect(() => {
    if (stockFilters.showTodosProductos) {
      untrack(() => { fillAllProductos() })
    } else {
      untrack(() => { almacenStock = almacenStockGetted })
    }
  })
  
  let rerenderHandler: ((() => void)|undefined) = undefined
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
      <button class="bx-green shrink-0" aria-label="agregar" onclick={() => {
        stockForm = { WarehouseID: stockFilters.warehouseID } as IProductoStock
        Core.openSideLayer(1)
      }}>
        <i class="icon-plus"></i>
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
  getFilterContent={(e) => {
    const productRecord = productos.recordsMap.get(e.base.ProductID)
    const skuText = e.skus.map(r => r.SKU).filter(Boolean).join(' ')
    const loteText = e.lotes.map(r => r.Lote).filter(Boolean).join(' ')
    return [productRecord?.Nombre, skuText, loteText].filter((value) => value).join(' ').toLowerCase()
  }}
/>

<Layer id={2} type="side" sideLayerSize={740} title={selectedSkuGroupTitle} titleCss="h2"
  css="px-12 py-10"
  onClose={() => {
    if (selectedSkuGroup) {
   		const selected = displayStock.find(x => x.base.ID === selectedSkuGroup?.base.ID) as IProductoStockDisplay
     	
      const newSkus = selectedSkus.filter(s => s._isNew && s.SKU)
      if (newSkus.length > 0) {
        groupToNewStockRecords.set(selected.groupKey, newSkus)
      }
      selected.computed = false
      console.log("selectedSkuGroup",$state.snapshot(selected),"|", rerenderHandler)
      rerenderHandler?.()
    }
    selectedSkus = []
    selectedSkuGroup = null
  }}
>
  <TableGrid columns={skuColumns} data={selectedSkus} height="calc(100vh - 14rem)"
 		cellCss="px-6" headerCss="px-6 flex items-center min-h-32" css="mt-6"
  	getRowId={e => `${e.SKU}_${e.Lote}`}
  />
</Layer>

<Layer id={3} type="side" sideLayerSize={620} title={selectedLoteGroupTitle} titleCss="h2"
  css="px-12 py-10"
  onClose={() => {
    if (selectedLoteGroup) {
   		const selected = displayStock.find(x => x.base.ID === selectedLoteGroup?.base.ID) as IProductoStockDisplay

      const newLotes = selectedLotes.filter(s => s._isNew && s.Lote)
      if (newLotes.length > 0) {
        groupToNewLoteRecords.set(selected.groupKey, newLotes)
      }
      selected.computed = false
      rerenderHandler?.()
    }
    selectedLotes = []
    selectedLoteGroup = null
  }}
>
  <TableGrid columns={loteColumns} data={selectedLotes} height="calc(100vh - 14rem)"
 		cellCss="px-6" headerCss="px-6 flex items-center min-h-32" css="mt-6"
  	getRowId={e => `${e.Lote}`}
  />
</Layer>

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
