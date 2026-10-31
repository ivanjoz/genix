<script lang="ts" module>
  export const PROJECT_PANEL_INFO_TAB = 1
  export const PROJECT_PANEL_EPICS_TAB = 2
</script>

<script lang="ts">
  import Layer from '#components/layers/Layer.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { useUI } from '@genix/ui';
  import { hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify';
  import {
    STATUS_DELETED, canWorkInProject, changedEpics, isProjectAdmin, makeEmptyProject, makeProjectForm, validateEpics, validateProject,
    type IEpic, type IProject,
  } from './project.ts';
  import type { ProjectBacklog, ProjectPeople, ProjectsService } from './project.svelte.ts';
  import ProjectEpicsForm from './ProjectEpicsForm.svelte';
  import ProjectForm from './ProjectForm.svelte';

  interface Props {
    projects: ProjectsService
    // The epics are read and saved through it: on the Backlog page the panel edits the backlog's own.
    backlog: ProjectBacklog
    people: ProjectPeople
    currentUserID: number
    layerID: number
  }

  const { projects, backlog, people, currentUserID, layerID }: Props = $props()

  const ui = useUI()
  // openProjectPanel sets the project; until then the form is a blank one.
  let projectForm = $state<IProject>(makeEmptyProject(0))
  let epicForms = $state<IEpic[]>([])
  let selectedTab = $state(PROJECT_PANEL_INFO_TAB)

  // The tabs save apart, each through its own route. Admins edit the project and its team; every
  // member of a project not archived edits its epics, which need the project created first.
  const storedProject = $derived(projects.recordsMap.get(projectForm.ID))
  const panelTabs = $derived.by(() => {
    const tabs: [number, string][] = []
    if (!projectForm.ID || (storedProject && isProjectAdmin(storedProject, currentUserID))) {
      tabs.push([PROJECT_PANEL_INFO_TAB, 'Information|Información'])
    }
    if (storedProject && canWorkInProject(storedProject, currentUserID)) {
      tabs.push([PROJECT_PANEL_EPICS_TAB, 'Epics|Épicas'])
    }
    return tabs
  })

  const copyStoredEpics = () => backlog.epics.filter((epic) => epic.ss !== STATUS_DELETED).map((epic) => ({ ...epic }))

  // On the Backlog page the project's epics are already loaded; from Proyectos they are read here.
  const loadEpicForms = async (projectID: number) => {
    if (backlog.projectID !== projectID) {
      epicForms = []
      await backlog.loadEpics(projectID)
    }
    epicForms = copyStoredEpics()
  }

  export const openProjectPanel = (project: IProject, tab = PROJECT_PANEL_INFO_TAB) => {
    projectForm = makeProjectForm(project)
    selectedTab = panelTabs.some(([tabID]) => tabID === tab) ? tab : panelTabs[0]?.[0]
    ui.openSideLayer(layerID)
    if (project.ID) { void loadEpicForms(project.ID) }
    // Candidates are read on opening: whoever gains or loses Edit meanwhile shows up right.
    if (panelTabs.some(([tabID]) => tabID === PROJECT_PANEL_INFO_TAB)) { void people.loadCandidates() }
  }

  // The panel stays open after a save: the other tab may still have changes, and a project just
  // created goes on with its epics.
  const saveProject = async () => {
    const validationError = validateProject(projectForm)
    if (validationError) {
      notifyFailure(tr(validationError))
      return
    }
    showLoading(tr('Saving project...|Guardando proyecto...'))
    try {
      const isNewProject = !projectForm.ID
      const savedProject = await projects.save(projectForm)
      projectForm = makeProjectForm(savedProject)
      if (isNewProject) { void loadEpicForms(savedProject.ID) }
      notifySuccess('Project saved|Proyecto guardado')
    } catch {
      // genix-ui already notified the backend's error message. Keep the form so the user can fix it.
    }
    hideLoading()
  }

  const saveEpics = async () => {
    const epicsToSave = changedEpics(backlog.epics, epicForms)
    if (epicsToSave.length === 0) {
      notifySuccess('No epic changed|Ninguna épica cambió')
      return
    }
    const validationError = validateEpics(epicsToSave)
    if (validationError) {
      notifyFailure(tr(validationError))
      return
    }
    showLoading(tr('Saving epics...|Guardando épicas...'))
    try {
      await backlog.saveEpics(epicsToSave)
      epicForms = copyStoredEpics()
      notifySuccess('Epics saved|Épicas guardadas')
    } catch {
      // genix-ui already notified the backend's error message. Keep the forms so the user can fix them.
    }
    hideLoading()
  }
</script>

<Layer type="side" id={layerID} sideLayerSize={620} css="px-8 py-8 md:px-16 md:py-10"
  title={projectForm.ID ? projectForm.Name : 'New project|Nuevo proyecto'} titleCss="h2 mb-6"
  options={panelTabs} bind:selected={selectedTab}
  onClose={() => { projectForm = makeEmptyProject(0) }}
  onSave={() => { void (selectedTab === PROJECT_PANEL_EPICS_TAB ? saveEpics() : saveProject()) }}>
  {#if selectedTab === PROJECT_PANEL_EPICS_TAB}
    <ProjectEpicsForm bind:epicForms projectID={projectForm.ID}
      canDelete={!!storedProject && isProjectAdmin(storedProject, currentUserID)} />
  {:else}
    <ProjectForm bind:projectForm {people} />
  {/if}
</Layer>
