<script lang="ts">
  import { untrack } from 'svelte'
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import ReportDailyState from './ReportDailyState.svelte'
  import ReportNetProgress from './ReportNetProgress.svelte'
  import { CrmReportsState, currentShortYear } from './reports.svelte.ts'
  import { afterStateOf, beforeStateOf } from './reports.ts'

  const VIEW_NET_PROGRESS = 1
  const VIEW_DAILY_STATE = 2

  const report = new CrmReportsState()
  let view = $state(VIEW_NET_PROGRESS)

  const MONTH_NAMES = ['Jan|Ene', 'Feb|Feb', 'Mar|Mar', 'Apr|Abr', 'May|May', 'Jun|Jun', 'Jul|Jul', 'Aug|Ago', 'Sep|Set', 'Oct|Oct', 'Nov|Nov', 'Dec|Dic']
  const monthOptions = MONTH_NAMES.map((name, index) => ({ id: index + 1, name: tr(name) }))
  const yearOptions = Array.from({ length: 5 }, (_, index) => currentShortYear() - index)
    .map((year) => ({ id: year, name: String(2000 + year) }))

  // Pipelines the user may report on; a soft-deleted one (full-report access) keeps a generic name.
  const pipelineOptions = $derived(report.scope.PipelineIDs.map((pipelineID) => ({
    id: pipelineID, name: report.pipelines.records.find((pipeline) => pipeline.ID === pipelineID)?.Name ?? `Pipeline #${pipelineID}`,
  })))
  // The seller filter lists everyone who owned an opportunity of the pipeline in the loaded rows.
  const ownerOptions = $derived([...new Set(report.rowsToToday.flatMap((row) => [beforeStateOf(row)?.ownerID ?? 0, afterStateOf(row).ownerID]))]
    .filter(Boolean).map((ownerID) => ({ id: ownerID, name: report.ownerName(ownerID) })))

  report.loadScope()

  // Reload when the pipeline or the year changes; months, sellers and amounts only re-filter.
  $effect(() => {
    if (!report.selectedPipelineID) return
    void report.year
    untrack(() => report.load())
  })
</script>

<Page title="CRM Reports|Reportes CRM">
  <div class="flex items-end gap-12 flex-wrap mb-12" aria-label="Report filters">
    <SearchSelect options={pipelineOptions} keyId="id" keyName="name" label="Pipeline" css="w-240"
      selected={report.selectedPipelineID} onChange={(option) => { report.selectedPipelineID = option.id }} />
    <SearchSelect options={yearOptions} keyId="id" keyName="name" label="Year|Año" css="w-120"
      selected={report.year} onChange={(option) => { report.year = option.id }} />
    <SearchSelect options={monthOptions} keyId="id" keyName="name" label="From|Desde" css="w-120"
      selected={report.monthFrom} onChange={(option) => { report.monthFrom = Math.min(option.id, report.monthTo) }} />
    <SearchSelect options={monthOptions} keyId="id" keyName="name" label="To|Hasta" css="w-120"
      selected={report.monthTo} onChange={(option) => { report.monthTo = Math.max(option.id, report.monthFrom) }} />
    <Checkbox label="Show amounts|Ver montos" checked={report.showAmounts}
      onToggle={(isChecked) => { report.showAmounts = isChecked }} />
    {#if report.isLoading}
      <span class="text-fg-muted">{tr('Loading...|Cargando...')}</span>
    {/if}
  </div>

  {#if ownerOptions.length}
    <div class="mb-12">
      <AvatarToggleGroup options={ownerOptions} bind:selected={report.ownerIDs} label="Filter by seller|Filtrar por vendedor"
        optionLabel={(ownerName) => `Show only ${ownerName}|Mostrar solo a ${ownerName}`} />
    </div>
  {/if}

  {#if !report.scope.PipelineIDs.length}
    <div class="text-fg-muted p-16">
      {tr('You have no pipelines to report on. Ask an administrator for pipeline membership or the full-report access.|No tiene pipelines para reportar. Pida a un administrador que lo agregue a un pipeline o el acceso de reporte completo.')}
    </div>
  {:else}
    <OptionsStrip selected={view} useMobileGrid={true}
      options={[[VIEW_NET_PROGRESS, tr('Net progress|Avance neto')], [VIEW_DAILY_STATE, tr('Daily state|Estado diario')]]}
      onSelect={(option) => { view = option[0] as number }} />
    <div class="mt-12">
      {#if view === VIEW_NET_PROGRESS}
        <ReportNetProgress {report} />
      {:else if view === VIEW_DAILY_STATE}
        <ReportDailyState {report} />
      {/if}
    </div>
  {/if}
</Page>
