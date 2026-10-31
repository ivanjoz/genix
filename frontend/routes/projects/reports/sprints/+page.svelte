<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import Card from '#components/cards/Card.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import T from '#components/misc/T.svelte';
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte';
  import { formatTime } from '#libs/helpers.ts';
  import Page from '#domain/Page.svelte';
  import { untrack } from 'svelte';
  import { SPRINT_STATE_PLANNED } from '../../project.ts';
  import { ProjectBacklog, ProjectsService, lastOpenedProjectID, projectPickerOptions, rememberOpenedProject } from '../../project.svelte.ts';
  import { SPRINT_STATES, sortSprints, sprintCards } from '../../sprints/sprints.ts';
  import SprintBurndown from './SprintBurndown.svelte';

  // One tab per report of the chosen sprint; a new report is one more option.
  const BURNDOWN_REPORT = 1

  const projects = new ProjectsService(true)
  const backlog = new ProjectBacklog(projects)

  let selectedReport = $state(BURNDOWN_REPORT)

  // The project and the sprint live in the URL (?project=&sprint=), like on the Sprints page.
  const projectID = $derived(Number(page.url.searchParams.get('project')) || 0)
  const sprintIDInURL = $derived(Number(page.url.searchParams.get('sprint')) || 0)
  // A planned sprint has nothing to report yet: only started ones are listed.
  const startedSprints = $derived(sortSprints(backlog.sprints).filter((sprint) => sprint.State !== SPRINT_STATE_PLANNED))
  // Without ?sprint= the most recent one (the active one, else the latest closed) is shown.
  const selectedSprint = $derived(startedSprints.find((sprint) => sprint.ID === sprintIDInURL) || startedSprints[0])
  const storiesByID = $derived(new Map(backlog.stories.map((story) => [story.ID, story])))

  const showReport = (selectedProjectID: number, selectedSprintID = 0) => {
    void goto(`/projects/reports/sprints?project=${selectedProjectID}${selectedSprintID ? `&sprint=${selectedSprintID}` : ''}`, { replace: true })
  }

  // Without ?project= (first visit, or the side menu clicked while already here) the last opened
  // project is reopened.
  $effect(() => {
    if (!projectID) {
      untrack(() => { if (lastOpenedProjectID()) { showReport(lastOpenedProjectID()) } })
      return
    }
    untrack(() => {
      rememberOpenedProject(projectID)
      void backlog.load(projectID)
    })
  })
</script>

<Page title="Sprints|Sprints" options={projectPickerOptions(projects.records)} useMenuSearchBox
  selectedOption={projectID || undefined} onSelectOption={(selectedProjectID) => showReport(selectedProjectID)}>
  <div class="flex flex-col md:flex-row gap-16">
    <aside class="flex flex-col gap-8 md:w-280 shrink-0">
      {#each startedSprints as sprint (sprint.ID)}
        {@const isSelected = sprint.ID === selectedSprint?.ID}
        <Card label={`Show the reports of ${sprint.Name}|Ver los reportes de ${sprint.Name}`} onClick={() => showReport(projectID, sprint.ID)}
          css="flex flex-col gap-4 p-10 rounded-lg border cursor-pointer {isSelected ? 'border-accent-border bg-accent-bg' : 'border-line bg-surface hover:border-accent-border'}">
          <div class="flex items-center gap-8">
            <span class="ff-bold grow truncate">{sprint.Name}</span>
            <Badge text={SPRINT_STATES[sprint.State]?.name} css={SPRINT_STATES[sprint.State]?.badgeCss} />
          </div>
          <div class="text-[14px] text-fg-muted">{formatTime(sprint.StartDate, 'd/m/Y')} → {formatTime(sprint.EndDate, 'd/m/Y')}</div>
        </Card>
      {/each}
      {#if backlog.isLoaded && startedSprints.length === 0}
        <div class="text-fg-muted text-[14px] py-8"><T text="No sprint of this project has started yet.|Ningún sprint de este proyecto ha iniciado todavía." /></div>
      {/if}
    </aside>

    <section class="grow min-w-0 flex flex-col gap-12">
      {#if !projectID}
        <div class="text-fg-muted py-24 text-center"><T text="Choose a project.|Elige un proyecto." /></div>
      {:else if selectedSprint}
        <OptionsStrip selected={selectedReport} options={[[BURNDOWN_REPORT, 'Burndown|Burndown']]}
          onSelect={(option) => { selectedReport = option[0] as number }} />
        {#if selectedReport === BURNDOWN_REPORT}
          <SprintBurndown sprint={selectedSprint} cards={sprintCards(selectedSprint, storiesByID)} />
        {/if}
      {/if}
    </section>
  </div>
</Page>
