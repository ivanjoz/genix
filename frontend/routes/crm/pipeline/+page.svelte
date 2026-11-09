<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import Button from '#components/buttons/Button.svelte'
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Kanban from '#components/layers/Kanban.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatN } from '#libs/helpers.ts'
  import type { ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'
  import OpportunityCard from './OpportunityCard.svelte'
  import OpportunityCloseModal from './OpportunityCloseModal.svelte'
  import OpportunityLayer from './OpportunityLayer.svelte'
  import OpportunityMoveModal from './OpportunityMoveModal.svelte'
  import { PipelineBoard } from './pipeline.svelte.ts'
  import {
    activeStagesOf, canCreateOpportunity, canMoveOpportunity, columnTotals, isStale, sUnixTimeOf,
  } from './pipeline.ts'

  const ui = useUI()
  const board = new PipelineBoard()
  const OPPORTUNITY_LAYER_ID = 1
  const CLOSE_MODAL_ID = 2
  const MOVE_MODAL_ID = 3

  let activeOpportunity = $state({} as ICrmOpportunity)
  let closeOutcome = $state(0)

  const stages = $derived(activeStagesOf(board.selectedPipeline))
  const columns = $derived(stages.map((stage) => ({ id: stage.ID, name: stage.Name, color: stage.Color })))
  const allCards = $derived(board.opportunities?.records ?? [])
  // The owner filter lists the people holding cards in this pipeline; none chosen shows everyone.
  let selectedOwnerIDs = $state<number[]>([])
  const ownerOptions = $derived([...new Set(allCards.map((opportunity) => opportunity.OwnerUserID).filter(Boolean))]
    .map((ownerID) => ({ id: ownerID, name: ownerNameOf(ownerID) })))
  const cards = $derived(selectedOwnerIDs.length
    ? allCards.filter((opportunity) => selectedOwnerIDs.includes(opportunity.OwnerUserID))
    : allCards)
  const pipelineOptions = $derived(board.visiblePipelines.map((pipeline) => ({ id: pipeline.ID, name: pipeline.Name })))
  const nowSUnixTime = sUnixTimeOf(Date.now())

  // Open the remembered pipeline as soon as the user's pipelines have synced.
  $effect(() => {
    if (board.visiblePipelines.length && !untrack(() => board.selectedPipelineID)) {
      untrack(() => board.selectInitialPipeline())
    }
  })

  const previousLabelOf = (opportunity: ICrmOpportunity) => {
    if (!opportunity.PreviousPipelineID) return ''
    const previousPipeline = board.pipelines.recordsMap.get(opportunity.PreviousPipelineID)
    const previousStage = previousPipeline?.Stages.find((stage) => stage.ID === opportunity.PreviousStageID)
    return [previousPipeline?.Name, previousStage?.Name].filter(Boolean).join(' · ')
  }

  const ownerNameOf = (userID: number) => {
    const owner = board.users.usuariosMap.get(userID)
    return owner ? `${owner.FirstName} ${owner.LastName}` : ''
  }

  const openOpportunity = (opportunity: ICrmOpportunity) => {
    activeOpportunity = opportunity
    ui.openSideLayer(OPPORTUNITY_LAYER_ID)
  }

  const openNewOpportunity = () => {
    activeOpportunity = {
      ID: 0, PipelineID: board.selectedPipelineID, StageID: stages[0]?.ID ?? 0, Title: '', ClientID: 0,
      OwnerUserID: board.userID, Amount: 0, Probability: stages[0]?.Probability ?? 0, Outcome: 0, CustomValues: [],
    } as unknown as ICrmOpportunity
    ui.openSideLayer(OPPORTUNITY_LAYER_ID)
  }

  const onMove = (cardID: number, toStageID: number) => {
    const movedOpportunity = cards.find((opportunity) => opportunity.ID === cardID)
    if (movedOpportunity) board.moveOpportunity(movedOpportunity, movedOpportunity.PipelineID, toStageID)
  }
</script>

<!-- The page is fixed to the viewport height so each stage column scrolls its own cards. -->
<Page title="Opportunities|Oportunidades" fixedFullHeight={true} containerCss="flex flex-col">

  <div class="flex items-center justify-between mb-8 gap-12 flex-wrap" aria-label="Pipeline toolbar">
    <SearchSelect options={pipelineOptions} keyId="id" keyName="name" label="Pipeline" css="w-256"
      selected={board.selectedPipelineID} onChange={(option) => board.selectPipeline(option.id)} />
    {#if ownerOptions.length}
      <AvatarToggleGroup options={ownerOptions} bind:selected={selectedOwnerIDs} label="Filter by owner|Filtrar por responsable"
        optionLabel={(ownerName) => `Show only the opportunities of ${ownerName}|Mostrar solo las oportunidades de ${ownerName}`} />
    {/if}
    {#if canCreateOpportunity(board.level)}
      <Button color="green" icon="icon-[fa--plus]" name="New|Nueva" label="Opens the side layer to create an opportunity in the first stage."
        onClick={openNewOpportunity} />
    {/if}
  </div>

  {#if !board.visiblePipelines.length}
    <div class="text-fg-muted p-16">
      {tr('You are not assigned to any pipeline yet. Ask an administrator to give you access.|Aún no tiene pipelines asignados. Pida acceso a un administrador.')}
    </div>
  {:else}
    <Kanban {columns} {cards} getCardID={(opportunity) => opportunity.ID} getColumnID={(opportunity) => opportunity.StageID}
      canDrag={() => canMoveOpportunity(board.level)} onmove={onMove} css="grow min-h-0 pb-12">
      {#snippet columnFooter(column, columnCards)}
        {@const totals = columnTotals(columnCards)}
        <div class="px-4 text-fg-muted flex justify-between" aria-label="Column totals">
          <span class="ff-mono">{formatN(totals.amount / 100, 2)}</span>
          <span class="ff-mono">≈ {formatN(totals.weightedAmount / 100, 2)}</span>
        </div>
      {/snippet}
      {#snippet card(opportunity)}
        <OpportunityCard {opportunity}
          clientName={board.clients.recordsMap.get(opportunity.ClientID)?.Name ?? ''}
          ownerName={ownerNameOf(opportunity.OwnerUserID)}
          isStale={isStale(opportunity, stages.find((stage) => stage.ID === opportunity.StageID), nowSUnixTime)}
          previousLabel={previousLabelOf(opportunity)}
          onOpen={() => openOpportunity(opportunity)} />
      {/snippet}
    </Kanban>
  {/if}
</Page>

<OpportunityLayer {board} layerID={OPPORTUNITY_LAYER_ID} opportunity={activeOpportunity}
  onClose={() => ui.openSideLayer(0)}
  onMove={(opportunity) => { activeOpportunity = opportunity; ui.openModal(MOVE_MODAL_ID) }}
  onCloseAs={(opportunity, outcome) => { activeOpportunity = opportunity; closeOutcome = outcome; ui.openModal(CLOSE_MODAL_ID) }} />
<OpportunityCloseModal {board} modalID={CLOSE_MODAL_ID} opportunity={activeOpportunity} outcome={closeOutcome}
  onDone={() => ui.openSideLayer(0)} />
<OpportunityMoveModal {board} modalID={MOVE_MODAL_ID} opportunity={activeOpportunity} onDone={() => ui.openSideLayer(0)} />
