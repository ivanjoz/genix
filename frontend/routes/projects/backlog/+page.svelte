<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import Button from '#components/buttons/Button.svelte';
  import T from '#components/misc/T.svelte';
  import Page from '#domain/Page.svelte';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { untrack } from 'svelte';
  import { canWorkInProject, isProjectArchived } from '../project.ts';
  import { ProjectBacklog, ProjectPeople, ProjectsService, lastOpenedProjectID, projectPickerOptions, rememberOpenedProject } from '../project.svelte.ts';
  import ProjectPanel, { PROJECT_PANEL_EPICS_TAB } from '../ProjectPanel.svelte';
  import StoryPanel from '../story/StoryPanel.svelte';
  import { makeBacklogFilters } from './backlog.ts';
  import BacklogFilters from './BacklogFilters.svelte';
  import BacklogView from './BacklogView.svelte';

  const STORY_LAYER_ID = 1
  const PROJECT_LAYER_ID = 2

  const projects = new ProjectsService(true)
  const backlog = new ProjectBacklog(projects)
  const people = new ProjectPeople()
  const currentUserID = security.getUserInfo()?.ID || 0

  let storyPanel = $state<StoryPanel>()
  let projectPanel = $state<ProjectPanel>()
  let filters = $state(makeBacklogFilters())

  // The project lives in the URL (?project=), so a link or a reload opens the same one.
  const projectID = $derived(Number(page.url.searchParams.get('project')) || 0)
  const canWork = $derived(!!backlog.project && canWorkInProject(backlog.project, currentUserID))

  // Epics are edited in the project panel, the same one as on Proyectos, opened on its Epics tab.
  const openEpicsPanel = () => { if (backlog.project) { projectPanel?.openProjectPanel(backlog.project, PROJECT_PANEL_EPICS_TAB) } }

  const selectProject = (selectedProjectID: number) => {
    void goto(`/projects/backlog?project=${selectedProjectID}`, { replace: true })
  }

  // Without ?project= (first visit, or the side menu clicked while already here, which keeps the
  // page mounted) the last opened project is reopened.
  $effect(() => {
    if (!projectID) {
      untrack(() => { if (lastOpenedProjectID()) { selectProject(lastOpenedProjectID()) } })
      return
    }
    untrack(() => {
      rememberOpenedProject(projectID)
      void backlog.load(projectID).then(() => {
        void people.rememberUsers((backlog.project?.Members || []).map((member) => member.UserID))
      })
    })
  })
</script>

<!-- On desktop the page is pinned to the viewport: only the story list (BacklogView) scrolls, so the toolbar stays in view. -->
<Page title="Backlog" options={projectPickerOptions(projects.records)} useMenuSearchBox
  selectedOption={projectID || undefined} onSelectOption={selectProject}
  containerCss="md:flex md:flex-col md:h-[calc(100vh-var(--header-height))]">
  <div class="flex items-end gap-12 mb-12 flex-wrap shrink-0">
    {#if backlog.project}
      <BacklogFilters bind:filters {backlog} {people} />
    {/if}
    {#if backlog.project && isProjectArchived(backlog.project)}
      <span class="rounded px-8 py-4 bg-surface-strong text-fg-soft text-[14px]"><T text="Archived project: read only.|Proyecto archivado: solo lectura." /></span>
    {:else if backlog.project && !canWork}
      <span class="rounded px-8 py-4 bg-surface-muted text-fg-muted text-[14px]"><T text="You are not on this project's team: read only.|No estás en el equipo de este proyecto: solo lectura." /></span>
    {/if}
    {#if backlog.project && canWork}
      <div class="flex gap-8 ml-auto">
        <Button icon="icon-[fa--pencil]" color="orange" label="Edit the project's epics|Editar las épicas del proyecto"
          onClick={openEpicsPanel} />
        <!-- With an epic filtered, the new story starts in that epic. -->
        <Button name="Story|Historia" icon="icon-[fa--plus]" color="green" label="Opens the form to create a story.|Abre el formulario para crear una historia."
          hideNameOnMobile onClick={() => storyPanel?.openNewStory(projectID, filters.epicID > 0 ? filters.epicID : 0)} />
      </div>
    {/if}
  </div>

  {#if !projectID}
    <div class="text-fg-muted py-24 text-center"><T text="Choose a project.|Elige un proyecto." /></div>
  {:else if backlog.project}
    <BacklogView {backlog} {people} bind:filters {canWork}
      onOpenStory={(story) => storyPanel?.openStoryPanel(story)}
      onNewStory={(epicID) => storyPanel?.openNewStory(projectID, epicID)}
      onEditEpics={openEpicsPanel} />
  {/if}

  <StoryPanel bind:this={storyPanel} {backlog} {people} {currentUserID} layerID={STORY_LAYER_ID} />
  <ProjectPanel bind:this={projectPanel} {projects} {backlog} {people} {currentUserID} layerID={PROJECT_LAYER_ID} />
</Page>
