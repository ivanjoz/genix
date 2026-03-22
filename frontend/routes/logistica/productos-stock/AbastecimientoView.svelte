<script lang="ts">
import Input from '$components/Input.svelte'
import Layer from '$components/Layer.svelte'
import ChartCanvas from '$components/ChartCanvas.svelte'
import CardsList from '$components/vTable/CardsList.svelte'
import TableGrid from '$components/vTable/TableGrid.svelte'
import type { TableGridColumn } from '$components/vTable/tableGridTypes'
import type { ICardCell } from '$components/vTable/types'
import { Core } from '$core/store.svelte'
import { FechaHelper } from '$libs/fecha'
import { formatN, formatTime, Loading, Notify, throttle } from '$libs/helpers'
import { onDestroy, onMount, untrack } from 'svelte'
import { ClientProviderService, ClientProviderType } from '../../negocio/clientes/clientes-proveedores.svelte'
import { ProductosService } from '../../negocio/productos/productos.svelte'
import {
  AlmacenMovimientosGroupedService,
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
  const groupedMovementsService = new AlmacenMovimientosGroupedService()
  const fechaHelper = new FechaHelper()
  const salesWindowDays = 30
  const debugProductID = 10034
  const yAxisWidthPx = 28
  const ventasWidthRatio = 0.30

  // Freeze the 30-day window in fechaUnix units so every row uses the same oldest->newest categories.
  const last30FechaUnix = Array.from({ length: salesWindowDays }, (_, dayOffset) => { 
    return fechaHelper.fechaUnixCurrent() - (salesWindowDays - 1) + dayOffset
  })

  let pageContentElement = $state<HTMLDivElement | undefined>(undefined)
  let pageContentWidthPx = $state(0)
  let pageResizeObserver: ResizeObserver | undefined

  const updatePageContentWidth = () => {
    if (!pageContentElement) { return }
    pageContentWidthPx = Math.max(0, Math.floor(pageContentElement.clientWidth))
  }

  const ventasPixelMetrics = $derived.by(() => {
    const ventasBaseWidthPx = Math.max(120, Math.round(pageContentWidthPx * ventasWidthRatio))
    const ventasBarWidthPx = Math.max(1, Math.round(ventasBaseWidthPx / salesWindowDays))
    const ventasChartWidthPx = ventasBarWidthPx * salesWindowDays
    const ventasColumnWidthPx = ventasChartWidthPx + yAxisWidthPx

    return {
      ventasBaseWidthPx,
      ventasBarWidthPx,
      ventasChartWidthPx,
      ventasColumnWidthPx,
    }
  })

  const salesHeaderLabels = $derived.by(() => {
    const labelsPerGroup = 3
    const groupedLabels = []

    // Use every third day so the header axis stays readable and matches the 30-point chart width.
    for (let groupStartIndex = 0; groupStartIndex < last30FechaUnix.length; groupStartIndex += labelsPerGroup) {
      const fechaUnix = last30FechaUnix[groupStartIndex] + 2
      groupedLabels.push({
        fechaUnix,
        day: String(formatTime(fechaUnix, 'd') || ''),
        month: String(formatTime(fechaUnix, 'M') || ''),
      })
    }

    return groupedLabels
  })

  $effect(() => {
    if (!groupedMovementsService.isReady) {
      return
    }

    // Keep logging side-effects outside the tracked section so service updates do not create render loops.
    untrack(() => {
      console.log('AbastecimientoView::groupedMovementRecords', groupedMovementsService.records)
    })
  })

  $effect(() => {
    const debugProductMovements = groupedMovementsService.productMovementsMap.get(debugProductID)
    if (!debugProductMovements) {
      return
    }

    const debugOutflowsValues = last30FechaUnix.map((fechaUnix) => debugProductMovements.getOutflows(fechaUnix) || 0)
    const debugInflowsValues = last30FechaUnix.map((fechaUnix) => debugProductMovements.getInflows(fechaUnix) || 0)
    const debugFinalStockValues = last30FechaUnix.map((fechaUnix) => debugProductMovements.getFinalStock(fechaUnix) || 0)
    const debugFinalStockLineValues = debugFinalStockValues.map((finalStockValue) => finalStockValue !== 0 ? finalStockValue : null)

    // Keep the chart diagnostics grouped so the raw vector and rendered series are easy to compare.
    untrack(() => {
      console.group(`AbastecimientoView::debugProduct:${debugProductID}`)
      console.log('rawMovementVector', Array.from(debugProductMovements.v))
      console.table(last30FechaUnix.map((fechaUnix, dayIndex) => ({
        dayIndex,
        fechaUnix,
        dateLabel: String(formatTime(fechaUnix, 'Y-m-d') || ''),
        fechaIndex: debugProductMovements.getFechaIndex(fechaUnix),
        outflows: debugOutflowsValues[dayIndex],
        inflows: debugInflowsValues[dayIndex],
        finalStock: debugFinalStockValues[dayIndex],
        finalStockLineValue: debugFinalStockLineValues[dayIndex],
      })))
      console.log('chartSeries', {
        outflowsValues: debugOutflowsValues,
        inflowsValues: debugInflowsValues,
        finalStockValues: debugFinalStockValues,
        finalStockLineValues: debugFinalStockLineValues,
      })
      console.groupEnd()
    })
  })

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

  const filteredProductSupplyRows = $derived.by(() => {
    const normalizedFilterText = supplyFilterText.trim().toLowerCase()
    if (!normalizedFilterText) {
      return productSupplyTableRows
    }

    return productSupplyTableRows.filter((productSupplyRecord) => {
      const productRecord = productos.recordsMap.get(productSupplyRecord.ProductID)
      const providerNames = (productSupplyRecord.ProviderSupply || []).map((providerSupplyRow) => {
        return providers.recordsMap.get(providerSupplyRow.ProviderID)?.Name || ''
      })

      // Keep the filter inline with the virtualized grid so we only render matching rows.
      const filterableContent = [
        productRecord?.Nombre || '',
        String(productSupplyRecord.MinimunStock || 0),
        String(productSupplyRecord.SalesPerDayEstimated || 0),
        ...providerNames,
      ].filter((value) => value).join(' ').toLowerCase()

      return filterableContent.includes(normalizedFilterText)
    })
  })

  const productSupplyColumns = $derived.by((): TableGridColumn<IProductSupplyRow>[] => {
    return [
      {
        id: 'product-name',
        header: 'Producto',
        width: '32%',
        cellCss: 'px-6 py-4 leading-[1.1] whitespace-normal',
        getValue: (productSupplyRecord) => productos.recordsMap.get(productSupplyRecord.ProductID)?.Nombre || `Producto-${productSupplyRecord.ProductID}`,
      },
      {
        id: 'minimum-stock',
        header: 'Stock Min.',
        width: '6%',
        align: 'right',
        cellCss: 'px-6 text-right',
        getValue: (productSupplyRecord) => productSupplyRecord.MinimunStock || 0,
      },
      {
        id: 'sales-last-30-days',
        header: 'Movimientos Stock',
        width: `${ventasPixelMetrics.ventasColumnWidthPx}px`,
        useCellRenderer: true,
        cellCss: 'px-0',
      },
      {
        id: 'sales-per-day',
        header: 'Ventas / Día',
        width: '8%',
        align: 'right',
        cellCss: 'px-6 text-right',
        getValue: (productSupplyRecord) => productSupplyRecord.SalesPerDayEstimated || 0,
      },
      {
        id: 'providers',
        header: 'Proveedores',
        width: '18%',
        cellCss: 'px-6 whitespace-normal',
        getValue: (productSupplyRecord) => {
          return (productSupplyRecord.ProviderSupply || []).map((providerSupplyRow) => {
            return providers.recordsMap.get(providerSupplyRow.ProviderID)?.Name || `Proveedor-${providerSupplyRow.ProviderID}`
          }).join(', ')
        },
      },
    ]
  })

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
        contentCss: 'w-full justify-end text-right pr-6',
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
        contentCss: 'w-full justify-end text-right pr-6',
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
        contentCss: 'w-full justify-end text-right pr-6',
        inputCss: 'text-right pr-6',
        getValue: (providerSupplyRow) => formatN((providerSupplyRow.Price || 0) / 100, 2),
        render: e => e.Price ? formatN((e.Price||0)/100,2) : "",
        onCellEdit: (providerSupplyRow, nextValue) => {
          const parsedPrice = parseFloat(String(nextValue || '0'))
          providerSupplyRow.Price = Math.round((parsedPrice||0) * 100)
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

  onMount(() => {
    updatePageContentWidth()

    if (!pageContentElement) { return }

    pageResizeObserver = new ResizeObserver(() => {
      updatePageContentWidth()
    })
    pageResizeObserver.observe(pageContentElement)
  })

  onDestroy(() => {
    pageResizeObserver?.disconnect()
  })

</script>

<div class="flex w-full" bind:this={pageContentElement}>
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
	        {filteredProductSupplyRows.length} registros
	      </div>
	    </div>
	  </div>
	  	<TableGrid
	      css="h-full w-full"
        headerCss="text-[14px]"
	      height="calc(100vh - var(--header-height) - 60px - 1rem)"
        rowHeight={48}
	      columns={productSupplyColumns}
	      data={filteredProductSupplyRows}
        selectedRowId={productSupplyForm?.ProductID || undefined}
        getRowId={(productSupplyRecord) => productSupplyRecord.ProductID}
	      onRowClick={(selectedProductSupplyRecord) => {
	        openProductSupplyLayer(selectedProductSupplyRecord)
	      }}
	    >
        {#snippet headerRenderer(columnDefinition, _columnIndex)}
          {#if columnDefinition.id === 'sales-last-30-days'}
            <div class="flex flex-col gap-2 py-2">
              <div class="px-10">{columnDefinition.header}</div>
              <div class="min-w-0" style={`width:${ventasPixelMetrics.ventasColumnWidthPx}px`}>
                <div class="grid items-center text-[11px] text-slate-500" style={`padding-left:${yAxisWidthPx - 4}px;grid-template-columns:repeat(${salesHeaderLabels.length}, ${ventasPixelMetrics.ventasBarWidthPx * 3}px)`}>
                  {#each salesHeaderLabels as salesHeaderLabel (salesHeaderLabel.fechaUnix)}
                    <div class="overflow-hidden text-right text-ellipsis whitespace-nowrap leading-[1.1]">
	                    <div>{salesHeaderLabel.day}</div>
	                    <div>{salesHeaderLabel.month}</div>
                    </div>
                  {/each}
                </div>
              </div>
            </div>
          {:else}
            <div class="px-10 py-8">
              {columnDefinition.header}
            </div>
          {/if}
        {/snippet}
        {#snippet cellRenderer(productSupplyRecord, columnDefinition)}
          {#if columnDefinition.id === 'sales-last-30-days'}
            <!-- Keep the sales chart inline with TableGrid so virtualization owns the full row render path. -->
            {@const productMovements = groupedMovementsService.productMovementsMap.get(productSupplyRecord.ProductID)}
            {@const outflowsValues = last30FechaUnix.map((fechaUnix) => productMovements?.getOutflows(fechaUnix) || 0)}
            {@const inflowsValues = last30FechaUnix.map((fechaUnix) => productMovements?.getInflows(fechaUnix) || 0)}
            {@const finalStockValues = last30FechaUnix.map((fechaUnix) => productMovements?.getFinalStock(fechaUnix) || 0)}
            <div class="h-50 min-w-0" style={`width:${ventasPixelMetrics.ventasColumnWidthPx}px`}>
              <ChartCanvas
                id={productSupplyRecord.ProductID}
                height={48}
                className="h-full w-full"
                fixedPointWidthPx={ventasPixelMetrics.ventasBarWidthPx}
                data={[
                  // Keep the red outflow stack first so demand pressure remains visually dominant.
                  { name: 'Salidas', type: 'bar', values: outflowsValues, color: '#ef4444' },
                  { name: 'Entradas', type: 'bar', values: inflowsValues, color: '#3b82f6' },
                  // Draw the reconstructed daily final stock so replenishment decisions use the latest stock history.
                  {
                    name: 'Stock Final',
                    type: 'line',
                    values: finalStockValues.map((finalStockValue) => finalStockValue !== 0 ? finalStockValue : null),
                    color: '#000000',
                    lineWidth: 1,
                    pointSize: 3
                  },
                ]}
              />
            </div>
          {/if}
        {/snippet}
      </TableGrid>
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
