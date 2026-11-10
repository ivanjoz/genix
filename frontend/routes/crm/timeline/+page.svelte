<script lang="ts">
  import { untrack } from 'svelte'
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import TimelineGantt from './TimelineGantt.svelte'
  import { CrmTimelineState } from './timeline.svelte.ts'

  // Opportunities as a Gantt per seller: how each one moved through the stages day by day, and how
  // it ended (won, lost, left for another pipeline, or still open).
  const MAX_DAYS = 366
  const timeline = new CrmTimelineState()

  const pipelineOptions = $derived(timeline.scope.PipelineIDs.map((pipelineID) => ({
    id: pipelineID, name: timeline.pipelines.records.find((pipeline) => pipeline.ID === pipelineID)?.Name ?? `Pipeline #${pipelineID}`,
  })))
  const modeOptions = [
    { id: 'start', name: tr('Started in the range|Iniciadas en el rango') },
    { id: 'last', name: tr('Last change in the range|Último cambio en el rango') },
  ]
  const ownerOptions = $derived([...new Set(timeline.cards.map((card) => card.ownerID))].filter(Boolean)
    .map((ownerID) => ({ id: ownerID, name: timeline.ownerName(ownerID) })))
  const rangeIsValid = $derived(timeline.filters.toDay >= timeline.filters.fromDay && timeline.filters.toDay - timeline.filters.fromDay < MAX_DAYS)

  timeline.loadScope()
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

  {#if ownerOptions.length}
    <div class="mb-12">
      <AvatarToggleGroup options={ownerOptions} bind:selected={timeline.ownerIDs} label="Filter by seller|Filtrar por vendedor"
        optionLabel={(ownerName) => `Show only ${ownerName}|Mostrar solo a ${ownerName}`} />
    </div>
  {/if}

  {#if !timeline.scope.PipelineIDs.length}
    <div class="text-fg-muted p-16">{tr('You have no pipelines to report on.|No tiene pipelines para reportar.')}</div>
  {:else if !rangeIsValid}
    <div class="text-fg-muted p-16">{tr('Pick a range of up to one year, ending after it starts.|Elija un rango de hasta un año, que termine después de empezar.')}</div>
  {:else}
    <div class="text-fg-muted mb-8">
      {tr('Each card is an opportunity: a dot for every stage it entered, ★ won, ✕ lost, ↦ moved to another pipeline, » still open. Open opportunities always show. Hover a card for its dates.|Cada tarjeta es una oportunidad: un punto por cada etapa a la que entró, ★ ganada, ✕ perdida, ↦ pasó a otro pipeline, » sigue abierta. Las abiertas siempre se muestran. Pase el mouse sobre una tarjeta para ver sus fechas.')}
    </div>
    <TimelineGantt sellers={timeline.sellers} stages={timeline.selectedPipeline?.Stages ?? []}
      fromDay={timeline.filters.fromDay} toDay={timeline.filters.toDay} ownerName={(userID) => timeline.ownerName(userID)} />
  {/if}
</Page>
