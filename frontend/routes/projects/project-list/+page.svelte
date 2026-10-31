<script lang="ts">
  import { goto } from '$app/navigation';
  import { untrack } from 'svelte';
  import Button from '#components/buttons/Button.svelte';
  import Checkbox from '#components/form/Checkbox.svelte';
  import FilterInput from '#components/form/FilterInput.svelte';
  import Layer from '#components/layers/Layer.svelte';
  import T from '#components/misc/T.svelte';
  import Page from '#domain/Page.svelte';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import { ACCESS_ID_PROJECTS, ACCESS_LEVEL_WRITE, isProjectAdmin, makeEmptyProject } from '../project.ts';
  import { ProjectBacklog, ProjectPeople, ProjectsService } from '../project.svelte.ts';
  import ProjectPanel from '../ProjectPanel.svelte';
  import ProjectCard from './ProjectCard.svelte';
  import { visibleProjects } from './project-list.ts';

  const PROJECT_LAYER_ID = 1

  const projects = new ProjectsService(true)
  // Only the panel's Epics tab reads it: the epics of the project being edited.
  const backlog = new ProjectBacklog(projects)
  const people = new ProjectPeople()
  const currentUserID = security.getUserInfo()?.ID || 0
  // The write level of the projects access creates projects; each project's admins edit it.
  const canCreateProjects = security.checkAcceso(ACCESS_ID_PROJECTS, ACCESS_LEVEL_WRITE)

  let projectPanel = $state<ProjectPanel>()
  let filterText = $state('')
  let showArchived = $state(false)
  const shownProjects = $derived(visibleProjects(projects.records, filterText, showArchived))

  // The avatars name every member of the listed projects.
  $effect(() => {
    const memberUserIDs = projects.records.flatMap((project) => project.Members.map((member) => member.UserID))
    untrack(() => { void people.rememberUsers(memberUserIDs) })
  })
</script>

<Page title="Projects|Proyectos">
  <div class="flex items-center gap-12 mb-12 flex-wrap">
    <FilterInput label="Filter projects|Filtrar proyectos" css="w-full md:w-260" icon="icon-[fa--search]" bind:value={filterText} />
    <Checkbox label="Show archived|Mostrar archivados" checked={showArchived} onToggle={(isChecked) => { showArchived = isChecked }} />
    {#if canCreateProjects}
      <Button name="New project|Nuevo proyecto" label="Opens the form to create a project.|Abre el formulario para crear un proyecto."
        color="green" icon="icon-[fa--plus]" hideNameOnMobile css="ml-auto"
        onClick={() => projectPanel?.openProjectPanel(makeEmptyProject(currentUserID))} />
    {/if}
  </div>

  <Layer type="content">
    {#if shownProjects.length === 0}
      <div class="text-fg-muted py-24 text-center">
        <T text={projects.records.length === 0 ? 'There are no projects yet.|Todavía no hay proyectos.' : 'No project matches the filter.|Ningún proyecto coincide con el filtro.'} />
      </div>
    {/if}
    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12 pb-16">
      {#each shownProjects as project (project.ID)}
        <ProjectCard {project} {currentUserID} userName={(userID) => people.userName(userID)}
          onOpen={(openedProject) => goto(`/projects/backlog?project=${openedProject.ID}`)}
          onEdit={isProjectAdmin(project, currentUserID) ? (editedProject) => projectPanel?.openProjectPanel(editedProject) : undefined} />
      {/each}
    </div>
  </Layer>

  <ProjectPanel bind:this={projectPanel} {projects} {backlog} {people} {currentUserID} layerID={PROJECT_LAYER_ID} />
</Page>
