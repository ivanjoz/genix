<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import T from '#components/misc/T.svelte';
  import { formatTime, todayUnixDay } from '#libs/helpers.ts';
  import { tr } from '#core/store.svelte.ts';
  import { draggableItem, dropZone, type IDropPosition } from '#libs/drag-drop.ts';
  import { useUI } from '@genix/ui';
  import {
    SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, SPRINT_STATE_PLANNED, STORY_STATE_STOPPED, storyState, type ISprint, type IStory,
  } from '../project.ts';
  import type { ProjectBacklog, ProjectPeople } from '../project.svelte.ts';
  import { formatHours } from '../backlog/backlog.ts';
  import CardMoveModal from './CardMoveModal.svelte';
  import AssigneeFilter from '../AssigneeFilter.svelte';
  import SprintCard from './SprintCard.svelte';
  import SprintMenu from './SprintMenu.svelte';
  import SprintStateBar from './SprintStateBar.svelte';
  import {
    BOARD_COLUMNS, SPRINT_STATES, cardsInState, cardsOfAssignees, endOfColumnPosition, isNotCompleted, needsMoveReason, sprintCards, sprintDaysLeft,
    sprintAssigneeIDs, sprintMoveRequest, sprintProgress, type ISprintCard,
  } from './sprints.ts';
  import { moveSprintCard } from './sprints.svelte.ts';

  interface Props {
    backlog: ProjectBacklog
    people: ProjectPeople
    sprint: ISprint
    canWork: boolean
    canAdmin: boolean
    moveModalID: number
    menuLayerID: number
    onOpenStory: (story: IStory) => void
    onStart: () => void
    onClose: () => void
  }

  const { backlog, people, sprint, canWork, canAdmin, moveModalID, menuLayerID, onOpenStory, onStart, onClose }: Props = $props()

  const ui = useUI()
  const BY_COLUMN = 1
  const BY_ASSIGNEE = 2
  // The burndown and the other charts live in the Reports page.
  const layoutOptions = [{ id: BY_COLUMN, name: tr('Board|Tablero') }, { id: BY_ASSIGNEE, name: tr('By assignee|Por responsable') }]
  let boardLayout = $state({ mode: BY_COLUMN })
  let moveModal = $state<CardMoveModal>()

  const isMobile = $derived(ui.state.deviceType === 3)
  // The plain board fills the screen on a computer and each column scrolls on its own. By assignee
  // stacks one board per person, so the page scrolls; a phone stacks the columns, so it scrolls too.
  const hasColumnScroll = $derived(boardLayout.mode !== BY_ASSIGNEE && !isMobile)
  // Only the active sprint's board moves; planned boards are edited in the planner, closed ones are history.
  const canMove = $derived(canWork && sprint.State === SPRINT_STATE_ACTIVE)
  const storiesByID = $derived(new Map(backlog.stories.map((story) => [story.ID, story])))
  const cards = $derived(sprintCards(sprint, storiesByID))
  const progress = $derived(sprintProgress(cards))
  const today = todayUnixDay()

  // The people filter narrows the cards on the board; the progress, the bar and the menu still
  // count the whole sprint.
  let selectedAssigneeIDs = $state<number[]>([])
  const shownCards = $derived(cardsOfAssignees(cards, selectedAssigneeIDs))

  // One lane per assignee holding cards, then the unassigned ones. One lane (all) on the plain board.
  const lanes = $derived.by(() => {
    if (boardLayout.mode !== BY_ASSIGNEE) { return [{ assigneeID: -1, name: '', cards: shownCards }] }
    return sprintAssigneeIDs(shownCards, people.userName.bind(people)).map((assigneeID) => ({
      assigneeID,
      name: assigneeID ? people.userName(assigneeID) : tr('Unassigned|Sin responsable'),
      cards: cardsOfAssignees(shownCards, [assigneeID]),
    }))
  })

  const isCardPayload = (payload: unknown): payload is ISprintCard => !!payload && typeof payload === 'object' && 'sprintStory' in payload

  const moveCard = (card: ISprintCard, toState: number, dropPosition: IDropPosition | null, reason: string) => {
    const position = dropPosition || endOfColumnPosition(cards, toState, card.story.ID)
    moveSprintCard(backlog, sprintMoveRequest(sprint, card.story.ID, toState, position, reason))
  }

  // A drop on Stopped asks the reason first; any other drop moves at once.
  const dropCard = (card: ISprintCard, toState: number, dropPosition: IDropPosition) => {
    if (needsMoveReason(card.sprintStory, toState)) {
      moveModal?.openReasonModal(card, toState, dropPosition)
      return
    }
    moveCard(card, toState, dropPosition, '')
  }
</script>

<div class={hasColumnScroll ? 'flex flex-col h-[calc(100vh-var(--header-height)-20px)]' : ''}>
<!-- Two compact rows: the sprint, its dates and progress with the state bar under them, and its menu;
  then the goal (cut at 40% of the width, in full on hover) and the people filter. The hours are in the menu. -->
<div class="flex flex-col gap-4 mb-12 shrink-0">
  <div class="flex items-center gap-12">
    <div class="min-w-0">
      <div class="flex items-center gap-8 flex-wrap text-[14px] text-fg-muted">
        <span class="h2 ff-bold text-fg">{sprint.Name}</span>
        <Badge text={SPRINT_STATES[sprint.State]?.name} css={SPRINT_STATES[sprint.State]?.badgeCss} />
        <span>{formatTime(sprint.StartDate, 'd/m/Y')} → {formatTime(sprint.EndDate, 'd/m/Y')}</span>
        {#if sprint.State === SPRINT_STATE_ACTIVE && today > sprint.EndDate}
          <span>·</span><span class="text-red-fg"><T text="Past its end date|Pasó su fecha de fin" /></span>
        {:else if sprint.State === SPRINT_STATE_ACTIVE}
          <span>·</span><span><b>{sprintDaysLeft(sprint, today)}</b> <T text="days left|días restantes" /></span>
        {/if}
        <span>·</span><span><b>{progress.finishedCount}</b> / {progress.storiesCount} <T text="stories finished|historias finalizadas" /></span>
      </div>
      <SprintStateBar {cards} css="mt-4" />
    </div>
    <!-- Everything about the sprint itself (progress, grouping, start / close, reports) is in its menu. -->
    <Button icon="icon-[fa--bars]" color="purple" css="ml-auto" label="Opens the sprint's menu: progress, grouping and actions.|Abre el menú del sprint: avance, agrupación y acciones."
      onClick={() => ui.openSideLayer(menuLayerID)} />
  </div>
  {#if sprint.Goal || cards.length > 0}
    <div class="text-[14px] flex items-center gap-12">
      {#if sprint.Goal}<span class="text-fg-soft truncate max-w-[40%]" title={sprint.Goal}>{sprint.Goal}</span>{/if}
      {#if cards.length > 0}
        <AssigneeFilter assigneeIDs={sprintAssigneeIDs(cards, people.userName.bind(people))} {people} bind:selectedAssigneeIDs />
      {/if}
    </div>
  {/if}
</div>

{#if sprint.State === SPRINT_STATE_PLANNED}
  <div class="text-[14px] text-fg-muted mb-8"><T text="The sprint has not started: its cards move once it starts.|El sprint no ha iniciado: sus tarjetas se mueven cuando inicie." /></div>
{:else if sprint.State === SPRINT_STATE_CLOSED}
  <div class="text-[14px] text-fg-muted mb-8"><T text="Closed sprint: read only.|Sprint cerrado: solo lectura." /></div>
{/if}

{#snippet stateZone(laneCards: ISprintCard[], state: number, zoneCss: string)}
  {@const stateCards = cardsInState(laneCards, state)}
  <div class="flex flex-col gap-6 min-h-40 rounded p-4 data-[drop-active]:bg-accent-bg {zoneCss}"
    {@attach canMove && !isMobile ? dropZone({ accepts: isCardPayload, onDrop: (card, dropPosition) => dropCard(card, state, dropPosition) }) : undefined}>
    {#each stateCards as card (card.story.ID)}
      <div data-drag-id={card.story.ID} class="data-[dragging]:opacity-40 rounded-md
        data-[drop-before]:shadow-[0_-3px_0_var(--accent-solid)] data-[drop-after]:shadow-[0_3px_0_var(--accent-solid)]"
        {@attach canMove && !isMobile ? draggableItem(() => card) : undefined}>
        <SprintCard {card} {people} isNotCompleted={isNotCompleted(sprint, card.sprintStory)} isMoving={backlog.isStoryMoving(card.story.ID)}
          onOpen={(openedCard) => onOpenStory(openedCard.story)}
          onMoveMenu={canMove && isMobile ? (movedCard) => moveModal?.openMoveMenu(movedCard) : undefined} />
      </div>
    {/each}
  </div>
{/snippet}

<div class="flex flex-col gap-16 {hasColumnScroll ? 'grow min-h-0 pb-8' : 'pb-24'}">
  {#each lanes as lane (lane.assigneeID)}
    <section class={hasColumnScroll ? 'h-full' : ''}>
      {#if lane.name}
        <div class="ff-bold mb-6 flex items-center gap-8">
          {lane.name}
          <span class="text-[14px] text-fg-muted font-normal">{lane.cards.length} · {formatHours(sprintProgress(lane.cards).committedHours)}</span>
        </div>
      {/if}
      <div class="grid grid-cols-1 md:grid-cols-4 gap-10 {hasColumnScroll ? 'h-full' : ''}">
        {#each BOARD_COLUMNS as column (column.id)}
          <!-- A column sharing several states (Blocked / In review) stacks one container per state;
            the last one grows to the row's height. Such a column scrolls as one. -->
          <div class="flex flex-col gap-10 {hasColumnScroll ? 'min-h-0 overflow-y-auto' : ''}">
            {#each column.states as state, stateIndex (state)}
              {@const isBlocked = state === STORY_STATE_STOPPED}
              <div class="rounded-lg p-6 flex flex-col {isBlocked ? 'bg-red-bg' : 'bg-surface-muted'} {stateIndex === column.states.length - 1 ? 'grow' : ''}">
                <div class="flex items-center gap-6 px-4 py-4 text-[14px] ff-bold {isBlocked ? 'text-red-fg' : 'text-fg-soft'}">
                  <T text={storyState(state).name} /><span class="text-fg-muted font-normal">{cardsInState(lane.cards, state).length}</span>
                </div>
                {@render stateZone(lane.cards, state, 'grow')}
              </div>
            {/each}
          </div>
        {/each}
      </div>
    </section>
  {/each}
  {#if cards.length === 0}
    <div class="text-fg-muted text-center py-16"><T text="This sprint has no stories yet.|Este sprint todavía no tiene historias." /></div>
  {/if}
</div>
</div>

<CardMoveModal bind:this={moveModal} modalID={moveModalID} onMove={moveCard} />
<SprintMenu {sprint} {cards} {boardLayout} {layoutOptions} {canAdmin} {today} layerID={menuLayerID} {onStart} {onClose} />
