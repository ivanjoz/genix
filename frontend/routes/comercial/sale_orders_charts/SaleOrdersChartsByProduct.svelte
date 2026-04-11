<script lang="ts">
  import ChartCanvas, { type ChartCanvasSeries } from '$components/ChartCanvas.svelte'
  import CheckboxOptions from '$components/CheckboxOptions.svelte'
  import HighlightText from '$components/micro/HighlightText.svelte'
  import VirtualCards from '$components/VirtualCards.svelte'
  import { FechaHelper } from '$libs/fecha'
  import { formatN, formatTime, wordInclude, throttle } from '$libs/helpers'
  import type { IProducto } from '$routes/negocio/productos/productos.svelte'
  import type { ISaleSummaryRecord } from './sale_orders_charts.svelte'

  type TChartMetricMode = 'amount' | 'quantity'

  interface IChartMetricForm {
    metricMode?: TChartMetricMode
  }

  interface IProductChartCard {
    productID: number
    productName: string
    paidValues: number[]
    unpaidValues: number[]
    priceValues: Array<number | null>
    priceAxisMin: number
    priceAxisMax: number
    totalMetricValue: number
    totalPaidMetricValue: number
    totalUnpaidMetricValue: number
    priceReference: number
  }

  interface SaleOrdersChartsByProductProps {
    chartMetricForm: IChartMetricForm
    saleSummaryRecords: ISaleSummaryRecord[]
    productsByIdMap: Map<number, IProducto>
  }

  const fechaHelper = new FechaHelper()
  const TOTAL_DAYS_TO_RENDER = 45
  const CARD_ROW_HEIGHT_PX = 206
  const chartMetricSelectionOptions: Array<{ ID: TChartMetricMode; Nombre: string }> = [
    { ID: 'amount', Nombre: 'Por Monto Facturado' },
    { ID: 'quantity', Nombre: 'Por Cantidad' }
  ]

  let {
    chartMetricForm,
    saleSummaryRecords,
    productsByIdMap,
  }: SaleOrdersChartsByProductProps = $props()

  let filterText = $state('')

  const selectedMetricMode = $derived<TChartMetricMode>(chartMetricForm.metricMode || 'amount')

  const getLineAxisRangeWithMargin = (lineValues: Array<number | null>) => {
    const numericLineValues = lineValues.filter((lineValue): lineValue is number => {
      return lineValue !== null && lineValue !== undefined
    })

    if (!numericLineValues.length) {
      return {
        minValue: 0,
        maxValue: 1
      }
    }

    const minLineValue = Math.min(...numericLineValues)
    const maxLineValue = Math.max(...numericLineValues)
    const lowerMargin = Math.max(Math.abs(minLineValue) * 0.2, 1)
    const upperMargin = Math.max(Math.abs(maxLineValue) * 0.2, 1)

    return {
      minValue: minLineValue - lowerMargin,
      maxValue: maxLineValue + upperMargin
    }
  }

  // Normalize mixed backend date formats to internal "fecha unix" day units.
  const toFechaUnix = (rawFechaValue: number): number => {
    if (!rawFechaValue) return 0

    if (rawFechaValue > 10_000 && rawFechaValue < 100_000) {
      return Math.floor(rawFechaValue)
    }

    const normalizedFechaValue = rawFechaValue > 1_000_000_000_000
      ? Math.floor(rawFechaValue / 1000)
      : rawFechaValue

    return fechaHelper.toFechaUnix(normalizedFechaValue)
  }

  const last45FechaUnix = $derived.by(() => {
    const latestFechaUnixInSummary = saleSummaryRecords.reduce((latestFechaUnix, summaryRecord) => {
      return Math.max(latestFechaUnix, toFechaUnix(summaryRecord.Fecha || 0))
    }, 0)
    const anchorFechaUnix = latestFechaUnixInSummary || fechaHelper.fechaUnixCurrent()

    return Array.from({ length: TOTAL_DAYS_TO_RENDER }, (_, dayIndex) => {
      return anchorFechaUnix - (TOTAL_DAYS_TO_RENDER - 1) + dayIndex
    })
  })

  const productChartCards = $derived.by((): IProductChartCard[] => {
    const productChartsByID = new Map<number, IProductChartCard>()
    const fechaIndexByUnix = new Map<number, number>()

    for (let dayIndex = 0; dayIndex < last45FechaUnix.length; dayIndex += 1) {
      fechaIndexByUnix.set(last45FechaUnix[dayIndex], dayIndex)
    }

    for (const summaryRecord of saleSummaryRecords) {
      const recordFechaUnix = toFechaUnix(summaryRecord.Fecha || 0)
      const recordDayIndex = fechaIndexByUnix.get(recordFechaUnix)
      if (recordDayIndex === undefined) {
        continue
      }

      const productIDs = summaryRecord.ProductIDs || []
      const totalAmountsInCents = summaryRecord.TotalAmount || []
      const unpaidAmountsInCents = summaryRecord.TotalDebtAmount || []
      const soldQuantities = summaryRecord.Quantity || []
      const pendingQuantities = summaryRecord.QuantityPendingDelivery || []

      for (let recordIndex = 0; recordIndex < productIDs.length; recordIndex += 1) {
        const productID = productIDs[recordIndex] || 0
        if (!productID) { continue }

        const productRecord = productsByIdMap.get(productID)
        const totalMetricValue = selectedMetricMode === 'quantity'
          ? (soldQuantities[recordIndex] || 0)
          : (totalAmountsInCents[recordIndex] || 0) / 100
        const unpaidMetricValue = selectedMetricMode === 'quantity'
          ? (pendingQuantities[recordIndex] || 0)
          : (unpaidAmountsInCents[recordIndex] || 0) / 100
        const paidMetricValue = Math.max(0, totalMetricValue - unpaidMetricValue)

        let productChartCard = productChartsByID.get(productID)
        if (!productChartCard) {
          const productPrice = (productRecord?.PrecioFinal || productRecord?.Precio || 0) / 100
          const priceValues = Array.from({ length: TOTAL_DAYS_TO_RENDER }, () => productPrice > 0 ? productPrice : null)
          const priceAxisRange = getLineAxisRangeWithMargin(priceValues)
          productChartCard = {
            productID,
            productName: productRecord?.Nombre || `Producto #${productID}`,
            paidValues: Array.from({ length: TOTAL_DAYS_TO_RENDER }, () => 0),
            unpaidValues: Array.from({ length: TOTAL_DAYS_TO_RENDER }, () => 0),
            priceValues,
            priceAxisMin: priceAxisRange.minValue,
            priceAxisMax: priceAxisRange.maxValue,
            totalMetricValue: 0,
            totalPaidMetricValue: 0,
            totalUnpaidMetricValue: 0,
            priceReference: productPrice
          }
          productChartsByID.set(productID, productChartCard)
        }

        productChartCard.paidValues[recordDayIndex] += paidMetricValue
        productChartCard.unpaidValues[recordDayIndex] += unpaidMetricValue
        productChartCard.totalMetricValue += totalMetricValue
        productChartCard.totalPaidMetricValue += paidMetricValue
        productChartCard.totalUnpaidMetricValue += unpaidMetricValue
      }
    }

    return [...productChartsByID.values()].sort((leftCard, rightCard) => {
      return rightCard.totalMetricValue - leftCard.totalMetricValue
    })
  })

  const chartLegendSeries = $derived.by((): ChartCanvasSeries[] => {
    return [
      { name: 'Ventas no pagadas', type: 'bar', values: [1], color: '#ef4444' },
      { name: 'Ventas pagadas', type: 'bar', values: [1], color: '#3b82f6' },
      { name: 'Precio', type: 'line', values: [1], color: '#111827', lineWidth: 2 }
    ]
  })

  const filterWords = $derived.by(() => {
    return (filterText || '').toLowerCase().split(' ').filter((filterWord) => filterWord.length > 0)
  })

  const filteredProductChartCards = $derived.by(() => {
    if (!filterText) { return productChartCards }

    return productChartCards.filter((productChartCard) => {
      return wordInclude(productChartCard.productName.toLowerCase(), filterWords)
    })
  })

  const formatMetricValue = (metricValue: number) => {
    if (!metricValue) { return '0' }
    return selectedMetricMode === 'quantity'
      ? formatN(metricValue, 0)
      : `S/ ${formatN(metricValue, 2)}`
  }
</script>

<div class="mb-12 flex flex-wrap items-center gap-10">
  <CheckboxOptions useButtons
    options={chartMetricSelectionOptions}
    saveOn={chartMetricForm}
    save={'metricMode'}
    keyId={'ID'}
    keyName={'Nombre'}
    type="single"
  />
  <div class="i-search mr-auto w-200">
    <div><i class="icon-search"></i></div>
    <input class="w-full" autocomplete="off" type="text" onkeyup={ev => {
      ev.stopPropagation()
      throttle(() => {
        filterText = ((ev.target as HTMLInputElement).value || '').toLowerCase().trim()
      }, 150)
    }}>
  </div>
  <div class="flex items-center gap-12 text-[13px] text-slate-600">
    <div>{filteredProductChartCards.length} productos</div>
    <div>{TOTAL_DAYS_TO_RENDER} días</div>
  </div>
</div>

{#if saleSummaryRecords.length === 0}
  <div class="text-gray-500">No hay registros de ventas para mostrar.</div>
{:else if filteredProductChartCards.length === 0}
  <div class="text-gray-500">No se encontraron productos con ventas en los últimos 45 días.</div>
{:else}
  <div class="flex w-full">
    <div class="mb-8 ml-auto mt-[-6px] flex flex-wrap items-center gap-12">
      {#each chartLegendSeries as chartLegendItem (chartLegendItem.name)}
        <div class="flex items-center gap-8 text-[13px] text-slate-600">
          {#if chartLegendItem.type === 'line'}
            <i class="icon-tag"></i>
          {:else}
            <div class="h-10 w-18 rounded-[3px] bg-slate-900"
              style={`background:${chartLegendItem.color || '#000000'}`}
            ></div>
          {/if}
          <span>{chartLegendItem.name}</span>
        </div>
      {/each}
    </div>
  </div>

  <div class="sale-orders-cards-virtual-shell h-[calc(100vh-var(--header-height)-124px)] min-h-[320px] overflow-hidden">
    <VirtualCards
      items={filteredProductChartCards}
      height="100%"
      maxColumns={3}
      estimatedRowHeight={CARD_ROW_HEIGHT_PX}
      bufferSize={4}
    >
      {#snippet children(productChartCard, _cardIndex)}
        <div class="rounded-[12px] border border-slate-200 bg-white p-8 shadow-[0_10px_24px_rgba(15,23,42,0.04)]">
          <div class="mb-8">
            <div class="line-clamp-2 font-semibold leading-[1.2] text-slate-800">
              <HighlightText text={productChartCard.productName} words={filterWords} />
            </div>
          </div>

          <div class="mb-8 flex flex-wrap items-center gap-8 rounded-[10px] bg-slate-50 px-8 py-6 text-[14px] leading-none">
            {#if productChartCard.totalPaidMetricValue > 0}
              <div class="flex items-center gap-6 text-slate-800">
                <div class="h-10 w-10 rounded-[3px] bg-[#3b82f6]"></div>
                <div>{formatMetricValue(productChartCard.totalPaidMetricValue)}</div>
              </div>
            {/if}
            {#if productChartCard.totalUnpaidMetricValue > 0}
              <div class="flex items-center gap-6 text-slate-800">
                <div class="h-10 w-10 rounded-[3px] bg-[#ef4444]"></div>
                <div>{formatMetricValue(productChartCard.totalUnpaidMetricValue)}</div>
              </div>
            {/if}
            <div class="flex items-center gap-6 text-slate-800">
              <i class="icon-tag text-slate-600"></i>
              <div>
                {productChartCard.priceReference ? `S/ ${formatN(productChartCard.priceReference, 2)}` : 'Sin precio'}
              </div>
            </div>
          </div>

          <ChartCanvas
            id={`sale-orders-product-${productChartCard.productID}-${selectedMetricMode}`}
            height={80}
            className="h-full w-full"
            dateLabels={last45FechaUnix}
            dateLabelEvery={6}
            dateLabelFormatter={(fechaUnix) => String(formatTime(Number(fechaUnix || 0), 'd-M') || '')}
            useHtmlRendered={true}
            data={[
              { name: 'Ventas no pagadas', type: 'bar', values: productChartCard.unpaidValues, color: '#ef4444' },
              { name: 'Ventas pagadas', type: 'bar', values: productChartCard.paidValues, color: '#3b82f6' },
              {
                name: 'Precio',
                type: 'line',
                values: productChartCard.priceValues,
                color: '#111827',
                lineWidth: 1.5,
                // Keep price readable against sales volume bars with its own configured range.
                useOwnAxis: true,
                yAxisMin: productChartCard.priceAxisMin,
                yAxisMax: productChartCard.priceAxisMax
              }
            ]}
          />
        </div>
      {/snippet}
    </VirtualCards>
  </div>
{/if}

<style>
  .sale-orders-cards-virtual-shell :global(.virtual-list-items) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-bottom: 12px;
  }
</style>
