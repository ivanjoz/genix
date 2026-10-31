<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import Page from '#domain/Page.svelte';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { untrack } from 'svelte';
  import AssigneeFilter from '../../AssigneeFilter.svelte';
  import { backlogFilterOptions } from '../../backlog/backlog.ts';
  import { ProjectBacklog, ProjectPeople, ProjectsService, lastOpenedProjectID, projectPickerOptions, rememberOpenedProject } from '../../project.svelte.ts';
  import StoryPanel from '../../story/StoryPanel.svelte';
  import { isStoryInGanttFilters, storyAssigneeIDs } from '../timeline.ts';
  import ProjectGantt from './ProjectGantt.svelte';

  const STORY_LAYER_ID = 1
  const GANTT_WEEKS_VIEW = 1
  const GANTT_SPRINT_VIEW = 2

  const projects = new ProjectsService(true)
  const backlog = new ProjectBacklog(projects)
  const people = new ProjectPeople()
  const currentUserID = security.getUserInfo()?.ID || 0

  let storyPanel = $state<StoryPanel>()
  const ganttViewOptions = [{ id: GANTT_WEEKS_VIEW, name: tr('Weeks|Semanas') }, { id: GANTT_SPRINT_VIEW, name: tr('By sprint|Por sprint') }]
  // The Gantt's view and filters: a story state (0 = any) and the chosen people (none = everyone).
  // The chosen view survives a reload: the browser remembers it.
  const GANTT_VIEW_STORAGE_KEY = 'projects_gantt_view'
  let ganttFilters = $state({
    view: Number(localStorage.getItem(GANTT_VIEW_STORAGE_KEY)) || GANTT_WEEKS_VIEW, state: 0, assigneeIDs: [] as number[],
  })
  $effect(() => { localStorage.setItem(GANTT_VIEW_STORAGE_KEY, String(ganttFilters.view)) })
  const ganttStateOptions = $derived(backlogFilterOptions([], [], () => '', tr).stateOptions)

  // The project lives in the URL (?project=), like on the Backlog: a link or a reload opens the same one.
  const projectID = $derived(Number(page.url.searchParams.get('project')) || 0)

  const openSprintBoard = (sprintID: number) => { void goto(`/projects/sprints?project=${projectID}&sprint=${sprintID}`) }

  const selectProject = (selectedProjectID: number) => {
    void goto(`/projects/reports/gantt?project=${selectedProjectID}`, { replace: true })
  }

  // Without ?project= (first visit, or the side menu clicked while already here) the last opened
  // project is reopened.
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

<!-- On desktop the page is pinned to the viewport: only the chart scrolls, so the toolbar stays in view. -->
<Page title="Gantt" options={projectPickerOptions(projects.records)} useMenuSearchBox
  selectedOption={projectID || undefined} onSelectOption={selectProject}
  containerCss="md:flex md:flex-col md:h-[calc(100vh-var(--header-height))]">
  {#if backlog.project}
    <div class="flex items-end gap-12 mb-12 flex-wrap shrink-0">
      <div class="flex items-center gap-12">
        <SearchSelect placeholder="All states|Todos los estados" options={ganttStateOptions} keyId="id" keyName="name" saveOn={ganttFilters} save="state" css="w-200" />
        <AssigneeFilter assigneeIDs={storyAssigneeIDs(backlog.stories, (userID) => people.userName(userID))} {people} bind:selectedAssigneeIDs={ganttFilters.assigneeIDs} />
      </div>
      <CheckboxOptions saveOn={ganttFilters} save="view" options={ganttViewOptions} keyId="id" keyName="name" useButtonsSlim={true} css="ml-auto" />
    </div>
  {/if}

  {#if !projectID}
    <div class="text-fg-muted py-24 text-center"><T text="Choose a project.|Elige un proyecto." /></div>
  {:else if backlog.project}
    <ProjectGantt {backlog} {people} isSprintView={ganttFilters.view === GANTT_SPRINT_VIEW}
      isStoryShown={(story) => isStoryInGanttFilters(story, ganttFilters.state || 0, ganttFilters.assigneeIDs)}
      onOpenStory={(story) => storyPanel?.openStoryPanel(story)} onOpenSprint={openSprintBoard} />
  {/if}

  <StoryPanel bind:this={storyPanel} {backlog} {people} {currentUserID} layerID={STORY_LAYER_ID} />
</Page>
