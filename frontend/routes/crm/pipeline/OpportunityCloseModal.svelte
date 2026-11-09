<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import { useUI } from '@genix/ui'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { OpportunityOutcome, type ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'
  import type { PipelineBoard } from './pipeline.svelte.ts'

  // Won, lost (asks for the reason) or reopen: the backend decides if the user may.
  let { board, modalID, opportunity, outcome, onDone }: {
    board: PipelineBoard
    modalID: number
    opportunity: ICrmOpportunity
    outcome: number
    onDone: () => void
  } = $props()

  const ui = useUI()
  let closeForm = $state({ LostReasonID: 0, LostNote: '' })
  const reasonOptions = $derived(board.lostReasons.records.map((reason) => ({ id: reason.ID, name: reason.Name })))
  const title = $derived(outcome === OpportunityOutcome.WON ? 'Close as won|Cerrar como ganada'
    : outcome === OpportunityOutcome.LOST ? 'Close as lost|Cerrar como perdida' : 'Reopen opportunity|Reabrir oportunidad')

  const confirmClose = async () => {
    if (outcome === OpportunityOutcome.LOST && !closeForm.LostReasonID) {
      notifyFailure(tr('Choose a lost reason.|Elija un motivo de pérdida.'))
      return
    }
    showLoading(tr('Saving...|Guardando...'))
    try {
      await board.closeOpportunity(opportunity.ID, outcome, closeForm.LostReasonID, closeForm.LostNote)
      closeForm = { LostReasonID: 0, LostNote: '' }
      ui.closeModal(modalID)
      onDone()
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }
</script>

<Modal id={modalID} size={4} bodyCss="px-16 py-14" title={tr(title)} onSave={confirmClose}>
  <div class="grid grid-cols-24 gap-10" aria-label="Close opportunity form">
    <div class="col-span-24 font-bold">{opportunity.Title}</div>
    {#if outcome === OpportunityOutcome.LOST}
      <SearchSelect bind:saveOn={closeForm} save="LostReasonID" label="Reason|Motivo" required={true}
        options={reasonOptions} keyId="id" keyName="name" css="col-span-24" />
      <Input bind:saveOn={closeForm} save="LostNote" label="Note|Nota" css="col-span-24" />
    {:else if outcome === OpportunityOutcome.WON}
      <div class="col-span-24 text-gray-600">{tr('The client stops being a prospect.|El cliente deja de ser prospecto.')}</div>
    {/if}
  </div>
</Modal>
