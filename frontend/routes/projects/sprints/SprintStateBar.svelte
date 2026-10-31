<script lang="ts">
  import { tr } from '#core/store.svelte.ts';
  import { STORY_STATE_FINISHED, STORY_STATE_IN_DEVELOPMENT, STORY_STATE_IN_REVIEW, STORY_STATE_STOPPED, storyState } from '../project.ts';
  import { formatHours } from '../backlog/backlog.ts';
  import { sprintStateShares, type ISprintCard } from './sprints.ts';

  // The sprint's committed hours as one bar: each colored segment is the share of the hours in a
  // state, from the closest to done; the gray rest is still pending.
  interface Props {
    cards: ISprintCard[]
    css?: string
  }

  const { cards, css = '' }: Props = $props()

  const SEGMENT_COLORS: Record<number, string> = {
    [STORY_STATE_FINISHED]: 'bg-green-solid',
    [STORY_STATE_IN_REVIEW]: 'bg-orange-solid',
    [STORY_STATE_IN_DEVELOPMENT]: 'bg-purple-solid',
    [STORY_STATE_STOPPED]: 'bg-red-solid',
  }
  const segments = $derived(sprintStateShares(cards, [STORY_STATE_FINISHED, STORY_STATE_IN_REVIEW, STORY_STATE_IN_DEVELOPMENT, STORY_STATE_STOPPED]))
</script>

<div class="flex h-8 rounded-full bg-surface-muted overflow-hidden {css}">
  {#each segments as segment (segment.state)}
    {#if segment.percent > 0}
      <div class="h-full {SEGMENT_COLORS[segment.state]}" style:width="{segment.percent}%"
        title="{tr(storyState(segment.state).name)}: {formatHours(segment.hours)} ({Math.round(segment.percent)}%)"></div>
    {/if}
  {/each}
</div>
