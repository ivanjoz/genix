<script lang="ts">
  import { useUI } from '@genix/ui'
  import Page from '#domain/Page.svelte'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import { formatPeriod, periodOfDay } from '#routes/hr/payroll/payroll.ts'
  import { onDestroy } from 'svelte'
  import CashFlowChart from './CashFlowChart.svelte'
  import CashFlowTable from './CashFlowTable.svelte'
  import CashFlowWarnings from './CashFlowWarnings.svelte'
  import PlannedEntriesTable from './PlannedEntriesTable.svelte'
  import { CashFlowState } from './cash-flow.svelte.ts'
  import {
    buildCashFlowGrid, CASH_FLOW_HORIZON_DAYS, CASH_FLOW_LAYER_NAMES, CASH_FLOW_STREAMS, CASH_FLOW_YEAR_DAYS, cashBanksByLane,
    CASH_BANK_TYPE_DETRACTIONS, cashFlowLanes, detractionsOpeningOf, firstNegativeDay, LANE_DETRACTIONS, LANE_TOTAL, monthlyGrid,
    payTaxesFromDetractions, PLANNED_STREAM_ID, type ICashFlowCell, type ICashFlowGrid, type ICashFlowRow,
  } from './cash-flow.ts'
  import { pageWarnings, rankWarnings, usesDollars } from './cash-flow-warnings.ts'

  // The cash flow forecast: today's cash rolled forward with what the company owes and is owed
  // (Known), what its rules will generate (Scheduled), what its history forecasts (Forecast, which
  // the user can leave out) and what it plans. By day for 13 weeks or by month for a year; one lane
  // at a time: a site, the company lane (no site) or the whole company.
  const ui = useUI()
  const WARNINGS_LAYER_ID = 1
  const DRILLDOWN_LAYER_ID = 2
  const VIEW_DAILY = 1
  const VIEW_MONTHLY = 2
  const VIEW_PLANNED = 3
  const cashFlow = new CashFlowState()
  onDestroy(() => cashFlow.destroy())

  let viewOptions = $state({ lane: LANE_TOTAL, includeForecast: true })
  let drilldown = $state(null as { title: string, items: ICashFlowCell['items'], isLoading: boolean } | null)

  const isMonthly = $derived(ui.state.pageOptionSelected === VIEW_MONTHLY)
  const detractionsOpening = $derived(detractionsOpeningOf(cashFlow.cashBanksService.Cajas))
  const hasDetractionsAccount = $derived(cashFlow.cashBanksService.Cajas.some((cashBank) => cashBank.Type === CASH_BANK_TYPE_DETRACTIONS && cashBank.ss > 0))
  const lanes = $derived(cashFlowLanes(cashFlow.sitesService.Sedes, hasDetractionsAccount))
  const laneCashBanks = $derived(cashBanksByLane(cashFlow.cashBanksService.Cajas))
  // The detracciones account's balance pays the taxes first: that part leaves the available cash.
  const shownCommitments = $derived(payTaxesFromDetractions(viewOptions.includeForecast
    ? [...cashFlow.commitments, ...cashFlow.forecastCommitments] : cashFlow.commitments, detractionsOpening))
  // The daily view's 13 weeks, or the year folded by month.
  const gridOf = (lane: number): ICashFlowGrid => {
    const dailyGrid = buildCashFlowGrid({
      commitments: shownCommitments, entries: cashFlow.entriesService.records, cashBanks: laneCashBanks,
      lane, today: cashFlow.today, horizonDays: isMonthly ? CASH_FLOW_YEAR_DAYS : CASH_FLOW_HORIZON_DAYS,
      usdSellRate: cashFlow.usdSellRate || 1000,
    })
    return isMonthly ? monthlyGrid(dailyGrid) : dailyGrid
  }
  const grid = $derived(gridOf(viewOptions.lane))
  // The chart compares every lane, whichever one the table shows.
  const chartLanes = $derived(lanes.map((lane) => ({ name: lane.name, closing: gridOf(lane.id).closing })))
  const negativeDay = $derived(firstNegativeDay(grid))
  const columnTitle = (day: number) => isMonthly ? formatPeriod(periodOfDay(day)) : formatTime(day, 'd-m-Y') as string

  const warnings = $derived(rankWarnings([...cashFlow.serverWarnings, ...cashFlow.documentWarnings, ...cashFlow.forecastWarnings, ...pageWarnings({
    cashBanks: laneCashBanks, entries: cashFlow.entriesService.records, today: cashFlow.today,
    usdSellRate: cashFlow.usdSellRate,
    usesDollars: usesDollars([...cashFlow.commitments, ...cashFlow.cashBanksService.Cajas].map((record) => record.CurrencyType)),
  })]))
  const highWarningsCount = $derived(warnings.filter((warning) => warning.severity === 'high').length)

  const streamName = (streamID: number) => streamID === PLANNED_STREAM_ID
    ? 'Planned|Planificado' : CASH_FLOW_STREAMS.find((stream) => stream.id === streamID)?.name || ''

  // A cell's frame items stand for documents: they are swapped for them once loaded.
  const openDrilldown = async (row: ICashFlowRow, dayIndex: number) => {
    const items = row.cells[dayIndex].items
    const opened = { title: `${tr(streamName(row.streamID))} · ${columnTitle(grid.days[dayIndex])}`, items, isLoading: true }
    drilldown = opened
    ui.openSideLayer(DRILLDOWN_LAYER_ID)
    try {
      const loadedItems = await cashFlow.drilldownItems(items, viewOptions.lane === LANE_TOTAL ? undefined : viewOptions.lane)
      if (drilldown?.title === opened.title) drilldown = { ...opened, items: loadedItems, isLoading: false }
    } catch (error) {
      console.error('[cash-flow] drill-down failed', error)
      if (drilldown?.title === opened.title) drilldown = { ...opened, isLoading: false }
    }
  }

  type IDrilldownItem = ICashFlowCell['items'][number]
  const drilldownColumns: ITableColumn<IDrilldownItem>[] = [
    { header: 'Record|Registro', css: 'px-6', getValue: (item) => tr(item.label) },
    { header: 'Date|Fecha', headerCss: 'w-110', css: 'px-6', getValue: (item) => formatTime(item.date, 'd-m-Y') as string },
    { header: 'Due|Vence', headerCss: 'w-110', css: 'px-6',
      getValue: (item) => item.dueDate ? formatTime(item.dueDate, 'd-m-Y') as string : '' },
    { header: 'Layer|Capa', headerCss: 'w-110', css: 'px-6', getValue: (item) => tr(CASH_FLOW_LAYER_NAMES[item.layer] || '') },
    { header: 'Amount S/|Monto S/', headerCss: 'w-120', css: 'px-6 text-right',
      setCellCss: (item) => item.amountPEN < 0 ? 'text-red-fg' : undefined,
      getValue: (item) => formatN(item.amountPEN / 100, 2) },
  ]
</script>

<Page title="Cash Flow|Flujo de Caja" options={[
  { id: VIEW_DAILY, name: 'Daily 13 weeks|Diario 13 semanas' }, { id: VIEW_MONTHLY, name: 'Monthly 12 months|Mensual 12 meses' },
  { id: VIEW_PLANNED, name: 'Planned|Planificado' },
]}>
  {#if ui.state.pageOptionSelected === VIEW_PLANNED}
    <PlannedEntriesTable entriesService={cashFlow.entriesService} today={cashFlow.today}
      siteOptions={lanes.filter((lane) => lane.id !== LANE_TOTAL && lane.id !== LANE_DETRACTIONS).map((lane) => ({ id: lane.id, name: tr(lane.name) }))} />
  {:else}
    <div class="flex flex-wrap items-center gap-10 mb-10">
      <SearchSelect bind:saveOn={viewOptions} save="lane" options={lanes.map((lane) => ({ ...lane, name: tr(lane.name) }))}
        keyId="id" keyName="name" css="w-260" notEmpty={true} label="" placeholder="Site|Sede" />
      <Checkbox bind:saveOn={viewOptions} save="includeForecast" label="Include forecast|Incluir pronóstico" />
      <Button color={highWarningsCount ? 'red' : 'blue'} mode="subtle" icon="icon-[fa--exclamation-triangle]"
        name={`${warnings.length} ${tr('warnings|alertas')}`} label="Opens what the forecast had to assume."
        onClick={() => ui.openSideLayer(WARNINGS_LAYER_ID)} />
      <Button mode="subtle" icon="icon-[fa--refresh]" label="Recomputes the commitments and the forecast." onClick={() => cashFlow.load()} />
      {#if cashFlow.forecastStatus === 'loading' || cashFlow.forecastStatus === 'fitting'}
        <span class="text-fg-muted">{tr('Forecasting...|Pronosticando...')}</span>
      {:else if cashFlow.forecastStatus === 'error'}
        <span class="text-red-fg">{tr('Forecast failed|El pronóstico falló')}: {cashFlow.forecastError}</span>
      {/if}
      <div class="ml-auto {negativeDay ? 'text-red-fg ff-bold' : 'text-fg-muted'}">
        {#if cashFlow.isLoading}
          {tr('Computing...|Calculando...')}
        {:else if cashFlow.loadError}
          <span class="text-red-fg">{cashFlow.loadError}</span>
        {:else if negativeDay}
          {tr('Cash runs short on|La caja queda en negativo el')} {columnTitle(negativeDay)}
        {:else if isMonthly}
          {tr('Cash stays positive for 12 months|La caja se mantiene positiva 12 meses')}
        {:else}
          {tr('Cash stays positive for 13 weeks|La caja se mantiene positiva 13 semanas')}
        {/if}
      </div>
    </div>
    <CashFlowChart days={grid.days} lanes={chartLanes} {isMonthly} />
    <div class="mt-12">
      <CashFlowTable {grid} {isMonthly} onCellClick={openDrilldown} />
    </div>
  {/if}

  <Layer type="side" id={WARNINGS_LAYER_ID} width={640} title={tr('Forecast warnings|Alertas del pronóstico')} titleCss="h2 ff-bold">
    <CashFlowWarnings {warnings} />
  </Layer>

  <Layer type="side" id={DRILLDOWN_LAYER_ID} width={820} titleCss="h2 ff-bold" title={drilldown?.title || ''}
    onClose={() => { drilldown = null }}>
    {#if drilldown}
      {#if drilldown.isLoading}
        <div class="text-fg-muted mt-8">{tr('Loading the documents...|Cargando los documentos...')}</div>
      {/if}
      <VTable columns={drilldownColumns} data={drilldown.items} disableVirtualizer={true}
        estimateSize={38} css="w-full mt-8" />
    {/if}
  </Layer>
</Page>
