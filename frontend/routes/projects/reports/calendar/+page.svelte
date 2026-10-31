<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import T from '#components/misc/T.svelte';
  import Page from '#domain/Page.svelte';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { untrack } from 'svelte';
  import { ProjectBacklog, ProjectPeople, ProjectsService, lastOpenedProjectID, projectPickerOptions, rememberOpenedProject } from '../../project.svelte.ts';
  import StoryPanel from '../../story/StoryPanel.svelte';
  import ProjectCalendar from './ProjectCalendar.svelte';

  const STORY_LAYER_ID = 1

  const projects = new ProjectsService(true)
  const backlog = new ProjectBacklog(projects)
  const people = new ProjectPeople()
  const currentUserID = security.getUserInfo()?.ID || 0

  let storyPanel = $state<StoryPanel>()

  // The project lives in the URL (?project=), like on the Backlog: a link or a reload opens the same one.
  const projectID = $derived(Number(page.url.searchParams.get('project')) || 0)

  const openSprintBoard = (sprintID: number) => { void goto(`/projects/sprints?project=${projectID}&sprint=${sprintID}`) }

  const selectProject = (selectedProjectID: number) => {
    void goto(`/projects/reports/calendar?project=${selectedProjectID}`, { replace: true })
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

<!-- On desktop the page is pinned to the viewport: only the calendar scrolls, so its month toolbar stays in view. -->
<Page title="Calendar|Calendario" options={projectPickerOptions(projects.records)} useMenuSearchBox
  selectedOption={projectID || undefined} onSelectOption={selectProject}
  containerCss="md:flex md:flex-col md:h-[calc(100vh-var(--header-height))]">
  {#if !projectID}
    <div class="text-fg-muted py-24 text-center"><T text="Choose a project.|Elige un proyecto." /></div>
  {:else if backlog.project}
    <ProjectCalendar {backlog} onOpenStory={(story) => storyPanel?.openStoryPanel(story)} onOpenSprint={openSprintBoard} />
  {/if}

  <StoryPanel bind:this={storyPanel} {backlog} {people} {currentUserID} layerID={STORY_LAYER_ID} />
</Page>
