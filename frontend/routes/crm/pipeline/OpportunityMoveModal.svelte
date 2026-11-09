<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import { useUI } from '@genix/ui'
  import FieldsForm from '#components/form/FieldsForm.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import { tr } from '#core/store.svelte.ts'
  import type { ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'
  import type { PipelineBoard } from './pipeline.svelte.ts'
  import {
    activeStagesOf, buildOpportunityFormFields, canEditOpportunity, memberLevel, missingRequiredFieldIDs,
    readCustomValues, writeCustomValues,
  } from './pipeline.ts'

  // Moves a card to another stage or pipeline. A target pipeline may require fields the card does
  // not have yet: they are asked here, saved on the card, and then the move is sent.
  let { board, modalID, opportunity, onDone }: {
    board: PipelineBoard
    modalID: number
    opportunity: ICrmOpportunity
    onDone: () => void
  } = $props()

  const ui = useUI()
  let moveForm = $state({ PipelineID: 0, StageID: 0 })
  let missingValues = $state({} as Record<string, any>)

  // The pipelines the user can create in (a move needs Edit on the target) — the source included.
  const pipelineOptions = $derived(board.visiblePipelines.filter((pipeline) => memberLevel(pipeline, board.userID) >= 2)
    .map((pipeline) => ({ id: pipeline.ID, name: pipeline.Name })))
  const targetPipeline = $derived(board.pipelines.recordsMap.get(moveForm.PipelineID))
  const stageOptions = $derived(activeStagesOf(targetPipeline)
    .filter((stage) => !(moveForm.PipelineID === opportunity.PipelineID && stage.ID === opportunity.StageID))
    .map((stage) => ({ id: stage.ID, name: stage.Name })))
  const missingFieldIDs = $derived(targetPipeline ? missingRequiredFieldIDs(opportunity.CustomValues ?? [], targetPipeline.Fields) : [])
  const missingFormFields = $derived(targetPipeline
    ? buildOpportunityFormFields({ ...targetPipeline, Fields: targetPipeline.Fields.filter((field) => missingFieldIDs.includes(field.FieldID)) },
      board.customFieldsMap, { users: board.users.usuarios.map((user) => ({ id: user.ID, name: `${user.FirstName} ${user.LastName}` })) })
    : [])

  $effect(() => {
    moveForm = { PipelineID: opportunity.PipelineID, StageID: 0 }
    missingValues = {}
  })

  const confirmMove = async () => {
    if (!moveForm.StageID) { notifyFailure(tr('Choose the target stage.|Elija la etapa de destino.')); return }
    showLoading(tr('Moving...|Moviendo...'))
    try {
      if (missingFieldIDs.length) {
        if (!canEditOpportunity(board.level, opportunity, board.userID)) {
          throw tr('The target pipeline requires fields only the owner or a Full user can fill.|El pipeline de destino exige campos que solo el responsable o un usuario Completo puede llenar.')
        }
        const mergedValues = [...(opportunity.CustomValues ?? []), ...readCustomValues(missingValues, board.customFieldsMap)]
        await board.saveOpportunity({ ...$state.snapshot(opportunity), CustomValues: mergedValues })
        opportunity.CustomValues = mergedValues
      }
      await board.moveOpportunity(opportunity, moveForm.PipelineID, moveForm.StageID)
      ui.closeModal(modalID)
      onDone()
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }
</script>

<Modal id={modalID} size={5} bodyCss="px-16 py-14" title={tr('Move opportunity|Mover oportunidad')} onSave={confirmMove}>
  <div class="grid grid-cols-24 gap-10" aria-label="Move opportunity form">
    <div class="col-span-24 font-bold">{opportunity.Title}</div>
    <SearchSelect bind:saveOn={moveForm} save="PipelineID" label="Pipeline" required={true} options={pipelineOptions}
      keyId="id" keyName="name" css="col-span-24 md:col-span-12" onChange={() => { moveForm.StageID = 0 }} />
    <SearchSelect bind:saveOn={moveForm} save="StageID" label="Stage|Etapa" required={true} options={stageOptions}
      keyId="id" keyName="name" css="col-span-24 md:col-span-12" />
  </div>
  {#if missingFormFields.length}
    <div class="mt-12 mb-6 text-gray-600">{tr('This pipeline requires these fields:|Este pipeline exige estos campos:')}</div>
    <FieldsForm fields={missingFormFields} bind:values={missingValues} />
  {/if}
</Modal>
