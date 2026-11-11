<script lang="ts">
  import { Chart } from '#components/charts/index.ts'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import { activeStagesOf } from '../pipeline/pipeline.ts'
  import type { CrmReportsState } from './reports.svelte.ts'
  import { lostDealsInRange, paretoOf, type LostDeal, type ParetoRow } from './reports.ts'

  // R7: deals still lost at the end of the range, by reason (Pareto: largest first, running share),
  // then by seller and by the stage they were lost at.

  interface Props { report: CrmReportsState }
  const { report }: Props = $props()

  const lostDeals = $derived(lostDealsInRange(report.rowsInRange, report.selectedPipelineID, report.ownerIDs))
  const stages = $derived(activeStagesOf(report.selectedPipeline))

  const reasonName = (reasonID: number) => reasonID
    ? report.lostReasons.records.find((reason) => reason.ID === reasonID)?.Name ?? `#${reasonID}`
    : tr('No reason|Sin motivo')
  // Lost deals keep the stage they died at; a stage removed since then keeps a generic label.
  const stageName = (stageID: number) => report.selectedPipeline?.Stages.find((stage) => stage.ID === stageID)?.Name ?? `#${stageID}`

  const byReason = $derived(paretoOf(lostDeals, (deal) => deal.reasonID))
  const bySeller = $derived(paretoOf(lostDeals, (deal) => deal.ownerID))
  // Stages in board order, not by count: "where in the funnel" reads top to bottom.
  const byStage = $derived.by(() => {
    const position = (stageID: number) => stages.findIndex((stage) => stage.ID === stageID)
    return paretoOf(lostDeals, (deal: LostDeal) => deal.stageID).sort((left, right) => position(left.key) - position(right.key))
  })

  const paretoColumns = (firstHeader: string, nameOf: (key: number) => string, withCumulative: boolean): ITableColumn<ParetoRow>[] => [
    { header: firstHeader, css: 'px-6', highlight: true, getValue: (row) => nameOf(row.key) },
    { header: 'Deals|Oportunidades', css: 'px-6 text-right ff-mono', getValue: (row) => String(row.count) },
    { header: 'Amount|Monto', css: 'px-6 text-right ff-mono', getValue: (row) => formatN(row.amount / 100, 2) },
    { header: '% of lost|% de perdidas', css: 'px-6 text-right ff-mono', getValue: (row) => `${Math.round(row.percent)}%` },
    ...(withCumulative ? [{ header: 'Cumulative|Acumulado', css: 'px-6 text-right ff-mono',
      getValue: (row: ParetoRow) => `${Math.round(row.cumulativePercent)}%` }] : []),
  ]
</script>

<div class="flex flex-col gap-16" aria-label="Lost reasons">
  {#if !lostDeals.length}
    <div class="text-fg-muted p-16">{tr('No opportunities lost in the selected months.|No hay oportunidades perdidas en los meses elegidos.')}</div>
  {:else}
    <div class="bg-white/80 rounded-lg p-12">
      <Chart type="bar" series={[{ name: tr('Lost deals|Oportunidades perdidas'), color: '#ef4444', values: byReason.map((row) => row.count) }]}
        labels={byReason.map((row) => reasonName(row.key))} height={260} onlyIntegers={true} showLegend={false} />
    </div>
    <VTable columns={paretoColumns(tr('Reason|Motivo'), reasonName, true)} data={byReason} css="w-full" />
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-16">
      <VTable columns={paretoColumns(tr('Seller|Vendedor'), (ownerID) => report.ownerName(ownerID), false)} data={bySeller} css="w-full" />
      <VTable columns={paretoColumns(tr('Lost at stage|Etapa donde se perdió'), stageName, false)} data={byStage} css="w-full" />
    </div>
  {/if}
</div>
