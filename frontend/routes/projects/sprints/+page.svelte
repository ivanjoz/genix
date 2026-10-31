<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import Button from '#components/buttons/Button.svelte';
  import T from '#components/misc/T.svelte';
  import { todayUnixDay } from '#libs/helpers.ts';
  import { tr } from '#core/store.svelte.ts';
  import Page from '#domain/Page.svelte';
  import { goBackStep, openBackStep } from '#libs/back-steps.svelte.ts';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { hideLoading, notifySuccess, showLoading } from '@genix/ui/notify';
  import { onMount, untrack } from 'svelte';
  import { canWorkInProject, isProjectAdmin, isProjectArchived, projectCode, type ISprint } from '../project.ts';
  import {
    ProjectBacklog, ProjectPeople, ProjectsService, lastOpenedProjectID, projectPickerOptions, rememberOpenedProject,
  } from '../project.svelte.ts';
  import StoryPanel from '../story/StoryPanel.svelte';
  import SprintBoard from './SprintBoard.svelte';
  import SprintCloseModal from './SprintCloseModal.svelte';
  import SprintList from './SprintList.svelte';
  import SprintPlanner from './SprintPlanner.svelte';
  import { makeEmptySprint } from './sprints.ts';
  import { startSprint } from './sprints.svelte.ts';

  const STORY_LAYER_ID = 1
  const PLANNER_STORY_LAYER_ID = 2
  const MOVE_MODAL_ID = 3
  const CLOSE_MODAL_ID = 4
  const MENU_LAYER_ID = 5
  const SPRINT_NAME_MODAL_ID = 6
  // The board reads the backlog again this often while it is on screen, so moves by others show up.
  const BOARD_REFRESH_MS = 30_000

  const projects = new ProjectsService(true)
  const backlog = new ProjectBacklog(projects)
  const people = new ProjectPeople()
  const currentUserID = security.getUserInfo()?.ID || 0

  let storyPanel = $state<StoryPanel>()
  let closeModal = $state<SprintCloseModal>()
  // The sprint in edit mode (planner), null on the list and the board.
  let plannerSprint = $state<ISprint | null>(null)

  // The project and the open sprint live in the URL (?project=&sprint=): a link or a reload opens the same board.
  const projectID = $derived(Number(page.url.searchParams.get('project')) || 0)
  const sprintID = $derived(Number(page.url.searchParams.get('sprint')) || 0)
  const openSprint = $derived(sprintID ? backlog.sprints.find((sprint) => sprint.ID === sprintID && sprint.ss !== 0) : undefined)
  const isBoardOpen = $derived(!!openSprint && !plannerSprint)
  const isPlannerOpen = $derived(!!plannerSprint)
  const isSprintList = $derived(!isBoardOpen && !isPlannerOpen)
  const canWork = $derived(!!backlog.project && canWorkInProject(backlog.project, currentUserID))
  const canAdmin = $derived(!!backlog.project && canWork && isProjectAdmin(backlog.project, currentUserID))

  // Opening a board is a browser history step, so "back" returns to the list.
  const showSprint = (selectedProjectID: number, selectedSprintID = 0, replace = false) => {
    void goto(`/projects/sprints?project=${selectedProjectID}${selectedSprintID ? `&sprint=${selectedSprintID}` : ''}`, { replace })
  }

  // The planner is a back step: the browser's back button returns to the list, like the header's.
  $effect(() => {
    if (!isPlannerOpen) { return }
    return openBackStep(() => { plannerSprint = null })
  })

  onMount(() => {
    const refreshBoard = () => {
      if (projectID && document.visibilityState === 'visible' && !plannerSprint && !backlog.hasPendingMoves) { void backlog.load(projectID) }
    }
    const refreshInterval = setInterval(refreshBoard, BOARD_REFRESH_MS)
    window.addEventListener('focus', refreshBoard)
    return () => {
      clearInterval(refreshInterval)
      window.removeEventListener('focus', refreshBoard)
    }
  })

  // Without ?project= (first visit, or the side menu clicked while already here, which keeps the
  // page mounted) the last opened project is reopened.
  $effect(() => {
    if (!projectID) {
      untrack(() => { if (lastOpenedProjectID()) { showSprint(lastOpenedProjectID(), 0, true) } })
      return
    }
    untrack(() => {
      plannerSprint = null
      rememberOpenedProject(projectID)
      void backlog.load(projectID).then(() => {
        void people.rememberUsers((backlog.project?.Members || []).map((member) => member.UserID))
      })
    })
  })

  const confirmStartSprint = async (sprint: ISprint) => {
    showLoading(tr('Starting sprint...|Iniciando sprint...'))
    try {
      await startSprint(backlog, sprint)
      notifySuccess(`${sprint.Name} started|${sprint.Name} iniciado`)
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }
</script>

<!-- On the list the header is the project picker. On a board or the planner it names the sprint, under it its project,
  and its back button returns to the list. Only the board has a white background; the list and the planner keep the gray one. -->
<Page title={plannerSprint ? (plannerSprint.ID ? `Plan ${plannerSprint.Name}|Planificar ${plannerSprint.Name}` : 'New sprint|Nuevo sprint')
    : isBoardOpen && openSprint ? openSprint.Name : 'Sprints'}
  options={isSprintList ? projectPickerOptions(projects.records) : undefined}
  useMenuSearchBox={isSprintList} selectedOption={projectID || undefined} onSelectOption={(selectedProjectID) => showSprint(selectedProjectID, 0, true)}
  background={isBoardOpen ? 'white' : undefined}
  subtitle={(isBoardOpen || isPlannerOpen) && backlog.project ? `${projectCode(backlog.project)} · ${backlog.project.Name}` : undefined}
  showBackButton={isBoardOpen || isPlannerOpen} onBack={() => { if (isPlannerOpen) { goBackStep() } else { showSprint(projectID) } }}>
  <!-- The list's toolbar; on a board only its read-only notice, if any, so no empty row takes space. The planner has none. -->
  {#if !isPlannerOpen && (!isBoardOpen || (backlog.project && (isProjectArchived(backlog.project) || !canWork)))}
  <div class="flex items-end gap-12 mb-12 flex-wrap">
    {#if backlog.project && isProjectArchived(backlog.project)}
      <span class="rounded px-8 py-4 bg-surface-strong text-fg-soft text-[14px]"><T text="Archived project: read only.|Proyecto archivado: solo lectura." /></span>
    {:else if backlog.project && !canWork}
      <span class="rounded px-8 py-4 bg-surface-muted text-fg-muted text-[14px]"><T text="You are not on this project's team: read only.|No estás en el equipo de este proyecto: solo lectura." /></span>
    {/if}
    {#if backlog.project && canAdmin && !sprintID && !plannerSprint}
      <Button name="New sprint|Nuevo sprint" icon="icon-[fa--plus]" color="green" css="ml-auto" hideNameOnMobile
        label="Plans a new sprint with stories of the backlog.|Planifica un nuevo sprint con historias del backlog."
        onClick={() => { if (backlog.project) { plannerSprint = makeEmptySprint(backlog.project.ID, backlog.sprints, todayUnixDay()) } }} />
    {/if}
  </div>
  {/if}

  {#if !projectID}
    <div class="text-fg-muted py-24 text-center"><T text="Choose a project.|Elige un proyecto." /></div>
  {:else if backlog.project && plannerSprint}
    {#key plannerSprint}
      <!-- The planner opens from the list only: cancelling or deleting goes back to it, saving opens the board. -->
      <SprintPlanner {backlog} {people} sprint={plannerSprint} {currentUserID} storyLayerID={PLANNER_STORY_LAYER_ID} moveModalID={MOVE_MODAL_ID} sprintNameModalID={SPRINT_NAME_MODAL_ID}
        onDone={(savedSprint) => {
          if (!savedSprint) { goBackStep(); return }
          plannerSprint = null
          showSprint(projectID, savedSprint.ID)
        }} />
    {/key}
  {:else if backlog.project && sprintID}
    {#if openSprint}
      <SprintBoard {backlog} {people} sprint={openSprint} {canWork} {canAdmin} moveModalID={MOVE_MODAL_ID} menuLayerID={MENU_LAYER_ID}
        onOpenStory={(story) => storyPanel?.openStoryPanel(story)}        onStart={() => { if (openSprint) { void confirmStartSprint(openSprint) } }}
        onClose={() => { if (openSprint) { closeModal?.openCloseModal(openSprint) } }} />
    {:else if backlog.isLoaded}
      <div class="text-fg-muted py-24 text-center"><T text="This sprint does not exist.|Este sprint no existe." /></div>
    {/if}
  {:else if backlog.project}
    <SprintList {backlog} {canAdmin} onOpen={(sprint) => showSprint(projectID, sprint.ID)} onPlan={(sprint) => { plannerSprint = sprint }} />
  {/if}

  <StoryPanel bind:this={storyPanel} {backlog} {people} {currentUserID} layerID={STORY_LAYER_ID} />
  <SprintCloseModal bind:this={closeModal} {backlog} modalID={CLOSE_MODAL_ID} />
</Page>
