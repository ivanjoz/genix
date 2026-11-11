<script lang="ts">
  import { untrack } from 'svelte'
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import ReportDailyState from './ReportDailyState.svelte'
  import ReportFunnel from './ReportFunnel.svelte'
  import ReportLostReasons from './ReportLostReasons.svelte'
  import ReportNetProgress from './ReportNetProgress.svelte'
  import ReportScorecard from './ReportScorecard.svelte'
  import ReportWaterfall from './ReportWaterfall.svelte'
  import { CrmReportsState, currentShortYear } from './reports.svelte.ts'
  import { formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { addMonthsToPeriod, afterStateOf, beforeStateOf, periodFirstDay, periodOfUnixDay, periodsBetween } from './reports.ts'

  const VIEW_NET_PROGRESS = 1
  const VIEW_DAILY_STATE = 2
  const VIEW_WATERFALL = 3
  const VIEW_FUNNEL = 4
  const VIEW_SCORECARD = 5
  const VIEW_LOST_REASONS = 6
  const VIEW_OPTIONS = [
    [VIEW_NET_PROGRESS, 'Net progress|Avance neto'], [VIEW_DAILY_STATE, 'Daily state|Estado diario'],
    [VIEW_WATERFALL, 'Waterfall|Cascada'], [VIEW_FUNNEL, 'Funnel|Embudo'], [VIEW_SCORECARD, 'Sellers|Vendedores'],
    [VIEW_LOST_REASONS, 'Lost reasons|Motivos de pérdida'],
  ] as const
  // What each view measures, shown on the right of the seller filter.
  const VIEW_DESCRIPTIONS: Record<number, string> = {
    [VIEW_NET_PROGRESS]: 'Opportunities that reached each stage, net of steps back, losses and hand-overs. Negatives mean the seller lost ground.|Oportunidades que llegaron a cada etapa, netas de retrocesos, pérdidas y reasignaciones. Los negativos indican que el vendedor perdió terreno.',
    [VIEW_DAILY_STATE]: 'Open opportunities in each stage at the end of each day.|Oportunidades abiertas en cada etapa al cierre de cada día.',
    [VIEW_WATERFALL]: 'How the open pipeline went from the opening to the closing of the range: new, renegotiated, won, lost, reopened and handed over.|Cómo pasó el pipeline abierto de la apertura al cierre del rango: nuevas, renegociadas, ganadas, perdidas, reabiertas y reasignadas.',
    [VIEW_FUNNEL]: 'Of the opportunities that entered the pipeline in the range, how far they got, and how long they stayed in each stage.|De las oportunidades que entraron al pipeline en el rango, hasta dónde llegaron y cuánto estuvieron en cada etapa.',
    [VIEW_SCORECARD]: 'Win rate, won amount, sales cycle, velocity, activities and goal of each seller, with team subtotals.|Tasa de cierre, monto ganado, ciclo de venta, velocidad, actividades y meta de cada vendedor, con subtotales por equipo.',
    [VIEW_LOST_REASONS]: 'Opportunities still lost at the end of the range, by reason, seller and stage.|Oportunidades que siguen perdidas al final del rango, por motivo, vendedor y etapa.',
  }

  const report = new CrmReportsState()
  let view = $state(VIEW_NET_PROGRESS)

  // "From" spans January five years back to the current month; "To" reaches 12 months ahead
  // (open deals expected to close later). Newest first.
  const currentPeriod = periodOfUnixDay(todayUnixDay())
  const periodOption = (period: number) => ({ id: period, name: formatTime(periodFirstDay(period), 'M Y') as string })
  const fromOptions = periodsBetween((currentShortYear() - 5) * 100 + 1, currentPeriod).reverse().map(periodOption)
  const toOptions = periodsBetween((currentShortYear() - 5) * 100 + 1, addMonthsToPeriod(currentPeriod, 12)).reverse().map(periodOption)

  // Pipelines the user may report on (active ones only, from the cached pipelines list).
  const pipelineOptions = $derived(report.reportPipelineIDs.map((pipelineID) => ({
    id: pipelineID, name: report.pipelines.records.find((pipeline) => pipeline.ID === pipelineID)?.Name ?? `Pipeline #${pipelineID}`,
  })))
  // The seller filter lists everyone who owned an opportunity of the pipeline in the loaded rows.
  const ownerOptions = $derived([...new Set(report.rowsToToday.flatMap((row) => [beforeStateOf(row)?.ownerID ?? 0, afterStateOf(row).ownerID]))]
    .filter(Boolean).map((ownerID) => ({ id: ownerID, name: report.ownerName(ownerID) })))

  $effect(() => {
    void report.reportPipelineIDs
    untrack(() => report.selectDefaultPipeline())
  })

  // Reload when the pipeline or the start year changes; periods, sellers and amounts only re-filter.
  $effect(() => {
    if (!report.selectedPipelineID) return
    void report.yearFrom
    untrack(() => report.load())
  })
</script>

<Page title="CRM Reports|Reportes CRM">
  <!-- View selector on the left, a separator, then the filters, all on one line. -->
  <div class="flex items-end gap-12 flex-wrap mb-12" aria-label="Report filters">
    {#if report.reportPipelineIDs.length}
      <OptionsStrip selected={view} useMobileGrid={true}
        options={VIEW_OPTIONS.map(([viewID, label]) => [viewID, tr(label)])}
        onSelect={(option) => { view = option[0] as number }} />
      <div class="self-stretch w-1 bg-gray-300" aria-hidden="true"></div>
    {/if}
    <SearchSelect options={pipelineOptions} keyId="id" keyName="name" label="Pipeline" css="w-240"
      selected={report.selectedPipelineID} onChange={(option) => { report.selectedPipelineID = option.id }} />
    <SearchSelect options={fromOptions} keyId="id" keyName="name" label="From|Desde" css="w-140"
      selected={report.periodFrom} onChange={(option) => { report.periodFrom = Math.min(option.id, report.periodTo) }} />
    <SearchSelect options={toOptions} keyId="id" keyName="name" label="To|Hasta" css="w-140"
      selected={report.periodTo} onChange={(option) => { report.periodTo = Math.max(option.id, report.periodFrom) }} />
    <Checkbox label="Show amounts|Ver montos" checked={report.showAmounts}
      onToggle={(isChecked) => { report.showAmounts = isChecked }} />
    {#if report.isLoading}
      <span class="text-fg-muted">{tr('Loading...|Cargando...')}</span>
    {/if}
  </div>

  <!-- Second row: seller filter on the left, what the current view measures on the right. -->
  <div class="flex items-center justify-between gap-16 mb-12">
    {#if ownerOptions.length}
      <AvatarToggleGroup options={ownerOptions} bind:selected={report.ownerIDs} label="Filter by seller|Filtrar por vendedor"
        optionLabel={(ownerName) => `Show only ${ownerName}|Mostrar solo a ${ownerName}`} />
    {/if}
    <div class="ml-auto max-w-600 text-right text-fg-muted leading-tight">
      {tr(VIEW_DESCRIPTIONS[view])}
    </div>
  </div>

  {#if !report.reportPipelineIDs.length}
    <div class="text-fg-muted p-16">
      {tr('You have no pipelines to report on. Ask an administrator for pipeline membership or the full-report access.|No tiene pipelines para reportar. Pida a un administrador que lo agregue a un pipeline o el acceso de reporte completo.')}
    </div>
  {:else}
    {#if view === VIEW_NET_PROGRESS}
      <ReportNetProgress {report} />
    {:else if view === VIEW_DAILY_STATE}
      <ReportDailyState {report} />
    {:else if view === VIEW_WATERFALL}
      <ReportWaterfall {report} />
    {:else if view === VIEW_FUNNEL}
      <ReportFunnel {report} />
    {:else if view === VIEW_SCORECARD}
      <ReportScorecard {report} />
    {:else if view === VIEW_LOST_REASONS}
      <ReportLostReasons {report} />
    {/if}
  {/if}
</Page>
