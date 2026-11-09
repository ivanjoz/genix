<script lang="ts">
  import { untrack } from 'svelte'
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SelectBox from '#components/form/SelectBox.svelte'
  import T from '#components/misc/T.svelte'
  import { tr } from '#core/store.svelte.ts'
  import AttachmentFiles from '#domain/attachments/AttachmentFiles.svelte'
  import type { IAttachment } from '#domain/attachments/attachments.ts'
  import { formatTime } from '#libs/helpers.ts'
  import {
    getCrmOpportunityActivity, postCrmOpportunityActivity, uploadCrmAttachment, type ICrmActivity,
  } from '#services/crm/crm-activity.svelte.ts'
  import { SYSTEM_ACTIVITY_TYPES } from '#services/crm/crm-activity-types.ts'
  import { getCrmOpportunityLog, type ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
  import type { PipelineBoard } from './pipeline.svelte.ts'
  import { activityTypeOf, buildOpportunityTimeline, stageLogDescription } from './pipeline.ts'

  // The "Activity" tab of an opportunity: the composer (pick a type, write, attach) over one
  // timeline of what the reps logged and the stage history. Any member of the pipeline may write.
  let { board, opportunityID }: { board: PipelineBoard, opportunityID: number } = $props()

  const makeEmptyActivityForm = (typeID: number) => ({ TypeID: typeID, Text: '', Attachments: [] as IAttachment[] })

  let activities = $state([] as ICrmActivity[])
  let stageLogs = $state([] as ICrmOpportunityStageLog[])
  let isLoaded = $state(false)
  // Note is preselected; the chosen type is kept after sending.
  let activityForm = $state(makeEmptyActivityForm(SYSTEM_ACTIVITY_TYPES[0].id))
  let uploadingCount = $state(0)
  let activityAttachments = $state<ReturnType<typeof AttachmentFiles>>()

  const timeline = $derived(buildOpportunityTimeline(activities, stageLogs))

  // Reload both lists whenever the panel shows another opportunity. A late answer for the previous
  // one is dropped.
  $effect(() => {
    const loadingOpportunityID = opportunityID
    untrack(() => {
      isLoaded = false
      activities = []
      stageLogs = []
      activityForm = makeEmptyActivityForm(activityForm.TypeID)
      Promise.all([getCrmOpportunityActivity(loadingOpportunityID), getCrmOpportunityLog(loadingOpportunityID)])
        .then(([loadedActivities, loadedStageLogs]) => {
          if (loadingOpportunityID !== opportunityID) return
          activities = loadedActivities ?? []
          stageLogs = loadedStageLogs ?? []
        })
        .catch(() => {})
        .finally(() => { isLoaded = true })
    })
  })

  const userName = (userID: number) => {
    const user = board.users.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}` : `#${userID}`
  }

  const stageName = (pipelineID: number, stageID: number) =>
    board.pipelines.recordsMap.get(pipelineID)?.Stages.find((stage) => stage.ID === stageID)?.Name ?? ''

  const sendActivity = async () => {
    if (uploadingCount > 0) {
      notifyFailure(tr('Wait for the files to finish uploading.|Espera a que terminen de subir los archivos.'))
      return
    }
    if (!activityForm.Text.trim() && activityForm.Attachments.length === 0) {
      notifyFailure(tr('Write something or attach a file.|Escribe algo o adjunta un archivo.'))
      return
    }
    showLoading(tr('Saving activity...|Guardando actividad...'))
    try {
      const savedActivity = await postCrmOpportunityActivity({ OpportunityID: opportunityID, ...$state.snapshot(activityForm) })
      activities = [...activities, savedActivity]
      activityForm = makeEmptyActivityForm(activityForm.TypeID)
    } catch {
      // genix-ui already notified the backend's error message. Keep the text to retry.
    }
    hideLoading()
  }
</script>

<div class="flex flex-col gap-12 mt-12">
  <div class="flex flex-col gap-6 rounded border border-line bg-surface-soft p-8">
    <div class="flex items-center gap-8">
      <span class="grow min-w-0 truncate text-fg-muted"><T text="Write the activity or drop a file|Escribe la actividad o arrastra un archivo" /></span>
      <SelectBox options={board.activityTypeOptions} keyId="id" keyName="name" columns={2} css="w-200 shrink-0"
        selected={activityForm.TypeID} onChange={(typeOption) => { if (typeOption) activityForm.TypeID = typeOption.id }}>
        {#snippet optionRenderer(typeOption)}
          <span class="flex items-center gap-6 min-w-0"><i class="{typeOption.icon} shrink-0"></i><span class="truncate"><T text={typeOption.name} /></span></span>
        {/snippet}
      </SelectBox>
      <Button icon="icon-[fa--paperclip]" label="Attach files of up to 4 MB each.|Adjunta archivos de hasta 4 MB cada uno."
        css="bn-white" onClick={() => activityAttachments?.pickFiles()} />
      <Button icon="icon-[fa--paper-plane]" color="blue" label="Saves the activity.|Guarda la actividad." onClick={sendActivity} />
    </div>
    <!-- Files dropped on the text box are attached to the activity. -->
    <Input saveOn={activityForm} save="Text" useTextArea={true} rows={3} useFileDragUpload={true}
      onFileDrop={(files) => activityAttachments?.uploadFiles(files)} />
    <AttachmentFiles bind:this={activityAttachments} bind:attachments={activityForm.Attachments} bind:uploadingCount
      uploadFile={(file) => uploadCrmAttachment(opportunityID, file)} canEdit={true} showDropZone={false} />
  </div>

  {#if !isLoaded}
    <div class="text-fg-muted"><T text="Loading...|Cargando..." /></div>
  {:else if timeline.length === 0}
    <div class="text-fg-muted"><T text="No activity yet.|Todavía no hay actividad." /></div>
  {/if}

  {#each timeline as entry (`${entry.kind}-${entry.kind === 'activity' ? entry.activity.ID : entry.stageLog.ID}`)}
    {#if entry.kind === 'activity'}
      {@const activityType = activityTypeOf(board.activityTypeOptions, entry.activity.TypeID)}
      <div class="rounded border border-line bg-surface p-10">
        <div class="flex items-center gap-8 mb-4 text-[14px]">
          <i class="{activityType.icon} text-fg-soft"></i>
          <span class="ff-bold"><T text={activityType.name} /></span>
          <span class="text-fg-muted">· {userName(entry.activity.AuthorID)} · {formatTime(entry.activity.Created, 'd/m/Y h:n')}</span>
        </div>
        {#if entry.activity.Text}
          <div class="whitespace-pre-wrap break-words">{entry.activity.Text}</div>
        {/if}
        {#if entry.activity.Attachments?.length}
          <div class="mt-6">
            <AttachmentFiles attachments={entry.activity.Attachments} canEdit={false} />
          </div>
        {/if}
      </div>
    {:else}
      <div class="flex gap-8 text-[14px] text-fg-muted px-4">
        <i class="icon-[fa--history] mt-3 shrink-0 text-fg-subtle"></i>
        <div>
          <span><T text={stageLogDescription(entry.stageLog, stageName)} /></span>
          <span class="text-fg-subtle"> · {userName(entry.stageLog.UserID)} · {formatTime(entry.stageLog.Time, 'd/m/Y h:n')}</span>
        </div>
      </div>
    {/if}
  {/each}
</div>
