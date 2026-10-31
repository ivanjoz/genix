<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Card from '#components/cards/Card.svelte';
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte';
  import DateInput from '#components/form/DateInput.svelte';
  import FilterInput from '#components/form/FilterInput.svelte';
  import Input from '#components/form/Input.svelte';
  import Modal from '#components/layers/Modal.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import Avatar from '#components/misc/Avatar.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { draggableItem, dropZone } from '#libs/drag-drop.ts';
  import { useUI } from '@genix/ui';
  import { confirmWarn, hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify';
  import {
    SPRINT_STATE_PLANNED, STORY_STATES, STORY_TYPES, storyCode, storyPriority, storyType, type ISprint, type IStory,
  } from '../project.ts';
  import type { ProjectBacklog, ProjectPeople } from '../project.svelte.ts';
  import StoryPanel from '../story/StoryPanel.svelte';
  import { NONE_FILTER, formatHours } from '../backlog/backlog.ts';
  import CardMoveModal from './CardMoveModal.svelte';
  import {
    PLAN_BY_ASSIGNEE, PLAN_BY_STATE, needsMoveReason, planGroupsByAssignee, planGroupsByState, planStoryState, plannableStories, sprintSaveRequest,
    validateSprint, type IPlanStateChange, type ISprintCard,
  } from './sprints.ts';
  import { deleteSprint, saveSprintPlan } from './sprints.svelte.ts';

  interface Props {
    backlog: ProjectBacklog
    people: ProjectPeople
    // The sprint being planned: a new one (ID 0) or an open one.
    sprint: ISprint
    currentUserID: number
    storyLayerID: number
    // Asks the reason of a story dropped on Blocked.
    moveModalID: number
    sprintNameModalID: number
    onDone: (savedSprint: ISprint | null) => void
  }

  const { backlog, people, sprint, currentUserID, storyLayerID, moveModalID, sprintNameModalID, onDone }: Props = $props()

  let storyPanel = $state<StoryPanel>()
  let moveModal = $state<CardMoveModal>()

  const ui = useUI()
  // svelte-ignore state_referenced_locally
  let sprintForm = $state<ISprint>({ ...sprint })
  // svelte-ignore state_referenced_locally
  let plannedStoryIDs = $state<number[]>((sprint.Stories || []).map((sprintStory) => sprintStory.StoryID))
  // Assignees set by dropping a story on a person; saved with the plan.
  let assigneeByStoryID = $state(new Map<number, number>())
  // States set by dropping a story on a state's group; saved with the plan.
  let stateChangeByStoryID = $state(new Map<number, IPlanStateChange>())
  // The modal's draft of the name and goal: applied to the plan on "Apply", saved with it.
  let sprintNameForm = $state({ Name: '', Goal: '' })
  let planView = $state({ mode: PLAN_BY_STATE })
  let backlogFilters = $state({ text: '', epicID: 0, typeID: 0 })

  const isMobile = $derived(ui.state.deviceType === 3)
  const storiesByID = $derived(new Map(backlog.stories.map((story) => [story.ID, story])))
  const plannedStories = $derived(plannedStoryIDs.flatMap((storyID) => storiesByID.get(storyID) || []))
  const planGroups = $derived(planView.mode === PLAN_BY_ASSIGNEE
    ? planGroupsByAssignee(plannedStories, backlog.project?.Members || [], assigneeByStoryID, (userID) => people.userName(userID))
    : planGroupsByState(plannedStories, sprintForm, stateChangeByStoryID, STORY_STATES))
  const plannedHours = $derived(plannedStories.reduce((total, story) => total + (story.EstimatedHours || 0), 0))

  const offeredStories = $derived.by(() => {
    const filterText = backlogFilters.text.trim().toLowerCase()
    return plannableStories(backlog.stories, backlog.sprints, sprintForm.ID, new Set(plannedStoryIDs)).filter((story) =>
      (!backlogFilters.typeID || story.Type === backlogFilters.typeID)
      && (!backlogFilters.epicID || (backlogFilters.epicID === NONE_FILTER ? !story.EpicID : story.EpicID === backlogFilters.epicID))
      && (!filterText || `${storyCode(story)} ${story.Title} ${story.Summary || ''}`.toLowerCase().includes(filterText)))
  })

  // genix-ui selects show option names as given: translate them here.
  const viewOptions = [{ id: PLAN_BY_STATE, name: tr('By state|Por estado') }, { id: PLAN_BY_ASSIGNEE, name: tr('By assignee|Por responsable') }]
  const typeOptions = STORY_TYPES.map((type) => ({ id: type.id, name: tr(type.name) }))
  const epicOptions = $derived([{ ID: NONE_FILTER, Name: tr('No epic|Sin épica') },
    ...backlog.epics.map((epic) => ({ ID: epic.ID, Name: epic.Name }))])

  // The two drag payloads: a story offered by the backlog column, and a story of the plan.
  type IPlanPayload = { source: 'backlog' | 'plan', story: IStory }
  const isPlanPayload = (payload: unknown): payload is IPlanPayload => !!payload && typeof payload === 'object' && 'source' in payload
  // The group under the pointer while dragging: a dashed accent outline drawn inside it, so no scroll container clips it.
  const dropActiveCss = 'data-[drop-active]:bg-accent-bg-strong data-[drop-active]:outline-2 data-[drop-active]:outline-dashed '
    + 'data-[drop-active]:outline-accent-solid data-[drop-active]:-outline-offset-2'

  const addToPlan = (story: IStory) => {
    if (!plannedStoryIDs.includes(story.ID)) { plannedStoryIDs = [...plannedStoryIDs, story.ID] }
  }

  const removeFromPlan = (story: IStory) => {
    plannedStoryIDs = plannedStoryIDs.filter((storyID) => storyID !== story.ID)
    if (stateChangeByStoryID.delete(story.ID)) { stateChangeByStoryID = new Map(stateChangeByStoryID) }
  }

  const setPlanState = (story: IStory, state: number, reason: string) => {
    addToPlan(story)
    stateChangeByStoryID = new Map(stateChangeByStoryID).set(story.ID, { State: state, Reason: reason })
  }

  // A drop on a group adds the story to the plan. By assignee it also assigns it; by state it takes
  // that state, and Blocked asks the reason first (cancelling it leaves the story where it was).
  const dropOnGroup = (payload: IPlanPayload, groupID: number) => {
    if (planView.mode === PLAN_BY_ASSIGNEE) {
      addToPlan(payload.story)
      assigneeByStoryID = new Map(assigneeByStoryID).set(payload.story.ID, groupID)
      return
    }
    const currentState = planStoryState(sprintForm, stateChangeByStoryID, payload.story.ID)
    if (groupID === currentState) {
      addToPlan(payload.story)
      return
    }
    if (needsMoveReason({ StoryID: payload.story.ID, State: currentState, Rank: 0, AddedAt: 0 }, groupID)) {
      const card: ISprintCard = { sprintStory: { StoryID: payload.story.ID, State: currentState, Rank: 0, AddedAt: 0 }, story: payload.story }
      moveModal?.openReasonModal(card, groupID, { previousID: '', nextID: '' })
      return
    }
    setPlanState(payload.story, groupID, '')
  }

  const openSprintNameModal = () => {
    sprintNameForm = { Name: sprintForm.Name || '', Goal: sprintForm.Goal || '' }
    ui.openModal(sprintNameModalID)
  }

  const applySprintNameForm = () => {
    sprintForm.Name = sprintNameForm.Name.trim()
    sprintForm.Goal = sprintNameForm.Goal.trim()
    ui.closeModal(sprintNameModalID)
  }

  const savePlan = async () => {
    const validationError = validateSprint(sprintForm)
    if (validationError) {
      notifyFailure(tr(validationError))
      return
    }
    showLoading(tr('Saving sprint...|Guardando sprint...'))
    try {
      const savedSprint = await saveSprintPlan(backlog, sprintSaveRequest(sprintForm, plannedStoryIDs, assigneeByStoryID, stateChangeByStoryID, storiesByID))
      notifySuccess(sprintForm.ID ? 'Sprint saved|Sprint guardado' : `${savedSprint.Name} created|${savedSprint.Name} creado`)
      onDone(savedSprint)
    } catch {
      // genix-ui already notified the backend's error message. Keep the plan so it can be fixed.
    }
    hideLoading()
  }

  const confirmDeleteSprint = async () => {
    const isConfirmed = await confirmWarn({
      title: tr('Delete sprint|Eliminar sprint'),
      message: tr(`Delete ${sprintForm.Name}? Its stories go back to the backlog.|¿Eliminar ${sprintForm.Name}? Sus historias vuelven al backlog.`),
    })
    if (!isConfirmed) { return }
    showLoading(tr('Deleting sprint...|Eliminando sprint...'))
    try {
      await deleteSprint(backlog, sprintForm)
      notifySuccess('Sprint deleted|Sprint eliminado')
      onDone(null)
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }
</script>

{#snippet storyRow(story: IStory, source: 'backlog' | 'plan')}
  {@const type = storyType(story.Type)}
  {@const priority = storyPriority(story.Priority)}
  {@const assigneeID = assigneeByStoryID.get(story.ID) ?? (story.AssigneeID || 0)}
  <!-- A click opens the story's form; on a computer the row is also dragged. -->
  <div data-drag-id={story.ID} class="data-[dragging]:opacity-40" {@attach isMobile ? undefined : draggableItem(() => ({ source, story }))}>
    <Card label={`Open ${storyCode(story)} ${story.Title}|Abrir ${storyCode(story)} ${story.Title}`} onClick={() => storyPanel?.openStoryPanel(story)}
      css="flex items-center gap-8 px-8 py-6 bg-surface rounded border border-line hover:border-accent-border {isMobile ? 'cursor-pointer' : 'cursor-grab'}">
      {#if source === 'backlog'}
        <!-- The backlog column is narrow: the type's icon at the top left with the code under it, and the title in up to 2 lines. -->
        <div class="flex flex-col items-start shrink-0 self-start">
          <i class="{type.icon} {type.iconCss}" title={tr(type.name)}></i>
          <span class="text-[14px] text-fg-muted">{storyCode(story)}</span>
        </div>
        <span class="grow min-w-0 line-clamp-2 leading-[1.4]">{story.Title}</span>
      {:else}
        <i class="{type.icon} {type.iconCss} shrink-0" title={tr(type.name)}></i>
        <span class="text-[14px] text-fg-muted shrink-0">{storyCode(story)}</span>
        <span class="grow min-w-0 truncate">{story.Title}</span>
      {/if}
      <i class="{priority.icon} {priority.iconCss} shrink-0" title={tr(priority.name)}></i>
      <span class="text-[14px] text-fg-muted shrink-0">{story.EstimatedHours ? formatHours(story.EstimatedHours) : ''}</span>
      {#if assigneeID}
        <Avatar name={people.userName(assigneeID)} colorSeed={assigneeID} />
      {/if}
      <!-- On a computer a story is planned by dragging it; the button is for phones, which can't drag. -->
      {#if source === 'backlog'}
        {#if isMobile}
          <Button icon="icon-[fa--plus]" label={`Add ${storyCode(story)} to the sprint|Agregar ${storyCode(story)} al sprint`}
            onClick={(ev) => { ev?.stopPropagation(); addToPlan(story) }} />
        {/if}
      {:else}
        <Button icon="icon-[fa--times]" label={`Take ${storyCode(story)} out of the sprint|Quitar ${storyCode(story)} del sprint`}
          onClick={(ev) => { ev?.stopPropagation(); removeFromPlan(story) }} />
      {/if}
    </Card>
  </div>
{/snippet}

<!-- The backlog is a column of the page, at the left of the plan, and each column scrolls on its own on a computer.
  On a phone they stack, the plan first: the backlog can be long and the plan is what is being built. -->
<div class="flex flex-col-reverse md:flex-row gap-12 md:h-[calc(100vh-var(--header-height)-20px)]">
<section class="md:w-[36%] md:max-w-460 md:min-w-320 shrink-0 flex flex-col gap-8 rounded-lg bg-surface border border-line p-10 md:overflow-y-auto">
  <span class="h2 ff-bold">Backlog</span>
  <FilterInput label="Filter stories|Filtrar historias" icon="icon-[fa--search]" bind:value={backlogFilters.text} css="w-full" />
  <div class="grid grid-cols-2 gap-8">
    <SearchSelect label="Epic|Épica" placeholder="All epics|Todas las épicas" options={epicOptions} keyId="ID" keyName="Name" saveOn={backlogFilters} save="epicID" />
    <SearchSelect label="Type|Tipo" placeholder="All types|Todos los tipos" options={typeOptions} keyId="id" keyName="name" saveOn={backlogFilters} save="typeID" />
  </div>
  <div class="text-[14px] text-fg-muted">{offeredStories.length} <T text="stories that can be planned|historias planificables" /></div>
  <div class="flex flex-col gap-6 min-h-120 grow rounded p-4 {dropActiveCss}"
    {@attach isMobile ? undefined : dropZone({ accepts: (payload): payload is IPlanPayload => isPlanPayload(payload) && payload.source === 'plan',
      onDrop: (payload) => removeFromPlan(payload.story) })}>
    {#each offeredStories as story (story.ID)}
      {@render storyRow(story, 'backlog')}
    {/each}
    {#if offeredStories.length === 0}
      <div class="text-[14px] text-fg-subtle text-center py-12"><T text="No story to plan.|No hay historias por planificar." /></div>
    {/if}
  </div>
</section>

<!-- On a computer the name line stays fixed and only what is below it scrolls. -->
<div class="grow min-w-0 flex flex-col">
  <!-- The sprint's name and goal head the plan; they are edited in a modal. -->
  <!-- One line with the buttons: the goal is cut at half the width, in full on hover. -->
  <div class="flex items-center gap-8 mb-10 shrink-0">
    <div class="flex items-center gap-12 min-w-0 max-w-[50%]">
      <div class="min-w-0">
        <div class="text-[17px] ff-bold truncate {sprintForm.Name ? '' : 'text-fg-muted'}">{sprintForm.Name || tr('Sprint N (automatic)|Sprint N (automático)')}</div>
        <div class="text-[15px] truncate {sprintForm.Goal ? 'text-fg-soft' : 'text-fg-subtle'}" title={sprintForm.Goal}>{sprintForm.Goal || tr('No goal|Sin objetivo')}</div>
      </div>
      <Button icon="icon-[fa--pencil]" color="purple" useCircle css="shrink-0" label="Edit the sprint's name and goal|Editar el nombre y el objetivo del sprint" onClick={openSprintNameModal} />
    </div>
    <div class="flex gap-8 ml-auto shrink-0">
      {#if sprintForm.ID && sprintForm.State === SPRINT_STATE_PLANNED}
        <Button icon="icon-[fa--trash]" color="red" label="Deletes this planned sprint.|Elimina este sprint planificado." onClick={confirmDeleteSprint} />
      {/if}
      <Button name="Save|Guardar" icon="icon-[fa--floppy-o]" color="blue" hideNameOnMobile label="Saves the sprint and its stories.|Guarda el sprint y sus historias."
        onClick={() => { void savePlan() }} />
      <Button icon="icon-[fa--times]" color="yellow" label="Leaves without saving.|Sale sin guardar." onClick={() => onDone(null)} />
    </div>
  </div>

  <div class="md:grow md:min-h-0 md:overflow-y-auto">
  <!-- The dates at the left; the plan's view and its totals at the right. -->
  <!-- genix-ui fields only label on top: the dates' labels sit at their side as plain text. -->
  <div class="flex items-center gap-16 flex-wrap mb-10">
    <label class="flex items-center gap-8">
      <span class="text-[14px] text-fg-muted"><T text="Start|Inicio" /></span>
      <DateInput placeholder="Start|Inicio" saveOn={sprintForm} save="StartDate" css="w-140" />
    </label>
    <label class="flex items-center gap-8">
      <span class="text-[14px] text-fg-muted"><T text="End|Fin" /></span>
      <DateInput placeholder="End|Fin" saveOn={sprintForm} save="EndDate" css="w-140" />
    </label>
    <div class="flex items-center gap-12 flex-wrap ml-auto">
      <CheckboxOptions saveOn={planView} save="mode" options={viewOptions} keyId="id" keyName="name" useButtonsSlim={true} />
      <span class="text-[14px] text-fg-muted">{plannedStories.length} <T text="stories|historias" /> · <b>{formatHours(plannedHours)}</b></span>
    </div>
  </div>

  <!-- The whole group, its title included, is the drop target: a state's or a person's. -->
  <div class="flex flex-col gap-10 pb-24">
    {#each planGroups as group (group.id)}
      <section class="rounded-lg bg-surface-muted p-8 {dropActiveCss}"
        {@attach isMobile ? undefined : dropZone({ accepts: isPlanPayload, onDrop: (payload) => dropOnGroup(payload, group.id) })}>
        <div class="flex items-center gap-8 mb-6 text-[14px]">
          <span class="ff-bold text-fg-soft"><T text={group.name} /></span>
          <span class="text-fg-muted">{group.stories.length} · {formatHours(group.estimatedHours)}</span>
        </div>
        <div class="flex flex-col gap-6 min-h-40">
          {#each group.stories as story (story.ID)}
            {@render storyRow(story, 'plan')}
          {/each}
        </div>
      </section>
    {/each}
  </div>
  </div>
</div>
</div>

<!-- Edits a story right away, apart from the plan. An assignee saved there replaces the one dropped in "By assignee". -->
<StoryPanel bind:this={storyPanel} {backlog} {people} {currentUserID} layerID={storyLayerID} isFormOnly={true}
  onStoryChanged={(story) => { if (assigneeByStoryID.delete(story.ID)) { assigneeByStoryID = new Map(assigneeByStoryID) } }} />
<Modal id={sprintNameModalID} size={4} title="Sprint name and goal|Nombre y objetivo del sprint" saveButtonLabel="Apply|Aplicar"
  saveIcon="icon-[fa--check]" onSave={applySprintNameForm}>
  <div class="flex flex-col gap-12">
    <Input label="Name|Nombre" saveOn={sprintNameForm} save="Name" placeholder={tr('Sprint N (automatic)|Sprint N (automático)')} focusOnOpen={true} />
    <Input label="Goal|Objetivo" saveOn={sprintNameForm} save="Goal" useTextArea={true} rows={3} />
  </div>
</Modal>
<CardMoveModal bind:this={moveModal} modalID={moveModalID} onMove={(card, toState, _dropPosition, reason) => setPlanState(card.story, toState, reason)} />
