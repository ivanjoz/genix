<script lang="ts">
  import { untrack } from 'svelte'
  import { useUI } from '@genix/ui'
  import OpportunityEmailModal from '#domain/crm/OpportunityEmailModal.svelte'
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import FieldsForm from '#components/form/FieldsForm.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import Badge from '#components/misc/Badge.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { postClientProviders } from '#services/crm/client-provider.svelte.ts'
  import { OpportunityOutcome, type ICrmOpportunity } from '#services/crm/crm-opportunity.svelte.ts'
  import { STANDARD_FIELD } from '#services/crm/crm-standard-field.ts'
  import { cancelCrmApprovalRequest, getCrmOpportunityApproval, type ICrmTask } from '#services/crm/crm-task.svelte.ts'
  import OpportunityActivity from './OpportunityActivity.svelte'
  import type { PipelineBoard } from './pipeline.svelte.ts'
  import {
    buildOpportunityFormFields, canChangeOwner, canCreateOpportunity, canEditOpportunity, canMoveOpportunity, readCustomValues,
    requiredStandardFieldsAtStage, writeCustomValues,
  } from './pipeline.ts'

  // Side panel of one opportunity. "Information": its own fields, the pipeline's custom fields and
  // the stage actions. "Activity": what the reps logged and the stage history (OpportunityActivity).
  let { board, layerID, opportunity, onClose, onMove, onCloseAs }: {
    board: PipelineBoard
    layerID: number
    /** The opportunity to edit, or a blank one (ID 0) carrying the pipeline and stage to create it in. */
    opportunity: ICrmOpportunity
    onClose: () => void
    onMove: (opportunity: ICrmOpportunity) => void
    onCloseAs: (opportunity: ICrmOpportunity, outcome: number) => void
  } = $props()

  let form = $state({} as ICrmOpportunity)
  let customValuesForm = $state({} as Record<string, any>)
  let prospectForm = $state({ Name: '' })

  const INFORMATION_VIEW = 1
  const ACTIVITY_VIEW = 2
  let layerView = $state(INFORMATION_VIEW)
  // A sent email is a new timeline entry: bumping the key reloads the Activity view.
  const EMAIL_MODAL_ID = 8201
  const ui = useUI()
  let activityReloadKey = $state(0)

  const isNew = $derived(form.ID === 0)
  const isOpen = $derived(form.Outcome === OpportunityOutcome.OPEN)
  const canEdit = $derived(isNew ? canCreateOpportunity(board.level) : canEditOpportunity(board.level, form, board.userID))
  const userOptions = $derived(board.users.usuarios
    .map((user) => ({ id: user.ID, name: `${user.FirstName} ${user.LastName}` })))
  const clientOptions = $derived(board.clients.records.map((client) => ({ id: client.ID, name: client.Name })))
  // What the opportunity's current stage requires: custom fields in the form, standard ones marked.
  const customFormFields = $derived(board.selectedPipeline
    ? buildOpportunityFormFields(board.selectedPipeline, board.customFieldsMap, { users: userOptions }, form.StageID) : [])
  const requiredStandardFields = $derived(requiredStandardFieldsAtStage(board.selectedPipeline, form.StageID))

  // Reload the form whenever the page hands over another opportunity.
  $effect(() => {
    const incoming = $state.snapshot(opportunity)
    untrack(() => {
      form = structuredClone(incoming)
      customValuesForm = writeCustomValues(incoming.CustomValues ?? [], board.customFieldsMap)
      prospectForm = { Name: '' }
      // A new opportunity has no timeline yet: it only shows the form.
      if (incoming.ID === 0) layerView = INFORMATION_VIEW
    })
  })

  // The approval the card waits on: the card carries only its ID, the task says who must decide.
  let pendingApproval = $state<ICrmTask>()
  $effect(() => {
    const waitingOpportunityID = opportunity.PendingApprovalTaskID ? opportunity.ID : 0
    untrack(() => {
      pendingApproval = undefined
      if (waitingOpportunityID) {
        getCrmOpportunityApproval(waitingOpportunityID).then((task) => {
          if (opportunity.ID === waitingOpportunityID) pendingApproval = task
        }).catch(() => {})
      }
    })
  })

  const cancelApprovalRequest = async () => {
    if (!pendingApproval || !board.opportunities) return
    showLoading(tr('Cancelling request...|Cancelando solicitud...'))
    try {
      await cancelCrmApprovalRequest(board.opportunities, pendingApproval.ID)
      onClose()
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }

  const currentStageName = $derived(
    board.pipelines.recordsMap.get(form.PipelineID)?.Stages.find((stage) => stage.ID === form.StageID)?.Name ?? '')

  const save = async () => {
    showLoading(tr('Saving opportunity...|Guardando oportunidad...'))
    try {
      let clientID = form.ClientID
      // A lead with no record yet is created as a prospect client first (the server needs a ClientID).
      if (!clientID && prospectForm.Name.trim()) {
        const [prospect] = await postClientProviders([{
          ID: 0, Type: 1, Name: prospectForm.Name.trim(), PersonType: 2, IsProspect: true,
        } as any])
        clientID = prospect.ID
        await board.clients.fetch()
      }
      await board.saveOpportunity({
        ...$state.snapshot(form), ClientID: clientID, CustomValues: readCustomValues(customValuesForm, board.customFieldsMap),
      })
      onClose()
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }
</script>

<Layer
  id={layerID}
  type="side"
  sideLayerSize={640}
  title={isNew ? 'New opportunity|Nueva oportunidad' : form.Title}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  options={isNew ? undefined : [[INFORMATION_VIEW, 'Information|Información'], [ACTIVITY_VIEW, 'Activity|Actividad']]}
  bind:selected={layerView}
  onSave={canEdit && isOpen && layerView === INFORMATION_VIEW ? save : undefined}
  onClose={onClose}
>
  {#snippet optionsSide()}
    <!-- Where the opportunity stands, on the tabs' row: it describes it in both views. -->
    <div class="flex items-center gap-8 flex-wrap text-[14px]">
      {#if form.Outcome === OpportunityOutcome.WON}
        <Badge text="Won|Ganada" css="bg-green-bg-strong text-green-fg" />
      {:else if form.Outcome === OpportunityOutcome.LOST}
        <Badge text="Lost|Perdida" css="bg-red-bg-strong text-red-fg" />
      {/if}
      {#if currentStageName}
        <Badge text={currentStageName} />
      {/if}
    </div>
  {/snippet}

  {#if layerView === ACTIVITY_VIEW && !isNew}
    <!-- Anyone who sees the pipeline may email the client; the email lands on the timeline below. -->
    <div class="flex justify-end mt-8">
      <Button icon="icon-[fa--envelope]" color="blue" name="Send email|Enviar correo" label="Writes an email from a template and sends it to the client."
        onClick={() => ui.openModal(EMAIL_MODAL_ID)} />
    </div>
    {#key activityReloadKey}
      <OpportunityActivity {board} opportunityID={form.ID} />
    {/key}
  {:else}
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Opportunity form">
      <Input bind:saveOn={form} save="Title" label="Title|Título" required={true} disabled={!canEdit || !isOpen} css="col-span-24" />
      <SearchSelect bind:saveOn={form} save="ClientID" label="Client|Cliente" required={!prospectForm.Name} options={clientOptions}
        keyId="id" keyName="name" disabled={!canEdit || !isOpen} css="col-span-24 md:col-span-12" />
      {#if isNew}
        <Input bind:saveOn={prospectForm} save="Name" label="…or new prospect|…o nuevo prospecto" css="col-span-24 md:col-span-12" />
      {/if}
      <Input bind:saveOn={form} save="Amount" type="number" baseDecimals={2} label="Amount|Monto" disabled={!canEdit || !isOpen} css="col-span-12"
        required={requiredStandardFields.includes(STANDARD_FIELD.AMOUNT)} />
      <Input bind:saveOn={form} save="Probability" type="number" label="Probability %|Probabilidad %" disabled={!canEdit || !isOpen} css="col-span-12" />
      <DateInput bind:saveOn={form} save="ExpectedCloseDate" label="Expected close|Cierre estimado" disabled={!canEdit || !isOpen} css="col-span-12"
        required={requiredStandardFields.includes(STANDARD_FIELD.EXPECTED_CLOSE_DATE)} />
      <SearchSelect bind:saveOn={form} save="OwnerUserID" label="Owner|Responsable" options={userOptions} keyId="id" keyName="name"
        disabled={!canChangeOwner(board.level) || !isOpen} css="col-span-12" />
    </div>
  
    {#if customFormFields.length}
      <FieldsForm fields={customFormFields} bind:values={customValuesForm} css="mt-12" />
    {/if}
  
    {#if form.PendingApprovalTaskID && pendingApproval}
      <!-- The server refuses a move or close while the approval is pending, so the actions give way to it. -->
      <div class="mt-16 rounded border border-line bg-surface-soft p-10 flex items-center gap-10" aria-label="Pending approval">
        <i class="icon-[fa--gavel]"></i>
        <div class="grow min-w-0">
          <div class="ff-bold">{pendingApproval.Title}</div>
          <div class="text-fg-muted">{tr('Waiting for|Esperando a')} {board.users.usuariosMap.get(pendingApproval.AssignedUserID)?.FirstName ?? `#${pendingApproval.AssignedUserID}`}</div>
        </div>
        {#if pendingApproval.CreatedBy === board.userID && board.opportunities}
          <Button color="red" name="Cancel request|Cancelar solicitud" label="Withdraws the approval request and releases the card."
            onClick={cancelApprovalRequest} />
        {/if}
      </div>
    {:else if !isNew && canMoveOpportunity(board.level)}
      <div class="flex gap-8 mt-16 flex-wrap" aria-label="Opportunity actions">
        {#if isOpen}
          <Button color="blue" name="Move to...|Mover a..." label="Opens the dialog to move it to another stage or pipeline." onClick={() => onMove(form)} />
          <Button color="green" name="Won|Ganada" label="Closes the opportunity as won." onClick={() => onCloseAs(form, OpportunityOutcome.WON)} />
          <Button color="red" name="Lost|Perdida" label="Closes the opportunity as lost and asks for the reason." onClick={() => onCloseAs(form, OpportunityOutcome.LOST)} />
        {:else}
          <Button color="blue" name="Reopen|Reabrir" label="Reopens the closed opportunity in its stage." onClick={() => onCloseAs(form, OpportunityOutcome.OPEN)} />
        {/if}
      </div>
    {/if}
  {/if}
</Layer>

{#if !isNew}
  <OpportunityEmailModal modalID={EMAIL_MODAL_ID} opportunityID={form.ID} emailTemplates={board.emailTemplates}
    onSent={() => { activityReloadKey++ }} />
{/if}
