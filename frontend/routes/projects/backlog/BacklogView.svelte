<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Card from '#components/cards/Card.svelte';
  import SelectedTags from '#components/form/SelectedTags.svelte';
  import Avatar from '#components/misc/Avatar.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import CollapsibleSection from '#components/misc/CollapsibleSection.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { draggableItem, dropZone } from '#libs/drag-drop.ts';
  import { useUI } from '@genix/ui';
  import {
    projectColor, storyCode, storyPriority, storySprintNote, storyState, storyType,
    type IStory,
  } from '../project.ts';
  import type { ProjectBacklog, ProjectPeople } from '../project.svelte.ts';
  import {
    backlogFilterOptions, backlogRankRequest, filterBacklogStories, formatHours, groupBacklog, hasActiveFilters, sumEstimatedHours,
    type IBacklogFilters,
  } from './backlog.ts';

  interface Props {
    backlog: ProjectBacklog
    people: ProjectPeople
    // Owned by the page: the filters are edited in BacklogFilters, next to the project select.
    filters: IBacklogFilters
    canWork: boolean
    onOpenStory: (story: IStory) => void
    onNewStory: (epicID: number) => void
    onEditEpics: () => void
  }

  let { backlog, people, filters = $bindable(), canWork, onOpenStory, onNewStory, onEditEpics }: Props = $props()

  const ui = useUI()
  let collapsedEpicIDs = $state(new Set<number>())

  const sprintsByID = $derived(new Map(backlog.sprints.map((sprint) => [sprint.ID, sprint])))
  const shownStories = $derived(filterBacklogStories(backlog.stories, filters, sprintsByID))
  const isFiltered = $derived(hasActiveFilters(filters))
  // Empty epics show while nothing is filtered, so stories can be dropped into them.
  const groups = $derived(groupBacklog(shownStories, backlog.epics, !isFiltered))
  // Reordering by dragging needs the whole list in view (and a mouse).
  const canReorder = $derived(canWork && !isFiltered && ui.state.deviceType !== 3)
  const filterOptions = $derived(backlogFilterOptions(backlog.epics, backlog.project?.Members || [], (userID) => people.userName(userID), tr))

  // A dragged payload is a story when it carries the fields every story has.
  const isStoryPayload = (payload: unknown): payload is IStory => !!payload && typeof payload === 'object' && 'BacklogRank' in payload && 'Title' in payload

  const toggleEpic = (epicID: number) => {
    const nextCollapsed = new Set(collapsedEpicIDs)
    if (nextCollapsed.has(epicID)) { nextCollapsed.delete(epicID) } else { nextCollapsed.add(epicID) }
    collapsedEpicIDs = nextCollapsed
  }

  const moveStory = (story: IStory, epicID: number, previousID: string, nextID: string) => {
    if (!backlog.project) { return }
    backlog.rankStory(backlogRankRequest(backlog.project.ID, story.ID, epicID, { previousID, nextID }))
  }
</script>

<!-- On desktop the page is pinned to the viewport (+page.svelte) and the backlog scrolls in this box,
  so the project select and the filters stay in view. -->
<div class="md:flex-1 md:min-h-0 md:overflow-y-auto">
<!-- min-h-27 = a filter tag's height: the line keeps its height when the first tag shows or the last goes. -->
<div class="flex items-center gap-12 mb-8 flex-wrap min-h-27 text-[14px] text-fg-muted">
  <!-- min-w-240 = the project select's width, so the tags start right under the filter button. -->
  <div class="flex items-center gap-12 md:min-w-240">
    <span>{shownStories.length} <T text="stories|historias" /></span>
    <span>·</span>
    <span><T text="Estimated|Estimado" />: <b>{formatHours(sumEstimatedHours(shownStories))}</b></span>
  </div>
  <!-- The active layer filters, readable and removable while the layer is closed. -->
  <SelectedTags saveOn={filters} useBorder fields={[
    { save: 'typeID', options: filterOptions.typeOptions, keyId: 'id', keyName: 'name' },
    { save: 'state', options: filterOptions.stateOptions, keyId: 'id', keyName: 'name' },
    { save: 'epicID', options: filterOptions.epicOptions, keyId: 'ID', keyName: 'Name' },
    { save: 'assigneeID', options: filterOptions.assigneeOptions, keyId: 'ID', keyName: 'Name' },
    { save: 'onlyPlannable', label: 'To plan only|Solo por planificar' },
  ]} />
  {#if canReorder}
    <span class="ml-auto hidden md:inline text-fg-subtle"><T text="Drag the stories to reorder them or move them to another epic.|Arrastra las historias para ordenarlas o moverlas a otra épica." /></span>
  {/if}
</div>

{#if backlog.isLoaded && groups.length === 0}
  <div class="text-fg-muted py-24 text-center">
    <T text={backlog.stories.length === 0 ? 'The backlog is empty: create the first story.|El backlog está vacío: crea la primera historia.' : 'No story matches the filters.|Ninguna historia coincide con los filtros.'} />
  </div>
{/if}

<div class="flex flex-col gap-12 pb-24">
  {#each groups as group (group.epic?.ID ?? 0)}
    {@const epicID = group.epic?.ID ?? 0}
    {@const isCollapsed = collapsedEpicIDs.has(epicID)}
    <CollapsibleSection collapsed={isCollapsed} onToggle={() => toggleEpic(epicID)}>
      {#snippet header()}
        <span class="w-12 h-12 rounded-full shrink-0 {group.epic ? projectColor(group.epic.Color).swatchCss : 'bg-line-strong'}"></span>
        <span class="ff-bold">{group.epic ? group.epic.Name : tr('No epic|Sin épica')}</span>
        <span class="text-[14px] text-fg-muted">{group.stories.length} · {formatHours(group.estimatedHours)}</span>
      {/snippet}
      {#snippet actions()}
        {#if group.epic && canWork}
          <Button name="Story|Historia" icon="icon-[fa--plus] text-[11px]" label={`Creates a story in ${group.epic.Name}|Crea una historia en ${group.epic.Name}`}
            color="green" mode="subtle" height="small" onClick={() => onNewStory(epicID)} />
          <Button icon="icon-[fa--pencil]" label={`Edit the epic ${group.epic.Name}|Editar la épica ${group.epic.Name}`}
            color="blue" mode="white" height="small" useCircle onClick={onEditEpics} />
        {/if}
      {/snippet}
      <div class="flex flex-col min-h-12 data-[drop-active]:bg-accent-bg"
        {@attach canReorder ? dropZone({
          accepts: isStoryPayload,
          onDrop: (story, dropPosition) => moveStory(story, epicID, dropPosition.previousID, dropPosition.nextID),
        }) : undefined}>
        {#each group.stories as story (story.ID)}
          {@const type = storyType(story.Type)}
          {@const priority = storyPriority(story.Priority)}
          {@const sprintNote = storySprintNote(story, sprintsByID)}
          <div data-drag-id={story.ID} class="border-b border-line-soft last:border-b-0 data-[dragging]:opacity-40
            data-[drop-before]:shadow-[inset_0_3px_0_var(--accent-solid)] data-[drop-after]:shadow-[inset_0_-3px_0_var(--accent-solid)]"
            {@attach canReorder ? draggableItem(() => story) : undefined}>
            <Card label={`Open ${storyCode(story)} ${story.Title}|Abrir ${storyCode(story)} ${story.Title}`}
              onClick={() => onOpenStory(story)}
              css="flex items-center gap-8 flex-wrap md:flex-nowrap px-10 py-6 cursor-pointer hover:bg-surface-soft {canReorder ? 'md:cursor-grab' : ''}">
              <!-- Fixed-width slot: the type icons differ in width and the titles must line up. -->
              <span class="w-18 shrink-0 flex justify-center">
                {#if backlog.isStoryMoving(story.ID)}
                  <i class="icon-[fa--spinner] animate-spin text-accent-solid" title={tr('Saving the move...|Guardando el movimiento...')}></i>
                {:else}
                  <i class="{type.icon} {type.iconCss}" title={tr(type.name)}></i>
                {/if}
              </span>
              <span class="text-[14px] text-fg-muted shrink-0 w-70">{storyCode(story)}</span>
              <div class="grow min-w-0 basis-[60%] md:basis-auto">
                <div class="truncate">{story.Title}</div>
                {#if story.Summary}<div class="truncate text-[14px] text-fg-muted">{story.Summary}</div>{/if}
              </div>
              <i class="{priority.icon} {priority.iconCss} shrink-0" title={tr(priority.name)}></i>
              {#if story.CommentsCount}
                <span class="text-[14px] text-fg-muted shrink-0"><i class="icon-[fa--comment-o]"></i> {story.CommentsCount}</span>
              {/if}
              {#if story.AttachmentsCount}
                <span class="text-[14px] text-fg-muted shrink-0"><i class="icon-[fa--paperclip]"></i> {story.AttachmentsCount}</span>
              {/if}
              {#if sprintNote}
                <Badge text={sprintNote.text} size="small" css={sprintNote.isNotCompleted ? 'bg-orange-bg-strong text-orange-fg' : 'bg-surface-muted text-fg-muted'} />
              {/if}
              <Badge text={storyState(story.State).name} size="small" css={storyState(story.State).badgeCss} />
              <span class="text-[14px] text-fg-muted shrink-0 w-48 text-right">{story.EstimatedHours ? formatHours(story.EstimatedHours) : ''}</span>
              {#if story.AssigneeID}
                <Avatar name={people.userName(story.AssigneeID)} colorSeed={story.AssigneeID} size={28} />
              {:else}
                <span class="w-28 h-28 shrink-0"></span>
              {/if}
            </Card>
          </div>
        {/each}
        {#if group.stories.length === 0}
          <div class="px-10 py-8 text-[14px] text-fg-subtle"><T text="No stories in this epic.|Sin historias en esta épica." /></div>
        {/if}
      </div>
    </CollapsibleSection>
  {/each}
</div>
</div>
