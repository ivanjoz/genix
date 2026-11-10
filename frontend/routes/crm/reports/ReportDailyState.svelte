<script lang="ts">
  import { Chart, type IChartSeries } from '#components/charts/index.ts'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN, formatTime, todayUnixDay } from '#libs/helpers.ts'
  import { activeStagesOf } from '../pipeline/pipeline.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import { dailyStageStates, periodFirstDay, periodLastDay } from './reports.ts'

  // R2: how many open opportunities (or how much) sat in each stage at the end of each day.
  // A snapshot, so a deal that moves back and forth is never counted twice.

  interface Props { report: CrmReportsState }
  const { report }: Props = $props()

  interface StageRow {
    stageID: number
    name: string
    start: { count: number, amount: number }
    end: { count: number, amount: number }
  }

  const today = todayUnixDay()
  const stages = $derived(activeStagesOf(report.selectedPipeline))
  const fromDay = $derived(periodFirstDay(report.periodFrom))
  const toDay = $derived(Math.min(periodLastDay(report.periodTo), today))
  const states = $derived(fromDay > today ? [] : dailyStageStates(
    report.rowsToToday, report.selectedPipelineID, report.openNow, fromDay, toDay, today, report.ownerIDs,
  ))

  const measure = (total: { count: number, amount: number } | undefined) =>
    report.showAmounts ? (total?.amount ?? 0) / 100 : (total?.count ?? 0)
  const formatValue = (value: number) => report.showAmounts ? formatN(value, 2) : String(value)

  const chartSeries = $derived(stages.map((stage): IChartSeries => ({
    name: stage.Name, color: stage.Color || '#94a3b8',
    values: states.map((state) => measure(state.byStage.get(stage.ID))),
  })))

  const empty = { count: 0, amount: 0 }
  const stageRows = $derived(stages.map((stage): StageRow => ({
    stageID: stage.ID, name: stage.Name,
    start: states[0]?.byStage.get(stage.ID) ?? empty,
    end: states.at(-1)?.byStage.get(stage.ID) ?? empty,
  })))

  const tableColumns: ITableColumn<StageRow>[] = [
    { header: 'Stage|Etapa', css: 'px-6', highlight: true, getValue: (row) => row.name },
    { header: 'At the start|Al inicio', css: 'px-6 text-right ff-mono', getValue: (row) => formatValue(measure(row.start)) },
    { header: 'At the end|Al final', css: 'px-6 text-right ff-mono', getValue: (row) => formatValue(measure(row.end)) },
    { header: 'Change|Variación', css: 'px-6 text-right ff-mono',
      getValue: (row) => formatValue(measure(row.end) - measure(row.start)),
      setCellCss: (row) => measure(row.end) < measure(row.start) ? 'text-red-500' : undefined },
  ]
</script>

<div class="flex flex-col gap-16" aria-label="Daily pipeline state">
  <div class="text-fg-muted">
    {tr('Open opportunities in each stage at the end of each day.|Oportunidades abiertas en cada etapa al cierre de cada día.')}
  </div>
  {#if !states.length}
    <div class="text-fg-muted p-16">{tr('The selected months are in the future.|Los meses elegidos están en el futuro.')}</div>
  {:else}
    <Chart type="bar" stacked={true} series={chartSeries} labels={states.map((state) => state.day)} height={300}
      formatLabel={(unixDay) => formatTime(Number(unixDay), 'd-M') as string}
      formatTooltipLabel={(unixDay) => formatTime(Number(unixDay), 'd-M-Y') as string}
      formatValue={(value) => formatValue(value)} onlyIntegers={!report.showAmounts} />
    <VTable columns={tableColumns} data={stageRows} css="w-full" />
  {/if}
</div>
