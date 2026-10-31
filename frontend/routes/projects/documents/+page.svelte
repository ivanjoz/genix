<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import T from '#components/misc/T.svelte';
  import Page from '#domain/Page.svelte';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { untrack } from 'svelte';
  import { canWorkInProject } from '../project.ts';
  import {
    ProjectBacklog, ProjectPeople, ProjectsService, lastOpenedProjectID, projectPickerOptions, rememberOpenedProject,
  } from '../project.svelte.ts';
  import StoryPanel from '../story/StoryPanel.svelte';
  import { projectDocuments, storyDocumentGroups } from './documents.ts';
  import { ProjectDocumentsService } from './documents.svelte.ts';
  import ProjectDocuments from './ProjectDocuments.svelte';
  import StoryDocuments from './StoryDocuments.svelte';

  const STORY_LAYER_ID = 1

  const projects = new ProjectsService(true)
  const backlog = new ProjectBacklog(projects)
  const people = new ProjectPeople()
  const currentUserID = security.getUserInfo()?.ID || 0

  let documentsService = $state<ProjectDocumentsService | null>(null)
  let storyPanel = $state<StoryPanel>()

  // The project lives in the URL (?project=), like on Backlog and Sprints.
  const projectID = $derived(Number(page.url.searchParams.get('project')) || 0)
  const canWork = $derived(!!backlog.project && canWorkInProject(backlog.project, currentUserID))
  const documents = $derived(documentsService?.records || [])
  const storiesByID = $derived(new Map(backlog.stories.map((story) => [story.ID, story])))
  const sprintsByID = $derived(new Map(backlog.sprints.map((sprint) => [sprint.ID, sprint])))

  const selectProject = (selectedProjectID: number) => {
    void goto(`/projects/documents?project=${selectedProjectID}`, { replace: true })
  }

  // Without ?project= (first visit, or the side menu clicked while already here) the last opened
  // project is reopened. The stories and sprints name the files' groups.
  $effect(() => {
    if (!projectID) {
      untrack(() => { if (lastOpenedProjectID()) { selectProject(lastOpenedProjectID()) } })
      return
    }
    untrack(() => {
      rememberOpenedProject(projectID)
      documentsService = new ProjectDocumentsService(projectID)
      void documentsService.fetch()
      void backlog.load(projectID).then(() => {
        void people.rememberUsers((backlog.project?.Members || []).map((member) => member.UserID))
      })
    })
  })
</script>

<Page title="Documents|Documentos" options={projectPickerOptions(projects.records)} useMenuSearchBox
  selectedOption={projectID || undefined} onSelectOption={selectProject}>
  {#if !projectID}
    <div class="text-fg-muted py-24 text-center"><T text="Choose a project.|Elige un proyecto." /></div>
  {:else}
    <div class="flex flex-col md:flex-row gap-16">
      <aside class="flex flex-col gap-12 md:w-340 shrink-0">
        {#if documentsService}
          <ProjectDocuments {documentsService} documents={projectDocuments(documents)} canEdit={canWork} />
        {/if}
      </aside>
      <div class="grow min-w-0">
        <StoryDocuments storyDocumentGroups={storyDocumentGroups(documents, storiesByID, sprintsByID)}
          onOpenStory={(story) => storyPanel?.openStoryPanel(story)} />
      </div>
    </div>
  {/if}

  <!-- A story's files are edited in the story: saving it or commenting reloads the documents. -->
  <StoryPanel bind:this={storyPanel} {backlog} {people} {currentUserID} layerID={STORY_LAYER_ID}
    onStoryChanged={() => { void documentsService?.fetchOnline() }} />
</Page>
