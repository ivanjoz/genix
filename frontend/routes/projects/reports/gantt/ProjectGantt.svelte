<script lang="ts">
  import Gantt from '#components/gantt/Gantt.svelte';
  import Avatar from '#components/misc/Avatar.svelte';
  import { todayUnixDay } from '#libs/helpers.ts';
  import { tr } from '#core/store.svelte.ts';
  import type { IStory } from '../../project.ts';
  import type { ProjectBacklog, ProjectPeople } from '../../project.svelte.ts';
  import { projectGantt, projectMarkers, sprintGantt } from '../timeline.ts';

  interface Props {
    backlog: ProjectBacklog
    people: ProjectPeople
    // By sprint: the sprints on the header, the epics and their people on the side, the stories
    // over the days worked. Otherwise each story's sprints, by week.
    isSprintView: boolean
    // The page's state and people filters.
    isStoryShown: (story: IStory) => boolean
    onOpenStory: (story: IStory) => void
    onOpenSprint: (sprintID: number) => void
  }

  const { backlog, people, isSprintView, isStoryShown, onOpenStory, onOpenSprint }: Props = $props()

  const ganttRows = $derived(backlog.project && !isSprintView
    ? projectGantt(backlog.project, backlog.epics, backlog.stories, backlog.sprints, tr, isStoryShown) : null)
  const workGantt = $derived(isSprintView
    ? sprintGantt(backlog.epics, backlog.stories, backlog.sprints, todayUnixDay(), tr, (userID) => people.userName(userID), isStoryShown) : null)
  const markers = $derived(backlog.project ? projectMarkers(backlog.project, tr) : [])
</script>

{#if workGantt}
  <!-- 3x the week zoom's 28px: a one-day story still shows its code. -->
  <Gantt rows={workGantt.rows} headerBands={workGantt.headerBands} {markers} zoom="week" dayWidth={84} wrapBarLabels labelWidth={220} css="mb-24 md:mb-12 md:min-h-0 md:overflow-y-auto"
    emptyText="No story has been started yet.|Ninguna historia se ha iniciado todavía."
    onBarClick={(_row, bar) => { if (bar.story) { onOpenStory(bar.story) } }}>
    {#snippet rowRender(row)}
      {#if row.isEpicTitle}
        <span class="truncate font-semibold text-purple-fg">{row.label}</span>
      {:else if row.assigneeID !== undefined}
        <span class="truncate pl-12">{row.label}</span>
        {#if row.assigneeID}
          <Avatar name={row.label} colorSeed={row.assigneeID} size={24} css="ml-auto" />
        {/if}
      {/if}
    {/snippet}
  </Gantt>
{:else if ganttRows}
  <Gantt rows={ganttRows} {markers} zoom="week" labelWidth={300} css="mb-24 md:mb-12 md:min-h-0 md:overflow-y-auto"
    emptyText="No story has been in a sprint yet.|Ninguna historia ha estado en un sprint todavía."
    onBarClick={(row, bar) => {
      if (row.story) { onOpenStory(row.story) } else if (bar.sprintID) { onOpenSprint(bar.sprintID) }
    }} />
{/if}
