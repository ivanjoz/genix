<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Card from '#components/cards/Card.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import T from '#components/misc/T.svelte';
  import { formatTime } from '#libs/helpers.ts';
  import { SPRINT_STATE_CLOSED, type ISprint } from '../project.ts';
  import type { ProjectBacklog } from '../project.svelte.ts';
  import { formatHours } from '../backlog/backlog.ts';
  import { SPRINT_STATES, sortSprints, sprintCards, sprintProgress } from './sprints.ts';

  interface Props {
    backlog: ProjectBacklog
    canAdmin: boolean
    onOpen: (sprint: ISprint) => void
    onPlan: (sprint: ISprint) => void
  }

  const { backlog, canAdmin, onOpen, onPlan }: Props = $props()

  const storiesByID = $derived(new Map(backlog.stories.map((story) => [story.ID, story])))
  const sprints = $derived(sortSprints(backlog.sprints))
</script>

{#if backlog.isLoaded && sprints.length === 0}
  <div class="text-fg-muted py-24 text-center"><T text="This project has no sprints yet.|Este proyecto todavía no tiene sprints." /></div>
{/if}

<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-12 pb-24">
  {#each sprints as sprint (sprint.ID)}
    {@const progress = sprintProgress(sprintCards(sprint, storiesByID))}
    {@const finishedPercent = progress.committedHours ? Math.round(progress.finishedHours / progress.committedHours * 100) : 0}
    <Card label={`Open the board of ${sprint.Name}|Abrir el tablero de ${sprint.Name}`} onClick={() => onOpen(sprint)}
      css="group flex flex-col gap-8 p-12 rounded-lg bg-surface border border-line cursor-pointer hover:border-accent-border hover:shadow-sm">
      <div class="flex items-center gap-8">
        <span class="ff-bold truncate">{sprint.Name}</span>
        <!-- Planning a sprint (a closed one is read only) shows on hover; always visible on touch screens, which have no hover. -->
        {#if canAdmin && sprint.State !== SPRINT_STATE_CLOSED}
          <Button icon="icon-[fa--pencil]" useCircle label={`Plan ${sprint.Name}: edit it and its stories.|Planificar ${sprint.Name}: editarlo y sus historias.`}
            css="md:opacity-0 md:group-hover:opacity-100 focus-visible:opacity-100" onClick={() => onPlan(sprint)} />
        {/if}
        <span class="grow"></span>
        <Badge text={SPRINT_STATES[sprint.State]?.name} css={SPRINT_STATES[sprint.State]?.badgeCss} />
      </div>
      <div class="text-[14px] text-fg-muted">{formatTime(sprint.StartDate, 'd/m/Y')} → {formatTime(sprint.EndDate, 'd/m/Y')}</div>
      {#if sprint.Goal}<div class="text-[14px] text-fg-soft line-clamp-2">{sprint.Goal}</div>{/if}
      <div class="flex items-center gap-12 text-[14px] text-fg-muted">
        <span><b>{progress.finishedCount}</b> / {progress.storiesCount} <T text="stories|historias" /></span>
        <span class="ml-auto"><b>{formatHours(progress.finishedHours)}</b> / {formatHours(progress.committedHours)}</span>
      </div>
      <div class="h-6 rounded-full bg-surface-muted overflow-hidden" title="{finishedPercent}%">
        <div class="h-full bg-green-solid" style:width="{finishedPercent}%"></div>
      </div>
    </Card>
  {/each}
</div>
