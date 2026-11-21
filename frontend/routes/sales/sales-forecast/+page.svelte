<script lang="ts">
  import { onDestroy } from 'svelte'
  import Page from '#domain/Page.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import Button from '#components/buttons/Button.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import NumericSlider from '#components/form/NumericSlider.svelte'
  import { ChartCanvas } from '@genix/ui/charts'
  import { useUI } from '@genix/ui'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import {
    DEFAULT_FORECAST_PARAMS, FORECAST_PARAM_GROUPS, HOLDOUT_DAYS, MIN_FIT_DAYS, forecastChartOf,
    type IForecastRow, type IForecastWarning,
  } from './sales-forecast.ts'
  import { SalesForecastState } from './sales-forecast.svelte.ts'

  const ui = useUI()
  const WARNINGS_LAYER_ID = 1
  const forecastState = new SalesForecastState()
  forecastState.loadInputs()
  onDestroy(() => forecastState.destroy())

  // 0 = every site.
  let siteFilter = $state({ SiteID: 0 })
  let filterText = $state('')
  let selectedRowKey = $state('')

  const siteOptions = $derived(forecastState.warehouses.Sedes.filter((site) => site.ss > 0))

  const productName = (productID: number) => forecastState.products.recordsMap.get(productID)?.Name || `#${productID}`
  const siteName = (siteID: number) => forecastState.warehouses.SedesMap.get(siteID)?.Name || `#${siteID}`

  const filteredRows = $derived.by(() => {
    const words = filterText.toLowerCase().split(' ').filter((word) => word.length > 0)
    return forecastState.rows.filter((row) => {
      if (siteFilter.SiteID && row.siteID !== siteFilter.SiteID) return false
      const name = productName(row.productID).toLowerCase()
      return words.every((word) => name.includes(word))
    })
  })

  // The chart follows the selected row, or adds up every row the filters leave.
  const selectedRow = $derived(forecastState.rows.find((row) => row.key === selectedRowKey))
  const chartRows = $derived(selectedRow ? [selectedRow] : filteredRows)
  const seriesByKey = $derived(new Map(forecastState.series.map((series) => [`${series.siteID}_${series.productID}`, series])))
  const chart = $derived(forecastChartOf(chartRows, seriesByKey, forecastState.window, forecastState.params.horizonDays))
  const chartTitle = $derived(selectedRow
    ? `${productName(selectedRow.productID)} — ${siteName(selectedRow.siteID)}`
    : tr(`${filteredRows.length} products added up|${filteredRows.length} productos sumados`))

  const statusText = $derived.by(() => {
    switch (forecastState.status) {
      case 'loading': return tr('Reading sales and stock…|Leyendo ventas y stock…')
      case 'fitting': return tr(`Fitting ${forecastState.series.length} series…|Ajustando ${forecastState.series.length} series…`)
      case 'ready': return tr(`${forecastState.rows.length} series · fitted in ${(forecastState.elapsedMs / 1000).toFixed(1)} s|${forecastState.rows.length} series · ajustado en ${(forecastState.elapsedMs / 1000).toFixed(1)} s`)
      case 'empty': return tr('No sales in the history.|No hay ventas en la historia.')
      case 'short': return tr(`Sales recorded for less than ${HOLDOUT_DAYS + MIN_FIT_DAYS} days: not enough to forecast.|Ventas registradas por menos de ${HOLDOUT_DAYS + MIN_FIT_DAYS} días: no alcanza para pronosticar.`)
      case 'error': return forecastState.errorMessage
    }
  })

  const highWarningsCount = $derived(forecastState.warnings.filter((warning) => warning.severity === 'high').length)

  const formatRatio = (value: number) => Number.isFinite(value) ? formatN(value, 2) : '-'

  const columns: ITableColumn<IForecastRow>[] = [
    { header: 'Product|Producto', css: 'px-6', getValue: (row) => productName(row.productID) },
    { header: 'Site|Sede', headerCss: 'w-120', css: 'px-6', getValue: (row) => siteName(row.siteID) },
    { header: 'Model|Modelo', headerCss: 'w-110', css: 'px-6', getValue: (row) => row.model === 'Tsb' ? tr('Intermittent|Intermitente') : 'Holt-Winters' },
    { header: 'Scen.|Esc.', headerCss: 'w-56', css: 'px-6 text-center', getValue: (row) => row.scenario },
    { header: 'Stock|Stock', headerCss: 'w-80', css: 'px-6 text-right', getValue: (row) => formatN(row.stockToday, 1),
      setCellCss: (row) => row.emptyShelfDays > 0 ? 'text-red-600' : undefined },
    { header: `Sold ${HOLDOUT_DAYS}d|Vendido ${HOLDOUT_DAYS}d`, headerCss: 'w-96', css: 'px-6 text-right', getValue: (row) => formatN(row.lastSales, 0) },
    { header: 'Next 7d|Próx. 7d', headerCss: 'w-90', css: 'px-6 text-right', getValue: (row) => formatN(row.next7, 1) },
    { header: () => tr(`Next ${forecastState.params.horizonDays}d|Próx. ${forecastState.params.horizonDays}d`), headerCss: 'w-96', css: 'px-6 text-right ff-bold',
      getValue: (row) => formatN(row.horizonTotal, 1) },
    { header: 'WAPE|WAPE', headerCss: 'w-70', css: 'px-6 text-right', getValue: (row) => formatRatio(row.wape) },
    { header: 'MASE|MASE', headerCss: 'w-70', css: 'px-6 text-right', getValue: (row) => formatRatio(row.mase),
      setCellCss: (row) => row.worseThanNaive ? 'text-red-600' : undefined },
  ]

  const warningGroupNames: Record<IForecastWarning['group'], string> = {
    history: 'History|Historia',
    data: 'Data|Datos',
    calendar: 'Calendar|Calendario',
    stockout: 'Stockouts|Quiebres',
    accuracy: 'Accuracy|Precisión',
  }
  const severityCss: Record<IForecastWarning['severity'], string> = {
    high: 'bg-red-500',
    medium: 'bg-orange-400',
    low: 'bg-slate-400',
  }
  // A warning lists at most this many products; the rest is counted.
  const MAX_WARNING_ITEMS = 60
</script>

<Page title="Sales Forecast|Pronóstico de Ventas" fixedFullHeight={true}>
  <Layer type="content">
    <div class="flex h-full min-h-0 gap-14">
      <!-- Parameters: each slider commits on release and runs the forecast again. -->
      <aside class="flex w-[22%] min-w-240 shrink-0 flex-col overflow-y-auto border-r border-line pr-10" aria-label="Forecast parameters">
        {#each FORECAST_PARAM_GROUPS as paramGroup (paramGroup.title)}
          <div class="mb-16">
            <div class="ff-bold mb-8 text-fg-soft">{tr(paramGroup.title)}</div>
            {#each paramGroup.sliders as slider (slider.key)}
              <div class="mb-12 flex flex-col items-center">
                <NumericSlider bind:saveOn={forecastState.params} save={slider.key}
                  min={slider.min} max={slider.max} step={slider.step} suffix={slider.suffix || ''}
                  label={slider.label} labelMaxLength={26} referenceWidth={200}
                  onChange={() => forecastState.applyParams()} />
                <div class="mt-2 text-[14px] leading-snug text-fg-muted">{tr(slider.help)}</div>
              </div>
            {/each}
          </div>
        {/each}
        <Button name="Restore defaults|Restaurar valores" icon="icon-[fa--undo]" mode="subtle" color="blue" css="mb-12 self-center"
          onClick={() => { forecastState.params = { ...DEFAULT_FORECAST_PARAMS }; forecastState.applyParams() }} />
      </aside>

      <section class="flex min-w-0 flex-1 flex-col">
        <div class="mb-10 flex flex-wrap items-end gap-10">
          <SearchSelect saveOn={siteFilter} save="SiteID" css="w-200" label="Site|Sede"
            options={siteOptions} keyId="ID" keyName="Name" placeholder={tr('All sites|Todas las sedes')} />
          <FilterInput bind:value={filterText} css="w-240" placeholder={tr('Product…|Producto…')} />
          <div class="flex-1 self-center text-fg-muted" class:text-red-600={forecastState.status === 'error'}>{statusText}</div>
          <Button name={tr(`Warnings (${forecastState.warnings.length})|Advertencias (${forecastState.warnings.length})`)}
            icon="icon-[fa--exclamation-triangle]" color={highWarningsCount > 0 ? 'red' : 'orange'} mode="subtle"
            onClick={() => ui.openSideLayer(WARNINGS_LAYER_ID)} />
        </div>

        <div class="mb-4 flex items-center gap-10">
          <div class="ff-bold text-fg-soft">{chartTitle}</div>
          {#if selectedRow}
            <Button name="Show all|Ver todos" height="tiny" mode="minimal" color="blue" onClick={() => { selectedRowKey = '' }} />
          {/if}
        </div>
        <ChartCanvas id="sales-forecast" height={240} className="mb-12 w-full shrink-0" useHtmlRendered={true} showTooltip={true} barMode="stacked"
          dateLabels={chart.days} dateLabelEvery={7}
          dateLabelFormatter={(day) => String(formatTime(Number(day), 'd-M') || '')}
          tooltipValueFormatter={(value) => formatN(value, 1)}
          data={[
            { name: tr('Sales|Ventas'), type: 'bar', values: chart.sales, color: '#3b82f6' },
            { name: tr('Forecast|Pronóstico'), type: 'bar', values: chart.forecast, color: '#f59e0b' },
            { name: tr('Stock|Stock'), type: 'line', values: chart.stock, color: '#10b981', lineWidth: 2,
              rightAxisFormatter: (value) => formatN(value, 0) },
          ]} />

        <div class="min-h-0 flex-1">
          <VTable css="w-full" maxHeight="100%" columns={columns} data={filteredRows}
            emptyMessage={tr('No series to show.|No hay series para mostrar.')}
            selected={selectedRow} isSelected={(row, selected) => typeof selected === 'object' && row.key === selected.key}
            onRowClick={(row) => { selectedRowKey = row.key === selectedRowKey ? '' : row.key }} />
        </div>
      </section>
    </div>
  </Layer>

  <Layer type="side" id={WARNINGS_LAYER_ID} width={620} css="px-8 py-8 md:px-14 md:py-10"
    title={tr('Forecast warnings|Advertencias del pronóstico')} titleCss="h2 ff-bold">
    {#if forecastState.warnings.length === 0}
      <div class="mt-8 text-fg-muted">{tr('Nothing to report: the data looks complete.|Nada que reportar: los datos se ven completos.')}</div>
    {/if}
    {#each forecastState.warnings as warning (warning.title)}
      <div class="mb-12 rounded-md border border-line p-10">
        <div class="flex items-center gap-8">
          <span class="h-10 w-10 shrink-0 rounded-full {severityCss[warning.severity]}"></span>
          <span class="ff-bold">{tr(warning.title)}</span>
          <span class="ml-auto text-[14px] text-fg-muted">{tr(warningGroupNames[warning.group])}</span>
        </div>
        <div class="mt-4 text-fg-soft">{tr(warning.detail)}</div>
        {#if warning.items.length > 0}
          <ul class="mt-6 max-h-200 overflow-y-auto pl-18 text-fg-muted">
            {#each warning.items.slice(0, MAX_WARNING_ITEMS) as item, itemIndex (itemIndex)}
              <li class="list-disc">{item}</li>
            {/each}
          </ul>
          {#if warning.items.length > MAX_WARNING_ITEMS}
            <div class="mt-4 text-fg-muted">{tr(`… and ${warning.items.length - MAX_WARNING_ITEMS} more|… y ${warning.items.length - MAX_WARNING_ITEMS} más`)}</div>
          {/if}
        {/if}
      </div>
    {/each}
  </Layer>
</Page>
