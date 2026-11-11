<script lang="ts">
  import { useUI } from '@genix/ui'
  import SelectBox from '#components/form/SelectBox.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { formatTime } from '#libs/helpers.ts'
  import { TASK_KIND } from '#services/crm/crm-task.ts'
  import type { ICrmTask } from '#services/crm/crm-task.svelte.ts'
  import OpportunityEmailModal from '#domain/crm/OpportunityEmailModal.svelte'
  import TaskCard from './TaskCard.svelte'
  import TaskLayer from './TaskLayer.svelte'
  import { ALL_TEAM_MEMBERS, CrmTasksState, type TasksView } from './tasks.svelte.ts'
  import {
    packDaysIntoRows, relativeDayLabel, TASK_RESULT_BADGE_CSS, TASK_RESULT_LABELS, taskColumnsForWidth,
  } from './tasks.ts'

  // The inbox of the CRM: the activities the stages proposed and the approvals waiting for the user,
  // as cards under a divider per due day. A card opens the task's side layer, where it is closed.
  // A team leader also reads their members' open tasks, read-only: only the assignee acts.
  const TASK_LAYER_ID = 8101
  const EMAIL_MODAL_ID = 8103
  const ui = useUI()
  const tasksState = new CrmTasksState()
  let activeTask = $state<ICrmTask>()

  const viewOptions = $derived([
    ['mine', 'My tasks|Mis tareas'],
    ...(tasksState.teamMemberIDs.length ? [['team', 'My team|Mi equipo']] : []),
    ['done', 'Done|Hechas'],
  ].map(([viewID, label]) => [viewID, tr(label)]))
  const memberOptions = $derived([
    { id: ALL_TEAM_MEMBERS, name: 'All|Todos' },
    ...tasksState.teamMemberIDs.map((memberID) => ({ id: memberID, name: tasksState.userName(memberID) })),
  ])
  const isReadOnly = $derived(tasksState.view !== 'mine')

  const openTask = (task: ICrmTask) => {
    activeTask = task
    ui.openSideLayer(TASK_LAYER_ID)
  }

  // Columns follow the list's width (max 4); short consecutive days share a row.
  const CARD_GAP = 8
  let taskListWidth = $state(0)
  const taskColumns = $derived(taskColumnsForWidth(taskListWidth, CARD_GAP))
  const dayRows = $derived(packDaysIntoRows(tasksState.openGroups.days, taskColumns))
  // Closed tasks: the date column (plus its bar and margin) takes part of the width.
  const CLOSED_DAY_LABEL_WIDTH = 96
  const closedColumns = $derived(taskColumnsForWidth(taskListWidth - CLOSED_DAY_LABEL_WIDTH - 12, CARD_GAP))

  const dayDividerLabel = (dueDate: number) => {
    const relativeLabel = relativeDayLabel(dueDate, tasksState.today)
    const dateLabel = formatTime(dueDate, 'd M Y') as string
    return relativeLabel ? `${tr(relativeLabel)} · ${dateLabel}` : dateLabel
  }
</script>

<!-- One day (or the approvals) inside a row: its divider over its cards, spanning as many columns
     as it has cards. Its inner grid uses the row's gap, so cards line up across rows. -->
{#snippet daySegment(label: string, tasks: ICrmTask[], isOverdue: boolean)}
  {@const spanColumns = Math.min(tasks.length, taskColumns)}
  <div class="min-w-0" style="grid-column: span {spanColumns}">
    <div class="flex items-center gap-10 mb-8" aria-label="Day divider">
      <span class="ff-bold shrink-0" class:text-red-700={isOverdue}>{label} ({tasks.length})</span>
      <div class="grow border-t {isOverdue ? 'border-red-border' : 'border-line'}"></div>
    </div>
    <div class="grid gap-8" style="grid-template-columns: repeat({spanColumns}, minmax(0, 1fr))">
      {#each tasks as task (task.ID)}
        <TaskCard {task} badgeText={tasksState.stageName(task)} badgeCss="bg-blue-bg-strong text-blue-fg"
          sellerName={tasksState.showsSeller ? tasksState.userName(task.AssignedUserID) : ''}
          isSelected={activeTask?.ID === task.ID && ui.state.sideLayerId === TASK_LAYER_ID}
          onOpen={() => openTask(task)} />
      {/each}
    </div>
  </div>
{/snippet}

<!-- The page is fixed to the viewport height: the tabs stay put and only the task list scrolls. -->
<Page title="CRM Tasks|Tareas CRM" fixedFullHeight={true} containerCss="flex flex-col">
  <!-- Fixed height: the bar does not jump when the team view adds its member select. -->
  <div class="flex items-center gap-12 h-44 mb-8 shrink-0">
    <OptionsStrip selected={tasksState.view} options={viewOptions} onSelect={(option) => tasksState.selectView(option[0] as TasksView)} />
    {#if tasksState.view === 'team'}
      <SelectBox options={memberOptions} keyId="id" keyName="name" css="w-240"
        selected={tasksState.teamMemberID} onChange={(option) => { if (option) tasksState.selectTeamMember(option.id) }} />
    {/if}
  </div>

  <div class="grow min-h-0 overflow-y-auto pr-4 pb-12" aria-label="Task list">
  <div bind:clientWidth={taskListWidth} class="pt-4">
  {#if tasksState.view === 'done'}
    <!-- One block per day the tasks were closed: the date beside a vertical bar, then its cards.
         Days never share a row here. -->
    {#each tasksState.closedDays as closedDay (closedDay.doneDay)}
      <div class="flex mb-14" aria-label="Closed tasks of one day">
        <div class="shrink-0 text-right text-blue-600 border-r-3 border-blue-500 pr-10 mr-12"
          style="width: {CLOSED_DAY_LABEL_WIDTH}px">
          <div class="text-[14px] leading-tight">{formatTime(closedDay.doneDay, 'Y')}</div>
          <div class="ff-bold text-[20px] leading-tight">{formatTime(closedDay.doneDay, 'M-d')}</div>
        </div>
        <div class="grow min-w-0 grid gap-8" style="grid-template-columns: repeat({closedColumns}, minmax(0, 1fr))">
          {#each closedDay.tasks as task (task.ID)}
            <TaskCard {task} badgeText={TASK_RESULT_LABELS[task.TaskStatus] ?? ''} badgeCss={TASK_RESULT_BADGE_CSS[task.TaskStatus] ?? ''}
              timeLabel={formatTime(task.DoneTime, 'h:n') as string}
              isSelected={activeTask?.ID === task.ID && ui.state.sideLayerId === TASK_LAYER_ID}
              onOpen={() => openTask(task)} />
          {/each}
        </div>
      </div>
    {:else}
      <div class="text-fg-muted p-16">{tr('Nothing finished yet.|Todavía no hay tareas terminadas.')}</div>
    {/each}
  {:else}
    <div>
      {#if tasksState.openGroups.approvals.length}
        <div class="grid gap-8 mb-14" style="grid-template-columns: repeat({taskColumns}, minmax(0, 1fr))">
          {@render daySegment(tr('Approvals waiting|Aprobaciones pendientes'), tasksState.openGroups.approvals, false)}
        </div>
      {/if}
      {#each dayRows as dayRow (dayRow[0].dueDate)}
        <div class="grid gap-8 mb-14" style="grid-template-columns: repeat({taskColumns}, minmax(0, 1fr))">
          {#each dayRow as day (day.dueDate)}
            {@render daySegment(dayDividerLabel(day.dueDate), day.tasks, day.isOverdue)}
          {/each}
        </div>
      {/each}
    </div>
    {#if !tasksState.openGroups.approvals.length && !tasksState.openGroups.days.length}
      <div class="text-fg-muted p-16">{tr('No open tasks.|No hay tareas abiertas.')}</div>
    {/if}
  {/if}
  </div>
  </div>
</Page>

{#if activeTask}
  <TaskLayer {tasksState} layerID={TASK_LAYER_ID} task={activeTask} {isReadOnly} onOpenEmail={() => ui.openModal(EMAIL_MODAL_ID)} />
  <OpportunityEmailModal modalID={EMAIL_MODAL_ID} opportunityID={activeTask.OpportunityID} emailTemplates={tasksState.emailTemplates}
    presetTemplateID={activeTask.Kind === TASK_KIND.SEND_EMAIL ? (tasksState.templateOf(activeTask)?.EmailTemplateID ?? 0) : 0}
    taskID={activeTask.Kind === TASK_KIND.SEND_EMAIL ? activeTask.ID : 0}
    onSent={async () => { ui.openSideLayer(0); await tasksState.mineOpen.fetchOnline() }} />
{/if}
