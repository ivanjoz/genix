<script lang="ts">
  import Layer from '#components/layers/Layer.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { useUI } from '@genix/ui';
  import { untrack } from 'svelte';
  import { confirmWarn, hideLoading, notifyFailure, notifySuccess, showLoading } from '@genix/ui/notify';
  import { canWorkInProject, isProjectAdmin, storyCode, storySprintNote, storyState, type IStory } from '../project.ts';
  import type { ProjectBacklog, ProjectPeople } from '../project.svelte.ts';
  import StoryForm from './StoryForm.svelte';
  import StoryTimeline from './StoryTimeline.svelte';
  import { makeEmptyStory, makeEmptyStoryDetail, makeStoryDetailForm, validateStory, type IStoryDetail } from './story.ts';
  import { OpenStory } from './story.svelte.ts';

  interface Props {
    backlog: ProjectBacklog
    people: ProjectPeople
    currentUserID: number
    layerID: number
    // Only the form, for the sprint planner: no activity tab, no state / sprint tags, no delete.
    isFormOnly?: boolean
    // The story saved, deleted or commented. A page showing what the story holds besides the
    // backlog (the Documents page) reloads it; the sprint planner drops its pending assignee.
    onStoryChanged?: (story: IStory) => void
  }

  const { backlog, people, currentUserID, layerID, isFormOnly = false, onStoryChanged }: Props = $props()

  const ui = useUI()
  const DETAIL_VIEW = 1
  const ACTIVITY_VIEW = 2

  const openStory = new OpenStory()
  let storyForm = $state<IStory>(makeEmptyStory(0))
  // null while the detail of an existing story loads: the editor reads its value once, so it
  // renders only when the detail is there.
  let detailForm = $state<IStoryDetail | null>(null)
  let detailFormUpdated = $state(-1)
  let layerView = $state(DETAIL_VIEW)
  let uploadingCount = $state(0)

  const project = $derived(backlog.project)
  const canWork = $derived(!!project && canWorkInProject(project, currentUserID))
  const canDelete = $derived(!isFormOnly && !!project && canWork && isProjectAdmin(project, currentUserID) && storyForm.ID > 0)
  const sprintsByID = $derived(new Map(backlog.sprints.map((sprint) => [sprint.ID, sprint])))
  const sprintNote = $derived(storyForm.ID ? storySprintNote(storyForm, sprintsByID) : null)

  // The detail paints from the by-IDs cache, then again if the server holds a newer copy (its upd, the cache slot's version, moved).
  $effect(() => {
    const loadedDetail = openStory.detail
    if (!loadedDetail) { return }
    untrack(() => {
      if (loadedDetail.ID !== storyForm.ID || (loadedDetail.upd || 0) === detailFormUpdated) { return }
      detailForm = makeStoryDetailForm(loadedDetail)
      detailFormUpdated = loadedDetail.upd || 0
    })
  })

  // Authors may have left the team: name them by ID.
  $effect(() => {
    const activityUserIDs = openStory.activity.flatMap((activity) => [activity.AuthorID, activity.AssigneeID || 0])
    untrack(() => { void people.rememberUsers(activityUserIDs) })
  })

  export const openStoryPanel = (story: IStory) => {
    storyForm = { ...story }
    detailForm = null
    detailFormUpdated = -1
    layerView = DETAIL_VIEW
    ui.openSideLayer(layerID)
    void openStory.load(story.ProjectID, story.ID)
  }

  export const openNewStory = (projectID: number, epicID: number) => {
    openStory.reset()
    storyForm = makeEmptyStory(projectID, epicID)
    detailForm = makeEmptyStoryDetail(projectID)
    detailFormUpdated = -1
    layerView = DETAIL_VIEW
    ui.openSideLayer(layerID)
  }

  const saveStory = async () => {
    if (!detailForm) { return }
    if (uploadingCount > 0) {
      notifyFailure(tr('Wait for the files to finish uploading.|Espera a que terminen de subir los archivos.'))
      return
    }
    // A cleared select leaves null.
    const storyToSave = {
      ...storyForm, EstimatedHours: Number(storyForm.EstimatedHours) || 0, EpicID: storyForm.EpicID || 0, AssigneeID: storyForm.AssigneeID || 0,
    }
    const validationError = validateStory(storyToSave)
    if (validationError) {
      notifyFailure(tr(validationError))
      return
    }
    showLoading(tr('Saving story...|Guardando historia...'))
    try {
      const saved = await backlog.saveStory(storyToSave, detailForm)
      notifySuccess(storyForm.ID ? 'Story saved|Historia guardada' : `Story ${storyCode(saved.Story)} created|Historia ${storyCode(saved.Story)} creada`)
      onStoryChanged?.(saved.Story)
      ui.openSideLayer(0)
    } catch {
      // genix-ui already notified the backend's error message. Keep the form so the user can fix it.
    }
    hideLoading()
  }

  const confirmDeleteStory = async () => {
    const isConfirmed = await confirmWarn({
      title: tr('Delete story|Eliminar historia'),
      message: tr(`Delete ${storyCode(storyForm)} "${storyForm.Title}"? It also leaves the open sprints.|¿Eliminar ${storyCode(storyForm)} "${storyForm.Title}"? También sale de los sprints abiertos.`),
    })
    if (!isConfirmed) { return }
    showLoading(tr('Deleting story...|Eliminando historia...'))
    try {
      await backlog.deleteStory(storyForm)
      notifySuccess('Story deleted|Historia eliminada')
      onStoryChanged?.(storyForm)
      ui.openSideLayer(0)
    } catch {
      // genix-ui already notified the backend's error message.
    }
    hideLoading()
  }

  // A comment bumps the story's counter, as the backend did.
  const countComment = () => {
    const commentedStory = backlog.stories.find((story) => story.ID === storyForm.ID)
    if (commentedStory) { backlog.applyStory({ ...commentedStory, CommentsCount: (commentedStory.CommentsCount || 0) + 1 }) }
    onStoryChanged?.(storyForm)
  }
</script>

<Layer type="side" id={layerID} sideLayerSize={780} css="px-8 py-8 md:px-16 md:py-10"
  title={storyForm.ID ? `${storyCode(storyForm)} · ${storyForm.Title}` : 'New story|Nueva historia'} titleCss="h2 mb-6"
  options={storyForm.ID && !isFormOnly ? [[DETAIL_VIEW, 'Detail|Detalle'], [ACTIVITY_VIEW, 'Activity|Actividad']] : undefined}
  bind:selected={layerView}
  onClose={() => { openStory.reset(); detailForm = null }}
  onSave={canWork && layerView === DETAIL_VIEW ? () => { void saveStory() } : undefined}
  onDelete={canDelete ? confirmDeleteStory : undefined} buttonDeleteOnTitle={true}>
  {#snippet optionsSide()}
    <!-- State and sprint tags on the tabs' row: they describe the story in both views. -->
    {#if !isFormOnly}
      <div class="flex items-center gap-8 flex-wrap text-[14px]">
        <Badge text={storyState(storyForm.State).name} css={storyState(storyForm.State).badgeCss} />
        {#if sprintNote}
          <Badge text={sprintNote.text} css={sprintNote.isNotCompleted ? 'bg-orange-bg-strong text-orange-fg' : 'bg-surface-muted text-fg-soft'} />
        {/if}
      </div>
    {/if}
  {/snippet}
  {#if layerView === DETAIL_VIEW}
    <StoryForm {backlog} {people} bind:storyForm bind:detailForm bind:uploadingCount {canWork} />
  {:else}
    <div class="mt-12">
      <StoryTimeline {openStory} projectID={storyForm.ProjectID} storyID={storyForm.ID}
        canComment={canWork} userName={(userID) => people.userName(userID)}
        sprintName={(sprintID) => sprintsByID.get(sprintID)?.Name || `Sprint #${sprintID}`} onCommented={countComment} />
    </div>
  {/if}
</Layer>
