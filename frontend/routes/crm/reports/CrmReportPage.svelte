<script lang="ts" module>
  import type { Component } from 'svelte'
  import type { CrmReportsState } from './reports.svelte.ts'

  export interface ICrmReportView {
    label: string
    /** What the view measures, shown on the right of the seller filter. */
    description: string
    component: Component<{ report: CrmReportsState }>
  }
</script>

<script lang="ts">
  import { untrack } from 'svelte'
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { currentShortYear, getCrmReportsState } from './reports.svelte.ts'
  import { addMonthsToPeriod, afterStateOf, beforeStateOf, periodFirstDay, periodOfUnixDay, periodsBetween } from './reports.ts'

  // Shell of the CRM report pages: the filters every report shares and a strip with the page's views.
  interface Props {
    title: string
    views: ICrmReportView[]
  }
  const { title, views }: Props = $props()

  const report = getCrmReportsState()
  let viewIndex = $state(0)
  const selectedView = $derived(views[viewIndex])

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

  // Reload when the page opens or the pipeline or start year changes; periods, sellers and amounts
  // only re-filter. Opening another report page reloads too: cached groups only fetch what changed.
  $effect(() => {
    if (!report.selectedPipelineID) return
    void report.yearFrom
    untrack(() => report.load())
  })
</script>

<Page {title}>
  <!-- View selector on the left, a separator, then the filters, all on one line. -->
  <div class="flex items-end gap-12 flex-wrap mb-12" aria-label="Report filters">
    {#if report.reportPipelineIDs.length}
      <OptionsStrip selected={viewIndex} useMobileGrid={true}
        options={views.map((view, index) => [index, tr(view.label)])}
        onSelect={(option) => { viewIndex = option[0] as number }} />
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
      {tr(selectedView.description)}
    </div>
  </div>

  {#if !report.reportPipelineIDs.length}
    <div class="text-fg-muted p-16">
      {tr('You have no pipelines to report on. Ask an administrator for pipeline membership or the full-report access.|No tiene pipelines para reportar. Pida a un administrador que lo agregue a un pipeline o el acceso de reporte completo.')}
    </div>
  {:else}
    <selectedView.component {report} />
  {/if}
</Page>
