<script lang="ts">
  import { onDestroy } from 'svelte'
  import Page from '#domain/Page.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import FilterInput from '#components/form/FilterInput.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import NumericSlider from '#components/form/NumericSlider.svelte'
  import { CellSimpleChart, ChartCanvas } from '@genix/ui/charts'
  import { useUI } from '@genix/ui'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime } from '#libs/helpers.ts'
  import {
    DEFAULT_FORECAST_PARAMS, FORECAST_PARAM_SLIDERS, HOLDOUT_DAYS, MIN_FIT_DAYS, forecastChartOf, rowsMergedByProduct,
    forecastConfidence, LOW_CONFIDENCE, PROJECTION_WEEKS, projectionWeeksInMoney,
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

  const filteredRows = $derived.by(() => {
    const words = filterText.toLowerCase().split(' ').filter((word) => word.length > 0)
    return forecastState.rows.filter((row) => {
      if (siteFilter.SiteID && row.siteID !== siteFilter.SiteID) return false
      const name = productName(row.productID).toLowerCase()
      return words.every((word) => name.includes(word))
    })
  })

  // With every site, a product shows as one row adding up its sites.
  const tableRows = $derived(siteFilter.SiteID ? filteredRows : rowsMergedByProduct(filteredRows))

  // The chart follows the selected product (its filtered sites), or adds up every row the filters leave.
  const selectedRow = $derived(tableRows.find((row) => row.key === selectedRowKey))
  const chartRows = $derived(selectedRow ? filteredRows.filter((row) => row.productID === selectedRow.productID) : filteredRows)
  const seriesByKey = $derived(new Map(forecastState.series.map((series) => [`${series.siteID}_${series.productID}`, series])))
  const chart = $derived(forecastChartOf(chartRows, seriesByKey, forecastState.window, forecastState.params.horizonDays))
  const productsCount = $derived(new Set(forecastState.rows.map((row) => row.productID)).size)

  const statusText = $derived.by(() => {
    switch (forecastState.status) {
      case 'loading': return tr('Reading sales and stock…|Leyendo ventas y stock…')
      case 'fitting': return tr(`Fitting ${forecastState.series.length} series…|Ajustando ${forecastState.series.length} series…`)
      case 'ready': return tr(`${productsCount} products calculated|${productsCount} productos calculados`)
      case 'empty': return tr('No sales in the history.|No hay ventas en la historia.')
      case 'short': return tr(`Sales recorded for less than ${HOLDOUT_DAYS + MIN_FIT_DAYS} days: not enough to forecast.|Ventas registradas por menos de ${HOLDOUT_DAYS + MIN_FIT_DAYS} días: no alcanza para pronosticar.`)
      case 'error': return forecastState.errorMessage
    }
  })

  const highWarningsCount = $derived(forecastState.warnings.filter((warning) => warning.severity === 'high').length)

  const formatRatio = (value: number) => Number.isFinite(value) ? formatN(value, 2) : '-'

  // Translated at render time, so a language switch reaches them.
  const modelNames: Record<IForecastRow['model'], string> = {
    HoltWinters: 'Holt-Winters',
    Tsb: 'Intermittent|Intermitente',
    Mixed: 'Mixed|Mixto',
  }

  // Model, scenario, WAPE and MASE are for whoever tunes the forecast; Confidence sums them up for everyone.
  let showStatisticalColumns = $state(false)
  let showCellBars = $state(true)
  // The projection in soles at each product's current list price; Stock and Sold stay in units.
  let showProjectionInMoney = $state(false)

  // The weeks a projection cell shows: units, or soles; undefined when the product has no price in soles.
  const displayedWeeks = (row: IForecastRow) => {
    if (!showProjectionInMoney) return row.projectionWeeks
    const unitPrice = forecastState.unitPriceInSoles(row.productID)
    return unitPrice === undefined ? undefined : projectionWeeksInMoney(row.projectionWeeks, unitPrice)
  }

  const columns: ITableColumn<IForecastRow>[] = $derived([
    { header: 'Product|Producto', css: 'px-6', getValue: (row) => productName(row.productID) },
    { header: 'Model|Modelo', headerCss: 'w-110', css: 'px-6', hidden: !showStatisticalColumns, getValue: (row) => tr(modelNames[row.model]) },
    { header: 'Scen.|Esc.', headerCss: 'w-56', css: 'px-6 text-center', hidden: !showStatisticalColumns,
      getValue: (row) => row.scenario === 'Mixed' ? tr('Mixed|Mixto') : row.scenario },
    { header: 'Stock|Stock', headerCss: 'w-80', css: 'px-6 text-right', getValue: (row) => formatN(row.stockToday, 1),
      setCellCss: (row) => row.emptyShelfDays > 0 ? 'text-red-600' : undefined },
    { header: `Sold ${HOLDOUT_DAYS}d|Vendido ${HOLDOUT_DAYS}d`, headerCss: 'w-96', css: 'px-6 text-right', getValue: (row) => formatN(row.lastSales, 0) },
    // One subcolumn per week, headed by its first day, drawn by the cellRenderer snippet as 7 daily bars under the week's total.
    // min-h-24! overrides the VTable's 36px header rows, so the two stacked header rows stay as tall as a single one.
    { header: showProjectionInMoney ? '4-Week Projection (S/)|Proyección 4 Semanas (S/)' : '4-Week Projection|Proyección 4 Semanas', headerInnerCss: 'min-h-24! text-[14px]',
      subcols: Array.from({ length: PROJECTION_WEEKS }, (_, weekIndex) => ({
        id: `${PROJECTION_WEEK_COLUMN}${weekIndex}`, headerCss: 'w-80', headerInnerCss: 'min-h-24! text-[14px]', css: 'px-2',
        header: () => String(formatTime(forecastState.window.lastDay + 1 + weekIndex * 7, 'M-d') || ''),
      })) },
    { header: 'Confidence|Confianza', headerCss: 'w-96', css: 'px-6 text-right', getValue: (row) => formatRatio(forecastConfidence(row.wape, row.mase)),
      setCellCss: (row) => forecastConfidence(row.wape, row.mase) < LOW_CONFIDENCE ? 'text-red-600' : undefined },
    { header: 'WAPE|WAPE', headerCss: 'w-70', css: 'px-6 text-right', hidden: !showStatisticalColumns, getValue: (row) => formatRatio(row.wape) },
    { header: 'MASE|MASE', headerCss: 'w-70', css: 'px-6 text-right', hidden: !showStatisticalColumns, getValue: (row) => formatRatio(row.mase),
      setCellCss: (row) => row.worseThanNaive ? 'text-red-600' : undefined },
  ])

  const PROJECTION_WEEK_COLUMN = 'projection_week_'

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

<Page title="Sales Forecast|Pronóstico de Ventas" fixedFullHeight={true} noPadding={true}>
  <Layer type="content">
    <!-- Explicit viewport height: the Layer wrapper has none, and the aside needs a bound to scroll on its own. -->
    <div class="flex h-[calc(100vh-var(--header-height))] min-h-0 gap-14">
      <!-- Parameters: each slider commits on release and runs the forecast again.
           White panel flush against the side menu, so the page carries no padding. -->
      <aside class="flex w-[22%] min-w-240 shrink-0 flex-col overflow-y-auto border-r border-line bg-surface px-12 pt-10" aria-label="Forecast parameters">
        <Checkbox label="Show statistical indicators|Mostrar Indicadores Estadísticos" css="mb-8"
          checked={showStatisticalColumns} onToggle={(isChecked) => { showStatisticalColumns = isChecked }} />
        <Checkbox label="Show bars in cells|Mostrar barras en celdas" css="mb-8"
          checked={showCellBars} onToggle={(isChecked) => { showCellBars = isChecked }} />
        <Checkbox label="Calculate in money|Calcular en dinero" css="mb-14"
          checked={showProjectionInMoney} onToggle={(isChecked) => { showProjectionInMoney = isChecked }} />
        {#each FORECAST_PARAM_SLIDERS as slider (slider.key)}
          <div class="mb-16 flex flex-col items-center">
            <NumericSlider bind:saveOn={forecastState.params} save={slider.key}
              min={slider.min} max={slider.max} step={slider.step} suffix={slider.suffix || ''}
              label={slider.label} labelMaxLength={26} referenceWidth={200}
              onChange={() => forecastState.applyParams()} />
            <div class="mt-2 text-[14px] leading-snug text-fg-muted">{tr(slider.help)}</div>
          </div>
        {/each}
        <Button name="Restore defaults|Restaurar valores" icon="icon-[fa--undo]" mode="subtle" color="blue" css="mb-12 self-center"
          onClick={() => { forecastState.params = { ...DEFAULT_FORECAST_PARAMS }; forecastState.applyParams() }} />
      </aside>

      <section class="flex min-w-0 flex-1 flex-col pt-10 pr-10">
        <div class="mb-10 flex flex-wrap items-end gap-10">
          <SearchSelect saveOn={siteFilter} save="SiteID" css="w-240" label="Site|Sede" labelPosition="side"
            options={siteOptions} keyId="ID" keyName="Name" placeholder={tr('All sites|Todas las sedes')} />
          <FilterInput bind:value={filterText} css="w-240" placeholder={tr('Product…|Producto…')} />
          <div class="flex-1 self-center text-fg-muted" class:text-red-600={forecastState.status === 'error'}>{statusText}</div>
          <Button name={tr(`Warnings (${forecastState.warnings.length})|Advertencias (${forecastState.warnings.length})`)}
            icon="icon-[fa--exclamation-triangle]" color={highWarningsCount > 0 ? 'red' : 'orange'} mode="subtle"
            onClick={() => ui.openSideLayer(WARNINGS_LAYER_ID)} />
        </div>

        {#if selectedRow}
          <div class="mb-4 flex items-center gap-10">
            <div class="ff-bold text-fg-soft">{productName(selectedRow.productID)}</div>
            <Button name="Show all|Ver todos" height="tiny" mode="minimal" color="blue" onClick={() => { selectedRowKey = '' }} />
          </div>
        {/if}
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
          <VTable css="w-full" maxHeight="100%" columns={columns} data={tableRows}
            emptyMessage={tr('No series to show.|No hay series para mostrar.')}
            selected={selectedRow} isSelected={(row, selected) => typeof selected === 'object' && row.key === selected.key}
            onRowClick={(row) => { selectedRowKey = row.key === selectedRowKey ? '' : row.key }}>
            {#snippet cellRenderer(row, column)}
              {@const weeks = displayedWeeks(row)}
              {@const week = weeks?.[Number(String(column.id).slice(PROJECTION_WEEK_COLUMN.length))]}
              {#if !weeks}
                <div class="text-center text-fg-muted" title={tr('No price in soles|Sin precio en soles')}>-</div>
              {:else if week && week.days.length > 0}
                <!-- Faded bars as the cell's background; the week's total reads on top of them.
                     The row's busiest day tops the scale, so its 4 weeks compare at a glance. -->
                <div class="relative flex h-28 justify-center">
                  {#if showCellBars}
                    <div class="h-full opacity-35">
                      <CellSimpleChart values={week.days} maxValue={Math.max(...weeks.flatMap((oneWeek) => oneWeek.days))}
                        barWidth={7} barGap={2} barColor="#f59e0b" />
                    </div>
                  {/if}
                  <span class="absolute inset-0 flex items-center justify-center ff-bold">{formatN(week.total, showProjectionInMoney ? 0 : 1)}</span>
                </div>
              {/if}
            {/snippet}
          </VTable>
        </div>
      </section>
    </div>
  </Layer>

  <Layer type="side" id={WARNINGS_LAYER_ID} width={620} floatOverContent={true} css="px-8 py-8 md:px-14 md:py-10"
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
