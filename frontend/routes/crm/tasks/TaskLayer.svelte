<script lang="ts">
  import { untrack } from 'svelte'
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import { useUI } from '@genix/ui'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SelectBox from '#components/form/SelectBox.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import Badge from '#components/misc/Badge.svelte'
  import T from '#components/misc/T.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import { postCrmOpportunityActivity } from '#services/crm/crm-activity.svelte.ts'
  import { TASK_KIND, TASK_STATUS } from '#services/crm/crm-task.ts'
  import type { ICrmTask } from '#services/crm/crm-task.svelte.ts'
  import type { CrmTasksState } from './tasks.svelte.ts'
  import { TASK_RESULT_BADGE_CSS, TASK_RESULT_LABELS } from './tasks.ts'

  // Side panel of one task, where the assignee closes it:
  // - activity: logs the activity the stage asked for (the server closes the task in the same transaction);
  // - approval: approves (runs the held move as the approver) or rejects (releases the card, note in its timeline);
  // - send email: opens the email composer, which closes the task on send.
  // A template task can also be dismissed. A leader reading a member's task gets it read-only.
  let { tasksState, layerID, task, isReadOnly, onOpenEmail }: {
    tasksState: CrmTasksState
    layerID: number
    task: ICrmTask
    isReadOnly: boolean
    onOpenEmail: () => void
  } = $props()

  const ui = useUI()
  let taskForm = $state({ TypeID: 0, Text: '' })

  // The stage's template names the activity type to log; opening another task resets the form.
  $effect(() => {
    const template = tasksState.templateOf(task)
    untrack(() => {
      taskForm = { TypeID: template?.ActivityTypeID || tasksState.activityTypeOptions[0]?.id || 0, Text: '' }
    })
  })

  const closeLayer = () => ui.openSideLayer(0)

  /** Runs one closing action behind the loading overlay; on failure genix-ui already showed the backend's message. */
  const runAndClose = async (loadingMessage: string, action: () => Promise<unknown>) => {
    showLoading(tr(loadingMessage))
    try {
      await action()
      closeLayer()
    } catch {
      // The form keeps its text to retry.
    }
    hideLoading()
  }

  const completeActivity = () => {
    if (!taskForm.Text.trim()) { notifyFailure(tr('Write what was done.|Escribe lo que se hizo.')); return }
    runAndClose('Saving activity...|Guardando actividad...', async () => {
      await postCrmOpportunityActivity({ OpportunityID: task.OpportunityID, TypeID: taskForm.TypeID, Text: taskForm.Text }, task.ID)
      await tasksState.mineOpen.fetchOnline()
    })
  }

  const decideApproval = (approve: boolean) =>
    runAndClose('Saving decision...|Guardando decisión...', () => tasksState.decideApproval(task, approve, taskForm.Text.trim()))

  const dismissTask = () => runAndClose('Dismissing task...|Descartando tarea...', () => tasksState.dismissTask(task))
</script>

<Layer id={layerID} type="side" sideLayerSize={520} title={task.Title} titleCss="h2 mb-6" css="px-12 py-10" onClose={closeLayer}>
  <div class="flex flex-col gap-10 pt-6" aria-label="Task detail">
    <div class="flex items-center gap-8 flex-wrap">
      {#if tasksState.stageName(task)}<Badge text={tasksState.stageName(task)} css="bg-blue-bg-strong text-blue-fg" />{/if}
      {#if task.DueDate}<span class="text-fg-muted">{tr('Due|Vence')} {formatTime(task.DueDate, 'd M Y')}</span>{/if}
    </div>
    <div>
      <div class="text-fg-muted">{tr('Client|Cliente')}</div>
      <div class="ff-bold">{task.ClientName || '-'}</div>
    </div>
    <div>
      <div class="text-fg-muted">{tr('Opportunity|Oportunidad')}</div>
      <div class="ff-bold">{task.OpportunityTitle}</div>
    </div>
    {#if task.Kind === TASK_KIND.APPROVAL}
      <div class="text-fg-muted">{tr('Requested by|Solicitada por')} {tasksState.userName(task.CreatedBy)}</div>
    {/if}

    <!-- A closed task (the "Done" view): how it ended, when, and the note left on it. -->
    {#if task.TaskStatus !== TASK_STATUS.PENDING}
      <div class="border-t border-line my-4"></div>
      <div class="flex items-center gap-8 flex-wrap">
        <Badge text={TASK_RESULT_LABELS[task.TaskStatus] ?? ''} css={TASK_RESULT_BADGE_CSS[task.TaskStatus] ?? ''} />
        <span class="text-fg-muted">{formatTime(task.DoneTime, 'd M Y h:n')} · {tasksState.userName(task.DoneBy)}</span>
      </div>
      {#if task.ResultNote}<div class="whitespace-pre-wrap">{task.ResultNote}</div>{/if}
    {/if}

    {#if !isReadOnly}
      <div class="border-t border-line my-4"></div>
      {#if task.Kind === TASK_KIND.ACTIVITY}
        <SelectBox options={tasksState.activityTypeOptions} keyId="id" keyName="name" columns={2} css="w-full"
          selected={taskForm.TypeID} onChange={(typeOption) => { if (typeOption) taskForm.TypeID = typeOption.id }}>
          {#snippet optionRenderer(typeOption)}
            <span class="flex items-center gap-6 min-w-0"><i class="{typeOption.icon} shrink-0"></i><span class="truncate"><T text={typeOption.name} /></span></span>
          {/snippet}
        </SelectBox>
        <Input saveOn={taskForm} save="Text" useTextArea={true} rows={5} label="Activity|Actividad" required={true} />
        <div class="flex justify-end gap-8">
          <Button icon="icon-[fa--times]" css="bn-white" name="Dismiss|Descartar" label="Dismiss the task.|Descartar la tarea." onClick={dismissTask} />
          <Button icon="icon-[fa--check]" color="green" name="Complete|Completar" label="Log the activity and close the task.|Registrar la actividad y cerrar la tarea." onClick={completeActivity} />
        </div>
      {:else if task.Kind === TASK_KIND.APPROVAL}
        <Input saveOn={taskForm} save="Text" useTextArea={true} rows={4} label="Note|Nota" />
        <div class="flex justify-end gap-8">
          <Button icon="icon-[fa--times]" color="red" name="Reject|Rechazar" label="Reject the request.|Rechazar la solicitud." onClick={() => decideApproval(false)} />
          <Button icon="icon-[fa--check]" color="green" name="Approve|Aprobar" label="Approve the request.|Aprobar la solicitud." onClick={() => decideApproval(true)} />
        </div>
      {:else if task.Kind === TASK_KIND.SEND_EMAIL}
        <div class="flex justify-end gap-8">
          <Button icon="icon-[fa--times]" css="bn-white" name="Dismiss|Descartar" label="Dismiss the task.|Descartar la tarea." onClick={dismissTask} />
          <Button icon="icon-[fa--envelope]" color="blue" name="Write email|Redactar correo" label="Open the email composer.|Abrir el redactor de correo." onClick={onOpenEmail} />
        </div>
      {/if}
    {/if}
  </div>
</Layer>
