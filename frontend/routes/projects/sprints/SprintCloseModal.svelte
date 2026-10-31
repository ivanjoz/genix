<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import Modal from '#components/layers/Modal.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import T from '#components/misc/T.svelte';
  import { todayUnixDay } from '#libs/helpers.ts';
  import { tr } from '#core/store.svelte.ts';
  import { useUI } from '@genix/ui';
  import { hideLoading, notifySuccess, showLoading } from '@genix/ui/notify';
  import { storyCode, storyState, type ISprint } from '../project.ts';
  import type { ProjectBacklog } from '../project.svelte.ts';
  import { defaultCarryOvers, makeEmptySprint, plannedSprints } from './sprints.ts';
  import { closeSprint, saveSprintPlan } from './sprints.svelte.ts';

  interface Props {
    backlog: ProjectBacklog
    modalID: number
  }

  const { backlog, modalID }: Props = $props()

  const ui = useUI()
  // SearchSelect shows a 0 as empty: "back to the backlog" is -1 here and 0 in the request.
  const BACKLOG_DESTINATION = -1

  let closingSprint = $state<ISprint | null>(null)
  let carryOverRows = $state<{ StoryID: number, destinationID: number }[]>([])

  const storiesByID = $derived(new Map(backlog.stories.map((story) => [story.ID, story])))
  const destinationOptions = $derived([
    ...plannedSprints(backlog.sprints).map((sprint) => ({ ID: sprint.ID, Name: sprint.Name })),
    { ID: BACKLOG_DESTINATION, Name: tr('Back to the backlog|Volver al backlog') },
  ])
  const hasPlannedSprint = $derived(plannedSprints(backlog.sprints).length > 0)

  export const openCloseModal = (sprint: ISprint) => {
    closingSprint = sprint
    carryOverRows = defaultCarryOvers(sprint, backlog.sprints)
      .map((carryOver) => ({ StoryID: carryOver.StoryID, destinationID: carryOver.ToSprintID || BACKLOG_DESTINATION }))
    ui.openModal(modalID)
  }

  // The next sprint, created here so the unfinished stories have somewhere to go.
  const createNextSprint = async () => {
    if (!backlog.project) { return }
    showLoading(tr('Creating sprint...|Creando sprint...'))
    try {
      const nextSprint = await saveSprintPlan(backlog, { Sprint: makeEmptySprint(backlog.project.ID, backlog.sprints, todayUnixDay()) })
      carryOverRows = carryOverRows.map((row) => row.destinationID === BACKLOG_DESTINATION ? { ...row, destinationID: nextSprint.ID } : row)
      notifySuccess(`${nextSprint.Name} created|${nextSprint.Name} creado`)
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }

  const confirmClose = async () => {
    if (!closingSprint) { return }
    showLoading(tr('Closing sprint...|Cerrando sprint...'))
    try {
      await closeSprint(backlog, closingSprint, carryOverRows.map((row) => ({ StoryID: row.StoryID, ToSprintID: Math.max(row.destinationID, 0) })))
      notifySuccess(`${closingSprint.Name} closed|${closingSprint.Name} cerrado`)
      ui.closeModal(modalID)
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }
</script>

<Modal id={modalID} size={4} title={closingSprint ? `Close ${closingSprint.Name}|Cerrar ${closingSprint.Name}` : ''}
  saveButtonLabel="Close sprint|Cerrar sprint" saveIcon="icon-[fa--flag-checkered]" onSave={() => { void confirmClose() }}>
  {#if carryOverRows.length === 0}
    <div class="text-fg-soft"><T text="Every story is finished. The sprint closes as is.|Todas las historias están finalizadas. El sprint se cierra tal cual." /></div>
  {:else}
    <div class="text-fg-soft mb-10">
      <T text="These stories were not completed. Choose where each one goes on: it keeps its state there.|Estas historias no se culminaron. Elige dónde continúa cada una: allí conserva su estado." />
    </div>
    {#if !hasPlannedSprint}
      <div class="flex items-center gap-8 mb-10 text-[14px] text-fg-muted">
        <T text="There is no planned sprint to carry them to.|No hay un sprint planificado al cual pasarlas." />
        <Button name="Create next sprint|Crear siguiente sprint" icon="icon-[fa--plus]" label="Creates the next sprint, right after this one.|Crea el siguiente sprint, a continuación de este."
          onClick={() => { void createNextSprint() }} />
      </div>
    {/if}
    <div class="flex flex-col gap-8">
      {#each carryOverRows as row (row.StoryID)}
        {@const story = storiesByID.get(row.StoryID)}
        {@const sprintStory = closingSprint?.Stories?.find((candidate) => candidate.StoryID === row.StoryID)}
        <div class="flex items-center gap-10 flex-wrap md:flex-nowrap">
          <div class="grow min-w-0 basis-full md:basis-auto">
            <div class="truncate">{story ? `${storyCode(story)} · ${story.Title}` : `#${row.StoryID}`}</div>
            <Badge text={storyState(sprintStory?.State).name} size="small" css={storyState(sprintStory?.State).badgeCss} />
          </div>
          <SearchSelect label="Goes to|Pasa a" options={destinationOptions} keyId="ID" keyName="Name" saveOn={row} save="destinationID"
            notEmpty={true} css="w-full md:w-260 shrink-0" />
        </div>
      {/each}
    </div>
  {/if}
</Modal>
