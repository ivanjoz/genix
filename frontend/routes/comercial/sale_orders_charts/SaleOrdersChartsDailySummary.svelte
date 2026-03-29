<script lang="ts">
  import CheckboxOptions from '$components/CheckboxOptions.svelte'
  import SquareBarSized from '$components/micro/SquareBarSized.svelte'
  import VirtualCards from '$components/VirtualCards.svelte'
  import { FechaHelper } from '$libs/fecha'
  import { formatN, formatTime } from '$libs/helpers'
  import type { IProducto } from '$routes/negocio/productos/productos.svelte'
  import type { ISaleSummaryRecord } from './sale_orders_charts.svelte'

  type TChartMetricMode = 'amount' | 'quantity'

  interface IChartMetricForm {
    metricMode?: TChartMetricMode
  }

  interface IDailyProductSummary {
    productID: number
    productName: string
    metricValue: number
  }

  interface IDailySummaryCard {
    fechaUnix: number
    totalSalesMetricValue: number
    paidMetricValue: number
    unpaidMetricValue: number
    paidRatio: number
    unpaidRatio: number
    totalQuantity: number
    deliveredQuantity: number
    deliveredRatio: number
    variationAgainstLast7DaysAvg: number | null
    variationAgainstPreviousWeekday: number | null
    topProducts: IDailyProductSummary[]
    topMetricMaxValue: number
  }

  interface SaleOrdersChartsDailySummaryProps {
    chartMetricForm: IChartMetricForm
    saleSummaryRecords: ISaleSummaryRecord[]
    productsByIdMap: Map<number, IProducto>
  }

  const fechaHelper = new FechaHelper()
  const CARD_ROW_HEIGHT_PX = 332
  const chartMetricSelectionOptions: Array<{ ID: TChartMetricMode; Nombre: string }> = [
    { ID: 'amount', Nombre: 'Por Monto Facturado' },
    { ID: 'quantity', Nombre: 'Por Cantidad' }
  ]

  let {
    chartMetricForm,
    saleSummaryRecords,
    productsByIdMap,
  }: SaleOrdersChartsDailySummaryProps = $props()

  const selectedMetricMode = $derived<TChartMetricMode>(chartMetricForm.metricMode || 'amount')

  const fechaUnixRangeToRender = $derived.by(() => {
    const currentFechaUnix = fechaHelper.fechaUnixCurrent()
    const oldestFechaUnixInSummary = saleSummaryRecords.reduce((oldestFechaUnix, summaryRecord) => {
      if (!oldestFechaUnix) { return summaryRecord.Fecha }
      return Math.min(oldestFechaUnix, summaryRecord.Fecha)
    }, 0)
    const firstFechaUnix = oldestFechaUnixInSummary || currentFechaUnix
    const totalDaysToRender = Math.max(1, currentFechaUnix - firstFechaUnix + 1)

    return Array.from({ length: totalDaysToRender }, (_, dayIndex) => {
      return currentFechaUnix - dayIndex
    })
  })

  const dailySummaryCards = $derived.by((): IDailySummaryCard[] => {
    const summaryRecordByFechaUnix = new Map<number, ISaleSummaryRecord>()

    for (const summaryRecord of saleSummaryRecords) {
      summaryRecordByFechaUnix.set(summaryRecord.Fecha, summaryRecord)
    }

    const dailyCards: IDailySummaryCard[] = []

    for (const fechaUnix of fechaUnixRangeToRender) {
      const summaryRecord = summaryRecordByFechaUnix.get(fechaUnix)
      if (!summaryRecord) {
        dailyCards.push({
          fechaUnix,
          totalSalesMetricValue: 0,
          paidMetricValue: 0,
          unpaidMetricValue: 0,
          paidRatio: 0,
          unpaidRatio: 0,
          totalQuantity: 0,
          deliveredQuantity: 0,
          deliveredRatio: 0,
          variationAgainstLast7DaysAvg: null,
          variationAgainstPreviousWeekday: null,
          topProducts: [],
          topMetricMaxValue: 0
        })
        continue
      }

      const productIDs = summaryRecord.ProductIDs || []
      const totalAmountsInCents = summaryRecord.TotalAmount || []
      const unpaidAmountsInCents = summaryRecord.TotalDebtAmount || []
      const soldQuantities = summaryRecord.Quantity || []
      const pendingQuantities = summaryRecord.QuantityPendingDelivery || []
      const topProducts: IDailyProductSummary[] = []

      let paidMetricValue = 0
      let unpaidMetricValue = 0
      let totalQuantity = 0
      let deliveredQuantity = 0

      for (let recordIndex = 0; recordIndex < productIDs.length; recordIndex += 1) {
        const productID = productIDs[recordIndex] || 0
        if (!productID) { continue }

        const productName = productsByIdMap.get(productID)?.Nombre || `Producto #${productID}`
        const totalQuantityForProduct = soldQuantities[recordIndex] || 0
        const pendingQuantityForProduct = pendingQuantities[recordIndex] || 0
        const deliveredQuantityForProduct = Math.max(0, totalQuantityForProduct - pendingQuantityForProduct)
        const totalAmount = (totalAmountsInCents[recordIndex] || 0) / 100
        const unpaidAmount = (unpaidAmountsInCents[recordIndex] || 0) / 100
        const paidAmount = Math.max(0, totalAmount - unpaidAmount)

        paidMetricValue += selectedMetricMode === 'quantity' ? deliveredQuantityForProduct : paidAmount
        unpaidMetricValue += selectedMetricMode === 'quantity' ? pendingQuantityForProduct : unpaidAmount
        totalQuantity += totalQuantityForProduct
        deliveredQuantity += deliveredQuantityForProduct

        const productMetricValue = selectedMetricMode === 'quantity'
          ? totalQuantityForProduct
          : totalAmount

        if (productMetricValue > 0) {
          topProducts.push({
            productID,
            productName,
            metricValue: productMetricValue
          })
        }
      }

      topProducts.sort((leftProduct, rightProduct) => {
        return rightProduct.metricValue - leftProduct.metricValue
      })

      const topProductsLimited = topProducts.slice(0, 10)
      const topMetricMaxValue = topProductsLimited.reduce((maxValue, topProduct) => {
        return Math.max(maxValue, topProduct.metricValue)
      }, 0)
      const totalInvoicedMetricValue = paidMetricValue + unpaidMetricValue
      const totalSalesMetricValue = totalInvoicedMetricValue
      const paidRatio = totalInvoicedMetricValue > 0 ? paidMetricValue / totalInvoicedMetricValue : 0
      const unpaidRatio = totalInvoicedMetricValue > 0 ? unpaidMetricValue / totalInvoicedMetricValue : 0
      const deliveredRatio = totalQuantity > 0 ? deliveredQuantity / totalQuantity : 0

      dailyCards.push({
        fechaUnix,
        totalSalesMetricValue,
        paidMetricValue,
        unpaidMetricValue,
        paidRatio,
        unpaidRatio,
        totalQuantity,
        deliveredQuantity,
        deliveredRatio,
        variationAgainstLast7DaysAvg: null,
        variationAgainstPreviousWeekday: null,
        topProducts: topProductsLimited,
        topMetricMaxValue
      })
    }

    const dailyCardByFechaUnix = new Map<number, IDailySummaryCard>()
    for (const dailyCard of dailyCards) {
      dailyCardByFechaUnix.set(dailyCard.fechaUnix, dailyCard)
    }

    for (const dailyCard of dailyCards) {
      const previous7DaysCards = Array.from({ length: 7 }, (_, dayOffset) => {
        return dailyCardByFechaUnix.get(dailyCard.fechaUnix - (dayOffset + 1))
      }).filter((record): record is IDailySummaryCard => Boolean(record))

      const previous7DaysAverage = previous7DaysCards.length > 0
        ? previous7DaysCards.reduce((totalValue, previousCard) => {
          return totalValue + previousCard.totalSalesMetricValue
        }, 0) / previous7DaysCards.length
        : 0

      const previousWeekdayCard = dailyCardByFechaUnix.get(dailyCard.fechaUnix - 7)

      dailyCard.variationAgainstLast7DaysAvg = previous7DaysAverage > 0
        ? dailyCard.totalSalesMetricValue - previous7DaysAverage
        : null

      dailyCard.variationAgainstPreviousWeekday = previousWeekdayCard && previousWeekdayCard.totalSalesMetricValue > 0
        ? dailyCard.totalSalesMetricValue - previousWeekdayCard.totalSalesMetricValue
        : null
    }

    return dailyCards
  })

  const formatMetricValue = (metricValue: number) => {
    if (!metricValue) { return selectedMetricMode === 'quantity' ? '0' : '0.00' }

    return selectedMetricMode === 'quantity'
      ? formatN(metricValue, 0)
      : `${formatN(metricValue, 2)}`
  }

  const getTopProductWidthPercent = (metricValue: number, topMetricMaxValue: number) => {
    if (!metricValue || !topMetricMaxValue) { return 0 }
    return Math.max(18, Math.round((metricValue / topMetricMaxValue) * 100))
  }

  const formatVariationValue = (variationValue: number | null) => {
    if (variationValue === null || variationValue === undefined) { return '--' }
    return `${variationValue > 0 ? '+' : ''}${formatN(variationValue, 0)}`
  }

  const getVariationColorClass = (variationValue: number | null) => {
    if (variationValue === null || variationValue === undefined) { return 'text-slate-400' }
    return variationValue >= 0 ? 'text-green-600' : 'text-red-500'
  }

  const getVariationIconClass = (variationValue: number | null) => {
    if (variationValue === null || variationValue === undefined) { return 'icon-minus' }
    return variationValue >= 0 ? 'icon-up-bold' : 'icon-down-bold'
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
  <div class="ml-auto flex items-center gap-12 text-[13px] text-slate-600">
    <div>{dailySummaryCards.length} días</div>
    <div>Top 10 productos</div>
  </div>
</div>

{#if saleSummaryRecords.length === 0}
  <div class="text-gray-500">No hay registros de ventas para mostrar.</div>
{:else if dailySummaryCards.length === 0}
  <div class="text-gray-500">No se encontraron días con ventas en los últimos 45 días.</div>
{:else}
  <div class="sale-orders-daily-cards-shell h-[calc(100vh-var(--header-height)-124px)] min-h-[320px] overflow-hidden">
    <VirtualCards
      items={dailySummaryCards}
      height="100%"
      maxColumns={3}
      estimatedRowHeight={CARD_ROW_HEIGHT_PX}
      bufferSize={4}
    >
      {#snippet children(dailySummaryCard, _cardIndex)}
        <div class="rounded-[12px] border border-slate-200 bg-white p-8 md:p-14 shadow-[0_10px_24px_rgba(15,23,42,0.04)]">
          <div class="mb-16 flex items-start gap-14">
            <div class="flex w-90 md:w-110 shrink-0">
              <div class="flex flex-col gap-10">
                <div class="text-[20px] ff-bold leading-none text-slate-900">
                  {String(formatTime(dailySummaryCard.fechaUnix, 'M-d') || '')}
                </div>
                <div class={`flex ff-mono items-center text-[15px] leading-none ${getVariationColorClass(dailySummaryCard.variationAgainstLast7DaysAvg)}`}>
                  <i class={getVariationIconClass(dailySummaryCard.variationAgainstLast7DaysAvg)}></i>
                  <span>{formatVariationValue(dailySummaryCard.variationAgainstLast7DaysAvg)}</span>
                </div>
                <div class={`flex ff-mono items-center text-[15px] leading-none ${getVariationColorClass(dailySummaryCard.variationAgainstPreviousWeekday)}`}>
                  <i class={getVariationIconClass(dailySummaryCard.variationAgainstPreviousWeekday)}></i>
                  <span>{formatVariationValue(dailySummaryCard.variationAgainstPreviousWeekday)}</span>
                </div>
              </div>
            </div>

            <div class="grid h-94 w-full grid-cols-3 items-end gap-10">
              <SquareBarSized useStripedLines="#F0F0F0"
                label="Ingresado"
                value={formatMetricValue(dailySummaryCard.paidMetricValue)}
                size={dailySummaryCard.paidRatio}
                backgroundColor="#93c5fd"
                background="linear-gradient(32deg, rgb(178 244 255) 0%, rgb(127 205 251) 99%)"
              />

              <SquareBarSized
                label="Por Cobrar"
                value={formatMetricValue(dailySummaryCard.unpaidMetricValue)}
                size={dailySummaryCard.unpaidRatio}
                backgroundColor="#fdd1cd"
                background="linear-gradient(32deg, #ffe5e5 0%, #ffb5b5 99%)"
              />

              <SquareBarSized useStripedLines="#F0F0F0"
                label="Entregado"
                value={`${formatN(dailySummaryCard.deliveredRatio * 100, 0)}%`}
                sublabel={`${formatN(dailySummaryCard.deliveredQuantity, 0)} / ${formatN(dailySummaryCard.totalQuantity, 0)}`}
                size={dailySummaryCard.deliveredRatio}
                backgroundColor="#d9e7bf"
                background="linear-gradient(32deg, #d1fdd5 0%, #94ed9c 99%)"
              />
            </div>
          </div>

          <div class="flex flex-col gap-4">
            {#if dailySummaryCard.topProducts.length === 0}
              <div class="py-24 text-[14px] text-slate-500">
                Sin Ventas
              </div>
            {:else}
              {#each dailySummaryCard.topProducts as topProductSummary (topProductSummary.productID)}
                <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-10">
                  <div class="relative h-28 overflow-hidden rounded-[4px] bg-[#f6f0fc]">
                    <div
                      class="absolute inset-y-0 left-0 bar-bg"
                      style={`width:${getTopProductWidthPercent(topProductSummary.metricValue, dailySummaryCard.topMetricMaxValue)}%`}
                    ></div>
                    <div class="relative z-[1] line-clamp-1 px-12 pt-6 text-xs leading-[1.2] text-slate-900">
                      {topProductSummary.productName}
                    </div>
                  </div>

                  <div class="w-70 text-right text-[14px] leading-none ff-mono text-slate-900">
                    {formatMetricValue(topProductSummary.metricValue)}
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        </div>
      {/snippet}
    </VirtualCards>
  </div>
{/if}

<style>
  .sale-orders-daily-cards-shell :global(.virtual-list-items) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-bottom: 12px;
  }
  .bar-bg {
 		background: linear-gradient(352deg, rgb(241 205 255) 0%, rgb(240 225 255) 99%);
  }
</style>
