<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import T from '#components/misc/T.svelte';
  import { formatTime, todayUnixDay } from '#libs/helpers.ts';
  import { tr } from '#core/store.svelte.ts';
  import { formatHours } from '../../backlog/backlog.ts';
  import { SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, SPRINT_STATE_PLANNED, storyCode, storyState, type ISprint } from '../../project.ts';
  import { SPRINT_STATES } from '../../sprints/sprints.ts';
  import {
    VISIBLE_WEEKS, WEEKLY_CAPACITY_HOURS, defaultFirstWeekStartDay, personWeeks, weekMeterScale, weekStartDay, type IAssignedStoryRow,
  } from './by-person.ts';

  interface Props {
    rows: IAssignedStoryRow[]
    sprintsByProject: Map<number, ISprint[]>
  }

  const { rows, sprintsByProject }: Props = $props()

  const today = todayUnixDay()
  const currentWeekStartDay = weekStartDay(today)
  let firstWeekStartDay = $state(defaultFirstWeekStartDay(today))

  const weeks = $derived(personWeeks(rows, sprintsByProject, firstWeekStartDay, today))
  // Every meter shares one scale, so the columns compare at a glance.
  const meterScaleHours = $derived(weekMeterScale(weeks))
  const meterPercent = (hours: number) => `${Math.min(100, (hours / meterScaleHours) * 100)}%`
  // A running sprint in red, a closed one with a green check, a planned one as a muted calendar.
  const SPRINT_STATE_ICONS: Record<number, string> = {
    [SPRINT_STATE_PLANNED]: 'icon-[fa--calendar-o]',
    [SPRINT_STATE_ACTIVE]: 'icon-[fa--refresh] text-red-solid',
    [SPRINT_STATE_CLOSED]: 'icon-[fa--check] text-green-solid',
  }
</script>

<div class="flex flex-col xl:grow xl:min-h-0">
<div class="flex items-center gap-8 mb-10 flex-wrap shrink-0">
  <Button icon="icon-[fa--chevron-left]" label="Previous week|Semana anterior" onClick={() => { firstWeekStartDay -= 7 }} />
  <span class="ff-bold w-160 text-center">{String(formatTime(firstWeekStartDay, 'd/m'))} → {String(formatTime(firstWeekStartDay + VISIBLE_WEEKS * 7 - 1, 'd/m'))}</span>
  <Button icon="icon-[fa--chevron-right]" label="Next week|Semana siguiente" onClick={() => { firstWeekStartDay += 7 }} />
  {#if firstWeekStartDay !== defaultFirstWeekStartDay(today)}
    <Button name="Today|Hoy" icon="icon-[fa--calendar-o]" label="Back to the weeks around today.|Vuelve a las semanas de hoy." onClick={() => { firstWeekStartDay = defaultFirstWeekStartDay(today) }} />
  {/if}
  <span class="ml-auto flex items-center gap-16 flex-wrap text-[14px] text-fg-muted">
    <span class="flex items-center gap-6"><span class="w-16 h-8 rounded bg-blue-solid"></span><T text="Assigned: the estimate of the week's stories, finished and planned|Asignadas: la estimación de las historias de la semana, finalizadas y planificadas" /></span>
    <span class="flex items-center gap-6"><span class="w-2 h-14 bg-fg-muted"></span>{WEEKLY_CAPACITY_HOURS} h</span>
  </span>
</div>

<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 xl:grid-rows-1 gap-10 items-start pb-24 xl:pb-8 xl:grow xl:min-h-0">
  {#each weeks as week (week.startDay)}
    {@const isOverloaded = week.assignedHours > WEEKLY_CAPACITY_HOURS}
    {@const isCurrentWeek = week.startDay === currentWeekStartDay}
    <section class="rounded-lg border bg-surface flex flex-col min-w-0 xl:max-h-full overflow-hidden {isCurrentWeek ? 'border-accent-border' : 'border-line'}">
      <header class="p-10 flex flex-col gap-6 border-b border-line shrink-0">
        <div class="flex items-center gap-8">
          <span class="ff-bold">{String(formatTime(week.startDay, 'd/m'))} → {String(formatTime(week.startDay + 6, 'd/m'))}</span>
          {#if isCurrentWeek}
            <span class="rounded px-6 text-[14px] bg-accent-bg-strong text-accent-fg"><T text="This week|Esta semana" /></span>
          {/if}
          {#if isOverloaded}
            <i class="icon-[fa--exclamation-triangle] text-red-fg ml-auto" title={tr('Overload|Sobrecarga')}></i>
          {/if}
        </div>
        <div class="flex items-baseline gap-6 flex-wrap">
          <span class="h3 ff-bold {isOverloaded ? 'text-red-fg' : ''}">{formatHours(week.assignedHours)}</span>
          <span class="text-[14px] text-fg-muted">/ {WEEKLY_CAPACITY_HOURS} h</span>
          <span class="text-[14px] text-fg-muted ml-auto"><T text="Finished|Finalizadas" /> <span class="ff-bold text-fg">{formatHours(week.finishedHours)}</span></span>
        </div>
        <div class="relative h-8 rounded bg-surface-muted">
          <div class="h-full rounded {isOverloaded ? 'bg-red-solid' : 'bg-blue-solid'}" style:width={meterPercent(week.assignedHours)}></div>
          <div class="absolute -top-3 -bottom-3 w-2 bg-fg-muted" style:left={meterPercent(WEEKLY_CAPACITY_HOURS)}></div>
        </div>
      </header>

      <!-- On wide screens each week scrolls on its own, as the sprint board's columns do. -->
      <div class="xl:min-h-0 xl:overflow-y-auto">
        {#if week.projects.length === 0}
          <div class="p-10 text-[14px] text-fg-muted"><T text="No story this week.|Sin historias esta semana." /></div>
        {/if}
        {#each week.projects as weekProject (weekProject.projectID)}
          <div class="px-10 pt-6 pb-2 bg-surface-muted flex items-center gap-8 text-[14px] ff-bold">
            <span class="truncate">{weekProject.label}</span>
            <span class="ml-auto shrink-0">{formatHours(weekProject.assignedHours)}</span>
          </div>
          {#each weekProject.sprints as weekSprint (weekSprint.sprint.ID)}
            <!-- The sprint's line sits in the project's gray block; its icon tells whether it is running or closed. -->
            <div class="px-10 pt-2 pb-6 bg-surface-muted flex items-center gap-6 text-[14px] text-slate-500"
              title={tr(SPRINT_STATES[weekSprint.sprint.State]?.name || '')}>
              <i class="shrink-0 {SPRINT_STATE_ICONS[weekSprint.sprint.State] || SPRINT_STATE_ICONS[SPRINT_STATE_PLANNED]}"></i>
              <span class="truncate">{weekSprint.sprint.Name}</span>
            </div>
            <ul class="flex flex-col">
              {#each weekSprint.stories as weekStory (weekStory.row.story.ID)}
                {@const story = weekStory.row.story}
                <li class="px-10 py-6 flex flex-col gap-4 text-[14px] border-b border-line last:border-b-0">
                  <div class="flex gap-6">
                    <span class="ff-bold shrink-0">{storyCode(story)}</span>
                    <span class="line-clamp-2">{story.Title}</span>
                  </div>
                  <div class="flex items-center gap-6">
                    <span class="rounded px-6 {storyState(weekStory.state).badgeCss}"><T text={storyState(weekStory.state).name} /></span>
                    <span class="ml-auto">{formatHours(story.EstimatedHours || 0)}</span>
                  </div>
                </li>
              {/each}
            </ul>
          {/each}
        {/each}
      </div>
    </section>
  {/each}
</div>
</div>
