<script lang="ts">
  import Calendar from '#components/calendar/Calendar.svelte';
  import Card from '#components/cards/Card.svelte';
  import T from '#components/misc/T.svelte';
  import MonthInlineSelector from '#domain/MonthInlineSelector.svelte';
  import { calendarMonthCodeOfDay, todayUnixDay } from '#libs/helpers.ts';
  import { tr } from '#core/store.svelte.ts';
  import type { IStory } from '../../project.ts';
  import type { ProjectBacklog } from '../../project.svelte.ts';
  import { projectCalendarActivities } from '../timeline.ts';

  interface Props {
    backlog: ProjectBacklog
    onOpenStory: (story: IStory) => void
    onOpenSprint: (sprintID: number) => void
  }

  const { backlog, onOpenStory, onOpenSprint }: Props = $props()

  let monthCode = $state(calendarMonthCodeOfDay(todayUnixDay()))

  const activities = $derived(backlog.project ? projectCalendarActivities(backlog.project, backlog.stories, backlog.sprints, tr) : [])
</script>

<div class="flex items-center gap-8 mb-10 flex-wrap shrink-0">
  <MonthInlineSelector bind:monthCode />
  <span class="text-[14px] text-fg-muted ml-auto">
    <T text="Sprints over their dates, and each story on the day it was finished.|Los sprints en sus fechas, y cada historia el día en que se finalizó." />
  </span>
</div>

<Calendar mode="month" month={monthCode} {activities} css="mb-24 md:mb-12 md:min-h-0 md:overflow-y-auto">
  {#snippet activityRender(activity)}
    {#if activity.story || activity.sprintID}
      <Card label={`Open ${activity.title}|Abrir ${activity.title}`} css="flex items-center gap-4 text-sm font-semibold leading-tight cursor-pointer min-w-0"
        onClick={() => { if (activity.story) { onOpenStory(activity.story) } else if (activity.sprintID) { onOpenSprint(activity.sprintID) } }}>
        {#if activity.icon}<i class="{activity.icon} shrink-0"></i>{/if}
        <span class="truncate">{activity.title}</span>
      </Card>
    {:else}
      <div class="flex items-center gap-4 text-sm font-semibold leading-tight min-w-0">
        {#if activity.icon}<i class="{activity.icon} shrink-0"></i>{/if}
        <span class="truncate">{activity.title}</span>
      </div>
    {/if}
  {/snippet}
</Calendar>
