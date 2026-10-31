<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Card from '#components/cards/Card.svelte';
  import Avatar from '#components/misc/Avatar.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { storyCode, storyPriority, storyType } from '../project.ts';
  import type { ProjectPeople } from '../project.svelte.ts';
  import { formatHours } from '../backlog/backlog.ts';
  import type { ISprintCard } from './sprints.ts';

  interface Props {
    card: ISprintCard
    people: ProjectPeople
    isNotCompleted: boolean
    // Its move is still being written: a spinner replaces the type icon.
    isMoving: boolean
    // Set on mobile, where cards move through a menu instead of being dragged.
    onMoveMenu?: (card: ISprintCard) => void
    onOpen: (card: ISprintCard) => void
  }

  const { card, people, isNotCompleted, isMoving, onMoveMenu, onOpen }: Props = $props()

  const type = $derived(storyType(card.story.Type))
  const priority = $derived(storyPriority(card.story.Priority))
  const code = $derived(storyCode(card.story))
</script>

<Card label={`Open ${code} ${card.story.Title}|Abrir ${code} ${card.story.Title}`} onClick={() => onOpen(card)}
  css="flex flex-col gap-6 p-8 rounded-md bg-surface border border-line shadow-sm cursor-pointer hover:border-accent-border
    {isNotCompleted ? 'border-l-4 border-l-orange-400' : ''}">
  <div class="flex items-start gap-6">
    {#if isMoving}
      <i class="icon-[fa--spinner] animate-spin text-accent-solid shrink-0 mt-3" title={tr('Saving the move...|Guardando el movimiento...')}></i>
    {:else}
      <i class="{type.icon} {type.iconCss} shrink-0 mt-3" title={tr(type.name)}></i>
    {/if}
    <span class="grow min-w-0 leading-snug">{card.story.Title}</span>
    {#if onMoveMenu}
      <Button icon="icon-[fa--exchange]" label={`Move ${code}|Mover ${code}`} onClick={(ev) => { ev.stopPropagation(); onMoveMenu(card) }} />
    {/if}
  </div>
  {#if isNotCompleted}
    <Badge text="Not completed|No culminada" size="small" css="self-start bg-orange-bg-strong text-orange-fg" />
  {/if}
  <div class="flex items-center gap-8 text-[14px] text-fg-muted">
    <span>{code}</span>
    <i class="{priority.icon} {priority.iconCss}" title={tr(priority.name)}></i>
    {#if card.story.CommentsCount}<span><i class="icon-[fa--comment-o]"></i> {card.story.CommentsCount}</span>{/if}
    {#if card.story.AttachmentsCount}<span><i class="icon-[fa--paperclip]"></i> {card.story.AttachmentsCount}</span>{/if}
    <span class="ml-auto">{card.story.EstimatedHours ? formatHours(card.story.EstimatedHours) : ''}</span>
    {#if card.story.AssigneeID}
      <Avatar name={people.userName(card.story.AssigneeID)} colorSeed={card.story.AssigneeID} />
    {/if}
  </div>
</Card>
