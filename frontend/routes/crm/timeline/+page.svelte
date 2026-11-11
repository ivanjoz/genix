<script lang="ts">
  import { untrack } from 'svelte'
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import GanttTimeline from '#components/gantt/GanttTimeline.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import { CrmTimelineState } from './timeline.svelte.ts'
  import { CLOSED_COLOR, END_MARKS, END_MARK_BY_OUTCOME, endMarkCounts } from './timeline.ts'

  // Opportunities as a Gantt per seller: how each one moved through the stages day by day, and how
  // it ended (won, lost, left for another pipeline, or still open).
  const MAX_DAYS = 366
  const timeline = new CrmTimelineState()

  const pipelineOptions = $derived(timeline.reportPipelineIDs.map((pipelineID) => ({
    id: pipelineID, name: timeline.pipelines.records.find((pipeline) => pipeline.ID === pipelineID)?.Name ?? `Pipeline #${pipelineID}`,
  })))
  const modeOptions = [
    { id: 'start', name: tr('Started in the range|Iniciadas en el rango') },
    { id: 'last', name: tr('Last change in the range|Último cambio en el rango') },
  ]
  const ownerOptions = $derived([...new Set(timeline.cards.map((card) => card.ownerID))].filter(Boolean)
    .map((ownerID) => ({ id: ownerID, name: timeline.ownerName(ownerID) })))
  const rangeIsValid = $derived(timeline.filters.toDay >= timeline.filters.fromDay && timeline.filters.toDay - timeline.filters.fromDay < MAX_DAYS)
  const stageByID = $derived(new Map((timeline.selectedPipeline?.Stages ?? []).map((stage) => [stage.ID, stage])))

  $effect(() => {
    void timeline.reportPipelineIDs
    untrack(() => timeline.selectDefaultPipeline())
  })
  $effect(() => {
    const pipelineID = timeline.selectedPipelineID
    if (pipelineID) untrack(() => timeline.selectPipeline(pipelineID))
  })
  // Titles and sellers of the cards come from the records-by-ID cache, only for the IDs not held.
  $effect(() => {
    const opportunityIDs = (timeline.statusRows?.records ?? []).map((row) => row.OpportunityID)
    untrack(() => timeline.resolveOpportunities(opportunityIDs))
  })
</script>

<Page title="CRM Timeline|Línea de tiempo CRM">
  <div class="flex items-end gap-12 flex-wrap mb-12" aria-label="Timeline filters">
    <SearchSelect options={pipelineOptions} keyId="id" keyName="name" label="Pipeline" css="w-240"
      selected={timeline.selectedPipelineID} onChange={(option) => { timeline.selectedPipelineID = option.id }} />
    <SearchSelect options={modeOptions} keyId="id" keyName="name" label="Show|Mostrar" css="w-240"
      selected={timeline.filters.mode} onChange={(option) => { timeline.filters.mode = option.id as 'start' | 'last' }} />
    <DateInput bind:saveOn={timeline.filters} save="fromDay" label="From|Desde" css="w-150" />
    <DateInput bind:saveOn={timeline.filters} save="toDay" label="To|Hasta" css="w-150" />
  </div>

  {#if !timeline.reportPipelineIDs.length}
    <div class="text-fg-muted p-16">{tr('You have no pipelines to report on.|No tiene pipelines para reportar.')}</div>
  {:else if !rangeIsValid}
    <div class="text-fg-muted p-16">{tr('Pick a range of up to one year, ending after it starts.|Elija un rango de hasta un año, que termine después de empezar.')}</div>
  {:else}
    <!-- Seller filter on the left, the card-ending legend on the right -->
    <div class="flex items-center gap-12 flex-wrap mb-12">
      {#if ownerOptions.length}
        <AvatarToggleGroup options={ownerOptions} bind:selected={timeline.ownerIDs} label="Filter by seller|Filtrar por vendedor"
          optionLabel={(ownerName) => `Show only ${ownerName}|Mostrar solo a ${ownerName}`} />
      {/if}
      <!-- Same icons and colors the Gantt uses for each card ending -->
      <div class="ml-auto flex items-center gap-12 text-fg-muted">
        {#each Object.values(END_MARKS) as endMark}
          <span class="flex items-center gap-4"><i class={endMark.icon}></i>{tr(endMark.label)}</span>
        {/each}
      </div>
    </div>
    <GanttTimeline groups={timeline.sellers} fromDay={timeline.filters.fromDay} toDay={timeline.filters.toDay}
      groupsTitle="Seller|Vendedor" emptyText="No opportunities in the selected range.|No hay oportunidades en el rango elegido."
      css="max-h-[calc(100vh-14rem)]">
      <!-- The seller and the totals of how their cards ended -->
      {#snippet groupRender(seller)}
        {@const counts = endMarkCounts(seller.items)}
        <div class="ff-bold text-green-700">{seller.label}</div>
        <div class="text-fg-muted flex gap-8 flex-wrap">
          <span title={tr('Won|Ganadas')}><i class={END_MARKS.won.icon}></i> {counts.won}</span>
          <span title={tr('Lost|Perdidas')}><i class={END_MARKS.lost.icon}></i> {counts.lost}</span>
          <span title={tr('Still open|Abiertas')}><i class={END_MARKS.open.icon}></i> {counts.open}</span>
          {#if counts.left}<span title={tr('Left the pipeline|Salieron del pipeline')}><i class={END_MARKS.left.icon}></i> {counts.left}</span>{/if}
        </div>
      {/snippet}
      <!-- Every stage the card entered, with its day and how it closed -->
      {#snippet tooltipRender(card)}
        <div class="ff-bold mb-2">{card.label}</div>
        {#each card.points as point}
          {@const endMark = point.icon ? END_MARKS[END_MARK_BY_OUTCOME[point.outcome]] : undefined}
          {@const stage = stageByID.get(point.stageID)}
          <div class="flex items-center gap-6">
            <span class="text-fg-muted ff-mono text-[14px]">{formatTime(point.day, 'M-d')}</span>
            <span class="inline-block h-10 w-10 shrink-0 rounded-full" style:background={stage?.Color || CLOSED_COLOR}></span>
            <span>{tr(stage?.Name ?? `#${point.stageID}`)}</span>
            {#if endMark}<span class="flex items-center gap-4 ff-bold"><i class={endMark.icon}></i>{tr(endMark.label)}</span>{/if}
          </div>
        {/each}
      {/snippet}
    </GanttTimeline>
  {/if}
</Page>
