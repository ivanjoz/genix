<script lang="ts">
import Input from '$components/Input.svelte'
import Layer from '$components/Layer.svelte'
import CardsList from '$components/vTable/CardsList.svelte'
import VTable from '$components/vTable/VTable.svelte'
import type { ICardCell, ITableColumn } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { formatN, Loading, Notify, throttle } from '$libs/helpers'
import { ClientProviderService, ClientProviderType } from '../../negocio/clientes/clientes-proveedores.svelte'
import { ProductosService } from '../../negocio/productos/productos.svelte'
import {
  createEmptyProviderSupplyRow,
  normalizeProviderSupplyRows,
  postProductSupply,
  ProductSupplyService,
  type IProductSupplyProviderRow,
  type IProductSupplyRow,
} from './abastecimiento.svelte'

  const productos = new ProductosService()
  const providers = new ClientProviderService(ClientProviderType.PROVIDER)
  const productSupplyService = new ProductSupplyService()

  let supplyFilterText = $state('')
  let productSupplyForm = $state({
    ProductID: 0,
    MinimunStock: 0,
    SalesPerDayEstimated: 0,
    ProviderSupply: [],
  } as IProductSupplyRow)

  const productSupplyTableRows = $derived.by(() => {
    return productos.records.map((productRecord) => {
      const savedProductSupplyRecord = productSupplyService.recordsMap.get(productRecord.ID)
      return {
        ProductID: productRecord.ID,
        MinimunStock: savedProductSupplyRecord?.MinimunStock || 0,
        SalesPerDayEstimated: savedProductSupplyRecord?.SalesPerDayEstimated || 0,
        ProviderSupply: normalizeProviderSupplyRows(savedProductSupplyRecord?.ProviderSupply || []).filter((providerSupplyRow) => {
          return providerSupplyRow.ProviderID > 0
        }),
      } as IProductSupplyRow
    })
  })

  const productSupplyColumns: ITableColumn<IProductSupplyRow>[] = [
    {
      header: 'Producto',
      highlight: true,
      cellCss: 'px-6',
      getValue: (productSupplyRecord) => productos.recordsMap.get(productSupplyRecord.ProductID)?.Nombre || `Producto-${productSupplyRecord.ProductID}`,
    },
    {
      header: 'Stock Min.',
      headerCss: 'w-112',
      cellCss: 'px-6 text-right',
      getValue: (productSupplyRecord) => productSupplyRecord.MinimunStock || 0,
    },
    {
      header: 'Ventas / Día',
      headerCss: 'w-128',
      cellCss: 'px-6 text-right',
      getValue: (productSupplyRecord) => productSupplyRecord.SalesPerDayEstimated || 0,
    },
    {
      header: 'Proveedores',
      cellCss: 'px-6',
      getValue: (productSupplyRecord) => {
        return (productSupplyRecord.ProviderSupply || []).map((providerSupplyRow) => {
          return providers.recordsMap.get(providerSupplyRow.ProviderID)?.Name || `Proveedor-${providerSupplyRow.ProviderID}`
        }).join(', ')
      },
    },
  ]

  const providerSupplyCards = $derived.by((): ICardCell<IProductSupplyProviderRow>[] => {
    // Keep provider options reactive so card selectors receive records loaded after the first render.
    return [
      {
        label: 'Proveedor',
        field: 'ProviderID',
        itemCss: 'col-span-24 md:col-span-12',
        cellOptions: providers.records || [],
        cellOptionsKeyId: 'ID',
        cellOptionsKeyName: 'Name',
        onCellSelect: (providerSupplyRow, selectedValue) => {
          providerSupplyRow.ProviderID = Number(selectedValue || 0)
        },
      },
      {
        label: 'Capacidad',
        field: 'Capacity',
        type: 'number',
        itemCss: 'col-span-12 md:col-span-4',
        inputCss: 'text-right pr-6',
        getValue: (providerSupplyRow) => providerSupplyRow.Capacity || 0,
        onCellEdit: (providerSupplyRow, nextValue) => {
          providerSupplyRow.Capacity = parseInt(String(nextValue || '0'))
        },
      },
      {
        label: 'Entrega',
        field: 'DeliveryTime',
        type: 'number',
        itemCss: 'col-span-12 md:col-span-4',
        inputCss: 'text-right pr-6',
        getValue: (providerSupplyRow) => providerSupplyRow.DeliveryTime || 0,
        onCellEdit: (providerSupplyRow, nextValue) => {
          providerSupplyRow.DeliveryTime = parseInt(String(nextValue || '0'))
        },
      },
      {
        label: 'Precio',
        field: 'Price',
        itemCss: 'col-span-12 md:col-span-4',
        inputCss: 'text-right pr-6',
        getValue: (providerSupplyRow) => formatN((providerSupplyRow.Price || 0) / 100, 2),
        onCellEdit: (providerSupplyRow, nextValue) => {
          const parsedPrice = parseFloat(String(nextValue || '0'))
          providerSupplyRow.Price = Number.isFinite(parsedPrice)
            ? Math.round(parsedPrice * 100)
            : 0
        },
      },
    ]
  })

  function openProductSupplyLayer(selectedProductSupplyRecord: IProductSupplyRow) {
    // Clone the record and sanitize provider rows so the layer only renders real cards.
    productSupplyForm = {
      ...selectedProductSupplyRecord,
      ProviderSupply: normalizeProviderSupplyRows(selectedProductSupplyRecord.ProviderSupply || []),
    }
    Core.openSideLayer(2)
  }

  function addProviderSupplyRow() {
    console.debug('AbastecimientoView::addProviderSupplyRow')
    productSupplyForm.ProviderSupply = [
      ...(productSupplyForm.ProviderSupply || []),
      createEmptyProviderSupplyRow(),
    ]
  }

  function removeProviderSupplyRow(providerRowIndex: number) {
    console.debug('AbastecimientoView::removeProviderSupplyRow', { providerRowIndex })
    const currentProviderSupplyRows = [...(productSupplyForm.ProviderSupply || [])]
    currentProviderSupplyRows.splice(providerRowIndex, 1)
    productSupplyForm.ProviderSupply = currentProviderSupplyRows
  }

  async function saveProductSupply() {
    if (!productSupplyForm.ProductID) {
      Notify.failure('Debe seleccionar un producto válido.')
      return
    }

    Loading.standard('Guardando abastecimiento...')

    try {
      const savedProductSupply = await postProductSupply(productSupplyForm)
      const normalizedSavedProductSupply = {
        ...savedProductSupply,
        ProviderSupply: (savedProductSupply.ProviderSupply || []).filter((providerSupplyRow: IProductSupplyProviderRow) => providerSupplyRow.ProviderID > 0),
        ss: savedProductSupply.ss || 1,
        upd: savedProductSupply.upd || 0,
        UpdatedBy: savedProductSupply.UpdatedBy || 0,
      } as IProductSupplyRow

      // Keep the local table responsive while the delta refresh finishes in the background.
      const nextProductSupplyRecordsByProductID = new Map<number, IProductSupplyRow>()
      for (const existingProductSupplyRecord of productSupplyService.records) {
        nextProductSupplyRecordsByProductID.set(existingProductSupplyRecord.ProductID, existingProductSupplyRecord)
      }
      nextProductSupplyRecordsByProductID.set(normalizedSavedProductSupply.ProductID, normalizedSavedProductSupply)
      productSupplyService.handler([...nextProductSupplyRecordsByProductID.values()])
      productSupplyService.fetchOnline()

      Core.hideSideLayer()
      Notify.success('Configuración de abastecimiento guardada correctamente.')
    } catch (saveError) {
      Notify.failure(String(saveError))
    } finally {
      Loading.remove()
    }
  }

</script>

<div class="flex w-full">
  <Layer type="content">
	  <div class="mb-6 flex items-center justify-between">
	    <div class="i-search mr-16 w-320 max-w-full">
	      <div><i class="icon-search"></i></div>
	      <input class="w-full" autocomplete="off" type="text" placeholder="Buscar producto o proveedor"
	        onkeyup={(event) => {
	          event.stopPropagation()
	          throttle(() => {
	            supplyFilterText = ((event.target as HTMLInputElement).value || '').trim()
	          }, 150)
	        }}
	      />
	    </div>
	    <div class="flex items-center">
	      <div class="h6 ff-bold pr-8 text-slate-500">
	        {productSupplyTableRows.length} registros
	      </div>
	    </div>
	  </div>
  	<VTable
	      css="h-full w-full"
	      maxHeight="calc(100vh - var(--header-height) - 60px - 1rem)"
	      columns={productSupplyColumns}
	      data={productSupplyTableRows}
	      filterText={supplyFilterText}
	      useFilterCache={true}
	      getFilterContent={(productSupplyRecord) => {
	        const productRecord = productos.recordsMap.get(productSupplyRecord.ProductID)
	        const providerNames = (productSupplyRecord.ProviderSupply || []).map((providerSupplyRow) => {
	          return providers.recordsMap.get(providerSupplyRow.ProviderID)?.Name || ''
	        })
	
	        return [
	          productRecord?.Nombre || '',
	          String(productSupplyRecord.MinimunStock || 0),
	          String(productSupplyRecord.SalesPerDayEstimated || 0),
	          ...providerNames,
	        ].filter((value) => value).join(' ').toLowerCase()
	      }}
	      selected={productSupplyForm?.ProductID}
	      isSelected={(productSupplyRecord, selectedProductID) => productSupplyRecord.ProductID === selectedProductID}
	      onRowClick={(selectedProductSupplyRecord) => {
	        openProductSupplyLayer(selectedProductSupplyRecord)
	      }}
	    /> 
  </Layer>
</div>

<Layer id={2} type="side" sideLayerSize={720}
  title={`Abastecimiento ${productos.recordsMap.get(productSupplyForm.ProductID)?.Nombre || ''}`}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={saveProductSupply}
  onClose={() => {
    productSupplyForm = {
      ProductID: 0,
      MinimunStock: 0,
      SalesPerDayEstimated: 0,
      ProviderSupply: [],
    }
  }}
>
  <div class="grid grid-cols-24 gap-10 mt-8">
    <Input
      label="Producto"
      saveOn={{ Nombre: productos.recordsMap.get(productSupplyForm.ProductID)?.Nombre || '' }}
      save="Nombre"
      css="col-span-24"
      disabled={true}
    />
    <Input
      label="Stock mínimo"
      saveOn={productSupplyForm}
      save="MinimunStock"
      css="col-span-24 md:col-span-12"
      type="number"
    />
    <Input
      label="Ventas / Día estimadas"
      saveOn={productSupplyForm}
      save="SalesPerDayEstimated"
      css="col-span-24 md:col-span-12"
      type="number"
    />
  </div>

  <div class="mt-16">
    <div class="mb-8 flex items-center justify-between">
      <div class="h4 ff-bold">Proveedores</div>
      <button class="bx-green h-32 px-10" type="button" aria-label="Agregar proveedor" title="Agregar proveedor"
        onclick={() => {
          addProviderSupplyRow()
        }}
      >
        <i class="icon-plus"></i>
      </button>
    </div>

    <CardsList
      css="w-full"
      maxHeight="320px"
      cardCss="p-14"
      viewportClass="p-4"
      cells={providerSupplyCards}
      data={productSupplyForm.ProviderSupply || []}
      emptyMessage="No hay proveedores agregados."
      buttonDeleteHandler={(_, providerRowIndex) => {
        removeProviderSupplyRow(providerRowIndex)
      }}
    />
  </div>
</Layer>
