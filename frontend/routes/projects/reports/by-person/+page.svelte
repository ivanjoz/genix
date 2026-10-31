<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import T from '#components/misc/T.svelte';
  import VTable from '#components/vTable/VTable.svelte';
  import type { ITableColumn } from '#components/vTable/types.ts';
  import { tr } from '#core/store.svelte.ts';
  import { formatTime } from '#libs/helpers.ts';
  import { security } from '#libs/ui-runtime.svelte.ts';
  import Page from '#domain/Page.svelte';
  import { useUI } from '@genix/ui';
  import { projectCode, storyCode, storyPriority, storyState, storyType } from '../../project.ts';
  import { ProjectPeople, ProjectsService } from '../../project.svelte.ts';
  import { ASSIGNED_STORIES_DAYS, assignedStoryRows, type IAssignedStoryRow } from './by-person.ts';
  import { AssignedStories } from './by-person.svelte.ts';
  import PersonGantt from './PersonGantt.svelte';
  import PersonWeeks from './PersonWeeks.svelte';

  const LIST_TAB = 1
  const GANTT_TAB = 2
  const WEEKS_TAB = 3

  const ui = useUI()

  // Every project's stories of one person: no project picker, the projects only name the rows.
  const projects = new ProjectsService(true)
  const currentUserID = security.getUserInfo()?.ID || 0
  const people = new ProjectPeople()
  const report = new AssignedStories()
  // The picked person lives in the URL (?person=), so a reload keeps it; without it, the current user.
  const initialAssigneeID = Number(page.url.searchParams.get('person')) || currentUserID
  const personForm = $state({ assigneeID: initialAssigneeID })

  // The picker offers the team candidates, plus the current user (even when they only hold View) and
  // the person of the URL when they are not candidates.
  const pinnedUserIDs = [...new Set([currentUserID, initialAssigneeID])]
  void people.loadCandidates()
  void people.rememberUsers(pinnedUserIDs)
  void report.load(initialAssigneeID)

  const personOptions = $derived([
    ...pinnedUserIDs.filter((userID) => !people.candidates.some((candidate) => candidate.ID === userID))
      .map((userID) => ({ ID: userID, Name: people.userName(userID) })),
    ...people.candidates,
  ])

  const selectPerson = (assigneeID: number) => {
    void goto(`/projects/reports/by-person?person=${assigneeID}`, { replace: true })
    void report.load(assigneeID)
  }
  const rows = $derived(assignedStoryRows(report.assignments, report.storiesByProject, projects.recordsMap))

  const columns: ITableColumn<IAssignedStoryRow>[] = [
    { header: 'Project|Proyecto', getValue: (row) => row.project ? `${projectCode(row.project)} · ${row.project.Name}` : `#${row.story.ProjectID}`,
      mobile: { order: 1, css: 'col-span-24 text-fg-muted' } },
    { header: 'Story|Historia', headerCss: 'w-64', css: 'text-center ff-bold', getValue: (row) => storyCode(row.story),
      mobile: { order: 2, css: 'col-span-4 ff-bold' } },
    { header: 'Title|Título', highlight: true, getValue: (row) => row.story.Title,
      render: (row) => ({ css: 'flex items-center gap-6', children: [
        { css: `${storyType(row.story.Type).icon} ${storyType(row.story.Type).iconCss} shrink-0` }, { text: row.story.Title },
      ] }),
      mobile: { order: 3, css: 'col-span-20' } },
    { header: 'State|Estado', headerCss: 'w-140', getValue: (row) => tr(storyState(row.story.State).name),
      render: (row) => ({ css: `inline-block rounded px-8 py-2 text-[14px] ${storyState(row.story.State).badgeCss}`, text: tr(storyState(row.story.State).name) }),
      mobile: { order: 4, css: 'col-span-12' } },
    { header: 'Priority|Prioridad', headerCss: 'w-110', getValue: (row) => tr(storyPriority(row.story.Priority).name),
      render: (row) => ({ css: 'flex items-center gap-6', children: [
        { css: `${storyPriority(row.story.Priority).icon} ${storyPriority(row.story.Priority).iconCss}` }, { text: tr(storyPriority(row.story.Priority).name) },
      ] }),
      mobile: { order: 5, css: 'col-span-12' } },
    { header: 'Assigned|Asignada', headerCss: 'w-150', css: 'text-center', getValue: (row) => row.assignedAt, render: (row) => String(formatTime(row.assignedAt, 'd/m/Y h:n')),
      mobile: { order: 6, css: 'col-span-24', labelLeft: 'Assigned:|Asignada:' } },
  ]
</script>

<Page title="By person|Por persona" options={[{ id: LIST_TAB, name: 'List|Lista' }, { id: GANTT_TAB, name: 'Gantt' }, { id: WEEKS_TAB, name: 'Calendar|Calendario' }]}>
  <!-- The Gantt, and the Calendar on wide screens, fill the screen: their content scrolls instead of the page. -->
  <div class="flex flex-col gap-12 {ui.state.pageOptionSelected === GANTT_TAB ? 'h-[calc(100vh-var(--header-height)-20px)]'
    : ui.state.pageOptionSelected === WEEKS_TAB ? 'xl:h-[calc(100vh-var(--header-height)-20px)]' : ''}">
    <div class="flex flex-wrap items-end gap-12">
      <SearchSelect label="Person|Persona" options={personOptions} keyId="ID" keyName="Name" saveOn={personForm} save="assigneeID" notEmpty
        css="w-full md:w-280" onChange={(person) => { if (person) { selectPerson(person.ID) } }} />
      <div class="text-[14px] text-fg-muted pb-6">
        <T text={`Stories assigned in the last ${ASSIGNED_STORIES_DAYS} days, in every project.|Historias asignadas en los últimos ${ASSIGNED_STORIES_DAYS} días, en todos los proyectos.`} />
      </div>
    </div>
    {#if !report.isLoaded}
      <div class="text-fg-muted py-24 text-center"><T text="Loading…|Cargando…" /></div>
    {:else if ui.state.pageOptionSelected === GANTT_TAB}
      <PersonGantt {rows} sprintsByProject={report.sprintsByProject} />
    {:else if ui.state.pageOptionSelected === WEEKS_TAB}
      <PersonWeeks {rows} sprintsByProject={report.sprintsByProject} />
    {:else}
      <VTable {columns} data={rows} maxHeight="calc(100vh - var(--header-height) - 110px)"
        emptyMessage={`No story was assigned to this person in the last ${ASSIGNED_STORIES_DAYS} days.|No se asignó ninguna historia a esta persona en los últimos ${ASSIGNED_STORIES_DAYS} días.`} />
    {/if}
  </div>
</Page>
