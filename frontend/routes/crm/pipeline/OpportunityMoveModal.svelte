<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import { useUI } from '@genix/ui'
  import DateInput from '#components/form/DateInput.svelte'
  import FieldsForm from '#components/form/FieldsForm.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import { tr } from '#core/store.svelte.ts'
  import type { ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'
  import { STANDARD_FIELD } from '#services/crm/crm-standard-field.ts'
  import type { PipelineBoard } from './pipeline.svelte.ts'
  import {
    activeStagesOf, buildOpportunityFormFields, canEditOpportunity, hasMissingRequirements, memberLevel,
    missingRequirements, readCustomValues, writeCustomValues,
  } from './pipeline.ts'

  // Moves a card to another stage or pipeline. The target stage may require data the card does not
  // have yet (custom fields, amount, expected close): it is asked here, saved on the card, and then
  // the move is sent. presetStageID is the stage a drag dropped the card on (0 when opened by button).
  let { board, modalID, opportunity, presetStageID = 0, onDone }: {
    board: PipelineBoard
    modalID: number
    opportunity: ICrmOpportunity
    presetStageID?: number
    onDone: () => void
  } = $props()

  const ui = useUI()
  let moveForm = $state({ PipelineID: 0, StageID: 0 })
  let missingValues = $state({} as Record<string, any>)
  let missingStandardValues = $state({ Amount: 0, ExpectedCloseDate: 0 })

  // The pipelines the user can create in (a move needs Edit on the target) — the source included.
  const pipelineOptions = $derived(board.visiblePipelines.filter((pipeline) => memberLevel(pipeline, board.userID) >= 2)
    .map((pipeline) => ({ id: pipeline.ID, name: pipeline.Name })))
  const targetPipeline = $derived(board.pipelines.recordsMap.get(moveForm.PipelineID))
  const stageOptions = $derived(activeStagesOf(targetPipeline)
    .filter((stage) => !(moveForm.PipelineID === opportunity.PipelineID && stage.ID === opportunity.StageID))
    .map((stage) => ({ id: stage.ID, name: stage.ApproverUserID ? `${stage.Name} (${tr('needs approval|requiere aprobación')})` : stage.Name })))
  // What the chosen stage requires and the card lacks; nothing until a stage is chosen.
  const missing = $derived(moveForm.StageID
    ? missingRequirements(targetPipeline, moveForm.StageID, opportunity) : { fieldIDs: [], standardFields: [] })
  const missingFormFields = $derived(targetPipeline
    ? buildOpportunityFormFields({ ...targetPipeline, Fields: targetPipeline.Fields.filter((field) => missing.fieldIDs.includes(field.FieldID)) },
      board.customFieldsMap, { users: board.users.usuarios.map((user) => ({ id: user.ID, name: `${user.FirstName} ${user.LastName}` })) },
      moveForm.StageID)
    : [])

  $effect(() => {
    moveForm = { PipelineID: opportunity.PipelineID, StageID: presetStageID }
    missingValues = {}
    missingStandardValues = { Amount: opportunity.Amount, ExpectedCloseDate: opportunity.ExpectedCloseDate }
  })

  const confirmMove = async () => {
    if (!moveForm.StageID) { notifyFailure(tr('Choose the target stage.|Elija la etapa de destino.')); return }
    showLoading(tr('Moving...|Moviendo...'))
    try {
      if (hasMissingRequirements(missing)) {
        if (!canEditOpportunity(board.level, opportunity, board.userID)) {
          throw tr('The target stage requires data only the owner or a Full user can fill.|La etapa de destino exige datos que solo el responsable o un usuario Completo puede llenar.')
        }
        const needsAmount = missing.standardFields.includes(STANDARD_FIELD.AMOUNT)
        const needsCloseDate = missing.standardFields.includes(STANDARD_FIELD.EXPECTED_CLOSE_DATE)
        if (needsAmount && !missingStandardValues.Amount) throw tr('Enter the amount.|Ingrese el monto.')
        if (needsCloseDate && !missingStandardValues.ExpectedCloseDate) throw tr('Enter the expected close date.|Ingrese el cierre estimado.')
        const completed = {
          CustomValues: [...(opportunity.CustomValues ?? []), ...readCustomValues(missingValues, board.customFieldsMap)],
          ...(needsAmount ? { Amount: missingStandardValues.Amount } : {}),
          ...(needsCloseDate ? { ExpectedCloseDate: missingStandardValues.ExpectedCloseDate } : {}),
        }
        await board.saveOpportunity({ ...$state.snapshot(opportunity), ...completed })
        Object.assign(opportunity, completed)
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
  {#if hasMissingRequirements(missing)}
    <div class="mt-12 mb-6 text-gray-600">{tr('This stage requires:|Esta etapa exige:')}</div>
    {#if missing.standardFields.length}
      <div class="grid grid-cols-24 gap-10 mb-10" aria-label="Required standard fields">
        {#if missing.standardFields.includes(STANDARD_FIELD.AMOUNT)}
          <Input bind:saveOn={missingStandardValues} save="Amount" type="number" baseDecimals={2} label="Amount|Monto" required={true} css="col-span-12" />
        {/if}
        {#if missing.standardFields.includes(STANDARD_FIELD.EXPECTED_CLOSE_DATE)}
          <DateInput bind:saveOn={missingStandardValues} save="ExpectedCloseDate" label="Expected close|Cierre estimado" required={true} css="col-span-12" />
        {/if}
      </div>
    {/if}
    {#if missingFormFields.length}
      <FieldsForm fields={missingFormFields} bind:values={missingValues} />
    {/if}
  {/if}
</Modal>
