<script lang="ts">
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, todayUnixDay } from '#libs/helpers.ts'
  import { activeStagesOf } from '../pipeline/pipeline.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import { averageOf, funnelReport, medianOf, periodFirstDay, periodLastDay } from './reports.ts'

  // R4: of the opportunities that entered the pipeline in the range, how deep each one got (lost
  // deals count where they died), and how long finished stays lasted in each stage.

  interface Props { report: CrmReportsState }
  const { report }: Props = $props()

  const stages = $derived(activeStagesOf(report.selectedPipeline))
  const fromDay = $derived(periodFirstDay(report.periodFrom))
  const toDay = $derived(Math.min(periodLastDay(report.periodTo), todayUnixDay()))
  // The cohort's later history is needed too, so the rows run to today, not only the range.
  const funnel = $derived(funnelReport(report.rowsToToday, report.selectedPipelineID, fromDay, toDay, report.ownerIDs))

  const percentOf = (part: number, whole: number) => whole > 0 ? Math.round(part * 100 / whole) : 0
  // Columns are positions labelled with the stage now at that depth (E2).
  const positionLabel = (position: number) => stages[position - 1]?.Name ?? tr(`Stage ${position}|Etapa ${position}`)
  const funnelBars = $derived(funnel.reachedByPosition.map((reached, index) => ({
    label: positionLabel(index + 1),
    color: stages[index]?.Color || '#94a3b8',
    reached,
    ofEntered: percentOf(reached, funnel.entered),
    // Conversion from the position above: where the funnel leaks.
    fromPrevious: index === 0 ? 100 : percentOf(reached, funnel.reachedByPosition[index - 1]),
  })))

  interface StageTimeRow {
    stageID: number
    name: string
    stays: number[]
  }
  const stageTimeRows = $derived(stages.map((stage): StageTimeRow => ({
    stageID: stage.ID, name: stage.Name, stays: funnel.stayDaysByStage.get(stage.ID) ?? [],
  })))
  const days = (value: number) => formatN(value, 1)
  const stageTimeColumns: ITableColumn<StageTimeRow>[] = [
    { header: 'Stage|Etapa', css: 'px-6', highlight: true, getValue: (row) => row.name },
    { header: 'Finished stays|Estadías terminadas', css: 'px-6 text-right ff-mono', getValue: (row) => String(row.stays.length) },
    { header: 'Median days|Mediana días', css: 'px-6 text-right ff-mono', getValue: (row) => row.stays.length ? days(medianOf(row.stays)) : '—' },
    { header: 'Average days|Promedio días', css: 'px-6 text-right ff-mono', getValue: (row) => row.stays.length ? days(averageOf(row.stays)) : '—' },
    { header: 'Longest|Máximo', css: 'px-6 text-right ff-mono', getValue: (row) => row.stays.length ? days(Math.max(...row.stays)) : '—' },
  ]
</script>

<div class="flex flex-col gap-16" aria-label="Funnel and time in stage">
  {#if !funnel.entered}
    <div class="text-fg-muted p-16">{tr('No opportunities entered the pipeline in the selected months.|Ninguna oportunidad entró al pipeline en los meses elegidos.')}</div>
  {:else}
    <div class="flex gap-24 flex-wrap" aria-label="Cohort totals">
      <div><span class="text-fg-muted">{tr('Entered|Ingresaron')}</span> <span class="ff-bold">{funnel.entered}</span></div>
      <div><span class="text-fg-muted">{tr('Won|Ganadas')}</span> <span class="ff-bold">{funnel.won}</span>
        <span class="text-fg-muted">({percentOf(funnel.won, funnel.entered)}%)</span></div>
      <div><span class="text-fg-muted">{tr('Lost|Perdidas')}</span> <span class="ff-bold">{funnel.lost}</span>
        <span class="text-fg-muted">({percentOf(funnel.lost, funnel.entered)}%)</span></div>
      <div><span class="text-fg-muted">{tr('Still open or moved out|Abiertas o transferidas')}</span>
        <span class="ff-bold">{funnel.entered - funnel.won - funnel.lost}</span></div>
    </div>

    <div class="bg-white/80 rounded-lg p-12 flex flex-col gap-6" aria-label="Funnel chart">
      {#each funnelBars as bar, index (index)}
        <div class="grid grid-cols-[200px_1fr_160px] items-center gap-8 group">
          <span class="truncate">{bar.label}</span>
          <div class="relative h-18">
            <div class="absolute top-0 left-0 h-full rounded-[4px] group-hover:opacity-80" style="width: max({bar.ofEntered}%, 2px); background: {bar.color}"
              title="{bar.label}: {bar.reached} ({bar.ofEntered}%)"></div>
          </div>
          <span class="text-right ff-mono">{bar.reached} · {bar.ofEntered}%
            {#if index > 0}<span class="text-fg-muted"> ({bar.fromPrevious}%)</span>{/if}</span>
        </div>
      {/each}
      <div class="text-fg-muted text-right">{tr('% of entered · (conversion from the stage above)|% de las ingresadas · (conversión desde la etapa anterior)')}</div>
    </div>

    <VTable columns={stageTimeColumns} data={stageTimeRows} css="w-full" />
  {/if}
</div>
